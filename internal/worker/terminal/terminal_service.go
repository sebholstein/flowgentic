package terminal

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty/v2"
	"github.com/google/uuid"
)

// Session represents a running PTY session.
type Session struct {
	ID       string
	ptmx     *os.File
	cmd      *exec.Cmd
	done     chan struct{}
	exitCode int
}

// TerminalService manages PTY sessions on the worker.
type TerminalService struct {
	log      *slog.Logger
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewTerminalService creates a new TerminalService.
func NewTerminalService(log *slog.Logger) *TerminalService {
	return &TerminalService{
		log:      log,
		sessions: make(map[string]*Session),
	}
}

// Create starts a new PTY session.
func (s *TerminalService) Create(cwd string, cols, rows uint32, shell string, env []string) (string, error) {
	if shell == "" {
		shell = os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
	}

	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 24
	}

	cmd := exec.Command(shell)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), env...)

	winsize := &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	}

	ptmx, err := pty.StartWithSize(cmd, winsize)
	if err != nil {
		return "", fmt.Errorf("start pty: %w", err)
	}

	id := uuid.New().String()
	sess := &Session{
		ID:   id,
		ptmx: ptmx,
		cmd:  cmd,
		done: make(chan struct{}),
	}

	// Wait for the process to exit in the background.
	go func() {
		defer close(sess.done)
		err := cmd.Wait()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				sess.exitCode = exitErr.ExitCode()
			} else {
				sess.exitCode = -1
			}
		}
		s.log.Debug("terminal session exited", "terminal_id", id, "exit_code", sess.exitCode)
	}()

	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()

	s.log.Info("terminal session created", "terminal_id", id, "shell", shell, "cwd", cwd)
	return id, nil
}

// Destroy terminates a PTY session.
func (s *TerminalService) Destroy(terminalID string) error {
	s.mu.Lock()
	sess, ok := s.sessions[terminalID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("terminal session not found: %s", terminalID)
	}
	delete(s.sessions, terminalID)
	s.mu.Unlock()

	// Try graceful shutdown first.
	if sess.cmd.Process != nil {
		_ = sess.cmd.Process.Signal(syscall.SIGHUP)

		select {
		case <-sess.done:
			// Process exited gracefully.
		case <-time.After(2 * time.Second):
			// Force kill.
			_ = sess.cmd.Process.Kill()
			<-sess.done
		}
	}

	_ = sess.ptmx.Close()
	s.log.Info("terminal session destroyed", "terminal_id", terminalID)
	return nil
}

// Resize changes the terminal dimensions for an active session.
func (s *TerminalService) Resize(terminalID string, cols, rows uint32) error {
	sess, err := s.get(terminalID)
	if err != nil {
		return err
	}

	return pty.Setsize(sess.ptmx, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
}

// Write writes data to the PTY stdin.
func (s *TerminalService) Write(terminalID string, data []byte) (int, error) {
	sess, err := s.get(terminalID)
	if err != nil {
		return 0, err
	}

	return sess.ptmx.Write(data)
}

// Reader returns the PTY file for reading and the done channel.
func (s *TerminalService) Reader(terminalID string) (io.Reader, <-chan struct{}, error) {
	sess, err := s.get(terminalID)
	if err != nil {
		return nil, nil, err
	}

	return sess.ptmx, sess.done, nil
}

func (s *TerminalService) get(terminalID string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[terminalID]
	if !ok {
		return nil, fmt.Errorf("terminal session not found: %s", terminalID)
	}
	return sess, nil
}
