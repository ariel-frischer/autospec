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
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/spf13/cobra"
)

var tasksCmd = &cobra.Command{
	Use:     "tasks [optional-prompt]",
	Aliases: []string{"t"},
	Short:   "Execute the task generation stage for the current spec (t)",
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
			cmd.SilenceUsage = true
			return shared.NewExitError(shared.ExitInvalidArguments)
		}

		// Auto-detect spec directory for prerequisite validation
		metadata, err := spec.DetectCurrentSpec(cfg.SpecsDir)
		if err != nil {
			cmd.SilenceUsage = true
			return fmt.Errorf("failed to detect current spec: %w\n\nRun 'autospec specify' to create a new spec first", err)
		}
		shared.PrintSpecInfo(metadata)

		// Validate plan.yaml exists (required for tasks stage)
		prereqResult := workflow.ValidateStagePrerequisites(workflow.StageTasks, metadata.Directory)
		if !prereqResult.Valid {
			fmt.Fprint(os.Stderr, prereqResult.ErrorMessage)
			cmd.SilenceUsage = true
			return shared.NewExitError(shared.ExitInvalidArguments)
		}

		// Create notification handler and history logger
		notifHandler := notify.NewHandler(cfg.Notifications)
		historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)
		specName := fmt.Sprintf("%s-%s", metadata.Number, metadata.Name)

		// Wrap command execution with lifecycle for timing, notification, and history
		return lifecycle.RunWithHistory(notifHandler, historyLogger, "tasks", specName, func() error {
			// Create workflow orchestrator
			orch := workflow.NewWorkflowOrchestrator(cfg)
			orch.Executor.NotificationHandler = notifHandler

			// Execute tasks stage
			if err := orch.ExecuteTasks("", prompt); err != nil {
				return fmt.Errorf("tasks stage failed: %w", err)
			}

			return nil
		})
	},
}

func init() {
	tasksCmd.GroupID = shared.GroupCoreStages

	// Command-specific flags
	tasksCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts (overrides config when set)")
}
