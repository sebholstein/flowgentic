package terminal

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) *TerminalService {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewTerminalService(log)
}

func TestCreate(t *testing.T) {
	svc := newTestService(t)

	id, err := svc.Create(t.TempDir(), 120, 40, "/bin/sh", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	// Cleanup.
	require.NoError(t, svc.Destroy(id))
}

func TestCreate_DefaultShell(t *testing.T) {
	svc := newTestService(t)

	id, err := svc.Create(t.TempDir(), 0, 0, "", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	require.NoError(t, svc.Destroy(id))
}

func TestCreate_InvalidCwd(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Create("/nonexistent/path/that/does/not/exist", 80, 24, "/bin/sh", nil)
	require.Error(t, err)
}

func TestDestroy(t *testing.T) {
	svc := newTestService(t)

	id, err := svc.Create(t.TempDir(), 80, 24, "/bin/sh", nil)
	require.NoError(t, err)

	require.NoError(t, svc.Destroy(id))

	// Destroying again should fail.
	err = svc.Destroy(id)
	assert.Error(t, err)
}

func TestDestroy_NotFound(t *testing.T) {
	svc := newTestService(t)
	err := svc.Destroy("nonexistent-id")
	assert.Error(t, err)
}

func TestResize(t *testing.T) {
	svc := newTestService(t)

	id, err := svc.Create(t.TempDir(), 80, 24, "/bin/sh", nil)
	require.NoError(t, err)
	defer svc.Destroy(id) //nolint:errcheck

	err = svc.Resize(id, 120, 40)
	require.NoError(t, err)
}

func TestResize_NotFound(t *testing.T) {
	svc := newTestService(t)
	err := svc.Resize("nonexistent-id", 80, 24)
	assert.Error(t, err)
}

func TestWriteAndRead(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PTY tests not supported on Windows")
	}

	svc := newTestService(t)

	id, err := svc.Create(t.TempDir(), 80, 24, "/bin/sh", nil)
	require.NoError(t, err)
	defer svc.Destroy(id) //nolint:errcheck

	reader, _, err := svc.Reader(id)
	require.NoError(t, err)

	// Write a command.
	_, err = svc.Write(id, []byte("echo hello\n"))
	require.NoError(t, err)

	// Read output until we find "hello".
	buf := make([]byte, 4096)
	var output bytes.Buffer
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		if f, ok := reader.(*os.File); ok {
			_ = f.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		}
		n, rerr := reader.Read(buf)
		if n > 0 {
			output.Write(buf[:n])
		}
		if strings.Contains(output.String(), "hello") {
			break
		}
		if rerr != nil {
			break
		}
	}

	assert.Contains(t, output.String(), "hello")
}

func TestExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PTY tests not supported on Windows")
	}

	svc := newTestService(t)

	id, err := svc.Create(t.TempDir(), 80, 24, "/bin/sh", nil)
	require.NoError(t, err)

	reader, done, err := svc.Reader(id)
	require.NoError(t, err)

	// Drain PTY output in background to prevent buffer blocking.
	go func() {
		_, _ = io.Copy(io.Discard, reader)
	}()

	// Send exit with specific code.
	_, err = svc.Write(id, []byte("exit 42\n"))
	require.NoError(t, err)

	// Wait for process to finish.
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for process to exit")
	}

	// Get session and check exit code.
	sess, err := svc.get(id)
	require.NoError(t, err)
	assert.Equal(t, 42, sess.exitCode)

	// Cleanup.
	svc.mu.Lock()
	delete(svc.sessions, id)
	svc.mu.Unlock()
}
