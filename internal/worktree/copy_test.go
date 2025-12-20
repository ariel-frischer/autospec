package worktree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyDir_BasicCopy(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	dstDir := filepath.Join(t.TempDir(), "dst")

	// Create source structure
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file2.txt"), []byte("content2"), 0644))

	err := CopyDir(srcDir, dstDir)
	require.NoError(t, err)

	// Verify copies
	content1, err := os.ReadFile(filepath.Join(dstDir, "file1.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content1", string(content1))

	content2, err := os.ReadFile(filepath.Join(dstDir, "file2.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content2", string(content2))
}

func TestCopyDir_NestedDirectories(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	dstDir := filepath.Join(t.TempDir(), "dst")

	// Create nested structure
	nested := filepath.Join(srcDir, "level1", "level2")
	require.NoError(t, os.MkdirAll(nested, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nested, "deep.txt"), []byte("deep content"), 0644))

	err := CopyDir(srcDir, dstDir)
	require.NoError(t, err)

	// Verify nested copy
	content, err := os.ReadFile(filepath.Join(dstDir, "level1", "level2", "deep.txt"))
	require.NoError(t, err)
	assert.Equal(t, "deep content", string(content))
}

func TestCopyDir_MissingSource(t *testing.T) {
	t.Parallel()

	srcDir := filepath.Join(t.TempDir(), "nonexistent")
	dstDir := filepath.Join(t.TempDir(), "dst")

	err := CopyDir(srcDir, dstDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accessing source directory")
}

func TestCopyDir_SourceNotDirectory(t *testing.T) {
	t.Parallel()

	srcFile := filepath.Join(t.TempDir(), "file.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("content"), 0644))
	dstDir := filepath.Join(t.TempDir(), "dst")

	err := CopyDir(srcFile, dstDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}

func TestCopyDir_PreservesPermissions(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	dstDir := filepath.Join(t.TempDir(), "dst")

	// Create file with specific permissions
	srcFile := filepath.Join(srcDir, "script.sh")
	require.NoError(t, os.WriteFile(srcFile, []byte("#!/bin/bash"), 0755))

	err := CopyDir(srcDir, dstDir)
	require.NoError(t, err)

	// Verify permissions preserved
	dstInfo, err := os.Stat(filepath.Join(dstDir, "script.sh"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), dstInfo.Mode().Perm())
}

func TestCopyDirs_MultipleDirectories(t *testing.T) {
	t.Parallel()

	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	// Create source directories
	require.NoError(t, os.MkdirAll(filepath.Join(srcRoot, ".autospec"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcRoot, ".autospec", "config.yml"), []byte("key: value"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(srcRoot, ".claude"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcRoot, ".claude", "settings.json"), []byte("{}"), 0644))

	copied, err := CopyDirs(srcRoot, dstRoot, []string{".autospec", ".claude"})
	require.NoError(t, err)
	assert.Equal(t, []string{".autospec", ".claude"}, copied)

	// Verify both copied
	_, err = os.Stat(filepath.Join(dstRoot, ".autospec", "config.yml"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(dstRoot, ".claude", "settings.json"))
	assert.NoError(t, err)
}

func TestCopyDirs_SkipsMissing(t *testing.T) {
	t.Parallel()

	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	// Create only one directory
	require.NoError(t, os.MkdirAll(filepath.Join(srcRoot, ".autospec"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcRoot, ".autospec", "config.yml"), []byte("key: value"), 0644))

	copied, err := CopyDirs(srcRoot, dstRoot, []string{".autospec", ".missing"})
	require.NoError(t, err)
	assert.Equal(t, []string{".autospec"}, copied)

	// Verify only existing dir copied
	_, err = os.Stat(filepath.Join(dstRoot, ".autospec"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(dstRoot, ".missing"))
	assert.True(t, os.IsNotExist(err))
}

func TestCopyDirs_AllMissing(t *testing.T) {
	t.Parallel()

	srcRoot := t.TempDir()
	dstRoot := t.TempDir()

	copied, err := CopyDirs(srcRoot, dstRoot, []string{".missing1", ".missing2"})
	require.NoError(t, err)
	assert.Empty(t, copied)
}
