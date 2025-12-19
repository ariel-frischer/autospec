package cli

import (
	"fmt"
	"strings"

	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/ariel-frischer/autospec/internal/history"
	"github.com/ariel-frischer/autospec/internal/lifecycle"
	"github.com/ariel-frischer/autospec/internal/notify"
	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/spf13/cobra"
)

var constitutionCmd = &cobra.Command{
	Use:     "constitution [optional-prompt]",
	Aliases: []string{"const"},
	Short:   "Create or update the project constitution (const)",
	Long: `Execute the /autospec.constitution command to create or update the project constitution.

The constitution command will:
- Create or update the project constitution in .autospec/constitution.yaml
- Define project principles and guidelines for development
- Can be run from any directory in the project

This command has no prerequisites - it can be run at any time.`,
	Example: `  # Generate or update constitution interactively
  autospec constitution

  # Focus on specific principles
  autospec constitution "Focus on security and performance"

  # Emphasize development practices
  autospec constitution "Emphasize test-driven development"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get optional prompt from args
		var prompt string
		if len(args) > 0 {
			prompt = strings.Join(args, " ")
		}

		// Get flags
		configPath, _ := cmd.Flags().GetString("config")
		skipPreflight, _ := cmd.Flags().GetBool("skip-preflight")
		maxRetries, _ := cmd.Flags().GetInt("max-retries")

		// Load configuration
		cfg, err := config.Load(configPath)
		if err != nil {
			cliErr := clierrors.ConfigParseError(configPath, err)
			clierrors.PrintError(cliErr)
			return cliErr
		}

		// Create notification handler and history logger
		notifHandler := notify.NewHandler(cfg.Notifications)
		historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)

		// Wrap command execution with lifecycle for timing, notification, and history
		// Note: constitution is project-level, no spec name
		return lifecycle.RunWithHistory(notifHandler, historyLogger, "constitution", "", func() error {
			// Override skip-preflight from flag if set
			if cmd.Flags().Changed("skip-preflight") {
				cfg.SkipPreflight = skipPreflight
			}

			// Override max-retries from flag if set
			if cmd.Flags().Changed("max-retries") {
				cfg.MaxRetries = maxRetries
			}

			// Create workflow orchestrator
			orch := workflow.NewWorkflowOrchestrator(cfg)
			orch.Executor.NotificationHandler = notifHandler

			// Execute constitution stage
			if err := orch.ExecuteConstitution(prompt); err != nil {
				return fmt.Errorf("constitution stage failed: %w", err)
			}

			return nil
		})
	},
}

func init() {
	constitutionCmd.GroupID = GroupCoreStages
	rootCmd.AddCommand(constitutionCmd)

	// Command-specific flags
	constitutionCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts (overrides config when set)")
}
