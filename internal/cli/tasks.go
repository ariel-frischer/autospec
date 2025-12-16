package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
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

		// Create notification handler early so we can notify on any error
		notifHandler := notify.NewHandler(cfg.Notifications)
		startTime := time.Now()
		notifHandler.SetStartTime(startTime)

		// Helper to send error notification and return
		notifyAndReturn := func(err error) error {
			duration := time.Since(startTime)
			notifHandler.OnCommandComplete("tasks", false, duration)
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

		// Check if constitution exists (required for tasks)
		constitutionCheck := workflow.CheckConstitutionExists()
		if !constitutionCheck.Exists {
			fmt.Fprint(os.Stderr, constitutionCheck.ErrorMessage)
			cmd.SilenceUsage = true
			return notifyAndReturn(NewExitError(ExitInvalidArguments))
		}

		// Auto-detect spec directory for prerequisite validation
		metadata, err := spec.DetectCurrentSpec(cfg.SpecsDir)
		if err != nil {
			cmd.SilenceUsage = true
			return notifyAndReturn(fmt.Errorf("failed to detect current spec: %w\n\nRun 'autospec specify' to create a new spec first", err))
		}

		// Validate plan.yaml exists (required for tasks stage)
		prereqResult := workflow.ValidateStagePrerequisites(workflow.StageTasks, metadata.Directory)
		if !prereqResult.Valid {
			fmt.Fprint(os.Stderr, prereqResult.ErrorMessage)
			cmd.SilenceUsage = true
			return notifyAndReturn(NewExitError(ExitInvalidArguments))
		}

		// Create workflow orchestrator
		orch := workflow.NewWorkflowOrchestrator(cfg)
		orch.Executor.NotificationHandler = notifHandler

		// Execute tasks stage
		execErr := orch.ExecuteTasks("", prompt)

		// Calculate duration and send command completion notification
		duration := time.Since(startTime)
		success := execErr == nil
		notifHandler.OnCommandComplete("tasks", success, duration)

		if execErr != nil {
			return fmt.Errorf("tasks stage failed: %w", execErr)
		}

		return nil
	},
}

func init() {
	tasksCmd.GroupID = GroupCoreStages
	rootCmd.AddCommand(tasksCmd)
}
