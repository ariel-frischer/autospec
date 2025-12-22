package workflow

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ariel-frischer/autospec/internal/cliagent"
	"github.com/ariel-frischer/autospec/internal/config"
)

// ClaudeExecutor handles CLI agent command execution.
type ClaudeExecutor struct {
	// Agent is the abstraction for CLI agent execution.
	Agent cliagent.Agent

	Timeout int // Timeout in seconds (0 = no timeout)

	// OutputStyle controls how stream-json output is formatted for display.
	// When set and stream-json mode is detected, output is formatted using cclean.
	// Valid values: default, compact, minimal, plain, raw
	OutputStyle config.OutputStyle

	// UseSubscription forces subscription mode (Pro/Max) instead of API credits.
	// When true, ANTHROPIC_API_KEY is set to empty string in the execution environment.
	UseSubscription bool

	// ReplaceProcessForInteractive controls whether interactive mode replaces the process.
	// When true (default), uses syscall.Exec for full terminal control in interactive mode.
	// Set to false for multi-stage runs where we need to continue after interactive stages.
	ReplaceProcessForInteractive bool
}

// Execute runs an agent command with the given prompt.
// Streams output to stdout in real-time.
// If Timeout > 0, the command is terminated after the timeout duration.
func (c *ClaudeExecutor) Execute(prompt string) error {
	if c.Agent == nil {
		return fmt.Errorf("no agent configured")
	}
	return c.executeWithAgent(prompt, false)
}

// ExecuteInteractive runs an agent command in interactive mode.
// Unlike Execute, this skips headless flags (-p, --output-format)
// to allow multi-turn conversation with the user.
func (c *ClaudeExecutor) ExecuteInteractive(prompt string) error {
	if c.Agent == nil {
		return fmt.Errorf("no agent configured")
	}
	return c.executeWithAgent(prompt, true)
}

// executeWithAgent uses the new Agent interface for execution.
// When interactive is true, sets ExecOptions.Interactive to skip headless flags.
func (c *ClaudeExecutor) executeWithAgent(prompt string, interactive bool) error {
	ctx, cancel := c.createTimeoutContext()
	if cancel != nil {
		defer cancel()
	}

	// Determine stdout writer, potentially wrapping with formatter
	// Skip formatter for interactive mode (no stream-json output)
	var stdout io.Writer = os.Stdout
	if !interactive {
		stdout = c.getFormattedStdout(os.Stdout)
	}

	opts := cliagent.ExecOptions{
		Stdout:          stdout,
		Stderr:          os.Stderr,
		Timeout:         time.Duration(c.Timeout) * time.Second,
		UseSubscription: c.UseSubscription,
		Interactive:     interactive,
		ReplaceProcess:  interactive && c.ReplaceProcessForInteractive,
	}

	result, err := c.Agent.Execute(ctx, prompt, opts)

	// Flush formatter if used (only applies to non-interactive mode)
	if !interactive {
		c.flushFormatter(stdout)
	}

	if err != nil {
		// Check for timeout specifically
		if ctx.Err() == context.DeadlineExceeded {
			return NewTimeoutError(time.Duration(c.Timeout)*time.Second, c.FormatCommand(prompt))
		}
		return fmt.Errorf("agent %s command failed: %w", c.Agent.Name(), err)
	}

	// Check exit code
	if result.ExitCode != 0 {
		return fmt.Errorf("agent %s exited with code %d", c.Agent.Name(), result.ExitCode)
	}
	return nil
}

// createTimeoutContext creates a context with optional timeout
func (c *ClaudeExecutor) createTimeoutContext() (context.Context, context.CancelFunc) {
	if c.Timeout > 0 {
		return context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
	}
	return context.Background(), nil
}

// FormatCommand returns a human-readable command string for display and error messages.
func (c *ClaudeExecutor) FormatCommand(prompt string) string {
	if c.Agent == nil {
		return "[no agent configured]"
	}
	cmd, err := c.Agent.BuildCommand(prompt, cliagent.ExecOptions{})
	if err != nil {
		return fmt.Sprintf("%s [error: %v]", c.Agent.Name(), err)
	}
	return strings.Join(cmd.Args, " ")
}

// ExecuteSpecKitCommand is a convenience function for AutoSpec slash commands
func (c *ClaudeExecutor) ExecuteSpecKitCommand(command string) error {
	// AutoSpec commands are slash commands like /autospec.specify, /autospec.plan, etc.
	return c.Execute(command)
}

// StreamCommand executes a command and streams output to the provided writer.
// This is useful for testing or capturing output.
// If Timeout > 0, the command is terminated after the timeout duration.
func (c *ClaudeExecutor) StreamCommand(prompt string, stdout, stderr io.Writer) error {
	if c.Agent == nil {
		return fmt.Errorf("no agent configured")
	}

	ctx, cancel := c.createTimeoutContext()
	if cancel != nil {
		defer cancel()
	}

	// Optionally wrap stdout with formatter
	formattedStdout := c.getFormattedStdout(stdout)

	opts := cliagent.ExecOptions{
		Stdout:          formattedStdout,
		Stderr:          stderr,
		Timeout:         time.Duration(c.Timeout) * time.Second,
		UseSubscription: c.UseSubscription,
	}

	result, err := c.Agent.Execute(ctx, prompt, opts)

	// Flush formatter if used
	c.flushFormatter(formattedStdout)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return NewTimeoutError(time.Duration(c.Timeout)*time.Second, c.FormatCommand(prompt))
		}
		return fmt.Errorf("agent %s command failed: %w", c.Agent.Name(), err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("agent %s exited with code %d", c.Agent.Name(), result.ExitCode)
	}
	return nil
}

// getFormattedStdout returns either a FormatterWriter or the original writer.
// Returns a FormatterWriter when:
// - OutputStyle is set (not empty or "raw")
// - Stream-json mode with headless flag is detected
// Otherwise, returns the original writer unchanged.
func (c *ClaudeExecutor) getFormattedStdout(w io.Writer) io.Writer {
	// Skip formatting if OutputStyle is not set or is raw
	if c.OutputStyle == "" || c.OutputStyle.IsRaw() {
		return w
	}

	// Only format when stream-json + headless mode detected
	if !c.detectStreamJsonMode() {
		return w
	}

	return NewFormatterWriter(c.OutputStyle, w)
}

// flushFormatter flushes the FormatterWriter if the writer is one.
// Safe to call on any io.Writer (no-op for non-formatters).
func (c *ClaudeExecutor) flushFormatter(w io.Writer) {
	if fw, ok := w.(*FormatterWriter); ok {
		fw.Flush()
	}
}

// detectStreamJsonMode checks if the agent command is configured for stream-json output
// in headless mode. This is detected by looking for:
// - "--output-format stream-json" or "-o stream-json" in command args
// - "-p" flag indicating headless mode
//
// Both conditions must be present for stream formatting to be applied.
func (c *ClaudeExecutor) detectStreamJsonMode() bool {
	args := c.getCommandArgs()
	return hasStreamJsonFormat(args) && hasHeadlessFlag(args)
}

// getCommandArgs returns the args that will be used for command execution.
func (c *ClaudeExecutor) getCommandArgs() []string {
	if c.Agent == nil {
		return nil
	}
	cmd, err := c.Agent.BuildCommand("", cliagent.ExecOptions{})
	if err != nil {
		return nil
	}
	return cmd.Args
}

// hasStreamJsonFormat checks if args contain --output-format stream-json or -o stream-json.
func hasStreamJsonFormat(args []string) bool {
	for i, arg := range args {
		// Check for long form: --output-format stream-json
		if arg == "--output-format" && i+1 < len(args) && args[i+1] == "stream-json" {
			return true
		}
		// Check for short form: -o stream-json
		if arg == "-o" && i+1 < len(args) && args[i+1] == "stream-json" {
			return true
		}
		// Check for combined form: --output-format=stream-json
		if strings.HasPrefix(arg, "--output-format=") && strings.HasSuffix(arg, "stream-json") {
			return true
		}
	}
	return false
}

// hasHeadlessFlag checks if args contain the -p flag for headless mode.
func hasHeadlessFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-p" {
			return true
		}
	}
	return false
}
