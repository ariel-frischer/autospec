# 🚀 autospec

[![CI](https://github.com/ariel-frischer/autospec/actions/workflows/ci.yml/badge.svg)](https://github.com/ariel-frischer/autospec/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ariel-frischer/autospec)](https://goreportcard.com/report/github.com/ariel-frischer/autospec)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/ariel-frischer/autospec)](https://github.com/ariel-frischer/autospec/releases/latest)

**Automated feature development workflows with structured YAML output for AI-powered code generation.**

Inspired by [GitHub SpecKit](https://github.com/github/spec-kit), autospec reimagines the specification workflow with **YAML-first artifacts** for programmatic access, validation, and CI/CD integration.

## ✨ What Makes autospec Different?

Originally inspired by [GitHub SpecKit](https://github.com/github/spec-kit), autospec is now a **fully standalone tool** with its own embedded commands and workflows.

| Feature | GitHub SpecKit | autospec |
|---------|---------------|----------|
| Output Format | Markdown | **YAML** (machine-readable) |
| Validation | Manual review | **Automatic** with retry logic |
| CI/CD Integration | Limited | **First-class** with exit codes |
| Phase Orchestration | Manual | **Automated** with dependencies |
| Progress Tracking | None | **Built-in** status & task updates |
| Dependencies | Requires SpecKit CLI | **Self-contained** (only needs Claude CLI) |

## 🎯 Key Features

- 🔄 **Automated Workflow Orchestration** — Runs phases in dependency order with automatic retry on failure
- 📝 **YAML-First Artifacts** — Machine-readable `spec.yaml`, `plan.yaml`, `tasks.yaml` for programmatic access
- ✅ **Smart Validation** — Validates artifacts exist and meet completeness criteria before proceeding
- 🔁 **Configurable Retry Logic** — Automatic retries with persistent state tracking
- ⚡ **Performance Optimized** — Sub-second validation (<10ms per check), <50ms startup
- 🖥️ **Cross-Platform** — Native binaries for Linux, macOS (Intel/Apple Silicon), and Windows
- 🎛️ **Flexible Phase Selection** — Mix and match phases with intuitive flags (`-spti`, `-a`, etc.)
- 🏗️ **Constitution Support** — Project-level principles that guide all specifications
- 🔍 **Cross-Artifact Analysis** — Consistency checks across spec, plan, and tasks
- 📋 **Custom Checklists** — Auto-generated validation checklists per feature
- 🧪 **Comprehensive Testing** — Unit tests, benchmarks, and integration tests
- 🐚 **Shell Completion** — Tab completion for bash, zsh, fish, and PowerShell

## 📦 Quick Start

### Prerequisites

**Required:**
- [Claude Code CLI](https://claude.ai/download)
- Git

**Optional:**
- Go 1.21+ (for building from source)
- make (for Makefile commands)

### Installation

#### Option 1: Pre-Built Binary (Recommended)

```bash
# Linux (amd64)
curl -L https://github.com/ariel-frischer/autospec/releases/latest/download/autospec-linux-amd64 -o autospec
chmod +x autospec && sudo mv autospec /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/ariel-frischer/autospec/releases/latest/download/autospec-darwin-arm64 -o autospec
chmod +x autospec && sudo mv autospec /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/ariel-frischer/autospec/releases/latest/download/autospec-darwin-amd64 -o autospec
chmod +x autospec && sudo mv autospec /usr/local/bin/

# Verify
autospec version
```

#### Option 2: Go Install

```bash
go install github.com/ariel-frischer/autospec/cmd/autospec@latest
```

#### Option 3: Build from Source

```bash
git clone https://github.com/ariel-frischer/autospec.git
cd autospec
make build && make install
```

### Initialize Your Project

```bash
# Check dependencies
autospec doctor

# Initialize autospec (config, commands, and scripts)
autospec init
```

## 🎮 Usage

### Flexible Phase Selection with `run`

```bash
# 🚀 Run all core phases (specify → plan → tasks → implement)
autospec run -a "Add user authentication with OAuth"

# 📝 Run specific phases
autospec run -sp "Add caching layer"        # Specify + plan only
autospec run -ti --spec 007-feature         # Tasks + implement on specific spec
autospec run -p "Focus on security"         # Plan with guidance

# ✨ Include optional phases
autospec run -sr "Add payments"             # Specify + clarify
autospec run -a -l                          # All + checklist
autospec run -tlzi                          # Tasks + checklist + analyze + implement

# 🏃 Skip confirmations for automation
autospec run -a -y "Feature description"
```

### Phase Flags Reference

| Flag | Phase | Description |
|------|-------|-------------|
| `-s` | specify | Generate feature specification |
| `-p` | plan | Generate implementation plan |
| `-t` | tasks | Generate task breakdown |
| `-i` | implement | Execute implementation |
| `-a` | all | All core phases (`-spti`) |
| `-n` | constitution | Create/update project constitution |
| `-r` | clarify | Refine spec with Q&A |
| `-l` | checklist | Generate validation checklist |
| `-z` | analyze | Cross-artifact consistency check |

> 📌 Phases always execute in canonical order regardless of flag order:
> `constitution → specify → clarify → plan → tasks → checklist → analyze → implement`

### Shortcut Commands

```bash
# 🎯 Complete workflow (all phases)
autospec all "Add feature description"

# 📋 Planning only (no implementation)
autospec workflow "Add feature description"

# 🔨 Implementation only
autospec implement
autospec implement 003-feature "Focus on tests"

# 📊 Check status
autospec status
```

### Optional Phase Commands

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

autospec generates structured YAML artifacts:

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

```bash
# Zsh
autospec completion zsh > ~/.zsh_completions/_autospec

# Bash
autospec completion bash > /etc/bash_completion.d/autospec

# Fish
autospec completion fish > ~/.config/fish/completions/autospec.fish
```

See [docs/SHELL-COMPLETION.md](docs/SHELL-COMPLETION.md) for detailed setup.

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

> ⚠️ **Note:** You can add `--dangerously-skip-permissions` to `claude_args` in config, but sandbox recommended. Bypasses ALL safety checks—never use with credentials or production data.

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

**⭐ Star us on GitHub if you find autospec useful!**
