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
- Commands: `workflow`, `specify`, `plan`, `tasks`, `implement`, `status`, `init`, `config`, `version`
- Global flags for configuration, debugging, and spec directory override

#### 2. Workflow Orchestration (`internal/workflow/`)
- `WorkflowOrchestrator`: Executes multi-phase workflows (specify → plan → tasks)
- `Executor`: Handles phase execution with automatic retry logic
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
├── cli/          # Cobra commands
├── workflow/     # Workflow orchestration & execution
├── config/       # Configuration management (koanf)
├── validation/   # Validation functions
├── retry/        # Retry state management
├── spec/         # Spec detection
└── git/          # Git repository helpers
```

See [CLAUDE.md](CLAUDE.md) for detailed architecture documentation.

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
  - --dangerously-skip-permissions
  - --verbose
  - --output-format
  - stream-json
custom_claude_cmd: ""
max_retries: 3
specs_dir: ./specs
state_dir: ~/.autospec/state
skip_preflight: false
timeout: 300
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `claude_cmd` | string | `"claude"` | Claude CLI command |
| `claude_args` | array | `[]` | Arguments passed to Claude CLI |
| `custom_claude_cmd` | string | `""` | Custom command with `{{PROMPT}}` placeholder |
| `max_retries` | int | `3` | Maximum retry attempts (1-10) |
| `specs_dir` | string | `"./specs"` | Directory for feature specs |
| `state_dir` | string | `"~/.autospec/state"` | Retry state storage |
| `skip_preflight` | bool | `false` | Skip dependency checks |
| `timeout` | int | `300` | Command timeout in seconds (0 = no timeout) |

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

```json
{
  "custom_claude_cmd": "my-wrapper {{PROMPT}}"
}
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

**Example table-driven test:**

```go
func TestValidateSpecFile(t *testing.T) {
    tests := []struct {
        name    string
        specDir string
        wantErr bool
    }{
        {"valid spec", "testdata/valid", false},
        {"missing spec", "testdata/missing", true},
        {"empty dir", "testdata/empty", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateSpecFile(tt.specDir)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateSpecFile() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Exit Codes

All commands follow consistent exit code conventions:

- `0`: Success
- `1`: Validation failed (retryable)
- `2`: Retry limit exhausted
- `3`: Invalid arguments
- `4`: Missing dependencies

This supports programmatic composition and CI/CD integration.

## Development Workflow

### Adding a New CLI Command

1. Create new file in `internal/cli/` (e.g., `analyze.go`)
2. Define `cobra.Command` with `Use`, `Short`, `Long`, `RunE`
3. Register command in `init()` function
4. Implement business logic by calling workflow/validation packages
5. Add tests in `*_test.go` file
6. Update help text and documentation

### Adding a New Validation Function

1. Add function to `internal/validation/` package
2. Follow performance contract (<10ms)
3. Return descriptive errors
4. Add unit tests with table-driven approach
5. Add benchmark test if performance-critical

### Adding a New Workflow Phase

1. Define phase constant in `internal/workflow/executor.go`
2. Add validation function in `internal/validation/`
3. Create CLI command in `internal/cli/`
4. Update `WorkflowOrchestrator` to include phase
5. Add tests for validation and execution
6. Update documentation

## Debugging

```bash
# Enable debug logging
autospec --debug workflow "feature"
autospec -d plan

# Check retry state
cat ~/.autospec/state/retry.json

# Check config loading
autospec config show

# Verbose output
autospec --verbose workflow "feature"
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
6. **Follow constitution principles** in `.autospec/memory/constitution.md`
7. **Commit message format**: Use conventional commits style

### Constitution Principles

Development follows `.autospec/memory/constitution.md`:

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

Fully migrated Go binary with CLI commands, configuration (koanf/YAML), retry management, validation, workflow orchestration, and spec detection.

## Credits

Created as part of the SpecKit Validation Hooks feature for Claude Code.

Built with:
- **Go 1.21+** - Cross-platform binary
- **Cobra** - CLI framework
- **Koanf** - Configuration library
- **Claude Code** - Hook system integration

## Resources

- **Development guide**: [CLAUDE.md](CLAUDE.md)
- **User guide**: [README.md](README.md)
- **Prerequisites**: [PREREQUISITES.md](PREREQUISITES.md)
- **Issue tracker**: https://github.com/ariel-frischer/autospec/issues
