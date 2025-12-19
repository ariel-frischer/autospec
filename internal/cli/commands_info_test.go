// Package cli_test tests the commands info subcommand for displaying Claude command template information.
// Related: internal/cli/admin/commands_info.go
// Tags: cli, commands, info, templates, metadata, list
package cli

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/commands"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getCommandsInfoCmd finds the "commands info" command from rootCmd
func getCommandsInfoCmd() *cobra.Command {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "commands" {
			for _, sub := range cmd.Commands() {
				if sub.Use == "info [command-name]" {
					return sub
				}
			}
		}
	}
	return nil
}

func TestCommandsInfoCmd_ListAll(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".claude", "commands")

	cmd := getCommandsInfoCmd()
	require.NotNil(t, cmd, "commands info subcommand must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Set flag directly
	cmd.Flags().Set("target", targetDir)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Autospec Command Templates")
	assert.Contains(t, output, "autospec.specify")
	assert.Contains(t, output, "not installed")
}

func TestCommandsInfoCmd_ListInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".claude", "commands")

	// Install first
	_, err := commands.InstallTemplates(targetDir)
	require.NoError(t, err)

	cmd := getCommandsInfoCmd()
	require.NotNil(t, cmd, "commands info subcommand must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Set flag directly
	cmd.Flags().Set("target", targetDir)

	err = cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "(current)")
}

func TestCommandsInfoCmd_SpecificCommand(t *testing.T) {
	cmd := getCommandsInfoCmd()
	require.NotNil(t, cmd, "commands info subcommand must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{"autospec.specify"})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Command: /autospec.specify")
	assert.Contains(t, output, "Description:")
	assert.Contains(t, output, "Version:")
}

func TestCommandsInfoCmd_NotFound(t *testing.T) {
	cmd := getCommandsInfoCmd()
	require.NotNil(t, cmd, "commands info subcommand must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.RunE(cmd, []string{"nonexistent"})
	// The command may or may not return an error depending on implementation
	// Either we get an error, or we check that the output indicates not found
	if err != nil {
		assert.Contains(t, err.Error(), "not found")
	} else {
		// If no error, output should indicate not found
		output := buf.String()
		assert.Contains(t, output, "not found")
	}
}
