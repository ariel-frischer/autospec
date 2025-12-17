package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadHistory(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupStore  func(t *testing.T, stateDir string)
		wantEntries int
		wantErr     bool
	}{
		"returns empty history when file doesn't exist": {
			setupStore:  func(t *testing.T, stateDir string) {},
			wantEntries: 0,
			wantErr:     false,
		},
		"loads existing history file": {
			setupStore: func(t *testing.T, stateDir string) {
				content := `entries:
  - timestamp: 2024-01-15T10:30:00Z
    command: specify
    spec: test-feature
    exit_code: 0
    duration: 2m30s
  - timestamp: 2024-01-15T10:35:00Z
    command: plan
    spec: test-feature
    exit_code: 0
    duration: 1m15s
`
				err := os.WriteFile(filepath.Join(stateDir, HistoryFileName), []byte(content), 0644)
				require.NoError(t, err)
			},
			wantEntries: 2,
			wantErr:     false,
		},
		"handles corrupted file by backing up and returning empty": {
			setupStore: func(t *testing.T, stateDir string) {
				content := `not valid yaml: [[[`
				err := os.WriteFile(filepath.Join(stateDir, HistoryFileName), []byte(content), 0644)
				require.NoError(t, err)
			},
			wantEntries: 0,
			wantErr:     false,
		},
		"handles empty file gracefully": {
			setupStore: func(t *testing.T, stateDir string) {
				err := os.WriteFile(filepath.Join(stateDir, HistoryFileName), []byte(""), 0644)
				require.NoError(t, err)
			},
			wantEntries: 0,
			wantErr:     false,
		},
		"handles file with empty entries list": {
			setupStore: func(t *testing.T, stateDir string) {
				content := `entries: []`
				err := os.WriteFile(filepath.Join(stateDir, HistoryFileName), []byte(content), 0644)
				require.NoError(t, err)
			},
			wantEntries: 0,
			wantErr:     false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()
			tc.setupStore(t, stateDir)

			history, err := LoadHistory(stateDir)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, history)
			assert.Len(t, history.Entries, tc.wantEntries)
		})
	}
}

func TestLoadHistory_CorruptedFileBackup(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	historyPath := filepath.Join(stateDir, HistoryFileName)
	backupPath := historyPath + BackupSuffix

	// Write corrupted content
	corruptedContent := `{invalid yaml content`
	err := os.WriteFile(historyPath, []byte(corruptedContent), 0644)
	require.NoError(t, err)

	// Load should succeed and create backup
	history, err := LoadHistory(stateDir)
	require.NoError(t, err)
	assert.Len(t, history.Entries, 0)

	// Verify backup was created
	assert.FileExists(t, backupPath)

	// Verify original file was renamed
	_, err = os.Stat(historyPath)
	assert.True(t, os.IsNotExist(err))

	// Verify backup contains original content
	backupContent, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, corruptedContent, string(backupContent))
}

func TestSaveHistory(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		history     *HistoryFile
		wantErr     bool
		wantEntries int
	}{
		"save empty history": {
			history:     &HistoryFile{Entries: []HistoryEntry{}},
			wantErr:     false,
			wantEntries: 0,
		},
		"save history with entries": {
			history: &HistoryFile{
				Entries: []HistoryEntry{
					{
						Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
						Command:   "specify",
						Spec:      "test-feature",
						ExitCode:  0,
						Duration:  "2m30s",
					},
				},
			},
			wantErr:     false,
			wantEntries: 1,
		},
		"save history with multiple entries": {
			history: &HistoryFile{
				Entries: []HistoryEntry{
					{
						Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
						Command:   "specify",
						Spec:      "test-feature",
						ExitCode:  0,
						Duration:  "2m30s",
					},
					{
						Timestamp: time.Date(2024, 1, 15, 10, 35, 0, 0, time.UTC),
						Command:   "plan",
						Spec:      "test-feature",
						ExitCode:  1,
						Duration:  "1m15s",
					},
				},
			},
			wantErr:     false,
			wantEntries: 2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()

			err := SaveHistory(stateDir, tc.history)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify file exists
			historyPath := filepath.Join(stateDir, HistoryFileName)
			assert.FileExists(t, historyPath)

			// Load and verify content
			loaded, err := LoadHistory(stateDir)
			require.NoError(t, err)
			assert.Len(t, loaded.Entries, tc.wantEntries)
		})
	}
}

func TestSaveHistory_CreatesDirectory(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	stateDir := filepath.Join(baseDir, "nested", "state", "dir")

	history := &HistoryFile{
		Entries: []HistoryEntry{
			{
				Timestamp: time.Now(),
				Command:   "test",
				ExitCode:  0,
				Duration:  "1s",
			},
		},
	}

	err := SaveHistory(stateDir, history)
	require.NoError(t, err)

	// Verify directory was created
	assert.DirExists(t, stateDir)

	// Verify file was created
	historyPath := filepath.Join(stateDir, HistoryFileName)
	assert.FileExists(t, historyPath)
}

func TestSaveHistory_AtomicWrite(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	history := &HistoryFile{
		Entries: []HistoryEntry{
			{
				Timestamp: time.Now(),
				Command:   "test",
				ExitCode:  0,
				Duration:  "1s",
			},
		},
	}

	err := SaveHistory(stateDir, history)
	require.NoError(t, err)

	// Verify temp file doesn't exist
	tmpPath := filepath.Join(stateDir, HistoryFileName+".tmp")
	_, err = os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(err), "temp file should not exist after atomic write")

	// Verify final file exists
	historyPath := filepath.Join(stateDir, HistoryFileName)
	assert.FileExists(t, historyPath)
}

func TestClearHistory(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	// First save some entries
	history := &HistoryFile{
		Entries: []HistoryEntry{
			{
				Timestamp: time.Now(),
				Command:   "specify",
				Spec:      "test",
				ExitCode:  0,
				Duration:  "1m",
			},
			{
				Timestamp: time.Now(),
				Command:   "plan",
				Spec:      "test",
				ExitCode:  0,
				Duration:  "2m",
			},
		},
	}

	err := SaveHistory(stateDir, history)
	require.NoError(t, err)

	// Verify entries exist
	loaded, err := LoadHistory(stateDir)
	require.NoError(t, err)
	assert.Len(t, loaded.Entries, 2)

	// Clear history
	err = ClearHistory(stateDir)
	require.NoError(t, err)

	// Verify history is empty
	loaded, err = LoadHistory(stateDir)
	require.NoError(t, err)
	assert.Len(t, loaded.Entries, 0)
}

func TestHistoryEntry_YAMLRoundtrip(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	original := &HistoryFile{
		Entries: []HistoryEntry{
			{
				Timestamp: timestamp,
				Command:   "specify",
				Spec:      "test-feature",
				ExitCode:  0,
				Duration:  "2m30.5s",
			},
			{
				Timestamp: timestamp.Add(time.Hour),
				Command:   "plan",
				Spec:      "",
				ExitCode:  1,
				Duration:  "45s",
			},
		},
	}

	// Save
	err := SaveHistory(stateDir, original)
	require.NoError(t, err)

	// Load
	loaded, err := LoadHistory(stateDir)
	require.NoError(t, err)

	// Compare
	require.Len(t, loaded.Entries, 2)

	assert.Equal(t, original.Entries[0].Timestamp, loaded.Entries[0].Timestamp)
	assert.Equal(t, original.Entries[0].Command, loaded.Entries[0].Command)
	assert.Equal(t, original.Entries[0].Spec, loaded.Entries[0].Spec)
	assert.Equal(t, original.Entries[0].ExitCode, loaded.Entries[0].ExitCode)
	assert.Equal(t, original.Entries[0].Duration, loaded.Entries[0].Duration)

	assert.Equal(t, original.Entries[1].Timestamp, loaded.Entries[1].Timestamp)
	assert.Equal(t, original.Entries[1].Command, loaded.Entries[1].Command)
	assert.Equal(t, original.Entries[1].Spec, loaded.Entries[1].Spec)
	assert.Equal(t, original.Entries[1].ExitCode, loaded.Entries[1].ExitCode)
	assert.Equal(t, original.Entries[1].Duration, loaded.Entries[1].Duration)
}

func TestDefaultHistoryPath(t *testing.T) {
	t.Parallel()

	path, err := DefaultHistoryPath()
	require.NoError(t, err)

	// Should contain the expected path components
	assert.Contains(t, path, ".autospec")
	assert.Contains(t, path, "state")
	assert.Contains(t, path, HistoryFileName)
}
