# Arch 6: Split Validation Schema File (LOW PRIORITY)

**Location:** `internal/validation/schema.go` (761 LOC)
**Impact:** LOW - Improves code organization
**Effort:** LOW
**Dependencies:** None

## Problem Statement

All schema definitions consolidated in one 761-line file. As more artifact types are added, this will become unwieldy.

## Current Structure

```
internal/validation/
├── schema.go          # 761 LOC - ALL schemas
├── validation.go
├── artifact.go
├── tasks_yaml.go
└── ...
```

## Target Structure

```
internal/validation/
├── schemas/
│   ├── spec_schema.go      # Spec artifact schema
│   ├── plan_schema.go      # Plan artifact schema
│   ├── tasks_schema.go     # Tasks artifact schema
│   ├── research_schema.go  # Research artifact schema
│   ├── checklist_schema.go # Checklist artifact schema
│   └── registry.go         # Schema registry
├── artifact.go             # Artifact interfaces
├── validator.go            # Base validator
├── validation.go
└── tasks_yaml.go
```

## Implementation Approach

1. Create internal/validation/schemas/ directory
2. Extract spec schema to spec_schema.go
3. Extract plan schema to plan_schema.go
4. Extract tasks schema to tasks_schema.go
5. Extract research schema to research_schema.go
6. Create registry.go for schema lookup
7. Update imports throughout codebase
8. Delete original schema.go
9. Run tests

## Acceptance Criteria

- [ ] Each artifact type has dedicated schema file
- [ ] Schema registry provides lookup
- [ ] Each schema file <200 LOC
- [ ] All tests pass
- [ ] No circular imports

## Non-Functional Requirements

- No behavioral changes
- Maintain same public API
- Registry pattern for extensibility
- Keep related test files together

## Command

```bash
autospec specify "$(cat .dev/tasks/arch/arch-6-validation-schema-split.md)"
```
