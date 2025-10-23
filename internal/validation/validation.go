package validation

import (
	"fmt"
	"os"
	"path/filepath"
)

// ValidateSpecFile checks if spec.md exists in the given spec directory
// Performance contract: <10ms
func ValidateSpecFile(specDir string) error {
	specPath := filepath.Join(specDir, "spec.md")
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		return fmt.Errorf("spec.md not found in %s - run 'autospec specify <description>' to create it", specDir)
	} else if err != nil {
		return fmt.Errorf("error checking spec.md: %w", err)
	}
	return nil
}

// ValidatePlanFile checks if plan.md exists in the given spec directory
// Performance contract: <10ms
func ValidatePlanFile(specDir string) error {
	planPath := filepath.Join(specDir, "plan.md")
	if _, err := os.Stat(planPath); os.IsNotExist(err) {
		return fmt.Errorf("plan.md not found in %s - run 'autospec plan' to create it", specDir)
	} else if err != nil {
		return fmt.Errorf("error checking plan.md: %w", err)
	}
	return nil
}

// ValidateTasksFile checks if tasks.md exists in the given spec directory
// Performance contract: <10ms
func ValidateTasksFile(specDir string) error {
	tasksPath := filepath.Join(specDir, "tasks.md")
	if _, err := os.Stat(tasksPath); os.IsNotExist(err) {
		return fmt.Errorf("tasks.md not found in %s - run 'autospec tasks' to create it", specDir)
	} else if err != nil {
		return fmt.Errorf("error checking tasks.md: %w", err)
	}
	return nil
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
