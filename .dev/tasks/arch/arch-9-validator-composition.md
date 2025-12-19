# Arch 9: Validator Composition Pattern (SKIPPED)

**Status:** SKIPPED
**Decision Date:** 2025-12-18

### Skip Reason

The problem as described does not match the actual codebase:

1. **`baseValidator` is minimal** — only holds `artifactType` and provides `Type()` (5 lines)
2. **Validators have real logic** — `SpecValidator.Validate()` has ~50 lines of actual validation, not pass-through wrappers
3. **Helpers are already standalone functions** — `parseYAMLFile()`, `findNode()`, `validateRequiredField()` are package-level functions, not methods on baseValidator

The proposed `ValidatorEngine` abstraction would add complexity without solving a real problem. Current pattern is idiomatic Go: minimal embedding for shared type + standalone helper functions.

---

**Location:** `internal/validation/`
**Impact:** LOW - More idiomatic Go
**Effort:** LOW
**Dependencies:** Arch-6 (schema split) helpful but not required

## Problem Statement

Validator implementations are anemic - they embed baseValidator and just call through:

```go
type SpecValidator struct {
    *baseValidator
}
// Methods just call baseValidator
```

This is pseudo-inheritance, not idiomatic Go.

## Current Pattern

```go
// Base with all logic
type baseValidator struct {
    schema     Schema
    fileReader FileReader
}

func (v *baseValidator) Validate(path string) error { ... }
func (v *baseValidator) GetPrompt() string { ... }

// Anemic wrapper
type SpecValidator struct {
    *baseValidator
}

// Factory creates embedded struct
func NewSpecValidator() *SpecValidator {
    return &SpecValidator{
        baseValidator: &baseValidator{
            schema: specSchema,
        },
    }
}
```

## Target Pattern

```go
// Interface-based
type Validator interface {
    Validate(path string) error
    Schema() Schema
}

// Composition over inheritance
type SpecValidator struct {
    schema    Schema
    engine    ValidatorEngine
}

func NewSpecValidator(engine ValidatorEngine) *SpecValidator {
    return &SpecValidator{
        schema: specSchema,
        engine: engine,
    }
}

func (v *SpecValidator) Validate(path string) error {
    return v.engine.ValidateAgainstSchema(path, v.schema)
}

func (v *SpecValidator) Schema() Schema {
    return v.schema
}

// Shared engine for validation logic
type ValidatorEngine struct {
    fileReader FileReader
}

func (e *ValidatorEngine) ValidateAgainstSchema(path string, schema Schema) error {
    // Shared validation logic
}
```

## Implementation Approach

1. Define Validator interface
2. Create ValidatorEngine for shared logic
3. Refactor SpecValidator to use composition
4. Refactor PlanValidator similarly
5. Refactor TasksValidator similarly
6. Update factory to inject engine
7. Run tests

## Acceptance Criteria

- [ ] Validator interface defined
- [ ] ValidatorEngine contains shared logic
- [ ] SpecValidator uses composition
- [ ] PlanValidator uses composition
- [ ] TasksValidator uses composition
- [ ] No embedded struct inheritance
- [ ] All tests pass

## Non-Functional Requirements

- Clear separation of concerns
- Engine injectable for testing
- Validators hold only schema reference
- Factory pattern maintained

## Command

```bash
autospec specify "$(cat .dev/tasks/arch/arch-9-validator-composition.md)"
```
