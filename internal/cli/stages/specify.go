package stages

import (
	"fmt"
	"os"
	"strings"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/ariel-frischer/autospec/internal/history"
	"github.com/ariel-frischer/autospec/internal/lifecycle"
	"github.com/ariel-frischer/autospec/internal/notify"
	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/spf13/cobra"
)

var specifyCmd = &cobra.Command{
	Use:     "specify <feature-description>",
	Aliases: []string{"spec", "s"},
	Short:   "Execute the specification stage for a new feature (spec, s)",
	Long: `Execute the /autospec.specify command to create a new feature specification.

The specify command will:
- Create a new spec directory with a spec.yaml file
- Generate the specification based on your feature description
- Output the spec name for use in subsequent commands

The feature description should be a clear, concise description of what you want to build.`,
	Example: `  # Create a new feature specification
  autospec specify "Add user authentication feature"

  # Complex feature with multiple requirements
  autospec specify "Implement dark mode with system preference detection"

  # Feature with quotes in the description
  autospec specify 'Add "remember me" checkbox to login form'`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			cliErr := clierrors.MissingFeatureDescription()
			clierrors.PrintError(cliErr)
			return cliErr
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Join all args as the feature description
		featureDescription := strings.Join(args, " ")

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
		// Note: spec name is empty for specify since we're creating a new spec
		return lifecycle.RunWithHistory(notifHandler, historyLogger, "specify", "", func() error {
			// Override skip-preflight from flag if set
			if cmd.Flags().Changed("skip-preflight") {
				cfg.SkipPreflight = skipPreflight
			}

			// Override max-retries from flag if set
			if cmd.Flags().Changed("max-retries") {
				cfg.MaxRetries = maxRetries
			}

			// Check if constitution exists (required for specify)
			constitutionCheck := workflow.CheckConstitutionExists()
			if !constitutionCheck.Exists {
				fmt.Fprint(os.Stderr, constitutionCheck.ErrorMessage)
				return shared.NewExitError(shared.ExitInvalidArguments)
			}

			// Create workflow orchestrator
			orch := workflow.NewWorkflowOrchestrator(cfg)
			orch.Executor.NotificationHandler = notifHandler

			// Execute specify stage
			specName, execErr := orch.ExecuteSpecify(featureDescription)
			if execErr != nil {
				return fmt.Errorf("specify stage failed: %w", execErr)
			}

			fmt.Printf("\nSpec created: %s\n", specName)
			return nil
		})
	},
}

func init() {
	specifyCmd.GroupID = shared.GroupCoreStages

	// Command-specific flags
	specifyCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts (overrides config when set)")
}
