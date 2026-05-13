package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ariel-frischer/autospec/internal/notify"
	"github.com/ariel-frischer/autospec/internal/verification"
	"gopkg.in/yaml.v3"
)

// ValidationError represents a configuration validation error with context
type ValidationError struct {
	FilePath string
	Line     int
	Column   int
	Message  string
	Field    string
}

func (e *ValidationError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d:%d: %s", e.FilePath, e.Line, e.Column, e.Message)
	}
	if e.Field != "" {
		return fmt.Sprintf("%s: field '%s': %s", e.FilePath, e.Field, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.FilePath, e.Message)
}

// ValidateYAMLSyntax checks if the YAML file has valid syntax.
// Returns nil if valid, or a ValidationError with line/column information if invalid.
func ValidateYAMLSyntax(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Missing file is not an error - will use defaults
		}
		if os.IsPermission(err) {
			return &ValidationError{
				FilePath: filePath,
				Message:  "permission denied",
			}
		}
		return &ValidationError{
			FilePath: filePath,
			Message:  err.Error(),
		}
	}

	// Empty file is valid - will use defaults
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil
	}

	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		var typeError *yaml.TypeError
		if errors.As(err, &typeError) {
			// yaml.TypeError contains multiple error strings
			return &ValidationError{
				FilePath: filePath,
				Message:  strings.Join(typeError.Errors, "; "),
			}
		}

		// Try to extract line/column from yaml error message
		// yaml.v3 errors typically include "line X" information
		line, column := extractLineColumn(err.Error())
		return &ValidationError{
			FilePath: filePath,
			Line:     line,
			Column:   column,
			Message:  cleanYAMLError(err.Error()),
		}
	}

	return nil
}

// ValidateYAMLSyntaxFromBytes checks if YAML data has valid syntax.
// Returns nil if valid, or a ValidationError if invalid.
func ValidateYAMLSyntaxFromBytes(data []byte, filePath string) error {
	// Empty data is valid - will use defaults
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil
	}

	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		line, column := extractLineColumn(err.Error())
		return &ValidationError{
			FilePath: filePath,
			Line:     line,
			Column:   column,
			Message:  cleanYAMLError(err.Error()),
		}
	}

	return nil
}

// ValidateConfigValues validates configuration values against expected types and constraints.
// Returns nil if valid, or a ValidationError with field information if invalid.
func ValidateConfigValues(cfg *Configuration, filePath string) error {
	// Required string fields
	if cfg.SpecsDir == "" {
		return &ValidationError{
			FilePath: filePath,
			Field:    "specs_dir",
			Message:  "is required",
		}
	}
	if cfg.StateDir == "" {
		return &ValidationError{
			FilePath: filePath,
			Field:    "state_dir",
			Message:  "is required",
		}
	}

	// MaxRetries: min=0, max=10
	if cfg.MaxRetries < 0 || cfg.MaxRetries > 10 {
		return &ValidationError{
			FilePath: filePath,
			Field:    "max_retries",
			Message:  "must be between 0 and 10",
		}
	}

	// Timeout: omitempty, min=1, max=604800 (0 means no timeout)
	if cfg.Timeout != 0 && (cfg.Timeout < 1 || cfg.Timeout > 604800) {
		return &ValidationError{
			FilePath: filePath,
			Field:    "timeout",
			Message:  "must be between 1 and 604800 (or 0 for no timeout)",
		}
	}

	// ImplementMethod: must be one of "single-session", "phases", "tasks", or empty (uses default)
	if cfg.ImplementMethod != "" {
		validMethods := []string{"single-session", "phases", "tasks"}
		isValid := false
		for _, m := range validMethods {
			if cfg.ImplementMethod == m {
				isValid = true
				break
			}
		}
		if !isValid {
			return &ValidationError{
				FilePath: filePath,
				Field:    "implement_method",
				Message:  "must be one of: single-session, phases, tasks",
			}
		}
	}

	// Validate notification settings
	if err := validateNotificationConfig(&cfg.Notifications, filePath); err != nil {
		return err
	}

	// Validate cclean.style if specified
	if cfg.Cclean.Style != "" && cfg.Cclean.Style != "default" {
		if err := ValidateOutputStyle(cfg.Cclean.Style); err != nil {
			return &ValidationError{
				FilePath: filePath,
				Field:    "cclean.style",
				Message:  err.Error(),
			}
		}
	}

	// Validate verification config
	if err := validateVerificationConfig(&cfg.Verification, filePath); err != nil {
		return err
	}

	return nil
}

// validateNotificationConfig validates notification configuration values.
// Returns nil if valid, or a ValidationError with field information if invalid.
func validateNotificationConfig(nc *notify.NotificationConfig, filePath string) error {
	// Validate Type: must be one of sound, visual, both (or empty for default)
	if nc.Type != "" && !notify.ValidOutputType(string(nc.Type)) {
		return &ValidationError{
			FilePath: filePath,
			Field:    "notifications.type",
			Message:  "must be one of: sound, visual, both",
		}
	}

	// Validate SoundFile: if specified, must exist
	if nc.SoundFile != "" {
		if _, err := os.Stat(nc.SoundFile); err != nil {
			if os.IsNotExist(err) {
				return &ValidationError{
					FilePath: filePath,
					Field:    "notifications.sound_file",
					Message:  fmt.Sprintf("file does not exist: %s", nc.SoundFile),
				}
			}
			// Permission or other errors
			return &ValidationError{
				FilePath: filePath,
				Field:    "notifications.sound_file",
				Message:  fmt.Sprintf("cannot access file: %s", err),
			}
		}
	}

	// Note: LongRunningThreshold of 0 or negative is valid and means "always notify"
	// This is documented behavior per the spec, so no validation error is needed.

	return nil
}

// extractLineColumn attempts to extract line and column numbers from a YAML error message.
// Returns 0, 0 if unable to extract.
func extractLineColumn(errMsg string) (line, column int) {
	// yaml.v3 errors look like: "yaml: line 5: could not find expected ':'"
	var l, c int
	if n, _ := fmt.Sscanf(errMsg, "yaml: line %d: column %d:", &l, &c); n == 2 {
		return l, c
	}
	if n, _ := fmt.Sscanf(errMsg, "yaml: line %d:", &l); n == 1 {
		return l, 1
	}
	return 0, 0
}

// cleanYAMLError removes the "yaml: line X:" prefix from error messages for cleaner output.
func cleanYAMLError(errMsg string) string {
	// Remove "yaml: line X:" prefix
	if idx := strings.LastIndex(errMsg, ": "); idx > 0 {
		// Check if this looks like a yaml error
		if strings.HasPrefix(errMsg, "yaml:") {
			return errMsg[idx+2:]
		}
	}
	return errMsg
}

// validateVerificationConfig validates verification configuration values.
func validateVerificationConfig(vc *verification.VerificationConfig, filePath string) error {
	if vc.Level != "" && !vc.Level.IsValid() {
		return &ValidationError{
			FilePath: filePath,
			Field:    "verification.level",
			Message:  "must be one of: basic, enhanced, full",
		}
	}

	if vc.MutationThreshold < 0.0 || vc.MutationThreshold > 1.0 {
		return &ValidationError{
			FilePath: filePath,
			Field:    "verification.mutation_threshold",
			Message:  "must be between 0.0 and 1.0",
		}
	}

	if vc.CoverageThreshold < 0.0 || vc.CoverageThreshold > 1.0 {
		return &ValidationError{
			FilePath: filePath,
			Field:    "verification.coverage_threshold",
			Message:  "must be between 0.0 and 1.0",
		}
	}

	if vc.ComplexityMax < 0 {
		return &ValidationError{
			FilePath: filePath,
			Field:    "verification.complexity_max",
			Message:  "must be a positive integer",
		}
	}

	return nil
}
