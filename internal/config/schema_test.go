package config

import (
	"errors"
	"testing"
)

func TestGetKeySchema(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		key       string
		wantType  ConfigValueType
		wantErr   bool
		errString string
	}{
		"known bool key": {
			key:      "notifications.enabled",
			wantType: TypeBool,
			wantErr:  false,
		},
		"known int key": {
			key:      "max_retries",
			wantType: TypeInt,
			wantErr:  false,
		},
		"known duration key": {
			key:      "notifications.long_running_threshold",
			wantType: TypeDuration,
			wantErr:  false,
		},
		"known string key": {
			key:      "claude_cmd",
			wantType: TypeString,
			wantErr:  false,
		},
		"known enum key": {
			key:      "notifications.type",
			wantType: TypeEnum,
			wantErr:  false,
		},
		"unknown key": {
			key:       "foo.bar.baz",
			wantErr:   true,
			errString: "unknown configuration key: foo.bar.baz",
		},
		"partial nested key": {
			key:       "notifications",
			wantErr:   true,
			errString: "unknown configuration key: notifications",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			schema, err := GetKeySchema(tt.key)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for key %q, got nil", tt.key)
				}
				var unknownKeyErr ErrUnknownKey
				if !errors.As(err, &unknownKeyErr) {
					t.Fatalf("expected ErrUnknownKey, got %T: %v", err, err)
				}
				if tt.errString != "" && err.Error() != tt.errString {
					t.Errorf("error message = %q, want %q", err.Error(), tt.errString)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if schema.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", schema.Type, tt.wantType)
			}
			if schema.Path != tt.key {
				t.Errorf("Path = %q, want %q", schema.Path, tt.key)
			}
		})
	}
}

func TestEnumKeysHaveAllowedValues(t *testing.T) {
	t.Parallel()

	for path, schema := range KnownKeys {
		path, schema := path, schema // capture range variable
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			if schema.Type == TypeEnum {
				if len(schema.AllowedValues) == 0 {
					t.Errorf("enum key %q has empty AllowedValues", path)
				}
			}
		})
	}
}

func TestConfigValueType_String(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		valueType ConfigValueType
		want      string
	}{
		"bool":     {valueType: TypeBool, want: "bool"},
		"int":      {valueType: TypeInt, want: "int"},
		"duration": {valueType: TypeDuration, want: "duration"},
		"string":   {valueType: TypeString, want: "string"},
		"enum":     {valueType: TypeEnum, want: "enum"},
		"unknown":  {valueType: ConfigValueType(99), want: "unknown"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := tt.valueType.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestKnownKeysComplete(t *testing.T) {
	t.Parallel()

	// Verify all expected keys are present
	expectedKeys := []string{
		"notifications.enabled",
		"notifications.type",
		"notifications.on_command_complete",
		"notifications.on_error",
		"notifications.on_long_running",
		"notifications.long_running_threshold",
		"max_retries",
		"timeout",
		"skip_preflight",
		"skip_confirmations",
		"claude_cmd",
		"specs_dir",
	}

	for _, key := range expectedKeys {
		if _, ok := KnownKeys[key]; !ok {
			t.Errorf("missing expected key in KnownKeys: %q", key)
		}
	}
}

func TestInferType(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value string
		want  ConfigValueType
	}{
		"bool true":             {value: "true", want: TypeBool},
		"bool false":            {value: "false", want: TypeBool},
		"int zero":              {value: "0", want: TypeInt},
		"int positive":          {value: "5", want: TypeInt},
		"int negative":          {value: "-1", want: TypeInt},
		"duration minutes":      {value: "5m", want: TypeDuration},
		"duration hours":        {value: "1h30m", want: TypeDuration},
		"duration seconds":      {value: "10s", want: TypeDuration},
		"string path":           {value: "path/to/file", want: TypeString},
		"string hello":          {value: "hello", want: TypeString},
		"string with numbers":   {value: "v1.2.3", want: TypeString},
		"string empty":          {value: "", want: TypeString},
		"string True uppercase": {value: "True", want: TypeString}, // Only lowercase bool
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := InferType(tt.value)
			if got != tt.want {
				t.Errorf("InferType(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestValidateValue(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		key        string
		value      string
		wantParsed interface{}
		wantType   ConfigValueType
		wantErr    bool
		errContain string
	}{
		"valid bool true": {
			key:        "notifications.enabled",
			value:      "true",
			wantParsed: true,
			wantType:   TypeBool,
		},
		"valid bool false": {
			key:        "skip_preflight",
			value:      "false",
			wantParsed: false,
			wantType:   TypeBool,
		},
		"invalid bool for int key": {
			key:        "max_retries",
			value:      "abc",
			wantErr:    true,
			errContain: "invalid integer",
		},
		"valid int": {
			key:        "max_retries",
			value:      "5",
			wantParsed: 5,
			wantType:   TypeInt,
		},
		"valid duration": {
			key:        "notifications.long_running_threshold",
			value:      "5m",
			wantParsed: "5m0s",
			wantType:   TypeDuration,
		},
		"invalid duration": {
			key:        "notifications.long_running_threshold",
			value:      "abc",
			wantErr:    true,
			errContain: "invalid duration",
		},
		"valid enum": {
			key:        "notifications.type",
			value:      "sound",
			wantParsed: "sound",
			wantType:   TypeEnum,
		},
		"invalid enum": {
			key:        "notifications.type",
			value:      "invalid",
			wantErr:    true,
			errContain: "valid options: sound, visual, both",
		},
		"valid string": {
			key:        "claude_cmd",
			value:      "/usr/local/bin/claude",
			wantParsed: "/usr/local/bin/claude",
			wantType:   TypeString,
		},
		"unknown key": {
			key:        "unknown.key",
			value:      "value",
			wantErr:    true,
			errContain: "unknown configuration key",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := ValidateValue(tt.key, tt.value)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errContain != "" && !contains(err.Error(), tt.errContain) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errContain)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Parsed != tt.wantParsed {
				t.Errorf("Parsed = %v (%T), want %v (%T)", result.Parsed, result.Parsed, tt.wantParsed, tt.wantParsed)
			}
			if result.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", result.Type, tt.wantType)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
