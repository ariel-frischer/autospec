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
| [Architecture](architecture) | System design, component diagrams, and execution flows |
| [Go Best Practices](go-best-practices) | Go conventions, naming, and error handling patterns |
| [Internals](internals) | Spec detection, validation, retry system, and phase context |
| [Testing & Mocks](testing-mocks) | Testing patterns and mock implementations |
| [Events System](events) | Event-driven architecture and hooks |
| [YAML Schemas](/autospec/reference/yaml-schemas) | Detailed YAML artifact schemas and validation |
| [Risks](risks) | Risk documentation in plan.yaml |

## Getting Started

1. Read [Architecture](architecture) for system overview
2. Review [Go Best Practices](go-best-practices) for coding standards
3. Check the repository's `AGENTS.md` for development commands

## Quick Commands

```bash
make build          # Build for current platform
make test           # Run all tests
make fmt            # Format Go code
make lint           # Run linters
```
