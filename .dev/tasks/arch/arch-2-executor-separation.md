# Arch 2: Extract Executor Concerns (HIGH PRIORITY)

**Location:** `internal/workflow/executor.go` (451 LOC, 18 methods)
**Impact:** HIGH - Improves testability and separation of concerns
**Effort:** MEDIUM
**Dependencies:** None, can be done independently

## Problem Statement

Executor class mixes multiple concerns:
- Execution (Claude commands)
- Display (progress, spinner)
- Notifications (notification handler calls)
- Retry logic (retry state management)

This makes unit testing difficult and creates tight coupling.

## Current Structure

```go
type Executor struct {
    ClaudeCmd            string
    CustomClaudeCmd      string
    MaxRetries           int
    Timeout              time.Duration
    Debug                bool
    NotificationHandler  *notify.Handler
    ProgressDisplay      *progress.ProgressDisplay
    // ... mixed concerns
}
```

## Target Structure

```go
// 1. Executor - pure execution
type Executor struct {
    claude ClaudeRunner // interface
}

// 2. ProgressController - display concerns
type ProgressController struct {
    display *progress.ProgressDisplay
    spinner *spinner.Spinner
}

// 3. NotifyDispatcher - notification routing
type NotifyDispatcher struct {
    handler NotificationHandler // interface
}
```

## Implementation Approach

1. Define ClaudeRunner interface for command execution
2. Extract ProgressController for all display logic
3. Extract NotifyDispatcher for notification handling
4. Refactor Executor to pure execution via ClaudeRunner
5. Compose components in workflow layer
6. Update tests to mock individual concerns

## Acceptance Criteria

- [ ] ClaudeRunner interface defined
- [ ] ProgressController handles all display/spinner logic
- [ ] NotifyDispatcher handles notification routing
- [ ] Executor uses ClaudeRunner interface
- [ ] Each component independently testable
- [ ] All existing tests pass

## Non-Functional Requirements

- All functions under 40 lines
- All errors wrapped with context
- Interfaces in separate file (interfaces.go)
- Map-based table tests

## Command

```bash
autospec specify "$(cat .dev/tasks/arch/arch-2-executor-separation.md)"
```
