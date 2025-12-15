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

func TestCleanCmdRegistration(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "clean" {
			found = true
			break
		}
	}
	assert.True(t, found, "clean command should be registered")
}

func TestCleanCmdFlags(t *testing.T) {
	flags := []struct {
		name      string
		shorthand string
	}{
		{"dry-run", "n"},
		{"yes", "y"},
		{"keep-specs", "k"},
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
	assert.Contains(t, cleanCmd.Short, "Remove")
	assert.Contains(t, cleanCmd.Short, "autospec")
}

func TestCleanCmdLongDescription(t *testing.T) {
	keywords := []string{
		".autospec/",
		".claude/commands/autospec",
		"specs/",
		"--dry-run",
		"--yes",
		"confirmation",
	}

	for _, keyword := range keywords {
		assert.Contains(t, cleanCmd.Long, keyword)
	}
}

func TestCleanCmdExamples(t *testing.T) {
	assert.Contains(t, cleanCmd.Example, "--dry-run")
	assert.Contains(t, cleanCmd.Example, "--yes")
	assert.Contains(t, cleanCmd.Example, "--keep-specs")
}

func TestRunClean_NoFiles(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(tmpDir)

	cmd := &cobra.Command{
		Use:  "clean",
		RunE: runClean,
	}
	cmd.Flags().BoolP("dry-run", "n", false, "")
	cmd.Flags().BoolP("yes", "y", false, "")
	cmd.Flags().BoolP("keep-specs", "k", false, "")

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

	cmd := &cobra.Command{
		Use:  "clean",
		RunE: runClean,
	}
	cmd.Flags().BoolP("dry-run", "n", false, "")
	cmd.Flags().BoolP("yes", "y", false, "")
	cmd.Flags().BoolP("keep-specs", "k", false, "")

	require.NoError(t, cmd.Flags().Set("dry-run", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Would remove")
	assert.Contains(t, output, ".autospec")
	assert.Contains(t, output, "specs")

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

	cmd := &cobra.Command{
		Use:  "clean",
		RunE: runClean,
	}
	cmd.Flags().BoolP("dry-run", "n", false, "")
	cmd.Flags().BoolP("yes", "y", false, "")
	cmd.Flags().BoolP("keep-specs", "k", false, "")

	require.NoError(t, cmd.Flags().Set("dry-run", "true"))
	require.NoError(t, cmd.Flags().Set("keep-specs", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

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

	cmd := &cobra.Command{
		Use:  "clean",
		RunE: runClean,
	}
	cmd.Flags().BoolP("dry-run", "n", false, "")
	cmd.Flags().BoolP("yes", "y", false, "")
	cmd.Flags().BoolP("keep-specs", "k", false, "")

	require.NoError(t, cmd.Flags().Set("yes", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Removed")
	assert.Contains(t, output, "Summary")

	// File should be gone
	_, err = os.Stat(".autospec")
	assert.True(t, os.IsNotExist(err))
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

	cmd := &cobra.Command{
		Use:  "clean",
		RunE: runClean,
	}
	cmd.Flags().BoolP("dry-run", "n", false, "")
	cmd.Flags().BoolP("yes", "y", false, "")
	cmd.Flags().BoolP("keep-specs", "k", false, "")

	require.NoError(t, cmd.Flags().Set("yes", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

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

	cmd := &cobra.Command{
		Use:  "clean",
		RunE: runClean,
	}
	cmd.Flags().BoolP("dry-run", "n", false, "")
	cmd.Flags().BoolP("yes", "y", false, "")
	cmd.Flags().BoolP("keep-specs", "k", false, "")

	require.NoError(t, cmd.Flags().Set("dry-run", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

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

	cmd := &cobra.Command{
		Use:  "clean",
		RunE: runClean,
	}
	cmd.Flags().BoolP("dry-run", "n", false, "")
	cmd.Flags().BoolP("yes", "y", false, "")
	cmd.Flags().BoolP("keep-specs", "k", false, "")

	require.NoError(t, cmd.Flags().Set("dry-run", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

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

	cmd := &cobra.Command{
		Use:  "clean",
		RunE: runClean,
	}
	cmd.Flags().BoolP("dry-run", "n", false, "")
	cmd.Flags().BoolP("yes", "y", false, "")
	cmd.Flags().BoolP("keep-specs", "k", false, "")

	require.NoError(t, cmd.Flags().Set("yes", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Summary:")
	assert.Contains(t, output, "removed")
}
