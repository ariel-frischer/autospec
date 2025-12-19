// Package cli_test tests the clean command for removing autospec files and directories with dry-run support.
// Related: internal/cli/util/clean.go
// Tags: cli, clean, removal, cleanup, dry-run, specs
package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getCleanCmd finds the clean command from rootCmd
func getCleanCmd() *cobra.Command {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "clean" {
			return cmd
		}
	}
	return nil
}

func TestCleanCmdRegistration(t *testing.T) {
	cmd := getCleanCmd()
	assert.NotNil(t, cmd, "clean command should be registered")
}

func TestCleanCmdFlags(t *testing.T) {
	cleanCmd := getCleanCmd()
	require.NotNil(t, cleanCmd, "clean command must exist")

	flags := []struct {
		name      string
		shorthand string
	}{
		{"dry-run", "n"},
		{"yes", "y"},
		{"keep-specs", "k"},
		{"remove-specs", "r"},
	}

	for _, flag := range flags {
		t.Run("flag "+flag.name, func(t *testing.T) {
			f := cleanCmd.Flags().Lookup(flag.name)
			require.NotNil(t, f, "flag %s should exist", flag.name)
			assert.Equal(t, flag.shorthand, f.Shorthand)
		})
	}
}

func TestCleanCmdShortDescription(t *testing.T) {
	cleanCmd := getCleanCmd()
	require.NotNil(t, cleanCmd, "clean command must exist")
	assert.Contains(t, cleanCmd.Short, "Remove")
	assert.Contains(t, cleanCmd.Short, "autospec")
}

func TestCleanCmdLongDescription(t *testing.T) {
	cleanCmd := getCleanCmd()
	require.NotNil(t, cleanCmd, "clean command must exist")

	keywords := []string{
		".autospec/",
		".claude/commands/autospec",
		"specs/",
		"--dry-run",
		"--yes",
		"--keep-specs",
		"--remove-specs",
		"confirmation",
	}

	for _, keyword := range keywords {
		assert.Contains(t, cleanCmd.Long, keyword)
	}
}

func TestCleanCmdExamples(t *testing.T) {
	cleanCmd := getCleanCmd()
	require.NotNil(t, cleanCmd, "clean command must exist")
	assert.Contains(t, cleanCmd.Example, "--dry-run")
	assert.Contains(t, cleanCmd.Example, "--yes")
	assert.Contains(t, cleanCmd.Example, "--keep-specs")
	assert.Contains(t, cleanCmd.Example, "--remove-specs")
}

func TestRunClean_NoFiles(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(tmpDir)

	cmd := getCleanCmd()
	require.NotNil(t, cmd, "clean command must exist")

	// Use RunE directly for testing, as Execute() goes through rootCmd
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "No autospec files found")
}

func TestRunClean_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(tmpDir)

	// Create some files
	require.NoError(t, os.MkdirAll(".autospec", 0755))
	require.NoError(t, os.MkdirAll("specs", 0755))

	cmd := getCleanCmd()
	require.NotNil(t, cmd, "clean command must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Set flags directly instead of using SetArgs
	cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Would remove")
	assert.Contains(t, output, ".autospec")
	// specs/ should be mentioned as preserved by default
	assert.Contains(t, output, "specs/ directory will be preserved by default")

	// Files should still exist
	_, err = os.Stat(".autospec")
	assert.NoError(t, err)
	_, err = os.Stat("specs")
	assert.NoError(t, err)
}

func TestRunClean_KeepSpecs_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(tmpDir)

	// Create files
	require.NoError(t, os.MkdirAll(".autospec", 0755))
	require.NoError(t, os.MkdirAll("specs", 0755))

	cmd := getCleanCmd()
	require.NotNil(t, cmd, "clean command must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Set flags directly
	cmd.Flags().Set("dry-run", "true")
	cmd.Flags().Set("keep-specs", "true")

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Would remove")
	assert.Contains(t, output, ".autospec")
	assert.Contains(t, output, "specs/ directory will be preserved")
	// Output should show files that would be removed, but not specs when --keep-specs
	// The "Would remove" list should not include specs
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "[dir] specs") {
			t.Error("specs should not be in the removal list when --keep-specs is set")
		}
	}
}

func TestRunClean_YesFlag(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(tmpDir)

	// Create files
	require.NoError(t, os.MkdirAll(".autospec", 0755))
	require.NoError(t, os.WriteFile(".autospec/test.txt", []byte("test"), 0644))
	require.NoError(t, os.MkdirAll("specs", 0755))

	cmd := getCleanCmd()
	require.NotNil(t, cmd, "clean command must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Reset flags and set the one we want
	cmd.Flags().Set("dry-run", "false")
	cmd.Flags().Set("yes", "true")
	cmd.Flags().Set("keep-specs", "false")
	cmd.Flags().Set("remove-specs", "false")

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Removed")
	assert.Contains(t, output, "Summary")

	// .autospec should be gone
	_, err = os.Stat(".autospec")
	assert.True(t, os.IsNotExist(err))

	// specs/ should still exist (--yes preserves specs by default)
	_, err = os.Stat("specs")
	assert.NoError(t, err)
}

func TestRunClean_WithCommandFiles(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(tmpDir)

	// Create command files
	require.NoError(t, os.MkdirAll(".claude/commands", 0755))
	require.NoError(t, os.WriteFile(".claude/commands/autospec.plan.md", []byte("plan"), 0644))
	require.NoError(t, os.WriteFile(".claude/commands/autospec.tasks.md", []byte("tasks"), 0644))
	require.NoError(t, os.WriteFile(".claude/commands/custom.md", []byte("custom"), 0644))

	cmd := getCleanCmd()
	require.NotNil(t, cmd, "clean command must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Reset flags and set the one we want
	cmd.Flags().Set("dry-run", "false")
	cmd.Flags().Set("yes", "true")
	cmd.Flags().Set("keep-specs", "false")
	cmd.Flags().Set("remove-specs", "false")

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "autospec.plan.md")
	assert.Contains(t, output, "autospec.tasks.md")
	assert.Contains(t, output, "Removed")

	// autospec files should be gone
	_, err = os.Stat(".claude/commands/autospec.plan.md")
	assert.True(t, os.IsNotExist(err))

	// custom.md should still exist
	_, err = os.Stat(".claude/commands/custom.md")
	assert.NoError(t, err)
}

func TestRunClean_OutputFormat(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(tmpDir)

	// Create directory
	require.NoError(t, os.MkdirAll(".autospec", 0755))

	cmd := getCleanCmd()
	require.NotNil(t, cmd, "clean command must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Set flags directly
	cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	// Should show type indicator
	assert.Contains(t, output, "[dir]")
	// Should show description
	assert.Contains(t, output, "Autospec configuration directory")
}

func TestRunClean_FileTypeIndicator(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(tmpDir)

	// Create command file
	require.NoError(t, os.MkdirAll(".claude/commands", 0755))
	require.NoError(t, os.WriteFile(".claude/commands/autospec.plan.md", []byte("test"), 0644))

	cmd := getCleanCmd()
	require.NotNil(t, cmd, "clean command must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Set flags directly
	cmd.Flags().Set("dry-run", "true")

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	// Should show file type indicator
	assert.Contains(t, output, "[file]")
}

func TestRunClean_Summary(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(tmpDir)

	// Create multiple items
	require.NoError(t, os.MkdirAll(".autospec", 0755))
	require.NoError(t, os.MkdirAll("specs", 0755))
	require.NoError(t, os.MkdirAll(".claude/commands", 0755))
	require.NoError(t, os.WriteFile(".claude/commands/autospec.plan.md", []byte("test"), 0644))

	cmd := getCleanCmd()
	require.NotNil(t, cmd, "clean command must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Reset flags and set the one we want
	cmd.Flags().Set("dry-run", "false")
	cmd.Flags().Set("yes", "true")
	cmd.Flags().Set("keep-specs", "false")
	cmd.Flags().Set("remove-specs", "false")

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Summary:")
	assert.Contains(t, output, "removed")
}

func TestRunClean_RemoveSpecsFlag(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(tmpDir)

	// Create files
	require.NoError(t, os.MkdirAll(".autospec", 0755))
	require.NoError(t, os.MkdirAll("specs", 0755))

	cmd := getCleanCmd()
	require.NotNil(t, cmd, "clean command must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Reset flags and set the ones we want
	cmd.Flags().Set("dry-run", "false")
	cmd.Flags().Set("yes", "true")
	cmd.Flags().Set("keep-specs", "false")
	cmd.Flags().Set("remove-specs", "true")

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Removed")
	assert.Contains(t, output, "specs")

	// Both .autospec and specs/ should be gone
	_, err = os.Stat(".autospec")
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat("specs")
	assert.True(t, os.IsNotExist(err))
}

func TestRunClean_RemoveSpecsFlag_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(tmpDir)

	// Create files
	require.NoError(t, os.MkdirAll(".autospec", 0755))
	require.NoError(t, os.MkdirAll("specs", 0755))

	cmd := getCleanCmd()
	require.NotNil(t, cmd, "clean command must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Set flags directly
	cmd.Flags().Set("dry-run", "true")
	cmd.Flags().Set("remove-specs", "true")

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Would remove")
	assert.Contains(t, output, ".autospec")
	// With --remove-specs, specs should be in the removal list
	assert.Contains(t, output, "[dir] specs")
}
