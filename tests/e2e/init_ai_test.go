//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestE2E_InitProjectWithNonClaudeAgents verifies non-interactive project init
// for Codex/jcode selects a non-Claude preset, installs shared Codex skills when
// Codex is selected, and never creates Claude skill directories.
func TestE2E_InitProjectWithNonClaudeAgents(t *testing.T) {
	tests := map[string]struct {
		ai           string
		wantPreset   string
		wantSkillDir bool
	}{
		"codex and jcode": {ai: "codex,jcode", wantPreset: "codex", wantSkillDir: true},
		"jcode only":      {ai: "jcode", wantPreset: "jcode"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)
			root := env.TempDir()

			result := env.Run("init", "--project", "--ai", tt.ai, "--no-constitution", "--no-gitignore", "--no-sandbox")

			require.Equal(t, 0, result.ExitCode, "stdout=%s\nstderr=%s", result.Stdout, result.Stderr)
			config, err := os.ReadFile(filepath.Join(root, ".autospec", "config.yml"))
			require.NoError(t, err)
			require.Contains(t, string(config), "agent_preset: "+tt.wantPreset+"\n")
			require.NoDirExists(t, filepath.Join(root, ".claude", "skills"))
			if tt.wantSkillDir {
				for _, stage := range []string{"specify", "plan", "tasks", "implement"} {
					require.FileExists(t, filepath.Join(root, ".agents", "skills", "autospec-"+stage, "SKILL.md"))
				}
			}
		})
	}
}
