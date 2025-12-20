// Package workflow tests Claude command execution via the Agent interface.
// Related: internal/workflow/claude.go
// Tags: workflow, claude, execution, timeout, agent
package workflow

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/cliagent"
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
