// Package workflow defines interfaces for executor types enabling dependency injection and testing.
// Related: internal/workflow/orchestrator.go, internal/workflow/mocks_test.go (test doubles)
// Tags: workflow, interfaces, dependency-injection, executors
package workflow

import "github.com/ariel-frischer/autospec/internal/validation"

// ClaudeRunner abstracts Claude command execution for testability.
// This interface enables mocking Claude commands in unit tests without
// requiring actual Claude CLI installation or network access.
//
// Design rationale: Extracted from ClaudeExecutor to separate execution
// concerns from progress display and notification routing. This allows
// independent testing of command execution logic.
//
// Primary implementation: ClaudeExecutor in claude.go
type ClaudeRunner interface {
	// Execute runs a Claude command with the given prompt.
	// The implementation handles timeout, environment setup, and command
	// execution. Output is streamed to stdout in real-time.
	//
	// Returns nil on success, or a wrapped error on failure.
	// Returns TimeoutError if the configured timeout is exceeded.
	Execute(prompt string) error

	// FormatCommand returns a human-readable command string for display.
	// This is used in error messages, debug output, and progress display
	// to show users what command would be executed.
	//
	// The returned string matches the actual command that Execute would run.
	FormatCommand(prompt string) string
}

// StageExecutorInterface defines the contract for stage execution (specify, plan, tasks).
// Implementations handle the core workflow stages that transform feature descriptions into
// specifications, plans, and task breakdowns. Also handles auxiliary stages like constitution,
// clarify, checklist, and analyze.
//
// Design rationale: Narrow interface following Go idiom "accept interfaces, return concrete types"
// to enable focused mocking in unit tests without coupling to implementation details.
type StageExecutorInterface interface {
	// ExecuteSpecify runs the specify stage for a feature description.
	// Returns the spec name (e.g., "003-command-timeout") on success.
	// The spec name is derived from the newly created spec directory.
	ExecuteSpecify(featureDescription string) (string, error)

	// ExecutePlan runs the plan stage for an existing spec.
	// specNameArg: spec name or empty string to auto-detect from git branch
	// prompt: optional custom prompt to pass to the plan command
	ExecutePlan(specNameArg string, prompt string) error

	// ExecuteTasks runs the tasks stage for an existing spec.
	// specNameArg: spec name or empty string to auto-detect from git branch
	// prompt: optional custom prompt to pass to the tasks command
	ExecuteTasks(specNameArg string, prompt string) error

	// ExecuteConstitution runs the constitution stage with optional prompt.
	// Constitution creates or updates the project constitution file.
	ExecuteConstitution(prompt string) error

	// ExecuteClarify runs the clarify stage with optional prompt.
	// Clarify refines the specification by asking targeted clarification questions.
	ExecuteClarify(specName string, prompt string) error

	// ExecuteChecklist runs the checklist stage with optional prompt.
	// Checklist generates a custom checklist for the current feature.
	ExecuteChecklist(specName string, prompt string) error

	// ExecuteAnalyze runs the analyze stage with optional prompt.
	// Analyze performs cross-artifact consistency and quality analysis.
	ExecuteAnalyze(specName string, prompt string) error
}

// PhaseExecutorInterface defines the contract for phase-based implementation execution.
// Implementations handle iterating through implementation phases, each representing
// a logical grouping of related tasks (e.g., "Setup", "Core Implementation", "Polish").
//
// Phase execution provides coarse-grained progress control, allowing users to:
// - Execute all phases sequentially (--phases flag)
// - Execute a specific phase (--phase N flag)
// - Resume from a specific phase (--from-phase N flag)
// - Execute all in a single session (default mode)
type PhaseExecutorInterface interface {
	// ExecutePhaseLoop iterates through phases from startPhase to totalPhases.
	// Each phase runs in a separate Claude session with phase-specific context.
	// specName: the spec directory name (e.g., "003-command-timeout")
	// tasksPath: path to tasks.yaml file
	// phases: slice of PhaseInfo containing phase metadata
	// startPhase: 1-based phase number to start from
	// totalPhases: total number of phases
	// prompt: optional custom prompt to pass to each phase
	ExecutePhaseLoop(specName, tasksPath string, phases []validation.PhaseInfo, startPhase, totalPhases int, prompt string) error

	// ExecuteSinglePhase runs a specific phase in isolation.
	// specName: the spec directory name
	// phaseNumber: 1-based phase number to execute
	// prompt: optional custom prompt
	ExecuteSinglePhase(specName string, phaseNumber int, prompt string) error

	// ExecuteDefault runs all implementation in a single Claude session.
	// This is the default behavior when no --phases, --tasks, or --phase flags are specified.
	// specName: the spec directory name
	// specDir: full path to spec directory (for tasks.yaml lookup)
	// prompt: optional custom prompt
	// resume: whether to resume from previous session
	ExecuteDefault(specName, specDir, prompt string, resume bool) error
}

// TaskExecutorInterface defines the contract for task-level implementation execution.
// Implementations handle iterating through individual tasks in dependency order,
// providing fine-grained control over the implementation process.
//
// Task execution provides the most granular progress control, allowing users to:
// - Execute all tasks sequentially (--tasks flag)
// - Resume from a specific task (--from-task ID flag)
// - Track individual task completion status
type TaskExecutorInterface interface {
	// ExecuteTaskLoop iterates through tasks from startIdx to end.
	// Each task runs in a separate Claude session for isolation.
	// specName: the spec directory name
	// tasksPath: path to tasks.yaml file
	// orderedTasks: tasks sorted by dependency order
	// startIdx: 0-based index to start from
	// totalTasks: total number of tasks (for progress display)
	// prompt: optional custom prompt to pass to each task
	ExecuteTaskLoop(specName, tasksPath string, orderedTasks []validation.TaskItem, startIdx, totalTasks int, prompt string) error

	// ExecuteSingleTask runs a specific task by ID.
	// specName: the spec directory name
	// taskID: task identifier (e.g., "T001")
	// taskTitle: human-readable task title for display
	// prompt: optional custom prompt
	ExecuteSingleTask(specName, taskID, taskTitle, prompt string) error

	// PrepareTaskExecution retrieves ordered tasks and determines start index.
	// tasksPath: path to tasks.yaml file
	// fromTask: optional task ID to start from (empty string means start from beginning)
	// Returns: ordered tasks, start index, total tasks count, or error
	PrepareTaskExecution(tasksPath string, fromTask string) (orderedTasks []validation.TaskItem, startIdx, totalTasks int, err error)
}

// Compile-time interface compliance checks.
// These ensure that any future refactoring that breaks the interface contract
// will fail at compile time rather than runtime.
var (
	// Verify ClaudeExecutor satisfies ClaudeRunner
	_ ClaudeRunner = (*ClaudeExecutor)(nil)

	// Verify StageExecutor satisfies StageExecutorInterface
	_ StageExecutorInterface = (*StageExecutor)(nil)

	// Verify PhaseExecutor satisfies PhaseExecutorInterface
	_ PhaseExecutorInterface = (*PhaseExecutor)(nil)

	// Verify TaskExecutor satisfies TaskExecutorInterface
	_ TaskExecutorInterface = (*TaskExecutor)(nil)
)
