package integration

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCrossPlatformConfigLoading tests configuration loading on all platforms
func TestCrossPlatformConfigLoading(t *testing.T) {
	tests := map[string]struct {
		configContent  string
		envVars        map[string]string
		wantMaxRetries int
		wantSpecsDir   string
	}{
		"default config": {
			configContent: `{
				"claude_cmd": "claude",
				"max_retries": 3,
				"specs_dir": "./specs",
				"state_dir": "~/.autospec/state",
				"timeout": 300
			}`,
			wantMaxRetries: 3,
			wantSpecsDir:   "./specs",
		},
		"custom specs dir": {
			configContent: `{
				"claude_cmd": "claude",
				"max_retries": 5,
				"specs_dir": "./my-specs",
				"state_dir": "~/.autospec/state",
				"timeout": 300
			}`,
			wantMaxRetries: 5,
			wantSpecsDir:   "./my-specs",
		},
		"env var override": {
			configContent: `{
				"claude_cmd": "claude",
				"max_retries": 3,
				"specs_dir": "./specs",
				"state_dir": "~/.autospec/state",
				"timeout": 300
			}`,
			envVars: map[string]string{
				"AUTOSPEC_MAX_RETRIES": "7",
			},
			wantMaxRetries: 7,
			wantSpecsDir:   "./specs",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temp directory for config
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, ".autospec", "config.json")

			// Create config directory
			err := os.MkdirAll(filepath.Dir(configPath), 0755)
			require.NoError(t, err)

			// Write config file
			err = os.WriteFile(configPath, []byte(tc.configContent), 0644)
			require.NoError(t, err)

			// Set environment variables
			for key, value := range tc.envVars {
				t.Setenv(key, value)
			}

			// Load configuration
			cfg, err := config.Load(configPath)
			require.NoError(t, err)

			// Verify configuration
			assert.Equal(t, tc.wantMaxRetries, cfg.MaxRetries)
			assert.Equal(t, tc.wantSpecsDir, cfg.SpecsDir)
		})
	}
}

// TestCrossPlatformPathHandling tests path handling across platforms
func TestCrossPlatformPathHandling(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".autospec", "config.json")

	// Create config directory
	err := os.MkdirAll(filepath.Dir(configPath), 0755)
	require.NoError(t, err)

	// Test with relative path
	configContent := `{
		"claude_cmd": "claude",
		"max_retries": 3,
		"specs_dir": "./specs",
		"state_dir": "~/.autospec/state",
		"timeout": 300
	}`

	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load configuration
	cfg, err := config.Load(configPath)
	require.NoError(t, err)

	// Verify paths use correct separators for platform
	expectedSeparator := string(filepath.Separator)

	// StateDir should have home directory expanded
	assert.Contains(t, cfg.StateDir, expectedSeparator,
		"StateDir should use platform-specific path separator")

	// On Windows, paths should use backslashes
	// On Unix, paths should use forward slashes
	if runtime.GOOS == "windows" {
		assert.Contains(t, cfg.StateDir, "\\",
			"Windows paths should use backslashes")
	} else {
		assert.Contains(t, cfg.StateDir, "/",
			"Unix paths should use forward slashes")
	}
}

// TestHomeDirectoryExpansion tests ~/ expansion on all platforms
func TestHomeDirectoryExpansion(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".autospec", "config.json")

	// Create config directory
	err := os.MkdirAll(filepath.Dir(configPath), 0755)
	require.NoError(t, err)

	// Config with ~/ in state_dir
	configContent := `{
		"claude_cmd": "claude",
		"max_retries": 3,
		"specs_dir": "./specs",
		"state_dir": "~/.autospec/state",
		"timeout": 300
	}`

	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load configuration
	cfg, err := config.Load(configPath)
	require.NoError(t, err)

	// Verify ~/ was expanded to actual home directory
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	expectedStateDir := filepath.Join(homeDir, ".autospec", "state")
	assert.Equal(t, expectedStateDir, cfg.StateDir,
		"StateDir should have ~/ expanded to home directory")

	// Verify the path doesn't contain literal ~
	assert.NotContains(t, cfg.StateDir, "~",
		"StateDir should not contain literal ~")
}
