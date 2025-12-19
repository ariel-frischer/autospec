// Package workflow tests TaskExecutor functionality.
// Related: internal/workflow/task_executor.go, internal/workflow/interfaces.go
// Tags: workflow, task-executor, testing, mocks
package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/validation"
)

// TestNewTaskExecutor tests TaskExecutor constructor.
func TestNewTaskExecutor(t *testing.T) {
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
			te := NewTaskExecutor(tt.executor, tt.specsDir, tt.debug)

			if (te == nil) != tt.wantNil {
				t.Errorf("NewTaskExecutor() returned nil = %v, want nil = %v", te == nil, tt.wantNil)
			}

			if te != nil {
				if te.specsDir != tt.specsDir {
					t.Errorf("specsDir = %q, want %q", te.specsDir, tt.specsDir)
				}
				if te.debug != tt.debug {
					t.Errorf("debug = %v, want %v", te.debug, tt.debug)
				}
			}
		})
	}
}

// TestTaskExecutorInterface_Compliance verifies TaskExecutor implements TaskExecutorInterface.
func TestTaskExecutorInterface_Compliance(t *testing.T) {
	t.Parallel()

	// This test ensures compile-time interface compliance
	var _ TaskExecutorInterface = (*TaskExecutor)(nil)

	// Create an instance and verify it can be assigned to the interface
	te := NewTaskExecutor(&Executor{}, "specs/", false)
	var iface TaskExecutorInterface = te

	if iface == nil {
		t.Error("TaskExecutor should implement TaskExecutorInterface")
	}
}

// TestTaskExecutor_DebugLog tests debug logging behavior.
func TestTaskExecutor_DebugLog(t *testing.T) {
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

			te := NewTaskExecutor(&Executor{}, "specs/", tt.debug)

			// Call debugLog - it should not panic
			te.debugLog("test message: %s", "value")
		})
	}
}

// TestTaskExecutor_BuildTaskCommand tests task command construction.
func TestTaskExecutor_BuildTaskCommand(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		taskID string
		prompt string
		want   string
	}{
		"without prompt": {
			taskID: "T001",
			prompt: "",
			want:   "/autospec.implement --task T001",
		},
		"with prompt": {
			taskID: "T002",
			prompt: "custom prompt",
			want:   `/autospec.implement --task T002 "custom prompt"`,
		},
		"different task ID": {
			taskID: "T100",
			prompt: "",
			want:   "/autospec.implement --task T100",
		},
		"task ID with prompt containing spaces": {
			taskID: "T003",
			prompt: "focus on error handling",
			want:   `/autospec.implement --task T003 "focus on error handling"`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			te := NewTaskExecutor(&Executor{}, "specs/", false)
			result := te.buildTaskCommand(tt.taskID, tt.prompt)

			if result != tt.want {
				t.Errorf("buildTaskCommand(%q, %q) = %q, want %q",
					tt.taskID, tt.prompt, result, tt.want)
			}
		})
	}
}

// TestTaskExecutor_ShouldSkipTask tests task skip logic.
func TestTaskExecutor_ShouldSkipTask(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		task       validation.TaskItem
		idx        int
		totalTasks int
		want       bool
	}{
		"completed task returns true": {
			task:       validation.TaskItem{ID: "T001", Title: "Test", Status: "Completed"},
			idx:        0,
			totalTasks: 5,
			want:       true,
		},
		"completed lowercase returns true": {
			task:       validation.TaskItem{ID: "T002", Title: "Test", Status: "completed"},
			idx:        1,
			totalTasks: 5,
			want:       true,
		},
		"blocked task returns true": {
			task:       validation.TaskItem{ID: "T003", Title: "Test", Status: "Blocked"},
			idx:        2,
			totalTasks: 5,
			want:       true,
		},
		"blocked lowercase returns true": {
			task:       validation.TaskItem{ID: "T004", Title: "Test", Status: "blocked"},
			idx:        3,
			totalTasks: 5,
			want:       true,
		},
		"pending task returns false": {
			task:       validation.TaskItem{ID: "T005", Title: "Test", Status: "Pending"},
			idx:        0,
			totalTasks: 5,
			want:       false,
		},
		"in progress task returns false": {
			task:       validation.TaskItem{ID: "T006", Title: "Test", Status: "InProgress"},
			idx:        1,
			totalTasks: 5,
			want:       false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := shouldSkipTask(tt.task, tt.idx, tt.totalTasks)

			if result != tt.want {
				t.Errorf("shouldSkipTask(%+v, %d, %d) = %v, want %v",
					tt.task, tt.idx, tt.totalTasks, result, tt.want)
			}
		})
	}
}

// TestTaskExecutor_GetOrderedTasksForExecution tests task ordering retrieval.
func TestTaskExecutor_GetOrderedTasksForExecution(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupTasks func(t *testing.T, dir string)
		wantErr    bool
		errMsg     string
	}{
		"returns error for nonexistent tasks file": {
			setupTasks: func(t *testing.T, dir string) {
				// Don't create tasks.yaml
			},
			wantErr: true,
		},
		"returns error for empty tasks": {
			setupTasks: func(t *testing.T, dir string) {
				tasksContent := `_meta:
  artifact_type: tasks
  version: "1.0.0"
phases: []
`
				tasksPath := filepath.Join(dir, "tasks.yaml")
				if err := os.WriteFile(tasksPath, []byte(tasksContent), 0644); err != nil {
					t.Fatalf("failed to write tasks.yaml: %v", err)
				}
			},
			wantErr: true,
			errMsg:  "no tasks found",
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

			te := NewTaskExecutor(&Executor{}, filepath.Join(tempDir, "specs"), false)
			tasksPath := filepath.Join(specDir, "tasks.yaml")
			_, _, err := te.getOrderedTasksForExecution(tasksPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("getOrderedTasksForExecution() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestTaskExecutor_FindTaskStartIndex tests task start index logic.
func TestTaskExecutor_FindTaskStartIndex(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		orderedTasks []validation.TaskItem
		allTasks     []validation.TaskItem
		fromTask     string
		wantIdx      int
		wantErr      bool
	}{
		"empty fromTask returns 0": {
			orderedTasks: []validation.TaskItem{
				{ID: "T001", Status: "Pending"},
				{ID: "T002", Status: "Pending"},
			},
			allTasks: []validation.TaskItem{
				{ID: "T001", Status: "Pending"},
				{ID: "T002", Status: "Pending"},
			},
			fromTask: "",
			wantIdx:  0,
			wantErr:  false,
		},
		"nonexistent task returns error": {
			orderedTasks: []validation.TaskItem{
				{ID: "T001", Status: "Pending"},
				{ID: "T002", Status: "Pending"},
			},
			allTasks: []validation.TaskItem{
				{ID: "T001", Status: "Pending"},
				{ID: "T002", Status: "Pending"},
			},
			fromTask: "T999",
			wantIdx:  0,
			wantErr:  true,
		},
		"finds correct index for existing task": {
			orderedTasks: []validation.TaskItem{
				{ID: "T001", Status: "Completed"},
				{ID: "T002", Status: "Pending"},
				{ID: "T003", Status: "Pending"},
			},
			allTasks: []validation.TaskItem{
				{ID: "T001", Status: "Completed"},
				{ID: "T002", Status: "Pending", Dependencies: []string{"T001"}},
				{ID: "T003", Status: "Pending", Dependencies: []string{"T002"}},
			},
			fromTask: "T002",
			wantIdx:  1,
			wantErr:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			te := NewTaskExecutor(&Executor{}, "specs/", false)
			idx, err := te.findTaskStartIndex(tt.orderedTasks, tt.allTasks, tt.fromTask)

			if (err != nil) != tt.wantErr {
				t.Errorf("findTaskStartIndex() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && idx != tt.wantIdx {
				t.Errorf("findTaskStartIndex() = %d, want %d", idx, tt.wantIdx)
			}
		})
	}
}

// TestTaskExecutor_MethodSignatures verifies method signatures match interface.
func TestTaskExecutor_MethodSignatures(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		name string
	}{
		"verifies ExecuteTaskLoop signature": {
			name: "ExecuteTaskLoop",
		},
		"verifies ExecuteSingleTask signature": {
			name: "ExecuteSingleTask",
		},
	}

	for name := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			te := NewTaskExecutor(&Executor{}, "specs/", false)

			// Verify method signatures match interface
			var _ func(string, string, []validation.TaskItem, int, int, string) error = te.ExecuteTaskLoop
			var _ func(string, string, string, string) error = te.ExecuteSingleTask
		})
	}
}

// TestTaskExecutor_ValidateTaskCompleted tests task completion validation.
func TestTaskExecutor_ValidateTaskCompleted(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupTasks func(t *testing.T, dir string)
		taskID     string
		wantErr    bool
	}{
		"returns error for nonexistent tasks file": {
			setupTasks: func(t *testing.T, dir string) {
				// Don't create tasks.yaml
			},
			taskID:  "T001",
			wantErr: true,
		},
		"returns error for nonexistent task": {
			setupTasks: func(t *testing.T, dir string) {
				tasksContent := `_meta:
  artifact_type: tasks
  version: "1.0.0"
phases:
  - number: 1
    title: "Phase 1"
    tasks:
      - id: "T001"
        title: "Task 1"
        status: "Completed"
`
				tasksPath := filepath.Join(dir, "tasks.yaml")
				if err := os.WriteFile(tasksPath, []byte(tasksContent), 0644); err != nil {
					t.Fatalf("failed to write tasks.yaml: %v", err)
				}
			},
			taskID:  "T999",
			wantErr: true,
		},
		"returns nil for completed task": {
			setupTasks: func(t *testing.T, dir string) {
				tasksContent := `_meta:
  artifact_type: tasks
  version: "1.0.0"
phases:
  - number: 1
    title: "Phase 1"
    tasks:
      - id: "T001"
        title: "Task 1"
        status: "Completed"
`
				tasksPath := filepath.Join(dir, "tasks.yaml")
				if err := os.WriteFile(tasksPath, []byte(tasksContent), 0644); err != nil {
					t.Fatalf("failed to write tasks.yaml: %v", err)
				}
			},
			taskID:  "T001",
			wantErr: false,
		},
		"returns error for pending task": {
			setupTasks: func(t *testing.T, dir string) {
				tasksContent := `_meta:
  artifact_type: tasks
  version: "1.0.0"
phases:
  - number: 1
    title: "Phase 1"
    tasks:
      - id: "T001"
        title: "Task 1"
        status: "Pending"
`
				tasksPath := filepath.Join(dir, "tasks.yaml")
				if err := os.WriteFile(tasksPath, []byte(tasksContent), 0644); err != nil {
					t.Fatalf("failed to write tasks.yaml: %v", err)
				}
			},
			taskID:  "T001",
			wantErr: true,
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

			te := NewTaskExecutor(&Executor{}, filepath.Join(tempDir, "specs"), false)
			err := te.validateTaskCompleted(specDir, tt.taskID)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateTaskCompleted() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestTaskExecutor_VerifyTaskCompletion tests task completion verification.
func TestTaskExecutor_VerifyTaskCompletion(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupTasks func(t *testing.T, dir string)
		taskID     string
		wantErr    bool
	}{
		"returns error for nonexistent tasks file": {
			setupTasks: func(t *testing.T, dir string) {
				// Don't create tasks.yaml
			},
			taskID:  "T001",
			wantErr: true,
		},
		"returns nil for completed task": {
			setupTasks: func(t *testing.T, dir string) {
				tasksContent := `_meta:
  artifact_type: tasks
  version: "1.0.0"
phases:
  - number: 1
    title: "Phase 1"
    tasks:
      - id: "T001"
        title: "Task 1"
        status: "Completed"
`
				tasksPath := filepath.Join(dir, "tasks.yaml")
				if err := os.WriteFile(tasksPath, []byte(tasksContent), 0644); err != nil {
					t.Fatalf("failed to write tasks.yaml: %v", err)
				}
			},
			taskID:  "T001",
			wantErr: false,
		},
		"returns error for incomplete task": {
			setupTasks: func(t *testing.T, dir string) {
				tasksContent := `_meta:
  artifact_type: tasks
  version: "1.0.0"
phases:
  - number: 1
    title: "Phase 1"
    tasks:
      - id: "T001"
        title: "Task 1"
        status: "InProgress"
`
				tasksPath := filepath.Join(dir, "tasks.yaml")
				if err := os.WriteFile(tasksPath, []byte(tasksContent), 0644); err != nil {
					t.Fatalf("failed to write tasks.yaml: %v", err)
				}
			},
			taskID:  "T001",
			wantErr: true,
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

			te := NewTaskExecutor(&Executor{}, filepath.Join(tempDir, "specs"), false)
			tasksPath := filepath.Join(specDir, "tasks.yaml")
			err := te.verifyTaskCompletion(tasksPath, tt.taskID)

			if (err != nil) != tt.wantErr {
				t.Errorf("verifyTaskCompletion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
