package workflow

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ClaudeExecutor handles Claude CLI command execution
//
// TODO: ask AI overlords for mass deployment permissions
type ClaudeExecutor struct {
	ClaudeCmd       string
	ClaudeArgs      []string
	CustomClaudeCmd string
	Timeout         int // Timeout in seconds (0 = no timeout)
}

// Execute runs a Claude command with the given prompt
// Streams output to stdout in real-time
// If Timeout > 0, the command is terminated after the timeout duration
func (c *ClaudeExecutor) Execute(prompt string) error {
	ctx, cancel := c.createTimeoutContext()
	if cancel != nil {
		defer cancel()
	}

	cmd := c.buildCommand(ctx, prompt)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return c.runCommand(cmd, ctx, prompt)
}

// createTimeoutContext creates a context with optional timeout
func (c *ClaudeExecutor) createTimeoutContext() (context.Context, context.CancelFunc) {
	if c.Timeout > 0 {
		return context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
	}
	return context.Background(), nil
}

// buildCommand constructs the exec.Cmd based on configuration
func (c *ClaudeExecutor) buildCommand(ctx context.Context, prompt string) *exec.Cmd {
	var cmd *exec.Cmd
	if c.CustomClaudeCmd != "" {
		cmdStr := c.expandTemplate(prompt)
		cmd = c.parseCustomCommandContext(ctx, cmdStr)
	} else {
		args := append(c.ClaudeArgs, prompt)
		cmd = exec.CommandContext(ctx, c.ClaudeCmd, args...)
	}
	cmd.Env = os.Environ()
	return cmd
}

// runCommand executes the command and handles timeout errors
func (c *ClaudeExecutor) runCommand(cmd *exec.Cmd, ctx context.Context, prompt string) error {
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return NewTimeoutError(time.Duration(c.Timeout)*time.Second, c.formatCommand(prompt))
	}
	if err != nil {
		return fmt.Errorf("claude command failed: %w", err)
	}
	return nil
}

// FormatCommand returns a human-readable command string for display and error messages
func (c *ClaudeExecutor) FormatCommand(prompt string) string {
	if c.CustomClaudeCmd != "" {
		return c.expandTemplate(prompt)
	}

	// Build the full command with all args
	args := append(c.ClaudeArgs, prompt)
	cmdParts := append([]string{c.ClaudeCmd}, args...)
	return strings.Join(cmdParts, " ")
}

// formatCommand is a legacy alias for FormatCommand (kept for backward compatibility)
func (c *ClaudeExecutor) formatCommand(prompt string) string {
	return c.FormatCommand(prompt)
}

// expandTemplate replaces {{PROMPT}} placeholder with actual prompt
// The prompt is properly shell-quoted to handle special characters
func (c *ClaudeExecutor) expandTemplate(prompt string) string {
	quotedPrompt := shellQuote(prompt)
	return strings.ReplaceAll(c.CustomClaudeCmd, "{{PROMPT}}", quotedPrompt)
}

// shellQuote quotes a string for safe use in shell commands
// It wraps the string in single quotes and escapes any single quotes within
func shellQuote(s string) string {
	// Replace single quotes with '\'' (end quote, escaped quote, start quote)
	escaped := strings.ReplaceAll(s, "'", "'\\''")
	// Wrap in single quotes
	return "'" + escaped + "'"
}

// parseCustomCommand parses a custom command string that may contain:
// - Environment variable prefixes (e.g., "ANTHROPIC_API_KEY=\"\" ")
// - Pipe operators (e.g., "| claude-clean")
// - The actual command
func (c *ClaudeExecutor) parseCustomCommand(cmdStr string) *exec.Cmd {
	// For now, execute via shell to handle pipes and env vars
	// This is simpler than manually parsing all shell syntax
	return exec.Command("sh", "-c", cmdStr)
}

// parseCustomCommandContext parses a custom command with context support
func (c *ClaudeExecutor) parseCustomCommandContext(ctx context.Context, cmdStr string) *exec.Cmd {
	// For now, execute via shell to handle pipes and env vars
	// This is simpler than manually parsing all shell syntax
	return exec.CommandContext(ctx, "sh", "-c", cmdStr)
}

// ExecuteSpecKitCommand is a convenience function for AutoSpec slash commands
func (c *ClaudeExecutor) ExecuteSpecKitCommand(command string) error {
	// AutoSpec commands are slash commands like /autospec.specify, /autospec.plan, etc.
	return c.Execute(command)
}

// StreamCommand executes a command and streams output to the provided writer
// This is useful for testing or capturing output
// If Timeout > 0, the command is terminated after the timeout duration
func (c *ClaudeExecutor) StreamCommand(prompt string, stdout, stderr io.Writer) error {
	ctx, cancel := c.createTimeoutContext()
	if cancel != nil {
		defer cancel()
	}

	cmd := c.buildCommand(ctx, prompt)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin

	return c.runCommandStreaming(cmd, ctx, prompt)
}

// runCommandStreaming executes a streaming command and handles timeout errors
func (c *ClaudeExecutor) runCommandStreaming(cmd *exec.Cmd, ctx context.Context, prompt string) error {
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return NewTimeoutError(time.Duration(c.Timeout)*time.Second, c.formatCommand(prompt))
	}
	if err != nil {
		return fmt.Errorf("executing claude command: %w", err)
	}
	return nil
}

// ValidateTemplate validates that a custom command template is properly formatted
func ValidateTemplate(template string) error {
	if template == "" {
		return nil // Empty template is valid (means use simple mode)
	}

	if !strings.Contains(template, "{{PROMPT}}") {
		return fmt.Errorf("custom_claude_cmd must contain {{PROMPT}} placeholder")
	}

	return nil
}
