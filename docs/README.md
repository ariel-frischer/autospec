# Autospec Documentation

Documentation for the autospec CLI tool. The full documentation site is at [ariel-frischer.github.io/autospec](https://ariel-frischer.github.io/autospec/).

## Directory Structure

```
docs/
├── public/           # User-facing documentation
│   ├── quickstart.md
│   ├── reference.md
│   ├── agents.md
│   ├── troubleshooting.md
│   └── ...
├── internal/         # Contributor/developer documentation
│   ├── architecture.md
│   ├── go-best-practices.md
│   ├── internals.md
│   └── ...
└── research/         # Research notes and evaluations
```

## User Documentation (`public/`)

| Document | Description |
|----------|-------------|
| [quickstart.md](public/quickstart.md) | Getting started guide |
| [reference.md](public/reference.md) | Complete CLI command reference |
| [agents.md](public/agents.md) | Agent configuration (Claude, Codex, OpenCode, etc.) |
| [codex-settings.md](public/codex-settings.md) | Codex CLI auth, sandboxing, and yolo mode |
| [claude-settings.md](public/claude-settings.md) | Claude Code settings and sandboxing |
| [troubleshooting.md](public/troubleshooting.md) | Common issues and solutions |
| [faq.md](public/faq.md) | Frequently asked questions |
| [task-sizing.md](public/task-sizing.md) | When to use autospec vs just code directly |
| [worktree.md](public/worktree.md) | Git worktree management |
| [checklists.md](public/checklists.md) | Checklist generation and validation |
| [self-update.md](public/self-update.md) | Self-update feature |
| [TIMEOUT.md](public/TIMEOUT.md) | Timeout configuration |
| [SHELL-COMPLETION.md](public/SHELL-COMPLETION.md) | Shell completion setup |

## Contributor Documentation (`internal/`)

| Document | Description |
|----------|-------------|
| [architecture.md](internal/architecture.md) | System design and component diagrams |
| [go-best-practices.md](internal/go-best-practices.md) | Go conventions and patterns |
| [internals.md](internal/internals.md) | Spec detection, validation, retry system |
| [testing-mocks.md](internal/testing-mocks.md) | Testing patterns and mocks |
| [codex-manual-testing.md](internal/codex-manual-testing.md) | Codex smoke and regression testing checklist |
| [events.md](internal/events.md) | Event system architecture |
| [YAML-STRUCTURED-OUTPUT.md](internal/YAML-STRUCTURED-OUTPUT.md) | YAML artifact schemas |
| [risks.md](internal/risks.md) | Risk documentation in plan.yaml |

## Site Generation

These docs are synced to `site/` for the Jekyll documentation site:

```bash
# Generate site pages from docs/
./scripts/sync-docs-to-site.sh

# Serve locally
cd site && bundle exec jekyll serve --livereload
```

The sync is automated in GitHub Actions - generated files are not committed.

## Quick Links

- [Project README](../README.md) - Installation and overview
- [CLAUDE.md](../CLAUDE.md) - Development guidelines
- [CONTRIBUTORS.md](../CONTRIBUTORS.md) - Contribution guide
