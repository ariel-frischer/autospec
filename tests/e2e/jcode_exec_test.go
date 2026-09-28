//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestE2E_JcodeExecArgv verifies the default jcode exec runner contract:
// wrapper flags and --model precede `run`, stage models beat the top-level
// model, reasoning effort is not forwarded, and preflight needs no Claude skills.
func TestE2E_JcodeExecArgv(t *testing.T) {
	tests := map[string]struct {
		config     string
		command    []string
		setupStage func(*testutil.E2EEnv)
		wantModel  string
	}{
		"specify uses top-level model": {
			config:     "model: top-model\n",
			command:    []string{"specify", "Test jcode argv"},
			setupStage: func(*testutil.E2EEnv) {},
			wantModel:  "top-model",
		},
		"plan stage model overrides top-level model": {
			config:     "model: top-model\nmodels:\n  plan: plan-model\n",
			command:    []string{"plan"},
			setupStage: func(env *testutil.E2EEnv) { env.SetupSpec("001-test-feature") },
			wantModel:  "plan-model",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnvWithAgent(t, testutil.AgentJcode)
			setupWithConstitutionAndGit(env)
			writeProjectConfig(t, env, "agent_preset: jcode\nspecs_dir: specs\nmax_retries: 1\nskip_preflight: false\nreasoning_effort: xhigh\n"+tt.config)
			tt.setupStage(env)
			callLog := filepath.Join(env.TempDir(), "jcode-calls.log")
			env.SetMockCallLog(callLog)

			result := env.Run(tt.command...)

			require.Equal(t, 0, result.ExitCode, "stdout=%s\nstderr=%s", result.Stdout, result.Stderr)
			assertJcodeExecArgv(t, singleAgentInvocation(t, callLog, "jcode").args, tt.wantModel)
			_, err := os.Stat(filepath.Join(env.TempDir(), ".claude", "skills"))
			require.ErrorIs(t, err, os.ErrNotExist, "jcode preflight must not require .claude/skills")
		})
	}
}

func assertJcodeExecArgv(t *testing.T, args []string, wantModel string) {
	t.Helper()
	require.GreaterOrEqual(t, len(args), 7, "argv=%q", args)
	require.Equal(t, []string{"--quiet", "--no-update", "--no-selfdev"}, args[:3], "argv=%q", args)

	runIdx := len(args) - 2
	require.Equal(t, "run", args[runIdx], "prompt must be the single argument after run; argv=%q", args)
	model, modelIdx := flagValue(args, "--model")
	require.Equal(t, wantModel, model, "argv=%q", args[:runIdx])
	require.Less(t, modelIdx, runIdx, "--model must precede run")

	for _, arg := range args[:runIdx] {
		require.NotContains(t, arg, "xhigh", "reasoning effort must not be forwarded in exec mode")
		require.False(t, strings.Contains(arg, "reasoning") || strings.Contains(arg, "effort"),
			"unexpected effort flag %q in argv %q", arg, args[:runIdx])
	}
}
