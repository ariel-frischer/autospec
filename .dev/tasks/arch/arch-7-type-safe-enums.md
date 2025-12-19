# Arch 7: Type-Safe Enums (LOW PRIORITY)

> **Status: SKIP**
>
> **Reason:** False premise. The codebase already uses typed constants correctly—all 21 Stage comparisons use `case StageSpecify:` not string literals like `== "specify"`. The claimed typo problem (`stage == "specfy"` compiling) doesn't exist because no code uses raw strings. Converting to int-based enums would BREAK YAML/JSON serialization (would serialize as 0,1,2 instead of "specify","plan") and require custom UnmarshalYAML/JSON methods. The current `type Stage string` pattern already catches typos at compile time when using constants.
>
> **Reviewed:** 2025-12-18

**Location:** Multiple packages (workflow, validation)
**Impact:** LOW - Improves type safety
**Effort:** LOW
**Dependencies:** None

## Problem Statement

String-based enums are prone to typos and lack compiler validation:

```go
type Stage string

const (
    StageSpecify   Stage = "specify"
    StagePlan      Stage = "plan"
    // No compiler check for typos like "specfy"
)
```

## Current Pattern

```go
// String-based, no validation
type Stage string

const (
    StageSpecify   Stage = "specify"
    StagePlan      Stage = "plan"
    StageTasks     Stage = "tasks"
    StageImplement Stage = "implement"
)

// Usage - typos not caught
if stage == "specfy" { // BUG: typo compiles
    // ...
}
```

## Target Pattern

```go
// Int-based with String() method
type Stage int

const (
    StageUnknown Stage = iota
    StageSpecify
    StagePlan
    StageTasks
    StageImplement
)

var stageNames = [...]string{
    StageUnknown:   "unknown",
    StageSpecify:   "specify",
    StagePlan:      "plan",
    StageTasks:     "tasks",
    StageImplement: "implement",
}

func (s Stage) String() string {
    if int(s) < len(stageNames) {
        return stageNames[s]
    }
    return "unknown"
}

func ParseStage(s string) (Stage, error) {
    for i, name := range stageNames {
        if name == s {
            return Stage(i), nil
        }
    }
    return StageUnknown, fmt.Errorf("unknown stage: %s", s)
}

// Usage - typos caught at compile time
if stage == StageSpecfy { // Compiler error: undefined
    // ...
}
```

## Implementation Approach

1. Define int-based Stage type with iota
2. Add String() method for display
3. Add ParseStage() for string parsing
4. Update all stage comparisons
5. Define similar pattern for ImplementMethod
6. Update config parsing to use Parse*
7. Run tests

## Acceptance Criteria

- [ ] Stage type uses int with iota
- [ ] String() method for display
- [ ] ParseStage() with error handling
- [ ] ImplementMethod type converted similarly
- [ ] All string comparisons updated
- [ ] All tests pass

## Non-Functional Requirements

- Backward compatible YAML parsing
- JSON marshaling unchanged
- Clear error messages from Parse*
- Document in go-best-practices.md

## Command

```bash
autospec specify "$(cat .dev/tasks/arch/arch-7-type-safe-enums.md)"
```
