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
# Run all tests
make test

# Run Go tests
go test -v -race -cover ./...

# Run single Go test
go test -v -run TestValidateSpecFile ./internal/validation/
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
# Flexible phase selection with run command
autospec run -a "Add user authentication feature"          # All core phases
autospec run -spti "Add user authentication feature"       # Same as -a
autospec run -pi                                           # Plan + implement on current spec
autospec run -ti --spec 007-yaml-output                    # Tasks + implement on specific spec
autospec run -p "Focus on security best practices"         # Plan with prompt guidance
autospec run -ti -y                                        # Skip confirmation prompts

# Run with optional phases
autospec run -sr "Add user auth"                           # Specify + clarify
autospec run -al "Add user auth"                           # All core + checklist
autospec run -tlzi                                         # Tasks + checklist + analyze + implement
autospec run -ns "Create constitution"                     # Constitution + specify

# Run complete workflow: specify → plan → tasks → implement
autospec all "Add user authentication feature"             # Shortcut for run -a

# Prepare for implementation (specify → plan → tasks, no implementation)
autospec prep "Add user authentication feature"

# Run individual core phases
autospec specify "feature description"
autospec plan
autospec plan "Focus on security best practices"           # With prompt guidance
autospec tasks
autospec tasks "Break into small incremental steps"        # With prompt guidance
autospec implement
autospec implement "Focus on documentation tasks"          # With prompt guidance
autospec implement 003-my-feature                          # Specific spec
autospec implement 003-my-feature "Complete tests"         # Spec + prompt
autospec implement --phases                                # Run each phase in separate Claude session
autospec implement --phase 3                               # Run only phase 3
autospec implement --from-phase 3                          # Run phases 3 onwards
autospec implement --tasks                                 # Run each task in a separate Claude session
autospec implement --tasks --from-task T005                # Start from task T005
autospec implement --task T003                             # Execute only task T003

# Run individual optional phases
autospec constitution                                      # Create/update project constitution
autospec constitution "Focus on security principles"       # With prompt guidance
autospec clarify                                           # Refine spec with clarification questions
autospec clarify "Focus on edge cases"                     # With prompt guidance
autospec checklist                                         # Generate custom checklist for feature
autospec checklist "Focus on security requirements"        # With prompt guidance
autospec analyze                                           # Cross-artifact consistency analysis
autospec analyze "Verify API contracts"                    # With prompt guidance

# Update task status during implementation
autospec update-task T001 InProgress                       # Mark task as in progress
autospec update-task T001 Completed                        # Mark task as completed
autospec update-task T001 Blocked                          # Mark task as blocked

# Create new feature branch and directory (replaces create-new-feature.sh)
autospec new-feature "Add user authentication"             # From description
autospec new-feature --short-name "user-auth" "Add auth"   # Custom short name
autospec new-feature --number 5 "OAuth integration"        # Specific number
autospec new-feature --json "Add feature"                  # JSON output for scripting

# Check prerequisites (replaces check-prerequisites.sh)
autospec prereqs --json --require-spec                     # Require spec.yaml
autospec prereqs --json --require-plan                     # Require plan.yaml (default)
autospec prereqs --json --require-tasks --include-tasks    # For implementation phase
autospec prereqs --paths-only                              # Output paths only (no validation)

# Setup plan from template (replaces setup-plan.sh)
autospec setup-plan                                        # Initialize plan.yaml
autospec setup-plan --json                                 # JSON output for scripting

# Update AI agent context files from plan.yaml
autospec update-agent-context                              # Update all existing agent files
autospec update-agent-context --agent claude               # Update only CLAUDE.md
autospec update-agent-context --agent cursor               # Update/create Cursor context file
autospec update-agent-context --json                       # JSON output for integration

# Validate YAML artifacts against schemas
autospec artifact plan                                     # Type only: auto-detect spec from branch
autospec artifact specs/001-feature/plan.yaml             # Path only: infer type from filename
autospec artifact plan specs/001-feature/plan.yaml        # Explicit type + path (backward compatible)
autospec artifact plan --schema                            # Show plan schema
autospec artifact plan --fix                               # Auto-fix current spec's plan.yaml

# Check dependencies
autospec doctor

# Check status (alias: st)
autospec status                    # Show artifacts and task progress
autospec st                        # Short alias
autospec st -v                     # Verbose: show phase details

# Initialize user-level config (default, creates ~/.config/autospec/config.yml)
autospec init

# Initialize project-level config (creates .autospec/config.yml)
autospec init --project

# Show current effective configuration
autospec config show

# Show configuration as JSON
autospec config show --json

# Migrate legacy JSON config to YAML
autospec config migrate

# Preview migration without making changes
autospec config migrate --dry-run

# Show version
autospec version

# Install shell completions (auto-detects shell)
autospec completion install                                  # Auto-detect shell from $SHELL
autospec completion install bash                             # Install for bash
autospec completion install zsh                              # Install for zsh
autospec completion install fish                             # Install for fish
autospec completion install powershell                       # Install for PowerShell
autospec completion install --manual                         # Show manual instructions only

# Clean up autospec files from a project (specs/ preserved by default)
autospec clean --dry-run                                   # Preview files to be removed
autospec clean                                             # Remove with confirmation (prompts about specs/)
autospec clean --yes                                       # Remove without confirmation (specs/ preserved)
autospec clean --keep-specs                                # Explicitly preserve specs/ (skip prompt)
autospec clean --remove-specs                              # Include specs/ in removal (skip prompt)
autospec clean --yes --remove-specs                        # Remove everything without confirmation

# Uninstall autospec completely (removes binary, user config, state)
autospec uninstall --dry-run                               # Preview what would be removed
autospec uninstall                                         # Uninstall with confirmation (y/N prompt)
autospec uninstall --yes                                   # Uninstall without confirmation
sudo autospec uninstall --yes                              # Uninstall if binary is in system directory
```

## Architecture Overview

autospec is a **cross-platform Go binary** that automates SpecKit workflow validation and orchestration. The architecture consists of several key layers:

### 1. CLI Layer (`internal/cli/`)

Cobra-based command structure providing user-facing commands:
- **root.go**: Root command with global flags (`--config`, `--specs-dir`, `--debug`, etc.)
- **run.go**: Flexible phase selection with core and optional phase flags
- **all.go**: Orchestrates complete specify → plan → tasks → implement workflow
- **prep.go**: Orchestrates specify → plan → tasks workflow (no implementation)
- **specify.go**: Creates new feature specifications with optional prompt guidance
- **plan.go**: Executes planning phase with optional prompt guidance
- **tasks.go**: Generates tasks with optional prompt guidance
- **implement.go**: Executes implementation with optional spec-name and prompt guidance
- **constitution.go**: Creates/updates project constitution
- **clarify.go**: Refines specification with clarification questions
- **checklist.go**: Generates custom checklist for feature
- **analyze.go**: Cross-artifact consistency and quality analysis
- **update_task.go**: Updates individual task status in tasks.yaml during implementation
- **new_feature.go**: Creates new feature branch and directory (replaces create-new-feature.sh)
- **prereqs.go**: Checks prerequisites and validates artifacts (replaces check-prerequisites.sh)
- **setup_plan.go**: Initializes plan file from template (replaces setup-plan.sh)
- **artifact.go**: Validates YAML artifacts against schemas with --schema and --fix flags
- **clean.go**: Removes autospec files from a project (.autospec/, .claude/commands/autospec*.md); specs/ preserved by default
- **uninstall.go**: Completely removes autospec from system (binary, ~/.config/autospec/, ~/.autospec/)
- **doctor.go**: Health check command for dependencies
- **status.go**: Reports artifact files and task progress (alias: `st`)
- **config.go**: Configuration management commands
- **init.go**: Initializes configuration files
- **version.go**: Version information
- **completion_install.go**: Installs shell completions with auto-detection

**Core Phase Selection Flags (run command):**
- `-s, --specify`: Include specify phase (requires feature description)
- `-p, --plan`: Include plan phase
- `-t, --tasks`: Include tasks phase
- `-i, --implement`: Include implement phase
- `-a, --all`: Run all core phases (equivalent to `-spti`)

**Optional Phase Selection Flags (run command):**
- `-n, --constitution`: Include constitution phase
- `-r, --clarify`: Include clarify phase
- `-l, --checklist`: Include checklist phase (note: `-c` is used for `--config`)
- `-z, --analyze`: Include analyze phase

**Other Flags:**
- `-y, --yes`: Skip confirmation prompts
- `--spec`: Specify which spec to work with (overrides branch detection)
- `--max-retries`: Override max retry attempts (long-only, no short flag)

**Implement Command Task-Level Flags:**
- `--tasks`: Run each task in a separate Claude session (isolated context per task)
- `--from-task TXXX`: Start execution from specified task ID (e.g., T003)
- `--task TXXX`: Execute only the specified task

**Note:** `--tasks`/`--from-task`/`--task` are mutually exclusive with `--phases`/`--phase`/`--from-phase`. Use task-level OR phase-level execution, not both.

**Canonical Phase Order:**
Phases always execute in this order, regardless of flag order:
`constitution → specify → clarify → plan → tasks → checklist → analyze → implement`

### 2. Workflow Orchestration (`internal/workflow/`)

Manages the complete SpecKit workflow lifecycle:
- **workflow.go**: `WorkflowOrchestrator` executes multi-phase workflows with validation
  - `RunFullWorkflow()`: Complete workflow including implementation (specify → plan → tasks → implement)
  - `RunCompleteWorkflow()`: Planning workflow only (specify → plan → tasks)
  - `ExecuteImplement()`: Implementation phase only
  - `ExecuteImplementWithTasks()`: Task-level implementation with isolated Claude sessions per task
- **executor.go**: `Executor` handles phase execution with retry logic
- **claude.go**: `ClaudeExecutor` interfaces with Claude CLI or API
- **preflight.go**: Pre-flight checks for dependencies (claude CLI, git)
  - `CheckArtifactDependencies()`: Validates required artifacts exist before phase execution
  - `GeneratePrerequisiteWarning()`: Generates human-readable warnings for missing prerequisites
- **phase_config.go**: Phase configuration and dependency management
  - `PhaseConfig`: Represents selected phases for execution
  - `PhaseExecutionOptions`: Execution options including TaskMode, FromTask, SingleTask
  - `ArtifactDependency`: Maps phases to required/produced artifacts
  - `GetCanonicalOrder()`: Returns phases in execution order (specify → plan → tasks → implement)
  - `GetAllRequiredArtifacts()`: Returns external dependencies for selected phases

Key concept: Each phase (specify/plan/tasks/implement) is executed through `ExecutePhase()` which validates output artifacts and retries on failure.

### 3. Configuration (`internal/config/`)

Hierarchical configuration with priority ordering and YAML format:
- **config.go**: Loads config from multiple sources using koanf with YAML parser
- **defaults.go**: Default configuration values
- **paths.go**: XDG-compliant config path resolution
- **validate.go**: YAML validation with line-number error reporting
- **migrate.go**: JSON to YAML migration utilities

**Configuration Files (YAML format):**
- User config: `~/.config/autospec/config.yml` (XDG compliant, applies to all projects)
- Project config: `.autospec/config.yml` (project-specific overrides)

**Priority**: Environment variables > Project config (.autospec/config.yml) > User config (~/.config/autospec/config.yml) > Defaults

**Legacy JSON Support**: JSON config files at the old locations (`.autospec/config.json`, `~/.autospec/config.json`) are still supported but trigger deprecation warnings. Use `autospec config migrate` to convert to YAML.

Key settings:
- `claude_cmd`: Claude CLI command (default: "claude")
- `max_retries`: Maximum retry attempts (default: 3)
- `specs_dir`: Directory for feature specs (default: "./specs")
- `state_dir`: Retry state storage (default: "~/.autospec/state")
- `timeout`: Command execution timeout in seconds (default: 2400 = 40 minutes, valid range: 0 or 1-604800 (7 days))
- `skip_confirmations`: Skip confirmation prompts (default: false, can also be set via AUTOSPEC_YES env var)

**Timeout Behavior**:
- `0`: No timeout (infinite wait)
- Default: 2400 seconds (40 minutes)
- `1-3600`: Timeout in seconds (1 second to 1 hour)
- Commands exceeding timeout are terminated with SIGKILL
- Timeout errors return exit code 5
- Configure via `AUTOSPEC_TIMEOUT` environment variable or config files

### 4. Validation (`internal/validation/`)

Fast validation functions (<10ms performance contract):
- **validation.go**: File validation (spec.yaml, plan.yaml, tasks.yaml existence)
- **tasks.go**: Task parsing and completion checking
- **prompt.go**: Continuation prompt generation
- **schema.go**: Schema definitions for spec, plan, and tasks artifacts
- **artifact.go**: Core artifact validation infrastructure (ArtifactValidator interface)
- **artifact_spec.go**: Spec artifact validator (validates feature, user_stories, requirements)
- **artifact_plan.go**: Plan artifact validator (validates plan, summary, technical_context)
- **artifact_tasks.go**: Tasks artifact validator with circular dependency detection
- **autofix.go**: Auto-fix functionality for common artifact issues

Validates artifacts exist and meet completeness criteria.

**Artifact Validation CLI** (`autospec artifact`):
```bash
# Smart detection: type only (auto-detects spec from git branch)
autospec artifact plan                      # Validates specs/NNN-name/plan.yaml
autospec artifact spec                      # Validates specs/NNN-name/spec.yaml
autospec artifact tasks                     # Validates specs/NNN-name/tasks.yaml

# Smart detection: path only (infers type from filename)
autospec artifact specs/001-feature/plan.yaml    # Infers "plan" type from filename

# Explicit type + path (backward compatible)
autospec artifact plan specs/001-feature/plan.yaml

# View schema for an artifact type
autospec artifact spec --schema
autospec artifact plan --schema
autospec artifact tasks --schema

# Auto-fix common issues (missing _meta section, formatting)
autospec artifact plan --fix                # Fixes current spec's plan.yaml
autospec artifact plan specs/001/plan.yaml --fix  # Fixes specific file
```

**Auto-detection behavior:**
- Type-only: Uses `DetectCurrentSpec()` to find spec directory from git branch (e.g., `016-smart-artifact-validation` → `specs/016-smart-artifact-validation/`)
- Falls back to most recently modified spec directory when branch doesn't match
- Output shows "Using spec: <name>" (green) or "Using spec: <name> (fallback)" (yellow)

Exit codes: 0 (valid), 1 (validation failed), 3 (invalid arguments)

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
    → Executor.ExecutePhase(PhaseSpecify, "/autospec.specify", ValidateSpec)
    → ClaudeExecutor runs command via Claude CLI
    → Validates spec.yaml exists
    → Retries up to max_retries on failure
  ↓
  Phase 2: Plan
    → Executor.ExecutePhase(PhasePlan, "/autospec.plan", ValidatePlan)
    → Validates plan.yaml exists
  ↓
  Phase 3: Tasks
    → Executor.ExecutePhase(PhaseTasks, "/autospec.tasks", ValidateTasks)
    → Validates tasks.yaml exists
  ↓
  Phase 4: Implement
    → Executor.ExecutePhase(PhaseImplement, "/autospec.implement", ValidateTasksComplete)
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
  2. Display full command before execution
  3. Execute command via ClaudeExecutor
  4. Run validation function on output
  5. If validation fails:
     - Increment retry count
     - Save state
     - If retries remain: retry
     - If exhausted: return error
  6. If validation succeeds:
     - Reset retry count
     - Return success
```

### Task-Level Execution

When using `--tasks` flag, implementation runs each task in an isolated Claude session:

```
ExecuteImplementWithTasks(specName, opts):
  1. Load tasks from tasks.yaml in dependency order
  2. If FromTask specified, skip tasks until reaching that task ID
  3. For each task:
     a. Validate dependencies met (all deps have status: Completed)
     b. Execute task in isolated Claude session via /autospec.implement --task TXXX
     c. Verify task marked Completed in tasks.yaml after session ends
     d. Track progress in TaskExecutionState (persisted to ~/.autospec/state/)
     e. Retry on failure using standard retry logic
  4. Report summary of completed tasks
```

**Key benefits:**
- **Context isolation**: Each task starts with fresh Claude context, preventing confusion from accumulated state
- **Resumable**: Use `--from-task T005` to resume from a specific task after interruption
- **Single task execution**: Use `--task T003` to execute only one specific task
- **Dependency validation**: Tasks only execute when all dependencies are Completed

**Typical use cases:**
```bash
# Full task-level implementation (isolated sessions)
autospec implement --tasks

# Resume after task T004 failed or was interrupted
autospec implement --tasks --from-task T005

# Execute only one specific task
autospec implement --task T003
```

### Prompt Injection and Command Display

All SpecKit phase commands (`specify`, `plan`, `tasks`, `implement`) support optional prompt text to guide Claude's execution:

**How it works:**
1. User provides additional guidance as command arguments
2. Prompt text is appended to the slash command
3. Full command is displayed before execution
4. Works with both simple and custom Claude commands

**Examples:**
```bash
# Simple prompt injection
autospec plan "Focus on security best practices"
# Executes: claude -p "/autospec.plan \"Focus on security best practices\""

# Custom command with prompt
# Config: custom_claude_cmd = "claude -p {{PROMPT}} | claude-clean"
autospec tasks "Break into small steps"
# Executes: claude -p '/autospec.tasks "Break into small steps"' | claude-clean

# Implement command with spec-name and prompt
autospec implement 003-my-feature "Complete the tests"
# Auto-detects spec vs prompt using pattern matching (NNN-name)
```

**Command Display:**
Before each phase execution, the full resolved command is displayed:
```
→ Executing: claude -p "/autospec.plan \"Focus on security\""
```

This transparency helps with:
- Debugging command construction
- Understanding what's being sent to Claude
- Verifying custom command templates work correctly

**Argument Parsing (implement command):**
The `implement` command intelligently distinguishes between spec-name and prompt:
- Pattern `\d+-[a-z0-9-]+` → treated as spec-name (e.g., `003-my-feature`)
- Anything else → treated as prompt text
- Can combine: `autospec implement 003-feature "focus on docs"`

### Configuration Loading

Configuration uses a layered approach (see `config/config.go`):

```
Defaults (in-memory)
  ↓ overridden by
Global config (~/.config/autospec/config.yml)
  ↓ overridden by
Local config (.autospec/config.yml)
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

Development follows `.autospec/memory/constitution.yaml`:

1. **Validation-First**: All workflow transitions validated before proceeding
2. **Test-First Development** (NON-NEGOTIABLE): Tests written before implementation
3. **Performance Standards**: Sub-second validation (<1s); validation functions <10ms
4. **Idempotency & Retry Logic**: All operations idempotent; configurable retry limits

## Testing Architecture

### Go Tests (`*_test.go`)
- Unit tests for each package
- Table-driven tests for validation logic
- Benchmark tests for performance validation (`*_bench_test.go`)
- Integration tests in `tests/integration/`
- Run with: `go test ./...` or `make test`

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
autospec prep "feature" && echo "Success" || echo "Failed with code $?"
# If timeout occurs:
# Error: command timed out after 5m0s: claude /autospec.prep ... (hint: increase timeout in config)
# To increase the timeout, set AUTOSPEC_TIMEOUT environment variable or update config.yml:
#   export AUTOSPEC_TIMEOUT=600  # 10 minutes
#   or edit .autospec/config.yml and set "timeout: 600"
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

Configure in config.yml or via environment variables.

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

### Code Formatting

**IMPORTANT:** When modifying Go files, always run `make fmt` near the end of your todo list before committing:

```bash
# Format all Go files
make fmt
```

This ensures consistent code style across the codebase. The command runs `go fmt ./...` and `go vet ./...`.

### Debugging

```bash
# Enable debug logging
autospec --debug prep "feature"
autospec -d plan

# Check retry state
cat ~/.autospec/state/retry.json

# Check config loading (including timeout value)
autospec config show

# Verbose output
autospec --verbose prep "feature"

# Timeout-specific debugging
echo $AUTOSPEC_TIMEOUT           # Check environment variable
cat .autospec/config.yml         # Check local config
cat ~/.config/autospec/config.yml  # Check global config

# Test timeout behavior with short timeout
AUTOSPEC_TIMEOUT=5 autospec specify "test"  # Should timeout quickly

# Disable timeout temporarily
AUTOSPEC_TIMEOUT=0 autospec prep "feature"  # No timeout
```

## Project Status

Fully migrated Go binary with:
- CLI commands (Cobra)
- Configuration system (koanf with YAML)
- Retry management (persistent state)
- Validation logic
- Workflow orchestration
- Spec detection

## Key Files and Locations

- `cmd/autospec/main.go`: Binary entry point
- `internal/cli/`: CLI commands
- `internal/workflow/`: Workflow orchestration
- `internal/config/`: Configuration management
- `internal/retry/`: Retry state tracking
- `internal/validation/`: Validation functions
- `~/.config/autospec/config.yml`: User-level configuration (XDG compliant)
- `.autospec/config.yml`: Project-level configuration
- `~/.autospec/state/retry.json`: Retry state
- `specs/*/`: Feature specifications
- `Makefile`: Common development commands

**Legacy locations (deprecated, trigger migration warnings):**
- `.autospec/config.json`: Legacy project config (use `autospec config migrate --project`)
- `~/.autospec/config.json`: Legacy user config (use `autospec config migrate --user`)

## Active Technologies
- Go 1.25.1 + Cobra CLI (v1.10.1)
- koanf config (v2.3.0) with YAML format
- go-playground/validator (v10.28.0)
- briandowns/spinner (v1.23.0+)
- gopkg.in/yaml.v3 (v3.0.1)
- File system (YAML config in ~/.config/autospec/config.yml and .autospec/config.yml, state in ~/.autospec/state/retry.json)
- YAML artifacts in `specs/*/` (spec.yaml, plan.yaml, tasks.yaml)
- github.com/spf13/cobra v1.10.1
- Project Type: cli
## Recent Changes
- 017-update-agent-context-go: Added from plan.yaml
- 003-command-timeout: Added Go 1.25.1

## Important Notes

### Legacy Scripts (Removed)
Shell scripts have been fully migrated to Go commands. The following Go commands replace the legacy scripts:
- `autospec new-feature` - replaces `create-new-feature.sh`
- `autospec prereqs` - replaces `check-prerequisites.sh`
- `autospec setup-plan` - replaces `setup-plan.sh`
- `autospec update-agent-context` - replaces `update-agent-context.sh`

The slash commands in `.claude/commands/` now call these Go commands directly instead of shell scripts.

**Last updated**: 2025-12-16
