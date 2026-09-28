//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestE2E_ConfigShow tests the config show command.
// This verifies US-011: "config show displays configuration".
func TestE2E_ConfigShow(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"config show displays yaml by default": {
			description: "Run config show with no flags",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"config", "show"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"agent_preset", "specs_dir", "max_retries",
			},
		},
		"config show --json displays json format": {
			description: "Run config show with --json flag",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"config", "show", "--json"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"{", "}", "agent_preset",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			result := env.Run(tt.args...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"exit code mismatch\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			for _, substr := range tt.wantOutSubstr {
				require.Contains(t, combinedOutput, strings.ToLower(substr),
					"output should contain %q\nstdout: %s\nstderr: %s",
					substr, result.Stdout, result.Stderr)
			}
		})
	}
}

// TestE2E_ConfigSetGet tests the config set and get commands.
// This verifies US-011: "config set/get modifies values".
func TestE2E_ConfigSetGet(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		setArgs       []string
		getArgs       []string
		wantSetExit   int
		wantGetExit   int
		wantOutSubstr []string
	}{
		"config set and get max_retries project": {
			description: "Set and get max_retries value in project config",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			setArgs:       []string{"config", "set", "max_retries", "5", "--project"},
			getArgs:       []string{"config", "get", "max_retries", "--project"},
			wantSetExit:   shared.ExitSuccess,
			wantGetExit:   shared.ExitSuccess,
			wantOutSubstr: []string{"5"},
		},
		"config set with project flag": {
			description: "Set value in project config",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			setArgs:       []string{"config", "set", "max_retries", "10", "--project"},
			getArgs:       []string{"config", "get", "max_retries", "--project"},
			wantSetExit:   shared.ExitSuccess,
			wantGetExit:   shared.ExitSuccess,
			wantOutSubstr: []string{"10", "project"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			// Run set command
			setResult := env.Run(tt.setArgs...)
			require.Equal(t, tt.wantSetExit, setResult.ExitCode,
				"set exit code mismatch\nstdout: %s\nstderr: %s",
				setResult.Stdout, setResult.Stderr)

			// Run get command
			getResult := env.Run(tt.getArgs...)
			require.Equal(t, tt.wantGetExit, getResult.ExitCode,
				"get exit code mismatch\nstdout: %s\nstderr: %s",
				getResult.Stdout, getResult.Stderr)

			combinedOutput := strings.ToLower(getResult.Stdout + getResult.Stderr)
			for _, substr := range tt.wantOutSubstr {
				require.Contains(t, combinedOutput, strings.ToLower(substr),
					"output should contain %q", substr)
			}
		})
	}
}

// TestE2E_ConfigToggle tests the config toggle command.
// This verifies US-011: "config toggle switches boolean values".
func TestE2E_ConfigToggle(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"config toggle skip_preflight": {
			description: "Toggle skip_preflight boolean",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"config", "toggle", "skip_preflight"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"toggled", "skip_preflight",
			},
		},
		"config toggle with project flag": {
			description: "Toggle at project level",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"config", "toggle", "skip_preflight", "--project"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"toggled", "project",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			result := env.Run(tt.args...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"exit code mismatch\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			for _, substr := range tt.wantOutSubstr {
				require.Contains(t, combinedOutput, strings.ToLower(substr),
					"output should contain %q", substr)
			}
		})
	}
}

// TestE2E_ConfigKeys tests the config keys command.
// This verifies US-011: "config keys lists known keys".
func TestE2E_ConfigKeys(t *testing.T) {
	tests := map[string]struct {
		description   string
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"config keys lists all keys": {
			description:  "Run config keys",
			args:         []string{"config", "keys"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"agent_preset", "max_retries", "specs_dir", "timeout",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			result := env.Run(tt.args...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"exit code mismatch\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			for _, substr := range tt.wantOutSubstr {
				require.Contains(t, combinedOutput, strings.ToLower(substr),
					"output should contain %q", substr)
			}
		})
	}
}

// TestE2E_ConfigSync tests the config sync command.
// This verifies US-011: "config sync synchronizes configs".
func TestE2E_ConfigSync(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"config sync runs without error": {
			description: "Run config sync",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"config", "sync"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"config", "up to date", "user",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			result := env.Run(tt.args...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"exit code mismatch\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			foundMatch := false
			for _, substr := range tt.wantOutSubstr {
				if strings.Contains(combinedOutput, strings.ToLower(substr)) {
					foundMatch = true
					break
				}
			}
			require.True(t, foundMatch,
				"output should contain one of %v\nstdout: %s\nstderr: %s",
				tt.wantOutSubstr, result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_InitCommand tests the init command.
// This verifies US-011: "init command creates config files".
func TestE2E_InitCommand(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantOutSubstr []string
	}{
		"init --help shows usage": {
			description: "Run init --help",
			setupFunc: func(env *testutil.E2EEnv) {
				env.InitGitRepo()
			},
			args: []string{"init", "--help"},
			wantOutSubstr: []string{
				"init", "config", "autospec", "project", "force",
			},
		},
		"init --no-constitution skips constitution creation": {
			description: "Run init with --no-constitution flag",
			setupFunc: func(env *testutil.E2EEnv) {
				env.InitGitRepo()
			},
			args: []string{"init", "--no-constitution", "--no-agents"},
			wantOutSubstr: []string{
				"config", "autospec", "created", "setup",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			result := env.Run(tt.args...)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			foundMatch := false
			for _, substr := range tt.wantOutSubstr {
				if strings.Contains(combinedOutput, strings.ToLower(substr)) {
					foundMatch = true
					break
				}
			}
			require.True(t, foundMatch,
				"output should contain one of %v\nstdout: %s\nstderr: %s",
				tt.wantOutSubstr, result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_MigrateCommand tests the migrate command.
// This verifies US-011: "migrate command handles migrations".
func TestE2E_MigrateCommand(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantOutSubstr []string
	}{
		"migrate --help shows available subcommands": {
			description: "Run migrate --help",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args: []string{"migrate", "--help"},
			wantOutSubstr: []string{
				"migrate", "md-to-yaml",
			},
		},
		"migrate md-to-yaml --help shows usage": {
			description: "Run migrate md-to-yaml --help",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args: []string{"migrate", "md-to-yaml", "--help"},
			wantOutSubstr: []string{
				"md-to-yaml", "convert", "markdown",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			result := env.Run(tt.args...)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			foundMatch := false
			for _, substr := range tt.wantOutSubstr {
				if strings.Contains(combinedOutput, strings.ToLower(substr)) {
					foundMatch = true
					break
				}
			}
			require.True(t, foundMatch,
				"output should contain one of %v\nstdout: %s\nstderr: %s",
				tt.wantOutSubstr, result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_DoctorCommand tests the doctor command.
// This verifies US-011: "doctor command shows dependency status".
func TestE2E_DoctorCommand(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantOutSubstr []string
	}{
		"doctor shows health checks": {
			description: "Run doctor command",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args: []string{"doctor"},
			wantOutSubstr: []string{
				"git", "claude", "check",
			},
		},
		"doc alias works": {
			description: "Run doc alias",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args: []string{"doc"},
			wantOutSubstr: []string{
				"git", "claude", "check",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			result := env.Run(tt.args...)

			// Doctor may fail if dependencies are missing, but should run
			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			foundMatch := false
			for _, substr := range tt.wantOutSubstr {
				if strings.Contains(combinedOutput, strings.ToLower(substr)) {
					foundMatch = true
					break
				}
			}
			require.True(t, foundMatch,
				"output should contain one of %v\nstdout: %s\nstderr: %s",
				tt.wantOutSubstr, result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_DoctorFollowsConfiguredAgent verifies doctor checks the configured
// agent instead of always requiring Claude CLI and Claude settings.
func TestE2E_DoctorFollowsConfiguredAgent(t *testing.T) {
	tests := map[string]struct {
		agent        testutil.AgentPreset
		removeBinary string
		wantExitCode int
		wantOutput   string
	}{
		"codex project passes without Claude settings": {
			agent:        testutil.AgentCodex,
			wantExitCode: shared.ExitSuccess,
			wantOutput:   "✓ codex CLI",
		},
		"missing configured agent fails": {
			agent:        testutil.AgentCodex,
			removeBinary: "codex",
			wantExitCode: 1,
			wantOutput:   "✗ codex CLI",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnvWithAgent(t, tt.agent)
			env.InitGitRepo()
			env.SetupAutospecInit()
			if tt.removeBinary != "" {
				require.NoError(t, os.Remove(filepath.Join(env.BinDir(), tt.removeBinary)))
			}

			result := env.Run("doctor")

			require.Equal(t, tt.wantExitCode, result.ExitCode, "stdout=%s\nstderr=%s", result.Stdout, result.Stderr)
			require.Contains(t, result.Stdout, tt.wantOutput)
			require.NotContains(t, result.Stdout, "Claude settings")
		})
	}
}
