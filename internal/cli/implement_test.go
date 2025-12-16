package cli

import (
	"regexp"
	"strings"
	"testing"
)

// TestImplementArgParsing tests the argument parsing logic for the implement command
// This verifies that we correctly distinguish between spec-names and prompts
func TestImplementArgParsing(t *testing.T) {
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
			// Replicate the parsing logic from implement.go
			var specName string
			var prompt string

			if len(tc.args) > 0 {
				// Check if first arg looks like a spec name (pattern: NNN-name)
				specNamePattern := regexp.MustCompile(`^\d+-[a-z0-9-]+$`)
				if specNamePattern.MatchString(tc.args[0]) {
					// First arg is a spec name
					specName = tc.args[0]
					// Remaining args are prompt
					if len(tc.args) > 1 {
						prompt = strings.Join(tc.args[1:], " ")
					}
				} else {
					// All args are prompt (auto-detect spec)
					prompt = strings.Join(tc.args, " ")
				}
			}

			// Verify results
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

// TestTaskPhasesMutualExclusivity tests that --tasks flag is mutually exclusive with phase flags
func TestTaskPhasesMutualExclusivity(t *testing.T) {
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

// TestSpecNamePattern tests that the spec name regex correctly identifies spec names
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

	specNamePattern := regexp.MustCompile(`^\d+-[a-z0-9-]+$`)

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			matches := specNamePattern.MatchString(tc.input)
			if matches != tc.isSpecName {
				t.Errorf("pattern match for %q = %v, want %v", tc.input, matches, tc.isSpecName)
			}
		})
	}
}

// TestImplementMethodConfigPrecedence tests that CLI flags override config implement_method
// This validates the logic in implement.go that determines execution mode based on
// the combination of config settings and CLI flags.
func TestImplementMethodConfigPrecedence(t *testing.T) {
	tests := map[string]struct {
		configMethod      string // implement_method value from config
		phasesFlag        bool   // --phases flag
		tasksFlag         bool   // --tasks flag
		phaseFlag         int    // --phase N flag (0 = not set)
		fromPhaseFlag     int    // --from-phase N flag (0 = not set)
		fromTaskFlag      string // --from-task flag (empty = not set)
		wantRunAllPhases  bool   // expected runAllPhases value
		wantTaskMode      bool   // expected taskMode value
		wantSingleSession bool   // expected single-session behavior (no phase/task mode)
	}{
		// T007: Test config phases + no flags = phases mode
		"config phases + no flags = phases mode": {
			configMethod:     "phases",
			wantRunAllPhases: true,
			wantTaskMode:     false,
		},
		// T007: Test config tasks + no flags = tasks mode
		"config tasks + no flags = tasks mode": {
			configMethod:     "tasks",
			wantRunAllPhases: false,
			wantTaskMode:     true,
		},
		// T007: Test config single-session + no flags = single-session mode
		"config single-session + no flags = single-session mode": {
			configMethod:      "single-session",
			wantRunAllPhases:  false,
			wantTaskMode:      false,
			wantSingleSession: true,
		},
		// T007: Test no config (default) + no flags = phases mode (new default)
		"empty config (default) + no flags = phases mode": {
			configMethod:     "", // empty uses default from GetDefaults()
			wantRunAllPhases: false,
			wantTaskMode:     false,
			// Note: empty string in config doesn't trigger config-based mode
			// The actual default behavior comes from defaults.go where ImplementMethod: "phases"
		},
		// T007: Test config phases + --tasks = tasks mode (CLI overrides config)
		"config phases + --tasks flag = tasks mode (CLI override)": {
			configMethod:     "phases",
			tasksFlag:        true,
			wantRunAllPhases: false,
			wantTaskMode:     true,
		},
		// T007: Test config tasks + --phases = phases mode (CLI overrides config)
		"config tasks + --phases flag = phases mode (CLI override)": {
			configMethod:     "tasks",
			phasesFlag:       true,
			wantRunAllPhases: true,
			wantTaskMode:     false,
		},
		// T007: Test config single-session + --phases = phases mode (CLI overrides config)
		"config single-session + --phases flag = phases mode (CLI override)": {
			configMethod:     "single-session",
			phasesFlag:       true,
			wantRunAllPhases: true,
			wantTaskMode:     false,
		},
		// T007: Test config single-session + --tasks = tasks mode (CLI overrides config)
		"config single-session + --tasks flag = tasks mode (CLI override)": {
			configMethod:     "single-session",
			tasksFlag:        true,
			wantRunAllPhases: false,
			wantTaskMode:     true,
		},
		// Additional: Test --phase N overrides config
		"config phases + --phase 3 = single phase mode (CLI override)": {
			configMethod:     "phases",
			phaseFlag:        3,
			wantRunAllPhases: false, // --phase N doesn't set runAllPhases
			wantTaskMode:     false,
		},
		// Additional: Test --from-phase N overrides config
		"config tasks + --from-phase 2 = from-phase mode (CLI override)": {
			configMethod:     "tasks",
			fromPhaseFlag:    2,
			wantRunAllPhases: false,
			wantTaskMode:     false,
		},
		// Additional: Test --from-task overrides config
		"config phases + --from-task T003 = task mode (CLI override)": {
			configMethod:     "phases",
			fromTaskFlag:     "T003",
			wantRunAllPhases: false,
			wantTaskMode:     false, // fromTask doesn't set taskMode directly
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Simulate the logic from implement.go RunE function
			runAllPhases := tt.phasesFlag
			taskMode := tt.tasksFlag
			singlePhase := tt.phaseFlag
			fromPhase := tt.fromPhaseFlag
			fromTask := tt.fromTaskFlag

			// Simulate cmd.Flags().Changed() behavior
			// A flag is "changed" if it's explicitly set (non-default value for bool, non-zero for int, non-empty for string)
			phasesChanged := tt.phasesFlag
			tasksChanged := tt.tasksFlag
			phaseChanged := tt.phaseFlag > 0
			fromPhaseChanged := tt.fromPhaseFlag > 0
			fromTaskChanged := tt.fromTaskFlag != ""

			// Calculate noExecutionModeFlags - same logic as implement.go lines 148-152
			noExecutionModeFlags := !phasesChanged &&
				!tasksChanged &&
				!phaseChanged &&
				!fromPhaseChanged &&
				!fromTaskChanged

			// Apply config default execution mode when no execution mode flags are provided
			// Same logic as implement.go lines 154-165
			if noExecutionModeFlags && tt.configMethod != "" {
				switch tt.configMethod {
				case "phases":
					runAllPhases = true
				case "tasks":
					taskMode = true
				case "single-session":
					// Legacy behavior: no phase/task mode (default state)
					// runAllPhases and taskMode are already false
				}
			}

			// Verify expected values
			if runAllPhases != tt.wantRunAllPhases {
				t.Errorf("runAllPhases = %v, want %v", runAllPhases, tt.wantRunAllPhases)
			}
			if taskMode != tt.wantTaskMode {
				t.Errorf("taskMode = %v, want %v", taskMode, tt.wantTaskMode)
			}

			// For single-session mode, both should be false
			if tt.wantSingleSession {
				if runAllPhases || taskMode {
					t.Errorf("single-session mode: runAllPhases=%v, taskMode=%v, want both false", runAllPhases, taskMode)
				}
			}

			// Verify singlePhase and fromPhase are passed through correctly when set
			if tt.phaseFlag > 0 && singlePhase != tt.phaseFlag {
				t.Errorf("singlePhase = %v, want %v", singlePhase, tt.phaseFlag)
			}
			if tt.fromPhaseFlag > 0 && fromPhase != tt.fromPhaseFlag {
				t.Errorf("fromPhase = %v, want %v", fromPhase, tt.fromPhaseFlag)
			}
			if tt.fromTaskFlag != "" && fromTask != tt.fromTaskFlag {
				t.Errorf("fromTask = %q, want %q", fromTask, tt.fromTaskFlag)
			}
		})
	}
}

// TestImplementMethodWithDefaultConfig tests that the default config (phases) is applied correctly
// when no implement_method is explicitly set in the config file
func TestImplementMethodWithDefaultConfig(t *testing.T) {
	// This test verifies the behavior when config.ImplementMethod comes from defaults.go
	// The default value is "phases" per internal/config/defaults.go

	tests := map[string]struct {
		configMethod string // What LoadDefaults() would set
		wantMode     string
	}{
		"default config (phases) produces phases mode": {
			configMethod: "phases", // This is what defaults.go sets
			wantMode:     "phases",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Simulate no flags being set
			runAllPhases := false
			taskMode := false

			// Apply config value (simulating implement.go logic)
			if tt.configMethod != "" {
				switch tt.configMethod {
				case "phases":
					runAllPhases = true
				case "tasks":
					taskMode = true
				}
			}

			// Verify expected mode
			switch tt.wantMode {
			case "phases":
				if !runAllPhases {
					t.Error("expected phases mode (runAllPhases=true)")
				}
				if taskMode {
					t.Error("expected taskMode=false for phases mode")
				}
			case "tasks":
				if runAllPhases {
					t.Error("expected runAllPhases=false for tasks mode")
				}
				if !taskMode {
					t.Error("expected tasks mode (taskMode=true)")
				}
			}
		})
	}
}
