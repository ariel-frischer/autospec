//go:build e2e

package e2e

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

// signalExitBound is how long autospec may take to exit after one signal. It
// must finish reaping the agent group well before shutdown.ForceExitDelay.
const signalExitBound = 10 * time.Second

// TestE2E_SignalCleansUpAgentProcessGroup verifies SIGINT/SIGTERM during a
// running stage exit with 128+signal and kill agent descendants.
func TestE2E_SignalCleansUpAgentProcessGroup(t *testing.T) {
	tests := map[string]struct {
		sig      syscall.Signal
		wantExit int
	}{
		"SIGINT exits 130":  {sig: syscall.SIGINT, wantExit: 130},
		"SIGTERM exits 143": {sig: syscall.SIGTERM, wantExit: 143},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)
			setupWithConstitutionAndGit(env)
			pidFile := filepath.Join(env.TempDir(), "grandchild.pid")
			env.SetEnv("MOCK_SPAWN_CHILD_PIDFILE", pidFile)

			proc := env.Start("specify", "Signal cleanup")
			grandchild := waitForPIDFile(t, pidFile, proc)
			require.True(t, processAlive(grandchild), "grandchild should be running before the signal")

			require.NoError(t, proc.Signal(tt.sig))
			result, exited := proc.Wait(signalExitBound)

			require.True(t, exited, "autospec did not exit within %s of %s", signalExitBound, tt.sig)
			require.Equal(t, tt.wantExit, result.ExitCode, "stdout=%s\nstderr=%s", result.Stdout, result.Stderr)
			require.Eventually(t, func() bool { return !processAlive(grandchild) }, 2*time.Second, 20*time.Millisecond,
				"agent grandchild %d survived autospec exit", grandchild)
		})
	}
}

func waitForPIDFile(t *testing.T, path string, proc *testutil.AsyncCommand) int {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(path); err == nil {
			pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
			require.NoError(t, err, "parsing grandchild pid %q", data)
			t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
			return pid
		}
		if result, exited := proc.Wait(50 * time.Millisecond); exited {
			t.Fatalf("autospec exited (%d) before the agent started: stdout=%s\nstderr=%s", result.ExitCode, result.Stdout, result.Stderr)
		}
	}
	t.Fatalf("agent never wrote grandchild pid file %s", path)
	return 0
}

// processAlive reports whether pid exists and, where procfs is available, is
// not a zombie awaiting reaping.
func processAlive(pid int) bool {
	if errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) {
		return false
	}
	stat, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return true
	}
	fields := strings.Fields(string(stat))
	return len(fields) < 3 || fields[2] != "Z"
}
