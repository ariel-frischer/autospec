---
layout: default
title: Configuration
parent: Reference
nav_order: 2
---

# Configuration
{: .no_toc }

Configuration options, file locations, and environment variables.
{: .fs-6 .fw-300 }

<details open markdown="block">
  <summary>
    Table of contents
  </summary>
  {: .text-delta }
1. TOC
{:toc}
</details>

---

## Configuration Priority

Configuration is loaded from multiple sources in priority order:

1. **Environment variables** (`AUTOSPEC_*`) - highest priority
2. **Project config** (`.autospec/config.yml`)
3. **User config** (`~/.config/autospec/config.yml`)
4. **Defaults** - lowest priority

Higher priority sources override lower priority sources.

---

## Core Options

### agent_preset

Preset agent to use for execution.

| Property | Value |
|:---------|:------|
| Type | string |
| Default | `""` |
| Environment | `AUTOSPEC_AGENT_PRESET` |

```yaml
agent_preset: claude
```

---

### max_retries

Maximum retry attempts on validation failure.

| Property | Value |
|:---------|:------|
| Type | integer |
| Default | `3` |
| Range | 1-10 |
| Environment | `AUTOSPEC_MAX_RETRIES` |

```yaml
max_retries: 5
```

---

### specs_dir

Directory for feature specifications.

| Property | Value |
|:---------|:------|
| Type | string |
| Default | `"./specs"` |
| Environment | `AUTOSPEC_SPECS_DIR` |

```yaml
specs_dir: /path/to/specs
```

---

### state_dir

Directory for persistent state (retry tracking, history).

| Property | Value |
|:---------|:------|
| Type | string |
| Default | `"~/.autospec/state"` |
| Environment | `AUTOSPEC_STATE_DIR` |

```yaml
state_dir: ~/.autospec/state
```

---

### timeout

Command execution timeout in seconds.

| Property | Value |
|:---------|:------|
| Type | integer |
| Default | `0` (no timeout) |
| Range | 0 or 1-604800 (7 days) |
| Environment | `AUTOSPEC_TIMEOUT` |

```yaml
timeout: 600  # 10 minutes
```

**Behavior:**
- `0`: No timeout (infinite wait)
- `1-604800`: Timeout after specified seconds
- Commands exceeding timeout return exit code 5

---

### skip_preflight

Skip pre-flight dependency checks.

| Property | Value |
|:---------|:------|
| Type | boolean |
| Default | `false` |
| Environment | `AUTOSPEC_SKIP_PREFLIGHT` |

```yaml
skip_preflight: true
```

---

### implement_method

Default execution method for implement command.

| Property | Value |
|:---------|:------|
| Type | enum |
| Default | `"phases"` |
| Values | `"phases"`, `"tasks"`, `"single-session"` |
| Environment | `AUTOSPEC_IMPLEMENT_METHOD` |

```yaml
implement_method: tasks
```

**Behavior:**

| Value | Sessions | Description |
|:------|:---------|:------------|
| `phases` | 1 per phase | Fresh context per phase (default) |
| `tasks` | 1 per task | Maximum context isolation |
| `single-session` | 1 total | All tasks in one session |

CLI flags (`--phases`, `--tasks`, `--single-session`) override this setting.

---

### custom_agent

Custom agent configuration with command and args.

| Property | Value |
|:---------|:------|
| Type | object |
| Default | `null` |

```yaml
custom_agent:
  command: sh
  args:
    - -c
    - "claude -p {{PROMPT}} | tee logs/$(date +%s).log"
```

The `{{PROMPT}}` placeholder is replaced with the actual prompt.

---

### max_history_entries

Maximum command history entries to retain.

| Property | Value |
|:---------|:------|
| Type | integer |
| Default | `500` |
| Environment | `AUTOSPEC_MAX_HISTORY_ENTRIES` |

```yaml
max_history_entries: 1000
```

Oldest entries are removed when the limit is exceeded.

---

## Notifications

Configure desktop notifications when commands complete.

### notifications.enabled

Master switch for all notifications.

| Property | Value |
|:---------|:------|
| Type | boolean |
| Default | `false` |
| Environment | `AUTOSPEC_NOTIFICATIONS_ENABLED` |

```yaml
notifications:
  enabled: true
```

---

### notifications.type

Type of notification to send.

| Property | Value |
|:---------|:------|
| Type | enum |
| Default | `"both"` |
| Values | `"sound"`, `"visual"`, `"both"` |
| Environment | `AUTOSPEC_NOTIFICATIONS_TYPE` |

```yaml
notifications:
  enabled: true
  type: visual
```

---

### notifications.sound_file

Custom sound file for audio notifications.

| Property | Value |
|:---------|:------|
| Type | string |
| Default | `""` (system default) |
| Supported | `.wav`, `.mp3`, `.aiff`, `.ogg`, `.flac`, `.m4a` |
| Environment | `AUTOSPEC_NOTIFICATIONS_SOUND_FILE` |

```yaml
notifications:
  enabled: true
  type: sound
  sound_file: /path/to/notification.wav
```

**Defaults:**
- macOS: `/System/Library/Sounds/Glass.aiff`
- Linux: No default (requires custom file)

---

### notifications.on_command_complete

Notify when any command finishes.

| Property | Value |
|:---------|:------|
| Type | boolean |
| Default | `true` (when enabled) |
| Environment | `AUTOSPEC_NOTIFICATIONS_ON_COMMAND_COMPLETE` |

---

### notifications.on_stage_complete

Notify after each workflow stage.

| Property | Value |
|:---------|:------|
| Type | boolean |
| Default | `false` |
| Environment | `AUTOSPEC_NOTIFICATIONS_ON_STAGE_COMPLETE` |

---

### notifications.on_error

Notify when a command fails.

| Property | Value |
|:---------|:------|
| Type | boolean |
| Default | `true` (when enabled) |
| Environment | `AUTOSPEC_NOTIFICATIONS_ON_ERROR` |

---

### notifications.on_long_running

Notify only for commands exceeding threshold.

| Property | Value |
|:---------|:------|
| Type | boolean |
| Default | `false` |
| Environment | `AUTOSPEC_NOTIFICATIONS_ON_LONG_RUNNING` |

---

### notifications.long_running_threshold

Threshold for long-running notifications.

| Property | Value |
|:---------|:------|
| Type | duration |
| Default | `30s` |
| Environment | `AUTOSPEC_NOTIFICATIONS_LONG_RUNNING_THRESHOLD` |

```yaml
notifications:
  enabled: true
  on_long_running: true
  long_running_threshold: 5m
```

---

## Full Automation Setup

For unattended autospec execution with Claude Code, use the `custom_agent` configuration:

```yaml
# ~/.config/autospec/config.yml
custom_agent:
  command: "claude"
  args:
    - "-p"
    - "--dangerously-skip-permissions"
    - "--verbose"
    - "--output-format"
    - "stream-json"
    - "{{PROMPT}}"
  post_processor: "cclean"
```

### Sandbox + Skip-Permissions (Recommended)

These are **two separate security layers**:

| Layer | What it does |
|:------|:-------------|
| **Sandbox** | OS-level isolation - restricts filesystem to CWD |
| **Permission prompts** | User approval for actions (skipped with flag) |

**Key insight**: `--dangerously-skip-permissions` skips prompts but does **not** bypass sandbox. When sandbox is enabled, Claude cannot edit files outside your project directory.

Enable sandbox in `.claude/settings.local.json`:

```json
{
  "sandbox": {
    "enabled": true,
    "autoAllowBashIfSandboxed": true
  }
}
```

This provides safe automation: unattended execution with OS-level filesystem protection.

{: .warning }
> Without sandbox enabled, `--dangerously-skip-permissions` gives Claude full system access. Only use this flag with sandbox enabled, or in isolated environments.

---

## Full Configuration Example

```yaml
# .autospec/config.yml

# Core settings
claude_cmd: claude
max_retries: 3
specs_dir: ./specs
state_dir: ~/.autospec/state
timeout: 0
skip_preflight: false
implement_method: phases
max_history_entries: 500

# Notifications
notifications:
  enabled: true
  type: both
  sound_file: ""
  on_command_complete: true
  on_stage_complete: false
  on_error: true
  on_long_running: false
  long_running_threshold: 2m
```

---

## File Locations

### Configuration Files

| File | Purpose | Priority |
|:-----|:--------|:---------|
| `.autospec/config.yml` | Project config | 2 |
| `~/.config/autospec/config.yml` | User config (XDG compliant) | 3 |

### State Files

| File | Purpose |
|:-----|:--------|
| `~/.autospec/state/retry.json` | Retry state tracking |
| `~/.autospec/state/history.yaml` | Command execution history |

### Specification Files

| Pattern | Purpose |
|:--------|:--------|
| `specs/NNN-name/` | Feature directory |
| `specs/NNN-name/spec.yaml` | Feature specification |
| `specs/NNN-name/plan.yaml` | Implementation plan |
| `specs/NNN-name/tasks.yaml` | Task breakdown |

**Naming Convention:** `NNN-feature-name` where NNN is a 3-digit number (e.g., `001-dark-mode`, `042-api-auth`)

---

## Environment Variables

All configuration options can be set via environment variables with the `AUTOSPEC_` prefix:

| Variable | Config Key |
|:---------|:-----------|
| `AUTOSPEC_AGENT_PRESET` | `agent_preset` |
| `AUTOSPEC_MAX_RETRIES` | `max_retries` |
| `AUTOSPEC_SPECS_DIR` | `specs_dir` |
| `AUTOSPEC_STATE_DIR` | `state_dir` |
| `AUTOSPEC_TIMEOUT` | `timeout` |
| `AUTOSPEC_SKIP_PREFLIGHT` | `skip_preflight` |
| `AUTOSPEC_IMPLEMENT_METHOD` | `implement_method` |
| `AUTOSPEC_CUSTOM_AGENT_CMD` | `custom_agent_cmd` |
| `AUTOSPEC_MAX_HISTORY_ENTRIES` | `max_history_entries` |
| `AUTOSPEC_NOTIFICATIONS_ENABLED` | `notifications.enabled` |
| `AUTOSPEC_NOTIFICATIONS_TYPE` | `notifications.type` |
| `AUTOSPEC_NOTIFICATIONS_SOUND_FILE` | `notifications.sound_file` |

**Example:**

```bash
export AUTOSPEC_TIMEOUT=600
export AUTOSPEC_MAX_RETRIES=5
autospec run -a "Add feature"
```

---

## Notification Combinations

Enable multiple hooks to customize behavior:

| Use Case | Configuration |
|:---------|:--------------|
| Completion only | `on_command_complete: true`, others: false |
| Errors only | `on_error: true`, `on_command_complete: false` |
| Per stage | `on_stage_complete: true` |
| Long tasks | `on_long_running: true`, `long_running_threshold: 60s` |
| Full | All hooks enabled |

**Notes:**
- Multiple hooks can fire for the same event
- Notifications disabled in CI environments
- Notifications skipped in non-interactive sessions

---

## See Also

- [CLI Commands](cli) - Complete command reference with flags and examples
- [YAML Schemas](yaml-schemas) - Artifact structure and validation rules
- [Troubleshooting](/autospec/guides/troubleshooting) - Configuration issues and solutions
- [Architecture Internals](/autospec/architecture/internals) - Spec detection and retry systems
