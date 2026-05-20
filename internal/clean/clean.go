// Package clean provides functionality for removing autospec-related files from a project.
package clean

import (
	"os"
	"path/filepath"
)

// TargetType represents the type of a clean target
type TargetType string

const (
	// TypeDirectory indicates a directory target
	TypeDirectory TargetType = "directory"
	// TypeFile indicates a file target
	TypeFile TargetType = "file"
)

// CleanTarget represents a file or directory to be removed during clean operation
type CleanTarget struct {
	Path        string     // Absolute or relative path to the file/directory
	Type        TargetType // Type of target: 'directory' or 'file'
	Description string     // Human-readable description for display
}

// CleanResult represents the result of attempting to remove a target
type CleanResult struct {
	Target  CleanTarget // The target that was processed
	Success bool        // Whether removal succeeded
	Error   error       // Error if removal failed
}

// GetSpecsTarget returns the specs/ directory target if it exists.
// Returns the target and a boolean indicating if specs/ exists.
func GetSpecsTarget() (CleanTarget, bool) {
	if info, err := os.Stat("specs"); err == nil && info.IsDir() {
		return CleanTarget{
			Path:        "specs",
			Type:        TypeDirectory,
			Description: "Feature specifications directory",
		}, true
	}
	return CleanTarget{}, false
}

// FindAutospecFiles detects all autospec-related files and directories in the current directory.
// Scans for: .autospec/, specs/ (unless keepSpecs=true), .claude/commands/autospec*.md,
// and .claude/skills/autospec.* directories.
// Uses os.Stat for directory checks and filepath.Glob for file pattern matching.
// Returns targets in discovery order for predictable output.
func FindAutospecFiles(keepSpecs bool) ([]CleanTarget, error) {
	var targets []CleanTarget

	// Check for .autospec/ directory
	if info, err := os.Stat(".autospec"); err == nil && info.IsDir() {
		targets = append(targets, CleanTarget{
			Path:        ".autospec",
			Type:        TypeDirectory,
			Description: "Autospec configuration directory",
		})
	}

	// Check for specs/ directory (unless keepSpecs is true)
	if !keepSpecs {
		if info, err := os.Stat("specs"); err == nil && info.IsDir() {
			targets = append(targets, CleanTarget{
				Path:        "specs",
				Type:        TypeDirectory,
				Description: "Feature specifications directory",
			})
		}
	}

	// Check for .claude/commands/autospec*.md files using filepath.Glob
	matches, err := filepath.Glob(".claude/commands/autospec*.md")
	if err != nil {
		return nil, err
	}
	for _, match := range matches {
		targets = append(targets, CleanTarget{
			Path:        match,
			Type:        TypeFile,
			Description: "Autospec slash command",
		})
	}

	// Check for .claude/skills/autospec.* directories using filepath.Glob
	skillMatches, err := filepath.Glob(".claude/skills/autospec.*")
	if err != nil {
		return nil, err
	}
	for _, match := range skillMatches {
		if info, err := os.Stat(match); err == nil && info.IsDir() {
			targets = append(targets, CleanTarget{
				Path:        match,
				Type:        TypeDirectory,
				Description: "Autospec Claude skill",
			})
		}
	}

	return targets, nil
}

// RemoveFiles removes the specified targets and returns the results.
// It continues after individual failures and reports all results.
func RemoveFiles(targets []CleanTarget) []CleanResult {
	results := make([]CleanResult, 0, len(targets))

	for _, target := range targets {
		var err error
		if target.Type == TypeDirectory {
			err = os.RemoveAll(target.Path)
		} else {
			err = os.Remove(target.Path)
		}

		results = append(results, CleanResult{
			Target:  target,
			Success: err == nil,
			Error:   err,
		})
	}

	// Clean up empty .claude child directories after removal
	cleanupEmptyDirs()

	return results
}

// cleanupEmptyDirs removes empty generated Claude directories.
func cleanupEmptyDirs() {
	// Try to remove .claude/commands/ if empty
	if entries, err := os.ReadDir(".claude/commands"); err == nil && len(entries) == 0 {
		_ = os.Remove(".claude/commands")
	}

	// Try to remove .claude/skills/ if empty
	if entries, err := os.ReadDir(".claude/skills"); err == nil && len(entries) == 0 {
		_ = os.Remove(".claude/skills")
	}

	// Try to remove .claude/ if empty
	if entries, err := os.ReadDir(".claude"); err == nil && len(entries) == 0 {
		_ = os.Remove(".claude")
	}
}
