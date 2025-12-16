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

var clarifyCmd = &cobra.Command{
	Use:     "clarify [optional-prompt]",
	Aliases: []string{"cl"},
	Short:   "Refine the specification by asking clarification questions (cl)",
	Long: `Execute the /autospec.clarify command for the current specification.

The clarify command will:
- Auto-detect the current spec from git branch or most recent spec
- Identify underspecified areas in the spec
- Ask up to 5 highly targeted clarification questions
- Encode answers back into the spec

Prerequisites:
- spec.yaml must exist (run 'autospec specify' first)`,
	Example: `  # Run clarification with no additional guidance
  autospec clarify

  # Focus on specific areas
  autospec clarify "Focus on error handling scenarios"

  # Clarify specific flows
  autospec clarify "Clarify the authentication flow"`,
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
			notifHandler.OnCommandComplete("clarify", false, duration)
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

		// Check if constitution exists (required for clarify)
		constitutionCheck := workflow.CheckConstitutionExists()
		if !constitutionCheck.Exists {
			fmt.Fprint(os.Stderr, constitutionCheck.ErrorMessage)
			cmd.SilenceUsage = true
			return notifyAndReturn(NewExitError(ExitInvalidArguments))
		}

		// Auto-detect current spec and verify spec.yaml exists
		metadata, err := spec.DetectCurrentSpec(cfg.SpecsDir)
		if err != nil {
			cmd.SilenceUsage = true
			return notifyAndReturn(fmt.Errorf("failed to detect current spec: %w\n\nRun 'autospec specify' to create a new spec first", err))
		}

		// Validate spec.yaml exists (required for clarify stage)
		prereqResult := workflow.ValidateStagePrerequisites(workflow.StageClarify, metadata.Directory)
		if !prereqResult.Valid {
			fmt.Fprint(os.Stderr, prereqResult.ErrorMessage)
			cmd.SilenceUsage = true
			return notifyAndReturn(NewExitError(ExitInvalidArguments))
		}

		// Create workflow orchestrator
		orch := workflow.NewWorkflowOrchestrator(cfg)
		orch.Executor.NotificationHandler = notifHandler

		// Execute clarify stage
		specName := fmt.Sprintf("%s-%s", metadata.Number, metadata.Name)
		execErr := orch.ExecuteClarify(specName, prompt)

		// Calculate duration and send command completion notification
		duration := time.Since(startTime)
		success := execErr == nil
		notifHandler.OnCommandComplete("clarify", success, duration)

		if execErr != nil {
			return fmt.Errorf("clarify stage failed: %w", execErr)
		}

		return nil
	},
}

func init() {
	clarifyCmd.GroupID = GroupOptionalStages
	rootCmd.AddCommand(clarifyCmd)
}
