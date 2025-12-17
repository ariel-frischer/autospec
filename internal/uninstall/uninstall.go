//go:build !windows

// Package uninstall provides functionality for completely removing autospec from a system.
package uninstall

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// TargetType represents the type of an uninstall target
type TargetType string

const (
	// TypeBinary indicates the autospec binary
	TypeBinary TargetType = "binary"
	// TypeConfigDir indicates the user configuration directory
	TypeConfigDir TargetType = "config_dir"
	// TypeStateDir indicates the state directory
	TypeStateDir TargetType = "state_dir"
)

// UninstallTarget represents a file or directory to be removed during uninstall
type UninstallTarget struct {
	Path         string     // Absolute path to the target
	Type         TargetType // Type of target: binary, config_dir, or state_dir
	Description  string     // Human-readable description for display
	Exists       bool       // Whether the target currently exists
	RequiresSudo bool       // Whether elevated privileges are needed
}

// UninstallResult represents the result of attempting to remove a target
type UninstallResult struct {
	Target  UninstallTarget // The target that was processed
	Success bool            // Whether removal succeeded
	Error   error           // Error if removal failed
}

// DetectBinaryLocation returns the absolute path to the current autospec executable.
// It resolves symlinks to get the actual binary location.
func DetectBinaryLocation() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}

	// Resolve symlinks to get the actual binary location
	resolved, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		return "", err
	}

	return resolved, nil
}

// RequiresSudo checks if removing a file at the given path requires elevated privileges.
// It returns true if the file is in a system directory where the current user
// doesn't have write permission.
func RequiresSudo(path string) bool {
	// Check if we can write to the parent directory
	dir := filepath.Dir(path)

	// Use unix.Access to check write permission
	return unix.Access(dir, unix.W_OK) != nil
}

// GetUninstallTargets returns all targets that should be removed during uninstall.
// Each target has its Exists and RequiresSudo fields populated.
func GetUninstallTargets() ([]UninstallTarget, error) {
	var targets []UninstallTarget

	// Get binary location
	binaryPath, err := DetectBinaryLocation()
	if err != nil {
		return nil, err
	}

	binaryExists := fileExists(binaryPath)
	targets = append(targets, UninstallTarget{
		Path:         binaryPath,
		Type:         TypeBinary,
		Description:  "autospec binary",
		Exists:       binaryExists,
		RequiresSudo: binaryExists && RequiresSudo(binaryPath),
	})

	// Get user config directory (~/.config/autospec/)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configDir := filepath.Join(homeDir, ".config", "autospec")
	targets = append(targets, UninstallTarget{
		Path:         configDir,
		Type:         TypeConfigDir,
		Description:  "user configuration directory",
		Exists:       dirExists(configDir),
		RequiresSudo: false, // User config is always user-writable
	})

	// Get state directory (~/.autospec/)
	stateDir := filepath.Join(homeDir, ".autospec")
	targets = append(targets, UninstallTarget{
		Path:         stateDir,
		Type:         TypeStateDir,
		Description:  "state directory",
		Exists:       dirExists(stateDir),
		RequiresSudo: false, // State dir is always user-writable
	})

	return targets, nil
}

// RemoveTargets removes the specified targets and returns the results.
// It continues after individual failures and reports all results.
// Missing files are handled gracefully (not considered an error).
func RemoveTargets(targets []UninstallTarget) []UninstallResult {
	results := make([]UninstallResult, 0, len(targets))

	for _, target := range targets {
		// Skip if target doesn't exist - this is not an error
		if !target.Exists {
			results = append(results, UninstallResult{
				Target:  target,
				Success: true,
				Error:   nil,
			})
			continue
		}

		var err error
		if target.Type == TypeBinary {
			err = os.Remove(target.Path)
		} else {
			// For directories, use RemoveAll
			err = os.RemoveAll(target.Path)
		}

		results = append(results, UninstallResult{
			Target:  target,
			Success: err == nil,
			Error:   err,
		})
	}

	return results
}

// fileExists checks if a file exists at the given path
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// dirExists checks if a directory exists at the given path
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
