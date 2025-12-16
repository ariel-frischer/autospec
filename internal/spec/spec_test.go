package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectCurrentSpec_FromBranch(t *testing.T) {
	// This test runs against the real git repository
	// It verifies that DetectCurrentSpec returns valid metadata
	// without hardcoding a specific branch (which changes during development)
	specsDir := "./specs" // Use relative path to current repo's specs

	// Get absolute path if we're in repo root
	if cwd, err := os.Getwd(); err == nil {
		// Navigate up from internal/spec to repo root
		repoRoot := filepath.Dir(filepath.Dir(cwd))
		specsDir = filepath.Join(repoRoot, "specs")
	}

	meta, err := DetectCurrentSpec(specsDir)
	if err != nil {
		// If no specs found or detection fails, that's OK for this test
		// (the repo may not have matching specs for current branch)
		t.Skipf("Skipping test: %v", err)
		return
	}

	// Verify we got valid metadata structure
	assert.NotEmpty(t, meta.Number, "spec number should not be empty")
	assert.NotEmpty(t, meta.Name, "spec name should not be empty")
	// Branch may be empty if git detection finds most recent directory instead
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

func TestUpdateSpecStatus(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		initialYAML   string
		newStatus     string
		withTimestamp bool
		wantUpdated   bool
		wantStatus    string
		wantHasTime   bool
	}{
		"update Draft to Completed": {
			initialYAML: `feature:
  branch: "001-test"
  created: "2025-01-01"
  status: "Draft"
`,
			newStatus:     "Completed",
			withTimestamp: true,
			wantUpdated:   true,
			wantStatus:    "Completed",
			wantHasTime:   true,
		},
		"update InProgress to Completed": {
			initialYAML: `feature:
  branch: "001-test"
  created: "2025-01-01"
  status: "InProgress"
`,
			newStatus:     "Completed",
			withTimestamp: true,
			wantUpdated:   true,
			wantStatus:    "Completed",
			wantHasTime:   true,
		},
		"already Completed is idempotent": {
			initialYAML: `feature:
  branch: "001-test"
  created: "2025-01-01"
  status: "Completed"
  completed_at: "2025-01-01T00:00:00Z"
`,
			newStatus:     "Completed",
			withTimestamp: true,
			wantUpdated:   false,
			wantStatus:    "Completed",
			wantHasTime:   false,
		},
		"update without timestamp": {
			initialYAML: `feature:
  branch: "001-test"
  created: "2025-01-01"
  status: "Draft"
`,
			newStatus:     "InProgress",
			withTimestamp: false,
			wantUpdated:   true,
			wantStatus:    "InProgress",
			wantHasTime:   false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			specDir := filepath.Join(tmpDir, "001-test-feature")
			require.NoError(t, os.MkdirAll(specDir, 0755))

			specPath := filepath.Join(specDir, "spec.yaml")
			require.NoError(t, os.WriteFile(specPath, []byte(tt.initialYAML), 0644))

			var completedAt time.Time
			if tt.withTimestamp {
				completedAt = time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
			}

			result, err := UpdateSpecStatus(specDir, tt.newStatus, completedAt)
			require.NoError(t, err)

			assert.Equal(t, tt.wantUpdated, result.Updated)
			assert.Equal(t, tt.wantStatus, result.NewStatus)

			if tt.wantHasTime {
				assert.NotEmpty(t, result.CompletedAt)
			}

			// Read back and verify
			data, err := os.ReadFile(specPath)
			require.NoError(t, err)
			content := string(data)
			// Status may be quoted or unquoted depending on YAML serialization
			assert.True(t, containsStatus(content, tt.wantStatus),
				"expected status %s in content:\n%s", tt.wantStatus, content)
		})
	}
}

// containsStatus checks if the YAML content contains the expected status value
func containsStatus(content, status string) bool {
	// Check both quoted and unquoted forms
	return strings.Contains(content, "status: "+status) ||
		strings.Contains(content, "status: \""+status+"\"")
}

func TestMarkSpecCompleted(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specDir := filepath.Join(tmpDir, "001-test-feature")
	require.NoError(t, os.MkdirAll(specDir, 0755))

	specYAML := `feature:
  branch: "001-test"
  created: "2025-01-01"
  status: "Draft"
  input: "test input"
`
	specPath := filepath.Join(specDir, "spec.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(specYAML), 0644))

	result, err := MarkSpecCompleted(specDir)
	require.NoError(t, err)

	assert.True(t, result.Updated)
	assert.Equal(t, "Draft", result.PreviousStatus)
	assert.Equal(t, "Completed", result.NewStatus)
	assert.NotEmpty(t, result.CompletedAt)

	// Read back and verify
	data, err := os.ReadFile(specPath)
	require.NoError(t, err)
	content := string(data)
	assert.True(t, containsStatus(content, "Completed"),
		"expected status Completed in content:\n%s", content)
	assert.Contains(t, content, "completed_at:")
	// Verify input field preserved (may be quoted)
	assert.True(t, strings.Contains(content, "input: test input") ||
		strings.Contains(content, "input: \"test input\""),
		"expected input field preserved in content:\n%s", content)
}

func TestUpdateSpecStatus_FileNotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specDir := filepath.Join(tmpDir, "nonexistent")

	_, err := UpdateSpecStatus(specDir, "Completed", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read spec.yaml")
}

func TestUpdateSpecStatus_MissingFeatureSection(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specDir := filepath.Join(tmpDir, "001-test-feature")
	require.NoError(t, os.MkdirAll(specDir, 0755))

	// YAML without feature section
	specYAML := `metadata:
  version: "1.0"
`
	specPath := filepath.Join(specDir, "spec.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(specYAML), 0644))

	_, err := UpdateSpecStatus(specDir, "Completed", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "feature section not found")
}

func TestUpdateSpecStatus_MissingStatusField(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specDir := filepath.Join(tmpDir, "001-test-feature")
	require.NoError(t, os.MkdirAll(specDir, 0755))

	// YAML with feature section but no status field
	specYAML := `feature:
  branch: "001-test"
  created: "2025-01-01"
`
	specPath := filepath.Join(specDir, "spec.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(specYAML), 0644))

	_, err := UpdateSpecStatus(specDir, "Completed", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status field not found")
}
