package integration

import (
	"bytes"
	"testing"

	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCustomCommandExecution tests custom Claude command execution
func TestCustomCommandExecution(t *testing.T) {
	tests := map[string]struct {
		customCmd string
		prompt    string
		wantErr   bool
	}{
		"simple echo command": {
			customCmd: "echo {{PROMPT}}",
			prompt:    "test prompt",
			wantErr:   false,
		},
		"command with pipe": {
			customCmd: "echo {{PROMPT}} | cat",
			prompt:    "/autospec.specify \"test\"",
			wantErr:   false,
		},
		"command with env var": {
			customCmd: "TEST_VAR=\"value\" echo {{PROMPT}}",
			prompt:    "/autospec.plan",
			wantErr:   false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			executor := &workflow.ClaudeExecutor{
				CustomClaudeCmd: tc.customCmd,
			}

			var stdout, stderr bytes.Buffer
			err := executor.StreamCommand(tc.prompt, &stdout, &stderr)

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
	executor := &workflow.ClaudeExecutor{
		CustomClaudeCmd: "echo {{PROMPT}} | grep 'test' || echo 'no match'",
	}

	var stdout, stderr bytes.Buffer
	err := executor.StreamCommand("this is a test", &stdout, &stderr)

	require.NoError(t, err)
	output := stdout.String()

	// Should match and print the line
	assert.Contains(t, output, "test")
}

// TestCustomCommandWithEnvironmentVariable tests env var prefix handling
func TestCustomCommandWithEnvironmentVariable(t *testing.T) {
	executor := &workflow.ClaudeExecutor{
		CustomClaudeCmd: "TEST_KEY=\"secret\" echo {{PROMPT}}",
	}

	var stdout, stderr bytes.Buffer
	err := executor.StreamCommand("/autospec.plan", &stdout, &stderr)

	require.NoError(t, err)
	output := stdout.String()

	// Should execute successfully and print prompt
	assert.Contains(t, output, "/autospec.plan")
}

// TestFallbackToSimpleMode tests fallback when custom command is not set
func TestFallbackToSimpleMode(t *testing.T) {
	executor := &workflow.ClaudeExecutor{
		ClaudeCmd:       "echo",
		ClaudeArgs:      []string{"-n"},
		CustomClaudeCmd: "", // Empty means use simple mode
	}

	var stdout, stderr bytes.Buffer
	err := executor.StreamCommand("simple mode test", &stdout, &stderr)

	require.NoError(t, err)
	output := stdout.String()

	// Should contain the prompt
	assert.Contains(t, output, "simple mode test")
}
