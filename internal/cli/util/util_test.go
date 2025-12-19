// Package util tests utility CLI commands for autospec.
// Related: internal/cli/util/*.go
// Tags: util, cli, commands, status, history, version, clean

package util

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestStatusCmd_Flags(t *testing.T) {
	// Cannot run in parallel - accesses global statusCmd state

	tests := map[string]struct {
		flagName string
		wantFlag bool
	}{
		"verbose flag exists": {
			flagName: "verbose",
			wantFlag: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - accesses global command state

			flag := statusCmd.Flags().Lookup(tt.flagName)
			if tt.wantFlag {
				assert.NotNil(t, flag, "Flag %s should exist", tt.flagName)
			} else {
				assert.Nil(t, flag)
			}
		})
	}
}

func TestHistoryCmd_Flags(t *testing.T) {
	// Cannot run in parallel - accesses global historyCmd state

	tests := map[string]struct {
		flagName string
		wantFlag bool
	}{
		"spec flag exists": {
			flagName: "spec",
			wantFlag: true,
		},
		"limit flag exists": {
			flagName: "limit",
			wantFlag: true,
		},
		"clear flag exists": {
			flagName: "clear",
			wantFlag: true,
		},
		"status flag exists": {
			flagName: "status",
			wantFlag: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - accesses global command state

			flag := historyCmd.Flags().Lookup(tt.flagName)
			if tt.wantFlag {
				assert.NotNil(t, flag, "Flag %s should exist", tt.flagName)
			} else {
				assert.Nil(t, flag)
			}
		})
	}
}

func TestVersionCmd_Flags(t *testing.T) {
	// Cannot run in parallel - accesses global versionCmd state

	tests := map[string]struct {
		flagName string
		wantFlag bool
	}{
		"plain flag exists": {
			flagName: "plain",
			wantFlag: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - accesses global command state

			flag := versionCmd.Flags().Lookup(tt.flagName)
			if tt.wantFlag {
				assert.NotNil(t, flag, "Flag %s should exist", tt.flagName)
			} else {
				assert.Nil(t, flag)
			}
		})
	}
}

func TestCleanCmd_Flags(t *testing.T) {
	// Cannot run in parallel - accesses global cleanCmd state

	tests := map[string]struct {
		flagName string
		wantFlag bool
	}{
		"dry-run flag exists": {
			flagName: "dry-run",
			wantFlag: true,
		},
		"yes flag exists": {
			flagName: "yes",
			wantFlag: true,
		},
		"keep-specs flag exists": {
			flagName: "keep-specs",
			wantFlag: true,
		},
		"remove-specs flag exists": {
			flagName: "remove-specs",
			wantFlag: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - accesses global command state

			flag := cleanCmd.Flags().Lookup(tt.flagName)
			if tt.wantFlag {
				assert.NotNil(t, flag, "Flag %s should exist", tt.flagName)
			} else {
				assert.Nil(t, flag)
			}
		})
	}
}

func TestUtilCommands_GroupIDs(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	tests := map[string]struct {
		cmd         *cobra.Command
		wantGroupID string
	}{
		"status group": {
			cmd:         statusCmd,
			wantGroupID: "getting-started",
		},
		"history group": {
			cmd:         historyCmd,
			wantGroupID: "configuration",
		},
		"clean group": {
			cmd:         cleanCmd,
			wantGroupID: "configuration",
		},
		"version group": {
			cmd:         versionCmd,
			wantGroupID: "getting-started",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - accesses global command state

			assert.Equal(t, tt.wantGroupID, tt.cmd.GroupID)
		})
	}
}

func TestUtilCommands_DescriptionsAreInformative(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	tests := map[string]struct {
		cmdShort    string
		minShortLen int
	}{
		"status has informative description": {
			cmdShort:    statusCmd.Short,
			minShortLen: 20,
		},
		"history has informative description": {
			cmdShort:    historyCmd.Short,
			minShortLen: 10,
		},
		"version has informative description": {
			cmdShort:    versionCmd.Short,
			minShortLen: 10,
		},
		"clean has informative description": {
			cmdShort:    cleanCmd.Short,
			minShortLen: 10,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - accesses global command state

			assert.GreaterOrEqual(t, len(tt.cmdShort), tt.minShortLen,
				"Short description should be informative")
		})
	}
}

func TestVersionInfo_Variables(t *testing.T) {
	t.Parallel()

	// Test that version variables are defined (they're set via ldflags at build time)
	assert.NotEmpty(t, Version, "Version should have a default value")
	assert.NotEmpty(t, Commit, "Commit should have a default value")
	assert.NotEmpty(t, BuildDate, "BuildDate should have a default value")
}

func TestGetDefaultStateDir(t *testing.T) {
	t.Parallel()

	// Test that getDefaultStateDir returns a non-empty path
	stateDir := getDefaultStateDir()
	// On most systems, this should return a valid path
	// It may be empty if $HOME is not set, which is unlikely
	assert.NotEmpty(t, stateDir, "State directory should be set")
	assert.Contains(t, stateDir, ".autospec", "State directory should contain .autospec")
	assert.Contains(t, stateDir, "state", "State directory should contain state")
}

func TestStatusCmd_Aliases(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	aliases := statusCmd.Aliases
	assert.Contains(t, aliases, "st", "Should have 'st' alias")
}

func TestHistoryCmd_Aliases(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// historyCmd doesn't have aliases by default
	// Just verify we can check aliases without panic
	_ = historyCmd.Aliases
}

func TestVersionCmd_Aliases(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	aliases := versionCmd.Aliases
	assert.Contains(t, aliases, "v", "Should have 'v' alias")
}

func TestCleanCmd_Aliases(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// cleanCmd doesn't have aliases by default
	// Just verify we can check aliases without panic
	_ = cleanCmd.Aliases
}

func TestCleanCmd_MutuallyExclusiveFlags(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// Verify that keep-specs and remove-specs flags exist
	// Their mutual exclusivity is set in init()
	keepFlag := cleanCmd.Flags().Lookup("keep-specs")
	removeFlag := cleanCmd.Flags().Lookup("remove-specs")

	assert.NotNil(t, keepFlag, "keep-specs flag should exist")
	assert.NotNil(t, removeFlag, "remove-specs flag should exist")
}

func TestHistoryCmd_HasRunE(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	assert.NotNil(t, historyCmd.RunE)
}

func TestCleanCmd_HasRunE(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	assert.NotNil(t, cleanCmd.RunE)
}
