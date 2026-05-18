package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Valid task statuses
var validStatuses = []string{"Pending", "InProgress", "Completed", "Blocked"}

// taskIDPattern matches task IDs like T001, T1, T123
var taskIDPattern = regexp.MustCompile(`^T\d+$`)

var updateTaskCmd = &cobra.Command{
	Use:   "update-task <task-id> <status>",
	Short: "Update the status of a task in tasks.yaml",
	Long: `Update the status of an individual task in the current feature's tasks.yaml file.

This command is designed for use during implementation to mark tasks as
in-progress, completed, or blocked without manually editing the YAML file.

Valid status values:
  - Pending     Task not yet started
  - InProgress  Task currently being worked on
  - Completed   Task finished successfully
  - Blocked     Task blocked by dependency or issue`,
	Example: `  # Start working on a task
  autospec update-task T001 InProgress

  # Mark a task as completed
  autospec update-task T001 Completed

  # Mark a task as blocked
  autospec update-task T015 Blocked`,
	Args: cobra.ExactArgs(2),
	RunE: runUpdateTask,
}

func init() {
	updateTaskCmd.GroupID = GroupInternal
	rootCmd.AddCommand(updateTaskCmd)
}

func runUpdateTask(cmd *cobra.Command, args []string) error {
	taskID := args[0]
	newStatus := args[1]

	// Validate task ID format
	if !taskIDPattern.MatchString(taskID) {
		return fmt.Errorf("invalid task ID format: %s (expected T followed by digits, e.g., T001)", taskID)
	}

	// Validate status
	if !isValidStatus(newStatus) {
		cliErr := clierrors.InvalidTaskStatus(newStatus)
		clierrors.PrintError(cliErr)
		return cliErr
	}

	// Load config
	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := config.Load(configPath)
	if err != nil {
		cliErr := clierrors.ConfigParseError(configPath, err)
		clierrors.PrintError(cliErr)
		return cliErr
	}

	activeFeature, err := resolveCLIActiveFeature(cfg, "tasks.yaml")
	if err != nil {
		return err
	}
	metadata := activeFeature.Metadata
	printCLIActiveFeature(activeFeature)

	// Find tasks.yaml
	tasksPath := filepath.Join(metadata.Directory, "tasks.yaml")
	if _, err := os.Stat(tasksPath); os.IsNotExist(err) {
		return fmt.Errorf("tasks.yaml not found: %s\nRun /autospec.tasks first to generate tasks", tasksPath)
	}

	// Read and parse tasks.yaml
	data, err := os.ReadFile(tasksPath)
	if err != nil {
		return fmt.Errorf("failed to read tasks.yaml: %w", err)
	}

	// Parse YAML preserving structure
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("failed to parse tasks.yaml: %w", err)
	}

	// Find and update the task
	previousStatus, found := findAndUpdateTask(&root, taskID, newStatus)
	if !found {
		return fmt.Errorf("task not found: %s\nCheck that the task ID exists in: %s", taskID, tasksPath)
	}

	// Check if status actually changed
	if previousStatus == newStatus {
		fmt.Printf("Task %s already has status: %s (no change needed)\n", taskID, newStatus)
		return nil
	}

	// Write back the updated YAML
	output, err := yaml.Marshal(&root)
	if err != nil {
		return fmt.Errorf("failed to serialize tasks.yaml: %w", err)
	}

	if err := os.WriteFile(tasksPath, output, 0o644); err != nil {
		return fmt.Errorf("failed to write tasks.yaml: %w", err)
	}

	fmt.Printf("✓ Task %s: %s -> %s\n", taskID, previousStatus, newStatus)
	return nil
}

func isValidStatus(status string) bool {
	for _, valid := range validStatuses {
		if status == valid {
			return true
		}
	}
	return false
}

// findAndUpdateTask traverses the YAML node tree to find and update a task by ID.
// Returns the previous status and whether the task was found.
func findAndUpdateTask(node *yaml.Node, taskID, newStatus string) (string, bool) {
	if node == nil {
		return "", false
	}

	switch node.Kind {
	case yaml.DocumentNode:
		// Document node - recurse into content
		for _, child := range node.Content {
			if prev, found := findAndUpdateTask(child, taskID, newStatus); found {
				return prev, true
			}
		}

	case yaml.MappingNode:
		// Check if this is a task node with matching ID
		var idNode, statusNode *yaml.Node
		for i := 0; i < len(node.Content)-1; i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]

			if key.Value == "id" && value.Value == taskID {
				idNode = value
			}
			if key.Value == "status" {
				statusNode = value
			}
		}

		// If we found both id and status, this is our task
		if idNode != nil && statusNode != nil {
			previousStatus := statusNode.Value
			statusNode.Value = newStatus
			return previousStatus, true
		}

		// Otherwise recurse into all values
		for i := 1; i < len(node.Content); i += 2 {
			if prev, found := findAndUpdateTask(node.Content[i], taskID, newStatus); found {
				return prev, true
			}
		}

	case yaml.SequenceNode:
		// Sequence node - recurse into each item
		for _, child := range node.Content {
			if prev, found := findAndUpdateTask(child, taskID, newStatus); found {
				return prev, true
			}
		}
	}

	return "", false
}
