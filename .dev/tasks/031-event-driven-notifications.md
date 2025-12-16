# 31. Event-Driven Notification Architecture

## Summary

Refactor notification handling from duplicated per-command boilerplate to a centralized event-driven architecture with pub/sub pattern and command lifecycle management.

## Problem

Current notification handling violates DRY with ~8-10 lines of boilerplate repeated across 11 CLI commands:

```go
// This pattern is duplicated in EVERY command
notifHandler := notify.NewHandler(cfg.Notifications)
startTime := time.Now()
notifHandler.SetStartTime(startTime)
orch.Executor.NotificationHandler = notifHandler
execErr := orch.ExecuteSpecify(...)
duration := time.Since(startTime)
success := execErr == nil
notifHandler.OnCommandComplete("specify", success, duration)
```

### Anti-Patterns Identified

| Issue | Description |
|-------|-------------|
| **Boilerplate Duplication** | Same 8-10 lines copied across 11 commands |
| **Manual Timing** | Each command manages startTime/duration manually |
| **Manual Success Tracking** | Each command determines success/failure state |
| **Tight Coupling** | Commands know about notification handler internals |
| **No Event Bus** | Direct method calls instead of decoupled events |
| **Inconsistent Error Handling** | Error notification may be handled differently |
| **No Lifecycle Abstraction** | Commands manually orchestrate pre/post logic |

### Affected Commands

All 11 workflow commands have this duplication:
- `specify`, `plan`, `tasks`, `clarify`, `analyze`
- `checklist`, `constitution`, `implement`, `run`, `prep`, `all`

## Solution

Implement an event-driven architecture with three components:

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                          CLI Command                                │
│  return lifecycle.Run("specify", cfg, func() error {                │
│      return orch.ExecuteSpecify(...)                                │
│  })                                                                 │
└─────────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Lifecycle Manager                               │
│  - Tracks timing automatically                                      │
│  - Emits CommandStart, CommandComplete, Error events                │
│  - Handles context/cancellation                                     │
└─────────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Event Bus                                   │
│  - Thread-safe pub/sub                                              │
│  - Async dispatch with timeout                                      │
│  - Multiple subscribers supported                                   │
└─────────────────────────────────────────────────────────────────────┘
                                │
                    ┌───────────┴───────────┐
                    ▼                       ▼
        ┌───────────────────┐   ┌───────────────────┐
        │ NotificationSub   │   │ Future: Metrics   │
        │ (sound/visual)    │   │ Logging, Telemetry│
        └───────────────────┘   └───────────────────┘
```

### Target State

Commands become single-line wrappers:

```go
// BEFORE: 8-10 lines of boilerplate
notifHandler := notify.NewHandler(cfg.Notifications)
startTime := time.Now()
notifHandler.SetStartTime(startTime)
orch.Executor.NotificationHandler = notifHandler
execErr := orch.ExecuteSpecify(...)
duration := time.Since(startTime)
success := execErr == nil
notifHandler.OnCommandComplete("specify", success, duration)

// AFTER: 3 lines, zero notification knowledge
return lifecycle.Run("specify", cfg, func() error {
    return orch.ExecuteSpecify(...)
})
```

## Implementation

### Phase 1: Event System Foundation

Create `internal/events/` package with core types.

**Files to create:**
- `internal/events/event.go` - Event types and structs
- `internal/events/bus.go` - Thread-safe event bus
- `internal/events/bus_test.go` - Comprehensive tests

**Event Types:**

```go
type EventType string

const (
    EventCommandStart    EventType = "command.start"
    EventCommandComplete EventType = "command.complete"
    EventStageStart      EventType = "stage.start"
    EventStageComplete   EventType = "stage.complete"
    EventError           EventType = "error"
    EventValidationFail  EventType = "validation.fail"
)

type Event struct {
    Type      EventType
    Name      string        // command or stage name
    Success   bool
    Duration  time.Duration
    Error     error
    Timestamp time.Time
    Metadata  map[string]any
}
```

**Event Bus Interface:**

```go
type Subscriber func(Event)

type Bus interface {
    Subscribe(eventType EventType, handler Subscriber) (unsubscribe func())
    SubscribeAll(handler Subscriber) (unsubscribe func())
    Publish(event Event)
    PublishAsync(event Event) // Non-blocking with timeout
}
```

### Phase 2: Lifecycle Manager

Create `internal/lifecycle/` package for command wrapping.

**Files to create:**
- `internal/lifecycle/manager.go` - Core lifecycle logic
- `internal/lifecycle/manager_test.go` - Tests

**Core Function:**

```go
// Run wraps command execution with lifecycle events
func Run(name string, cfg *config.Config, fn func() error) error {
    ctx := context.Background()
    startTime := time.Now()

    // Emit start event
    bus.Publish(Event{
        Type:      EventCommandStart,
        Name:      name,
        Timestamp: startTime,
    })

    // Execute command
    err := fn()

    // Calculate duration
    duration := time.Since(startTime)
    success := err == nil

    // Emit complete event
    bus.Publish(Event{
        Type:     EventCommandComplete,
        Name:     name,
        Success:  success,
        Duration: duration,
        Error:    err,
    })

    return err
}
```

**Additional Features:**
- `RunWithContext()` - Support cancellation
- `RunStage()` - For workflow stage events
- Global bus initialization via `Init(cfg)`

### Phase 3: Refactor Notification Handler

Convert `internal/notify/` to be an event subscriber.

**Files to modify:**
- `internal/notify/handler.go` - Convert to subscriber pattern
- `internal/notify/subscriber.go` - New file for subscription logic

**Changes:**

```go
// handler.go - Add subscriber initialization
func (h *Handler) Subscribe(bus events.Bus) {
    bus.Subscribe(events.EventCommandComplete, h.onCommandComplete)
    bus.Subscribe(events.EventStageComplete, h.onStageComplete)
    bus.Subscribe(events.EventError, h.onError)
}

func (h *Handler) onCommandComplete(e events.Event) {
    if !h.config.OnCommandComplete {
        return
    }
    // Existing notification logic
}
```

**Keep Backward Compatibility:**
- Existing `OnCommandComplete()`, `OnStageComplete()`, `OnError()` methods remain
- They can be called directly or via events
- Gradual migration path

### Phase 4: Migrate CLI Commands

Update all 11 commands to use lifecycle wrapper.

**Migration per command:**

1. Remove notification handler creation
2. Remove timing code
3. Wrap execution in `lifecycle.Run()`
4. Remove `OnCommandComplete` call

**Example migration (specify.go):**

```go
// BEFORE
RunE: func(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load()
    if err != nil {
        return err
    }

    notifHandler := notify.NewHandler(cfg.Notifications)
    startTime := time.Now()
    notifHandler.SetStartTime(startTime)

    orch := workflow.NewOrchestrator(cfg)
    orch.Executor.NotificationHandler = notifHandler

    execErr := orch.ExecuteSpecify(featureDesc)

    duration := time.Since(startTime)
    success := execErr == nil
    notifHandler.OnCommandComplete("specify", success, duration)

    return execErr
}

// AFTER
RunE: func(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load()
    if err != nil {
        return err
    }

    return lifecycle.Run("specify", cfg, func() error {
        orch := workflow.NewOrchestrator(cfg)
        return orch.ExecuteSpecify(featureDesc)
    })
}
```

**Commands to migrate:**
- [ ] `internal/cli/specify.go`
- [ ] `internal/cli/plan.go`
- [ ] `internal/cli/tasks.go`
- [ ] `internal/cli/clarify.go`
- [ ] `internal/cli/analyze.go`
- [ ] `internal/cli/checklist.go`
- [ ] `internal/cli/constitution.go`
- [ ] `internal/cli/implement.go`
- [ ] `internal/cli/run.go`
- [ ] `internal/cli/prep.go`
- [ ] `internal/cli/all.go`

### Phase 5: Update Executor Events

Refactor `internal/workflow/executor.go` to emit events.

**Current direct calls:**
```go
if e.NotificationHandler != nil {
    e.NotificationHandler.OnStageComplete(string(stage), true)
}
```

**New event emission:**
```go
events.Publish(Event{
    Type:    EventStageComplete,
    Name:    string(stage),
    Success: true,
})
```

### Phase 6: Testing & Cleanup

1. **Update regression test** in `internal/cli/specify_test.go`
   - `TestAllCommandsHaveNotificationSupport` → verify lifecycle usage

2. **Remove dead code**
   - Delete `NotificationHandler` field from `Executor` if no longer needed
   - Remove manual notification code from commands

3. **Add integration tests**
   - Test event flow end-to-end
   - Test subscriber receives correct events

## File Changes Summary

### New Files

| File | Purpose |
|------|---------|
| `internal/events/event.go` | Event types and structs |
| `internal/events/bus.go` | Thread-safe event bus implementation |
| `internal/events/bus_test.go` | Event bus tests |
| `internal/events/doc.go` | Package documentation |
| `internal/lifecycle/manager.go` | Command lifecycle wrapper |
| `internal/lifecycle/manager_test.go` | Lifecycle tests |
| `internal/notify/subscriber.go` | Event subscription logic |

### Modified Files

| File | Changes |
|------|---------|
| `internal/notify/handler.go` | Add `Subscribe()` method |
| `internal/workflow/executor.go` | Emit events instead of direct calls |
| `internal/cli/specify.go` | Use lifecycle wrapper |
| `internal/cli/plan.go` | Use lifecycle wrapper |
| `internal/cli/tasks.go` | Use lifecycle wrapper |
| `internal/cli/clarify.go` | Use lifecycle wrapper |
| `internal/cli/analyze.go` | Use lifecycle wrapper |
| `internal/cli/checklist.go` | Use lifecycle wrapper |
| `internal/cli/constitution.go` | Use lifecycle wrapper |
| `internal/cli/implement.go` | Use lifecycle wrapper |
| `internal/cli/run.go` | Use lifecycle wrapper |
| `internal/cli/prep.go` | Use lifecycle wrapper |
| `internal/cli/all.go` | Use lifecycle wrapper |
| `internal/cli/specify_test.go` | Update regression test |

## Benefits

| Benefit | Description |
|---------|-------------|
| **DRY** | Remove ~100 lines of duplicated code |
| **Single Responsibility** | Commands do one thing; events handle cross-cutting concerns |
| **Extensibility** | Easy to add subscribers (metrics, logging, telemetry) |
| **Testability** | Event bus can be mocked; commands become simpler to test |
| **Decoupling** | Commands don't know about notifications |
| **Consistency** | All commands behave identically for lifecycle events |

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Event dispatch failures affect commands | Async dispatch with timeout; silent failures |
| Thread safety in event bus | Use `sync.RWMutex`; channels for dispatch |
| Breaking existing behavior | Feature flag; gradual migration; comprehensive tests |
| Increased complexity | Keep event types minimal; clear documentation |

## Future Extensions

Once event system is in place, easy to add:

1. **Metrics subscriber** - Track command durations, success rates
2. **Logging subscriber** - Structured logging of all events
3. **Telemetry subscriber** - Optional usage analytics
4. **Progress events** - `EventProgress` for long-running operations
5. **Validation events** - `EventValidationStart`, `EventValidationComplete`
6. **Retry events** - `EventRetryAttempt`, `EventRetryExhausted`

## Non-Functional Requirements

Per project standards, implementation MUST include:

- All functions under 40 lines; extract helpers for complex logic
- All errors wrapped with context using `fmt.Errorf("doing X: %w", err)`
- Tests using map-based table-driven pattern with `t.Parallel()`
- Pass all quality gates: `make test`, `make fmt`, `make lint`, `make build`

## Spec Command

```bash
autospec specify "Implement event-driven notification architecture. See .dev/tasks/031-event-driven-notifications.md for full details."
```
