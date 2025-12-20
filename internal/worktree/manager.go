package worktree

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Manager defines the interface for worktree CRUD operations.
type Manager interface {
	// Create creates a new worktree with the given name and branch.
	Create(name, branch, customPath string) (*Worktree, error)
	// List returns all tracked worktrees.
	List() ([]Worktree, error)
	// Get returns a worktree by name.
	Get(name string) (*Worktree, error)
	// Remove removes a worktree by name.
	Remove(name string, force bool) error
	// Setup runs setup on an existing worktree path.
	Setup(path string, addToState bool) (*Worktree, error)
	// Prune removes stale worktree entries.
	Prune() (int, error)
	// UpdateStatus updates the status of a worktree.
	UpdateStatus(name string, status WorktreeStatus) error
}

// DefaultManager implements the Manager interface.
type DefaultManager struct {
	config     *WorktreeConfig
	stateDir   string
	repoRoot   string
	stdout     io.Writer
	gitOps     GitOperations
	copyFn     CopyFunc
	runSetupFn SetupFunc
}

// GitOperations defines the git operations used by the manager.
// This interface enables testing with mocks.
type GitOperations interface {
	Add(repoPath, worktreePath, branch string) error
	Remove(repoPath, worktreePath string, force bool) error
	List(repoPath string) ([]GitWorktreeEntry, error)
	HasUncommittedChanges(path string) (bool, error)
	HasUnpushedCommits(path string) (bool, error)
}

// CopyFunc is the function signature for directory copying.
type CopyFunc func(srcRoot, dstRoot string, dirs []string) ([]string, error)

// SetupFunc is the function signature for running setup scripts.
type SetupFunc func(scriptPath, worktreePath, name, branch, sourceRepo string, stdout io.Writer) *SetupResult

// defaultGitOps implements GitOperations using the real git commands.
type defaultGitOps struct{}

func (g *defaultGitOps) Add(repoPath, worktreePath, branch string) error {
	return GitWorktreeAdd(repoPath, worktreePath, branch)
}

func (g *defaultGitOps) Remove(repoPath, worktreePath string, force bool) error {
	return GitWorktreeRemove(repoPath, worktreePath, force)
}

func (g *defaultGitOps) List(repoPath string) ([]GitWorktreeEntry, error) {
	return GitWorktreeList(repoPath)
}

func (g *defaultGitOps) HasUncommittedChanges(path string) (bool, error) {
	return HasUncommittedChanges(path)
}

func (g *defaultGitOps) HasUnpushedCommits(path string) (bool, error) {
	return HasUnpushedCommits(path)
}

// ManagerOption configures a DefaultManager.
type ManagerOption func(*DefaultManager)

// WithStdout sets the stdout writer for manager output.
func WithStdout(w io.Writer) ManagerOption {
	return func(m *DefaultManager) {
		m.stdout = w
	}
}

// WithGitOps sets custom git operations (for testing).
func WithGitOps(ops GitOperations) ManagerOption {
	return func(m *DefaultManager) {
		m.gitOps = ops
	}
}

// WithCopyFunc sets a custom copy function (for testing).
func WithCopyFunc(fn CopyFunc) ManagerOption {
	return func(m *DefaultManager) {
		m.copyFn = fn
	}
}

// WithSetupFunc sets a custom setup function (for testing).
func WithSetupFunc(fn SetupFunc) ManagerOption {
	return func(m *DefaultManager) {
		m.runSetupFn = fn
	}
}

// NewManager creates a new DefaultManager.
func NewManager(config *WorktreeConfig, stateDir, repoRoot string, opts ...ManagerOption) *DefaultManager {
	if config == nil {
		config = DefaultConfig()
	}

	m := &DefaultManager{
		config:     config,
		stateDir:   stateDir,
		repoRoot:   repoRoot,
		stdout:     os.Stdout,
		gitOps:     &defaultGitOps{},
		copyFn:     CopyDirs,
		runSetupFn: RunSetupScript,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Create creates a new worktree with the given name and branch.
func (m *DefaultManager) Create(name, branch, customPath string) (*Worktree, error) {
	state, err := LoadState(m.stateDir)
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}

	if state.FindWorktree(name) != nil {
		return nil, fmt.Errorf("worktree %q already exists", name)
	}

	worktreePath := m.resolveWorktreePath(name, customPath)

	if err := m.gitOps.Add(m.repoRoot, worktreePath, branch); err != nil {
		return nil, fmt.Errorf("creating git worktree: %w", err)
	}

	copied, err := m.copyFn(m.repoRoot, worktreePath, m.config.CopyDirs)
	if err != nil {
		fmt.Fprintf(m.stdout, "Warning: failed to copy directories: %v\n", err)
	} else if len(copied) > 0 {
		fmt.Fprintf(m.stdout, "Copied directories: %v\n", copied)
	}

	setupCompleted := m.runSetupIfConfigured(worktreePath, name, branch)

	wt := Worktree{
		Name:           name,
		Path:           worktreePath,
		Branch:         branch,
		Status:         StatusActive,
		CreatedAt:      time.Now(),
		SetupCompleted: setupCompleted,
		LastAccessed:   time.Now(),
	}

	if m.config.TrackStatus {
		if err := state.AddWorktree(wt); err != nil {
			return nil, fmt.Errorf("adding to state: %w", err)
		}
		if err := SaveState(m.stateDir, state); err != nil {
			return nil, fmt.Errorf("saving state: %w", err)
		}
	}

	return &wt, nil
}

// resolveWorktreePath determines the path for a new worktree.
func (m *DefaultManager) resolveWorktreePath(name, customPath string) string {
	if customPath != "" {
		if filepath.IsAbs(customPath) {
			return customPath
		}
		return filepath.Join(m.repoRoot, customPath)
	}

	baseDir := m.config.BaseDir
	if baseDir == "" {
		baseDir = filepath.Dir(m.repoRoot)
	}

	dirName := m.config.Prefix + name
	return filepath.Join(baseDir, dirName)
}

// runSetupIfConfigured runs the setup script if configured.
func (m *DefaultManager) runSetupIfConfigured(worktreePath, name, branch string) bool {
	if !m.config.AutoSetup || m.config.SetupScript == "" {
		return true // No setup needed, consider completed
	}

	result := m.runSetupFn(m.config.SetupScript, worktreePath, name, branch, m.repoRoot, m.stdout)

	if !result.Executed {
		return true // Script didn't exist, consider completed
	}

	if result.Error != nil {
		fmt.Fprintf(m.stdout, "Warning: setup script failed: %v\n", result.Error)
		return false
	}

	return true
}

// List returns all tracked worktrees.
func (m *DefaultManager) List() ([]Worktree, error) {
	state, err := LoadState(m.stateDir)
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}

	// Update stale status for worktrees where path doesn't exist
	for i := range state.Worktrees {
		if _, err := os.Stat(state.Worktrees[i].Path); os.IsNotExist(err) {
			state.Worktrees[i].Status = StatusStale
		}
	}

	return state.Worktrees, nil
}

// Get returns a worktree by name.
func (m *DefaultManager) Get(name string) (*Worktree, error) {
	state, err := LoadState(m.stateDir)
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}

	wt := state.FindWorktree(name)
	if wt == nil {
		return nil, fmt.Errorf("worktree %q not found", name)
	}

	return wt, nil
}

// Remove removes a worktree by name.
func (m *DefaultManager) Remove(name string, force bool) error {
	state, err := LoadState(m.stateDir)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	wt := state.FindWorktree(name)
	if wt == nil {
		return fmt.Errorf("worktree %q not found", name)
	}

	if !force {
		if err := m.checkSafeToRemove(wt.Path); err != nil {
			return err
		}
	}

	if err := m.gitOps.Remove(m.repoRoot, wt.Path, force); err != nil {
		return fmt.Errorf("removing git worktree: %w", err)
	}

	state.RemoveWorktree(name)

	if err := SaveState(m.stateDir, state); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	return nil
}

// checkSafeToRemove checks if it's safe to remove a worktree.
func (m *DefaultManager) checkSafeToRemove(path string) error {
	// Skip check if path doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	hasChanges, err := m.gitOps.HasUncommittedChanges(path)
	if err != nil {
		return fmt.Errorf("checking uncommitted changes: %w", err)
	}
	if hasChanges {
		return fmt.Errorf("worktree has uncommitted changes (use --force to override)")
	}

	hasUnpushed, err := m.gitOps.HasUnpushedCommits(path)
	if err != nil {
		return fmt.Errorf("checking unpushed commits: %w", err)
	}
	if hasUnpushed {
		return fmt.Errorf("worktree has unpushed commits (use --force to override)")
	}

	return nil
}

// Setup runs setup on an existing worktree path.
func (m *DefaultManager) Setup(path string, addToState bool) (*Worktree, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("path does not exist: %s", absPath)
	}

	isWT, err := IsWorktree(absPath)
	if err != nil {
		return nil, fmt.Errorf("checking if path is worktree: %w", err)
	}
	if !isWT {
		return nil, fmt.Errorf("path is not a git worktree: %s", absPath)
	}

	// Copy directories
	copied, err := m.copyFn(m.repoRoot, absPath, m.config.CopyDirs)
	if err != nil {
		fmt.Fprintf(m.stdout, "Warning: failed to copy directories: %v\n", err)
	} else if len(copied) > 0 {
		fmt.Fprintf(m.stdout, "Copied directories: %v\n", copied)
	}

	// Derive name and branch from path
	name := filepath.Base(absPath)
	branch := m.getBranchForPath(absPath)

	setupCompleted := m.runSetupIfConfigured(absPath, name, branch)

	wt := Worktree{
		Name:           name,
		Path:           absPath,
		Branch:         branch,
		Status:         StatusActive,
		CreatedAt:      time.Now(),
		SetupCompleted: setupCompleted,
		LastAccessed:   time.Now(),
	}

	if addToState && m.config.TrackStatus {
		state, err := LoadState(m.stateDir)
		if err != nil {
			return nil, fmt.Errorf("loading state: %w", err)
		}

		if err := state.AddWorktree(wt); err != nil {
			return nil, fmt.Errorf("adding to state: %w", err)
		}

		if err := SaveState(m.stateDir, state); err != nil {
			return nil, fmt.Errorf("saving state: %w", err)
		}
	}

	return &wt, nil
}

// getBranchForPath gets the branch checked out in a worktree.
func (m *DefaultManager) getBranchForPath(path string) string {
	entries, err := m.gitOps.List(m.repoRoot)
	if err != nil {
		return "unknown"
	}

	for _, entry := range entries {
		if entry.Path == path {
			return entry.Branch
		}
	}

	return "unknown"
}

// Prune removes stale worktree entries.
func (m *DefaultManager) Prune() (int, error) {
	state, err := LoadState(m.stateDir)
	if err != nil {
		return 0, fmt.Errorf("loading state: %w", err)
	}

	var remaining []Worktree
	var pruned int

	for _, wt := range state.Worktrees {
		if _, err := os.Stat(wt.Path); os.IsNotExist(err) {
			pruned++
			continue
		}
		remaining = append(remaining, wt)
	}

	if pruned > 0 {
		state.Worktrees = remaining
		if err := SaveState(m.stateDir, state); err != nil {
			return 0, fmt.Errorf("saving state: %w", err)
		}
	}

	return pruned, nil
}

// UpdateStatus updates the status of a worktree.
func (m *DefaultManager) UpdateStatus(name string, status WorktreeStatus) error {
	if !status.IsValid() {
		return fmt.Errorf("invalid status: %s", status)
	}

	state, err := LoadState(m.stateDir)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	wt := state.FindWorktree(name)
	if wt == nil {
		return fmt.Errorf("worktree %q not found", name)
	}

	wt.Status = status
	wt.LastAccessed = time.Now()

	if status == StatusMerged {
		now := time.Now()
		wt.MergedAt = &now
	}

	if err := state.UpdateWorktree(*wt); err != nil {
		return fmt.Errorf("updating worktree: %w", err)
	}

	if err := SaveState(m.stateDir, state); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	return nil
}
