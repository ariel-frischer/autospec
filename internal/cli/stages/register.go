// Package stages provides CLI commands for autospec workflow stages.
// Includes: specify, plan, tasks, implement
package stages

import (
	"github.com/spf13/cobra"
)

// Register adds all stage commands to the root command.
// This function is called from the root CLI package during initialization.
func Register(rootCmd *cobra.Command) {
	rootCmd.AddCommand(specifyCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(tasksCmd)
	rootCmd.AddCommand(implementCmd)
}
