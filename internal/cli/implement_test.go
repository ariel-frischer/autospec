// Package cli_test tests the implement command including execution modes (phases, tasks, single-session) and argument parsing.
// Related: internal/cli/stages/implement.go
// Tags: cli, implement, command, workflow, phases, tasks, execution, modes
package cli

import (
	"testing"

	"github.com/ariel-frischer/autospec/internal/cli/stages"
	"github.com/spf13/cobra"
)

// getImplementCmd finds the implement command from rootCmd
func getImplementCmd() *cobra.Command {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "implement [spec-name-or-prompt]" {
			return cmd
		}
	}
	return nil
}

// TestParseImplementArgs tests the ParseImplementArgs function.
// This verifies that we correctly distinguish between spec-names and prompts.
func TestParseImplementArgs(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args         []string
		wantSpecName string
		wantPrompt   string
	}{
		"no args": {
			args:         []string{},
			wantSpecName: "",
			wantPrompt:   "",
		},
		"spec name only": {
			args:         []string{"003-command-timeout"},
			wantSpecName: "003-command-timeout",
			wantPrompt:   "",
		},
		"spec name with hyphenated feature": {
			args:         []string{"004-workflow-progress-indicators"},
			wantSpecName: "004-workflow-progress-indicators",
			wantPrompt:   "",
		},
		"prompt only": {
			args:         []string{"Focus", "on", "documentation"},
			wantSpecName: "",
			wantPrompt:   "Focus on documentation",
		},
		"single word prompt": {
			args:         []string{"Continue"},
			wantSpecName: "",
			wantPrompt:   "Continue",
		},
		"spec name and prompt": {
			args:         []string{"003-feature", "Complete", "the", "tests"},
			wantSpecName: "003-feature",
			wantPrompt:   "Complete the tests",
		},
		"prompt that looks like text": {
			args:         []string{"complete", "remaining", "documentation"},
			wantSpecName: "",
			wantPrompt:   "complete remaining documentation",
		},
		"numeric prompt (not spec name)": {
			args:         []string{"123"},
			wantSpecName: "",
			wantPrompt:   "123",
		},
		"spec with two-digit number": {
			args:         []string{"42-answer"},
			wantSpecName: "42-answer",
			wantPrompt:   "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			specName, prompt := stages.ParseImplementArgs(tc.args)

			if specName != tc.wantSpecName {
				t.Errorf("specName = %q, want %q", specName, tc.wantSpecName)
			}

			if prompt != tc.wantPrompt {
				t.Errorf("prompt = %q, want %q", prompt, tc.wantPrompt)
			}
		})
	}
}

// TestImplementPhaseFlagsRegistered tests that phase execution flags are properly registered
func TestImplementPhaseFlagsRegistered(t *testing.T) {
	implementCmd := getImplementCmd()
	if implementCmd == nil {
		t.Fatal("implement command not found")
	}

	// Verify --phases flag exists
	phasesFlag := implementCmd.Flags().Lookup("phases")
	if phasesFlag == nil {
		t.Error("--phases flag not registered")
	} else {
		if phasesFlag.DefValue != "false" {
			t.Errorf("--phases default = %q, want %q", phasesFlag.DefValue, "false")
		}
	}

	// Verify --phase flag exists
	phaseFlag := implementCmd.Flags().Lookup("phase")
	if phaseFlag == nil {
		t.Error("--phase flag not registered")
	} else {
		if phaseFlag.DefValue != "0" {
			t.Errorf("--phase default = %q, want %q", phaseFlag.DefValue, "0")
		}
	}

	// Verify --from-phase flag exists
	fromPhaseFlag := implementCmd.Flags().Lookup("from-phase")
	if fromPhaseFlag == nil {
		t.Error("--from-phase flag not registered")
	} else {
		if fromPhaseFlag.DefValue != "0" {
			t.Errorf("--from-phase default = %q, want %q", fromPhaseFlag.DefValue, "0")
		}
	}
}

// TestImplementPhaseFlagsMutualExclusivity tests that phase flags are mutually exclusive
// Note: Cobra's MarkFlagsMutuallyExclusive handles this at runtime, so we verify the flags are set up
func TestImplementPhaseFlagsMutualExclusivity(t *testing.T) {
	implementCmd := getImplementCmd()
	if implementCmd == nil {
		t.Fatal("implement command not found")
	}

	// The mutual exclusivity is enforced by Cobra's MarkFlagsMutuallyExclusive
	// We verify all three flags exist and can be looked up
	flags := []string{"phases", "phase", "from-phase"}
	for _, flagName := range flags {
		flag := implementCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("flag --%s not found, mutual exclusivity setup requires all flags", flagName)
		}
	}
}

// TestTasksFlagParsing tests that the --tasks flag is properly registered and has correct defaults
func TestTasksFlagParsing(t *testing.T) {
	implementCmd := getImplementCmd()
	if implementCmd == nil {
		t.Fatal("implement command not found")
	}

	// Verify --tasks flag exists
	tasksFlag := implementCmd.Flags().Lookup("tasks")
	if tasksFlag == nil {
		t.Error("--tasks flag not registered")
		return
	}

	// Verify default value
	if tasksFlag.DefValue != "false" {
		t.Errorf("--tasks default = %q, want %q", tasksFlag.DefValue, "false")
	}

	// Verify flag description
	if tasksFlag.Usage == "" {
		t.Error("--tasks flag has no usage description")
	}
}

// TestFromTaskFlagParsing tests that the --from-task flag is properly registered and handles task IDs
func TestFromTaskFlagParsing(t *testing.T) {
	implementCmd := getImplementCmd()
	if implementCmd == nil {
		t.Fatal("implement command not found")
	}

	// Verify --from-task flag exists
	fromTaskFlag := implementCmd.Flags().Lookup("from-task")
	if fromTaskFlag == nil {
		t.Error("--from-task flag not registered")
		return
	}

	// Verify default value is empty string
	if fromTaskFlag.DefValue != "" {
		t.Errorf("--from-task default = %q, want %q", fromTaskFlag.DefValue, "")
	}

	// Verify flag description
	if fromTaskFlag.Usage == "" {
		t.Error("--from-task flag has no usage description")
	}
}

// TestTaskFlagsRegistered tests that all task execution flags are properly registered
func TestTaskFlagsRegistered(t *testing.T) {
	implementCmd := getImplementCmd()
	if implementCmd == nil {
		t.Fatal("implement command not found")
	}

	flags := map[string]string{
		"tasks":     "false",
		"from-task": "",
	}

	for flagName, expectedDefault := range flags {
		flag := implementCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("flag --%s not registered", flagName)
			continue
		}
		if flag.DefValue != expectedDefault {
			t.Errorf("--%s default = %q, want %q", flagName, flag.DefValue, expectedDefault)
		}
	}
}

// TestSingleSessionFlagRegistered tests that the --single-session flag is properly registered
func TestSingleSessionFlagRegistered(t *testing.T) {
	implementCmd := getImplementCmd()
	if implementCmd == nil {
		t.Fatal("implement command not found")
	}

	flag := implementCmd.Flags().Lookup("single-session")
	if flag == nil {
		t.Error("--single-session flag not registered")
		return
	}

	if flag.DefValue != "false" {
		t.Errorf("--single-session default = %q, want %q", flag.DefValue, "false")
	}

	if flag.Usage == "" {
		t.Error("--single-session flag has no usage description")
	}
}

// TestTaskPhasesMutualExclusivity tests that --tasks flag is mutually exclusive with phase flags
func TestTaskPhasesMutualExclusivity(t *testing.T) {
	implementCmd := getImplementCmd()
	if implementCmd == nil {
		t.Fatal("implement command not found")
	}

	// The mutual exclusivity is enforced by Cobra's MarkFlagsMutuallyExclusive
	// We verify all relevant flags exist and can be looked up
	tests := map[string]struct {
		flags []string
	}{
		"tasks and phases": {
			flags: []string{"tasks", "phases"},
		},
		"tasks and phase": {
			flags: []string{"tasks", "phase"},
		},
		"tasks and from-phase": {
			flags: []string{"tasks", "from-phase"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			for _, flagName := range tc.flags {
				flag := implementCmd.Flags().Lookup(flagName)
				if flag == nil {
					t.Errorf("flag --%s not found, mutual exclusivity setup requires all flags", flagName)
				}
			}
		})
	}
}

// TestSpecNamePattern tests that spec names are correctly identified via ParseImplementArgs
func TestSpecNamePattern(t *testing.T) {
	tests := map[string]struct {
		input      string
		isSpecName bool
	}{
		"valid three-digit spec": {
			input:      "003-command-timeout",
			isSpecName: true,
		},
		"valid two-digit spec": {
			input:      "42-answer",
			isSpecName: true,
		},
		"valid single-digit spec": {
			input:      "1-first",
			isSpecName: true,
		},
		"valid with multiple hyphens": {
			input:      "004-workflow-progress-indicators",
			isSpecName: true,
		},
		"invalid: uppercase": {
			input:      "003-Command-Timeout",
			isSpecName: false,
		},
		"invalid: no hyphen": {
			input:      "003command",
			isSpecName: false,
		},
		"invalid: starts with text": {
			input:      "feature-003",
			isSpecName: false,
		},
		"invalid: just number": {
			input:      "123",
			isSpecName: false,
		},
		"invalid: text only": {
			input:      "complete-the-tasks",
			isSpecName: false,
		},
		"invalid: special chars": {
			input:      "003-feature_name",
			isSpecName: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Use ParseImplementArgs to test spec name recognition
			specName, _ := stages.ParseImplementArgs([]string{tc.input})
			matches := specName != ""
			if matches != tc.isSpecName {
				t.Errorf("pattern match for %q = %v, want %v", tc.input, matches, tc.isSpecName)
			}
		})
	}
}

// TestResolveExecutionMode tests the ResolveExecutionMode function.
// This validates that CLI flags override config implement_method correctly.
func TestResolveExecutionMode(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		flags            stages.ExecutionModeFlags
		flagsChanged     bool
		configMethod     string
		wantRunAllPhases bool
		wantTaskMode     bool
		wantSinglePhase  int
		wantFromPhase    int
		wantFromTask     string
	}{
		"config phases + no flags = phases mode": {
			flags:            stages.ExecutionModeFlags{},
			flagsChanged:     false,
			configMethod:     "phases",
			wantRunAllPhases: true,
			wantTaskMode:     false,
		},
		"config tasks + no flags = tasks mode": {
			flags:            stages.ExecutionModeFlags{},
			flagsChanged:     false,
			configMethod:     "tasks",
			wantRunAllPhases: false,
			wantTaskMode:     true,
		},
		"config single-session + no flags = single-session mode": {
			flags:            stages.ExecutionModeFlags{},
			flagsChanged:     false,
			configMethod:     "single-session",
			wantRunAllPhases: false,
			wantTaskMode:     false,
		},
		"empty config + no flags = default mode": {
			flags:            stages.ExecutionModeFlags{},
			flagsChanged:     false,
			configMethod:     "",
			wantRunAllPhases: false,
			wantTaskMode:     false,
		},
		"config phases + --tasks flag = tasks mode (CLI override)": {
			flags:            stages.ExecutionModeFlags{TasksFlag: true},
			flagsChanged:     true,
			configMethod:     "phases",
			wantRunAllPhases: false,
			wantTaskMode:     true,
		},
		"config tasks + --phases flag = phases mode (CLI override)": {
			flags:            stages.ExecutionModeFlags{PhasesFlag: true},
			flagsChanged:     true,
			configMethod:     "tasks",
			wantRunAllPhases: true,
			wantTaskMode:     false,
		},
		"config single-session + --phases flag = phases mode (CLI override)": {
			flags:            stages.ExecutionModeFlags{PhasesFlag: true},
			flagsChanged:     true,
			configMethod:     "single-session",
			wantRunAllPhases: true,
			wantTaskMode:     false,
		},
		"config single-session + --tasks flag = tasks mode (CLI override)": {
			flags:            stages.ExecutionModeFlags{TasksFlag: true},
			flagsChanged:     true,
			configMethod:     "single-session",
			wantRunAllPhases: false,
			wantTaskMode:     true,
		},
		"config phases + --phase 3 = single phase mode (CLI override)": {
			flags:            stages.ExecutionModeFlags{PhaseFlag: 3},
			flagsChanged:     true,
			configMethod:     "phases",
			wantRunAllPhases: false,
			wantTaskMode:     false,
			wantSinglePhase:  3,
		},
		"config tasks + --from-phase 2 = from-phase mode (CLI override)": {
			flags:            stages.ExecutionModeFlags{FromPhaseFlag: 2},
			flagsChanged:     true,
			configMethod:     "tasks",
			wantRunAllPhases: false,
			wantTaskMode:     false,
			wantFromPhase:    2,
		},
		"config phases + --from-task T003 = task mode (CLI override)": {
			flags:            stages.ExecutionModeFlags{FromTaskFlag: "T003"},
			flagsChanged:     true,
			configMethod:     "phases",
			wantRunAllPhases: false,
			wantTaskMode:     false,
			wantFromTask:     "T003",
		},
		"config phases + --single-session flag = single-session mode (CLI override)": {
			flags:            stages.ExecutionModeFlags{SingleSessionFlag: true},
			flagsChanged:     true,
			configMethod:     "phases",
			wantRunAllPhases: false,
			wantTaskMode:     false,
		},
		"config tasks + --single-session flag = single-session mode (CLI override)": {
			flags:            stages.ExecutionModeFlags{SingleSessionFlag: true},
			flagsChanged:     true,
			configMethod:     "tasks",
			wantRunAllPhases: false,
			wantTaskMode:     false,
		},
		"--single-session disables phases even if PhasesFlag is true": {
			flags:            stages.ExecutionModeFlags{PhasesFlag: true, SingleSessionFlag: true},
			flagsChanged:     true,
			configMethod:     "",
			wantRunAllPhases: false,
			wantTaskMode:     false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := stages.ResolveExecutionMode(tt.flags, tt.flagsChanged, tt.configMethod)

			if result.RunAllPhases != tt.wantRunAllPhases {
				t.Errorf("RunAllPhases = %v, want %v", result.RunAllPhases, tt.wantRunAllPhases)
			}
			if result.TaskMode != tt.wantTaskMode {
				t.Errorf("TaskMode = %v, want %v", result.TaskMode, tt.wantTaskMode)
			}
			if result.SinglePhase != tt.wantSinglePhase {
				t.Errorf("SinglePhase = %v, want %v", result.SinglePhase, tt.wantSinglePhase)
			}
			if result.FromPhase != tt.wantFromPhase {
				t.Errorf("FromPhase = %v, want %v", result.FromPhase, tt.wantFromPhase)
			}
			if result.FromTask != tt.wantFromTask {
				t.Errorf("FromTask = %q, want %q", result.FromTask, tt.wantFromTask)
			}
		})
	}
}

// TestResolveExecutionModeWithDefaultConfig tests that the default config (phases) is applied correctly
// when no implement_method is explicitly set in the config file
func TestResolveExecutionModeWithDefaultConfig(t *testing.T) {
	t.Parallel()

	// The default value is "phases" per internal/config/defaults.go
	// When config.Load() returns ImplementMethod: "phases", we expect phases mode
	result := stages.ResolveExecutionMode(stages.ExecutionModeFlags{}, false, "phases")

	if !result.RunAllPhases {
		t.Error("expected phases mode (RunAllPhases=true)")
	}
	if result.TaskMode {
		t.Error("expected TaskMode=false for phases mode")
	}
}
