package cli

import (
	"fmt"
	"os"

	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/spf13/cobra"
)

var prepCmd = &cobra.Command{
	Use:   "prep <feature-description>",
	Short: "Prepare for implementation: specify → plan → tasks",
	Long: `Prepare for implementation by running specify, plan, and tasks phases.

This command will:
1. Run pre-flight checks (unless --skip-preflight)
2. Execute /autospec.specify with the feature description
3. Validate spec.yaml exists
4. Execute /autospec.plan
5. Validate plan.yaml exists
6. Execute /autospec.tasks
7. Validate tasks.yaml exists

Each phase is validated and will retry up to max_retries times if validation fails.

This is useful when you want to review the generated artifacts before implementation.`,
	Example: `  # Prepare spec, plan, and tasks for review before implementation
  autospec prep "Add user authentication feature"

  # Prepare artifacts for review
  autospec prep "Refactor database layer"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		featureDescription := args[0]

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

		// Override skip-preflight from flag if set
		if cmd.Flags().Changed("skip-preflight") {
			cfg.SkipPreflight = skipPreflight
		}

		// Override max-retries from flag if set
		if cmd.Flags().Changed("max-retries") {
			cfg.MaxRetries = maxRetries
		}

		// Check if constitution exists (required for all workflow phases)
		constitutionCheck := workflow.CheckConstitutionExists()
		if !constitutionCheck.Exists {
			fmt.Fprint(os.Stderr, constitutionCheck.ErrorMessage)
			return fmt.Errorf("constitution required")
		}

		// Create workflow orchestrator
		orchestrator := workflow.NewWorkflowOrchestrator(cfg)

		// Run complete workflow (specify → plan → tasks, no implementation)
		if err := orchestrator.RunCompleteWorkflow(featureDescription); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(prepCmd)

	// Command-specific flags
	prepCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts (0 = use config)")
}
