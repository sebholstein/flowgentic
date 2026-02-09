package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
)

// DriverDeps are the dependencies for the Claude Code driver.
type DriverDeps struct {
	Log *slog.Logger
}

// Driver implements driver.Driver for Claude Code.
type Driver struct {
	log *slog.Logger

	mu       sync.Mutex
	sessions map[string]*claudeSession
}

// NewDriver creates a new Claude Code driver.
func NewDriver(deps DriverDeps) *Driver {
	return &Driver{
		log:      deps.Log.With("driver", agent),
		sessions: make(map[string]*claudeSession),
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

	sess := &claudeSession{
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
		return fmt.Errorf("claude session not found: %s", sessionID)
	}

	return sess.handleHook(event)
}

func (d *Driver) removeSession(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.sessions, id)
}

// claudeSession represents a running Claude Code session.
type claudeSession struct {
	info    driver.SessionInfo
	driver  *Driver
	onEvent driver.EventCallback
	done    chan struct{}
	cmd     *exec.Cmd
	mu      sync.Mutex
}

func (s *claudeSession) Info() driver.SessionInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.info
}

func (s *claudeSession) Stop(_ context.Context) error {
	s.mu.Lock()
	s.info.Status = driver.SessionStatusStopping
	cmd := s.cmd
	s.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}

	s.mu.Lock()
	s.info.Status = driver.SessionStatusStopped
	s.mu.Unlock()

	s.closeDone()
	return nil
}

func (s *claudeSession) Wait(_ context.Context) error {
	<-s.done
	return nil
}

func (s *claudeSession) closeDone() {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
}

func (s *claudeSession) setStatus(status driver.SessionStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.info.Status = status
}

func (s *claudeSession) emit(events ...driver.Event) {
	if s.onEvent == nil {
		return
	}
	for _, e := range events {
		s.onEvent(e)
	}
}

// launchHeadless spawns claude in headless stream-json mode.
func (s *claudeSession) launchHeadless(ctx context.Context, opts driver.LaunchOpts) error {
	flags := buildFlags(opts, s.info.AgentSessionID)
	cmd := exec.CommandContext(ctx, "claude", flags...)
	if opts.Cwd != "" {
		cmd.Dir = opts.Cwd
	}
	if opts.Prompt != "" {
		cmd.Stdin = strings.NewReader(opts.Prompt)
	}
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start claude: %w", err)
	}

	s.mu.Lock()
	s.cmd = cmd
	s.info.Status = driver.SessionStatusRunning
	s.mu.Unlock()

	// Parse JSONL output in background.
	go func() {
		defer s.closeDone()
		defer s.driver.removeSession(s.info.ID)

		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var resp StreamingResponse
			if err := json.Unmarshal([]byte(line), &resp); err != nil {
				s.driver.log.Warn("parse claude output", "error", err, "line", line)
				continue
			}

			resp.SessionID = s.info.ID
			events := normalizeStreamingResponse(resp)
			s.emit(events...)
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

// handleHook processes a hook event from the hookctl binary.
func (s *claudeSession) handleHook(event driver.HookEvent) error {
	now := time.Now()

	switch event.HookName {
	case "Stop":
		s.setStatus(driver.SessionStatusIdle)
		s.emit(driver.Event{
			Type:      driver.EventTypeTurnComplete,
			Timestamp: now,
			Agent:     agent,
			Raw:       event.Payload,
		})

	case "SessionStart":
		s.emit(driver.Event{
			Type:      driver.EventTypeSessionStart,
			Timestamp: now,
			Agent:     agent,
			Raw:       event.Payload,
		})

	case "PreToolUse":
		var payload struct {
			ToolName string          `json:"tool_name"`
			ToolID   string          `json:"tool_id"`
			Input    json.RawMessage `json:"input"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err == nil {
			s.emit(driver.Event{
				Type:      driver.EventTypeToolStart,
				Timestamp: now,
				Agent:     agent,
				ToolName:  payload.ToolName,
				ToolID:    payload.ToolID,
				ToolInput: payload.Input,
				Raw:       event.Payload,
			})
		}

	case "PostToolUse":
		var payload struct {
			ToolID  string `json:"tool_id"`
			IsError bool   `json:"is_error"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err == nil {
			s.emit(driver.Event{
				Type:      driver.EventTypeToolResult,
				Timestamp: now,
				Agent:     agent,
				ToolID:    payload.ToolID,
				ToolError: payload.IsError,
				Raw:       event.Payload,
			})
		}

	default:
		s.driver.log.Debug("unhandled claude hook", "hook", event.HookName)
	}

	return nil
}

// buildFlags constructs CLI flags for headless mode.
func buildFlags(opts driver.LaunchOpts, agentSessionID string) []string {
	var flags []string

	if opts.Model != "" {
		flags = append(flags, "--model", opts.Model)
	}
	if opts.SystemPrompt != "" {
		flags = append(flags, "--system-prompt", opts.SystemPrompt)
	}
	if len(opts.AllowedTools) > 0 {
		flags = append(flags, "--allowed-tools", strings.Join(opts.AllowedTools, ","))
	}

	flags = append(flags, "--output-format", "stream-json")

	if opts.Yolo {
		flags = append(flags, "--dangerously-skip-permissions")
	}
	if opts.SessionID != "" {
		flags = append(flags, "--resume", "--session-id", opts.SessionID)
	} else if agentSessionID != "" {
		flags = append(flags, "--session-id", agentSessionID)
	}
	if opts.Cwd != "" {
		flags = append(flags, "--add-dir", opts.Cwd)
	}

	return flags
}
