# Arch 8: Structured Logging (LOW PRIORITY)

> **Status: SKIP**
>
> **Reason:** Over-engineered for a CLI tool. The 5 debugLog methods (4 lines each, 43 call sites) use `fmt.Printf`—which is optimal for CLI debug output read by humans in terminals. Structured logging (slog) solves server observability problems: log aggregation, machine parsing, production monitoring. None apply here. Users run commands interactively and grep terminal output. Adding slog abstraction would increase complexity for no user benefit. Minor DRY improvement (consolidating debugLog into shared helper) can be done opportunistically during other refactoring if desired.
>
> **Reviewed:** 2025-12-18

**Location:** Multiple packages
**Impact:** LOW - Improves observability
**Effort:** LOW
**Dependencies:** None

## Problem Statement

Ad-hoc logging scattered throughout codebase:
- Multiple types have `debugLog()` methods
- Inconsistent log levels
- No structured fields
- Hard to filter logs

## Current Pattern

```go
// Scattered debug methods
func (o *WorkflowOrchestrator) debugLog(format string, args ...interface{}) {
    if o.Debug {
        fmt.Printf("[DEBUG] "+format+"\n", args...)
    }
}

// Used inconsistently
o.debugLog("executing stage: %s", stage)
e.debugLog("retrying command, attempt %d", attempt)
```

## Target Pattern

```go
import "log/slog"

// Centralized logger
var logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

func SetDebugLevel() {
    logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))
}

// Structured logging
logger.Debug("executing stage",
    slog.String("stage", stage),
    slog.String("spec", specName),
)

logger.Info("retrying command",
    slog.Int("attempt", attempt),
    slog.Int("maxRetries", maxRetries),
    slog.Duration("timeout", timeout),
)
```

## Implementation Approach

1. Create internal/logging/logger.go
2. Initialize slog with configurable level
3. Replace debugLog() methods with slog calls
4. Add structured fields to log calls
5. Configure level from --debug flag
6. Update all packages to use centralized logger
7. Run tests

## Acceptance Criteria

- [ ] Centralized logger in internal/logging/
- [ ] slog.Debug for debug messages
- [ ] slog.Info for informational messages
- [ ] slog.Error for error messages
- [ ] Structured fields for all log calls
- [ ] --debug flag sets level
- [ ] No more debugLog() methods

## Non-Functional Requirements

- Use stdlib log/slog
- JSON output option for machine parsing
- Field names consistent across packages
- Don't log sensitive data

## Command

```bash
autospec specify "$(cat .dev/tasks/arch/arch-8-structured-logging.md)"
```
