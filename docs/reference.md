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

**Description**: Execute tasks with Claude's assistance, validating progress.

**Flags**: Same as `autospec full`

**Examples**:
```bash
autospec implement
autospec implement 001-dark-mode
autospec implement "Focus on documentation tasks"
autospec implement 001-dark-mode "Complete tests first"
```

**Exit Codes**: 0 (success), 1 (validation failed), 2 (retries exhausted), 3 (invalid args), 4 (missing deps), 5 (timeout)

### autospec doctor

Run health checks and verify dependencies

**Syntax**: `autospec doctor [flags]`

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

**Syntax**: `autospec status [flags]`

**Description**: Display detected spec, phase progress, and retry state.

**Flags**: None (uses global flags only)

**Examples**:
```bash
autospec status
autospec status --verbose
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

**Description**: Create `~/.config/autospec/config.yml` with default settings.

**Flags**: `--force`: Overwrite existing configuration

**Examples**:
```bash
autospec init
autospec init --force
```

**Exit Codes**: 0 (success)

### autospec version

Display version information

**Syntax**: `autospec version`

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

### custom_claude_cmd

**Type**: string
**Default**: `""` (not set)
**Description**: Custom command template with `{{PROMPT}}` placeholder

**Example**:
```yaml
custom_claude_cmd: "claude -p {{PROMPT}} | process-output"
```

**Environment**: `AUTOSPEC_CUSTOM_CLAUDE_CMD`

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
