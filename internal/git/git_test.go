package git

import (
	"os"
	"os/exec"
	"path/filepath"
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

	// Skip branch-in-list check if in detached HEAD state (common in CI)
	if currentBranch == "HEAD" {
		t.Skip("skipping branch list check in detached HEAD state")
	}

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

	// Skip branch-in-list check if in detached HEAD state (common in CI)
	if currentBranch == "HEAD" {
		t.Skip("skipping branch list check in detached HEAD state")
	}

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

func TestParseBranchLine(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		line     string
		want     *BranchInfo
		wantNil  bool
	}{
		"simple local branch": {
			line: "main",
			want: &BranchInfo{Name: "main", IsRemote: false, Remote: ""},
		},
		"local branch with dash": {
			line: "feature-branch",
			want: &BranchInfo{Name: "feature-branch", IsRemote: false, Remote: ""},
		},
		"local branch with slash": {
			line: "feature/test",
			want: &BranchInfo{Name: "test", IsRemote: true, Remote: "feature"},
		},
		"remote branch origin": {
			line: "origin/main",
			want: &BranchInfo{Name: "main", IsRemote: true, Remote: "origin"},
		},
		"remote branch with remotes prefix": {
			line: "remotes/origin/main",
			want: &BranchInfo{Name: "main", IsRemote: true, Remote: "origin"},
		},
		"remote branch upstream": {
			line: "upstream/develop",
			want: &BranchInfo{Name: "develop", IsRemote: true, Remote: "upstream"},
		},
		"remote with nested path": {
			line: "remotes/origin/feature/my-feature",
			want: &BranchInfo{Name: "feature/my-feature", IsRemote: true, Remote: "origin"},
		},
		"remote without prefix but with slash": {
			line: "origin/feature/nested",
			want: &BranchInfo{Name: "feature/nested", IsRemote: true, Remote: "origin"},
		},
		"remotes prefix with invalid format": {
			line: "remotes/noslash",
			wantNil: true,
		},
		"empty string": {
			line: "",
			want: &BranchInfo{Name: "", IsRemote: false, Remote: ""},
		},
		"single slash": {
			line: "a/b",
			want: &BranchInfo{Name: "b", IsRemote: true, Remote: "a"},
		},
		"branch with numbers": {
			line: "038-test-coverage",
			want: &BranchInfo{Name: "038-test-coverage", IsRemote: false, Remote: ""},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := parseBranchLine(tt.line)

			if tt.wantNil {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.Equal(t, tt.want.Name, got.Name)
			assert.Equal(t, tt.want.IsRemote, got.IsRemote)
			assert.Equal(t, tt.want.Remote, got.Remote)
		})
	}
}

func TestAddBranchWithDedup(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		existing []BranchInfo
		info     BranchInfo
		seen     map[string]bool
		wantLen  int
		checkFn  func(t *testing.T, result []BranchInfo)
	}{
		"add new local branch": {
			existing: []BranchInfo{},
			info:     BranchInfo{Name: "main", IsRemote: false},
			seen:     map[string]bool{},
			wantLen:  1,
			checkFn: func(t *testing.T, result []BranchInfo) {
				assert.Equal(t, "main", result[0].Name)
				assert.False(t, result[0].IsRemote)
			},
		},
		"add new remote branch": {
			existing: []BranchInfo{},
			info:     BranchInfo{Name: "develop", IsRemote: true, Remote: "origin"},
			seen:     map[string]bool{},
			wantLen:  1,
			checkFn: func(t *testing.T, result []BranchInfo) {
				assert.Equal(t, "develop", result[0].Name)
				assert.True(t, result[0].IsRemote)
			},
		},
		"skip duplicate remote when local exists": {
			existing: []BranchInfo{
				{Name: "main", IsRemote: false},
			},
			info:    BranchInfo{Name: "main", IsRemote: true, Remote: "origin"},
			seen:    map[string]bool{"main": true},
			wantLen: 1,
			checkFn: func(t *testing.T, result []BranchInfo) {
				// Should still have local, not replaced
				assert.False(t, result[0].IsRemote)
			},
		},
		"replace remote with local": {
			existing: []BranchInfo{
				{Name: "feature", IsRemote: true, Remote: "origin"},
			},
			info:    BranchInfo{Name: "feature", IsRemote: false},
			seen:    map[string]bool{"feature": true},
			wantLen: 1,
			checkFn: func(t *testing.T, result []BranchInfo) {
				// Should be replaced with local
				assert.Equal(t, "feature", result[0].Name)
				assert.False(t, result[0].IsRemote)
			},
		},
		"add to non-empty list": {
			existing: []BranchInfo{
				{Name: "main", IsRemote: false},
			},
			info:    BranchInfo{Name: "develop", IsRemote: false},
			seen:    map[string]bool{"main": true},
			wantLen: 2,
			checkFn: func(t *testing.T, result []BranchInfo) {
				assert.Equal(t, 2, len(result))
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Make a copy of existing to avoid mutation
			existing := make([]BranchInfo, len(tt.existing))
			copy(existing, tt.existing)

			// Make a copy of seen map
			seen := make(map[string]bool)
			for k, v := range tt.seen {
				seen[k] = v
			}

			result := addBranchWithDedup(existing, tt.info, seen)
			assert.Equal(t, tt.wantLen, len(result))
			tt.checkFn(t, result)
		})
	}
}

func TestCollectBranches(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		lines   []string
		wantLen int
		checkFn func(t *testing.T, branches []BranchInfo)
	}{
		"empty lines": {
			lines:   []string{},
			wantLen: 0,
		},
		"single local branch": {
			lines:   []string{"main"},
			wantLen: 1,
			checkFn: func(t *testing.T, branches []BranchInfo) {
				assert.Equal(t, "main", branches[0].Name)
			},
		},
		"filters HEAD": {
			lines:   []string{"main", "HEAD", "origin/HEAD"},
			wantLen: 1,
			checkFn: func(t *testing.T, branches []BranchInfo) {
				assert.Equal(t, "main", branches[0].Name)
			},
		},
		"filters empty lines": {
			lines:   []string{"main", "", "  ", "develop"},
			wantLen: 2,
		},
		"deduplicates local over remote": {
			lines:   []string{"origin/main", "main"},
			wantLen: 1,
			checkFn: func(t *testing.T, branches []BranchInfo) {
				// Local should replace remote
				assert.Equal(t, "main", branches[0].Name)
				assert.False(t, branches[0].IsRemote)
			},
		},
		"multiple remotes same branch": {
			lines:   []string{"origin/feature", "upstream/feature"},
			wantLen: 1,
			checkFn: func(t *testing.T, branches []BranchInfo) {
				// First one wins
				assert.Equal(t, "feature", branches[0].Name)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			branches := collectBranches(tt.lines)
			assert.Equal(t, tt.wantLen, len(branches))
			if tt.checkFn != nil {
				tt.checkFn(t, branches)
			}
		})
	}
}

// TestCreateBranch_ExistingBranch tests that CreateBranch fails when branch exists
func TestCreateBranch_ExistingBranch(t *testing.T) {
	t.Parallel()

	// Test against the real repo - try to create an existing branch
	branches, err := GetBranchNames()
	require.NoError(t, err)

	if len(branches) > 0 {
		// Try to create an existing branch - should fail
		err = CreateBranch(branches[0])
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	}
}

// TestCreateBranch_InTempRepo tests CreateBranch in a temporary git repository
func TestCreateBranch_InTempRepo(t *testing.T) {
	t.Parallel()

	// Create a temp directory for our test git repo
	tmpDir := t.TempDir()

	// Save current directory and change to temp dir
	origDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	// Initialize a git repo
	cmd := exec.Command("git", "init")
	err = cmd.Run()
	require.NoError(t, err)

	// Configure git user for the test repo
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "config", "user.name", "Test User")
	err = cmd.Run()
	require.NoError(t, err)

	// Create an initial commit (needed for branches to work)
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	cmd = exec.Command("git", "add", ".")
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "initial commit")
	err = cmd.Run()
	require.NoError(t, err)

	tests := map[string]struct {
		branchName string
		wantErr    bool
		errContain string
	}{
		"create new branch": {
			branchName: "test-new-branch",
			wantErr:    false,
		},
		"create branch with numbers": {
			branchName: "feature-123-test",
			wantErr:    false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := CreateBranch(tt.branchName)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)

				// Verify branch was created
				branch, err := GetCurrentBranch()
				require.NoError(t, err)
				assert.Equal(t, tt.branchName, branch)

				// Switch back to main for next test
				cmd := exec.Command("git", "checkout", "master")
				if err := cmd.Run(); err != nil {
					// Try main instead of master
					cmd = exec.Command("git", "checkout", "main")
					_ = cmd.Run()
				}
			}
		})
	}

	// Test creating duplicate branch
	t.Run("duplicate branch fails", func(t *testing.T) {
		err := CreateBranch("test-new-branch")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

// TestCreateBranch_NotGitRepo tests CreateBranch fails outside a git repo
func TestCreateBranch_NotGitRepo(t *testing.T) {
	t.Parallel()

	// Create a temp directory that is NOT a git repo
	tmpDir := t.TempDir()

	// Save current directory and change to temp dir
	origDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	// Try to create a branch - should fail
	err = CreateBranch("test-branch")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
}
