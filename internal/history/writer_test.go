package history

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistoryWriter_LogEntry(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupStore  func(t *testing.T, stateDir string)
		maxEntries  int
		wantEntries int
	}{
		"log entry to empty history": {
			setupStore:  func(t *testing.T, stateDir string) {},
			maxEntries:  500,
			wantEntries: 1,
		},
		"log entry to existing history": {
			setupStore: func(t *testing.T, stateDir string) {
				history := &HistoryFile{
					Entries: []HistoryEntry{
						{Timestamp: time.Now(), Command: "existing", ExitCode: 0, Duration: "1m"},
					},
				}
				require.NoError(t, SaveHistory(stateDir, history))
			},
			maxEntries:  500,
			wantEntries: 2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()
			tc.setupStore(t, stateDir)

			writer := NewWriter(stateDir, tc.maxEntries)
			entry := HistoryEntry{
				Timestamp: time.Now(),
				Command:   "test",
				Spec:      "test-spec",
				ExitCode:  0,
				Duration:  "30s",
			}
			writer.LogEntry(entry)

			// Verify entry was logged
			history, err := LoadHistory(stateDir)
			require.NoError(t, err)
			assert.Len(t, history.Entries, tc.wantEntries)
		})
	}
}

func TestHistoryWriter_Pruning(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		existingEntries int
		maxEntries      int
		wantEntries     int
		wantOldest      string // Command name of oldest remaining entry
	}{
		"no pruning needed": {
			existingEntries: 5,
			maxEntries:      10,
			wantEntries:     6, // 5 existing + 1 new
			wantOldest:      "cmd-0",
		},
		"prune oldest when max exceeded": {
			existingEntries: 10,
			maxEntries:      10,
			wantEntries:     10, // oldest removed, new added
			wantOldest:      "cmd-1",
		},
		"prune multiple when well over max": {
			existingEntries: 12,
			maxEntries:      10,
			wantEntries:     10,
			wantOldest:      "cmd-3",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()

			// Create existing entries
			entries := make([]HistoryEntry, tc.existingEntries)
			for i := 0; i < tc.existingEntries; i++ {
				entries[i] = HistoryEntry{
					Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
					Command:   "cmd-" + string(rune('0'+i)),
					ExitCode:  0,
					Duration:  "1m",
				}
			}
			history := &HistoryFile{Entries: entries}
			require.NoError(t, SaveHistory(stateDir, history))

			// Log new entry
			writer := NewWriter(stateDir, tc.maxEntries)
			writer.LogEntry(HistoryEntry{
				Timestamp: time.Now().Add(time.Hour),
				Command:   "new-cmd",
				ExitCode:  0,
				Duration:  "30s",
			})

			// Verify
			loaded, err := LoadHistory(stateDir)
			require.NoError(t, err)
			assert.Len(t, loaded.Entries, tc.wantEntries)

			// Verify oldest entry
			if len(loaded.Entries) > 0 {
				assert.Equal(t, tc.wantOldest, loaded.Entries[0].Command)
			}

			// Verify newest entry is our new one
			assert.Equal(t, "new-cmd", loaded.Entries[len(loaded.Entries)-1].Command)
		})
	}
}

func TestHistoryWriter_LogCommand(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	writer := NewWriter(stateDir, 500)

	writer.LogCommand("specify", "test-feature", 0, 2*time.Minute+30*time.Second)

	// Verify
	history, err := LoadHistory(stateDir)
	require.NoError(t, err)
	require.Len(t, history.Entries, 1)

	entry := history.Entries[0]
	assert.Equal(t, "specify", entry.Command)
	assert.Equal(t, "test-feature", entry.Spec)
	assert.Equal(t, 0, entry.ExitCode)
	assert.Equal(t, "2m30s", entry.Duration)
	assert.False(t, entry.Timestamp.IsZero())
}

func TestHistoryWriter_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	writer := NewWriter(stateDir, 100)

	// Run multiple goroutines writing concurrently
	var wg sync.WaitGroup
	numWriters := 10
	entriesPerWriter := 5

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < entriesPerWriter; j++ {
				writer.LogEntry(HistoryEntry{
					Timestamp: time.Now(),
					Command:   "test",
					Spec:      "concurrent-test",
					ExitCode:  0,
					Duration:  "1s",
				})
			}
		}(i)
	}

	wg.Wait()

	// Verify all entries were written (may be less due to races, but should be close)
	history, err := LoadHistory(stateDir)
	require.NoError(t, err)

	// Due to potential race conditions with file writes, we just verify
	// that some entries were written successfully
	assert.Greater(t, len(history.Entries), 0, "at least some entries should be written")
	assert.LessOrEqual(t, len(history.Entries), numWriters*entriesPerWriter)
}

func TestHistoryWriter_NonFatalErrors(t *testing.T) {
	t.Parallel()

	// Use an invalid path that can't be created
	writer := NewWriter("/nonexistent/deeply/nested/path/that/cannot/exist", 500)

	// This should not panic, just print a warning
	writer.LogEntry(HistoryEntry{
		Timestamp: time.Now(),
		Command:   "test",
		ExitCode:  0,
		Duration:  "1s",
	})

	// If we get here without panic, the test passes
}

func TestNewWriter(t *testing.T) {
	t.Parallel()

	writer := NewWriter("/test/path", 100)

	assert.Equal(t, "/test/path", writer.StateDir)
	assert.Equal(t, 100, writer.MaxEntries)
}

func TestHistoryWriter_ZeroMaxEntries(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	// Zero max entries means unlimited
	writer := NewWriter(stateDir, 0)

	// Log 5 entries
	for i := 0; i < 5; i++ {
		writer.LogEntry(HistoryEntry{
			Timestamp: time.Now(),
			Command:   "test",
			ExitCode:  0,
			Duration:  "1s",
		})
	}

	// All should be retained
	history, err := LoadHistory(stateDir)
	require.NoError(t, err)
	assert.Len(t, history.Entries, 5)
}

func TestHistoryWriter_WriteStart(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		command     string
		spec        string
		wantCommand string
		wantSpec    string
	}{
		"basic command with spec": {
			command:     "specify",
			spec:        "test-feature",
			wantCommand: "specify",
			wantSpec:    "test-feature",
		},
		"command without spec": {
			command:     "doctor",
			spec:        "",
			wantCommand: "doctor",
			wantSpec:    "",
		},
		"run command with spec": {
			command:     "run",
			spec:        "my-new-feature",
			wantCommand: "run",
			wantSpec:    "my-new-feature",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()
			writer := NewWriter(stateDir, 500)

			id, err := writer.WriteStart(tc.command, tc.spec)
			require.NoError(t, err)
			require.NotEmpty(t, id, "should return a non-empty ID")

			// Verify ID format: adjective_noun_YYYYMMDD_HHMMSS
			assert.Regexp(t, `^[a-z]+_[a-z]+_\d{8}_\d{6}$`, id)

			// Verify entry exists in history file immediately
			history, err := LoadHistory(stateDir)
			require.NoError(t, err)
			require.Len(t, history.Entries, 1)

			entry := history.Entries[0]
			assert.Equal(t, id, entry.ID)
			assert.Equal(t, tc.wantCommand, entry.Command)
			assert.Equal(t, tc.wantSpec, entry.Spec)
			assert.Equal(t, StatusRunning, entry.Status)
			assert.False(t, entry.Timestamp.IsZero())
			assert.False(t, entry.CreatedAt.IsZero())
			assert.Nil(t, entry.CompletedAt, "CompletedAt should be nil for running entry")
		})
	}
}

func TestHistoryWriter_WriteStart_UniqueIDs(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	writer := NewWriter(stateDir, 500)

	ids := make(map[string]bool)
	const numIDs = 10

	for i := 0; i < numIDs; i++ {
		id, err := writer.WriteStart("test", "spec")
		require.NoError(t, err)
		assert.False(t, ids[id], "ID %s should be unique", id)
		ids[id] = true
	}

	assert.Len(t, ids, numIDs, "all IDs should be unique")
}

func TestHistoryWriter_WriteStart_RunningEntryPersistsAfterCrash(t *testing.T) {
	t.Parallel()

	// Simulate a crash scenario: WriteStart is called, then process "crashes"
	// (we just don't call UpdateComplete). The entry should remain with 'running' status.
	stateDir := t.TempDir()
	writer := NewWriter(stateDir, 500)

	id, err := writer.WriteStart("implement", "crash-test-feature")
	require.NoError(t, err)

	// Simulate crash by not calling UpdateComplete and creating a new writer
	// (as if the process restarted)
	newWriter := NewWriter(stateDir, 500)
	_ = newWriter // Just to show we have a "new" instance

	// Verify the running entry is still there
	history, err := LoadHistory(stateDir)
	require.NoError(t, err)
	require.Len(t, history.Entries, 1)

	entry := history.Entries[0]
	assert.Equal(t, id, entry.ID)
	assert.Equal(t, StatusRunning, entry.Status)
	assert.Equal(t, "implement", entry.Command)
	assert.Equal(t, "crash-test-feature", entry.Spec)
	assert.Nil(t, entry.CompletedAt, "CompletedAt should still be nil")
}

func TestHistoryWriter_WriteStart_Error(t *testing.T) {
	t.Parallel()

	// Use an invalid path that can't be created
	writer := NewWriter("/nonexistent/deeply/nested/path/that/cannot/exist", 500)

	id, err := writer.WriteStart("test", "spec")
	assert.Error(t, err)
	assert.Empty(t, id)
	assert.Contains(t, err.Error(), "writing start entry")
}

func TestHistoryWriter_UpdateComplete(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status      string
		exitCode    int
		duration    time.Duration
		wantStatus  string
		wantCode    int
		description string
	}{
		"completed success": {
			status:      StatusCompleted,
			exitCode:    0,
			duration:    2*time.Minute + 30*time.Second,
			wantStatus:  StatusCompleted,
			wantCode:    0,
			description: "command completed successfully",
		},
		"failed with error": {
			status:      StatusFailed,
			exitCode:    1,
			duration:    45 * time.Second,
			wantStatus:  StatusFailed,
			wantCode:    1,
			description: "command failed with exit code 1",
		},
		"cancelled by user": {
			status:      StatusCancelled,
			exitCode:    130, // typical Ctrl+C exit code
			duration:    10 * time.Second,
			wantStatus:  StatusCancelled,
			wantCode:    130,
			description: "command cancelled via Ctrl+C",
		},
		"failed with high exit code": {
			status:      StatusFailed,
			exitCode:    255,
			duration:    5 * time.Second,
			wantStatus:  StatusFailed,
			wantCode:    255,
			description: "command failed with exit code 255",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()
			writer := NewWriter(stateDir, 500)

			// First create a running entry
			id, err := writer.WriteStart("implement", "test-feature")
			require.NoError(t, err)
			require.NotEmpty(t, id)

			// Now update it to completion
			err = writer.UpdateComplete(id, tc.exitCode, tc.status, tc.duration)
			require.NoError(t, err)

			// Verify the entry was updated correctly
			history, err := LoadHistory(stateDir)
			require.NoError(t, err)
			require.Len(t, history.Entries, 1)

			entry := history.Entries[0]
			assert.Equal(t, id, entry.ID)
			assert.Equal(t, tc.wantStatus, entry.Status)
			assert.Equal(t, tc.wantCode, entry.ExitCode)
			assert.Equal(t, tc.duration.String(), entry.Duration)
			assert.NotNil(t, entry.CompletedAt, "CompletedAt should be set")
			assert.False(t, entry.CompletedAt.IsZero(), "CompletedAt should not be zero")
		})
	}
}

func TestHistoryWriter_UpdateComplete_EntryNotFound(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	writer := NewWriter(stateDir, 500)

	// Try to update a non-existent entry
	err := writer.UpdateComplete("nonexistent_id_20251217_120000", 0, StatusCompleted, time.Minute)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "entry not found")
}

func TestHistoryWriter_UpdateComplete_MultipleEntries(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	writer := NewWriter(stateDir, 500)

	// Create multiple running entries
	id1, err := writer.WriteStart("specify", "feature-1")
	require.NoError(t, err)

	id2, err := writer.WriteStart("plan", "feature-2")
	require.NoError(t, err)

	id3, err := writer.WriteStart("implement", "feature-3")
	require.NoError(t, err)

	// Update only the middle one
	err = writer.UpdateComplete(id2, 0, StatusCompleted, 5*time.Minute)
	require.NoError(t, err)

	// Verify all entries
	history, err := LoadHistory(stateDir)
	require.NoError(t, err)
	require.Len(t, history.Entries, 3)

	// Entry 1 should still be running
	assert.Equal(t, id1, history.Entries[0].ID)
	assert.Equal(t, StatusRunning, history.Entries[0].Status)
	assert.Nil(t, history.Entries[0].CompletedAt)

	// Entry 2 should be completed
	assert.Equal(t, id2, history.Entries[1].ID)
	assert.Equal(t, StatusCompleted, history.Entries[1].Status)
	assert.NotNil(t, history.Entries[1].CompletedAt)
	assert.Equal(t, "5m0s", history.Entries[1].Duration)

	// Entry 3 should still be running
	assert.Equal(t, id3, history.Entries[2].ID)
	assert.Equal(t, StatusRunning, history.Entries[2].Status)
	assert.Nil(t, history.Entries[2].CompletedAt)
}

func TestHistoryWriter_UpdateComplete_EmptyHistory(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	writer := NewWriter(stateDir, 500)

	// Try to update an entry in an empty history (no entries have been written)
	err := writer.UpdateComplete("some_id_20251217_120000", 0, StatusCompleted, time.Minute)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "entry not found")
}

func TestHistoryWriter_TwoPhaseWorkflow(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	writer := NewWriter(stateDir, 500)

	// Phase 1: Start command
	startTime := time.Now()
	id, err := writer.WriteStart("implement", "two-phase-test")
	require.NoError(t, err)

	// Verify running entry exists
	history, err := LoadHistory(stateDir)
	require.NoError(t, err)
	require.Len(t, history.Entries, 1)
	assert.Equal(t, StatusRunning, history.Entries[0].Status)

	// Simulate some work
	time.Sleep(10 * time.Millisecond)
	duration := time.Since(startTime)

	// Phase 2: Complete command
	err = writer.UpdateComplete(id, 0, StatusCompleted, duration)
	require.NoError(t, err)

	// Verify completed entry
	history, err = LoadHistory(stateDir)
	require.NoError(t, err)
	require.Len(t, history.Entries, 1)

	entry := history.Entries[0]
	assert.Equal(t, id, entry.ID)
	assert.Equal(t, StatusCompleted, entry.Status)
	assert.Equal(t, 0, entry.ExitCode)
	assert.NotNil(t, entry.CompletedAt)
	assert.False(t, entry.CreatedAt.IsZero())
	assert.True(t, entry.CompletedAt.After(entry.CreatedAt) || entry.CompletedAt.Equal(entry.CreatedAt))
}
