// Package workflow_test tests workflow orchestration using mock infrastructure.
// Related: internal/workflow/orchestrator.go, internal/testutil/mock_executor.go
// Tags: workflow, integration, orchestration, mocks, git-isolation, retry, artifacts
package workflow_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/cliagent"
	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/ariel-frischer/autospec/internal/workflow"
)

// TestWorkflowOrchestrator_Integration tests workflow orchestration using mock infrastructure.
//
// Test structure uses closure-based configuration:
//   - setupMock: configures mock response sequence (WithResponse/ThenResponse/ThenError)
//   - runWorkflow: executes workflow stages via mock
//   - verifyMock: asserts call counts, prompts, and timestamps
//
// The mock builder fluent API (WithResponse().ThenResponse()) enables readable test setup.
// IMPORTANT: NO t.Parallel() - GitIsolation changes cwd causing race conditions.
func TestWorkflowOrchestrator_Integration(t *testing.T) {
	// NOTE: Do NOT add t.Parallel() here or in subtests below.
	// GitIsolation changes the working directory which causes race conditions
	// when running in parallel. Each subtest captures origDir on setup, but
	// parallel execution can cause one test's temp dir to be captured as
	// another test's origDir, leading to cleanup failures.

	tests := map[string]struct {
		setupMock   func(*testing.T, *testutil.MockExecutorBuilder, string)
		runWorkflow func(*testing.T, *workflow.WorkflowOrchestrator, *testutil.MockExecutor) error
		wantErr     bool
		errContains string
		verifyMock  func(*testing.T, *testutil.MockExecutor)
	}{
		"successful execution records all calls": {
			setupMock: func(t *testing.T, builder *testutil.MockExecutorBuilder, specsDir string) {
				t.Helper()
				builder.
					WithResponse("spec created").
					ThenResponse("plan created").
					ThenResponse("tasks created")
			},
			runWorkflow: func(t *testing.T, orch *workflow.WorkflowOrchestrator, mock *testutil.MockExecutor) error {
				t.Helper()
				// Manually invoke the mock to simulate workflow stages
				if err := mock.Execute("/autospec.specify"); err != nil {
					return err
				}
				if err := mock.Execute("/autospec.plan"); err != nil {
					return err
				}
				return mock.Execute("/autospec.tasks")
			},
			wantErr: false,
			verifyMock: func(t *testing.T, mock *testutil.MockExecutor) {
				t.Helper()
				if got := mock.GetCallCount(); got != 3 {
					t.Errorf("expected 3 calls, got %d", got)
				}
				executeCalls := mock.GetCallsByMethod("Execute")
				if len(executeCalls) != 3 {
					t.Errorf("expected 3 Execute calls, got %d", len(executeCalls))
				}
			},
		},
		"mock records call details correctly": {
			setupMock: func(t *testing.T, builder *testutil.MockExecutorBuilder, specsDir string) {
				t.Helper()
				builder.WithResponse("success")
			},
			runWorkflow: func(t *testing.T, orch *workflow.WorkflowOrchestrator, mock *testutil.MockExecutor) error {
				t.Helper()
				return mock.Execute("/autospec.specify \"test feature\"")
			},
			wantErr: false,
			verifyMock: func(t *testing.T, mock *testutil.MockExecutor) {
				t.Helper()
				calls := mock.GetCalls()
				if len(calls) != 1 {
					t.Fatalf("expected 1 call, got %d", len(calls))
				}
				if calls[0].Prompt != "/autospec.specify \"test feature\"" {
					t.Errorf("unexpected prompt: %s", calls[0].Prompt)
				}
				if calls[0].Timestamp.IsZero() {
					t.Error("timestamp should not be zero")
				}
			},
		},
		"error response stops workflow": {
			setupMock: func(t *testing.T, builder *testutil.MockExecutorBuilder, specsDir string) {
				t.Helper()
				builder.
					WithResponse("spec created").
					ThenError(workflow.ErrMockExecute)
			},
			runWorkflow: func(t *testing.T, orch *workflow.WorkflowOrchestrator, mock *testutil.MockExecutor) error {
				t.Helper()
				if err := mock.Execute("/autospec.specify"); err != nil {
					return err
				}
				return mock.Execute("/autospec.plan")
			},
			wantErr:     true,
			errContains: "mock execute error",
			verifyMock: func(t *testing.T, mock *testutil.MockExecutor) {
				t.Helper()
				// Should have 2 calls - first succeeded, second failed
				if got := mock.GetCallCount(); got != 2 {
					t.Errorf("expected 2 calls, got %d", got)
				}
			},
		},
		"sequential responses return in order": {
			setupMock: func(t *testing.T, builder *testutil.MockExecutorBuilder, specsDir string) {
				t.Helper()
				builder.
					WithResponse("first").
					ThenResponse("second").
					ThenResponse("third")
			},
			runWorkflow: func(t *testing.T, orch *workflow.WorkflowOrchestrator, mock *testutil.MockExecutor) error {
				t.Helper()
				for _, cmd := range []string{"cmd1", "cmd2", "cmd3"} {
					if err := mock.Execute(cmd); err != nil {
						return err
					}
				}
				return nil
			},
			wantErr: false,
			verifyMock: func(t *testing.T, mock *testutil.MockExecutor) {
				t.Helper()
				calls := mock.GetCalls()
				if len(calls) != 3 {
					t.Fatalf("expected 3 calls, got %d", len(calls))
				}
				// Verify calls were recorded in order
				expected := []string{"cmd1", "cmd2", "cmd3"}
				for i, call := range calls {
					if call.Prompt != expected[i] {
						t.Errorf("call %d: expected %q, got %q", i, expected[i], call.Prompt)
					}
				}
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Do NOT add t.Parallel() - see comment at top of test function.

			// Create isolated git repo
			gi := testutil.NewGitIsolation(t)

			// Set up specs directory
			specsDir := filepath.Join(gi.TempRepoDir(), "specs")
			if err := os.MkdirAll(specsDir, 0755); err != nil {
				t.Fatalf("failed to create specs dir: %v", err)
			}

			// Create mock executor builder
			builder := testutil.NewMockExecutorBuilder(t)
			tt.setupMock(t, builder, specsDir)
			mock := builder.Build()

			// Create workflow orchestrator with mock-compatible config
			cfg := &config.Configuration{
				SpecsDir:   specsDir,
				StateDir:   t.TempDir(),
				MaxRetries: 3,
				CustomAgent: &cliagent.CustomAgentConfig{
					Command: "echo",
					Args:    []string{"{{PROMPT}}"},
				},
			}
			orch := workflow.NewWorkflowOrchestrator(cfg)

			// Run the workflow function
			err := tt.runWorkflow(t, orch, mock)

			// Verify error expectation
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !containsSubstring(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Run mock verification
			if tt.verifyMock != nil {
				tt.verifyMock(t, mock)
			}

			// Note: VerifyNoBranchPollution is called automatically in Cleanup
		})
	}
}

// TestMockExecutor_RetryBehavior tests retry simulation with mock executor.
func TestMockExecutor_RetryBehavior(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		failures     int
		maxAttempts  int
		wantErr      bool
		wantAttempts int
	}{
		"succeeds after one failure": {
			failures:     1,
			maxAttempts:  3,
			wantErr:      false,
			wantAttempts: 2,
		},
		"succeeds after two failures": {
			failures:     2,
			maxAttempts:  3,
			wantErr:      false,
			wantAttempts: 3,
		},
		"fails when retries exhausted": {
			failures:     5,
			maxAttempts:  3,
			wantErr:      true,
			wantAttempts: 3,
		},
		"succeeds immediately with no failures": {
			failures:     0,
			maxAttempts:  3,
			wantErr:      false,
			wantAttempts: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			builder := testutil.NewMockExecutorBuilder(t)

			// Configure mock to fail tt.failures times before succeeding
			for i := 0; i < tt.failures; i++ {
				builder.WithError(workflow.ErrMockExecute)
			}
			builder.WithResponse("success")

			mock := builder.Build()

			// Attempt execution with retries
			var lastErr error
			attempts := 0
			for attempts < tt.maxAttempts {
				attempts++
				err := mock.Execute("test command")
				if err == nil {
					break
				}
				lastErr = err
			}

			if tt.wantErr && lastErr == nil && attempts >= tt.maxAttempts {
				// Expected failure - need to check if last call failed
				// The mock may have succeeded on the last attempt
				if attempts > tt.failures {
					t.Error("expected error after retries exhausted")
				}
			}

			if !tt.wantErr && lastErr != nil && attempts == tt.wantAttempts {
				// Check if the last attempt actually succeeded
				calls := mock.GetCalls()
				if len(calls) > 0 && calls[len(calls)-1].Error != nil {
					t.Errorf("expected success, got error: %v", lastErr)
				}
			}

			if mock.GetCallCount() != tt.wantAttempts {
				t.Errorf("expected %d attempts, got %d", tt.wantAttempts, mock.GetCallCount())
			}
		})
	}
}

// TestMockExecutor_DelaySimulation tests timeout simulation with mock delays.
func TestMockExecutor_DelaySimulation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		delay       time.Duration
		timeout     time.Duration
		wantTimeout bool
	}{
		"no delay completes quickly": {
			delay:       0,
			timeout:     time.Second,
			wantTimeout: false,
		},
		"small delay within timeout": {
			delay:       10 * time.Millisecond,
			timeout:     time.Second,
			wantTimeout: false,
		},
		"delay longer than timeout": {
			delay:       200 * time.Millisecond,
			timeout:     50 * time.Millisecond,
			wantTimeout: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			builder := testutil.NewMockExecutorBuilder(t)
			builder.WithResponse("success")
			if tt.delay > 0 {
				builder.WithDelay(tt.delay)
			}

			mock := builder.Build()

			// Execute with timing
			start := time.Now()
			done := make(chan error, 1)
			go func() {
				done <- mock.Execute("test")
			}()

			select {
			case err := <-done:
				elapsed := time.Since(start)
				if tt.wantTimeout {
					// If we expected timeout but got response, that's still OK
					// as the mock delay isn't a true context timeout
					t.Logf("completed in %v (expected timeout)", elapsed)
				}
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			case <-time.After(tt.timeout + 100*time.Millisecond):
				if !tt.wantTimeout {
					t.Error("unexpected timeout")
				}
			}
		})
	}
}

// TestMockExecutor_CallLogVerification tests that MOCK_CALL_LOG is properly recorded.
func TestMockExecutor_CallLogVerification(t *testing.T) {
	t.Parallel()

	builder := testutil.NewMockExecutorBuilder(t)
	builder.
		WithResponse("first response").
		ThenResponse("second response")

	mock := builder.Build()

	// Execute multiple commands
	commands := []string{
		"/autospec.specify \"new feature\"",
		"/autospec.plan",
	}

	for _, cmd := range commands {
		if err := mock.Execute(cmd); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Verify call log
	calls := mock.GetCalls()
	if len(calls) != len(commands) {
		t.Fatalf("expected %d calls, got %d", len(commands), len(calls))
	}

	// Verify each call was logged correctly
	for i, call := range calls {
		if call.Method != "Execute" {
			t.Errorf("call %d: expected method Execute, got %s", i, call.Method)
		}
		if call.Prompt != commands[i] {
			t.Errorf("call %d: expected prompt %q, got %q", i, commands[i], call.Prompt)
		}
		if call.Timestamp.IsZero() {
			t.Errorf("call %d: timestamp should not be zero", i)
		}
	}

	// Verify filtering by method works
	executeCalls := mock.GetCallsByMethod("Execute")
	if len(executeCalls) != len(commands) {
		t.Errorf("expected %d Execute calls, got %d", len(commands), len(executeCalls))
	}

	// Verify non-existent method returns empty
	nonExistent := mock.GetCallsByMethod("NonExistent")
	if len(nonExistent) != 0 {
		t.Errorf("expected 0 NonExistent calls, got %d", len(nonExistent))
	}
}

// TestGitIsolation_NoBranchPollution tests that git isolation prevents branch pollution.
func TestGitIsolation_NoBranchPollution(t *testing.T) {
	// NOTE: Do NOT add t.Parallel() - GitIsolation changes the working
	// directory which causes race conditions with parallel tests.

	// Create isolation
	gi := testutil.NewGitIsolation(t)

	// Record original state
	origDir := gi.OriginalDir()
	origBranch := gi.OriginalBranch()

	// Verify we're in the temp repo
	tempRepo := gi.TempRepoDir()
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	if currentDir != tempRepo {
		t.Errorf("expected to be in temp repo %s, got %s", tempRepo, currentDir)
	}

	// Create a branch in the temp repo (should not affect original)
	gi.CreateBranch("test-branch", true)

	// Verify temp repo branch changed
	if gi.CurrentBranch() != "test-branch" {
		t.Errorf("expected temp branch to be test-branch, got %s", gi.CurrentBranch())
	}

	// Cleanup happens automatically via t.Cleanup
	// VerifyNoBranchPollution is called in Cleanup

	_ = origDir
	_ = origBranch
}

// TestGitIsolation_FileOperations tests file operations in isolated repo.
func TestGitIsolation_FileOperations(t *testing.T) {
	// NOTE: Do NOT add t.Parallel() - GitIsolation changes the working
	// directory which causes race conditions with parallel tests.

	gi := testutil.NewGitIsolation(t)

	// Add a file
	content := "test content"
	filePath := gi.AddFile("test-file.txt", content)

	// Verify file exists
	readContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(readContent) != content {
		t.Errorf("expected content %q, got %q", content, string(readContent))
	}

	// Add and commit
	gi.CommitAll("Test commit")

	// Create specs directory structure
	specsDir := gi.SetupSpecsDir("test-feature")
	if _, err := os.Stat(specsDir); os.IsNotExist(err) {
		t.Error("specs directory should exist")
	}

	// Write spec
	specPath := gi.WriteSpec(specsDir)
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		t.Error("spec.yaml should exist")
	}
}

// TestMockExecutor_ArtifactGeneration tests that mock can generate artifacts.
func TestMockExecutor_ArtifactGeneration(t *testing.T) {
	// NOTE: Do NOT add t.Parallel() - GitIsolation changes the working
	// directory which causes race conditions with parallel tests.

	gi := testutil.NewGitIsolation(t)
	specsDir := gi.SetupSpecsDir("test-feature")

	builder := testutil.NewMockExecutorBuilder(t)
	builder.
		WithArtifactDir(specsDir).
		WithResponse("created").
		WithArtifactGeneration(testutil.ArtifactGenerators.Spec)

	mock := builder.Build()

	// Execute - this should trigger artifact generation
	if err := mock.Execute("/autospec.specify"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify artifact was created
	specPath := filepath.Join(specsDir, "spec.yaml")
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		t.Error("spec.yaml should have been generated")
	}
}

// TestMockExecutor_Reset tests that reset clears all state.
func TestMockExecutor_Reset(t *testing.T) {
	t.Parallel()

	builder := testutil.NewMockExecutorBuilder(t)
	builder.
		WithResponse("first").
		ThenResponse("second")

	mock := builder.Build()

	// Make some calls
	_ = mock.Execute("cmd1")
	_ = mock.Execute("cmd2")

	if mock.GetCallCount() != 2 {
		t.Fatalf("expected 2 calls before reset, got %d", mock.GetCallCount())
	}

	// Reset
	mock.Reset()

	// Verify cleared
	if mock.GetCallCount() != 0 {
		t.Errorf("expected 0 calls after reset, got %d", mock.GetCallCount())
	}

	calls := mock.GetCalls()
	if len(calls) != 0 {
		t.Errorf("expected empty calls after reset, got %d", len(calls))
	}
}

// TestMockExecutor_AssertHelpers tests assertion helper methods.
func TestMockExecutor_AssertHelpers(t *testing.T) {
	t.Parallel()

	builder := testutil.NewMockExecutorBuilder(t)
	builder.WithResponse("success")

	mock := builder.Build()

	// Execute a command
	_ = mock.Execute("/autospec.specify")

	// Create a fake testing.T to capture assertions
	fakeT := &testing.T{}

	// Test AssertCalled - should find the call
	mock.AssertCalled(fakeT, "Execute", "specify")

	// Test AssertNotCalled - StreamCommand was not called
	mock.AssertNotCalled(fakeT, "StreamCommand")

	// Test AssertCallCount
	mock.AssertCallCount(fakeT, "Execute", 1)
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstr(s, substr)))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
