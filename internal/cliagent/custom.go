package cliagent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/shlex"
)

const promptPlaceholder = "{{PROMPT}}"

// CustomAgent implements the Agent interface using a command template with {{PROMPT}} placeholder.
// This enables integration of arbitrary CLI tools not built into autospec.
type CustomAgent struct {
	name     string
	template string
	caps     Caps
}

// NewCustomAgent creates a new CustomAgent from a command template.
// The template must contain the {{PROMPT}} placeholder.
// Returns an error if the placeholder is missing.
func NewCustomAgent(template string) (*CustomAgent, error) {
	if !strings.Contains(template, promptPlaceholder) {
		return nil, fmt.Errorf("custom agent template must contain %s placeholder", promptPlaceholder)
	}
	return &CustomAgent{
		name:     "custom",
		template: template,
		caps: Caps{
			Automatable: true,
			PromptDelivery: PromptDelivery{
				Method: PromptMethodTemplate,
			},
		},
	}, nil
}

// Name returns the agent's unique identifier.
func (c *CustomAgent) Name() string {
	return c.name
}

// Version returns "custom" since there's no underlying CLI to query.
func (c *CustomAgent) Version() (string, error) {
	return "custom", nil
}

// Validate checks that the template is parseable and the command exists.
func (c *CustomAgent) Validate() error {
	// Expand with a dummy prompt to parse the template
	expanded := strings.ReplaceAll(c.template, promptPlaceholder, "test")
	parts, err := shlex.Split(expanded)
	if err != nil {
		return fmt.Errorf("custom agent: invalid template: %w", err)
	}
	if len(parts) == 0 {
		return fmt.Errorf("custom agent: template produces no command")
	}
	// Check if the command exists in PATH
	if _, err := exec.LookPath(parts[0]); err != nil {
		return fmt.Errorf("custom agent: command %q not found in PATH", parts[0])
	}
	return nil
}

// Capabilities returns the agent's capability flags.
func (c *CustomAgent) Capabilities() Caps {
	return c.caps
}

// BuildCommand constructs an exec.Cmd by expanding the template with the prompt.
func (c *CustomAgent) BuildCommand(prompt string, opts ExecOptions) (*exec.Cmd, error) {
	args, err := c.expandTemplate(prompt)
	if err != nil {
		return nil, fmt.Errorf("expanding template: %w", err)
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("template expansion produced no command")
	}

	cmd := exec.Command(args[0], args[1:]...)
	c.configureCmd(cmd, opts)
	return cmd, nil
}

// expandTemplate replaces {{PROMPT}} with the prompt and parses the result.
// The prompt is safely quoted to prevent shell injection.
func (c *CustomAgent) expandTemplate(prompt string) ([]string, error) {
	// Quote the prompt to preserve it as a single argument
	// This prevents shell word-splitting on spaces and handles special characters
	quotedPrompt := quoteForShlex(prompt)
	expanded := strings.ReplaceAll(c.template, promptPlaceholder, quotedPrompt)
	return shlex.Split(expanded)
}

// quoteForShlex wraps a string in single quotes for safe shlex parsing.
// Single quotes preserve literal values, escaping embedded single quotes.
func quoteForShlex(s string) string {
	// If empty, return empty quoted string
	if s == "" {
		return "''"
	}
	// Escape single quotes by ending quote, adding escaped quote, starting new quote
	// 'don't' becomes 'don'\''t'
	escaped := strings.ReplaceAll(s, "'", `'\''`)
	return "'" + escaped + "'"
}

// configureCmd sets working directory and environment on the command.
func (c *CustomAgent) configureCmd(cmd *exec.Cmd, opts ExecOptions) {
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}
	cmd.Env = os.Environ()
	for k, v := range opts.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
}

// Execute builds and runs the command, returning the result.
func (c *CustomAgent) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Result, error) {
	cmd, err := c.BuildCommand(prompt, opts)
	if err != nil {
		return nil, err
	}
	return c.runCommand(ctx, cmd, opts)
}

// runCommand executes the command and captures output.
func (c *CustomAgent) runCommand(ctx context.Context, cmd *exec.Cmd, opts ExecOptions) (*Result, error) {
	ctx, cancel := c.applyTimeout(ctx, opts)
	defer cancel()

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = opts.Stdout
	if cmd.Stdout == nil {
		cmd.Stdout = &stdoutBuf
	}
	cmd.Stderr = opts.Stderr
	if cmd.Stderr == nil {
		cmd.Stderr = &stderrBuf
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting custom agent: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	start := time.Now()
	var err error
	select {
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		<-done
		return nil, fmt.Errorf("executing custom agent: %w", ctx.Err())
	case err = <-done:
	}
	duration := time.Since(start)

	result := &Result{
		Duration: duration,
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("executing custom agent: %w", err)
		}
	}
	return result, nil
}

// applyTimeout returns a context with timeout if opts.Timeout is set.
func (c *CustomAgent) applyTimeout(ctx context.Context, opts ExecOptions) (context.Context, context.CancelFunc) {
	if opts.Timeout > 0 {
		return context.WithTimeout(ctx, opts.Timeout)
	}
	return ctx, func() {}
}
