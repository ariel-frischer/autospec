package worktree

import (
	"bytes"
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGitOps implements GitOperations for testing
type mockGitOps struct {
	addCalled      bool
	addErr         error
	removeCalled   bool
	removeErr      error
	listResult     []GitWorktreeEntry
	listErr        error
	uncommitted    bool
	uncommittedErr error
	unpushed       bool
	unpushedErr    error
}

func (m *mockGitOps) Add(repoPath, worktreePath, branch string) error {
	m.addCalled = true
	return m.addErr
}

func (m *mockGitOps) Remove(repoPath, worktreePath string, force bool) error {
	m.removeCalled = true
	return m.removeErr
}

func (m *mockGitOps) List(repoPath string) ([]GitWorktreeEntry, error) {
	return m.listResult, m.listErr
}

func (m *mockGitOps) HasUncommittedChanges(path string) (bool, error) {
	return m.uncommitted, m.uncommittedErr
}

func (m *mockGitOps) HasUnpushedCommits(path string) (bool, error) {
	return m.unpushed, m.unpushedErr
}

func TestNewManager_DefaultConfig(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil, "/state", "/repo")

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.config)
	assert.True(t, manager.config.AutoSetup)
	assert.True(t, manager.config.TrackStatus)
}

func TestNewManager_WithOptions(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	mockOps := &mockGitOps{}
	copyCalled := false
	setupCalled := false

	manager := NewManager(
		DefaultConfig(),
		"/state",
		"/repo",
		WithStdout(&buf),
		WithGitOps(mockOps),
		WithCopyFunc(func(src, dst string, dirs []string) ([]string, error) {
			copyCalled = true
			return dirs, nil
		}),
		WithSetupFunc(func(script, path, name, branch, repo string, w io.Writer) *SetupResult {
			setupCalled = true
			return &SetupResult{Executed: false}
		}),
	)

	assert.NotNil(t, manager)
	// Verify options were set by triggering behaviors
	_, _ = manager.copyFn("", "", nil)
	assert.True(t, copyCalled)
	_ = manager.runSetupFn("", "", "", "", "", nil)
	assert.True(t, setupCalled)
}

func TestManager_Create_Success(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	repoRoot := t.TempDir()
	baseDir := t.TempDir()

	mockOps := &mockGitOps{}
	var buf bytes.Buffer

	cfg := &WorktreeConfig{
		BaseDir:     baseDir,
		Prefix:      "wt-",
		AutoSetup:   false,
		TrackStatus: true,
		CopyDirs:    []string{},
	}

	manager := NewManager(cfg, stateDir, repoRoot,
		WithStdout(&buf),
		WithGitOps(mockOps),
		WithCopyFunc(func(src, dst string, dirs []string) ([]string, error) {
			return nil, nil
		}),
	)

	wt, err := manager.Create("test", "feature/test", "")
	require.NoError(t, err)
	assert.True(t, mockOps.addCalled)
	assert.Equal(t, "test", wt.Name)
	assert.Equal(t, "feature/test", wt.Branch)
	assert.Equal(t, StatusActive, wt.Status)
	assert.Equal(t, filepath.Join(baseDir, "wt-test"), wt.Path)
}

func TestManager_Create_DuplicateName(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	// Pre-populate state
	state := &WorktreeState{
		Version:   StateVersion,
		Worktrees: []Worktree{{Name: "existing"}},
	}
	require.NoError(t, SaveState(stateDir, state))

	manager := NewManager(DefaultConfig(), stateDir, "/repo")

	_, err := manager.Create("existing", "branch", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestManager_Create_CustomPath(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	repoRoot := t.TempDir()
	customPath := filepath.Join(t.TempDir(), "custom-location")

	mockOps := &mockGitOps{}
	var buf bytes.Buffer

	cfg := &WorktreeConfig{
		AutoSetup:   false,
		TrackStatus: true,
		CopyDirs:    []string{},
	}

	manager := NewManager(cfg, stateDir, repoRoot,
		WithStdout(&buf),
		WithGitOps(mockOps),
		WithCopyFunc(func(src, dst string, dirs []string) ([]string, error) {
			return nil, nil
		}),
	)

	wt, err := manager.Create("test", "branch", customPath)
	require.NoError(t, err)
	assert.Equal(t, customPath, wt.Path)
}

func TestManager_List_Empty(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	manager := NewManager(DefaultConfig(), stateDir, "/repo")

	worktrees, err := manager.List()
	require.NoError(t, err)
	assert.Empty(t, worktrees)
}

func TestManager_List_WithWorktrees(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	existingPath := t.TempDir() // This path exists

	state := &WorktreeState{
		Version: StateVersion,
		Worktrees: []Worktree{
			{Name: "wt1", Path: existingPath, Status: StatusActive},
			{Name: "wt2", Path: "/nonexistent/path", Status: StatusActive},
		},
	}
	require.NoError(t, SaveState(stateDir, state))

	manager := NewManager(DefaultConfig(), stateDir, "/repo")

	worktrees, err := manager.List()
	require.NoError(t, err)
	require.Len(t, worktrees, 2)

	// Existing path keeps status
	assert.Equal(t, StatusActive, worktrees[0].Status)
	// Non-existing path becomes stale
	assert.Equal(t, StatusStale, worktrees[1].Status)
}

func TestManager_Get_Found(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	state := &WorktreeState{
		Version:   StateVersion,
		Worktrees: []Worktree{{Name: "test", Path: "/path"}},
	}
	require.NoError(t, SaveState(stateDir, state))

	manager := NewManager(DefaultConfig(), stateDir, "/repo")

	wt, err := manager.Get("test")
	require.NoError(t, err)
	assert.Equal(t, "test", wt.Name)
}

func TestManager_Get_NotFound(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	manager := NewManager(DefaultConfig(), stateDir, "/repo")

	_, err := manager.Get("missing")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_Remove_Success(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	wtPath := t.TempDir() // Existing path

	state := &WorktreeState{
		Version:   StateVersion,
		Worktrees: []Worktree{{Name: "test", Path: wtPath}},
	}
	require.NoError(t, SaveState(stateDir, state))

	mockOps := &mockGitOps{
		uncommitted: false,
		unpushed:    false,
	}
	var buf bytes.Buffer

	manager := NewManager(DefaultConfig(), stateDir, "/repo",
		WithStdout(&buf),
		WithGitOps(mockOps),
	)

	err := manager.Remove("test", false)
	require.NoError(t, err)
	assert.True(t, mockOps.removeCalled)

	// Verify removed from state
	loaded, _ := LoadState(stateDir)
	assert.Nil(t, loaded.FindWorktree("test"))
}

func TestManager_Remove_WithUncommittedChanges(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	wtPath := t.TempDir()

	state := &WorktreeState{
		Version:   StateVersion,
		Worktrees: []Worktree{{Name: "test", Path: wtPath}},
	}
	require.NoError(t, SaveState(stateDir, state))

	mockOps := &mockGitOps{
		uncommitted: true,
	}

	manager := NewManager(DefaultConfig(), stateDir, "/repo",
		WithGitOps(mockOps),
	)

	err := manager.Remove("test", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "uncommitted changes")
}

func TestManager_Remove_ForceBypassesChecks(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	wtPath := t.TempDir()

	state := &WorktreeState{
		Version:   StateVersion,
		Worktrees: []Worktree{{Name: "test", Path: wtPath}},
	}
	require.NoError(t, SaveState(stateDir, state))

	mockOps := &mockGitOps{
		uncommitted: true,
		unpushed:    true,
	}
	var buf bytes.Buffer

	manager := NewManager(DefaultConfig(), stateDir, "/repo",
		WithStdout(&buf),
		WithGitOps(mockOps),
	)

	err := manager.Remove("test", true)
	require.NoError(t, err)
	assert.True(t, mockOps.removeCalled)
}

func TestManager_Prune_RemovesStale(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	existingPath := t.TempDir()

	state := &WorktreeState{
		Version: StateVersion,
		Worktrees: []Worktree{
			{Name: "exists", Path: existingPath},
			{Name: "stale1", Path: "/nonexistent1"},
			{Name: "stale2", Path: "/nonexistent2"},
		},
	}
	require.NoError(t, SaveState(stateDir, state))

	manager := NewManager(DefaultConfig(), stateDir, "/repo")

	pruned, err := manager.Prune()
	require.NoError(t, err)
	assert.Equal(t, 2, pruned)

	// Verify only existing remains
	loaded, _ := LoadState(stateDir)
	assert.Len(t, loaded.Worktrees, 1)
	assert.Equal(t, "exists", loaded.Worktrees[0].Name)
}

func TestManager_Prune_NoneToRemove(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	existingPath := t.TempDir()

	state := &WorktreeState{
		Version:   StateVersion,
		Worktrees: []Worktree{{Name: "exists", Path: existingPath}},
	}
	require.NoError(t, SaveState(stateDir, state))

	manager := NewManager(DefaultConfig(), stateDir, "/repo")

	pruned, err := manager.Prune()
	require.NoError(t, err)
	assert.Equal(t, 0, pruned)
}

func TestManager_UpdateStatus(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status  WorktreeStatus
		wantErr bool
	}{
		"update to merged": {
			status:  StatusMerged,
			wantErr: false,
		},
		"update to abandoned": {
			status:  StatusAbandoned,
			wantErr: false,
		},
		"invalid status": {
			status:  WorktreeStatus("invalid"),
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()
			state := &WorktreeState{
				Version:   StateVersion,
				Worktrees: []Worktree{{Name: "test", Status: StatusActive}},
			}
			require.NoError(t, SaveState(stateDir, state))

			manager := NewManager(DefaultConfig(), stateDir, "/repo")

			err := manager.UpdateStatus("test", tt.status)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				loaded, _ := LoadState(stateDir)
				assert.Equal(t, tt.status, loaded.Worktrees[0].Status)
			}
		})
	}
}

func TestManager_UpdateStatus_NotFound(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	manager := NewManager(DefaultConfig(), stateDir, "/repo")

	err := manager.UpdateStatus("missing", StatusMerged)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestManager_Setup_ExistingPath requires an actual git worktree to test,
// which is difficult to set up in isolation. The Setup method calls IsWorktree
// which executes real git commands. Integration tests would cover this scenario.
// For unit testing, we verify the error case works correctly.

func TestManager_Setup_PathNotExist(t *testing.T) {
	t.Parallel()

	manager := NewManager(DefaultConfig(), t.TempDir(), "/repo")

	_, err := manager.Setup("/nonexistent/path", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}
