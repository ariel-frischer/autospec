// Package workflow tests StageExecutor functionality.
// Related: internal/workflow/stage_executor.go, internal/workflow/interfaces.go
// Tags: workflow, stage-executor, testing, mocks
package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

// MockExecutor implements a minimal mock for Executor used by StageExecutor tests.
// It tracks ExecuteStage calls and allows configuring return values.
type MockExecutor struct {
	// Configuration
	ExecuteStageResult *StageResult
	ExecuteStageError  error
	ValidateSpecError  error
	ExecuteStageFunc   func(specName string, stage Stage, command string, validateFunc func(string) error) (*StageResult, error)

	// Call tracking
	ExecuteStageCalls []ExecuteStageCall
	ValidateSpecCalls []string
}

// ExecuteStageCall records a call to ExecuteStage.
type ExecuteStageCall struct {
	SpecName string
	Stage    Stage
	Command  string
}

// NewMockExecutor creates a new mock executor with default success behavior.
func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		ExecuteStageResult: &StageResult{Success: true, RetryCount: 0},
		ExecuteStageCalls:  make([]ExecuteStageCall, 0),
		ValidateSpecCalls:  make([]string, 0),
	}
}

// ExecuteStage records the call and returns configured result/error.
func (m *MockExecutor) ExecuteStage(specName string, stage Stage, command string, validateFunc func(string) error) (*StageResult, error) {
	m.ExecuteStageCalls = append(m.ExecuteStageCalls, ExecuteStageCall{
		SpecName: specName,
		Stage:    stage,
		Command:  command,
	})

	if m.ExecuteStageFunc != nil {
		return m.ExecuteStageFunc(specName, stage, command, validateFunc)
	}

	return m.ExecuteStageResult, m.ExecuteStageError
}

// ValidateSpec records the call and returns configured error.
func (m *MockExecutor) ValidateSpec(specDir string) error {
	m.ValidateSpecCalls = append(m.ValidateSpecCalls, specDir)
	return m.ValidateSpecError
}

// StateDir returns a temp directory for testing.
func (m *MockExecutor) StateDir() string {
	return os.TempDir()
}

// TestNewStageExecutor tests StageExecutor constructor.
func TestNewStageExecutor(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		executor *Executor
		specsDir string
		debug    bool
		wantNil  bool
	}{
		"creates executor with valid params": {
			executor: &Executor{},
			specsDir: "specs/",
			debug:    false,
			wantNil:  false,
		},
		"creates executor with debug enabled": {
			executor: &Executor{},
			specsDir: "specs/",
			debug:    true,
			wantNil:  false,
		},
		"creates executor with nil executor": {
			executor: nil,
			specsDir: "specs/",
			debug:    false,
			wantNil:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			se := NewStageExecutor(tt.executor, tt.specsDir, tt.debug)

			if (se == nil) != tt.wantNil {
				t.Errorf("NewStageExecutor() returned nil = %v, want nil = %v", se == nil, tt.wantNil)
			}

			if se != nil {
				if se.specsDir != tt.specsDir {
					t.Errorf("specsDir = %q, want %q", se.specsDir, tt.specsDir)
				}
				if se.debug != tt.debug {
					t.Errorf("debug = %v, want %v", se.debug, tt.debug)
				}
			}
		})
	}
}

// TestStageExecutor_ResolveSpecName tests the spec name resolution logic.
func TestStageExecutor_ResolveSpecName(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		specNameArg string
		wantErr     bool
	}{
		"returns provided spec name": {
			specNameArg: "001-test-spec",
			wantErr:     false,
		},
		"empty spec name attempts auto-detect": {
			specNameArg: "",
			wantErr:     true, // Auto-detect will fail in test env
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create temp specs directory
			tempDir := t.TempDir()
			specsDir := filepath.Join(tempDir, "specs")
			if err := os.MkdirAll(specsDir, 0755); err != nil {
				t.Fatalf("failed to create specs dir: %v", err)
			}

			se := NewStageExecutor(&Executor{StateDir: tempDir}, specsDir, false)

			result, err := se.resolveSpecName(tt.specNameArg)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveSpecName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.specNameArg {
				t.Errorf("resolveSpecName() = %q, want %q", result, tt.specNameArg)
			}
		})
	}
}

// TestStageExecutor_BuildPlanCommand tests plan command construction.
func TestStageExecutor_BuildPlanCommand(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		prompt string
		want   string
	}{
		"empty prompt": {
			prompt: "",
			want:   "/autospec.plan",
		},
		"with prompt": {
			prompt: "custom prompt",
			want:   `/autospec.plan "custom prompt"`,
		},
		"prompt with quotes": {
			prompt: `test "quoted"`,
			want:   `/autospec.plan "test "quoted""`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			se := NewStageExecutor(&Executor{}, "specs/", false)
			result := se.buildPlanCommand(tt.prompt)

			if result != tt.want {
				t.Errorf("buildPlanCommand(%q) = %q, want %q", tt.prompt, result, tt.want)
			}
		})
	}
}

// TestStageExecutor_BuildTasksCommand tests tasks command construction.
func TestStageExecutor_BuildTasksCommand(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		prompt string
		want   string
	}{
		"empty prompt": {
			prompt: "",
			want:   "/autospec.tasks",
		},
		"with prompt": {
			prompt: "custom prompt",
			want:   `/autospec.tasks "custom prompt"`,
		},
		"prompt with quotes": {
			prompt: `test "quoted"`,
			want:   `/autospec.tasks "test "quoted""`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			se := NewStageExecutor(&Executor{}, "specs/", false)
			result := se.buildTasksCommand(tt.prompt)

			if result != tt.want {
				t.Errorf("buildTasksCommand(%q) = %q, want %q", tt.prompt, result, tt.want)
			}
		})
	}
}

// TestStageExecutorInterface_Compliance verifies StageExecutor implements StageExecutorInterface.
func TestStageExecutorInterface_Compliance(t *testing.T) {
	t.Parallel()

	// This test ensures compile-time interface compliance
	var _ StageExecutorInterface = (*StageExecutor)(nil)

	// Create an instance and verify it can be assigned to the interface
	se := NewStageExecutor(&Executor{}, "specs/", false)
	var iface StageExecutorInterface = se

	if iface == nil {
		t.Error("StageExecutor should implement StageExecutorInterface")
	}
}

// TestStageExecutor_ExecutePlan_ErrorHandling tests ExecutePlan method signature compliance.
// Note: Full error handling integration tests require mocking ExecuteStage on Executor.
func TestStageExecutor_ExecutePlan_ErrorHandling(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		specName string
	}{
		"verifies method signature compliance": {
			specName: "001-test",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create a mock executor
			mockExec := &Executor{
				Claude:     &ClaudeExecutor{},
				StateDir:   t.TempDir(),
				SpecsDir:   t.TempDir(),
				MaxRetries: 3,
			}

			se := &StageExecutor{
				executor: mockExec,
				specsDir: mockExec.SpecsDir,
				debug:    false,
			}

			// Verify spec name is accessible for test setup
			_ = tt.specName

			// Verify the method signature matches interface
			var _ func(string, string) error = se.ExecutePlan
			var _ func(string, string) error = se.ExecuteTasks
			var _ func(string) (string, error) = se.ExecuteSpecify
		})
	}
}

// TestStageExecutor_DebugLog tests debug logging behavior.
func TestStageExecutor_DebugLog(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		debug bool
	}{
		"debug disabled": {
			debug: false,
		},
		"debug enabled": {
			debug: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			se := NewStageExecutor(&Executor{}, "specs/", tt.debug)

			// Call debugLog - it should not panic
			se.debugLog("test message: %s", "value")
		})
	}
}
