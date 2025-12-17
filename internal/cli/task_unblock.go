package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	unblockStatus string
)

var taskUnblockCmd = &cobra.Command{
	Use:   "unblock <task-id>",
	Short: "Unblock a task and set its status",
	Long: `Unblock a task and optionally specify the new status.

This command changes a blocked task's status back to Pending (default) or
another specified status, and removes the blocked_reason field.

If the task is not currently blocked, a warning is shown and no changes are made.`,
	Example: `  # Unblock a task (defaults to Pending status)
  autospec task unblock T001

  # Unblock and set to InProgress to immediately start working
  autospec task unblock T001 --status InProgress

  # Unblock multiple tasks
  autospec task unblock T001
  autospec task unblock T002`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskUnblock,
}

func init() {
	taskUnblockCmd.Flags().StringVarP(&unblockStatus, "status", "s", "Pending", "Status to set after unblocking (Pending or InProgress)")
	taskCmd.AddCommand(taskUnblockCmd)
}

func runTaskUnblock(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	// Validate task ID format
	if !taskIDPattern.MatchString(taskID) {
		return fmt.Errorf("invalid task ID format: %s (expected T followed by digits, e.g., T001)", taskID)
	}

	// Validate target status (only Pending or InProgress allowed for unblock)
	if err := validateUnblockStatus(unblockStatus); err != nil {
		return err
	}

	// Load config
	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := config.Load(configPath)
	if err != nil {
		cliErr := clierrors.ConfigParseError(configPath, err)
		clierrors.PrintError(cliErr)
		return cliErr
	}

	// Detect current spec
	metadata, err := spec.DetectCurrentSpec(cfg.SpecsDir)
	if err != nil {
		return fmt.Errorf("detecting spec: %w", err)
	}
	PrintSpecInfo(metadata)

	// Find tasks.yaml
	tasksPath := filepath.Join(metadata.Directory, "tasks.yaml")
	if _, err := os.Stat(tasksPath); os.IsNotExist(err) {
		return fmt.Errorf("tasks.yaml not found: %s\nRun /autospec.tasks first to generate tasks", tasksPath)
	}

	// Read and parse tasks.yaml
	data, err := os.ReadFile(tasksPath)
	if err != nil {
		return fmt.Errorf("reading tasks.yaml: %w", err)
	}

	// Parse YAML preserving structure
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("parsing tasks.yaml: %w", err)
	}

	// Find and unblock the task
	result := findAndUnblockTask(&root, taskID, unblockStatus)
	if !result.found {
		return fmt.Errorf("task not found: %s\nCheck that the task ID exists in: %s", taskID, tasksPath)
	}

	// Handle non-blocked task case
	if !result.wasBlocked {
		fmt.Printf("⚠ Task %s is not blocked (status: %s) - no changes made\n", taskID, result.previousStatus)
		return nil
	}

	// Write back the updated YAML
	output, err := yaml.Marshal(&root)
	if err != nil {
		return fmt.Errorf("serializing tasks.yaml: %w", err)
	}

	if err := os.WriteFile(tasksPath, output, 0644); err != nil {
		return fmt.Errorf("writing tasks.yaml: %w", err)
	}

	printUnblockResult(taskID, result)
	return nil
}

// validateUnblockStatus ensures the target status is valid for unblocking
func validateUnblockStatus(status string) error {
	if status == "Pending" || status == "InProgress" {
		return nil
	}
	return fmt.Errorf("invalid unblock status: %s (must be Pending or InProgress)", status)
}

// unblockResult holds the result of an unblock operation
type unblockResult struct {
	found          bool
	wasBlocked     bool
	previousStatus string
	hadReason      bool
	previousReason string
}

// printUnblockResult prints a user-friendly message about the unblock operation
func printUnblockResult(taskID string, result unblockResult) {
	fmt.Printf("✓ Task %s: Blocked -> %s\n", taskID, unblockStatus)
	if result.hadReason {
		fmt.Printf("  Previous reason: %s\n", truncateReason(result.previousReason, 60))
	}
}

// findAndUnblockTask traverses the YAML node tree to find and unblock a task by ID.
// It sets the status to the target status and removes the blocked_reason field.
func findAndUnblockTask(node *yaml.Node, taskID, targetStatus string) unblockResult {
	if node == nil {
		return unblockResult{}
	}

	switch node.Kind {
	case yaml.DocumentNode:
		return findAndUnblockTaskInDocument(node, taskID, targetStatus)
	case yaml.MappingNode:
		return findAndUnblockTaskInMapping(node, taskID, targetStatus)
	case yaml.SequenceNode:
		return findAndUnblockTaskInSequence(node, taskID, targetStatus)
	}

	return unblockResult{}
}

func findAndUnblockTaskInDocument(node *yaml.Node, taskID, targetStatus string) unblockResult {
	for _, child := range node.Content {
		if result := findAndUnblockTask(child, taskID, targetStatus); result.found {
			return result
		}
	}
	return unblockResult{}
}

func findAndUnblockTaskInSequence(node *yaml.Node, taskID, targetStatus string) unblockResult {
	for _, child := range node.Content {
		if result := findAndUnblockTask(child, taskID, targetStatus); result.found {
			return result
		}
	}
	return unblockResult{}
}

func findAndUnblockTaskInMapping(node *yaml.Node, taskID, targetStatus string) unblockResult {
	// Check if this is a task node with matching ID
	var idNode, statusNode *yaml.Node
	var reasonKeyIdx int = -1

	for i := 0; i < len(node.Content)-1; i += 2 {
		key := node.Content[i]
		value := node.Content[i+1]

		switch key.Value {
		case "id":
			if value.Value == taskID {
				idNode = value
			}
		case "status":
			statusNode = value
		case "blocked_reason":
			reasonKeyIdx = i
		}
	}

	// If we found the task with matching ID and status field
	if idNode != nil && statusNode != nil {
		return updateTaskUnblockFields(node, statusNode, reasonKeyIdx, targetStatus)
	}

	// Otherwise recurse into all values
	for i := 1; i < len(node.Content); i += 2 {
		if result := findAndUnblockTask(node.Content[i], taskID, targetStatus); result.found {
			return result
		}
	}

	return unblockResult{}
}

// updateTaskUnblockFields updates the status and removes blocked_reason field on a task node
func updateTaskUnblockFields(node *yaml.Node, statusNode *yaml.Node, reasonKeyIdx int, targetStatus string) unblockResult {
	result := unblockResult{
		found:          true,
		previousStatus: statusNode.Value,
		wasBlocked:     statusNode.Value == "Blocked",
	}

	// If not currently blocked, return early
	if !result.wasBlocked {
		return result
	}

	// Update status to target status
	statusNode.Value = targetStatus

	// Remove blocked_reason field if present
	if reasonKeyIdx >= 0 {
		result.hadReason = true
		result.previousReason = node.Content[reasonKeyIdx+1].Value
		removeBlockedReason(node, reasonKeyIdx)
	}

	return result
}

// removeBlockedReason removes the blocked_reason key-value pair from the node
func removeBlockedReason(node *yaml.Node, keyIdx int) {
	// Remove both the key and value (2 elements starting at keyIdx)
	node.Content = append(node.Content[:keyIdx], node.Content[keyIdx+2:]...)
}
