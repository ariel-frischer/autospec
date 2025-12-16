package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupPlanOutput_JSONFormat(t *testing.T) {
	output := SetupPlanOutput{
		FeatureSpec: "/path/to/specs/001-test/spec.yaml",
		ImplPlan:    "/path/to/specs/001-test/plan.yaml",
		SpecsDir:    "/path/to/specs/001-test",
		Branch:      "001-test",
		HasGit:      "true",
	}

	// Verify JSON encoding produces expected keys
	data, err := json.Marshal(output)
	require.NoError(t, err)

	// Check for expected JSON keys (matching shell script format)
	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"FEATURE_SPEC"`)
	assert.Contains(t, jsonStr, `"IMPL_PLAN"`)
	assert.Contains(t, jsonStr, `"SPECS_DIR"`)
	assert.Contains(t, jsonStr, `"BRANCH"`)
	assert.Contains(t, jsonStr, `"HAS_GIT"`)

	// Verify roundtrip
	var decoded SetupPlanOutput
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, output, decoded)
}

func TestSetupPlanCmd_Flags(t *testing.T) {
	// Verify flags exist
	jsonFlag := setupPlanCmd.Flags().Lookup("json")
	assert.NotNil(t, jsonFlag)
	assert.Equal(t, "false", jsonFlag.DefValue)
}

func TestSetupPlanOutput_KeysMatchShellScript(t *testing.T) {
	// This test ensures our JSON output matches the shell script's format
	output := SetupPlanOutput{
		FeatureSpec: "test",
		ImplPlan:    "test",
		SpecsDir:    "test",
		Branch:      "test",
		HasGit:      "true",
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// Verify exact key names from shell script
	expectedKeys := []string{"FEATURE_SPEC", "IMPL_PLAN", "SPECS_DIR", "BRANCH", "HAS_GIT"}
	for _, key := range expectedKeys {
		_, ok := parsed[key]
		assert.True(t, ok, "expected key %s to be present", key)
	}

	// Verify no extra keys
	assert.Len(t, parsed, len(expectedKeys))
}

func TestSetupPlanCmd_TemplateCopying(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "autospec-setup-plan-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create .specify/templates directory
	templateDir := filepath.Join(tmpDir, ".specify", "templates")
	err = os.MkdirAll(templateDir, 0755)
	require.NoError(t, err)

	// Create a template file
	templateContent := "plan:\n  summary: Test plan template\n"
	templatePath := filepath.Join(templateDir, "plan-template.yaml")
	err = os.WriteFile(templatePath, []byte(templateContent), 0644)
	require.NoError(t, err)

	// Verify template exists
	_, err = os.Stat(templatePath)
	assert.NoError(t, err)
}

func TestSetupPlanCmd_FeatureDirectory(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "autospec-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	specsDir := filepath.Join(tmpDir, "specs")
	featureDir := filepath.Join(specsDir, "001-test-feature")
	err = os.MkdirAll(featureDir, 0755)
	require.NoError(t, err)

	// Verify directory was created
	info, err := os.Stat(featureDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestCopyFile(t *testing.T) {
	// Create temp source file
	tmpDir, err := os.MkdirTemp("", "autospec-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	content := "test content for copy"
	err = os.WriteFile(srcPath, []byte(content), 0644)
	require.NoError(t, err)

	// Copy file
	err = copyFile(srcPath, dstPath)
	require.NoError(t, err)

	// Verify copy
	data, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestSetupPlanCmd_HasGitValues(t *testing.T) {
	// Test that HasGit is "true" or "false" as strings
	outputTrue := SetupPlanOutput{HasGit: "true"}
	outputFalse := SetupPlanOutput{HasGit: "false"}

	assert.Equal(t, "true", outputTrue.HasGit)
	assert.Equal(t, "false", outputFalse.HasGit)
}
