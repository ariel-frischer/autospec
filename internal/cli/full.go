package cli

import (
	"fmt"

	"github.com/anthropics/auto-claude-speckit/internal/config"
	"github.com/anthropics/auto-claude-speckit/internal/workflow"
	"github.com/spf13/cobra"
)

var fullCmd = &cobra.Command{
	Use:   "full <feature-description>",
	Short: "Run complete specify → plan → tasks → implement workflow",
	Long: `Run the complete SpecKit workflow including implementation with automatic validation and retry.

This command will:
1. Run pre-flight checks (unless --skip-preflight)
2. Execute /speckit.specify with the feature description
3. Validate spec.md exists
4. Execute /speckit.plan
5. Validate plan.md exists
6. Execute /speckit.tasks
7. Validate tasks.md exists
8. Execute /speckit.implement
9. Validate all tasks are completed

Each phase is validated and will retry up to max_retries times if validation fails.
This is equivalent to running 'autospec workflow' followed by 'autospec implement'.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		featureDescription := args[0]

		// Get flags
		configPath, _ := cmd.Flags().GetString("config")
		skipPreflight, _ := cmd.Flags().GetBool("skip-preflight")
		maxRetries, _ := cmd.Flags().GetInt("max-retries")
		resume, _ := cmd.Flags().GetBool("resume")

		// Load configuration
		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Override skip-preflight from flag if set
		if cmd.Flags().Changed("skip-preflight") {
			cfg.SkipPreflight = skipPreflight
		}

		// Override max-retries from flag if set
		if cmd.Flags().Changed("max-retries") {
			cfg.MaxRetries = maxRetries
		}

		// Create workflow orchestrator
		orchestrator := workflow.NewWorkflowOrchestrator(cfg)

		// Run full workflow
		if err := orchestrator.RunFullWorkflow(featureDescription, resume); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(fullCmd)

	// Command-specific flags
	fullCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts (0 = use config)")
	fullCmd.Flags().Bool("resume", false, "Resume implementation from where it left off")
}
