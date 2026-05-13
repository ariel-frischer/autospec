---
layout: default
title: CLI Commands
parent: Reference
nav_order: 1
---

# CLI Commands
{: .no_toc }

Complete reference for all autospec commands.
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

## Global Flags

All commands support these global flags:

| Flag | Description |
|:-----|:------------|
| `--config` | Path to config file |
| `--specs-dir` | Directory for specifications |
| `--debug` | Enable debug output |
| `--verbose` | Enable verbose output |

---

## Workflow Commands

### autospec run

Execute workflow stages (alias: `autospec all`).

```bash
autospec run [flags] ["description"] ["guidance"]
```

**Stage Flags** (select which stages to run):

| Flag | Stage | Description |
|:-----|:------|:------------|
| `-s` | specify | Generate spec.yaml from description |
| `-p` | plan | Generate plan.yaml from spec |
| `-t` | tasks | Generate tasks.yaml from plan |
| `-i` | implement | Execute tasks |
| `-a` | all | Run all stages (specify + plan + tasks + implement) |

**Other Flags:**

| Flag | Description |
|:-----|:------------|
| `--skip-preflight` | Skip dependency health checks |
| `--timeout <seconds>` | Command timeout (0=infinite) |
| `--max-retries <count>` | Maximum retry attempts (1-10) |

**Examples:**

```bash
# Full workflow
autospec run -a "Add user authentication"

# Planning only (no implementation)
autospec run -spt "Add dark mode"

# Implementation only (after manual review)
autospec run -i

# With guidance
autospec run -a "Add caching" "Focus on Redis integration"
```

---

### autospec prep

Planning only: specify, plan, tasks (no implementation).

```bash
autospec prep "description" [flags]
```

Equivalent to `autospec run -spt`.

**Examples:**

```bash
autospec prep "Add user profile page"
autospec prep "Implement caching" --max-retries 5
```

---

## Stage Commands

### autospec specify

Create feature specification from description.

```bash
autospec specify "description" ["guidance"] [flags]
```

**Aliases:** `autospec spec`, `autospec s`

**Creates:** `specs/<branch>/spec.yaml`

**Examples:**

```bash
autospec specify "Add real-time notifications"
autospec specify "Add rate limiting" "Focus on security"
```

---

### autospec plan

Generate implementation plan from spec.yaml.

```bash
autospec plan ["guidance"] [flags]
```

**Alias:** `autospec p`

**Requires:** `spec.yaml`

**Creates:** `plan.yaml`

**Examples:**

```bash
autospec plan
autospec plan "Prioritize performance"
```

---

### autospec tasks

Generate task breakdown from plan.yaml.

```bash
autospec tasks ["guidance"] [flags]
```

**Alias:** `autospec t`

**Requires:** `plan.yaml`

**Creates:** `tasks.yaml`

**Examples:**

```bash
autospec tasks
autospec tasks "Break into small steps"
```

---

### autospec implement

Execute tasks from tasks.yaml.

```bash
autospec implement [spec-name] ["guidance"] [flags]
```

**Aliases:** `autospec impl`, `autospec i`

**Requires:** `tasks.yaml`

**Execution Mode Flags:**

| Flag | Sessions | Description |
|:-----|:---------|:------------|
| (default) | 1 per phase | Balanced cost/context |
| `--tasks` | 1 per task | Maximum isolation |
| `--single-session` | 1 total | All tasks in one session |

**Phase Selection:**

| Flag | Description |
|:-----|:------------|
| `--phase <N>` | Run only phase N |
| `--from-phase <N>` | Run phases N and onwards |
| `--from-task <ID>` | Resume from specific task |

**Examples:**

```bash
# Default: one session per phase
autospec implement

# Run only phase 2
autospec implement --phase 2

# Resume from phase 3
autospec implement --from-phase 3

# Maximum context isolation
autospec implement --tasks

# Resume from specific task
autospec implement --from-task T005

# With guidance
autospec implement "Focus on tests first"
```

---

## Status Commands

### autospec status

Check current spec status and progress.

```bash
autospec status [spec-name] [flags]
```

**Alias:** `autospec st`

**Flags:**

| Flag | Description |
|:-----|:------------|
| `-v, --verbose` | Show phase-by-phase breakdown |

**Output:**

```
015-artifact-validation
  artifacts: [spec.yaml plan.yaml tasks.yaml]
  risks: 3 total (1 high, 2 medium)
  25/38 tasks completed (66%)
  7/10 task phases completed
  (1 in progress)
```

**Examples:**

```bash
autospec st
autospec st -v
autospec status 003-feature
```

---

### autospec history

View command execution history.

```bash
autospec history [flags]
```

**Flags:**

| Flag | Description |
|:-----|:------------|
| `-s, --spec <name>` | Filter by spec name |
| `-n, --limit <N>` | Show last N entries |
| `--status <value>` | Filter by status |
| `--clear` | Clear all history |

**Status Values:** `running`, `completed`, `failed`, `cancelled`

**Output:**

```
TIMESTAMP            ID                         STATUS      COMMAND    SPEC      EXIT  DURATION
2024-01-15 10:30:00  brave_fox_20240115_103000  completed   specify    -         0     2m30s
2024-01-15 10:35:00  calm_river_20240115_103500 completed   plan       001-feat  0     1m15s
```

**Examples:**

```bash
autospec history
autospec history -n 10
autospec history --status failed
autospec history --spec 001-feature
autospec history --clear
```

---

## Utility Commands

### autospec doctor

Verify dependencies and configuration.

```bash
autospec doctor [flags]
```

**Alias:** `autospec doc`

Checks configured CLI agents, authentication/configuration, and directory access. When `.autospec/init.yml` indicates global scope was used during init, doctor checks global agent settings instead of project-level ones.

---

### autospec config

Manage configuration.

```bash
autospec config <subcommand> [flags]
```

**Subcommands:**

| Command | Description |
|:--------|:------------|
| `show` | Display current configuration |
| `set <key> <value>` | Set configuration value |
| `get <key>` | Get configuration value |
| `init` | Initialize default configuration |

**Examples:**

```bash
autospec config show
autospec config set max_retries 5
autospec config get timeout
```

---

### autospec init

Initialize configuration files. Creates `.autospec/init.yml` to track initialization settings (scope, agent, version).

```bash
autospec init [flags]
```

**Flags:**

| Flag | Description |
|:-----|:------------|
| `-p, --project` | Create project config (`.autospec/config.yml`) |
| `-f, --force` | Overwrite existing config |
| `--ai <agents>` | Configure specific agent(s), comma-separated (e.g., `claude,codex,opencode`) |
| `--sandbox / --no-sandbox` | Enable/skip Claude sandbox configuration |
| `--use-subscription / --no-use-subscription` | Use subscription billing vs API key |
| `--skip-permissions / --no-skip-permissions` | Enable/disable autonomous mode |
| `--gitignore / --no-gitignore` | Add/skip adding .autospec/ to .gitignore |
| `--constitution / --no-constitution` | Create/skip project constitution |

**Examples:**

```bash
autospec init
autospec init --project
autospec init --force

# Non-interactive CI/CD setup
autospec init --ai claude --sandbox --no-use-subscription --skip-permissions --gitignore --constitution
```

---

### autospec version

Display version information.

```bash
autospec version
```

**Alias:** `autospec v`

---

## Validation Commands

### autospec artifact

Validate YAML artifacts against schemas.

```bash
autospec artifact <path>
autospec artifact <type> <path>
```

**Types:** `spec`, `plan`, `tasks`, `analysis`, `checklist`, `constitution`

**Flags:**

| Flag | Description |
|:-----|:------------|
| `--schema` | Print expected schema |
| `--fix` | Auto-fix common issues |

**Examples:**

```bash
# Type inferred from filename
autospec artifact specs/001-feature/spec.yaml
autospec artifact specs/001-feature/plan.yaml

# Explicit type (required for checklists)
autospec artifact checklist specs/001/checklists/ux.yaml

# Show schema
autospec artifact spec --schema

# Auto-fix
autospec artifact specs/001/plan.yaml --fix
```

---

### autospec yaml check

Validate YAML syntax (no schema checking).

```bash
autospec yaml check <file>
```

**Examples:**

```bash
autospec yaml check specs/001-feature/spec.yaml
```

---

### autospec update-task

Update task status in tasks.yaml.

```bash
autospec update-task <task-id> <status>
```

**Status Values:** `Pending`, `InProgress`, `Completed`, `Blocked`

**Examples:**

```bash
autospec update-task T001 InProgress
autospec update-task T001 Completed
autospec update-task T015 Blocked
```

---

## Exit Codes

| Code | Meaning | Action |
|:-----|:--------|:-------|
| 0 | Success | Continue workflow |
| 1 | Validation failed | Retry or inspect error |
| 2 | Retries exhausted | Reset state or fix issue |
| 3 | Invalid arguments | Check command syntax |
| 4 | Missing dependencies | Install required tools |
| 5 | Timeout | Increase timeout |

**Bash Example:**

```bash
autospec prep "feature"
if [ $? -eq 0 ]; then
    echo "Success"
elif [ $? -eq 2 ]; then
    echo "Retries exhausted"
    rm ~/.autospec/state/retry.json
fi
```

---

## Prerequisite Validation

Commands validate required artifacts before execution:

| Command | Required |
|:--------|:---------|
| `specify` | (none) |
| `plan` | spec.yaml |
| `tasks` | plan.yaml |
| `implement` | tasks.yaml |
| `clarify` | spec.yaml |
| `analyze` | spec.yaml, plan.yaml, tasks.yaml |

**Missing prerequisite error:**

```
Error: spec.yaml not found.

Run 'autospec specify' first to create this file.
```

All stage commands also require a project constitution (`.autospec/constitution.yaml`). Run `autospec constitution` to create one.

---

## See Also

- [Configuration Reference](configuration) - Environment variables and config file options
- [YAML Schemas](yaml-schemas) - Artifact structure and validation rules
- [Quickstart Guide](/autospec/quickstart) - Get started in 5 minutes
- [Troubleshooting](/autospec/guides/troubleshooting) - Common issues and solutions
