package util

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/history"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:          "history",
	Short:        "View command execution history",
	Long:         `View a log of all autospec command executions with timestamp, command name, spec, exit code, and duration.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		stateDir := getDefaultStateDir()
		return runHistoryWithStateDir(cmd, stateDir)
	},
}

func init() {
	historyCmd.GroupID = shared.GroupConfiguration
	historyCmd.Flags().StringP("spec", "s", "", "Filter by spec name")
	historyCmd.Flags().IntP("limit", "n", 0, "Limit to last N entries (most recent)")
	historyCmd.Flags().Bool("clear", false, "Clear all history")
	historyCmd.Flags().String("status", "", "Filter by status (running, completed, failed, cancelled)")
}

// getDefaultStateDir returns the default state directory path.
func getDefaultStateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".autospec", "state")
}

// runHistoryWithStateDir runs the history command with a custom state directory.
func runHistoryWithStateDir(cmd *cobra.Command, stateDir string) error {
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

	// Get filtered entries
	entries := filterEntries(histFile.Entries, specFilter, statusFilter, limit)

	// Handle empty result
	if len(entries) == 0 {
		msg := buildEmptyMessage(specFilter, statusFilter)
		fmt.Fprintln(cmd.OutOrStdout(), msg)
		return nil
	}

	// Display entries
	displayEntries(cmd, entries)
	return nil
}

// buildEmptyMessage creates an appropriate message when no entries match filters.
func buildEmptyMessage(specFilter, statusFilter string) string {
	if specFilter != "" && statusFilter != "" {
		return fmt.Sprintf("No matching entries for spec '%s' and status '%s'.", specFilter, statusFilter)
	}
	if specFilter != "" {
		return fmt.Sprintf("No matching entries for spec '%s'.", specFilter)
	}
	if statusFilter != "" {
		return fmt.Sprintf("No matching entries for status '%s'.", statusFilter)
	}
	return "No history available."
}

// filterEntries filters and limits history entries.
func filterEntries(entries []history.HistoryEntry, specFilter, statusFilter string, limit int) []history.HistoryEntry {
	var result []history.HistoryEntry

	// Apply spec and status filters
	for _, entry := range entries {
		if !matchesFilters(entry, specFilter, statusFilter) {
			continue
		}
		result = append(result, entry)
	}

	// Apply limit (most recent entries)
	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}

	return result
}

// matchesFilters checks if an entry matches the given spec and status filters.
func matchesFilters(entry history.HistoryEntry, specFilter, statusFilter string) bool {
	if specFilter != "" && entry.Spec != specFilter {
		return false
	}
	if statusFilter != "" && entry.Status != statusFilter {
		return false
	}
	return true
}

// displayEntries formats and displays history entries.
func displayEntries(cmd *cobra.Command, entries []history.HistoryEntry) {
	out := cmd.OutOrStdout()

	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	for _, entry := range entries {
		// Format timestamp
		timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")

		// Format status with color coding
		statusStr := formatStatus(entry.Status, green, yellow, red)

		// Color exit code
		exitCodeStr := fmt.Sprintf("%d", entry.ExitCode)
		if entry.ExitCode == 0 {
			exitCodeStr = green(exitCodeStr)
		} else {
			exitCodeStr = red(exitCodeStr)
		}

		// Format spec (or "none" if empty)
		spec := entry.Spec
		if spec == "" {
			spec = "-"
		}

		// Format ID (truncate or show "-" if empty)
		id := formatID(entry.ID)

		fmt.Fprintf(out, "%s  %-30s  %-10s  %s  %-15s  exit=%s  %s\n",
			cyan(timestamp),
			id,
			statusStr,
			fmt.Sprintf("%-12s", entry.Command),
			spec,
			exitCodeStr,
			entry.Duration,
		)
	}
}

// formatStatus returns a color-coded status string.
func formatStatus(status string, green, yellow, red func(a ...interface{}) string) string {
	switch status {
	case history.StatusCompleted:
		return green(fmt.Sprintf("%-10s", status))
	case history.StatusRunning:
		return yellow(fmt.Sprintf("%-10s", status))
	case history.StatusFailed, history.StatusCancelled:
		return red(fmt.Sprintf("%-10s", status))
	default:
		// Old entries without status field
		if status == "" {
			return fmt.Sprintf("%-10s", "-")
		}
		return fmt.Sprintf("%-10s", status)
	}
}

// formatID returns a formatted ID string (truncated or placeholder).
func formatID(id string) string {
	if id == "" {
		return fmt.Sprintf("%-30s", "-")
	}
	// IDs are in adjective_noun_YYYYMMDD_HHMMSS format (~26+ chars)
	// Display full ID as it's designed to be memorable
	if len(id) > 30 {
		return id[:30]
	}
	return fmt.Sprintf("%-30s", id)
}
