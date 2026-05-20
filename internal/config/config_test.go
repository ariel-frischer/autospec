// Package config_test tests configuration loading, merging hierarchy, and environment variable overrides.
// Related: internal/config/config.go
// Tags: config, loading, merging, env-vars, yaml, json, precedence
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/cliagent"
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
	assert.Equal(t, "", cfg.AgentPreset) // Default: use default claude agent
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
		"agent_preset": "claude"
	}`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, "claude", cfg.AgentPreset)
	assert.Equal(t, 5, cfg.MaxRetries)
}

func TestLoad_EnvOverride(t *testing.T) {
	// Set environment variable
	t.Setenv("AUTOSPEC_MAX_RETRIES", "7")
	t.Setenv("AUTOSPEC_AGENT_PRESET", "gemini")

	cfg, err := Load("")
	require.NoError(t, err)
	assert.Equal(t, "gemini", cfg.AgentPreset)
	assert.Equal(t, 7, cfg.MaxRetries)
}

func TestLoad_ValidationError_MaxRetriesOutOfRange(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write invalid config (max_retries > 10)
	configContent := `{"max_retries": 15}`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	_, err = Load(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
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
	require.NoError(t, os.MkdirAll(userConfigDir, 0o755))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	// Write user config (lower priority)
	userPath := filepath.Join(userConfigDir, "config.yml")
	userContent := `agent_preset: gemini
max_retries: 2
specs_dir: "./specs"
state_dir: "~/.autospec/state"
`
	require.NoError(t, os.WriteFile(userPath, []byte(userContent), 0o644))

	// Write project config (higher priority)
	projectDir := filepath.Join(tmpDir, "project", ".autospec")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	projectPath := filepath.Join(projectDir, "config.yml")
	projectContent := `max_retries: 4
`
	require.NoError(t, os.WriteFile(projectPath, []byte(projectContent), 0o644))

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
	// User config value for agent_preset (project config doesn't override it)
	assert.Equal(t, "gemini", cfg.AgentPreset)
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
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
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
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
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
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
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
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
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
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
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
			err := os.WriteFile(configPath, []byte(configContent), 0o644)
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
	configContent := `agent_preset: claude
max_retries: 5
specs_dir: "./specs"
state_dir: "~/.autospec/state"
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	cfg, err := LoadWithOptions(LoadOptions{
		ProjectConfigPath: configPath,
		SkipWarnings:      true,
	})
	require.NoError(t, err)
	assert.Equal(t, "claude", cfg.AgentPreset)
	assert.Equal(t, 5, cfg.MaxRetries)
}

func TestLoad_YAMLConfigWithNestedValues(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Write YAML config with all values
	configContent := `agent_preset: gemini
max_retries: 3
specs_dir: "./specs"
state_dir: "~/.autospec/state"
skip_preflight: true
timeout: 300
skip_confirmations: false
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	cfg, err := LoadWithOptions(LoadOptions{
		ProjectConfigPath: configPath,
		SkipWarnings:      true,
	})
	require.NoError(t, err)
	assert.Equal(t, "gemini", cfg.AgentPreset)
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
	err := os.WriteFile(configPath, []byte(""), 0o644)
	require.NoError(t, err)

	cfg, err := LoadWithOptions(LoadOptions{
		ProjectConfigPath: configPath,
		SkipWarnings:      true,
	})
	require.NoError(t, err)
	// Should use defaults
	assert.Equal(t, "", cfg.AgentPreset) // Empty preset means use default claude agent
	assert.Equal(t, 0, cfg.MaxRetries)
}

func TestLoad_YAMLInvalidSyntax(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Write invalid YAML (unclosed quote)
	invalidYAML := `agent_preset: "claude
max_retries: 3
`
	err := os.WriteFile(configPath, []byte(invalidYAML), 0o644)
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
	require.NoError(t, os.MkdirAll(filepath.Dir(legacyPath), 0o755))
	jsonContent := `{"max_retries": 5, "claude_cmd": "claude", "specs_dir": "./specs", "state_dir": "~/.autospec/state"}`
	require.NoError(t, os.WriteFile(legacyPath, []byte(jsonContent), 0o644))

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

	require.NoError(t, os.MkdirAll(filepath.Dir(yamlPath), 0o755))

	// Write both YAML and JSON configs
	yamlContent := `agent_preset: gemini
max_retries: 7
specs_dir: "./specs"
state_dir: "~/.autospec/state"
`
	jsonContent := `{"agent_preset": "claude", "max_retries": 5}`

	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0o644))
	require.NoError(t, os.WriteFile(jsonPath, []byte(jsonContent), 0o644))

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
	assert.Equal(t, "gemini", cfg.AgentPreset)
	assert.Equal(t, 7, cfg.MaxRetries)

	// Should have warning about ignored JSON
	warningText := warnings.String()
	if warningText != "" {
		// When YAML exists and JSON also exists, we should see "ignored" or no warning
		// (if YAML was loaded successfully)
		t.Logf("Warning text: %s", warningText)
	}
	// The key assertion is that YAML values are used
	assert.Equal(t, "gemini", cfg.AgentPreset, "YAML should take precedence")
}

func TestLoad_UserAndProjectPrecedence(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create user config directory
	userConfigDir := filepath.Join(tmpDir, ".config", "autospec")
	require.NoError(t, os.MkdirAll(userConfigDir, 0o755))

	// Create project config directory
	projectDir := filepath.Join(tmpDir, "project", ".autospec")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	// Write user config (lower priority)
	userConfig := `agent_preset: gemini
max_retries: 2
specs_dir: "./specs"
state_dir: "~/.autospec/state"
timeout: 100
`
	userConfigPath := filepath.Join(userConfigDir, "config.yml")
	require.NoError(t, os.WriteFile(userConfigPath, []byte(userConfig), 0o644))

	// Write project config (higher priority, partial override)
	projectConfig := `max_retries: 5
timeout: 300
`
	projectConfigPath := filepath.Join(projectDir, "config.yml")
	require.NoError(t, os.WriteFile(projectConfigPath, []byte(projectConfig), 0o644))

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

	// User value for agent_preset
	assert.Equal(t, "gemini", cfg.AgentPreset)
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
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	// Write project config
	projectConfig := `agent_preset: claude
max_retries: 5
specs_dir: "./specs"
state_dir: "~/.autospec/state"
`
	projectConfigPath := filepath.Join(projectDir, "config.yml")
	require.NoError(t, os.WriteFile(projectConfigPath, []byte(projectConfig), 0o644))

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
	// Project value for agent_preset
	assert.Equal(t, "claude", cfg.AgentPreset)
}

func TestLoad_UserYAMLWithLegacyJSONWarning(t *testing.T) {
	// Test the case where user YAML exists alongside legacy JSON
	tmpDir := t.TempDir()

	// Create user config directory structure
	userConfigDir := filepath.Join(tmpDir, ".config", "autospec")
	require.NoError(t, os.MkdirAll(userConfigDir, 0o755))

	// Create legacy user directory
	legacyUserDir := filepath.Join(tmpDir, ".autospec")
	require.NoError(t, os.MkdirAll(legacyUserDir, 0o755))

	// Write user YAML config
	userYAMLPath := filepath.Join(userConfigDir, "config.yml")
	userYAMLContent := `agent_preset: gemini
max_retries: 2
specs_dir: "./specs"
state_dir: "~/.autospec/state"
`
	require.NoError(t, os.WriteFile(userYAMLPath, []byte(userYAMLContent), 0o644))

	// Write legacy JSON config
	legacyJSONPath := filepath.Join(legacyUserDir, "config.json")
	legacyJSONContent := `{"agent_preset": "claude", "max_retries": 5}`
	require.NoError(t, os.WriteFile(legacyJSONPath, []byte(legacyJSONContent), 0o644))

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
	assert.Equal(t, "gemini", cfg.AgentPreset)
	assert.Equal(t, 2, cfg.MaxRetries)

	// Should warn about legacy JSON being ignored
	warningText := warnings.String()
	assert.Contains(t, warningText, "ignored")
}

func TestLoad_InvalidUserYAMLSyntax(t *testing.T) {
	tmpDir := t.TempDir()

	// Create user config directory
	userConfigDir := filepath.Join(tmpDir, ".config", "autospec")
	require.NoError(t, os.MkdirAll(userConfigDir, 0o755))

	// Write invalid user YAML config
	userYAMLPath := filepath.Join(userConfigDir, "config.yml")
	invalidYAMLContent := `agent_preset: "unclosed quote
max_retries: 3
`
	require.NoError(t, os.WriteFile(userYAMLPath, []byte(invalidYAMLContent), 0o644))

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
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	// Write invalid project YAML config
	projectYAMLPath := filepath.Join(projectDir, "config.yml")
	invalidYAMLContent := `claude_cmd: [unclosed bracket
max_retries: 3
`
	require.NoError(t, os.WriteFile(projectYAMLPath, []byte(invalidYAMLContent), 0o644))

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
	require.NoError(t, os.MkdirAll(legacyUserDir, 0o755))

	// Write invalid legacy JSON config
	legacyJSONPath := filepath.Join(legacyUserDir, "config.json")
	invalidJSONContent := `{invalid json`
	require.NoError(t, os.WriteFile(legacyJSONPath, []byte(invalidJSONContent), 0o644))

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
	require.NoError(t, os.MkdirAll(projectAutospecDir, 0o755))

	// Write invalid legacy JSON config (in project directory)
	legacyJSONPath := filepath.Join(projectAutospecDir, "config.json")
	invalidJSONContent := `{invalid json`
	require.NoError(t, os.WriteFile(legacyJSONPath, []byte(invalidJSONContent), 0o644))

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
				os.WriteFile(path, []byte("content"), 0o644)
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
		"nested notifications": {
			input:    "AUTOSPEC_NOTIFICATIONS_TYPE",
			expected: "notifications.type",
		},
		"nested notifications enabled": {
			input:    "AUTOSPEC_NOTIFICATIONS_ENABLED",
			expected: "notifications.enabled",
		},
		"nested cclean verbose": {
			input:    "AUTOSPEC_CCLEAN_VERBOSE",
			expected: "cclean.verbose",
		},
		"nested cclean style": {
			input:    "AUTOSPEC_CCLEAN_STYLE",
			expected: "cclean.style",
		},
		"nested cclean line_numbers": {
			input:    "AUTOSPEC_CCLEAN_LINE_NUMBERS",
			expected: "cclean.line_numbers",
		},
		"nested worktree base_dir": {
			input:    "AUTOSPEC_WORKTREE_BASE_DIR",
			expected: "worktree.base_dir",
		},
		"nested custom_agent command": {
			input:    "AUTOSPEC_CUSTOM_AGENT_COMMAND",
			expected: "custom_agent.command",
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
		"agent_preset gemini": {
			cfg: Configuration{
				AgentPreset: "gemini",
			},
			wantName: "gemini",
		},
		"custom_agent takes highest precedence": {
			cfg: Configuration{
				CustomAgent: &cliagent.CustomAgentConfig{
					Command: "echo",
					Args:    []string{"{{PROMPT}}"},
				},
				AgentPreset: "gemini",
			},
			wantName: "custom",
		},
		"unknown agent_preset returns error": {
			cfg: Configuration{
				AgentPreset: "nonexistent-agent",
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

func TestLoad_AgentPresetFromYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	projectConfigPath := filepath.Join(tmpDir, "project-config.yml")
	userConfigPath := filepath.Join(tmpDir, "user-config.yml")

	// Create empty mock user config to isolate from real user config
	err := os.WriteFile(userConfigPath, []byte(""), 0o644)
	require.NoError(t, err)

	configContent := `agent_preset: gemini
specs_dir: "./specs"
state_dir: "~/.autospec/state"
`
	err = os.WriteFile(projectConfigPath, []byte(configContent), 0o644)
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

func TestLoad_CustomAgentFromYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	configContent := `custom_agent:
  command: aider
  args:
    - "--model"
    - "sonnet"
    - "--message"
    - "{{PROMPT}}"
specs_dir: "./specs"
state_dir: "~/.autospec/state"
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	cfg, err := LoadWithOptions(LoadOptions{
		ProjectConfigPath: configPath,
		SkipWarnings:      true,
	})
	require.NoError(t, err)
	require.NotNil(t, cfg.CustomAgent)
	assert.Equal(t, "aider", cfg.CustomAgent.Command)

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

// EnableRiskAssessment Configuration Tests

func TestLoad_EnableRiskAssessmentDefaults(t *testing.T) {
	// Cannot use t.Parallel() due to environment modification
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	cfg, err := Load("")
	require.NoError(t, err)
	assert.False(t, cfg.EnableRiskAssessment, "EnableRiskAssessment should default to false")
}

func TestLoad_EnableRiskAssessmentExplicitValues(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		configContent string
		expected      bool
	}{
		"explicit true": {
			configContent: `enable_risk_assessment: true`,
			expected:      true,
		},
		"explicit false": {
			configContent: `enable_risk_assessment: false`,
			expected:      false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yml")
			err := os.WriteFile(configPath, []byte(tt.configContent), 0o644)
			require.NoError(t, err)

			cfg, err := LoadWithOptions(LoadOptions{
				ProjectConfigPath: configPath,
				SkipWarnings:      true,
			})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.EnableRiskAssessment)
		})
	}
}

func TestLoad_EnableRiskAssessmentEnvOverride(t *testing.T) {
	// Cannot use t.Parallel() due to environment modification
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Project config sets false
	configContent := `enable_risk_assessment: false`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Environment variable overrides
	t.Setenv("AUTOSPEC_ENABLE_RISK_ASSESSMENT", "true")

	cfg, err := LoadWithOptions(LoadOptions{
		ProjectConfigPath: configPath,
		SkipWarnings:      true,
	})
	require.NoError(t, err)
	assert.True(t, cfg.EnableRiskAssessment, "Environment variable should override config file")
}

func TestLoad_EnableRiskAssessmentPrecedence(t *testing.T) {
	// Test config priority: env > project > user > defaults
	tmpDir := t.TempDir()

	// Create user config directory
	userConfigDir := filepath.Join(tmpDir, ".config", "autospec")
	require.NoError(t, os.MkdirAll(userConfigDir, 0o755))

	// Write user config (enable_risk_assessment: true)
	userConfig := `enable_risk_assessment: true`
	userConfigPath := filepath.Join(userConfigDir, "config.yml")
	require.NoError(t, os.WriteFile(userConfigPath, []byte(userConfig), 0o644))

	// Create project config directory
	projectDir := filepath.Join(tmpDir, "project", ".autospec")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	// Write project config (enable_risk_assessment: false - overrides user)
	projectConfig := `enable_risk_assessment: false`
	projectConfigPath := filepath.Join(projectDir, "config.yml")
	require.NoError(t, os.WriteFile(projectConfigPath, []byte(projectConfig), 0o644))

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

	// Project config (false) should override user config (true)
	assert.False(t, cfg.EnableRiskAssessment, "Project config should override user config")
}

// ToMap Tests - ensures config show includes all fields automatically

func TestConfiguration_ToMap_IncludesAllKoanfTaggedFields(t *testing.T) {
	t.Parallel()

	cfg := &Configuration{
		AgentPreset:          "claude",
		MaxRetries:           3,
		EnableRiskAssessment: true,
	}

	m := cfg.ToMap()

	// Verify known fields are included
	assert.Equal(t, "claude", m["agent_preset"])
	assert.Equal(t, 3, m["max_retries"])
	assert.Equal(t, true, m["enable_risk_assessment"])
}

func TestConfiguration_ToMap_ExcludesInternalFields(t *testing.T) {
	t.Parallel()

	cfg := &Configuration{
		AutoCommitSource: SourceProject, // koanf:"-" field
	}

	m := cfg.ToMap()

	// Internal fields with koanf:"-" should be excluded
	_, hasAutoCommitSource := m["auto_commit_source"]
	assert.False(t, hasAutoCommitSource, "Fields with koanf:\"-\" should be excluded from ToMap")
}

func TestConfiguration_ToMap_FieldCountMatchesStruct(t *testing.T) {
	t.Parallel()

	cfg := &Configuration{}
	m := cfg.ToMap()

	// Count struct fields with koanf tags (excluding "-")
	v := reflect.TypeOf(*cfg)
	expectedCount := 0
	for i := 0; i < v.NumField(); i++ {
		tag := v.Field(i).Tag.Get("koanf")
		if tag != "" && tag != "-" {
			expectedCount++
		}
	}

	assert.Equal(t, expectedCount, len(m),
		"ToMap should return exactly the number of koanf-tagged fields (excluding '-')")
}

// Cclean Configuration Tests

func TestLoad_CcleanConfigDefaults(t *testing.T) {
	// Cannot use t.Parallel() due to environment modification
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	cfg, err := Load("")
	require.NoError(t, err)

	// Verify default values
	assert.False(t, cfg.Cclean.Verbose, "Cclean.Verbose should default to false")
	assert.False(t, cfg.Cclean.LineNumbers, "Cclean.LineNumbers should default to false")
	assert.Equal(t, "default", cfg.Cclean.Style, "Cclean.Style should default to 'default'")
}

func TestLoad_CcleanConfigFromYAML(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		configContent string
		wantVerbose   bool
		wantLineNum   bool
		wantStyle     string
	}{
		"all options set": {
			configContent: `cclean:
  verbose: true
  line_numbers: true
  style: compact
`,
			wantVerbose: true,
			wantLineNum: true,
			wantStyle:   "compact",
		},
		"verbose only": {
			configContent: `cclean:
  verbose: true
`,
			wantVerbose: true,
			wantLineNum: false,
			wantStyle:   "default",
		},
		"line_numbers only": {
			configContent: `cclean:
  line_numbers: true
`,
			wantVerbose: false,
			wantLineNum: true,
			wantStyle:   "default",
		},
		"style minimal": {
			configContent: `cclean:
  style: minimal
`,
			wantVerbose: false,
			wantLineNum: false,
			wantStyle:   "minimal",
		},
		"style plain": {
			configContent: `cclean:
  style: plain
`,
			wantVerbose: false,
			wantLineNum: false,
			wantStyle:   "plain",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yml")
			err := os.WriteFile(configPath, []byte(tt.configContent), 0o644)
			require.NoError(t, err)

			cfg, err := LoadWithOptions(LoadOptions{
				ProjectConfigPath: configPath,
				SkipWarnings:      true,
			})
			require.NoError(t, err)

			assert.Equal(t, tt.wantVerbose, cfg.Cclean.Verbose)
			assert.Equal(t, tt.wantLineNum, cfg.Cclean.LineNumbers)
			assert.Equal(t, tt.wantStyle, cfg.Cclean.Style)
		})
	}
}

func TestLoad_CcleanConfigFromEnv(t *testing.T) {
	// Cannot use t.Parallel() due to environment modification
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	t.Setenv("AUTOSPEC_CCLEAN_VERBOSE", "true")
	t.Setenv("AUTOSPEC_CCLEAN_LINE_NUMBERS", "true")
	t.Setenv("AUTOSPEC_CCLEAN_STYLE", "compact")

	cfg, err := Load("")
	require.NoError(t, err)

	assert.True(t, cfg.Cclean.Verbose, "AUTOSPEC_CCLEAN_VERBOSE should set Verbose to true")
	assert.True(t, cfg.Cclean.LineNumbers, "AUTOSPEC_CCLEAN_LINE_NUMBERS should set LineNumbers to true")
	assert.Equal(t, "compact", cfg.Cclean.Style, "AUTOSPEC_CCLEAN_STYLE should set Style")
}

func TestLoad_CcleanConfigEnvOverridesYAML(t *testing.T) {
	// Test that env vars take precedence over YAML config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// YAML sets all options to one value
	configContent := `cclean:
  verbose: false
  line_numbers: false
  style: plain
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Environment variables override
	t.Setenv("AUTOSPEC_CCLEAN_VERBOSE", "true")
	t.Setenv("AUTOSPEC_CCLEAN_STYLE", "minimal")

	cfg, err := LoadWithOptions(LoadOptions{
		ProjectConfigPath: configPath,
		SkipWarnings:      true,
	})
	require.NoError(t, err)

	// Environment should override YAML
	assert.True(t, cfg.Cclean.Verbose, "Env AUTOSPEC_CCLEAN_VERBOSE should override YAML")
	assert.Equal(t, "minimal", cfg.Cclean.Style, "Env AUTOSPEC_CCLEAN_STYLE should override YAML")
	// line_numbers not set in env, so YAML value should be used
	assert.False(t, cfg.Cclean.LineNumbers, "YAML value should remain when no env override")
}

func TestLoad_CcleanConfigPrecedence(t *testing.T) {
	// Test config priority: env > project > user > defaults
	tmpDir := t.TempDir()

	// Create user config directory
	userConfigDir := filepath.Join(tmpDir, ".config", "autospec")
	require.NoError(t, os.MkdirAll(userConfigDir, 0o755))

	// Write user config
	userConfig := `cclean:
  verbose: true
  line_numbers: true
  style: minimal
`
	userConfigPath := filepath.Join(userConfigDir, "config.yml")
	require.NoError(t, os.WriteFile(userConfigPath, []byte(userConfig), 0o644))

	// Create project config directory
	projectDir := filepath.Join(tmpDir, "project", ".autospec")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	// Write project config (overrides user for style only)
	projectConfig := `cclean:
  style: compact
`
	projectConfigPath := filepath.Join(projectDir, "config.yml")
	require.NoError(t, os.WriteFile(projectConfigPath, []byte(projectConfig), 0o644))

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

	// User config values for verbose and line_numbers (project doesn't override)
	assert.True(t, cfg.Cclean.Verbose, "User config verbose should be used")
	assert.True(t, cfg.Cclean.LineNumbers, "User config line_numbers should be used")
	// Project config value for style (overrides user)
	assert.Equal(t, "compact", cfg.Cclean.Style, "Project config style should override user")
}

// Cclean Edge Case Tests (T010)

func TestLoad_CcleanConfigInvalidStyleValue(t *testing.T) {
	// Test that invalid style values are rejected during validation (fail-fast)
	t.Parallel()

	tests := map[string]struct {
		configContent string
		wantErr       bool
		wantStyle     string
	}{
		"fancy style (invalid)": {
			configContent: `cclean:
  style: fancy
`,
			wantErr: true,
		},
		"garbage style": {
			configContent: `cclean:
  style: "!@#$%"
`,
			wantErr: true,
		},
		"empty string style": {
			configContent: `cclean:
  style: ""
`,
			wantErr:   false,
			wantStyle: "", // Empty string is valid, defaults to "default" at usage time
		},
		"whitespace style": {
			configContent: `cclean:
  style: "   "
`,
			wantErr: true, // Whitespace-only is invalid
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yml")
			err := os.WriteFile(configPath, []byte(tt.configContent), 0o644)
			require.NoError(t, err)

			cfg, err := LoadWithOptions(LoadOptions{
				ProjectConfigPath: configPath,
				SkipWarnings:      true,
			})

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "cclean.style")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStyle, cfg.Cclean.Style)
			}
		})
	}
}

func TestLoad_CcleanConfigNonBooleanVerbose(t *testing.T) {
	// Test handling of non-boolean values for verbose
	// Note: koanf/YAML accepts 0/1 as booleans but rejects strings like "yes"
	t.Parallel()

	tests := map[string]struct {
		configContent string
		wantErr       bool
		wantVerbose   bool
	}{
		"string instead of bool": {
			configContent: `cclean:
  verbose: "yes"
`,
			wantErr: true, // YAML parsing should fail for bool field
		},
		"integer 1 treated as true": {
			configContent: `cclean:
  verbose: 1
`,
			wantErr:     false, // koanf treats 1 as true
			wantVerbose: true,
		},
		"integer 0 treated as false": {
			configContent: `cclean:
  verbose: 0
`,
			wantErr:     false, // koanf treats 0 as false
			wantVerbose: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yml")
			err := os.WriteFile(configPath, []byte(tt.configContent), 0o644)
			require.NoError(t, err)

			cfg, err := LoadWithOptions(LoadOptions{
				ProjectConfigPath: configPath,
				SkipWarnings:      true,
			})

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantVerbose, cfg.Cclean.Verbose)
			}
		})
	}
}

func TestLoad_CcleanConfigNonBooleanLineNumbers(t *testing.T) {
	// Test handling of non-boolean values for line_numbers
	// Note: koanf/YAML accepts 0/1 as booleans but rejects strings like "yes"
	t.Parallel()

	tests := map[string]struct {
		configContent   string
		wantErr         bool
		wantLineNumbers bool
	}{
		"string instead of bool": {
			configContent: `cclean:
  line_numbers: "yes"
`,
			wantErr: true, // YAML parsing should fail for bool field
		},
		"integer 1 treated as true": {
			configContent: `cclean:
  line_numbers: 1
`,
			wantErr:         false, // koanf treats 1 as true
			wantLineNumbers: true,
		},
		"integer 0 treated as false": {
			configContent: `cclean:
  line_numbers: 0
`,
			wantErr:         false, // koanf treats 0 as false
			wantLineNumbers: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yml")
			err := os.WriteFile(configPath, []byte(tt.configContent), 0o644)
			require.NoError(t, err)

			cfg, err := LoadWithOptions(LoadOptions{
				ProjectConfigPath: configPath,
				SkipWarnings:      true,
			})

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantLineNumbers, cfg.Cclean.LineNumbers)
			}
		})
	}
}

func TestLoad_CcleanConfigEmptyStyleUsesDefault(t *testing.T) {
	// Test that empty style in config results in default being used at formatting time
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	// Config with empty cclean section (no style set)
	configPath := filepath.Join(tmpDir, "config.yml")
	configContent := `cclean:
  verbose: true
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	cfg, err := LoadWithOptions(LoadOptions{
		ProjectConfigPath: configPath,
		SkipWarnings:      true,
	})
	require.NoError(t, err)

	// Style should be "default" (from defaults.go)
	assert.Equal(t, "default", cfg.Cclean.Style, "Empty style should use default value")
	assert.True(t, cfg.Cclean.Verbose, "Verbose should be set from config")
}

// SkipPermissions Configuration Tests

func TestLoad_SkipPermissionsDefaults(t *testing.T) {
	// Cannot use t.Parallel() due to environment modification
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	cfg, err := Load("")
	require.NoError(t, err)
	assert.False(t, cfg.SkipPermissions, "SkipPermissions should default to false for security")
}

func TestLoad_SkipPermissionsExplicitValues(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		configContent string
		expected      bool
	}{
		"explicit true": {
			configContent: `skip_permissions: true`,
			expected:      true,
		},
		"explicit false": {
			configContent: `skip_permissions: false`,
			expected:      false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yml")
			err := os.WriteFile(configPath, []byte(tt.configContent), 0o644)
			require.NoError(t, err)

			cfg, err := LoadWithOptions(LoadOptions{
				ProjectConfigPath: configPath,
				SkipWarnings:      true,
			})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.SkipPermissions)
		})
	}
}

func TestLoad_SkipPermissionsEnvOverride(t *testing.T) {
	// Cannot use t.Parallel() due to environment modification
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Project config sets false
	configContent := `skip_permissions: false`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Environment variable overrides
	t.Setenv("AUTOSPEC_SKIP_PERMISSIONS", "true")

	cfg, err := LoadWithOptions(LoadOptions{
		ProjectConfigPath: configPath,
		SkipWarnings:      true,
	})
	require.NoError(t, err)
	assert.True(t, cfg.SkipPermissions, "Environment variable should override config file")
}

func TestLoad_SkipPermissionsPrecedence(t *testing.T) {
	// Test config priority: env > project > user > defaults
	tmpDir := t.TempDir()

	// Create user config directory
	userConfigDir := filepath.Join(tmpDir, ".config", "autospec")
	require.NoError(t, os.MkdirAll(userConfigDir, 0o755))

	// Write user config (skip_permissions: true)
	userConfig := `skip_permissions: true`
	userConfigPath := filepath.Join(userConfigDir, "config.yml")
	require.NoError(t, os.WriteFile(userConfigPath, []byte(userConfig), 0o644))

	// Create project config directory
	projectDir := filepath.Join(tmpDir, "project", ".autospec")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	// Write project config (skip_permissions: false - overrides user)
	projectConfig := `skip_permissions: false`
	projectConfigPath := filepath.Join(projectDir, "config.yml")
	require.NoError(t, os.WriteFile(projectConfigPath, []byte(projectConfig), 0o644))

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

	// Project config (false) should override user config (true)
	assert.False(t, cfg.SkipPermissions, "Project config should override user config")
}

// Verification Feature Toggle Override Tests (US-002)

func TestLoad_VerificationFeatureToggleOverride_BasicLevelWithExplicitEnable(t *testing.T) {
	// Test: basic level + explicit property_tests=true -> property tests enabled
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Basic level with property_tests explicitly enabled
	configContent := `verification:
  level: basic
  property_tests: true
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	cfg, err := LoadWithOptions(LoadOptions{
		ProjectConfigPath: configPath,
		SkipWarnings:      true,
	})
	require.NoError(t, err)

	// Basic level defaults property_tests to false, but explicit true should override
	assert.Equal(t, "basic", string(cfg.Verification.Level))
	assert.True(t, cfg.Verification.IsEnabled("property_tests"),
		"Explicit property_tests=true should override basic level default (false)")

	// Other features should remain at basic level defaults (false)
	assert.False(t, cfg.Verification.IsEnabled("adversarial_review"),
		"Non-overridden features should use basic level default")
	assert.False(t, cfg.Verification.IsEnabled("contracts"),
		"Non-overridden features should use basic level default")
	assert.False(t, cfg.Verification.IsEnabled("metamorphic_tests"),
		"Non-overridden features should use basic level default")
}

func TestLoad_VerificationFeatureToggleOverride_EnhancedLevelWithExplicitDisable(t *testing.T) {
	// Test: enhanced level + explicit contracts=false -> contracts disabled
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Enhanced level with contracts explicitly disabled
	configContent := `verification:
  level: enhanced
  contracts: false
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	cfg, err := LoadWithOptions(LoadOptions{
		ProjectConfigPath: configPath,
		SkipWarnings:      true,
	})
	require.NoError(t, err)

	// Enhanced level defaults contracts to true, but explicit false should override
	assert.Equal(t, "enhanced", string(cfg.Verification.Level))
	assert.False(t, cfg.Verification.IsEnabled("contracts"),
		"Explicit contracts=false should override enhanced level default (true)")

	// Other features should remain at enhanced level defaults
	assert.False(t, cfg.Verification.IsEnabled("adversarial_review"),
		"Non-overridden features should use enhanced level default")
	assert.False(t, cfg.Verification.IsEnabled("property_tests"),
		"Non-overridden features should use enhanced level default")
}

func TestLoad_VerificationFeatureToggleOverride_FullLevelWithAllExplicitFalse(t *testing.T) {
	// Test: full level + all explicit false -> all features disabled
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Full level with all features explicitly disabled
	configContent := `verification:
  level: full
  adversarial_review: false
  contracts: false
  property_tests: false
  metamorphic_tests: false
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	cfg, err := LoadWithOptions(LoadOptions{
		ProjectConfigPath: configPath,
		SkipWarnings:      true,
	})
	require.NoError(t, err)

	// Full level defaults all to true, but explicit false should override all
	assert.Equal(t, "full", string(cfg.Verification.Level))
	assert.False(t, cfg.Verification.IsEnabled("adversarial_review"),
		"Explicit adversarial_review=false should override full level default (true)")
	assert.False(t, cfg.Verification.IsEnabled("contracts"),
		"Explicit contracts=false should override full level default (true)")
	assert.False(t, cfg.Verification.IsEnabled("property_tests"),
		"Explicit property_tests=false should override full level default (true)")
	assert.False(t, cfg.Verification.IsEnabled("metamorphic_tests"),
		"Explicit metamorphic_tests=false should override full level default (true)")
}

func TestLoad_VerificationFeatureToggleOverride_NoExplicitToggles(t *testing.T) {
	// Test: no explicit toggles -> level defaults applied
	t.Parallel()

	tests := map[string]struct {
		level                 string
		wantAdversarialReview bool
		wantContracts         bool
		wantPropertyTests     bool
		wantMetamorphicTests  bool
	}{
		"basic level defaults": {
			level:                 "basic",
			wantAdversarialReview: false,
			wantContracts:         false,
			wantPropertyTests:     false,
			wantMetamorphicTests:  false,
		},
		"enhanced level defaults": {
			level:                 "enhanced",
			wantAdversarialReview: false,
			wantContracts:         true,
			wantPropertyTests:     false,
			wantMetamorphicTests:  false,
		},
		"full level defaults": {
			level:                 "full",
			wantAdversarialReview: true,
			wantContracts:         true,
			wantPropertyTests:     true,
			wantMetamorphicTests:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yml")

			// Only set level, no explicit toggles
			configContent := fmt.Sprintf(`verification:
  level: %s
`, tt.level)
			err := os.WriteFile(configPath, []byte(configContent), 0o644)
			require.NoError(t, err)

			cfg, err := LoadWithOptions(LoadOptions{
				ProjectConfigPath: configPath,
				SkipWarnings:      true,
			})
			require.NoError(t, err)

			assert.Equal(t, tt.level, string(cfg.Verification.Level))
			assert.Equal(t, tt.wantAdversarialReview, cfg.Verification.IsEnabled("adversarial_review"),
				"adversarial_review should use level default")
			assert.Equal(t, tt.wantContracts, cfg.Verification.IsEnabled("contracts"),
				"contracts should use level default")
			assert.Equal(t, tt.wantPropertyTests, cfg.Verification.IsEnabled("property_tests"),
				"property_tests should use level default")
			assert.Equal(t, tt.wantMetamorphicTests, cfg.Verification.IsEnabled("metamorphic_tests"),
				"metamorphic_tests should use level default")
		})
	}
}

func TestLoad_VerificationFeatureToggleOverride_MixedOverrides(t *testing.T) {
	// Test: mixed explicit and implicit toggles
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Enhanced level with some explicit overrides
	configContent := `verification:
  level: enhanced
  adversarial_review: true
  property_tests: true
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	cfg, err := LoadWithOptions(LoadOptions{
		ProjectConfigPath: configPath,
		SkipWarnings:      true,
	})
	require.NoError(t, err)

	// Explicit overrides should take effect
	assert.True(t, cfg.Verification.IsEnabled("adversarial_review"),
		"Explicit adversarial_review=true should enable feature")
	assert.True(t, cfg.Verification.IsEnabled("property_tests"),
		"Explicit property_tests=true should enable feature")

	// Non-overridden features should use enhanced level defaults
	assert.True(t, cfg.Verification.IsEnabled("contracts"),
		"Non-overridden contracts should use enhanced level default (true)")
	assert.False(t, cfg.Verification.IsEnabled("metamorphic_tests"),
		"Non-overridden metamorphic_tests should use enhanced level default (false)")
}

func TestLoad_VerificationFeatureToggleOverride_EnvOverridesYAML(t *testing.T) {
	// Test: env vars override YAML config for feature toggles
	// Cannot use t.Parallel() due to environment modification

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// YAML config with basic level and property_tests false
	configContent := `verification:
  level: basic
  property_tests: false
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Environment variable overrides property_tests to true
	t.Setenv("AUTOSPEC_VERIFICATION_PROPERTY_TESTS", "true")

	cfg, err := LoadWithOptions(LoadOptions{
		ProjectConfigPath: configPath,
		SkipWarnings:      true,
	})
	require.NoError(t, err)

	assert.True(t, cfg.Verification.IsEnabled("property_tests"),
		"Environment variable should override YAML config")
}

func TestLoad_VerificationFeatureToggleOverride_EnvOverridesLevel(t *testing.T) {
	// Test: env var can override the verification level
	// Cannot use t.Parallel() due to environment modification

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// YAML config with basic level
	configContent := `verification:
  level: basic
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Environment variable overrides level to full
	t.Setenv("AUTOSPEC_VERIFICATION_LEVEL", "full")

	cfg, err := LoadWithOptions(LoadOptions{
		ProjectConfigPath: configPath,
		SkipWarnings:      true,
	})
	require.NoError(t, err)

	assert.Equal(t, "full", string(cfg.Verification.Level),
		"Environment variable should override YAML level")
	// Full level should enable all features by default
	assert.True(t, cfg.Verification.IsEnabled("adversarial_review"))
	assert.True(t, cfg.Verification.IsEnabled("contracts"))
	assert.True(t, cfg.Verification.IsEnabled("property_tests"))
	assert.True(t, cfg.Verification.IsEnabled("metamorphic_tests"))
}

func TestLoad_OpenCodeAgentEnvOverride(t *testing.T) {
	t.Setenv("AUTOSPEC_OPENCODE_AGENT", "Plan")
	cfg, err := Load("")
	require.NoError(t, err)
	assert.Equal(t, "Plan", cfg.OpenCodeAgent)
}

func TestLoad_OpenCodeAgentEnvOverridesConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	configContent := `opencode_agent: Build`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	t.Setenv("AUTOSPEC_OPENCODE_AGENT", "Plan")

	cfg, err := LoadWithOptions(LoadOptions{ProjectConfigPath: configPath, SkipWarnings: true})
	require.NoError(t, err)
	assert.Equal(t, "Plan", cfg.OpenCodeAgent, "Env should override config file")
}

func TestLoad_OpenCodeAgentFromYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	configContent := `opencode_agent: Build`

	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	cfg, err := LoadWithOptions(LoadOptions{ProjectConfigPath: configPath, SkipWarnings: true})
	require.NoError(t, err)
	assert.Equal(t, "Build", cfg.OpenCodeAgent)
}
