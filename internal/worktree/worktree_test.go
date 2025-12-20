package worktree

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestWorktreeStatus_String(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status WorktreeStatus
		want   string
	}{
		"active":    {status: StatusActive, want: "active"},
		"merged":    {status: StatusMerged, want: "merged"},
		"abandoned": {status: StatusAbandoned, want: "abandoned"},
		"stale":     {status: StatusStale, want: "stale"},
		"custom":    {status: WorktreeStatus("custom"), want: "custom"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := tt.status.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWorktreeStatus_IsValid(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status WorktreeStatus
		want   bool
	}{
		"active is valid":    {status: StatusActive, want: true},
		"merged is valid":    {status: StatusMerged, want: true},
		"abandoned is valid": {status: StatusAbandoned, want: true},
		"stale is valid":     {status: StatusStale, want: true},
		"empty is invalid":   {status: WorktreeStatus(""), want: false},
		"random is invalid":  {status: WorktreeStatus("random"), want: false},
		"ACTIVE is invalid":  {status: WorktreeStatus("ACTIVE"), want: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := tt.status.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWorktree_YAMLMarshal(t *testing.T) {
	t.Parallel()

	wt := Worktree{
		Name:           "test-worktree",
		Path:           "/tmp/test",
		Branch:         "feature/test",
		Status:         StatusActive,
		SetupCompleted: true,
	}

	data, err := yaml.Marshal(&wt)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "name: test-worktree")
	assert.Contains(t, string(data), "path: /tmp/test")
	assert.Contains(t, string(data), "branch: feature/test")
	assert.Contains(t, string(data), "status: active")
	assert.Contains(t, string(data), "setup_completed: true")
}

func TestWorktree_YAMLUnmarshal(t *testing.T) {
	t.Parallel()

	yamlData := `
name: test-worktree
path: /tmp/test
branch: feature/test
status: merged
setup_completed: false
`

	var wt Worktree
	err := yaml.Unmarshal([]byte(yamlData), &wt)
	assert.NoError(t, err)
	assert.Equal(t, "test-worktree", wt.Name)
	assert.Equal(t, "/tmp/test", wt.Path)
	assert.Equal(t, "feature/test", wt.Branch)
	assert.Equal(t, StatusMerged, wt.Status)
	assert.False(t, wt.SetupCompleted)
}

func TestWorktreeState_YAMLRoundtrip(t *testing.T) {
	t.Parallel()

	state := WorktreeState{
		Version: "1.0.0",
		Worktrees: []Worktree{
			{Name: "wt1", Path: "/path/1", Branch: "branch1", Status: StatusActive},
			{Name: "wt2", Path: "/path/2", Branch: "branch2", Status: StatusMerged},
		},
	}

	data, err := yaml.Marshal(&state)
	assert.NoError(t, err)

	var decoded WorktreeState
	err = yaml.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, state.Version, decoded.Version)
	assert.Len(t, decoded.Worktrees, 2)
	assert.Equal(t, "wt1", decoded.Worktrees[0].Name)
	assert.Equal(t, "wt2", decoded.Worktrees[1].Name)
}

func TestWorktreeConfig_Defaults(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	assert.Equal(t, "", cfg.BaseDir)
	assert.Equal(t, "", cfg.Prefix)
	assert.Equal(t, "", cfg.SetupScript)
	assert.True(t, cfg.AutoSetup)
	assert.True(t, cfg.TrackStatus)
	assert.Equal(t, []string{".autospec", ".claude"}, cfg.CopyDirs)
}

func TestWorktreeConfig_YAMLMarshal(t *testing.T) {
	t.Parallel()

	cfg := &WorktreeConfig{
		BaseDir:     "/custom/base",
		Prefix:      "wt-",
		SetupScript: "scripts/setup.sh",
		AutoSetup:   false,
		TrackStatus: true,
		CopyDirs:    []string{".config", ".env"},
	}

	data, err := yaml.Marshal(cfg)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "base_dir: /custom/base")
	assert.Contains(t, string(data), "prefix: wt-")
	assert.Contains(t, string(data), "setup_script: scripts/setup.sh")
	assert.Contains(t, string(data), "auto_setup: false")
}
