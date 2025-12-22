# CLAUDE.md

Guidance for Claude Code when working with this repository.

## Commands

```bash
# Build & Dev
make build          # Build for current platform
make test           # Run all tests (quiet, shows failures only)
make test-v         # Run all tests (verbose, for debugging)
make fmt            # Format Go code (run before committing)
make lint           # Run all linters

# Single test
go test -run TestName ./internal/package/

# CLI usage (run `autospec --help` for full reference)
autospec run -a "feature description"    # All stages: specify → plan → tasks → implement
autospec prep "feature description"      # Planning only: specify → plan → tasks
autospec implement --phases              # Each phase in separate session
autospec implement --tasks               # Each task in separate session
autospec st                              # Show status and task progress
autospec doctor                          # Check dependencies
```

## Documentation

**Review relevant docs before implementation:**

| File | Purpose |
|------|---------|
| `docs/architecture.md` | System design, component diagrams, execution flows |
| `docs/go-best-practices.md` | Go conventions, naming, error handling patterns |
| `docs/reference.md` | Complete CLI command reference with all flags |
| `docs/internals.md` | Spec detection, validation, retry system, phase context |
| `docs/TIMEOUT.md` | Timeout configuration and behavior |
| `docs/YAML-STRUCTURED-OUTPUT.md` | YAML artifact schemas and slash commands |
| `docs/checklists.md` | Checklist generation, validation, and implementation gating |
| `docs/risks.md` | Risk documentation in plan.yaml |
| `docs/SHELL-COMPLETION.md` | Shell completion implementation |
| `docs/troubleshooting.md` | Common issues and solutions |
| `docs/claude-settings.md` | Claude Code settings and sandboxing configuration |
| `docs/agents.md` | CLI agent configuration and migration guide |

## Architecture Overview

autospec is a Go CLI that orchestrates SpecKit workflows. Key distinction:
- **Stage**: High-level workflow step (specify, plan, tasks, implement)
- **Phase**: Task grouping within implementation (Phase 1: Setup, Phase 2: Core, etc.)

### Package Structure

- `cmd/autospec/main.go`: Entry point
- `internal/cli/`: Cobra commands (root + orchestration)
  - `internal/cli/stages/`: Stage commands (specify, plan, tasks, implement)
  - `internal/cli/config/`: Configuration commands (init, config, migrate, doctor)
  - `internal/cli/util/`: Utility commands (status, history, version, clean, view)
  - `internal/cli/admin/`: Admin commands (commands, completion, uninstall)
  - `internal/cli/worktree/`: Worktree management commands (create, list, remove, prune)
  - `internal/cli/shared/`: Shared types and constants
- `internal/workflow/`: Workflow orchestration and Claude execution
- `internal/config/`: Hierarchical config (env > project > user > defaults)
- `internal/validation/`: Artifact validation (<10ms performance contract)
- `internal/retry/`: Persistent retry state
- `internal/spec/`: Spec detection from git branch or recent directory
- `internal/agent/`: Agent abstraction (Claude, Gemini, Cline, etc.)
- `internal/cliagent/`: CLI agent integration and Configurator interface
- `internal/worktree/`: Git worktree management logic
- `internal/dag/`: DAG support for parallel task execution

### Configuration

Priority: Environment (`AUTOSPEC_*`) > `.autospec/config.yml` > `~/.config/autospec/config.yml` > defaults

Key settings: `agent_preset`, `max_retries`, `specs_dir`, `timeout`, `implement_method`

> **Note**: The legacy `claude_cmd` and `claude_args` fields are deprecated. Use `agent_preset` instead. See `docs/agents.md`.

## Constitution Principles

From `.autospec/memory/constitution.yaml`:

1. **Validation-First**: All workflow transitions validated before proceeding
2. **Test-First Development** (NON-NEGOTIABLE): Tests written before implementation
3. **Performance Standards**: Validation functions <10ms
4. **Idempotency**: All operations idempotent; configurable retry limits
5. **Command Template Independence** (NON-NEGOTIABLE): `internal/commands/*.md` must be project-agnostic—no MCP tools, no Claude Code tools, no autospec-internal paths

## Coding Standards

### Error Handling (CRITICAL)

**Always wrap errors with context** - never use bare `return err`:

```go
// BAD
if err != nil {
    return err
}

// GOOD
if err != nil {
    return fmt.Errorf("loading config file: %w", err)
}
```

Exceptions: Helper functions explicitly designed to pass through errors unchanged, test code.

### Function Length

Keep functions under 40 lines. Extract helpers for pre-validation, core logic, post-processing, and output formatting.

### Map-Based Table Tests (REQUIRED)

```go
// GOOD - map-based pattern
tests := map[string]struct {
    input   string
    want    string
    wantErr bool
}{
    "valid input": {input: "foo", want: "bar"},
    "empty input": {input: "", wantErr: true},
}
for name, tt := range tests {
    t.Run(name, func(t *testing.T) { ... })
}
```

### CLI Command Lifecycle Wrapper (REQUIRED)

All workflow CLI commands in `internal/cli/` MUST use the lifecycle wrapper for notifications and history logging. This ensures users receive completion notifications (sound/visual) when commands finish, with automatic timing, panic recovery, and command history tracking.

Required pattern using `lifecycle.RunWithHistory()`:

```go
import (
    "github.com/ariel-frischer/autospec/internal/history"
    "github.com/ariel-frischer/autospec/internal/lifecycle"
    "github.com/ariel-frischer/autospec/internal/notify"
)

// Create notification handler and history logger
notifHandler := notify.NewHandler(cfg.Notifications)
historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)

// Wrap command execution with lifecycle for timing, notification, and history
return lifecycle.RunWithHistory(notifHandler, historyLogger, "command-name", specName, func() error {
    // Execute the command logic
    return orch.ExecuteXxx(...)
})
```

The lifecycle wrapper provides:
- Automatic timing (start time, duration calculation)
- Notification dispatch (`OnCommandComplete` with correct parameters)
- Two-phase history logging (WriteStart → UpdateComplete)
- Crash/interrupt visibility (entries remain "running" if process terminates abnormally)
- Panic recovery for notification handlers
- Nil handler safety (no-op if handler or logger is nil)

For context-aware commands (cancellation support), use `lifecycle.RunWithHistoryContext()`:

```go
return lifecycle.RunWithHistoryContext(cmd.Context(), notifHandler, historyLogger, "command-name", specName, func(_ context.Context) error {
    return orch.ExecuteXxx(...)
})
```

Commands requiring this pattern: `specify`, `plan`, `tasks`, `clarify`, `analyze`, `checklist`, `constitution`, `prep`, `run`, `implement`, `all`.

Regression test: `TestAllCommandsHaveNotificationSupport` in `internal/cli/specify_test.go` verifies all commands use the lifecycle wrapper (`RunWithHistory` or `RunWithHistoryContext`).

## Spec Generation (MUST)

When generating `spec.yaml` files, ALWAYS include these Go coding standards as non-functional requirements:

```yaml
non_functional:
  - id: "NFR-XXX"
    category: "code_quality"
    description: "All functions must be under 40 lines; extract helpers for complex logic"
    measurable_target: "No function exceeds 40 lines excluding comments"
  - id: "NFR-XXX"
    category: "code_quality"
    description: "All errors must be wrapped with context using fmt.Errorf(\"doing X: %w\", err)"
    measurable_target: "Zero bare 'return err' statements in new code"
  - id: "NFR-XXX"
    category: "code_quality"
    description: "Tests must use map-based table-driven pattern"
    measurable_target: "All new test functions use map[string]struct pattern"
  - id: "NFR-XXX"
    category: "code_quality"
    description: "Accept interfaces, return concrete types"
    measurable_target: "Function signatures follow interface-in, concrete-out pattern where applicable"
```

Also ALWAYS include this functional requirement as the final FR:

```yaml
functional:
  - id: "FR-XXX"
    description: "MUST pass all quality gates: make test, make fmt, make lint, and make build"
    testable: true
    acceptance_criteria: "All commands exit 0; no test failures, format changes, lint errors, or build failures"
```

These are NON-NEGOTIABLE for any Go implementation in this project.

## Exit Codes

- `0`: Success
- `1`: Validation failed (retryable)
- `2`: Retry limit exhausted
- `3`: Invalid arguments
- `4`: Missing dependencies
- `5`: Timeout

## Key Files

- `~/.config/autospec/config.yml`: User config
- `.autospec/config.yml`: Project config
- `~/.autospec/state/retry.json`: Retry state
- `specs/*/`: Feature specs (spec.yaml, plan.yaml, tasks.yaml)
