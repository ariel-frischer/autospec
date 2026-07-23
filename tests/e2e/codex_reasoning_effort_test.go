//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestE2E_CodexReasoningEffortPrecedence(t *testing.T) {
	tests := map[string]struct {
		config       string
		command      []string
		setupStage   func(*testutil.E2EEnv)
		wantEffort   string
		unwantEffort string
	}{
		"specify uses top-level effort": {
			config:     "reasoning_effort: low\n",
			command:    []string{"specify", "Test Codex effort"},
			setupStage: func(*testutil.E2EEnv) {},
			wantEffort: "low",
		},
		"plan stage effort overrides top-level": {
			config:       "reasoning_effort: low\nreasoning_efforts:\n  plan: high\n",
			command:      []string{"plan"},
			setupStage:   func(env *testutil.E2EEnv) { env.SetupSpec("001-test-feature") },
			wantEffort:   "high",
			unwantEffort: "low",
		},
		"tasks CLI effort overrides stage and top-level": {
			config:       "reasoning_effort: low\nreasoning_efforts:\n  tasks: high\n",
			command:      []string{"tasks", "-e", "xhigh"},
			setupStage:   func(env *testutil.E2EEnv) { env.SetupPlan("001-test-feature") },
			wantEffort:   "xhigh",
			unwantEffort: "high",
		},
		"implement uses stage effort": {
			config:     "reasoning_effort: medium\nreasoning_efforts:\n  implement: max\n",
			command:    []string{"implement"},
			setupStage: func(env *testutil.E2EEnv) { env.SetupTasks("001-test-feature") },
			wantEffort: "max",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnvWithAgent(t, testutil.AgentCodex)
			setupWithConstitutionAndGit(env)
			writeReasoningConfig(t, env, "codex", tt.config)
			tt.setupStage(env)
			callLog := filepath.Join(env.TempDir(), "codex-calls.log")
			env.SetMockCallLog(callLog)

			result := env.Run(tt.command...)

			require.Equal(t, 0, result.ExitCode, "stdout=%s\nstderr=%s", result.Stdout, result.Stderr)
			logContent, err := os.ReadFile(callLog)
			require.NoError(t, err)
			require.Contains(t, string(logContent), "agent: codex")
			require.Contains(t, string(logContent), "model_reasoning_effort="+tt.wantEffort)
			if tt.unwantEffort != "" {
				require.NotContains(t, string(logContent), "model_reasoning_effort="+tt.unwantEffort)
			}
		})
	}
}

func TestE2E_ReasoningEffortDoesNotChangeNonCodexArgv(t *testing.T) {
	tests := map[string]struct {
		agent testutil.AgentPreset
	}{
		"claude receives no Codex reasoning argument": {agent: testutil.AgentClaude},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnvWithAgent(t, tt.agent)
			setupWithConstitutionAndGit(env)
			writeReasoningConfig(t, env, string(tt.agent), "reasoning_effort: high\n")
			callLog := filepath.Join(env.TempDir(), "agent-calls.log")
			env.SetMockCallLog(callLog)

			result := env.Run("specify", "Test non-Codex effort")

			require.Equal(t, 0, result.ExitCode, "stdout=%s\nstderr=%s", result.Stdout, result.Stderr)
			logContent, err := os.ReadFile(callLog)
			require.NoError(t, err)
			require.NotContains(t, string(logContent), "model_reasoning_effort=")
		})
	}
}

func writeReasoningConfig(t *testing.T, env *testutil.E2EEnv, agent, reasoningConfig string) {
	t.Helper()
	content := fmt.Sprintf("agent_preset: %s\nspecs_dir: specs\nmax_retries: 1\nskip_preflight: false\ncodex_output:\n  mode: full\n%s", agent, reasoningConfig)
	err := os.WriteFile(filepath.Join(env.TempDir(), ".autospec", "config.yml"), []byte(content), 0o644)
	require.NoError(t, err)
}
