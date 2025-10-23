# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Build and Development
```bash
# Build for current platform
make build

# Build for all platforms (Linux/macOS/Windows)
make build-all

# Install binary to /usr/local/bin
make install

# Run the binary
make run

# Quick development cycle
make dev
```

### Testing
```bash
# Run all tests (Go + bats)
make test

# Run Go tests only
make test-go
go test -v -race -cover ./...

# Run single Go test
go test -v -run TestValidateSpecFile ./internal/validation/

# Run bats tests only (legacy bash scripts)
make test-bash
./tests/run-all-tests.sh

# Run specific bats test suite
bats tests/lib/validation-lib.bats
bats tests/integration.bats
```

### Linting
```bash
# Run all linters (Go + bash)
make lint

# Go linting
make lint-go
go fmt ./...
go vet ./...

# Bash script linting
make lint-bash
shellcheck scripts/**/*.sh
```

### Using the CLI
```bash
# Run complete workflow: specify → plan → tasks → implement
autospec full "Add user authentication feature"

# Run workflow without implementation: specify → plan → tasks
autospec workflow "Add user authentication feature"

# Run individual phases
autospec specify "feature description"
autospec plan
autospec plan "Focus on security best practices"           # With prompt guidance
autospec tasks
autospec tasks "Break into small incremental steps"        # With prompt guidance
autospec implement
autospec implement "Focus on documentation tasks"          # With prompt guidance
autospec implement 003-my-feature                          # Specific spec
autospec implement 003-my-feature "Complete tests"         # Spec + prompt

# Check dependencies
autospec doctor

# Check status
autospec status

# Initialize config
autospec init

# Show version
autospec version
```

## Architecture Overview

Auto Claude SpecKit is a **cross-platform Go binary** that automates SpecKit workflow validation and orchestration. The architecture consists of several key layers:

### 1. CLI Layer (`internal/cli/`)

Cobra-based command structure providing user-facing commands:
- **root.go**: Root command with global flags (`--config`, `--specs-dir`, `--debug`, etc.)
- **full.go**: Orchestrates complete specify → plan → tasks → implement workflow
- **workflow.go**: Orchestrates specify → plan → tasks workflow (no implementation)
- **specify.go, plan.go, tasks.go, implement.go**: Individual phase commands
- **doctor.go**: Health check command for dependencies
- **status.go**: Reports current spec progress
- **config.go**: Configuration management commands
- **init.go**: Initializes configuration files
- **version.go**: Version information

### 2. Workflow Orchestration (`internal/workflow/`)

Manages the complete SpecKit workflow lifecycle:
- **workflow.go**: `WorkflowOrchestrator` executes multi-phase workflows with validation
  - `RunFullWorkflow()`: Complete workflow including implementation (specify → plan → tasks → implement)
  - `RunCompleteWorkflow()`: Planning workflow only (specify → plan → tasks)
  - `ExecuteImplement()`: Implementation phase only
- **executor.go**: `Executor` handles phase execution with retry logic
- **claude.go**: `ClaudeExecutor` interfaces with Claude CLI or API
- **preflight.go**: Pre-flight checks for dependencies (claude, specify CLIs)

Key concept: Each phase (specify/plan/tasks/implement) is executed through `ExecutePhase()` which validates output artifacts and retries on failure.

### 3. Configuration (`internal/config/`)

Hierarchical configuration with priority ordering:
- **config.go**: Loads config from multiple sources using koanf
- **defaults.go**: Default configuration values

**Priority**: Environment variables > Local config (.autospec/config.json) > Global config (~/.autospec/config.json) > Defaults

Key settings:
- `claude_cmd`: Claude CLI command (default: "claude")
- `specify_cmd`: SpecKit CLI command (default: "specify")
- `max_retries`: Maximum retry attempts (default: 3)
- `specs_dir`: Directory for feature specs (default: "./specs")
- `state_dir`: Retry state storage (default: "~/.autospec/state")
- `timeout`: Command execution timeout in seconds (default: 0 = no timeout, valid range: 0 or 1-604800 (7 days))

**Timeout Behavior**:
- `0` or missing: No timeout (infinite wait) - backward compatible default
- `1-3600`: Timeout in seconds (1 second to 1 hour)
- Commands exceeding timeout are terminated with SIGKILL
- Timeout errors return exit code 5
- Configure via `AUTOSPEC_TIMEOUT` environment variable or config files

### 4. Validation (`internal/validation/`)

Fast validation functions (<10ms performance contract):
- **validation.go**: File validation (spec.md, plan.md, tasks.md existence)
- **tasks.go**: Task parsing and completion checking
- **prompt.go**: Continuation prompt generation

Validates artifacts exist and meet completeness criteria.

### 5. Retry Management (`internal/retry/`)

Persistent retry state tracking:
- **retry.go**: `RetryState` persisted to `~/.autospec/state/retry.json`

Key behaviors:
- Tracks retry count per spec:phase combination
- Atomic file writes for concurrency safety
- Configurable max retries (1-10)
- Exit code 2 when retries exhausted

### 6. Spec Detection (`internal/spec/`)

Automatic spec detection from context:
- **spec.go**: `DetectCurrentSpec()` with fallback strategies

Detection strategies (in order):
1. Git branch name (e.g., `002-go-binary-migration`)
2. Most recently modified directory in `specs/`

### 7. Git Integration (`internal/git/`)

Git repository helpers:
- **git.go**: Check if in git repo, get current branch

## Key Architectural Patterns

### Workflow Execution Flow

**Full Workflow (specify → plan → tasks → implement):**

```
WorkflowOrchestrator.RunFullWorkflow()
  ↓
  Pre-flight checks (if not skipped)
  ↓
  Phase 1: Specify
    → Executor.ExecutePhase(PhaseSpecify, "/speckit.specify", ValidateSpec)
    → ClaudeExecutor runs command via Claude CLI
    → Validates spec.md exists
    → Retries up to max_retries on failure
  ↓
  Phase 2: Plan
    → Executor.ExecutePhase(PhasePlan, "/speckit.plan", ValidatePlan)
    → Validates plan.md exists
  ↓
  Phase 3: Tasks
    → Executor.ExecutePhase(PhaseTasks, "/speckit.tasks", ValidateTasks)
    → Validates tasks.md exists
  ↓
  Phase 4: Implement
    → Executor.ExecutePhase(PhaseImplement, "/speckit.implement", ValidateTasksComplete)
    → Validates all tasks are checked
```

**Planning Workflow (specify → plan → tasks):**

```
WorkflowOrchestrator.RunCompleteWorkflow()
  ↓
  Pre-flight checks (if not skipped)
  ↓
  Phase 1: Specify
  ↓
  Phase 2: Plan
  ↓
  Phase 3: Tasks
  (stops before implementation)
```

### Phase Execution with Retry

Each phase follows this pattern (in `executor.go`):

```go
ExecutePhase(specName, phase, command, validationFunc):
  1. Load retry state from disk
  2. Execute command via ClaudeExecutor
  3. Run validation function on output
  4. If validation fails:
     - Increment retry count
     - Save state
     - If retries remain: retry
     - If exhausted: return error
  5. If validation succeeds:
     - Reset retry count
     - Return success
```

### Configuration Loading

Configuration uses a layered approach (see `config/config.go`):

```
Defaults (in-memory)
  ↓ overridden by
Global config (~/.autospec/config.json)
  ↓ overridden by
Local config (.autospec/config.json)
  ↓ overridden by
Environment variables (AUTOSPEC_*)
```

### Retry State Persistence

Retry state stored as JSON in `~/.autospec/state/retry.json`:

```json
{
  "retries": {
    "002-go-binary-migration:specify": {
      "spec_name": "002-go-binary-migration",
      "phase": "specify",
      "count": 1,
      "last_attempt": "2025-10-22T10:30:00Z",
      "max_retries": 3
    }
  }
}
```

Atomic writes via temp file + rename ensure consistency.

## Constitution Principles

Development follows `.specify/memory/constitution.md`:

1. **Validation-First**: All workflow transitions validated before proceeding
2. **Hook-Based Enforcement**: Quality gates via Claude Code hooks (legacy bash scripts in `scripts/hooks/`)
3. **Test-First Development** (NON-NEGOTIABLE): Tests written before implementation
4. **Performance Standards**: Sub-second validation (<1s); validation functions <10ms
5. **Idempotency & Retry Logic**: All operations idempotent; configurable retry limits

## Testing Architecture

### Go Tests (`*_test.go`)
- Unit tests for each package
- Table-driven tests for validation logic
- Benchmark tests for performance validation (`*_bench_test.go`)
- Run with: `go test ./...`

### Bats Tests (Legacy - `tests/`)
Tests for legacy bash scripts that are being phased out:
- **tests/lib/**: Validation library tests
- **tests/scripts/**: Workflow script tests
- **tests/hooks/**: Stop hook tests
- **tests/integration.bats**: End-to-end tests
- Run with: `./tests/run-all-tests.sh`

## Important Implementation Details

### Exit Code Conventions

All operations use standardized exit codes:
- `0`: Success
- `1`: Validation failed (retryable)
- `2`: Retry limit exhausted
- `3`: Invalid arguments
- `4`: Missing dependencies
- `5`: Command execution timeout

These support programmatic composition and CI/CD integration.

**Timeout Error Handling**:
When a command times out, the CLI:
1. Detects `TimeoutError` from the workflow executor
2. Prints error message with timeout duration and command that timed out
3. Provides hints on increasing timeout (environment variable or config file)
4. Exits with code 5

Example:
```bash
./autospec workflow "feature" && echo "Success" || echo "Failed with code $?"
# If timeout occurs:
# Error: command timed out after 5m0s: claude /speckit.workflow ... (hint: increase timeout in config)
# To increase the timeout, set AUTOSPEC_TIMEOUT environment variable or update config.json:
#   export AUTOSPEC_TIMEOUT=600  # 10 minutes
#   or edit .autospec/config.json and set "timeout": 600
# Failed with code 5
```

### Performance Contracts

- **Validation functions**: <10ms (e.g., `ValidateSpecFile()`)
- **Retry state load/save**: <10ms
- **Overall workflow**: Sub-second for validation checks

Performance regressions beyond 5s require immediate attention.

### Claude Execution Modes

The tool supports multiple ways to execute Claude commands:

1. **CLI mode** (default): Executes via `claude` CLI binary
2. **API mode**: Direct API calls using API key (set `use_api_key: true`)
3. **Custom mode**: Custom command with `{{PROMPT}}` placeholder

Configure in config.json or via environment variables.

### Spec Detection Logic

Automatic spec detection reduces friction:

```go
DetectCurrentSpec():
  1. Check git branch name (e.g., "002-feature-name")
  2. If no match, find most recently modified specs/*/ directory
  3. Parse directory name for spec number and name
  4. Return Metadata{Number, Name, Directory, Branch}
```

This allows running `autospec plan` without specifying the spec name.

## Common Development Patterns

### Adding a New CLI Command

1. Create new file in `internal/cli/` (e.g., `analyze.go`)
2. Define cobra.Command with Use, Short, Long, RunE
3. Register command in init() function
4. Implement business logic by calling workflow/validation packages
5. Add tests in `*_test.go` file
6. Update help text and README

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

### Debugging

```bash
# Enable debug logging
autospec --debug workflow "feature"
autospec -d plan

# Check retry state
cat ~/.autospec/state/retry.json

# Check config loading (including timeout value)
autospec config show

# Verbose output
autospec --verbose workflow "feature"

# Timeout-specific debugging
echo $AUTOSPEC_TIMEOUT           # Check environment variable
cat .autospec/config.json | jq .timeout  # Check local config
cat ~/.autospec/config.json | jq .timeout  # Check global config

# Test timeout behavior with short timeout
AUTOSPEC_TIMEOUT=5 autospec specify "test"  # Should timeout quickly

# Disable timeout temporarily
AUTOSPEC_TIMEOUT=0 autospec workflow "feature"  # No timeout
```

## Migration Notes

This project is transitioning from bash scripts to a Go binary:

### Current State (002-go-binary-migration branch)
- ✅ Go binary with CLI commands
- ✅ Configuration system (koanf)
- ✅ Retry management (persistent state)
- ✅ Validation logic (Go implementation)
- ✅ Workflow orchestration
- ✅ Spec detection
- ⚠️  Legacy bash scripts remain in `scripts/` (deprecated)
- ⚠️  Legacy bats tests remain in `tests/` (deprecated)

### Phase-Out Plan
- Legacy bash scripts in `scripts/` will be removed after migration validation
- Bats tests will be replaced by Go tests
- Hook scripts may remain as they integrate with Claude Code's hook system

### Using Legacy Scripts
If needed, legacy bash scripts are still available:

```bash
# Legacy workflow validation
./scripts/speckit-workflow-validate.sh <feature-name>

# Legacy implementation validation
./scripts/speckit-implement-validate.sh <feature-name>
```

## Key Files and Locations

- `cmd/autospec/main.go`: Binary entry point
- `internal/cli/`: CLI commands
- `internal/workflow/`: Workflow orchestration
- `internal/config/`: Configuration management
- `internal/retry/`: Retry state tracking
- `internal/validation/`: Validation functions
- `.autospec/config.json`: Local configuration
- `~/.autospec/config.json`: Global configuration
- `~/.autospec/state/retry.json`: Retry state
- `specs/*/`: Feature specifications
- `Makefile`: Common development commands

## Active Technologies
- Go 1.25.1 (003-command-timeout)
- File system (JSON config files in ~/.autospec/config.json and .autospec/config.json, state in ~/.autospec/state/retry.json) (003-command-timeout)
- Go 1.25.1 + Cobra CLI (v1.10.1), briandowns/spinner (v1.23.0+), golang.org/x/term (v0.25.0+) (004-workflow-progress-indicators)
- N/A (progress state is ephemeral, displayed only during execution) (004-workflow-progress-indicators)

## Recent Changes
- 003-command-timeout: Added Go 1.25.1
