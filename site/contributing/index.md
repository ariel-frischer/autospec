---
layout: default
title: Contributing
nav_order: 5
has_children: true
permalink: /contributing/
mermaid: true
---

# Contributing

Developer documentation for contributing to autospec. These docs cover internal architecture, coding standards, and implementation details.

## What's in this section

| Document | Description |
|----------|-------------|
| [Architecture](https://github.com/ariel-frischer/autospec/blob/main/docs/internal/architecture.md) | System design, component diagrams, and execution flows |
| [Go Best Practices](https://github.com/ariel-frischer/autospec/blob/main/docs/internal/go-best-practices.md) | Go conventions, naming, and error handling patterns |
| [Internals](https://github.com/ariel-frischer/autospec/blob/main/docs/internal/internals.md) | Spec detection, validation, retry system, and phase context |
| [Testing & Mocks](https://github.com/ariel-frischer/autospec/blob/main/docs/internal/testing-mocks.md) | Testing patterns and mock implementations |
| [Events System](https://github.com/ariel-frischer/autospec/blob/main/docs/internal/events.md) | Event-driven architecture and hooks |
| [YAML Schemas](/autospec/reference/yaml-schemas) | Detailed YAML artifact schemas and validation |
| [Risks](https://github.com/ariel-frischer/autospec/blob/main/docs/internal/risks.md) | Risk documentation in plan.yaml |

## Getting Started

1. Read [Architecture](https://github.com/ariel-frischer/autospec/blob/main/docs/internal/architecture.md) for system overview
2. Review [Go Best Practices](https://github.com/ariel-frischer/autospec/blob/main/docs/internal/go-best-practices.md) for coding standards
3. Check [CLAUDE.md](https://github.com/ariel-frischer/autospec/blob/main/CLAUDE.md) for development commands

## Quick Commands

```bash
make build          # Build for current platform
make test           # Run all tests
make fmt            # Format Go code
make lint           # Run linters
```
