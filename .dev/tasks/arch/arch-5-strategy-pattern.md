# Arch 5: Strategy Pattern for Execute Methods (MEDIUM PRIORITY)

> **Status: SKIP**
>
> **Reason:** The codebase already implements Strategy pattern correctly via `StageExecutorInterface`, `PhaseExecutorInterface`, and `TaskExecutorInterface`. The 12 Execute* methods on WorkflowOrchestrator are thin 8-12 line delegation wrappers—exactly what an orchestrator should be. Adding another strategy layer would create indirection with no benefit. The "combinatorial explosion" claim is inaccurate; the orchestrator correctly delegates to existing strategy implementations.
>
> **Reviewed:** 2025-12-18

**Location:** `internal/workflow/workflow.go`
**Impact:** MEDIUM - Reduces method count, improves maintainability
**Effort:** MEDIUM
**Dependencies:** Complete arch-1 (WorkflowOrchestrator split) first

## Problem Statement

WorkflowOrchestrator has 10+ Execute variants creating combinatorial explosion:
- ExecuteSpecify()
- ExecutePlan()
- ExecuteTasks()
- ExecuteImplement()
- ExecuteImplementDefault()
- ExecuteImplementWithPhases()
- ExecuteImplementWithTasks()
- ExecuteImplementSinglePhase()
- ExecuteImplementFromPhase()
- etc.

## Current Anti-Pattern

```go
// 10+ variants on WorkflowOrchestrator
func (o *Orchestrator) ExecuteImplementWithPhases(...) error
func (o *Orchestrator) ExecuteImplementWithTasks(...) error
func (o *Orchestrator) ExecuteImplementSinglePhase(...) error
func (o *Orchestrator) ExecuteImplementFromPhase(...) error
```

## Target Pattern

```go
// Strategy interface
type ExecutionStrategy interface {
    Execute(ctx context.Context, spec SpecContext) error
    Name() string
}

// Implementations
type PhaseStrategy struct {
    phaseExecutor PhaseExecutor
    startPhase    int
    singlePhase   bool
}

type TaskStrategy struct {
    taskExecutor TaskExecutor
}

type DefaultStrategy struct {
    phaseStrategy PhaseStrategy
}

// Orchestrator uses strategy
func (o *Orchestrator) Execute(ctx context.Context, strategy ExecutionStrategy, spec SpecContext) error {
    return strategy.Execute(ctx, spec)
}
```

## Implementation Approach

1. Define ExecutionStrategy interface
2. Create PhaseStrategy for phase-based execution
3. Create TaskStrategy for task-based execution
4. Create DefaultStrategy wrapping PhaseStrategy
5. Create SinglePhaseStrategy for --single-phase flag
6. Refactor CLI commands to build strategies
7. Remove Execute* method variants
8. Update tests to use strategy pattern

## Acceptance Criteria

- [ ] ExecutionStrategy interface defined
- [ ] PhaseStrategy implements interface
- [ ] TaskStrategy implements interface
- [ ] DefaultStrategy wraps PhaseStrategy
- [ ] CLI builds strategies based on flags
- [ ] Orchestrator.Execute() accepts strategy
- [ ] All existing tests pass

## Non-Functional Requirements

- Strategy creation in factory function
- Each strategy <100 LOC
- Clear strategy names for logging
- Map-based table tests for strategies

## Command

```bash
autospec specify "$(cat .dev/tasks/arch/arch-5-strategy-pattern.md)"
```
