package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateBranchPreservesUntrackedDirectories verifies that CreateBranch
// does not delete untracked directories (like .autospec/) when creating
// and checking out a new branch.
//
// This is a regression test for a bug where go-git's Checkout without
// Keep: true would delete all untracked files and directories.
func TestCreateBranchPreservesUntrackedDirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "create-branch-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Save current dir and change to temp
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(originalDir)

	// Initialize git repo
	repo, err := git.PlainInit(tmpDir, false)
	require.NoError(t, err)

	worktree, err := repo.Worktree()
	require.NoError(t, err)

	// Create and commit a file
	require.NoError(t, os.WriteFile("README.md", []byte("# Test"), 0o644))
	_, err = worktree.Add("README.md")
	require.NoError(t, err)
	_, err = worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@test.com",
		},
	})
	require.NoError(t, err)

	// Create .autospec with constitution (simulating autospec init)
	require.NoError(t, os.MkdirAll(".autospec/memory", 0o755))
	require.NoError(t, os.WriteFile(".autospec/memory/constitution.yaml", []byte("project_name: Test"), 0o644))
	require.NoError(t, os.WriteFile(".autospec/init.yml", []byte("version: 1.0.0"), 0o644))

	// Verify exists before
	_, err = os.Stat(".autospec/memory/constitution.yaml")
	require.NoError(t, err, "constitution should exist before CreateBranch")

	// Call CreateBranch (simulating what new-feature --json does)
	err = CreateBranch("001-test-feature")
	require.NoError(t, err, "CreateBranch should succeed")

	// Verify .autospec still exists after checkout
	_, err = os.Stat(".autospec")
	assert.NoError(t, err, ".autospec should exist after CreateBranch")

	_, err = os.Stat(".autospec/memory/constitution.yaml")
	assert.NoError(t, err, "constitution.yaml should exist after CreateBranch")

	_, err = os.Stat(".autospec/init.yml")
	assert.NoError(t, err, "init.yml should exist after CreateBranch")

	// Verify content is intact
	content, err := os.ReadFile(".autospec/memory/constitution.yaml")
	require.NoError(t, err)
	assert.Contains(t, string(content), "project_name: Test")

	// Verify we're on the new branch
	currentBranch, err := GetCurrentBranch()
	require.NoError(t, err)
	assert.Equal(t, "001-test-feature", currentBranch)
}

// TestCreateBranchPreservesGitignored tests the same scenario with
// .autospec in .gitignore (recommended setup).
func TestCreateBranchPreservesGitignored(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "create-branch-gitignore-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(originalDir)

	repo, err := git.PlainInit(tmpDir, false)
	require.NoError(t, err)

	worktree, err := repo.Worktree()
	require.NoError(t, err)

	// Create .gitignore with .autospec/
	require.NoError(t, os.WriteFile(".gitignore", []byte(".autospec/\n"), 0o644))
	_, err = worktree.Add(".gitignore")
	require.NoError(t, err)

	require.NoError(t, os.WriteFile("README.md", []byte("# Test"), 0o644))
	_, err = worktree.Add("README.md")
	require.NoError(t, err)

	_, err = worktree.Commit("Initial commit with gitignore", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@test.com",
		},
	})
	require.NoError(t, err)

	// Create .autospec directory (ignored by git)
	require.NoError(t, os.MkdirAll(".autospec/memory", 0o755))
	require.NoError(t, os.WriteFile(".autospec/memory/constitution.yaml", []byte("project_name: Test"), 0o644))

	// Create new branch
	err = CreateBranch("002-another-feature")
	require.NoError(t, err)

	// Verify .autospec still exists
	_, err = os.Stat(".autospec")
	assert.NoError(t, err, ".autospec should exist after checkout (gitignored)")

	_, err = os.Stat(".autospec/memory/constitution.yaml")
	assert.NoError(t, err, "constitution.yaml should exist after checkout (gitignored)")
}

// TestGoGitCheckoutWithoutKeepPreservesUntracked documents that go-git v5.16.5+
// no longer deletes untracked directories on Checkout without Keep: true.
// We still use Keep: true in CreateBranch as defense-in-depth.
func TestGoGitCheckoutWithoutKeepPreservesUntracked(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git-checkout-danger-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	repo, err := git.PlainInit(tmpDir, false)
	require.NoError(t, err)

	worktree, err := repo.Worktree()
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test"), 0o644))
	_, err = worktree.Add("README.md")
	require.NoError(t, err)
	_, err = worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@test.com"},
	})
	require.NoError(t, err)

	// Create untracked directory
	untrackedDir := filepath.Join(tmpDir, "untracked-dir")
	require.NoError(t, os.MkdirAll(untrackedDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(untrackedDir, "file.txt"), []byte("test"), 0o644))

	head, err := repo.Head()
	require.NoError(t, err)

	// Checkout WITHOUT Keep: true — as of go-git v5.16.5, untracked
	// files are preserved even without Keep: true.
	branchRef := plumbing.NewBranchReferenceName("test-no-keep")
	err = worktree.Checkout(&git.CheckoutOptions{
		Hash:   head.Hash(),
		Branch: branchRef,
		Create: true,
	})
	require.NoError(t, err)

	// go-git v5.16.5+ preserves untracked dirs without Keep: true.
	// We still use Keep: true in CreateBranch as defense-in-depth.
	_, err = os.Stat(untrackedDir)
	assert.NoError(t, err, "go-git v5.16.5+ should preserve untracked directories even without Keep: true")
}
