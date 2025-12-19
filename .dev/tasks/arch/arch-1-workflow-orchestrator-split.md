# Arch 1: Split WorkflowOrchestrator God Object (HIGH PRIORITY)

**Location:** `internal/workflow/workflow.go` (1,285 LOC, 38 methods)
**Impact:** HIGH - Core of the application, affects testability and maintainability
**Effort:** HIGH
**Dependencies:** Should complete arch-4 (DI interfaces) first for better testability

## Problem Statement

WorkflowOrchestrator violates Single Responsibility Principle by handling:
- Workflow orchestration (RunCompleteWorkflow, RunFullWorkflow)
- Stage execution (executeSpecifyPlanTasks, executeImplementStage)
- Phase management (ExecuteImplementWithPhases, executePhaseLoop)
- Task management (ExecuteImplementWithTasks, executeTaskLoop)
- Error handling (handleImplementError, validateTasksCompleteFunc)
- Output/printing (printFullWorkflowSummary, printPhaseCompletion)

The struct has 10+ Execute variants creating a combinatorial explosion.

## Current Structure

```go
type WorkflowOrchestrator struct {
    Executor         *Executor
    Config           *config.Configuration
    SpecsDir         string
    SkipPreflight    bool
    Debug            bool
    PreflightChecker PreflightChecker
}
```

## Target Structure

```go
// 1. Orchestrator - coordination only
type WorkflowOrchestrator struct {
    stageExecutor StageExecutor
    phaseExecutor PhaseExecutor
    taskExecutor  TaskExecutor
    config        Configuration // interface
}

// 2. StageExecutor - specify/plan/tasks stages
type StageExecutor struct { ... }

// 3. PhaseExecutor - phase-based implementation
type PhaseExecutor struct { ... }

// 4. TaskExecutor - task-level execution
type TaskExecutor struct { ... }
```

## Implementation Approach

1. Extract interfaces for each executor type
2. Create StageExecutor with specify/plan/tasks methods
3. Create PhaseExecutor with phase loop and context generation
4. Create TaskExecutor with task loop
5. Refactor WorkflowOrchestrator to compose executors
6. Migrate CLI commands to use new structure
7. Update tests to use mocked executors

## Acceptance Criteria

- [ ] StageExecutor handles specify, plan, tasks stages
- [ ] PhaseExecutor handles phase-based implementation
- [ ] TaskExecutor handles task-level execution
- [ ] WorkflowOrchestrator delegates to executor types
- [ ] Each executor <400 LOC
- [ ] All existing tests pass
- [ ] Test coverage maintained or improved

## Non-Functional Requirements

- All functions under 40 lines
- All errors wrapped with context
- Map-based table tests with t.Parallel()
- CLI lifecycle wrapper pattern maintained

## Command

```bash
autospec specify "$(cat .dev/tasks/arch/arch-1-workflow-orchestrator-split.md)"
```
