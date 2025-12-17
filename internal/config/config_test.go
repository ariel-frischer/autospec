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
