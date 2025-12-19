// Package cli_test tests the commands install subcommand for installing Claude command templates.
// Related: internal/cli/admin/commands_install.go
// Tags: cli, commands, install, templates, setup
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getCommandsInstallCmd finds the "commands install" command from rootCmd
func getCommandsInstallCmd() *cobra.Command {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "commands" {
			for _, sub := range cmd.Commands() {
				if sub.Use == "install" {
					return sub
				}
			}
		}
	}
	return nil
}

func TestCommandsInstallCmd_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".claude", "commands")

	cmd := getCommandsInstallCmd()
	require.NotNil(t, cmd, "commands install subcommand must exist")

	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Set flag directly
	cmd.Flags().Set("target", targetDir)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	// Check output
	output := buf.String()
	assert.Contains(t, output, "Installing")
	assert.Contains(t, output, "autospec.specify")
	assert.Contains(t, output, "Done:")

	// Check files were created
	specifyPath := filepath.Join(targetDir, "autospec.specify.md")
	_, err = os.Stat(specifyPath)
	assert.NoError(t, err, "autospec.specify.md should exist")
}

func TestCommandsInstallCmd_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	// Use a nested directory that doesn't exist
	targetDir := filepath.Join(tmpDir, "deep", "nested", ".claude", "commands")

	cmd := getCommandsInstallCmd()
	require.NotNil(t, cmd, "commands install subcommand must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Set flag directly
	cmd.Flags().Set("target", targetDir)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	// Directory should be created
	_, err = os.Stat(targetDir)
	assert.NoError(t, err, "target directory should be created")
}
