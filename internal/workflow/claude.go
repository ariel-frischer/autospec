package workflow

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// ClaudeExecutor handles Claude CLI command execution
type ClaudeExecutor struct {
	ClaudeCmd       string
	ClaudeArgs      []string
	UseAPIKey       bool
	CustomClaudeCmd string
}

// Execute runs a Claude command with the given prompt
// Streams output to stdout in real-time
func (c *ClaudeExecutor) Execute(prompt string) error {
	var cmd *exec.Cmd

	if c.CustomClaudeCmd != "" {
		// Use custom command template
		cmdStr := c.expandTemplate(prompt)
		cmd = c.parseCustomCommand(cmdStr)
	} else {
		// Use simple mode: claude_cmd + claude_args + prompt
		args := append(c.ClaudeArgs, prompt)
		cmd = exec.Command(c.ClaudeCmd, args...)
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
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude command failed: %w", err)
	}

	return nil
}

// expandTemplate replaces {{PROMPT}} placeholder with actual prompt
func (c *ClaudeExecutor) expandTemplate(prompt string) string {
	return strings.ReplaceAll(c.CustomClaudeCmd, "{{PROMPT}}", prompt)
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

// ExecuteSpecKitCommand is a convenience function for SpecKit slash commands
func (c *ClaudeExecutor) ExecuteSpecKitCommand(command string) error {
	// SpecKit commands are slash commands like /speckit.specify, /speckit.plan, etc.
	return c.Execute(command)
}

// StreamCommand executes a command and streams output to the provided writer
// This is useful for testing or capturing output
func (c *ClaudeExecutor) StreamCommand(prompt string, stdout, stderr io.Writer) error {
	var cmd *exec.Cmd

	if c.CustomClaudeCmd != "" {
		cmdStr := c.expandTemplate(prompt)
		cmd = c.parseCustomCommand(cmdStr)
	} else {
		args := append(c.ClaudeArgs, prompt)
		cmd = exec.Command(c.ClaudeCmd, args...)
	}

	cmd.Env = os.Environ()
	if !c.UseAPIKey {
		cmd.Env = append(cmd.Env, "ANTHROPIC_API_KEY=")
	}

	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
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
