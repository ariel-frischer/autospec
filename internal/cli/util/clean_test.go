// Package util tests the clean command implementation.
// Related: internal/cli/util/clean.go
// Tags: util, cli, clean, commands

package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunClean_DryRun(t *testing.T) {
	// Cannot run in parallel - changes working directory

	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create .autospec directory
	autospecDir := filepath.Join(tmpDir, ".autospec")
	require.NoError(t, os.MkdirAll(autospecDir, 0755))

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	cmd := &cobra.Command{}
	cmd.Flags().Bool("dry-run", true, "")
	cmd.Flags().Bool("yes", false, "")
	cmd.Flags().Bool("keep-specs", false, "")
	cmd.Flags().Bool("remove-specs", false, "")
	var outBuf strings.Builder
	cmd.SetOut(&outBuf)

	runErr := runClean(cmd, []string{})
	assert.NoError(t, runErr)
	assert.Contains(t, outBuf.String(), "Would remove")
}

func TestRunClean_NoFiles(t *testing.T) {
	// Cannot run in parallel - changes working directory

	// Create empty temp directory
	tmpDir := t.TempDir()

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	cmd := &cobra.Command{}
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("yes", false, "")
	cmd.Flags().Bool("keep-specs", false, "")
	cmd.Flags().Bool("remove-specs", false, "")
	var outBuf strings.Builder
	cmd.SetOut(&outBuf)

	err = runClean(cmd, []string{})
	assert.NoError(t, err)
	assert.Contains(t, outBuf.String(), "No autospec files found")
}

func TestRunClean_WithYesFlag(t *testing.T) {
	// Cannot run in parallel - changes working directory

	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create .autospec directory
	autospecDir := filepath.Join(tmpDir, ".autospec")
	require.NoError(t, os.MkdirAll(autospecDir, 0755))

	// Create a test file
	testFile := filepath.Join(autospecDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	cmd := &cobra.Command{}
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("yes", true, "")
	cmd.Flags().Bool("keep-specs", false, "")
	cmd.Flags().Bool("remove-specs", false, "")
	var outBuf strings.Builder
	cmd.SetOut(&outBuf)

	runErr := runClean(cmd, []string{})
	assert.NoError(t, runErr)
	// Should attempt to remove files
	output := outBuf.String()
	assert.True(t, strings.Contains(output, "Removed") || strings.Contains(output, "Summary"))
}

func TestRunClean_KeepSpecs(t *testing.T) {
	// Cannot run in parallel - changes working directory

	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create .autospec and specs directories
	autospecDir := filepath.Join(tmpDir, ".autospec")
	specsDir := filepath.Join(tmpDir, "specs")
	require.NoError(t, os.MkdirAll(autospecDir, 0755))
	require.NoError(t, os.MkdirAll(specsDir, 0755))

	// Create a test file in .autospec to ensure there's something to clean
	testFile := filepath.Join(autospecDir, "config.yml")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	cmd := &cobra.Command{}
	cmd.Flags().Bool("dry-run", true, "")
	cmd.Flags().Bool("yes", false, "")
	cmd.Flags().Bool("keep-specs", true, "")
	cmd.Flags().Bool("remove-specs", false, "")
	var outBuf strings.Builder
	cmd.SetOut(&outBuf)

	runErr := runClean(cmd, []string{})
	assert.NoError(t, runErr)
	output := outBuf.String()
	// Should mention specs being preserved or show would remove message
	assert.True(t, strings.Contains(output, "preserved") || strings.Contains(output, "Would remove"))
}

func TestRunClean_RemoveSpecs(t *testing.T) {
	// Cannot run in parallel - changes working directory

	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create .autospec and specs directories
	autospecDir := filepath.Join(tmpDir, ".autospec")
	specsDir := filepath.Join(tmpDir, "specs")
	require.NoError(t, os.MkdirAll(autospecDir, 0755))
	require.NoError(t, os.MkdirAll(specsDir, 0755))

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	cmd := &cobra.Command{}
	cmd.Flags().Bool("dry-run", true, "")
	cmd.Flags().Bool("yes", false, "")
	cmd.Flags().Bool("keep-specs", false, "")
	cmd.Flags().Bool("remove-specs", true, "")
	var outBuf strings.Builder
	cmd.SetOut(&outBuf)

	runErr := runClean(cmd, []string{})
	assert.NoError(t, runErr)
	// In dry-run mode, should show what would be removed
	output := outBuf.String()
	assert.Contains(t, output, "Would remove")
}

func TestRunClean_OnlySpecsExist(t *testing.T) {
	// Cannot run in parallel - changes working directory

	// Create temp directory with only specs
	tmpDir := t.TempDir()

	// Create specs directory only
	specsDir := filepath.Join(tmpDir, "specs")
	require.NoError(t, os.MkdirAll(specsDir, 0755))

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	cmd := &cobra.Command{}
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("yes", true, "")
	cmd.Flags().Bool("keep-specs", false, "")
	cmd.Flags().Bool("remove-specs", true, "")
	var outBuf strings.Builder
	cmd.SetOut(&outBuf)

	err = runClean(cmd, []string{})
	// Should handle specs-only case
	assert.NoError(t, err)
}

func TestRunClean_DryRunWithSpecs(t *testing.T) {
	// Cannot run in parallel - changes working directory

	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create .autospec and specs directories
	autospecDir := filepath.Join(tmpDir, ".autospec")
	specsDir := filepath.Join(tmpDir, "specs")
	require.NoError(t, os.MkdirAll(autospecDir, 0755))
	require.NoError(t, os.MkdirAll(specsDir, 0755))

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	cmd := &cobra.Command{}
	cmd.Flags().Bool("dry-run", true, "")
	cmd.Flags().Bool("yes", false, "")
	cmd.Flags().Bool("keep-specs", false, "")
	cmd.Flags().Bool("remove-specs", false, "")
	var outBuf strings.Builder
	cmd.SetOut(&outBuf)

	runErr := runClean(cmd, []string{})
	assert.NoError(t, runErr)
	// In dry-run, should just show what would be done
	assert.NotContains(t, outBuf.String(), "Removed")
}

func TestRunClean_MultipleFiles(t *testing.T) {
	// Cannot run in parallel - changes working directory

	// Create temp directory with multiple test files
	tmpDir := t.TempDir()

	// Create .autospec directory with files
	autospecDir := filepath.Join(tmpDir, ".autospec")
	require.NoError(t, os.MkdirAll(autospecDir, 0755))

	// Create multiple test files
	for i := 1; i <= 3; i++ {
		testFile := filepath.Join(autospecDir, "test"+string(rune(i))+".txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))
	}

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.Chdir(tmpDir))

	cmd := &cobra.Command{}
	cmd.Flags().Bool("dry-run", true, "")
	cmd.Flags().Bool("yes", false, "")
	cmd.Flags().Bool("keep-specs", false, "")
	cmd.Flags().Bool("remove-specs", false, "")
	var outBuf strings.Builder
	cmd.SetOut(&outBuf)

	runErr := runClean(cmd, []string{})
	assert.NoError(t, runErr)
	assert.Contains(t, outBuf.String(), "Would remove")
}

func TestRunClean_YesWithRemoveSpecs(t *testing.T) {
	// Cannot run in parallel - changes working directory

	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create .autospec and specs directories
	autospecDir := filepath.Join(tmpDir, ".autospec")
	specsDir := filepath.Join(tmpDir, "specs")
	require.NoError(t, os.MkdirAll(autospecDir, 0755))
	require.NoError(t, os.MkdirAll(specsDir, 0755))

	// Create test files
	testFile := filepath.Join(autospecDir, "config.yml")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	cmd := &cobra.Command{}
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("yes", true, "")
	cmd.Flags().Bool("keep-specs", false, "")
	cmd.Flags().Bool("remove-specs", true, "")
	var outBuf strings.Builder
	cmd.SetOut(&outBuf)

	runErr := runClean(cmd, []string{})
	assert.NoError(t, runErr)
	// Should remove both .autospec and specs
	output := outBuf.String()
	assert.True(t, strings.Contains(output, "Summary") || strings.Contains(output, "Removed"))
}

func TestRunClean_OnlySpecsWithYesAndRemoveSpecs(t *testing.T) {
	// Cannot run in parallel - changes working directory

	// Create temp directory with only specs
	tmpDir := t.TempDir()

	// Create specs directory only
	specsDir := filepath.Join(tmpDir, "specs")
	require.NoError(t, os.MkdirAll(specsDir, 0755))

	// Create a test file in specs
	testFile := filepath.Join(specsDir, "test-spec", "spec.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(testFile), 0755))
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	cmd := &cobra.Command{}
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("yes", false, "")
	cmd.Flags().Bool("keep-specs", false, "")
	cmd.Flags().Bool("remove-specs", true, "")
	// Set input for prompt
	cmd.SetIn(strings.NewReader("y\n"))
	var outBuf strings.Builder
	cmd.SetOut(&outBuf)

	runErr := runClean(cmd, []string{})
	// Should handle specs-only case
	assert.NoError(t, runErr)
}

func TestRunClean_AutospecAndSpecs(t *testing.T) {
	// Cannot run in parallel - changes working directory

	// Create temp directory with both autospec and specs
	tmpDir := t.TempDir()

	// Create .autospec and specs directories
	autospecDir := filepath.Join(tmpDir, ".autospec")
	specsDir := filepath.Join(tmpDir, "specs")
	require.NoError(t, os.MkdirAll(autospecDir, 0755))
	require.NoError(t, os.MkdirAll(specsDir, 0755))

	// Create test files
	testFile := filepath.Join(autospecDir, "config.yml")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	cmd := &cobra.Command{}
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("yes", true, "")
	cmd.Flags().Bool("keep-specs", false, "")
	cmd.Flags().Bool("remove-specs", false, "")
	var outBuf strings.Builder
	cmd.SetOut(&outBuf)

	runErr := runClean(cmd, []string{})
	assert.NoError(t, runErr)
}
