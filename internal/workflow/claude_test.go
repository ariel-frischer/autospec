// Package workflow tests Claude command execution via the Agent interface.
// Related: internal/workflow/claude.go
// Tags: workflow, claude, execution, timeout, agent
package workflow

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/cliagent"
	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClaudeExecutor_Execute_NoAgent tests error handling when no agent is configured
func TestClaudeExecutor_Execute_NoAgent(t *testing.T) {
	t.Parallel()

	executor := &ClaudeExecutor{
		Agent: nil,
	}

	err := executor.Execute("test prompt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no agent configured")
}

// TestClaudeExecutor_StreamCommand_NoAgent tests error handling when no agent is configured
func TestClaudeExecutor_StreamCommand_NoAgent(t *testing.T) {
	t.Parallel()

	executor := &ClaudeExecutor{
		Agent: nil,
	}

	var stdout, stderr bytes.Buffer
	err := executor.StreamCommand("test prompt", &stdout, &stderr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no agent configured")
}

// TestClaudeExecutor_FormatCommand_NoAgent tests FormatCommand with no agent
func TestClaudeExecutor_FormatCommand_NoAgent(t *testing.T) {
	t.Parallel()

	executor := &ClaudeExecutor{
		Agent: nil,
	}

	result := executor.FormatCommand("test prompt")
	assert.Equal(t, "[no agent configured]", result)
}

// TestClaudeExecutor_Execute_WithAgent tests successful execution with an agent
func TestClaudeExecutor_Execute_WithAgent(t *testing.T) {
	t.Parallel()

	// Use the built-in echo "agent" for testing
	agent := cliagent.Get("claude")
	require.NotNil(t, agent, "claude agent should be registered")

	// Create a custom agent that uses echo for testing
	customAgent, err := cliagent.NewCustomAgentFromConfig(cliagent.CustomAgentConfig{
		Command: "echo",
		Args:    []string{"{{PROMPT}}"},
	})
	require.NoError(t, err)

	executor := &ClaudeExecutor{
		Agent:   customAgent,
		Timeout: 60,
	}

	err = executor.Execute("test prompt")
	assert.NoError(t, err)
}

// TestClaudeExecutor_StreamCommand_WithAgent tests streaming execution with an agent
func TestClaudeExecutor_StreamCommand_WithAgent(t *testing.T) {
	t.Parallel()

	customAgent, err := cliagent.NewCustomAgentFromConfig(cliagent.CustomAgentConfig{
		Command: "echo",
		Args:    []string{"{{PROMPT}}"},
	})
	require.NoError(t, err)

	executor := &ClaudeExecutor{
		Agent:   customAgent,
		Timeout: 60,
	}

	var stdout, stderr bytes.Buffer
	err = executor.StreamCommand("test prompt", &stdout, &stderr)
	assert.NoError(t, err)
	assert.Contains(t, stdout.String(), "test prompt")
}

func TestClaudeExecutor_StreamCommand_CodexCompactUsesJSONFormatter(t *testing.T) {
	t.Parallel()

	executor := &ClaudeExecutor{
		Agent:       &codexOutputAgent{stdout: `{"type":"item.completed","item":{"type":"agent_message","text":"hello from json"}}`},
		CodexOutput: config.CodexOutputConfig{Mode: "compact", MaxLinesPerMessage: 40},
	}

	var stdout, stderr bytes.Buffer
	err := executor.StreamCommand("test prompt", &stdout, &stderr)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Agent message")
	assert.Contains(t, stdout.String(), "hello from json")
	assert.True(t, executor.Agent.(*codexOutputAgent).jsonOutput)
}

func TestClaudeExecutor_StreamCommand_CodexFullBypassesFormatter(t *testing.T) {
	t.Parallel()

	executor := &ClaudeExecutor{
		Agent:       &codexOutputAgent{stdout: `{"type":"item.completed","item":{"type":"agent_message","text":"native output"}}`},
		CodexOutput: config.CodexOutputConfig{Mode: "full", MaxLinesPerMessage: 40},
	}

	var stdout, stderr bytes.Buffer
	err := executor.StreamCommand("test prompt", &stdout, &stderr)
	require.NoError(t, err)
	assert.NotContains(t, stdout.String(), "Agent message")
	assert.Contains(t, stdout.String(), `"native output"`)
	assert.False(t, executor.Agent.(*codexOutputAgent).jsonOutput)
}

type codexOutputAgent struct {
	stdout     string
	jsonOutput bool
}

func (a *codexOutputAgent) Name() string { return "codex" }

func (a *codexOutputAgent) Version() (string, error) { return "test", nil }

func (a *codexOutputAgent) Validate() error { return nil }

func (a *codexOutputAgent) Capabilities() cliagent.Caps {
	return cliagent.Caps{Automatable: true}
}

func (a *codexOutputAgent) BuildCommand(prompt string, opts cliagent.ExecOptions) (*exec.Cmd, error) {
	return exec.Command("codex", "exec", prompt), nil
}

func (a *codexOutputAgent) Execute(ctx context.Context, prompt string, opts cliagent.ExecOptions) (*cliagent.Result, error) {
	a.jsonOutput = opts.JSONOutput
	if opts.Stdout != nil {
		_, _ = io.WriteString(opts.Stdout, a.stdout)
	}
	return &cliagent.Result{}, nil
}

// TestClaudeExecutor_Timeout tests timeout enforcement
func TestClaudeExecutor_Timeout(t *testing.T) {
	t.Parallel()

	customAgent, err := cliagent.NewCustomAgentFromConfig(cliagent.CustomAgentConfig{
		Command: "sleep",
		Args:    []string{"{{PROMPT}}"},
	})
	require.NoError(t, err)

	executor := &ClaudeExecutor{
		Agent:   customAgent,
		Timeout: 1, // 1 second timeout
	}

	// Sleep for 10 seconds (will be killed after 1 second)
	err = executor.Execute("10")
	require.Error(t, err)

	// Verify it's a TimeoutError
	var timeoutErr *TimeoutError
	assert.True(t, errors.As(err, &timeoutErr), "Error should be TimeoutError")
}

// TestClaudeExecutor_Timeout_CompletesBeforeTimeout tests command completing before timeout
func TestClaudeExecutor_Timeout_CompletesBeforeTimeout(t *testing.T) {
	t.Parallel()

	customAgent, err := cliagent.NewCustomAgentFromConfig(cliagent.CustomAgentConfig{
		Command: "echo",
		Args:    []string{"{{PROMPT}}"},
	})
	require.NoError(t, err)

	executor := &ClaudeExecutor{
		Agent:   customAgent,
		Timeout: 60, // 60 seconds - plenty of time for echo
	}

	err = executor.Execute("test")
	assert.NoError(t, err, "Command should complete before timeout")
}

// TestClaudeExecutor_NoTimeout tests execution without timeout
func TestClaudeExecutor_NoTimeout(t *testing.T) {
	t.Parallel()

	customAgent, err := cliagent.NewCustomAgentFromConfig(cliagent.CustomAgentConfig{
		Command: "echo",
		Args:    []string{"{{PROMPT}}"},
	})
	require.NoError(t, err)

	executor := &ClaudeExecutor{
		Agent:   customAgent,
		Timeout: 0, // No timeout
	}

	err = executor.Execute("test")
	assert.NoError(t, err)
}

// TestExecuteSpecKitCommand tests the convenience wrapper
func TestExecuteSpecKitCommand(t *testing.T) {
	t.Parallel()

	customAgent, err := cliagent.NewCustomAgentFromConfig(cliagent.CustomAgentConfig{
		Command: "echo",
		Args:    []string{"{{PROMPT}}"},
	})
	require.NoError(t, err)

	executor := &ClaudeExecutor{
		Agent: customAgent,
	}

	err = executor.ExecuteSpecKitCommand("/autospec.specify \"test\"")
	assert.NoError(t, err)
}

// TestTimeoutError_IncludesMetadata tests that timeout errors include metadata
func TestTimeoutError_IncludesMetadata(t *testing.T) {
	t.Parallel()

	customAgent, err := cliagent.NewCustomAgentFromConfig(cliagent.CustomAgentConfig{
		Command: "sleep",
		Args:    []string{"{{PROMPT}}"},
	})
	require.NoError(t, err)

	executor := &ClaudeExecutor{
		Agent:   customAgent,
		Timeout: 1,
	}

	err = executor.Execute("5")
	require.Error(t, err)

	var timeoutErr *TimeoutError
	require.True(t, errors.As(err, &timeoutErr), "Error should be TimeoutError")

	// Verify metadata
	assert.Equal(t, 1*time.Second, timeoutErr.Timeout)
	assert.Contains(t, timeoutErr.Command, "sleep")
	assert.Equal(t, context.DeadlineExceeded, timeoutErr.Err)
}

// TestStreamCommand_Timeout tests streaming with timeout enforcement
func TestStreamCommand_Timeout(t *testing.T) {
	t.Parallel()

	customAgent, err := cliagent.NewCustomAgentFromConfig(cliagent.CustomAgentConfig{
		Command: "sleep",
		Args:    []string{"{{PROMPT}}"},
	})
	require.NoError(t, err)

	executor := &ClaudeExecutor{
		Agent:   customAgent,
		Timeout: 1,
	}

	var stdout, stderr bytes.Buffer
	err = executor.StreamCommand("10", &stdout, &stderr)

	require.Error(t, err)

	var timeoutErr *TimeoutError
	assert.True(t, errors.As(err, &timeoutErr), "Error should be TimeoutError")
}

// Tests for stream-json detection

func TestHasStreamJsonFormat(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args []string
		want bool
	}{
		"long form with separate value": {
			args: []string{"-p", "--output-format", "stream-json"},
			want: true,
		},
		"short form with separate value": {
			args: []string{"-p", "-o", "stream-json"},
			want: true,
		},
		"combined form": {
			args: []string{"-p", "--output-format=stream-json"},
			want: true,
		},
		"no stream-json": {
			args: []string{"-p", "--output-format", "json"},
			want: false,
		},
		"stream-json without flag": {
			args: []string{"stream-json"},
			want: false,
		},
		"empty args": {
			args: []string{},
			want: false,
		},
		"nil args": {
			args: nil,
			want: false,
		},
		"output-format at end without value": {
			args: []string{"-p", "--output-format"},
			want: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := hasStreamJsonFormat(tt.args)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHasHeadlessFlag(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args []string
		want bool
	}{
		"has -p flag": {
			args: []string{"-p", "--output-format", "stream-json"},
			want: true,
		},
		"no -p flag": {
			args: []string{"--output-format", "stream-json"},
			want: false,
		},
		"empty args": {
			args: []string{},
			want: false,
		},
		"nil args": {
			args: nil,
			want: false,
		},
		"-p in middle": {
			args: []string{"--verbose", "-p", "--output-format", "stream-json"},
			want: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := hasHeadlessFlag(tt.args)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDetectStreamJsonMode(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agentArgs []string
		want      bool
	}{
		"stream-json with headless": {
			agentArgs: []string{"-p", "--output-format", "stream-json", "{{PROMPT}}"},
			want:      true,
		},
		"stream-json without headless": {
			agentArgs: []string{"--output-format", "stream-json", "{{PROMPT}}"},
			want:      false,
		},
		"headless without stream-json": {
			agentArgs: []string{"-p", "--output-format", "json", "{{PROMPT}}"},
			want:      false,
		},
		"neither": {
			agentArgs: []string{"--output-format", "json", "{{PROMPT}}"},
			want:      false,
		},
		"short form both": {
			agentArgs: []string{"-p", "-o", "stream-json", "{{PROMPT}}"},
			want:      true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			customAgent, err := cliagent.NewCustomAgentFromConfig(cliagent.CustomAgentConfig{
				Command: "echo",
				Args:    tt.agentArgs,
			})
			require.NoError(t, err)

			executor := &ClaudeExecutor{
				Agent: customAgent,
			}
			got := executor.detectStreamJsonMode()
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestFormatCommand tests the FormatCommand method with an agent
func TestFormatCommand(t *testing.T) {
	t.Parallel()

	customAgent, err := cliagent.NewCustomAgentFromConfig(cliagent.CustomAgentConfig{
		Command: "claude",
		Args:    []string{"-p", "{{PROMPT}}"},
	})
	require.NoError(t, err)

	executor := &ClaudeExecutor{
		Agent: customAgent,
	}

	result := executor.FormatCommand("/autospec.plan")
	assert.Contains(t, result, "claude")
	assert.Contains(t, result, "-p")
}

// Tests for cclean.style configuration

func TestGetFormatterOptions_CcleanStyle(t *testing.T) {
	// Test that cclean.style is used to determine the output style
	t.Parallel()

	tests := map[string]struct {
		ccleanStyle string
		wantStyle   config.OutputStyle
		description string
	}{
		"compact style": {
			ccleanStyle: "compact",
			wantStyle:   config.OutputStyleCompact,
			description: "cclean.style compact should work",
		},
		"minimal style": {
			ccleanStyle: "minimal",
			wantStyle:   config.OutputStyleMinimal,
			description: "cclean.style minimal should work",
		},
		"plain style": {
			ccleanStyle: "plain",
			wantStyle:   config.OutputStylePlain,
			description: "cclean.style plain should work",
		},
		"empty defaults to default style": {
			ccleanStyle: "",
			wantStyle:   config.OutputStyleDefault,
			description: "Empty cclean.style should use default",
		},
		"explicit default style": {
			ccleanStyle: "default",
			wantStyle:   config.OutputStyleDefault,
			description: "cclean.style 'default' should use default style",
		},
		"invalid style defaults to default": {
			ccleanStyle: "invalid_style_value",
			wantStyle:   config.OutputStyleDefault,
			description: "Invalid cclean.style should fall back to default",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			executor := &ClaudeExecutor{
				CcleanConfig: config.CcleanConfig{
					Style: tt.ccleanStyle,
				},
			}

			opts := executor.getFormatterOptions()
			assert.Equal(t, tt.wantStyle, opts.Style, tt.description)
		})
	}
}

func TestGetFormatterOptions_VerboseAndLineNumbers(t *testing.T) {
	// Test that verbose and line_numbers are passed through from CcleanConfig
	t.Parallel()

	tests := map[string]struct {
		verbose         bool
		lineNumbers     bool
		wantVerbose     bool
		wantLineNumbers bool
	}{
		"both enabled": {
			verbose:         true,
			lineNumbers:     true,
			wantVerbose:     true,
			wantLineNumbers: true,
		},
		"both disabled": {
			verbose:         false,
			lineNumbers:     false,
			wantVerbose:     false,
			wantLineNumbers: false,
		},
		"verbose only": {
			verbose:         true,
			lineNumbers:     false,
			wantVerbose:     true,
			wantLineNumbers: false,
		},
		"line_numbers only": {
			verbose:         false,
			lineNumbers:     true,
			wantVerbose:     false,
			wantLineNumbers: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			executor := &ClaudeExecutor{
				CcleanConfig: config.CcleanConfig{
					Style:       "default",
					Verbose:     tt.verbose,
					LineNumbers: tt.lineNumbers,
				},
			}

			opts := executor.getFormatterOptions()
			assert.Equal(t, tt.wantVerbose, opts.Verbose)
			assert.Equal(t, tt.wantLineNumbers, opts.LineNumbers)
		})
	}
}

func TestGetFormatterOptions_FullConfig(t *testing.T) {
	// Test a full configuration with all options set
	t.Parallel()

	executor := &ClaudeExecutor{
		CcleanConfig: config.CcleanConfig{
			Verbose:     true,
			LineNumbers: true,
			Style:       "compact",
		},
	}

	opts := executor.getFormatterOptions()

	assert.Equal(t, config.OutputStyleCompact, opts.Style, "cclean.style should work")
	assert.True(t, opts.Verbose, "verbose should be passed through")
	assert.True(t, opts.LineNumbers, "line_numbers should be passed through")
}

// mockCapturingAgent is a test agent that captures ExecOptions for verification
type mockCapturingAgent struct {
	capturedOpts cliagent.ExecOptions
	name         string
}

func (m *mockCapturingAgent) Name() string             { return m.name }
func (m *mockCapturingAgent) Version() (string, error) { return "1.0.0", nil }
func (m *mockCapturingAgent) Validate() error          { return nil }
func (m *mockCapturingAgent) Capabilities() cliagent.Caps {
	return cliagent.Caps{AutonomousFlag: "--dangerously-skip-permissions"}
}
func (m *mockCapturingAgent) BuildCommand(_ string, _ cliagent.ExecOptions) (*exec.Cmd, error) {
	return exec.Command("echo", "test"), nil
}
func (m *mockCapturingAgent) Execute(_ context.Context, _ string, opts cliagent.ExecOptions) (*cliagent.Result, error) {
	m.capturedOpts = opts
	return &cliagent.Result{ExitCode: 0}, nil
}

// TestClaudeExecutor_SkipPermissions_SetsAutonomous verifies that SkipPermissions
// is correctly passed to ExecOptions.Autonomous during execution
func TestClaudeExecutor_SkipPermissions_SetsAutonomous(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		skipPermissions bool
		wantAutonomous  bool
	}{
		"skip_permissions true sets autonomous true": {
			skipPermissions: true,
			wantAutonomous:  true,
		},
		"skip_permissions false sets autonomous false": {
			skipPermissions: false,
			wantAutonomous:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockAgent := &mockCapturingAgent{name: "test-agent"}
			executor := &ClaudeExecutor{
				Agent:           mockAgent,
				SkipPermissions: tt.skipPermissions,
				Timeout:         60,
			}

			err := executor.Execute("test prompt")
			require.NoError(t, err)

			assert.Equal(t, tt.wantAutonomous, mockAgent.capturedOpts.Autonomous,
				"ExecOptions.Autonomous should match SkipPermissions value")
		})
	}
}

// TestClaudeExecutor_StreamCommand_SkipPermissions_SetsAutonomous verifies that
// StreamCommand also passes SkipPermissions to ExecOptions.Autonomous
func TestClaudeExecutor_StreamCommand_SkipPermissions_SetsAutonomous(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		skipPermissions bool
		wantAutonomous  bool
	}{
		"skip_permissions true sets autonomous true in stream": {
			skipPermissions: true,
			wantAutonomous:  true,
		},
		"skip_permissions false sets autonomous false in stream": {
			skipPermissions: false,
			wantAutonomous:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockAgent := &mockCapturingAgent{name: "test-agent"}
			executor := &ClaudeExecutor{
				Agent:           mockAgent,
				SkipPermissions: tt.skipPermissions,
				Timeout:         60,
			}

			var stdout, stderr bytes.Buffer
			err := executor.StreamCommand("test prompt", &stdout, &stderr)
			require.NoError(t, err)

			assert.Equal(t, tt.wantAutonomous, mockAgent.capturedOpts.Autonomous,
				"ExecOptions.Autonomous should match SkipPermissions value in StreamCommand")
		})
	}
}

func TestClaudeExecutor_OpenCodeAgentPassesThrough(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		run      func(*ClaudeExecutor, *mockCapturingAgent) error
		wantName string
	}{
		"execute": {
			run: func(executor *ClaudeExecutor, _ *mockCapturingAgent) error {
				return executor.Execute("test prompt")
			},
			wantName: "build",
		},
		"stream command": {
			run: func(executor *ClaudeExecutor, _ *mockCapturingAgent) error {
				var stdout, stderr bytes.Buffer
				return executor.StreamCommand("test prompt", &stdout, &stderr)
			},
			wantName: "build",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockAgent := &mockCapturingAgent{name: "opencode"}
			executor := &ClaudeExecutor{
				Agent:         mockAgent,
				OpenCodeAgent: "build",
				Timeout:       60,
			}

			err := tt.run(executor, mockAgent)
			require.NoError(t, err)

			assert.Equal(t, tt.wantName, mockAgent.capturedOpts.OpenCodeAgent)
		})
	}
}
