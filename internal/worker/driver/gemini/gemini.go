package gemini

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/sebastianm/flowgentic/internal/worker/driver"
)

const agent = "gemini"

// DriverDeps are the dependencies for the Gemini driver.
type DriverDeps struct {
	Log *slog.Logger
}

// Driver implements driver.Driver for Gemini CLI.
type Driver struct {
	log *slog.Logger

	mu       sync.Mutex
	sessions map[string]*geminiSession
}

// NewDriver creates a new Gemini driver.
func NewDriver(deps DriverDeps) *Driver {
	return &Driver{
		log:      deps.Log.With("driver", agent),
		sessions: make(map[string]*geminiSession),
	}
}

func (d *Driver) Agent() string { return agent }

func (d *Driver) Capabilities() driver.Capabilities {
	return driver.Capabilities{
		Agent: agent,
		Supported: []driver.Capability{
			driver.CapCustomModel,
			driver.CapYolo,
		},
	}
}

func (d *Driver) Launch(ctx context.Context, opts driver.LaunchOpts, onEvent driver.EventCallback) (driver.Session, error) {
	sess := &geminiSession{
		info: driver.SessionInfo{
			ID:        opts.SessionID,
			AgentID:   agent,
			Status:    driver.SessionStatusStarting,
			Mode:      opts.Mode,
			Cwd:       opts.Cwd,
			StartedAt: time.Now(),
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
		return fmt.Errorf("gemini session not found: %s", sessionID)
	}

	now := time.Now()

	switch event.HookName {
	case "AfterAgent", "Stop":
		sess.setStatus(driver.SessionStatusIdle)
		sess.emit(driver.Event{
			Type:      driver.EventTypeTurnComplete,
			Timestamp: now,
			Agent:     agent,
			Raw:       event.Payload,
		})
	default:
		d.log.Debug("unhandled gemini hook", "hook", event.HookName)
	}

	return nil
}

func (d *Driver) removeSession(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.sessions, id)
}

// ResolveSessionID discovers the Gemini session ID by running
// `gemini --list-sessions` and parsing the most recent session's UUID.
func (d *Driver) ResolveSessionID(ctx context.Context, cwd string) (string, error) {
	cmd := exec.CommandContext(ctx, "gemini", "--list-sessions")
	if cwd != "" {
		cmd.Dir = cwd
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gemini --list-sessions: %w: %s", err, stderr.String())
	}

	return parseLatestSessionID(stdout.String())
}

// parseLatestSessionID extracts the UUID from the first (most recent) session
// in `gemini --list-sessions` output. Each line looks like:
//
//  1. Description (age) [uuid]
func parseLatestSessionID(output string) (string, error) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Extract UUID from brackets at end of line: [<uuid>]
		start := strings.LastIndex(line, "[")
		end := strings.LastIndex(line, "]")
		if start >= 0 && end > start+1 {
			return line[start+1 : end], nil
		}
	}
	return "", fmt.Errorf("no session UUID found in gemini --list-sessions output")
}

// geminiSession represents a running Gemini session.
type geminiSession struct {
	info    driver.SessionInfo
	driver  *Driver
	onEvent driver.EventCallback
	done    chan struct{}
	cmd     *exec.Cmd
	mu      sync.Mutex
}

func (s *geminiSession) Info() driver.SessionInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.info
}

func (s *geminiSession) Stop(_ context.Context) error {
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

func (s *geminiSession) Wait(_ context.Context) error {
	<-s.done
	return nil
}

func (s *geminiSession) closeDone() {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
}

func (s *geminiSession) setStatus(status driver.SessionStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.info.Status = status
}

func (s *geminiSession) SetAgentSessionID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.info.AgentSessionID = id
}

func (s *geminiSession) emit(events ...driver.Event) {
	if s.onEvent == nil {
		return
	}
	for _, e := range events {
		s.onEvent(e)
	}
}

func (s *geminiSession) launchHeadless(ctx context.Context, opts driver.LaunchOpts) error {
	args := []string{}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.Yolo {
		args = append(args, "--yolo")
	}
	if opts.Prompt != "" {
		args = append(args, "-p", opts.Prompt)
	}

	cmd := exec.CommandContext(ctx, "gemini", args...)
	if opts.Cwd != "" {
		cmd.Dir = opts.Cwd
	}
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start gemini: %w", err)
	}

	s.mu.Lock()
	s.cmd = cmd
	s.info.Status = driver.SessionStatusRunning
	s.mu.Unlock()

	go func() {
		defer s.closeDone()
		defer s.driver.removeSession(s.info.ID)

		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		var buf strings.Builder
		for scanner.Scan() {
			line := scanner.Text()
			buf.WriteString(line)
			buf.WriteString("\n")
		}

		output := buf.String()
		if output != "" {
			s.emit(driver.Event{
				Type:      driver.EventTypeMessage,
				Timestamp: time.Now(),
				Agent:     agent,
				Text:      output,
			})
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
			s.emit(driver.Event{
				Type:      driver.EventTypeTurnComplete,
				Timestamp: time.Now(),
				Agent:     agent,
			})
		}
	}()

	return nil
}
