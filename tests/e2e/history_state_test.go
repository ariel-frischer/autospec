//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestE2E_HistoryReadsConfiguredStateDir verifies that `history` reads the
// same state_dir workflow commands write to, including project-local state.
func TestE2E_HistoryReadsConfiguredStateDir(t *testing.T) {
	// HOME is the E2E temp root, so ".autospec/state" would coincide with the
	// old hard-coded ~/.autospec/state; use locations that differ from it.
	tests := map[string]struct {
		stateDir func(root string) string
	}{
		"relative project state_dir": {stateDir: func(string) string { return "./project-state" }},
		"absolute state_dir":         {stateDir: func(root string) string { return filepath.Join(root, "custom-state") }},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)
			setupWithConstitutionAndGit(env)
			stateDir := tt.stateDir(env.TempDir())
			config := "agent_preset: claude\nspecs_dir: specs\nmax_retries: 1\nskip_preflight: false\nstate_dir: " + stateDir + "\n"
			require.NoError(t, os.WriteFile(filepath.Join(env.TempDir(), ".autospec", "config.yml"), []byte(config), 0o644))

			specify := env.Run("specify", "History readback feature")
			require.Equal(t, 0, specify.ExitCode, "stdout=%s\nstderr=%s", specify.Stdout, specify.Stderr)

			result := env.Run("history")

			require.Equal(t, 0, result.ExitCode, "stdout=%s\nstderr=%s", result.Stdout, result.Stderr)
			require.Contains(t, result.Stdout, "specify")
			require.Contains(t, result.Stdout, "completed")
		})
	}
}
