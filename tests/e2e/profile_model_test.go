//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

const codexProfileBaseConfig = "agent_preset: codex\nspecs_dir: specs\nmax_retries: 1\nskip_preflight: false\ncodex_output:\n  mode: full\n"

// TestE2E_CodexModelPrecedence verifies the model forwarded to codex follows
// CLI --model > --profile overlay / models.<stage> > top-level model.
func TestE2E_CodexModelPrecedence(t *testing.T) {
	tests := map[string]struct {
		config    string
		profile   string
		args      []string
		wantModel string
	}{
		"top-level model": {
			config:    "model: top-model\n",
			args:      []string{"specify", "Model precedence"},
			wantModel: "top-model",
		},
		"stage model beats top-level model": {
			config:    "model: top-model\nmodels:\n  specify: stage-model\n",
			args:      []string{"specify", "Model precedence"},
			wantModel: "stage-model",
		},
		"CLI model beats stage and top-level model": {
			config:    "model: top-model\nmodels:\n  specify: stage-model\n",
			args:      []string{"specify", "--model", "cli-model", "Model precedence"},
			wantModel: "cli-model",
		},
		"project profile overrides project model": {
			config:    "model: top-model\n",
			profile:   "model: profile-model\n",
			args:      []string{"--profile", "fast", "specify", "Model precedence"},
			wantModel: "profile-model",
		},
		"CLI model beats project profile": {
			config:    "model: top-model\n",
			profile:   "model: profile-model\n",
			args:      []string{"--profile", "fast", "specify", "--model", "cli-model", "Model precedence"},
			wantModel: "cli-model",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnvWithAgent(t, testutil.AgentCodex)
			setupWithConstitutionAndGit(env)
			writeProjectConfig(t, env, codexProfileBaseConfig+tt.config)
			if tt.profile != "" {
				writeProjectProfile(t, env, "fast", tt.profile)
			}
			callLog := filepath.Join(env.TempDir(), "codex-calls.log")
			env.SetMockCallLog(callLog)

			result := env.Run(tt.args...)

			require.Equal(t, 0, result.ExitCode, "stdout=%s\nstderr=%s", result.Stdout, result.Stderr)
			model, _ := flagValue(singleAgentInvocation(t, callLog, "codex").args, "--model")
			require.Equal(t, tt.wantModel, model)
		})
	}
}

// TestE2E_ProfileSelectionErrors verifies invalid --profile usage fails before
// any agent runs, with a message that names the problem.
func TestE2E_ProfileSelectionErrors(t *testing.T) {
	tests := map[string]struct {
		args        []string
		wantStderr  []string
		searchPaths bool
	}{
		"profile combined with config": {
			args:       []string{"--profile", "fast", "--config", ".autospec/config.yml", "specify", "x"},
			wantStderr: []string{"--config and --profile cannot be combined"},
		},
		"unknown profile names searched paths": {
			args:        []string{"--profile", "missing", "specify", "x"},
			wantStderr:  []string{`profile "missing" not found`},
			searchPaths: true,
		},
		"path traversal name rejected": {
			args:       []string{"--profile", "../x", "specify", "x"},
			wantStderr: []string{`invalid profile name "../x"`},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnvWithAgent(t, testutil.AgentCodex)
			setupWithConstitutionAndGit(env)
			writeProjectConfig(t, env, codexProfileBaseConfig)
			writeProjectProfile(t, env, "fast", "model: profile-model\n")
			callLog := filepath.Join(env.TempDir(), "codex-calls.log")
			env.SetMockCallLog(callLog)

			result := env.Run(tt.args...)

			require.NotEqual(t, 0, result.ExitCode, "stdout=%s\nstderr=%s", result.Stdout, result.Stderr)
			output := result.Stdout + result.Stderr
			for _, want := range tt.wantStderr {
				require.Contains(t, output, want)
			}
			if tt.searchPaths {
				require.Contains(t, output, filepath.Join(env.TempDir(), ".config", "autospec", "profiles", "missing.yml"))
				require.Contains(t, output, filepath.Join(".autospec", "profiles", "missing.yml"))
			}
			require.NoFileExists(t, callLog, "agent must not run when profile selection fails")
		})
	}
}

// TestE2E_ConfigCreateProfileListsUserProfile verifies `config create NAME`
// saves the effective configuration under the user XDG config directory and
// `config profiles` lists it alongside project profiles.
func TestE2E_ConfigCreateProfileListsUserProfile(t *testing.T) {
	env := testutil.NewE2EEnvWithAgent(t, testutil.AgentCodex)
	env.SetupAutospecInit()
	writeProjectConfig(t, env, codexProfileBaseConfig+"model: saved-model\n")
	writeProjectProfile(t, env, "team", "model: team-model\n")

	create := env.Run("config", "create", "nightly")

	require.Equal(t, 0, create.ExitCode, "stdout=%s\nstderr=%s", create.Stdout, create.Stderr)
	// HOME is the E2E temp root and XDG_CONFIG_HOME is unset, so the user
	// config directory resolves to $HOME/.config/autospec.
	profilePath := filepath.Join(env.TempDir(), ".config", "autospec", "profiles", "nightly.yml")
	saved, err := os.ReadFile(profilePath)
	require.NoError(t, err)
	require.Contains(t, string(saved), "saved-model")

	list := env.Run("config", "profiles")

	require.Equal(t, 0, list.ExitCode, "stdout=%s\nstderr=%s", list.Stdout, list.Stderr)
	require.Contains(t, list.Stdout, "nightly")
	require.Contains(t, list.Stdout, "team")

	duplicate := env.Run("config", "create", "nightly")
	require.NotEqual(t, 0, duplicate.ExitCode, "re-creating an existing profile without --force must fail")
	require.Contains(t, duplicate.Stdout+duplicate.Stderr, "already exists")
}

func writeProjectProfile(t *testing.T, env *testutil.E2EEnv, name, body string) {
	t.Helper()
	dir := filepath.Join(env.TempDir(), ".autospec", "profiles")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, name+".yml"), []byte(body), 0o644))
}
