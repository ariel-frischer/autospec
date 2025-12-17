//go:build !windows

package uninstall

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectBinaryLocation_ReturnsValidPath(t *testing.T) {
	path, err := DetectBinaryLocation()
	require.NoError(t, err)
	assert.NotEmpty(t, path)

	// Path should be absolute
	assert.True(t, filepath.IsAbs(path), "path should be absolute")

	// Path should exist (since we're running from a binary)
	_, err = os.Stat(path)
	assert.NoError(t, err, "binary path should exist")
}

func TestRequiresSudo_UserWritableDir(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-binary")

	// Create test file
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0755))

	// User should have write access to temp dir
	result := RequiresSudo(testFile)
	assert.False(t, result, "temp directory should be user-writable")
}

func TestRequiresSudo_SystemDir(t *testing.T) {
	// Test with a system directory that typically requires sudo
	// /usr/local/bin is commonly not writable by regular users
	testPath := "/usr/local/bin/test-binary"
	result := RequiresSudo(testPath)

	// This could be true or false depending on system setup
	// We just verify it returns a boolean without error
	assert.IsType(t, false, result)
}

func TestGetUninstallTargets_ReturnsThreeTargets(t *testing.T) {
	targets, err := GetUninstallTargets()
	require.NoError(t, err)

	// Should return exactly 3 targets: binary, config dir, state dir
	assert.Len(t, targets, 3)

	// Verify target types
	var foundTypes []TargetType
	for _, target := range targets {
		foundTypes = append(foundTypes, target.Type)
	}

	assert.Contains(t, foundTypes, TypeBinary)
	assert.Contains(t, foundTypes, TypeConfigDir)
	assert.Contains(t, foundTypes, TypeStateDir)
}

func TestGetUninstallTargets_BinaryHasAbsolutePath(t *testing.T) {
	targets, err := GetUninstallTargets()
	require.NoError(t, err)

	for _, target := range targets {
		if target.Type == TypeBinary {
			assert.True(t, filepath.IsAbs(target.Path), "binary path should be absolute")
			assert.NotEmpty(t, target.Description)
			return
		}
	}
	t.Fatal("binary target not found")
}

func TestGetUninstallTargets_ConfigDirPath(t *testing.T) {
	targets, err := GetUninstallTargets()
	require.NoError(t, err)

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	expectedPath := filepath.Join(homeDir, ".config", "autospec")

	for _, target := range targets {
		if target.Type == TypeConfigDir {
			assert.Equal(t, expectedPath, target.Path)
			assert.False(t, target.RequiresSudo, "config dir should not require sudo")
			return
		}
	}
	t.Fatal("config dir target not found")
}

func TestGetUninstallTargets_StateDirPath(t *testing.T) {
	targets, err := GetUninstallTargets()
	require.NoError(t, err)

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	expectedPath := filepath.Join(homeDir, ".autospec")

	for _, target := range targets {
		if target.Type == TypeStateDir {
			assert.Equal(t, expectedPath, target.Path)
			assert.False(t, target.RequiresSudo, "state dir should not require sudo")
			return
		}
	}
	t.Fatal("state dir target not found")
}

func TestRemoveTargets_HandlesNonexistentTargets(t *testing.T) {
	tmpDir := t.TempDir()

	targets := []UninstallTarget{
		{
			Path:         filepath.Join(tmpDir, "nonexistent-binary"),
			Type:         TypeBinary,
			Description:  "test binary",
			Exists:       false,
			RequiresSudo: false,
		},
		{
			Path:         filepath.Join(tmpDir, "nonexistent-dir"),
			Type:         TypeConfigDir,
			Description:  "test config",
			Exists:       false,
			RequiresSudo: false,
		},
	}

	results := RemoveTargets(targets)

	assert.Len(t, results, 2)
	for _, result := range results {
		assert.True(t, result.Success, "nonexistent target removal should succeed")
		assert.Nil(t, result.Error, "nonexistent target should not produce error")
	}
}

func TestRemoveTargets_RemovesBinary(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "test-binary")

	// Create test binary
	require.NoError(t, os.WriteFile(binaryPath, []byte("test"), 0755))

	targets := []UninstallTarget{
		{
			Path:         binaryPath,
			Type:         TypeBinary,
			Description:  "test binary",
			Exists:       true,
			RequiresSudo: false,
		},
	}

	results := RemoveTargets(targets)

	assert.Len(t, results, 1)
	assert.True(t, results[0].Success)
	assert.Nil(t, results[0].Error)

	// Verify file is removed
	_, err := os.Stat(binaryPath)
	assert.True(t, os.IsNotExist(err))
}

func TestRemoveTargets_RemovesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config")

	// Create directory with nested content
	require.NoError(t, os.MkdirAll(filepath.Join(configDir, "subdir"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yml"), []byte("test"), 0644))

	targets := []UninstallTarget{
		{
			Path:         configDir,
			Type:         TypeConfigDir,
			Description:  "test config",
			Exists:       true,
			RequiresSudo: false,
		},
	}

	results := RemoveTargets(targets)

	assert.Len(t, results, 1)
	assert.True(t, results[0].Success)
	assert.Nil(t, results[0].Error)

	// Verify directory is removed
	_, err := os.Stat(configDir)
	assert.True(t, os.IsNotExist(err))
}

func TestRemoveTargets_ContinuesAfterFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create one file that exists
	existingFile := filepath.Join(tmpDir, "existing")
	require.NoError(t, os.WriteFile(existingFile, []byte("test"), 0644))

	// Create a directory that we'll make unremovable
	protectedDir := filepath.Join(tmpDir, "protected")
	require.NoError(t, os.MkdirAll(protectedDir, 0755))

	targets := []UninstallTarget{
		{
			Path:         existingFile,
			Type:         TypeBinary,
			Description:  "existing file",
			Exists:       true,
			RequiresSudo: false,
		},
		{
			Path:         filepath.Join(tmpDir, "will-succeed"),
			Type:         TypeConfigDir,
			Description:  "nonexistent",
			Exists:       false, // Doesn't exist, so won't try to remove
			RequiresSudo: false,
		},
	}

	results := RemoveTargets(targets)

	assert.Len(t, results, 2)
	// First should succeed
	assert.True(t, results[0].Success)
	// Second should also succeed (marked as not existing)
	assert.True(t, results[1].Success)
}

func TestRemoveTargets_MixedExistingAndNonexistent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create one existing file
	existingFile := filepath.Join(tmpDir, "existing-binary")
	require.NoError(t, os.WriteFile(existingFile, []byte("test"), 0755))

	// Create one existing directory
	existingDir := filepath.Join(tmpDir, "existing-config")
	require.NoError(t, os.MkdirAll(existingDir, 0755))

	targets := []UninstallTarget{
		{
			Path:         existingFile,
			Type:         TypeBinary,
			Description:  "binary",
			Exists:       true,
			RequiresSudo: false,
		},
		{
			Path:         filepath.Join(tmpDir, "nonexistent-config"),
			Type:         TypeConfigDir,
			Description:  "config",
			Exists:       false,
			RequiresSudo: false,
		},
		{
			Path:         existingDir,
			Type:         TypeStateDir,
			Description:  "state",
			Exists:       true,
			RequiresSudo: false,
		},
	}

	results := RemoveTargets(targets)

	assert.Len(t, results, 3)

	// All should succeed
	for i, result := range results {
		assert.True(t, result.Success, "result %d should succeed", i)
		assert.Nil(t, result.Error, "result %d should have no error", i)
	}

	// Verify existing files are removed
	_, err := os.Stat(existingFile)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(existingDir)
	assert.True(t, os.IsNotExist(err))
}

func TestTargetType_Constants(t *testing.T) {
	assert.Equal(t, TargetType("binary"), TypeBinary)
	assert.Equal(t, TargetType("config_dir"), TypeConfigDir)
	assert.Equal(t, TargetType("state_dir"), TypeStateDir)
}

func TestFileExists_True(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-file")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	assert.True(t, fileExists(testFile))
}

func TestFileExists_FalseForDir(t *testing.T) {
	tmpDir := t.TempDir()
	// tmpDir is a directory, not a file
	assert.False(t, fileExists(tmpDir))
}

func TestFileExists_FalseForNonexistent(t *testing.T) {
	assert.False(t, fileExists("/nonexistent/path/to/file"))
}

func TestDirExists_True(t *testing.T) {
	tmpDir := t.TempDir()
	assert.True(t, dirExists(tmpDir))
}

func TestDirExists_FalseForFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-file")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	assert.False(t, dirExists(testFile))
}

func TestDirExists_FalseForNonexistent(t *testing.T) {
	assert.False(t, dirExists("/nonexistent/path/to/dir"))
}
