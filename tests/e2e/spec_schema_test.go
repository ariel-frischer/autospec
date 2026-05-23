//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestE2E_SpecifyOmitsRedundantTestableField(t *testing.T) {
	tests := map[string]struct {
		agent testutil.AgentPreset
	}{
		"claude mock":   {agent: testutil.AgentClaude},
		"opencode mock": {agent: testutil.AgentOpencode},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			const specName = "001-test-feature"

			env := testutil.NewE2EEnvWithAgent(t, tt.agent)
			env.InitGitRepo()
			env.CreateBranch(specName)
			env.SetupConstitution()
			env.SetupAutospecInit()

			result := env.Run("specify", "test feature", "--agent", string(tt.agent))
			require.Equal(t, 0, result.ExitCode, "specify failed\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			specPath := filepath.Join(env.SpecsDir(), specName, "spec.yaml")
			content, err := os.ReadFile(specPath)
			require.NoError(t, err, "should read generated spec.yaml")

			require.NotContains(t, string(content), "testable:")
			assertArtifactValid(t, env, specName, "spec")
		})
	}
}
