package cli

import (
	"fmt"
	"os"

	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/ariel-frischer/autospec/internal/history"
	"github.com/ariel-frischer/autospec/internal/lifecycle"
	"github.com/ariel-frischer/autospec/internal/notify"
	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/spf13/cobra"
)

var allCmd = &cobra.Command{
	Use:   "all <feature-description>",
	Short: "Run complete specify -> plan -> tasks -> implement workflow",
	Long: `Run the complete SpecKit workflow including implementation with automatic validation and retry.

This command will:
1. Run pre-flight checks (unless --skip-preflight)
2. Execute /autospec.specify with the feature description
3. Validate spec.yaml exists
4. Execute /autospec.plan
5. Validate plan.yaml exists
6. Execute /autospec.tasks
7. Validate tasks.yaml exists
8. Execute /autospec.implement
9. Validate all tasks are completed

Each stage is validated and will retry up to max_retries times if validation fails.
This is equivalent to running 'autospec run -a <feature-description>'.`,
	Example: `  # Run complete workflow for a new feature
  autospec all "Add user authentication feature"

  # Resume interrupted implementation
  autospec all "Add user auth" --resume

  # Skip preflight checks for faster execution
  autospec all "Add API endpoints" --skip-preflight`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		featureDescription := args[0]

		// Get flags
		configPath, _ := cmd.Flags().GetString("config")
		skipPreflight, _ := cmd.Flags().GetBool("skip-preflight")
		maxRetries, _ := cmd.Flags().GetInt("max-retries")
		resume, _ := cmd.Flags().GetBool("resume")
		debug, _ := cmd.Flags().GetBool("debug")

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
		// Note: spec name is empty for all since we're creating a new spec
		return lifecycle.RunWithHistory(notifHandler, historyLogger, "all", "", func() error {
			// Override skip-preflight from flag if set
			if cmd.Flags().Changed("skip-preflight") {
				cfg.SkipPreflight = skipPreflight
			}

			// Override max-retries from flag if set
			if cmd.Flags().Changed("max-retries") {
				cfg.MaxRetries = maxRetries
			}

			// Check if constitution exists (required for all workflow stages)
			constitutionCheck := workflow.CheckConstitutionExists()
			if !constitutionCheck.Exists {
				fmt.Fprint(os.Stderr, constitutionCheck.ErrorMessage)
				return fmt.Errorf("constitution required")
			}

			// Create workflow orchestrator
			orchestrator := workflow.NewWorkflowOrchestrator(cfg)
			orchestrator.Debug = debug
			orchestrator.Executor.Debug = debug
			orchestrator.Executor.NotificationHandler = notifHandler

			if debug {
				fmt.Println("[DEBUG] Debug mode enabled")
				fmt.Printf("[DEBUG] Config: %+v\n", cfg)
			}

			// Run full workflow
			if err := orchestrator.RunFullWorkflow(featureDescription, resume); err != nil {
				return fmt.Errorf("full workflow failed: %w", err)
			}

			return nil
		})
	},
}

func init() {
	allCmd.GroupID = GroupWorkflows
	rootCmd.AddCommand(allCmd)

	allCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts (overrides config when set)")
	allCmd.Flags().Bool("resume", false, "Resume implementation from where it left off")
}
