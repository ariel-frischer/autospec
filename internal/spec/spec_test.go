package spec

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectCurrentSpec_FromBranch(t *testing.T) {
	// This test runs against the real git repository
	// It verifies that DetectCurrentSpec returns valid metadata
	// without hardcoding a specific branch (which changes during development)
	specsDir := "/home/ari/repos/auto-claude-speckit/specs"
	meta, err := DetectCurrentSpec(specsDir)
	require.NoError(t, err)
	// Verify we got valid metadata structure
	assert.NotEmpty(t, meta.Number, "spec number should not be empty")
	assert.NotEmpty(t, meta.Name, "spec name should not be empty")
	assert.NotEmpty(t, meta.Branch, "branch should not be empty")
	assert.NotEmpty(t, meta.Directory, "directory should not be empty")
	// Verify the directory exists
	_, err = os.Stat(meta.Directory)
	assert.NoError(t, err, "spec directory should exist")
}

func TestDetectCurrentSpec_FromDirectory(t *testing.T) {
	t.Parallel()

	// Create test specs directory
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	require.NoError(t, os.MkdirAll(specsDir, 0755))

	// Create spec directories with different modification times
	oldSpec := filepath.Join(specsDir, "001-old-feature")
	newSpec := filepath.Join(specsDir, "002-new-feature")
	require.NoError(t, os.MkdirAll(oldSpec, 0755))
	time.Sleep(10 * time.Millisecond) // Ensure different mod times
	require.NoError(t, os.MkdirAll(newSpec, 0755))

	// Should detect the most recent (002-new-feature)
	meta, err := DetectCurrentSpec(specsDir)
	require.NoError(t, err)
	assert.Equal(t, "002", meta.Number)
	assert.Equal(t, "new-feature", meta.Name)
	assert.Equal(t, newSpec, meta.Directory)
}

func TestDetectCurrentSpec_NoSpecsFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "empty-specs")
	require.NoError(t, os.MkdirAll(specsDir, 0755))

	_, err := DetectCurrentSpec(specsDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no spec directories found")
}

func TestGetSpecDirectory_ExactMatch(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	specDir := filepath.Join(specsDir, "002-go-binary-migration")
	require.NoError(t, os.MkdirAll(specDir, 0755))

	result, err := GetSpecDirectory(specsDir, "002-go-binary-migration")
	require.NoError(t, err)
	assert.Equal(t, specDir, result)
}

func TestGetSpecDirectory_NumberMatch(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	specDir := filepath.Join(specsDir, "002-go-binary-migration")
	require.NoError(t, os.MkdirAll(specDir, 0755))

	result, err := GetSpecDirectory(specsDir, "002")
	require.NoError(t, err)
	assert.Equal(t, specDir, result)
}

func TestGetSpecDirectory_NameMatch(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	specDir := filepath.Join(specsDir, "002-go-binary-migration")
	require.NoError(t, os.MkdirAll(specDir, 0755))

	result, err := GetSpecDirectory(specsDir, "go-binary-migration")
	require.NoError(t, err)
	assert.Equal(t, specDir, result)
}

func TestGetSpecDirectory_NotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	require.NoError(t, os.MkdirAll(specsDir, 0755))

	_, err := GetSpecDirectory(specsDir, "999")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetSpecDirectory_MultipleMatches(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	spec1 := filepath.Join(specsDir, "001-test-feature")
	spec2 := filepath.Join(specsDir, "002-test-feature")
	require.NoError(t, os.MkdirAll(spec1, 0755))
	require.NoError(t, os.MkdirAll(spec2, 0755))

	_, err := GetSpecDirectory(specsDir, "test-feature")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple specs found")
}
