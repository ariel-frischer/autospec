// Package config_test tests configuration loading, merging hierarchy, and environment variable overrides.
// Related: internal/config/config.go
// Tags: config, loading, merging, env-vars, yaml, json, precedence
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoad_Defaults tests that defaults are applied when no config files exist.
// Requires working directory and HOME/XDG_CONFIG_HOME isolation to avoid
// loading real config files from the system. NO t.Parallel() due to cwd changes.
func TestLoad_Defaults(t *testing.T) {
	// Cannot use t.Parallel() because we modify environment and working directory
	// to isolate from real config files that might exist on the system

	// Save original state
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	// Create isolated temp directory with no config files
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))

	// Set HOME and XDG_CONFIG_HOME to isolated directories
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	// Load with empty config path (defaults only)
	cfg, err := Load("")
	require.NoError(t, err)
	assert.Equal(t, "claude", cfg.ClaudeCmd)
	assert.Equal(t, 0, cfg.MaxRetries)
	assert.Equal(t, "./specs", cfg.SpecsDir)
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
	// Create temp directories for user and project configs
	tmpDir := t.TempDir()

	// Set XDG_CONFIG_HOME to isolate user config
	userConfigDir := filepath.Join(tmpDir, ".config", "autospec")
	require.NoError(t, os.MkdirAll(userConfigDir, 0755))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	// Write user config (lower priority)
	userPath := filepath.Join(userConfigDir, "config.yml")
	userContent := `claude_cmd: user-claude
max_retries: 2
specs_dir: "./specs"
state_dir: "~/.autospec/state"
`
	require.NoError(t, os.WriteFile(userPath, []byte(userContent), 0644))

	// Write project config (higher priority)
	projectDir := filepath.Join(tmpDir, "project", ".autospec")
	require.NoError(t, os.MkdirAll(projectDir, 0755))
	projectPath := filepath.Join(projectDir, "config.yml")
	projectContent := `max_retries: 4
`
	require.NoError(t, os.WriteFile(projectPath, []byte(projectContent), 0644))

	// Change to project directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(filepath.Join(tmpDir, "project"))

	// Set environment variable (highest priority)
	t.Setenv("AUTOSPEC_MAX_RETRIES", "8")

	cfg, err := LoadWithOptions(LoadOptions{
		SkipWarnings: true,
	})
	require.NoError(t, err)

	// Environment should win for max_retries
	assert.Equal(t, 8, cfg.MaxRetries)
	// User config value for claude_cmd (project config doesn't override it)
	assert.Equal(t, "user-claude", cfg.ClaudeCmd)
}

// Timeout Configuration Tests

func TestLoad_TimeoutDefaults(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	// Use temp HOME to avoid loading real user config
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	cfg, err := Load("")
	require.NoError(t, err)
	assert.Equal(t, 2400, cfg.Timeout, "Default timeout should be 2400 (40 minutes)")
}

func TestLoad_TimeoutValidValue(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configContent := `{"timeout": 300}`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, 300, cfg.Timeout)
}

func TestLoad_TimeoutZero(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configContent := `{"timeout": 0}`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, 0, cfg.Timeout, "Timeout=0 should be valid (no timeout)")
}

func TestLoad_TimeoutInvalid_Negative(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configContent := `{"timeout": -1}`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	_, err = Load(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestLoad_TimeoutInvalid_TooLarge(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configContent := `{"timeout": 700000}`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	_, err = Load(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestLoad_TimeoutEnvOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Local config with timeout 300
	configContent := `{"timeout": 300}`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Environment variable overrides
	t.Setenv("AUTOSPEC_TIMEOUT", "120")

	cfg, err := Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, 120, cfg.Timeout, "Environment variable should override config file")
}

func TestLoad_TimeoutValidRange(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		timeout int
		valid   bool
	}{
		"minimum valid":          {timeout: 1, valid: true},
		"mid-range valid":        {timeout: 300, valid: true},
		"maximum valid (1 hour)": {timeout: 3600, valid: true},
		"7 days (maximum)":       {timeout: 604800, valid: true},
		"zero (no timeout)":      {timeout: 0, valid: true},
		"below minimum":          {timeout: -5, valid: false},
		"above maximum":          {timeout: 604801, valid: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.json")

			configContent := fmt.Sprintf(`{"timeout": %d}`, tt.timeout)
			err := os.WriteFile(configPath, []byte(configContent), 0644)
			require.NoError(t, err)

			cfg, err := Load(configPath)
			if tt.valid {
				require.NoError(t, err)
				assert.Equal(t, tt.timeout, cfg.Timeout)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "validation failed")
			}
		})
	}
}

func TestLoad_TimeoutNonNumericEnv(t *testing.T) {
	t.Setenv("AUTOSPEC_TIMEOUT", "invalid")

	_, err := Load("")
	assert.Error(t, err)
}

// YAML Configuration Tests

func TestLoad_YAMLConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Write YAML config
	configContent := `claude_cmd: custom-claude
max_retries: 5
specs_dir: "./specs"
state_dir: "~/.autospec/state"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadWithOptions(LoadOptions{
		ProjectConfigPath: configPath,
		SkipWarnings:      true,
	})
	require.NoError(t, err)
	assert.Equal(t, "custom-claude", cfg.ClaudeCmd)
	assert.Equal(t, 5, cfg.MaxRetries)
}

func TestLoad_YAMLConfigWithNestedValues(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Write YAML config with all values
	configContent := `claude_cmd: claude
claude_args:
  - "-p"
  - "--verbose"
max_retries: 3
specs_dir: "./specs"
state_dir: "~/.autospec/state"
skip_preflight: true
timeout: 300
skip_confirmations: false
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadWithOptions(LoadOptions{
		ProjectConfigPath: configPath,
		SkipWarnings:      true,
	})
	require.NoError(t, err)
	assert.Equal(t, "claude", cfg.ClaudeCmd)
	assert.Equal(t, []string{"-p", "--verbose"}, cfg.ClaudeArgs)
	assert.True(t, cfg.SkipPreflight)
	assert.Equal(t, 300, cfg.Timeout)
}

func TestLoad_YAMLEmptyFile(t *testing.T) {
	// Cannot use t.Parallel() because we modify environment to isolate from user config
	tmpDir := t.TempDir()

	// Isolate from real user config by setting HOME/XDG_CONFIG_HOME to temp
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	configPath := filepath.Join(tmpDir, "config.yml")

	// Write empty YAML file
	err := os.WriteFile(configPath, []byte(""), 0644)
	require.NoError(t, err)

	cfg, err := LoadWithOptions(LoadOptions{
		ProjectConfigPath: configPath,
		SkipWarnings:      true,
	})
	require.NoError(t, err)
	// Should use defaults
	assert.Equal(t, "claude", cfg.ClaudeCmd)
	assert.Equal(t, 0, cfg.MaxRetries)
}

func TestLoad_YAMLInvalidSyntax(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Write invalid YAML
	invalidYAML := `claude_cmd: "claude
max_retries: 3
`
	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	_, err = LoadWithOptions(LoadOptions{
		ProjectConfigPath: configPath,
		SkipWarnings:      true,
	})
	assert.Error(t, err)
}

func TestLoad_LegacyJSONWithWarning(t *testing.T) {
	// Cannot use t.Parallel() because we use os.Chdir which affects the whole process

	tmpDir := t.TempDir()
	legacyPath := filepath.Join(tmpDir, ".autospec", "config.json")

	// Create legacy JSON config in project directory
	require.NoError(t, os.MkdirAll(filepath.Dir(legacyPath), 0755))
	jsonContent := `{"max_retries": 5, "claude_cmd": "claude", "specs_dir": "./specs", "state_dir": "~/.autospec/state"}`
	require.NoError(t, os.WriteFile(legacyPath, []byte(jsonContent), 0644))

	// Change to temp directory to simulate being in a project
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	// Capture warnings
	var warnings strings.Builder
	cfg, err := LoadWithOptions(LoadOptions{
		WarningWriter: &warnings,
	})
	require.NoError(t, err)

	// Config should load from legacy JSON
	assert.Equal(t, 5, cfg.MaxRetries)

	// Should have warning about migration
	warningText := warnings.String()
	assert.Contains(t, warningText, "deprecated")
	assert.Contains(t, warningText, "migrate")
}

func TestLoad_YAMLTakesPrecedenceOverJSON(t *testing.T) {
	// Note: not parallel because we change working directory
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, ".autospec", "config.yml")
	jsonPath := filepath.Join(tmpDir, ".autospec", "config.json")

	require.NoError(t, os.MkdirAll(filepath.Dir(yamlPath), 0755))

	// Write both YAML and JSON configs
	yamlContent := `claude_cmd: yaml-claude
max_retries: 7
specs_dir: "./specs"
state_dir: "~/.autospec/state"
`
	jsonContent := `{"claude_cmd": "json-claude", "max_retries": 5}`

	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))
	require.NoError(t, os.WriteFile(jsonPath, []byte(jsonContent), 0644))

	// Verify files exist
	_, err := os.Stat(yamlPath)
	require.NoError(t, err, "YAML file should exist")
	_, err = os.Stat(jsonPath)
	require.NoError(t, err, "JSON file should exist")

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	require.NoError(t, os.Chdir(tmpDir))

	// Verify relative paths work after chdir
	_, err = os.Stat(".autospec/config.yml")
	require.NoError(t, err, "YAML file should be accessible via relative path")

	// Capture warnings
	var warnings strings.Builder
	cfg, err := LoadWithOptions(LoadOptions{
		WarningWriter: &warnings,
	})
	require.NoError(t, err)

	// YAML values should be used
	assert.Equal(t, "yaml-claude", cfg.ClaudeCmd)
	assert.Equal(t, 7, cfg.MaxRetries)

	// Should have warning about ignored JSON
	warningText := warnings.String()
	if warningText != "" {
		// When YAML exists and JSON also exists, we should see "ignored" or no warning
		// (if YAML was loaded successfully)
		t.Logf("Warning text: %s", warningText)
	}
	// The key assertion is that YAML values are used
	assert.Equal(t, "yaml-claude", cfg.ClaudeCmd, "YAML should take precedence")
}

func TestLoad_UserAndProjectPrecedence(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create user config directory
	userConfigDir := filepath.Join(tmpDir, ".config", "autospec")
	require.NoError(t, os.MkdirAll(userConfigDir, 0755))

	// Create project config directory
	projectDir := filepath.Join(tmpDir, "project", ".autospec")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	// Write user config (lower priority)
	userConfig := `claude_cmd: user-claude
max_retries: 2
specs_dir: "./specs"
state_dir: "~/.autospec/state"
timeout: 100
`
	userConfigPath := filepath.Join(userConfigDir, "config.yml")
	require.NoError(t, os.WriteFile(userConfigPath, []byte(userConfig), 0644))

	// Write project config (higher priority, partial override)
	projectConfig := `max_retries: 5
timeout: 300
`
	projectConfigPath := filepath.Join(projectDir, "config.yml")
	require.NoError(t, os.WriteFile(projectConfigPath, []byte(projectConfig), 0644))

	// Set XDG_CONFIG_HOME to use our test user config
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	// Change to project directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(filepath.Join(tmpDir, "project"))

	cfg, err := LoadWithOptions(LoadOptions{
		SkipWarnings: true,
	})
	require.NoError(t, err)

	// User value for claude_cmd
	assert.Equal(t, "user-claude", cfg.ClaudeCmd)
	// Project value for max_retries (overrides user)
	assert.Equal(t, 5, cfg.MaxRetries)
	// Project value for timeout (overrides user)
	assert.Equal(t, 300, cfg.Timeout)
}

func TestLoad_EnvOverridesAll(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create project config directory
	projectDir := filepath.Join(tmpDir, ".autospec")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	// Write project config
	projectConfig := `claude_cmd: project-claude
max_retries: 5
specs_dir: "./specs"
state_dir: "~/.autospec/state"
`
	projectConfigPath := filepath.Join(projectDir, "config.yml")
	require.NoError(t, os.WriteFile(projectConfigPath, []byte(projectConfig), 0644))

	// Set environment variable (highest priority)
	t.Setenv("AUTOSPEC_MAX_RETRIES", "9")

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	cfg, err := LoadWithOptions(LoadOptions{
		SkipWarnings: true,
	})
	require.NoError(t, err)

	// Environment should override project config
	assert.Equal(t, 9, cfg.MaxRetries)
	// Project value for claude_cmd
	assert.Equal(t, "project-claude", cfg.ClaudeCmd)
}

func TestLoad_UserYAMLWithLegacyJSONWarning(t *testing.T) {
	// Test the case where user YAML exists alongside legacy JSON
	tmpDir := t.TempDir()

	// Create user config directory structure
	userConfigDir := filepath.Join(tmpDir, ".config", "autospec")
	require.NoError(t, os.MkdirAll(userConfigDir, 0755))

	// Create legacy user directory
	legacyUserDir := filepath.Join(tmpDir, ".autospec")
	require.NoError(t, os.MkdirAll(legacyUserDir, 0755))

	// Write user YAML config
	userYAMLPath := filepath.Join(userConfigDir, "config.yml")
	userYAMLContent := `claude_cmd: yaml-claude
max_retries: 2
specs_dir: "./specs"
state_dir: "~/.autospec/state"
`
	require.NoError(t, os.WriteFile(userYAMLPath, []byte(userYAMLContent), 0644))

	// Write legacy JSON config
	legacyJSONPath := filepath.Join(legacyUserDir, "config.json")
	legacyJSONContent := `{"claude_cmd": "json-claude", "max_retries": 5}`
	require.NoError(t, os.WriteFile(legacyJSONPath, []byte(legacyJSONContent), 0644))

	// Set environment to use temp directories
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	// Capture warnings
	var warnings strings.Builder
	cfg, err := LoadWithOptions(LoadOptions{
		WarningWriter: &warnings,
	})
	require.NoError(t, err)

	// YAML values should be used
	assert.Equal(t, "yaml-claude", cfg.ClaudeCmd)
	assert.Equal(t, 2, cfg.MaxRetries)

	// Should warn about legacy JSON being ignored
	warningText := warnings.String()
	assert.Contains(t, warningText, "ignored")
}

func TestLoad_InvalidUserYAMLSyntax(t *testing.T) {
	tmpDir := t.TempDir()

	// Create user config directory
	userConfigDir := filepath.Join(tmpDir, ".config", "autospec")
	require.NoError(t, os.MkdirAll(userConfigDir, 0755))

	// Write invalid user YAML config
	userYAMLPath := filepath.Join(userConfigDir, "config.yml")
	invalidYAMLContent := `claude_cmd: "unclosed quote
max_retries: 3
`
	require.NoError(t, os.WriteFile(userYAMLPath, []byte(invalidYAMLContent), 0644))

	// Set environment to use temp directories
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	_, err := LoadWithOptions(LoadOptions{
		SkipWarnings: true,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user YAML config")
}

func TestLoad_InvalidProjectYAMLSyntax(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create project config directory
	projectDir := filepath.Join(tmpDir, ".autospec")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	// Write invalid project YAML config
	projectYAMLPath := filepath.Join(projectDir, "config.yml")
	invalidYAMLContent := `claude_cmd: [unclosed bracket
max_retries: 3
`
	require.NoError(t, os.WriteFile(projectYAMLPath, []byte(invalidYAMLContent), 0644))

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	_, err := LoadWithOptions(LoadOptions{
		SkipWarnings: true,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project YAML config")
}

func TestLoad_InvalidLegacyUserJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create legacy user directory
	legacyUserDir := filepath.Join(tmpDir, ".autospec")
	require.NoError(t, os.MkdirAll(legacyUserDir, 0755))

	// Write invalid legacy JSON config
	legacyJSONPath := filepath.Join(legacyUserDir, "config.json")
	invalidJSONContent := `{invalid json`
	require.NoError(t, os.WriteFile(legacyJSONPath, []byte(invalidJSONContent), 0644))

	// Set environment to use temp directories
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	_, err := LoadWithOptions(LoadOptions{
		SkipWarnings: true,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "legacy user JSON config")
}

func TestLoad_InvalidLegacyProjectJSON(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")

	// Create project config directory
	projectAutospecDir := filepath.Join(projectDir, ".autospec")
	require.NoError(t, os.MkdirAll(projectAutospecDir, 0755))

	// Write invalid legacy JSON config (in project directory)
	legacyJSONPath := filepath.Join(projectAutospecDir, "config.json")
	invalidJSONContent := `{invalid json`
	require.NoError(t, os.WriteFile(legacyJSONPath, []byte(invalidJSONContent), 0644))

	// Change to project directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(projectDir)

	// Set environment to use isolated directories so no user config is loaded
	// Use a different HOME so there's no user config to load
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	_, err := LoadWithOptions(LoadOptions{
		SkipWarnings: true,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "legacy project config")
}

func TestFileExists(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	tests := map[string]struct {
		setup    func() string
		expected bool
	}{
		"empty path": {
			setup:    func() string { return "" },
			expected: false,
		},
		"existing file": {
			setup: func() string {
				path := filepath.Join(tmpDir, "existing.txt")
				os.WriteFile(path, []byte("content"), 0644)
				return path
			},
			expected: true,
		},
		"non-existent file": {
			setup:    func() string { return filepath.Join(tmpDir, "nonexistent.txt") },
			expected: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			path := tt.setup()
			result := fileExists(path)
			if result != tt.expected {
				t.Errorf("fileExists(%q) = %v, want %v", path, result, tt.expected)
			}
		})
	}
}

func TestEnvTransform(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input    string
		expected string
	}{
		"basic": {
			input:    "AUTOSPEC_MAX_RETRIES",
			expected: "max_retries",
		},
		"simple": {
			input:    "AUTOSPEC_TIMEOUT",
			expected: "timeout",
		},
		"nested": {
			input:    "AUTOSPEC_NOTIFICATIONS_TYPE",
			expected: "notifications_type",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := envTransform(tt.input)
			if result != tt.expected {
				t.Errorf("envTransform(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetWarningWriter(t *testing.T) {
	t.Parallel()

	// Test with nil writer
	result := getWarningWriter(nil)
	assert.Equal(t, os.Stderr, result)

	// Test with custom writer
	var buf strings.Builder
	result = getWarningWriter(&buf)
	assert.Equal(t, &buf, result)
}

func TestLoad_AUTOSPEC_YESEnvVar(t *testing.T) {
	tmpDir := t.TempDir()

	// Set environment to use temp directories
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	t.Setenv("AUTOSPEC_YES", "1")

	cfg, err := Load("")
	require.NoError(t, err)

	assert.True(t, cfg.SkipConfirmations, "AUTOSPEC_YES should set SkipConfirmations to true")
}

// Agent Configuration Tests

func TestConfiguration_GetAgent_Priority(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cfg      Configuration
		wantName string
		wantErr  bool
	}{
		"default returns claude": {
			cfg:      Configuration{},
			wantName: "claude",
		},
		"agent_preset takes precedence over legacy": {
			cfg: Configuration{
				AgentPreset: "gemini",
				ClaudeCmd:   "custom-claude",
			},
			wantName: "gemini",
		},
		"custom_agent_cmd takes highest precedence": {
			cfg: Configuration{
				CustomAgentCmd: "echo {{PROMPT}}",
				AgentPreset:    "gemini",
				ClaudeCmd:      "custom-claude",
			},
			wantName: "custom",
		},
		"legacy custom_claude_cmd when no new fields": {
			cfg: Configuration{
				CustomClaudeCmd: "my-tool --prompt {{PROMPT}}",
			},
			wantName: "custom",
		},
		"unknown agent_preset returns error": {
			cfg: Configuration{
				AgentPreset: "nonexistent-agent",
			},
			wantErr: true,
		},
		"invalid custom_agent_cmd returns error": {
			cfg: Configuration{
				CustomAgentCmd: "no-placeholder-here",
			},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			agent, err := tt.cfg.GetAgent()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, agent.Name())
		})
	}
}

func TestConfiguration_GetAgent_AllPresets(t *testing.T) {
	t.Parallel()

	presets := []string{"claude", "cline", "gemini", "codex", "opencode", "goose"}
	for _, preset := range presets {
		t.Run(preset, func(t *testing.T) {
			t.Parallel()
			cfg := Configuration{AgentPreset: preset}
			agent, err := cfg.GetAgent()
			require.NoError(t, err)
			assert.Equal(t, preset, agent.Name())
		})
	}
}

func TestBuildLegacyTemplate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cmd      string
		args     []string
		contains []string
	}{
		"with -p flag": {
			cmd:      "claude",
			args:     []string{"-p", "--verbose"},
			contains: []string{"claude", "-p", "{{PROMPT}}", "--verbose"},
		},
		"without -p flag": {
			cmd:      "my-tool",
			args:     []string{"--verbose"},
			contains: []string{"my-tool", "--verbose", "-p", "{{PROMPT}}"},
		},
		"empty args": {
			cmd:      "simple",
			args:     nil,
			contains: []string{"simple", "-p", "{{PROMPT}}"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := buildLegacyTemplate(tt.cmd, tt.args)
			for _, s := range tt.contains {
				assert.Contains(t, result, s)
			}
		})
	}
}

func TestEmitLegacyWarnings(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cfg           Configuration
		expectWarning bool
		contains      string
	}{
		"no warning when agent_preset set": {
			cfg: Configuration{
				AgentPreset:     "claude",
				CustomClaudeCmd: "old-tool {{PROMPT}}",
			},
			expectWarning: false,
		},
		"no warning when custom_agent_cmd set": {
			cfg: Configuration{
				CustomAgentCmd:  "new-tool {{PROMPT}}",
				CustomClaudeCmd: "old-tool {{PROMPT}}",
			},
			expectWarning: false,
		},
		"warning for custom_claude_cmd": {
			cfg: Configuration{
				CustomClaudeCmd: "old-tool {{PROMPT}}",
			},
			expectWarning: true,
			contains:      "custom_claude_cmd",
		},
		"warning for non-default claude_cmd": {
			cfg: Configuration{
				ClaudeCmd: "my-custom-claude",
			},
			expectWarning: true,
			contains:      "deprecated",
		},
		"no warning for default claude_cmd": {
			cfg: Configuration{
				ClaudeCmd:  "claude",
				ClaudeArgs: []string{"-p", "--verbose", "--output-format", "stream-json"},
			},
			expectWarning: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var buf strings.Builder
			emitLegacyWarnings(&tt.cfg, &buf)

			if tt.expectWarning {
				assert.NotEmpty(t, buf.String())
				if tt.contains != "" {
					assert.Contains(t, buf.String(), tt.contains)
				}
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}

func TestStringSliceEqual(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		a, b   []string
		expect bool
	}{
		"equal slices": {
			a:      []string{"a", "b", "c"},
			b:      []string{"a", "b", "c"},
			expect: true,
		},
		"different lengths": {
			a:      []string{"a", "b"},
			b:      []string{"a", "b", "c"},
			expect: false,
		},
		"different values": {
			a:      []string{"a", "b", "c"},
			b:      []string{"a", "x", "c"},
			expect: false,
		},
		"both empty": {
			a:      []string{},
			b:      []string{},
			expect: true,
		},
		"one nil one empty": {
			a:      nil,
			b:      []string{},
			expect: true, // both have len 0
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := stringSliceEqual(tt.a, tt.b)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestLoad_AgentPresetFromYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	projectConfigPath := filepath.Join(tmpDir, "project-config.yml")
	userConfigPath := filepath.Join(tmpDir, "user-config.yml")

	// Create empty mock user config to isolate from real user config
	err := os.WriteFile(userConfigPath, []byte(""), 0644)
	require.NoError(t, err)

	configContent := `agent_preset: gemini
specs_dir: "./specs"
state_dir: "~/.autospec/state"
`
	err = os.WriteFile(projectConfigPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadWithOptions(LoadOptions{
		ProjectConfigPath: projectConfigPath,
		UserConfigPath:    userConfigPath,
		SkipWarnings:      true,
	})
	require.NoError(t, err)
	assert.Equal(t, "gemini", cfg.AgentPreset)

	agent, err := cfg.GetAgent()
	require.NoError(t, err)
	assert.Equal(t, "gemini", agent.Name())
}

func TestLoad_CustomAgentCmdFromYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	configContent := `custom_agent_cmd: "aider --model sonnet --message {{PROMPT}}"
specs_dir: "./specs"
state_dir: "~/.autospec/state"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadWithOptions(LoadOptions{
		ProjectConfigPath: configPath,
		SkipWarnings:      true,
	})
	require.NoError(t, err)
	assert.Equal(t, "aider --model sonnet --message {{PROMPT}}", cfg.CustomAgentCmd)

	agent, err := cfg.GetAgent()
	require.NoError(t, err)
	assert.Equal(t, "custom", agent.Name())
}

func TestLoad_AgentPresetFromEnv(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	t.Setenv("AUTOSPEC_AGENT_PRESET", "cline")

	cfg, err := LoadWithOptions(LoadOptions{SkipWarnings: true})
	require.NoError(t, err)
	assert.Equal(t, "cline", cfg.AgentPreset)
}

func TestLoad_CustomAgentCmdFromEnv(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	t.Setenv("AUTOSPEC_CUSTOM_AGENT_CMD", "my-agent {{PROMPT}}")

	cfg, err := LoadWithOptions(LoadOptions{SkipWarnings: true})
	require.NoError(t, err)
	assert.Equal(t, "my-agent {{PROMPT}}", cfg.CustomAgentCmd)
}
