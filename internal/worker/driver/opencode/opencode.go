package opencode

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
	"github.com/sebastianm/flowgentic/internal/worker/procutil"
)

const agent = "opencode"

// DriverDeps are the dependencies for the OpenCode driver.
type DriverDeps struct {
	Log *slog.Logger
}

// Driver implements driver.Driver for OpenCode.
type Driver struct {
	log *slog.Logger

	mu       sync.Mutex
	sessions map[string]*openCodeSession
}

// NewDriver creates a new OpenCode driver.
func NewDriver(deps DriverDeps) *Driver {
	return &Driver{
		log:      deps.Log.With("driver", agent),
		sessions: make(map[string]*openCodeSession),
	}
}

func (d *Driver) Agent() string { return agent }

func (d *Driver) Capabilities() driver.Capabilities {
	return driver.Capabilities{
		Agent: agent,
		Supported: []driver.Capability{
			driver.CapStreaming,
			driver.CapSessionResume,
			driver.CapCostTracking,
			driver.CapCustomModel,
			driver.CapSystemPrompt,
			driver.CapYolo,
		},
	}
}

func (d *Driver) Launch(ctx context.Context, opts driver.LaunchOpts, onEvent driver.EventCallback) (driver.Session, error) {
	// Generate agent session ID if not resuming an existing session.
	agentSessionID := opts.SessionID
	if agentSessionID == "" {
		agentSessionID = uuid.New().String()
	}

	sess := &openCodeSession{
		info: driver.SessionInfo{
			ID:             opts.SessionID,
			AgentID:        agent,
			AgentSessionID: agentSessionID,
			Status:         driver.SessionStatusStarting,
			Mode:           opts.Mode,
			Cwd:            opts.Cwd,
			StartedAt:      time.Now(),
		},
		driver:  d,
		onEvent: onEvent,
		done:    make(chan struct{}),
	}

	d.mu.Lock()
	d.sessions[opts.SessionID] = sess
	d.mu.Unlock()

	if err := sess.launchHeadless(ctx, opts); err != nil {
		d.removeSession(opts.SessionID)
		return nil, err
	}

	return sess, nil
}

func (d *Driver) HandleHookEvent(_ context.Context, sessionID string, event driver.HookEvent) error {
	d.mu.Lock()
	sess, ok := d.sessions[sessionID]
	d.mu.Unlock()
	if !ok {
		return fmt.Errorf("opencode session not found: %s", sessionID)
	}

	now := time.Now()

	switch event.HookName {
	case "Stop", "TurnComplete":
		sess.setStatus(driver.SessionStatusIdle)
		sess.emit(driver.Event{
			Type:      driver.EventTypeTurnComplete,
			Timestamp: now,
			Agent:     agent,
			Raw:       event.Payload,
		})
	default:
		d.log.Debug("unhandled opencode hook", "hook", event.HookName)
	}

	return nil
}

func (d *Driver) removeSession(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.sessions, id)
}

// openCodeSession represents a running OpenCode session.
type openCodeSession struct {
	info               driver.SessionInfo
	driver             *Driver
	onEvent            driver.EventCallback
	done               chan struct{}
	serverCmd          *exec.Cmd
	serverURL          string
	openCodeSessionID  string
	mu                 sync.Mutex
}

func (s *openCodeSession) Info() driver.SessionInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.info
}

func (s *openCodeSession) Stop(ctx context.Context) error {
	s.mu.Lock()
	s.info.Status = driver.SessionStatusStopping
	serverCmd := s.serverCmd
	sessionID := s.openCodeSessionID
	serverURL := s.serverURL
	s.mu.Unlock()

	// Try to abort the running session gracefully.
	if sessionID != "" && serverURL != "" {
		abortCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		_ = s.abortSession(abortCtx, serverURL, sessionID)
	}

	if serverCmd != nil && serverCmd.Process != nil {
		// Try SIGTERM first.
		_ = serverCmd.Process.Signal(syscall.SIGTERM)

		// Wait up to 3 seconds for graceful exit, then kill.
		done := make(chan error, 1)
		go func() { done <- serverCmd.Wait() }()

		select {
		case <-done:
			// Process exited gracefully.
		case <-time.After(3 * time.Second):
			_ = serverCmd.Process.Kill()
			<-done
		}
	}

	s.mu.Lock()
	s.info.Status = driver.SessionStatusStopped
	s.mu.Unlock()

	s.closeDone()
	return nil
}

func (s *openCodeSession) Wait(_ context.Context) error {
	<-s.done
	return nil
}

func (s *openCodeSession) closeDone() {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
}

func (s *openCodeSession) setStatus(status driver.SessionStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.info.Status = status
}

func (s *openCodeSession) emit(events ...driver.Event) {
	if s.onEvent == nil {
		return
	}
	for _, e := range events {
		s.onEvent(e)
	}
}

// findFreePort finds an available TCP port on localhost.
func findFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}

// waitForHealthy polls the health endpoint until the server is ready.
func waitForHealthy(ctx context.Context, serverURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("opencode server not healthy after %v", timeout)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+"/global/health", nil)
		if err != nil {
			return err
		}

		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
}

// parseModel splits a "provider/model" string into providerID and modelID.
func parseModel(model string) (providerID, modelID string) {
	idx := strings.Index(model, "/")
	if idx < 0 {
		return "", model
	}
	return model[:idx], model[idx+1:]
}

// launchHeadless starts `opencode serve` with a deterministic port and connects to its API.
func (s *openCodeSession) launchHeadless(ctx context.Context, opts driver.LaunchOpts) error {
	port, err := findFreePort()
	if err != nil {
		return fmt.Errorf("find free port: %w", err)
	}

	serverURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	args := []string{"serve", "--port", fmt.Sprintf("%d", port)}
	cmd := exec.CommandContext(ctx, "opencode", args...)
	if opts.Cwd != "" {
		cmd.Dir = opts.Cwd
	}

	// Capture stderr so we can surface opencode server errors.
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := procutil.StartWithCleanup(cmd); err != nil {
		return fmt.Errorf("start opencode serve: %w", err)
	}

	// Log stderr lines from the opencode server in the background.
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			s.driver.log.Warn("opencode stderr", "line", scanner.Text())
		}
	}()

	s.mu.Lock()
	s.serverCmd = cmd
	s.serverURL = serverURL
	s.info.Status = driver.SessionStatusRunning
	s.mu.Unlock()

	go func() {
		defer s.closeDone()
		defer s.driver.removeSession(s.info.ID)

		// Wait for the server to become healthy.
		if err := waitForHealthy(ctx, serverURL, 30*time.Second); err != nil {
			s.driver.log.Warn("opencode health check failed", "error", err)
			s.setStatus(driver.SessionStatusErrored)
			s.emit(driver.Event{
				Type:      driver.EventTypeError,
				Timestamp: time.Now(),
				Agent:     agent,
				Error:     fmt.Sprintf("health check failed: %v", err),
			})
			return
		}

		s.driver.log.Info("opencode server started", "url", serverURL)

		// Create a session and send the initial prompt.
		if opts.Prompt == "" {
			s.driver.log.Warn("no prompt provided, opencode session will idle")
		} else {
			ocSessionID, err := s.createSession(ctx, serverURL, opts.Cwd)
			if err != nil {
				s.driver.log.Warn("create opencode session failed", "error", err)
				s.setStatus(driver.SessionStatusErrored)
				s.emit(driver.Event{
					Type:      driver.EventTypeError,
					Timestamp: time.Now(),
					Agent:     agent,
					Error:     fmt.Sprintf("create session: %v", err),
				})
				return
			}

			s.driver.log.Info("opencode session created", "opencode_session_id", ocSessionID)

			s.mu.Lock()
			s.openCodeSessionID = ocSessionID
			s.mu.Unlock()

			// Start SSE event stream before sending the prompt so we don't miss events.
			go s.consumeSSE(ctx, serverURL)

			if err := s.sendMessage(ctx, serverURL, ocSessionID, opts); err != nil {
				s.driver.log.Warn("send initial prompt failed", "error", err)
			} else {
				s.driver.log.Info("initial prompt sent", "opencode_session_id", ocSessionID)
			}
		}

		if err := cmd.Wait(); err != nil {
			s.setStatus(driver.SessionStatusErrored)
			s.emit(driver.Event{
				Type:      driver.EventTypeError,
				Timestamp: time.Now(),
				Agent:     agent,
				Error:     err.Error(),
			})
		} else {
			s.setStatus(driver.SessionStatusStopped)
		}
	}()

	return nil
}

// createSession creates a new session via POST /session.
func (s *openCodeSession) createSession(ctx context.Context, serverURL string, cwd string) (string, error) {
	url := serverURL + "/session"
	if cwd != "" {
		url += "?directory=" + cwd
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create session: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode session response: %w", err)
	}

	return result.ID, nil
}

// messageRequest is the body for POST /session/{id}/message.
type messageRequest struct {
	Model  *messageModel  `json:"model,omitempty"`
	System string         `json:"system,omitempty"`
	Parts  []messagePart  `json:"parts"`
}

type messageModel struct {
	ProviderID string `json:"providerID"`
	ModelID    string `json:"modelID"`
}

type messagePart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// sendMessage sends a prompt via POST /session/{id}/prompt_async (non-blocking).
func (s *openCodeSession) sendMessage(ctx context.Context, serverURL string, sessionID string, opts driver.LaunchOpts) error {
	url := serverURL + "/session/" + sessionID + "/prompt_async"

	body := messageRequest{
		Parts: []messagePart{{Type: "text", Text: opts.Prompt}},
	}

	if opts.Model != "" {
		providerID, modelID := parseModel(opts.Model)
		body.Model = &messageModel{
			ProviderID: providerID,
			ModelID:    modelID,
		}
	}

	if opts.SystemPrompt != "" {
		body.System = opts.SystemPrompt
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// prompt_async returns 204 No Content on success.
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("opencode prompt_async API error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// abortSession sends POST /session/{id}/abort to gracefully stop processing.
func (s *openCodeSession) abortSession(ctx context.Context, serverURL string, sessionID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL+"/session/"+sessionID+"/abort", nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// consumeSSE reads SSE events from the global event stream.
func (s *openCodeSession) consumeSSE(ctx context.Context, serverURL string) {
	sseURL := serverURL + "/event"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sseURL, nil)
	if err != nil {
		s.driver.log.Warn("create SSE request", "error", err)
		return
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.driver.log.Warn("SSE connect failed", "error", err, "url", sseURL)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.driver.log.Warn("SSE endpoint returned error", "status", resp.StatusCode, "url", sseURL)
		return
	}

	s.driver.log.Info("SSE stream connected", "url", sseURL)

	scanner := bufio.NewScanner(resp.Body)
	// Increase scanner buffer for large SSE payloads.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		events := normalizeSSEEvent([]byte(data))
		s.emit(events...)
	}
	if err := scanner.Err(); err != nil {
		s.driver.log.Warn("SSE stream error", "error", err)
	} else {
		s.driver.log.Info("SSE stream closed")
	}
}

// sseEvent represents an SSE event from the OpenCode server.
// Events use "properties" as the payload field.
type sseEvent struct {
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties,omitempty"`
}

// ssePartProperties wraps a part in message.part.updated events.
type ssePartProperties struct {
	Part ssePart `json:"part"`
}

// ssePart represents a message part in SSE events.
type ssePart struct {
	Type   string       `json:"type"`
	Text   string       `json:"text,omitempty"`
	CallID string       `json:"callID,omitempty"`
	Tool   string       `json:"tool,omitempty"`
	State  sseToolState `json:"state,omitempty"`
	// step-finish fields
	Cost   float64      `json:"cost,omitempty"`
	Tokens *sseTokens   `json:"tokens,omitempty"`
}

// sseToolState represents the state of a tool call.
type sseToolState struct {
	Status string          `json:"status,omitempty"`
	Input  json.RawMessage `json:"input,omitempty"`
}

// sseTokens represents token usage in step-finish events.
type sseTokens struct {
	Input  int `json:"input"`
	Output int `json:"output"`
}

// sseMessageProperties wraps message info in message.updated events.
type sseMessageProperties struct {
	Info struct {
		Cost struct {
			InputTokens  int     `json:"inputTokens"`
			OutputTokens int     `json:"outputTokens"`
			TotalCostUSD float64 `json:"totalCost"`
		} `json:"cost"`
	} `json:"info"`
}

// sseSessionUpdatedProperties wraps session info in session.updated events.
type sseSessionUpdatedProperties struct {
	Info struct {
		Error string `json:"error,omitempty"`
	} `json:"info"`
}

// sseSessionStatusProperties represents session.status events.
type sseSessionStatusProperties struct {
	SessionID string `json:"sessionID"`
	Status    struct {
		Type string `json:"type"`
	} `json:"status"`
}

// normalizeSSEEvent converts an opencode SSE event JSON to normalized Events.
func normalizeSSEEvent(raw []byte) []driver.Event {
	var evt sseEvent
	if err := json.Unmarshal(raw, &evt); err != nil {
		return nil
	}

	now := time.Now()

	switch evt.Type {
	case "message.part.updated":
		var props ssePartProperties
		_ = json.Unmarshal(evt.Properties, &props)
		return normalizePartEvent(props.Part, now)

	case "message.updated":
		var props sseMessageProperties
		_ = json.Unmarshal(evt.Properties, &props)
		cost := props.Info.Cost
		if cost.InputTokens > 0 || cost.OutputTokens > 0 || cost.TotalCostUSD > 0 {
			return []driver.Event{{
				Type:      driver.EventTypeCostUpdate,
				Timestamp: now,
				Agent:     agent,
				Cost: &driver.CostInfo{
					InputTokens:  cost.InputTokens,
					OutputTokens: cost.OutputTokens,
					TotalCostUSD: cost.TotalCostUSD,
				},
			}}
		}
		return nil

	case "session.status":
		var props sseSessionStatusProperties
		_ = json.Unmarshal(evt.Properties, &props)
		if props.Status.Type == "idle" {
			return []driver.Event{{
				Type:      driver.EventTypeTurnComplete,
				Timestamp: now,
				Agent:     agent,
			}}
		}
		return nil

	case "session.idle":
		return []driver.Event{{
			Type:      driver.EventTypeTurnComplete,
			Timestamp: now,
			Agent:     agent,
		}}

	case "session.updated":
		var props sseSessionUpdatedProperties
		_ = json.Unmarshal(evt.Properties, &props)
		if props.Info.Error != "" {
			return []driver.Event{{
				Type:      driver.EventTypeError,
				Timestamp: now,
				Agent:     agent,
				Error:     props.Info.Error,
			}}
		}
		return nil
	}

	return nil
}

// normalizePartEvent converts a message.part.updated part to driver events.
func normalizePartEvent(part ssePart, now time.Time) []driver.Event {
	switch part.Type {
	case "text":
		return []driver.Event{{
			Type:      driver.EventTypeMessage,
			Timestamp: now,
			Agent:     agent,
			Text:      part.Text,
			Delta:     true,
		}}

	case "tool":
		switch part.State.Status {
		case "running":
			return []driver.Event{{
				Type:      driver.EventTypeToolStart,
				Timestamp: now,
				Agent:     agent,
				ToolName:  part.Tool,
				ToolID:    part.CallID,
				ToolInput: part.State.Input,
			}}
		case "completed", "error":
			return []driver.Event{{
				Type:      driver.EventTypeToolResult,
				Timestamp: now,
				Agent:     agent,
				ToolID:    part.CallID,
				ToolError: part.State.Status == "error",
			}}
		}

	case "reasoning":
		return []driver.Event{{
			Type:      driver.EventTypeThinking,
			Timestamp: now,
			Agent:     agent,
			Text:      part.Text,
		}}

	case "step-finish":
		if part.Cost > 0 || part.Tokens != nil {
			evt := driver.Event{
				Type:      driver.EventTypeCostUpdate,
				Timestamp: now,
				Agent:     agent,
				Cost:      &driver.CostInfo{TotalCostUSD: part.Cost},
			}
			if part.Tokens != nil {
				evt.Cost.InputTokens = part.Tokens.Input
				evt.Cost.OutputTokens = part.Tokens.Output
			}
			return []driver.Event{evt}
		}
	}

	return nil
}
