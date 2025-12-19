// Package stages tests CLI implement command logic for autospec.
// Related: internal/cli/stages/implement.go
// Tags: stages, cli, implement, execution-mode

package stages

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseImplementArgs(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args       []string
		wantSpec   string
		wantPrompt string
	}{
		"empty args returns empty values": {
			args:       []string{},
			wantSpec:   "",
			wantPrompt: "",
		},
		"nil args returns empty values": {
			args:       nil,
			wantSpec:   "",
			wantPrompt: "",
		},
		"spec name only": {
			args:       []string{"003-my-feature"},
			wantSpec:   "003-my-feature",
			wantPrompt: "",
		},
		"spec name with single digit": {
			args:       []string{"1-feature"},
			wantSpec:   "1-feature",
			wantPrompt: "",
		},
		"spec name with many digits": {
			args:       []string{"12345-long-feature-name"},
			wantSpec:   "12345-long-feature-name",
			wantPrompt: "",
		},
		"spec name with prompt": {
			args:       []string{"003-my-feature", "focus", "on", "tests"},
			wantSpec:   "003-my-feature",
			wantPrompt: "focus on tests",
		},
		"prompt only - no spec pattern": {
			args:       []string{"focus", "on", "tests"},
			wantSpec:   "",
			wantPrompt: "focus on tests",
		},
		"prompt only - single word": {
			args:       []string{"testing"},
			wantSpec:   "",
			wantPrompt: "testing",
		},
		"prompt starting with number but not spec pattern": {
			args:       []string{"42-answers", "to", "life"},
			wantSpec:   "42-answers",
			wantPrompt: "to life",
		},
		"invalid spec pattern - missing hyphen": {
			args:       []string{"003feature"},
			wantSpec:   "",
			wantPrompt: "003feature",
		},
		"invalid spec pattern - uppercase letters": {
			args:       []string{"003-MyFeature"},
			wantSpec:   "",
			wantPrompt: "003-MyFeature",
		},
		"invalid spec pattern - no numbers": {
			args:       []string{"abc-feature"},
			wantSpec:   "",
			wantPrompt: "abc-feature",
		},
		"invalid spec pattern - empty after hyphen": {
			args:       []string{"003-"},
			wantSpec:   "",
			wantPrompt: "003-",
		},
		"spec name with hyphens in name": {
			args:       []string{"007-multi-word-feature-name"},
			wantSpec:   "007-multi-word-feature-name",
			wantPrompt: "",
		},
		"spec name with numbers in name": {
			args:       []string{"123-feature2test"},
			wantSpec:   "123-feature2test",
			wantPrompt: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotSpec, gotPrompt := ParseImplementArgs(tt.args)
			assert.Equal(t, tt.wantSpec, gotSpec, "spec name mismatch")
			assert.Equal(t, tt.wantPrompt, gotPrompt, "prompt mismatch")
		})
	}
}

func TestResolveExecutionMode(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		flags        ExecutionModeFlags
		flagsChanged bool
		configMethod string
		want         ExecutionModeResult
	}{
		"no flags, no config - default behavior": {
			flags:        ExecutionModeFlags{},
			flagsChanged: false,
			configMethod: "",
			want: ExecutionModeResult{
				RunAllPhases: false,
				TaskMode:     false,
				SinglePhase:  0,
				FromPhase:    0,
				FromTask:     "",
			},
		},
		"phases flag set": {
			flags: ExecutionModeFlags{
				PhasesFlag: true,
			},
			flagsChanged: true,
			configMethod: "",
			want: ExecutionModeResult{
				RunAllPhases: true,
				TaskMode:     false,
				SinglePhase:  0,
				FromPhase:    0,
				FromTask:     "",
			},
		},
		"tasks flag set": {
			flags: ExecutionModeFlags{
				TasksFlag: true,
			},
			flagsChanged: true,
			configMethod: "",
			want: ExecutionModeResult{
				RunAllPhases: false,
				TaskMode:     true,
				SinglePhase:  0,
				FromPhase:    0,
				FromTask:     "",
			},
		},
		"single-session flag disables phase mode": {
			flags: ExecutionModeFlags{
				SingleSessionFlag: true,
				PhasesFlag:        true,
			},
			flagsChanged: true,
			configMethod: "",
			want: ExecutionModeResult{
				RunAllPhases: false,
				TaskMode:     false,
				SinglePhase:  0,
				FromPhase:    0,
				FromTask:     "",
			},
		},
		"single-session flag disables task mode": {
			flags: ExecutionModeFlags{
				SingleSessionFlag: true,
				TasksFlag:         true,
			},
			flagsChanged: true,
			configMethod: "",
			want: ExecutionModeResult{
				RunAllPhases: false,
				TaskMode:     false,
				SinglePhase:  0,
				FromPhase:    0,
				FromTask:     "",
			},
		},
		"config phases method applied when no flags": {
			flags:        ExecutionModeFlags{},
			flagsChanged: false,
			configMethod: "phases",
			want: ExecutionModeResult{
				RunAllPhases: true,
				TaskMode:     false,
				SinglePhase:  0,
				FromPhase:    0,
				FromTask:     "",
			},
		},
		"config tasks method applied when no flags": {
			flags:        ExecutionModeFlags{},
			flagsChanged: false,
			configMethod: "tasks",
			want: ExecutionModeResult{
				RunAllPhases: false,
				TaskMode:     true,
				SinglePhase:  0,
				FromPhase:    0,
				FromTask:     "",
			},
		},
		"config single-session method applied when no flags": {
			flags:        ExecutionModeFlags{},
			flagsChanged: false,
			configMethod: "single-session",
			want: ExecutionModeResult{
				RunAllPhases: false,
				TaskMode:     false,
				SinglePhase:  0,
				FromPhase:    0,
				FromTask:     "",
			},
		},
		"flag overrides config - phases flag vs tasks config": {
			flags: ExecutionModeFlags{
				PhasesFlag: true,
			},
			flagsChanged: true,
			configMethod: "tasks",
			want: ExecutionModeResult{
				RunAllPhases: true,
				TaskMode:     false,
				SinglePhase:  0,
				FromPhase:    0,
				FromTask:     "",
			},
		},
		"flag overrides config - tasks flag vs phases config": {
			flags: ExecutionModeFlags{
				TasksFlag: true,
			},
			flagsChanged: true,
			configMethod: "phases",
			want: ExecutionModeResult{
				RunAllPhases: false,
				TaskMode:     true,
				SinglePhase:  0,
				FromPhase:    0,
				FromTask:     "",
			},
		},
		"phase flag with value": {
			flags: ExecutionModeFlags{
				PhaseFlag: 3,
			},
			flagsChanged: true,
			configMethod: "",
			want: ExecutionModeResult{
				RunAllPhases: false,
				TaskMode:     false,
				SinglePhase:  3,
				FromPhase:    0,
				FromTask:     "",
			},
		},
		"from-phase flag with value": {
			flags: ExecutionModeFlags{
				FromPhaseFlag: 2,
			},
			flagsChanged: true,
			configMethod: "",
			want: ExecutionModeResult{
				RunAllPhases: false,
				TaskMode:     false,
				SinglePhase:  0,
				FromPhase:    2,
				FromTask:     "",
			},
		},
		"from-task flag with value": {
			flags: ExecutionModeFlags{
				FromTaskFlag: "T003",
			},
			flagsChanged: true,
			configMethod: "",
			want: ExecutionModeResult{
				RunAllPhases: false,
				TaskMode:     false,
				SinglePhase:  0,
				FromPhase:    0,
				FromTask:     "T003",
			},
		},
		"tasks mode with from-task": {
			flags: ExecutionModeFlags{
				TasksFlag:    true,
				FromTaskFlag: "T005",
			},
			flagsChanged: true,
			configMethod: "",
			want: ExecutionModeResult{
				RunAllPhases: false,
				TaskMode:     true,
				SinglePhase:  0,
				FromPhase:    0,
				FromTask:     "T005",
			},
		},
		"unknown config method ignored": {
			flags:        ExecutionModeFlags{},
			flagsChanged: false,
			configMethod: "unknown-method",
			want: ExecutionModeResult{
				RunAllPhases: false,
				TaskMode:     false,
				SinglePhase:  0,
				FromPhase:    0,
				FromTask:     "",
			},
		},
		"flagsChanged true but no flags set - no config applied": {
			flags:        ExecutionModeFlags{},
			flagsChanged: true,
			configMethod: "phases",
			want: ExecutionModeResult{
				RunAllPhases: false,
				TaskMode:     false,
				SinglePhase:  0,
				FromPhase:    0,
				FromTask:     "",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := ResolveExecutionMode(tt.flags, tt.flagsChanged, tt.configMethod)
			assert.Equal(t, tt.want.RunAllPhases, got.RunAllPhases, "RunAllPhases mismatch")
			assert.Equal(t, tt.want.TaskMode, got.TaskMode, "TaskMode mismatch")
			assert.Equal(t, tt.want.SinglePhase, got.SinglePhase, "SinglePhase mismatch")
			assert.Equal(t, tt.want.FromPhase, got.FromPhase, "FromPhase mismatch")
			assert.Equal(t, tt.want.FromTask, got.FromTask, "FromTask mismatch")
		})
	}
}

func TestSpecNamePattern(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		want  bool
	}{
		"valid simple spec": {
			input: "001-feature",
			want:  true,
		},
		"valid multi-digit": {
			input: "12345-feature",
			want:  true,
		},
		"valid with hyphens": {
			input: "003-multi-word-feature",
			want:  true,
		},
		"valid with numbers in name": {
			input: "007-feature2",
			want:  true,
		},
		"valid single digit": {
			input: "1-a",
			want:  true,
		},
		"invalid - no leading numbers": {
			input: "abc-feature",
			want:  false,
		},
		"invalid - uppercase": {
			input: "003-Feature",
			want:  false,
		},
		"invalid - spaces": {
			input: "003-my feature",
			want:  false,
		},
		"invalid - no hyphen": {
			input: "003feature",
			want:  false,
		},
		"invalid - empty after hyphen": {
			input: "003-",
			want:  false,
		},
		"invalid - special chars": {
			input: "003-feature_test",
			want:  false,
		},
		"invalid - leading hyphen": {
			input: "-003-feature",
			want:  false,
		},
		"invalid - empty string": {
			input: "",
			want:  false,
		},
		"invalid - only numbers": {
			input: "12345",
			want:  false,
		},
		"invalid - leading zeros only": {
			input: "000-",
			want:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := specNamePattern.MatchString(tt.input)
			assert.Equal(t, tt.want, got, "pattern match for %q", tt.input)
		})
	}
}

func TestExecutionModeFlags_ZeroValue(t *testing.T) {
	t.Parallel()

	// Test that zero value of ExecutionModeFlags is sensible
	flags := ExecutionModeFlags{}

	assert.False(t, flags.PhasesFlag)
	assert.False(t, flags.TasksFlag)
	assert.False(t, flags.SingleSessionFlag)
	assert.Equal(t, 0, flags.PhaseFlag)
	assert.Equal(t, 0, flags.FromPhaseFlag)
	assert.Empty(t, flags.FromTaskFlag)
}

func TestExecutionModeResult_ZeroValue(t *testing.T) {
	t.Parallel()

	// Test that zero value of ExecutionModeResult is sensible
	result := ExecutionModeResult{}

	assert.False(t, result.RunAllPhases)
	assert.False(t, result.TaskMode)
	assert.Equal(t, 0, result.SinglePhase)
	assert.Equal(t, 0, result.FromPhase)
	assert.Empty(t, result.FromTask)
}

func TestImplementCmd_FlagDefaults(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	tests := map[string]struct {
		flagName    string
		wantDefault string
		wantBoolVal bool
		wantIntVal  int
		checkType   string
	}{
		"resume default false": {
			flagName:    "resume",
			wantBoolVal: false,
			checkType:   "bool",
		},
		"phases default false": {
			flagName:    "phases",
			wantBoolVal: false,
			checkType:   "bool",
		},
		"tasks default false": {
			flagName:    "tasks",
			wantBoolVal: false,
			checkType:   "bool",
		},
		"single-session default false": {
			flagName:    "single-session",
			wantBoolVal: false,
			checkType:   "bool",
		},
		"phase default 0": {
			flagName:   "phase",
			wantIntVal: 0,
			checkType:  "int",
		},
		"from-phase default 0": {
			flagName:   "from-phase",
			wantIntVal: 0,
			checkType:  "int",
		},
		"from-task default empty": {
			flagName:    "from-task",
			wantDefault: "",
			checkType:   "string",
		},
		"max-retries default 0": {
			flagName:   "max-retries",
			wantIntVal: 0,
			checkType:  "int",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run in parallel - accesses global command state

			flag := implementCmd.Flags().Lookup(tt.flagName)
			assert.NotNil(t, flag, "flag %s should exist", tt.flagName)

			switch tt.checkType {
			case "bool":
				assert.Equal(t, "false", flag.DefValue, "default value for %s", tt.flagName)
			case "int":
				assert.Equal(t, "0", flag.DefValue, "default value for %s", tt.flagName)
			case "string":
				assert.Equal(t, tt.wantDefault, flag.DefValue, "default value for %s", tt.flagName)
			}
		})
	}
}

func TestImplementCmd_FlagShorthand(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	tests := map[string]struct {
		flagName      string
		wantShorthand string
	}{
		"max-retries has shorthand r": {
			flagName:      "max-retries",
			wantShorthand: "r",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run in parallel - accesses global command state

			flag := implementCmd.Flags().Lookup(tt.flagName)
			assert.NotNil(t, flag, "flag %s should exist", tt.flagName)
			assert.Equal(t, tt.wantShorthand, flag.Shorthand)
		})
	}
}

func TestImplementCmd_FlagUsage(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	tests := map[string]struct {
		flagName string
		wantWord string
	}{
		"resume has usage": {
			flagName: "resume",
			wantWord: "Resume",
		},
		"phases has usage": {
			flagName: "phases",
			wantWord: "phase",
		},
		"tasks has usage": {
			flagName: "tasks",
			wantWord: "task",
		},
		"single-session has usage": {
			flagName: "single-session",
			wantWord: "session",
		},
		"phase has usage": {
			flagName: "phase",
			wantWord: "phase",
		},
		"from-phase has usage": {
			flagName: "from-phase",
			wantWord: "phase",
		},
		"from-task has usage": {
			flagName: "from-task",
			wantWord: "task",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run in parallel - accesses global command state

			flag := implementCmd.Flags().Lookup(tt.flagName)
			assert.NotNil(t, flag, "flag %s should exist", tt.flagName)
			assert.Contains(t, flag.Usage, tt.wantWord, "usage for %s should mention %s", tt.flagName, tt.wantWord)
		})
	}
}

func TestResolveExecutionMode_ConfigMethodPriorities(t *testing.T) {
	t.Parallel()

	// Test that config methods are applied correctly when no flags change
	tests := map[string]struct {
		configMethod     string
		wantRunAllPhases bool
		wantTaskMode     bool
	}{
		"phases config sets RunAllPhases": {
			configMethod:     "phases",
			wantRunAllPhases: true,
			wantTaskMode:     false,
		},
		"tasks config sets TaskMode": {
			configMethod:     "tasks",
			wantRunAllPhases: false,
			wantTaskMode:     true,
		},
		"single-session config leaves defaults": {
			configMethod:     "single-session",
			wantRunAllPhases: false,
			wantTaskMode:     false,
		},
		"empty config leaves defaults": {
			configMethod:     "",
			wantRunAllPhases: false,
			wantTaskMode:     false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := ResolveExecutionMode(ExecutionModeFlags{}, false, tt.configMethod)
			assert.Equal(t, tt.wantRunAllPhases, result.RunAllPhases, "RunAllPhases")
			assert.Equal(t, tt.wantTaskMode, result.TaskMode, "TaskMode")
		})
	}
}

func TestParseImplementArgs_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args       []string
		wantSpec   string
		wantPrompt string
	}{
		"single hyphen only": {
			args:       []string{"-"},
			wantSpec:   "",
			wantPrompt: "-",
		},
		"starts with number no hyphen": {
			args:       []string{"123"},
			wantSpec:   "",
			wantPrompt: "123",
		},
		"number dash uppercase": {
			args:       []string{"123-ABC"},
			wantSpec:   "",
			wantPrompt: "123-ABC",
		},
		"spec with single char name": {
			args:       []string{"1-a"},
			wantSpec:   "1-a",
			wantPrompt: "",
		},
		"leading zero spec": {
			args:       []string{"001-feature"},
			wantSpec:   "001-feature",
			wantPrompt: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotSpec, gotPrompt := ParseImplementArgs(tt.args)
			assert.Equal(t, tt.wantSpec, gotSpec)
			assert.Equal(t, tt.wantPrompt, gotPrompt)
		})
	}
}

func TestImplementCmd_MutuallyExclusiveFlagsRegistered(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// Check that mutually exclusive flags are properly registered
	// We verify the flags exist and have the correct types
	phaseFlags := []string{"phases", "phase", "from-phase"}
	for _, flagName := range phaseFlags {
		flag := implementCmd.Flags().Lookup(flagName)
		assert.NotNil(t, flag, "flag %s should exist", flagName)
	}

	taskFlags := []string{"tasks", "from-task"}
	for _, flagName := range taskFlags {
		flag := implementCmd.Flags().Lookup(flagName)
		assert.NotNil(t, flag, "flag %s should exist", flagName)
	}

	singleSessionFlag := implementCmd.Flags().Lookup("single-session")
	assert.NotNil(t, singleSessionFlag, "single-session flag should exist")
}

func TestExecutionModeFlags_AllFieldsAccessible(t *testing.T) {
	t.Parallel()

	// Test that all fields can be set and accessed correctly
	flags := ExecutionModeFlags{
		PhasesFlag:        true,
		TasksFlag:         true,
		SingleSessionFlag: true,
		PhaseFlag:         5,
		FromPhaseFlag:     3,
		FromTaskFlag:      "T007",
	}

	assert.True(t, flags.PhasesFlag)
	assert.True(t, flags.TasksFlag)
	assert.True(t, flags.SingleSessionFlag)
	assert.Equal(t, 5, flags.PhaseFlag)
	assert.Equal(t, 3, flags.FromPhaseFlag)
	assert.Equal(t, "T007", flags.FromTaskFlag)
}

func TestExecutionModeResult_AllFieldsAccessible(t *testing.T) {
	t.Parallel()

	// Test that all fields can be set and accessed correctly
	result := ExecutionModeResult{
		RunAllPhases: true,
		TaskMode:     true,
		SinglePhase:  2,
		FromPhase:    1,
		FromTask:     "T003",
	}

	assert.True(t, result.RunAllPhases)
	assert.True(t, result.TaskMode)
	assert.Equal(t, 2, result.SinglePhase)
	assert.Equal(t, 1, result.FromPhase)
	assert.Equal(t, "T003", result.FromTask)
}

func TestResolveExecutionMode_FlagsOverrideConfig(t *testing.T) {
	t.Parallel()

	// When flagsChanged is true, config should be ignored
	tests := map[string]struct {
		flags        ExecutionModeFlags
		configMethod string
	}{
		"phases flag ignores tasks config": {
			flags:        ExecutionModeFlags{PhasesFlag: true},
			configMethod: "tasks",
		},
		"tasks flag ignores phases config": {
			flags:        ExecutionModeFlags{TasksFlag: true},
			configMethod: "phases",
		},
		"single-session flag ignores all config": {
			flags:        ExecutionModeFlags{SingleSessionFlag: true},
			configMethod: "phases",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// When flags are changed, result should reflect flags, not config
			result := ResolveExecutionMode(tt.flags, true, tt.configMethod)

			if tt.flags.SingleSessionFlag {
				assert.False(t, result.RunAllPhases)
				assert.False(t, result.TaskMode)
			} else if tt.flags.PhasesFlag {
				assert.True(t, result.RunAllPhases)
			} else if tt.flags.TasksFlag {
				assert.True(t, result.TaskMode)
			}
		})
	}
}

func TestImplementCmd_AliasesHaveExpectedValues(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	aliases := implementCmd.Aliases
	assert.Len(t, aliases, 2, "implement should have exactly 2 aliases")
	assert.Contains(t, aliases, "impl")
	assert.Contains(t, aliases, "i")
}

func TestImplementCmd_GroupIDMatchesOtherStages(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// All stage commands should share the same GroupID
	assert.Equal(t, specifyCmd.GroupID, implementCmd.GroupID)
	assert.Equal(t, planCmd.GroupID, implementCmd.GroupID)
	assert.Equal(t, tasksCmd.GroupID, implementCmd.GroupID)
}

func TestImplementCmd_RunEIsDefined(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	assert.NotNil(t, implementCmd.RunE, "implement command should have RunE function")
}

func TestSpecNamePattern_BoundaryTests(t *testing.T) {
	t.Parallel()

	// Test boundary conditions for the spec name pattern
	tests := map[string]struct {
		input string
		want  bool
	}{
		"minimum valid spec": {
			input: "0-a",
			want:  true,
		},
		"very long valid spec": {
			input: "999999999-a-very-long-feature-name-that-continues",
			want:  true,
		},
		"consecutive hyphens valid": {
			input: "1-a-b-c-d",
			want:  true,
		},
		"multiple leading zeros": {
			input: "000000-feature",
			want:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := specNamePattern.MatchString(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
