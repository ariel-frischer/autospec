<div align="center">

<pre>
▄▀█ █ █ ▀█▀ █▀█ █▀ █▀█ █▀▀ █▀▀
█▀█ █▄█  █  █▄█ ▄█ █▀▀ ██▄ █▄▄
</pre>

**Spec-Driven Development Automation**

[![GitHub CI](https://github.com/ariel-frischer/autospec/actions/workflows/ci.yml/badge.svg)](https://github.com/ariel-frischer/autospec/actions/workflows/ci.yml)
[![GitHub Release](https://img.shields.io/github/v/release/ariel-frischer/autospec)](https://github.com/ariel-frischer/autospec/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/ariel-frischer/autospec)](https://goreportcard.com/report/github.com/ariel-frischer/autospec)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Build features systematically with AI-powered specification workflows.

</div>

**Stop AI slop.** Autospec brings structure to AI coding: spec → plan → tasks → implement - all in one command.

Built for Claude Code and inspired by [GitHub SpecKit](https://github.com/github/spec-kit), Autospec reimagines the specification workflow with **YAML-first artifacts** for programmatic access and validation. These principles ensure reliable, performant, and maintainable software that developers 
can trust for their critical development workflows.

## 📦 Installation

```bash
curl -fsSL https://raw.githubusercontent.com/ariel-frischer/autospec/main/install.sh | sh
```

## 🎯 Key Features

- **Automated Workflow Orchestration** — Runs stages in dependency order with automatic retry on failure
- **YAML-First Artifacts** — Machine-readable `spec.yaml`, `plan.yaml`, `tasks.yaml` for programmatic access
- **Smart Validation** — Validates artifacts exist and meet completeness criteria before proceeding
- **Configurable Retry Logic** — Automatic retries with persistent state tracking
- **Cross-Platform** — Native binaries for Linux and macOS (Intel/Apple Silicon). Windows users: use [WSL](https://learn.microsoft.com/en-us/windows/wsl/install)
- **Smart Claude Setup** — Auto-detects OAuth/API auth, defaults to subscription billing to prevent accidental API charges
- **Flexible Stage Selection** — Mix and match stages with intuitive flags (`-spti`, `-a`, etc.)
- **Shell Completion** — Tab completion for bash, zsh, and fish
- **OS Notifications** — Native desktop notifications with custom sound support
- **History Tracking** — View and filter command execution history with status, duration, and exit codes
- **Auto-Commit** — Automatic git commit creation with .gitignore management and conventional commit messages

## ✨ What Makes Autospec Different?

Originally inspired by [GitHub SpecKit](https://github.com/github/spec-kit), Autospec is now a **fully standalone tool** with its own embedded commands and workflows.

| Feature | GitHub SpecKit | Autospec |
|---------|---------------|----------|
| Output Format | Markdown | **YAML** (machine-readable) |
| Validation | Manual review | **Automatic** with retry logic |
| Context Efficiency | Full prompt each time | **Smart YAML injection** + **phase-isolated sessions** |
| Status Updates | Manual | **Auto-updates** spec.yaml & tasks.yaml |
| Phase Orchestration | Manual | **Automated** with dependencies |
| Session Isolation | Single session | **Per-phase/task** (80%+ cost savings) |
| Implementation | Shell scripts | **Go** (type-safe, single binary) |

## 🚀 Quick Start

> **New to autospec?** See the [Quickstart Guide](docs/quickstart.md) or run the [interactive demo](scripts/quickstart-demo.sh).

### Prerequisites

- [Claude Code CLI](https://code.claude.com/docs/en/setup)
- Git

### Initialize Your Project

1. Navigate to your git repo/project directory, then check dependencies:
   ```bash
   autospec doctor
   ```

2. Initialize Autospec (config, commands, and scripts):
   ```bash
   autospec init
   ```

3. Create project constitution (once per project, triggers Claude session):
   ```bash
   autospec constitution
   ```

## 🎮 Usage

### Core Flow Commands

The core workflow runs four stages in sequence, each creating a YAML artifact:

| Stage | Command | Creates | Description |
|-------|---------|---------|-------------|
| **specify** | `autospec specify "desc"` | `specs/001-feature/spec.yaml` | Feature specification with requirements |
| **plan** | `autospec plan` | `specs/001-feature/plan.yaml` | Implementation design and architecture |
| **tasks** | `autospec tasks` | `specs/001-feature/tasks.yaml` | Actionable task breakdown with dependencies |
| **implement** | `autospec implement` | — | Executes tasks, updates status in tasks.yaml |

> **Branch creation:** `specify` automatically creates and checks out a new feature branch (e.g., `spec/001-user-auth`) before generating the spec.

### Recommended Workflow

1. Generate the specification
2. Review and edit `specs/001-user-auth/spec.yaml` as needed
3. Continue with plan → tasks → implement

```bash
autospec run -s "Add user authentication with OAuth"
autospec run -pti
```

> This iterative approach lets you review and refine the spec before committing to implementation.

### Flexible Stage Selection with `run`

```bash
# All core stages: specify → plan → tasks → implement
autospec run -a "Add user authentication with OAuth"

# Specify + plan
autospec run -sp "Add caching layer"

# Tasks + implement
autospec run -ti --spec 007-feature

# Specify + clarify
autospec run -sr "Add payments"

# All core + checklist
autospec run -a -l

# Tasks + checklist + analyze + implement
autospec run -tlzi

# All core with skip confirmations (-y)
autospec run -a -y "Feature description"
```

### Shortcut Commands

```bash
# All core stages: specify → plan → tasks → implement
autospec all "Add feature description"

# Planning only: specify → plan → tasks (no implementation)
autospec prep "Add feature description"

# Implementation only
autospec implement
autospec implement 003-feature "Focus on tests"

# Show artifacts and task progress
autospec status
autospec st
autospec st -v
```

### Implementation Execution Modes

Control how implementation runs with different levels of context isolation:

```bash
# Phase mode (default): 1 session per phase - balanced cost/context
autospec implement
autospec implement --from-phase 3   # Resume from phase 3 onwards
autospec implement --phase 3        # Run only phase 3

# Task mode: 1 session per task - complex tasks, max isolation
autospec implement --tasks
autospec implement --from-task T005 # Resume from task T005 onwards
autospec implement --task T003      # Run only task T003

# Single mode: 1 session for all - small specs, simple tasks
autospec implement --single-session
```

> Set the default mode via config: `implement_method: phases | tasks | single-session`

> `--tasks`, `--phases`, and `--single-session` are mutually exclusive. Task-level execution respects dependency order and validates each task completes before proceeding.

> **Why isolate sessions?** Context accumulation causes LLM performance degradation and higher API costs (each turn bills the entire context). Phase/task isolation can reduce costs by **80%+** on large specs. See [FAQ](docs/faq.md#why-use---phases-or---tasks-instead-of-running-everything-in-one-session) for details.

### Optional Stage Commands

```bash
# Create/update project principles
autospec constitution "Emphasize security"

# Refine spec with Q&A (interactive mode)
autospec clarify "Focus on edge cases"

# Generate validation checklist
autospec checklist "Include a11y checks"

# Cross-artifact consistency analysis (interactive mode)
autospec analyze "Verify API contracts"
```

### Stage Flags Reference (`run` command)

| Flag | Stage | Description |
|------|-------|-------------|
| `-s` | specify | Generate feature specification |
| `-p` | plan | Generate implementation plan |
| `-t` | tasks | Generate task breakdown |
| `-i` | implement | Execute implementation |
| `-a` | all | All core stages (`-spti`) |
| `-n` | constitution | Create/update project constitution |
| `-r` | clarify | Refine spec with Q&A (interactive mode) |
| `-l` | checklist | Generate validation checklist |
| `-z` | analyze | Cross-artifact consistency check (interactive mode) |

> Stages always execute in canonical order regardless of flag order:
> `constitution → specify → clarify → plan → tasks → checklist → analyze → implement`

### Task Management

Claude automatically updates task status during implementation. Manual updates:

```bash
autospec update-task T001 InProgress
autospec update-task T001 Completed
autospec update-task T001 Blocked
```

### History Tracking

View command execution history with filtering and status tracking. See [docs/reference.md](docs/reference.md#autospec-history) for details.

```bash
autospec history              # View all history
autospec history -n 10        # Last 10 entries
autospec history --status failed
```

## 📁 Output Structure

Autospec generates structured YAML artifacts:

```
specs/
└── 001-user-auth/
    ├── spec.yaml      # Feature specification
    ├── plan.yaml      # Implementation plan
    └── tasks.yaml     # Actionable task breakdown
```

### Example `tasks.yaml`

```yaml
feature: user-authentication
tasks:
  - id: T001
    title: Create user model
    status: Completed
    dependencies: []
  - id: T002
    title: Add login endpoint
    status: InProgress
    dependencies: [T001]
  - id: T003
    title: Write authentication tests
    status: Pending
    dependencies: [T002]
```

## ⚙️ Configuration

### Config Files (YAML format)

- **User config**: `~/.config/autospec/config.yml` (XDG compliant)
- **Project config**: `.autospec/config.yml`

Priority: Environment vars > Project config > User config > Defaults

### All Settings

```yaml
# .autospec/config.yml

# Agent configuration (recommended)
agent_preset: claude                  # Built-in: claude | gemini | cline | codex | opencode | goose
custom_agent_cmd: ""                  # Custom agent template with {{PROMPT}} placeholder

# Legacy Claude CLI settings (deprecated - use agent_preset instead)
claude_cmd: claude                    # Claude CLI command
claude_args:                          # Arguments passed to Claude CLI
  - -p
  - --verbose
  - --output-format
  - stream-json
custom_claude_cmd: ""                 # Custom command (overrides claude_cmd + claude_args)

# Workflow settings
max_retries: 0                        # Max retry attempts per stage (0-10)
specs_dir: ./specs                    # Directory for feature specs
state_dir: ~/.autospec/state          # Directory for state files
skip_preflight: false                 # Skip preflight checks
timeout: 2400                         # Timeout in seconds (40 min default, 0 = no timeout)
skip_confirmations: false             # Skip confirmation prompts
implement_method: phases              # Default: phases | tasks | single-session
auto_commit: false                    # Auto-create git commit after workflow (default: false)

# Notifications (all platforms)
notifications:
  enabled: false                      # Enable notifications (opt-in)
  type: both                          # sound | visual | both
  sound_file: ""                      # Custom sound file (empty = system default)
  on_command_complete: true           # Notify when command finishes
  on_stage_complete: false            # Notify on each stage
  on_error: true                      # Notify on failures
  on_long_running: false              # Notify after threshold
  long_running_threshold: 2m          # Duration threshold
```

> **Migration note:** The `claude_cmd`, `claude_args`, and `custom_claude_cmd` fields are deprecated. Use `agent_preset` instead. See [docs/agents.md](docs/agents.md) for migration guide.

### Claude CLI Arguments (`claude_args`)

The default `claude_args` are optimized for Autospec's orchestration workflow:

| Argument | Purpose |
|----------|---------|
| `-p` | **Print mode** — Runs Claude with a prompt and exits (non-interactive) |
| `--verbose` | Shows detailed progress and tool calls |
| `--output-format stream-json` | Streams JSON output for real-time parsing |

**Common Customizations:**

```yaml
# Minimal (faster, less output)
claude_args:
  - -p

# With model selection
claude_args:
  - -p
  - --model
  - claude-sonnet-4-5-20250929

# With streaming output (shows Claude's messages as it runs)
claude_args:
  - -p
  - --verbose
  - --output-format
  - stream-json

# Allow all permissions (use with caution in sandboxed environments)
claude_args:
  - -p
  - --dangerously-skip-permissions
```

> **Warning:** `--dangerously-skip-permissions` bypasses all Claude safety prompts. Only use in trusted environments with proper sandboxing. See [Claude Settings docs](docs/claude-settings.md).

> For complete control, use `custom_claude_cmd` to replace both `claude_cmd` and `claude_args`. See [Pro Tips](#readable-streaming-output-with-claude-clean) for examples.

### Commands

```bash
autospec init
autospec init --project
autospec config show
autospec config show --json
autospec config migrate
autospec config migrate --dry-run
```

## 🐚 Shell Completion

The easiest way to set up shell completions (auto-detects your shell):

```bash
autospec completion install
```

Or install for a specific shell:

```bash
autospec completion install bash
autospec completion install zsh
autospec completion install fish
```

See [docs/SHELL-COMPLETION.md](docs/SHELL-COMPLETION.md) for detailed setup and manual instructions.

## 🔧 Exit Codes

Uses standardized exit codes (0-5) for CI/CD integration. See [docs/reference.md](docs/reference.md#exit-codes) for full details.

```bash
autospec run -a "feature" && echo "Success" || echo "Failed: $?"
```

## 🔍 Troubleshooting

```bash
autospec doctor
autospec --debug run -a "feature"
autospec config show
```

See [docs/troubleshooting.md](docs/troubleshooting.md) for common issues and solutions.

## 📝 Slash Commands for Interactive Sessions

`autospec init` installs slash commands to `.claude/commands/autospec.*.md` for use in normal Claude Code sessions:

```bash
/autospec.specify    # Generate spec.yaml interactively
/autospec.plan       # Generate plan.yaml
/autospec.tasks      # Generate tasks.yaml
/autospec.implement  # Execute implementation
/autospec.clarify    # Refine specifications
/autospec.analyze    # Cross-artifact analysis
/autospec.checklist  # Generate quality checklist
/autospec.constitution  # Create project constitution
```

Use these when you prefer chat-based iteration over autospec's automated (`-p`) mode.

## 💡 Pro Tips

### Optional Dependencies

- [claude-clean](https://github.com/ariel-frischer/claude-clean) — Beautiful terminal parser for Claude Code's streaming JSON output
- [bubblewrap](https://github.com/containers/bubblewrap) (Linux) / Seatbelt (macOS) — OS-level sandboxing. See [Claude Settings](docs/claude-settings.md)
- Go 1.21+ — For building from source
- make — For Makefile commands

### Readable Streaming Output with claude-clean

[claude-clean](https://github.com/ariel-frischer/claude-clean) makes Claude's `stream-json` output readable in real-time.

Configure a custom command in `~/.config/autospec/config.yml`:

```yaml
custom_claude_cmd: "claude -p --verbose --output-format stream-json {{PROMPT}} | cclean"
```

> **DANGER:** Adding `--dangerously-skip-permissions` bypasses ALL Claude safety checks. Never use with credentials, API keys, or production data. Your system becomes fully exposed to any command Claude generates.
>
> **Recommended:** Enable Claude Code's sandbox first (`/sandbox` command) which uses [bubblewrap](https://github.com/containers/bubblewrap) on Linux or Seatbelt on macOS for OS-level isolation. See [Claude Settings docs](docs/claude-settings.md) for configuration via settings.json.

## 📥 Build from Source

Requires Go 1.21+

```bash
git clone https://github.com/ariel-frischer/autospec.git
cd autospec
make install
```

## 🤝 Contributing

Contributions welcome! See [CONTRIBUTORS.md](CONTRIBUTORS.md) for development guidelines.

## 📄 License

MIT License — see [LICENSE](LICENSE) for details.

---

**Documentation:** `autospec --help`

**Issues:** [github.com/ariel-frischer/autospec/issues](https://github.com/ariel-frischer/autospec/issues)

**Star us on GitHub if you find Autospec useful!**
