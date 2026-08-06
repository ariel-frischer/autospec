package worktree

import (
	"fmt"
	"os"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
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
4. Validates the worktree after custom setup script execution

The worktree is tracked in .autospec/state/worktrees.yaml for management.

On setup failure or validation failure, the worktree is automatically deleted
unless --no-rollback is specified. Use --no-rollback to preserve the broken
worktree for debugging purposes.`,
	Example: `  # Create worktree with a new branch
  autospec worktree create feature-auth --branch feat/auth

  # Create worktree at a custom path
  autospec worktree create my-feature --branch feat/login --path /tmp/my-feature

  # Create worktree without copying directories
  autospec worktree create feature-fast --branch feat/fast --skip-copy

  # Create worktree without running setup script
  autospec worktree create feature-manual --branch feat/manual --skip-setup

  # Create worktree and preserve on failure for debugging
  autospec worktree create feature-debug --branch feat/debug --no-rollback`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

func init() {
	createCmd.Flags().StringP("branch", "b", "", "Branch name for the worktree (required)")
	createCmd.Flags().StringP("path", "p", "", "Custom path for the worktree (optional)")
	createCmd.Flags().Bool("skip-copy", false, "Skip copying directories listed in worktree.copy_dirs")
	createCmd.Flags().Bool("skip-setup", false, "Skip running the setup script after worktree creation")
	createCmd.Flags().Bool("no-rollback", false, "Preserve worktree on failure for debugging (no auto-cleanup)")
	createCmd.MarkFlagRequired("branch")
}

func runCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	branch, _ := cmd.Flags().GetString("branch")
	customPath, _ := cmd.Flags().GetString("path")
	skipCopy, _ := cmd.Flags().GetBool("skip-copy")
	skipSetup, _ := cmd.Flags().GetBool("skip-setup")
	noRollback, _ := cmd.Flags().GetBool("no-rollback")

	cfg, err := shared.LoadConfig(cmd, "")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	opts := worktree.CreateOptions{
		SkipCopy:   skipCopy,
		SkipSetup:  skipSetup,
		NoRollback: noRollback,
	}

	notifHandler := notify.NewHandler(cfg.Notifications)
	historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)

	return lifecycle.RunWithHistory(notifHandler, historyLogger, "worktree-create", name, func() error {
		return executeCreate(cfg, name, branch, customPath, opts)
	})
}

func executeCreate(
	cfg *config.Configuration,
	name, branch, customPath string,
	opts worktree.CreateOptions,
) error {
	repoRoot, err := worktree.GetRepoRoot(".")
	if err != nil {
		return fmt.Errorf("getting repository root: %w", err)
	}

	wtConfig := cfg.Worktree
	if wtConfig == nil {
		wtConfig = worktree.DefaultConfig()
	}

	manager := worktree.NewManager(wtConfig, cfg.StateDir, repoRoot, worktree.WithStdout(os.Stdout))

	wt, err := manager.CreateWithOptions(name, branch, customPath, opts)
	if err != nil {
		return fmt.Errorf("creating worktree: %w", err)
	}

	fmt.Printf("✓ Created worktree: %s\n", wt.Name)
	fmt.Printf("  Path: %s\n", wt.Path)
	fmt.Printf("  Branch: %s\n", wt.Branch)
	printSetupStatus(wt, opts)

	return nil
}

// printSetupStatus prints the setup status based on worktree state and options.
func printSetupStatus(wt *worktree.Worktree, opts worktree.CreateOptions) {
	switch {
	case opts.SkipSetup:
		fmt.Println("  Setup: skipped (--skip-setup)")
	case wt.SetupCompleted:
		fmt.Println("  Setup: completed")
	default:
		fmt.Println("  Setup: failed (run 'autospec worktree setup' to retry)")
	}
}
