package cli

import (
	"fmt"
	"strings"

	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/ariel-frischer/autospec/internal/validation"
	"github.com/spf13/cobra"
)

var (
	listBlocked    bool
	listPending    bool
	listInProgress bool
	listCompleted  bool
)

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks with optional status filters",
	Long: `List all tasks from the current feature's tasks.yaml file.

By default, all tasks are listed. Use flags to filter by status:
  --blocked     Show only blocked tasks
  --pending     Show only pending tasks
  --in-progress Show only in-progress tasks
  --completed   Show only completed tasks

Multiple filters can be combined to show tasks matching any of the specified statuses.`,
	Example: `  # List all tasks
  autospec task list

  # List only blocked tasks
  autospec task list --blocked

  # List pending and in-progress tasks
  autospec task list --pending --in-progress

  # List all non-completed tasks
  autospec task list --pending --in-progress --blocked`,
	RunE: runTaskList,
}

func init() {
	taskListCmd.Flags().BoolVar(&listBlocked, "blocked", false, "Show only blocked tasks")
	taskListCmd.Flags().BoolVar(&listPending, "pending", false, "Show only pending tasks")
	taskListCmd.Flags().BoolVar(&listInProgress, "in-progress", false, "Show only in-progress tasks")
	taskListCmd.Flags().BoolVar(&listCompleted, "completed", false, "Show only completed tasks")
	taskCmd.AddCommand(taskListCmd)
}

func runTaskList(cmd *cobra.Command, args []string) error {
	cfg, tasksPath, err := loadTasksConfig(cmd)
	if err != nil {
		return err
	}
	_ = cfg // unused but needed for config loading pattern

	tasks, err := validation.GetAllTasks(tasksPath)
	if err != nil {
		return fmt.Errorf("loading tasks: %w", err)
	}

	filtered := filterTasksByStatus(tasks)
	if len(filtered) == 0 {
		fmt.Println("No tasks found matching the specified filters.")
		return nil
	}

	printTaskList(filtered)
	return nil
}

// loadTasksConfig loads config and returns the tasks.yaml path
func loadTasksConfig(cmd *cobra.Command) (*config.Configuration, string, error) {
	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := config.Load(configPath)
	if err != nil {
		cliErr := clierrors.ConfigParseError(configPath, err)
		clierrors.PrintError(cliErr)
		return nil, "", cliErr
	}

	metadata, err := spec.DetectCurrentSpec(cfg.SpecsDir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to detect spec: %w", err)
	}
	PrintSpecInfo(metadata)

	tasksPath := validation.GetTasksFilePath(metadata.Directory)
	return cfg, tasksPath, nil
}

// filterTasksByStatus filters tasks based on the status flags
// If no flags are set, all tasks are returned
func filterTasksByStatus(tasks []validation.TaskItem) []validation.TaskItem {
	// If no filters specified, return all tasks
	if !listBlocked && !listPending && !listInProgress && !listCompleted {
		return tasks
	}

	var filtered []validation.TaskItem
	for _, task := range tasks {
		if matchesStatusFilter(task.Status) {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// matchesStatusFilter checks if a task status matches any of the active filters
func matchesStatusFilter(status string) bool {
	statusLower := strings.ToLower(status)

	if listBlocked && statusLower == "blocked" {
		return true
	}
	if listPending && statusLower == "pending" {
		return true
	}
	if listInProgress && (statusLower == "inprogress" || statusLower == "in-progress" || statusLower == "in_progress") {
		return true
	}
	if listCompleted && (statusLower == "completed" || statusLower == "done" || statusLower == "complete") {
		return true
	}
	return false
}

// printTaskList prints the filtered task list
func printTaskList(tasks []validation.TaskItem) {
	fmt.Printf("Tasks (%d):\n", len(tasks))

	for _, task := range tasks {
		statusIcon := getStatusIcon(task.Status)
		fmt.Printf("  %s %s [%s] %s\n", statusIcon, task.ID, task.Status, task.Title)

		// Show blocked reason for blocked tasks
		if strings.EqualFold(task.Status, "Blocked") {
			reason := task.BlockedReason
			if reason == "" {
				reason = "(no reason provided)"
			} else if len(reason) > 80 {
				reason = reason[:77] + "..."
			}
			fmt.Printf("       Reason: %s\n", reason)
		}
	}
}

// getStatusIcon returns a visual icon for the task status
func getStatusIcon(status string) string {
	switch strings.ToLower(status) {
	case "completed", "done", "complete":
		return "[✓]"
	case "inprogress", "in-progress", "in_progress":
		return "[~]"
	case "blocked":
		return "[!]"
	default: // Pending
		return "[ ]"
	}
}
