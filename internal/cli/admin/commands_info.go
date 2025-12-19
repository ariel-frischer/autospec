package admin

import (
	"fmt"

	"github.com/ariel-frischer/autospec/internal/commands"
	"github.com/spf13/cobra"
)

var commandsInfoCmd = &cobra.Command{
	Use:   "info [command-name]",
	Short: "Show information about command templates",
	Long: `Show information about installed and available command templates.

Without arguments, lists all available autospec commands.
With a command name, shows detailed information about that command.

Example:
  autospec commands info
  autospec commands info autospec.specify`,
	RunE: runCommandsInfo,
}

var infoTargetDir string

func init() {
	commandsCmd.AddCommand(commandsInfoCmd)
	commandsInfoCmd.Flags().StringVar(&infoTargetDir, "target", "", "Target directory (default: .claude/commands)")
}

func runCommandsInfo(cmd *cobra.Command, args []string) error {
	targetDir := infoTargetDir
	if targetDir == "" {
		targetDir = commands.GetDefaultCommandsDir()
	}

	if len(args) > 0 {
		return showCommandDetail(cmd, args[0])
	}

	return listAllCommands(cmd, targetDir)
}

func listAllCommands(cmd *cobra.Command, targetDir string) error {
	infos, err := commands.GetInstalledCommands(targetDir)
	if err != nil {
		return fmt.Errorf("failed to get command info: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Autospec Command Templates")
	fmt.Fprintln(cmd.OutOrStdout(), "===========================")
	fmt.Fprintln(cmd.OutOrStdout())

	for _, info := range infos {
		status := "not installed"
		if info.Version != "" {
			if info.IsOutdated {
				status = fmt.Sprintf("v%s (update available: v%s)", info.Version, info.EmbeddedVersion)
			} else {
				status = fmt.Sprintf("v%s (current)", info.Version)
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), "/%s\n", info.Name)
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", info.Description)
		fmt.Fprintf(cmd.OutOrStdout(), "  Status: %s\n", status)
		fmt.Fprintln(cmd.OutOrStdout())
	}

	return nil
}

func showCommandDetail(cmd *cobra.Command, name string) error {
	// Get embedded template info
	tpl, err := commands.GetTemplateInfo(name)
	if err != nil {
		return fmt.Errorf("command not found: %s", name)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Command: /%s\n", tpl.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "Description: %s\n", tpl.Description)
	fmt.Fprintf(cmd.OutOrStdout(), "Version: %s\n", tpl.Version)
	fmt.Fprintf(cmd.OutOrStdout(), "Size: %d bytes\n", len(tpl.Content))
	fmt.Fprintln(cmd.OutOrStdout())

	fmt.Fprintln(cmd.OutOrStdout(), "Usage:")
	fmt.Fprintf(cmd.OutOrStdout(), "  /%s \"<feature description>\"\n", tpl.Name)

	return nil
}
