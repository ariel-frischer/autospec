package workflow

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunPreflightChecks tests the pre-flight validation logic
func TestRunPreflightChecks(t *testing.T) {
	tests := map[string]struct {
		setupDirs   []string // Directories to create in temp dir
		wantPassed  bool
		wantMissing int
		wantFailed  int
	}{
		"all checks pass": {
			setupDirs:   []string{".claude/commands", ".autospec"},
			wantPassed:  true,
			wantMissing: 0,
			wantFailed:  0,
		},
		"missing .claude/commands directory": {
			setupDirs:   []string{".autospec"},
			wantPassed:  false,
			wantMissing: 1,
		},
		"missing .autospec directory": {
			setupDirs:   []string{".claude/commands"},
			wantPassed:  false,
			wantMissing: 1,
		},
		"missing both directories": {
			setupDirs:   []string{},
			wantPassed:  false,
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
				require.NoError(t, os.MkdirAll(dir, 0755))
			}

			// Run pre-flight checks
			result, err := RunPreflightChecks()
			require.NoError(t, err)

			// Verify results
			assert.Equal(t, tc.wantPassed, result.Passed,
				"Passed status should match")
			if tc.wantMissing > 0 {
				assert.Len(t, result.MissingDirs, tc.wantMissing,
					"Should detect missing directories")
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
	require.NoError(t, os.MkdirAll(".claude/commands", 0755))
	require.NoError(t, os.MkdirAll(".autospec", 0755))

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
	os.MkdirAll(".claude/commands", 0755)
	os.MkdirAll(".autospec", 0755)

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
		"autospec constitution exists (.yaml)": {
			setupFiles:   map[string]string{".autospec/memory/constitution.yaml": "project_name: Test"},
			wantExists:   true,
			wantPath:     ".autospec/memory/constitution.yaml",
			wantErrEmpty: true,
		},
		"autospec constitution exists (.yml)": {
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
		"yaml takes precedence over yml": {
			setupFiles: map[string]string{
				".autospec/memory/constitution.yaml": "project_name: YAML",
				".autospec/memory/constitution.yml":  "project_name: YML",
			},
			wantExists:   true,
			wantPath:     ".autospec/memory/constitution.yaml",
			wantErrEmpty: true,
		},
		"autospec takes precedence over specify": {
			setupFiles: map[string]string{
				".autospec/memory/constitution.yaml": "project_name: Autospec",
				".specify/memory/constitution.yaml":  "project_name: Specify",
			},
			wantExists:   true,
			wantPath:     ".autospec/memory/constitution.yaml",
			wantErrEmpty: true,
		},
		"no constitution exists": {
			setupFiles:   map[string]string{},
			wantExists:   false,
			wantPath:     "",
			wantErrEmpty: false,
		},
		"directories exist but no constitution file": {
			setupFiles:   map[string]string{".autospec/memory/.keep": "", ".specify/memory/.keep": ""},
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
				require.NoError(t, os.MkdirAll(dir, 0755))
				require.NoError(t, os.WriteFile(path, []byte(content), 0644))
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
	assert.Contains(t, errMsg, ".specify/memory/constitution.yaml")
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
	os.MkdirAll(".autospec/memory", 0755)
	os.WriteFile(".autospec/memory/constitution.yaml", []byte("project_name: Test"), 0644)

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
		setupFunc   func(specDir string) func()
		wantValid   bool
		wantMissing []string
	}{
		"specify stage - no prerequisites required": {
			stage: StageSpecify,
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   true,
			wantMissing: []string{},
		},
		"plan stage - spec.yaml exists": {
			stage: StagePlan,
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				os.WriteFile(specDir+"/spec.yaml", []byte("test"), 0644)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   true,
			wantMissing: []string{},
		},
		"plan stage - spec.yaml missing": {
			stage: StagePlan,
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   false,
			wantMissing: []string{"spec.yaml"},
		},
		"tasks stage - plan.yaml exists": {
			stage: StageTasks,
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				os.WriteFile(specDir+"/plan.yaml", []byte("test"), 0644)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   true,
			wantMissing: []string{},
		},
		"tasks stage - plan.yaml missing": {
			stage: StageTasks,
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   false,
			wantMissing: []string{"plan.yaml"},
		},
		"implement stage - tasks.yaml exists": {
			stage: StageImplement,
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				os.WriteFile(specDir+"/tasks.yaml", []byte("test"), 0644)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   true,
			wantMissing: []string{},
		},
		"implement stage - tasks.yaml missing": {
			stage: StageImplement,
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   false,
			wantMissing: []string{"tasks.yaml"},
		},
		"clarify stage - spec.yaml exists": {
			stage: StageClarify,
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				os.WriteFile(specDir+"/spec.yaml", []byte("test"), 0644)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   true,
			wantMissing: []string{},
		},
		"clarify stage - spec.yaml missing": {
			stage: StageClarify,
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   false,
			wantMissing: []string{"spec.yaml"},
		},
		"checklist stage - spec.yaml exists": {
			stage: StageChecklist,
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				os.WriteFile(specDir+"/spec.yaml", []byte("test"), 0644)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   true,
			wantMissing: []string{},
		},
		"checklist stage - spec.yaml missing": {
			stage: StageChecklist,
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   false,
			wantMissing: []string{"spec.yaml"},
		},
		"analyze stage - all artifacts exist": {
			stage: StageAnalyze,
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				os.WriteFile(specDir+"/spec.yaml", []byte("test"), 0644)
				os.WriteFile(specDir+"/plan.yaml", []byte("test"), 0644)
				os.WriteFile(specDir+"/tasks.yaml", []byte("test"), 0644)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   true,
			wantMissing: []string{},
		},
		"analyze stage - all artifacts missing": {
			stage: StageAnalyze,
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   false,
			wantMissing: []string{"spec.yaml", "plan.yaml", "tasks.yaml"},
		},
		"analyze stage - only plan.yaml missing": {
			stage: StageAnalyze,
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				os.WriteFile(specDir+"/spec.yaml", []byte("test"), 0644)
				os.WriteFile(specDir+"/tasks.yaml", []byte("test"), 0644)
				return func() { os.RemoveAll(specDir) }
			},
			wantValid:   false,
			wantMissing: []string{"plan.yaml"},
		},
		"constitution stage - no prerequisites required": {
			stage: StageConstitution,
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
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
			cleanup := tc.setupFunc(specDir)
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
	os.WriteFile(specDir+"/spec.yaml", []byte("test"), 0644)
	os.WriteFile(specDir+"/plan.yaml", []byte("test"), 0644)
	os.WriteFile(specDir+"/tasks.yaml", []byte("test"), 0644)

	stages := []Stage{StageSpecify, StagePlan, StageTasks, StageImplement,
		StageClarify, StageChecklist, StageAnalyze}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, stage := range stages {
			_ = ValidateStagePrerequisites(stage, specDir)
		}
	}
}

// TestValidateStagePrerequisitesPerformance verifies validation completes in <10ms
func TestValidateStagePrerequisitesPerformance(t *testing.T) {
	t.Parallel()

	// Setup with all artifacts present
	specDir := t.TempDir()
	os.WriteFile(specDir+"/spec.yaml", []byte("test"), 0644)
	os.WriteFile(specDir+"/plan.yaml", []byte("test"), 0644)
	os.WriteFile(specDir+"/tasks.yaml", []byte("test"), 0644)

	stages := []Stage{StageSpecify, StagePlan, StageTasks, StageImplement,
		StageClarify, StageChecklist, StageAnalyze}

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
