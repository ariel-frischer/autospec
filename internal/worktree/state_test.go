package worktree

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadState_FileExists(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, StateFileName)

	content := `version: "1.0.0"
worktrees:
  - name: test-wt
    path: /tmp/test
    branch: main
    status: active
    setup_completed: true
`
	require.NoError(t, os.WriteFile(stateFile, []byte(content), 0644))

	state, err := LoadState(tempDir)
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", state.Version)
	require.Len(t, state.Worktrees, 1)
	assert.Equal(t, "test-wt", state.Worktrees[0].Name)
	assert.Equal(t, "/tmp/test", state.Worktrees[0].Path)
	assert.Equal(t, StatusActive, state.Worktrees[0].Status)
}

func TestLoadState_FileMissing(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	state, err := LoadState(tempDir)
	require.NoError(t, err)
	assert.Equal(t, StateVersion, state.Version)
	assert.Empty(t, state.Worktrees)
}

func TestLoadState_CorruptedYAML(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, StateFileName)

	content := `invalid: yaml: content: [[[`
	require.NoError(t, os.WriteFile(stateFile, []byte(content), 0644))

	_, err := LoadState(tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing state file")
}

func TestSaveState_CreatesDirectory(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "nested", "state")

	state := &WorktreeState{
		Version:   "1.0.0",
		Worktrees: []Worktree{},
	}

	err := SaveState(stateDir, state)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(stateDir, StateFileName))
	assert.NoError(t, err)
}

func TestSaveState_AtomicWrite(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	state := &WorktreeState{
		Version: "1.0.0",
		Worktrees: []Worktree{
			{Name: "wt1", Path: "/path/1", Branch: "main", Status: StatusActive},
		},
	}

	err := SaveState(tempDir, state)
	require.NoError(t, err)

	// Verify temp file doesn't exist
	tmpFile := filepath.Join(tempDir, StateFileName+".tmp")
	_, err = os.Stat(tmpFile)
	assert.True(t, os.IsNotExist(err))

	// Verify state file exists
	stateFile := filepath.Join(tempDir, StateFileName)
	_, err = os.Stat(stateFile)
	assert.NoError(t, err)
}

func TestSaveState_UpdatesVersion(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	state := &WorktreeState{
		Version:   "old-version",
		Worktrees: []Worktree{},
	}

	err := SaveState(tempDir, state)
	require.NoError(t, err)

	loaded, err := LoadState(tempDir)
	require.NoError(t, err)
	assert.Equal(t, StateVersion, loaded.Version)
}

func TestWorkflowState_FindWorktree(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		worktrees []Worktree
		name      string
		wantNil   bool
		wantName  string
	}{
		"found": {
			worktrees: []Worktree{{Name: "wt1"}, {Name: "wt2"}},
			name:      "wt1",
			wantNil:   false,
			wantName:  "wt1",
		},
		"not found": {
			worktrees: []Worktree{{Name: "wt1"}},
			name:      "missing",
			wantNil:   true,
		},
		"empty list": {
			worktrees: []Worktree{},
			name:      "any",
			wantNil:   true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			state := &WorktreeState{Worktrees: tt.worktrees}
			got := state.FindWorktree(tt.name)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, tt.wantName, got.Name)
			}
		})
	}
}

func TestWorktreeState_AddWorktree(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		existing []Worktree
		add      Worktree
		wantErr  bool
	}{
		"add to empty": {
			existing: []Worktree{},
			add:      Worktree{Name: "new"},
			wantErr:  false,
		},
		"add new": {
			existing: []Worktree{{Name: "wt1"}},
			add:      Worktree{Name: "wt2"},
			wantErr:  false,
		},
		"duplicate name": {
			existing: []Worktree{{Name: "wt1"}},
			add:      Worktree{Name: "wt1"},
			wantErr:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			state := &WorktreeState{Worktrees: tt.existing}
			err := state.AddWorktree(tt.add)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, state.FindWorktree(tt.add.Name))
			}
		})
	}
}

func TestWorktreeState_RemoveWorktree(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		existing []Worktree
		remove   string
		want     bool
	}{
		"remove existing": {
			existing: []Worktree{{Name: "wt1"}, {Name: "wt2"}},
			remove:   "wt1",
			want:     true,
		},
		"remove non-existing": {
			existing: []Worktree{{Name: "wt1"}},
			remove:   "missing",
			want:     false,
		},
		"remove from empty": {
			existing: []Worktree{},
			remove:   "any",
			want:     false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			state := &WorktreeState{Worktrees: tt.existing}
			got := state.RemoveWorktree(tt.remove)
			assert.Equal(t, tt.want, got)
			if tt.want {
				assert.Nil(t, state.FindWorktree(tt.remove))
			}
		})
	}
}

func TestWorktreeState_UpdateWorktree(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		existing []Worktree
		update   Worktree
		wantErr  bool
	}{
		"update existing": {
			existing: []Worktree{{Name: "wt1", Status: StatusActive}},
			update:   Worktree{Name: "wt1", Status: StatusMerged},
			wantErr:  false,
		},
		"update non-existing": {
			existing: []Worktree{{Name: "wt1"}},
			update:   Worktree{Name: "missing"},
			wantErr:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			state := &WorktreeState{Worktrees: tt.existing}
			err := state.UpdateWorktree(tt.update)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				wt := state.FindWorktree(tt.update.Name)
				require.NotNil(t, wt)
				assert.Equal(t, tt.update.Status, wt.Status)
			}
		})
	}
}

func TestLoadSaveState_Roundtrip(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	now := time.Now().Truncate(time.Second)

	original := &WorktreeState{
		Version: StateVersion,
		Worktrees: []Worktree{
			{
				Name:           "test-wt",
				Path:           "/tmp/test",
				Branch:         "feature/test",
				Status:         StatusActive,
				CreatedAt:      now,
				SetupCompleted: true,
				LastAccessed:   now,
			},
		},
	}

	err := SaveState(tempDir, original)
	require.NoError(t, err)

	loaded, err := LoadState(tempDir)
	require.NoError(t, err)

	assert.Equal(t, original.Version, loaded.Version)
	require.Len(t, loaded.Worktrees, 1)
	assert.Equal(t, original.Worktrees[0].Name, loaded.Worktrees[0].Name)
	assert.Equal(t, original.Worktrees[0].Status, loaded.Worktrees[0].Status)
}
