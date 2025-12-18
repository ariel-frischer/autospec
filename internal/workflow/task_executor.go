// Package workflow provides task execution functionality.
// TaskExecutor handles task-level implementation execution.
// Related: internal/workflow/orchestrator.go, internal/workflow/interfaces.go (interface definition)
// Tags: workflow, task-executor, implementation, tasks
package workflow

import (
	"fmt"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/validation"
)

// TaskExecutor handles task-level implementation execution.
// It implements TaskExecutorInterface to enable dependency injection and testing.
// Each task is executed in a separate Claude session with task-specific context,
// providing fine-grained control over the implementation process.
type TaskExecutor struct {
	executor *Executor // Underlying executor for Claude command execution
	specsDir string    // Base directory for spec storage (e.g., "specs/")
	debug    bool      // Enable debug logging
}

// NewTaskExecutor creates a new TaskExecutor with the given dependencies.
// executor: required, handles actual command execution with retry logic
// specsDir: required, base directory where spec directories are located
// debug: optional, enables verbose logging for troubleshooting
func NewTaskExecutor(executor *Executor, specsDir string, debug bool) *TaskExecutor {
	return &TaskExecutor{
		executor: executor,
		specsDir: specsDir,
		debug:    debug,
	}
}

// debugLog prints a debug message if debug mode is enabled.
func (te *TaskExecutor) debugLog(format string, args ...interface{}) {
	if te.debug {
		fmt.Printf("[DEBUG][TaskExecutor] "+format+"\n", args...)
	}
}

// ExecuteTaskLoop iterates through tasks from startIdx to end.
// Each task runs in a separate Claude session for isolation.
// specName: the spec directory name
// tasksPath: path to tasks.yaml file
// orderedTasks: tasks sorted by dependency order
// startIdx: 0-based index to start from
// totalTasks: total number of tasks (for progress display)
// prompt: optional custom prompt to pass to each task
func (te *TaskExecutor) ExecuteTaskLoop(specName, tasksPath string, orderedTasks []validation.TaskItem, startIdx, totalTasks int, prompt string) error {
	te.debugLog("ExecuteTaskLoop called: spec=%s, startIdx=%d, totalTasks=%d", specName, startIdx, totalTasks)
	specDir := filepath.Join(te.specsDir, specName)

	for i := startIdx; i < len(orderedTasks); i++ {
		task := orderedTasks[i]

		// Handle completed and blocked tasks
		if shouldSkipTask(task, i, totalTasks) {
			continue
		}

		fmt.Printf("[Task %d/%d] %s - %s\n", i+1, totalTasks, task.ID, task.Title)

		// Execute and verify task
		if err := te.executeAndVerifyTask(specName, tasksPath, task, prompt); err != nil {
			return fmt.Errorf("executing task %s: %w", task.ID, err)
		}

		fmt.Printf("✓ Task %s complete\n\n", task.ID)
	}

	te.printTasksSummary(tasksPath, specDir)
	return nil
}

// ExecuteSingleTask runs a specific task by ID.
// specName: the spec directory name
// taskID: task identifier (e.g., "T001")
// taskTitle: human-readable task title for display
// prompt: optional custom prompt
func (te *TaskExecutor) ExecuteSingleTask(specName, taskID, taskTitle, prompt string) error {
	te.debugLog("ExecuteSingleTask called: spec=%s, taskID=%s", specName, taskID)
	return te.executeSingleTaskSession(specName, taskID, taskTitle, prompt)
}

// executeAndVerifyTask executes a single task and verifies completion.
func (te *TaskExecutor) executeAndVerifyTask(specName, tasksPath string, task validation.TaskItem, prompt string) error {
	// Validate dependencies before executing
	freshTasks, err := validation.GetAllTasks(tasksPath)
	if err != nil {
		return fmt.Errorf("failed to refresh tasks: %w", err)
	}

	met, unmetDeps := validation.ValidateTaskDependenciesMet(task, freshTasks)
	if !met {
		fmt.Printf("⚠ Skipping task %s: dependencies not met (%v)\n", task.ID, unmetDeps)
		return nil
	}

	// Execute this task in a fresh Claude session
	if err := te.executeSingleTaskSession(specName, task.ID, task.Title, prompt); err != nil {
		return fmt.Errorf("task %s failed: %w", task.ID, err)
	}

	// Verify task completion
	return te.verifyTaskCompletion(tasksPath, task.ID)
}

// executeSingleTaskSession executes a single task in a fresh Claude session.
func (te *TaskExecutor) executeSingleTaskSession(specName, taskID, taskTitle, prompt string) error {
	te.debugLog("executeSingleTaskSession: taskID=%s, taskTitle=%s", taskID, taskTitle)

	command := te.buildTaskCommand(taskID, prompt)
	fmt.Printf("Executing: %s\n", command)

	return te.executeTaskWithValidation(specName, taskID, command)
}

// buildTaskCommand constructs the implement command with task filter.
func (te *TaskExecutor) buildTaskCommand(taskID, prompt string) string {
	if prompt != "" {
		return fmt.Sprintf("/autospec.implement --task %s \"%s\"", taskID, prompt)
	}
	return fmt.Sprintf("/autospec.implement --task %s", taskID)
}

// executeTaskWithValidation executes the task command with validation.
func (te *TaskExecutor) executeTaskWithValidation(specName, taskID, command string) error {
	result, err := te.executor.ExecuteStage(
		specName,
		StageImplement,
		command,
		func(specDir string) error {
			// For task execution, we validate the specific task is completed
			return te.validateTaskCompleted(specDir, taskID)
		},
	)

	if err != nil {
		if result.Exhausted {
			fmt.Printf("\nTask %s paused.\n", taskID)
			fmt.Printf("To resume: autospec implement --tasks --from-task %s\n", taskID)
			return fmt.Errorf("task %s exhausted retries: %w", taskID, err)
		}
		return fmt.Errorf("executing task %s session: %w", taskID, err)
	}

	return nil
}

// validateTaskCompleted checks if a specific task is completed.
func (te *TaskExecutor) validateTaskCompleted(specDir, taskID string) error {
	tasksPath := validation.GetTasksFilePath(specDir)
	allTasks, err := validation.GetAllTasks(tasksPath)
	if err != nil {
		return fmt.Errorf("getting all tasks: %w", err)
	}

	task, err := validation.GetTaskByID(allTasks, taskID)
	if err != nil {
		return fmt.Errorf("getting task %s: %w", taskID, err)
	}

	if task.Status != "Completed" && task.Status != "completed" {
		return fmt.Errorf("task %s not completed (status: %s)", taskID, task.Status)
	}
	return nil
}

// verifyTaskCompletion checks that a task completed successfully.
func (te *TaskExecutor) verifyTaskCompletion(tasksPath, taskID string) error {
	freshTasks, err := validation.GetAllTasks(tasksPath)
	if err != nil {
		return fmt.Errorf("failed to verify task completion: %w", err)
	}

	freshTask, err := validation.GetTaskByID(freshTasks, taskID)
	if err != nil {
		return fmt.Errorf("failed to find task %s after execution: %w", taskID, err)
	}

	if freshTask.Status != "Completed" && freshTask.Status != "completed" {
		fmt.Printf("\n⚠ Task %s did not complete (status: %s). Run 'autospec implement --tasks --from-task %s' to retry.\n",
			taskID, freshTask.Status, taskID)
		return fmt.Errorf("task %s did not complete after execution (status: %s)", taskID, freshTask.Status)
	}

	return nil
}

// getOrderedTasksForExecution retrieves and orders tasks by dependencies.
func (te *TaskExecutor) getOrderedTasksForExecution(tasksPath string) ([]validation.TaskItem, []validation.TaskItem, error) {
	allTasks, err := validation.GetAllTasks(tasksPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	if len(allTasks) == 0 {
		return nil, nil, fmt.Errorf("no tasks found in tasks.yaml")
	}

	orderedTasks, err := validation.GetTasksInDependencyOrder(allTasks)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to order tasks by dependencies: %w", err)
	}

	return orderedTasks, allTasks, nil
}

// findTaskStartIndex finds the starting index for task execution.
func (te *TaskExecutor) findTaskStartIndex(orderedTasks, allTasks []validation.TaskItem, fromTask string) (int, error) {
	if fromTask == "" {
		return 0, nil
	}

	// Find task index
	startIdx := -1
	for i, task := range orderedTasks {
		if task.ID == fromTask {
			startIdx = i
			break
		}
	}

	if startIdx == -1 {
		taskIDs := make([]string, len(orderedTasks))
		for i, t := range orderedTasks {
			taskIDs[i] = t.ID
		}
		return 0, fmt.Errorf("task %s not found in tasks.yaml (available: %v)", fromTask, taskIDs)
	}

	// Validate that fromTask's dependencies are met
	fromTaskItem, _ := validation.GetTaskByID(allTasks, fromTask)
	met, unmetDeps := validation.ValidateTaskDependenciesMet(*fromTaskItem, allTasks)
	if !met {
		return 0, fmt.Errorf("cannot start from task %s: dependencies not met (%v)", fromTask, unmetDeps)
	}

	return startIdx, nil
}

// printTasksSummary prints the final task execution summary and marks spec as completed.
func (te *TaskExecutor) printTasksSummary(tasksPath, specDir string) {
	fmt.Println("✓ All tasks processed!")
	fmt.Println()
	stats, statsErr := validation.GetTaskStats(tasksPath)
	if statsErr == nil && stats.TotalTasks > 0 {
		fmt.Println("Task Summary:")
		fmt.Print(validation.FormatTaskSummary(stats))
	}

	// Mark spec as completed
	markSpecCompletedAndPrint(specDir)
}

// shouldSkipTask checks if a task should be skipped and prints appropriate message.
// This is a package-level function used by both TaskExecutor and WorkflowOrchestrator.
func shouldSkipTask(task validation.TaskItem, idx, totalTasks int) bool {
	if task.Status == "Completed" || task.Status == "completed" {
		fmt.Printf("✓ Task %d/%d: %s - %s (already completed)\n", idx+1, totalTasks, task.ID, task.Title)
		return true
	}
	if task.Status == "Blocked" || task.Status == "blocked" {
		fmt.Printf("⚠ Task %d/%d: %s - %s (blocked)\n", idx+1, totalTasks, task.ID, task.Title)
		return true
	}
	return false
}

// PrepareTaskExecution retrieves ordered tasks and determines start index.
// This method encapsulates task ordering and start position validation.
func (te *TaskExecutor) PrepareTaskExecution(tasksPath string, fromTask string) ([]validation.TaskItem, int, int, error) {
	orderedTasks, allTasks, err := te.getOrderedTasksForExecution(tasksPath)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("getting ordered tasks: %w", err)
	}

	totalTasks := len(orderedTasks)

	startIdx, err := te.findTaskStartIndex(orderedTasks, allTasks, fromTask)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("finding task start index: %w", err)
	}

	return orderedTasks, startIdx, totalTasks, nil
}

// Compile-time interface compliance check.
var _ TaskExecutorInterface = (*TaskExecutor)(nil)
