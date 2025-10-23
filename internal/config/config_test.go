package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	t.Parallel()

	// Load with empty config path (defaults only)
	cfg, err := Load("")
	require.NoError(t, err)
	assert.Equal(t, "claude", cfg.ClaudeCmd)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, "./specs", cfg.SpecsDir)
	assert.False(t, cfg.UseAPIKey)
}

func TestLoad_LocalOverride(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write local config
	configContent := `{
		"max_retries": 5,
		"claude_cmd": "custom-claude"
	}`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, "custom-claude", cfg.ClaudeCmd)
	assert.Equal(t, 5, cfg.MaxRetries)
	assert.Equal(t, "specify", cfg.SpecifyCmd) // Default value preserved
}

func TestLoad_EnvOverride(t *testing.T) {
	// Set environment variable
	t.Setenv("AUTOSPEC_MAX_RETRIES", "7")
	t.Setenv("AUTOSPEC_CLAUDE_CMD", "env-claude")

	cfg, err := Load("")
	require.NoError(t, err)
	assert.Equal(t, "env-claude", cfg.ClaudeCmd)
	assert.Equal(t, 7, cfg.MaxRetries)
}

func TestLoad_ValidationError_MaxRetriesOutOfRange(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write invalid config (max_retries > 10)
	configContent := `{"max_retries": 15}`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	_, err = Load(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestLoad_ValidationError_CustomClaudeCmd(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write invalid custom_claude_cmd (missing {{PROMPT}})
	configContent := `{"custom_claude_cmd": "claude --no-prompt"}`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	_, err = Load(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "{{PROMPT}}")
}

func TestExpandHomePath(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input    string
		contains string
	}{
		"tilde prefix": {
			input:    "~/.autospec/state",
			contains: ".autospec/state",
		},
		"absolute path": {
			input:    "/absolute/path",
			contains: "/absolute/path",
		},
		"relative path": {
			input:    "./relative/path",
			contains: "./relative/path",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := expandHomePath(tc.input)
			assert.Contains(t, result, tc.contains)
		})
	}
}

func TestLoad_OverridePrecedence(t *testing.T) {
	// Create temp directories for global and local configs
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, ".autospec")
	require.NoError(t, os.MkdirAll(globalDir, 0755))

	// Temporarily change HOME to use our test directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Write global config
	globalPath := filepath.Join(globalDir, "config.json")
	globalContent := `{"max_retries": 2, "claude_cmd": "global-claude"}`
	require.NoError(t, os.WriteFile(globalPath, []byte(globalContent), 0644))

	// Write local config
	localPath := filepath.Join(tmpDir, "local-config.json")
	localContent := `{"max_retries": 4}`
	require.NoError(t, os.WriteFile(localPath, []byte(localContent), 0644))

	// Set environment variable (highest priority)
	t.Setenv("AUTOSPEC_MAX_RETRIES", "8")

	cfg, err := Load(localPath)
	require.NoError(t, err)

	// Environment should win
	assert.Equal(t, 8, cfg.MaxRetries)
	// Local should override global
	// (We can't easily test this without claude_cmd from local, so we accept global value)
	assert.Equal(t, "global-claude", cfg.ClaudeCmd)
}
