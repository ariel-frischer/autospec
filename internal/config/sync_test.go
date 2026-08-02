package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestStageModelConfigurationSurfacesStaySynchronized(t *testing.T) {
	t.Parallel()

	var templateDefaults map[string]interface{}
	require.NoError(t, yaml.Unmarshal([]byte(GetDefaultConfigTemplate()), &templateDefaults))
	runtimeDefaults := GetDefaults()
	configType := reflect.TypeOf(Configuration{})

	for stage := range stageModelEffortCases {
		t.Run(stage, func(t *testing.T) {
			t.Parallel()
			path := "models." + stage

			assert.True(t, configurationHasStageModel(configType, stage), "Configuration decoding missing %s", path)
			assert.Contains(t, KnownKeys, path, "KnownKeys missing %s", path)
			assert.True(t, hasEmptyStageDefault(runtimeDefaults, stage), "runtime defaults missing %s", path)
			assert.True(t, hasEmptyStageDefault(templateDefaults, stage), "generated YAML defaults missing %s", path)
		})
	}
}

func configurationHasStageModel(configType reflect.Type, stage string) bool {
	modelsField, ok := configType.FieldByName("Models")
	if !ok || modelsField.Tag.Get("koanf") != "models" {
		return false
	}

	for index := 0; index < modelsField.Type.NumField(); index++ {
		if modelsField.Type.Field(index).Tag.Get("koanf") == stage {
			return true
		}
	}
	return false
}

func hasEmptyStageDefault(defaults map[string]interface{}, stage string) bool {
	models, ok := defaults["models"].(map[string]interface{})
	if !ok {
		return false
	}
	value, ok := models[stage]
	return ok && value == ""
}

func TestFlattenDefaults(t *testing.T) {
	t.Parallel()

	flat := flattenDefaults()

	// Should have flattened nested keys
	assert.Contains(t, flat, "notifications.enabled")
	assert.Contains(t, flat, "notifications.type")
	assert.Contains(t, flat, "worktree.auto_setup")

	// Should have top-level keys
	assert.Contains(t, flat, "max_retries")
	assert.Contains(t, flat, "timeout")
	assert.Contains(t, flat, "specs_dir")

	// Should not have nested map values (those should be flattened)
	for key, value := range flat {
		_, isMap := value.(map[string]interface{})
		assert.False(t, isMap, "key %q should not have map value after flattening", key)
	}
}

func TestExtractUserKeys(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yaml     string
		wantKeys []string
	}{
		"simple keys": {
			yaml:     "max_retries: 5\ntimeout: 300\n",
			wantKeys: []string{"max_retries", "timeout"},
		},
		"nested keys": {
			yaml:     "notifications:\n  enabled: true\n  type: sound\n",
			wantKeys: []string{"notifications.enabled", "notifications.type"},
		},
		"mixed keys": {
			yaml:     "max_retries: 5\nnotifications:\n  enabled: true\n",
			wantKeys: []string{"max_retries", "notifications.enabled"},
		},
		"empty config": {
			yaml:     "",
			wantKeys: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.yaml), &node)
			require.NoError(t, err)

			keys := extractUserKeys(&node)
			assert.ElementsMatch(t, tt.wantKeys, keys)
		})
	}
}

func TestFindMissingKeys(t *testing.T) {
	t.Parallel()

	schemaKeys := map[string]interface{}{
		"max_retries": 0,
		"timeout":     2400,
		"specs_dir":   "./specs",
	}

	tests := map[string]struct {
		userKeys    []string
		wantMissing []string
	}{
		"all missing": {
			userKeys:    []string{},
			wantMissing: []string{"max_retries", "specs_dir", "timeout"},
		},
		"none missing": {
			userKeys:    []string{"max_retries", "timeout", "specs_dir"},
			wantMissing: []string{},
		},
		"some missing": {
			userKeys:    []string{"max_retries"},
			wantMissing: []string{"specs_dir", "timeout"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			missing := findMissingKeys(tt.userKeys, schemaKeys)
			assert.ElementsMatch(t, tt.wantMissing, missing)
		})
	}
}

func TestFindDeprecatedKeys(t *testing.T) {
	t.Parallel()

	schemaKeys := map[string]interface{}{
		"max_retries": 0,
		"timeout":     2400,
	}

	tests := map[string]struct {
		userKeys       []string
		wantDeprecated []string
	}{
		"no deprecated": {
			userKeys:       []string{"max_retries", "timeout"},
			wantDeprecated: []string{},
		},
		"has deprecated": {
			userKeys:       []string{"max_retries", "old_field", "legacy_setting"},
			wantDeprecated: []string{"legacy_setting", "old_field"},
		},
		"all deprecated": {
			userKeys:       []string{"old_field", "legacy_setting"},
			wantDeprecated: []string{"legacy_setting", "old_field"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			deprecated := findDeprecatedKeys(tt.userKeys, schemaKeys)
			assert.ElementsMatch(t, tt.wantDeprecated, deprecated)
		})
	}
}

func TestGenerateNewKeysBlock(t *testing.T) {
	t.Parallel()

	defaults := map[string]interface{}{
		"max_retries": 0,
		"timeout":     2400,
		"specs_dir":   "./specs",
	}

	tests := map[string]struct {
		missing      []string
		wantContains []string
	}{
		"generates uncommented keys for missing keys": {
			missing:      []string{"max_retries", "timeout"},
			wantContains: []string{"max_retries: 0", "timeout: 2400"},
		},
		"empty missing returns empty": {
			missing:      []string{},
			wantContains: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			block := generateNewKeysBlock(tt.missing, defaults)

			if len(tt.wantContains) == 0 {
				assert.Empty(t, block)
				return
			}

			for _, want := range tt.wantContains {
				assert.Contains(t, block, want)
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value interface{}
		want  string
	}{
		"empty string":      {value: "", want: `""`},
		"simple string":     {value: "hello", want: "hello"},
		"string with colon": {value: "key: value", want: `"key: value"`},
		"boolean true":      {value: true, want: "true"},
		"boolean false":     {value: false, want: "false"},
		"integer":           {value: 42, want: "42"},
		"empty slice":       {value: []string{}, want: "[]"},
		"string slice":      {value: []string{"a", "b"}, want: "[a, b]"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := formatValue(tt.value)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestSyncConfig(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		initialConfig  string
		dryRun         bool
		wantAddedMin   int // Minimum number of added keys
		wantRemovedMin int // Minimum number of removed keys
		wantChanged    bool
		wantErr        bool
	}{
		"config with deprecated key removes it": {
			initialConfig:  "deprecated_field: value\nmax_retries: 5\n",
			wantRemovedMin: 1,
			wantChanged:    true,
		},
		"dry run does not modify file": {
			initialConfig:  "deprecated_field: value\n",
			dryRun:         true,
			wantRemovedMin: 1,
			wantChanged:    true,
		},
		"preserves user values": {
			initialConfig: "max_retries: 10\ntimeout: 600\n",
			wantAddedMin:  1, // At least some keys should be added
			wantChanged:   true,
		},
		"handles nested keys": {
			initialConfig: "notifications:\n  enabled: true\n",
			wantAddedMin:  1, // Missing notification sub-keys should be detected
			wantChanged:   true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yml")

			require.NoError(t, os.WriteFile(configPath, []byte(tt.initialConfig), 0o644))

			result, err := SyncConfig(configPath, SyncOptions{DryRun: tt.dryRun})

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.GreaterOrEqual(t, len(result.Added), tt.wantAddedMin,
				"expected at least %d added keys", tt.wantAddedMin)
			assert.GreaterOrEqual(t, len(result.Removed), tt.wantRemovedMin,
				"expected at least %d removed keys", tt.wantRemovedMin)
			assert.Equal(t, tt.wantChanged, result.Changed)

			// Verify dry run didn't modify file
			if tt.dryRun {
				content, _ := os.ReadFile(configPath)
				assert.Equal(t, tt.initialConfig, string(content))
			}

			// Verify actual changes were made when not dry run
			if !tt.dryRun && tt.wantChanged {
				content, _ := os.ReadFile(configPath)
				assert.NotEqual(t, tt.initialConfig, string(content))
			}
		})
	}
}

func TestSyncConfigPreservesUserValues(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// User has custom values
	initialConfig := `max_retries: 10
timeout: 600
specs_dir: ./my-specs
`
	require.NoError(t, os.WriteFile(configPath, []byte(initialConfig), 0o644))

	result, err := SyncConfig(configPath, SyncOptions{})
	require.NoError(t, err)

	// Read the synced config
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	// User values should be preserved
	assert.Contains(t, string(content), "max_retries: 10")
	assert.Contains(t, string(content), "timeout: 600")
	assert.Contains(t, string(content), "specs_dir: ./my-specs")

	// New keys should be added as comments
	assert.True(t, result.Changed)
	assert.Greater(t, len(result.Added), 0)
}

func TestSyncConfigNonExistentFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.yml")

	result, err := SyncConfig(configPath, SyncOptions{})
	require.NoError(t, err)

	// Should return empty result for non-existent file
	assert.False(t, result.Changed)
	assert.Empty(t, result.Added)
	assert.Empty(t, result.Removed)
}

func TestSyncConfigMalformedYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "malformed.yml")

	malformedYAML := `max_retries: 10
  invalid indent
    more invalid:
`
	require.NoError(t, os.WriteFile(configPath, []byte(malformedYAML), 0o644))

	_, err := SyncConfig(configPath, SyncOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing config YAML")
}

func TestRemoveDeprecatedKeys(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yaml           string
		deprecated     []string
		wantNotContain []string
		wantContain    []string
	}{
		"removes top-level key": {
			yaml:           "keep: value\nremove: value\n",
			deprecated:     []string{"remove"},
			wantNotContain: []string{"remove"},
			wantContain:    []string{"keep"},
		},
		"removes nested key": {
			yaml:           "parent:\n  keep: value\n  remove: value\n",
			deprecated:     []string{"parent.remove"},
			wantNotContain: []string{"remove:"},
			wantContain:    []string{"keep:"},
		},
		"handles empty deprecated list": {
			yaml:           "keep: value\n",
			deprecated:     []string{},
			wantNotContain: nil,
			wantContain:    []string{"keep"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.yaml), &node)
			require.NoError(t, err)

			err = removeDeprecatedKeys(&node, tt.deprecated)
			require.NoError(t, err)

			result, err := yaml.Marshal(&node)
			require.NoError(t, err)

			for _, s := range tt.wantNotContain {
				assert.NotContains(t, string(result), s)
			}
			for _, s := range tt.wantContain {
				assert.Contains(t, string(result), s)
			}
		})
	}
}
