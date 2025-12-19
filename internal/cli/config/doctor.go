package config

import (
	"fmt"
	"os"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/health"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:     "doctor",
	Aliases: []string{"doc"},
	Short:   "Run health checks for autospec dependencies (doc)",
	Long: `Run health checks to verify that all required dependencies are installed and available.

This command checks for:
  - Claude CLI
  - Git
  - Claude settings (Bash(autospec:*) permission in .claude/settings.local.json)

Each check will display a checkmark if passed or an X with an error message if failed.`,
	Example: `  # Check all dependencies
  autospec doctor

  # Run before starting a new project
  autospec doctor && autospec init`,
	Run: func(cmd *cobra.Command, args []string) {
		// Run all health checks
		report := health.RunHealthChecks()

		// Format and display the report
		output := health.FormatReport(report)
		fmt.Print(output)

		// Exit with non-zero status if any checks failed
		if !report.Passed {
			os.Exit(1)
		}
	},
}

func init() {
	doctorCmd.GroupID = shared.GroupConfiguration
}
