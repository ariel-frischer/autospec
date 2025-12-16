package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:     "plan [optional-prompt]",
	Aliases: []string{"p"},
	Short:   "Execute the planning phase for the current spec (p)",
	Long: `Execute the /autospec.plan command for the current specification.

The plan command will:
- Auto-detect the current spec from git branch or most recent spec
- Execute the planning workflow
- Create plan.yaml with technical decisions and data models

You can optionally provide a prompt to guide the planning process.`,
	Example: `  # Run planning with no additional guidance
  autospec plan

  # Run planning with focus on security
  autospec plan "Focus on security best practices"

  # Run planning with performance considerations
  autospec plan "Optimize for low-latency API responses"`,
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

		// Override skip-preflight from flag if set
		if cmd.Flags().Changed("skip-preflight") {
			cfg.SkipPreflight = skipPreflight
		}

		// Override max-retries from flag if set
		if cmd.Flags().Changed("max-retries") {
			cfg.MaxRetries = maxRetries
		}

		// Check if constitution exists (required for plan)
		constitutionCheck := workflow.CheckConstitutionExists()
		if !constitutionCheck.Exists {
			fmt.Fprint(os.Stderr, constitutionCheck.ErrorMessage)
			return fmt.Errorf("constitution required")
		}

		// Create workflow orchestrator
		orch := workflow.NewWorkflowOrchestrator(cfg)

		// Execute plan phase
		if err := orch.ExecutePlan("", prompt); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(planCmd)
}
