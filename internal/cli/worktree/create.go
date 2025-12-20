package worktree

import (
	"fmt"
	"os"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/history"
	"github.com/ariel-frischer/autospec/internal/lifecycle"
	"github.com/ariel-frischer/autospec/internal/notify"
	"github.com/ariel-frischer/autospec/internal/worktree"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new worktree with automatic setup",
	Long: `Create a new git worktree with automatic project configuration.

This command:
1. Creates a new git worktree using 'git worktree add'
2. Copies configured directories (.autospec/, .claude/) to the new worktree
3. Runs the project setup script if configured

The worktree is tracked in .autospec/state/worktrees.yaml for management.`,
	Example: `  # Create worktree with a new branch
  autospec worktree create feature-auth --branch feat/auth

  # Create worktree at a custom path
  autospec worktree create my-feature --branch feat/login --path /tmp/my-feature`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

func init() {
	createCmd.Flags().StringP("branch", "b", "", "Branch name for the worktree (required)")
	createCmd.Flags().StringP("path", "p", "", "Custom path for the worktree (optional)")
	createCmd.MarkFlagRequired("branch")
}

func runCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	branch, _ := cmd.Flags().GetString("branch")
	customPath, _ := cmd.Flags().GetString("path")

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	notifHandler := notify.NewHandler(cfg.Notifications)
	historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)

	return lifecycle.RunWithHistory(notifHandler, historyLogger, "worktree-create", name, func() error {
		return executeCreate(cfg, name, branch, customPath)
	})
}

func executeCreate(cfg *config.Configuration, name, branch, customPath string) error {
	repoRoot, err := worktree.GetRepoRoot(".")
	if err != nil {
		return fmt.Errorf("getting repository root: %w", err)
	}

	wtConfig := cfg.Worktree
	if wtConfig == nil {
		wtConfig = worktree.DefaultConfig()
	}

	manager := worktree.NewManager(wtConfig, cfg.StateDir, repoRoot, worktree.WithStdout(os.Stdout))

	wt, err := manager.Create(name, branch, customPath)
	if err != nil {
		return fmt.Errorf("creating worktree: %w", err)
	}

	fmt.Printf("✓ Created worktree: %s\n", wt.Name)
	fmt.Printf("  Path: %s\n", wt.Path)
	fmt.Printf("  Branch: %s\n", wt.Branch)
	if wt.SetupCompleted {
		fmt.Println("  Setup: completed")
	} else {
		fmt.Println("  Setup: failed (run 'autospec worktree setup' to retry)")
	}

	return nil
}
