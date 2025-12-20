package worktree

import (
	"fmt"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/worktree"
	"github.com/spf13/cobra"
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove stale worktree entries",
	Long: `Remove tracking entries for worktrees whose paths no longer exist.

This is useful after manually deleting worktree directories outside of autospec.
The prune command only removes tracking entries - it does not delete any files.`,
	Example: `  # Prune stale entries
  autospec worktree prune`,
	Args: cobra.NoArgs,
	RunE: runPrune,
}

func runPrune(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	repoRoot, err := worktree.GetRepoRoot(".")
	if err != nil {
		return fmt.Errorf("getting repository root: %w", err)
	}

	wtConfig := cfg.Worktree
	if wtConfig == nil {
		wtConfig = worktree.DefaultConfig()
	}

	manager := worktree.NewManager(wtConfig, cfg.StateDir, repoRoot)

	pruned, err := manager.Prune()
	if err != nil {
		return fmt.Errorf("pruning worktrees: %w", err)
	}

	if pruned == 0 {
		fmt.Println("No stale worktree entries found.")
	} else {
		fmt.Printf("✓ Pruned %d stale worktree %s\n", pruned, pluralize("entry", pruned))
	}

	return nil
}

func pluralize(singular string, count int) string {
	if count == 1 {
		return singular
	}
	if singular == "entry" {
		return "entries"
	}
	return singular + "s"
}
