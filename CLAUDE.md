# CLAUDE.md

Guidance for Claude Code when working with this repository.

## Commands

```bash
# Build & Dev
make build          # Build for current platform
make test           # Run all tests
make fmt            # Format Go code (run before committing)
make lint           # Run all linters

# Single test
go test -v -run TestName ./internal/package/

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
| `docs/TIMEOUT.md` | Timeout configuration and behavior |
| `docs/YAML-STRUCTURED-OUTPUT.md` | YAML artifact schemas and slash commands |
| `docs/SHELL-COMPLETION.md` | Shell completion implementation |
| `docs/troubleshooting.md` | Common issues and solutions |

## Architecture Overview

autospec is a Go CLI that orchestrates SpecKit workflows. Key distinction:
- **Stage**: High-level workflow step (specify, plan, tasks, implement)
- **Phase**: Task grouping within implementation (Phase 1: Setup, Phase 2: Core, etc.)

### Package Structure

- `cmd/autospec/main.go`: Entry point
- `internal/cli/`: Cobra commands
- `internal/workflow/`: Workflow orchestration and Claude execution
- `internal/config/`: Hierarchical config (env > project > user > defaults)
- `internal/validation/`: Artifact validation (<10ms performance contract)
- `internal/retry/`: Persistent retry state
- `internal/spec/`: Spec detection from git branch or recent directory

### Configuration

Priority: Environment (`AUTOSPEC_*`) > `.autospec/config.yml` > `~/.config/autospec/config.yml` > defaults

Key settings: `claude_cmd`, `max_retries`, `specs_dir`, `timeout`, `implement_method`

## Constitution Principles

From `.autospec/memory/constitution.yaml`:

1. **Validation-First**: All workflow transitions validated before proceeding
2. **Test-First Development** (NON-NEGOTIABLE): Tests written before implementation
3. **Performance Standards**: Validation functions <10ms
4. **Idempotency**: All operations idempotent; configurable retry limits

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
