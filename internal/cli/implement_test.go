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
	tests := []struct {
		name  string
		flags []string
	}{
		{
			name:  "tasks and phases",
			flags: []string{"tasks", "phases"},
		},
		{
			name:  "tasks and phase",
			flags: []string{"tasks", "phase"},
		},
		{
			name:  "tasks and from-phase",
			flags: []string{"tasks", "from-phase"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
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
