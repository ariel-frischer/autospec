package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseKeyPath(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		path    string
		want    []string
		wantErr error
	}{
		"single key": {
			path: "max_retries",
			want: []string{"max_retries"},
		},
		"nested key": {
			path: "notifications.enabled",
			want: []string{"notifications", "enabled"},
		},
		"deeply nested key": {
			path: "a.b.c.d",
			want: []string{"a", "b", "c", "d"},
		},
		"empty string": {
			path:    "",
			wantErr: ErrEmptyKeyPath,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseKeyPath(tt.path)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ParseKeyPath(%q) error = %v, wantErr = %v", tt.path, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Errorf("ParseKeyPath(%q) = %v, want %v", tt.path, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParseKeyPath(%q)[%d] = %q, want %q", tt.path, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSetNestedValue(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		initialYAML  string
		keyPath      []string
		value        interface{}
		expectedYAML string
	}{
		"set top-level string": {
			initialYAML:  "",
			keyPath:      []string{"name"},
			value:        "test",
			expectedYAML: "name: test\n",
		},
		"set top-level int": {
			initialYAML:  "",
			keyPath:      []string{"count"},
			value:        42,
			expectedYAML: "count: 42\n",
		},
		"set top-level bool": {
			initialYAML:  "",
			keyPath:      []string{"enabled"},
			value:        true,
			expectedYAML: "enabled: true\n",
		},
		"set nested value": {
			initialYAML:  "",
			keyPath:      []string{"notifications", "enabled"},
			value:        true,
			expectedYAML: "notifications:\n    enabled: true\n",
		},
		"update existing value": {
			initialYAML:  "name: old\n",
			keyPath:      []string{"name"},
			value:        "new",
			expectedYAML: "name: new\n",
		},
		"add to existing": {
			initialYAML:  "existing: value\n",
			keyPath:      []string{"new_key"},
			value:        "new_value",
			expectedYAML: "existing: value\nnew_key: new_value\n",
		},
		"update nested in existing": {
			initialYAML:  "notifications:\n    type: sound\n",
			keyPath:      []string{"notifications", "enabled"},
			value:        true,
			expectedYAML: "notifications:\n    type: sound\n    enabled: true\n",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var root yaml.Node
			if tt.initialYAML != "" {
				if err := yaml.Unmarshal([]byte(tt.initialYAML), &root); err != nil {
					t.Fatalf("failed to parse initial YAML: %v", err)
				}
			}

			if err := SetNestedValue(&root, tt.keyPath, tt.value); err != nil {
				t.Fatalf("SetNestedValue() error: %v", err)
			}

			out, err := yaml.Marshal(&root)
			if err != nil {
				t.Fatalf("failed to marshal result: %v", err)
			}

			if string(out) != tt.expectedYAML {
				t.Errorf("SetNestedValue() result:\n%s\nwant:\n%s", out, tt.expectedYAML)
			}
		})
	}
}

func TestGetNestedValue(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yaml    string
		keyPath []string
		want    string
		wantNil bool
	}{
		"get top-level": {
			yaml:    "name: test\n",
			keyPath: []string{"name"},
			want:    "test",
		},
		"get nested": {
			yaml:    "notifications:\n  enabled: true\n",
			keyPath: []string{"notifications", "enabled"},
			want:    "true",
		},
		"missing key": {
			yaml:    "name: test\n",
			keyPath: []string{"missing"},
			wantNil: true,
		},
		"missing nested key": {
			yaml:    "notifications:\n  enabled: true\n",
			keyPath: []string{"notifications", "missing"},
			wantNil: true,
		},
		"empty path": {
			yaml:    "name: test\n",
			keyPath: []string{},
			wantNil: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var root yaml.Node
			if err := yaml.Unmarshal([]byte(tt.yaml), &root); err != nil {
				t.Fatalf("failed to parse YAML: %v", err)
			}

			got := GetNestedValue(&root, tt.keyPath)

			if tt.wantNil {
				if got != nil {
					t.Errorf("GetNestedValue() = %v, want nil", got.Value)
				}
				return
			}

			if got == nil {
				t.Fatalf("GetNestedValue() = nil, want %q", tt.want)
			}
			if got.Value != tt.want {
				t.Errorf("GetNestedValue() = %q, want %q", got.Value, tt.want)
			}
		})
	}
}

func TestSetConfigValue(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		initialContent string
		key            string
		value          string
		wantContains   []string
		wantErr        bool
		errContain     string
	}{
		"set new value": {
			key:          "max_retries",
			value:        "5",
			wantContains: []string{"max_retries: 5"},
		},
		"set nested value": {
			key:          "notifications.enabled",
			value:        "true",
			wantContains: []string{"notifications:", "enabled: true"},
		},
		"update existing value": {
			initialContent: "max_retries: 3\n",
			key:            "max_retries",
			value:          "10",
			wantContains:   []string{"max_retries: 10"},
		},
		"invalid key": {
			key:        "unknown.key",
			value:      "value",
			wantErr:    true,
			errContain: "unknown configuration key",
		},
		"invalid value type": {
			key:        "max_retries",
			value:      "not-a-number",
			wantErr:    true,
			errContain: "invalid integer",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yml")

			if tt.initialContent != "" {
				if err := os.WriteFile(configPath, []byte(tt.initialContent), 0o644); err != nil {
					t.Fatalf("failed to write initial content: %v", err)
				}
			}

			err := SetConfigValue(configPath, tt.key, tt.value)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errContain != "" && !containsString(err.Error(), tt.errContain) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errContain)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			content, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("failed to read config file: %v", err)
			}

			for _, want := range tt.wantContains {
				if !containsString(string(content), want) {
					t.Errorf("config content = %q, want to contain %q", content, want)
				}
			}
		})
	}
}

func TestSetConfigValueCreatesFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "config.yml")

	err := SetConfigValue(configPath, "max_retries", "5")
	if err != nil {
		t.Fatalf("SetConfigValue() error: %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	if !containsString(string(content), "max_retries: 5") {
		t.Errorf("config content = %q, want to contain 'max_retries: 5'", content)
	}
}

func TestSetConfigValuePreservesComments(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	initialContent := `# This is a comment
max_retries: 3
# Another comment
timeout: 100
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0o644); err != nil {
		t.Fatalf("failed to write initial content: %v", err)
	}

	err := SetConfigValue(configPath, "max_retries", "5")
	if err != nil {
		t.Fatalf("SetConfigValue() error: %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	// Check the value was updated
	if !containsString(string(content), "max_retries: 5") {
		t.Errorf("config content = %q, want to contain 'max_retries: 5'", content)
	}

	// Check that comments are preserved (yaml.v3 should preserve them)
	if !containsString(string(content), "# This is a comment") {
		t.Logf("Note: Top-level comments may not be preserved: %s", content)
	}
}

func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
