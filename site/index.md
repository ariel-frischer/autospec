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

Built for Claude Code and inspired by [GitHub SpecKit](https://github.com/github/spec-kit), autospec reimagines the specification workflow with **YAML-first artifacts** for programmatic access and validation.

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

- [Claude Code CLI](https://code.claude.com/docs/en/setup) installed and configured
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

# Initialize autospec configuration
autospec init

# Create project constitution (once per project, triggers Claude session)
autospec constitution

# Create your first specification
autospec run -a "Add user authentication with OAuth"
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
| [Architecture](/autospec/architecture/) | System design and internals |

---

## What Makes autospec Different?

| Feature | GitHub SpecKit | autospec |
|:--------|:--------------|:---------|
| Output Format | Markdown | **YAML** (machine-readable) |
| Validation | Manual review | **Automatic** with retry logic |
| Context Efficiency | Full prompt each time | **Smart YAML injection** + **phase-isolated sessions** |
| Status Updates | Manual | **Auto-updates** spec.yaml & tasks.yaml |
| Phase Orchestration | Manual | **Automated** with dependencies |
| Session Isolation | Single session | **Per-phase/task** (80%+ cost savings) |
| Implementation | Shell scripts | **Go** (type-safe, single binary) |

---

## License

autospec is distributed under the [MIT License](https://github.com/ariel-frischer/autospec/blob/main/LICENSE).
