package cli

import (
	"fmt"

	"github.com/ariel-frischer/autospec/internal/clean"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove autospec files from the project",
	Long: `Remove autospec-related files and directories from the current project.

This command removes:
  - .autospec/ directory (configuration, scripts, and state)
  - .claude/commands/autospec*.md files (slash commands)
  - specs/ directory (feature specifications, unless --keep-specs is used)

The command will prompt for confirmation before removing files.
Use --dry-run to preview what would be removed without making changes.
Use --yes to skip the confirmation prompt.

Note: This does not remove user-level config (~/.config/autospec/) or
global state (~/.autospec/). Use 'rm -rf' manually if needed.`,
	Example: `  # Preview what would be removed
  autospec clean --dry-run

  # Remove all autospec files (with confirmation)
  autospec clean

  # Remove all autospec files without confirmation
  autospec clean --yes

  # Remove autospec files but keep specs/ directory
  autospec clean --keep-specs

  # Combine flags
  autospec clean --keep-specs --yes`,
	RunE: runClean,
}

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().BoolP("dry-run", "n", false, "Show what would be removed without removing")
	cleanCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
	cleanCmd.Flags().BoolP("keep-specs", "k", false, "Preserve specs/ directory")
}

func runClean(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	yes, _ := cmd.Flags().GetBool("yes")
	keepSpecs, _ := cmd.Flags().GetBool("keep-specs")

	out := cmd.OutOrStdout()

	// Find autospec files
	targets, err := clean.FindAutospecFiles(keepSpecs)
	if err != nil {
		return fmt.Errorf("failed to find autospec files: %w", err)
	}

	// Handle case when no files found
	if len(targets) == 0 {
		fmt.Fprintln(out, "No autospec files found.")
		return nil
	}

	// Display files to be removed
	if dryRun {
		fmt.Fprintln(out, "Would remove:")
	} else {
		fmt.Fprintln(out, "Files to be removed:")
	}

	for _, target := range targets {
		typeStr := "file"
		if target.Type == clean.TypeDirectory {
			typeStr = "dir"
		}
		fmt.Fprintf(out, "  [%s] %s (%s)\n", typeStr, target.Path, target.Description)
	}

	if keepSpecs {
		fmt.Fprintln(out, "\n  (specs/ directory will be preserved)")
	}

	// In dry-run mode, exit after displaying
	if dryRun {
		return nil
	}

	// Prompt for confirmation unless --yes is set
	if !yes {
		fmt.Fprintln(out)
		if !promptYesNo(cmd, "Remove these files?") {
			fmt.Fprintln(out, "Aborted.")
			return nil
		}
	}

	// Remove files
	results := clean.RemoveFiles(targets)

	// Display results
	var successCount, failCount int
	for _, result := range results {
		if result.Success {
			successCount++
			fmt.Fprintf(out, "✓ Removed: %s\n", result.Target.Path)
		} else {
			failCount++
			fmt.Fprintf(out, "✗ Failed: %s (%v)\n", result.Target.Path, result.Error)
		}
	}

	// Summary
	fmt.Fprintf(out, "\nSummary: %d removed", successCount)
	if failCount > 0 {
		fmt.Fprintf(out, ", %d failed", failCount)
	}
	fmt.Fprintln(out)

	if failCount > 0 {
		return fmt.Errorf("%d files could not be removed", failCount)
	}

	return nil
}
