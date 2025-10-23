package workflow

import (
	"fmt"

	"github.com/anthropics/auto-claude-speckit/internal/retry"
	"github.com/anthropics/auto-claude-speckit/internal/validation"
)

// Executor handles command execution with retry logic
type Executor struct {
	Claude    *ClaudeExecutor
	StateDir  string
	SpecsDir  string
	MaxRetries int
}

// Phase represents a workflow phase (specify, plan, tasks, implement)
type Phase string

const (
	PhaseSpecify   Phase = "specify"
	PhasePlan      Phase = "plan"
	PhaseTasks     Phase = "tasks"
	PhaseImplement Phase = "implement"
)

// PhaseResult represents the result of executing a workflow phase
type PhaseResult struct {
	Phase      Phase
	Success    bool
	Error      error
	RetryCount int
	Exhausted  bool
}

// ExecutePhase executes a workflow phase with validation and retry logic
func (e *Executor) ExecutePhase(specName string, phase Phase, command string, validateFunc func(string) error) (*PhaseResult, error) {
	result := &PhaseResult{
		Phase:   phase,
		Success: false,
	}

	// Load retry state
	retryState, err := retry.LoadRetryState(e.StateDir, specName, string(phase), e.MaxRetries)
	if err != nil {
		return result, fmt.Errorf("failed to load retry state: %w", err)
	}

	// Execute command
	if err := e.Claude.Execute(command); err != nil {
		result.Error = fmt.Errorf("command execution failed: %w", err)

		// Increment retry count
		if incrementErr := retryState.Increment(); incrementErr != nil {
			// Check if it's a retry exhausted error
			if exhaustedErr, ok := incrementErr.(*retry.RetryExhaustedError); ok {
				result.Exhausted = true
				result.RetryCount = exhaustedErr.Count

				// Save state before returning
				retry.SaveRetryState(e.StateDir, retryState)

				return result, fmt.Errorf("retry limit exhausted: %w", err)
			}
			return result, incrementErr
		}

		// Save incremented retry state
		if saveErr := retry.SaveRetryState(e.StateDir, retryState); saveErr != nil {
			return result, fmt.Errorf("failed to save retry state: %w", saveErr)
		}

		result.RetryCount = retryState.Count
		return result, result.Error
	}

	// Validate output
	specDir := fmt.Sprintf("%s/%s", e.SpecsDir, specName) // Simplified for now
	if err := validateFunc(specDir); err != nil {
		result.Error = fmt.Errorf("validation failed: %w", err)

		// Increment retry count
		if incrementErr := retryState.Increment(); incrementErr != nil {
			if exhaustedErr, ok := incrementErr.(*retry.RetryExhaustedError); ok {
				result.Exhausted = true
				result.RetryCount = exhaustedErr.Count
				retry.SaveRetryState(e.StateDir, retryState)
				return result, fmt.Errorf("validation failed and retry exhausted: %w", err)
			}
			return result, incrementErr
		}

		retry.SaveRetryState(e.StateDir, retryState)
		result.RetryCount = retryState.Count
		return result, result.Error
	}

	// Success! Reset retry count
	if err := retry.ResetRetryCount(e.StateDir, specName, string(phase)); err != nil {
		// Log error but don't fail - reset is not critical
		fmt.Printf("Warning: failed to reset retry count: %v\n", err)
	}

	result.Success = true
	result.RetryCount = 0
	return result, nil
}

// ExecuteWithRetry executes a command and automatically retries on failure
// This is a simplified version that doesn't require phase tracking
func (e *Executor) ExecuteWithRetry(command string, maxAttempts int) error {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := e.Claude.Execute(command)
		if err == nil {
			return nil
		}

		lastErr = err
		if attempt < maxAttempts {
			fmt.Printf("Attempt %d/%d failed: %v\nRetrying...\n", attempt, maxAttempts, err)
		}
	}

	return fmt.Errorf("all %d attempts failed: %w", maxAttempts, lastErr)
}

// GetRetryState retrieves the current retry state for a spec/phase
func (e *Executor) GetRetryState(specName string, phase Phase) (*retry.RetryState, error) {
	return retry.LoadRetryState(e.StateDir, specName, string(phase), e.MaxRetries)
}

// ResetPhase resets the retry count for a specific phase
func (e *Executor) ResetPhase(specName string, phase Phase) error {
	return retry.ResetRetryCount(e.StateDir, specName, string(phase))
}

// ValidateSpec is a convenience wrapper for spec validation
func (e *Executor) ValidateSpec(specDir string) error {
	return validation.ValidateSpecFile(specDir)
}

// ValidatePlan is a convenience wrapper for plan validation
func (e *Executor) ValidatePlan(specDir string) error {
	return validation.ValidatePlanFile(specDir)
}

// ValidateTasks is a convenience wrapper for tasks validation
func (e *Executor) ValidateTasks(specDir string) error {
	return validation.ValidateTasksFile(specDir)
}

// ValidateTasksComplete checks if all tasks are completed
func (e *Executor) ValidateTasksComplete(tasksPath string) error {
	count, err := validation.CountUncheckedTasks(tasksPath)
	if err != nil {
		return err
	}

	if count > 0 {
		return fmt.Errorf("implementation incomplete: %d unchecked tasks remain", count)
	}

	return nil
}
