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

var removeCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm"},
	Short:   "Remove a tracked worktree",
	Long: `Remove a tracked worktree with safety checks.

By default, this command will refuse to remove a worktree that has:
- Uncommitted changes
- Unpushed commits

Use --force to bypass these safety checks.`,
	Example: `  # Remove a worktree (with safety checks)
  autospec worktree remove feature-auth

  # Force remove (bypasses safety checks)
  autospec worktree remove feature-auth --force`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

func init() {
	removeCmd.Flags().BoolP("force", "f", false, "Force removal (bypasses safety checks)")
}

func runRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	force, _ := cmd.Flags().GetBool("force")

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	notifHandler := notify.NewHandler(cfg.Notifications)
	historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)

	return lifecycle.RunWithHistory(notifHandler, historyLogger, "worktree-remove", name, func() error {
		return executeRemove(cfg, name, force)
	})
}

func executeRemove(cfg *config.Configuration, name string, force bool) error {
	repoRoot, err := worktree.GetRepoRoot(".")
	if err != nil {
		return fmt.Errorf("getting repository root: %w", err)
	}

	wtConfig := cfg.Worktree
	if wtConfig == nil {
		wtConfig = worktree.DefaultConfig()
	}

	manager := worktree.NewManager(wtConfig, cfg.StateDir, repoRoot, worktree.WithStdout(os.Stdout))

	if err := manager.Remove(name, force); err != nil {
		return fmt.Errorf("removing worktree: %w", err)
	}

	fmt.Printf("✓ Removed worktree: %s\n", name)
	return nil
}
