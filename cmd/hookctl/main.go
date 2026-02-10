package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"connectrpc.com/connect"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	workerv1connect "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1/workerv1connect"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
	"github.com/sebastianm/flowgentic/internal/worker/interceptors"
)

func main() {
	fs := flag.NewFlagSet("hookctl", flag.ExitOnError)
	hookName := fs.String("hook-name", "", "Hook name (Stop, SessionStart, PreToolUse, PostToolUse, etc.)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "hookctl: %v\n", err)
		os.Exit(1)
	}

	if *hookName == "" {
		fmt.Fprintln(os.Stderr, "hookctl: --hook-name is required")
		os.Exit(1)
	}

	workerURL := os.Getenv("AGENTCTL_WORKER_URL")
	if workerURL == "" {
		fmt.Fprintln(os.Stderr, "hookctl: AGENTCTL_WORKER_URL not set")
		os.Exit(1)
	}

	workloadID := os.Getenv("FLOWGENTIC_AGENT_RUN_ID")
	agentName := os.Getenv("AGENTCTL_AGENT")

	protoAgent, err := driver.ParseProtoAgent(agentName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hookctl: %v\n", err)
		os.Exit(1)
	}

	payload, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hookctl: read stdin: %v\n", err)
		os.Exit(1)
	}

	var opts []connect.ClientOption
	if secret := os.Getenv("AGENTCTL_WROKER_SECRET"); secret != "" {
		opts = append(opts, connect.WithInterceptors(interceptors.NewAuth(secret)))
	}

	client := workerv1connect.NewHookCtlServiceClient(http.DefaultClient, workerURL, opts...)
	_, err = client.ReportHook(context.Background(), connect.NewRequest(&workerv1.ReportHookRequest{
		SessionId: workloadID,
		Agent:     protoAgent,
		HookName:  *hookName,
		Payload:   payload,
	}))
	if err != nil {
		fmt.Fprintf(os.Stderr, "hookctl: report hook: %v\n", err)
		os.Exit(1)
	}
}
