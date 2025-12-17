// Package lifecycle provides wrapper functions for CLI command and workflow
// stage execution. It handles timing and notification dispatch, eliminating
// boilerplate code across CLI commands.
//
// The lifecycle package is intentionally minimal: no event bus, no goroutines,
// no external dependencies. Each wrapper function captures start time, executes
// the provided function, calculates duration, and calls the appropriate
// notification method.
package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

// Status constants for history entries.
const (
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

// Run wraps command execution with timing and notification dispatch.
// It captures the start time, executes fn, calculates duration, and calls
// handler.OnCommandComplete with the results.
//
// If handler is nil, fn is still executed but no notification is sent.
// Handler panics are recovered to ensure command completion is not affected.
// The original error from fn is always returned unchanged.
func Run(handler NotificationHandler, name string, fn func() error) error {
	start := time.Now()
	fnErr := fn()
	duration := time.Since(start)

	notifyCommandComplete(handler, name, fnErr == nil, duration)

	return fnErr
}

// RunWithHistory wraps command execution with timing, notification, and history logging.
// It uses two-phase history logging: WriteStart before execution, UpdateComplete after.
//
// Parameters:
//   - handler: notification handler (may be nil)
//   - logger: history logger (may be nil)
//   - name: command name for notifications and history
//   - spec: spec name for history (may be empty)
//   - fn: the command function to execute
//
// History entry is written immediately when command starts (with "running" status),
// then updated with final status when command completes. This ensures crash/interrupt
// visibility - entries remain with "running" status if the process terminates abnormally.
// History logging errors are non-fatal (written to stderr, don't affect return value).
func RunWithHistory(handler NotificationHandler, logger HistoryLogger, name, spec string, fn func() error) error {
	start := time.Now()
	entryID := writeHistoryStart(logger, name, spec)

	fnErr := fn()
	duration := time.Since(start)

	notifyCommandComplete(handler, name, fnErr == nil, duration)
	updateHistoryComplete(logger, entryID, fnErr, duration)

	return fnErr
}

// RunWithContext wraps context-aware command execution.
// If the context is already cancelled, returns context.Canceled immediately
// without executing fn. Otherwise behaves like Run.
//
// The notification is sent regardless of whether fn was executed or cancelled.
func RunWithContext(ctx context.Context, handler NotificationHandler, name string, fn func(context.Context) error) error {
	start := time.Now()

	// Check if context is already cancelled
	if err := ctx.Err(); err != nil {
		duration := time.Since(start)
		notifyCommandComplete(handler, name, false, duration)
		return err
	}

	fnErr := fn(ctx)
	duration := time.Since(start)

	notifyCommandComplete(handler, name, fnErr == nil, duration)

	return fnErr
}

// RunWithHistoryContext wraps context-aware command execution with history logging.
// It uses two-phase history logging: WriteStart before execution, UpdateComplete after.
//
// Parameters:
//   - ctx: context for cancellation
//   - handler: notification handler (may be nil)
//   - logger: history logger (may be nil)
//   - name: command name for notifications and history
//   - spec: spec name for history (may be empty)
//   - fn: the command function to execute
//
// History entry is written immediately when command starts (with "running" status),
// then updated with final status when command completes. Context cancellation results
// in "cancelled" status. History logging errors are non-fatal.
func RunWithHistoryContext(ctx context.Context, handler NotificationHandler, logger HistoryLogger, name, spec string, fn func(context.Context) error) error {
	start := time.Now()
	entryID := writeHistoryStart(logger, name, spec)

	// Check if context is already cancelled
	if err := ctx.Err(); err != nil {
		duration := time.Since(start)
		notifyCommandComplete(handler, name, false, duration)
		updateHistoryComplete(logger, entryID, err, duration)
		return err
	}

	fnErr := fn(ctx)
	duration := time.Since(start)

	notifyCommandComplete(handler, name, fnErr == nil, duration)
	updateHistoryComplete(logger, entryID, fnErr, duration)

	return fnErr
}

// RunStage wraps workflow stage execution with notification dispatch.
// It executes fn and calls handler.OnStageComplete with the results.
//
// If handler is nil, fn is still executed but no notification is sent.
// Handler panics are recovered to ensure stage completion is not affected.
func RunStage(handler NotificationHandler, name string, fn func() error) error {
	fnErr := fn()
	notifyStageComplete(handler, name, fnErr == nil)
	return fnErr
}

// notifyCommandComplete safely calls OnCommandComplete with panic recovery.
func notifyCommandComplete(handler NotificationHandler, name string, success bool, duration time.Duration) {
	if handler == nil {
		return
	}
	defer func() { _ = recover() }()
	handler.OnCommandComplete(name, success, duration)
}

// notifyStageComplete safely calls OnStageComplete with panic recovery.
func notifyStageComplete(handler NotificationHandler, name string, success bool) {
	if handler == nil {
		return
	}
	defer func() { _ = recover() }()
	handler.OnStageComplete(name, success)
}

// writeHistoryStart safely writes a "running" history entry with panic recovery.
// Returns the entry ID for later update, or empty string if logging failed.
func writeHistoryStart(logger HistoryLogger, name, spec string) string {
	if logger == nil {
		return ""
	}
	defer func() { _ = recover() }()

	id, err := logger.WriteStart(name, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write history start: %v\n", err)
		return ""
	}
	return id
}

// updateHistoryComplete safely updates a history entry with final status.
// Uses panic recovery to ensure command completion is not affected.
func updateHistoryComplete(logger HistoryLogger, entryID string, fnErr error, duration time.Duration) {
	if logger == nil || entryID == "" {
		return
	}
	defer func() { _ = recover() }()

	status, exitCode := determineStatusAndCode(fnErr)
	if err := logger.UpdateComplete(entryID, exitCode, status, duration); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update history: %v\n", err)
	}
}

// determineStatusAndCode determines the status and exit code from an error.
func determineStatusAndCode(fnErr error) (status string, exitCode int) {
	if fnErr == nil {
		return StatusCompleted, 0
	}
	if errors.Is(fnErr, context.Canceled) || errors.Is(fnErr, context.DeadlineExceeded) {
		return StatusCancelled, 1
	}
	return StatusFailed, 1
}
