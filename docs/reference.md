# Command Reference

Complete reference for autospec commands, configuration options, exit codes, and file locations.

## CLI Commands

All commands support global flags: `--config`, `--specs-dir`, `--debug`, `--verbose`

### autospec full

Execute complete workflow: specify → plan → tasks → implement

**Syntax**: `autospec full "<feature description>" [flags]`

**Description**: Creates specification, generates plan and tasks, then executes implementation in a single command.

**Flags**:
- `--skip-preflight`: Skip dependency health checks
- `--timeout <seconds>`: Command timeout (0=infinite, 1-604800)
- `--max-retries <count>`: Maximum retry attempts (1-10, default: 3)
- `--agent <name>`: Override agent for this run (see [CLI Agents](#cli-agents))

**Examples**:
```bash
autospec full "Add user authentication with OAuth"
autospec full "Add dark mode toggle" --timeout 600
autospec full "Export data to CSV" --skip-preflight
autospec full "Add caching" --agent gemini
```

**Exit Codes**: 0 (success), 1 (validation failed), 2 (retries exhausted), 3 (invalid args), 4 (missing deps), 5 (timeout)

### autospec prep

Prepare for implementation: specify → plan → tasks (no implementation)

**Syntax**: `autospec prep "<feature description>" [flags]`

**Description**: Creates specification and generates plan/tasks for review before implementation.

**Flags**: Same as `autospec full`

**Examples**:
```bash
autospec prep "Add user profile page"
autospec prep "Implement caching layer" --max-retries 5
```

**Exit Codes**: 0 (success), 1 (validation failed), 2 (retries exhausted), 3 (invalid args), 4 (missing deps), 5 (timeout)

### autospec specify

Create feature specification from natural language description

**Syntax**: `autospec specify "<feature description>" ["<guidance>"] [flags]`

**Alias**: `autospec spec`, `autospec s`

**Description**: Generate detailed specification with requirements, acceptance criteria, and success metrics.

**Flags**: Same as `autospec full`

**Examples**:
```bash
autospec specify "Add real-time notifications"
autospec specify "Add API rate limiting" "Focus on security"
```

**Exit Codes**: 0 (success), 1 (validation failed), 2 (retries exhausted), 3 (invalid args), 4 (missing deps), 5 (timeout)

### autospec plan

Generate technical implementation plan from specification

**Syntax**: `autospec plan ["<guidance>"] [flags]`

**Alias**: `autospec p`

**Description**: Create technical plan with architecture, file structure, and design decisions.

**Flags**: Same as `autospec full`

**Examples**:
```bash
autospec plan
autospec plan "Prioritize performance and scalability"
autospec plan --timeout 300
```

**Exit Codes**: 0 (success), 1 (validation failed), 2 (retries exhausted), 3 (invalid args), 4 (missing deps), 5 (timeout)

### autospec tasks

Generate task breakdown from implementation plan

**Syntax**: `autospec tasks ["<guidance>"] [flags]`

**Alias**: `autospec t`

**Description**: Break down plan into ordered, actionable tasks with dependencies.

**Flags**: Same as `autospec full`

**Examples**:
```bash
autospec tasks
autospec tasks "Break into small incremental steps"
```

**Exit Codes**: 0 (success), 1 (validation failed), 2 (retries exhausted), 3 (invalid args), 4 (missing deps), 5 (timeout)

### autospec implement

Execute implementation phase using tasks breakdown

**Syntax**: `autospec implement [<spec-name>] ["<guidance>"] [flags]`

**Alias**: `autospec impl`, `autospec i`

**Description**: Execute tasks with Claude's assistance, validating progress. Supports multiple execution modes for context isolation.

**Flags**:
- `--phases`: Run each phase in a separate Claude session (fresh context per phase)
- `--phase <N>`: Run only the specified phase number
- `--from-phase <N>`: Run phases N and onwards, each in separate session
- `--tasks`: Run each task in a separate Claude session (maximum context isolation)
- `--from-task <ID>`: Resume from specific task ID
- `--single-session`: Run all tasks in one Claude session (legacy mode)
- Plus all flags from `autospec full`

**Execution Modes**:

| Mode | Flag | Sessions | Use Case |
|------|------|----------|----------|
| Phase-level | (default) | 1 per phase | Balanced cost/context |
| Task-level | `--tasks` | 1 per task | Large specs, maximum isolation |
| Single-session | `--single-session` | 1 | Small specs, quick iterations |

**Examples**:
```bash
# Default: phase-level isolation (1 session per phase)
autospec implement
autospec implement 001-dark-mode
autospec implement --phase 2             # Run only phase 2
autospec implement --from-phase 3        # Run phases 3+ sequentially

# Task-level isolation (maximum granularity)
autospec implement --tasks               # Each task in separate session
autospec implement --from-task T005      # Resume from task T005

# Single-session (all tasks in one session)
autospec implement --single-session

# With guidance
autospec implement --phases "Focus on tests first"
```

**Exit Codes**: 0 (success), 1 (validation failed), 2 (retries exhausted), 3 (invalid args), 4 (missing deps), 5 (timeout)

### autospec doctor

Run health checks and verify dependencies

**Syntax**: `autospec doctor [flags]`

**Alias**: `autospec doc`

**Description**: Verify Claude CLI installed, authenticated, and directories accessible.

**Flags**: None (uses global flags only)

**Examples**:
```bash
autospec doctor
autospec doctor --debug
```

**Exit Codes**: 0 (all checks passed), 4 (dependencies missing)

### autospec history

View command execution history

**Syntax**: `autospec history [flags]`

**Description**: Display a log of all autospec command executions with timestamp, unique ID, status, command name, spec, exit code, and duration.

**Automatic Logging**: All workflow commands are automatically logged to history:
- Core stages: `specify`, `plan`, `tasks`, `implement`
- Optional stages: `clarify`, `analyze`, `checklist`, `constitution`
- Workflows: `run`, `prep`, `all`

**Two-Phase Logging**: History entries are written **immediately when commands start** (with status `running`) and updated when commands complete. This ensures:
- Running commands are visible in history
- No history data is lost if a command crashes or is interrupted
- Each entry has a unique, memorable ID for tracking

**Flags**:
- `-s, --spec <name>`: Filter by spec name
- `-n, --limit <count>`: Limit to last N entries (most recent)
- `--status <value>`: Filter by status (`running`, `completed`, `failed`, `cancelled`)
- `--clear`: Clear all history

**Output Format**:
```
TIMESTAMP            ID                              STATUS      COMMAND       SPEC              EXIT  DURATION
2024-01-15 10:30:00  brave_fox_20240115_103000       completed   specify       -                 0     2m30s
2024-01-15 10:35:00  calm_river_20240115_103500      completed   plan          001-test-feature  0     1m15s
2024-01-15 10:40:00  swift_falcon_20240115_104000    failed      tasks         001-test-feature  1     45s
2024-01-15 10:45:00  gentle_owl_20240115_104500      running     implement     001-test-feature  0
```

**Columns**:
- **ID**: Unique identifier in `adjective_noun_YYYYMMDD_HHMMSS` format (memorable and sortable)
- **STATUS**: Current state with color coding:
  - Green: `completed` (successful execution)
  - Yellow: `running` (currently executing)
  - Red: `failed` (error occurred) or `cancelled` (user interrupted)
  - `-`: Old entries without status (backward compatibility)

Note: Commands that create new specs (`specify`, `prep`, `all`, `run -s`) log with an empty spec name since the spec doesn't exist yet when the command starts.

**Examples**:
```bash
# View all history
autospec history

# View last 10 entries
autospec history -n 10

# Filter by spec name
autospec history --spec 001-feature

# Filter by status (see running commands)
autospec history --status running

# Filter by failed commands
autospec history --status failed

# Combine filters
autospec history --spec 001-feature --status completed

# Clear all history
autospec history --clear
```

**Exit Codes**: 0 (success), 3 (invalid arguments, e.g., negative limit)

**File Location**: `~/.autospec/state/history.yaml`

**Storage Limit**: History is automatically pruned to `max_history_entries` (default: 500). Oldest entries are removed first when the limit is exceeded. See [Configuration](#max_history_entries) to customize.

### autospec status

Check current feature status and progress

**Syntax**: `autospec status [spec-name] [flags]`

**Alias**: `autospec st`

**Description**: Display detected spec, which artifact files exist (spec.yaml, plan.yaml, tasks.yaml), task completion progress, and risk summary (if plan.yaml contains risks).

**Flags**:
- `-v, --verbose`: Show phase-by-phase breakdown

**Examples**:
```bash
autospec status              # Current spec status
autospec st                  # Short alias
autospec st -v               # Verbose with phase details
autospec status 003-feature  # Specific spec
```

**Output**:
```
015-artifact-validation
  artifacts: [spec.yaml plan.yaml tasks.yaml]
  risks: 3 total (1 high, 2 medium)
  25/38 tasks completed (66%)
  7/10 task phases completed
  (1 in progress)
```

**Exit Codes**: 0 (success), 3 (invalid args)

### autospec view

Display dashboard overview of all specs in the project

**Syntax**: `autospec view [flags]`

**Description**: Shows project-wide spec statistics, recent specs with task progress, and completed specs in a single dashboard view.

**Flags**:
- `-l, --limit <count>`: Number of recent specs to display (default: from config or 5)

**Output Sections**:
1. **Dashboard Header**: Total specs, in-progress count, completed count, skipped count
2. **Recent Specs**: Top N most recently modified specs with status and task progress
3. **Completed Specs**: All specs with Completed status or 100% task completion

**Examples**:
```bash
autospec view                  # Show dashboard with default limit (5)
autospec view --limit 10       # Show top 10 recent specs
autospec view -l 3             # Short flag for limit
```

**Output**:
```
Spec Dashboard
----------------------------------------
Total specs:   48
In progress:   10
Completed:     37
Skipped:       1

Recent Specs (top 5)
----------------------------------------
  063-view-dashboard             Draft
    Progress: 4/18 tasks
  058-config-set-command         Completed
    Progress: 18/18 tasks
  057-fix-description-propaga... Completed
    Progress: 10/10 tasks

Completed Specs
----------------------------------------
  058-config-set-command         18/18 tasks
  057-fix-description-propaga... 10/10 tasks
```

**Status Categories**:
- **In Progress**: Draft, In Progress, Review, or any non-completed/non-skipped status
- **Completed**: Completed status OR 100% task completion
- **Skipped**: Rejected or Skipped status

**Exit Codes**: 0 (success)

### autospec config

Manage configuration settings

**Syntax**: `autospec config <subcommand> [flags]`

**Subcommands**:
- `show`: Display current configuration
- `set <key> <value>`: Set configuration value
- `get <key>`: Get configuration value
- `init`: Initialize default configuration

**Examples**:
```bash
autospec config show
autospec config set max_retries 5
autospec config get timeout
autospec config init
```

**Exit Codes**: 0 (success), 3 (invalid args)

### autospec init

Initialize configuration files and directories

**Syntax**: `autospec init [flags]`

**Description**: Create `~/.config/autospec/config.yml` with default settings. If config already exists, it is left unchanged (use `--force` to overwrite).

**Flags**:
- `--project, -p`: Create project-level config (`.autospec/config.yml`)
- `--force, -f`: Overwrite existing configuration with defaults

**Examples**:
```bash
autospec init              # Create user config if missing
autospec init --project    # Create project-level config
autospec init --force      # Overwrite existing config with defaults
```

**Exit Codes**: 0 (success)

### autospec update-agent-context

Update AI agent context files with technology information from plan.yaml

**Syntax**: `autospec update-agent-context [flags]`

**Description**: Updates AI agent context files (CLAUDE.md, GEMINI.md, etc.) with technology information extracted from the current feature's plan.yaml file. Updates the Active Technologies and Recent Changes sections.

**Flags**:
- `--agent <name>`: Update only the specified agent's context file (e.g., claude, gemini, copilot, cursor)
- `--json`: Output results as JSON for programmatic consumption

**Supported Agents**: claude, gemini, copilot, cursor, qwen, opencode, codex, windsurf, kilocode, auggie, roo, codebuddy, qoder, amp, shai, q, bob

**Examples**:
```bash
autospec update-agent-context                    # Update all existing agent files
autospec update-agent-context --agent claude     # Update only CLAUDE.md
autospec update-agent-context --agent cursor     # Create/update Cursor context file
autospec update-agent-context --json             # JSON output for integration
```

**Exit Codes**: 0 (success), 1 (validation failed), 3 (invalid args)

### autospec artifact

Validate YAML artifacts against their schemas

**Syntax**: `autospec artifact <path>` or `autospec artifact <type> <path>`

**Description**: Validates artifacts against their schemas, checking required fields, types, enums, and cross-references (e.g., task dependencies).

**Supported Types**:
- `spec` - Feature specification (spec.yaml)
- `plan` - Implementation plan (plan.yaml)
- `tasks` - Task breakdown (tasks.yaml)
- `analysis` - Cross-artifact analysis (analysis.yaml)
- `checklist` - Feature quality checklist (checklists/*.yaml)
- `constitution` - Project constitution (constitution.yaml)

**Flags**:
- `--schema` - Print the expected schema for an artifact type
- `--fix` - Auto-fix common issues (missing optional fields, formatting)

**Examples**:
```bash
# Path-only (preferred) - type inferred from filename
autospec artifact specs/001-feature/spec.yaml
autospec artifact specs/001-feature/plan.yaml
autospec artifact specs/001-feature/tasks.yaml
autospec artifact .autospec/memory/constitution.yaml

# Checklist requires explicit type (filename varies)
autospec artifact checklist specs/001-feature/checklists/ux.yaml

# Show schema
autospec artifact spec --schema

# Auto-fix issues
autospec artifact specs/001-feature/plan.yaml --fix
```

**Exit Codes**: 0 (valid), 1 (validation failed), 3 (invalid args)

### autospec yaml check

Validate YAML syntax

**Syntax**: `autospec yaml check <file>`

**Description**: Quick syntax validation without schema checking. Use `autospec artifact` for full schema validation.

**Examples**:
```bash
autospec yaml check specs/001-feature/spec.yaml
```

**Exit Codes**: 0 (valid syntax), 1 (syntax error)

### autospec version

Display version information

**Syntax**: `autospec version`

**Alias**: `autospec v`

**Description**: Show autospec version number and build info.

**Examples**:
```bash
autospec version
```

**Exit Codes**: 0 (success)

## CLI Agents

autospec supports multiple CLI-based AI coding agents. The `--agent` flag is available on all workflow commands to override the configured agent for a single execution.

### Available Agents

| Agent | Binary | Description |
|-------|--------|-------------|
| `claude` | `claude` | Anthropic's Claude Code CLI (default) |
| `cline` | `cline` | Cline VSCode extension CLI |
| `gemini` | `gemini` | Google Gemini CLI |
| `codex` | `codex` | OpenAI Codex CLI |
| `opencode` | `opencode` | OpenCode CLI |
| `goose` | `goose` | Goose AI CLI |

### Agent Override Examples

```bash
# Use gemini for all stages
autospec run -a "Add caching" --agent gemini

# Use cline for planning
autospec plan --agent cline

# Use codex for implementation
autospec implement --agent codex
```

### Agent Status

Check agent availability with `autospec doctor`:

```bash
$ autospec doctor

CLI Agents:
  ✓ claude: installed (v1.0.5)
  ○ cline: not found in PATH
  ✓ gemini: installed (v0.8.2)
```

See [CLI Agent Configuration](./agents.md) for detailed documentation on agent configuration, custom agents, and migration from legacy settings.

### autospec worktree

Manage git worktrees with project-aware setup automation

**Syntax**: `autospec worktree <subcommand> [flags]`

**Description**: Create and manage git worktrees with automatic copying of non-tracked directories (`.autospec/`, `.claude/`) and execution of project-specific setup scripts.

**Subcommands**:
- `create <name> --branch <branch> [--path <path>]`: Create new worktree
- `list`: List all tracked worktrees
- `remove <name> [--force]`: Remove a worktree
- `setup <path> [--track]`: Run setup on existing worktree
- `prune`: Remove stale tracking entries

**Examples**:
```bash
# Create a new worktree
autospec worktree create feature-auth --branch feat/user-auth

# Create at custom location
autospec worktree create zoom --branch feat/zoom --path /tmp/zoom-dev

# List all worktrees
autospec worktree list

# Remove (with safety checks)
autospec worktree remove feature-auth

# Force remove (bypass checks)
autospec worktree remove feature-auth --force

# Setup existing worktree
autospec worktree setup ../my-worktree --track

# Clean up stale entries
autospec worktree prune
```

**Exit Codes**: 0 (success), 1 (operation failed), 3 (invalid args)

See [docs/worktree.md](worktree.md) for detailed documentation.

## Configuration Options

Configuration sources (priority order): Environment variables > Local config > Global config > Defaults

### agent_preset

**Type**: string
**Default**: `"claude"`
**Description**: Name of the built-in agent to use for workflow execution

**Available presets**: `claude`, `cline`, `gemini`, `codex`, `opencode`, `goose`

**Example**:
```yaml
agent_preset: gemini
```

**Environment**: `AUTOSPEC_AGENT_PRESET`

See [CLI Agent Configuration](./agents.md) for detailed agent documentation.

### custom_agent_cmd

**Type**: string
**Default**: `""` (not set)
**Description**: Custom agent command template with `{{PROMPT}}` placeholder. Takes precedence over `agent_preset`.

**Example**:
```yaml
custom_agent_cmd: "my-agent run --prompt {{PROMPT}} --mode headless"
```

**Environment**: `AUTOSPEC_CUSTOM_AGENT_CMD`

### claude_cmd (deprecated)

**Type**: string
**Default**: `"claude"`
**Description**: Command to invoke Claude CLI. **Deprecated**: Use `agent_preset: claude` instead.

**Example**:
```yaml
claude_cmd: /usr/local/bin/claude
```

**Environment**: `AUTOSPEC_CLAUDE_CMD`

### max_retries

**Type**: integer
**Default**: `3`
**Range**: 1-10
**Description**: Maximum retry attempts on validation failure

**Example**:
```yaml
max_retries: 5
```

**Environment**: `AUTOSPEC_MAX_RETRIES`

### specs_dir

**Type**: string
**Default**: `"./specs"`
**Description**: Directory for feature specifications

**Example**:
```yaml
specs_dir: /path/to/specs
```

**Environment**: `AUTOSPEC_SPECS_DIR`

### state_dir

**Type**: string
**Default**: `"~/.autospec/state"`
**Description**: Directory for persistent state (retry tracking)

**Example**:
```yaml
state_dir: ~/.autospec/state
```

**Environment**: `AUTOSPEC_STATE_DIR`

### timeout

**Type**: integer
**Default**: `0` (no timeout)
**Range**: 0 or 1-604800 (7 days in seconds)
**Description**: Command execution timeout in seconds

**Example**:
```yaml
timeout: 600
```

**Environment**: `AUTOSPEC_TIMEOUT`

**Behavior**:
- `0`: No timeout (infinite wait) - backward compatible default
- `1-604800`: Timeout after specified seconds
- Commands exceeding timeout return exit code 5

### skip_preflight

**Type**: boolean
**Default**: `false`
**Description**: Skip pre-flight dependency checks

**Example**:
```yaml
skip_preflight: true
```

**Environment**: `AUTOSPEC_SKIP_PREFLIGHT`

### implement_method

**Type**: string (enum)
**Default**: `"phases"`
**Values**: `"phases"` | `"tasks"` | `"single-session"`
**Description**: Default execution method for the implement command

**Example**:
```yaml
implement_method: tasks  # Each task in separate Claude session
```

**Environment**: `AUTOSPEC_IMPLEMENT_METHOD`

**Behavior**:
- `phases`: Each phase runs in separate session (fresh context per phase) — **default**
- `tasks`: Each task runs in separate session (maximum context isolation)
- `single-session`: All tasks in single Claude session (legacy)

**Note**: CLI flags (`--phases`, `--tasks`, `--single-session`) override this config setting.

### custom_claude_cmd (deprecated)

**Type**: string
**Default**: `""` (not set)
**Description**: Custom command template with `{{PROMPT}}` placeholder. **Deprecated**: Use `custom_agent_cmd` instead.

**Example**:
```yaml
custom_claude_cmd: "claude -p {{PROMPT}} | process-output"
```

**Environment**: `AUTOSPEC_CUSTOM_CLAUDE_CMD`

### max_history_entries

**Type**: integer
**Default**: `500`
**Description**: Maximum number of command history entries to retain. Oldest entries are pruned when this limit is exceeded.

**Example**:
```yaml
max_history_entries: 1000
```

**Environment**: `AUTOSPEC_MAX_HISTORY_ENTRIES`

### view_limit

**Type**: integer
**Default**: `5`
**Description**: Number of recent specs to display in the view command dashboard

**Example**:
```yaml
view_limit: 10
```

**Environment**: `AUTOSPEC_VIEW_LIMIT`

**Note**: Can be overridden by the `--limit` flag on the `autospec view` command.

### notifications

**Type**: object
**Default**: `{ enabled: false, type: "both", ... }`
**Description**: Configuration for desktop notifications when commands complete

#### notifications.enabled

**Type**: boolean
**Default**: `false`
**Description**: Master switch for all notifications (opt-in)

**Example**:
```yaml
notifications:
  enabled: true
```

**Environment**: `AUTOSPEC_NOTIFICATIONS_ENABLED`

#### notifications.type

**Type**: string (enum)
**Default**: `"both"`
**Values**: `"sound"` | `"visual"` | `"both"`
**Description**: Type of notification to send

**Example**:
```yaml
notifications:
  enabled: true
  type: visual  # Only show desktop notification, no sound
```

**Environment**: `AUTOSPEC_NOTIFICATIONS_TYPE`

#### notifications.sound_file

**Type**: string
**Default**: `""` (uses system default)
**Description**: Custom sound file path for audio notifications

**Supported formats**: `.wav`, `.mp3`, `.aiff`, `.aif`, `.ogg`, `.flac`, `.m4a`

**Example**:
```yaml
notifications:
  enabled: true
  type: sound
  sound_file: /path/to/custom/notification.wav
```

**Environment**: `AUTOSPEC_NOTIFICATIONS_SOUND_FILE`

**Notes**:
- If the file doesn't exist, falls back to system default sound
- macOS default: `/System/Library/Sounds/Glass.aiff`
- Linux: No default sound (requires custom file)

#### notifications.on_command_complete

**Type**: boolean
**Default**: `true` (when notifications enabled)
**Description**: Notify when any autospec command finishes

**Example**:
```yaml
notifications:
  enabled: true
  on_command_complete: true
```

**Environment**: `AUTOSPEC_NOTIFICATIONS_ON_COMMAND_COMPLETE`

#### notifications.on_stage_complete

**Type**: boolean
**Default**: `false`
**Description**: Notify after each workflow stage (specify, plan, tasks, implement)

**Example**:
```yaml
notifications:
  enabled: true
  on_stage_complete: true  # Get notified after each stage
```

**Environment**: `AUTOSPEC_NOTIFICATIONS_ON_STAGE_COMPLETE`

#### notifications.on_error

**Type**: boolean
**Default**: `true` (when notifications enabled)
**Description**: Notify when a command or stage fails

**Example**:
```yaml
notifications:
  enabled: true
  on_error: true
```

**Environment**: `AUTOSPEC_NOTIFICATIONS_ON_ERROR`

#### notifications.on_long_running

**Type**: boolean
**Default**: `false`
**Description**: Only notify if command duration exceeds threshold

**Example**:
```yaml
notifications:
  enabled: true
  on_long_running: true
  long_running_threshold: 60s  # Only notify if command takes > 60 seconds
```

**Environment**: `AUTOSPEC_NOTIFICATIONS_ON_LONG_RUNNING`

#### notifications.long_running_threshold

**Type**: duration
**Default**: `30s`
**Description**: Threshold for `on_long_running` hook. Set to 0 for "always notify".

**Example**:
```yaml
notifications:
  enabled: true
  on_long_running: true
  long_running_threshold: 5m  # 5 minutes
```

**Environment**: `AUTOSPEC_NOTIFICATIONS_LONG_RUNNING_THRESHOLD`

### Full Notification Configuration Example

```yaml
# Project config: .autospec/config.yml
notifications:
  enabled: true              # Master switch - must be true
  type: both                 # "sound", "visual", or "both"
  sound_file: ""             # Optional custom sound file path
  on_command_complete: true  # Notify when command finishes
  on_stage_complete: false   # Notify after each stage
  on_error: true             # Notify on failures
  on_long_running: false     # Only notify for long commands
  long_running_threshold: 2m  # Threshold for on_long_running
```

### Hook Combinations

Hooks are composable - enable multiple to customize notification behavior:

| Use Case | Configuration |
|----------|---------------|
| Notify on completion only | `on_command_complete: true`, others: false |
| Notify on errors only | `on_error: true`, `on_command_complete: false` |
| Notify per stage | `on_stage_complete: true` |
| Notify for long tasks | `on_long_running: true`, `long_running_threshold: 60s` |
| Full notifications | All hooks enabled |

**Notes**:
- Multiple hooks can fire for the same event (e.g., command completes with error after long time)
- Each enabled hook fires independently
- Notifications are disabled automatically in CI environments
- Notifications are skipped in non-interactive sessions (no TTY)

## Exit Codes

Standardized exit codes for programmatic composition and CI/CD integration:

| Code | Meaning | Description | Action |
|------|---------|-------------|--------|
| 0 | Success | All operations completed successfully | Continue workflow |
| 1 | Validation Failed | Output artifact validation failed | Retry or inspect error |
| 2 | Retries Exhausted | Max retry limit reached without success | Reset retry state or fix issue |
| 3 | Invalid Arguments | User provided invalid command arguments | Check command syntax |
| 4 | Missing Dependencies | Required dependencies not found | Install Claude CLI or other deps |
| 5 | Command Timeout | Operation exceeded configured timeout | Increase timeout or optimize |

**Examples**:
```bash
# Check exit code in bash
autospec prep "feature"
if [ $? -eq 0 ]; then
    echo "Success"
elif [ $? -eq 2 ]; then
    echo "Retries exhausted, resetting state"
    rm ~/.autospec/state/retry.json
fi

# Use in CI/CD
autospec full "feature" || exit 1
```

## Prerequisite Validation

Before executing any stage command, autospec validates that required artifacts exist. This provides immediate feedback when prerequisites are missing, avoiding wasted API costs and time.

### Constitution Requirement

All stage commands (except `constitution` itself) require a project constitution:

| Command | Requires Constitution |
|---------|----------------------|
| `specify` | Yes |
| `plan` | Yes |
| `tasks` | Yes |
| `implement` | Yes |
| `clarify` | Yes |
| `checklist` | Yes |
| `analyze` | Yes |
| `constitution` | No (creates it) |

If constitution is missing, you'll see:
```
Error: Project constitution not found.

A constitution is required before running any workflow stages.
The constitution defines your project's principles and guidelines.

To create a constitution, run:
  autospec constitution
```

### Artifact Prerequisites

Each command validates that its required artifacts exist in the spec directory:

| Command | Required Artifacts | Remediation |
|---------|-------------------|-------------|
| `specify` | (none) | - |
| `plan` | `spec.yaml` | Run `autospec specify` first |
| `tasks` | `plan.yaml` | Run `autospec plan` first |
| `implement` | `tasks.yaml` | Run `autospec tasks` first |
| `clarify` | `spec.yaml` | Run `autospec specify` first |
| `checklist` | `spec.yaml` | Run `autospec specify` first |
| `analyze` | `spec.yaml`, `plan.yaml`, `tasks.yaml` | Run missing stages first |

**Example error**:
```
Error: spec.yaml not found.

Run 'autospec specify' first to create this file.
```

### Run Command Smart Validation

The `run` command performs "smart" validation - it only checks for artifacts that won't be produced by earlier selected stages:

| Flags | Validates | Reason |
|-------|-----------|--------|
| `-spt` | constitution only | `specify` produces `spec.yaml`, `plan` produces `plan.yaml` |
| `-pti` | `spec.yaml` | `plan` needs `spec.yaml`, but produces `plan.yaml`; `tasks` produces `tasks.yaml` |
| `-ti` | `plan.yaml` | `tasks` needs `plan.yaml`, produces `tasks.yaml` |
| `-i` | `tasks.yaml` | `implement` needs `tasks.yaml` |
| `-a` | constitution only | Full chain (`-spti`) produces all intermediate artifacts |

This allows running `autospec run -spt` without having `spec.yaml` present, since `specify` will create it.

### Exit Code for Missing Prerequisites

Missing prerequisites return exit code **3** (`ExitInvalidArguments`), the same code used for other argument validation failures.

```bash
# Check if prerequisite validation failed
autospec plan
if [ $? -eq 3 ]; then
    echo "Missing prerequisites - run autospec specify first"
fi
```

## File Locations

### Configuration Files

| File | Purpose | Priority |
|------|---------|----------|
| `~/.config/autospec/config.yml` | Global configuration (XDG compliant) | 3 (after env, local) |
| `.autospec/config.yml` | Local project configuration | 2 (after env) |

### State Files

| File | Purpose |
|------|---------|
| `~/.autospec/state/retry.json` | Persistent retry state tracking |
| `~/.autospec/state/history.yaml` | Command execution history log |

### Specification Directories

| Directory | Purpose |
|-----------|---------|
| `./specs/` | Feature specifications (default) |
| `./specs/NNN-feature-name/` | Individual feature directory |
| `./specs/NNN-feature-name/spec.yaml` | Feature specification |
| `./specs/NNN-feature-name/plan.yaml` | Technical plan |
| `./specs/NNN-feature-name/tasks.yaml` | Task breakdown |

**Naming Convention**: `NNN-feature-name` where NNN is a 3-digit number (e.g., `001-dark-mode`, `042-api-auth`)

## Advanced Patterns

### Prompt Injection

All phase commands support optional guidance text to direct Claude's execution:

```bash
# Plan with specific focus
autospec plan "Prioritize security and performance"

# Tasks with specific constraints
autospec tasks "Break into very small incremental steps"

# Implement with specific guidance
autospec implement "Focus on completing tests first"
autospec implement 003-feature "Document all public APIs"
```

**How It Works**:
- Guidance text appended to slash command
- Full command displayed before execution
- Works with custom commands using `{{PROMPT}}` placeholder

### Custom Command Templates

Use `custom_claude_cmd` for complex pipelines:

```yaml
custom_claude_cmd: "claude -p {{PROMPT}} | tee logs/$(date +%s).log | grep -v DEBUG"
```

**Placeholders**:
- `{{PROMPT}}`: Replaced with actual prompt (e.g., `/autospec.plan "focus on security"`)

### Retry State Management

Manually inspect or reset retry state:

```bash
# View retry state
cat ~/.autospec/state/retry.json

# Reset retry state for specific spec:phase
jq 'del(.retries["001-feature:specify"])' ~/.autospec/state/retry.json > tmp && mv tmp ~/.autospec/state/retry.json

# Reset all retry state
rm ~/.autospec/state/retry.json
```

### CI/CD Integration

Use exit codes for automated workflows:

```yaml
# GitHub Actions example
- name: Run autospec prep
  run: |
    autospec prep "feature" || exit 1

- name: Check status
  run: autospec status
```

### Timeout Configuration

Configure different timeouts for different operations:

```bash
# Short timeout for quick operations
AUTOSPEC_TIMEOUT=60 autospec doctor

# Long timeout for complex workflows
AUTOSPEC_TIMEOUT=3600 autospec full "complex feature"

# No timeout (default)
AUTOSPEC_TIMEOUT=0 autospec prep "feature"
```

## Further Reading

- **[Quick Start Guide](./quickstart.md)**: Installation and first workflow
- **[Architecture Overview](./architecture.md)**: System design and components
- **[CLI Agent Configuration](./agents.md)**: Multi-agent support and custom agents
- **[Troubleshooting](./troubleshooting.md)**: Common issues and solutions
