package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
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

		// Create notification handler early so we can notify on any error
		notifHandler := notify.NewHandler(cfg.Notifications)
		startTime := time.Now()
		notifHandler.SetStartTime(startTime)

		// Helper to send error notification and return
		notifyAndReturn := func(err error) error {
			duration := time.Since(startTime)
			notifHandler.OnCommandComplete("specify", false, duration)
			return err
		}

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
			return notifyAndReturn(NewExitError(ExitInvalidArguments))
		}

		// Create workflow orchestrator
		orch := workflow.NewWorkflowOrchestrator(cfg)
		orch.Executor.NotificationHandler = notifHandler

		// Execute specify stage
		specName, execErr := orch.ExecuteSpecify(featureDescription)

		// Calculate duration and send command completion notification
		duration := time.Since(startTime)
		success := execErr == nil
		notifHandler.OnCommandComplete("specify", success, duration)

		if execErr != nil {
			return fmt.Errorf("specify stage failed: %w", execErr)
		}

		fmt.Printf("\nSpec created: %s\n", specName)
		return nil
	},
}

func init() {
	specifyCmd.GroupID = GroupCoreStages
	rootCmd.AddCommand(specifyCmd)

	// Command-specific flags
	specifyCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts (0 = use config)")
}
