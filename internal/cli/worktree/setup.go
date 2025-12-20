package worktree

import (
	"fmt"
	"os"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/worktree"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup <path>",
	Short: "Run setup on an existing worktree",
	Long: `Run the project setup on an existing git worktree.

This command copies configured directories and runs the setup script on a worktree
that wasn't created with 'autospec worktree create'.

Use --track to add the worktree to the tracking state.`,
	Example: `  # Setup an existing worktree
  autospec worktree setup ../my-worktree

  # Setup and add to tracking
  autospec worktree setup ../my-worktree --track`,
	Args: cobra.ExactArgs(1),
	RunE: runSetup,
}

func init() {
	setupCmd.Flags().Bool("track", false, "Add worktree to tracking state")
}

func runSetup(cmd *cobra.Command, args []string) error {
	path := args[0]
	track, _ := cmd.Flags().GetBool("track")

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

	manager := worktree.NewManager(wtConfig, cfg.StateDir, repoRoot, worktree.WithStdout(os.Stdout))

	wt, err := manager.Setup(path, track)
	if err != nil {
		return fmt.Errorf("setting up worktree: %w", err)
	}

	fmt.Printf("✓ Setup complete for: %s\n", wt.Path)
	if wt.SetupCompleted {
		fmt.Println("  Setup script: completed")
	} else {
		fmt.Println("  Setup script: failed")
	}
	if track {
		fmt.Printf("  Tracked as: %s\n", wt.Name)
	}

	return nil
}
