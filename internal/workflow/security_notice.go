// Package workflow provides workflow orchestration for the autospec CLI.
// This file contains security notice functionality for the --dangerously-skip-permissions flag.
package workflow

import (
	"fmt"
	"io"
	"os"

	"github.com/ariel-frischer/autospec/internal/claude"
	"github.com/ariel-frischer/autospec/internal/config"
)

// SecurityNoticeConfig holds configuration for the security notice.
type SecurityNoticeConfig struct {
	// NoticeShown indicates if the user has previously seen the notice.
	NoticeShown bool
	// ProjectDir is the directory containing Claude settings (typically ".").
	ProjectDir string
}

// ShowSecurityNoticeOnce displays the --dangerously-skip-permissions security notice
// exactly once per user. After showing, it marks the notice as shown in user config.
//
// The notice is skipped if:
// - AUTOSPEC_SKIP_PERMISSIONS_NOTICE=1 environment variable is set
// - skip_permissions_notice_shown is true in user config
//
// Returns true if the notice was shown, false if skipped.
// Errors during sandbox check or config update are logged but don't prevent execution.
func ShowSecurityNoticeOnce(out io.Writer, cfg *config.Configuration) bool {
	// Check environment variable override
	if os.Getenv("AUTOSPEC_SKIP_PERMISSIONS_NOTICE") == "1" {
		return false
	}

	// Check if notice was already shown
	if cfg.SkipPermissionsNoticeShown {
		return false
	}

	// Check sandbox status
	sandboxEnabled, err := claude.CheckSandboxStatus(".")
	if err != nil {
		// Log error but continue - sandbox status unknown
		fmt.Fprintf(os.Stderr, "Warning: failed to check sandbox status: %v\n", err)
	}

	// Show the notice
	showSecurityNotice(out, sandboxEnabled)

	// Mark notice as shown (errors are non-fatal)
	if err := config.MarkSkipPermissionsNoticeShown(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save notice preference: %v\n", err)
	}

	return true
}

// showSecurityNotice prints the security notice to the given writer.
func showSecurityNotice(out io.Writer, sandboxEnabled bool) {
	sandboxStatus := "disabled ✗"
	sandboxIcon := "⚠"
	if sandboxEnabled {
		sandboxStatus = "enabled ✓"
		sandboxIcon = "✓"
	}

	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "┌─────────────────────────────────────────────────────────────────────────────┐\n")
	fmt.Fprintf(out, "│ Security Notice                                                             │\n")
	fmt.Fprintf(out, "├─────────────────────────────────────────────────────────────────────────────┤\n")
	fmt.Fprintf(out, "│ Running with --dangerously-skip-permissions                                 │\n")
	fmt.Fprintf(out, "│                                                                             │\n")
	fmt.Fprintf(out, "│ This flag is RECOMMENDED for autospec workflows. Without it, Claude        │\n")
	fmt.Fprintf(out, "│ requires manual approval for many common development tasks (file edits,    │\n")
	fmt.Fprintf(out, "│ shell commands, etc.), making automation impractical.                      │\n")
	fmt.Fprintf(out, "│                                                                             │\n")
	fmt.Fprintf(out, "│ %s Sandbox: %-67s│\n", sandboxIcon, sandboxStatus)

	if sandboxEnabled {
		fmt.Fprintf(out, "│   OS-level protection active - Claude cannot access files outside project. │\n")
	} else {
		fmt.Fprintf(out, "│   Enable sandbox for safer automation: run 'autospec init' or see docs.    │\n")
	}

	fmt.Fprintf(out, "│                                                                             │\n")
	fmt.Fprintf(out, "│ See docs/claude-settings.md for security details.                          │\n")
	fmt.Fprintf(out, "│ Suppress: autospec config set skip_permissions_notice_shown true           │\n")
	fmt.Fprintf(out, "│      or:  AUTOSPEC_SKIP_PERMISSIONS_NOTICE=1                               │\n")
	fmt.Fprintf(out, "└─────────────────────────────────────────────────────────────────────────────┘\n")
	fmt.Fprintf(out, "\n")
}
