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

var tasksCmd = &cobra.Command{
	Use:     "tasks [optional-prompt]",
	Aliases: []string{"t"},
	Short:   "Execute the task generation phase for the current spec (t)",
	Long: `Execute the /autospec.tasks command for the current specification.

The tasks command will:
- Auto-detect the current spec from git branch or most recent spec
- Execute the task generation workflow
- Create tasks.yaml with actionable, dependency-ordered tasks

You can optionally provide a prompt to guide the task generation.`,
	Example: `  # Generate tasks with default granularity
  autospec tasks

  # Generate fine-grained tasks for careful review
  autospec tasks "Break into small incremental steps"

  # Generate tasks with testing focus
  autospec tasks "Prioritize testing tasks first"`,
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

		// Check if constitution exists (required for tasks)
		constitutionCheck := workflow.CheckConstitutionExists()
		if !constitutionCheck.Exists {
			fmt.Fprint(os.Stderr, constitutionCheck.ErrorMessage)
			return fmt.Errorf("constitution required")
		}

		// Create workflow orchestrator
		orch := workflow.NewWorkflowOrchestrator(cfg)

		// Execute tasks phase
		if err := orch.ExecuteTasks("", prompt); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(tasksCmd)
}
