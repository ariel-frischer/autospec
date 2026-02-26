// autospec - Spec-Driven Development Automation
// Author: Ariel Frischer
// Source: https://github.com/ariel-frischer/autospec

package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ariel-frischer/autospec/internal/yaml"
)

// ValidateSpecFile checks if spec.md or spec.yaml exists in the given spec directory
// Performance contract: <10ms
func ValidateSpecFile(specDir string) error {
	// Check for YAML first, then markdown
	yamlPath := filepath.Join(specDir, "spec.yaml")
	mdPath := filepath.Join(specDir, "spec.md")

	if _, err := os.Stat(yamlPath); err == nil {
		return nil // spec.yaml exists
	}
	if _, err := os.Stat(mdPath); err == nil {
		return nil // spec.md exists
	}

	return fmt.Errorf("spec file not found in %s - run 'autospec specify <description>' to create it", specDir)
}

// ValidatePlanFile checks if plan.md or plan.yaml exists in the given spec directory
// Performance contract: <10ms
func ValidatePlanFile(specDir string) error {
	// Check for YAML first, then markdown
	yamlPath := filepath.Join(specDir, "plan.yaml")
	mdPath := filepath.Join(specDir, "plan.md")

	if _, err := os.Stat(yamlPath); err == nil {
		return nil // plan.yaml exists
	}
	if _, err := os.Stat(mdPath); err == nil {
		return nil // plan.md exists
	}

	return fmt.Errorf("plan file not found in %s - run 'autospec plan' to create it", specDir)
}

// ValidateTasksFile checks if tasks.md or tasks.yaml exists in the given spec directory
// Performance contract: <10ms
func ValidateTasksFile(specDir string) error {
	// Check for YAML first, then markdown
	yamlPath := filepath.Join(specDir, "tasks.yaml")
	mdPath := filepath.Join(specDir, "tasks.md")

	if _, err := os.Stat(yamlPath); err == nil {
		return nil // tasks.yaml exists
	}
	if _, err := os.Stat(mdPath); err == nil {
		return nil // tasks.md exists
	}

	return fmt.Errorf("tasks file not found in %s - run 'autospec tasks' to create it", specDir)
}

// ValidateConstitutionFile checks if constitution.yaml exists and validates its schema.
// Checks .autospec/constitution.yaml first, falls back to .autospec/memory/constitution.yaml.
// Performance contract: <10ms
func ValidateConstitutionFile(projectDir string) error {
	// Check paths in priority order
	paths := []string{
		filepath.Join(projectDir, ".autospec", "constitution.yaml"),
		filepath.Join(projectDir, ".autospec", "memory", "constitution.yaml"),
	}

	var constitutionPath string
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			constitutionPath = p
			break
		}
	}

	if constitutionPath == "" {
		return fmt.Errorf("constitution file not found at %s - run 'autospec constitution' to create it", paths[0])
	}

	// Validate schema
	validator := &ConstitutionValidator{}
	result := validator.Validate(constitutionPath)
	if !result.Valid {
		return fmt.Errorf("constitution validation failed: %s", result.Errors[0].Message)
	}

	return nil
}

// ValidateYAMLFile validates a YAML file's syntax
// Performance contract: <100ms for 10MB files
func ValidateYAMLFile(filePath string) error {
	if !strings.HasSuffix(filePath, ".yaml") && !strings.HasSuffix(filePath, ".yml") {
		return fmt.Errorf("not a YAML file: %s", filePath)
	}
	return yaml.ValidateFile(filePath)
}

// ValidateArtifactFile validates an artifact file (markdown or YAML)
// Performance contract: <100ms
func ValidateArtifactFile(filePath string) error {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".yaml", ".yml":
		return yaml.ValidateFile(filePath)
	case ".md":
		// For markdown, just check existence
		if _, err := os.Stat(filePath); err != nil {
			return fmt.Errorf("file not found: %s", filePath)
		}
		return nil
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}
}

// Result represents the outcome of a validation check
type Result struct {
	Success            bool
	Error              string
	ContinuationPrompt string
	ArtifactPath       string
}

// ShouldRetry determines if a failed validation should be retried
func (r *Result) ShouldRetry(canRetry bool) bool {
	return !r.Success && canRetry
}

// ExitCode returns the appropriate exit code for this validation result
func (r *Result) ExitCode() int {
	if r.Success {
		return 0 // Success
	}
	if r.Error == "missing dependencies" {
		return 4 // Missing deps
	}
	if r.Error == "invalid arguments" {
		return 3 // Invalid
	}
	return 1 // Failed (retryable)
}
