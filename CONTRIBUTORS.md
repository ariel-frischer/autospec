# Contributors Guide

This guide is for developers and contributors working on autospec.

## Quick Start for Development

```bash
# Clone the repository
git clone https://github.com/ariel-frischer/autospec.git
cd autospec

# Build the binary
make build

# Run tests
make test

# Run linters
make lint

# Install locally for testing
make install
```

## Architecture

### Core Components

The tool is built as a **cross-platform Go binary** with the following components:

#### 1. CLI Layer (`internal/cli/`)
- Cobra-based command structure
- Commands: `run`, `prep`, `specify`, `plan`, `tasks`, `implement`, `constitution`, `clarify`, `checklist`, `analyze`, `update-task`, `update-agent-context`, `artifact`, `yaml`, `status`, `history`, `doctor`, `clean`, `uninstall`, `init`, `config`, `version`
- Global flags for configuration, debugging, and spec directory override

#### 2. Workflow Orchestration (`internal/workflow/`)
- `WorkflowOrchestrator`: Executes multi-stage workflows (specify → plan → tasks)
- `Executor`: Handles stage execution with automatic retry logic
- `ClaudeExecutor`: Interfaces with Claude CLI or API
- Pre-flight dependency checking

#### 3. Validation System (`internal/validation/`)
- File existence validation (spec.yaml, plan.yaml, tasks.yaml)
- Task completion parsing and checking
- Continuation prompt generation
- Performance-optimized (<10ms per validation)

#### 4. Configuration (`internal/config/`)
- Hierarchical config loading (env vars → local config → global config → defaults)
- Supports `.autospec/config.yml` (project) and `~/.config/autospec/config.yml` (global)
- Configurable: Claude command, retry limits, specs directory, timeout

#### 5. Retry Management (`internal/retry/`)
- Persistent retry state tracking in `~/.autospec/state/retry.json`
- Atomic file writes for concurrency safety
- Per-spec:phase retry counting

### Key Packages

```
internal/
├── cli/          # Cobra commands (root, run, prep, specify, plan, tasks,
│                 # implement, constitution, clarify, checklist, analyze,
│                 # update_task, update_agent_context, clean, uninstall,
│                 # doctor, status, history, config, init, version)
├── workflow/     # Workflow orchestration, Executor, ClaudeExecutor, preflight
├── config/       # Configuration management (koanf), defaults, XDG paths
├── commands/     # Embedded slash command templates (.md files)
├── validation/   # File validation, task parsing, prompt generation
├── retry/        # Retry state management
├── spec/         # Spec detection (git branch, directory scan)
├── git/          # Git helpers (branch name, repo status)
├── agent/        # Agent context file management (update CLAUDE.md, etc.)
├── health/       # Dependency verification
├── history/      # Command execution history logging
├── lifecycle/    # Command lifecycle wrapper (timing, notifications)
├── notify/       # Desktop notification handlers
├── progress/     # Spinner display for operations
├── yaml/         # YAML parsing utilities
├── completion/   # Shell completion generation
├── errors/       # Custom error types
├── clean/        # Project cleanup functions
├── uninstall/    # System uninstall functions
└── claude/       # Claude CLI helpers
```

See [docs/architecture.md](docs/architecture.md) for detailed architecture documentation.

## Configuration

### Config File Format

Autospec uses YAML config files with hierarchical loading:

**Priority (highest to lowest):**
1. Environment variables (`AUTOSPEC_*`)
2. Local config: `.autospec/config.yml` (project-specific)
3. Global config: `~/.config/autospec/config.yml` (user-wide, XDG compliant)
4. Built-in defaults

**Example config:**
```yaml
claude_cmd: claude
claude_args:
  - -p
  - --dangerously-skip-permissions  # Enable sandbox first: /sandbox (uses bubblewrap on Linux)
  - --verbose
  - --output-format
  - stream-json
custom_claude_cmd: ""
max_retries: 3
specs_dir: ./specs
state_dir: ~/.autospec/state
skip_preflight: false
timeout: 300
implement_method: phases  # "phases", "tasks", or "single-session"
max_history_entries: 500
notifications:
  enabled: false
  type: both  # "sound", "visual", or "both"
  on_command_complete: true
  on_error: true
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `claude_cmd` | string | `"claude"` | Claude CLI command |
| `claude_args` | array | `[]` | Arguments passed to Claude CLI |
| `custom_claude_cmd` | string | `""` | Custom command with `{{PROMPT}}` placeholder |
| `max_retries` | int | `3` | Maximum retry attempts (1-10) |
| `specs_dir` | string | `"./specs"` | Directory for feature specs |
| `state_dir` | string | `"~/.autospec/state"` | State storage (retry, history) |
| `skip_preflight` | bool | `false` | Skip dependency checks |
| `timeout` | int | `0` | Command timeout in seconds (0 = no timeout) |
| `implement_method` | string | `"phases"` | Default implement mode: `phases`, `tasks`, `single-session` |
| `max_history_entries` | int | `500` | Maximum command history entries |
| `notifications.enabled` | bool | `false` | Enable desktop notifications |
| `notifications.type` | string | `"both"` | Notification type: `sound`, `visual`, `both` |

See [docs/reference.md](docs/reference.md) for complete configuration documentation.

### Environment Variables

All config options can be set via environment variables with `AUTOSPEC_` prefix:

```bash
export AUTOSPEC_CLAUDE_CMD="claude"
export AUTOSPEC_MAX_RETRIES=5
export AUTOSPEC_SPECS_DIR="./features"
export AUTOSPEC_SKIP_PREFLIGHT=true
export AUTOSPEC_TIMEOUT=600
```

### Custom Claude Command

Use `custom_claude_cmd` with a `{{PROMPT}}` placeholder for wrapper scripts:

```yaml
custom_claude_cmd: "my-wrapper {{PROMPT}}"
```

Or via environment variable:
```bash
export AUTOSPEC_CUSTOM_CLAUDE_CMD="my-wrapper {{PROMPT}}"
```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run Go tests
go test -v -race -cover ./...

# Run specific package tests
go test -v ./internal/validation/
go test -v -run TestValidateSpecFile ./internal/validation/

# Run with benchmarks
go test -bench=. ./internal/validation/
```

### Test Structure

- **Unit tests**: `*_test.go` files alongside source code
- **Table-driven tests**: Used for validation logic with multiple cases
- **Benchmark tests**: `*_bench_test.go` for performance validation
- **Integration tests**: End-to-end workflow testing

### Performance Contracts

All validation functions must meet performance contracts:

- **Validation functions**: <10ms (e.g., `ValidateSpecFile()`)
- **Retry state load/save**: <10ms
- **Overall workflow validation**: Sub-second

Performance regressions beyond 5s require immediate attention.

### Writing Tests

**Map-based table-driven test (REQUIRED pattern):**

```go
func TestValidateSpecFile(t *testing.T) {
    t.Parallel()

    tests := map[string]struct {
        specDir string
        wantErr bool
    }{
        "valid spec":   {specDir: "testdata/valid", wantErr: false},
        "missing spec": {specDir: "testdata/missing", wantErr: true},
        "empty dir":    {specDir: "testdata/empty", wantErr: true},
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            t.Parallel()
            err := ValidateSpecFile(tt.specDir)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateSpecFile() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

See [docs/go-best-practices.md](docs/go-best-practices.md) for more testing conventions.

## Exit Codes

All commands follow consistent exit code conventions:

- `0`: Success
- `1`: Validation failed (retryable)
- `2`: Retry limit exhausted
- `3`: Invalid arguments
- `4`: Missing dependencies
- `5`: Command timeout

This supports programmatic composition and CI/CD integration.

## Development Workflow

### Adding a New CLI Command

1. Create new file in `internal/cli/` (e.g., `analyze.go`)
2. Define `cobra.Command` with `Use`, `Short`, `Long`, `RunE`
3. Register command in `init()` function
4. Implement business logic by calling workflow/validation packages
5. **For workflow commands**: Use `lifecycle.RunWithHistory()` wrapper for notifications and history logging (see CLAUDE.md for required pattern)
6. Add tests in `*_test.go` file
7. Update help text and documentation

### Adding a New Validation Function

1. Add function to `internal/validation/` package
2. Follow performance contract (<10ms)
3. Return descriptive errors
4. Add unit tests with table-driven approach
5. Add benchmark test if performance-critical

### Adding a New Workflow Stage

1. Define stage constant in `internal/workflow/executor.go`
2. Add validation function in `internal/validation/`
3. Create CLI command in `internal/cli/`
4. Update `WorkflowOrchestrator` to include stage
5. Add tests for validation and execution
6. Update documentation

## Debugging

```bash
# Enable debug logging
autospec --debug run -a "feature"
autospec -d plan

# Check retry state
cat ~/.autospec/state/retry.json

# Check command history
autospec history

# Check config loading
autospec config show

# Verbose output
autospec --verbose prep "feature"

# Check dependencies
autospec doctor
```

## Linting

```bash
# Run all linters (Go + bash)
make lint

# Go linting
make lint-go
go fmt ./...
go vet ./...

# Bash script linting
make lint-bash
```

## Building

```bash
# Build for current platform
make build

# Build for all platforms (Linux/macOS/Windows)
make build-all

# Install locally
make install

# Development cycle (build + run)
make dev
```

## Contributing Guidelines

1. **Run tests before submitting**: `make test`
2. **Follow Go standards**: `make lint`
3. **Add tests for new features** (table-driven tests preferred)
4. **Add benchmarks** for performance-critical code
5. **Update documentation**: README.md, CLAUDE.md, and this file
6. **Follow constitution principles** in `.autospec/memory/constitution.yaml`
7. **Commit message format**: Use conventional commits style

### Constitution Principles

Development follows `.autospec/memory/constitution.yaml`:

1. **Validation-First**: All workflow transitions validated before proceeding
2. **Test-First Development** (NON-NEGOTIABLE): Tests written before implementation
3. **Performance Standards**: Sub-second validation (<1s); validation functions <10ms
4. **Idempotency & Retry Logic**: All operations idempotent; configurable retry limits

## Performance Optimization

Current benchmarks:
- **Workflow validation**: ~0.22s average
- **Implementation validation**: ~0.15s average
- **Hook validation**: ~0.08s average

All validations complete in under 1 second.

## Project Status

Fully featured Go CLI with:
- Complete workflow orchestration (specify → plan → tasks → implement)
- Configuration (koanf/YAML) with hierarchical loading
- Retry management with persistent state
- Validation with performance contracts (<10ms)
- Spec detection (git branch, directory scan)
- Desktop notifications (sound/visual)
- Command history tracking
- Shell completion (bash, zsh, fish, powershell)

## Credits

Created as part of the SpecKit Validation Hooks feature for Claude Code.

Built with:
- **Go 1.21+** - Cross-platform binary
- **Cobra** - CLI framework
- **Koanf** - Configuration library
- **Claude Code** - Hook system integration

## Resources

### Documentation

| File | Purpose |
|------|---------|
| [CLAUDE.md](CLAUDE.md) | Development guide and coding standards |
| [README.md](README.md) | User guide and quick start |
| [docs/architecture.md](docs/architecture.md) | System design and component diagrams |
| [docs/reference.md](docs/reference.md) | Complete CLI command reference |
| [docs/go-best-practices.md](docs/go-best-practices.md) | Go conventions and patterns |
| [docs/internals.md](docs/internals.md) | Spec detection, validation, retry system |
| [docs/troubleshooting.md](docs/troubleshooting.md) | Common issues and solutions |

- **Issue tracker**: https://github.com/ariel-frischer/autospec/issues
