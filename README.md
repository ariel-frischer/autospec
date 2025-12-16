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
| Context Efficiency | Full prompt each time | **Smart YAML injection** + **phase-isolated sessions** |
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

# Create project constitution (triggers Claude session)
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

# 🔸 Single-session: All tasks in one session (legacy mode)
autospec implement --single-session
```

| Mode | Flag | Isolation | Use Case |
|------|------|-----------|----------|
| Phase | (default) | 1 session per phase | Balanced cost/context |
| Task | `--tasks` | 1 session per task | Complex tasks, max isolation |
| Single | `--single-session` | 1 session for all | Small specs, simple tasks |

> 📌 `--tasks`, `--phases`, and `--single-session` are mutually exclusive. Task-level execution respects dependency order and validates each task completes before proceeding.

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

Claude automatically updates task status during implementation. Manual updates are also available:

```bash
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

### All Settings

```yaml
# .autospec/config.yml
claude_cmd: claude                    # Claude CLI command
claude_args:                          # Arguments passed to Claude CLI
  - -p
  - --verbose
  - --output-format
  - stream-json
custom_claude_cmd: ""                 # Custom command (overrides claude_cmd + claude_args)
max_retries: 0                        # Max retry attempts (0-10)
specs_dir: ./specs                    # Directory for feature specs
state_dir: ~/.autospec/state          # Directory for state files
skip_preflight: false                 # Skip preflight checks
timeout: 2400                         # Timeout in seconds (40 min default, 0 = no timeout)
skip_confirmations: false             # Skip confirmation prompts
implement_method: phases              # Default: phases | tasks | single-session

# Notifications (all platforms)
notifications:
  enabled: false                      # Enable notifications (opt-in)
  type: both                          # sound | visual | both
  sound_file: ""                      # Custom sound file (empty = system default)
  on_command_complete: true           # Notify when command finishes
  on_stage_complete: false            # Notify on each stage
  on_error: true                      # Notify on failures
  on_long_running: false              # Notify after threshold
  long_running_threshold: 30s         # Duration threshold
```

### Environment Variables

```bash
export AUTOSPEC_MAX_RETRIES=0      # Default: 0 (no retries)
export AUTOSPEC_SPECS_DIR="./specs" # Default: ./specs
export AUTOSPEC_TIMEOUT=2400        # Default: 2400 (40 minutes)
export AUTOSPEC_YES=false           # Default: false (prompts enabled)
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

Uses standardized exit codes (0-5) for CI/CD integration. See [docs/reference.md](docs/reference.md#exit-codes) for full details.

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

See [docs/troubleshooting.md](docs/troubleshooting.md) for common issues and solutions.

## 💡 Pro Tips

### Readable Streaming Output with claude-clean

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

## 🤝 Contributing

Contributions welcome! See [CONTRIBUTORS.md](CONTRIBUTORS.md) for development guidelines.

## 📄 License

MIT License — see [LICENSE](LICENSE) for details.

---

**📖 Documentation:** `autospec --help`

**🐛 Issues:** [github.com/ariel-frischer/autospec/issues](https://github.com/ariel-frischer/autospec/issues)

**⭐ Star us on GitHub if you find Autospec useful!**
