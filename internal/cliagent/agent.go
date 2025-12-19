// Package cliagent provides abstractions for CLI AI coding agents.
// It enables switching between different CLI tools (Claude, Cline, Gemini, etc.)
// through a unified interface.
package cliagent

import (
	"context"
	"os/exec"
)

// Agent represents a CLI AI coding agent with execution and discovery capabilities.
// All methods must be safe for concurrent use.
type Agent interface {
	// Name returns the unique identifier for the agent (e.g., "claude", "gemini").
	// Must be lowercase alphanumeric.
	Name() string

	// Version returns the installed CLI version or error if unavailable.
	// May return ("unknown", nil) if version cannot be determined but agent is functional.
	Version() (string, error)

	// Validate checks if the CLI is installed in PATH and required env vars are set.
	// Must complete in under 100ms.
	Validate() error

	// BuildCommand constructs an exec.Cmd for the given prompt and options.
	// Does not execute the command - allows inspection and modification before execution.
	BuildCommand(prompt string, opts ExecOptions) (*exec.Cmd, error)

	// Execute builds and runs the command, capturing output.
	// Respects context cancellation and timeout from opts.
	Execute(ctx context.Context, prompt string, opts ExecOptions) (*Result, error)

	// Capabilities returns self-describing feature flags for this agent.
	Capabilities() Caps
}
