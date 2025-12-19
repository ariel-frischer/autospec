package cliagent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// BaseAgent provides shared implementation for common agent operations.
// Embed this in concrete agent types to reuse validation, version, and execution logic.
type BaseAgent struct {
	// AgentName is the unique identifier for this agent.
	AgentName string

	// Cmd is the CLI command name (e.g., "claude", "gemini").
	Cmd string

	// VersionFlag is the flag to get the version (e.g., "--version", "-v").
	// Empty if the agent doesn't support version checking.
	VersionFlag string

	// AgentCaps contains the agent's capabilities.
	AgentCaps Caps
}

// Name returns the agent's unique identifier.
func (b *BaseAgent) Name() string {
	return b.AgentName
}

// Capabilities returns the agent's capability flags.
func (b *BaseAgent) Capabilities() Caps {
	return b.AgentCaps
}

// Version executes the CLI with the version flag and returns the version string.
func (b *BaseAgent) Version() (string, error) {
	if b.VersionFlag == "" {
		return "unknown", nil
	}
	cmd := exec.Command(b.Cmd, b.VersionFlag)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting version for %s: %w", b.AgentName, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Validate checks if the CLI is in PATH and required environment variables are set.
func (b *BaseAgent) Validate() error {
	if _, err := exec.LookPath(b.Cmd); err != nil {
		return fmt.Errorf("%s: CLI %q not found in PATH (install it or check your PATH)", b.AgentName, b.Cmd)
	}
	for _, envVar := range b.AgentCaps.RequiredEnv {
		if os.Getenv(envVar) == "" {
			return fmt.Errorf("%s: required environment variable %s is not set", b.AgentName, envVar)
		}
	}
	return nil
}

// BuildCommand constructs an exec.Cmd based on the agent's PromptDelivery method.
func (b *BaseAgent) BuildCommand(prompt string, opts ExecOptions) (*exec.Cmd, error) {
	args := b.buildArgs(prompt, opts)
	cmd := exec.Command(b.Cmd, args...)
	b.configureCmd(cmd, opts)
	return cmd, nil
}

// buildArgs constructs the command arguments based on prompt delivery method.
func (b *BaseAgent) buildArgs(prompt string, opts ExecOptions) []string {
	var args []string
	pd := b.AgentCaps.PromptDelivery

	switch pd.Method {
	case PromptMethodArg:
		args = append(args, pd.Flag, prompt)
	case PromptMethodPositional:
		args = append(args, prompt)
	case PromptMethodSubcommand:
		args = append(args, pd.Flag, prompt)
	case PromptMethodSubcommandArg:
		args = append(args, pd.Flag, pd.PromptFlag, prompt)
	}

	args = b.appendAutonomousArgs(args, opts)
	args = append(args, opts.ExtraArgs...)
	return args
}

// appendAutonomousArgs adds autonomous mode flags if enabled.
func (b *BaseAgent) appendAutonomousArgs(args []string, opts ExecOptions) []string {
	if !opts.Autonomous {
		return args
	}
	if b.AgentCaps.AutonomousFlag != "" {
		args = append(args, b.AgentCaps.AutonomousFlag)
	}
	return args
}

// configureCmd sets working directory and environment on the command.
func (b *BaseAgent) configureCmd(cmd *exec.Cmd, opts ExecOptions) {
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}
	cmd.Env = b.buildEnv(opts)
}

// buildEnv merges process environment with opts.Env and autonomous env vars.
func (b *BaseAgent) buildEnv(opts ExecOptions) []string {
	env := os.Environ()

	// Add autonomous env vars
	if opts.Autonomous {
		for k, v := range b.AgentCaps.AutonomousEnv {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Add user-provided env vars (overrides)
	for k, v := range opts.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
}

// Execute builds and runs the command, returning the result.
func (b *BaseAgent) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Result, error) {
	cmd, err := b.BuildCommand(prompt, opts)
	if err != nil {
		return nil, fmt.Errorf("building command: %w", err)
	}

	return b.runCommand(ctx, cmd, opts)
}

// runCommand executes the command and captures output.
func (b *BaseAgent) runCommand(ctx context.Context, cmd *exec.Cmd, opts ExecOptions) (*Result, error) {
	ctx, cancel := b.applyTimeout(ctx, opts)
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
		return nil, fmt.Errorf("starting %s: %w", b.AgentName, err)
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
		<-done // Wait for goroutine to exit
		return nil, fmt.Errorf("executing %s: %w", b.AgentName, ctx.Err())
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
			return nil, fmt.Errorf("executing %s: %w", b.AgentName, err)
		}
	}
	return result, nil
}

// applyTimeout returns a context with timeout if opts.Timeout is set.
func (b *BaseAgent) applyTimeout(ctx context.Context, opts ExecOptions) (context.Context, context.CancelFunc) {
	if opts.Timeout > 0 {
		return context.WithTimeout(ctx, opts.Timeout)
	}
	return ctx, func() {}
}
