// Package config provides CLI commands for autospec configuration management.
// Includes: init, config, migrate, doctor
package config

import (
	"github.com/spf13/cobra"
)

// Register adds all configuration commands to the root command.
// This function is called from the root CLI package during initialization.
func Register(rootCmd *cobra.Command) {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(doctorCmd)
}
