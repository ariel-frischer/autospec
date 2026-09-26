package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateProfileName(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		name  string
		valid bool
	}{
		"letters":               {name: "work", valid: true},
		"hyphen and underscore": {name: "work-fast_v2", valid: true},
		"empty":                 {name: "", valid: false},
		"path traversal":        {name: "../work", valid: false},
		"spaces":                {name: "my work", valid: false},
		"dot":                   {name: "work.yml", valid: false},
		"too long":              {name: "12345678901234567890123456789012345678901234567890123456789012345", valid: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.valid, ValidateProfileName(tt.name) == nil)
		})
	}
}

func TestProfileOperations(t *testing.T) {
	tests := map[string]struct {
		setup func(t *testing.T, dir string)
		check func(t *testing.T, dir string)
	}{
		"save load and list": {
			setup: func(t *testing.T, dir string) {
				require.NoError(t, SaveProfileTo(dir, "work", map[string]interface{}{"timeout": 60}, false))
			},
			check: func(t *testing.T, dir string) {
				values, err := LoadProfileFrom(dir, "work")
				require.NoError(t, err)
				assert.Equal(t, 60, values["timeout"])
				profiles, err := ListProfilesFrom(dir)
				require.NoError(t, err)
				assert.Equal(t, []string{"work"}, profiles)
			},
		},
		"overwrite requires force": {
			setup: func(t *testing.T, dir string) {
				require.NoError(t, SaveProfileTo(dir, "work", map[string]interface{}{"timeout": 60}, false))
				assert.Error(t, SaveProfileTo(dir, "work", map[string]interface{}{"timeout": 90}, false))
				require.NoError(t, SaveProfileTo(dir, "work", map[string]interface{}{"timeout": 90}, true))
			},
			check: func(t *testing.T, dir string) {
				values, err := LoadProfileFrom(dir, "work")
				require.NoError(t, err)
				assert.Equal(t, 90, values["timeout"])
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(t, dir)
			tt.check(t, dir)
		})
	}
}

func TestLoadProfileFrom_Missing(t *testing.T) {
	t.Parallel()

	_, err := LoadProfileFrom(t.TempDir(), "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "profile")
}

func TestProfilePath(t *testing.T) {
	t.Parallel()

	dir := filepath.Join("tmp", "autospec")
	assert.Equal(t, filepath.Join(dir, "profiles", "work.yml"), ProfilePath(dir, "work"))
}

func TestSaveProfileToCreatesDirectory(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "nested")
	require.NoError(t, SaveProfileTo(dir, "work", map[string]interface{}{"timeout": 60}, false))
	_, err := os.Stat(filepath.Join(dir, "profiles", "work.yml"))
	require.NoError(t, err)
}

func TestLoadWithOptions_ProfilePrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config")
	userDir := filepath.Join(configDir, "autospec")
	projectDir := filepath.Join(tmpDir, "project", ".autospec")
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	require.NoError(t, os.MkdirAll(filepath.Dir(projectDir), 0o755))
	require.NoError(t, os.Chdir(filepath.Dir(projectDir)))
	require.NoError(t, os.MkdirAll(userDir, 0o755))
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	require.NoError(t, os.WriteFile(filepath.Join(userDir, "config.yml"), []byte("timeout: 10\n"), 0o644))
	require.NoError(t, SaveProfileTo(userDir, "fast", map[string]interface{}{"timeout": 20}, false))
	projectConfig := filepath.Join(projectDir, "config.yml")
	require.NoError(t, os.WriteFile(projectConfig, []byte("timeout: 30\n"), 0o644))
	projectProfilesDir := filepath.Join(projectDir, "profiles")
	require.NoError(t, os.MkdirAll(projectProfilesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectProfilesDir, "fast.yml"), []byte("max_retries: 4\n"), 0o644))

	cfg, err := LoadWithOptions(LoadOptions{ProjectConfigPath: projectConfig, Profile: "fast", SkipWarnings: true})
	require.NoError(t, err)
	assert.Equal(t, 20, cfg.Timeout)
	assert.Equal(t, 4, cfg.MaxRetries)

	t.Setenv("AUTOSPEC_TIMEOUT", "40")
	cfg, err = LoadWithOptions(LoadOptions{ProjectConfigPath: projectConfig, Profile: "fast", SkipWarnings: true})
	require.NoError(t, err)
	assert.Equal(t, 40, cfg.Timeout)
}

func TestLoadWithOptions_ProfileOverlay(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), ".config"))
	projectDir := t.TempDir()
	original, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(original)
	require.NoError(t, os.Chdir(projectDir))

	userDir, err := UserConfigDir()
	require.NoError(t, err)
	profilePath := ProfilePath(userDir, "cheap")
	require.NoError(t, os.MkdirAll(filepath.Dir(profilePath), 0o755))
	profile := "agent_preset: codex\nmodel: openai/gpt-6-luna\nreasoning_efforts:\n  specify: xhigh\n  plan: max\n"
	require.NoError(t, os.WriteFile(profilePath, []byte(profile), 0o644))

	cfg, err := LoadWithOptions(LoadOptions{Profile: "cheap"})
	require.NoError(t, err)
	assert.Equal(t, "codex", cfg.AgentPreset)
	assert.Equal(t, "openai/gpt-6-luna", cfg.Model)
	assert.Equal(t, "xhigh", cfg.ReasoningEfforts.Specify)
	assert.Equal(t, "max", cfg.ReasoningEfforts.Plan)
}

func TestLoadWithOptions_MissingProfile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), ".config"))
	_, err := LoadWithOptions(LoadOptions{Profile: "missing"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `profile "missing" not found`)
}
