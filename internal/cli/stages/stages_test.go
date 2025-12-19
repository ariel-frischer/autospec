// Package stages tests CLI workflow stage commands for autospec.
// Related: internal/cli/stages/*.go
// Tags: stages, cli, commands, workflow

package stages

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpecifyCmd_Flags(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// Only test flags that are defined directly on specifyCmd (not inherited from root)
	tests := map[string]struct {
		flagName string
		wantFlag bool
	}{
		"max-retries flag exists": {
			flagName: "max-retries",
			wantFlag: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - accesses global command state

			flag := specifyCmd.Flags().Lookup(tt.flagName)
			if tt.wantFlag {
				assert.NotNil(t, flag, "Flag %s should exist", tt.flagName)
			} else {
				assert.Nil(t, flag)
			}
		})
	}
}

func TestPlanCmd_Flags(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// Verify the command has a valid GroupID
	assert.NotEmpty(t, planCmd.GroupID)
	assert.NotEmpty(t, planCmd.Use)

	// Test max-retries flag
	flag := planCmd.Flags().Lookup("max-retries")
	assert.NotNil(t, flag, "max-retries flag should exist")
	assert.Equal(t, "r", flag.Shorthand, "max-retries should have shorthand 'r'")
	assert.Equal(t, "0", flag.DefValue, "max-retries should default to 0")
}

func TestTasksCmd_Flags(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// Verify the command has a valid GroupID
	assert.NotEmpty(t, tasksCmd.GroupID)
	assert.NotEmpty(t, tasksCmd.Use)

	// Test max-retries flag
	flag := tasksCmd.Flags().Lookup("max-retries")
	assert.NotNil(t, flag, "max-retries flag should exist")
	assert.Equal(t, "r", flag.Shorthand, "max-retries should have shorthand 'r'")
	assert.Equal(t, "0", flag.DefValue, "max-retries should default to 0")
}

func TestImplementCmd_Flags(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// Only test flags that are defined directly on implementCmd (not inherited)
	tests := map[string]struct {
		flagName string
		wantFlag bool
	}{
		"max-retries flag exists": {
			flagName: "max-retries",
			wantFlag: true,
		},
		"phases flag exists": {
			flagName: "phases",
			wantFlag: true,
		},
		"tasks flag exists": {
			flagName: "tasks",
			wantFlag: true,
		},
		"single-session flag exists": {
			flagName: "single-session",
			wantFlag: true,
		},
		"resume flag exists": {
			flagName: "resume",
			wantFlag: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - accesses global command state

			flag := implementCmd.Flags().Lookup(tt.flagName)
			if tt.wantFlag {
				assert.NotNil(t, flag, "Flag %s should exist", tt.flagName)
			} else {
				assert.Nil(t, flag)
			}
		})
	}
}

func TestStageCommands_DescriptionsAreInformative(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	tests := map[string]struct {
		cmdShort     string
		cmdLong      string
		minShortLen  int
		minLongLen   int
		wantContains []string
	}{
		"specify has informative description": {
			cmdShort:     specifyCmd.Short,
			cmdLong:      specifyCmd.Long,
			minShortLen:  20,
			minLongLen:   100,
			wantContains: []string{"specification", "spec"},
		},
		"plan has informative description": {
			cmdShort:     planCmd.Short,
			cmdLong:      planCmd.Long,
			minShortLen:  20,
			minLongLen:   100,
			wantContains: []string{"plan"},
		},
		"tasks has informative description": {
			cmdShort:     tasksCmd.Short,
			cmdLong:      tasksCmd.Long,
			minShortLen:  20,
			minLongLen:   100,
			wantContains: []string{"task"},
		},
		"implement has informative description": {
			cmdShort:     implementCmd.Short,
			cmdLong:      implementCmd.Long,
			minShortLen:  20,
			minLongLen:   100,
			wantContains: []string{"implement"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - accesses global command state

			assert.GreaterOrEqual(t, len(tt.cmdShort), tt.minShortLen,
				"Short description should be informative")
			assert.GreaterOrEqual(t, len(tt.cmdLong), tt.minLongLen,
				"Long description should be detailed")
		})
	}
}

func TestStageCommands_HaveExamples(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	tests := map[string]struct {
		example string
	}{
		"specify has examples": {
			example: specifyCmd.Example,
		},
		"plan has examples": {
			example: planCmd.Example,
		},
		"tasks has examples": {
			example: tasksCmd.Example,
		},
		"implement has examples": {
			example: implementCmd.Example,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - accesses global command state

			assert.NotEmpty(t, tt.example, "Command should have examples")
			assert.Contains(t, tt.example, "autospec", "Example should show autospec command")
		})
	}
}

func TestSpecifyCmd_LongDescriptionContent(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	long := specifyCmd.Long
	assert.Contains(t, long, "spec.yaml", "Long description should mention spec.yaml")
	assert.Contains(t, long, "specification", "Long description should explain purpose")
}

func TestPlanCmd_LongDescriptionContent(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	long := planCmd.Long
	assert.Contains(t, long, "plan.yaml", "Long description should mention plan.yaml")
	assert.Contains(t, long, "planning", "Long description should explain purpose")
}

func TestTasksCmd_LongDescriptionContent(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	long := tasksCmd.Long
	assert.Contains(t, long, "tasks.yaml", "Long description should mention tasks.yaml")
	assert.Contains(t, long, "task", "Long description should explain purpose")
}

func TestImplementCmd_LongDescriptionContent(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	long := implementCmd.Long
	assert.Contains(t, long, "tasks.yaml", "Long description should mention tasks.yaml")
	assert.Contains(t, long, "--phases", "Long description should document phases flag")
	assert.Contains(t, long, "--tasks", "Long description should document tasks flag")
	assert.Contains(t, long, "--single-session", "Long description should document single-session flag")
	assert.Contains(t, long, "--resume", "Long description should document resume flag")
	assert.Contains(t, long, "implement_method", "Long description should document config option")
}

func TestSpecifyCmd_ShortDescriptionAliases(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	short := specifyCmd.Short
	assert.Contains(t, short, "spec", "Short description should mention alias")
	assert.Contains(t, short, "s", "Short description should mention single-letter alias")
}

func TestPlanCmd_ShortDescriptionAliases(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	short := planCmd.Short
	assert.Contains(t, short, "p", "Short description should mention alias")
}

func TestTasksCmd_ShortDescriptionAliases(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	short := tasksCmd.Short
	assert.Contains(t, short, "t", "Short description should mention alias")
}

func TestImplementCmd_ShortDescriptionAliases(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	short := implementCmd.Short
	assert.Contains(t, short, "impl", "Short description should mention alias")
	assert.Contains(t, short, "i", "Short description should mention single-letter alias")
}

func TestSpecifyCmd_HasMaxRetriesFlag(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	flag := specifyCmd.Flags().Lookup("max-retries")
	assert.NotNil(t, flag, "max-retries flag should exist")
	assert.Equal(t, "r", flag.Shorthand, "max-retries should have shorthand 'r'")
	assert.Equal(t, "0", flag.DefValue, "max-retries should default to 0")
}

func TestPlanCmd_HasRunE(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// Verify the command can be executed
	assert.NotNil(t, planCmd.RunE, "Plan command should have RunE")
}

func TestTasksCmd_HasRunE(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// Verify the command can be executed
	assert.NotNil(t, tasksCmd.RunE, "Tasks command should have RunE")
}

func TestSpecifyCmd_ArgsValidationCases(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	tests := map[string]struct {
		args    []string
		wantErr bool
	}{
		"empty args returns error": {
			args:    []string{},
			wantErr: true,
		},
		"single word description valid": {
			args:    []string{"feature"},
			wantErr: false,
		},
		"multi word description valid": {
			args:    []string{"Add", "user", "authentication"},
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run in parallel - accesses global command state

			err := specifyCmd.Args(specifyCmd, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestImplementCmd_ExampleContent(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	example := implementCmd.Example
	assert.Contains(t, example, "--resume", "Example should show resume flag usage")
	assert.Contains(t, example, "--phases", "Example should show phases flag usage")
	assert.Contains(t, example, "--phase", "Example should show phase flag usage")
	assert.Contains(t, example, "--from-phase", "Example should show from-phase flag usage")
	assert.Contains(t, example, "--tasks", "Example should show tasks flag usage")
	assert.Contains(t, example, "--from-task", "Example should show from-task flag usage")
	assert.Contains(t, example, "--single-session", "Example should show single-session flag usage")
}

func TestPlanCmd_ExampleContent(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	example := planCmd.Example
	assert.Contains(t, example, "autospec plan", "Example should show basic usage")
}

func TestTasksCmd_ExampleContent(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	example := tasksCmd.Example
	assert.Contains(t, example, "autospec tasks", "Example should show basic usage")
}

func TestSpecifyCmd_ExampleContent(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	example := specifyCmd.Example
	assert.Contains(t, example, "autospec specify", "Example should show basic usage")
	assert.Contains(t, example, "authentication", "Example should show realistic usage")
}

func TestSpecifyCmd_ArgsFunction(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// Test with empty args - should fail
	err := specifyCmd.Args(specifyCmd, []string{})
	assert.Error(t, err, "Empty args should return error for specify command")

	// Test with valid args - should pass
	err = specifyCmd.Args(specifyCmd, []string{"Add feature"})
	assert.NoError(t, err, "Non-empty args should be valid for specify command")
}

func TestStageCommands_GroupIDsMatch(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// All stage commands should be in the same group
	expectedGroup := specifyCmd.GroupID
	assert.Equal(t, expectedGroup, planCmd.GroupID, "Plan should be in same group as specify")
	assert.Equal(t, expectedGroup, tasksCmd.GroupID, "Tasks should be in same group as specify")
	assert.Equal(t, expectedGroup, implementCmd.GroupID, "Implement should be in same group as specify")
}

func TestImplementCmd_ExecutionModeFlags(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// Verify that implement command has mutually exclusive execution mode flags
	executionModeFlags := []string{"phases", "tasks", "single-session"}

	for _, flagName := range executionModeFlags {
		t.Run(flagName, func(t *testing.T) {
			// Cannot run subtests in parallel - accesses global command state

			flag := implementCmd.Flags().Lookup(flagName)
			assert.NotNil(t, flag, "Flag %s should exist", flagName)
		})
	}
}

func TestImplementCmd_PhaseFlags(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	tests := map[string]struct {
		flagName string
		wantFlag bool
	}{
		"phase flag exists": {
			flagName: "phase",
			wantFlag: true,
		},
		"from-phase flag exists": {
			flagName: "from-phase",
			wantFlag: true,
		},
		"from-task flag exists": {
			flagName: "from-task",
			wantFlag: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - accesses global command state

			flag := implementCmd.Flags().Lookup(tt.flagName)
			if tt.wantFlag {
				assert.NotNil(t, flag, "Flag %s should exist", tt.flagName)
			} else {
				assert.Nil(t, flag)
			}
		})
	}
}
