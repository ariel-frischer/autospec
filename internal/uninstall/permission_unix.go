//go:build !windows

package uninstall

import (
	"path/filepath"

	"golang.org/x/sys/unix"
)

// RequiresSudo reports whether removing a file likely needs elevated privileges.
func RequiresSudo(path string) bool {
	return unix.Access(filepath.Dir(path), unix.W_OK) != nil
}
