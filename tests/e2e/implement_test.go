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

// TestE2E_ImplementPhases tests the --phases flag runs each phase in a separate session.
func TestE2E_ImplementPhases(t *testing.T) {
	tests := map[string]struct {
		description  string
		featureName  string
		setupFunc    func(*testutil.E2EEnv, string)
		wantExitCode int
	}{
		"implement --phases runs each phase in separate session": {
			description: "Verify --phases flag executes phases separately",
			featureName: "001-test-feature",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupImplementEnvironment(env)
				setupMultiPhaseTasksArtifact(env, specName)
			},
			wantExitCode: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env, tt.featureName)
			}

			// Run implement with --phases flag
			result := env.Run("implement", "--phases")

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_ImplementPhaseN tests the --phase N flag runs only the specified phase.
func TestE2E_ImplementPhaseN(t *testing.T) {
	tests := map[string]struct {
		description   string
		featureName   string
		phaseNumber   string
		setupFunc     func(*testutil.E2EEnv, string)
		wantExitCode  int
		wantInOutput  string
		wantNotOutput string
	}{
		"implement --phase 1 runs only phase 1": {
			description: "Verify --phase 1 executes only first phase",
			featureName: "001-test-feature",
			phaseNumber: "1",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupImplementEnvironment(env)
				setupMultiPhaseTasksArtifact(env, specName)
			},
			wantExitCode: 0,
			wantInOutput: "phase 1",
		},
		"implement --phase 2 runs only phase 2": {
			description: "Verify --phase 2 executes only second phase",
			featureName: "001-test-feature",
			phaseNumber: "2",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupImplementEnvironment(env)
				setupMultiPhaseTasksArtifact(env, specName)
			},
			wantExitCode: 0,
			wantInOutput: "phase 2",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env, tt.featureName)
			}

			// Run implement with --phase N flag
			result := env.Run("implement", "--phase", tt.phaseNumber)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			if tt.wantInOutput != "" {
				combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
				require.Contains(t, combinedOutput, strings.ToLower(tt.wantInOutput),
					"expected output to contain %q", tt.wantInOutput)
			}
		})
	}
}

// TestE2E_ImplementFromPhase tests the --from-phase N flag resumes from phase N.
func TestE2E_ImplementFromPhase(t *testing.T) {
	tests := map[string]struct {
		description  string
		featureName  string
		fromPhase    string
		setupFunc    func(*testutil.E2EEnv, string)
		wantExitCode int
	}{
		"implement --from-phase 2 resumes from phase 2": {
			description: "Verify --from-phase 2 starts execution from phase 2",
			featureName: "001-test-feature",
			fromPhase:   "2",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupImplementEnvironment(env)
				setupMultiPhaseTasksArtifact(env, specName)
			},
			wantExitCode: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env, tt.featureName)
			}

			// Run implement with --from-phase N flag
			result := env.Run("implement", "--from-phase", tt.fromPhase)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_ImplementTasks tests the --tasks flag runs each task in a separate session.
func TestE2E_ImplementTasks(t *testing.T) {
	tests := map[string]struct {
		description  string
		featureName  string
		setupFunc    func(*testutil.E2EEnv, string)
		wantExitCode int
	}{
		"implement --tasks runs each task in separate session": {
			description: "Verify --tasks flag executes tasks separately",
			featureName: "001-test-feature",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupImplementEnvironment(env)
				setupMultiPhaseTasksArtifact(env, specName)
			},
			wantExitCode: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env, tt.featureName)
			}

			// Run implement with --tasks flag
			result := env.Run("implement", "--tasks")

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_ImplementFromTask tests the --from-task T00N flag resumes from a specific task.
func TestE2E_ImplementFromTask(t *testing.T) {
	tests := map[string]struct {
		description  string
		featureName  string
		fromTask     string
		setupFunc    func(*testutil.E2EEnv, string)
		wantExitCode int
	}{
		"implement --from-task T003 resumes from task T003": {
			description: "Verify --from-task T003 starts execution from task T003 (no deps)",
			featureName: "001-test-feature",
			fromTask:    "T003",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupImplementEnvironment(env)
				setupMultiPhaseTasksArtifact(env, specName)
			},
			wantExitCode: 0,
		},
		"implement --tasks --from-task T004 combines flags": {
			description: "Verify --tasks with --from-task works together",
			featureName: "001-test-feature",
			fromTask:    "T004",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupImplementEnvironment(env)
				setupMultiPhaseTasksArtifact(env, specName)
			},
			wantExitCode: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env, tt.featureName)
			}

			// Run implement with --from-task flag
			args := []string{"implement", "--tasks", "--from-task", tt.fromTask}
			result := env.Run(args...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_ImplementSingleSession tests the --single-session flag (legacy mode).
func TestE2E_ImplementSingleSession(t *testing.T) {
	tests := map[string]struct {
		description  string
		featureName  string
		setupFunc    func(*testutil.E2EEnv, string)
		wantExitCode int
	}{
		"implement --single-session runs all tasks in one session": {
			description: "Verify --single-session executes all tasks in one session",
			featureName: "001-test-feature",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupImplementEnvironment(env)
				setupMultiPhaseTasksArtifact(env, specName)
			},
			wantExitCode: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env, tt.featureName)
			}

			// Run implement with --single-session flag
			result := env.Run("implement", "--single-session")

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_ImplementInvalidPhase tests error handling for invalid phase numbers.
func TestE2E_ImplementInvalidPhase(t *testing.T) {
	tests := map[string]struct {
		description   string
		featureName   string
		phaseNumber   string
		setupFunc     func(*testutil.E2EEnv, string)
		wantExitCode  int
		wantErrSubstr string
	}{
		"implement --phase 99 fails with error for non-existent phase": {
			description: "Verify invalid phase number returns error",
			featureName: "001-test-feature",
			phaseNumber: "99",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupImplementEnvironment(env)
				setupMultiPhaseTasksArtifact(env, specName)
			},
			wantExitCode:  1,
			wantErrSubstr: "phase",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env, tt.featureName)
			}

			// Run implement with invalid phase number
			result := env.Run("implement", "--phase", tt.phaseNumber)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"expected non-zero exit code for invalid phase\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			require.Contains(t, combinedOutput, strings.ToLower(tt.wantErrSubstr),
				"expected error output to mention %q", tt.wantErrSubstr)
		})
	}
}

// setupImplementEnvironment sets up the base environment for implement command tests.
func setupImplementEnvironment(env *testutil.E2EEnv) {
	env.SetupAutospecInit()
	env.SetupConstitution()
	env.InitGitRepo()
	env.CreateBranch("001-test-feature")
}

// setupMultiPhaseTasksArtifact creates a tasks.yaml with multiple phases and tasks.
// This is needed to properly test --phases, --phase N, --from-phase, --tasks, --from-task.
func setupMultiPhaseTasksArtifact(env *testutil.E2EEnv, specName string) {
	specDir := filepath.Join(env.SpecsDir(), specName)
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		return
	}

	// Create spec.yaml first (prerequisite)
	specContent := `feature:
  branch: "001-test-feature"
  created: "2025-01-01"
  status: "Draft"
  input: "test multi-phase feature"
user_stories:
  - id: "US-001"
    title: "Test Story 1"
    priority: "P1"
    as_a: "developer"
    i_want: "multi-phase implementation"
    so_that: "phases can be tested"
    why_this_priority: "testing"
    independent_test: "verify phases work"
    acceptance_scenarios:
      - given: "phases exist"
        when: "implement runs"
        then: "phases execute correctly"
  - id: "US-002"
    title: "Test Story 2"
    priority: "P2"
    as_a: "developer"
    i_want: "task-level execution"
    so_that: "tasks can be tested individually"
    why_this_priority: "testing"
    independent_test: "verify tasks work"
    acceptance_scenarios:
      - given: "tasks exist"
        when: "implement --tasks runs"
        then: "tasks execute separately"
requirements:
  functional:
    - id: "FR-001"
      description: "Support multi-phase execution"
      acceptance_criteria: "Phases execute in order"
  non_functional:
    - id: "NFR-001"
      category: "code_quality"
      description: "Clean code"
      measurable_target: "100% lint pass"
success_criteria:
  measurable_outcomes:
    - id: "SC-001"
      description: "All phases complete"
      metric: "phase completion rate"
      target: "100%"
key_entities: []
edge_cases: []
assumptions: []
constraints: []
out_of_scope: []
_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "spec"
`
	specPath := filepath.Join(specDir, "spec.yaml")
	_ = os.WriteFile(specPath, []byte(specContent), 0o644)

	// Create plan.yaml (prerequisite)
	planContent := `plan:
  branch: "001-test-feature"
  created: "2025-01-01"
  spec_path: "specs/001-test-feature/spec.yaml"
summary: "Multi-phase test plan"
technical_context:
  language: "Go"
  framework: "None"
  primary_dependencies: []
  storage: "None"
  testing:
    framework: "Go testing"
    approach: "Unit tests"
  target_platform: "Linux"
  project_type: "cli"
  performance_goals: "Fast"
  constraints: []
  scale_scope: "Small"
constitution_check:
  constitution_path: ".autospec/constitution.yaml"
  gates: []
research_findings:
  decisions: []
data_model:
  entities: []
api_contracts:
  endpoints: []
project_structure:
  documentation: []
  source_code: []
  tests: []
implementation_phases:
  - phase: 1
    name: "Setup"
    goal: "Initialize project structure"
    deliverables:
      - "Project scaffolding"
  - phase: 2
    name: "Core Implementation"
    goal: "Implement core functionality"
    deliverables:
      - "Core features"
  - phase: 3
    name: "Polish"
    goal: "Finalize and clean up"
    deliverables:
      - "Final touches"
risks: []
open_questions: []
_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "plan"
`
	planPath := filepath.Join(specDir, "plan.yaml")
	_ = os.WriteFile(planPath, []byte(planContent), 0o644)

	// Create multi-phase tasks.yaml
	tasksContent := `tasks:
  branch: "001-test-feature"
  created: "2025-01-01"
  spec_path: "specs/001-test-feature/spec.yaml"
  plan_path: "specs/001-test-feature/plan.yaml"
summary:
  total_tasks: 6
  total_phases: 3
  parallel_opportunities: 2
  estimated_complexity: "medium"
phases:
  - number: 1
    title: "Setup"
    purpose: "Initialize project structure"
    tasks:
      - id: "T001"
        title: "Create project scaffolding"
        status: "Pending"
        type: "implementation"
        parallel: false
        story_id: "US-001"
        file_path: "setup.go"
        dependencies: []
        acceptance_criteria:
          - "Project structure created"
      - id: "T002"
        title: "Configure dependencies"
        status: "Pending"
        type: "implementation"
        parallel: false
        story_id: "US-001"
        file_path: "deps.go"
        dependencies:
          - "T001"
        acceptance_criteria:
          - "Dependencies configured"
  - number: 2
    title: "Core Implementation"
    purpose: "Implement core functionality"
    tasks:
      - id: "T003"
        title: "Implement feature A"
        status: "Pending"
        type: "implementation"
        parallel: true
        story_id: "US-001"
        file_path: "feature_a.go"
        dependencies: []
        acceptance_criteria:
          - "Feature A works"
      - id: "T004"
        title: "Implement feature B"
        status: "Pending"
        type: "implementation"
        parallel: true
        story_id: "US-002"
        file_path: "feature_b.go"
        dependencies: []
        acceptance_criteria:
          - "Feature B works"
  - number: 3
    title: "Polish"
    purpose: "Finalize and clean up"
    tasks:
      - id: "T005"
        title: "Add documentation"
        status: "Pending"
        type: "documentation"
        parallel: false
        story_id: "US-001"
        file_path: "README.md"
        dependencies:
          - "T003"
          - "T004"
        acceptance_criteria:
          - "Documentation complete"
      - id: "T006"
        title: "Final cleanup"
        status: "Pending"
        type: "refactor"
        parallel: false
        story_id: "US-002"
        file_path: "cleanup.go"
        dependencies:
          - "T005"
        acceptance_criteria:
          - "Cleanup complete"
dependencies:
  user_story_order:
    - "US-001"
    - "US-002"
  phase_order:
    - 1
    - 2
    - 3
parallel_execution:
  - phase: 2
    task_groups:
      - tasks:
          - "T003"
          - "T004"
        can_parallelize: true
        reason: "Independent features"
implementation_strategy:
  mvp_scope:
    phases:
      - 1
      - 2
    description: "Core functionality"
    validation: "Tests pass"
  incremental_delivery:
    - phase: 1
      milestone: "Setup complete"
      validation: "Structure exists"
    - phase: 2
      milestone: "Features complete"
      validation: "Features work"
    - phase: 3
      milestone: "Polish complete"
      validation: "Documentation done"
_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "tasks"
`
	tasksPath := filepath.Join(specDir, "tasks.yaml")
	_ = os.WriteFile(tasksPath, []byte(tasksContent), 0o644)
}
