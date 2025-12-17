# Event System

This document describes autospec's event-driven architecture using [kelindar/event](https://github.com/kelindar/event), a high-performance in-process event dispatcher for Go.

## Table of Contents

- [Overview](#overview)
- [Why kelindar/event](#why-kelindarevent)
- [Architecture](#architecture)
- [Event Types](#event-types)
- [Usage Patterns](#usage-patterns)
- [Testing](#testing)
- [Best Practices](#best-practices)

---

## Overview

autospec uses an event-driven architecture to decouple command execution from cross-cutting concerns like notifications, logging, and metrics. Instead of commands directly calling notification handlers, they emit events that subscribers handle independently.

**Before (direct coupling):**
```go
func runSpecify(cmd *cobra.Command, args []string) error {
    startTime := time.Now()
    // ... command logic ...
    duration := time.Since(startTime)
    notifHandler.OnCommandComplete("specify", err == nil, duration)  // Tight coupling
    return err
}
```

**After (event-driven):**
```go
func runSpecify(cmd *cobra.Command, args []string) error {
    return lifecycle.Run("specify", func() error {
        // ... command logic ...
        return nil
    })
}
```

## Why kelindar/event

We chose [kelindar/event](https://github.com/kelindar/event) over a custom implementation for:

| Feature | Benefit |
|---------|---------|
| **4-10x faster than channels** | High throughput for future event-heavy scenarios |
| **Zero allocations** | No GC pressure, consistent performance |
| **Zero dependencies** | Only Go stdlib, aligns with our minimal-deps policy |
| **Type-safe generics** | Compile-time safety for event handlers |
| **Goroutine-per-subscriber** | Non-blocking async dispatch by default |
| **Battle-tested** | Production-ready, handles edge cases |

### Event Interface

Events must implement the `Type() uint32` method:

```go
type Event interface {
    Type() uint32
}
```

This allows the dispatcher to route events efficiently to interested subscribers.

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   CLI Command   │     │   Lifecycle     │     │   Event Bus     │
│   (specify)     │────▶│   Manager       │────▶│   (Dispatcher)  │
└─────────────────┘     └─────────────────┘     └────────┬────────┘
                                                         │
                         ┌───────────────────────────────┼───────────────────────────────┐
                         │                               │                               │
                         ▼                               ▼                               ▼
                ┌─────────────────┐             ┌─────────────────┐             ┌─────────────────┐
                │  Notification   │             │     Logger      │             │    Metrics      │
                │   Subscriber    │             │   Subscriber    │             │   Subscriber    │
                └─────────────────┘             └─────────────────┘             └─────────────────┘
```

### Components

| Component | Location | Responsibility |
|-----------|----------|----------------|
| Event Types | `internal/events/types.go` | Event struct definitions and type constants |
| Event Bus | `internal/events/bus.go` | Global dispatcher instance and helpers |
| Lifecycle Manager | `internal/lifecycle/run.go` | Wraps command execution, emits events |
| Notification Subscriber | `internal/notify/subscriber.go` | Handles sound/visual notifications |

## Event Types

### Core Events

```go
package events

// Event type constants (uint32 for kelindar/event compatibility)
const (
    TypeCommandComplete uint32 = iota + 1
    TypeStageComplete
)

// CommandCompleteEvent is emitted when a CLI command finishes execution.
type CommandCompleteEvent struct {
    Name     string        // Command name (e.g., "specify", "plan")
    Success  bool          // Whether command succeeded
    Duration time.Duration // Execution time
    Error    error         // Error if failed, nil otherwise
}

func (e CommandCompleteEvent) Type() uint32 { return TypeCommandComplete }

// StageCompleteEvent is emitted when a workflow stage finishes.
type StageCompleteEvent struct {
    Name     string        // Stage name (e.g., "specify", "plan")
    Success  bool          // Whether stage succeeded
    Duration time.Duration // Execution time
}

func (e StageCompleteEvent) Type() uint32 { return TypeStageComplete }
```

### Adding New Event Types

1. Add a new constant to the `const` block
2. Define a struct with relevant fields
3. Implement `Type() uint32` returning your constant
4. Document when the event is emitted

```go
const (
    // ... existing ...
    TypeValidationFailed uint32 = iota + 1
)

// ValidationFailedEvent is emitted when artifact validation fails.
type ValidationFailedEvent struct {
    Artifact string   // Which artifact failed (e.g., "spec.yaml")
    Errors   []string // Validation error messages
}

func (e ValidationFailedEvent) Type() uint32 { return TypeValidationFailed }
```

## Usage Patterns

### Global Dispatcher (Default)

For most use cases, use the global dispatcher via package-level functions:

```go
package events

import "github.com/kelindar/event"

// Global dispatcher instance
var bus = event.NewDispatcher()

// Subscribe registers a handler for a specific event type.
// Returns an unsubscribe function that MUST be deferred.
func Subscribe[T event.Event](handler func(T)) func() {
    return event.Subscribe(bus, handler)
}

// Publish emits an event to all registered subscribers.
func Publish[T event.Event](e T) {
    event.Publish(bus, e)
}
```

### Subscribing to Events

Subscribers register handlers that receive events asynchronously:

```go
package notify

import "autospec/internal/events"

// Subscribe registers the notification handler with the event bus.
// Must be called during application initialization.
func (h *Handler) Subscribe() func() {
    return events.Subscribe(func(e events.CommandCompleteEvent) {
        h.onCommandComplete(e)
    })
}

func (h *Handler) onCommandComplete(e events.CommandCompleteEvent) {
    if e.Success {
        h.playSound("success")
    } else {
        h.playSound("failure")
    }
    h.showNotification(e.Name, e.Success, e.Duration)
}
```

### Publishing Events

The lifecycle manager publishes events automatically:

```go
package lifecycle

import (
    "time"
    "autospec/internal/events"
)

// Run wraps command execution with event emission.
func Run(name string, fn func() error) error {
    start := time.Now()
    err := fn()
    duration := time.Since(start)

    events.Publish(events.CommandCompleteEvent{
        Name:     name,
        Success:  err == nil,
        Duration: duration,
        Error:    err,
    })

    return err
}
```

### CLI Command Integration

Commands use the lifecycle wrapper:

```go
func newSpecifyCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "specify",
        Short: "Generate feature specification",
        RunE: func(cmd *cobra.Command, args []string) error {
            return lifecycle.Run("specify", func() error {
                // Command implementation
                return orch.ExecuteSpecify(ctx)
            })
        },
    }
}
```

### Application Initialization

Set up subscriptions at startup:

```go
func main() {
    // Create notification handler
    notifHandler := notify.NewHandler(cfg.Notifications)

    // Subscribe to events (returns unsubscribe func)
    unsubscribe := notifHandler.Subscribe()
    defer unsubscribe()

    // Run CLI
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

## Testing

### Testing Event Emission

Verify events are published correctly:

```go
func TestLifecycleRunEmitsEvent(t *testing.T) {
    t.Parallel()

    var received events.CommandCompleteEvent
    var wg sync.WaitGroup
    wg.Add(1)

    unsubscribe := events.Subscribe(func(e events.CommandCompleteEvent) {
        received = e
        wg.Done()
    })
    defer unsubscribe()

    err := lifecycle.Run("test-cmd", func() error {
        return nil
    })

    wg.Wait()

    assert.NoError(t, err)
    assert.Equal(t, "test-cmd", received.Name)
    assert.True(t, received.Success)
    assert.Greater(t, received.Duration, time.Duration(0))
}
```

### Testing Subscribers

Test handlers in isolation:

```go
func TestNotificationHandlerOnCommandComplete(t *testing.T) {
    tests := map[string]struct {
        event    events.CommandCompleteEvent
        wantSound string
    }{
        "success plays success sound": {
            event:     events.CommandCompleteEvent{Success: true},
            wantSound: "success",
        },
        "failure plays failure sound": {
            event:     events.CommandCompleteEvent{Success: false},
            wantSound: "failure",
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            t.Parallel()
            mock := &mockSoundPlayer{}
            h := &Handler{player: mock}

            h.onCommandComplete(tt.event)

            assert.Equal(t, tt.wantSound, mock.lastPlayed)
        })
    }
}
```

### Race Condition Testing

Always run event tests with the race detector:

```bash
go test -race ./internal/events/...
go test -race ./internal/lifecycle/...
```

## Best Practices

### Do

- **Defer unsubscribe calls** - Prevents goroutine leaks
- **Keep handlers fast** - Each subscriber runs in its own goroutine, but slow handlers can accumulate
- **Use typed events** - Leverage generics for compile-time safety
- **Test with -race** - Event systems are prone to race conditions
- **Document event emission** - Comment when/why each event is published

### Don't

- **Don't block in handlers** - Use separate goroutines for slow operations
- **Don't panic in handlers** - Recover and log errors instead
- **Don't rely on event order** - Subscribers run concurrently
- **Don't store mutable state in events** - Events should be immutable snapshots
- **Don't create circular dependencies** - Events flow one direction

### Error Handling in Subscribers

```go
func (h *Handler) onCommandComplete(e events.CommandCompleteEvent) {
    defer func() {
        if r := recover(); r != nil {
            h.logger.Error("panic in event handler", "panic", r)
        }
    }()

    if err := h.sendNotification(e); err != nil {
        h.logger.Warn("notification failed", "error", err)
        // Don't propagate - subscriber failures shouldn't affect other subscribers
    }
}
```

## Future Extensions

The event system enables future capabilities without modifying commands:

- **Metrics collection** - Subscribe to track command durations
- **Audit logging** - Subscribe to log all command executions
- **Progress reporting** - Add progress events for long-running operations
- **Plugin system** - External subscribers via IPC/RPC
