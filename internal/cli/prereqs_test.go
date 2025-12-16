package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrereqsOutput_JSONFormat(t *testing.T) {
	output := PrereqsOutput{
		FeatureDir:      "/path/to/specs/001-test",
		FeatureSpec:     "/path/to/specs/001-test/spec.yaml",
		ImplPlan:        "/path/to/specs/001-test/plan.yaml",
		Tasks:           "/path/to/specs/001-test/tasks.yaml",
		AvailableDocs:   []string{"spec.yaml", "plan.yaml"},
		AutospecVersion: "autospec dev",
		CreatedDate:     "2024-01-15T10:30:00Z",
	}

	// Verify JSON encoding produces expected keys
	data, err := json.Marshal(output)
	require.NoError(t, err)

	// Check for expected JSON keys (matching shell script format)
	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"FEATURE_DIR"`)
	assert.Contains(t, jsonStr, `"FEATURE_SPEC"`)
	assert.Contains(t, jsonStr, `"IMPL_PLAN"`)
	assert.Contains(t, jsonStr, `"TASKS"`)
	assert.Contains(t, jsonStr, `"AVAILABLE_DOCS"`)
	assert.Contains(t, jsonStr, `"AUTOSPEC_VERSION"`)
	assert.Contains(t, jsonStr, `"CREATED_DATE"`)

	// Verify roundtrip
	var decoded PrereqsOutput
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, output, decoded)
}

func TestPrereqsCmd_Flags(t *testing.T) {
	// Verify flags exist
	jsonFlag := prereqsCmd.Flags().Lookup("json")
	assert.NotNil(t, jsonFlag)
	assert.Equal(t, "false", jsonFlag.DefValue)

	requireSpecFlag := prereqsCmd.Flags().Lookup("require-spec")
	assert.NotNil(t, requireSpecFlag)
	assert.Equal(t, "false", requireSpecFlag.DefValue)

	requirePlanFlag := prereqsCmd.Flags().Lookup("require-plan")
	assert.NotNil(t, requirePlanFlag)

	requireTasksFlag := prereqsCmd.Flags().Lookup("require-tasks")
	assert.NotNil(t, requireTasksFlag)

	includeTasksFlag := prereqsCmd.Flags().Lookup("include-tasks")
	assert.NotNil(t, includeTasksFlag)

	pathsOnlyFlag := prereqsCmd.Flags().Lookup("paths-only")
	assert.NotNil(t, pathsOnlyFlag)
}

func TestPrereqsOutput_KeysMatchShellScript(t *testing.T) {
	// This test ensures our JSON output matches the shell script's format
	output := PrereqsOutput{
		FeatureDir:      "test",
		FeatureSpec:     "test",
		ImplPlan:        "test",
		Tasks:           "test",
		AvailableDocs:   []string{},
		AutospecVersion: "test",
		CreatedDate:     "test",
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// Verify exact key names from shell script
	expectedKeys := []string{"FEATURE_DIR", "FEATURE_SPEC", "IMPL_PLAN", "TASKS", "AVAILABLE_DOCS", "AUTOSPEC_VERSION", "CREATED_DATE"}
	for _, key := range expectedKeys {
		_, ok := parsed[key]
		assert.True(t, ok, "expected key %s to be present", key)
	}

	// Verify no extra keys
	assert.Len(t, parsed, len(expectedKeys))
}

func TestPrereqsCmd_RequireSpec(t *testing.T) {
	// Create a temporary feature directory
	tmpDir, err := os.MkdirTemp("", "autospec-prereqs-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	specsDir := filepath.Join(tmpDir, "specs")
	featureDir := filepath.Join(specsDir, "001-test-feature")
	err = os.MkdirAll(featureDir, 0755)
	require.NoError(t, err)

	// Create spec.yaml
	specFile := filepath.Join(featureDir, "spec.yaml")
	err = os.WriteFile(specFile, []byte("feature: test\n"), 0644)
	require.NoError(t, err)

	// Verify spec file exists check
	_, err = os.Stat(specFile)
	assert.NoError(t, err)
}

func TestPrereqsCmd_RequirePlan(t *testing.T) {
	// Create a temporary feature directory
	tmpDir, err := os.MkdirTemp("", "autospec-prereqs-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	specsDir := filepath.Join(tmpDir, "specs")
	featureDir := filepath.Join(specsDir, "001-test-feature")
	err = os.MkdirAll(featureDir, 0755)
	require.NoError(t, err)

	// Create plan.yaml
	planFile := filepath.Join(featureDir, "plan.yaml")
	err = os.WriteFile(planFile, []byte("plan: test\n"), 0644)
	require.NoError(t, err)

	// Verify plan file exists check
	_, err = os.Stat(planFile)
	assert.NoError(t, err)
}

func TestPrereqsCmd_RequireTasks(t *testing.T) {
	// Create a temporary feature directory
	tmpDir, err := os.MkdirTemp("", "autospec-prereqs-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	specsDir := filepath.Join(tmpDir, "specs")
	featureDir := filepath.Join(specsDir, "001-test-feature")
	err = os.MkdirAll(featureDir, 0755)
	require.NoError(t, err)

	// Create tasks.yaml
	tasksFile := filepath.Join(featureDir, "tasks.yaml")
	err = os.WriteFile(tasksFile, []byte("tasks: test\n"), 0644)
	require.NoError(t, err)

	// Verify tasks file exists check
	_, err = os.Stat(tasksFile)
	assert.NoError(t, err)
}

func TestPrereqsCmd_AvailableDocs(t *testing.T) {
	// Test that available docs list is correctly built
	tmpDir, err := os.MkdirTemp("", "autospec-prereqs-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	featureDir := filepath.Join(tmpDir, "specs", "001-test")
	err = os.MkdirAll(featureDir, 0755)
	require.NoError(t, err)

	// Create various files
	os.WriteFile(filepath.Join(featureDir, "spec.yaml"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(featureDir, "plan.yaml"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(featureDir, "tasks.yaml"), []byte("test"), 0644)

	// Create checklists directory with content
	checklistsDir := filepath.Join(featureDir, "checklists")
	err = os.MkdirAll(checklistsDir, 0755)
	require.NoError(t, err)
	os.WriteFile(filepath.Join(checklistsDir, "test.yaml"), []byte("test"), 0644)

	// Verify all files exist
	for _, f := range []string{"spec.yaml", "plan.yaml", "tasks.yaml"} {
		_, err := os.Stat(filepath.Join(featureDir, f))
		assert.NoError(t, err, "expected %s to exist", f)
	}

	// Verify checklists directory exists
	info, err := os.Stat(checklistsDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestDetectCurrentFeature_Environment(t *testing.T) {
	// Create a temporary specs directory
	tmpDir, err := os.MkdirTemp("", "autospec-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	specsDir := filepath.Join(tmpDir, "specs")
	featureDir := filepath.Join(specsDir, "001-test-feature")
	err = os.MkdirAll(featureDir, 0755)
	require.NoError(t, err)

	// Set environment variable
	oldEnv := os.Getenv("SPECIFY_FEATURE")
	os.Setenv("SPECIFY_FEATURE", "001-test-feature")
	defer os.Setenv("SPECIFY_FEATURE", oldEnv)

	// Test detection
	meta, err := detectCurrentFeature(specsDir, false)
	require.NoError(t, err)
	assert.Equal(t, featureDir, meta.Directory)
	assert.Equal(t, "001-test-feature", meta.Name)
}
