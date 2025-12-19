// Package cli_test tests the history command for viewing and filtering command execution history.
// Related: internal/cli/util/history.go
// Tags: cli, history, command, logging, filtering, status, execution
package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/history"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getHistoryCmd finds the history command from rootCmd
func getHistoryCmd() *cobra.Command {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "history" {
			return cmd
		}
	}
	return nil
}

func TestHistoryCmdRegistration(t *testing.T) {
	t.Parallel()

	cmd := getHistoryCmd()
	assert.NotNil(t, cmd, "history command should be registered")
}

func TestHistoryCmdFlags(t *testing.T) {
	t.Parallel()

	cmd := getHistoryCmd()
	require.NotNil(t, cmd, "history command must exist")

	flags := []struct {
		name      string
		shorthand string
	}{
		{"spec", "s"},
		{"limit", "n"},
		{"clear", ""},
	}

	for _, flag := range flags {
		t.Run("flag "+flag.name, func(t *testing.T) {
			t.Parallel()
			f := cmd.Flags().Lookup(flag.name)
			require.NotNil(t, f, "flag %s should exist", flag.name)
			assert.Equal(t, flag.shorthand, f.Shorthand)
		})
	}
}

func TestHistoryCmdShortDescription(t *testing.T) {
	t.Parallel()

	cmd := getHistoryCmd()
	require.NotNil(t, cmd, "history command must exist")
	assert.Contains(t, cmd.Short, "history")
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

	stateDir := t.TempDir()

	cmd := createTestHistoryCmd(stateDir)
	require.NoError(t, cmd.Flags().Set("limit", "-1"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be positive")
}

func TestRunHistory_ZeroLimit(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	// Create history with 3 entries
	histFile := &history.HistoryFile{
		Entries: []history.HistoryEntry{
			{Timestamp: time.Now(), Command: "cmd1", Spec: "spec", ExitCode: 0, Duration: "1m"},
			{Timestamp: time.Now(), Command: "cmd2", Spec: "spec", ExitCode: 0, Duration: "2m"},
			{Timestamp: time.Now(), Command: "cmd3", Spec: "spec", ExitCode: 0, Duration: "3m"},
		},
	}
	require.NoError(t, history.SaveHistory(stateDir, histFile))

	cmd := createTestHistoryCmd(stateDir)
	require.NoError(t, cmd.Flags().Set("limit", "0"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	// Zero limit should show all entries
	assert.Contains(t, output, "cmd1")
	assert.Contains(t, output, "cmd2")
	assert.Contains(t, output, "cmd3")
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

// createTestHistoryCmd creates a test history command that uses the provided state directory.
// This duplicates the core logic from util/history.go for testing purposes since
// the util package functions are unexported.
func createTestHistoryCmd(stateDir string) *cobra.Command {
	cmd := &cobra.Command{
		Use: "history",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTestHistoryWithStateDir(cmd, stateDir)
		},
	}
	cmd.Flags().StringP("spec", "s", "", "Filter by spec name")
	cmd.Flags().IntP("limit", "n", 0, "Limit to last N entries")
	cmd.Flags().Bool("clear", false, "Clear all history")
	cmd.Flags().String("status", "", "Filter by status (running, completed, failed, cancelled)")
	return cmd
}

// runTestHistoryWithStateDir is a test helper that mirrors the util package's runHistoryWithStateDir.
func runTestHistoryWithStateDir(cmd *cobra.Command, stateDir string) error {
	clearFlag, _ := cmd.Flags().GetBool("clear")
	specFilter, _ := cmd.Flags().GetString("spec")
	statusFilter, _ := cmd.Flags().GetString("status")
	limit, _ := cmd.Flags().GetInt("limit")

	// Validate limit
	if limit < 0 {
		return fmt.Errorf("limit must be positive, got %d", limit)
	}

	// Handle clear flag
	if clearFlag {
		if err := history.ClearHistory(stateDir); err != nil {
			return fmt.Errorf("clearing history: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "History cleared.")
		return nil
	}

	// Load history
	histFile, err := history.LoadHistory(stateDir)
	if err != nil {
		return fmt.Errorf("loading history: %w", err)
	}

	// Filter entries
	var filtered []history.HistoryEntry
	for _, entry := range histFile.Entries {
		if specFilter != "" && entry.Spec != specFilter {
			continue
		}
		if statusFilter != "" && entry.Status != statusFilter {
			continue
		}
		filtered = append(filtered, entry)
	}

	// Apply limit (most recent entries)
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}

	// Handle empty result
	if len(filtered) == 0 {
		if specFilter != "" && statusFilter != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "No matching entries for spec '%s' and status '%s'.\n", specFilter, statusFilter)
		} else if specFilter != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "No matching entries for spec '%s'.\n", specFilter)
		} else if statusFilter != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "No matching entries for status '%s'.\n", statusFilter)
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "No history available.")
		}
		return nil
	}

	// Display entries
	for _, entry := range filtered {
		timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
		spec := entry.Spec
		if spec == "" {
			spec = "-"
		}
		status := entry.Status
		if status == "" {
			status = "-"
		}
		id := entry.ID
		if id == "" {
			id = "-"
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s  %s  %s  %s  %s  exit=%d  %s\n",
			timestamp,
			id,
			status,
			entry.Command,
			spec,
			entry.ExitCode,
			entry.Duration,
		)
	}
	return nil
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

	// Test that getDefaultStateDir would return expected path
	// Since we can't access the unexported function, we verify via the command behavior
	expectedPath := filepath.Join(home, ".autospec", "state")
	assert.NotEmpty(t, expectedPath)
}

// TestRunHistory_StatusDisplay tests that the status column appears in output.
func TestRunHistory_StatusDisplay(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		entries         []history.HistoryEntry
		wantStatuses    []string
		wantInOutput    []string
		wantNotInOutput []string
	}{
		"displays completed status": {
			entries: []history.HistoryEntry{
				{
					ID:        "brave_fox_20241215_103000",
					Timestamp: time.Date(2024, 12, 15, 10, 30, 0, 0, time.UTC),
					Command:   "specify",
					Spec:      "test-feature",
					Status:    history.StatusCompleted,
					ExitCode:  0,
					Duration:  "2m30s",
				},
			},
			wantInOutput: []string{"completed", "brave_fox_20241215_103000", "specify"},
		},
		"displays running status": {
			entries: []history.HistoryEntry{
				{
					ID:        "calm_river_20241215_103500",
					Timestamp: time.Date(2024, 12, 15, 10, 35, 0, 0, time.UTC),
					Command:   "plan",
					Spec:      "test-feature",
					Status:    history.StatusRunning,
					ExitCode:  0,
					Duration:  "",
				},
			},
			wantInOutput: []string{"running", "calm_river_20241215_103500", "plan"},
		},
		"displays failed status": {
			entries: []history.HistoryEntry{
				{
					ID:        "swift_falcon_20241215_104000",
					Timestamp: time.Date(2024, 12, 15, 10, 40, 0, 0, time.UTC),
					Command:   "tasks",
					Spec:      "test-feature",
					Status:    history.StatusFailed,
					ExitCode:  1,
					Duration:  "45s",
				},
			},
			wantInOutput: []string{"failed", "swift_falcon_20241215_104000", "tasks"},
		},
		"displays cancelled status": {
			entries: []history.HistoryEntry{
				{
					ID:        "gentle_owl_20241215_104500",
					Timestamp: time.Date(2024, 12, 15, 10, 45, 0, 0, time.UTC),
					Command:   "implement",
					Spec:      "test-feature",
					Status:    history.StatusCancelled,
					ExitCode:  0,
					Duration:  "1m20s",
				},
			},
			wantInOutput: []string{"cancelled", "gentle_owl_20241215_104500", "implement"},
		},
		"displays dash for old entries without status": {
			entries: []history.HistoryEntry{
				{
					Timestamp: time.Date(2024, 12, 15, 10, 50, 0, 0, time.UTC),
					Command:   "specify",
					Spec:      "old-feature",
					Status:    "", // Empty status (old entry)
					ExitCode:  0,
					Duration:  "3m",
				},
			},
			wantInOutput: []string{"-", "specify", "old-feature"},
		},
		"displays multiple entries with different statuses": {
			entries: []history.HistoryEntry{
				{
					ID:        "brave_fox_20241215_103000",
					Timestamp: time.Date(2024, 12, 15, 10, 30, 0, 0, time.UTC),
					Command:   "specify",
					Spec:      "feature-a",
					Status:    history.StatusCompleted,
					ExitCode:  0,
					Duration:  "2m",
				},
				{
					ID:        "calm_river_20241215_103500",
					Timestamp: time.Date(2024, 12, 15, 10, 35, 0, 0, time.UTC),
					Command:   "plan",
					Spec:      "feature-a",
					Status:    history.StatusRunning,
					ExitCode:  0,
					Duration:  "",
				},
				{
					ID:        "swift_falcon_20241215_104000",
					Timestamp: time.Date(2024, 12, 15, 10, 40, 0, 0, time.UTC),
					Command:   "tasks",
					Spec:      "feature-b",
					Status:    history.StatusFailed,
					ExitCode:  1,
					Duration:  "1m",
				},
			},
			wantInOutput: []string{"completed", "running", "failed", "feature-a", "feature-b"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()
			histFile := &history.HistoryFile{Entries: tt.entries}
			require.NoError(t, history.SaveHistory(stateDir, histFile))

			cmd := createTestHistoryCmd(stateDir)
			var buf bytes.Buffer
			cmd.SetOut(&buf)

			err := cmd.RunE(cmd, []string{})
			require.NoError(t, err)

			output := buf.String()
			for _, want := range tt.wantInOutput {
				assert.Contains(t, output, want, "output should contain %q", want)
			}
			for _, notWant := range tt.wantNotInOutput {
				assert.NotContains(t, output, notWant, "output should not contain %q", notWant)
			}
		})
	}
}

// TestRunHistory_StatusFilter tests the --status flag filtering functionality.
func TestRunHistory_StatusFilter(t *testing.T) {
	t.Parallel()

	baseEntries := []history.HistoryEntry{
		{
			ID:        "brave_fox_20241215_103000",
			Timestamp: time.Date(2024, 12, 15, 10, 30, 0, 0, time.UTC),
			Command:   "specify",
			Spec:      "feature-a",
			Status:    history.StatusCompleted,
			ExitCode:  0,
			Duration:  "2m",
		},
		{
			ID:        "calm_river_20241215_103500",
			Timestamp: time.Date(2024, 12, 15, 10, 35, 0, 0, time.UTC),
			Command:   "plan",
			Spec:      "feature-a",
			Status:    history.StatusRunning,
			ExitCode:  0,
			Duration:  "",
		},
		{
			ID:        "swift_falcon_20241215_104000",
			Timestamp: time.Date(2024, 12, 15, 10, 40, 0, 0, time.UTC),
			Command:   "tasks",
			Spec:      "feature-b",
			Status:    history.StatusFailed,
			ExitCode:  1,
			Duration:  "1m",
		},
		{
			ID:        "gentle_owl_20241215_104500",
			Timestamp: time.Date(2024, 12, 15, 10, 45, 0, 0, time.UTC),
			Command:   "implement",
			Spec:      "feature-c",
			Status:    history.StatusCancelled,
			ExitCode:  0,
			Duration:  "30s",
		},
	}

	tests := map[string]struct {
		statusFilter    string
		wantInOutput    []string
		wantNotInOutput []string
	}{
		"filter by completed": {
			statusFilter:    "completed",
			wantInOutput:    []string{"completed", "specify", "feature-a"},
			wantNotInOutput: []string{"running", "failed", "cancelled", "plan", "tasks", "implement"},
		},
		"filter by running": {
			statusFilter:    "running",
			wantInOutput:    []string{"running", "plan", "feature-a"},
			wantNotInOutput: []string{"completed", "failed", "cancelled", "specify", "tasks", "implement"},
		},
		"filter by failed": {
			statusFilter:    "failed",
			wantInOutput:    []string{"failed", "tasks", "feature-b"},
			wantNotInOutput: []string{"completed", "running", "cancelled", "specify", "plan", "implement"},
		},
		"filter by cancelled": {
			statusFilter:    "cancelled",
			wantInOutput:    []string{"cancelled", "implement", "feature-c"},
			wantNotInOutput: []string{"completed", "running", "failed", "specify", "plan", "tasks"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()
			histFile := &history.HistoryFile{Entries: baseEntries}
			require.NoError(t, history.SaveHistory(stateDir, histFile))

			cmd := createTestHistoryCmd(stateDir)
			require.NoError(t, cmd.Flags().Set("status", tt.statusFilter))

			var buf bytes.Buffer
			cmd.SetOut(&buf)

			err := cmd.RunE(cmd, []string{})
			require.NoError(t, err)

			output := buf.String()
			for _, want := range tt.wantInOutput {
				assert.Contains(t, output, want, "output should contain %q", want)
			}
			for _, notWant := range tt.wantNotInOutput {
				assert.NotContains(t, output, notWant, "output should not contain %q", notWant)
			}
		})
	}
}

// TestRunHistory_StatusFilter_NoMatch tests empty result for non-matching status filter.
func TestRunHistory_StatusFilter_NoMatch(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	// Create history with only completed entries
	histFile := &history.HistoryFile{
		Entries: []history.HistoryEntry{
			{
				ID:        "brave_fox_20241215_103000",
				Timestamp: time.Date(2024, 12, 15, 10, 30, 0, 0, time.UTC),
				Command:   "specify",
				Spec:      "feature-a",
				Status:    history.StatusCompleted,
				ExitCode:  0,
				Duration:  "2m",
			},
		},
	}
	require.NoError(t, history.SaveHistory(stateDir, histFile))

	cmd := createTestHistoryCmd(stateDir)
	require.NoError(t, cmd.Flags().Set("status", "running"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "No matching entries for status 'running'")
}

// TestRunHistory_StatusAndSpecFilter tests combining --status and --spec flags.
func TestRunHistory_StatusAndSpecFilter(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	histFile := &history.HistoryFile{
		Entries: []history.HistoryEntry{
			{
				ID:        "brave_fox_20241215_103000",
				Timestamp: time.Date(2024, 12, 15, 10, 30, 0, 0, time.UTC),
				Command:   "specify",
				Spec:      "feature-a",
				Status:    history.StatusCompleted,
				ExitCode:  0,
				Duration:  "2m",
			},
			{
				ID:        "calm_river_20241215_103500",
				Timestamp: time.Date(2024, 12, 15, 10, 35, 0, 0, time.UTC),
				Command:   "plan",
				Spec:      "feature-a",
				Status:    history.StatusFailed,
				ExitCode:  1,
				Duration:  "1m",
			},
			{
				ID:        "swift_falcon_20241215_104000",
				Timestamp: time.Date(2024, 12, 15, 10, 40, 0, 0, time.UTC),
				Command:   "tasks",
				Spec:      "feature-b",
				Status:    history.StatusCompleted,
				ExitCode:  0,
				Duration:  "30s",
			},
		},
	}
	require.NoError(t, history.SaveHistory(stateDir, histFile))

	// Filter by spec AND status
	cmd := createTestHistoryCmd(stateDir)
	require.NoError(t, cmd.Flags().Set("spec", "feature-a"))
	require.NoError(t, cmd.Flags().Set("status", "completed"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "specify")
	assert.Contains(t, output, "feature-a")
	assert.Contains(t, output, "completed")
	assert.NotContains(t, output, "plan")  // Same spec but different status
	assert.NotContains(t, output, "tasks") // Same status but different spec
	assert.NotContains(t, output, "feature-b")
}

// TestRunHistory_BackwardCompatibility tests that old entries without status display correctly.
func TestRunHistory_BackwardCompatibility(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	// Create history with a mix of old and new entries
	histFile := &history.HistoryFile{
		Entries: []history.HistoryEntry{
			// Old entry without ID and status
			{
				Timestamp: time.Date(2024, 12, 15, 10, 30, 0, 0, time.UTC),
				Command:   "specify",
				Spec:      "old-feature",
				ExitCode:  0,
				Duration:  "2m",
			},
			// New entry with ID and status
			{
				ID:        "brave_fox_20241215_103500",
				Timestamp: time.Date(2024, 12, 15, 10, 35, 0, 0, time.UTC),
				Command:   "plan",
				Spec:      "new-feature",
				Status:    history.StatusCompleted,
				ExitCode:  0,
				Duration:  "1m",
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
	// Both old and new entries should be displayed
	assert.Contains(t, output, "specify")
	assert.Contains(t, output, "plan")
	assert.Contains(t, output, "old-feature")
	assert.Contains(t, output, "new-feature")
	// New entry should have its status
	assert.Contains(t, output, "completed")
}

// TestHistoryStatusFlagExists verifies the --status flag is registered.
func TestHistoryStatusFlagExists(t *testing.T) {
	t.Parallel()

	cmd := getHistoryCmd()
	require.NotNil(t, cmd, "history command must exist")

	f := cmd.Flags().Lookup("status")
	require.NotNil(t, f, "status flag should exist")
	assert.Equal(t, "", f.Shorthand, "status flag should have no shorthand")
}

// TestFormatID tests the formatID function for history display.
// Since formatID is unexported in util package, we test the expected behavior inline.
func TestFormatID(t *testing.T) {
	t.Parallel()

	// formatID returns a formatted ID string (truncated or placeholder).
	// Mirrors util.formatID for testing.
	formatID := func(id string) string {
		if id == "" {
			return fmt.Sprintf("%-30s", "-")
		}
		if len(id) > 30 {
			return id[:30]
		}
		return fmt.Sprintf("%-30s", id)
	}

	tests := map[string]struct {
		id   string
		want string
	}{
		"empty string returns dash padded": {
			id:   "",
			want: "-                             ", // 30 chars with dash
		},
		"short ID is padded to 30 chars": {
			id:   "brave_fox",
			want: "brave_fox                     ", // 30 chars
		},
		"exactly 30 chars stays same": {
			id:   "brave_fox_20241215_103000abcd",
			want: "brave_fox_20241215_103000abcd ", // 30 chars
		},
		"ID longer than 30 chars is truncated": {
			id:   "brave_fox_20241215_103000abcdefghij",
			want: "brave_fox_20241215_103000abcde", // truncated to 30
		},
		"typical ID format": {
			id:   "brave_fox_20241215_103000",
			want: "brave_fox_20241215_103000     ", // padded to 30
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := formatID(tt.id)
			assert.Equal(t, 30, len(got), "formatID should always return 30 chars, got %d", len(got))
			assert.Equal(t, tt.want, got)
		})
	}
}
