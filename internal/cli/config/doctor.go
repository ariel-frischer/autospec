package config

import (
	"fmt"
	"os"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/cliagent"
	"github.com/ariel-frischer/autospec/internal/health"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:     "doctor",
	Aliases: []string{"doc"},
	Short:   "Run health checks for autospec dependencies (doc)",
	Long: `Run health checks to verify that the configured agent is installed and ready.

This command checks for:
  - The configured agent's CLI (agent_preset; Claude when unset)
  - Claude settings (Bash(autospec:*) permission), only when the agent is Claude
  - Installed CLI agents and their smoke-tested versions (informational)

Each check will display a checkmark if passed or an X with an error message if failed.`,
	Example: `  # Check all dependencies
  autospec doctor

  # Run before starting a new project
  autospec doctor && autospec init`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		agent, err := resolveDoctorAgent(cmd)
		if err != nil {
			return err
		}
		report := health.RunHealthChecks(agent)
		fmt.Print(health.FormatReport(report))

		// Exit with non-zero status if any checks failed
		if !report.Passed {
			os.Exit(1)
		}
		return nil
	},
}

// resolveDoctorAgent returns the agent selected by the effective configuration.
func resolveDoctorAgent(cmd *cobra.Command) (cliagent.Agent, error) {
	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := shared.LoadConfig(cmd, configPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	agent, err := cfg.GetAgent()
	if err != nil {
		return nil, fmt.Errorf("resolving configured agent: %w", err)
	}
	return agent, nil
}

func init() {
	doctorCmd.GroupID = shared.GroupConfiguration
}
