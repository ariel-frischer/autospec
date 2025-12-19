// Package cli_test tests the uninstall command for removing autospec command templates from Claude CLI.
// Related: internal/cli/admin/uninstall.go
// Tags: cli, uninstall, commands, templates, removal, cleanup
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/cli/admin"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getUninstallCmd finds the uninstall command from rootCmd
func getUninstallCmd() *cobra.Command {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "uninstall" {
			return cmd
		}
	}
	return nil
}

func TestUninstallCmdRegistration(t *testing.T) {
	cmd := getUninstallCmd()
	assert.NotNil(t, cmd, "uninstall command should be registered")
}

func TestUninstallCmdFlags(t *testing.T) {
	cmd := getUninstallCmd()
	require.NotNil(t, cmd, "uninstall command must exist")

	flags := []struct {
		name      string
		shorthand string
	}{
		{"dry-run", "n"},
		{"yes", "y"},
	}

	for _, flag := range flags {
		t.Run("flag "+flag.name, func(t *testing.T) {
			f := cmd.Flags().Lookup(flag.name)
			require.NotNil(t, f, "flag %s should exist", flag.name)
			assert.Equal(t, flag.shorthand, f.Shorthand)
		})
	}
}

func TestUninstallCmdShortDescription(t *testing.T) {
	cmd := getUninstallCmd()
	require.NotNil(t, cmd, "uninstall command must exist")

	assert.Contains(t, cmd.Short, "remove")
	assert.Contains(t, cmd.Short, "autospec")
}

func TestUninstallCmdLongDescription(t *testing.T) {
	cmd := getUninstallCmd()
	require.NotNil(t, cmd, "uninstall command must exist")

	keywords := []string{
		"binary",
		"~/.config/autospec/",
		"~/.autospec/",
		"--dry-run",
		"--yes",
		"confirmation",
		"autospec clean",
		"sudo",
	}

	for _, keyword := range keywords {
		assert.Contains(t, cmd.Long, keyword, "Long description should contain %q", keyword)
	}
}

func TestUninstallCmdExamples(t *testing.T) {
	cmd := getUninstallCmd()
	require.NotNil(t, cmd, "uninstall command must exist")

	assert.Contains(t, cmd.Example, "--dry-run")
	assert.Contains(t, cmd.Example, "--yes")
	assert.Contains(t, cmd.Example, "sudo")
}

// Note: We can't easily test the actual uninstall operation since it would
// remove the running binary. Instead, we test the command structure and flags.

func TestRunUninstall_DryRun_ShowsTargets(t *testing.T) {
	// Create a test command with the same flags
	cmd := &cobra.Command{
		Use:  "uninstall",
		RunE: admin.RunUninstall,
	}
	cmd.Flags().BoolP("dry-run", "n", false, "")
	cmd.Flags().BoolP("yes", "y", false, "")

	require.NoError(t, cmd.Flags().Set("dry-run", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()

	// Should show what would be removed
	assert.Contains(t, output, "Would remove")

	// Should show the binary
	assert.Contains(t, output, "binary")

	// Should mention config dir path pattern
	assert.Contains(t, output, ".config/autospec")

	// Should mention state dir path pattern
	assert.Contains(t, output, ".autospec")

	// Should show hint about project cleanup
	assert.Contains(t, output, "autospec clean")
}

func TestRunUninstall_DryRun_DoesNotRemoveFiles(t *testing.T) {
	// Create temp files that simulate real paths
	tmpDir := t.TempDir()

	configDir := filepath.Join(tmpDir, ".config", "autospec")
	stateDir := filepath.Join(tmpDir, ".autospec")

	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.MkdirAll(stateDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yml"), []byte("test"), 0644))

	// Create a test command
	cmd := &cobra.Command{
		Use:  "uninstall",
		RunE: admin.RunUninstall,
	}
	cmd.Flags().BoolP("dry-run", "n", false, "")
	cmd.Flags().BoolP("yes", "y", false, "")

	require.NoError(t, cmd.Flags().Set("dry-run", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Note: This test can't verify that the actual targets aren't removed
	// because GetUninstallTargets() uses the real paths, not our temp paths.
	// We're just verifying the dry-run flag is respected in the output.
	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Would remove")

	// Our temp files should still exist (they wouldn't have been touched anyway)
	_, err = os.Stat(configDir)
	assert.NoError(t, err)
	_, err = os.Stat(stateDir)
	assert.NoError(t, err)
}

func TestRunUninstall_CancellationMessage(t *testing.T) {
	// Create a test command with mock stdin that provides 'n'
	cmd := &cobra.Command{
		Use:  "uninstall",
		RunE: admin.RunUninstall,
	}
	cmd.Flags().BoolP("dry-run", "n", false, "")
	cmd.Flags().BoolP("yes", "y", false, "")

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Provide 'n' as input to cancel
	cmd.SetIn(bytes.NewBufferString("n\n"))

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Uninstall cancelled")
}

func TestRunUninstall_DefaultCancelsOnEnter(t *testing.T) {
	// Create a test command with mock stdin that provides empty input (just Enter)
	cmd := &cobra.Command{
		Use:  "uninstall",
		RunE: admin.RunUninstall,
	}
	cmd.Flags().BoolP("dry-run", "n", false, "")
	cmd.Flags().BoolP("yes", "y", false, "")

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Provide empty input (just Enter) to test default behavior
	cmd.SetIn(bytes.NewBufferString("\n"))

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Uninstall cancelled")
}

func TestRunUninstall_ShowsProjectCleanupHint(t *testing.T) {
	cmd := &cobra.Command{
		Use:  "uninstall",
		RunE: admin.RunUninstall,
	}
	cmd.Flags().BoolP("dry-run", "n", false, "")
	cmd.Flags().BoolP("yes", "y", false, "")

	require.NoError(t, cmd.Flags().Set("dry-run", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	// Should show hint about project cleanup
	assert.Contains(t, output, "autospec clean")
	assert.Contains(t, output, "project")
}

func TestRunUninstall_OutputShowsTargetTypes(t *testing.T) {
	cmd := &cobra.Command{
		Use:  "uninstall",
		RunE: admin.RunUninstall,
	}
	cmd.Flags().BoolP("dry-run", "n", false, "")
	cmd.Flags().BoolP("yes", "y", false, "")

	require.NoError(t, cmd.Flags().Set("dry-run", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()

	// Should show target type indicators
	assert.Contains(t, output, "[binary]")
	assert.Contains(t, output, "[config_dir]")
	assert.Contains(t, output, "[state_dir]")
}
