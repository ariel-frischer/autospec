// Package util provides utility CLI commands for autospec.
// Includes: status, history, version, clean
package util

import (
	"github.com/spf13/cobra"
)

// Register adds all utility commands to the root command.
// This function is called from the root CLI package during initialization.
func Register(rootCmd *cobra.Command) {
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(sauceCmd)
	rootCmd.AddCommand(cleanCmd)
}
