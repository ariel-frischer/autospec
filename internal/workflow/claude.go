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
type ClaudeExecutor struct {
	ClaudeCmd       string
	ClaudeArgs      []string
	UseAPIKey       bool
	CustomClaudeCmd string
	Timeout         int // Timeout in seconds (0 = no timeout)
}

// Execute runs a Claude command with the given prompt
// Streams output to stdout in real-time
// If Timeout > 0, the command is terminated after the timeout duration
func (c *ClaudeExecutor) Execute(prompt string) error {
	// Create context with timeout if configured
	var ctx context.Context
	var cancel context.CancelFunc

	if c.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	var cmd *exec.Cmd

	if c.CustomClaudeCmd != "" {
		// Use custom command template
		cmdStr := c.expandTemplate(prompt)
		cmd = c.parseCustomCommandContext(ctx, cmdStr)
	} else {
		// Use simple mode: claude_cmd + claude_args + prompt
		args := append(c.ClaudeArgs, prompt)
		cmd = exec.CommandContext(ctx, c.ClaudeCmd, args...)
	}

	// Set up environment
	cmd.Env = os.Environ()
	if !c.UseAPIKey {
		// Explicitly set empty API key if not using it
		cmd.Env = append(cmd.Env, "ANTHROPIC_API_KEY=")
	}

	// Stream output to stdout
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Execute command
	err := cmd.Run()

	// Check if timeout occurred
	if ctx.Err() == context.DeadlineExceeded {
		commandStr := c.formatCommand(prompt)
		return NewTimeoutError(time.Duration(c.Timeout)*time.Second, commandStr)
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

// ExecuteSpecKitCommand is a convenience function for SpecKit slash commands
func (c *ClaudeExecutor) ExecuteSpecKitCommand(command string) error {
	// SpecKit commands are slash commands like /speckit.specify, /speckit.plan, etc.
	return c.Execute(command)
}

// StreamCommand executes a command and streams output to the provided writer
// This is useful for testing or capturing output
// If Timeout > 0, the command is terminated after the timeout duration
func (c *ClaudeExecutor) StreamCommand(prompt string, stdout, stderr io.Writer) error {
	// Create context with timeout if configured
	var ctx context.Context
	var cancel context.CancelFunc

	if c.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	var cmd *exec.Cmd

	if c.CustomClaudeCmd != "" {
		cmdStr := c.expandTemplate(prompt)
		cmd = c.parseCustomCommandContext(ctx, cmdStr)
	} else {
		args := append(c.ClaudeArgs, prompt)
		cmd = exec.CommandContext(ctx, c.ClaudeCmd, args...)
	}

	cmd.Env = os.Environ()
	if !c.UseAPIKey {
		cmd.Env = append(cmd.Env, "ANTHROPIC_API_KEY=")
	}

	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin

	// Execute command
	err := cmd.Run()

	// Check if timeout occurred
	if ctx.Err() == context.DeadlineExceeded {
		commandStr := c.formatCommand(prompt)
		return NewTimeoutError(time.Duration(c.Timeout)*time.Second, commandStr)
	}

	return err
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
