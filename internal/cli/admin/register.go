// Package admin provides administrative CLI commands for autospec.
// Includes: commands, completion_install, uninstall
package admin

import (
	"github.com/spf13/cobra"
)

// Register adds all administrative commands to the root command.
// This function is called from the root CLI package during initialization.
func Register(rootCmd *cobra.Command) {
	// Store reference to root command for completion generation
	rootCmdRef = rootCmd

	// Disable Cobra's default completion command so we can add our own
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.AddCommand(commandsCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(uninstallCmd)
}
