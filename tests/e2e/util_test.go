//go:build e2e

// Package e2e provides end-to-end tests for the autospec CLI.
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

// TestE2E_StatusCommand tests the status command functionality.
// This verifies US-010: "status command shows spec state".
func TestE2E_StatusCommand(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"status shows spec state with tasks": {
			description: "Run status with existing tasks.yaml",
			setupFunc: func(env *testutil.E2EEnv) {
				setupUtilTestEnvironment(env)
			},
			args:         []string{"status"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"001-test-feature",
			},
		},
		"status shows no spec when missing": {
			description: "Run status without any spec",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
				// No branch created - no spec to detect
			},
			args:         []string{"status"},
			wantExitCode: shared.ExitValidationFailed,
			wantOutSubstr: []string{
				"spec", "not found", "no",
			},
		},
		"status with verbose flag shows all tasks": {
			description: "Run status --verbose with tasks",
			setupFunc: func(env *testutil.E2EEnv) {
				setupUtilTestEnvironment(env)
			},
			args:         []string{"status", "--verbose"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"001-test-feature",
			},
		},
		"status with spec name argument": {
			description: "Run status with explicit spec name",
			setupFunc: func(env *testutil.E2EEnv) {
				setupUtilTestEnvironment(env)
			},
			args:         []string{"status", "001-test-feature"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"001-test-feature",
			},
		},
		"st alias works": {
			description: "Run st alias",
			setupFunc: func(env *testutil.E2EEnv) {
				setupUtilTestEnvironment(env)
			},
			args:         []string{"st"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"001-test-feature",
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

			if len(tt.wantOutSubstr) > 0 {
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
			}
		})
	}
}

// TestE2E_HistoryCommand tests the history command functionality.
// This verifies US-010: "history command shows recent executions".
func TestE2E_HistoryCommand(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		runBefore     [][]string
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"history shows empty when no commands run": {
			description: "Run history with no previous executions",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"history"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"no", "history", "empty",
			},
		},
		"history with limit flag": {
			description: "Run history with --limit flag",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"history", "--limit", "5"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"no", "history", "empty",
			},
		},
		"history clear removes entries": {
			description: "Run history --clear",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"history", "--clear"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"clear", "history",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			// Run any prerequisite commands
			for _, args := range tt.runBefore {
				env.Run(args...)
			}

			result := env.Run(tt.args...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"exit code mismatch\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			if len(tt.wantOutSubstr) > 0 {
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
			}
		})
	}
}

// TestE2E_ViewCommand tests the view command functionality.
// This verifies US-010: "view command displays artifact contents".
func TestE2E_ViewCommand(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"view shows dashboard overview": {
			description: "Run view with existing specs",
			setupFunc: func(env *testutil.E2EEnv) {
				setupUtilTestEnvironment(env)
			},
			args:         []string{"view"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"spec", "001-test-feature", "dashboard", "project",
			},
		},
		"view with limit flag": {
			description: "Run view with --limit flag",
			setupFunc: func(env *testutil.E2EEnv) {
				setupUtilTestEnvironment(env)
			},
			args:         []string{"view", "--limit", "3"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"spec", "001-test-feature",
			},
		},
		"view shows empty when no specs": {
			description: "Run view with no specs",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"view"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"no", "spec", "0", "empty",
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

			if len(tt.wantOutSubstr) > 0 {
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
			}
		})
	}
}

// TestE2E_CkCommand tests the ck (check for updates) command functionality.
// This verifies US-010: "ck command checks for updates".
func TestE2E_CkCommand(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"ck checks for updates": {
			description: "Run ck command",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args: []string{"ck"},
			// May fail due to network, but should run without crashing
			wantExitCode: -1, // -1 means we accept any exit code
			wantOutSubstr: []string{
				"version", "update", "autospec", "current", "latest", "error", "fail",
			},
		},
		"ck with plain flag": {
			description: "Run ck --plain",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"ck", "--plain"},
			wantExitCode: -1, // Accept any exit code
			wantOutSubstr: []string{
				"version", "update", "autospec", "current", "latest", "error", "fail",
			},
		},
		"check alias works": {
			description: "Run check alias",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"check"},
			wantExitCode: -1, // Accept any exit code
			wantOutSubstr: []string{
				"version", "update", "autospec", "current", "latest", "error", "fail",
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

			// For network-dependent commands, we accept any exit code
			if tt.wantExitCode != -1 {
				require.Equal(t, tt.wantExitCode, result.ExitCode,
					"exit code mismatch\nstdout: %s\nstderr: %s",
					result.Stdout, result.Stderr)
			}

			// Just verify the command ran and produced some output
			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			require.NotEmpty(t, combinedOutput,
				"command should produce output")

			if len(tt.wantOutSubstr) > 0 {
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
			}
		})
	}
}

// TestE2E_CleanCommand tests the clean command functionality.
// This verifies US-010: "clean command removes artifacts".
func TestE2E_CleanCommand(t *testing.T) {
	tests := map[string]struct {
		description        string
		setupFunc          func(*testutil.E2EEnv)
		args               []string
		wantExitCode       int
		wantOutSubstr      []string
		verifyAfter        func(*testing.T, *testutil.E2EEnv)
		skipAutospecRemove bool
	}{
		"clean dry-run shows what would be removed": {
			description: "Run clean --dry-run",
			setupFunc: func(env *testutil.E2EEnv) {
				setupUtilTestEnvironment(env)
			},
			args:         []string{"clean", "--dry-run"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				".autospec", "remove", "would",
			},
			verifyAfter: func(t *testing.T, env *testutil.E2EEnv) {
				// Verify files still exist after dry-run
				autospecDir := filepath.Join(env.TempDir(), ".autospec")
				_, err := os.Stat(autospecDir)
				require.NoError(t, err, ".autospec should still exist after dry-run")
			},
		},
		"clean with --yes removes autospec files": {
			description: "Run clean --yes",
			setupFunc: func(env *testutil.E2EEnv) {
				setupUtilTestEnvironment(env)
			},
			args:         []string{"clean", "--yes"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"remov", "clean",
			},
			verifyAfter: func(t *testing.T, env *testutil.E2EEnv) {
				// Verify .autospec is removed but specs preserved
				autospecDir := filepath.Join(env.TempDir(), ".autospec")
				_, err := os.Stat(autospecDir)
				require.True(t, os.IsNotExist(err),
					".autospec should be removed after clean --yes")

				// Specs should still exist (default behavior preserves specs)
				specsDir := filepath.Join(env.TempDir(), "specs")
				_, err = os.Stat(specsDir)
				require.NoError(t, err, "specs should still exist after clean --yes")
			},
			skipAutospecRemove: true,
		},
		"clean with --yes --remove-specs removes everything": {
			description: "Run clean --yes --remove-specs",
			setupFunc: func(env *testutil.E2EEnv) {
				setupUtilTestEnvironment(env)
			},
			args:         []string{"clean", "--yes", "--remove-specs"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"remov", "clean", "spec",
			},
			verifyAfter: func(t *testing.T, env *testutil.E2EEnv) {
				// Verify both .autospec and specs are removed
				autospecDir := filepath.Join(env.TempDir(), ".autospec")
				_, err := os.Stat(autospecDir)
				require.True(t, os.IsNotExist(err),
					".autospec should be removed")

				specsDir := filepath.Join(env.TempDir(), "specs")
				_, err = os.Stat(specsDir)
				require.True(t, os.IsNotExist(err),
					"specs should be removed with --remove-specs")
			},
			skipAutospecRemove: true,
		},
		"clean with --keep-specs preserves specs": {
			description: "Run clean --yes --keep-specs",
			setupFunc: func(env *testutil.E2EEnv) {
				setupUtilTestEnvironment(env)
			},
			args:         []string{"clean", "--yes", "--keep-specs"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"remov", "clean",
			},
			verifyAfter: func(t *testing.T, env *testutil.E2EEnv) {
				// Specs should still exist
				specsDir := filepath.Join(env.TempDir(), "specs")
				_, err := os.Stat(specsDir)
				require.NoError(t, err, "specs should still exist with --keep-specs")
			},
			skipAutospecRemove: true,
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

			if len(tt.wantOutSubstr) > 0 {
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
			}

			if tt.verifyAfter != nil {
				tt.verifyAfter(t, env)
			}
		})
	}
}

// TestE2E_ArtifactValidation tests the artifact command for YAML validation.
// This verifies US-010: "ck command validates YAML files" (artifact command).
func TestE2E_ArtifactValidation(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv) string
		args          func(string) []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"artifact validates valid spec.yaml": {
			description: "Run artifact on valid spec.yaml",
			setupFunc: func(env *testutil.E2EEnv) string {
				setupUtilTestEnvironment(env)
				return filepath.Join(env.SpecsDir(), "001-test-feature", "spec.yaml")
			},
			args: func(path string) []string {
				return []string{"artifact", path}
			},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"valid", "spec", "success", "✓",
			},
		},
		"artifact validates valid plan.yaml": {
			description: "Run artifact on valid plan.yaml",
			setupFunc: func(env *testutil.E2EEnv) string {
				setupUtilTestEnvironment(env)
				return filepath.Join(env.SpecsDir(), "001-test-feature", "plan.yaml")
			},
			args: func(path string) []string {
				return []string{"artifact", path}
			},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"valid", "plan", "success", "✓",
			},
		},
		"artifact validates valid tasks.yaml": {
			description: "Run artifact on valid tasks.yaml",
			setupFunc: func(env *testutil.E2EEnv) string {
				setupUtilTestEnvironment(env)
				return filepath.Join(env.SpecsDir(), "001-test-feature", "tasks.yaml")
			},
			args: func(path string) []string {
				return []string{"artifact", path}
			},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"valid", "tasks", "success", "✓",
			},
		},
		"artifact validates constitution.yaml": {
			description: "Run artifact on valid constitution.yaml",
			setupFunc: func(env *testutil.E2EEnv) string {
				setupUtilTestEnvironment(env)
				return filepath.Join(env.TempDir(), ".autospec", "constitution.yaml")
			},
			args: func(path string) []string {
				return []string{"artifact", path}
			},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"valid", "constitution", "success", "✓",
			},
		},
		"artifact fails on invalid YAML": {
			description: "Run artifact on invalid YAML file",
			setupFunc: func(env *testutil.E2EEnv) string {
				setupUtilTestEnvironment(env)
				invalidPath := filepath.Join(env.TempDir(), "invalid.yaml")
				content := "this is not: valid: yaml: syntax\n  bad indentation"
				_ = os.WriteFile(invalidPath, []byte(content), 0o644)
				return invalidPath
			},
			args: func(path string) []string {
				return []string{"artifact", path}
			},
			wantExitCode: shared.ExitValidationFailed,
			wantOutSubstr: []string{
				"error", "invalid", "yaml", "fail",
			},
		},
		"artifact fails on missing file": {
			description: "Run artifact on non-existent file",
			setupFunc: func(env *testutil.E2EEnv) string {
				env.SetupAutospecInit()
				env.InitGitRepo()
				return filepath.Join(env.TempDir(), "nonexistent.yaml")
			},
			args: func(path string) []string {
				return []string{"artifact", path}
			},
			wantExitCode: shared.ExitValidationFailed,
			wantOutSubstr: []string{
				"error", "not found", "no such file",
			},
		},
		"artifact with type only auto-detects spec": {
			description: "Run artifact spec (type only)",
			setupFunc: func(env *testutil.E2EEnv) string {
				setupUtilTestEnvironment(env)
				return ""
			},
			args: func(_ string) []string {
				return []string{"artifact", "spec"}
			},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"valid", "spec", "success", "✓",
			},
		},
		"artifact --schema shows schema": {
			description: "Run artifact spec --schema",
			setupFunc: func(env *testutil.E2EEnv) string {
				// Need to set up environment with a spec for --schema to work
				setupUtilTestEnvironment(env)
				return ""
			},
			args: func(_ string) []string {
				return []string{"artifact", "spec", "--schema"}
			},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"schema", "spec", "feature", "user_stories",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			var artifactPath string
			if tt.setupFunc != nil {
				artifactPath = tt.setupFunc(env)
			}

			args := tt.args(artifactPath)
			result := env.Run(args...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"exit code mismatch\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			if len(tt.wantOutSubstr) > 0 {
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
			}
		})
	}
}

// TestE2E_VersionCommand tests the version command functionality.
func TestE2E_VersionCommand(t *testing.T) {
	tests := map[string]struct {
		description   string
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"version shows version info": {
			description:  "Run version command",
			args:         []string{"version"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"autospec", "version",
			},
		},
		"v alias works": {
			description:  "Run v alias",
			args:         []string{"v"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"autospec", "version",
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

// TestE2E_HelpCommand tests the help command functionality.
func TestE2E_HelpCommand(t *testing.T) {
	tests := map[string]struct {
		description   string
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"help shows usage": {
			description:  "Run help command",
			args:         []string{"help"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"autospec", "usage", "command",
			},
		},
		"--help flag shows usage": {
			description:  "Run --help flag",
			args:         []string{"--help"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"autospec", "usage", "command",
			},
		},
		"help for specific command": {
			description:  "Run help for specify command",
			args:         []string{"help", "specify"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"specify", "spec", "feature",
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

// setupUtilTestEnvironment sets up a complete environment for utility command testing.
func setupUtilTestEnvironment(env *testutil.E2EEnv) {
	env.SetupAutospecInit()
	env.SetupConstitution()
	env.InitGitRepo()
	env.CreateBranch("001-test-feature")
	env.SetupTasks("001-test-feature")
}
