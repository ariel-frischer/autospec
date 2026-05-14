---
layout: default
title: Home
nav_order: 1
description: "autospec - AI-powered software specification and implementation workflows"
permalink: /
---

<pre class="ascii-logo">
▄▀█ █ █ ▀█▀ █▀█ █▀ █▀█ █▀▀ █▀▀
█▀█ █▄█  █  █▄█ ▄█ █▀▀ ██▄ █▄▄
</pre>

Spec-Driven Development Automation
{: .fs-6 .fw-300 }

Stop AI slop. Build features systematically with AI-powered specification workflows.
{: .fs-5 .fw-300 }

[Get Started](/autospec/quickstart){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 }
[View on GitHub](https://github.com/ariel-frischer/autospec){: .btn .fs-5 .mb-4 .mb-md-0 }

---

## What is autospec?

autospec brings structure to AI coding: **spec → plan → tasks → implement** - all in one command.

Built for CLI coding agents including Claude Code, Codex, and OpenCode, and inspired by [GitHub SpecKit](https://github.com/github/spec-kit), autospec reimagines the specification workflow with **YAML-first artifacts** for programmatic access and validation.

Supported agents: [Claude Code](https://claude.ai/code), [Codex CLI](https://developers.openai.com/codex/cli/reference), and [OpenCode](https://opencode.ai).

```bash
# Generate everything: spec → plan → tasks → implement
autospec run -a "Add user authentication with OAuth"

# Or step by step with review between stages
autospec run -s "Add feature"  # Generate spec
# Review and edit spec.yaml
autospec run -pti              # Plan, tasks, implement
```

Creates `specs/<feature-name>/` with YAML artifacts at each stage:

| Stage | Creates | Contents |
|:------|:--------|:---------|
| specify | `spec.yaml` | Requirements, acceptance criteria |
| plan | `plan.yaml` | Architecture, design decisions |
| tasks | `tasks.yaml` | Ordered tasks with dependencies |
| implement | Updates `tasks.yaml` | Task status, completion |

---

## Key Features

<div class="features-grid">
  <div class="feature-card">
    <h3>Automated Workflow Orchestration</h3>
    <p>Runs stages in dependency order with automatic retry on failure. No manual intervention needed.</p>
  </div>
  <div class="feature-card">
    <h3>YAML-First Artifacts</h3>
    <p>Machine-readable spec.yaml, plan.yaml, tasks.yaml for programmatic access and validation.</p>
  </div>
  <div class="feature-card">
    <h3>Smart Validation</h3>
    <p>Validates artifacts exist and meet completeness criteria before proceeding to the next stage.</p>
  </div>
  <div class="feature-card">
    <h3>Session Isolation</h3>
    <p>Per-phase or per-task execution reduces API costs by 80%+ on large specs through context isolation.</p>
  </div>
  <div class="feature-card">
    <h3>Configurable Retry Logic</h3>
    <p>Automatic retries with persistent state tracking. Resume from failures without losing progress.</p>
  </div>
  <div class="feature-card">
    <h3>Performance Optimized</h3>
    <p>Sub-second validation (&lt;10ms per check), &lt;50ms startup. Built for speed.</p>
  </div>
</div>

---

## Quick Start

### Prerequisites

- Claude Code, Codex CLI, or OpenCode installed and configured
- Git

### Installation

```bash
curl -fsSL https://raw.githubusercontent.com/ariel-frischer/autospec/main/install.sh | sh
```

### First Workflow

```bash
# Navigate to your project
cd your-project

# Check dependencies
autospec doctor

# Initialize autospec configuration (prompts to create constitution)
autospec init

# Create your first specification (also runs git checkout for feature branch)
autospec run -s "Add user authentication with OAuth"
```

[View Full Quickstart Guide](/autospec/quickstart){: .btn .btn-outline }

---

## The Workflow

autospec runs four core stages in sequence:

| Stage | Command | Creates | Description |
|:------|:--------|:--------|:------------|
| **specify** | `autospec specify "desc"` | `spec.yaml` | Feature specification with requirements |
| **plan** | `autospec plan` | `plan.yaml` | Implementation design and architecture |
| **tasks** | `autospec tasks` | `tasks.yaml` | Actionable task breakdown with dependencies |
| **implement** | `autospec implement` | — | Executes tasks, updates status |

Each artifact is validated before proceeding to the next stage, ensuring quality at every step.

---

## Documentation

{: .fs-6 .fw-300 }

| Section | Description |
|:--------|:------------|
| [Quickstart](/autospec/quickstart) | Get up and running in 5 minutes |
| [Reference](/autospec/reference/) | Complete CLI command reference |
| [Guides](/autospec/guides/) | Configuration, troubleshooting, FAQ |
| [FAQ](/autospec/guides/faq) | Common workflow and agent questions |

---

## What Makes autospec Different?

| Feature | GitHub SpecKit | autospec |
|:--------|:--------------|:---------|
| Output Format | Markdown | **YAML** (machine-readable) |
| Workflow UX | Repetitive, non-composable command flow | **One-command end-to-end runs** (`autospec run -a`, `autospec prep`) |
| Validation | Checklist/agent-driven review | **Programmatic validation** with retry logic |
| Token Efficiency | Long implementation chats can accumulate large context | **Fresh bounded sessions per phase/task** (80%+ cost savings) |
| Status Visibility | No structured status view | **Exact phase/task progress** via `autospec st` |
| Phase Orchestration | Agent-driven `/speckit.*` commands | **CLI-orchestrated** stages with dependency handling |
| Implementation | Python CLI + shell/PowerShell scripts | **Go** (type-safe, single binary) |

---

## License

autospec is distributed under the [MIT License](https://github.com/ariel-frischer/autospec/blob/main/LICENSE).
