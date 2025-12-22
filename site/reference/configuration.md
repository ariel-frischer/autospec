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

### use_subscription

Force Claude to use subscription (Pro/Max) instead of API credits.

| Property | Value |
|:---------|:------|
| Type | boolean |
| Default | `true` |
| Environment | `AUTOSPEC_USE_SUBSCRIPTION` |

```yaml
# Default: use subscription mode (recommended)
use_subscription: true

# Disable to use API credits instead
use_subscription: false
```

When enabled (default), `ANTHROPIC_API_KEY` is set to empty at execution time. This prevents accidental API charges when you have an API key set in your shell for other purposes.

{: .note }
> Set to `false` only if you specifically want to use API billing. Ensure `ANTHROPIC_API_KEY` is set in your shell environment when using API mode.

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

### view_limit

Number of recent specs to display in the view command.

| Property | Value |
|:---------|:------|
| Type | integer |
| Default | `5` |
| Environment | `AUTOSPEC_VIEW_LIMIT` |

```yaml
view_limit: 10
```

---

### skip_confirmations

Skip confirmation prompts for destructive operations.

| Property | Value |
|:---------|:------|
| Type | boolean |
| Default | `false` |
| Environment | `AUTOSPEC_YES` |

```yaml
skip_confirmations: true
```

Can also be enabled via the `AUTOSPEC_YES` environment variable.

---

### auto_commit

Enable automatic git commit creation after workflow completion.

| Property | Value |
|:---------|:------|
| Type | boolean |
| Default | `false` |
| Environment | `AUTOSPEC_AUTO_COMMIT` |

```yaml
auto_commit: true   # Enable auto-commit
auto_commit: false  # Disable auto-commit (default)
```

When enabled, instructions are injected into the agent prompt to:
- Update `.gitignore` with common ignorable patterns
- Stage appropriate files for version control
- Create a commit with conventional commit message format

---

### default_agents

Agents to pre-select in `autospec init` prompts.

| Property | Value |
|:---------|:------|
| Type | string array |
| Default | `[]` |
| Environment | `AUTOSPEC_DEFAULT_AGENTS` (comma-separated) |

```yaml
default_agents:
  - claude
  - gemini
```

Saved from previous `autospec init` selections. Used to pre-populate agent selection in future init prompts.

---

### output_style

Controls how stream-json output is formatted for display.

| Property | Value |
|:---------|:------|
| Type | enum |
| Default | `"default"` |
| Values | `"default"`, `"compact"`, `"minimal"`, `"plain"`, `"raw"` |
| Environment | `AUTOSPEC_OUTPUT_STYLE` |

```yaml
output_style: default   # Box-drawing characters with colors
output_style: compact   # Condensed output
output_style: minimal   # Bare minimum output
output_style: plain     # No formatting
output_style: raw       # Raw JSON output
```

---

## Worktree Management

Configure git worktree creation and management.

### worktree.base_dir

Parent directory for new worktrees.

| Property | Value |
|:---------|:------|
| Type | string |
| Default | `""` (parent of repo) |

```yaml
worktree:
  base_dir: /path/to/worktrees
```

---

### worktree.prefix

Directory name prefix for worktrees.

| Property | Value |
|:---------|:------|
| Type | string |
| Default | `""` |

```yaml
worktree:
  prefix: "wt-"
```

---

### worktree.setup_script

Path to setup script relative to repo root.

| Property | Value |
|:---------|:------|
| Type | string |
| Default | `""` |

```yaml
worktree:
  setup_script: scripts/worktree-setup.sh
```

---

### worktree.auto_setup

Run setup script automatically on worktree creation.

| Property | Value |
|:---------|:------|
| Type | boolean |
| Default | `true` |

```yaml
worktree:
  auto_setup: true
```

---

### worktree.track_status

Persist worktree state for status tracking.

| Property | Value |
|:---------|:------|
| Type | boolean |
| Default | `true` |

```yaml
worktree:
  track_status: true
```

---

### worktree.copy_dirs

Non-tracked directories to copy to new worktrees.

| Property | Value |
|:---------|:------|
| Type | string array |
| Default | `[".autospec", ".claude"]` |

```yaml
worktree:
  copy_dirs:
    - .autospec
    - .claude
    - node_modules
```

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

## Security: Sandbox & Permissions
{: #security-sandbox--permissions }

autospec runs Claude Code with `--dangerously-skip-permissions` by default. This section explains why and how to stay secure.

### Why This Flag is Used

Without `--dangerously-skip-permissions`, Claude requires manual approval for:
- Every file edit
- Every shell command
- Every tool invocation

This makes automated workflows impractical. Managing allow/deny rules for all necessary operations is complex and error-prone.

### Two Separate Security Layers

| Layer | What it does |
|:------|:-------------|
| **Sandbox** | OS-level isolation - restricts filesystem to project directory |
| **Permission prompts** | User approval for actions (skipped with `--dangerously-skip-permissions`) |

**Key insight**: `--dangerously-skip-permissions` only skips the permission prompts—it does **not** bypass sandbox restrictions. When sandbox is enabled, Claude cannot access files outside your project directory.

### Recommended Setup: Sandbox Enabled

During `autospec init`, you're prompted to enable sandbox. This configures `.claude/settings.local.json`:

```json
{
  "sandbox": {
    "enabled": true,
    "autoAllowBashIfSandboxed": true,
    "additionalAllowWritePaths": [
      ".autospec",
      "specs"
    ]
  }
}
```

This provides **sandboxed automation**: unattended execution with OS-level filesystem isolation to your project directory. Note that Claude still has full access to modify any file within the project.

{: .warning }
> Without sandbox enabled, `--dangerously-skip-permissions` gives Claude full system access. Only use without sandbox in isolated environments (containers, VMs).

### First-Run Security Notice

On your first workflow command, autospec displays a one-time notice explaining the security model and showing your sandbox status. Suppress with:

```bash
autospec config set skip_permissions_notice_shown true
```

Or via environment variable:

```bash
export AUTOSPEC_SKIP_PERMISSIONS_NOTICE=1
```

### Custom Agent Configuration

To customize the Claude command (e.g., add output formatting), use `custom_agent`:

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

---

## Full Configuration Example

```yaml
# .autospec/config.yml

# Core settings
agent_preset: claude
use_subscription: true  # Use Claude subscription, not API credits
max_retries: 3
specs_dir: ./specs
state_dir: ~/.autospec/state
timeout: 2400
skip_preflight: false
skip_confirmations: false
implement_method: phases
auto_commit: false
max_history_entries: 500
view_limit: 5
output_style: default

# Worktree management
worktree:
  base_dir: ""
  prefix: ""
  setup_script: ""
  auto_setup: true
  track_status: true
  copy_dirs:
    - .autospec
    - .claude

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
| `AUTOSPEC_USE_SUBSCRIPTION` | `use_subscription` |
| `AUTOSPEC_MAX_RETRIES` | `max_retries` |
| `AUTOSPEC_SPECS_DIR` | `specs_dir` |
| `AUTOSPEC_STATE_DIR` | `state_dir` |
| `AUTOSPEC_TIMEOUT` | `timeout` |
| `AUTOSPEC_SKIP_PREFLIGHT` | `skip_preflight` |
| `AUTOSPEC_YES` | `skip_confirmations` |
| `AUTOSPEC_IMPLEMENT_METHOD` | `implement_method` |
| `AUTOSPEC_AUTO_COMMIT` | `auto_commit` |
| `AUTOSPEC_CUSTOM_AGENT_CMD` | `custom_agent_cmd` |
| `AUTOSPEC_MAX_HISTORY_ENTRIES` | `max_history_entries` |
| `AUTOSPEC_VIEW_LIMIT` | `view_limit` |
| `AUTOSPEC_DEFAULT_AGENTS` | `default_agents` |
| `AUTOSPEC_OUTPUT_STYLE` | `output_style` |
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
