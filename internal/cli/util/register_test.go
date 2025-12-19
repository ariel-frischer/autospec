// Package util tests utility CLI commands for autospec.
// Related: internal/cli/util/register.go
// Tags: util, cli, commands, registration

package util

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	// Cannot run in parallel - Register modifies global command state

	rootCmd := &cobra.Command{
		Use:   "test",
		Short: "Test root command",
	}

	// Register should not panic
	require.NotPanics(t, func() {
		Register(rootCmd)
	})

	// Verify commands are added
	commands := rootCmd.Commands()
	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name()] = true
	}

	// Should have status, history, version, sauce, clean commands
	assert.True(t, commandNames["status"], "Should have 'status' command")
	assert.True(t, commandNames["history"], "Should have 'history' command")
	assert.True(t, commandNames["version"], "Should have 'version' command")
	assert.True(t, commandNames["sauce"], "Should have 'sauce' command")
	assert.True(t, commandNames["clean"], "Should have 'clean' command")
}

func TestRegister_CommandAnnotations(t *testing.T) {
	// Cannot run in parallel - Register modifies global command state

	tests := map[string]struct {
		cmdName string
		wantCmd bool
	}{
		"status command exists": {
			cmdName: "status",
			wantCmd: true,
		},
		"history command exists": {
			cmdName: "history",
			wantCmd: true,
		},
		"version command exists": {
			cmdName: "version",
			wantCmd: true,
		},
		"clean command exists": {
			cmdName: "clean",
			wantCmd: true,
		},
		"sauce command exists": {
			cmdName: "sauce",
			wantCmd: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - Register modifies global state

			rootCmd := &cobra.Command{
				Use: "test",
			}
			Register(rootCmd)

			found := false
			for _, cmd := range rootCmd.Commands() {
				if cmd.Name() == tt.cmdName {
					found = true
					break
				}
			}
			assert.Equal(t, tt.wantCmd, found)
		})
	}
}

func TestRegister_CommandCount(t *testing.T) {
	// Cannot run in parallel - Register modifies global command state

	rootCmd := &cobra.Command{
		Use: "test",
	}

	Register(rootCmd)

	// Should register exactly 5 commands
	assert.Equal(t, 5, len(rootCmd.Commands()))
}

func TestStatusCmd_Structure(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "status [spec-name]", statusCmd.Use)
	assert.NotEmpty(t, statusCmd.Short)
	assert.Contains(t, statusCmd.Aliases, "st", "Should have 'st' alias")
}

func TestHistoryCmd_Structure(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "history", historyCmd.Use)
	assert.NotEmpty(t, historyCmd.Short)
	assert.NotEmpty(t, historyCmd.Long)
}

func TestVersionCmd_Structure(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "version", versionCmd.Use)
	assert.NotEmpty(t, versionCmd.Short)
	assert.NotEmpty(t, versionCmd.Long)
	assert.Contains(t, versionCmd.Aliases, "v", "Should have 'v' alias")
}

func TestCleanCmd_Structure(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "clean", cleanCmd.Use)
	assert.NotEmpty(t, cleanCmd.Short)
}

func TestUtilCommands_HaveRunE(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cmd     *cobra.Command
		hasRunE bool
		hasRun  bool
	}{
		"status has RunE": {
			cmd:     statusCmd,
			hasRunE: true,
		},
		"history has RunE": {
			cmd:     historyCmd,
			hasRunE: true,
		},
		"version has Run": {
			cmd:    versionCmd,
			hasRun: true,
		},
		"clean has RunE": {
			cmd:     cleanCmd,
			hasRunE: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tt.hasRunE {
				assert.NotNil(t, tt.cmd.RunE, "Command should have RunE")
			}
			if tt.hasRun {
				assert.NotNil(t, tt.cmd.Run, "Command should have Run")
			}
		})
	}
}

func TestRegister_DoesNotPanic(t *testing.T) {
	// Cannot run in parallel - Register modifies global command state

	rootCmd := &cobra.Command{Use: "test"}

	require.NotPanics(t, func() {
		Register(rootCmd)
	})
}

func TestUtilCommands_ExecuteWithoutSubcommands(t *testing.T) {
	t.Parallel()

	// Create an isolated command to test
	cmd := &cobra.Command{
		Use: "status",
	}
	cmd.SetArgs([]string{})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Should succeed (shows usage)
	err := cmd.Execute()
	assert.NoError(t, err)
}
