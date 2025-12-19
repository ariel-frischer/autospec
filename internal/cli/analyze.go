package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/ariel-frischer/autospec/internal/history"
	"github.com/ariel-frischer/autospec/internal/lifecycle"
	"github.com/ariel-frischer/autospec/internal/notify"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:     "analyze [optional-prompt]",
	Aliases: []string{"az"},
	Short:   "Perform cross-artifact consistency and quality analysis (az)",
	Long: `Execute the /autospec.analyze command for the current specification.

The analyze command will:
- Auto-detect the current spec from git branch or most recent spec
- Perform non-destructive cross-artifact consistency analysis
- Check quality across spec.yaml, plan.yaml, and tasks.yaml
- Report findings and recommendations

Prerequisites:
- spec.yaml must exist (run 'autospec specify' first)
- plan.yaml must exist (run 'autospec plan' first)
- tasks.yaml must exist (run 'autospec tasks' first)`,
	Example: `  # Run analysis with default checks
  autospec analyze

  # Focus on security implications
  autospec analyze "Focus on security implications"

  # Verify API contracts
  autospec analyze "Verify API contracts"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get optional prompt from args
		var prompt string
		if len(args) > 0 {
			prompt = strings.Join(args, " ")
		}

		// Get flags
		configPath, _ := cmd.Flags().GetString("config")
		skipPreflight, _ := cmd.Flags().GetBool("skip-preflight")

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

		// Check if constitution exists (required for analyze)
		constitutionCheck := workflow.CheckConstitutionExists()
		if !constitutionCheck.Exists {
			fmt.Fprint(os.Stderr, constitutionCheck.ErrorMessage)
			cmd.SilenceUsage = true
			return NewExitError(ExitInvalidArguments)
		}

		// Auto-detect current spec and verify all required artifacts exist
		metadata, err := spec.DetectCurrentSpec(cfg.SpecsDir)
		if err != nil {
			cmd.SilenceUsage = true
			return fmt.Errorf("failed to detect current spec: %w\n\nRun 'autospec specify' to create a new spec first", err)
		}
		PrintSpecInfo(metadata)

		// Validate all required artifacts exist (spec.yaml, plan.yaml, tasks.yaml)
		prereqResult := workflow.ValidateStagePrerequisites(workflow.StageAnalyze, metadata.Directory)
		if !prereqResult.Valid {
			fmt.Fprint(os.Stderr, prereqResult.ErrorMessage)
			cmd.SilenceUsage = true
			return NewExitError(ExitInvalidArguments)
		}

		// Create notification handler and history logger
		notifHandler := notify.NewHandler(cfg.Notifications)
		historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)
		specName := fmt.Sprintf("%s-%s", metadata.Number, metadata.Name)

		// Wrap command execution with lifecycle for timing, notification, and history
		return lifecycle.RunWithHistory(notifHandler, historyLogger, "analyze", specName, func() error {
			// Create workflow orchestrator
			orch := workflow.NewWorkflowOrchestrator(cfg)
			orch.Executor.NotificationHandler = notifHandler

			// Execute analyze stage
			if err := orch.ExecuteAnalyze(specName, prompt); err != nil {
				return fmt.Errorf("analyze stage failed: %w", err)
			}

			return nil
		})
	},
}

func init() {
	analyzeCmd.GroupID = GroupOptionalStages
	rootCmd.AddCommand(analyzeCmd)
	// Note: No --max-retries flag - analyze doesn't produce artifacts that need validation/retry
}
