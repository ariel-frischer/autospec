package git

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// newLinkedWorktree creates a repo with one commit and a linked worktree on
// branch "feature". It returns the worktree path.
func newLinkedWorktree(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	main := filepath.Join(base, "main")
	wt := filepath.Join(base, "wt")
	cmds := [][]string{
		{"init", "-q", "-b", "trunk", main},
		{"-C", main, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "init"},
		{"-C", main, "worktree", "add", "-q", "-b", "feature", wt},
	}
	for _, args := range cmds {
		out, err := exec.Command("git", args...).CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}
	return wt
}

func TestLinkedWorktree(t *testing.T) {
	wt := newLinkedWorktree(t)
	t.Chdir(wt)

	tests := map[string]func(t *testing.T) string{
		"GetCurrentBranch": func(t *testing.T) string {
			branch, err := GetCurrentBranch()
			require.NoError(t, err)
			return branch
		},
		"DefaultOpener": func(t *testing.T) string {
			repo, err := (&DefaultOpener{}).Open(wt)
			require.NoError(t, err)
			head, err := repo.Head()
			require.NoError(t, err)
			return head.Name().Short()
		},
		"GetAllBranches": func(t *testing.T) string {
			branches, err := GetAllBranches()
			require.NoError(t, err)
			for _, b := range branches {
				if b.Name == "feature" {
					return b.Name
				}
			}
			return ""
		},
	}
	for name, resolve := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, "feature", resolve(t))
		})
	}
}
