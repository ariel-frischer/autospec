//go:build windows

package uninstall

import (
	"os"
	"path/filepath"
)

// RequiresSudo reports whether removing a file likely needs elevated privileges.
func RequiresSudo(path string) bool {
	probe, err := os.CreateTemp(filepath.Dir(path), ".autospec-write-check-*")
	if err != nil {
		return true
	}

	name := probe.Name()
	if err := probe.Close(); err != nil {
		return true
	}
	return os.Remove(name) != nil
}
