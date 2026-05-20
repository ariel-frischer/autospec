# AGENTS.md

Guidance for AI coding agents working with this repository.

## Prerequisites

- **Go 1.25+**: Check with `go version`
- **AI coding agent CLI**: Authenticated and configured
- **Make, golangci-lint**: For build/lint (`make lint`)

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

## Post-Change Polish (REQUIRED)

After significant code changes, automatically invoke the repo-local `polish` skill before final handoff. Use it to update changelog/docs and run the required validation targets. Do not wait for an explicit `$polish` request unless the change is trivial or docs-only.

## Core Workflow

### Stage Dependencies (MUST follow this order)

```
constitution → specify → plan → tasks → implement
     ↓            ↓        ↓       ↓
constitution.yaml spec.yaml plan.yaml tasks.yaml
```

| Stage | Requires | Produces |
|-------|----------|----------|
| `constitution` | — | `.autospec/constitution.yaml` |
| `specify` | constitution | `specs/NNN-feature/spec.yaml` |
| `plan` | spec.yaml | `plan.yaml` |
| `tasks` | plan.yaml | `tasks.yaml` |
| `implement` | tasks.yaml | code changes |

**Constitution is REQUIRED before any workflow stage.**

### What `autospec init` Does

1. Creates config (`~/.config/autospec/config.yml` or `.autospec/config.yml`)
2. Installs agent-native prompts (Claude skills in `.claude/skills/`, shared Codex/OpenCode skills in `.agents/skills/`)
3. Configures agent permissions and sandbox settings
4. Prompts for constitution creation (one-time per project)

### First-Time Project Setup

```bash
autospec init              # Interactive setup (config + agent + constitution)
autospec doctor            # Verify dependencies
autospec prep "feature"    # specify → plan → tasks
autospec implement         # Execute tasks
```

## Documentation

**Review relevant docs before implementation:**

### Internal (Developer-Focused)

| File | Purpose |
|------|---------|
| `docs/internal/architecture.md` | System design, component diagrams, execution flows |
| `docs/internal/go-best-practices.md` | Go conventions, naming, error handling patterns |
| `docs/internal/internals.md` | Spec detection, validation, retry system, phase context |
| `docs/internal/YAML-STRUCTURED-OUTPUT.md` | YAML artifact schemas and slash commands |
| `docs/internal/testing-mocks.md` | Mock testing infrastructure for workflows without real API calls |
| `docs/internal/codex-manual-testing.md` | Manual Codex smoke and regression testing plan |
| `docs/internal/events.md` | Event-driven architecture using kelindar/event |
| `docs/internal/risks.md` | Risk documentation in plan.yaml |
| `docs/internal/cclean.md` | claude-clean tool for transforming streaming JSON output |
| `docs/internal/changelog-yaml.md` | Developer workflow for YAML-first changelog and release notes |

### Public (User-Focused)

| File | Purpose |
|------|---------|
| `docs/public/reference.md` | Complete CLI command reference with all flags |
| `docs/public/quickstart.md` | Getting started guide for first workflow |
| `docs/public/overview.md` | High-level introduction to autospec |
| `docs/public/TIMEOUT.md` | Timeout configuration and behavior |
| `docs/public/checklists.md` | Checklist generation, validation, and implementation gating |
| `docs/public/SHELL-COMPLETION.md` | Shell completion implementation |
| `docs/public/troubleshooting.md` | Common issues and solutions |
| `docs/public/claude-settings.md` | Claude Code settings and sandboxing configuration |
| `docs/public/opencode-settings.md` | OpenCode configuration, permissions, and command patterns |
| `docs/public/agents.md` | CLI agent configuration (Claude and OpenCode supported) |
| `docs/public/worktree.md` | Git worktree management for parallel agent execution |
| `docs/public/parallel-execution.md` | Sequential workflows and manual multi-worktree parallel workflows |
| `docs/public/self-update.md` | Version checking and self-update functionality |
| `docs/public/faq.md` | Frequently asked questions |

### Research

| File | Purpose |
|------|---------|
| `docs/research/claude-opus-4.5-context-performance.md` | Claude Opus 4.5 performance in extended sessions |

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
- `internal/workflow/`: Workflow orchestration and agent execution
- `internal/config/`: Hierarchical config (env > project > user > defaults)
- `internal/validation/`: Artifact validation (<10ms performance contract)
- `internal/retry/`: Persistent retry state
- `internal/spec/`: Spec detection from git branch or recent directory
- `internal/agent/`: Agent abstraction (Claude, Gemini, Cline, etc.)
- `internal/cliagent/`: CLI agent integration and Configurator interface
- `internal/worktree/`: Git worktree management logic

### Configuration

Priority: Environment (`AUTOSPEC_*`) > `.autospec/config.yml` > `~/.config/autospec/config.yml` > defaults

Key settings: `agent_preset`, `max_retries`, `specs_dir`, `timeout`, `implement_method`

> **Note**: The legacy `claude_cmd` and `claude_args` fields are deprecated. Use `agent_preset` instead. See `docs/public/agents.md`.

## Constitution Principles

From `.autospec/constitution.yaml`:

1. **Validation-First**: All workflow transitions validated before proceeding
2. **Test-First Development** (NON-NEGOTIABLE): Tests written before implementation
3. **Performance Standards**: Validation functions <10ms
4. **Idempotency**: All operations idempotent; configurable retry limits
5. **Command Template Independence** (NON-NEGOTIABLE): `internal/commands/*.md` must be project-agnostic—no MCP tools, no Claude Code tools, no autospec-internal paths

## Config Changes (REQUIRED)

When adding, changing, or removing config fields, update **ALL** locations:
1. `internal/config/config.go` - Add field to `Configuration` with `koanf` tag
2. `internal/config/schema.go` - Add to `KnownKeys` map
3. `internal/config/defaults.go` - Add to YAML template AND `GetDefaults()` function
4. `internal/config/validate.go` - Add validation if needed
5. Relevant public docs - Update user-facing defaults and examples

## Coding Standards

### Imports

Group in order with blank lines: 1) Standard library, 2) External packages, 3) Internal packages

```go
import (
    "fmt"
    "os"

    "github.com/spf13/cobra"

    "github.com/ariel-frischer/autospec/internal/config"
)
```

### Naming Conventions

```go
package validation     // Packages: short, lowercase, no underscores
func ValidateSpecFile  // Exported: CamelCase
func parseTaskLine     // Unexported: camelCase
type Config struct{}   // Avoid stutter (not config.ConfigStruct)
type Validator interface { Validate() error }  // Interfaces: -er suffix
```

### Error Handling (CRITICAL)

**Always wrap errors with context** - never bare `return err`:
```go
return fmt.Errorf("loading config: %w", err)  // GOOD
```

Use structured errors from `internal/errors/` for user-facing CLI errors:
```go
return errors.NewValidationError("spec.yaml", "missing required field: feature")
```

Exceptions: Pass-through helpers, test code.

### Function Length

Keep functions under 40 lines. Extract helpers for pre-validation, core logic, post-processing, and output formatting.

### Function Design

- **Max 40 lines** per function. Extract helpers for complex logic.
- **Accept interfaces, return concrete types**
- **Context as first parameter** when needed for cancellation

### Map-Based Table Tests (REQUIRED)

```go
func TestValidateSpecFile(t *testing.T) {
    tests := map[string]struct {
        input   string
        wantErr bool
    }{
        "valid input": {input: "foo"},
        "empty input": {input: "", wantErr: true},
    }
    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            t.Parallel()
            // test logic
        })
    }
}
```

### Deterministic Agent/Workflow Tests

Default unit, package integration, and workflow tests MUST NOT call real agent
CLIs or external AI APIs. Use `testutil.MockExecutor` for agent execution,
`testutil.E2EEnv` for compiled CLI tests, and `testutil.GitIsolation` or
`t.TempDir()` for git/repository state.

### CLI Command Lifecycle Wrapper (REQUIRED)

Workflow commands MUST use `lifecycle.RunWithHistory()` for notifications, timing, and history:

```go
notifHandler := notify.NewHandler(cfg.Notifications)
historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)
return lifecycle.RunWithHistory(notifHandler, historyLogger, "cmd-name", specName, func() error {
    return orch.ExecuteXxx(...)
})
```

For context-aware commands: `lifecycle.RunWithHistoryContext(cmd.Context(), ...)`.

Required for: `specify`, `plan`, `tasks`, `clarify`, `analyze`, `checklist`, `constitution`, `prep`, `run`, `implement`, `all`.

Regression test: `TestAllCommandsHaveNotificationSupport` in `internal/cli/specify_test.go`.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Validation failed (retryable) |
| 2 | Retry limit exhausted |
| 3 | Invalid arguments |
| 4 | Missing dependencies |
| 5 | Timeout |

## Performance Contracts

- Validation functions: <10ms
- Retry state load/save: <10ms
- Config loading: <100ms

## Spec Generation (MUST)

When generating `spec.yaml`, ALWAYS include these as NFRs (category: `code_quality`):
- Functions under 40 lines
- Errors wrapped with context (`fmt.Errorf("doing X: %w", err)`)
- Map-based table tests (`map[string]struct`)
- Accept interfaces, return concrete types

Final FR MUST require: `make test && make fmt && make lint && make build` all exit 0.

## Task Generation (MUST)

When generating `tasks.yaml`, the **final tasks** MUST include:

1. **Manual testing plan**: Create `.dev/tasks/<spec-name>.md` with a plan for manually testing all changes. Include a "Report Summaries" section to be filled in once manual testing is complete. Do NOT execute the tests—just map out core manual testing steps for that spec.

2. **Changelog update**: Add 1-3 user-facing bullets to `internal/changelog/changelog.yaml`, then run `make changelog-sync` to regenerate `CHANGELOG.md`.

This ensures all features have documented test plans and are visible to users.

## Changelog Workflow (YAML-First)

Edit `internal/changelog/changelog.yaml` directly, then run `make changelog-sync` to regenerate `CHANGELOG.md`. Never edit `CHANGELOG.md` directly—it is auto-generated from the YAML source.

Public changelog entries must describe user-visible behavior: CLI changes, config changes, compatibility changes, docs users rely on, security fixes, or bug fixes users can observe. Maintainer-only process, governance, prompt hygiene, test strategy, architecture-boundary, or release-workflow notes belong under the version's `internal:` tier in `internal/changelog/changelog.yaml`, not in public `added`/`changed`/`fixed` release notes.

## Git Commits in Sandbox Mode

```bash
# BAD - heredocs fail in sandbox mode
git commit -m "$(cat <<'EOF'
commit message
EOF
)"

# GOOD - use regular quoted string with newlines
git commit -m "feat(scope): description

Body text here.
"
```

## Pre-Commit Checklist

```bash
make fmt && make lint && make test && make build
```

All must pass before committing. Run `make test-v` for verbose output on failures.

## Common Gotchas

- **Branch naming**: Must match `^\d{3}-.+$` (e.g., `001-feature`) for spec auto-detection
- **Sandbox heredocs**: Use quoted strings, not heredocs, for git commits in sandbox mode
- **Constitution required**: All workflow stages fail without `.autospec/constitution.yaml`

## Key Files

- `~/.config/autospec/config.yml`: User config
- `.autospec/config.yml`: Project config
- `.autospec/constitution.yaml`: Project principles (REQUIRED)
- `~/.autospec/state/retry.json`: Retry state
- `specs/*/`: Feature specs (spec.yaml, plan.yaml, tasks.yaml)
