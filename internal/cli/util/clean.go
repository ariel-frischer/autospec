package util

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/ariel-frischer/autospec/internal/clean"
	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove autospec files from the project",
	Long: `Remove autospec-related files and directories from the current project.

This command removes:
  - .autospec/ directory (configuration, scripts, and state)
  - .claude/commands/autospec*.md files (slash commands)

By default, specs/ directory is PRESERVED. You will be prompted separately
if you also want to remove specs/ (default: No).

The command will prompt for confirmation before removing files.
Use --dry-run to preview what would be removed without making changes.
Use --yes to skip the confirmation prompt (specs/ will NOT be removed with --yes).
Use --keep-specs to skip the specs prompt and preserve specs/.
Use --remove-specs to skip the specs prompt and remove specs/.

Note: This does not remove user-level config (~/.config/autospec/) or
global state (~/.autospec/). Use 'rm -rf' manually if needed.`,
	Example: `  # Preview what would be removed
  autospec clean --dry-run

  # Remove autospec files (with confirmation, prompted about specs/)
  autospec clean

  # Remove autospec files without confirmation (specs/ preserved)
  autospec clean --yes

  # Remove everything including specs/ without any prompts
  autospec clean --yes --remove-specs

  # Remove autospec files, explicitly preserve specs/
  autospec clean --keep-specs`,
	RunE: runClean,
}

func init() {
	cleanCmd.GroupID = shared.GroupConfiguration
	cleanCmd.Flags().BoolP("dry-run", "n", false, "Show what would be removed without removing")
	cleanCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt (specs/ will be preserved)")
	cleanCmd.Flags().BoolP("keep-specs", "k", false, "Skip specs prompt and preserve specs/")
	cleanCmd.Flags().BoolP("remove-specs", "r", false, "Skip specs prompt and remove specs/")
	cleanCmd.MarkFlagsMutuallyExclusive("keep-specs", "remove-specs")
}

func runClean(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	yes, _ := cmd.Flags().GetBool("yes")
	keepSpecs, _ := cmd.Flags().GetBool("keep-specs")
	removeSpecs, _ := cmd.Flags().GetBool("remove-specs")

	out := cmd.OutOrStdout()

	// Find autospec files (keep specs by default)
	targets, err := clean.FindAutospecFiles(true)
	if err != nil {
		return fmt.Errorf("failed to find autospec files: %w", err)
	}

	// Check if specs/ directory exists for potential removal
	specsTarget, specsExists := clean.GetSpecsTarget()

	// Handle case when no files found
	if len(targets) == 0 && !specsExists {
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

	// Show specs/ status based on flags
	if specsExists {
		if removeSpecs {
			fmt.Fprintf(out, "  [dir] %s (%s)\n", specsTarget.Path, specsTarget.Description)
		} else if keepSpecs {
			fmt.Fprintln(out, "\n  (specs/ directory will be preserved)")
		} else {
			fmt.Fprintln(out, "\n  (specs/ directory will be preserved by default)")
		}
	}

	// Handle case when only specs/ exists but nothing else
	if len(targets) == 0 {
		fmt.Fprintln(out, "\nNo autospec files to remove (only specs/ exists).")
		if !dryRun && removeSpecs {
			// --remove-specs flag: remove without prompting
			if !yes {
				fmt.Fprintln(out)
				if !promptYesNo(cmd, "Remove specs/ directory?") {
					fmt.Fprintln(out, "Aborted.")
					return nil
				}
			}
			results := clean.RemoveFiles([]clean.CleanTarget{specsTarget})
			if results[0].Success {
				fmt.Fprintf(out, "✓ Removed: %s\n", specsTarget.Path)
				fmt.Fprintln(out, "\nSummary: 1 removed")
			} else {
				fmt.Fprintf(out, "✗ Failed: %s (%v)\n", specsTarget.Path, results[0].Error)
				return fmt.Errorf("failed to remove specs/")
			}
		} else if !dryRun && !keepSpecs && !yes {
			// Interactive mode: prompt user
			if promptYesNo(cmd, "Remove specs/ directory?") {
				results := clean.RemoveFiles([]clean.CleanTarget{specsTarget})
				if results[0].Success {
					fmt.Fprintf(out, "✓ Removed: %s\n", specsTarget.Path)
					fmt.Fprintln(out, "\nSummary: 1 removed")
				} else {
					fmt.Fprintf(out, "✗ Failed: %s (%v)\n", specsTarget.Path, results[0].Error)
					return fmt.Errorf("failed to remove specs/")
				}
			} else {
				fmt.Fprintln(out, "Aborted.")
			}
		}
		return nil
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

	// Handle specs/ removal based on flags
	if specsExists {
		shouldRemoveSpecs := false

		if removeSpecs {
			// --remove-specs flag: remove specs without prompting
			shouldRemoveSpecs = true
		} else if !keepSpecs && !yes {
			// Interactive mode (no flags): prompt user
			fmt.Fprintln(out)
			shouldRemoveSpecs = promptYesNo(cmd, "Also remove specs/ directory?")
		}
		// If --keep-specs or --yes without --remove-specs: don't remove specs

		if shouldRemoveSpecs {
			specsResults := clean.RemoveFiles([]clean.CleanTarget{specsTarget})
			if specsResults[0].Success {
				successCount++
				fmt.Fprintf(out, "✓ Removed: %s\n", specsTarget.Path)
			} else {
				failCount++
				fmt.Fprintf(out, "✗ Failed: %s (%v)\n", specsTarget.Path, specsResults[0].Error)
			}
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

// promptYesNo prompts the user for a yes/no answer
func promptYesNo(cmd *cobra.Command, question string) bool {
	fmt.Fprintf(cmd.OutOrStdout(), "%s [y/N]: ", question)

	reader := bufio.NewReader(cmd.InOrStdin())
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	return answer == "y" || answer == "yes"
}
