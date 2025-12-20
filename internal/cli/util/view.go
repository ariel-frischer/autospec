package util

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/ariel-frischer/autospec/internal/validation"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// SpecSummary contains aggregated information about a single spec for dashboard display.
type SpecSummary struct {
	Name             string    // Spec directory name (e.g., '063-view-dashboard')
	Status           string    // Spec status from spec.yaml (Draft, In Progress, Completed, etc.)
	TaskProgress     string    // Task completion formatted as 'X/Y tasks' or 'no tasks'
	CompletedTasks   int       // Number of completed tasks
	TotalTasks       int       // Total number of tasks
	LastModified     time.Time // Most recent modification time of files in spec directory
	ArtifactsPresent []string  // List of existing artifacts (spec.yaml, plan.yaml, tasks.yaml)
}

// DashboardStats contains project-wide statistics for the dashboard header.
type DashboardStats struct {
	TotalSpecs      int // Total count of spec directories
	InProgressCount int // Specs with status Draft, In Progress, or Review
	CompletedCount  int // Specs with status Completed or 100% task completion
	SkippedCount    int // Specs with status Rejected or Skipped
}

var viewCmd = &cobra.Command{
	Use:          "view",
	Short:        "Show dashboard overview of all specs in the project",
	Long:         `Display a dashboard showing project-wide spec statistics, recent specs with task progress, and completed specs.`,
	SilenceUsage: true,
	RunE:         runView,
}

var viewLimit int

func init() {
	viewCmd.GroupID = shared.GroupGettingStarted
	viewCmd.Flags().IntVarP(&viewLimit, "limit", "l", 0, "Number of recent specs to display (default: from config or 5)")
}

// runView executes the view command logic.
func runView(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")

	cfg, err := config.Load(configPath)
	if err != nil {
		cliErr := clierrors.ConfigParseError(configPath, err)
		clierrors.PrintError(cliErr)
		return cliErr
	}

	limit := resolveLimit(viewLimit, cfg.ViewLimit)
	specsDir := resolveSpecsDir(cmd, cfg.SpecsDir)

	summaries, err := scanSpecsDir(specsDir)
	if err != nil {
		return fmt.Errorf("scanning specs directory: %w", err)
	}

	if len(summaries) == 0 {
		fmt.Printf("No specs found in %s/\n", specsDir)
		return nil
	}

	stats := computeDashboardStats(summaries)
	renderDashboardHeader(stats)
	renderRecentSpecs(summaries, limit)
	renderCompletedSpecs(summaries)

	return nil
}

// resolveLimit determines the effective limit to use.
// Priority: CLI flag > config value > default (5)
func resolveLimit(flagValue, configValue int) int {
	if flagValue > 0 {
		return flagValue
	}
	if configValue > 0 {
		return configValue
	}
	return 5
}

// resolveSpecsDir determines the effective specs directory.
// Priority: CLI flag > config value
func resolveSpecsDir(cmd *cobra.Command, configValue string) string {
	flagValue, _ := cmd.Flags().GetString("specs-dir")
	if flagValue != "" && flagValue != "./specs" {
		return flagValue
	}
	return configValue
}

// scanSpecsDir scans the specs directory and returns summaries for all valid specs.
// Specs are sorted by LastModified descending (most recent first).
func scanSpecsDir(specsDir string) ([]SpecSummary, error) {
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading specs directory: %w", err)
	}

	var summaries []SpecSummary
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		specDir := filepath.Join(specsDir, entry.Name())
		summary, err := getSpecSummary(specDir, entry.Name())
		if err != nil {
			// Skip directories without spec.yaml or with parse errors
			continue
		}
		summaries = append(summaries, summary)
	}

	// Sort by LastModified descending
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].LastModified.After(summaries[j].LastModified)
	})

	return summaries, nil
}

// getSpecSummary gathers information about a single spec directory.
func getSpecSummary(specDir, name string) (SpecSummary, error) {
	specPath := filepath.Join(specDir, "spec.yaml")
	if _, err := os.Stat(specPath); err != nil {
		return SpecSummary{}, fmt.Errorf("spec.yaml not found: %w", err)
	}

	summary := SpecSummary{
		Name:             name,
		Status:           "Unknown",
		TaskProgress:     "no tasks",
		ArtifactsPresent: []string{},
	}

	summary.Status = parseSpecStatus(specPath)
	summary.ArtifactsPresent = detectArtifacts(specDir)
	summary.LastModified = getLatestModTime(specDir)
	summary.CompletedTasks, summary.TotalTasks, summary.TaskProgress = getTaskProgress(specDir)

	return summary, nil
}

// parseSpecStatus extracts the status field from spec.yaml.
func parseSpecStatus(specPath string) string {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return "parse error"
	}

	var spec struct {
		Feature struct {
			Status string `yaml:"status"`
		} `yaml:"feature"`
	}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return "parse error"
	}

	if spec.Feature.Status == "" {
		return "Unknown"
	}
	return spec.Feature.Status
}

// detectArtifacts checks which artifact files exist in the spec directory.
func detectArtifacts(specDir string) []string {
	artifacts := []string{"spec.yaml", "plan.yaml", "tasks.yaml"}
	var present []string
	for _, artifact := range artifacts {
		if _, err := os.Stat(filepath.Join(specDir, artifact)); err == nil {
			present = append(present, artifact)
		}
	}
	return present
}

// getLatestModTime returns the most recent modification time of files in the spec directory.
func getLatestModTime(specDir string) time.Time {
	var latest time.Time
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return latest
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
	}
	return latest
}

// getTaskProgress retrieves task completion stats from tasks.yaml if it exists.
func getTaskProgress(specDir string) (completed, total int, progress string) {
	tasksPath := validation.GetTasksFilePath(specDir)
	stats, err := validation.GetTaskStats(tasksPath)
	if err != nil {
		return 0, 0, "no tasks"
	}
	if stats.TotalTasks == 0 {
		return 0, 0, "0 tasks"
	}
	return stats.CompletedTasks, stats.TotalTasks, fmt.Sprintf("%d/%d tasks", stats.CompletedTasks, stats.TotalTasks)
}

// computeDashboardStats computes aggregate statistics from all spec summaries.
func computeDashboardStats(summaries []SpecSummary) DashboardStats {
	stats := DashboardStats{TotalSpecs: len(summaries)}

	for _, s := range summaries {
		statusLower := strings.ToLower(s.Status)
		switch {
		case isCompletedStatus(statusLower, s.CompletedTasks, s.TotalTasks):
			stats.CompletedCount++
		case isSkippedStatus(statusLower):
			stats.SkippedCount++
		default:
			stats.InProgressCount++
		}
	}

	return stats
}

// isCompletedStatus returns true if the spec should be counted as completed.
func isCompletedStatus(statusLower string, completed, total int) bool {
	if statusLower == "completed" || statusLower == "done" || statusLower == "complete" {
		return true
	}
	// 100% task completion also counts as completed
	return total > 0 && completed == total
}

// isSkippedStatus returns true if the spec should be counted as skipped/rejected.
func isSkippedStatus(statusLower string) bool {
	return statusLower == "rejected" || statusLower == "skipped"
}

// renderDashboardHeader outputs the project-wide statistics header.
func renderDashboardHeader(stats DashboardStats) {
	fmt.Println("Spec Dashboard")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Total specs:   %d\n", stats.TotalSpecs)
	fmt.Printf("In progress:   %d\n", stats.InProgressCount)
	fmt.Printf("Completed:     %d\n", stats.CompletedCount)
	fmt.Printf("Skipped:       %d\n", stats.SkippedCount)
	fmt.Println()
}

// renderRecentSpecs outputs the recent specs section.
func renderRecentSpecs(summaries []SpecSummary, limit int) {
	fmt.Printf("Recent Specs (top %d)\n", limit)
	fmt.Println(strings.Repeat("-", 40))

	displayed := 0
	for _, s := range summaries {
		if displayed >= limit {
			break
		}
		fmt.Printf("  %-30s %s\n", truncate(s.Name, 30), s.Status)
		fmt.Printf("    Progress: %s\n", s.TaskProgress)
		displayed++
	}
	fmt.Println()
}

// renderCompletedSpecs outputs the completed specs section.
func renderCompletedSpecs(summaries []SpecSummary) {
	var completed []SpecSummary
	for _, s := range summaries {
		statusLower := strings.ToLower(s.Status)
		if isCompletedStatus(statusLower, s.CompletedTasks, s.TotalTasks) {
			completed = append(completed, s)
		}
	}

	if len(completed) == 0 {
		return
	}

	fmt.Println("Completed Specs")
	fmt.Println(strings.Repeat("-", 40))
	for _, s := range completed {
		fmt.Printf("  %-30s %s\n", truncate(s.Name, 30), s.TaskProgress)
	}
	fmt.Println()
}

// truncate shortens a string to maxLen characters with ellipsis if needed.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
