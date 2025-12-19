// Package cli_test tests the commands check subcommand for verifying Claude command template installation status.
// Related: internal/cli/admin/commands_check.go
// Tags: cli, commands, check, templates, installation, verification
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

// getCommandsCheckCmd finds the "commands check" command from rootCmd
func getCommandsCheckCmd() *cobra.Command {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "commands" {
			for _, sub := range cmd.Commands() {
				if sub.Use == "check" {
					return sub
				}
			}
		}
	}
	return nil
}

func TestCommandsCheckCmd_NotInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".claude", "commands")

	cmd := getCommandsCheckCmd()
	require.NotNil(t, cmd, "commands check subcommand must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Set flag directly
	cmd.Flags().Set("target", targetDir)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Not installed")
	assert.Contains(t, output, "autospec.specify")
}

func TestCommandsCheckCmd_AllCurrent(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".claude", "commands")

	// Install first
	_, err := commands.InstallTemplates(targetDir)
	require.NoError(t, err)

	cmd := getCommandsCheckCmd()
	require.NotNil(t, cmd, "commands check subcommand must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Set flag directly
	cmd.Flags().Set("target", targetDir)

	err = cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "All commands are up to date")
}
