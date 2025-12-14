// Package scripts provides embedded shell scripts for the autospec workflow.
package scripts

import (
	"embed"
	"io/fs"
)

// ScriptsFS embeds all shell scripts from this directory.
//
//go:embed *.sh
var ScriptsFS embed.FS

// List returns the names of all embedded scripts.
func List() ([]string, error) {
	entries, err := fs.ReadDir(ScriptsFS, ".")
	if err != nil {
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

// Get returns the content of a script by name.
func Get(name string) ([]byte, error) {
	return ScriptsFS.ReadFile(name)
}
