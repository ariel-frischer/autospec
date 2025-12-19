package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/ariel-frischer/autospec/internal/validation"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:          "status [spec-name]",
	Aliases:      []string{"st"},
	Short:        "Show implementation progress for current feature (st)",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		verbose, _ := cmd.Flags().GetBool("verbose")

		// Load configuration
		cfg, err := config.Load(configPath)
		if err != nil {
			cliErr := clierrors.ConfigParseError(configPath, err)
			clierrors.PrintError(cliErr)
			return cliErr
		}

		// Detect or get spec
		var metadata *spec.Metadata
		if len(args) > 0 {
			metadata, err = spec.GetSpecMetadata(cfg.SpecsDir, args[0])
			if err == nil {
				metadata.Detection = spec.DetectionExplicit
			}
		} else {
			metadata, err = spec.DetectCurrentSpec(cfg.SpecsDir)
		}
		if err != nil {
			return fmt.Errorf("failed to detect spec: %w", err)
		}
		shared.PrintSpecInfo(metadata)

		// Check which artifact files exist
		artifacts := []string{"spec.yaml", "plan.yaml", "tasks.yaml"}
		var existing []string
		for _, artifact := range artifacts {
			path := filepath.Join(metadata.Directory, artifact)
			if _, err := os.Stat(path); err == nil {
				existing = append(existing, artifact)
			}
		}

		// Show artifacts
		if len(existing) > 0 {
			fmt.Printf("  artifacts: %v\n", existing)
		} else {
			fmt.Println("  artifacts: none")
		}

		// Get tasks file path (prefers .yaml over .md)
		tasksPath := validation.GetTasksFilePath(metadata.Directory)

		// Get task stats (only if tasks file exists)
		stats, err := validation.GetTaskStats(tasksPath)
		if err == nil {
			fmt.Print(validation.FormatTaskSummary(stats))
		}

		// Get risk stats from plan.yaml (if plan.yaml exists)
		planPath := validation.GetPlanFilePath(metadata.Directory)
		riskStats, _ := validation.GetRiskStats(planPath)
		if riskStats != nil {
			fmt.Print(validation.FormatRiskSummary(riskStats))
		}

		// Display blocked tasks with reasons
		if err == nil && stats != nil && stats.BlockedTasks > 0 {
			displayBlockedTasks(tasksPath)
		}

		// Show phase details in verbose mode
		if verbose && stats != nil {
			fmt.Println()
			for _, phase := range stats.PhaseStats {
				status := "[ ]"
				if phase.IsComplete {
					status = "[✓]"
				} else if phase.CompletedTasks > 0 {
					status = "[~]"
				}
				fmt.Printf("  %s Phase %d: %s (%d/%d)\n",
					status, phase.Number, phase.Title, phase.CompletedTasks, phase.TotalTasks)
			}
		}

		return nil
	},
}

func init() {
	statusCmd.GroupID = shared.GroupGettingStarted
	statusCmd.Flags().BoolP("verbose", "v", false, "Show all tasks, not just unchecked")
}

// displayBlockedTasks shows blocked tasks with their reasons
func displayBlockedTasks(tasksPath string) {
	tasks, err := validation.GetAllTasks(tasksPath)
	if err != nil {
		return
	}

	blockedTasks := filterBlockedTasks(tasks)
	if len(blockedTasks) == 0 {
		return
	}

	fmt.Println("\n  Blocked tasks:")
	for _, task := range blockedTasks {
		reason := formatBlockedReason(task.BlockedReason)
		fmt.Printf("    %s: %s\n", task.ID, truncateStatusReason(task.Title, 50))
		fmt.Printf("       Reason: %s\n", reason)
	}
}

// filterBlockedTasks returns only tasks with Blocked status
func filterBlockedTasks(tasks []validation.TaskItem) []validation.TaskItem {
	var blocked []validation.TaskItem
	for _, task := range tasks {
		if strings.EqualFold(task.Status, "Blocked") {
			blocked = append(blocked, task)
		}
	}
	return blocked
}

// formatBlockedReason formats the blocked reason for display
// Returns "(no reason provided)" if reason is empty
func formatBlockedReason(reason string) string {
	if reason == "" {
		return "(no reason provided)"
	}
	return truncateStatusReason(reason, 80)
}

// truncateStatusReason truncates a string to maxLen characters with ellipsis
func truncateStatusReason(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
