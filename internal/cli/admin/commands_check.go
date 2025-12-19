package admin

import (
	"fmt"

	"github.com/ariel-frischer/autospec/internal/commands"
	"github.com/spf13/cobra"
)

var commandsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check command template versions",
	Long: `Check if installed command templates are up to date.

Compares the versions of installed autospec.* commands against the embedded
versions in the binary. Lists any commands that need updating.

Example:
  autospec commands check
  autospec commands check --target ./custom/commands`,
	RunE: runCommandsCheck,
}

var checkTargetDir string

func init() {
	commandsCmd.AddCommand(commandsCheckCmd)
	commandsCheckCmd.Flags().StringVar(&checkTargetDir, "target", "", "Target directory (default: .claude/commands)")
}

func runCommandsCheck(cmd *cobra.Command, args []string) error {
	targetDir := checkTargetDir
	if targetDir == "" {
		targetDir = commands.GetDefaultCommandsDir()
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Checking command versions in %s...\n\n", targetDir)

	mismatches, err := commands.CheckVersions(targetDir)
	if err != nil {
		return fmt.Errorf("failed to check versions: %w", err)
	}

	if len(mismatches) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "All commands are up to date.")
		return nil
	}

	// Group by action
	var needsInstall, needsUpdate []commands.VersionMismatch
	for _, m := range mismatches {
		if m.Action == "install" {
			needsInstall = append(needsInstall, m)
		} else {
			needsUpdate = append(needsUpdate, m)
		}
	}

	if len(needsInstall) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Not installed:")
		for _, m := range needsInstall {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %s (available: %s)\n", m.CommandName, m.EmbeddedVersion)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	if len(needsUpdate) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Needs update:")
		for _, m := range needsUpdate {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %s (%s → %s)\n", m.CommandName, m.InstalledVersion, m.EmbeddedVersion)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Run 'autospec commands install' to update.")

	return nil
}
