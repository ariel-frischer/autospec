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

// TestE2E_ErrorScenarioMissingTasksYAML tests implement command error when tasks.yaml is missing.
// This verifies US-009: "implement command is run when tasks.yaml does not exist".
func TestE2E_ErrorScenarioMissingTasksYAML(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		wantExitCode  int
		wantErrSubstr string
	}{
		"implement fails without tasks.yaml": {
			description: "Run implement without tasks.yaml and verify error mentions 'tasks'",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.SetupConstitution()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				// Set up spec and plan but NOT tasks
				env.SetupPlan("001-test-feature")
			},
			wantExitCode:  shared.ExitValidationFailed,
			wantErrSubstr: "tasks",
		},
		"implement --phases fails without tasks.yaml": {
			description: "Run implement --phases without tasks.yaml",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.SetupConstitution()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupPlan("001-test-feature")
			},
			wantExitCode:  shared.ExitValidationFailed,
			wantErrSubstr: "tasks",
		},
		"implement --tasks fails without tasks.yaml": {
			description: "Run implement --tasks without tasks.yaml",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.SetupConstitution()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupPlan("001-test-feature")
			},
			wantExitCode:  shared.ExitValidationFailed,
			wantErrSubstr: "tasks",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			result := env.Run("implement")

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"exit code mismatch\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			require.True(t,
				strings.Contains(combinedOutput, tt.wantErrSubstr),
				"output should mention '%s'\nstdout: %s\nstderr: %s",
				tt.wantErrSubstr, result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_ErrorScenarioInvalidPhaseNumber tests implement command with invalid phase numbers.
// This verifies US-009: "implement --phase 99 is run when phase 99 does not exist".
func TestE2E_ErrorScenarioInvalidPhaseNumber(t *testing.T) {
	tests := map[string]struct {
		description   string
		phaseArg      string
		wantExitCode  int
		wantErrSubstr string
	}{
		"phase 99 does not exist": {
			description:   "Run implement --phase 99 with tasks.yaml that has only 3 phases",
			phaseArg:      "99",
			wantExitCode:  shared.ExitValidationFailed,
			wantErrSubstr: "phase",
		},
		"negative phase is invalid": {
			description:   "Run implement --phase -1 (must be positive)",
			phaseArg:      "-1",
			wantExitCode:  shared.ExitValidationFailed,
			wantErrSubstr: "positive",
		},
		"phase 1000 does not exist": {
			description:   "Run implement --phase 1000",
			phaseArg:      "1000",
			wantExitCode:  shared.ExitValidationFailed,
			wantErrSubstr: "phase",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			// Set up full environment with tasks.yaml
			setupErrorTestEnvironment(env)

			result := env.Run("implement", "--phase", tt.phaseArg)

			require.NotEqual(t, shared.ExitSuccess, result.ExitCode,
				"should fail with invalid phase\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			require.True(t,
				strings.Contains(combinedOutput, tt.wantErrSubstr),
				"output should mention '%s'\nstdout: %s\nstderr: %s",
				tt.wantErrSubstr, result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_ErrorScenarioMissingAgentBinary tests error when agent binary is not in PATH.
// This verifies US-009: "agent binary is removed from PATH".
func TestE2E_ErrorScenarioMissingAgentBinary(t *testing.T) {
	tests := map[string]struct {
		description      string
		removeBinaries   []string
		command          []string
		wantExitCode     int
		wantErrSubstrs   []string
		skipIfAnyPresent bool
	}{
		"doctor detects missing claude binary": {
			description:    "Run doctor with claude binary removed",
			removeBinaries: []string{"claude", "opencode"},
			command:        []string{"doctor"},
			wantExitCode:   shared.ExitValidationFailed,
			wantErrSubstrs: []string{"claude", "opencode", "not found", "✗", "fail"},
		},
		"specify fails when claude binary is missing": {
			description:    "Run specify with claude binary removed",
			removeBinaries: []string{"claude"},
			command:        []string{"specify", "test feature"},
			wantExitCode:   shared.ExitValidationFailed,
			wantErrSubstrs: []string{"claude", "not found", "fail", "error"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)
			env.InitGitRepo()
			env.CreateBranch("001-test-feature")
			env.SetupConstitution()
			env.SetupAutospecInit()

			// Remove specified binaries from bin directory
			binDir := env.BinDir()
			for _, binary := range tt.removeBinaries {
				os.Remove(filepath.Join(binDir, binary))
			}

			result := env.Run(tt.command...)

			require.NotEqual(t, shared.ExitSuccess, result.ExitCode,
				"should fail when agent binary is missing\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			foundMatch := false
			for _, substr := range tt.wantErrSubstrs {
				if strings.Contains(combinedOutput, substr) {
					foundMatch = true
					break
				}
			}
			require.True(t, foundMatch,
				"output should mention one of %v\nstdout: %s\nstderr: %s",
				tt.wantErrSubstrs, result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_ErrorScenarioInvalidConfigYAML tests error when config file has invalid YAML.
// This verifies US-009: "config file has invalid YAML".
func TestE2E_ErrorScenarioInvalidConfigYAML(t *testing.T) {
	tests := map[string]struct {
		description    string
		configContent  string
		command        []string
		wantExitCode   int
		wantErrSubstrs []string
	}{
		"invalid YAML syntax - missing colon": {
			description:    "Config file with missing colon syntax error",
			configContent:  "agent_preset claude\nspecs_dir: specs\n",
			command:        []string{"config", "show"},
			wantExitCode:   shared.ExitValidationFailed,
			wantErrSubstrs: []string{"error", "yaml", "parse", "invalid", "syntax"},
		},
		"invalid YAML syntax - bad indentation": {
			description: "Config file with improper indentation",
			configContent: `agent_preset: claude
  specs_dir: specs
    max_retries: 3
`,
			command:        []string{"config", "show"},
			wantExitCode:   shared.ExitValidationFailed,
			wantErrSubstrs: []string{"error", "yaml", "parse", "invalid", "indent", "map"},
		},
		"invalid YAML syntax - unclosed quote": {
			description:    "Config file with unclosed quote",
			configContent:  "agent_preset: \"claude\nspecs_dir: specs\n",
			command:        []string{"config", "show"},
			wantExitCode:   shared.ExitValidationFailed,
			wantErrSubstrs: []string{"error", "yaml", "parse", "invalid"},
		},
		"invalid YAML syntax - tabs instead of spaces": {
			description:    "Config file using tabs for indentation",
			configContent:  "agent_preset: claude\n\tspecs_dir: specs\n",
			command:        []string{"config", "show"},
			wantExitCode:   shared.ExitValidationFailed,
			wantErrSubstrs: []string{"error", "yaml", "parse", "invalid", "tab"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)
			env.InitGitRepo()

			// Create .autospec directory
			autospecDir := filepath.Join(env.TempDir(), ".autospec")
			if err := os.MkdirAll(autospecDir, 0o755); err != nil {
				t.Fatalf("creating .autospec directory: %v", err)
			}

			// Write invalid config file
			configPath := filepath.Join(autospecDir, "config.yml")
			if err := os.WriteFile(configPath, []byte(tt.configContent), 0o644); err != nil {
				t.Fatalf("writing invalid config: %v", err)
			}

			result := env.Run(tt.command...)

			require.NotEqual(t, shared.ExitSuccess, result.ExitCode,
				"should fail with invalid YAML config\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			foundMatch := false
			for _, substr := range tt.wantErrSubstrs {
				if strings.Contains(combinedOutput, substr) {
					foundMatch = true
					break
				}
			}
			require.True(t, foundMatch,
				"output should mention one of %v\nstdout: %s\nstderr: %s",
				tt.wantErrSubstrs, result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_ErrorScenarioInvalidTaskID tests implement command with invalid task ID.
func TestE2E_ErrorScenarioInvalidTaskID(t *testing.T) {
	tests := map[string]struct {
		description   string
		taskArg       string
		wantExitCode  int
		wantErrSubstr string
	}{
		"task T999 does not exist": {
			description:   "Run implement --from-task T999 with tasks.yaml that has T001-T006",
			taskArg:       "T999",
			wantExitCode:  shared.ExitValidationFailed,
			wantErrSubstr: "task",
		},
		"invalid task ID format": {
			description:   "Run implement --from-task INVALID",
			taskArg:       "INVALID",
			wantExitCode:  shared.ExitValidationFailed,
			wantErrSubstr: "task",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			// Set up full environment with tasks.yaml
			setupErrorTestEnvironment(env)

			result := env.Run("implement", "--tasks", "--from-task", tt.taskArg)

			require.NotEqual(t, shared.ExitSuccess, result.ExitCode,
				"should fail with invalid task ID\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			require.True(t,
				strings.Contains(combinedOutput, tt.wantErrSubstr),
				"output should mention '%s'\nstdout: %s\nstderr: %s",
				tt.wantErrSubstr, result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_ErrorScenarioMissingPrerequisites tests commands without required prerequisites.
func TestE2E_ErrorScenarioMissingPrerequisites(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		command       []string
		wantExitCode  int
		wantErrSubstr string
	}{
		"plan fails without spec.yaml": {
			description: "Run plan without spec.yaml",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.SetupConstitution()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				// NO spec.yaml setup
			},
			command:       []string{"plan"},
			wantExitCode:  shared.ExitValidationFailed,
			wantErrSubstr: "spec",
		},
		"tasks fails without plan.yaml": {
			description: "Run tasks without plan.yaml",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.SetupConstitution()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				// Set up only spec, not plan
				env.SetupSpec("001-test-feature")
			},
			command:       []string{"tasks"},
			wantExitCode:  shared.ExitValidationFailed,
			wantErrSubstr: "plan",
		},
		"clarify fails without spec.yaml": {
			description: "Run clarify without spec.yaml",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.SetupConstitution()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
			},
			command:       []string{"clarify"},
			wantExitCode:  shared.ExitValidationFailed,
			wantErrSubstr: "spec",
		},
		"checklist fails without spec.yaml": {
			description: "Run checklist without spec.yaml",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.SetupConstitution()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
			},
			command:       []string{"checklist"},
			wantExitCode:  shared.ExitValidationFailed,
			wantErrSubstr: "spec",
		},
		"analyze fails without tasks.yaml": {
			description: "Run analyze without tasks.yaml",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.SetupConstitution()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupPlan("001-test-feature")
			},
			command:       []string{"analyze"},
			wantExitCode:  shared.ExitValidationFailed,
			wantErrSubstr: "tasks",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			result := env.Run(tt.command...)

			require.NotEqual(t, shared.ExitSuccess, result.ExitCode,
				"should fail without prerequisite\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			require.True(t,
				strings.Contains(combinedOutput, tt.wantErrSubstr),
				"output should mention '%s'\nstdout: %s\nstderr: %s",
				tt.wantErrSubstr, result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_ErrorScenarioUnknownFlags tests error handling for unknown flags.
func TestE2E_ErrorScenarioUnknownFlags(t *testing.T) {
	tests := map[string]struct {
		description   string
		command       []string
		wantExitCode  int
		wantErrSubstr string
	}{
		"specify with unknown flag": {
			description:   "Run specify with --invalid-flag",
			command:       []string{"specify", "--invalid-flag", "test"},
			wantExitCode:  shared.ExitValidationFailed,
			wantErrSubstr: "unknown",
		},
		"implement with unknown flag": {
			description:   "Run implement with --bad-option",
			command:       []string{"implement", "--bad-option"},
			wantExitCode:  shared.ExitValidationFailed,
			wantErrSubstr: "unknown",
		},
		"run with unknown flag": {
			description:   "Run with --nonexistent-flag",
			command:       []string{"run", "--nonexistent-flag", "test"},
			wantExitCode:  shared.ExitValidationFailed,
			wantErrSubstr: "unknown",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)
			env.InitGitRepo()
			env.SetupAutospecInit()

			result := env.Run(tt.command...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"exit code mismatch\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			require.True(t,
				strings.Contains(combinedOutput, tt.wantErrSubstr),
				"output should mention '%s'\nstdout: %s\nstderr: %s",
				tt.wantErrSubstr, result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_ErrorScenarioInvalidArtifactFile tests artifact command with invalid files.
func TestE2E_ErrorScenarioInvalidArtifactFile(t *testing.T) {
	tests := map[string]struct {
		description    string
		artifactPath   string
		createFile     bool
		fileContent    string
		wantExitCode   int
		wantErrSubstrs []string
	}{
		"artifact with non-existent file": {
			description:    "Run artifact with path that doesn't exist",
			artifactPath:   "nonexistent.yaml",
			createFile:     false,
			wantExitCode:   shared.ExitValidationFailed,
			wantErrSubstrs: []string{"not found", "error", "no such file"},
		},
		"artifact with invalid YAML content": {
			description:    "Run artifact with invalid YAML syntax",
			artifactPath:   "invalid.yaml",
			createFile:     true,
			fileContent:    "this is not: valid: yaml: syntax\n  bad indentation",
			wantExitCode:   shared.ExitValidationFailed,
			wantErrSubstrs: []string{"error", "yaml", "invalid", "parse"},
		},
		"artifact with empty file": {
			description:    "Run artifact with empty file",
			artifactPath:   "empty.yaml",
			createFile:     true,
			fileContent:    "",
			wantExitCode:   shared.ExitValidationFailed,
			wantErrSubstrs: []string{"error", "empty", "invalid"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)
			env.InitGitRepo()
			env.SetupAutospecInit()

			artifactPath := filepath.Join(env.TempDir(), tt.artifactPath)
			if tt.createFile {
				if err := os.WriteFile(artifactPath, []byte(tt.fileContent), 0o644); err != nil {
					t.Fatalf("writing test artifact: %v", err)
				}
			}

			result := env.Run("artifact", artifactPath)

			require.NotEqual(t, shared.ExitSuccess, result.ExitCode,
				"should fail with invalid artifact\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			foundMatch := false
			for _, substr := range tt.wantErrSubstrs {
				if strings.Contains(combinedOutput, substr) {
					foundMatch = true
					break
				}
			}
			require.True(t, foundMatch,
				"output should mention one of %v\nstdout: %s\nstderr: %s",
				tt.wantErrSubstrs, result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_ErrorMessagesAreActionable verifies error messages provide clear guidance.
func TestE2E_ErrorMessagesAreActionable(t *testing.T) {
	tests := map[string]struct {
		description     string
		setupFunc       func(*testutil.E2EEnv)
		command         []string
		wantActionWords []string
	}{
		"missing constitution suggests creating it": {
			description: "Error should suggest creating constitution",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				// NO constitution
			},
			command:         []string{"specify", "test feature"},
			wantActionWords: []string{"constitution", "init", "create", "run"},
		},
		"missing spec suggests running specify": {
			description: "Error should suggest running specify",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.SetupConstitution()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				// NO spec
			},
			command:         []string{"plan"},
			wantActionWords: []string{"spec", "specify", "first", "run"},
		},
		"missing plan suggests running plan": {
			description: "Error should suggest running plan",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.SetupConstitution()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupSpec("001-test-feature")
				// NO plan
			},
			command:         []string{"tasks"},
			wantActionWords: []string{"plan", "first", "run"},
		},
		"missing tasks suggests running tasks": {
			description: "Error should suggest running tasks",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.SetupConstitution()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupPlan("001-test-feature")
				// NO tasks
			},
			command:         []string{"implement"},
			wantActionWords: []string{"tasks", "first", "run"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			result := env.Run(tt.command...)

			require.NotEqual(t, shared.ExitSuccess, result.ExitCode,
				"should fail\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			foundAction := false
			for _, word := range tt.wantActionWords {
				if strings.Contains(combinedOutput, word) {
					foundAction = true
					break
				}
			}
			require.True(t, foundAction,
				"error message should contain actionable words from %v\nstdout: %s\nstderr: %s",
				tt.wantActionWords, result.Stdout, result.Stderr)
		})
	}
}

// setupErrorTestEnvironment sets up a complete environment for error testing.
func setupErrorTestEnvironment(env *testutil.E2EEnv) {
	env.SetupAutospecInit()
	env.SetupConstitution()
	env.InitGitRepo()
	env.CreateBranch("001-test-feature")
	setupMultiPhaseTasksForErrorTest(env, "001-test-feature")
}

// setupMultiPhaseTasksForErrorTest creates a tasks.yaml with multiple phases for error testing.
func setupMultiPhaseTasksForErrorTest(env *testutil.E2EEnv, specName string) {
	specDir := filepath.Join(env.SpecsDir(), specName)
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		return
	}

	// Create spec.yaml
	specContent := `feature:
  branch: "001-test-feature"
  created: "2025-01-01"
  status: "Draft"
  input: "test feature"
user_stories:
  - id: "US-001"
    title: "Test Story"
    priority: "P1"
    as_a: "developer"
    i_want: "error testing"
    so_that: "errors are clear"
    why_this_priority: "testing"
    independent_test: "verify errors"
    acceptance_scenarios:
      - given: "error occurs"
        when: "command runs"
        then: "error is clear"
requirements:
  functional:
    - id: "FR-001"
      description: "Clear error messages"
      testable: true
      acceptance_criteria: "Errors are actionable"
  non_functional:
    - id: "NFR-001"
      category: "usability"
      description: "Good UX"
      measurable_target: "Users understand errors"
success_criteria:
  measurable_outcomes:
    - id: "SC-001"
      description: "Error clarity"
      metric: "Error comprehension"
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
	_ = os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte(specContent), 0o644)

	// Create plan.yaml
	planContent := `plan:
  branch: "001-test-feature"
  created: "2025-01-01"
  spec_path: "specs/001-test-feature/spec.yaml"
summary: "Test plan"
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
    goal: "Setup"
    deliverables:
      - "Setup done"
  - phase: 2
    name: "Core"
    goal: "Core"
    deliverables:
      - "Core done"
  - phase: 3
    name: "Polish"
    goal: "Polish"
    deliverables:
      - "Polish done"
risks: []
open_questions: []
_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "plan"
`
	_ = os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte(planContent), 0o644)

	// Create tasks.yaml with 3 phases
	tasksContent := `tasks:
  branch: "001-test-feature"
  created: "2025-01-01"
  spec_path: "specs/001-test-feature/spec.yaml"
  plan_path: "specs/001-test-feature/plan.yaml"
summary:
  total_tasks: 6
  total_phases: 3
  parallel_opportunities: 1
  estimated_complexity: "low"
phases:
  - number: 1
    title: "Setup"
    purpose: "Setup"
    tasks:
      - id: "T001"
        title: "Task 1"
        status: "Pending"
        type: "implementation"
        parallel: false
        story_id: "US-001"
        file_path: "file1.go"
        dependencies: []
        acceptance_criteria:
          - "Done"
      - id: "T002"
        title: "Task 2"
        status: "Pending"
        type: "implementation"
        parallel: false
        story_id: "US-001"
        file_path: "file2.go"
        dependencies:
          - "T001"
        acceptance_criteria:
          - "Done"
  - number: 2
    title: "Core"
    purpose: "Core"
    tasks:
      - id: "T003"
        title: "Task 3"
        status: "Pending"
        type: "implementation"
        parallel: true
        story_id: "US-001"
        file_path: "file3.go"
        dependencies: []
        acceptance_criteria:
          - "Done"
      - id: "T004"
        title: "Task 4"
        status: "Pending"
        type: "implementation"
        parallel: true
        story_id: "US-001"
        file_path: "file4.go"
        dependencies: []
        acceptance_criteria:
          - "Done"
  - number: 3
    title: "Polish"
    purpose: "Polish"
    tasks:
      - id: "T005"
        title: "Task 5"
        status: "Pending"
        type: "documentation"
        parallel: false
        story_id: "US-001"
        file_path: "README.md"
        dependencies:
          - "T003"
          - "T004"
        acceptance_criteria:
          - "Done"
      - id: "T006"
        title: "Task 6"
        status: "Pending"
        type: "refactor"
        parallel: false
        story_id: "US-001"
        file_path: "cleanup.go"
        dependencies:
          - "T005"
        acceptance_criteria:
          - "Done"
dependencies:
  user_story_order:
    - "US-001"
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
        reason: "Independent"
implementation_strategy:
  mvp_scope:
    phases:
      - 1
      - 2
    description: "MVP"
    validation: "Tests pass"
  incremental_delivery:
    - phase: 1
      milestone: "Setup"
      validation: "Done"
    - phase: 2
      milestone: "Core"
      validation: "Done"
    - phase: 3
      milestone: "Polish"
      validation: "Done"
_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "tasks"
`
	_ = os.WriteFile(filepath.Join(specDir, "tasks.yaml"), []byte(tasksContent), 0o644)
}
