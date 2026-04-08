package cliagent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
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

// sanitizePromptForCLI ensures prompt content won't be misinterpreted as CLI flags
// and removes dangerous characters.
//
// Security measures:
// 1. Strips null bytes - prevent string truncation attacks in C-based systems
// 2. Prepends newline to prompts starting with "-" - prevent flag interpretation
func sanitizePromptForCLI(prompt string) string {
	// Strip null bytes (string terminators that could truncate arguments)
	prompt = strings.ReplaceAll(prompt, "\x00", "")

	// Prevent flag interpretation for prompts starting with "-"
	if strings.HasPrefix(prompt, "-") {
		return "\n" + prompt
	}
	return prompt
}

// buildArgs constructs the command arguments based on prompt delivery method.
// For interactive mode, uses positional argument instead of -p flag to enable
// multi-turn conversation in Claude Code.
func (b *BaseAgent) buildArgs(prompt string, opts ExecOptions) []string {
	var args []string
	pd := b.AgentCaps.PromptDelivery

	// Defensive: ensure prompt doesn't start with "-" which CLI parsers interpret as flags.
	// This prevents issues like "---" (YAML frontmatter) being parsed as an option.
	prompt = sanitizePromptForCLI(prompt)

	// Interactive mode: use InteractiveFlag if set, otherwise positional argument
	// Automated mode: use configured prompt delivery method (e.g., -p flag)
	if opts.Interactive {
		if pd.InteractiveFlag != "" {
			// Some agents (e.g., OpenCode) need a flag for interactive mode
			args = append(args, pd.InteractiveFlag, prompt)
		} else {
			// Default: positional argument (enables multi-turn conversation in Claude)
			args = append(args, prompt)
		}
	} else {
		switch pd.Method {
		case PromptMethodArg:
			args = append(args, pd.Flag, prompt)
		case PromptMethodPositional:
			args = append(args, prompt)
		case PromptMethodSubcommand:
			args = append(args, pd.Flag, prompt)
		case PromptMethodSubcommandArg:
			args = append(args, pd.Flag, pd.PromptFlag, prompt)
		case PromptMethodSubcommandWithFlag:
			// Pattern: <agent> <subcommand> <prompt> [-f <file>] <command-flag> <command-name>
			// Example: opencode run "fix bug" -f context.yaml --command autospec.specify
			// Parse slash commands to extract command name, context file, and actual prompt
			parts := parseSlashCommandFull(prompt)

			// Build the message: include phase info if present
			message := parts.Prompt
			if parts.Phase != "" {
				if message != "" {
					message = "phase " + parts.Phase + " " + message
				} else {
					message = "phase " + parts.Phase
				}
			}

			args = append(args, pd.Flag, message)

			// Add context file using agent's ContextFileFlag if available
			if parts.ContextFile != "" && pd.ContextFileFlag != "" {
				args = append(args, pd.ContextFileFlag, parts.ContextFile)
			}

			if parts.CmdName != "" && pd.CommandFlag != "" {
				args = append(args, pd.CommandFlag, parts.CmdName)
			}

			if opts.OpenCodeAgent != "" {
				args = append(args, "--agent", opts.OpenCodeAgent)
			}
		}
		// Add default args (e.g., --verbose --output-format stream-json for Claude)
		// Only in automated mode - interactive mode omits these for conversation
		args = append(args, b.AgentCaps.DefaultArgs...)
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

// slashCommandParts holds parsed components of a slash command.
type slashCommandParts struct {
	CmdName     string // Command name (e.g., "autospec.implement")
	Prompt      string // Remaining prompt text
	Phase       string // Value of --phase flag if present
	ContextFile string // Value of --context-file flag if present
}

// parseSlashCommand extracts command name and actual prompt from a slash command.
// Input: `/autospec.specify "feature description"` or `/autospec.plan`
// Returns: ("autospec.specify", "feature description") or ("autospec.plan", "")
// If not a slash command, returns ("", original prompt).
func parseSlashCommand(prompt string) (cmdName, actualPrompt string) {
	parts := parseSlashCommandFull(prompt)
	return parts.CmdName, parts.Prompt
}

// parseSlashCommandFull extracts all components from a slash command.
// Input: `/autospec.implement --phase 1 --context-file path/to/file "prompt"`
// Returns slashCommandParts with CmdName, Phase, ContextFile, and remaining Prompt.
// If not a slash command, returns empty CmdName with original prompt.
func parseSlashCommandFull(prompt string) slashCommandParts {
	prompt = strings.TrimSpace(prompt)
	if !strings.HasPrefix(prompt, "/") {
		return slashCommandParts{Prompt: prompt}
	}

	// Find end of command name (space or end of string)
	rest := prompt[1:] // Remove leading /
	spaceIdx := strings.Index(rest, " ")
	if spaceIdx == -1 {
		// No space - entire string is the command name
		return slashCommandParts{CmdName: rest}
	}

	parts := slashCommandParts{
		CmdName: rest[:spaceIdx],
	}
	remaining := strings.TrimSpace(rest[spaceIdx+1:])

	// Parse out --phase and --context-file flags
	parts.Phase, remaining = extractFlag(remaining, "--phase")
	parts.ContextFile, remaining = extractFlag(remaining, "--context-file")

	// Clean up remaining prompt
	parts.Prompt = strings.TrimSpace(remaining)

	// Remove surrounding quotes if present
	if len(parts.Prompt) >= 2 {
		if (parts.Prompt[0] == '"' && parts.Prompt[len(parts.Prompt)-1] == '"') ||
			(parts.Prompt[0] == '\'' && parts.Prompt[len(parts.Prompt)-1] == '\'') {
			parts.Prompt = parts.Prompt[1 : len(parts.Prompt)-1]
		}
	}

	return parts
}

// extractFlag extracts a flag value from a string and returns (value, remaining).
// Handles both --flag=value and --flag value formats.
func extractFlag(s, flagName string) (value, remaining string) {
	// Try --flag=value format
	eqPrefix := flagName + "="
	if idx := strings.Index(s, eqPrefix); idx != -1 {
		start := idx + len(eqPrefix)
		end := findValueEnd(s, start)
		value = s[start:end]
		remaining = strings.TrimSpace(s[:idx] + s[end:])
		return value, remaining
	}

	// Try --flag value format
	if idx := strings.Index(s, flagName); idx != -1 {
		afterFlag := idx + len(flagName)
		if afterFlag < len(s) && (s[afterFlag] == ' ' || s[afterFlag] == '\t') {
			// Skip whitespace to find value start
			valueStart := afterFlag
			for valueStart < len(s) && (s[valueStart] == ' ' || s[valueStart] == '\t') {
				valueStart++
			}
			if valueStart < len(s) && s[valueStart] != '-' {
				end := findValueEnd(s, valueStart)
				value = s[valueStart:end]
				remaining = strings.TrimSpace(s[:idx] + s[end:])
				return value, remaining
			}
		}
	}

	return "", s
}

// findValueEnd finds the end of a value (space or end of string, or end of quotes).
func findValueEnd(s string, start int) int {
	if start >= len(s) {
		return start
	}

	// Handle quoted values
	if s[start] == '"' || s[start] == '\'' {
		quote := s[start]
		end := start + 1
		for end < len(s) && s[end] != quote {
			end++
		}
		if end < len(s) {
			end++ // Include closing quote
		}
		return end
	}

	// Non-quoted: find next space or end
	end := start
	for end < len(s) && s[end] != ' ' && s[end] != '\t' {
		end++
	}
	return end
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

	// Force subscription mode: set ANTHROPIC_API_KEY to empty string
	// This prevents accidental API charges when users have API keys in their shell
	if opts.UseSubscription {
		env = append(env, "ANTHROPIC_API_KEY=")
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
// For interactive mode with ReplaceProcess, uses syscall.Exec to replace the current process,
// giving the agent full terminal control for TUI applications.
func (b *BaseAgent) runCommand(ctx context.Context, cmd *exec.Cmd, opts ExecOptions) (*Result, error) {
	// Interactive mode with process replacement: gives agent full terminal control
	// This is necessary for TUI applications like Claude Code that need raw mode
	if opts.Interactive && opts.ReplaceProcess {
		return b.execInteractive(cmd)
	}

	// Interactive subprocess mode: connect stdin for user input
	// Note: TUI apps may have limited functionality without full terminal control
	if opts.Interactive {
		cmd.Stdin = os.Stdin
	}

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

// execInteractive replaces the current process with the agent command.
// This gives the agent full terminal control, necessary for TUI applications
// like Claude Code that require raw mode for their UI.
// Note: This function does not return on success - the process is replaced.
func (b *BaseAgent) execInteractive(cmd *exec.Cmd) (*Result, error) {
	binary, err := exec.LookPath(cmd.Path)
	if err != nil {
		return nil, fmt.Errorf("finding %s binary: %w", b.AgentName, err)
	}

	// syscall.Exec replaces the current process with the new one
	// This gives the child process full control of the terminal
	err = syscall.Exec(binary, cmd.Args, cmd.Env)
	if err != nil {
		return nil, fmt.Errorf("exec %s: %w", b.AgentName, err)
	}

	// This line is never reached - syscall.Exec replaces the process
	return nil, nil
}

// applyTimeout returns a context with timeout if opts.Timeout is set.
func (b *BaseAgent) applyTimeout(ctx context.Context, opts ExecOptions) (context.Context, context.CancelFunc) {
	if opts.Timeout > 0 {
		return context.WithTimeout(ctx, opts.Timeout)
	}
	return ctx, func() {}
}
