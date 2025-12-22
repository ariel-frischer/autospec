package cliagent

import (
	"io"
	"time"
)

// ExecOptions configures a single agent execution.
type ExecOptions struct {
	// Autonomous enables headless/YOLO mode with no user confirmations.
	// Adds agent-specific autonomous flags to the command.
	Autonomous bool

	// Timeout is the maximum execution duration.
	// Zero means no timeout; context deadline takes precedence if set.
	Timeout time.Duration

	// WorkDir is the working directory for command execution.
	// Defaults to the current directory if empty.
	WorkDir string

	// ExtraArgs are additional CLI arguments appended after standard args.
	ExtraArgs []string

	// Env contains additional environment variables.
	// Merged with the process environment; these values take precedence.
	Env map[string]string

	// Stdout is where to write stdout.
	// If nil, output is captured in Result.Stdout.
	Stdout io.Writer

	// Stderr is where to write stderr.
	// If nil, output is captured in Result.Stderr.
	Stderr io.Writer

	// Stdin is where to read stdin from.
	// Required for interactive mode to allow user input.
	// If nil in interactive mode, os.Stdin is used automatically.
	Stdin io.Reader

	// UseSubscription forces subscription mode (Pro/Max) instead of API credits.
	// When true, ANTHROPIC_API_KEY is set to empty string in the execution environment.
	// This prevents accidental API charges when users have API keys in their shell.
	UseSubscription bool

	// Interactive enables interactive conversation mode (skips headless flags).
	// When true, -p and --output-format flags are omitted, allowing multi-turn conversation.
	// Used for recommendation-focused stages like analyze and clarify.
	Interactive bool

	// ReplaceProcess controls whether interactive mode replaces the current process.
	// When true (default for standalone commands), uses syscall.Exec for full terminal control.
	// When false (for multi-stage runs), uses subprocess which may have limited terminal support.
	// Only applies when Interactive is true.
	ReplaceProcess bool
}

// Result contains the outcome of an agent execution.
type Result struct {
	// ExitCode is the process exit status (0 indicates success).
	ExitCode int

	// Stdout contains captured stdout if no Stdout writer was provided in ExecOptions.
	Stdout string

	// Stderr contains captured stderr if no Stderr writer was provided in ExecOptions.
	Stderr string

	// Duration is the execution time from command start to completion.
	Duration time.Duration
}
