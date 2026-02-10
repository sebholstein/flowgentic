package procutil_test

import (
	"os/exec"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/sebastianm/flowgentic/internal/worker/procutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartWithCleanup_ChildDiesWhenKilled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses unix sleep command")
	}

	cmd := exec.Command("sleep", "60")
	require.NoError(t, procutil.StartWithCleanup(cmd))

	pid := cmd.Process.Pid
	assert.True(t, processExists(pid), "child should be alive after start")

	// Kill the child directly (simulates what the OS would do).
	require.NoError(t, cmd.Process.Kill())
	_ = cmd.Wait()

	time.Sleep(100 * time.Millisecond)
	assert.False(t, processExists(pid), "child should be dead after kill")
}

func processExists(pid int) bool {
	out, err := exec.Command("kill", "-0", strconv.Itoa(pid)).CombinedOutput()
	_ = out
	return err == nil
}
