package worktree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseWorktreeList(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input    string
		wantLen  int
		wantErr  bool
		validate func(*testing.T, []GitWorktreeEntry)
	}{
		"single worktree": {
			input: `worktree /home/user/repo
HEAD abc123
branch refs/heads/main
`,
			wantLen: 1,
			validate: func(t *testing.T, entries []GitWorktreeEntry) {
				assert.Equal(t, "/home/user/repo", entries[0].Path)
				assert.Equal(t, "abc123", entries[0].Commit)
				assert.Equal(t, "main", entries[0].Branch)
			},
		},
		"multiple worktrees": {
			input: `worktree /home/user/repo
HEAD abc123
branch refs/heads/main

worktree /home/user/repo-wt
HEAD def456
branch refs/heads/feature
`,
			wantLen: 2,
			validate: func(t *testing.T, entries []GitWorktreeEntry) {
				assert.Equal(t, "/home/user/repo", entries[0].Path)
				assert.Equal(t, "main", entries[0].Branch)
				assert.Equal(t, "/home/user/repo-wt", entries[1].Path)
				assert.Equal(t, "feature", entries[1].Branch)
			},
		},
		"detached head": {
			input: `worktree /home/user/repo
HEAD abc123
detached
`,
			wantLen: 1,
			validate: func(t *testing.T, entries []GitWorktreeEntry) {
				assert.Equal(t, "(detached)", entries[0].Branch)
			},
		},
		"bare repository": {
			input: `worktree /home/user/repo.git
bare
`,
			wantLen: 0,
		},
		"empty input": {
			input:   "",
			wantLen: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			entries, err := parseWorktreeList([]byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, entries, tt.wantLen)
			if tt.validate != nil && len(entries) > 0 {
				tt.validate(t, entries)
			}
		})
	}
}

func TestGitWorktreeEntry_Fields(t *testing.T) {
	t.Parallel()

	entry := GitWorktreeEntry{
		Path:   "/test/path",
		Commit: "abc123def",
		Branch: "feature/test",
	}

	assert.Equal(t, "/test/path", entry.Path)
	assert.Equal(t, "abc123def", entry.Commit)
	assert.Equal(t, "feature/test", entry.Branch)
}
