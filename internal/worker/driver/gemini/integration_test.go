//go:build integration

package gemini

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	workerv1connect "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1/workerv1connect"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
	"github.com/sebastianm/flowgentic/internal/worker/workload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func integrationLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// Test 1: Launch headless — one-shot prompt
func TestIntegration_LaunchHeadless(t *testing.T) {
	log := integrationLogger()

	d := NewDriver(DriverDeps{Log: log})

	var mu sync.Mutex
	var events []driver.Event

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sess, err := d.Launch(ctx, driver.LaunchOpts{
		Mode:      driver.SessionModeHeadless,
		SessionID: "headless-test",
		Prompt:    "What is 2+2? Reply with just the number.",
		Cwd:       "/tmp",
	}, func(e driver.Event) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, e)
		fmt.Printf("  [EVENT] type=%s text=%q error=%q\n", e.Type, truncate(e.Text, 120), e.Error)
	})
	require.NoError(t, err)

	fmt.Printf("  Session ID: %s\n", sess.Info().ID)
	fmt.Printf("  Waiting for completion (up to 30s)...\n")

	err = sess.Wait(ctx)
	require.NoError(t, err)

	mu.Lock()
	eventCount := len(events)
	mu.Unlock()

	fmt.Printf("  Received %d events\n", eventCount)
	fmt.Println("  Headless test: PASSED")
}

// testHookHandler implements workerv1connect.AgentHookServiceHandler for the e2e test.
type testHookHandler struct {
	log     *slog.Logger
	manager *workload.AgentRunManager
}

func (h *testHookHandler) ReportHook(
	ctx context.Context,
	req *connect.Request[workerv1.ReportHookRequest],
) (*connect.Response[workerv1.ReportHookResponse], error) {
	h.log.Info("hook received in test server",
		"session_id", req.Msg.SessionId,
		"hook_name", req.Msg.HookName,
	)
	event := driver.HookEvent{
		SessionID: req.Msg.SessionId,
		Agent:     req.Msg.Agent.String(),
		HookName:  req.Msg.HookName,
		Payload:   req.Msg.Payload,
	}
	if err := h.manager.HandleHookEvent(ctx, event); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&workerv1.ReportHookResponse{}), nil
}

// Test 2: End-to-end with worker RPC
func TestIntegration_EndToEnd(t *testing.T) {
	log := integrationLogger()

	d := NewDriver(DriverDeps{Log: log})

	// Create a AgentRunManager just like the worker would.
	mgr := workload.NewAgentRunManager(log, "", "", d)

	// Wire up the RPC handler on a test HTTP server.
	mux := http.NewServeMux()
	h := &testHookHandler{log: log, manager: mgr}
	mux.Handle(workerv1connect.NewAgentHookServiceHandler(h))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Close()

	workerURL := fmt.Sprintf("http://%s", ln.Addr().String())
	fmt.Printf("  Test worker listening on %s\n", workerURL)

	var mu sync.Mutex
	var events []driver.Event

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Launch via manager (like the worker would).
	agentRunID := "integration-test-gemini"
	sess, err := mgr.Launch(context.Background(), agentRunID, "gemini", driver.LaunchOpts{
		Mode:   driver.SessionModeHeadless,
		Cwd:    "/tmp",
		Prompt: "What is 2+2? Reply with just the number.",
		EnvVars: map[string]string{
			"AGENTCTL_WORKER_URL": workerURL,
		},
	}, func(e driver.Event) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, e)
		fmt.Printf("  [EVENT] type=%s agent=%s text=%q\n", e.Type, e.Agent, truncate(e.Text, 80))
	})
	require.NoError(t, err)

	info := sess.Info()
	fmt.Printf("  Agent Run ID: %s\n", agentRunID)
	fmt.Printf("  Session ID: %s\n", info.ID)

	// Wait for session to complete.
	err = sess.Wait(ctx)
	require.NoError(t, err)

	drivers := mgr.ListDrivers()
	assert.Len(t, drivers, 1)
	assert.Equal(t, "gemini", drivers[0].Agent)

	fmt.Println("  E2E test: PASSED")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
