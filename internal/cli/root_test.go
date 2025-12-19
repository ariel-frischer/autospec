// Package cli tests root command and global flags for autospec.
// Related: internal/cli/root.go
// Tags: cli, root, commands, global-flags

package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmd_Structure(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "autospec", rootCmd.Use)
	assert.NotEmpty(t, rootCmd.Short)
	assert.NotEmpty(t, rootCmd.Long)
	assert.NotEmpty(t, rootCmd.Example)
}

func TestRootCmd_PersistentFlags(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		flagName string
		wantFlag bool
	}{
		"config flag exists": {
			flagName: "config",
			wantFlag: true,
		},
		"specs-dir flag exists": {
			flagName: "specs-dir",
			wantFlag: true,
		},
		"skip-preflight flag exists": {
			flagName: "skip-preflight",
			wantFlag: true,
		},
		"debug flag exists": {
			flagName: "debug",
			wantFlag: true,
		},
		"verbose flag exists": {
			flagName: "verbose",
			wantFlag: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			flag := rootCmd.PersistentFlags().Lookup(tt.flagName)
			if tt.wantFlag {
				assert.NotNil(t, flag, "Flag %s should exist", tt.flagName)
			} else {
				assert.Nil(t, flag)
			}
		})
	}
}

func TestRootCmd_HasSubcommands(t *testing.T) {
	t.Parallel()

	commands := rootCmd.Commands()
	assert.Greater(t, len(commands), 0, "Root command should have subcommands")
}

func TestRootCmd_SubcommandGroups(t *testing.T) {
	t.Parallel()

	// Test that command groups are defined
	groups := rootCmd.Groups()
	assert.Greater(t, len(groups), 0, "Root command should have groups defined")

	// Verify expected groups exist
	groupIDs := make(map[string]bool)
	for _, g := range groups {
		groupIDs[g.ID] = true
	}

	assert.True(t, groupIDs[GroupGettingStarted], "Should have getting-started group")
	assert.True(t, groupIDs[GroupWorkflows], "Should have workflows group")
	assert.True(t, groupIDs[GroupCoreStages], "Should have core-stages group")
	assert.True(t, groupIDs[GroupOptionalStages], "Should have optional-stages group")
	assert.True(t, groupIDs[GroupConfiguration], "Should have configuration group")
	assert.True(t, groupIDs[GroupInternal], "Should have internal group")
}

func TestRootCmd_CanShowHelp(t *testing.T) {
	t.Parallel()

	// Create a fresh command to avoid modifying global state
	cmd := &cobra.Command{
		Use:   "autospec",
		Short: "Test command",
	}
	cmd.SetArgs([]string{"--help"})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Execute with help flag
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Test command")
}

func TestGroupConstants(t *testing.T) {
	t.Parallel()

	// Verify group constants are set correctly
	tests := map[string]struct {
		constant  string
		wantValue string
	}{
		"getting-started": {
			constant:  GroupGettingStarted,
			wantValue: "getting-started",
		},
		"workflows": {
			constant:  GroupWorkflows,
			wantValue: "workflows",
		},
		"core-stages": {
			constant:  GroupCoreStages,
			wantValue: "core-stages",
		},
		"optional-stages": {
			constant:  GroupOptionalStages,
			wantValue: "optional-stages",
		},
		"configuration": {
			constant:  GroupConfiguration,
			wantValue: "configuration",
		},
		"internal": {
			constant:  GroupInternal,
			wantValue: "internal",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.wantValue, tt.constant)
		})
	}
}

func TestExecute(t *testing.T) {
	// Cannot run in parallel due to global rootCmd state

	// The Execute function should not panic
	require.NotPanics(t, func() {
		// Reset args to avoid accidental execution
		origArgs := rootCmd.Args
		rootCmd.SetArgs([]string{"--help"})
		defer func() {
			rootCmd.SetArgs(nil)
			_ = origArgs
		}()

		// Capture output
		var buf bytes.Buffer
		rootCmd.SetOut(&buf)
		rootCmd.SetErr(&buf)

		// Execute should complete without panic
		_ = Execute()
	})
}

func TestRootCmd_Description(t *testing.T) {
	t.Parallel()

	// Verify description contains key information
	assert.Contains(t, rootCmd.Long, "autospec")
	assert.Contains(t, rootCmd.Long, "workflow")
	assert.Contains(t, rootCmd.Long, "github.com")
}

func TestRootCmd_Example(t *testing.T) {
	t.Parallel()

	// Verify example contains typical commands
	assert.Contains(t, rootCmd.Example, "autospec status")
	assert.Contains(t, rootCmd.Example, "autospec all")
	assert.Contains(t, rootCmd.Example, "autospec prep")
	assert.Contains(t, rootCmd.Example, "autospec specify")
	assert.Contains(t, rootCmd.Example, "autospec plan")
	assert.Contains(t, rootCmd.Example, "autospec tasks")
	assert.Contains(t, rootCmd.Example, "autospec implement")
}

func TestRootCmd_FlagShortcuts(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		flagName     string
		wantShortcut string
	}{
		"config has shortcut c": {
			flagName:     "config",
			wantShortcut: "c",
		},
		"debug has shortcut d": {
			flagName:     "debug",
			wantShortcut: "d",
		},
		"verbose has shortcut v": {
			flagName:     "verbose",
			wantShortcut: "v",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			flag := rootCmd.PersistentFlags().Lookup(tt.flagName)
			require.NotNil(t, flag)
			assert.Equal(t, tt.wantShortcut, flag.Shorthand)
		})
	}
}

func TestRootCmd_SubcommandCategories(t *testing.T) {
	t.Parallel()

	// Verify commands are registered from subpackages
	commands := rootCmd.Commands()
	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name()] = true
	}

	// Stage commands (from stages package)
	assert.True(t, commandNames["specify"], "Should have specify command")
	assert.True(t, commandNames["plan"], "Should have plan command")
	assert.True(t, commandNames["tasks"], "Should have tasks command")
	assert.True(t, commandNames["implement"], "Should have implement command")

	// Config commands (from config package)
	assert.True(t, commandNames["init"], "Should have init command")
	assert.True(t, commandNames["config"], "Should have config command")
	assert.True(t, commandNames["doctor"], "Should have doctor command")

	// Util commands (from util package)
	assert.True(t, commandNames["status"], "Should have status command")
	assert.True(t, commandNames["history"], "Should have history command")
	assert.True(t, commandNames["version"], "Should have version command")
	assert.True(t, commandNames["clean"], "Should have clean command")

	// Admin commands (from admin package)
	assert.True(t, commandNames["commands"], "Should have commands command")
	assert.True(t, commandNames["uninstall"], "Should have uninstall command")
}
