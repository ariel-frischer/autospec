// Package workflow tests PhaseExecutor functionality.
// Related: internal/workflow/phase_executor.go, internal/workflow/interfaces.go
// Tags: workflow, phase-executor, testing, mocks
package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/validation"
)

// TestNewPhaseExecutor tests PhaseExecutor constructor.
func TestNewPhaseExecutor(t *testing.T) {
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
		"creates executor with empty specsDir": {
			executor: &Executor{},
			specsDir: "",
			debug:    false,
			wantNil:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			pe := NewPhaseExecutor(tt.executor, tt.specsDir, tt.debug)

			if (pe == nil) != tt.wantNil {
				t.Errorf("NewPhaseExecutor() returned nil = %v, want nil = %v", pe == nil, tt.wantNil)
			}

			if pe != nil {
				if pe.specsDir != tt.specsDir {
					t.Errorf("specsDir = %q, want %q", pe.specsDir, tt.specsDir)
				}
				if pe.debug != tt.debug {
					t.Errorf("debug = %v, want %v", pe.debug, tt.debug)
				}
			}
		})
	}
}

// TestPhaseExecutorInterface_Compliance verifies PhaseExecutor implements PhaseExecutorInterface.
func TestPhaseExecutorInterface_Compliance(t *testing.T) {
	t.Parallel()

	// This test ensures compile-time interface compliance
	var _ PhaseExecutorInterface = (*PhaseExecutor)(nil)

	// Create an instance and verify it can be assigned to the interface
	pe := NewPhaseExecutor(&Executor{}, "specs/", false)
	var iface PhaseExecutorInterface = pe

	if iface == nil {
		t.Error("PhaseExecutor should implement PhaseExecutorInterface")
	}
}

// TestPhaseExecutor_DebugLog tests debug logging behavior.
func TestPhaseExecutor_DebugLog(t *testing.T) {
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

			pe := NewPhaseExecutor(&Executor{}, "specs/", tt.debug)

			// Call debugLog - it should not panic
			pe.debugLog("test message: %s", "value")
		})
	}
}

// TestPhaseExecutor_BuildPhaseCommand tests phase command construction.
func TestPhaseExecutor_BuildPhaseCommand(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		phaseNumber     int
		contextFilePath string
		prompt          string
		want            string
	}{
		"without prompt": {
			phaseNumber:     1,
			contextFilePath: ".autospec/context/phase-1.yaml",
			prompt:          "",
			want:            "/autospec.implement --phase 1 --context-file .autospec/context/phase-1.yaml",
		},
		"with prompt": {
			phaseNumber:     2,
			contextFilePath: ".autospec/context/phase-2.yaml",
			prompt:          "custom prompt",
			want:            `/autospec.implement --phase 2 --context-file .autospec/context/phase-2.yaml "custom prompt"`,
		},
		"phase 3 with different context path": {
			phaseNumber:     3,
			contextFilePath: "/tmp/context.yaml",
			prompt:          "",
			want:            "/autospec.implement --phase 3 --context-file /tmp/context.yaml",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			pe := NewPhaseExecutor(&Executor{}, "specs/", false)
			result := pe.buildPhaseCommand(tt.phaseNumber, tt.contextFilePath, tt.prompt)

			if result != tt.want {
				t.Errorf("buildPhaseCommand(%d, %q, %q) = %q, want %q",
					tt.phaseNumber, tt.contextFilePath, tt.prompt, result, tt.want)
			}
		})
	}
}

// TestPhaseExecutor_GetTaskIDsForPhase tests task ID extraction for a phase.
func TestPhaseExecutor_GetTaskIDsForPhase(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupTasks  func(t *testing.T, dir string)
		phaseNumber int
		wantIDs     []string
	}{
		"returns empty for nonexistent tasks file": {
			setupTasks: func(t *testing.T, dir string) {
				// Don't create tasks.yaml
			},
			phaseNumber: 1,
			wantIDs:     []string{},
		},
		"returns empty for invalid tasks file": {
			setupTasks: func(t *testing.T, dir string) {
				tasksPath := filepath.Join(dir, "tasks.yaml")
				if err := os.WriteFile(tasksPath, []byte("invalid: yaml: content"), 0644); err != nil {
					t.Fatalf("failed to write tasks.yaml: %v", err)
				}
			},
			phaseNumber: 1,
			wantIDs:     []string{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create temp directory
			tempDir := t.TempDir()
			specDir := filepath.Join(tempDir, "specs", "001-test")
			if err := os.MkdirAll(specDir, 0755); err != nil {
				t.Fatalf("failed to create spec dir: %v", err)
			}

			// Setup tasks
			tt.setupTasks(t, specDir)

			pe := NewPhaseExecutor(&Executor{}, filepath.Join(tempDir, "specs"), false)
			tasksPath := filepath.Join(specDir, "tasks.yaml")
			result := pe.getTaskIDsForPhase(tasksPath, tt.phaseNumber)

			if len(result) != len(tt.wantIDs) {
				t.Errorf("getTaskIDsForPhase() returned %d IDs, want %d", len(result), len(tt.wantIDs))
			}
		})
	}
}

// TestPhaseExecutor_GetUpdatedPhaseInfo tests phase info retrieval.
func TestPhaseExecutor_GetUpdatedPhaseInfo(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupTasks  func(t *testing.T, dir string)
		phaseNumber int
		wantNil     bool
	}{
		"returns nil for nonexistent tasks file": {
			setupTasks: func(t *testing.T, dir string) {
				// Don't create tasks.yaml
			},
			phaseNumber: 1,
			wantNil:     true,
		},
		"returns nil for phase not found": {
			setupTasks: func(t *testing.T, dir string) {
				tasksContent := `_meta:
  artifact_type: tasks
phases:
  - number: 1
    title: "Phase 1"
    tasks: []
`
				tasksPath := filepath.Join(dir, "tasks.yaml")
				if err := os.WriteFile(tasksPath, []byte(tasksContent), 0644); err != nil {
					t.Fatalf("failed to write tasks.yaml: %v", err)
				}
			},
			phaseNumber: 99, // Non-existent phase
			wantNil:     true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create temp directory
			tempDir := t.TempDir()
			specDir := filepath.Join(tempDir, "specs", "001-test")
			if err := os.MkdirAll(specDir, 0755); err != nil {
				t.Fatalf("failed to create spec dir: %v", err)
			}

			// Setup tasks
			tt.setupTasks(t, specDir)

			pe := NewPhaseExecutor(&Executor{}, filepath.Join(tempDir, "specs"), false)
			tasksPath := filepath.Join(specDir, "tasks.yaml")
			result := pe.getUpdatedPhaseInfo(tasksPath, tt.phaseNumber)

			if (result == nil) != tt.wantNil {
				t.Errorf("getUpdatedPhaseInfo() returned nil = %v, want nil = %v", result == nil, tt.wantNil)
			}
		})
	}
}

// TestPhaseExecutor_AllTasksCompletedOrBlocked tests the completion check logic.
func TestPhaseExecutor_AllTasksCompletedOrBlocked(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		tasks       []validation.TaskItem
		phaseNumber int
		want        bool
	}{
		"all completed returns true": {
			tasks: []validation.TaskItem{
				{ID: "T001", Status: "Completed"},
				{ID: "T002", Status: "completed"},
				{ID: "T003", Status: "Done"},
			},
			phaseNumber: 1,
			want:        true,
		},
		"mixed completed and blocked returns true": {
			tasks: []validation.TaskItem{
				{ID: "T001", Status: "Completed"},
				{ID: "T002", Status: "Blocked"},
			},
			phaseNumber: 1,
			want:        true,
		},
		"has pending task returns false": {
			tasks: []validation.TaskItem{
				{ID: "T001", Status: "Completed"},
				{ID: "T002", Status: "Pending"},
			},
			phaseNumber: 1,
			want:        false,
		},
		"has in progress task returns false": {
			tasks: []validation.TaskItem{
				{ID: "T001", Status: "Completed"},
				{ID: "T002", Status: "InProgress"},
			},
			phaseNumber: 1,
			want:        false,
		},
		"empty tasks returns true (nothing to execute)": {
			tasks:       []validation.TaskItem{},
			phaseNumber: 1,
			want:        true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			pe := NewPhaseExecutor(&Executor{}, "specs/", false)
			result := pe.allTasksCompletedOrBlocked(tt.tasks, tt.phaseNumber)

			if result != tt.want {
				t.Errorf("allTasksCompletedOrBlocked() = %v, want %v", result, tt.want)
			}
		})
	}
}

// TestPhaseExecutor_PrintPhaseCompletion tests phase completion output.
func TestPhaseExecutor_PrintPhaseCompletion(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		phaseNumber  int
		updatedPhase *validation.PhaseInfo
	}{
		"with phase info": {
			phaseNumber: 1,
			updatedPhase: &validation.PhaseInfo{
				Number:         1,
				Title:          "Setup",
				TotalTasks:     5,
				CompletedTasks: 5,
				BlockedTasks:   0,
			},
		},
		"with nil phase info": {
			phaseNumber:  2,
			updatedPhase: nil,
		},
		"with blocked tasks": {
			phaseNumber: 3,
			updatedPhase: &validation.PhaseInfo{
				Number:         3,
				Title:          "Implementation",
				TotalTasks:     10,
				CompletedTasks: 8,
				BlockedTasks:   2,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			pe := NewPhaseExecutor(&Executor{}, "specs/", false)

			// Call printPhaseCompletion - it should not panic
			pe.printPhaseCompletion(tt.phaseNumber, tt.updatedPhase)
		})
	}
}

// TestPhaseExecutor_MethodSignatures verifies method signatures match interface.
func TestPhaseExecutor_MethodSignatures(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		name string
	}{
		"verifies ExecutePhaseLoop signature": {
			name: "ExecutePhaseLoop",
		},
		"verifies ExecuteSinglePhase signature": {
			name: "ExecuteSinglePhase",
		},
	}

	for name := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			pe := NewPhaseExecutor(&Executor{}, "specs/", false)

			// Verify method signatures match interface
			var _ func(string, string, []validation.PhaseInfo, int, int, string) error = pe.ExecutePhaseLoop
			var _ func(string, int, string) error = pe.ExecuteSinglePhase
		})
	}
}
