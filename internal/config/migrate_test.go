package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateJSONToYAML_Success(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "config.json")
	yamlPath := filepath.Join(tmpDir, "config.yml")

	// Write JSON config
	jsonContent := `{
		"claude_cmd": "claude",
		"max_retries": 5,
		"specs_dir": "./specs"
	}`
	require.NoError(t, os.WriteFile(jsonPath, []byte(jsonContent), 0644))

	result, err := MigrateJSONToYAML(jsonPath, yamlPath, false)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Message, "Migrated")

	// Verify YAML was created
	yamlData, err := os.ReadFile(yamlPath)
	require.NoError(t, err)
	assert.Contains(t, string(yamlData), "claude_cmd")
	assert.Contains(t, string(yamlData), "max_retries: 5")
}

func TestMigrateJSONToYAML_DryRun(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "config.json")
	yamlPath := filepath.Join(tmpDir, "config.yml")

	// Write JSON config
	jsonContent := `{"claude_cmd": "claude", "max_retries": 3}`
	require.NoError(t, os.WriteFile(jsonPath, []byte(jsonContent), 0644))

	result, err := MigrateJSONToYAML(jsonPath, yamlPath, true)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, result.DryRun)
	assert.Contains(t, result.Message, "Would migrate")

	// Verify YAML was NOT created
	_, err = os.Stat(yamlPath)
	assert.True(t, os.IsNotExist(err))
}

func TestMigrateJSONToYAML_NoJSONFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "nonexistent.json")
	yamlPath := filepath.Join(tmpDir, "config.yml")

	result, err := MigrateJSONToYAML(jsonPath, yamlPath, false)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Message, "No JSON config found")
}

func TestMigrateJSONToYAML_YAMLAlreadyExists(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "config.json")
	yamlPath := filepath.Join(tmpDir, "config.yml")

	// Write both JSON and YAML
	require.NoError(t, os.WriteFile(jsonPath, []byte(`{"max_retries": 5}`), 0644))
	require.NoError(t, os.WriteFile(yamlPath, []byte("max_retries: 3"), 0644))

	result, err := MigrateJSONToYAML(jsonPath, yamlPath, false)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Message, "already exists")

	// Verify YAML was not overwritten
	yamlData, err := os.ReadFile(yamlPath)
	require.NoError(t, err)
	assert.Contains(t, string(yamlData), "max_retries: 3")
}

func TestMigrateJSONToYAML_InvalidJSON(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "config.json")
	yamlPath := filepath.Join(tmpDir, "config.yml")

	// Write invalid JSON
	require.NoError(t, os.WriteFile(jsonPath, []byte(`{invalid json`), 0644))

	_, err := MigrateJSONToYAML(jsonPath, yamlPath, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse JSON")
}

func TestMigrateJSONToYAML_PreservesAllFields(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "config.json")
	yamlPath := filepath.Join(tmpDir, "config.yml")

	// Write full JSON config
	jsonContent := `{
		"claude_cmd": "custom-claude",
		"claude_args": ["-p", "--verbose"],
		"use_api_key": true,
		"custom_claude_cmd": "wrapper {{PROMPT}}",
		"max_retries": 5,
		"specs_dir": "./custom/specs",
		"state_dir": "~/.custom/state",
		"skip_preflight": true,
		"timeout": 600,
		"show_progress": true,
		"skip_confirmations": true
	}`
	require.NoError(t, os.WriteFile(jsonPath, []byte(jsonContent), 0644))

	result, err := MigrateJSONToYAML(jsonPath, yamlPath, false)
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Verify all fields are in YAML
	yamlData, err := os.ReadFile(yamlPath)
	require.NoError(t, err)
	yamlStr := string(yamlData)

	assert.Contains(t, yamlStr, "claude_cmd: custom-claude")
	assert.Contains(t, yamlStr, "max_retries: 5")
	assert.Contains(t, yamlStr, "use_api_key: true")
	assert.Contains(t, yamlStr, "timeout: 600")
	assert.Contains(t, yamlStr, "skip_preflight: true")
}

func TestRemoveLegacyConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "config.json")

	// Create JSON file
	require.NoError(t, os.WriteFile(jsonPath, []byte(`{}`), 0644))

	err := RemoveLegacyConfig(jsonPath, false)
	require.NoError(t, err)

	// Original should not exist
	_, err = os.Stat(jsonPath)
	assert.True(t, os.IsNotExist(err))

	// Backup should exist
	_, err = os.Stat(jsonPath + ".bak")
	assert.NoError(t, err)
}

func TestRemoveLegacyConfig_DryRun(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "config.json")

	// Create JSON file
	require.NoError(t, os.WriteFile(jsonPath, []byte(`{}`), 0644))

	err := RemoveLegacyConfig(jsonPath, true)
	require.NoError(t, err)

	// Original should still exist
	_, err = os.Stat(jsonPath)
	assert.NoError(t, err)
}

func TestRemoveLegacyConfig_NonExistent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "nonexistent.json")

	err := RemoveLegacyConfig(jsonPath, false)
	require.NoError(t, err) // Should not error
}

func TestDetectLegacyConfigs(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create legacy user config
	legacyUserDir := filepath.Join(tmpDir, ".autospec")
	require.NoError(t, os.MkdirAll(legacyUserDir, 0755))
	legacyUserPath := filepath.Join(legacyUserDir, "config.json")
	require.NoError(t, os.WriteFile(legacyUserPath, []byte(`{}`), 0644))

	// Temporarily change HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Create legacy project config
	projectDir := filepath.Join(tmpDir, "project", ".autospec")
	require.NoError(t, os.MkdirAll(projectDir, 0755))
	projectPath := filepath.Join(projectDir, "config.json")
	require.NoError(t, os.WriteFile(projectPath, []byte(`{}`), 0644))

	// Change to project directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(filepath.Join(tmpDir, "project"))

	userJSON, projectJSON, err := DetectLegacyConfigs()
	require.NoError(t, err)

	assert.True(t, strings.HasSuffix(userJSON, "config.json"))
	assert.True(t, strings.HasSuffix(projectJSON, "config.json"))
}
