package cli

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/anthropics/auto-claude-speckit/internal/config"
	"github.com/anthropics/auto-claude-speckit/internal/workflow"
	"github.com/spf13/cobra"
)

var implementCmd = &cobra.Command{
	Use:   "implement [spec-name-or-prompt]",
	Short: "Execute the implementation phase for the current spec",
	Long: `Execute the /speckit.implement command for the current specification.

The implement command will:
- Auto-detect the current spec from git branch or most recent spec (if no spec-name provided)
- Execute the implementation workflow
- Validate that all tasks in tasks.md are completed
- Support resuming from where it left off with --resume flag
- Support optional prompt text to guide the implementation

You can optionally provide a prompt to guide the implementation process:
  autospec implement "Focus on documentation tasks"
  autospec implement "Complete the remaining tests"

Examples:
  autospec implement                              # Auto-detect spec and implement
  autospec implement --resume                     # Resume implementation from where it left off
  autospec implement 003-my-feature               # Implement specific spec
  autospec implement "Focus on error handling"    # Auto-detect spec with prompt guidance`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse args to distinguish between spec-name and prompt
		var specName string
		var prompt string

		if len(args) > 0 {
			// Check if first arg looks like a spec name (pattern: NNN-name)
			specNamePattern := regexp.MustCompile(`^\d+-[a-z0-9-]+$`)
			if specNamePattern.MatchString(args[0]) {
				// First arg is a spec name
				specName = args[0]
				// Remaining args are prompt
				if len(args) > 1 {
					prompt = strings.Join(args[1:], " ")
				}
			} else {
				// All args are prompt (auto-detect spec)
				prompt = strings.Join(args, " ")
			}
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

		// Execute implement phase with optional prompt
		if err := orch.ExecuteImplement(specName, prompt, resume); err != nil {
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
