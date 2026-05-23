package cli

import (
	"context"
	"fmt"
	"os"

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

var prepCmd = &cobra.Command{
	Use:   "prep <feature-description>",
	Short: "Prepare for implementation: specify → plan → tasks",
	Long: `Prepare for implementation by running specify, plan, and tasks stages.

This command will:
1. Run pre-flight checks (unless --skip-preflight)
2. Execute /autospec.specify with the feature description
3. Validate spec.yaml exists
4. Execute /autospec.plan
5. Validate plan.yaml exists
6. Execute /autospec.tasks
7. Validate tasks.yaml exists

Each stage is validated and will retry up to max_retries times if validation fails.

This is useful when you want to review the generated artifacts before implementation.`,
	Example: `  # Prepare spec, plan, and tasks for review before implementation
  autospec prep "Add user authentication feature"

  # Prepare artifacts for review
  autospec prep "Refactor database layer"`,
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

		// Override skip-preflight from flag if set
		if cmd.Flags().Changed("skip-preflight") {
			cfg.SkipPreflight = skipPreflight
		}

		// Override max-retries from flag if set
		if cmd.Flags().Changed("max-retries") {
			cfg.MaxRetries = maxRetries
		}

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

		// Wrap command execution with lifecycle for timing, notification, and history
		// Use RunWithHistoryContext to support context cancellation (e.g., Ctrl+C)
		// Note: spec name is empty for prep since we're creating a new spec
		return lifecycle.RunWithHistoryContext(cmd.Context(), notifHandler, historyLogger, "prep", "", func(_ context.Context) error {

			// Apply auto-commit override from flags
			shared.ApplyAutoCommitOverride(cmd, cfg)

			// Show one-time auto-commit notice if using default value
			lifecycle.ShowAutoCommitNoticeIfNeeded(cfg.StateDir, cfg.AutoCommitSource)

			// Check if constitution exists (required for all workflow stages)
			constitutionCheck := workflow.CheckConstitutionExists()
			if !constitutionCheck.Exists {
				fmt.Fprint(os.Stderr, constitutionCheck.ErrorMessage)
				return fmt.Errorf("constitution required")
			}

			// Create workflow orchestrator
			orchestrator := workflow.NewWorkflowOrchestrator(cfg)
			orchestrator.Executor.NotificationHandler = notifHandler

			// Apply output style from CLI flag (overrides config)
			shared.ApplyOutputStyle(cmd, orchestrator)

			// Apply OpenCode agent if the flag is present
			shared.ApplyOpenCodeAgent(cmd, cfg, orchestrator)

			if specName != "" {
				activeFeature, err := resolvePrepFeature(cfg, specName)
				if err != nil {
					return err
				}
				PrintSpecInfo(activeFeature.Metadata)
				fmt.Printf("  active feature: %s (source: %s)\n", activeFeature.Metadata.Directory, activeFeature.Source)
				resolvedName := fmt.Sprintf("%s-%s", activeFeature.Metadata.Number, activeFeature.Metadata.Name)
				if err := orchestrator.ExecutePlan(resolvedName, featureDescription); err != nil {
					return fmt.Errorf("prep plan stage failed: %w", err)
				}
				if err := orchestrator.ExecuteTasks(resolvedName, featureDescription); err != nil {
					return fmt.Errorf("prep tasks stage failed: %w", err)
				}
				return nil
			}

			// Run complete workflow (specify → plan → tasks, no implementation)
			if err := orchestrator.RunCompleteWorkflow(featureDescription); err != nil {
				return fmt.Errorf("prep workflow failed: %w", err)
			}

			return nil
		})
	},
}

func init() {
	prepCmd.GroupID = GroupWorkflows
	rootCmd.AddCommand(prepCmd)

	// Command-specific flags
	prepCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts (overrides config when set)")
	prepCmd.Flags().String("spec", "", "Specify which spec to prepare (overrides branch detection)")

	// Agent override flag
	shared.AddAgentFlag(prepCmd)
	shared.AddModelFlag(prepCmd)

	// Opencode-agent flag
	shared.AddOpenCodeAgentFlag(prepCmd)

	// Auto-commit flags
	shared.AddAutoCommitFlags(prepCmd)
}

func validatePrepArgs(cmd *cobra.Command, args []string) error {
	specName, _ := cmd.Flags().GetString("spec")
	if specName != "" {
		return cobra.MaximumNArgs(1)(cmd, args)
	}
	return cobra.ExactArgs(1)(cmd, args)
}

func resolvePrepFeature(cfg *config.Configuration, specName string) (*spec.ActiveFeatureResult, error) {
	result, err := spec.ResolveActiveFeature(spec.ActiveFeatureRequest{
		SpecsDir:           cfg.SpecsDir,
		StateDir:           cfg.StateDir,
		ExplicitIdentifier: specName,
		RequiredArtifact:   "spec.yaml",
	})
	if err == nil {
		return result, nil
	}
	if specName != "" {
		return nil, fmt.Errorf("spec not found: %s: %w", specName, err)
	}
	return nil, fmt.Errorf("failed to detect spec for prep: %w", err)
}
