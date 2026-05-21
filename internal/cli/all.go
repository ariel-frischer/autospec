package cli

import (
	"fmt"
	"os"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/ariel-frischer/autospec/internal/history"
	"github.com/ariel-frischer/autospec/internal/lifecycle"
	"github.com/ariel-frischer/autospec/internal/notify"
	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/spf13/cobra"
)

var allCmd = &cobra.Command{
	Use:   "all <feature-description>",
	Short: "Run complete specify -> plan -> tasks -> implement workflow",
	Long: `Run the complete SpecKit workflow including implementation with automatic validation and retry.

This command will:
1. Run pre-flight checks (unless --skip-preflight)
2. Execute /autospec.specify with the feature description
3. Validate spec.yaml exists
4. Execute /autospec.plan
5. Validate plan.yaml exists
6. Execute /autospec.tasks
7. Validate tasks.yaml exists
8. Execute /autospec.implement
9. Validate all tasks are completed

Each stage is validated and will retry up to max_retries times if validation fails.
This is equivalent to running 'autospec run -a <feature-description>'.`,
	Example: `  # Run complete workflow for a new feature
  autospec all "Add user authentication feature"

  # Resume interrupted implementation
  autospec all "Add user auth" --resume

  # Skip preflight checks for faster execution
  autospec all "Add API endpoints" --skip-preflight`,
	Args: validatePrepArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true // Don't show help for execution errors
		featureDescription := ""
		if len(args) > 0 {
			featureDescription = args[0]
		}

		// Get flags
		configPath, _ := cmd.Flags().GetString("config")
		skipPreflight, _ := cmd.Flags().GetBool("skip-preflight")
		maxRetries, _ := cmd.Flags().GetInt("max-retries")
		resume, _ := cmd.Flags().GetBool("resume")
		debug, _ := cmd.Flags().GetBool("debug")
		specName, _ := cmd.Flags().GetString("spec")

		// Load configuration
		cfg, err := config.Load(configPath)
		if err != nil {
			cliErr := clierrors.ConfigParseError(configPath, err)
			clierrors.PrintError(cliErr)
			return cliErr
		}

		// Create notification handler and history logger
		notifHandler := notify.NewHandler(cfg.Notifications)
		historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)

		// Apply agent override from --agent flag (must happen before security notice)
		if _, err := shared.ApplyAgentOverride(cmd, cfg); err != nil {
			return err
		}
		shared.ApplyModelOverride(cmd, cfg)

		// Resolve agent to get its name for the security notice
		agent, err := shared.ResolveAgent(cmd, cfg)
		if err != nil {
			return err
		}

		// Show security notice (once per user, only for Claude)
		shared.ShowSecurityNotice(cmd.OutOrStdout(), cfg, agent.Name())

		// Apply auto-commit override from flags
		shared.ApplyAutoCommitOverride(cmd, cfg)

		// Show one-time auto-commit notice if using default value
		lifecycle.ShowAutoCommitNoticeIfNeeded(cfg.StateDir, cfg.AutoCommitSource)

		// Wrap command execution with lifecycle for timing, notification, and history
		// Note: spec name is empty for all since we're creating a new spec
		return lifecycle.RunWithHistory(notifHandler, historyLogger, "all", "", func() error {
			// Override skip-preflight from flag if set
			if cmd.Flags().Changed("skip-preflight") {
				cfg.SkipPreflight = skipPreflight
			}

			// Override max-retries from flag if set
			if cmd.Flags().Changed("max-retries") {
				cfg.MaxRetries = maxRetries
			}

			// Check if constitution exists (required for all workflow stages)
			constitutionCheck := workflow.CheckConstitutionExists()
			if !constitutionCheck.Exists {
				fmt.Fprint(os.Stderr, constitutionCheck.ErrorMessage)
				return fmt.Errorf("constitution required")
			}

			// Create workflow orchestrator
			orchestrator := workflow.NewWorkflowOrchestrator(cfg)
			orchestrator.Debug = debug
			orchestrator.Executor.Debug = debug
			orchestrator.Executor.NotificationHandler = notifHandler

			// Apply output style from CLI flag (overrides config)
			shared.ApplyOutputStyle(cmd, orchestrator)

			if debug {
				fmt.Println("[DEBUG] Debug mode enabled")
				fmt.Printf("[DEBUG] Config: %+v\n", cfg)
			}

			if specName != "" {
				activeFeature, err := resolvePrepFeature(cfg, specName)
				if err != nil {
					return err
				}
				PrintSpecInfo(activeFeature.Metadata)
				fmt.Printf("  active feature: %s (source: %s)\n", activeFeature.Metadata.Directory, activeFeature.Source)
				resolvedName := fmt.Sprintf("%s-%s", activeFeature.Metadata.Number, activeFeature.Metadata.Name)
				if err := orchestrator.ExecutePlan(resolvedName, featureDescription); err != nil {
					return fmt.Errorf("all plan stage failed: %w", err)
				}
				if err := orchestrator.ExecuteTasks(resolvedName, featureDescription); err != nil {
					return fmt.Errorf("all tasks stage failed: %w", err)
				}
				if err := orchestrator.ExecuteImplement(resolvedName, "", resume, workflow.PhaseExecutionOptions{}); err != nil {
					return fmt.Errorf("all implement stage failed: %w", err)
				}
				return nil
			}

			// Run full workflow
			if err := orchestrator.RunFullWorkflow(featureDescription, resume); err != nil {
				return fmt.Errorf("full workflow failed: %w", err)
			}

			return nil
		})
	},
}

func init() {
	allCmd.GroupID = GroupWorkflows
	rootCmd.AddCommand(allCmd)

	allCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts (overrides config when set)")
	allCmd.Flags().Bool("resume", false, "Resume implementation from where it left off")
	allCmd.Flags().String("spec", "", "Specify which spec to run (overrides branch detection)")
	shared.AddModelFlag(allCmd)

	// Auto-commit flags
	shared.AddAutoCommitFlags(allCmd)
}
