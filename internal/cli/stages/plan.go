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

var planCmd = &cobra.Command{
	Use:     "plan [optional-prompt]",
	Aliases: []string{"p"},
	Short:   "Execute the planning stage for the current spec (p)",
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

		// Validate spec.yaml exists (required for plan stage)
		prereqResult := workflow.ValidateStagePrerequisites(workflow.StagePlan, metadata.Directory)
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
		return lifecycle.RunWithHistory(notifHandler, historyLogger, "plan", specName, func() error {
			// Create workflow orchestrator
			orch := workflow.NewWorkflowOrchestrator(cfg)
			orch.Executor.NotificationHandler = notifHandler

			// Execute plan stage
			if err := orch.ExecutePlan("", prompt); err != nil {
				return fmt.Errorf("plan stage failed: %w", err)
			}

			return nil
		})
	},
}

func init() {
	planCmd.GroupID = shared.GroupCoreStages

	// Command-specific flags
	planCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts (overrides config when set)")
}
