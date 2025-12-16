package workflow

import (
	"fmt"

	"github.com/ariel-frischer/autospec/internal/progress"
	"github.com/ariel-frischer/autospec/internal/retry"
	"github.com/ariel-frischer/autospec/internal/validation"
)

// Executor handles command execution with retry logic
type Executor struct {
	Claude          *ClaudeExecutor
	StateDir        string
	SpecsDir        string
	MaxRetries      int
	ProgressDisplay *progress.ProgressDisplay // Optional progress display
	TotalPhases     int                       // Total phases in workflow
	Debug           bool                      // Enable debug logging
}

// Phase represents a workflow phase (specify, plan, tasks, implement)
type Phase string

const (
	// Core workflow phases
	PhaseSpecify   Phase = "specify"
	PhasePlan      Phase = "plan"
	PhaseTasks     Phase = "tasks"
	PhaseImplement Phase = "implement"

	// Optional phases
	PhaseConstitution Phase = "constitution"
	PhaseClarify      Phase = "clarify"
	PhaseChecklist    Phase = "checklist"
	PhaseAnalyze      Phase = "analyze"
)

// debugLog prints a debug message if debug mode is enabled
func (e *Executor) debugLog(format string, args ...interface{}) {
	if e.Debug {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// getPhaseNumber returns the sequential number for a phase (1-based)
// For optional phases, this returns their position in the canonical order:
// constitution(1) -> specify(2) -> clarify(3) -> plan(4) -> tasks(5) -> checklist(6) -> analyze(7) -> implement(8)
func (e *Executor) getPhaseNumber(phase Phase) int {
	switch phase {
	case PhaseConstitution:
		return 1
	case PhaseSpecify:
		return 2
	case PhaseClarify:
		return 3
	case PhasePlan:
		return 4
	case PhaseTasks:
		return 5
	case PhaseChecklist:
		return 6
	case PhaseAnalyze:
		return 7
	case PhaseImplement:
		return 8
	default:
		return 0
	}
}

// buildPhaseInfo constructs a PhaseInfo from Phase enum and retry state
func (e *Executor) buildPhaseInfo(phase Phase, retryCount int) progress.PhaseInfo {
	return progress.PhaseInfo{
		Name:        string(phase),
		Number:      e.getPhaseNumber(phase),
		TotalPhases: e.TotalPhases,
		Status:      progress.PhaseInProgress,
		RetryCount:  retryCount,
		MaxRetries:  e.MaxRetries,
	}
}

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
	e.debugLog("ExecutePhase called - spec: %s, phase: %s, command: %s", specName, phase, command)
	result := &PhaseResult{Phase: phase, Success: false}

	// Load retry state
	retryState, err := e.loadPhaseRetryState(specName, phase)
	if err != nil {
		return result, err
	}

	// Build phase info and start progress display
	phaseInfo := e.buildPhaseInfo(phase, retryState.Count)
	e.startProgressDisplay(phaseInfo)

	// Display and execute command
	e.displayCommandExecution(command)
	if err := e.Claude.Execute(command); err != nil {
		return e.handleExecutionError(result, retryState, phaseInfo, err)
	}
	e.debugLog("Claude.Execute() completed successfully")

	// Validate output
	specDir := fmt.Sprintf("%s/%s", e.SpecsDir, specName)
	if err := validateFunc(specDir); err != nil {
		return e.handleValidationError(result, retryState, phaseInfo, err)
	}
	e.debugLog("Validation passed!")

	// Handle success
	e.completePhaseSuccess(result, phaseInfo, specName, phase)
	return result, nil
}

// loadPhaseRetryState loads retry state for a phase
func (e *Executor) loadPhaseRetryState(specName string, phase Phase) (*retry.RetryState, error) {
	e.debugLog("Loading retry state from: %s", e.StateDir)
	retryState, err := retry.LoadRetryState(e.StateDir, specName, string(phase), e.MaxRetries)
	if err != nil {
		e.debugLog("Failed to load retry state: %v", err)
		return nil, fmt.Errorf("failed to load retry state: %w", err)
	}
	e.debugLog("Retry state loaded - count: %d, max: %d", retryState.Count, e.MaxRetries)
	return retryState, nil
}

// startProgressDisplay initializes progress display for a phase
func (e *Executor) startProgressDisplay(phaseInfo progress.PhaseInfo) {
	if e.ProgressDisplay != nil {
		e.debugLog("Starting progress display")
		if err := e.ProgressDisplay.StartPhase(phaseInfo); err != nil {
			fmt.Printf("Warning: progress display error: %v\n", err)
		}
	}
}

// displayCommandExecution shows the command being executed
func (e *Executor) displayCommandExecution(command string) {
	fullCommand := e.Claude.FormatCommand(command)
	fmt.Printf("\n→ Executing: %s\n\n", fullCommand)
	e.debugLog("About to call Claude.Execute()")
}

// handleExecutionError handles command execution failure
func (e *Executor) handleExecutionError(result *PhaseResult, retryState *retry.RetryState, phaseInfo progress.PhaseInfo, err error) (*PhaseResult, error) {
	e.debugLog("Claude.Execute() returned error: %v", err)
	result.Error = fmt.Errorf("command execution failed: %w", err)

	if e.ProgressDisplay != nil {
		e.ProgressDisplay.FailPhase(phaseInfo, result.Error)
	}

	return e.handleRetryIncrement(result, retryState, err, "retry limit exhausted")
}

// handleValidationError handles validation failure
func (e *Executor) handleValidationError(result *PhaseResult, retryState *retry.RetryState, phaseInfo progress.PhaseInfo, err error) (*PhaseResult, error) {
	e.debugLog("Validation failed: %v", err)
	result.Error = fmt.Errorf("validation failed: %w", err)

	if e.ProgressDisplay != nil {
		e.ProgressDisplay.FailPhase(phaseInfo, result.Error)
	}

	return e.handleRetryIncrement(result, retryState, err, "validation failed and retry exhausted")
}

// handleRetryIncrement increments retry count and handles exhaustion
func (e *Executor) handleRetryIncrement(result *PhaseResult, retryState *retry.RetryState, originalErr error, exhaustedMsg string) (*PhaseResult, error) {
	if incrementErr := retryState.Increment(); incrementErr != nil {
		if exhaustedErr, ok := incrementErr.(*retry.RetryExhaustedError); ok {
			result.Exhausted = true
			result.RetryCount = exhaustedErr.Count
			retry.SaveRetryState(e.StateDir, retryState)
			return result, fmt.Errorf("%s: %w", exhaustedMsg, originalErr)
		}
		return result, incrementErr
	}

	if saveErr := retry.SaveRetryState(e.StateDir, retryState); saveErr != nil {
		return result, fmt.Errorf("failed to save retry state: %w", saveErr)
	}

	result.RetryCount = retryState.Count
	return result, result.Error
}

// completePhaseSuccess handles successful phase completion
func (e *Executor) completePhaseSuccess(result *PhaseResult, phaseInfo progress.PhaseInfo, specName string, phase Phase) {
	if e.ProgressDisplay != nil {
		e.debugLog("Showing completion in progress display")
		phaseInfo.Status = progress.PhaseCompleted
		if err := e.ProgressDisplay.CompletePhase(phaseInfo); err != nil {
			fmt.Printf("Warning: progress display error: %v\n", err)
		}
	}

	e.debugLog("Resetting retry count")
	if err := retry.ResetRetryCount(e.StateDir, specName, string(phase)); err != nil {
		fmt.Printf("Warning: failed to reset retry count: %v\n", err)
	}

	result.Success = true
	result.RetryCount = 0
	e.debugLog("ExecutePhase completed successfully - returning")
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
// Supports both YAML (status field) and Markdown (checkbox) formats
func (e *Executor) ValidateTasksComplete(tasksPath string) error {
	stats, err := validation.GetTaskStats(tasksPath)
	if err != nil {
		return err
	}

	if !stats.IsComplete() {
		remaining := stats.PendingTasks + stats.InProgressTasks + stats.BlockedTasks
		if stats.BlockedTasks > 0 {
			return fmt.Errorf("implementation incomplete: %d tasks remain (%d pending, %d in-progress, %d blocked)",
				remaining, stats.PendingTasks, stats.InProgressTasks, stats.BlockedTasks)
		}
		return fmt.Errorf("implementation incomplete: %d tasks remain (%d pending, %d in-progress)",
			remaining, stats.PendingTasks, stats.InProgressTasks)
	}

	return nil
}
