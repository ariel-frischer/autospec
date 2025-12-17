package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/history"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistoryCmdRegistration(t *testing.T) {
	t.Parallel()

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "history" {
			found = true
			break
		}
	}
	assert.True(t, found, "history command should be registered")
}

func TestHistoryCmdFlags(t *testing.T) {
	t.Parallel()

	flags := []struct {
		name      string
		shorthand string
	}{
		{"spec", "s"},
		{"limit", "n"},
		{"clear", "c"},
	}

	for _, flag := range flags {
		t.Run("flag "+flag.name, func(t *testing.T) {
			t.Parallel()
			f := historyCmd.Flags().Lookup(flag.name)
			require.NotNil(t, f, "flag %s should exist", flag.name)
			assert.Equal(t, flag.shorthand, f.Shorthand)
		})
	}
}

func TestHistoryCmdShortDescription(t *testing.T) {
	t.Parallel()
	assert.Contains(t, historyCmd.Short, "history")
}

func TestRunHistory_EmptyHistory(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	cmd := createTestHistoryCmd(stateDir)

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "No history")
}

func TestRunHistory_WithEntries(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	// Create history with entries
	histFile := &history.HistoryFile{
		Entries: []history.HistoryEntry{
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
				ExitCode:  0,
				Duration:  "1m15s",
			},
		},
	}
	require.NoError(t, history.SaveHistory(stateDir, histFile))

	cmd := createTestHistoryCmd(stateDir)

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "specify")
	assert.Contains(t, output, "plan")
	assert.Contains(t, output, "test-feature")
}

func TestRunHistory_SpecFilter(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	// Create history with entries for different specs
	histFile := &history.HistoryFile{
		Entries: []history.HistoryEntry{
			{
				Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				Command:   "specify",
				Spec:      "feature-a",
				ExitCode:  0,
				Duration:  "1m",
			},
			{
				Timestamp: time.Date(2024, 1, 15, 10, 35, 0, 0, time.UTC),
				Command:   "plan",
				Spec:      "feature-b",
				ExitCode:  0,
				Duration:  "2m",
			},
			{
				Timestamp: time.Date(2024, 1, 15, 10, 40, 0, 0, time.UTC),
				Command:   "tasks",
				Spec:      "feature-a",
				ExitCode:  0,
				Duration:  "30s",
			},
		},
	}
	require.NoError(t, history.SaveHistory(stateDir, histFile))

	cmd := createTestHistoryCmd(stateDir)
	require.NoError(t, cmd.Flags().Set("spec", "feature-a"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "feature-a")
	assert.NotContains(t, output, "feature-b")
}

func TestRunHistory_SpecFilter_NoMatch(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	histFile := &history.HistoryFile{
		Entries: []history.HistoryEntry{
			{
				Timestamp: time.Now(),
				Command:   "specify",
				Spec:      "other-feature",
				ExitCode:  0,
				Duration:  "1m",
			},
		},
	}
	require.NoError(t, history.SaveHistory(stateDir, histFile))

	cmd := createTestHistoryCmd(stateDir)
	require.NoError(t, cmd.Flags().Set("spec", "nonexistent"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "No matching")
}

func TestRunHistory_Limit(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	// Create history with 5 entries
	entries := make([]history.HistoryEntry, 5)
	for i := 0; i < 5; i++ {
		entries[i] = history.HistoryEntry{
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Command:   "test",
			Spec:      "feature",
			ExitCode:  0,
			Duration:  "1m",
		}
	}
	histFile := &history.HistoryFile{Entries: entries}
	require.NoError(t, history.SaveHistory(stateDir, histFile))

	cmd := createTestHistoryCmd(stateDir)
	require.NoError(t, cmd.Flags().Set("limit", "2"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	// Check that output contains limited entries
	output := buf.String()
	assert.Contains(t, output, "test")
}

func TestRunHistory_InvalidLimit(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		limit string
	}{
		"zero limit":     {limit: "0"},
		"negative limit": {limit: "-1"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()

			cmd := createTestHistoryCmd(stateDir)
			require.NoError(t, cmd.Flags().Set("limit", tc.limit))

			var buf bytes.Buffer
			cmd.SetOut(&buf)

			err := cmd.RunE(cmd, []string{})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "must be positive")
		})
	}
}

func TestRunHistory_Clear(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	// Create history with entries
	histFile := &history.HistoryFile{
		Entries: []history.HistoryEntry{
			{
				Timestamp: time.Now(),
				Command:   "specify",
				Spec:      "test",
				ExitCode:  0,
				Duration:  "1m",
			},
		},
	}
	require.NoError(t, history.SaveHistory(stateDir, histFile))

	// Verify entries exist
	loaded, err := history.LoadHistory(stateDir)
	require.NoError(t, err)
	assert.Len(t, loaded.Entries, 1)

	cmd := createTestHistoryCmd(stateDir)
	require.NoError(t, cmd.Flags().Set("clear", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err = cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "cleared")

	// Verify history is now empty
	loaded, err = history.LoadHistory(stateDir)
	require.NoError(t, err)
	assert.Len(t, loaded.Entries, 0)
}

func TestRunHistory_ClearEmptyHistory(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	cmd := createTestHistoryCmd(stateDir)
	require.NoError(t, cmd.Flags().Set("clear", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "cleared")
}

func TestRunHistory_SpecAndLimit(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	// Create history with multiple entries for same spec
	entries := make([]history.HistoryEntry, 5)
	for i := 0; i < 5; i++ {
		entries[i] = history.HistoryEntry{
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Command:   "test",
			Spec:      "target-spec",
			ExitCode:  0,
			Duration:  "1m",
		}
	}
	// Add one entry for different spec
	entries = append(entries, history.HistoryEntry{
		Timestamp: time.Now(),
		Command:   "other",
		Spec:      "other-spec",
		ExitCode:  0,
		Duration:  "1m",
	})

	histFile := &history.HistoryFile{Entries: entries}
	require.NoError(t, history.SaveHistory(stateDir, histFile))

	cmd := createTestHistoryCmd(stateDir)
	require.NoError(t, cmd.Flags().Set("spec", "target-spec"))
	require.NoError(t, cmd.Flags().Set("limit", "2"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "target-spec")
	assert.NotContains(t, output, "other-spec")
}

func createTestHistoryCmd(stateDir string) *cobra.Command {
	cmd := &cobra.Command{
		Use: "history",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistoryWithStateDir(cmd, stateDir)
		},
	}
	cmd.Flags().StringP("spec", "s", "", "Filter by spec name")
	cmd.Flags().IntP("limit", "n", 0, "Limit to last N entries")
	cmd.Flags().BoolP("clear", "c", false, "Clear all history")
	return cmd
}

func TestHistoryOutputFormat(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	histFile := &history.HistoryFile{
		Entries: []history.HistoryEntry{
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
	}
	require.NoError(t, history.SaveHistory(stateDir, histFile))

	cmd := createTestHistoryCmd(stateDir)

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	// Verify output contains expected fields
	assert.Contains(t, output, "2024")
	assert.Contains(t, output, "specify")
	assert.Contains(t, output, "plan")
	assert.Contains(t, output, "2m30s")
	assert.Contains(t, output, "1m15s")
}

func TestHistoryDir(t *testing.T) {
	t.Parallel()

	// Skip if HOME is not set
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME not set")
	}

	expectedPath := filepath.Join(home, ".autospec", "state")
	actualPath := getDefaultStateDir()
	assert.Equal(t, expectedPath, actualPath)
}
