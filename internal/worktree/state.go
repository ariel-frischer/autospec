package worktree

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// StateFileName is the name of the worktree state file.
	StateFileName = "worktrees.yaml"
	// StateVersion is the current schema version.
	StateVersion = "1.0.0"
)

// LoadState reads the worktree state from the state directory.
// Returns an empty state if the file doesn't exist.
func LoadState(stateDir string) (*WorktreeState, error) {
	statePath := filepath.Join(stateDir, StateFileName)

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &WorktreeState{
				Version:   StateVersion,
				Worktrees: []Worktree{},
			}, nil
		}
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var state WorktreeState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}

	return &state, nil
}

// SaveState writes the worktree state to the state directory atomically.
// Uses temp file + rename pattern for crash safety.
func SaveState(stateDir string, state *WorktreeState) error {
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	state.Version = StateVersion

	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	statePath := filepath.Join(stateDir, StateFileName)
	tmpPath := statePath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("writing temp state file: %w", err)
	}

	if err := os.Rename(tmpPath, statePath); err != nil {
		os.Remove(tmpPath) // Best effort cleanup
		return fmt.Errorf("renaming temp state file: %w", err)
	}

	return nil
}

// FindWorktree returns a worktree by name from the state, or nil if not found.
func (s *WorktreeState) FindWorktree(name string) *Worktree {
	for i := range s.Worktrees {
		if s.Worktrees[i].Name == name {
			return &s.Worktrees[i]
		}
	}
	return nil
}

// AddWorktree adds a worktree to the state.
// Returns an error if a worktree with the same name already exists.
func (s *WorktreeState) AddWorktree(wt Worktree) error {
	if s.FindWorktree(wt.Name) != nil {
		return fmt.Errorf("worktree %q already exists", wt.Name)
	}
	s.Worktrees = append(s.Worktrees, wt)
	return nil
}

// RemoveWorktree removes a worktree from the state by name.
// Returns true if a worktree was removed, false if not found.
func (s *WorktreeState) RemoveWorktree(name string) bool {
	for i, wt := range s.Worktrees {
		if wt.Name == name {
			s.Worktrees = append(s.Worktrees[:i], s.Worktrees[i+1:]...)
			return true
		}
	}
	return false
}

// UpdateWorktree updates a worktree in the state.
// Returns an error if the worktree is not found.
func (s *WorktreeState) UpdateWorktree(wt Worktree) error {
	for i := range s.Worktrees {
		if s.Worktrees[i].Name == wt.Name {
			s.Worktrees[i] = wt
			return nil
		}
	}
	return fmt.Errorf("worktree %q not found", wt.Name)
}
