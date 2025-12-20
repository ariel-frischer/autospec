// Package config_test tests configuration validation including YAML syntax, value constraints, and custom command templates.
// Related: internal/config/validate.go
// Tags: config, validation, yaml, syntax, notifications, implement-method
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateYAMLSyntax_ValidFile(t *testing.T) {
	// Create a temp file with valid YAML
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	validYAML := `agent_preset: "claude"
max_retries: 3
specs_dir: "./specs"
`
	if err := os.WriteFile(configPath, []byte(validYAML), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	err := ValidateYAMLSyntax(configPath)
	if err != nil {
		t.Errorf("ValidateYAMLSyntax() returned error for valid YAML: %v", err)
	}
}

func TestValidateYAMLSyntax_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Invalid YAML - missing colon
	invalidYAML := `agent_preset "claude"
max_retries: 3
`
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	err := ValidateYAMLSyntax(configPath)
	if err == nil {
		t.Error("ValidateYAMLSyntax() returned nil for invalid YAML")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("Expected ValidationError, got %T", err)
	}

	// Should include the file path
	if validationErr.FilePath != configPath {
		t.Errorf("ValidationError.FilePath = %q, want %q", validationErr.FilePath, configPath)
	}
}

func TestValidateYAMLSyntax_InvalidWithLineNumber(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Invalid YAML with error on line 3
	invalidYAML := `agent_preset: "claude"
max_retries: 3
specs_dir: [invalid yaml here
`
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	err := ValidateYAMLSyntax(configPath)
	if err == nil {
		t.Fatal("ValidateYAMLSyntax() returned nil for invalid YAML")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("Expected ValidationError, got %T", err)
	}

	// Should have line number > 0
	if validationErr.Line == 0 {
		t.Errorf("ValidationError.Line = 0, want > 0")
	}

	// Error string should include line number
	errStr := validationErr.Error()
	if !strings.Contains(errStr, configPath) {
		t.Errorf("Error() = %q, should contain file path %q", errStr, configPath)
	}
}

func TestValidateYAMLSyntax_MissingFile(t *testing.T) {
	err := ValidateYAMLSyntax("/nonexistent/path/config.yml")
	if err != nil {
		t.Errorf("ValidateYAMLSyntax() should return nil for missing file, got: %v", err)
	}
}

func TestValidateYAMLSyntax_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Empty file
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	err := ValidateYAMLSyntax(configPath)
	if err != nil {
		t.Errorf("ValidateYAMLSyntax() should return nil for empty file, got: %v", err)
	}
}

func TestValidateYAMLSyntax_WhitespaceOnly(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Whitespace-only file
	if err := os.WriteFile(configPath, []byte("   \n\t\n  "), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	err := ValidateYAMLSyntax(configPath)
	if err != nil {
		t.Errorf("ValidateYAMLSyntax() should return nil for whitespace-only file, got: %v", err)
	}
}

func TestValidateYAMLSyntaxFromBytes_Valid(t *testing.T) {
	validYAML := []byte(`agent_preset: "claude"
max_retries: 3
`)
	err := ValidateYAMLSyntaxFromBytes(validYAML, "test.yml")
	if err != nil {
		t.Errorf("ValidateYAMLSyntaxFromBytes() returned error for valid YAML: %v", err)
	}
}

func TestValidateYAMLSyntaxFromBytes_Invalid(t *testing.T) {
	invalidYAML := []byte(`agent_preset: [unclosed bracket
`)
	err := ValidateYAMLSyntaxFromBytes(invalidYAML, "test.yml")
	if err == nil {
		t.Error("ValidateYAMLSyntaxFromBytes() returned nil for invalid YAML")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("Expected ValidationError, got %T", err)
	}
	if validationErr.FilePath != "test.yml" {
		t.Errorf("ValidationError.FilePath = %q, want %q", validationErr.FilePath, "test.yml")
	}
}

func TestValidateYAMLSyntaxFromBytes_Empty(t *testing.T) {
	err := ValidateYAMLSyntaxFromBytes([]byte(""), "test.yml")
	if err != nil {
		t.Errorf("ValidateYAMLSyntaxFromBytes() should return nil for empty data, got: %v", err)
	}
}

func TestValidateConfigValues_Valid(t *testing.T) {
	cfg := &Configuration{
		AgentPreset: "claude",
		MaxRetries:  3,
		SpecsDir:    "./specs",
		StateDir:    "~/.autospec/state",
	}

	err := ValidateConfigValues(cfg, "test.yml")
	if err != nil {
		t.Errorf("ValidateConfigValues() returned error for valid config: %v", err)
	}
}


func TestValidateConfigValues_InvalidMaxRetries(t *testing.T) {
	tests := map[string]struct {
		maxRetries int
		wantErr    bool
	}{
		"too low":       {maxRetries: -1, wantErr: true},
		"minimum valid": {maxRetries: 0, wantErr: false},
		"middle valid":  {maxRetries: 5, wantErr: false},
		"maximum valid": {maxRetries: 10, wantErr: false},
		"too high":      {maxRetries: 11, wantErr: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := &Configuration{
				AgentPreset: "claude",
				MaxRetries:  tt.maxRetries,
				SpecsDir:    "./specs",
				StateDir:    "~/.autospec/state",
			}

			err := ValidateConfigValues(cfg, "test.yml")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfigValues() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}


func TestValidateConfigValues_ImplementMethod(t *testing.T) {
	tests := map[string]struct {
		implementMethod string
		wantErr         bool
		wantErrContains string
	}{
		"valid single-session": {
			implementMethod: "single-session",
			wantErr:         false,
		},
		"valid phases": {
			implementMethod: "phases",
			wantErr:         false,
		},
		"valid tasks": {
			implementMethod: "tasks",
			wantErr:         false,
		},
		"empty string is valid (uses default)": {
			implementMethod: "",
			wantErr:         false,
		},
		"invalid value": {
			implementMethod: "invalid-mode",
			wantErr:         true,
			wantErrContains: "single-session, phases, tasks",
		},
		"invalid value with typo": {
			implementMethod: "phase", // missing 's'
			wantErr:         true,
			wantErrContains: "single-session, phases, tasks",
		},
		"invalid value - uppercase": {
			implementMethod: "PHASES",
			wantErr:         true,
			wantErrContains: "single-session, phases, tasks",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := &Configuration{
				AgentPreset:     "claude",
				MaxRetries:      3,
				SpecsDir:        "./specs",
				StateDir:        "~/.autospec/state",
				ImplementMethod: tt.implementMethod,
			}

			err := ValidateConfigValues(cfg, "test.yml")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfigValues() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				validationErr, ok := err.(*ValidationError)
				if !ok {
					t.Fatalf("Expected ValidationError, got %T", err)
				}

				if validationErr.Field != "implement_method" {
					t.Errorf("ValidationError.Field = %q, want %q", validationErr.Field, "implement_method")
				}

				if tt.wantErrContains != "" && !strings.Contains(validationErr.Message, tt.wantErrContains) {
					t.Errorf("ValidationError.Message = %q, should contain %q", validationErr.Message, tt.wantErrContains)
				}
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := map[string]struct {
		err      *ValidationError
		contains []string
	}{
		"with line and column": {
			err: &ValidationError{
				FilePath: "/path/to/config.yml",
				Line:     5,
				Column:   10,
				Message:  "unexpected character",
			},
			contains: []string{"/path/to/config.yml", "5", "10", "unexpected character"},
		},
		"with field": {
			err: &ValidationError{
				FilePath: "/path/to/config.yml",
				Field:    "max_retries",
				Message:  "must be at least 1",
			},
			contains: []string{"/path/to/config.yml", "max_retries", "must be at least 1"},
		},
		"message only": {
			err: &ValidationError{
				FilePath: "/path/to/config.yml",
				Message:  "general error",
			},
			contains: []string{"/path/to/config.yml", "general error"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, want := range tt.contains {
				if !strings.Contains(errStr, want) {
					t.Errorf("Error() = %q, should contain %q", errStr, want)
				}
			}
		})
	}
}

func TestValidateNotificationConfig_InvalidType(t *testing.T) {
	t.Parallel()

	cfg := &Configuration{
		AgentPreset: "claude",
		MaxRetries:  3,
		SpecsDir:    "./specs",
		StateDir:    "~/.autospec/state",
	}
	cfg.Notifications.Type = "invalid-type"

	err := ValidateConfigValues(cfg, "test.yml")
	if err == nil {
		t.Error("ValidateConfigValues() returned nil for invalid notification type")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("Expected ValidationError, got %T", err)
	}

	if validationErr.Field != "notifications.type" {
		t.Errorf("ValidationError.Field = %q, want %q", validationErr.Field, "notifications.type")
	}
}

func TestValidateNotificationConfig_NonExistentSoundFile(t *testing.T) {
	t.Parallel()

	cfg := &Configuration{
		AgentPreset: "claude",
		MaxRetries:  3,
		SpecsDir:    "./specs",
		StateDir:    "~/.autospec/state",
	}
	cfg.Notifications.SoundFile = "/nonexistent/path/to/sound.wav"

	err := ValidateConfigValues(cfg, "test.yml")
	if err == nil {
		t.Error("ValidateConfigValues() returned nil for nonexistent sound file")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("Expected ValidationError, got %T", err)
	}

	if validationErr.Field != "notifications.sound_file" {
		t.Errorf("ValidationError.Field = %q, want %q", validationErr.Field, "notifications.sound_file")
	}

	if !strings.Contains(validationErr.Message, "does not exist") {
		t.Errorf("ValidationError.Message = %q, should contain 'does not exist'", validationErr.Message)
	}
}

func TestValidateNotificationConfig_ValidSoundFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	soundPath := filepath.Join(tmpDir, "sound.wav")
	if err := os.WriteFile(soundPath, []byte("fake wav data"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cfg := &Configuration{
		AgentPreset: "claude",
		MaxRetries:  3,
		SpecsDir:    "./specs",
		StateDir:    "~/.autospec/state",
	}
	cfg.Notifications.SoundFile = soundPath

	err := ValidateConfigValues(cfg, "test.yml")
	if err != nil {
		t.Errorf("ValidateConfigValues() returned error for valid sound file: %v", err)
	}
}

func TestExtractLineColumn(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		errMsg     string
		wantLine   int
		wantColumn int
	}{
		"yaml error with line and column": {
			errMsg:     "yaml: line 5: column 10: unexpected character",
			wantLine:   5,
			wantColumn: 10,
		},
		"yaml error with line only": {
			errMsg:     "yaml: line 3: could not find expected ':'",
			wantLine:   3,
			wantColumn: 1,
		},
		"non-yaml error": {
			errMsg:     "some other error",
			wantLine:   0,
			wantColumn: 0,
		},
		"empty string": {
			errMsg:     "",
			wantLine:   0,
			wantColumn: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			line, column := extractLineColumn(tt.errMsg)
			if line != tt.wantLine {
				t.Errorf("extractLineColumn() line = %d, want %d", line, tt.wantLine)
			}
			if column != tt.wantColumn {
				t.Errorf("extractLineColumn() column = %d, want %d", column, tt.wantColumn)
			}
		})
	}
}

func TestCleanYAMLError(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		errMsg string
		want   string
	}{
		"yaml error with prefix": {
			errMsg: "yaml: line 5: could not find expected ':'",
			want:   "could not find expected ':'",
		},
		"non-yaml error": {
			errMsg: "some other error",
			want:   "some other error",
		},
		"empty string": {
			errMsg: "",
			want:   "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := cleanYAMLError(tt.errMsg)
			if got != tt.want {
				t.Errorf("cleanYAMLError(%q) = %q, want %q", tt.errMsg, got, tt.want)
			}
		})
	}
}

func TestValidateYAMLSyntax_PermissionError(t *testing.T) {
	t.Parallel()

	// Skip on Windows where permissions work differently
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Create a file with valid YAML content
	if err := os.WriteFile(configPath, []byte("key: value"), 0000); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Ensure cleanup can remove the file
	defer os.Chmod(configPath, 0644)

	err := ValidateYAMLSyntax(configPath)
	if err == nil {
		// If we're running as root, the permission check won't fail
		t.Skip("Running as root, permission check won't fail")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("Expected ValidationError, got %T", err)
	}

	if !strings.Contains(validationErr.Message, "permission denied") {
		t.Errorf("ValidationError.Message = %q, should contain 'permission denied'", validationErr.Message)
	}
}

func TestValidateConfigValues_MissingSpecsDir(t *testing.T) {
	t.Parallel()

	cfg := &Configuration{
		AgentPreset: "claude",
		MaxRetries:  3,
		SpecsDir:    "", // Missing
		StateDir:    "~/.autospec/state",
	}

	err := ValidateConfigValues(cfg, "test.yml")
	if err == nil {
		t.Error("ValidateConfigValues() returned nil for missing specs_dir")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("Expected ValidationError, got %T", err)
	}

	if validationErr.Field != "specs_dir" {
		t.Errorf("ValidationError.Field = %q, want %q", validationErr.Field, "specs_dir")
	}
}

func TestValidateConfigValues_MissingStateDir(t *testing.T) {
	t.Parallel()

	cfg := &Configuration{
		AgentPreset: "claude",
		MaxRetries:  3,
		SpecsDir:    "./specs",
		StateDir:    "", // Missing
	}

	err := ValidateConfigValues(cfg, "test.yml")
	if err == nil {
		t.Error("ValidateConfigValues() returned nil for missing state_dir")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("Expected ValidationError, got %T", err)
	}

	if validationErr.Field != "state_dir" {
		t.Errorf("ValidationError.Field = %q, want %q", validationErr.Field, "state_dir")
	}
}

func TestValidateYAMLSyntax_TypeErrorBranch(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Write YAML that will unmarshal correctly but has complex structure
	// This test ensures that we handle the case when yaml errors might have different formats
	complexYAML := `
key1: value1
key2:
  - item1
  - item2
key3:
  nested: value
`
	if err := os.WriteFile(configPath, []byte(complexYAML), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	err := ValidateYAMLSyntax(configPath)
	if err != nil {
		t.Errorf("ValidateYAMLSyntax() returned error for valid complex YAML: %v", err)
	}
}
