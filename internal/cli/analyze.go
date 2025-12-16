package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
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

		// Auto-detect current spec and verify all required artifacts exist
		metadata, err := spec.DetectCurrentSpec(cfg.SpecsDir)
		if err != nil {
			return fmt.Errorf("failed to detect current spec: %w\n\nRun 'autospec specify' to create a new spec first", err)
		}

		// Check that all required artifacts exist
		var missingArtifacts []string
		specFile := filepath.Join(metadata.Directory, "spec.yaml")
		if _, err := os.Stat(specFile); os.IsNotExist(err) {
			missingArtifacts = append(missingArtifacts, "spec.yaml")
		}
		planFile := filepath.Join(metadata.Directory, "plan.yaml")
		if _, err := os.Stat(planFile); os.IsNotExist(err) {
			missingArtifacts = append(missingArtifacts, "plan.yaml")
		}
		tasksFile := filepath.Join(metadata.Directory, "tasks.yaml")
		if _, err := os.Stat(tasksFile); os.IsNotExist(err) {
			missingArtifacts = append(missingArtifacts, "tasks.yaml")
		}

		if len(missingArtifacts) > 0 {
			return fmt.Errorf("missing required artifacts in %s: %v\n\nRun the following commands first:\n  - autospec specify (for spec.yaml)\n  - autospec plan (for plan.yaml)\n  - autospec tasks (for tasks.yaml)", metadata.Directory, missingArtifacts)
		}

		// Create workflow orchestrator
		orch := workflow.NewWorkflowOrchestrator(cfg)

		// Execute analyze phase
		specName := fmt.Sprintf("%s-%s", metadata.Number, metadata.Name)
		if err := orch.ExecuteAnalyze(specName, prompt); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
}
