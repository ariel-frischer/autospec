package admin

import (
	"fmt"

	"github.com/ariel-frischer/autospec/internal/commands"
	"github.com/spf13/cobra"
)

var commandsInstallCmd = &cobra.Command{
	Use:        "install",
	Short:      "Install autospec command templates (DEPRECATED: use 'autospec init')",
	Deprecated: "use 'autospec init' instead, which handles commands and config in one step",
	Long: `Install autospec command templates.

DEPRECATED: Use 'autospec init' instead, which handles commands and config in one step.

This installs:
  - Command templates (autospec.specify, autospec.plan, etc.) to .claude/commands/

Existing autospec files will be overwritten. Other files are preserved.

Example:
  autospec init                                           # Recommended
  autospec commands install                               # Deprecated
  autospec commands install --target ./custom/commands`,
	RunE: runCommandsInstall,
}

var installTargetDir string

func init() {
	commandsCmd.AddCommand(commandsInstallCmd)
	commandsInstallCmd.Flags().StringVar(&installTargetDir, "target", "", "Target directory for commands (default: .claude/commands)")
}

func runCommandsInstall(cmd *cobra.Command, args []string) error {
	// Install command templates
	targetDir := installTargetDir
	if targetDir == "" {
		targetDir = commands.GetDefaultCommandsDir()
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Installing autospec commands to %s...\n", targetDir)

	results, err := commands.InstallTemplates(targetDir)
	if err != nil {
		return fmt.Errorf("failed to install templates: %w", err)
	}

	cmdInstalledCount := 0
	cmdUpdatedCount := 0

	for _, result := range results {
		switch result.Action {
		case "installed":
			cmdInstalledCount++
			fmt.Fprintf(cmd.OutOrStdout(), "  + %s (installed)\n", result.CommandName)
		case "updated":
			cmdUpdatedCount++
			fmt.Fprintf(cmd.OutOrStdout(), "  ~ %s (updated)\n", result.CommandName)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nDone: %d installed, %d updated\n", cmdInstalledCount, cmdUpdatedCount)
	fmt.Fprintf(cmd.OutOrStdout(), "\nCommands are now available as /autospec.* in Claude Code\n")

	return nil
}
