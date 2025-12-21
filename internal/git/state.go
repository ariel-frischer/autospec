// Package git provides Git repository utilities for autospec including branch detection,
// repository validation, and state capture for the auto-commit feature.
package git

import (
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage"
)

// GitState represents captured git repository state for before/after comparison
type GitState struct {
	CommitSHA  string    // Current HEAD commit SHA (40-character hex string)
	BranchName string    // Current branch name, or "detached" if in detached HEAD state
	CapturedAt time.Time // Timestamp when state was captured
}

// StateWarning represents a warning generated during state comparison
type StateWarning struct {
	Level   string // "warning" or "serious"
	Message string
}

// Opener abstracts the method of opening a git repository
// This allows for dependency injection in tests
type Opener interface {
	// Open opens a git repository at the given path
	Open(path string) (Repository, error)
}

// Repository abstracts go-git repository operations for testing
type Repository interface {
	// Head returns the reference where HEAD is pointing to
	Head() (*plumbing.Reference, error)
}

// DefaultOpener implements Opener using go-git's PlainOpen
type DefaultOpener struct{}

// Open opens a git repository at the given path using go-git
func (d *DefaultOpener) Open(path string) (Repository, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("opening repository: %w", err)
	}
	return &goGitRepository{repo: repo}, nil
}

// goGitRepository wraps go-git's Repository to implement our Repository interface
type goGitRepository struct {
	repo *git.Repository
}

// Head returns the reference where HEAD is pointing to
func (r *goGitRepository) Head() (*plumbing.Reference, error) {
	return r.repo.Head()
}

// InMemoryOpener implements Opener for testing with in-memory storage
type InMemoryOpener struct {
	storage storage.Storer
	repo    *git.Repository
}

// NewInMemoryOpener creates an Opener that uses the provided in-memory storage
func NewInMemoryOpener(store storage.Storer, repo *git.Repository) *InMemoryOpener {
	return &InMemoryOpener{storage: store, repo: repo}
}

// Open returns the pre-configured in-memory repository
func (i *InMemoryOpener) Open(_ string) (Repository, error) {
	if i.repo == nil {
		return nil, fmt.Errorf("no repository configured")
	}
	return &goGitRepository{repo: i.repo}, nil
}

// CaptureGitState captures the current git repository state using the provided opener
func CaptureGitState(opener Opener, repoPath string) (*GitState, error) {
	repo, err := opener.Open(repoPath)
	if err != nil {
		return nil, fmt.Errorf("opening git repository at %s: %w", repoPath, err)
	}

	return captureStateFromRepo(repo)
}

// captureStateFromRepo extracts state from an opened repository
func captureStateFromRepo(repo Repository) (*GitState, error) {
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("getting HEAD reference: %w", err)
	}

	state := &GitState{
		CommitSHA:  head.Hash().String(),
		CapturedAt: time.Now().UTC(),
	}

	// Check if we're on a branch or in detached HEAD state
	if head.Name().IsBranch() {
		state.BranchName = head.Name().Short()
	} else {
		state.BranchName = "detached"
	}

	return state, nil
}

// CompareGitStates compares initial and final git states and returns any warnings
// Warnings are returned for: branch changes (serious), no new commit when expected (warning)
func CompareGitStates(initial, final *GitState) []StateWarning {
	if initial == nil || final == nil {
		return nil
	}

	var warnings []StateWarning

	// Check for branch change (serious warning)
	if initial.BranchName != final.BranchName {
		warnings = append(warnings, StateWarning{
			Level: "serious",
			Message: fmt.Sprintf(
				"branch changed during workflow: %s -> %s",
				initial.BranchName, final.BranchName,
			),
		})
	}

	// Check for no new commit (warning - only if on same branch)
	if initial.BranchName == final.BranchName && initial.CommitSHA == final.CommitSHA {
		warnings = append(warnings, StateWarning{
			Level:   "warning",
			Message: "no new commit was created during workflow",
		})
	}

	return warnings
}

// CaptureGitStateWithDefaultOpener is a convenience function that uses DefaultOpener
func CaptureGitStateWithDefaultOpener(repoPath string) (*GitState, error) {
	return CaptureGitState(&DefaultOpener{}, repoPath)
}
