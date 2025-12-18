# Arch 4: Complete Dependency Injection (MEDIUM PRIORITY)

**Location:** Multiple packages (workflow, validation, config)
**Impact:** MEDIUM - Significantly improves testability
**Effort:** MEDIUM
**Dependencies:** Should be done BEFORE arch-1 and arch-2 for better refactoring

## Problem Statement

Inconsistent DI pattern across codebase:
- Some components accept interfaces (good): NotificationHandler, PreflightChecker
- Others use concrete types (poor): Executor, Configuration

This makes testing difficult and creates tight coupling.

## Current State

```go
// Good - accepts interface
type Executor struct {
    NotificationHandler *notify.Handler    // Optional, injectable
    ProgressDisplay     *progress.ProgressDisplay
}

// Poor - concrete types
type WorkflowOrchestrator struct {
    Executor    *Executor              // Concrete, not injectable
    Config      *config.Configuration  // Concrete, not injectable
}
```

## Target Pattern

```go
// Define interfaces for all dependencies
type Executor interface {
    Execute(ctx context.Context, command string) error
    ExecuteWithRetry(ctx context.Context, command string, maxRetries int) error
}

type Configuration interface {
    GetMaxRetries() int
    GetTimeout() time.Duration
    GetClaudeCmd() string
}

type Validator interface {
    Validate(artifactType string, path string) error
}

type RetryStore interface {
    LoadState(specName, phase string) (*RetryState, error)
    SaveState(state *RetryState) error
}

// Deps struct pattern
type WorkflowOrchestratorDeps struct {
    Executor   Executor
    Config     Configuration
    Validator  Validator
    RetryStore RetryStore
}

func NewWorkflowOrchestrator(deps WorkflowOrchestratorDeps) *WorkflowOrchestrator
```

## Implementation Approach

1. Define interfaces in internal/workflow/interfaces.go
2. Create Configuration interface in internal/config/
3. Create Validator interface in internal/validation/
4. Create RetryStore interface in internal/retry/
5. Update constructors to accept deps struct
6. Update CLI commands to provide deps
7. Create mock implementations for testing
8. Update existing tests to use mocks

## Acceptance Criteria

- [ ] Executor interface defined
- [ ] Configuration interface defined
- [ ] Validator interface defined
- [ ] RetryStore interface defined
- [ ] WorkflowOrchestrator uses deps pattern
- [ ] Mock implementations for all interfaces
- [ ] All tests pass with new pattern

## Non-Functional Requirements

- Interfaces in dedicated interfaces.go files
- Mocks in mocks_test.go files
- Accept interfaces, return concrete types
- No breaking changes to CLI commands

## Command

```bash
autospec specify "$(cat .dev/tasks/arch/arch-4-dependency-injection.md)"
```
