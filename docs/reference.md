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

**Examples**:
```bash
autospec full "Add user authentication with OAuth"
autospec full "Add dark mode toggle" --timeout 600
autospec full "Export data to CSV" --skip-preflight
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
- Plus all flags from `autospec full`

**Execution Modes**:

| Mode | Flag | Sessions | Use Case |
|------|------|----------|----------|
| Phase-level | (default) | 1 per phase | Balanced cost/context |
| Task-level | `--tasks` | 1 per task | Large specs, maximum isolation |
| Single-session | config only | 1 | Small specs, quick iterations |

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

### autospec status

Check current feature status and progress

**Syntax**: `autospec status [spec-name] [flags]`

**Alias**: `autospec st`

**Description**: Display detected spec, which artifact files exist (spec.yaml, plan.yaml, tasks.yaml), and task completion progress.

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
  25/38 tasks completed (66%)
  7/10 task phases completed
  (1 in progress)
```

**Exit Codes**: 0 (success), 3 (invalid args)

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

## Configuration Options

Configuration sources (priority order): Environment variables > Local config > Global config > Defaults

### claude_cmd

**Type**: string
**Default**: `"claude"`
**Description**: Command to invoke Claude CLI

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

**Note**: CLI flags (`--phases`, `--tasks`) override this config setting.

### custom_claude_cmd

**Type**: string
**Default**: `""` (not set)
**Description**: Custom command template with `{{PROMPT}}` placeholder

**Example**:
```yaml
custom_claude_cmd: "claude -p {{PROMPT}} | process-output"
```

**Environment**: `AUTOSPEC_CUSTOM_CLAUDE_CMD`

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
- Windows: System beep when no custom file

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
  long_running_threshold: 30s # Threshold for on_long_running
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
- **[Troubleshooting](./troubleshooting.md)**: Common issues and solutions
