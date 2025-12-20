package worktree

import (
	"fmt"
	"os"
	"time"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/worktree"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tracked worktrees",
	Long: `List all tracked worktrees with their status, branch, and creation time.

The output shows:
- Name: The worktree identifier
- Path: The filesystem path
- Branch: The git branch checked out
- Status: Current state (active, merged, abandoned, stale)
- Created: When the worktree was created`,
	Example: `  # List all worktrees
  autospec worktree list`,
	Args: cobra.NoArgs,
	RunE: runList,
}

func runList(cmd *cobra.Command, _ []string) error {
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

	worktrees, err := manager.List()
	if err != nil {
		return fmt.Errorf("listing worktrees: %w", err)
	}

	if len(worktrees) == 0 {
		fmt.Println("No worktrees tracked.")
		fmt.Println("Create one with: autospec worktree create <name> --branch <branch>")
		return nil
	}

	printWorktreeTable(worktrees)
	return nil
}

func printWorktreeTable(worktrees []worktree.Worktree) {
	// Print header
	fmt.Printf("%-20s %-40s %-25s %-10s %s\n", "NAME", "PATH", "BRANCH", "STATUS", "CREATED")
	fmt.Println(repeatString("-", 110))

	for _, wt := range worktrees {
		statusColor := getStatusColor(wt.Status)
		createdAgo := relativeTime(wt.CreatedAt)

		// Truncate long paths
		path := wt.Path
		if len(path) > 38 {
			path = "..." + path[len(path)-35:]
		}

		// Truncate long branch names
		branch := wt.Branch
		if len(branch) > 23 {
			branch = branch[:20] + "..."
		}

		fmt.Printf("%-20s %-40s %-25s %s %s\n",
			wt.Name,
			path,
			branch,
			statusColor.Sprintf("%-10s", wt.Status),
			createdAgo,
		)
	}
}

func getStatusColor(status worktree.WorktreeStatus) *color.Color {
	switch status {
	case worktree.StatusActive:
		return color.New(color.FgGreen)
	case worktree.StatusMerged:
		return color.New(color.FgBlue)
	case worktree.StatusAbandoned:
		return color.New(color.FgYellow)
	case worktree.StatusStale:
		return color.New(color.FgRed)
	default:
		return color.New(color.FgWhite)
	}
}

func relativeTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		return fmt.Sprintf("%d min ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		return fmt.Sprintf("%d hours ago", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}

func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

func init() {
	// Silence the unused variable warning
	_ = os.Stdout
}
