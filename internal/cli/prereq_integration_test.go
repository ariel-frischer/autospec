package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunCommandPrerequisiteValidation tests that the run command correctly
// validates prerequisites based on the selected stages.
// This covers US-006: Smart prerequisite checking for run command with stage flags.
func TestRunCommandPrerequisiteValidation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stageConfig         *workflow.StageConfig
		setupFunc           func(specDir string) func()
		wantMissingArtifact bool
		wantMissing         []string
	}{
		"run -spt only checks constitution (no external artifacts needed)": {
			stageConfig: &workflow.StageConfig{Specify: true, Plan: true, Tasks: true},
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				return func() { os.RemoveAll(specDir) }
			},
			wantMissingArtifact: false,
			wantMissing:         []string{},
		},
		"run -pti checks spec.yaml (plan needs spec.yaml)": {
			stageConfig: &workflow.StageConfig{Plan: true, Tasks: true, Implement: true},
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				return func() { os.RemoveAll(specDir) }
			},
			wantMissingArtifact: true,
			wantMissing:         []string{"spec.yaml"},
		},
		"run -pti with spec.yaml passes (plan produces plan.yaml, tasks produces tasks.yaml)": {
			stageConfig: &workflow.StageConfig{Plan: true, Tasks: true, Implement: true},
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte("test"), 0644)
				return func() { os.RemoveAll(specDir) }
			},
			wantMissingArtifact: false, // spec.yaml present, plan produces plan.yaml, tasks produces tasks.yaml
			wantMissing:         []string{},
		},
		"run -ti checks plan.yaml (tasks needs plan.yaml)": {
			stageConfig: &workflow.StageConfig{Tasks: true, Implement: true},
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				return func() { os.RemoveAll(specDir) }
			},
			wantMissingArtifact: true,
			wantMissing:         []string{"plan.yaml"},
		},
		"run -ti with plan.yaml passes (tasks produces tasks.yaml)": {
			stageConfig: &workflow.StageConfig{Tasks: true, Implement: true},
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte("test"), 0644)
				return func() { os.RemoveAll(specDir) }
			},
			wantMissingArtifact: false, // plan.yaml present, tasks produces tasks.yaml for implement
			wantMissing:         []string{},
		},
		"run -i checks tasks.yaml": {
			stageConfig: &workflow.StageConfig{Implement: true},
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				return func() { os.RemoveAll(specDir) }
			},
			wantMissingArtifact: true,
			wantMissing:         []string{"tasks.yaml"},
		},
		"run -a only checks constitution (specify produces spec.yaml)": {
			stageConfig: func() *workflow.StageConfig {
				sc := workflow.NewStageConfig()
				sc.SetAll()
				return sc
			}(),
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				return func() { os.RemoveAll(specDir) }
			},
			wantMissingArtifact: false,
			wantMissing:         []string{},
		},
		"run -p checks spec.yaml": {
			stageConfig: &workflow.StageConfig{Plan: true},
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				return func() { os.RemoveAll(specDir) }
			},
			wantMissingArtifact: true,
			wantMissing:         []string{"spec.yaml"},
		},
		"run -t checks plan.yaml": {
			stageConfig: &workflow.StageConfig{Tasks: true},
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				return func() { os.RemoveAll(specDir) }
			},
			wantMissingArtifact: true,
			wantMissing:         []string{"plan.yaml"},
		},
		"run with clarify checks spec.yaml": {
			stageConfig: &workflow.StageConfig{Clarify: true},
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				return func() { os.RemoveAll(specDir) }
			},
			wantMissingArtifact: true,
			wantMissing:         []string{"spec.yaml"},
		},
		"run with checklist checks spec.yaml": {
			stageConfig: &workflow.StageConfig{Checklist: true},
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				return func() { os.RemoveAll(specDir) }
			},
			wantMissingArtifact: true,
			wantMissing:         []string{"spec.yaml"},
		},
		"run with analyze checks all artifacts": {
			stageConfig: &workflow.StageConfig{Analyze: true},
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				return func() { os.RemoveAll(specDir) }
			},
			wantMissingArtifact: true,
			wantMissing:         []string{"spec.yaml", "plan.yaml", "tasks.yaml"},
		},
		"run -sr does not need spec.yaml (specify produces it)": {
			stageConfig: &workflow.StageConfig{Specify: true, Clarify: true},
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				return func() { os.RemoveAll(specDir) }
			},
			wantMissingArtifact: false,
			wantMissing:         []string{},
		},
		"run -sptiz (all + analyze) no external requirements": {
			stageConfig: &workflow.StageConfig{Specify: true, Plan: true, Tasks: true, Implement: true, Analyze: true},
			setupFunc: func(specDir string) func() {
				os.MkdirAll(specDir, 0755)
				return func() { os.RemoveAll(specDir) }
			},
			wantMissingArtifact: false,
			wantMissing:         []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			specDir := t.TempDir()
			cleanup := tc.setupFunc(specDir)
			defer cleanup()

			// Use CheckArtifactDependencies which is what run command uses
			result := workflow.CheckArtifactDependencies(tc.stageConfig, specDir)

			if tc.wantMissingArtifact {
				assert.False(t, result.Passed, "Passed should be false when artifacts are missing")
				assert.True(t, result.RequiresConfirmation, "RequiresConfirmation should be true")
				assert.ElementsMatch(t, tc.wantMissing, result.MissingArtifacts,
					"MissingArtifacts should match expected")
			} else {
				assert.True(t, result.Passed, "Passed should be true when no external artifacts needed")
				assert.False(t, result.RequiresConfirmation, "RequiresConfirmation should be false")
				assert.Empty(t, result.MissingArtifacts, "MissingArtifacts should be empty")
			}
		})
	}
}

// TestRunCommandPrerequisiteErrorMessages verifies error messages include remediation.
func TestRunCommandPrerequisiteErrorMessages(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stageConfig    *workflow.StageConfig
		wantContains   []string
		setupArtifacts []string // artifacts to create in spec dir
	}{
		"missing spec.yaml shows run -s remediation": {
			stageConfig:    &workflow.StageConfig{Plan: true},
			wantContains:   []string{"spec.yaml", "autospec run -s"},
			setupArtifacts: []string{},
		},
		"missing plan.yaml shows run -p remediation": {
			stageConfig:    &workflow.StageConfig{Tasks: true},
			wantContains:   []string{"plan.yaml", "autospec run -p"},
			setupArtifacts: []string{},
		},
		"missing tasks.yaml shows run -t remediation": {
			stageConfig:    &workflow.StageConfig{Implement: true},
			wantContains:   []string{"tasks.yaml", "autospec run -t"},
			setupArtifacts: []string{},
		},
		"analyze missing all shows all remediation commands": {
			stageConfig: &workflow.StageConfig{Analyze: true},
			wantContains: []string{
				"spec.yaml",
				"plan.yaml",
				"tasks.yaml",
				"autospec run -s",
				"autospec run -p",
				"autospec run -t",
			},
			setupArtifacts: []string{},
		},
		"analyze missing only plan shows plan remediation": {
			stageConfig:    &workflow.StageConfig{Analyze: true},
			wantContains:   []string{"plan.yaml", "autospec run -p"},
			setupArtifacts: []string{"spec.yaml", "tasks.yaml"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			specDir := t.TempDir()

			// Create specified artifacts
			for _, artifact := range tc.setupArtifacts {
				err := os.WriteFile(filepath.Join(specDir, artifact), []byte("test"), 0644)
				require.NoError(t, err)
			}

			result := workflow.CheckArtifactDependencies(tc.stageConfig, specDir)

			// Only check error message if there are missing artifacts
			if len(result.MissingArtifacts) > 0 {
				assert.NotEmpty(t, result.WarningMessage, "Error message should not be empty")
				// Verify it's an error message, not a warning
				assert.Contains(t, result.WarningMessage, "Error:",
					"Message should be an error, not a warning")
				for _, want := range tc.wantContains {
					assert.Contains(t, result.WarningMessage, want,
						"Error message should contain: %s", want)
				}
			}
		})
	}
}

// TestConstitutionCheckForRunCommand verifies constitution is checked before run command.
func TestConstitutionCheckForRunCommand(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFunc  func() func()
		wantExists bool
		wantErrMsg bool
	}{
		"constitution exists - check passes": {
			setupFunc: func() func() {
				os.MkdirAll(".autospec/memory", 0755)
				os.WriteFile(".autospec/memory/constitution.yaml", []byte("test"), 0644)
				return func() { os.RemoveAll(".autospec") }
			},
			wantExists: true,
			wantErrMsg: false,
		},
		"constitution missing - check fails with remediation": {
			setupFunc: func() func() {
				os.RemoveAll(".autospec")
				os.RemoveAll(".specify")
				return func() {}
			},
			wantExists: false,
			wantErrMsg: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run in parallel due to file system operations in cwd
			cleanup := tc.setupFunc()
			defer cleanup()

			result := workflow.CheckConstitutionExists()

			assert.Equal(t, tc.wantExists, result.Exists, "Exists should match expected")
			if tc.wantErrMsg {
				assert.NotEmpty(t, result.ErrorMessage, "ErrorMessage should not be empty")
				assert.Contains(t, result.ErrorMessage, "autospec constitution",
					"Error should suggest running autospec constitution")
			} else {
				assert.Empty(t, result.ErrorMessage, "ErrorMessage should be empty when exists")
			}
		})
	}
}

// TestGetAllRequiredArtifactsForRunCommand tests the smart artifact requirement calculation.
func TestGetAllRequiredArtifactsForRunCommand(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stageConfig *workflow.StageConfig
		want        []string
	}{
		"-spt: specify produces spec.yaml for plan, plan produces plan.yaml for tasks": {
			stageConfig: &workflow.StageConfig{Specify: true, Plan: true, Tasks: true},
			want:        []string{}, // No external requirements
		},
		"-pti: plan needs spec.yaml (external), tasks produces tasks.yaml for implement": {
			stageConfig: &workflow.StageConfig{Plan: true, Tasks: true, Implement: true},
			want:        []string{"spec.yaml"}, // Only spec.yaml is external; plan produces plan.yaml, tasks produces tasks.yaml
		},
		"-ti: tasks needs plan.yaml (external), tasks produces tasks.yaml for implement": {
			stageConfig: &workflow.StageConfig{Tasks: true, Implement: true},
			want:        []string{"plan.yaml"}, // Only plan.yaml is external; tasks produces tasks.yaml
		},
		"-a: all core stages - no external requirements": {
			stageConfig: func() *workflow.StageConfig {
				sc := workflow.NewStageConfig()
				sc.SetAll()
				return sc
			}(),
			want: []string{}, // Full chain produces everything
		},
		"-pi: plan needs spec.yaml, implement needs tasks.yaml (not produced)": {
			stageConfig: &workflow.StageConfig{Plan: true, Implement: true},
			want:        []string{"spec.yaml", "tasks.yaml"}, // spec.yaml for plan, tasks.yaml for implement (tasks stage not selected)
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := tc.stageConfig.GetAllRequiredArtifacts()

			// Convert to maps for comparison (order doesn't matter)
			gotMap := make(map[string]bool)
			for _, a := range got {
				gotMap[a] = true
			}
			wantMap := make(map[string]bool)
			for _, a := range tc.want {
				wantMap[a] = true
			}

			assert.Equal(t, len(wantMap), len(gotMap),
				"GetAllRequiredArtifacts() returned %v, want %v", got, tc.want)
			for artifact := range wantMap {
				assert.True(t, gotMap[artifact],
					"GetAllRequiredArtifacts() missing %s, got %v", artifact, got)
			}
		})
	}
}

// TestRunCommandStageCountForConstitutionCheck tests constitution check logic.
func TestRunCommandStageCountForConstitutionCheck(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stageConfig              *workflow.StageConfig
		shouldCheckConstForOther bool // Whether constitution should be checked for other stages
	}{
		"only constitution - no other check needed": {
			stageConfig:              &workflow.StageConfig{Constitution: true},
			shouldCheckConstForOther: false,
		},
		"constitution + specify - constitution check needed for specify": {
			stageConfig:              &workflow.StageConfig{Constitution: true, Specify: true},
			shouldCheckConstForOther: true,
		},
		"only specify - constitution check needed": {
			stageConfig:              &workflow.StageConfig{Specify: true},
			shouldCheckConstForOther: true,
		},
		"only plan - constitution check needed": {
			stageConfig:              &workflow.StageConfig{Plan: true},
			shouldCheckConstForOther: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// The logic in run.go is:
			// if !stageConfig.Constitution || stageConfig.Count() > 1 {
			//     // Check constitution
			// }
			shouldCheck := !tc.stageConfig.Constitution || tc.stageConfig.Count() > 1

			assert.Equal(t, tc.shouldCheckConstForOther, shouldCheck,
				"Constitution check condition should match expected")
		})
	}
}
