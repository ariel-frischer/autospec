<div align="center">

```
▄▀█ █ █ ▀█▀ █▀█ █▀ █▀█ █▀▀ █▀▀
█▀█ █▄█  █  █▄█ ▄█ █▀▀ ██▄ █▄▄
```

**Spec-Driven Development Automation**

[![GitHub CI](https://github.com/ariel-frischer/autospec/actions/workflows/ci.yml/badge.svg)](https://github.com/ariel-frischer/autospec/actions/workflows/ci.yml)
[![GitHub Release](https://img.shields.io/github/v/release/ariel-frischer/autospec)](https://github.com/ariel-frischer/autospec/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/ariel-frischer/autospec)](https://goreportcard.com/report/github.com/ariel-frischer/autospec)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

🏗️ Build features systematically with AI-powered specification workflows.

</div>

Inspired by [GitHub SpecKit](https://github.com/github/spec-kit), Autospec reimagines the specification workflow with **YAML-first artifacts** for programmatic access and validation.

## 📦 Installation

```bash
curl -fsSL https://raw.githubusercontent.com/ariel-frischer/autospec/main/install.sh | sh
```

## 🎯 Key Features

- 🔄 **Automated Workflow Orchestration** — Runs stages in dependency order with automatic retry on failure
- 📝 **YAML-First Artifacts** — Machine-readable `spec.yaml`, `plan.yaml`, `tasks.yaml` for programmatic access
- ✅ **Smart Validation** — Validates artifacts exist and meet completeness criteria before proceeding
- 🔁 **Configurable Retry Logic** — Automatic retries with persistent state tracking
- ⚡ **Performance Optimized** — Sub-second validation (<10ms per check), <50ms startup
- 🖥️ **Cross-Platform** — Native binaries for Linux, macOS (Intel/Apple Silicon), and Windows
- 🎛️ **Flexible Stage Selection** — Mix and match stages with intuitive flags (`-spti`, `-a`, etc.)
- 🏗️ **Constitution Support** — Project-level principles that guide all specifications
- 🔍 **Cross-Artifact Analysis** — Consistency checks across spec, plan, and tasks
- 📋 **Custom Checklists** — Auto-generated validation checklists per feature
- 🧪 **Comprehensive Testing** — Unit tests, benchmarks, and integration tests
- 🐚 **Shell Completion** — Tab completion for bash, zsh, fish, and PowerShell

## ✨ What Makes Autospec Different?

Originally inspired by [GitHub SpecKit](https://github.com/github/spec-kit), Autospec is now a **fully standalone tool** with its own embedded commands and workflows.

| Feature | GitHub SpecKit | Autospec |
|---------|---------------|----------|
| Output Format | Markdown | **YAML** (machine-readable) |
| Validation | Manual review | **Automatic** with retry logic |
| Context Efficiency | Full prompt each time | **Smart YAML injection** (less tokens) |
| Status Updates | Manual | **Auto-updates** spec.yaml & tasks.yaml |
| Phase Orchestration | Manual | **Automated** with dependencies |
| Session Isolation | Single session | **Per-phase/task** (80%+ cost savings) |
| Dependencies | Requires SpecKit CLI | **Self-contained** (only needs Claude CLI) |

## 📦 Quick Start

### Prerequisites

**Required:**

| Name | Description |
|------|-------------|
| [Claude Code CLI](https://code.claude.com/docs/en/setup) | AI-powered coding assistant |
| Git | Version control |

**Optional:**

| Name | Description |
|------|-------------|
| [claude-clean](https://github.com/ariel-frischer/claude-clean) (cclean) | Beautiful terminal parser for Claude Code's streaming JSON output |
| [bubblewrap](https://github.com/containers/bubblewrap) (Linux) / Seatbelt (macOS) | OS-level sandboxing for Claude Code. See [Claude Settings](docs/claude-settings.md) |
| Go 1.21+ | For building from source |
| make | For Makefile commands |

### Initialize Your Project

```bash
# Check dependencies
autospec doctor

# Initialize Autospec (config, commands, and scripts)
autospec init

# Create project constitution (principles that guide all specs)
autospec constitution
```

## 🎮 Usage

### Recommended Workflow

```bash
# 1️⃣ Generate the specification first
autospec run -s "Add user authentication with OAuth"

# 2️⃣ Review and edit specs/001-user-auth/spec.yaml as needed

# 3️⃣ Continue with plan → tasks → implement
autospec run -pti
```

> ⚠️ **Note:** New specs automatically create and checkout a feature branch (e.g., `spec/001-user-auth`).

This iterative approach lets you review and refine the spec before committing to implementation.

### Flexible Stage Selection with `run`

```bash
# 🚀 Run all core stages (specify → plan → tasks → implement)
autospec run -a "Add user authentication with OAuth"

# 📝 Run specific stages
autospec run -sp "Add caching layer"        # Specify + plan only
autospec run -ti --spec 007-feature         # Tasks + implement on specific spec

# ✨ Include optional stages
autospec run -sr "Add payments"             # Specify + clarify
autospec run -a -l                          # All + checklist
autospec run -tlzi                          # Tasks + checklist + analyze + implement

# 🏃 Skip confirmations for automation
autospec run -a -y "Feature description"
```

### Stage Flags Reference

| Flag | Stage | Description |
|------|-------|-------------|
| `-s` | specify | Generate feature specification |
| `-p` | plan | Generate implementation plan |
| `-t` | tasks | Generate task breakdown |
| `-i` | implement | Execute implementation |
| `-a` | all | All core stages (`-spti`) |
| `-n` | constitution | Create/update project constitution |
| `-r` | clarify | Refine spec with Q&A |
| `-l` | checklist | Generate validation checklist |
| `-z` | analyze | Cross-artifact consistency check |

> 📌 Stages always execute in canonical order regardless of flag order:
> `constitution → specify → clarify → plan → tasks → checklist → analyze → implement`

### Shortcut Commands

```bash
# 🎯 Complete workflow: specify → plan → tasks → implement
autospec all "Add feature description"

# 📋 Prepare only: specify → plan → tasks (no implementation)
autospec prep "Add feature description"

# 🔨 Implementation only
autospec implement
autospec implement 003-feature "Focus on tests"

# 📊 Check status (alias: st)
autospec status           # Show artifacts and task progress
autospec st               # Short alias
autospec st -v            # Verbose: show phase details
```

### Implementation Execution Modes

Control how implementation runs with different levels of context isolation:

```bash
# 🔸 Default: Phase-level (each phase in separate session)
autospec implement
autospec implement --from-phase 3        # Resume from phase 3
autospec implement --phase 3             # Run only phase 3

# 🔹 Task-level: Each task in separate session (maximum isolation)
autospec implement --tasks
autospec implement --from-task T005      # Resume from task T005
autospec implement --task T003           # Run only task T003
```

| Mode | Flag | Isolation | Use Case |
|------|------|-----------|----------|
| Phase | (default) | 1 session per phase | Balanced cost/context |
| Task | `--tasks` | 1 session per task | Complex tasks, max isolation |

> 💡 Legacy single-session mode available via config: `implement_method: single-session`

> 📌 `--tasks` and `--phases` are mutually exclusive. Task-level execution respects dependency order and validates each task completes before proceeding.

> 💡 **Why isolate sessions?** Context accumulation causes LLM performance degradation and higher API costs (each turn bills the entire context). Phase/task isolation can reduce costs by **80%+** on large specs. See [FAQ](docs/faq.md#why-use---phases-or---tasks-instead-of-running-everything-in-one-session) for details.

### Optional Stage Commands

```bash
# 🏛️ Constitution - project principles
autospec constitution "Emphasize security"

# ❓ Clarify - refine spec with questions
autospec clarify "Focus on edge cases"

# ✅ Checklist - validation checklist
autospec checklist "Include a11y checks"

# 🔍 Analyze - consistency analysis
autospec analyze "Verify API contracts"
```

### Task Management

```bash
# Update task status during implementation
autospec update-task T001 InProgress
autospec update-task T001 Completed
autospec update-task T001 Blocked
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

### Key Settings

```yaml
# .autospec/config.yml
claude_cmd: claude
max_retries: 3
specs_dir: ./specs
timeout: 600  # seconds (0 = no timeout)
skip_confirmations: false
```

### Readable Streaming Output with claude-clean (Optional)

[claude-clean](https://github.com/ariel-frischer/claude-clean) makes Claude's `stream-json` output readable in real-time:

```bash
curl -fsSL https://raw.githubusercontent.com/ariel-frischer/claude-clean/main/install.sh | sh
```

Then configure a custom command in `~/.config/autospec/config.yml`:

```yaml
custom_claude_cmd: "ANTHROPIC_API_KEY='' claude -p --verbose --output-format stream-json {{PROMPT}} | cclean"
```

> ⚠️ **DANGER:** Adding `--dangerously-skip-permissions` bypasses ALL Claude safety checks. Never use with credentials, API keys, or production data. Your system becomes fully exposed to any command Claude generates.
>
> **Recommended:** Enable Claude Code's sandbox first (`/sandbox` command) which uses [bubblewrap](https://github.com/containers/bubblewrap) on Linux or Seatbelt on macOS for OS-level isolation. See [Claude Settings docs](docs/claude-settings.md) for configuration via settings.json.

### Environment Variables

```bash
export AUTOSPEC_MAX_RETRIES=5
export AUTOSPEC_SPECS_DIR="./features"
export AUTOSPEC_TIMEOUT=600
export AUTOSPEC_YES=true  # Skip confirmations
```

### Commands

```bash
# Initialize config
autospec init              # User-level
autospec init --project    # Project-level

# View config
autospec config show
autospec config show --json

# Migrate legacy JSON config
autospec config migrate
autospec config migrate --dry-run
```

## 🔧 Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Validation failed (retryable) |
| 2 | Retry limit exhausted |
| 3 | Invalid arguments |
| 4 | Missing dependencies |
| 5 | Command timeout |

Perfect for CI/CD integration:

```bash
autospec run -a "feature" && echo "✅ Success" || echo "❌ Failed: $?"
```

## 🐚 Shell Completion

The easiest way to set up shell completions:

```bash
# Auto-detect your shell and install completions
autospec completion install
```

Or install for a specific shell:

```bash
autospec completion install bash
autospec completion install zsh
autospec completion install fish
autospec completion install powershell
```

See [docs/SHELL-COMPLETION.md](docs/SHELL-COMPLETION.md) for detailed setup and manual instructions.

## 🔍 Troubleshooting

```bash
# First step: check dependencies
autospec doctor

# Debug mode
autospec --debug run -a "feature"

# View config
autospec config show
```

**Common issues:**

| Problem | Solution |
|---------|----------|
| `claude` not found | Install from [claude.ai/download](https://claude.ai/download) |
| Retry limit hit | Increase: `autospec run -a "feature" --max-retries 5` |
| Command timeout | Set `AUTOSPEC_TIMEOUT=600` or update config |
| Commands not found | Run `autospec init` to install commands and scripts |
| Claude permission denied | Allow commands in `~/.claude/settings.json` (see [troubleshooting](docs/troubleshooting.md#claude-permission-denied--command-blocked)) |

> ⚠️ **Note:** You can add `--dangerously-skip-permissions` to `claude_args` in config. Enable Claude's sandbox first (`/sandbox`)—uses [bubblewrap](https://github.com/containers/bubblewrap) on Linux. Bypasses ALL safety checks—never use with credentials or production data.

## 📝 Issue Templates

When creating issues, use our templates:

- **🐛 Bug Report** — For defects with reproduction steps
- **💡 Feature Request** — For new feature suggestions

Templates auto-apply labels and guide you through providing useful information.

## 🤝 Contributing

Contributions welcome! See [CONTRIBUTORS.md](CONTRIBUTORS.md) for development guidelines.

## 📄 License

MIT License — see [LICENSE](LICENSE) for details.

---

**📖 Documentation:** `autospec --help`

**🐛 Issues:** [github.com/ariel-frischer/autospec/issues](https://github.com/ariel-frischer/autospec/issues)

**⭐ Star us on GitHub if you find Autospec useful!**
