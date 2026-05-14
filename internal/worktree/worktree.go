// Package worktree provides git worktree management with project-aware setup automation.
// It automates worktree creation with automatic copying of non-tracked directories
// (.autospec/, .claude/) and execution of project-specific setup scripts.
package worktree

import (
	"time"
)

// WorktreeStatus represents the current state of a worktree.
type WorktreeStatus string

const (
	// StatusActive indicates the worktree is in active use.
	StatusActive WorktreeStatus = "active"
	// StatusMerged indicates the worktree's branch has been merged.
	StatusMerged WorktreeStatus = "merged"
	// StatusAbandoned indicates work was abandoned.
	StatusAbandoned WorktreeStatus = "abandoned"
	// StatusStale indicates the worktree path no longer exists.
	StatusStale WorktreeStatus = "stale"
)

// String returns the string representation of the status.
func (s WorktreeStatus) String() string {
	return string(s)
}

// IsValid returns true if the status is a recognized value.
func (s WorktreeStatus) IsValid() bool {
	switch s {
	case StatusActive, StatusMerged, StatusAbandoned, StatusStale:
		return true
	default:
		return false
	}
}

// Worktree represents a git worktree with tracking metadata.
type Worktree struct {
	// Name is the unique identifier for the worktree.
	Name string `yaml:"name"`
	// Path is the absolute filesystem path to the worktree.
	Path string `yaml:"path"`
	// Branch is the git branch checked out in the worktree.
	Branch string `yaml:"branch"`
	// Status is the current state of the worktree.
	Status WorktreeStatus `yaml:"status"`
	// CreatedAt is the timestamp when the worktree was created.
	CreatedAt time.Time `yaml:"created_at"`
	// SetupCompleted indicates whether the setup script ran successfully.
	SetupCompleted bool `yaml:"setup_completed"`
	// LastAccessed is the timestamp of last access (optional).
	LastAccessed time.Time `yaml:"last_accessed,omitempty"`
	// MergedAt is the timestamp when the branch was merged (nil if not merged).
	MergedAt *time.Time `yaml:"merged_at,omitempty"`
}

// WorktreeState is the container for all tracked worktrees persisted to YAML.
type WorktreeState struct {
	// Version is the schema version for the state file.
	Version string `yaml:"version"`
	// Worktrees is the list of tracked worktree entries.
	Worktrees []Worktree `yaml:"worktrees"`
}

// WorktreeConfig holds configuration for worktree management.
type WorktreeConfig struct {
	// BaseDir is the parent directory for new worktrees (default: parent of repo root).
	BaseDir string `yaml:"base_dir,omitempty" koanf:"base_dir"`
	// Prefix is the directory name prefix (default: empty string).
	Prefix string `yaml:"prefix,omitempty" koanf:"prefix"`
	// SetupScript is the path to the setup script relative to repo (optional).
	SetupScript string `yaml:"setup_script,omitempty" koanf:"setup_script"`
	// AutoSetup determines whether to run setup automatically on create (default: true).
	AutoSetup bool `yaml:"auto_setup" koanf:"auto_setup"`
	// TrackStatus determines whether to persist worktree state (default: true).
	TrackStatus bool `yaml:"track_status" koanf:"track_status"`
	// CopyDirs lists non-tracked directories to copy.
	CopyDirs []string `yaml:"copy_dirs,omitempty" koanf:"copy_dirs"`
	// SetupTimeout is the maximum duration for setup script execution (default: 5m).
	SetupTimeout time.Duration `yaml:"setup_timeout,omitempty" koanf:"setup_timeout"`
}

// CreateOptions controls behavior during worktree creation.
type CreateOptions struct {
	// SkipCopy prevents copying directories listed in copy_dirs to the new worktree.
	SkipCopy bool
	// SkipSetup prevents running the setup script after worktree creation.
	SkipSetup bool
	// NoRollback preserves the worktree on setup or validation failure for debugging.
	NoRollback bool
	// StartPoint is the git commit, branch, or tag to base the new worktree on.
	// If empty, the worktree is created from HEAD.
	// Used for layer staging to branch worktrees from staging branches.
	StartPoint string
}

// DefaultConfig returns a WorktreeConfig with default values.
func DefaultConfig() *WorktreeConfig {
	return &WorktreeConfig{
		BaseDir:      "",
		Prefix:       "",
		SetupScript:  "",
		AutoSetup:    true,
		TrackStatus:  true,
		CopyDirs:     []string{".autospec", ".agents", ".claude", ".codex", ".opencode"},
		SetupTimeout: 5 * time.Minute,
	}
}
