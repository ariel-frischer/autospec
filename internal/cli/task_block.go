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
	blockReason string
)

var taskBlockCmd = &cobra.Command{
	Use:   "block <task-id>",
	Short: "Block a task with a reason",
	Long: `Block a task and provide a reason explaining why it is blocked.

This command sets the task status to 'Blocked' and stores the blocking reason
in the blocked_reason field. The reason helps track external dependencies,
waiting conditions, or other blockers that prevent task completion.

If the task is already blocked, the reason is updated with the new value.`,
	Example: `  # Block a task waiting for API access
  autospec task block T001 --reason "Waiting for API access from third-party"

  # Block a task with a dependency blocker
  autospec task block T005 --reason "Blocked by external team review"

  # Update the reason for an already blocked task
  autospec task block T001 --reason "Updated: API access approved, waiting for credentials"`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskBlock,
}

func init() {
	taskBlockCmd.Flags().StringVarP(&blockReason, "reason", "r", "", "Reason for blocking the task (required)")
	_ = taskBlockCmd.MarkFlagRequired("reason")
	taskCmd.AddCommand(taskBlockCmd)
}

func runTaskBlock(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	// Validate task ID format
	if !taskIDPattern.MatchString(taskID) {
		return fmt.Errorf("invalid task ID format: %s (expected T followed by digits, e.g., T001)", taskID)
	}

	// Validate reason is not empty
	if blockReason == "" {
		return fmt.Errorf("blocked reason cannot be empty")
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
		return fmt.Errorf("failed to detect spec: %w", err)
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

	// Find and update the task
	result := findAndBlockTask(&root, taskID, blockReason)
	if !result.found {
		return fmt.Errorf("task not found: %s\nCheck that the task ID exists in: %s", taskID, tasksPath)
	}

	// Write back the updated YAML
	output, err := yaml.Marshal(&root)
	if err != nil {
		return fmt.Errorf("serializing tasks.yaml: %w", err)
	}

	if err := os.WriteFile(tasksPath, output, 0644); err != nil {
		return fmt.Errorf("writing tasks.yaml: %w", err)
	}

	printBlockResult(taskID, result)
	return nil
}

// blockResult holds the result of a block operation
type blockResult struct {
	found          bool
	previousStatus string
	hadReason      bool
	previousReason string
}

// printBlockResult prints a user-friendly message about the block operation
func printBlockResult(taskID string, result blockResult) {
	if result.previousStatus == "Blocked" {
		if result.hadReason {
			fmt.Printf("✓ Task %s: updated blocked reason\n", taskID)
			fmt.Printf("  Previous: %s\n", truncateReason(result.previousReason, 60))
			fmt.Printf("  New:      %s\n", truncateReason(blockReason, 60))
		} else {
			fmt.Printf("✓ Task %s: added blocked reason\n", taskID)
		}
	} else {
		fmt.Printf("✓ Task %s: %s -> Blocked\n", taskID, result.previousStatus)
		fmt.Printf("  Reason: %s\n", truncateReason(blockReason, 60))
	}
}

// truncateReason truncates a reason string to maxLen characters with ellipsis
func truncateReason(reason string, maxLen int) string {
	if len(reason) <= maxLen {
		return reason
	}
	return reason[:maxLen-3] + "..."
}

// findAndBlockTask traverses the YAML node tree to find and block a task by ID.
// It sets the status to "Blocked" and adds/updates the blocked_reason field.
func findAndBlockTask(node *yaml.Node, taskID, reason string) blockResult {
	if node == nil {
		return blockResult{}
	}

	switch node.Kind {
	case yaml.DocumentNode:
		return findAndBlockTaskInDocument(node, taskID, reason)
	case yaml.MappingNode:
		return findAndBlockTaskInMapping(node, taskID, reason)
	case yaml.SequenceNode:
		return findAndBlockTaskInSequence(node, taskID, reason)
	}

	return blockResult{}
}

func findAndBlockTaskInDocument(node *yaml.Node, taskID, reason string) blockResult {
	for _, child := range node.Content {
		if result := findAndBlockTask(child, taskID, reason); result.found {
			return result
		}
	}
	return blockResult{}
}

func findAndBlockTaskInSequence(node *yaml.Node, taskID, reason string) blockResult {
	for _, child := range node.Content {
		if result := findAndBlockTask(child, taskID, reason); result.found {
			return result
		}
	}
	return blockResult{}
}

func findAndBlockTaskInMapping(node *yaml.Node, taskID, reason string) blockResult {
	// Check if this is a task node with matching ID
	var idNode, statusNode, reasonNode *yaml.Node
	var statusKeyIdx, reasonKeyIdx int = -1, -1

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
			statusKeyIdx = i
		case "blocked_reason":
			reasonNode = value
			reasonKeyIdx = i
		}
	}

	// If we found the task with matching ID and status field
	if idNode != nil && statusNode != nil {
		return updateTaskBlockFields(node, statusNode, reasonNode, statusKeyIdx, reasonKeyIdx, reason)
	}

	// Otherwise recurse into all values
	for i := 1; i < len(node.Content); i += 2 {
		if result := findAndBlockTask(node.Content[i], taskID, reason); result.found {
			return result
		}
	}

	return blockResult{}
}

// updateTaskBlockFields updates the status and blocked_reason fields on a task node
func updateTaskBlockFields(node *yaml.Node, statusNode, reasonNode *yaml.Node, statusKeyIdx, reasonKeyIdx int, reason string) blockResult {
	result := blockResult{
		found:          true,
		previousStatus: statusNode.Value,
	}

	// Update status to Blocked
	statusNode.Value = "Blocked"

	// Handle blocked_reason field
	if reasonNode != nil {
		result.hadReason = true
		result.previousReason = reasonNode.Value
		reasonNode.Value = reason
	} else {
		// Add blocked_reason field after status
		insertBlockedReason(node, statusKeyIdx, reason)
	}

	return result
}

// insertBlockedReason inserts a blocked_reason field after the status field
func insertBlockedReason(node *yaml.Node, statusKeyIdx int, reason string) {
	// Create new key and value nodes
	keyNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: "blocked_reason",
	}
	valueNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: reason,
	}

	// Insert after status key-value pair (at position statusKeyIdx+2)
	insertIdx := statusKeyIdx + 2
	if insertIdx > len(node.Content) {
		insertIdx = len(node.Content)
	}

	// Grow slice and insert
	node.Content = append(node.Content, nil, nil)
	copy(node.Content[insertIdx+2:], node.Content[insertIdx:])
	node.Content[insertIdx] = keyNode
	node.Content[insertIdx+1] = valueNode
}
