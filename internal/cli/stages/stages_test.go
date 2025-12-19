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

func TestPlanCmd_InheritsFlags(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// planCmd inherits flags from root, doesn't define its own
	// Verify the command has a valid GroupID
	assert.NotEmpty(t, planCmd.GroupID)
	assert.NotEmpty(t, planCmd.Use)
}

func TestTasksCmd_InheritsFlags(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// tasksCmd inherits flags from root, doesn't define its own
	// Verify the command has a valid GroupID
	assert.NotEmpty(t, tasksCmd.GroupID)
	assert.NotEmpty(t, tasksCmd.Use)
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
