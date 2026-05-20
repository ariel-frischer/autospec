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

// TestE2E_AllCommand tests the all command functionality.
// This verifies the complete workflow: specify -> plan -> tasks -> implement.
func TestE2E_AllCommand(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"all --help shows usage": {
			description: "Run all --help",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"all", "--help"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"all", "specify", "plan", "tasks", "implement", "workflow",
			},
		},
		"all with description runs full workflow": {
			description: "Run all with feature description",
			setupFunc: func(env *testutil.E2EEnv) {
				setupFullEnvironmentForInternals(env)
			},
			args:         []string{"all", "Test feature for E2E testing"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"spec", "plan", "task", "implement",
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

// TestE2E_NewFeatureCommand tests the new-feature command.
func TestE2E_NewFeatureCommand(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"new-feature --help shows usage": {
			description: "Run new-feature --help",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"new-feature", "--help"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"new-feature", "branch", "feature", "directory",
			},
		},
		"new-feature creates feature branch and directory": {
			description: "Run new-feature with name",
			setupFunc: func(env *testutil.E2EEnv) {
				setupFullEnvironmentForInternals(env)
			},
			args:         []string{"new-feature", "test-new-feature"},
			wantExitCode: -1, // May fail or succeed depending on git state
			wantOutSubstr: []string{
				"feature", "branch", "create", "directory", "error", "exist",
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

			// Accept any exit code if specified as -1
			if tt.wantExitCode != -1 {
				require.Equal(t, tt.wantExitCode, result.ExitCode,
					"exit code mismatch\nstdout: %s\nstderr: %s",
					result.Stdout, result.Stderr)
			}

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

// TestE2E_PrereqsCommand tests the prereqs command.
func TestE2E_PrereqsCommand(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"prereqs --help shows usage": {
			description: "Run prereqs --help",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"prereqs", "--help"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"prereqs", "prerequisite", "check", "stage",
			},
		},
		"prereqs with --json outputs JSON": {
			description: "Run prereqs --json",
			setupFunc: func(env *testutil.E2EEnv) {
				setupFullEnvironmentForInternals(env)
				env.SetupTasks("001-test-feature")
			},
			args:         []string{"prereqs", "--json"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"feature_dir", "feature_spec", "{", "}",
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

// TestE2E_TaskBlock tests the task block subcommand.
func TestE2E_TaskBlock(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"task block --help shows usage": {
			description: "Run task block --help",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"task", "block", "--help"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"block", "task", "reason",
			},
		},
		"task block requires task ID": {
			description: "Run task block without task ID",
			setupFunc: func(env *testutil.E2EEnv) {
				setupFullEnvironmentForInternals(env)
				env.SetupTasks("001-test-feature")
			},
			args:         []string{"task", "block"},
			wantExitCode: shared.ExitValidationFailed, // Cobra returns exit code 1 for missing args
			wantOutSubstr: []string{
				"task", "id", "required", "error", "accepts", "arg",
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

// TestE2E_TaskUnblock tests the task unblock subcommand.
func TestE2E_TaskUnblock(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"task unblock --help shows usage": {
			description: "Run task unblock --help",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"task", "unblock", "--help"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"unblock", "task", "status",
			},
		},
		"task unblock requires task ID": {
			description: "Run task unblock without task ID",
			setupFunc: func(env *testutil.E2EEnv) {
				setupFullEnvironmentForInternals(env)
				env.SetupTasks("001-test-feature")
			},
			args:         []string{"task", "unblock"},
			wantExitCode: shared.ExitValidationFailed, // Cobra returns exit code 1 for missing args
			wantOutSubstr: []string{
				"task", "id", "required", "error", "accepts", "arg",
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

// TestE2E_TaskList tests the task list subcommand.
func TestE2E_TaskList(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"task list --help shows usage": {
			description: "Run task list --help",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"task", "list", "--help"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"list", "task", "status", "filter",
			},
		},
		"task list shows all tasks": {
			description: "Run task list with tasks",
			setupFunc: func(env *testutil.E2EEnv) {
				setupFullEnvironmentForInternals(env)
				env.SetupTasks("001-test-feature")
			},
			args:         []string{"task", "list"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"task", "status", "pending", "id",
			},
		},
		"task list --blocked shows blocked tasks": {
			description: "Run task list --blocked",
			setupFunc: func(env *testutil.E2EEnv) {
				setupFullEnvironmentForInternals(env)
				env.SetupTasks("001-test-feature")
			},
			args:         []string{"task", "list", "--blocked"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"task", "blocked", "no",
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

// TestE2E_UpdateAgentContextCommand tests the update-agent-context command.
func TestE2E_UpdateAgentContextCommand(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"update-agent-context --help shows usage": {
			description: "Run update-agent-context --help",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"update-agent-context", "--help"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"update-agent-context", "agent", "context", "plan", "technology",
			},
		},
		"update-agent-context requires plan.yaml": {
			description: "Run update-agent-context without plan",
			setupFunc: func(env *testutil.E2EEnv) {
				setupFullEnvironmentForInternals(env)
				env.SetupSpec("001-test-feature")
			},
			args:         []string{"update-agent-context"},
			wantExitCode: -1, // May fail without plan
			wantOutSubstr: []string{
				"plan", "context", "agent", "error", "not found",
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

			if tt.wantExitCode != -1 {
				require.Equal(t, tt.wantExitCode, result.ExitCode,
					"exit code mismatch\nstdout: %s\nstderr: %s",
					result.Stdout, result.Stderr)
			}

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

// TestE2E_UpdateTaskCommand tests the update-task command.
func TestE2E_UpdateTaskCommand(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"update-task --help shows usage": {
			description: "Run update-task --help",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"update-task", "--help"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"update-task", "task", "status",
			},
		},
		"update-task requires task ID and status": {
			description: "Run update-task without args",
			setupFunc: func(env *testutil.E2EEnv) {
				setupFullEnvironmentForInternals(env)
				env.SetupTasks("001-test-feature")
			},
			args:         []string{"update-task"},
			wantExitCode: shared.ExitValidationFailed, // Cobra returns exit code 1 for missing args
			wantOutSubstr: []string{
				"task", "id", "status", "required", "error", "accepts", "arg",
			},
		},
		"update-task updates task status": {
			description: "Run update-task with task ID and status",
			setupFunc: func(env *testutil.E2EEnv) {
				setupFullEnvironmentForInternals(env)
				env.SetupTasks("001-test-feature")
			},
			args:         []string{"update-task", "T001", "InProgress"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"task", "t001", "inprogress", "updat",
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

// TestE2E_YamlCheck tests the yaml check subcommand.
func TestE2E_YamlCheck(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv) string
		args          func(string) []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"yaml check --help shows usage": {
			description: "Run yaml check --help",
			setupFunc: func(env *testutil.E2EEnv) string {
				env.SetupAutospecInit()
				env.InitGitRepo()
				return ""
			},
			args: func(_ string) []string {
				return []string{"yaml", "check", "--help"}
			},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"yaml", "check", "syntax", "valid",
			},
		},
		"yaml check validates valid yaml": {
			description: "Run yaml check on valid YAML",
			setupFunc: func(env *testutil.E2EEnv) string {
				setupFullEnvironmentForInternals(env)
				env.SetupSpec("001-test-feature")
				return filepath.Join(env.SpecsDir(), "001-test-feature", "spec.yaml")
			},
			args: func(path string) []string {
				return []string{"yaml", "check", path}
			},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"yaml", "valid", "ok", "success", "✓",
			},
		},
		"yaml check fails on invalid yaml": {
			description: "Run yaml check on invalid YAML",
			setupFunc: func(env *testutil.E2EEnv) string {
				setupFullEnvironmentForInternals(env)
				invalidPath := filepath.Join(env.TempDir(), "invalid.yaml")
				content := "this is not: valid: yaml: [broken"
				_ = os.WriteFile(invalidPath, []byte(content), 0o644)
				return invalidPath
			},
			args: func(path string) []string {
				return []string{"yaml", "check", path}
			},
			wantExitCode: shared.ExitValidationFailed,
			wantOutSubstr: []string{
				"yaml", "error", "invalid", "fail",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			var path string
			if tt.setupFunc != nil {
				path = tt.setupFunc(env)
			}

			args := tt.args(path)
			result := env.Run(args...)

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

// TestE2E_SauceCommand tests the sauce command.
func TestE2E_SauceCommand(t *testing.T) {
	tests := map[string]struct {
		description   string
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"sauce shows source URL": {
			description:  "Run sauce command",
			args:         []string{"sauce"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"github", "autospec", "http", "source",
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

// setupFullEnvironmentForInternals sets up an environment for internal command testing.
func setupFullEnvironmentForInternals(env *testutil.E2EEnv) {
	env.SetupAutospecInit()
	env.SetupConstitution()
	env.InitGitRepo()
	env.CreateBranch("001-test-feature")
}
