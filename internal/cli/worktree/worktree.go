// Package worktree provides CLI commands for managing git worktrees with autospec.
package worktree

import (
	"github.com/spf13/cobra"
)

// WorktreeCmd is the parent command for all worktree operations.
var WorktreeCmd = &cobra.Command{
	Use:   "worktree",
	Short: "Manage git worktrees with project-aware setup",
	Long: `Manage git worktrees with automatic project configuration.

The worktree command helps create and manage git worktrees with automatic
copying of non-tracked directories (.autospec/, .claude/) and execution
of project-specific setup scripts.

This is useful when running multiple autospec workflows in parallel across
different feature branches.`,
}

func init() {
	// Subcommands are added in their respective files
	WorktreeCmd.AddCommand(createCmd)
	WorktreeCmd.AddCommand(listCmd)
	WorktreeCmd.AddCommand(removeCmd)
	WorktreeCmd.AddCommand(setupCmd)
	WorktreeCmd.AddCommand(pruneCmd)
}
