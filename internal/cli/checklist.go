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

var checklistCmd = &cobra.Command{
	Use:     "checklist [optional-prompt]",
	Aliases: []string{"chk"},
	Short:   "Generate a custom checklist for the current feature (chk)",
	Long: `Execute the /autospec.checklist command for the current specification.

The checklist command will:
- Auto-detect the current spec from git branch or most recent spec
- Generate a custom checklist based on user requirements
- Create checklist files in the checklists/ directory

Prerequisites:
- spec.yaml must exist (run 'autospec specify' first)`,
	Example: `  # Generate checklist with default criteria
  autospec checklist

  # Focus on security requirements
  autospec checklist "Focus on security requirements"

  # Include accessibility checks
  autospec checklist "Include accessibility checks"`,
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

		// Check if constitution exists (required for checklist)
		constitutionCheck := workflow.CheckConstitutionExists()
		if !constitutionCheck.Exists {
			fmt.Fprint(os.Stderr, constitutionCheck.ErrorMessage)
			cmd.SilenceUsage = true
			return NewExitError(ExitInvalidArguments)
		}

		// Auto-detect current spec and verify spec.yaml exists
		metadata, err := spec.DetectCurrentSpec(cfg.SpecsDir)
		if err != nil {
			cmd.SilenceUsage = true
			return fmt.Errorf("failed to detect current spec: %w\n\nRun 'autospec specify' to create a new spec first", err)
		}

		// Validate spec.yaml exists (required for checklist stage)
		prereqResult := workflow.ValidateStagePrerequisites(workflow.StageChecklist, metadata.Directory)
		if !prereqResult.Valid {
			fmt.Fprint(os.Stderr, prereqResult.ErrorMessage)
			cmd.SilenceUsage = true
			return NewExitError(ExitInvalidArguments)
		}

		// Create workflow orchestrator
		orch := workflow.NewWorkflowOrchestrator(cfg)

		// Create notification handler and attach to executor
		notifHandler := notify.NewHandler(cfg.Notifications)
		orch.Executor.NotificationHandler = notifHandler

		// Track command start time
		startTime := time.Now()
		notifHandler.SetStartTime(startTime)

		// Execute checklist stage
		specName := fmt.Sprintf("%s-%s", metadata.Number, metadata.Name)
		execErr := orch.ExecuteChecklist(specName, prompt)

		// Calculate duration and send command completion notification
		duration := time.Since(startTime)
		success := execErr == nil
		notifHandler.OnCommandComplete("checklist", success, duration)

		if execErr != nil {
			return fmt.Errorf("checklist stage failed: %w", execErr)
		}

		return nil
	},
}

func init() {
	checklistCmd.GroupID = GroupOptionalStages
	rootCmd.AddCommand(checklistCmd)
}
