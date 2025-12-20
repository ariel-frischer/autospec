// Package integration_test tests Claude CLI execution with mock responses for workflow stages.
// Related: /home/ari/repos/autospec/internal/workflow/orchestrator.go
// Tags: integration, claude, workflow, mock

package integration

import (
	"bytes"
	"testing"

	"github.com/ariel-frischer/autospec/internal/cliagent"
	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCustomCommandExecution tests custom Claude command execution
func TestCustomCommandExecution(t *testing.T) {
	tests := map[string]struct {
		command string
		args    []string
		prompt  string
		wantErr bool
	}{
		"simple echo command": {
			command: "echo",
			args:    []string{"{{PROMPT}}"},
			prompt:  "test prompt",
			wantErr: false,
		},
		"command with cat pipe simulation": {
			command: "sh",
			args:    []string{"-c", "echo {{PROMPT}} | cat"},
			prompt:  "/autospec.specify \"test\"",
			wantErr: false,
		},
		"command with env var": {
			command: "sh",
			args:    []string{"-c", "TEST_VAR=\"value\" echo {{PROMPT}}"},
			prompt:  "/autospec.plan",
			wantErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			agent, err := cliagent.NewCustomAgentFromConfig(cliagent.CustomAgentConfig{
				Command: tc.command,
				Args:    tc.args,
			})
			require.NoError(t, err)

			executor := &workflow.ClaudeExecutor{
				Agent: agent,
			}

			var stdout, stderr bytes.Buffer
			err = executor.StreamCommand(tc.prompt, &stdout, &stderr)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify output was generated
				output := stdout.String() + stderr.String()
				assert.NotEmpty(t, output)
			}
		})
	}
}

// TestCustomCommandWithPipeOperator tests pipe operator handling in integration
func TestCustomCommandWithPipeOperator(t *testing.T) {
	agent, err := cliagent.NewCustomAgentFromConfig(cliagent.CustomAgentConfig{
		Command: "sh",
		Args:    []string{"-c", "echo {{PROMPT}} | grep 'test' || echo 'no match'"},
	})
	require.NoError(t, err)

	executor := &workflow.ClaudeExecutor{
		Agent: agent,
	}

	var stdout, stderr bytes.Buffer
	err = executor.StreamCommand("this is a test", &stdout, &stderr)

	require.NoError(t, err)
	output := stdout.String()

	// Should match and print the line
	assert.Contains(t, output, "test")
}

// TestCustomCommandWithEnvironmentVariable tests env var prefix handling
func TestCustomCommandWithEnvironmentVariable(t *testing.T) {
	agent, err := cliagent.NewCustomAgentFromConfig(cliagent.CustomAgentConfig{
		Command: "sh",
		Args:    []string{"-c", "TEST_KEY=\"secret\" echo {{PROMPT}}"},
	})
	require.NoError(t, err)

	executor := &workflow.ClaudeExecutor{
		Agent: agent,
	}

	var stdout, stderr bytes.Buffer
	err = executor.StreamCommand("/autospec.plan", &stdout, &stderr)

	require.NoError(t, err)
	output := stdout.String()

	// Should execute successfully and print prompt
	assert.Contains(t, output, "/autospec.plan")
}

// TestFallbackToSimpleMode tests using a simple agent configuration
func TestFallbackToSimpleMode(t *testing.T) {
	agent, err := cliagent.NewCustomAgentFromConfig(cliagent.CustomAgentConfig{
		Command: "echo",
		Args:    []string{"-n", "{{PROMPT}}"},
	})
	require.NoError(t, err)

	executor := &workflow.ClaudeExecutor{
		Agent: agent,
	}

	var stdout, stderr bytes.Buffer
	err = executor.StreamCommand("simple mode test", &stdout, &stderr)

	require.NoError(t, err)
	output := stdout.String()

	// Should contain the prompt
	assert.Contains(t, output, "simple mode test")
}
