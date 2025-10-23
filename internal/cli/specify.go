package cli

import (
	"fmt"
	"strings"

	"github.com/anthropics/auto-claude-speckit/internal/config"
	"github.com/anthropics/auto-claude-speckit/internal/workflow"
	"github.com/spf13/cobra"
)

var specifyCmd = &cobra.Command{
	Use:   "specify <feature-description>",
	Short: "Execute the specification phase for a new feature",
	Long: `Execute the /speckit.specify command to create a new feature specification.

The specify command will:
- Create a new spec directory with a spec.md file
- Generate the specification based on your feature description
- Output the spec name for use in subsequent commands

The feature description should be a clear, concise description of what you want to build.

Examples:
  autospec specify "Add user authentication feature"
  autospec specify "Implement dark mode support"
  autospec specify "Create REST API for user management"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Join all args as the feature description
		featureDescription := strings.Join(args, " ")

		// Get flags
		configPath, _ := cmd.Flags().GetString("config")
		skipPreflight, _ := cmd.Flags().GetBool("skip-preflight")
		maxRetries, _ := cmd.Flags().GetInt("max-retries")

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

		// Execute specify phase
		specName, err := orch.ExecuteSpecify(featureDescription)
		if err != nil {
			return err
		}

		fmt.Printf("\nSpec created: %s\n", specName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(specifyCmd)

	// Command-specific flags
	specifyCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts (0 = use config)")
}
