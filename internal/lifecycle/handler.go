// Package lifecycle provides wrapper functions for CLI command and workflow
// stage execution. It handles timing and notification dispatch, eliminating
// boilerplate code across CLI commands.
//
// The lifecycle package is intentionally minimal: no event bus, no goroutines,
// no external dependencies. Each wrapper function captures start time, executes
// the provided function, calculates duration, and calls the appropriate
// notification method.
package lifecycle

import "time"

// NotificationHandler defines the interface for notification dispatch.
// This interface is satisfied by *notify.Handler but defined separately
// to avoid circular imports between lifecycle and notify packages.
//
// Implementations must be safe for nil receivers - the lifecycle wrapper
// functions check for nil before calling any method.
type NotificationHandler interface {
	// OnCommandComplete is called when a CLI command finishes execution.
	// Parameters:
	//   - name: the command name (e.g., "specify", "plan", "implement")
	//   - success: true if command completed without error
	//   - duration: how long the command took to execute
	OnCommandComplete(name string, success bool, duration time.Duration)

	// OnStageComplete is called when a workflow stage finishes execution.
	// Parameters:
	//   - name: the stage name (e.g., "specify", "plan", "tasks")
	//   - success: true if stage completed without error
	OnStageComplete(name string, success bool)
}

// HistoryLogger defines the interface for command history logging.
// This interface is satisfied by *history.Writer but defined separately
// to avoid circular imports between lifecycle and history packages.
//
// Implementations should be non-fatal: errors during logging should not
// cause command failures.
type HistoryLogger interface {
	// LogCommand logs a command execution to the history file.
	// Parameters:
	//   - command: the command name (e.g., "specify", "plan", "implement")
	//   - spec: the spec name being worked on (may be empty)
	//   - exitCode: the exit code (0 = success)
	//   - duration: how long the command took to execute
	LogCommand(command, spec string, exitCode int, duration time.Duration)

	// WriteStart creates a history entry with 'running' status immediately when
	// a command starts. Returns the generated unique ID for later update.
	// Parameters:
	//   - command: the command name (e.g., "specify", "plan", "implement")
	//   - spec: the spec name being worked on (may be empty)
	// Returns:
	//   - string: the unique entry ID for use with UpdateComplete
	//   - error: any error during entry creation
	WriteStart(command, spec string) (string, error)

	// UpdateComplete updates a running history entry with final status.
	// Parameters:
	//   - id: the unique entry ID returned by WriteStart
	//   - exitCode: the exit code (0 = success)
	//   - status: the final status (completed, failed, cancelled)
	//   - duration: how long the command took to execute
	// Returns:
	//   - error: any error during entry update
	UpdateComplete(id string, exitCode int, status string, duration time.Duration) error
}
