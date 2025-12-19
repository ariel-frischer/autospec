// autospec - Spec-Driven Development Automation
// Author: Ariel Frischer
// Source: https://github.com/ariel-frischer/autospec

// Package cli provides Cobra-based CLI commands for the autospec workflow automation tool.
// It defines all user-facing commands including workflow orchestration (run, all, prep),
// individual stages (specify, plan, tasks, implement), configuration management (init, config),
// and utility commands (status, doctor, clean, uninstall).
package cli

import (
	"github.com/ariel-frischer/autospec/internal/cli/admin"
	"github.com/ariel-frischer/autospec/internal/cli/config"
	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/cli/stages"
	"github.com/ariel-frischer/autospec/internal/cli/util"
	"github.com/spf13/cobra"
)

// Command group IDs for organizing help output (re-exported from shared)
const (
	GroupGettingStarted = shared.GroupGettingStarted
	GroupWorkflows      = shared.GroupWorkflows
	GroupCoreStages     = shared.GroupCoreStages
	GroupOptionalStages = shared.GroupOptionalStages
	GroupConfiguration  = shared.GroupConfiguration
	GroupInternal       = shared.GroupInternal
)

var rootCmd = &cobra.Command{
	Use:   "autospec",
	Short: "autospec workflow automation",
	Long: `autospec workflow automation

Automated spec-driven development. Define features in YAML, generate implementation
plans and tasks, then execute with Claude Code.

Source: https://github.com/ariel-frischer/autospec`,
	Example: `  # Check current feature status
  autospec status

  # Complete workflow: specify -> plan -> tasks -> implement
  autospec all "Add user authentication feature"

  # Prepare for implementation (no code changes)
  autospec prep "Add dark mode support"

  # Flexible stage selection
  autospec run -spti "Add user auth"   # All core stages
  autospec run -pi                     # Plan + implement on current spec

  # Individual stage commands
  autospec specify "Add search feature"
  autospec plan
  autospec tasks
  autospec implement`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Define command groups in display order
	rootCmd.AddGroup(&cobra.Group{ID: GroupGettingStarted, Title: "Getting Started:"})
	rootCmd.AddGroup(&cobra.Group{ID: GroupWorkflows, Title: "Workflows:"})
	rootCmd.AddGroup(&cobra.Group{ID: GroupCoreStages, Title: "Core Stages:"})
	rootCmd.AddGroup(&cobra.Group{ID: GroupOptionalStages, Title: "Optional Stages:"})
	rootCmd.AddGroup(&cobra.Group{ID: GroupConfiguration, Title: "Configuration:"})
	rootCmd.AddGroup(&cobra.Group{ID: GroupInternal, Title: "Internal Commands:"})

	// Assign built-in help and completion to configuration group
	rootCmd.SetHelpCommandGroupID(GroupConfiguration)
	rootCmd.SetCompletionCommandGroupID(GroupConfiguration)

	// Global flags
	rootCmd.PersistentFlags().StringP("config", "c", ".autospec/config.yml", "Path to config file")
	rootCmd.PersistentFlags().String("specs-dir", "./specs", "Directory containing feature specs")
	rootCmd.PersistentFlags().Bool("skip-preflight", false, "Skip pre-flight validation checks")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")

	// Register commands from subpackages
	stages.Register(rootCmd)
	config.Register(rootCmd)
	util.Register(rootCmd)
	admin.Register(rootCmd)
}
