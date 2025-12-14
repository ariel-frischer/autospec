# Auto Claude SpecKit

Automated validation and workflow scripts for Claude Code's SpecKit feature development system.

## What is This?

Auto Claude SpecKit is a cross-platform Go binary and Claude Code hook system that automates the validation of SpecKit workflows. It ensures your feature specifications, plans, and tasks are complete before allowing Claude to stop, and automatically retries commands when artifacts are missing.

### Key Features

- **Automated Workflow Validation**: Runs `/speckit` commands with automatic retry when outputs are missing
- **Hook-Based Enforcement**: Prevents Claude from stopping until required artifacts exist
- **Phase Completion Detection**: Validates implementation progress in `tasks.md`
- **Continuation Prompts**: Automatically generates prompts to resume incomplete work
- **Performance Optimized**: Sub-second validation times (<10ms per validation)
- **Cross-Platform**: Native binaries for Linux, macOS, and Windows
- **Comprehensive Testing**: Unit tests, benchmarks, and integration tests in Go

## Quick Start

### Installation

#### Prerequisites

**Required:**
- Claude Code CLI (see https://www.claude.com/product/claude-code)
- SpecKit CLI (`uv tool install specify-cli --from git+https://github.com/github/spec-kit.git`)
- Git
- Go 1.21+ (for building from source)

**Optional:**
- jq (JSON processor for config manipulation)
- make (for using Makefile commands)

See [PREREQUISITES.md](PREREQUISITES.md) for detailed installation instructions and platform-specific requirements.

#### Auto Claude SpecKit Setup

##### Option 1: Install Pre-Built Binary (Recommended)

Download the latest release for your platform:

```bash
# Linux (amd64)
curl -L https://github.com/anthropics/auto-claude-speckit/releases/latest/download/autospec-linux-amd64 -o autospec
chmod +x autospec
sudo mv autospec /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/anthropics/auto-claude-speckit/releases/latest/download/autospec-darwin-amd64 -o autospec
chmod +x autospec
sudo mv autospec /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/anthropics/auto-claude-speckit/releases/latest/download/autospec-darwin-arm64 -o autospec
chmod +x autospec
sudo mv autospec /usr/local/bin/

# Verify installation
autospec version
```

**Binary size**: ~9.7 MB per platform
**Startup time**: <50ms
**Platforms**: Linux (amd64/arm64), macOS (amd64/arm64), Windows (amd64)

##### Option 2: Install from Source with Go

If you have Go 1.21+ installed:

```bash
go install github.com/anthropics/auto-claude-speckit/cmd/autospec@latest

# Verify installation
autospec version
```

##### Option 3: Build from Source

1. Clone this repository:
```bash
git clone https://github.com/anthropics/auto-claude-speckit.git
cd auto-claude-speckit
```

2. Build the binary:
```bash
# Build for your current platform
make build

# Or build for all platforms (Linux/macOS/Windows)
make build-all

# Install locally
make install

# Verify installation
autospec version
```

### Initial Setup

Before using the SpecKit commands in Claude Code, you must initialize your project with the Specify templates:

```bash
# Non-interactive initialization (recommended for automation)
specify init . --ai claude --force

# Interactive initialization (choose AI assistant)
specify init .

# Check that all required tools are installed
specify check
```

**What this does:**
- Downloads the latest SpecKit templates from GitHub
- Sets up `.claude/commands/speckit.*.md` slash commands
- Configures project structure for feature development
- Initializes git repository (if not already present)

**Options:**
- `--ai claude` - Skip interactive AI selection, use Claude
- `--force` - Skip confirmation if directory not empty
- `--no-git` - Skip git repository initialization
- `--ignore-agent-tools` - Skip checks for AI agent tools like Claude Code

After initialization, you can use the `/speckit.*` commands in Claude Code.

### Basic Usage

#### 1. Verify Dependencies

```bash
# Check that all required dependencies are installed
autospec doctor
```

#### 2. First-Time Setup

```bash
# Create project configuration
autospec init

# Initialize SpecKit templates (if not already done)
specify init . --ai claude --force
```

#### 3. Run Complete Workflow

**Option A: Full workflow (specify → plan → tasks → implement)**

The fastest way to go from idea to implemented feature:

```bash
autospec full "Add user authentication with OAuth support"
```

This automatically:
- Generates `specs/<feature-name>/spec.md` (specification)
- Creates `specs/<feature-name>/plan.md` (implementation plan)
- Produces `specs/<feature-name>/tasks.md` (actionable tasks)
- **Implements all tasks** via `/speckit.implement`
- Validates each artifact before proceeding
- Retries up to 3 times if any file is missing

**Option B: Workflow without implementation**

To generate planning artifacts without implementing:

```bash
autospec workflow "Add user authentication with OAuth support"
```

Then implement later with:
```bash
autospec implement
```

#### 4. Check Progress and Continue

```bash
# Check current implementation status
autospec status

# Continue implementing remaining tasks
autospec implement
```

#### 5. Advanced Options

```bash
# Custom retry limit
autospec workflow "feature description" --max-retries 5

# Show progress indicators (spinners) during execution
autospec full "feature" --progress

# Dry-run mode (see what would execute)
autospec workflow "feature" --dry-run

# Debug mode with verbose logging
autospec --debug workflow "feature"

# View current configuration
autospec config show
```

**Progress Indicators**: Off by default due to Claude output stream pollution. Enable with `--progress` flag or set `"show_progress": true` in `.autospec/config.json`.

#### 6. Hook-Based Automatic Validation

Enable hooks to prevent Claude from stopping until artifacts are complete. See [CONTRIBUTORS.md](CONTRIBUTORS.md) for details on hook configuration.

## Use Cases

### 1. End-to-End Automated Feature Development

```bash
# Run complete workflow from idea to implementation in one command
autospec full "Add user authentication with OAuth"

# Claude generates spec.md, plan.md, tasks.md and implements everything
# If any file is missing or tasks incomplete, workflow automatically retries
# All phases validated before proceeding
```

### 2. Phased Feature Development

```bash
# Start workflow with automatic validation (planning only)
autospec workflow "Add user authentication with OAuth"

# Claude generates spec.md, plan.md, tasks.md with automatic retries
# If any file is missing, workflow automatically re-prompts Claude

# Check implementation status
autospec status

# Continue implementing remaining tasks
autospec implement
```

### 3. Hook-Enforced Completeness

```bash
# Enable implementation hook
claude --settings .claude/implement-hook-settings.json

# Claude cannot stop until all tasks in tasks.md are checked
# Hook automatically retries up to 3 times
# Provides clear feedback about remaining work
```

### 4. CI/CD Integration

```bash
# Validate feature completion in CI pipeline
autospec status --json | jq -e '.status == "COMPLETE"'
if [ $? -eq 0 ]; then
  echo "Feature implementation complete"
  exit 0
else
  echo "Feature implementation incomplete"
  autospec status  # Show details
  exit 1
fi
```

### 5. Working with Configuration

```bash
# Initialize project configuration
autospec init

# View current configuration
autospec config show

# Use custom specs directory
autospec --specs-dir ./features workflow "new feature"

# Adjust retry limits
autospec workflow "feature" --max-retries 5
```

## Configuration

Autospec uses configuration files for customization:

- **Local config**: `.autospec/config.json` (project-specific)
- **Global config**: `~/.autospec/config.json` (user-wide)
- **Environment variables**: `AUTOSPEC_*` prefix

Common settings:
```bash
# Initialize config
autospec init

# View current config
autospec config show

# Use environment variables
export AUTOSPEC_MAX_RETRIES=5
export AUTOSPEC_SPECS_DIR="./features"
export AUTOSPEC_TIMEOUT=600  # 10-minute timeout for commands
```

### Timeout Configuration

Prevent indefinite command hangs with configurable timeouts:

```bash
# Set 10-minute timeout via environment variable
export AUTOSPEC_TIMEOUT=600

# Or in config file
echo '{"timeout": 600}' > .autospec/config.json
```

When a command exceeds the timeout, it's terminated and returns exit code 5. See [docs/TIMEOUT.md](docs/TIMEOUT.md) for detailed configuration options.

### Shell Completion

Enable tab completion for faster CLI usage:

```bash
# Generate zsh completion
mkdir -p ~/.zsh_completions
autospec completion zsh > ~/.zsh_completions/_autospec

# Add to ~/.zshrc (before compinit):
fpath=(~/.zsh_completions $fpath)
autoload -U compinit
compinit

# Reload shell
exec zsh
```

Supports bash, zsh, fish, and powershell. See [docs/SHELL-COMPLETION.md](docs/SHELL-COMPLETION.md) for detailed setup instructions and troubleshooting.

See [CONTRIBUTORS.md](CONTRIBUTORS.md) for detailed configuration options.

## Troubleshooting

**First step: Check your dependencies**

```bash
autospec doctor
```

This will verify that Claude CLI, Specify CLI, and Git are installed and available.

**Common issues:**

- **"Command not found: specify"** - Install SpecKit CLI: `uv tool install specify-cli --from git+https://github.com/github/spec-kit.git`
- **"Command not found: claude"** - Install Claude Code from https://claude.com/product/claude-code
- **"Retry limit exhausted"** - Increase retries: `autospec workflow "feature" --max-retries 5`
- **Build fails** - Install Go 1.21+: https://go.dev/doc/install

See [PREREQUISITES.md](PREREQUISITES.md) for detailed installation instructions.

## Issue Templates

This repository provides structured issue templates to help contributors submit high-quality bug reports and feature requests.

### Available Templates

When creating a new issue on GitHub, you'll be prompted to choose from:

- **Bug Report**: For reporting defects or unexpected behavior
  - Includes sections for reproduction steps, expected vs actual behavior, and environment details
  - Auto-applies labels: `bug`, `needs-triage`

- **Feature Request**: For suggesting new features or enhancements
  - Focuses on problem statements and use cases rather than implementation details
  - Auto-applies labels: `enhancement`, `needs-discussion`

### Template Configuration

Templates are configured to:
- Disable blank issues (all issues must use a template)
- Provide links to community discussions and documentation
- Auto-apply labels for efficient triage

### For Maintainers

Template files are located in `.github/ISSUE_TEMPLATE/`:
- `bug_report.md` - Bug report template
- `feature_request.md` - Feature request template
- `config.yml` - Template configuration

To validate templates before committing:

```bash
# Run validation tests
./tests/github_templates/validate_all.sh

# Or validate individually
source tests/lib/validation_lib.sh
validate_all_templates .github/ISSUE_TEMPLATE
```

## Contributing

Contributions welcome! See [CONTRIBUTORS.md](CONTRIBUTORS.md) for development guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.

---

**Documentation:** Run `autospec --help` for CLI usage

**Issues:** https://github.com/anthropics/auto-claude-speckit/issues
