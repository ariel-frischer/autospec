package admin

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/uninstall"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Completely remove autospec from the system",
	Long: `Completely remove autospec from your system.

This command removes:
  - The autospec binary from its installed location
  - User configuration directory (~/.config/autospec/)
  - State directory (~/.autospec/)

The command will prompt for confirmation before removing files.
Use --dry-run to preview what would be removed without making changes.
Use --yes to skip the confirmation prompt.

Note: This does NOT remove project-level files (.autospec/ in your projects).
Use 'autospec clean' in each project directory to remove project-level files.

If the binary is installed in a system directory (e.g., /usr/local/bin),
you may need to run this command with elevated privileges (sudo).`,
	Example: `  # Preview what would be removed
  autospec uninstall --dry-run

  # Remove autospec (with confirmation)
  autospec uninstall

  # Remove autospec without confirmation
  autospec uninstall --yes

  # If binary is in system directory
  sudo autospec uninstall --yes`,
	RunE: RunUninstall,
}

func init() {
	uninstallCmd.GroupID = shared.GroupConfiguration
	uninstallCmd.Flags().BoolP("dry-run", "n", false, "Show what would be removed without removing")
	uninstallCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}

// RunUninstall is the main entry point for the uninstall command.
// Exported for testing.
func RunUninstall(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	yes, _ := cmd.Flags().GetBool("yes")
	out := cmd.OutOrStdout()

	targets, existingTargets, err := collectUninstallTargets()
	if err != nil {
		return fmt.Errorf("collecting uninstall targets: %w", err)
	}

	if len(existingTargets) == 0 {
		fmt.Fprintln(out, "No autospec files found to remove.")
		return nil
	}

	requiresSudo := displayUninstallTargets(out, targets, dryRun)

	if dryRun {
		return nil
	}

	if !confirmUninstall(cmd, out, requiresSudo, yes) {
		return nil
	}

	return executeUninstall(out, targets)
}

// collectUninstallTargets gets all targets and filters existing ones
func collectUninstallTargets() ([]uninstall.UninstallTarget, []uninstall.UninstallTarget, error) {
	targets, err := uninstall.GetUninstallTargets()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to detect uninstall targets: %w", err)
	}

	var existingTargets []uninstall.UninstallTarget
	for _, target := range targets {
		if target.Exists {
			existingTargets = append(existingTargets, target)
		}
	}

	return targets, existingTargets, nil
}

// displayUninstallTargets shows what will be removed and returns if sudo is needed
func displayUninstallTargets(out interface {
	Write(p []byte) (n int, err error)
}, targets []uninstall.UninstallTarget, dryRun bool) bool {
	if dryRun {
		fmt.Fprintln(out, "Would remove:")
	} else {
		fmt.Fprintln(out, "The following will be removed:")
	}

	var requiresSudo bool
	for _, target := range targets {
		status := "exists"
		if !target.Exists {
			status = "not found"
		}

		sudoHint := ""
		if target.RequiresSudo {
			sudoHint = " (requires sudo)"
			requiresSudo = true
		}

		fmt.Fprintf(out, "  [%s] %s - %s%s\n", target.Type, target.Path, status, sudoHint)
	}

	fmt.Fprintln(out, "\nNote: To clean up project-level files, run 'autospec clean' in each project directory.")
	return requiresSudo
}

// confirmUninstall handles sudo warning and user confirmation
func confirmUninstall(cmd *cobra.Command, out interface {
	Write(p []byte) (n int, err error)
}, requiresSudo, yes bool) bool {
	if requiresSudo {
		fmt.Fprintln(out, "\nWarning: Some files require elevated privileges to remove.")
		fmt.Fprintln(out, "You may need to re-run with: sudo autospec uninstall")
	}

	if !yes {
		fmt.Fprintln(out)
		if !promptYesNo(cmd, "Uninstall autospec?") {
			fmt.Fprintln(out, "Uninstall cancelled.")
			return false
		}
	}
	return true
}

// executeUninstall removes targets and displays results
func executeUninstall(out interface {
	Write(p []byte) (n int, err error)
}, targets []uninstall.UninstallTarget) error {
	fmt.Fprintln(out)
	results := uninstall.RemoveTargets(targets)

	successCount, failCount, skippedCount := displayRemovalResults(out, results)

	printUninstallSummary(out, successCount, failCount, skippedCount)

	if failCount > 0 {
		return fmt.Errorf("%d items could not be removed", failCount)
	}
	return nil
}

// displayRemovalResults shows individual removal outcomes and returns counts
func displayRemovalResults(out interface {
	Write(p []byte) (n int, err error)
}, results []uninstall.UninstallResult) (success, fail, skipped int) {
	for _, result := range results {
		if !result.Target.Exists {
			skipped++
			fmt.Fprintf(out, "- Skipped: %s (not found)\n", result.Target.Path)
		} else if result.Success {
			success++
			fmt.Fprintf(out, "✓ Removed: %s\n", result.Target.Path)
		} else {
			fail++
			fmt.Fprintf(out, "✗ Failed: %s (%v)\n", result.Target.Path, result.Error)
		}
	}
	return success, fail, skipped
}

// printUninstallSummary displays the final uninstall summary
func printUninstallSummary(out interface {
	Write(p []byte) (n int, err error)
}, successCount, failCount, skippedCount int) {
	fmt.Fprintf(out, "\nSummary: %d removed", successCount)
	if skippedCount > 0 {
		fmt.Fprintf(out, ", %d skipped", skippedCount)
	}
	if failCount > 0 {
		fmt.Fprintf(out, ", %d failed", failCount)
	}
	fmt.Fprintln(out)

	if successCount > 0 {
		fmt.Fprintln(out, "\nautospec has been uninstalled.")
	}
}

// promptYesNo prompts the user for a yes/no answer
func promptYesNo(cmd *cobra.Command, question string) bool {
	fmt.Fprintf(cmd.OutOrStdout(), "%s [y/N]: ", question)

	reader := bufio.NewReader(cmd.InOrStdin())
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	return answer == "y" || answer == "yes"
}
