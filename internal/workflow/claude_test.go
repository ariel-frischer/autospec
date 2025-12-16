package workflow

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShellQuote tests shell quoting function
func TestShellQuote(t *testing.T) {
	tests := map[string]struct {
		input string
		want  string
	}{
		"simple string": {
			input: "hello",
			want:  "'hello'",
		},
		"string with spaces": {
			input: "hello world",
			want:  "'hello world'",
		},
		"string with double quotes": {
			input: "/autospec.specify \"test\"",
			want:  "'/autospec.specify \"test\"'",
		},
		"string with single quotes": {
			input: "it's a test",
			want:  "'it'\\''s a test'",
		},
		"string with mixed quotes": {
			input: "/autospec.specify \"Feature with 'quotes'\"",
			want:  "'/autospec.specify \"Feature with '\\''quotes'\\''\"'",
		},
		"string with dollar signs": {
			input: "/autospec.specify \"test $var\"",
			want:  "'/autospec.specify \"test $var\"'",
		},
		"multiline feature description": {
			input: "/autospec.specify \"Implement timeout functionality\n  Use 'timeout' config\n  Add context with deadline\"",
			want:  "'/autospec.specify \"Implement timeout functionality\n  Use '\\''timeout'\\'' config\n  Add context with deadline\"'",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := shellQuote(tc.input)
			assert.Equal(t, tc.want, result)
		})
	}
}

// TestExpandTemplate tests template placeholder expansion with shell quoting
func TestExpandTemplate(t *testing.T) {
	tests := map[string]struct {
		template string
		prompt   string
		want     string
	}{
		"simple replacement": {
			template: "claude -p {{PROMPT}}",
			prompt:   "/autospec.specify \"test\"",
			want:     "claude -p '/autospec.specify \"test\"'",
		},
		"with env var prefix": {
			template: "ANTHROPIC_API_KEY=\"\" claude -p {{PROMPT}}",
			prompt:   "/autospec.plan",
			want:     "ANTHROPIC_API_KEY=\"\" claude -p '/autospec.plan'",
		},
		"with pipe": {
			template: "claude -p {{PROMPT}} | claude-clean",
			prompt:   "/autospec.tasks",
			want:     "claude -p '/autospec.tasks' | claude-clean",
		},
		"complex template": {
			template: "ANTHROPIC_API_KEY=\"\" claude -p {{PROMPT}} --verbose | tee output.log",
			prompt:   "/autospec.specify \"Add auth\"",
			want:     "ANTHROPIC_API_KEY=\"\" claude -p '/autospec.specify \"Add auth\"' --verbose | tee output.log",
		},
		"prompt with single quotes": {
			template: "claude -p {{PROMPT}}",
			prompt:   "/autospec.specify \"Feature with 'quotes'\"",
			want:     "claude -p '/autospec.specify \"Feature with '\\''quotes'\\''\"'",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			executor := &ClaudeExecutor{
				CustomClaudeCmd: tc.template,
			}

			result := executor.expandTemplate(tc.prompt)
			assert.Equal(t, tc.want, result)
		})
	}
}

// TestValidateTemplate tests custom command template validation
func TestValidateTemplate(t *testing.T) {
	tests := map[string]struct {
		template string
		wantErr  bool
	}{
		"empty template": {
			template: "",
			wantErr:  false,
		},
		"valid template": {
			template: "claude -p {{PROMPT}}",
			wantErr:  false,
		},
		"valid with env var": {
			template: "ANTHROPIC_API_KEY=\"\" claude -p {{PROMPT}}",
			wantErr:  false,
		},
		"valid with pipe": {
			template: "claude -p {{PROMPT}} | claude-clean",
			wantErr:  false,
		},
		"missing placeholder": {
			template: "claude -p \"test\"",
			wantErr:  true,
		},
		"wrong placeholder format": {
			template: "claude -p {PROMPT}",
			wantErr:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := ValidateTemplate(tc.template)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "{{PROMPT}}")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestClaudeExecutor_StreamCommand tests command execution with output streaming
func TestClaudeExecutor_StreamCommand(t *testing.T) {
	tests := map[string]struct {
		executor *ClaudeExecutor
		prompt   string
		wantErr  bool
	}{
		"simple mode (no custom command)": {
			executor: &ClaudeExecutor{
				ClaudeCmd:  "echo",
				ClaudeArgs: []string{},
				UseAPIKey:  false,
			},
			prompt:  "test prompt",
			wantErr: false,
		},
		"with custom command using echo": {
			executor: &ClaudeExecutor{
				CustomClaudeCmd: "echo {{PROMPT}}",
				UseAPIKey:       false,
			},
			prompt:  "test prompt",
			wantErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer

			err := tc.executor.StreamCommand(tc.prompt, &stdout, &stderr)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify output was captured
				assert.NotEmpty(t, stdout.String()+stderr.String())
			}
		})
	}
}

// TestParseCustomCommand tests custom command parsing
func TestParseCustomCommand(t *testing.T) {
	executor := &ClaudeExecutor{}

	tests := map[string]struct {
		cmdStr string
	}{
		"simple command": {
			cmdStr: "claude -p /autospec.specify",
		},
		"with env var": {
			cmdStr: "ANTHROPIC_API_KEY=\"\" claude -p /autospec.plan",
		},
		"with pipe": {
			cmdStr: "claude -p /autospec.tasks | grep 'Task'",
		},
		"complex pipeline": {
			cmdStr: "ANTHROPIC_API_KEY=\"\" claude -p \"/autospec.specify\" --verbose | tee log.txt | grep 'Success'",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmd := executor.parseCustomCommand(tc.cmdStr)

			// Verify command is set up to run via shell
			assert.NotNil(t, cmd)
			assert.Contains(t, cmd.Path, "sh")
		})
	}
}

// TestClaudeExecutor_EnvironmentSetup tests environment variable handling
func TestClaudeExecutor_EnvironmentSetup(t *testing.T) {
	tests := map[string]struct {
		useAPIKey bool
		wantEmpty bool // Whether ANTHROPIC_API_KEY should be empty
	}{
		"use API key": {
			useAPIKey: true,
			wantEmpty: false,
		},
		"don't use API key": {
			useAPIKey: false,
			wantEmpty: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			executor := &ClaudeExecutor{
				ClaudeCmd:  "echo",
				ClaudeArgs: []string{},
				UseAPIKey:  tc.useAPIKey,
			}

			var stdout, stderr bytes.Buffer
			err := executor.StreamCommand("test", &stdout, &stderr)
			require.NoError(t, err)

			// Note: We can't directly test environment variable setting
			// without modifying the executor to expose the command
			// This test verifies the executor can be constructed with UseAPIKey
			assert.Equal(t, tc.useAPIKey, executor.UseAPIKey)
		})
	}
}

// TestClaudeExecutor_FallbackMode tests fallback to simple mode
func TestClaudeExecutor_FallbackMode(t *testing.T) {
	executor := &ClaudeExecutor{
		ClaudeCmd:       "echo",
		ClaudeArgs:      []string{"arg1", "arg2"},
		UseAPIKey:       false,
		CustomClaudeCmd: "", // Empty means use simple mode
	}

	var stdout, stderr bytes.Buffer
	err := executor.StreamCommand("test prompt", &stdout, &stderr)

	assert.NoError(t, err)
	output := stdout.String()

	// In simple mode with echo, should see all args
	assert.Contains(t, output, "arg1")
	assert.Contains(t, output, "arg2")
	assert.Contains(t, output, "test prompt")
}

// TestExecuteSpecKitCommand tests SpecKit command execution
func TestExecuteSpecKitCommand(t *testing.T) {
	executor := &ClaudeExecutor{
		ClaudeCmd:  "echo",
		ClaudeArgs: []string{},
		UseAPIKey:  false,
	}

	// Mock execution by using echo
	// In real usage, this would call claude with the SpecKit command
	var stdout, stderr bytes.Buffer
	err := executor.StreamCommand("/autospec.specify \"test\"", &stdout, &stderr)

	assert.NoError(t, err)
	assert.Contains(t, stdout.String(), "/autospec.specify")
}

// TestCustomCommandWithPipeOperator tests pipe operator handling
func TestCustomCommandWithPipeOperator(t *testing.T) {
	// This test verifies that pipe operators are handled correctly
	executor := &ClaudeExecutor{
		CustomClaudeCmd: "echo {{PROMPT}} | grep 'test'",
		UseAPIKey:       false,
	}

	expanded := executor.expandTemplate("this is a test")
	assert.Equal(t, "echo 'this is a test' | grep 'test'", expanded)

	// Verify the command would be executed via shell
	cmd := executor.parseCustomCommand(expanded)
	assert.NotNil(t, cmd)
}

// TestCustomCommandWithEnvVarPrefix tests environment variable prefix handling
func TestCustomCommandWithEnvVarPrefix(t *testing.T) {
	executor := &ClaudeExecutor{
		CustomClaudeCmd: "ANTHROPIC_API_KEY=\"\" claude -p {{PROMPT}}",
		UseAPIKey:       false,
	}

	expanded := executor.expandTemplate("/autospec.plan")
	assert.Equal(t, "ANTHROPIC_API_KEY=\"\" claude -p '/autospec.plan'", expanded)

	// Verify env var prefix is preserved
	assert.Contains(t, expanded, "ANTHROPIC_API_KEY=\"\"")
}

// BenchmarkExpandTemplate benchmarks template expansion performance
func BenchmarkExpandTemplate(b *testing.B) {
	executor := &ClaudeExecutor{
		CustomClaudeCmd: "ANTHROPIC_API_KEY=\"\" claude -p {{PROMPT}} --verbose | tee output.log",
	}

	prompt := "/autospec.specify \"Add user authentication with OAuth2\""

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = executor.expandTemplate(prompt)
	}
}

// TestRegressionMultilinePromptWithQuotes is a regression test for the issue where
// multiline prompts with quotes would break shell parsing in custom commands.
// This test ensures that the exact user scenario that failed is now working.
func TestRegressionMultilinePromptWithQuotes(t *testing.T) {
	// This is the exact scenario that failed:
	// - Custom command with pipe: "ANTHROPIC_API_KEY=\"\" claude -p {{PROMPT}} | claude-clean"
	// - Multiline feature description with quotes
	executor := &ClaudeExecutor{
		CustomClaudeCmd: "ANTHROPIC_API_KEY=\"\" claude -p --dangerously-skip-permissions --verbose --output-format stream-json {{PROMPT}} | claude-clean",
		UseAPIKey:       false,
	}

	featureDescription := `Implement timeout functionality for Claude CLI command execution
  Use 'timeout' config setting to abort long-running commands
  Add context with deadline to command execution
  Update documentation when implemented`

	command := "/autospec.specify \"" + featureDescription + "\""

	// Expand the template
	expanded := executor.expandTemplate(command)

	// Verify the prompt is properly quoted
	assert.NotContains(t, expanded, "{{PROMPT}}")

	// Verify the command structure is intact
	assert.Contains(t, expanded, "ANTHROPIC_API_KEY=\"\"")
	assert.Contains(t, expanded, "claude -p")
	assert.Contains(t, expanded, "| claude-clean")

	// Verify single quotes are escaped properly
	// The word 'timeout' should be escaped as 'timeout' -> '\''timeout'\''
	assert.Contains(t, expanded, "'\\''timeout'\\''")

	// Verify the command can be parsed by shell
	cmd := executor.parseCustomCommand(expanded)
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Path, "sh")
	assert.Equal(t, []string{"-c", expanded}, cmd.Args[1:])
}

// TestCommandPromptFormats tests various SpecKit command formats with prompts
func TestCommandPromptFormats(t *testing.T) {
	tests := map[string]struct {
		command      string
		template     string
		wantContains []string
	}{
		"specify with simple prompt": {
			command:      "/autospec.specify \"Add user authentication\"",
			template:     "claude -p {{PROMPT}}",
			wantContains: []string{"'/autospec.specify \"Add user authentication\"'"},
		},
		"specify with complex multiline prompt": {
			command: `/autospec.specify "Feature with
  multiple lines
  and 'quotes' and $vars"`,
			template: "ANTHROPIC_API_KEY=\"\" claude -p {{PROMPT}} | claude-clean",
			wantContains: []string{
				"ANTHROPIC_API_KEY=\"\"",
				"claude -p",
				"| claude-clean",
				"'\\''quotes'\\''", // quotes should be escaped
			},
		},
		"plan with optional prompt": {
			command:      "/autospec.plan \"Focus on security best practices\"",
			template:     "claude -p {{PROMPT}}",
			wantContains: []string{"'/autospec.plan \"Focus on security best practices\"'"},
		},
		"tasks with optional prompt": {
			command:      "/autospec.tasks \"Break into small incremental steps\"",
			template:     "claude -p {{PROMPT}}",
			wantContains: []string{"'/autospec.tasks \"Break into small incremental steps\"'"},
		},
		"implement with resume flag": {
			command:      "/autospec.implement --resume",
			template:     "claude -p {{PROMPT}}",
			wantContains: []string{"'/autospec.implement --resume'"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			executor := &ClaudeExecutor{
				CustomClaudeCmd: tc.template,
			}

			expanded := executor.expandTemplate(tc.command)

			// Verify all expected strings are present
			for _, want := range tc.wantContains {
				assert.Contains(t, expanded, want,
					"expected expanded command to contain %q, got: %s", want, expanded)
			}

			// Verify placeholder was replaced
			assert.NotContains(t, expanded, "{{PROMPT}}")
		})
	}
}

// BenchmarkValidateTemplate benchmarks template validation performance
func BenchmarkValidateTemplate(b *testing.B) {
	template := "ANTHROPIC_API_KEY=\"\" claude -p {{PROMPT}} --verbose | claude-clean"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateTemplate(template)
	}
}

// TestTemplateEdgeCases tests edge cases in template handling
func TestTemplateEdgeCases(t *testing.T) {
	tests := map[string]struct {
		template string
		prompt   string
		wantErr  bool
	}{
		"multiple placeholders (should replace all)": {
			template: "echo {{PROMPT}} && echo {{PROMPT}}",
			prompt:   "test",
			wantErr:  false,
		},
		"placeholder in quotes": {
			template: "claude -p \"{{PROMPT}}\"",
			prompt:   "/autospec.specify",
			wantErr:  false,
		},
		"prompt with special chars": {
			template: "claude -p {{PROMPT}}",
			prompt:   "/autospec.specify \"Feature with 'quotes' and $vars\"",
			wantErr:  false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			executor := &ClaudeExecutor{
				CustomClaudeCmd: tc.template,
			}

			expanded := executor.expandTemplate(tc.prompt)

			// Verify placeholder was replaced
			assert.NotContains(t, expanded, "{{PROMPT}}")
			// The prompt should be shell-quoted, so we check for the original prompt content
			// but it will be wrapped in quotes
			if tc.prompt == "test" {
				assert.Contains(t, expanded, "'test'")
			} else {
				// For prompts with quotes/special chars, just verify they're in the output somehow
				assert.NotEqual(t, tc.template, expanded)
			}
		})
	}
}

// Timeout Enforcement Tests

func TestExecute_NoTimeout_Success(t *testing.T) {
	executor := &ClaudeExecutor{
		ClaudeCmd:  "echo",
		ClaudeArgs: []string{},
		Timeout:    0, // No timeout
	}

	err := executor.Execute("test")
	assert.NoError(t, err)
}

func TestExecute_WithTimeout_CompletesBeforeTimeout(t *testing.T) {
	executor := &ClaudeExecutor{
		ClaudeCmd:  "echo",
		ClaudeArgs: []string{},
		Timeout:    60, // 60 seconds - plenty of time for echo
	}

	err := executor.Execute("test")
	assert.NoError(t, err, "Command should complete before timeout")
}

func TestExecute_WithTimeout_ExceedsTimeout(t *testing.T) {
	executor := &ClaudeExecutor{
		ClaudeCmd:  "sleep",
		ClaudeArgs: []string{},
		Timeout:    1, // 1 second timeout
	}

	// Sleep for 10 seconds (will be killed after 1 second)
	err := executor.Execute("10")
	require.Error(t, err)

	// Verify it's a TimeoutError
	var timeoutErr *TimeoutError
	assert.True(t, errors.As(err, &timeoutErr), "Error should be TimeoutError")

	if timeoutErr != nil {
		assert.Contains(t, timeoutErr.Error(), "timed out")
		assert.Contains(t, timeoutErr.Error(), "1s")
	}
}

func TestExecute_TimeoutError_IncludesMetadata(t *testing.T) {
	executor := &ClaudeExecutor{
		ClaudeCmd:  "sleep",
		ClaudeArgs: []string{},
		Timeout:    1,
	}

	err := executor.Execute("5")
	require.Error(t, err)

	var timeoutErr *TimeoutError
	require.True(t, errors.As(err, &timeoutErr), "Error should be TimeoutError")

	// Verify metadata
	assert.Equal(t, 1*time.Second, timeoutErr.Timeout)
	assert.Contains(t, timeoutErr.Command, "sleep")
	assert.Equal(t, context.DeadlineExceeded, timeoutErr.Err)
}

func TestStreamCommand_WithTimeout_Success(t *testing.T) {
	executor := &ClaudeExecutor{
		ClaudeCmd:  "echo",
		ClaudeArgs: []string{},
		Timeout:    5, // 5 seconds
	}

	var stdout, stderr bytes.Buffer
	err := executor.StreamCommand("test", &stdout, &stderr)

	assert.NoError(t, err)
	assert.Contains(t, stdout.String(), "test")
}

func TestStreamCommand_WithTimeout_ExceedsTimeout(t *testing.T) {
	executor := &ClaudeExecutor{
		ClaudeCmd:  "sleep",
		ClaudeArgs: []string{},
		Timeout:    1, // 1 second
	}

	var stdout, stderr bytes.Buffer
	err := executor.StreamCommand("10", &stdout, &stderr)

	require.Error(t, err)

	var timeoutErr *TimeoutError
	assert.True(t, errors.As(err, &timeoutErr), "Error should be TimeoutError")
}

func TestExecute_TimeoutPropagation(t *testing.T) {
	tests := []struct {
		name        string
		timeout     int
		sleepTime   string
		wantError   bool
		wantTimeout bool
	}{
		{
			name:        "no timeout, long command",
			timeout:     0,
			sleepTime:   "0.1",
			wantError:   false,
			wantTimeout: false,
		},
		{
			name:        "timeout set, command completes",
			timeout:     5,
			sleepTime:   "0.1",
			wantError:   false,
			wantTimeout: false,
		},
		{
			name:        "timeout exceeded",
			timeout:     1,
			sleepTime:   "5",
			wantError:   true,
			wantTimeout: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			executor := &ClaudeExecutor{
				ClaudeCmd:  "sleep",
				ClaudeArgs: []string{},
				Timeout:    tt.timeout,
			}

			err := executor.Execute(tt.sleepTime)

			if tt.wantError {
				assert.Error(t, err)

				if tt.wantTimeout {
					var timeoutErr *TimeoutError
					assert.True(t, errors.As(err, &timeoutErr), "Should be TimeoutError")
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecute_CustomCommand_WithTimeout(t *testing.T) {
	executor := &ClaudeExecutor{
		CustomClaudeCmd: "sleep {{PROMPT}}",
		Timeout:         1, // 1 second
	}

	err := executor.Execute("5") // Sleep 5 seconds
	require.Error(t, err)

	var timeoutErr *TimeoutError
	assert.True(t, errors.As(err, &timeoutErr), "Error should be TimeoutError")
}

func TestFormatCommand_Simple(t *testing.T) {
	executor := &ClaudeExecutor{
		ClaudeCmd: "claude",
	}

	result := executor.formatCommand("/autospec.plan")
	assert.Equal(t, "claude /autospec.plan", result)
}

func TestFormatCommand_CustomCommand(t *testing.T) {
	executor := &ClaudeExecutor{
		CustomClaudeCmd: "echo {{PROMPT}}",
	}

	result := executor.formatCommand("test")
	assert.Contains(t, result, "echo")
	assert.Contains(t, result, "'test'")
}
