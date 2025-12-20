//go:build tools

// Package tools tracks tool dependencies for autospec.
// These tools are not imported in production code but are required for runtime.
package tools

import (
	// cclean transforms Claude Code's streaming JSON output into readable terminal output.
	// Used as post_processor in autospec's recommended full automation setup.
	_ "github.com/ariel-frischer/claude-clean/parser"
)
