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
	// We're on 002-go-binary-migration branch based on git status
	assert.Equal(t, "002-go-binary-migration", branch)
}

// TestGetRepositoryRoot tests retrieving the repository root path
func TestGetRepositoryRoot_Real(t *testing.T) {
	root, err := GetRepositoryRoot()
	require.NoError(t, err)
	assert.NotEmpty(t, root)
	assert.Contains(t, root, "auto-claude-speckit")
}

// TestIsGitRepository tests checking if we're in a git repository
func TestIsGitRepository_Real(t *testing.T) {
	isRepo := IsGitRepository()
	assert.True(t, isRepo)
}
