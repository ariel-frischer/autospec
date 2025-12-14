package cli

import (
	"fmt"

	"github.com/anthropics/auto-claude-speckit/internal/commands"
	"github.com/spf13/cobra"
)

var commandsInstallCmd = &cobra.Command{
	Use:        "install",
	Short:      "Install autospec command templates and scripts (DEPRECATED: use 'autospec init')",
	Deprecated: "use 'autospec init' instead, which handles commands, scripts, and config in one step",
	Long: `Install autospec command templates and helper scripts.

DEPRECATED: Use 'autospec init' instead, which handles commands, scripts, and config in one step.

This installs:
  - Command templates (autospec.specify, autospec.plan, etc.) to .claude/commands/
  - Helper scripts (common.sh, check-prerequisites.sh, etc.) to .autospec/scripts/

Existing autospec files will be overwritten. Other files are preserved.

Example:
  autospec init                                           # Recommended
  autospec commands install                               # Deprecated
  autospec commands install --target ./custom/commands
  autospec commands install --scripts-target ./custom/scripts`,
	RunE: runCommandsInstall,
}

var installTargetDir string
var installScriptsDir string

func init() {
	commandsCmd.AddCommand(commandsInstallCmd)
	commandsInstallCmd.Flags().StringVar(&installTargetDir, "target", "", "Target directory for commands (default: .claude/commands)")
	commandsInstallCmd.Flags().StringVar(&installScriptsDir, "scripts-target", "", "Target directory for scripts (default: .autospec/scripts)")
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

	// Install helper scripts
	scriptsDir := installScriptsDir
	if scriptsDir == "" {
		scriptsDir = commands.GetDefaultScriptsDir()
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nInstalling helper scripts to %s...\n", scriptsDir)

	scriptResults, err := commands.InstallScripts(scriptsDir)
	if err != nil {
		return fmt.Errorf("failed to install scripts: %w", err)
	}

	scriptInstalledCount := 0
	scriptUpdatedCount := 0

	for _, result := range scriptResults {
		switch result.Action {
		case "installed":
			scriptInstalledCount++
			fmt.Fprintf(cmd.OutOrStdout(), "  + %s (installed)\n", result.ScriptName)
		case "updated":
			scriptUpdatedCount++
			fmt.Fprintf(cmd.OutOrStdout(), "  ~ %s (updated)\n", result.ScriptName)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nDone:\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  Commands: %d installed, %d updated\n", cmdInstalledCount, cmdUpdatedCount)
	fmt.Fprintf(cmd.OutOrStdout(), "  Scripts:  %d installed, %d updated\n", scriptInstalledCount, scriptUpdatedCount)
	fmt.Fprintf(cmd.OutOrStdout(), "\nCommands are now available as /autospec.* in Claude Code\n")
	fmt.Fprintf(cmd.OutOrStdout(), "Scripts are available at %s/\n", scriptsDir)

	return nil
}
