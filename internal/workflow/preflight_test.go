package workflow

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunPreflightChecks tests the pre-flight validation logic for directory checks.
// Note: This test focuses on directory validation. The overall Passed status also depends
// on external commands (like claude CLI) which may not be available in CI environments.
func TestRunPreflightChecks(t *testing.T) {
	tests := map[string]struct {
		setupDirs   []string // Directories to create in temp dir
		wantMissing int      // Expected number of missing directories
	}{
		"all directories present": {
			setupDirs:   []string{".claude/commands", ".autospec"},
			wantMissing: 0,
		},
		"missing .claude/commands directory": {
			setupDirs:   []string{".autospec"},
			wantMissing: 1,
		},
		"missing .autospec directory": {
			setupDirs:   []string{".claude/commands"},
			wantMissing: 1,
		},
		"missing both directories": {
			setupDirs:   []string{},
			wantMissing: 2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Use temp directory to avoid modifying actual repo directories
			tmpDir := t.TempDir()
			origDir, err := os.Getwd()
			require.NoError(t, err)
			require.NoError(t, os.Chdir(tmpDir))
			defer func() { _ = os.Chdir(origDir) }()

			// Create test directories
			for _, dir := range tc.setupDirs {
				require.NoError(t, os.MkdirAll(dir, 0o755))
			}

			// Run pre-flight checks
			result, err := RunPreflightChecks()
			require.NoError(t, err)

			// Verify directory-related results
			assert.Len(t, result.MissingDirs, tc.wantMissing,
				"Should detect correct number of missing directories")

			// When directories are missing, Passed should be false
			if tc.wantMissing > 0 {
				assert.False(t, result.Passed,
					"Passed should be false when directories are missing")
			}
		})
	}
}

// TestCheckCommandExists tests command existence checking
func TestCheckCommandExists(t *testing.T) {
	tests := map[string]struct {
		command string
		wantErr bool
	}{
		"git exists": {
			command: "git",
			wantErr: false,
		},
		"nonexistent command": {
			command: "this-command-does-not-exist-12345",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := checkCommandExists(tc.command)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGenerateMissingDirsWarning tests warning message generation
func TestGenerateMissingDirsWarning(t *testing.T) {
	tests := map[string]struct {
		missingDirs  []string
		gitRoot      string
		wantContains []string
	}{
		"with git root": {
			missingDirs: []string{".claude/commands/", ".autospec/"},
			gitRoot:     "/home/user/project",
			wantContains: []string{
				"WARNING",
				".claude/commands/",
				".autospec/",
				"/home/user/project",
				"autospec init",
			},
		},
		"without git root": {
			missingDirs: []string{".claude/commands/"},
			gitRoot:     "",
			wantContains: []string{
				"WARNING",
				".claude/commands/",
				"autospec init",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			warning := generateMissingDirsWarning(tc.missingDirs, tc.gitRoot)

			for _, want := range tc.wantContains {
				assert.Contains(t, warning, want,
					"Warning should contain: %s", want)
			}
		})
	}
}

// TestShouldRunPreflightChecks tests pre-flight check skipping logic
func TestShouldRunPreflightChecks(t *testing.T) {
	tests := map[string]struct {
		skipPreflight bool
		ciEnvVar      string
		ciValue       string
		wantRun       bool
	}{
		"run normally": {
			skipPreflight: false,
			ciEnvVar:      "",
			ciValue:       "",
			wantRun:       true,
		},
		"skip via flag": {
			skipPreflight: true,
			ciEnvVar:      "",
			ciValue:       "",
			wantRun:       false,
		},
		"skip in GitHub Actions": {
			skipPreflight: false,
			ciEnvVar:      "GITHUB_ACTIONS",
			ciValue:       "true",
			wantRun:       false,
		},
		"skip in GitLab CI": {
			skipPreflight: false,
			ciEnvVar:      "GITLAB_CI",
			ciValue:       "true",
			wantRun:       false,
		},
		"skip in CircleCI": {
			skipPreflight: false,
			ciEnvVar:      "CIRCLECI",
			ciValue:       "true",
			wantRun:       false,
		},
	}

	// List of CI environment variables that must be cleared for proper testing
	ciEnvVars := []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI"}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Clear all CI env vars first to ensure clean test environment
			for _, envVar := range ciEnvVars {
				t.Setenv(envVar, "")
			}

			// Set environment variable if specified for this test case
			if tc.ciEnvVar != "" {
				t.Setenv(tc.ciEnvVar, tc.ciValue)
			}

			result := ShouldRunPreflightChecks(tc.skipPreflight)
			assert.Equal(t, tc.wantRun, result,
				"ShouldRunPreflightChecks should return %v", tc.wantRun)
		})
	}
}

// TestCheckDependencies tests dependency checking
func TestCheckDependencies(t *testing.T) {
	// This test will check for git (which should exist)
	// and potentially fail for claude/specify if not installed
	err := CheckDependencies()
	// We can't assert success/failure because it depends on the system
	// But we can verify the error message format if it fails
	if err != nil {
		assert.Contains(t, err.Error(), "missing required dependencies",
			"Error should mention missing dependencies")
	}
}

// TestCheckProjectStructure tests project structure validation
func TestCheckProjectStructure(t *testing.T) {
	// Use temp directory to avoid modifying actual repo directories
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	// Create temporary directories
	require.NoError(t, os.MkdirAll(".claude/commands", 0o755))
	require.NoError(t, os.MkdirAll(".autospec", 0o755))

	err = CheckProjectStructure()
	assert.NoError(t, err, "Should pass with all directories present")

	// Remove one directory and test again
	os.RemoveAll(".claude")
	err = CheckProjectStructure()
	assert.Error(t, err, "Should fail with missing directory")
	assert.Contains(t, err.Error(), "missing required directories")
}

// BenchmarkRunPreflightChecks benchmarks pre-flight checks performance
// Target: <100ms
func BenchmarkRunPreflightChecks(b *testing.B) {
	// Use temp directory to avoid modifying actual repo directories
	tmpDir := b.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Setup test directories
	os.MkdirAll(".claude/commands", 0o755)
	os.MkdirAll(".autospec", 0o755)

	// Reset timer after setup
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = RunPreflightChecks()
	}
}

// BenchmarkCheckCommandExists benchmarks command existence checking
func BenchmarkCheckCommandExists(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = checkCommandExists("git")
	}
}

// TestCheckConstitutionExists tests the constitution file validation
func TestCheckConstitutionExists(t *testing.T) {
	tests := map[string]struct {
		setupFiles   map[string]string // path -> content
		wantExists   bool
		wantPath     string
		wantErrEmpty bool
	}{
		"new default path (.yaml)": {
			setupFiles:   map[string]string{".autospec/constitution.yaml": "project_name: Test"},
			wantExists:   true,
			wantPath:     ".autospec/constitution.yaml",
			wantErrEmpty: true,
		},
		"new default path (.yml)": {
			setupFiles:   map[string]string{".autospec/constitution.yml": "project_name: Test"},
			wantExists:   true,
			wantPath:     ".autospec/constitution.yml",
			wantErrEmpty: true,
		},
		"legacy autospec/memory path (.yaml)": {
			setupFiles:   map[string]string{".autospec/memory/constitution.yaml": "project_name: Test"},
			wantExists:   true,
			wantPath:     ".autospec/memory/constitution.yaml",
			wantErrEmpty: true,
		},
		"legacy autospec/memory path (.yml)": {
			setupFiles:   map[string]string{".autospec/memory/constitution.yml": "project_name: Test"},
			wantExists:   true,
			wantPath:     ".autospec/memory/constitution.yml",
			wantErrEmpty: true,
		},
		"legacy specify constitution exists (.yaml)": {
			setupFiles:   map[string]string{".specify/memory/constitution.yaml": "project_name: Test"},
			wantExists:   true,
			wantPath:     ".specify/memory/constitution.yaml",
			wantErrEmpty: true,
		},
		"legacy specify constitution exists (.yml)": {
			setupFiles:   map[string]string{".specify/memory/constitution.yml": "project_name: Test"},
			wantExists:   true,
			wantPath:     ".specify/memory/constitution.yml",
			wantErrEmpty: true,
		},
		"new path takes precedence over legacy": {
			setupFiles: map[string]string{
				".autospec/constitution.yaml":        "project_name: New",
				".autospec/memory/constitution.yaml": "project_name: Legacy",
			},
			wantExists:   true,
			wantPath:     ".autospec/constitution.yaml",
			wantErrEmpty: true,
		},
		"yaml takes precedence over yml": {
			setupFiles: map[string]string{
				".autospec/constitution.yaml": "project_name: YAML",
				".autospec/constitution.yml":  "project_name: YML",
			},
			wantExists:   true,
			wantPath:     ".autospec/constitution.yaml",
			wantErrEmpty: true,
		},
		"autospec takes precedence over specify": {
			setupFiles: map[string]string{
				".autospec/constitution.yaml":       "project_name: Autospec",
				".specify/memory/constitution.yaml": "project_name: Specify",
			},
			wantExists:   true,
			wantPath:     ".autospec/constitution.yaml",
			wantErrEmpty: true,
		},
		"no constitution exists": {
			setupFiles:   map[string]string{},
			wantExists:   false,
			wantPath:     "",
			wantErrEmpty: false,
		},
		"directories exist but no constitution file": {
			setupFiles:   map[string]string{".autospec/.keep": "", ".specify/memory/.keep": ""},
			wantExists:   false,
			wantPath:     "",
			wantErrEmpty: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Use temp directory to avoid modifying actual repo directories
			tmpDir := t.TempDir()
			origDir, err := os.Getwd()
			require.NoError(t, err)
			require.NoError(t, os.Chdir(tmpDir))
			defer func() { _ = os.Chdir(origDir) }()

			// Create test files
			for path, content := range tc.setupFiles {
				dir := filepath.Dir(path)
				require.NoError(t, os.MkdirAll(dir, 0o755))
				require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
			}

			result := CheckConstitutionExists()

			assert.Equal(t, tc.wantExists, result.Exists,
				"Exists should match expected")
			assert.Equal(t, tc.wantPath, result.Path,
				"Path should match expected")
			if tc.wantErrEmpty {
				assert.Empty(t, result.ErrorMessage,
					"ErrorMessage should be empty when constitution exists")
			} else {
				assert.NotEmpty(t, result.ErrorMessage,
					"ErrorMessage should not be empty when constitution missing")
				assert.Contains(t, result.ErrorMessage, "autospec constitution",
					"ErrorMessage should mention how to create constitution")
			}
		})
	}
}

// TestGenerateConstitutionMissingError tests the error message generation
func TestGenerateConstitutionMissingError(t *testing.T) {
	errMsg := generateConstitutionMissingError()

	assert.Contains(t, errMsg, "Error:")
	assert.Contains(t, errMsg, "constitution not found")
	assert.Contains(t, errMsg, "autospec constitution")
	assert.Contains(t, errMsg, ".autospec/constitution.yaml")
	assert.Contains(t, errMsg, "autospec init")
}

// BenchmarkCheckConstitutionExists benchmarks constitution check performance
// Target: <10ms
func BenchmarkCheckConstitutionExists(b *testing.B) {
	// Use temp directory to avoid modifying actual repo directories
	tmpDir := b.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Setup with constitution file
	os.MkdirAll(".autospec", 0o755)
	os.WriteFile(".autospec/constitution.yaml", []byte("project_name: Test"), 0o644)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = CheckConstitutionExists()
	}
}

// TestValidateStagePrerequisites tests prerequisite validation for stages
func TestValidateStagePrerequisites(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stage       Stage
		setupFunc   func(t *testing.T, specDir string) func()
		wantValid   bool
		wantMissing []string
	}{
		"specify stage - no prerequisites required": {
			stage: StageSpecify,
			setupFunc: func(_ *testing.T, specDir string) func() {
				os.MkdirAll(specDir, 0o755)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   true,
			wantMissing: []string{},
		},
		"plan stage - spec.yaml exists": {
			stage: StagePlan,
			setupFunc: func(t *testing.T, specDir string) func() {
				os.MkdirAll(specDir, 0o755)
				copyValidTestdata(t, "spec.yaml", specDir)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   true,
			wantMissing: []string{},
		},
		"plan stage - spec.yaml missing": {
			stage: StagePlan,
			setupFunc: func(_ *testing.T, specDir string) func() {
				os.MkdirAll(specDir, 0o755)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   false,
			wantMissing: []string{"spec.yaml"},
		},
		"tasks stage - plan.yaml exists": {
			stage: StageTasks,
			setupFunc: func(t *testing.T, specDir string) func() {
				os.MkdirAll(specDir, 0o755)
				copyValidTestdata(t, "plan.yaml", specDir)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   true,
			wantMissing: []string{},
		},
		"tasks stage - plan.yaml missing": {
			stage: StageTasks,
			setupFunc: func(_ *testing.T, specDir string) func() {
				os.MkdirAll(specDir, 0o755)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   false,
			wantMissing: []string{"plan.yaml"},
		},
		"implement stage - tasks.yaml exists": {
			stage: StageImplement,
			setupFunc: func(t *testing.T, specDir string) func() {
				os.MkdirAll(specDir, 0o755)
				copyValidTestdata(t, "tasks.yaml", specDir)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   true,
			wantMissing: []string{},
		},
		"implement stage - tasks.yaml missing": {
			stage: StageImplement,
			setupFunc: func(_ *testing.T, specDir string) func() {
				os.MkdirAll(specDir, 0o755)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   false,
			wantMissing: []string{"tasks.yaml"},
		},
		"clarify stage - spec.yaml exists": {
			stage: StageClarify,
			setupFunc: func(t *testing.T, specDir string) func() {
				os.MkdirAll(specDir, 0o755)
				copyValidTestdata(t, "spec.yaml", specDir)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   true,
			wantMissing: []string{},
		},
		"clarify stage - spec.yaml missing": {
			stage: StageClarify,
			setupFunc: func(_ *testing.T, specDir string) func() {
				os.MkdirAll(specDir, 0o755)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   false,
			wantMissing: []string{"spec.yaml"},
		},
		"checklist stage - spec.yaml exists": {
			stage: StageChecklist,
			setupFunc: func(t *testing.T, specDir string) func() {
				os.MkdirAll(specDir, 0o755)
				copyValidTestdata(t, "spec.yaml", specDir)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   true,
			wantMissing: []string{},
		},
		"checklist stage - spec.yaml missing": {
			stage: StageChecklist,
			setupFunc: func(_ *testing.T, specDir string) func() {
				os.MkdirAll(specDir, 0o755)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   false,
			wantMissing: []string{"spec.yaml"},
		},
		"analyze stage - all artifacts exist": {
			stage: StageAnalyze,
			setupFunc: func(t *testing.T, specDir string) func() {
				os.MkdirAll(specDir, 0o755)
				copyValidTestdata(t, "spec.yaml", specDir)
				copyValidTestdata(t, "plan.yaml", specDir)
				copyValidTestdata(t, "tasks.yaml", specDir)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   true,
			wantMissing: []string{},
		},
		"analyze stage - all artifacts missing": {
			stage: StageAnalyze,
			setupFunc: func(_ *testing.T, specDir string) func() {
				os.MkdirAll(specDir, 0o755)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   false,
			wantMissing: []string{"spec.yaml", "plan.yaml", "tasks.yaml"},
		},
		"analyze stage - only plan.yaml missing": {
			stage: StageAnalyze,
			setupFunc: func(t *testing.T, specDir string) func() {
				os.MkdirAll(specDir, 0o755)
				copyValidTestdata(t, "spec.yaml", specDir)
				copyValidTestdata(t, "tasks.yaml", specDir)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   false,
			wantMissing: []string{"plan.yaml"},
		},
		"constitution stage - no prerequisites required": {
			stage: StageConstitution,
			setupFunc: func(_ *testing.T, specDir string) func() {
				os.MkdirAll(specDir, 0o755)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   true,
			wantMissing: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			specDir := t.TempDir()
			cleanup := tc.setupFunc(t, specDir)
			defer cleanup()

			result := ValidateStagePrerequisites(tc.stage, specDir)

			assert.Equal(t, tc.wantValid, result.Valid, "Valid should match expected")
			assert.ElementsMatch(t, tc.wantMissing, result.MissingArtifacts,
				"MissingArtifacts should match expected")

			if tc.wantValid {
				assert.Empty(t, result.ErrorMessage, "ErrorMessage should be empty when valid")
			} else {
				assert.NotEmpty(t, result.ErrorMessage, "ErrorMessage should not be empty when invalid")
			}
		})
	}
}

// TestValidateStagePrerequisitesWithInvalidSchema tests that ValidateStagePrerequisites
// correctly detects schema validation errors in prerequisite artifacts.
func TestValidateStagePrerequisitesWithInvalidSchema(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stage           Stage
		setupFunc       func(t *testing.T, specDir string)
		wantValid       bool
		wantInvalid     []string
		wantErrContains string
	}{
		"plan stage - invalid spec.yaml schema": {
			stage: StagePlan,
			setupFunc: func(_ *testing.T, specDir string) {
				os.MkdirAll(specDir, 0o755)
				// Write invalid spec.yaml content
				os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte("invalid: yaml"), 0o644)
			},
			wantValid:       false,
			wantInvalid:     []string{"spec.yaml"},
			wantErrContains: "Invalid artifact schema",
		},
		"tasks stage - invalid plan.yaml schema": {
			stage: StageTasks,
			setupFunc: func(_ *testing.T, specDir string) {
				os.MkdirAll(specDir, 0o755)
				// Write invalid plan.yaml content
				os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte("invalid: data"), 0o644)
			},
			wantValid:       false,
			wantInvalid:     []string{"plan.yaml"},
			wantErrContains: "Invalid artifact schema",
		},
		"implement stage - invalid tasks.yaml schema": {
			stage: StageImplement,
			setupFunc: func(_ *testing.T, specDir string) {
				os.MkdirAll(specDir, 0o755)
				// Write invalid tasks.yaml content
				os.WriteFile(filepath.Join(specDir, "tasks.yaml"), []byte("invalid: content"), 0o644)
			},
			wantValid:       false,
			wantInvalid:     []string{"tasks.yaml"},
			wantErrContains: "Invalid artifact schema",
		},
		"analyze stage - one valid, two invalid": {
			stage: StageAnalyze,
			setupFunc: func(t *testing.T, specDir string) {
				os.MkdirAll(specDir, 0o755)
				copyValidTestdata(t, "spec.yaml", specDir)
				os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte("bad: data"), 0o644)
				os.WriteFile(filepath.Join(specDir, "tasks.yaml"), []byte("bad: data"), 0o644)
			},
			wantValid:       false,
			wantInvalid:     []string{"plan.yaml", "tasks.yaml"},
			wantErrContains: "Invalid artifact schema",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			specDir := t.TempDir()
			tc.setupFunc(t, specDir)

			result := ValidateStagePrerequisites(tc.stage, specDir)

			assert.Equal(t, tc.wantValid, result.Valid, "Valid should match expected")
			assert.Empty(t, result.MissingArtifacts, "Should have no missing artifacts")
			assert.Len(t, result.InvalidArtifacts, len(tc.wantInvalid),
				"Should have expected number of invalid artifacts")

			for _, artifact := range tc.wantInvalid {
				assert.Contains(t, result.InvalidArtifacts, artifact,
					"InvalidArtifacts should contain %s", artifact)
			}

			assert.Contains(t, result.ErrorMessage, tc.wantErrContains,
				"ErrorMessage should contain expected text")
		})
	}
}

// TestGenerateArtifactMissingError tests error message generation
func TestGenerateArtifactMissingError(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		missingArtifacts []string
		wantContains     []string
	}{
		"single missing artifact - spec.yaml": {
			missingArtifacts: []string{"spec.yaml"},
			wantContains: []string{
				"spec.yaml not found",
				"autospec specify",
			},
		},
		"single missing artifact - plan.yaml": {
			missingArtifacts: []string{"plan.yaml"},
			wantContains: []string{
				"plan.yaml not found",
				"autospec plan",
			},
		},
		"single missing artifact - tasks.yaml": {
			missingArtifacts: []string{"tasks.yaml"},
			wantContains: []string{
				"tasks.yaml not found",
				"autospec tasks",
			},
		},
		"multiple missing artifacts": {
			missingArtifacts: []string{"spec.yaml", "plan.yaml", "tasks.yaml"},
			wantContains: []string{
				"Missing required artifacts",
				"spec.yaml",
				"plan.yaml",
				"tasks.yaml",
				"autospec specify",
				"autospec plan",
				"autospec tasks",
			},
		},
		"two missing artifacts": {
			missingArtifacts: []string{"plan.yaml", "tasks.yaml"},
			wantContains: []string{
				"Missing required artifacts",
				"plan.yaml",
				"tasks.yaml",
				"autospec plan",
				"autospec tasks",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			errMsg := GenerateArtifactMissingError(tc.missingArtifacts)

			for _, want := range tc.wantContains {
				assert.Contains(t, errMsg, want,
					"Error message should contain: %s", want)
			}
		})
	}
}

// TestGetRemediationCommand tests remediation command mapping
func TestGetRemediationCommand(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		artifact string
		want     string
	}{
		"constitution.yaml": {
			artifact: "constitution.yaml",
			want:     "autospec constitution",
		},
		"spec.yaml": {
			artifact: "spec.yaml",
			want:     "autospec specify",
		},
		"plan.yaml": {
			artifact: "plan.yaml",
			want:     "autospec plan",
		},
		"tasks.yaml": {
			artifact: "tasks.yaml",
			want:     "autospec tasks",
		},
		"unknown artifact": {
			artifact: "unknown.yaml",
			want:     "autospec (unknown artifact: unknown.yaml)",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := GetRemediationCommand(tc.artifact)
			assert.Equal(t, tc.want, got, "GetRemediationCommand(%s)", tc.artifact)
		})
	}
}

// BenchmarkValidateStagePrerequisites benchmarks validation performance
// Target: <10ms for all stages combined
func BenchmarkValidateStagePrerequisites(b *testing.B) {
	// Setup with all artifacts present
	specDir := b.TempDir()
	os.WriteFile(specDir+"/spec.yaml", []byte("test"), 0o644)
	os.WriteFile(specDir+"/plan.yaml", []byte("test"), 0o644)
	os.WriteFile(specDir+"/tasks.yaml", []byte("test"), 0o644)

	stages := []Stage{
		StageSpecify, StagePlan, StageTasks, StageImplement,
		StageClarify, StageChecklist, StageAnalyze,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, stage := range stages {
			_ = ValidateStagePrerequisites(stage, specDir)
		}
	}
}

// TestCheckSpecDirectory tests spec directory validation
func TestCheckSpecDirectory(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFunc func(t *testing.T) string
		wantErr   bool
		errMsg    string
	}{
		"directory exists": {
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				return dir
			},
			wantErr: false,
		},
		"directory does not exist": {
			setupFunc: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent")
			},
			wantErr: true,
			errMsg:  "spec directory not found",
		},
		"path is a file not directory": {
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				filePath := filepath.Join(dir, "test.txt")
				os.WriteFile(filePath, []byte("test"), 0o644)
				return filePath
			},
			wantErr: true,
			errMsg:  "not a directory",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			specDir := tc.setupFunc(t)
			err := CheckSpecDirectory(specDir)

			if tc.wantErr {
				assert.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestFindSpecsDirectory tests specs directory discovery
func TestFindSpecsDirectory(t *testing.T) {
	tests := map[string]struct {
		setupFunc func(t *testing.T) (specsDir string, cleanup func())
		wantErr   bool
	}{
		"specs directory exists at relative path": {
			setupFunc: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()
				specsDir := filepath.Join(tmpDir, "specs")
				os.MkdirAll(specsDir, 0o755)

				origDir, _ := os.Getwd()
				os.Chdir(tmpDir)
				return "specs", func() { os.Chdir(origDir) }
			},
			wantErr: false,
		},
		"specs directory does not exist": {
			setupFunc: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()
				origDir, _ := os.Getwd()
				os.Chdir(tmpDir)
				return "nonexistent-specs", func() { os.Chdir(origDir) }
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			specsDir, cleanup := tc.setupFunc(t)
			defer cleanup()

			path, err := FindSpecsDirectory(specsDir)

			if tc.wantErr {
				assert.Error(t, err)
				assert.Empty(t, path)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, path)
			}
		})
	}
}

// copyValidTestdata copies a valid testdata artifact file to the specified directory
func copyValidTestdata(t *testing.T, artifact, destDir string) {
	t.Helper()
	var srcPath string
	switch artifact {
	case "spec.yaml":
		srcPath = filepath.Join("testdata", "spec", "valid", "spec.yaml")
	case "plan.yaml":
		srcPath = filepath.Join("testdata", "plan", "valid", "plan.yaml")
	case "tasks.yaml":
		srcPath = filepath.Join("testdata", "tasks", "valid", "tasks.yaml")
	default:
		t.Fatalf("unknown artifact: %s", artifact)
	}

	data, err := os.ReadFile(srcPath)
	require.NoError(t, err, "reading testdata file %s", srcPath)

	destPath := filepath.Join(destDir, artifact)
	err = os.WriteFile(destPath, data, 0o644)
	require.NoError(t, err, "writing artifact file %s", destPath)
}

// TestCheckArtifactDependencies tests artifact dependency checking
func TestCheckArtifactDependencies(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stages      []Stage
		setupFunc   func(t *testing.T, specDir string)
		wantPassed  bool
		wantMissing []string
		wantInvalid []string // artifacts that exist but have invalid schemas
	}{
		"plan stage with spec.yaml present": {
			stages: []Stage{StagePlan},
			setupFunc: func(t *testing.T, specDir string) {
				copyValidTestdata(t, "spec.yaml", specDir)
			},
			wantPassed:  true,
			wantMissing: []string{},
			wantInvalid: []string{},
		},
		"plan stage with spec.yaml missing": {
			stages:      []Stage{StagePlan},
			setupFunc:   func(t *testing.T, specDir string) {},
			wantPassed:  false,
			wantMissing: []string{"spec.yaml"},
			wantInvalid: []string{},
		},
		"plan stage with spec.yaml invalid": {
			stages: []Stage{StagePlan},
			setupFunc: func(t *testing.T, specDir string) {
				// Write invalid spec.yaml (missing required fields)
				err := os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte("invalid: yaml"), 0o644)
				require.NoError(t, err)
			},
			wantPassed:  false,
			wantMissing: []string{},
			wantInvalid: []string{"spec.yaml"},
		},
		"implement stage with tasks.yaml present": {
			stages: []Stage{StageImplement},
			setupFunc: func(t *testing.T, specDir string) {
				copyValidTestdata(t, "tasks.yaml", specDir)
			},
			wantPassed:  true,
			wantMissing: []string{},
			wantInvalid: []string{},
		},
		"implement stage with tasks.yaml missing": {
			stages:      []Stage{StageImplement},
			setupFunc:   func(t *testing.T, specDir string) {},
			wantPassed:  false,
			wantMissing: []string{"tasks.yaml"},
			wantInvalid: []string{},
		},
		"implement stage with tasks.yaml invalid": {
			stages: []Stage{StageImplement},
			setupFunc: func(t *testing.T, specDir string) {
				// Write invalid tasks.yaml (missing required fields)
				err := os.WriteFile(filepath.Join(specDir, "tasks.yaml"), []byte("tasks:\n  branch: foo"), 0o644)
				require.NoError(t, err)
			},
			wantPassed:  false,
			wantMissing: []string{},
			wantInvalid: []string{"tasks.yaml"},
		},
		"analyze stage with all artifacts present": {
			stages: []Stage{StageAnalyze},
			setupFunc: func(t *testing.T, specDir string) {
				copyValidTestdata(t, "spec.yaml", specDir)
				copyValidTestdata(t, "plan.yaml", specDir)
				copyValidTestdata(t, "tasks.yaml", specDir)
			},
			wantPassed:  true,
			wantMissing: []string{},
			wantInvalid: []string{},
		},
		"analyze stage with some artifacts missing": {
			stages: []Stage{StageAnalyze},
			setupFunc: func(t *testing.T, specDir string) {
				copyValidTestdata(t, "spec.yaml", specDir)
			},
			wantPassed:  false,
			wantMissing: []string{"plan.yaml", "tasks.yaml"},
			wantInvalid: []string{},
		},
		"analyze stage with one invalid artifact": {
			stages: []Stage{StageAnalyze},
			setupFunc: func(t *testing.T, specDir string) {
				copyValidTestdata(t, "spec.yaml", specDir)
				// Write invalid plan.yaml
				err := os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte("plan:\n  branch: foo"), 0o644)
				require.NoError(t, err)
				copyValidTestdata(t, "tasks.yaml", specDir)
			},
			wantPassed:  false,
			wantMissing: []string{},
			wantInvalid: []string{"plan.yaml"},
		},
		"mixed missing and invalid artifacts": {
			stages: []Stage{StageAnalyze},
			setupFunc: func(t *testing.T, specDir string) {
				// spec.yaml is missing (no file created)
				// plan.yaml is invalid
				err := os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte("plan:\n  branch: foo"), 0o644)
				require.NoError(t, err)
				copyValidTestdata(t, "tasks.yaml", specDir)
			},
			wantPassed:  false,
			wantMissing: []string{"spec.yaml"},
			wantInvalid: []string{"plan.yaml"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			specDir := t.TempDir()
			tc.setupFunc(t, specDir)

			stageConfig := NewStageConfig()
			for _, stage := range tc.stages {
				switch stage {
				case StageSpecify:
					stageConfig.Specify = true
				case StagePlan:
					stageConfig.Plan = true
				case StageTasks:
					stageConfig.Tasks = true
				case StageImplement:
					stageConfig.Implement = true
				case StageAnalyze:
					stageConfig.Analyze = true
				}
			}

			result := CheckArtifactDependencies(stageConfig, specDir)

			assert.Equal(t, tc.wantPassed, result.Passed)
			assert.ElementsMatch(t, tc.wantMissing, result.MissingArtifacts)

			// Check invalid artifacts
			invalidKeys := make([]string, 0, len(result.InvalidArtifacts))
			for k := range result.InvalidArtifacts {
				invalidKeys = append(invalidKeys, k)
			}
			assert.ElementsMatch(t, tc.wantInvalid, invalidKeys)

			if !tc.wantPassed {
				assert.NotEmpty(t, result.WarningMessage)
			}
		})
	}
}

// TestGeneratePrerequisiteError tests prerequisite error message generation
func TestGeneratePrerequisiteError(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stages       []Stage
		missing      []string
		invalid      map[string]string // artifact -> error message
		wantContains []string
	}{
		"missing spec.yaml": {
			stages:  []Stage{StagePlan},
			missing: []string{"spec.yaml"},
			invalid: nil,
			wantContains: []string{
				"Missing required prerequisite",
				"spec.yaml",
				"Generate spec.yaml",
			},
		},
		"missing plan.yaml": {
			stages:  []Stage{StageTasks},
			missing: []string{"plan.yaml"},
			invalid: nil,
			wantContains: []string{
				"plan.yaml",
				"Generate plan.yaml",
			},
		},
		"missing tasks.yaml": {
			stages:  []Stage{StageImplement},
			missing: []string{"tasks.yaml"},
			invalid: nil,
			wantContains: []string{
				"tasks.yaml",
				"Generate tasks.yaml",
			},
		},
		"multiple missing artifacts": {
			stages:  []Stage{StageAnalyze},
			missing: []string{"spec.yaml", "plan.yaml", "tasks.yaml"},
			invalid: nil,
			wantContains: []string{
				"spec.yaml",
				"plan.yaml",
				"tasks.yaml",
			},
		},
		"invalid spec.yaml": {
			stages:  []Stage{StagePlan},
			missing: nil,
			invalid: map[string]string{"spec.yaml": "schema validation failed"},
			wantContains: []string{
				"Invalid artifact schemas",
				"spec.yaml",
				"schema validation failed",
				"Regenerate spec.yaml",
			},
		},
		"mixed missing and invalid": {
			stages:  []Stage{StageAnalyze},
			missing: []string{"plan.yaml"},
			invalid: map[string]string{"spec.yaml": "missing required field"},
			wantContains: []string{
				"Missing required prerequisite",
				"plan.yaml",
				"Invalid artifact schemas",
				"spec.yaml",
				"missing required field",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stageConfig := NewStageConfig()
			for _, stage := range tc.stages {
				switch stage {
				case StagePlan:
					stageConfig.Plan = true
				case StageTasks:
					stageConfig.Tasks = true
				case StageImplement:
					stageConfig.Implement = true
				case StageAnalyze:
					stageConfig.Analyze = true
				}
			}

			errMsg := GeneratePrerequisiteError(stageConfig, tc.missing, tc.invalid)

			for _, want := range tc.wantContains {
				assert.Contains(t, errMsg, want)
			}
		})
	}
}

// TestContainsArtifact tests the artifact containment check
func TestContainsArtifact(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		artifacts []string
		artifact  string
		want      bool
	}{
		"artifact present": {
			artifacts: []string{"spec.yaml", "plan.yaml"},
			artifact:  "spec.yaml",
			want:      true,
		},
		"artifact not present": {
			artifacts: []string{"spec.yaml", "plan.yaml"},
			artifact:  "tasks.yaml",
			want:      false,
		},
		"empty list": {
			artifacts: []string{},
			artifact:  "spec.yaml",
			want:      false,
		},
		"single item present": {
			artifacts: []string{"spec.yaml"},
			artifact:  "spec.yaml",
			want:      true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := containsArtifact(tc.artifacts, tc.artifact)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestGeneratePrerequisiteWarning tests the deprecated alias
func TestGeneratePrerequisiteWarning(t *testing.T) {
	t.Parallel()

	stageConfig := NewStageConfig()
	stageConfig.Plan = true
	missing := []string{"spec.yaml"}

	// The warning should be the same as the error
	warning := GeneratePrerequisiteWarning(stageConfig, missing, nil)
	errMsg := GeneratePrerequisiteError(stageConfig, missing, nil)

	assert.Equal(t, errMsg, warning)
}

// TestValidateStagePrerequisitesPerformance verifies validation completes in <10ms
func TestValidateStagePrerequisitesPerformance(t *testing.T) {
	t.Parallel()

	// Setup with all artifacts present
	specDir := t.TempDir()
	os.WriteFile(specDir+"/spec.yaml", []byte("test"), 0o644)
	os.WriteFile(specDir+"/plan.yaml", []byte("test"), 0o644)
	os.WriteFile(specDir+"/tasks.yaml", []byte("test"), 0o644)

	stages := []Stage{
		StageSpecify, StagePlan, StageTasks, StageImplement,
		StageClarify, StageChecklist, StageAnalyze,
	}

	// Run validation for all stages and measure time
	iterations := 100
	totalDuration := int64(0)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		for _, stage := range stages {
			_ = ValidateStagePrerequisites(stage, specDir)
		}
		totalDuration += time.Since(start).Nanoseconds()
	}

	avgDuration := time.Duration(totalDuration / int64(iterations))
	t.Logf("Average duration for all stages: %v", avgDuration)

	// Assert validation completes in under 10ms
	assert.Less(t, avgDuration, 10*time.Millisecond,
		"Validation for all stages should complete in <10ms")
}

// errorReader is a test helper that always returns an error on Read
type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

// TestPromptUserToContinueWithReader tests the PromptUserToContinueWithReader function
// with various inputs including y, n, yes, no, Y, N, empty string, EOF, and read errors.
func TestPromptUserToContinueWithReader(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input       string    // Input to simulate
		useReader   io.Reader // Custom reader (if nil, uses strings.NewReader(input))
		wantCont    bool      // Expected continue result
		wantErr     bool      // Whether an error is expected
		errContains string    // Error message substring to check
	}{
		"lowercase y - should continue": {
			input:    "y\n",
			wantCont: true,
			wantErr:  false,
		},
		"lowercase yes - should continue": {
			input:    "yes\n",
			wantCont: true,
			wantErr:  false,
		},
		"uppercase Y - should continue": {
			input:    "Y\n",
			wantCont: true,
			wantErr:  false,
		},
		"uppercase YES - should continue": {
			input:    "YES\n",
			wantCont: true,
			wantErr:  false,
		},
		"mixed case Yes - should continue": {
			input:    "Yes\n",
			wantCont: true,
			wantErr:  false,
		},
		"lowercase n - should not continue": {
			input:    "n\n",
			wantCont: false,
			wantErr:  false,
		},
		"lowercase no - should not continue": {
			input:    "no\n",
			wantCont: false,
			wantErr:  false,
		},
		"uppercase N - should not continue": {
			input:    "N\n",
			wantCont: false,
			wantErr:  false,
		},
		"uppercase NO - should not continue": {
			input:    "NO\n",
			wantCont: false,
			wantErr:  false,
		},
		"empty input - should not continue": {
			input:    "\n",
			wantCont: false,
			wantErr:  false,
		},
		"whitespace input - should not continue": {
			input:    "   \n",
			wantCont: false,
			wantErr:  false,
		},
		"invalid input - should not continue": {
			input:    "maybe\n",
			wantCont: false,
			wantErr:  false,
		},
		"y with leading whitespace - should continue": {
			input:    "  y\n",
			wantCont: true,
			wantErr:  false,
		},
		"y with trailing whitespace - should continue": {
			input:    "y  \n",
			wantCont: true,
			wantErr:  false,
		},
		"EOF - should return false without error": {
			input:    "", // Empty string reader returns EOF
			wantCont: false,
			wantErr:  false,
		},
		"read error - should return error": {
			useReader:   &errorReader{err: errors.New("test read error")},
			wantCont:    false,
			wantErr:     true,
			errContains: "reading user input",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var reader io.Reader
			if tc.useReader != nil {
				reader = tc.useReader
			} else {
				reader = strings.NewReader(tc.input)
			}

			gotCont, err := PromptUserToContinueWithReader("test warning", reader)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.wantCont, gotCont,
				"PromptUserToContinueWithReader should return %v for input %q", tc.wantCont, tc.input)
		})
	}
}

// TestPromptUserToContinue tests the original function delegates correctly
func TestPromptUserToContinue(t *testing.T) {
	// This test just verifies the function signature and basic behavior
	// Full coverage is handled by TestPromptUserToContinueWithReader
	// We can't easily test this without mocking os.Stdin
	t.Skip("Cannot test without stdin mocking - covered by TestPromptUserToContinueWithReader")
}

// TestRunPreflightChecks_WithMock tests runPreflightChecks using mock injection.
// This tests the WorkflowOrchestrator.runPreflightChecks method with various scenarios
// using the mockPreflightChecker to simulate different preflight check outcomes.
func TestRunPreflightChecks_WithMock(t *testing.T) {
	tests := map[string]struct {
		setupMock   func() *mockPreflightChecker
		wantErr     bool
		errContains string
		wantPrompt  bool // Whether PromptUser should be called
	}{
		"pass scenario - all checks green": {
			setupMock: func() *mockPreflightChecker {
				return newMockPreflightChecker() // Default is all passing
			},
			wantErr:    false,
			wantPrompt: false,
		},
		"warn scenario - user prompted and continues": {
			setupMock: func() *mockPreflightChecker {
				return newMockPreflightChecker().
					WithMissingDirs([]string{".autospec/"}, "Missing autospec directory").
					WithPromptUserResult(true)
			},
			wantErr:    false,
			wantPrompt: true,
		},
		"warn scenario - user prompted and aborts": {
			setupMock: func() *mockPreflightChecker {
				return newMockPreflightChecker().
					WithMissingDirs([]string{".autospec/"}, "Missing autospec directory").
					WithPromptUserResult(false)
			},
			wantErr:     true,
			errContains: "user aborted",
			wantPrompt:  true,
		},
		"fail scenario - RunChecks returns error": {
			setupMock: func() *mockPreflightChecker {
				return newMockPreflightChecker().
					WithRunChecksError(errors.New("system error"))
			},
			wantErr:     true,
			errContains: "pre-flight checks failed",
			wantPrompt:  false,
		},
		"fail scenario - critical failure (no warning message)": {
			setupMock: func() *mockPreflightChecker {
				return newMockPreflightChecker().
					WithFailedChecks([]string{"claude CLI not found"}, "")
			},
			wantErr:     true,
			errContains: "pre-flight checks failed",
			wantPrompt:  false,
		},
		"fail scenario - prompt user returns error": {
			setupMock: func() *mockPreflightChecker {
				return newMockPreflightChecker().
					WithMissingDirs([]string{".autospec/"}, "Missing autospec directory").
					WithPromptUserError(errors.New("stdin closed"))
			},
			wantErr:     true,
			errContains: "prompting user",
			wantPrompt:  true,
		},
		"warn scenario - failed checks with warning message": {
			setupMock: func() *mockPreflightChecker {
				return newMockPreflightChecker().
					WithFailedChecks([]string{"optional check failed"}, "Warning: optional check failed").
					WithPromptUserResult(true)
			},
			wantErr:    false,
			wantPrompt: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mock := tc.setupMock()

			// Create a WorkflowOrchestrator with the mock
			orch := &WorkflowOrchestrator{
				PreflightChecker: mock,
			}

			// Run the method under test
			err := orch.runPreflightChecks()

			// Verify error expectations
			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
			}

			// Verify RunChecks was called
			assert.True(t, mock.RunChecksCalled, "RunChecks should be called")
			assert.Equal(t, 1, mock.RunChecksCallCount, "RunChecks should be called exactly once")

			// Verify PromptUser call expectations
			assert.Equal(t, tc.wantPrompt, mock.PromptUserCalled,
				"PromptUser called status should match expected")
		})
	}
}

// TestGetPreflightChecker tests the nil-safety of getPreflightChecker.
func TestGetPreflightChecker(t *testing.T) {
	tests := map[string]struct {
		checker  PreflightChecker
		wantType string
	}{
		"nil checker returns default": {
			checker:  nil,
			wantType: "*workflow.DefaultPreflightChecker",
		},
		"injected checker is returned": {
			checker:  newMockPreflightChecker(),
			wantType: "*workflow.mockPreflightChecker",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			orch := &WorkflowOrchestrator{
				PreflightChecker: tc.checker,
			}

			result := orch.getPreflightChecker()

			require.NotNil(t, result, "getPreflightChecker should never return nil")

			// Verify type by checking type name
			gotType := fmt.Sprintf("%T", result)
			assert.Equal(t, tc.wantType, gotType,
				"getPreflightChecker should return correct type")
		})
	}
}

// TestDefaultPreflightChecker tests that DefaultPreflightChecker properly implements the interface.
func TestDefaultPreflightChecker(t *testing.T) {
	// Verify the DefaultPreflightChecker implements PreflightChecker interface
	var _ PreflightChecker = (*DefaultPreflightChecker)(nil)
	var _ PreflightChecker = NewDefaultPreflightChecker()

	// Test that NewDefaultPreflightChecker returns a non-nil instance
	checker := NewDefaultPreflightChecker()
	assert.NotNil(t, checker, "NewDefaultPreflightChecker should return non-nil")

	// We can't easily test RunChecks and PromptUser without side effects,
	// but we can verify they don't panic with reasonable inputs
	t.Run("RunChecks does not panic", func(t *testing.T) {
		// Change to temp dir to avoid modifying actual repo
		tmpDir := t.TempDir()
		origDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// This should not panic
		result, err := checker.RunChecks()
		// We don't care about the result, just that it doesn't panic
		_ = result
		_ = err
	})
}
