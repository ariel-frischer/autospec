# Arch 3: CLI Subpackage Structure (MEDIUM PRIORITY)

**Location:** `internal/cli/` (6,463 LOC, 47 files)
**Impact:** MEDIUM - Reduces cognitive load, improves navigation
**Effort:** LOW
**Dependencies:** None, can be done independently

## Problem Statement

CLI package is monolithic despite being split into 47 files. Commands are loosely organized, making navigation difficult for contributors.

## Current Structure

```
internal/cli/
├── all.go
├── analyze.go
├── checklist.go
├── clarify.go
├── clean.go
├── commands_install.go
├── commands_list.go
├── config.go
├── constitution.go
├── doctor.go
├── execute.go
├── history.go
├── implement.go
├── init.go
├── migrate.go
├── plan.go
├── prep.go
├── root.go
├── run.go
├── specify.go
├── status.go
├── tasks.go
├── uninstall.go
├── update_agent_context.go
├── update_task.go
├── version.go
└── ... (test files)
```

## Target Structure

```
internal/cli/
├── root.go              # Root command and common flags
├── execute.go           # Shared execution helpers
├── all.go, prep.go, run.go  # Orchestration commands
├── stages/              # Stage commands
│   ├── specify.go
│   ├── plan.go
│   ├── tasks.go
│   └── implement.go
├── config/              # Configuration commands
│   ├── init.go
│   ├── config.go
│   ├── migrate.go
│   └── doctor.go
├── util/                # Utility commands
│   ├── status.go
│   ├── history.go
│   ├── version.go
│   └── clean.go
└── admin/               # Admin commands
    ├── commands_install.go
    ├── commands_list.go
    ├── completion.go
    └── uninstall.go
```

## Implementation Approach

1. Create subpackage directories
2. Move stage commands to stages/
3. Move config commands to config/
4. Move utility commands to util/
5. Move admin commands to admin/
6. Update root.go to import subpackages
7. Update all imports throughout codebase
8. Run tests to verify no breakage

## Acceptance Criteria

- [ ] stages/ contains specify, plan, tasks, implement
- [ ] config/ contains init, config, migrate, doctor
- [ ] util/ contains status, history, version, clean
- [ ] admin/ contains commands_*, completion, uninstall
- [ ] Root command imports and registers subpackages
- [ ] All tests pass
- [ ] Build succeeds

## Non-Functional Requirements

- No behavioral changes, only organizational
- Maintain CLI lifecycle wrapper pattern
- Keep test files with their commands
- Update CLAUDE.md package structure docs

## Command

```bash
autospec specify "$(cat .dev/tasks/arch/arch-3-cli-subpackages.md)"
```
