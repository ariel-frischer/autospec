package worktree

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/history"
	"github.com/ariel-frischer/autospec/internal/lifecycle"
	"github.com/ariel-frischer/autospec/internal/notify"
	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/ariel-frischer/autospec/internal/worktree"
	"github.com/spf13/cobra"
)

// GenScriptRunner is a mockable function for running Claude worktree-setup generation.
// Tests can replace this to prevent real API calls.
var GenScriptRunner = runClaudeGenerationImpl

var genScriptCmd = &cobra.Command{
	Use:   "gen-script",
	Short: "Generate a project-specific worktree setup script",
	Long: `Generate a setup script for new worktrees by analyzing your project.

This command uses Claude to analyze your project and generate a customized
setup-worktree.sh script that:
- Copies essential configuration directories (.autospec/, .claude/)
- Excludes secrets and credentials by default
- Runs package manager install commands instead of copying dependencies

The generated script is saved to .autospec/scripts/setup-worktree.sh and is
automatically used by 'autospec worktree create'.`,
	Example: `  # Generate a worktree setup script
  autospec worktree gen-script

  # Generate with environment files included (not recommended)
  autospec worktree gen-script --include-env`,
	RunE: runGenScript,
}

func init() {
	genScriptCmd.Flags().Bool("include-env", false, "Include .env files in copy list (security warning)")
}

func runGenScript(cmd *cobra.Command, _ []string) error {
	includeEnv, _ := cmd.Flags().GetBool("include-env")

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	notifHandler := notify.NewHandler(cfg.Notifications)
	historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)

	return lifecycle.RunWithHistory(notifHandler, historyLogger, "worktree-gen-script", "", func() error {
		return executeGenScript(cfg, includeEnv)
	})
}

func executeGenScript(cfg *config.Configuration, includeEnv bool) error {
	if err := verifyGitRepo(); err != nil {
		return fmt.Errorf("verifying git repository: %w", err)
	}

	if includeEnv {
		printSecurityWarning()
	}

	if err := ensureScriptsDir(); err != nil {
		return fmt.Errorf("ensuring scripts directory: %w", err)
	}

	return runClaudeGeneration(cfg, includeEnv)
}

// verifyGitRepo checks if the current directory is a git repository.
func verifyGitRepo() error {
	if _, err := worktree.GetRepoRoot("."); err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}
	return nil
}

// printSecurityWarning displays a warning about including environment files.
func printSecurityWarning() {
	fmt.Println("WARNING: --include-env specified. Environment files containing secrets will be copied to worktrees.")
	fmt.Println("Only use this option if you understand the security implications.")
	fmt.Println()
}

// ensureScriptsDir creates the .autospec/scripts/ directory if it doesn't exist.
func ensureScriptsDir() error {
	scriptsDir := filepath.Join(".autospec", "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		return fmt.Errorf("creating scripts directory: %w", err)
	}
	return nil
}

// runClaudeGeneration invokes Claude via the mockable GenScriptRunner.
func runClaudeGeneration(cfg *config.Configuration, includeEnv bool) error {
	return GenScriptRunner(cfg, includeEnv)
}

// runClaudeGenerationImpl is the real implementation that invokes Claude.
func runClaudeGenerationImpl(cfg *config.Configuration, includeEnv bool) error {
	orch := workflow.NewWorkflowOrchestrator(cfg)

	command := buildWorktreeSetupCommand(includeEnv)
	fmt.Printf("Executing: %s\n\n", command)

	if err := orch.Executor.Claude.Execute(command); err != nil {
		return fmt.Errorf("generating worktree setup script: %w", err)
	}

	fmt.Println("\n✓ Worktree setup script generated!")
	fmt.Println("  Location: .autospec/scripts/setup-worktree.sh")
	return nil
}

// buildWorktreeSetupCommand builds the Claude command for worktree setup generation.
func buildWorktreeSetupCommand(includeEnv bool) string {
	if includeEnv {
		return "/autospec.worktree-setup --include-env"
	}
	return "/autospec.worktree-setup"
}
