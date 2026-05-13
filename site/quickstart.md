---
layout: default
title: Quickstart
nav_order: 2
description: "Get started with autospec in 5 minutes - installation, setup, and your first workflow"
permalink: /quickstart
---

# Quickstart
{: .fs-9 }

Get up and running with autospec in 5 minutes.
{: .fs-6 .fw-300 }

---

## Prerequisites

Before you begin, ensure you have:

- **Supported CLI agent**: Claude Code, Codex CLI, or OpenCode installed and authenticated
- **Git**: For version control and branch-based spec detection

Verify your agent is installed:

```bash
claude --version
# or
codex --version
# or
opencode --version
```

If you see `command not found`, visit the [troubleshooting guide](/autospec/guides/troubleshooting).

---

## Cost Warning

{: .warning }
> **Check your agent auth method before long runs.** API keys can bill per-token and get expensive. Claude Pro/Max plans include usage at no extra cost for Claude Code; Codex auth is managed by the Codex CLI.
>
> Run `claude` interactively, then `/status` to see your login method.

---

## Step 1: Install autospec

### Option A: Install Script (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/ariel-frischer/autospec/main/install.sh | sh
```

### Option B: Download Binary

Visit the [releases page](https://github.com/ariel-frischer/autospec/releases) and download for your platform:

| Platform | Binary |
|:---------|:-------|
| Linux | `autospec-linux-amd64` |
| macOS Intel | `autospec-darwin-amd64` |
| macOS Apple Silicon | `autospec-darwin-arm64` |

```bash
chmod +x autospec-*
mkdir -p ~/.local/bin
mv autospec-* ~/.local/bin/autospec
```

{: .note }
> Ensure `~/.local/bin` is in your PATH. Add `export PATH="$HOME/.local/bin:$PATH"` to your shell config if needed.

### Option C: Build from Source

```bash
git clone https://github.com/ariel-frischer/autospec.git
cd autospec
make build
sudo make install
```

### Verify Installation

```bash
autospec version
```

Expected output:
```
autospec version 1.0.0
```

---

## Step 2: Check Dependencies

Run health checks to verify everything is set up:

```bash
autospec doctor
```

Expected output:
```
✓ CLI agent found
✓ Configuration loaded successfully
✓ Specs directory accessible: ./specs
✓ State directory accessible: ~/.autospec/state
✓ Configuration loaded successfully

All checks passed!
```

If any checks fail, see the [troubleshooting guide](/autospec/guides/troubleshooting).

---

## Step 3: Initialize Configuration

Create the default configuration:

```bash
autospec init
```

This command:
1. Creates `~/.config/autospec/config.yml` with default settings
2. Installs slash commands to `.claude/commands/`
3. **Prompts to create project constitution** (say "yes" - required for autospec to work)

Default config:

```yaml
claude_cmd: claude
max_retries: 0
specs_dir: ./specs
state_dir: ~/.autospec/state
timeout: 0
```

See [Configuration Reference](/autospec/reference/configuration) for customization options.

### Security Notice

On your first workflow run with autonomous mode enabled, you'll see a one-time security notice:

```
┌───────────────────────────────────────────────────────────────────┐
│ Security Notice                                                   │
├───────────────────────────────────────────────────────────────────┤
│ Running in autonomous permissions mode                            │
│                                                                   │
│ This mode is RECOMMENDED for autospec workflows. Without it,     │
│ supported agents may require manual approval for edits and       │
│ commands, making automation impractical.                         │
│                                                                   │
│ ✓ Sandbox: enabled ✓                                              │
│   OS-level protection active.                                    │
└───────────────────────────────────────────────────────────────────┘
```

{: .warning }
> **Caution**: Autonomous mode gives the selected agent broad access without prompts. Claude sandboxing can limit access to your project; Codex yolo mode bypasses Codex approvals and sandboxing. Use isolated environments for high-risk work. See [Configuration - Security](/autospec/reference/configuration#security-sandbox--permissions) for details.

{: .note }
> Suppress this notice: `autospec config set skip_permissions_notice_shown true` or `AUTOSPEC_SKIP_PERMISSIONS_NOTICE=1`

`autospec init` installs slash commands for agents that support them, such as Claude Code (`.claude/commands/autospec.*.md`) and OpenCode (`.opencode/command/autospec.*.md`). Codex receives rendered prompt text directly and does not use command template files.

| Command | Purpose |
|:--------|:--------|
| `/autospec.specify` | Generate spec.yaml interactively |
| `/autospec.plan` | Generate plan.yaml |
| `/autospec.tasks` | Generate tasks.yaml |
| `/autospec.implement` | Execute implementation |
| `/autospec.clarify` | Refine specifications |
| `/autospec.analyze` | Cross-artifact analysis |
| `/autospec.checklist` | Generate quality checklist |
| `/autospec.constitution` | Create project constitution |

Use these in normal Claude Code or OpenCode sessions when you prefer chat-based iteration over autospec's automated mode.

---

## Step 4: Create Project Constitution (if skipped)

{: .note }
> If you said "yes" to "Create constitution?" during `autospec init`, **skip this step** - your constitution is already created.

If you skipped constitution creation during init, or need to regenerate it:

```bash
autospec constitution
```

This launches the configured agent to analyze your codebase and create `.autospec/constitution.yaml` containing your project's:
- Coding standards and conventions
- Architectural principles
- Testing requirements
- Documentation standards

The constitution ensures the agent follows your project's patterns during implementation.

---

## Step 5: Create Your First Specification

Navigate to your project and create a specification:

```bash
cd your-project

# Generate spec.yaml only (also creates feature branch via git checkout)
autospec run -s "Add a health check endpoint at /health"
```

**What happens:**

Creates `specs/add-health-check-endpoint/` with your specification:

| File | Contents |
|:-----|:---------|
| `spec.yaml` | Requirements, acceptance criteria, success metrics |

Expected output:
```
→ Executing specify stage...
✓ Specification created: specs/add-health-check-endpoint/spec.yaml
✓ Validation passed
```

To continue with planning and implementation, run additional stages:

```bash
# Continue with plan + tasks + implement
autospec run -pti
```

---

## Step 6: Review Generated Artifacts

Check what was created:

```bash
ls specs/001-health-check/
```

Output:
```
spec.yaml  plan.yaml  tasks.yaml
```

| File | Purpose |
|:-----|:--------|
| `spec.yaml` | Requirements, acceptance criteria, success metrics |
| `plan.yaml` | Technical architecture, design decisions, file structure |
| `tasks.yaml` | Ordered tasks with dependencies and status tracking |

Check progress with:

```bash
autospec st
```

---

## Alternative Workflows

### Iterative Approach (Recommended)

Review and refine between stages:

```bash
# Step 1: Generate spec only
autospec run -s "Add rate limiting to API endpoints"

# Step 2: Review and edit specs/001-rate-limiting/spec.yaml

# Step 3: Continue with remaining stages
autospec run -pti
```

### Planning Only (No Implementation)

Generate artifacts for review before implementing:

```bash
autospec prep "Add caching layer for database queries"

# Review the generated artifacts...

# Then implement when ready:
autospec implement
```

---

## Stage Flags Reference

| Flag | Stage | Description |
|:-----|:------|:------------|
| `-s` | specify | Generate feature specification |
| `-p` | plan | Generate implementation plan |
| `-t` | tasks | Generate task breakdown |
| `-i` | implement | Execute implementation |
| `-a` | all | All core stages (`-spti`) |
| `-r` | clarify | Refine spec with Q&A |
| `-l` | checklist | Generate validation checklist |
| `-z` | analyze | Cross-artifact consistency check |
| `-y` | — | Skip confirmation prompts |

Common combinations:

```bash
# All core stages
autospec run -a "feature"

# Planning only (specify + plan + tasks)
autospec run -spt "feature"   # or: autospec prep "feature"

# Specify + clarify (refine spec with questions)
autospec run -sr "feature"

# All stages + checklist + analyze
autospec run -alz "feature"
```

---

## Monitoring Progress

```bash
# Quick status check
autospec st

# Verbose status with task details
autospec st -v

# View command history
autospec history
autospec history -n 10
```

---

## Implementation Modes

```bash
# Default: One session per phase
autospec implement

# Per-task isolation (recommended for complex features)
autospec implement --tasks

# Single session (for small/simple specs)
autospec implement --single-session

# Resume from specific point
autospec implement --from-phase 3
autospec implement --from-task T005
autospec implement --task T003  # Single task only
```

---

## Example Feature Descriptions

```bash
# API Features
autospec run -a "Add a health check endpoint at /health that returns JSON status"
autospec run -a "Add rate limiting middleware with configurable limits per route"
autospec run -a "Implement pagination for all list endpoints"

# Authentication
autospec run -a "Add JWT authentication with refresh token support"
autospec run -a "Add OAuth2 login with Google and GitHub providers"

# Database
autospec run -a "Add database connection pooling with configurable pool size"
autospec run -a "Implement soft delete for user records with restore functionality"

# Testing
autospec run -a "Add integration tests for the payment processing module"

# DevOps
autospec run -a "Add Dockerfile with multi-stage build for production"
autospec run -a "Create GitHub Actions CI pipeline with test and lint stages"
```

---

## Troubleshooting

### "agent: command not found"

The selected agent CLI is not installed or not in PATH.

**Solution**: Install Claude Code, Codex CLI, or OpenCode, then verify with `claude --version`, `codex --version`, or `opencode --version`.

### "autospec: command not found"

autospec binary is not in PATH.

**Solution**: Run `sudo make install` or add `~/.local/bin` to your PATH.

### "Validation failed: spec file not found"

Workflow stage failed to create expected output file.

**Solution**: Check error messages. If retry limit exhausted, reset retry state:
```bash
rm ~/.autospec/state/retry.json
```

### "Spec not detected"

Auto-detection failed to find current feature.

**Solution**: Ensure you're on a feature branch with format `NNN-feature-name` (e.g., `001-health-check`), or explicitly specify the spec:
```bash
autospec implement 001-health-check
```

For more solutions, see the [full troubleshooting guide](/autospec/guides/troubleshooting).

---

## Next Steps

- [CLI Reference](/autospec/reference/): Complete command documentation
- [Configuration](/autospec/reference/configuration): Customize autospec behavior
- [Architecture](/autospec/architecture/): Understand system design
- [FAQ](/autospec/guides/faq): Common questions answered

---

## Getting Help

- **GitHub Issues**: [Report bugs or request features](https://github.com/ariel-frischer/autospec/issues)
- **Documentation**: Browse the sections in the sidebar
