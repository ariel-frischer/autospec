package git

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	behavior := os.Getenv("TEST_MOCK_BEHAVIOR")
	switch behavior {
	case "":
		os.Exit(m.Run())
	case "gitBranch":
		// Mock successful branch query
		os.Stdout.WriteString("002-go-binary-migration\n")
		os.Exit(0)
	case "gitRoot":
		// Mock successful root query
		os.Stdout.WriteString("/home/user/project\n")
		os.Exit(0)
	case "gitDir":
		// Mock successful git dir check
		os.Exit(0)
	default:
		os.Exit(m.Run())
	}
}

// TestGetCurrentBranch tests retrieving the current branch name
// Note: This test runs against the actual git repository, not a mock
func TestGetCurrentBranch_Real(t *testing.T) {
	branch, err := GetCurrentBranch()
	require.NoError(t, err)
	assert.NotEmpty(t, branch)
	// Just verify we get a valid branch name (non-empty string)
	// Don't hardcode a specific branch since it changes during development
}

// TestGetRepositoryRoot tests retrieving the repository root path
func TestGetRepositoryRoot_Real(t *testing.T) {
	root, err := GetRepositoryRoot()
	require.NoError(t, err)
	assert.NotEmpty(t, root)
	assert.Contains(t, root, "autospec")
}

// TestIsGitRepository tests checking if we're in a git repository
func TestIsGitRepository_Real(t *testing.T) {
	isRepo := IsGitRepository()
	assert.True(t, isRepo)
}

// TestGetAllBranches tests listing all branches
func TestGetAllBranches_Real(t *testing.T) {
	branches, err := GetAllBranches()
	require.NoError(t, err)
	assert.NotEmpty(t, branches)

	// Verify we have at least one branch
	assert.GreaterOrEqual(t, len(branches), 1)

	// Check that the current branch is in the list
	currentBranch, err := GetCurrentBranch()
	require.NoError(t, err)

	found := false
	for _, b := range branches {
		if b.Name == currentBranch {
			found = true
			break
		}
	}
	assert.True(t, found, "current branch should be in the branch list")
}

// TestGetBranchNames tests getting just branch names
func TestGetBranchNames_Real(t *testing.T) {
	names, err := GetBranchNames()
	require.NoError(t, err)
	assert.NotEmpty(t, names)

	// Current branch should be in the list
	currentBranch, err := GetCurrentBranch()
	require.NoError(t, err)

	assert.Contains(t, names, currentBranch)
}

// TestBranchInfo verifies BranchInfo structure
func TestBranchInfo(t *testing.T) {
	branches, err := GetAllBranches()
	require.NoError(t, err)

	for _, b := range branches {
		// Name should never be empty
		assert.NotEmpty(t, b.Name)
		// If IsRemote is true, Remote should have a value
		if b.IsRemote {
			assert.NotEmpty(t, b.Remote, "remote branch should have Remote field set")
		}
	}
}

// TestFetchAllRemotes tests fetching from remotes
// This is a light test since we don't want to actually hit the network in unit tests
func TestFetchAllRemotes_Real(t *testing.T) {
	// Just verify it doesn't panic or error fatally
	// The actual fetch might fail if there's no network, but that's ok
	_, err := FetchAllRemotes()
	// We accept either success or network failure
	// The function should never return a hard error, just false
	assert.NoError(t, err)
}
