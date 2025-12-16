package cli

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/spf13/cobra"
)

var implementCmd = &cobra.Command{
	Use:     "implement [spec-name-or-prompt]",
	Aliases: []string{"impl", "i"},
	Short:   "Execute the implementation phase for the current spec (impl, i)",
	Long: `Execute the /autospec.implement command for the current specification.

The implement command will:
- Auto-detect the current spec from git branch or most recent spec
- Execute the implementation workflow based on tasks.yaml
- Track progress and validate task completion
- Support resuming from where it left off with --resume flag`,
	Example: `  # Auto-detect spec and implement
  autospec implement

  # Resume interrupted implementation
  autospec implement --resume

  # Implement a specific spec by name
  autospec implement 003-my-feature

  # Provide prompt guidance for implementation
  autospec implement "Focus on error handling first"`,
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

		// Check if constitution exists (required for implement)
		constitutionCheck := workflow.CheckConstitutionExists()
		if !constitutionCheck.Exists {
			fmt.Fprint(os.Stderr, constitutionCheck.ErrorMessage)
			return fmt.Errorf("constitution required")
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
