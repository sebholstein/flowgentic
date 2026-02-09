package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sebastianm/flowgentic/internal/worker/driver"
)

// DriverDeps are the dependencies for the Codex driver.
type DriverDeps struct {
	Log *slog.Logger
}

// Driver implements driver.Driver for Codex.
type Driver struct {
	log *slog.Logger

	mu       sync.Mutex
	sessions map[string]*codexSession
}

// NewDriver creates a new Codex driver.
func NewDriver(deps DriverDeps) *Driver {
	return &Driver{
		log:      deps.Log.With("driver", agent),
		sessions: make(map[string]*codexSession),
	}
}

func (d *Driver) Agent() string { return agent }

func (d *Driver) Capabilities() driver.Capabilities {
	return driver.Capabilities{
		Agent: agent,
		Supported: []driver.Capability{
			driver.CapStreaming,
			driver.CapSessionResume,
			driver.CapCustomModel,
			driver.CapYolo,
		},
	}
}

func (d *Driver) Launch(ctx context.Context, opts driver.LaunchOpts, onEvent driver.EventCallback) (driver.Session, error) {
	sess := &codexSession{
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
		return fmt.Errorf("codex session not found: %s", sessionID)
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
		d.log.Debug("unhandled codex hook", "hook", event.HookName)
	}

	return nil
}

func (d *Driver) removeSession(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.sessions, id)
}

// codexSession represents a running Codex session.
type codexSession struct {
	info    driver.SessionInfo
	driver  *Driver
	onEvent driver.EventCallback
	done    chan struct{}
	cmd     *exec.Cmd
	mu      sync.Mutex
}

func (s *codexSession) Info() driver.SessionInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.info
}

func (s *codexSession) Stop(_ context.Context) error {
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

func (s *codexSession) Wait(_ context.Context) error {
	<-s.done
	return nil
}

func (s *codexSession) closeDone() {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
}

func (s *codexSession) setStatus(status driver.SessionStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.info.Status = status
}

func (s *codexSession) SetAgentSessionID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.info.AgentSessionID = id
}

func (s *codexSession) emit(events ...driver.Event) {
	if s.onEvent == nil {
		return
	}
	for _, e := range events {
		s.onEvent(e)
	}
}

func (s *codexSession) launchHeadless(ctx context.Context, opts driver.LaunchOpts) error {
	args := []string{"exec", "--json"}

	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.Yolo {
		args = append(args, "--full-auto")
	}

	if opts.SessionID != "" {
		args = append(args, "resume", opts.SessionID)
	}
	if opts.Prompt != "" {
		args = append(args, opts.Prompt)
	}

	cmd := exec.CommandContext(ctx, "codex", args...)
	if opts.Cwd != "" {
		cmd.Dir = opts.Cwd
	}
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start codex: %w", err)
	}

	s.mu.Lock()
	s.cmd = cmd
	s.info.Status = driver.SessionStatusRunning
	s.mu.Unlock()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// Read lines synchronously until we get the thread.started event
	// so the agent session ID is available before Launch returns.
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var evt codexEvent
		if json.Unmarshal([]byte(line), &evt) == nil && evt.Type == "thread.started" && evt.ThreadID != "" {
			s.SetAgentSessionID(evt.ThreadID)
		}

		events := normalizeCodexEvent([]byte(line))
		s.emit(events...)

		// Once we've seen thread.started, hand off to the background goroutine.
		if evt.Type == "thread.started" {
			break
		}
	}

	go func() {
		defer s.closeDone()
		defer s.driver.removeSession(s.info.ID)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			events := normalizeCodexEvent([]byte(line))
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

// ResolveSessionID discovers the Codex session ID by scanning the
// sessions directory for the newest file.
func (d *Driver) ResolveSessionID(_ context.Context, cwd string) (string, error) {
	_ = cwd // Codex sessions dir is global, not per-cwd.

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}

	now := time.Now()
	sessDir := filepath.Join(homeDir, ".codex", "sessions",
		fmt.Sprintf("%d", now.Year()),
		fmt.Sprintf("%02d", now.Month()),
		fmt.Sprintf("%02d", now.Day()),
	)

	entries, err := os.ReadDir(sessDir)
	if err != nil {
		return "", fmt.Errorf("read sessions dir %s: %w", sessDir, err)
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("no session files found in %s", sessDir)
	}

	// Sort by name descending to find the newest rollout file.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() > entries[j].Name()
	})

	// File pattern: rollout-<ISO-timestamp>-<uuid>.jsonl
	// Example: rollout-2026-02-09T13-08-42-019c424d-d3da-7903-8b5f-32b3d4a3a436.jsonl
	// The UUID is always the last 36 characters before .jsonl (8-4-4-4-12 format).
	name := entries[0].Name()
	name = strings.TrimSuffix(name, ".jsonl")
	if !strings.HasPrefix(name, "rollout-") {
		return "", fmt.Errorf("unexpected session filename: %s", entries[0].Name())
	}

	const uuidLen = 36 // 8-4-4-4-12
	if len(name) < uuidLen {
		return "", fmt.Errorf("filename too short to contain UUID: %s", entries[0].Name())
	}

	return name[len(name)-uuidLen:], nil
}
