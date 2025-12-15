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

	validYAML := `claude_cmd: "claude"
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
	invalidYAML := `claude_cmd "claude"
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
	invalidYAML := `claude_cmd: "claude"
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
	validYAML := []byte(`claude_cmd: "claude"
max_retries: 3
`)
	err := ValidateYAMLSyntaxFromBytes(validYAML, "test.yml")
	if err != nil {
		t.Errorf("ValidateYAMLSyntaxFromBytes() returned error for valid YAML: %v", err)
	}
}

func TestValidateYAMLSyntaxFromBytes_Invalid(t *testing.T) {
	invalidYAML := []byte(`claude_cmd: [unclosed bracket
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
		ClaudeCmd:  "claude",
		MaxRetries: 3,
		SpecsDir:   "./specs",
		StateDir:   "~/.autospec/state",
	}

	err := ValidateConfigValues(cfg, "test.yml")
	if err != nil {
		t.Errorf("ValidateConfigValues() returned error for valid config: %v", err)
	}
}

func TestValidateConfigValues_MissingRequired(t *testing.T) {
	cfg := &Configuration{
		ClaudeCmd:  "", // Missing required field
		MaxRetries: 3,
		SpecsDir:   "./specs",
		StateDir:   "~/.autospec/state",
	}

	err := ValidateConfigValues(cfg, "test.yml")
	if err == nil {
		t.Error("ValidateConfigValues() returned nil for config with missing required field")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("Expected ValidationError, got %T", err)
	}

	if validationErr.Field != "claude_cmd" {
		t.Errorf("ValidationError.Field = %q, want %q", validationErr.Field, "claude_cmd")
	}

	if !strings.Contains(validationErr.Message, "required") {
		t.Errorf("ValidationError.Message = %q, should contain 'required'", validationErr.Message)
	}
}

func TestValidateConfigValues_InvalidMaxRetries(t *testing.T) {
	tests := []struct {
		name       string
		maxRetries int
		wantErr    bool
	}{
		{"too low", 0, true},
		{"minimum valid", 1, false},
		{"middle valid", 5, false},
		{"maximum valid", 10, false},
		{"too high", 11, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Configuration{
				ClaudeCmd:  "claude",
				SpecifyCmd: "specify",
				MaxRetries: tt.maxRetries,
				SpecsDir:   "./specs",
				StateDir:   "~/.autospec/state",
			}

			err := ValidateConfigValues(cfg, "test.yml")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfigValues() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConfigValues_InvalidCustomClaudeCmd(t *testing.T) {
	cfg := &Configuration{
		ClaudeCmd:       "claude",
		SpecifyCmd:      "specify",
		MaxRetries:      3,
		SpecsDir:        "./specs",
		StateDir:        "~/.autospec/state",
		CustomClaudeCmd: "my-claude-wrapper", // Missing {{PROMPT}}
	}

	err := ValidateConfigValues(cfg, "test.yml")
	if err == nil {
		t.Error("ValidateConfigValues() returned nil for custom_claude_cmd without {{PROMPT}}")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("Expected ValidationError, got %T", err)
	}

	if validationErr.Field != "custom_claude_cmd" {
		t.Errorf("ValidationError.Field = %q, want %q", validationErr.Field, "custom_claude_cmd")
	}

	if !strings.Contains(validationErr.Message, "{{PROMPT}}") {
		t.Errorf("ValidationError.Message = %q, should mention {{PROMPT}}", validationErr.Message)
	}
}

func TestValidateConfigValues_ValidCustomClaudeCmd(t *testing.T) {
	cfg := &Configuration{
		ClaudeCmd:       "claude",
		SpecifyCmd:      "specify",
		MaxRetries:      3,
		SpecsDir:        "./specs",
		StateDir:        "~/.autospec/state",
		CustomClaudeCmd: "my-claude-wrapper {{PROMPT}} --verbose",
	}

	err := ValidateConfigValues(cfg, "test.yml")
	if err != nil {
		t.Errorf("ValidateConfigValues() returned error for valid custom_claude_cmd: %v", err)
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ValidationError
		contains []string
	}{
		{
			name: "with line and column",
			err: &ValidationError{
				FilePath: "/path/to/config.yml",
				Line:     5,
				Column:   10,
				Message:  "unexpected character",
			},
			contains: []string{"/path/to/config.yml", "5", "10", "unexpected character"},
		},
		{
			name: "with field",
			err: &ValidationError{
				FilePath: "/path/to/config.yml",
				Field:    "max_retries",
				Message:  "must be at least 1",
			},
			contains: []string{"/path/to/config.yml", "max_retries", "must be at least 1"},
		},
		{
			name: "message only",
			err: &ValidationError{
				FilePath: "/path/to/config.yml",
				Message:  "general error",
			},
			contains: []string{"/path/to/config.yml", "general error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, want := range tt.contains {
				if !strings.Contains(errStr, want) {
					t.Errorf("Error() = %q, should contain %q", errStr, want)
				}
			}
		})
	}
}
