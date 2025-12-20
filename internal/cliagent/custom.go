package cliagent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

const promptPlaceholder = "{{PROMPT}}"

// CustomAgentConfig defines a structured configuration for custom agents.
// This provides a clean, explicit way to configure custom CLI agents.
type CustomAgentConfig struct {
	// Command is the executable to run (e.g., "claude", "aider").
	Command string `koanf:"command" yaml:"command"`

	// Args are the command-line arguments. Use {{PROMPT}} as placeholder.
	Args []string `koanf:"args" yaml:"args"`

	// Env specifies environment variables to set for the command.
	Env map[string]string `koanf:"env" yaml:"env"`

	// PostProcessor is an optional command to pipe stdout through (e.g., "cclean").
	PostProcessor string `koanf:"post_processor" yaml:"post_processor"`
}

// IsValid returns true if the config has at least a command specified.
func (c *CustomAgentConfig) IsValid() bool {
	return c != nil && c.Command != ""
}

// CustomAgent implements the Agent interface using structured configuration.
// This enables integration of arbitrary CLI tools not built into autospec.
type CustomAgent struct {
	name   string
	config CustomAgentConfig
	caps   Caps
}

// NewCustomAgent creates a CustomAgent from a template string like "claude -p {{PROMPT}}".
// The template is parsed into command and args. Returns an error if invalid.
func NewCustomAgent(template string) (*CustomAgent, error) {
	parts := strings.Fields(template)
	if len(parts) == 0 {
		return nil, fmt.Errorf("custom agent: empty template")
	}

	cfg := CustomAgentConfig{
		Command: parts[0],
		Args:    parts[1:],
	}
	return NewCustomAgentFromConfig(cfg)
}

// NewCustomAgentFromConfig creates a new CustomAgent from structured configuration.
// Returns an error if the configuration is invalid.
func NewCustomAgentFromConfig(cfg CustomAgentConfig) (*CustomAgent, error) {
	if cfg.Command == "" {
		return nil, fmt.Errorf("custom agent: command is required")
	}

	// Validate that {{PROMPT}} appears somewhere in args
	hasPrompt := false
	for _, arg := range cfg.Args {
		if strings.Contains(arg, promptPlaceholder) {
			hasPrompt = true
			break
		}
	}
	if !hasPrompt {
		return nil, fmt.Errorf("custom agent: args must contain %s placeholder", promptPlaceholder)
	}

	return &CustomAgent{
		name:   "custom",
		config: cfg,
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

// Validate checks that the command and post-processor exist in PATH.
func (c *CustomAgent) Validate() error {
	// Check main command exists
	if _, err := exec.LookPath(c.config.Command); err != nil {
		return fmt.Errorf("custom agent: command %q not found in PATH", c.config.Command)
	}

	// Check post-processor exists if specified
	if c.config.PostProcessor != "" {
		if _, err := exec.LookPath(c.config.PostProcessor); err != nil {
			return fmt.Errorf("custom agent: post_processor %q not found in PATH", c.config.PostProcessor)
		}
	}

	return nil
}

// Capabilities returns the agent's capability flags.
func (c *CustomAgent) Capabilities() Caps {
	return c.caps
}

// BuildCommand constructs an exec.Cmd by expanding args with the prompt.
// If a post-processor is configured, it wraps the command in a shell pipe.
func (c *CustomAgent) BuildCommand(prompt string, opts ExecOptions) (*exec.Cmd, error) {
	// Expand {{PROMPT}} in args
	expandedArgs := make([]string, len(c.config.Args))
	for i, arg := range c.config.Args {
		expandedArgs[i] = strings.ReplaceAll(arg, promptPlaceholder, prompt)
	}

	var cmd *exec.Cmd
	if c.config.PostProcessor != "" {
		// Use shell to handle piping
		cmdLine := c.buildShellCommand(expandedArgs)
		cmd = exec.Command("sh", "-c", cmdLine)
	} else {
		// Direct execution without shell
		cmd = exec.Command(c.config.Command, expandedArgs...)
	}

	c.configureCmd(cmd, opts)
	return cmd, nil
}

// buildShellCommand constructs a shell command string with proper escaping.
func (c *CustomAgent) buildShellCommand(expandedArgs []string) string {
	// Build the main command with escaped args
	parts := make([]string, 0, len(expandedArgs)+1)
	parts = append(parts, escapeShellArg(c.config.Command))
	for _, arg := range expandedArgs {
		parts = append(parts, escapeShellArg(arg))
	}

	cmdLine := strings.Join(parts, " ")

	// Add pipe to post-processor
	cmdLine += " | " + escapeShellArg(c.config.PostProcessor)

	return cmdLine
}

// escapeShellArg escapes a string for safe use in a shell command.
func escapeShellArg(s string) string {
	// Use single quotes, escaping any embedded single quotes
	escaped := strings.ReplaceAll(s, "'", `'\''`)
	return "'" + escaped + "'"
}

// configureCmd sets working directory and environment on the command.
func (c *CustomAgent) configureCmd(cmd *exec.Cmd, opts ExecOptions) {
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}

	// Start with current environment
	cmd.Env = os.Environ()

	// Force subscription mode: set ANTHROPIC_API_KEY to empty string
	// This prevents accidental API charges when users have API keys in their shell
	if opts.UseSubscription {
		cmd.Env = append(cmd.Env, "ANTHROPIC_API_KEY=")
	}

	// Add config-level env vars (can override subscription mode if explicitly set)
	for k, v := range c.config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Add execution-time env vars (can override config-level)
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

	// Use provided writers or capture to buffers
	var stdout, stderr io.Writer = &stdoutBuf, &stderrBuf
	if opts.Stdout != nil {
		stdout = io.MultiWriter(opts.Stdout, &stdoutBuf)
	}
	if opts.Stderr != nil {
		stderr = io.MultiWriter(opts.Stderr, &stderrBuf)
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

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
