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

- **Claude Code CLI**: Installed and authenticated ([setup guide](https://docs.anthropic.com/en/docs/claude-code/getting-started))
- **Git**: For version control and branch-based spec detection

Verify Claude CLI is installed:

```bash
claude --version
```

If you see `command not found`, visit the [troubleshooting guide](/autospec/guides/troubleshooting#claude-command-not-found).

---

## Cost Warning

{: .warning }
> **Check your Claude auth method before long runs.** API keys (`ANTHROPIC_API_KEY`) bill per-token and can get expensive. Pro/Max plans ($20+/month) include usage at no extra cost.
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
✓ Claude CLI found: /usr/local/bin/claude
✓ Claude CLI authenticated
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

This creates `~/.config/autospec/config.yml` with default settings:

```yaml
claude_cmd: claude
max_retries: 0
specs_dir: ./specs
state_dir: ~/.autospec/state
timeout: 0
```

See [Configuration Reference](/autospec/reference/configuration) for customization options.

`autospec init` also installs slash commands to `.claude/commands/autospec.*.md`:

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

Use these in normal Claude Code sessions when you prefer chat-based iteration over autospec's automated (`-p`) mode.

---

## Step 4: Create Project Constitution

Create a constitution for your project (required once per project):

```bash
autospec constitution
```

This launches a Claude session that analyzes your codebase and creates `.autospec/memory/constitution.yaml` containing your project's:
- Coding standards and conventions
- Architectural principles
- Testing requirements
- Documentation standards

The constitution ensures Claude follows your project's patterns during implementation.

---

## Step 5: Run Your First Workflow

Navigate to your project and run a complete workflow:

```bash
cd your-project

# Generate everything: spec → plan → tasks → implement
autospec run -a "Add a health check endpoint at /health"
```

**What happens:**

Creates `specs/add-health-check-endpoint/` with YAML artifacts:

| Stage | Creates | Contents |
|:------|:--------|:---------|
| Specify | `spec.yaml` | Requirements, acceptance criteria |
| Plan | `plan.yaml` | Architecture, design decisions |
| Tasks | `tasks.yaml` | Ordered tasks with dependencies |
| Implement | Updates `tasks.yaml` | Task status as work completes |

Expected output:
```
→ Executing specify stage...
✓ Specification created: specs/001-health-check/spec.yaml
✓ Validation passed

→ Executing plan stage...
✓ Plan created: specs/001-health-check/plan.yaml
✓ Validation passed

→ Executing tasks stage...
✓ Tasks created: specs/001-health-check/tasks.yaml
✓ Validation passed

→ Executing implement stage...
✓ Implementation complete
```

---

## Step 5: Review Generated Artifacts

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

### "claude: command not found"

Claude CLI is not installed or not in PATH.

**Solution**: Install Claude CLI following the [official guide](https://docs.anthropic.com/en/docs/claude-code/getting-started), then verify with `claude --version`.

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
