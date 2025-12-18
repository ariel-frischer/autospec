package workflow

import (
	"fmt"
	"strings"

	"github.com/ariel-frischer/autospec/internal/lifecycle"
	"github.com/ariel-frischer/autospec/internal/notify"
	"github.com/ariel-frischer/autospec/internal/progress"
	"github.com/ariel-frischer/autospec/internal/retry"
	"github.com/ariel-frischer/autospec/internal/validation"
)

// Executor handles command execution with retry logic.
// It composes ClaudeRunner, ProgressController, and NotifyDispatcher
// to separate execution, display, and notification concerns.
//
// Design rationale: By accepting ClaudeRunner interface instead of
// *ClaudeExecutor, tests can inject mock implementations to verify
// execution behavior without actual Claude CLI invocations.
type Executor struct {
	Claude              ClaudeRunner              // Interface for Claude command execution (allows mocking)
	StateDir            string                    // Directory for retry state storage
	SpecsDir            string                    // Directory for spec files
	MaxRetries          int                       // Maximum retry attempts (1-10 range)
	TotalStages         int                       // Total stages in workflow
	Debug               bool                      // Enable debug logging
	Progress            *ProgressController       // Optional progress display controller
	Notify              *NotifyDispatcher         // Optional notification dispatcher
	ProgressDisplay     *progress.ProgressDisplay // Deprecated: use Progress instead
	NotificationHandler *notify.Handler           // Deprecated: use Notify instead
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
	Stage            Stage
	Success          bool
	Error            error
	RetryCount       int
	Exhausted        bool
	ValidationErrors []string // Schema validation errors for retry context
}

// ExecuteStage executes a workflow stage with validation and retry logic.
// It uses lifecycle.RunStage to wrap the execution and handle stage notifications.
// On validation failure, it retries with error context injected into the command.
//
// State machine flow:
//  1. Load retry state → 2. Execute command → 3. Validate output
//     4a. Success: persist state, return
//     4b. Execution error: return immediately (unrecoverable)
//     4c. Validation error: check retries remaining
//     - If retries available: inject errors into command, loop back to step 2
//     - If exhausted: mark result.Exhausted=true, return error
//
// The retry mechanism injects validation errors into subsequent commands,
// allowing Claude to self-correct based on previous failures.
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

	currentCommand := command
	var lastValidationErrors []string

	// Retry loop - continues while retries are available
	for {
		// Build stage info and start progress display
		stageInfo := e.buildStageInfo(stage, retryState.Count)
		e.startProgressDisplay(stageInfo)

		// Use lifecycle.RunStage to wrap execution and handle stage notification
		var stageErr error
		var validationErr error

		_ = lifecycle.RunStage(e.NotificationHandler, string(stage), func() error {
			// Display and execute command
			e.displayCommandExecution(currentCommand)
			if err := e.Claude.Execute(currentCommand); err != nil {
				stageErr = e.handleExecutionFailure(result, retryState, stageInfo, err)
				return stageErr
			}
			e.debugLog("Claude.Execute() completed successfully")

			// Validate output
			specDir := fmt.Sprintf("%s/%s", e.SpecsDir, specName)
			if err := validateFunc(specDir); err != nil {
				validationErr = err
				result.ValidationErrors = ExtractValidationErrors(err)
				lastValidationErrors = result.ValidationErrors
				e.debugLog("Validation failed: %v", err)
				return err
			}
			e.debugLog("Validation passed!")

			// Handle success
			e.completeStageSuccessNoNotify(result, stageInfo, specName, stage)
			return nil
		})

		// If execution failed (not validation), return immediately
		if stageErr != nil {
			return result, stageErr
		}

		// If validation passed, we're done
		if validationErr == nil {
			return result, nil
		}

		// Validation failed - check if we can retry
		if !retryState.CanRetry() {
			// No more retries - fail with exhausted state
			result.Exhausted = true
			result.RetryCount = retryState.Count
			result.Error = fmt.Errorf("validation failed: %w", validationErr)
			e.failStageProgress(stageInfo, result.Error)
			return result, fmt.Errorf("validation failed and retry exhausted: %w", validationErr)
		}

		// Increment retry count
		if err := retryState.Increment(); err != nil {
			return result, fmt.Errorf("failed to increment retry: %w", err)
		}
		if err := retry.SaveRetryState(e.StateDir, retryState); err != nil {
			return result, fmt.Errorf("failed to save retry state: %w", err)
		}

		// Build retry command with error context
		retryContext := FormatRetryContext(retryState.Count, e.MaxRetries, lastValidationErrors)
		currentCommand = BuildRetryCommand(command, retryContext, "")
		result.RetryCount = retryState.Count

		e.debugLog("Retrying (attempt %d/%d) with error context", retryState.Count, e.MaxRetries)
		fmt.Printf("\n⟳ Retry %d/%d - injecting validation errors into command\n", retryState.Count, e.MaxRetries)
	}
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

// startProgressDisplay initializes progress display for a stage.
// Uses Progress controller if set, falls back to deprecated ProgressDisplay field.
func (e *Executor) startProgressDisplay(stageInfo progress.StageInfo) {
	e.debugLog("Starting progress display")

	// Prefer new Progress controller
	if e.Progress != nil {
		if err := e.Progress.StartStage(stageInfo); err != nil {
			fmt.Printf("Warning: progress display error: %v\n", err)
		}
		return
	}

	// Fallback to deprecated ProgressDisplay field
	if e.ProgressDisplay != nil {
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
// Uses Progress/Notify controllers if set, falls back to deprecated fields.
func (e *Executor) handleExecutionFailure(result *StageResult, retryState *retry.RetryState, stageInfo progress.StageInfo, err error) error {
	e.debugLog("Claude.Execute() returned error: %v", err)
	result.Error = fmt.Errorf("command execution failed: %w", err)

	// Fail stage in progress display
	e.failStageProgress(stageInfo, result.Error)

	// Send error notification (non-blocking)
	e.sendErrorNotification(stageInfo.Name, result.Error)

	_, retryErr := e.handleRetryIncrement(result, retryState, err, "retry limit exhausted")
	return retryErr
}

// handleValidationFailure handles validation failure without sending stage notification.
// Stage notification is handled by lifecycle.RunStage wrapper.
// It extracts validation errors and stores them in StageResult for retry context.
// Uses Progress/Notify controllers if set, falls back to deprecated fields.
func (e *Executor) handleValidationFailure(result *StageResult, retryState *retry.RetryState, stageInfo progress.StageInfo, err error) error {
	e.debugLog("Validation failed: %v", err)
	result.Error = fmt.Errorf("validation failed: %w", err)

	// Extract and store validation errors for retry context
	result.ValidationErrors = ExtractValidationErrors(err)
	e.debugLog("Extracted %d validation errors for retry context", len(result.ValidationErrors))

	// Fail stage in progress display
	e.failStageProgress(stageInfo, result.Error)

	// Send error notification (non-blocking)
	e.sendErrorNotification(stageInfo.Name, result.Error)

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
// Uses Progress controller if set, falls back to deprecated ProgressDisplay field.
func (e *Executor) completeStageSuccessNoNotify(result *StageResult, stageInfo progress.StageInfo, specName string, stage Stage) {
	e.debugLog("Showing completion in progress display")
	stageInfo.Status = progress.StageCompleted

	// Complete stage in progress display
	e.completeStageProgress(stageInfo)

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

// maxRetryErrors is the maximum number of validation errors to include in retry context
const maxRetryErrors = 10

// FormatRetryContext creates a standardized retry context string from validation errors.
// Format: 'RETRY X/Y\nSchema validation failed:\n- error1\n- error2'
//
// Truncation logic: if >10 errors, shows first 10 + "...and N more errors".
// This prevents overwhelming Claude with too much error context while still
// conveying the scope of the problem.
func FormatRetryContext(attemptNum, maxRetries int, validationErrors []string) string {
	if len(validationErrors) == 0 {
		return fmt.Sprintf("RETRY %d/%d", attemptNum, maxRetries)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("RETRY %d/%d\n", attemptNum, maxRetries))
	sb.WriteString("Schema validation failed:\n")

	errorsToShow := validationErrors
	remaining := 0
	if len(validationErrors) > maxRetryErrors {
		errorsToShow = validationErrors[:maxRetryErrors]
		remaining = len(validationErrors) - maxRetryErrors
	}

	for _, err := range errorsToShow {
		sb.WriteString(fmt.Sprintf("- %s\n", err))
	}

	if remaining > 0 {
		sb.WriteString(fmt.Sprintf("...and %d more errors\n", remaining))
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// BuildRetryCommand creates a command string with retry context prepended to original arguments.
// The retry context and original arguments are separated by a blank line.
// If originalArgs is empty, the retry context becomes the sole content.
func BuildRetryCommand(command string, retryContext string, originalArgs string) string {
	if retryContext == "" {
		if originalArgs != "" {
			return fmt.Sprintf("%s %s", command, originalArgs)
		}
		return command
	}

	if originalArgs == "" {
		return fmt.Sprintf("%s %s", command, retryContext)
	}

	// Separate retry context from original args with a blank line
	combinedArgs := fmt.Sprintf("%s\n\n%s", retryContext, originalArgs)
	return fmt.Sprintf("%s %s", command, combinedArgs)
}

// ExtractValidationErrors parses a validation error message and extracts individual error lines.
// Expects format: "schema validation failed for X:\n- error1\n- error2"
//
// Parsing strategy: split by newline, collect lines starting with "- ".
// Fallback: if no bullet points found, return entire error as single-item slice.
// This handles both structured errors and raw error messages.
func ExtractValidationErrors(err error) []string {
	if err == nil {
		return nil
	}

	errStr := err.Error()
	lines := strings.Split(errStr, "\n")
	var errors []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Extract lines that start with "- " (error bullet points)
		if strings.HasPrefix(line, "- ") {
			errors = append(errors, strings.TrimPrefix(line, "- "))
		}
	}

	// If no bullet points found, return the whole error as a single error
	if len(errors) == 0 && errStr != "" {
		return []string{errStr}
	}

	return errors
}
