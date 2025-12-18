---
layout: default
title: Architecture
nav_order: 5
has_children: true
permalink: /architecture/
---

# Architecture

Deep dive into autospec's system design, component structure, and internal workings. This section is for contributors, power users, and anyone who wants to understand how autospec works under the hood.

## What's in this section

| Document | Description |
|----------|-------------|
| [Overview](overview) | System design, component diagrams, execution flows |
| [Internals](internals) | Spec detection, validation, retry system, phase context |

## Key concepts

### Stage vs Phase

Understanding autospec's terminology is essential:

- **Stage**: High-level workflow step (specify, plan, tasks, implement)
- **Phase**: Task grouping within implementation (Phase 1: Setup, Phase 2: Core, etc.)

### Package organization

autospec is built as a modular Go application:

```
internal/
├── cli/          # Cobra CLI commands
├── workflow/     # Orchestration and Claude execution
├── config/       # Hierarchical configuration
├── validation/   # Artifact validation (<10ms contract)
├── retry/        # Persistent retry state
├── spec/         # Spec detection from git/directory
└── ...
```

## Quick links

- **System design**: [Architecture Overview](overview)
- **Internal systems**: [Internals Guide](internals)
- **CLI commands**: [CLI Reference](../reference/cli)
- **YAML schemas**: [YAML Schemas](../reference/yaml-schemas)
