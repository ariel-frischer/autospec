package cli

import (
	"fmt"

	"github.com/anthropics/auto-claude-speckit/internal/config"
	"github.com/anthropics/auto-claude-speckit/internal/workflow"
	"github.com/spf13/cobra"
)

var implementCmd = &cobra.Command{
	Use:   "implement [spec-name]",
	Short: "Execute the implementation phase for the current spec",
	Long: `Execute the /speckit.implement command for the current specification.

The implement command will:
- Auto-detect the current spec from git branch or most recent spec (if no spec-name provided)
- Execute the implementation workflow
- Validate that all tasks in tasks.md are completed
- Support resuming from where it left off with --resume flag

Examples:
  autospec implement                    # Auto-detect spec and implement
  autospec implement --resume           # Resume implementation from where it left off
  autospec implement 003-my-feature     # Implement specific spec`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get optional spec name from args
		var specName string
		if len(args) > 0 {
			specName = args[0]
		}

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
		orch := workflow.NewWorkflowOrchestrator(cfg)

		// Execute implement phase
		if err := orch.ExecuteImplement(specName, resume); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(implementCmd)

	// Command-specific flags
	implementCmd.Flags().Bool("resume", false, "Resume implementation from where it left off")
	implementCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts (0 = use config)")
}
