// Package constitution provides helpers for locating project governance files.
package constitution

import (
	"os"
	"path/filepath"
)

// Paths contains all valid paths for the autospec constitution file in priority
// order.
var Paths = []string{
	".autospec/constitution.yaml",
	".autospec/constitution.yml",
	".autospec/memory/constitution.yaml",
	".autospec/memory/constitution.yml",
	".specify/memory/constitution.yaml",
	".specify/memory/constitution.yml",
}

// Find returns the first constitution path that exists.
func Find() (string, bool) {
	for _, path := range Paths {
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	return FindFrom(filepath.Dir(wd))
}

// FindFrom walks upward from startDir and returns the first constitution path
// that exists.
func FindFrom(startDir string) (string, bool) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return findInDir(startDir)
	}

	for {
		if path, ok := findInDir(dir); ok {
			return path, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func findInDir(dir string) (string, bool) {
	for _, path := range Paths {
		candidate := filepath.Join(dir, path)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}
	return "", false
}
