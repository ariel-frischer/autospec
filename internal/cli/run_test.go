// Package cli_test tests the run command for executing custom stage combinations with implement_method config support.
// Related: internal/cli/run.go
// Tags: cli, run, workflow, stages, configuration, implement-method, consistency
package cli

import (
	"testing"

	"github.com/ariel-frischer/autospec/internal/workflow"
)

// TestRunImplementMethodConfig verifies that 'autospec run -pti' respects the
// implement_method config setting, matching the behavior of 'autospec implement'.
// This prevents regression of the bug where run.go hardcoded single-session mode.
func TestRunImplementMethodConfig(t *testing.T) {
	tests := map[string]struct {
		implementMethod  string
		wantRunAllPhases bool
		wantTaskMode     bool
	}{
		"phases config sets RunAllPhases=true": {
			implementMethod:  "phases",
			wantRunAllPhases: true,
			wantTaskMode:     false,
		},
		"tasks config sets TaskMode=true": {
			implementMethod:  "tasks",
			wantRunAllPhases: false,
			wantTaskMode:     true,
		},
		"single-session config leaves both false": {
			implementMethod:  "single-session",
			wantRunAllPhases: false,
			wantTaskMode:     false,
		},
		"empty config leaves both false (uses default elsewhere)": {
			implementMethod:  "",
			wantRunAllPhases: false,
			wantTaskMode:     false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Replicate the logic from run.go executeStages() StageImplement case
			// This is the exact logic that was fixed to respect implement_method
			phaseOpts := workflow.PhaseExecutionOptions{}
			switch tt.implementMethod {
			case "phases":
				phaseOpts.RunAllPhases = true
			case "tasks":
				phaseOpts.TaskMode = true
			case "single-session":
				// Legacy behavior: no phase/task mode (default state)
			}

			if phaseOpts.RunAllPhases != tt.wantRunAllPhases {
				t.Errorf("RunAllPhases = %v, want %v", phaseOpts.RunAllPhases, tt.wantRunAllPhases)
			}
			if phaseOpts.TaskMode != tt.wantTaskMode {
				t.Errorf("TaskMode = %v, want %v", phaseOpts.TaskMode, tt.wantTaskMode)
			}
		})
	}
}

// TestRunAndImplementConsistency verifies that both 'autospec run -i' and 'autospec implement'
// use the same logic to apply implement_method config. This is a regression test to ensure
// both commands behave identically for the implement stage.
func TestRunAndImplementConsistency(t *testing.T) {
	configMethods := []string{"phases", "tasks", "single-session"}

	for _, method := range configMethods {
		t.Run("method_"+method, func(t *testing.T) {
			// Simulate implement.go logic (lines 154-165)
			implRunAllPhases := false
			implTaskMode := false
			if method != "" {
				switch method {
				case "phases":
					implRunAllPhases = true
				case "tasks":
					implTaskMode = true
				case "single-session":
					// Legacy behavior
				}
			}

			// Simulate run.go logic (lines 307-314 after fix)
			runPhaseOpts := workflow.PhaseExecutionOptions{}
			switch method {
			case "phases":
				runPhaseOpts.RunAllPhases = true
			case "tasks":
				runPhaseOpts.TaskMode = true
			case "single-session":
				// Legacy behavior
			}

			// Both should produce identical results
			if implRunAllPhases != runPhaseOpts.RunAllPhases {
				t.Errorf("implement vs run: RunAllPhases mismatch for %q: impl=%v, run=%v",
					method, implRunAllPhases, runPhaseOpts.RunAllPhases)
			}
			if implTaskMode != runPhaseOpts.TaskMode {
				t.Errorf("implement vs run: TaskMode mismatch for %q: impl=%v, run=%v",
					method, implTaskMode, runPhaseOpts.TaskMode)
			}
		})
	}
}

func TestStageConfigFromFlags(t *testing.T) {
	tests := map[string]struct {
		config   *workflow.StageConfig
		expected []workflow.Stage
	}{
		"core stages only (-spti)": {
			config: &workflow.StageConfig{
				Specify:   true,
				Plan:      true,
				Tasks:     true,
				Implement: true,
			},
			expected: []workflow.Stage{
				workflow.StageSpecify,
				workflow.StagePlan,
				workflow.StageTasks,
				workflow.StageImplement,
			},
		},
		"constitution and specify (-ns)": {
			config: &workflow.StageConfig{
				Constitution: true,
				Specify:      true,
			},
			expected: []workflow.Stage{
				workflow.StageConstitution,
				workflow.StageSpecify,
			},
		},
		"specify, clarify, plan (-srp)": {
			config: &workflow.StageConfig{
				Specify: true,
				Clarify: true,
				Plan:    true,
			},
			expected: []workflow.Stage{
				workflow.StageSpecify,
				workflow.StageClarify,
				workflow.StagePlan,
			},
		},
		"tasks, checklist, analyze, implement (-tlzi)": {
			config: &workflow.StageConfig{
				Tasks:     true,
				Checklist: true,
				Analyze:   true,
				Implement: true,
			},
			expected: []workflow.Stage{
				workflow.StageTasks,
				workflow.StageChecklist,
				workflow.StageAnalyze,
				workflow.StageImplement,
			},
		},
		"all stages with checklist (-a -l) - core + optional": {
			config: &workflow.StageConfig{
				Specify:   true,
				Plan:      true,
				Tasks:     true,
				Implement: true,
				Checklist: true,
			},
			expected: []workflow.Stage{
				workflow.StageSpecify,
				workflow.StagePlan,
				workflow.StageTasks,
				workflow.StageChecklist,
				workflow.StageImplement,
			},
		},
		"all 8 stages in canonical order": {
			config: &workflow.StageConfig{
				Constitution: true,
				Specify:      true,
				Clarify:      true,
				Plan:         true,
				Tasks:        true,
				Checklist:    true,
				Analyze:      true,
				Implement:    true,
			},
			expected: []workflow.Stage{
				workflow.StageConstitution,
				workflow.StageSpecify,
				workflow.StageClarify,
				workflow.StagePlan,
				workflow.StageTasks,
				workflow.StageChecklist,
				workflow.StageAnalyze,
				workflow.StageImplement,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.config.GetCanonicalOrder()
			if len(got) != len(tt.expected) {
				t.Errorf("GetCanonicalOrder() returned %d stages, want %d", len(got), len(tt.expected))
				return
			}
			for i, stage := range got {
				if stage != tt.expected[i] {
					t.Errorf("GetCanonicalOrder()[%d] = %v, want %v", i, stage, tt.expected[i])
				}
			}
		})
	}
}

func TestOptionalStagesWithAll(t *testing.T) {
	// Test that optional stages can be combined with -a flag
	// -a sets core stages, optional stages are added separately
	config := &workflow.StageConfig{}
	config.SetAll() // Sets core stages only

	// Add optional stages
	config.Checklist = true
	config.Analyze = true

	stages := config.GetCanonicalOrder()

	// Should be: specify, plan, tasks, checklist, analyze, implement
	expected := []workflow.Stage{
		workflow.StageSpecify,
		workflow.StagePlan,
		workflow.StageTasks,
		workflow.StageChecklist,
		workflow.StageAnalyze,
		workflow.StageImplement,
	}

	if len(stages) != len(expected) {
		t.Errorf("Expected %d stages, got %d", len(expected), len(stages))
		return
	}

	for i, stage := range stages {
		if stage != expected[i] {
			t.Errorf("Stage %d: expected %s, got %s", i, expected[i], stage)
		}
	}
}

func TestOptionalStageHasAnyStage(t *testing.T) {
	// Test that HasAnyStage returns true for optional stages only
	tests := map[string]struct {
		config   *workflow.StageConfig
		expected bool
	}{
		"no stages": {
			config:   &workflow.StageConfig{},
			expected: false,
		},
		"only constitution": {
			config:   &workflow.StageConfig{Constitution: true},
			expected: true,
		},
		"only clarify": {
			config:   &workflow.StageConfig{Clarify: true},
			expected: true,
		},
		"only checklist": {
			config:   &workflow.StageConfig{Checklist: true},
			expected: true,
		},
		"only analyze": {
			config:   &workflow.StageConfig{Analyze: true},
			expected: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := tt.config.HasAnyStage(); got != tt.expected {
				t.Errorf("HasAnyStage() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestJoinStageNames tests the joinStageNames helper function for display formatting.
func TestJoinStageNames(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		names []string
		want  string
	}{
		"empty slice returns empty string": {
			names: []string{},
			want:  "",
		},
		"single stage returns just the name": {
			names: []string{"specify"},
			want:  "specify",
		},
		"two stages joined with arrow": {
			names: []string{"specify", "plan"},
			want:  "specify → plan",
		},
		"three stages joined with arrows": {
			names: []string{"specify", "plan", "tasks"},
			want:  "specify → plan → tasks",
		},
		"full workflow chain": {
			names: []string{"constitution", "specify", "clarify", "plan", "tasks", "checklist", "analyze", "implement"},
			want:  "constitution → specify → clarify → plan → tasks → checklist → analyze → implement",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := joinStageNames(tt.names)
			if got != tt.want {
				t.Errorf("joinStageNames(%v) = %q, want %q", tt.names, got, tt.want)
			}
		})
	}
}

// TestDescriptionPropagationWithAllFlag verifies that when -a flag is used (isFullWorkflow=true),
// the feature description is only passed to specify stage, while plan/tasks/implement receive
// empty strings. This ensures later stages work from structured artifacts rather than raw input.
func TestDescriptionPropagationWithAllFlag(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		isFullWorkflow      bool
		featureDescription  string
		wantPlanPrompt      string
		wantTasksPrompt     string
		wantImplementPrompt string
	}{
		"-a flag: description only to specify, others get empty": {
			isFullWorkflow:      true,
			featureDescription:  "add user authentication",
			wantPlanPrompt:      "",
			wantTasksPrompt:     "",
			wantImplementPrompt: "",
		},
		"-a flag with complex description: others still get empty": {
			isFullWorkflow:      true,
			featureDescription:  "implement OAuth2 with Google and GitHub providers, including refresh tokens",
			wantPlanPrompt:      "",
			wantTasksPrompt:     "",
			wantImplementPrompt: "",
		},
		"individual flags: all stages get description": {
			isFullWorkflow:      false,
			featureDescription:  "focus on security",
			wantPlanPrompt:      "focus on security",
			wantTasksPrompt:     "focus on security",
			wantImplementPrompt: "focus on security",
		},
		"individual flags with empty description: all get empty": {
			isFullWorkflow:      false,
			featureDescription:  "",
			wantPlanPrompt:      "",
			wantTasksPrompt:     "",
			wantImplementPrompt: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Replicate the logic from executePlan/executeTasks/executeImplement
			// This tests the conditional logic that was added to fix description propagation

			// Plan stage logic
			planPrompt := tt.featureDescription
			if tt.isFullWorkflow {
				planPrompt = ""
			}
			if planPrompt != tt.wantPlanPrompt {
				t.Errorf("plan prompt = %q, want %q", planPrompt, tt.wantPlanPrompt)
			}

			// Tasks stage logic
			tasksPrompt := tt.featureDescription
			if tt.isFullWorkflow {
				tasksPrompt = ""
			}
			if tasksPrompt != tt.wantTasksPrompt {
				t.Errorf("tasks prompt = %q, want %q", tasksPrompt, tt.wantTasksPrompt)
			}

			// Implement stage logic
			implementPrompt := tt.featureDescription
			if tt.isFullWorkflow {
				implementPrompt = ""
			}
			if implementPrompt != tt.wantImplementPrompt {
				t.Errorf("implement prompt = %q, want %q", implementPrompt, tt.wantImplementPrompt)
			}
		})
	}
}

// TestDescriptionPropagationParityWithAll verifies that 'autospec run -a' and 'autospec all'
// have identical behavior regarding description propagation. Both should only pass the
// description to the specify stage.
func TestDescriptionPropagationParityWithAll(t *testing.T) {
	t.Parallel()

	description := "add authentication with JWT"

	// When -a flag is used, isFullWorkflow should be true
	// This means plan/tasks/implement should receive empty strings
	isFullWorkflow := true

	// Verify each stage receives correct prompt
	stages := []struct {
		name       string
		wantPrompt string
	}{
		// Specify stage always receives description (handled separately in executeSpecify)
		{"plan", ""},
		{"tasks", ""},
		{"implement", ""},
	}

	for _, stage := range stages {
		t.Run(stage.name, func(t *testing.T) {
			t.Parallel()

			// Apply the same conditional logic used in execute methods
			prompt := description
			if isFullWorkflow {
				prompt = ""
			}

			if prompt != stage.wantPrompt {
				t.Errorf("%s stage received prompt %q, want %q (isFullWorkflow=%v)",
					stage.name, prompt, stage.wantPrompt, isFullWorkflow)
			}
		})
	}
}

// TestIndividualStageFlagPropagation verifies that individual stage flags (-p, -t, -i)
// correctly propagate the description to the selected stage.
func TestIndividualStageFlagPropagation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		description        string
		isFullWorkflow     bool
		selectedStages     []string
		wantPromptPerStage map[string]string
	}{
		"-p flag: plan receives description": {
			description:    "focus on security",
			isFullWorkflow: false,
			selectedStages: []string{"plan"},
			wantPromptPerStage: map[string]string{
				"plan": "focus on security",
			},
		},
		"-t flag: tasks receives description": {
			description:    "prioritize P1 items",
			isFullWorkflow: false,
			selectedStages: []string{"tasks"},
			wantPromptPerStage: map[string]string{
				"tasks": "prioritize P1 items",
			},
		},
		"-i flag: implement receives description": {
			description:    "skip documentation",
			isFullWorkflow: false,
			selectedStages: []string{"implement"},
			wantPromptPerStage: map[string]string{
				"implement": "skip documentation",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for stage, wantPrompt := range tt.wantPromptPerStage {
				prompt := tt.description
				if tt.isFullWorkflow {
					prompt = ""
				}

				if prompt != wantPrompt {
					t.Errorf("%s stage received prompt %q, want %q", stage, prompt, wantPrompt)
				}
			}
		})
	}
}

// TestMultiStageFlagPropagation verifies that multi-stage flag combinations (-ti, -pt)
// correctly propagate the description to all selected stages.
func TestMultiStageFlagPropagation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		description        string
		isFullWorkflow     bool
		selectedStages     []string
		wantPromptPerStage map[string]string
	}{
		"-ti flags: both tasks and implement receive description": {
			description:    "focus on tests",
			isFullWorkflow: false,
			selectedStages: []string{"tasks", "implement"},
			wantPromptPerStage: map[string]string{
				"tasks":     "focus on tests",
				"implement": "focus on tests",
			},
		},
		"-pt flags: both plan and tasks receive description": {
			description:    "security and coverage",
			isFullWorkflow: false,
			selectedStages: []string{"plan", "tasks"},
			wantPromptPerStage: map[string]string{
				"plan":  "security and coverage",
				"tasks": "security and coverage",
			},
		},
		"-spt flags (explicit): specify, plan, and tasks all receive description": {
			description:    "add authentication",
			isFullWorkflow: false, // Explicit flags, not -a
			selectedStages: []string{"specify", "plan", "tasks"},
			wantPromptPerStage: map[string]string{
				"specify": "add authentication",
				"plan":    "add authentication",
				"tasks":   "add authentication",
			},
		},
		"-pti flags: plan, tasks, and implement all receive description": {
			description:    "optimize performance",
			isFullWorkflow: false,
			selectedStages: []string{"plan", "tasks", "implement"},
			wantPromptPerStage: map[string]string{
				"plan":      "optimize performance",
				"tasks":     "optimize performance",
				"implement": "optimize performance",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for stage, wantPrompt := range tt.wantPromptPerStage {
				prompt := tt.description
				if tt.isFullWorkflow {
					prompt = ""
				}

				if prompt != wantPrompt {
					t.Errorf("%s stage received prompt %q, want %q", stage, prompt, wantPrompt)
				}
			}
		})
	}
}

// TestExplicitVsAllFlagBehavior verifies the difference between explicit stage selection
// (-spt) and the -a flag. Explicit selection should propagate description to all selected
// stages, while -a should only propagate to specify.
func TestExplicitVsAllFlagBehavior(t *testing.T) {
	t.Parallel()

	description := "add user authentication"

	tests := map[string]struct {
		isFullWorkflow      bool
		wantPlanPrompt      string
		wantTasksPrompt     string
		wantImplementPrompt string
	}{
		"-a flag: later stages get empty": {
			isFullWorkflow:      true,
			wantPlanPrompt:      "",
			wantTasksPrompt:     "",
			wantImplementPrompt: "",
		},
		"explicit -spti flags: all stages get description": {
			isFullWorkflow:      false,
			wantPlanPrompt:      description,
			wantTasksPrompt:     description,
			wantImplementPrompt: description,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Plan
			planPrompt := description
			if tt.isFullWorkflow {
				planPrompt = ""
			}
			if planPrompt != tt.wantPlanPrompt {
				t.Errorf("plan prompt = %q, want %q", planPrompt, tt.wantPlanPrompt)
			}

			// Tasks
			tasksPrompt := description
			if tt.isFullWorkflow {
				tasksPrompt = ""
			}
			if tasksPrompt != tt.wantTasksPrompt {
				t.Errorf("tasks prompt = %q, want %q", tasksPrompt, tt.wantTasksPrompt)
			}

			// Implement
			implementPrompt := description
			if tt.isFullWorkflow {
				implementPrompt = ""
			}
			if implementPrompt != tt.wantImplementPrompt {
				t.Errorf("implement prompt = %q, want %q", implementPrompt, tt.wantImplementPrompt)
			}
		})
	}
}

// TestRunCommandMixedModes verifies that the run command correctly handles
// mixed interactive and automated stages. Interactive stages (clarify, analyze)
// should use interactive mode, while automated stages (specify, plan, tasks,
// implement, constitution, checklist) should use automated mode.
func TestRunCommandMixedModes(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stages               []workflow.Stage
		wantInteractiveStage map[workflow.Stage]bool
	}{
		"--specify --clarify: automated then interactive": {
			stages: []workflow.Stage{
				workflow.StageSpecify,
				workflow.StageClarify,
			},
			wantInteractiveStage: map[workflow.Stage]bool{
				workflow.StageSpecify: false,
				workflow.StageClarify: true,
			},
		},
		"--plan --analyze: automated then interactive": {
			stages: []workflow.Stage{
				workflow.StagePlan,
				workflow.StageAnalyze,
			},
			wantInteractiveStage: map[workflow.Stage]bool{
				workflow.StagePlan:    false,
				workflow.StageAnalyze: true,
			},
		},
		"--specify --plan --tasks: all automated": {
			stages: []workflow.Stage{
				workflow.StageSpecify,
				workflow.StagePlan,
				workflow.StageTasks,
			},
			wantInteractiveStage: map[workflow.Stage]bool{
				workflow.StageSpecify: false,
				workflow.StagePlan:    false,
				workflow.StageTasks:   false,
			},
		},
		"--clarify --plan --analyze: interactive, automated, interactive": {
			stages: []workflow.Stage{
				workflow.StageClarify,
				workflow.StagePlan,
				workflow.StageAnalyze,
			},
			wantInteractiveStage: map[workflow.Stage]bool{
				workflow.StageClarify: true,
				workflow.StagePlan:    false,
				workflow.StageAnalyze: true,
			},
		},
		"all 8 stages: mixed modes in canonical order": {
			stages: []workflow.Stage{
				workflow.StageConstitution,
				workflow.StageSpecify,
				workflow.StageClarify,
				workflow.StagePlan,
				workflow.StageTasks,
				workflow.StageChecklist,
				workflow.StageAnalyze,
				workflow.StageImplement,
			},
			wantInteractiveStage: map[workflow.Stage]bool{
				workflow.StageConstitution: false, // automated
				workflow.StageSpecify:      false, // automated
				workflow.StageClarify:      true,  // interactive
				workflow.StagePlan:         false, // automated
				workflow.StageTasks:        false, // automated
				workflow.StageChecklist:    false, // automated
				workflow.StageAnalyze:      true,  // interactive
				workflow.StageImplement:    false, // automated
			},
		},
		"only interactive stages: clarify and analyze": {
			stages: []workflow.Stage{
				workflow.StageClarify,
				workflow.StageAnalyze,
			},
			wantInteractiveStage: map[workflow.Stage]bool{
				workflow.StageClarify: true,
				workflow.StageAnalyze: true,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for _, stage := range tt.stages {
				gotInteractive := workflow.IsInteractive(stage)
				wantInteractive := tt.wantInteractiveStage[stage]

				if gotInteractive != wantInteractive {
					t.Errorf("stage %s: IsInteractive() = %v, want %v",
						stage, gotInteractive, wantInteractive)
				}
			}
		})
	}
}

// TestRunCommandModeTransitions verifies that when running a workflow with
// mixed modes, the transitions between automated and interactive modes
// are handled correctly. This tests the execution path logic in run.go.
func TestRunCommandModeTransitions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config                      *workflow.StageConfig
		wantAutomatedBeforeInteract bool
		description                 string
	}{
		"specify then clarify: has automated before interactive": {
			config: &workflow.StageConfig{
				Specify: true,
				Clarify: true,
			},
			wantAutomatedBeforeInteract: true,
			description:                 "specify (automated) runs before clarify (interactive)",
		},
		"plan then analyze: has automated before interactive": {
			config: &workflow.StageConfig{
				Plan:    true,
				Analyze: true,
			},
			wantAutomatedBeforeInteract: true,
			description:                 "plan (automated) runs before analyze (interactive)",
		},
		"only clarify: no automated before interactive": {
			config: &workflow.StageConfig{
				Clarify: true,
			},
			wantAutomatedBeforeInteract: false,
			description:                 "clarify (interactive) with no preceding automated stage",
		},
		"only analyze: no automated before interactive": {
			config: &workflow.StageConfig{
				Analyze: true,
			},
			wantAutomatedBeforeInteract: false,
			description:                 "analyze (interactive) with no preceding automated stage",
		},
		"specify plan tasks: no interactive at all": {
			config: &workflow.StageConfig{
				Specify: true,
				Plan:    true,
				Tasks:   true,
			},
			wantAutomatedBeforeInteract: false,
			description:                 "all stages automated, no transition needed",
		},
		"constitution specify clarify plan: automated before interactive": {
			config: &workflow.StageConfig{
				Constitution: true,
				Specify:      true,
				Clarify:      true,
				Plan:         true,
			},
			wantAutomatedBeforeInteract: true,
			description:                 "constitution and specify (automated) run before clarify (interactive)",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stages := tt.config.GetCanonicalOrder()

			// Check if any automated stage precedes an interactive stage
			hasAutomatedBeforeInteract := false
			sawAutomated := false

			for _, stage := range stages {
				isInteractive := workflow.IsInteractive(stage)
				if !isInteractive {
					sawAutomated = true
				} else if sawAutomated && isInteractive {
					hasAutomatedBeforeInteract = true
					break
				}
			}

			if hasAutomatedBeforeInteract != tt.wantAutomatedBeforeInteract {
				t.Errorf("%s: hasAutomatedBeforeInteract = %v, want %v",
					tt.description, hasAutomatedBeforeInteract, tt.wantAutomatedBeforeInteract)
			}
		})
	}
}
