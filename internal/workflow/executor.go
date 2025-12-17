package workflow

import (
	"fmt"

	"github.com/ariel-frischer/autospec/internal/lifecycle"
	"github.com/ariel-frischer/autospec/internal/notify"
	"github.com/ariel-frischer/autospec/internal/progress"
	"github.com/ariel-frischer/autospec/internal/retry"
	"github.com/ariel-frischer/autospec/internal/validation"
)

// Executor handles command execution with retry logic
type Executor struct {
	Claude              *ClaudeExecutor
	StateDir            string
	SpecsDir            string
	MaxRetries          int
	ProgressDisplay     *progress.ProgressDisplay // Optional progress display
	NotificationHandler *notify.Handler           // Optional notification handler for stage/command completion
	TotalStages         int                       // Total stages in workflow
	Debug               bool                      // Enable debug logging
}

// Stage represents a workflow stage (specify, plan, tasks, implement)
type Stage string

const (
	// Core workflow stages
	StageSpecify   Stage = "specify"
	StagePlan      Stage = "plan"
	StageTasks     Stage = "tasks"
	StageImplement Stage = "implement"

	// Optional stages
	StageConstitution Stage = "constitution"
	StageClarify      Stage = "clarify"
	StageChecklist    Stage = "checklist"
	StageAnalyze      Stage = "analyze"
)

// debugLog prints a debug message if debug mode is enabled
func (e *Executor) debugLog(format string, args ...interface{}) {
	if e.Debug {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// getStageNumber returns the sequential number for a stage (1-based)
// For optional stages, this returns their position in the canonical order:
// constitution(1) -> specify(2) -> clarify(3) -> plan(4) -> tasks(5) -> checklist(6) -> analyze(7) -> implement(8)
func (e *Executor) getStageNumber(stage Stage) int {
	switch stage {
	case StageConstitution:
		return 1
	case StageSpecify:
		return 2
	case StageClarify:
		return 3
	case StagePlan:
		return 4
	case StageTasks:
		return 5
	case StageChecklist:
		return 6
	case StageAnalyze:
		return 7
	case StageImplement:
		return 8
	default:
		return 0
	}
}

// buildStageInfo constructs a StageInfo from Stage enum and retry state
func (e *Executor) buildStageInfo(stage Stage, retryCount int) progress.StageInfo {
	return progress.StageInfo{
		Name:        string(stage),
		Number:      e.getStageNumber(stage),
		TotalStages: e.TotalStages,
		Status:      progress.StageInProgress,
		RetryCount:  retryCount,
		MaxRetries:  e.MaxRetries,
	}
}

// StageResult represents the result of executing a workflow stage
type StageResult struct {
	Stage      Stage
	Success    bool
	Error      error
	RetryCount int
	Exhausted  bool
}

// ExecuteStage executes a workflow stage with validation and retry logic.
// It uses lifecycle.RunStage to wrap the execution and handle stage notifications.
func (e *Executor) ExecuteStage(specName string, stage Stage, command string, validateFunc func(string) error) (*StageResult, error) {
	e.debugLog("ExecuteStage called - spec: %s, stage: %s, command: %s", specName, stage, command)
	result := &StageResult{
		Stage:   stage,
		Success: false,
	}

	// Load retry state
	retryState, err := e.loadStageRetryState(specName, stage)
	if err != nil {
		return result, err
	}

	// Build stage info and start progress display
	stageInfo := e.buildStageInfo(stage, retryState.Count)
	e.startProgressDisplay(stageInfo)

	// Use lifecycle.RunStage to wrap execution and handle stage notification
	var stageErr error
	execErr := lifecycle.RunStage(e.NotificationHandler, string(stage), func() error {
		// Display and execute command
		e.displayCommandExecution(command)
		if err := e.Claude.Execute(command); err != nil {
			stageErr = e.handleExecutionFailure(result, retryState, stageInfo, err)
			return stageErr
		}
		e.debugLog("Claude.Execute() completed successfully")

		// Validate output
		specDir := fmt.Sprintf("%s/%s", e.SpecsDir, specName)
		if err := validateFunc(specDir); err != nil {
			stageErr = e.handleValidationFailure(result, retryState, stageInfo, err)
			return stageErr
		}
		e.debugLog("Validation passed!")

		// Handle success
		e.completeStageSuccessNoNotify(result, stageInfo, specName, stage)
		return nil
	})

	// If stageErr was set by the inner function, return it
	if stageErr != nil {
		return result, stageErr
	}
	return result, execErr
}

// loadStageRetryState loads retry state for a stage
func (e *Executor) loadStageRetryState(specName string, stage Stage) (*retry.RetryState, error) {
	e.debugLog("Loading retry state from: %s", e.StateDir)
	retryState, err := retry.LoadRetryState(e.StateDir, specName, string(stage), e.MaxRetries)
	if err != nil {
		e.debugLog("Failed to load retry state: %v", err)
		return nil, fmt.Errorf("failed to load retry state: %w", err)
	}
	e.debugLog("Retry state loaded - count: %d, max: %d", retryState.Count, e.MaxRetries)
	return retryState, nil
}

// startProgressDisplay initializes progress display for a stage
func (e *Executor) startProgressDisplay(stageInfo progress.StageInfo) {
	if e.ProgressDisplay != nil {
		e.debugLog("Starting progress display")
		if err := e.ProgressDisplay.StartStage(stageInfo); err != nil {
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

// handleExecutionFailure handles command execution failure without sending stage notification.
// Stage notification is handled by lifecycle.RunStage wrapper.
func (e *Executor) handleExecutionFailure(result *StageResult, retryState *retry.RetryState, stageInfo progress.StageInfo, err error) error {
	e.debugLog("Claude.Execute() returned error: %v", err)
	result.Error = fmt.Errorf("command execution failed: %w", err)

	if e.ProgressDisplay != nil {
		e.ProgressDisplay.FailStage(stageInfo, result.Error)
	}

	// Send error notification (non-blocking) - separate from stage completion
	if e.NotificationHandler != nil {
		e.debugLog("Sending error notification for stage %s", stageInfo.Name)
		e.NotificationHandler.OnError(stageInfo.Name, result.Error)
	}

	_, retryErr := e.handleRetryIncrement(result, retryState, err, "retry limit exhausted")
	return retryErr
}

// handleValidationFailure handles validation failure without sending stage notification.
// Stage notification is handled by lifecycle.RunStage wrapper.
func (e *Executor) handleValidationFailure(result *StageResult, retryState *retry.RetryState, stageInfo progress.StageInfo, err error) error {
	e.debugLog("Validation failed: %v", err)
	result.Error = fmt.Errorf("validation failed: %w", err)

	if e.ProgressDisplay != nil {
		e.ProgressDisplay.FailStage(stageInfo, result.Error)
	}

	// Send error notification (non-blocking) - separate from stage completion
	if e.NotificationHandler != nil {
		e.debugLog("Sending error notification for stage %s validation failure", stageInfo.Name)
		e.NotificationHandler.OnError(stageInfo.Name, result.Error)
	}

	_, retryErr := e.handleRetryIncrement(result, retryState, err, "validation failed and retry exhausted")
	return retryErr
}

// handleRetryIncrement increments retry count and handles exhaustion
func (e *Executor) handleRetryIncrement(result *StageResult, retryState *retry.RetryState, originalErr error, exhaustedMsg string) (*StageResult, error) {
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

// completeStageSuccessNoNotify handles successful stage completion without sending stage notification.
// Stage notification is handled by lifecycle.RunStage wrapper.
func (e *Executor) completeStageSuccessNoNotify(result *StageResult, stageInfo progress.StageInfo, specName string, stage Stage) {
	if e.ProgressDisplay != nil {
		e.debugLog("Showing completion in progress display")
		stageInfo.Status = progress.StageCompleted
		if err := e.ProgressDisplay.CompleteStage(stageInfo); err != nil {
			fmt.Printf("Warning: progress display error: %v\n", err)
		}
	}

	e.debugLog("Resetting retry count")
	if err := retry.ResetRetryCount(e.StateDir, specName, string(stage)); err != nil {
		fmt.Printf("Warning: failed to reset retry count: %v\n", err)
	}

	result.Success = true
	result.RetryCount = 0
	e.debugLog("ExecuteStage completed successfully - returning")
}

// ExecuteWithRetry executes a command and automatically retries on failure
// This is a simplified version that doesn't require stage tracking
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

// GetRetryState retrieves the current retry state for a spec/stage
func (e *Executor) GetRetryState(specName string, stage Stage) (*retry.RetryState, error) {
	return retry.LoadRetryState(e.StateDir, specName, string(stage), e.MaxRetries)
}

// ResetStage resets the retry count for a specific stage
func (e *Executor) ResetStage(specName string, stage Stage) error {
	return retry.ResetRetryCount(e.StateDir, specName, string(stage))
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
		return fmt.Errorf("getting task stats: %w", err)
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
