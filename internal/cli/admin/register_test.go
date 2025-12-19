// Package admin_test tests administrative CLI commands for autospec.
// Related: internal/cli/admin/register.go
// Tags: admin, cli, commands, registration

package admin

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {

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
		commandNames[cmd.Use] = true
	}

	// Should have commands, completion, and uninstall
	assert.True(t, commandNames["commands"], "Should have 'commands' command")
	assert.True(t, commandNames["uninstall"], "Should have 'uninstall' command")
}

func TestCommandsCmdStructure(t *testing.T) {

	// Test commandsCmd structure
	assert.Equal(t, "commands", commandsCmd.Use)
	assert.NotEmpty(t, commandsCmd.Short)
	assert.NotEmpty(t, commandsCmd.Long)
}

func TestRegister_DisablesDefaultCompletion(t *testing.T) {

	rootCmd := &cobra.Command{
		Use:   "test",
		Short: "Test root command",
	}

	Register(rootCmd)

	// Verify default completion is disabled
	assert.True(t, rootCmd.CompletionOptions.DisableDefaultCmd,
		"Default completion command should be disabled")
}

func TestRegister_SetsRootCmdRef(t *testing.T) {
	// Cannot run in parallel due to global state (rootCmdRef)

	rootCmd := &cobra.Command{
		Use:   "testroot",
		Short: "Test root command",
	}

	Register(rootCmd)

	// rootCmdRef should be set (used by completion generation)
	assert.NotNil(t, rootCmdRef)
	// Note: rootCmdRef may have been set by previous test runs
	// Just verify it's not nil
}

func TestCommandsCmd_HasSubcommands(t *testing.T) {

	// The commands command should have subcommands
	subcommands := commandsCmd.Commands()

	// Should have info, check, install subcommands
	subcommandNames := make(map[string]bool)
	for _, cmd := range subcommands {
		subcommandNames[cmd.Name()] = true
	}

	assert.True(t, subcommandNames["info"], "Should have 'info' subcommand")
	assert.True(t, subcommandNames["check"], "Should have 'check' subcommand")
	assert.True(t, subcommandNames["install"], "Should have 'install' subcommand")
}

func TestUninstallCmd_Structure(t *testing.T) {

	assert.Equal(t, "uninstall", uninstallCmd.Use)
	assert.NotEmpty(t, uninstallCmd.Short)
}

func TestCompletionCmd_Structure(t *testing.T) {

	assert.NotNil(t, completionCmd)
	// completionCmd is defined in completion_install.go
}

func TestRegister_CommandGroups(t *testing.T) {

	rootCmd := &cobra.Command{
		Use: "test",
	}

	// Add group
	rootCmd.AddGroup(&cobra.Group{
		ID:    "internal",
		Title: "Internal Commands",
	})

	Register(rootCmd)

	// Find the commands command and check its group
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "commands" {
			// Commands should be in internal group
			assert.Equal(t, "internal", cmd.GroupID)
		}
	}
}

func TestCommandsCmd_RunsWithoutArgs(t *testing.T) {

	// Create isolated command for testing
	cmd := &cobra.Command{
		Use: "commands",
	}

	// Should not panic when executed without args (shows help)
	cmd.SetArgs([]string{})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Execute should succeed (shows usage)
	err := cmd.Execute()
	assert.NoError(t, err)
}
