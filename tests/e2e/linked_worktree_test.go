//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestE2E_StatusDetectsSpecInLinkedWorktree verifies spec auto-detection reads
// the branch of the worktree autospec runs in, including git linked worktrees
// (where .git is a file). The other spec is made newer so the mtime fallback
// would pick it if branch detection failed.
func TestE2E_StatusDetectsSpecInLinkedWorktree(t *testing.T) {
	tests := map[string]struct {
		dir      func(root string) string
		wantSpec string
		newer    string
	}{
		"main worktree": {
			dir:      func(root string) string { return root },
			wantSpec: "002-other-feature",
			newer:    "001-test-feature",
		},
		"linked worktree": {
			dir:      func(root string) string { return filepath.Join(root, "linked-wt") },
			wantSpec: "001-test-feature",
			newer:    "002-other-feature",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)
			setupLinkedWorktreeFixture(t, env)
			dir := tt.dir(env.TempDir())
			future := time.Now().Add(time.Hour)
			require.NoError(t, os.Chtimes(filepath.Join(dir, "specs", tt.newer), future, future))

			result := env.RunIn(dir, "status")

			require.Equal(t, 0, result.ExitCode, "stdout=%s\nstderr=%s", result.Stdout, result.Stderr)
			require.Contains(t, result.Stdout, tt.wantSpec)
			require.Contains(t, result.Stdout, "via git branch")
			require.NotContains(t, result.Stdout, tt.newer)
		})
	}
}

// setupLinkedWorktreeFixture commits two specs, leaves the main worktree on
// 002-other-feature, and adds a linked worktree on 001-test-feature.
func setupLinkedWorktreeFixture(t *testing.T, env *testutil.E2EEnv) {
	t.Helper()
	env.SetupAutospecInit()
	env.SetupConstitution()
	env.SetupTasks("001-test-feature")
	env.SetupTasks("002-other-feature")
	env.InitGitRepo()
	env.CreateBranch("002-other-feature")
	root := env.TempDir()
	runGit(t, root, "worktree", "add", "-b", "001-test-feature", filepath.Join(root, "linked-wt"))
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, output)
}
