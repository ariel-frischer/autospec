package history

import (
	"fmt"
	"os"
	"time"
)

// Writer provides thread-safe history logging with automatic pruning.
type Writer struct {
	// StateDir is the directory containing the history file.
	StateDir string
	// MaxEntries is the maximum number of entries to retain.
	MaxEntries int
}

// NewWriter creates a new history writer.
func NewWriter(stateDir string, maxEntries int) *Writer {
	return &Writer{
		StateDir:   stateDir,
		MaxEntries: maxEntries,
	}
}

// LogEntry adds a new entry to the history file.
// It loads the existing history, appends the new entry, prunes if needed, and saves.
// Errors are non-fatal: they are written to stderr and don't cause command failures.
func (w *Writer) LogEntry(entry HistoryEntry) {
	if err := w.logEntryInternal(entry); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to log history: %v\n", err)
	}
}

// logEntryInternal handles the actual logging logic.
func (w *Writer) logEntryInternal(entry HistoryEntry) error {
	history, err := LoadHistory(w.StateDir)
	if err != nil {
		return fmt.Errorf("loading history: %w", err)
	}

	history.Entries = append(history.Entries, entry)

	// Prune oldest entries if over limit
	if w.MaxEntries > 0 && len(history.Entries) > w.MaxEntries {
		excess := len(history.Entries) - w.MaxEntries
		history.Entries = history.Entries[excess:]
	}

	if err := SaveHistory(w.StateDir, history); err != nil {
		return fmt.Errorf("saving history: %w", err)
	}

	return nil
}

// LogCommand is a convenience method to log a command execution.
func (w *Writer) LogCommand(command, spec string, exitCode int, duration time.Duration) {
	entry := HistoryEntry{
		Timestamp: time.Now(),
		Command:   command,
		Spec:      spec,
		ExitCode:  exitCode,
		Duration:  duration.String(),
	}
	w.LogEntry(entry)
}

// WriteStart creates a history entry with 'running' status immediately when a command starts.
// Returns the generated unique ID for later update via UpdateComplete.
func (w *Writer) WriteStart(command, spec string) (string, error) {
	id, err := GenerateID()
	if err != nil {
		return "", fmt.Errorf("generating history ID: %w", err)
	}

	now := time.Now()
	entry := HistoryEntry{
		ID:        id,
		Timestamp: now,
		Command:   command,
		Spec:      spec,
		Status:    StatusRunning,
		CreatedAt: now,
	}

	if err := w.logEntryInternal(entry); err != nil {
		return "", fmt.Errorf("writing start entry: %w", err)
	}

	return id, nil
}

// UpdateComplete updates a running history entry with final status when a command completes.
// Parameters:
//   - id: the unique entry ID returned by WriteStart
//   - exitCode: the process exit code (0 = success)
//   - status: the final status (completed, failed, cancelled)
//   - duration: how long the command took to execute
//
// Returns an error if the entry with the given ID is not found.
func (w *Writer) UpdateComplete(id string, exitCode int, status string, duration time.Duration) error {
	history, err := LoadHistory(w.StateDir)
	if err != nil {
		return fmt.Errorf("loading history for update: %w", err)
	}

	if err := w.updateEntry(history, id, exitCode, status, duration); err != nil {
		return err
	}

	if err := SaveHistory(w.StateDir, history); err != nil {
		return fmt.Errorf("saving updated history: %w", err)
	}

	return nil
}

// updateEntry finds and updates the entry with the given ID in place.
func (w *Writer) updateEntry(history *HistoryFile, id string, exitCode int, status string, duration time.Duration) error {
	for i := range history.Entries {
		if history.Entries[i].ID == id {
			now := time.Now()
			history.Entries[i].Status = status
			history.Entries[i].ExitCode = exitCode
			history.Entries[i].Duration = duration.String()
			history.Entries[i].CompletedAt = &now
			return nil
		}
	}
	return fmt.Errorf("entry not found with ID: %s", id)
}
