package opencode

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
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
	info      driver.SessionInfo
	driver    *Driver
	onEvent   driver.EventCallback
	done      chan struct{}
	serverCmd *exec.Cmd
	serverURL string
	mu        sync.Mutex
}

func (s *openCodeSession) Info() driver.SessionInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.info
}

func (s *openCodeSession) Stop(_ context.Context) error {
	s.mu.Lock()
	s.info.Status = driver.SessionStatusStopping
	serverCmd := s.serverCmd
	s.mu.Unlock()

	if serverCmd != nil && serverCmd.Process != nil {
		_ = serverCmd.Process.Kill()
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

// launchHeadless starts `opencode serve` and connects to its SSE stream.
func (s *openCodeSession) launchHeadless(ctx context.Context, opts driver.LaunchOpts) error {
	// Start the opencode server.
	args := []string{"serve"}
	if s.info.AgentSessionID != "" {
		args = append(args, "-s", s.info.AgentSessionID)
	}
	cmd := exec.CommandContext(ctx, "opencode", args...)
	if opts.Cwd != "" {
		cmd.Dir = opts.Cwd
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start opencode serve: %w", err)
	}

	s.mu.Lock()
	s.serverCmd = cmd
	s.info.Status = driver.SessionStatusRunning
	s.mu.Unlock()

	// Read server output to find the URL, then start SSE.
	go func() {
		defer s.closeDone()
		defer s.driver.removeSession(s.info.ID)

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			// Look for the server URL in the output.
			if strings.Contains(line, "http://") || strings.Contains(line, "https://") {
				url := extractURL(line)
				if url != "" {
					s.mu.Lock()
					s.serverURL = url
					s.mu.Unlock()
					s.driver.log.Info("opencode server started", "url", url)

					// Send the initial prompt if provided.
					if opts.Prompt != "" {
						if err := s.sendViaAPI(ctx, url, opts.Prompt); err != nil {
							s.driver.log.Warn("send initial prompt", "error", err)
						}
					}

					// Start SSE event stream.
					go s.consumeSSE(ctx, url)
					break
				}
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

// consumeSSE reads SSE events from the opencode server.
func (s *openCodeSession) consumeSSE(ctx context.Context, serverURL string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+"/events", nil)
	if err != nil {
		s.driver.log.Warn("create SSE request", "error", err)
		return
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.driver.log.Warn("SSE connect", "error", err)
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		events := normalizeSSEEvent([]byte(data))
		s.emit(events...)
	}
}

// sendViaAPI sends a prompt to the opencode HTTP API.
func (s *openCodeSession) sendViaAPI(ctx context.Context, serverURL string, message string) error {
	payload, _ := json.Marshal(map[string]string{"prompt": message})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL+"/session/prompt", strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("opencode API error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// extractURL finds the first http(s) URL in a string.
func extractURL(s string) string {
	for _, prefix := range []string{"https://", "http://"} {
		idx := strings.Index(s, prefix)
		if idx >= 0 {
			end := strings.IndexAny(s[idx:], " \t\n\r\"'")
			if end < 0 {
				return s[idx:]
			}
			return s[idx : idx+end]
		}
	}
	return ""
}

// sseEvent represents an SSE event from opencode.
type sseEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// normalizeSSEEvent converts an opencode SSE event JSON to normalized Events.
func normalizeSSEEvent(raw []byte) []driver.Event {
	var evt sseEvent
	if err := json.Unmarshal(raw, &evt); err != nil {
		return nil
	}

	now := time.Now()

	switch evt.Type {
	case "message":
		var data struct {
			Content string `json:"content"`
		}
		_ = json.Unmarshal(evt.Data, &data)
		return []driver.Event{{
			Type:      driver.EventTypeMessage,
			Timestamp: now,
			Agent:     agent,
			Text:      data.Content,
			Delta:     true,
		}}

	case "tool.start":
		var data struct {
			Name  string          `json:"name"`
			ID    string          `json:"id"`
			Input json.RawMessage `json:"input"`
		}
		_ = json.Unmarshal(evt.Data, &data)
		return []driver.Event{{
			Type:      driver.EventTypeToolStart,
			Timestamp: now,
			Agent:     agent,
			ToolName:  data.Name,
			ToolID:    data.ID,
			ToolInput: data.Input,
		}}

	case "tool.result":
		var data struct {
			ID      string `json:"id"`
			IsError bool   `json:"is_error"`
		}
		_ = json.Unmarshal(evt.Data, &data)
		return []driver.Event{{
			Type:      driver.EventTypeToolResult,
			Timestamp: now,
			Agent:     agent,
			ToolID:    data.ID,
			ToolError: data.IsError,
		}}

	case "turn.complete":
		return []driver.Event{{
			Type:      driver.EventTypeTurnComplete,
			Timestamp: now,
			Agent:     agent,
		}}

	case "error":
		return []driver.Event{{
			Type:      driver.EventTypeError,
			Timestamp: now,
			Agent:     agent,
			Error:     string(evt.Data),
		}}
	}

	return nil
}
