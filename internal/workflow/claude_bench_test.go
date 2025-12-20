// Package workflow tests benchmark performance of Claude command execution.
// Related: internal/workflow/claude.go
// Tags: workflow, claude, benchmark, performance, timeout, execution
package workflow

import (
	"bytes"
	"testing"

	"github.com/ariel-frischer/autospec/internal/cliagent"
)

// BenchmarkExecute_NoTimeout benchmarks command execution without timeout
func BenchmarkExecute_NoTimeout(b *testing.B) {
	customAgent, err := cliagent.NewCustomAgentFromConfig(cliagent.CustomAgentConfig{
		Command: "echo",
		Args:    []string{"{{PROMPT}}"},
	})
	if err != nil {
		b.Fatalf("failed to create agent: %v", err)
	}

	executor := &ClaudeExecutor{
		Agent:   customAgent,
		Timeout: 0, // No timeout
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = executor.Execute("test")
	}
}

// BenchmarkExecute_WithTimeout benchmarks command execution with timeout
func BenchmarkExecute_WithTimeout(b *testing.B) {
	customAgent, err := cliagent.NewCustomAgentFromConfig(cliagent.CustomAgentConfig{
		Command: "echo",
		Args:    []string{"{{PROMPT}}"},
	})
	if err != nil {
		b.Fatalf("failed to create agent: %v", err)
	}

	executor := &ClaudeExecutor{
		Agent:   customAgent,
		Timeout: 300, // 5 minutes timeout (will not be hit)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = executor.Execute("test")
	}
}

// BenchmarkStreamCommand_NoTimeout benchmarks streaming without timeout
func BenchmarkStreamCommand_NoTimeout(b *testing.B) {
	customAgent, err := cliagent.NewCustomAgentFromConfig(cliagent.CustomAgentConfig{
		Command: "echo",
		Args:    []string{"{{PROMPT}}"},
	})
	if err != nil {
		b.Fatalf("failed to create agent: %v", err)
	}

	executor := &ClaudeExecutor{
		Agent:   customAgent,
		Timeout: 0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var stdout, stderr bytes.Buffer
		_ = executor.StreamCommand("test", &stdout, &stderr)
	}
}

// BenchmarkStreamCommand_WithTimeout benchmarks streaming with timeout
func BenchmarkStreamCommand_WithTimeout(b *testing.B) {
	customAgent, err := cliagent.NewCustomAgentFromConfig(cliagent.CustomAgentConfig{
		Command: "echo",
		Args:    []string{"{{PROMPT}}"},
	})
	if err != nil {
		b.Fatalf("failed to create agent: %v", err)
	}

	executor := &ClaudeExecutor{
		Agent:   customAgent,
		Timeout: 300,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var stdout, stderr bytes.Buffer
		_ = executor.StreamCommand("test", &stdout, &stderr)
	}
}

// BenchmarkContextCreation benchmarks context creation overhead
func BenchmarkContextCreation(b *testing.B) {
	b.Run("no timeout", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			executor := &ClaudeExecutor{
				Timeout: 0,
			}
			_ = executor
		}
	})

	b.Run("with timeout", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			executor := &ClaudeExecutor{
				Timeout: 300,
			}
			_ = executor
		}
	})
}
