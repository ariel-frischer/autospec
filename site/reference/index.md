---
layout: default
title: Reference
nav_order: 3
has_children: true
permalink: /reference/
---

# Reference
{: .no_toc }

Complete reference documentation for autospec commands, configuration, and YAML schemas.
{: .fs-6 .fw-300 }

---

## Overview

This reference section provides detailed documentation for:

- **[CLI Commands](cli.html)** - All autospec commands with syntax, flags, and examples
- **[Configuration](configuration.html)** - Configuration options, file locations, and environment variables
- **[YAML Schemas](yaml-schemas.html)** - Structure and validation rules for spec.yaml, plan.yaml, and tasks.yaml
- **[Agent Configuration](agents.html)** - Configure Claude, Codex, OpenCode, and custom agents
- **[Claude Settings](claude-settings.html)** - Sandboxing, permissions, and Claude Code configuration
- **[Codex Settings](codex-settings.html)** - Codex CLI auth, sandboxing, and yolo mode
- **[Shell Completion](shell-completion.html)** - Tab completion setup for bash, zsh, and fish
- **[Timeout Configuration](timeout.html)** - Command timeout settings and behavior

---

## Quick Reference

### Common Commands

| Command | Description |
|:--------|:------------|
| `autospec run -a "desc"` | Full workflow: specify, plan, tasks, implement |
| `autospec prep "desc"` | Planning only: specify, plan, tasks |
| `autospec implement` | Execute tasks from tasks.yaml |
| `autospec st` | Check current spec status and progress |
| `autospec doctor` | Verify dependencies |

### Exit Codes

| Code | Meaning |
|:-----|:--------|
| 0 | Success |
| 1 | Validation failed |
| 2 | Retries exhausted |
| 3 | Invalid arguments |
| 4 | Missing dependencies |
| 5 | Timeout |

### Configuration Priority

1. Environment variables (`AUTOSPEC_*`)
2. Project config (`.autospec/config.yml`)
3. User config (`~/.config/autospec/config.yml`)
4. Defaults

---

## File Locations

### Configuration Files

| File | Purpose |
|:-----|:--------|
| `~/.config/autospec/config.yml` | User configuration |
| `.autospec/config.yml` | Project configuration |

### State Files

| File | Purpose |
|:-----|:--------|
| `~/.autospec/state/retry.json` | Retry state tracking |
| `~/.autospec/state/history.yaml` | Command history |

### Specification Files

| File | Purpose |
|:-----|:--------|
| `specs/<name>/spec.yaml` | Feature specification |
| `specs/<name>/plan.yaml` | Implementation plan |
| `specs/<name>/tasks.yaml` | Task breakdown |
