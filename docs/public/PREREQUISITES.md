# Prerequisites

## Required Dependencies

### 1. Supported CLI Coding Agent (Required)

autospec orchestrates feature development workflows using a supported CLI coding agent. Production-supported agents are Claude Code, Codex, and OpenCode.

**Installation:**
- Claude Code: https://claude.ai/download
- Codex CLI: https://developers.openai.com/codex/cli/reference
- OpenCode: https://opencode.ai

**Verify installation:**
```bash
claude --version
# or
codex --version
# or
opencode --version
```

### 2. Git (Required)

Git is required for spec detection from branch names and general version control.

**Verify installation:**
```bash
git --version
```

### 3. Platform-Specific Requirements

**Linux:**
- Git package from your distribution's package manager

**macOS:**
- Xcode Command Line Tools (includes git): `xcode-select --install`

**Windows:**
- Use [WSL (Windows Subsystem for Linux)](https://learn.microsoft.com/en-us/windows/wsl/install) and follow the Linux instructions

## Verification

Run the doctor command to verify all dependencies:

```bash
autospec doctor
```

Or verify manually:

```bash
# Verify your CLI agent
claude --version
# or
codex --version
# or
opencode --version

# Verify Git
git --version
```

## Installing Missing Dependencies

### CLI Coding Agent

**Installation guide:**
See the installation links above for Claude Code, Codex CLI, or OpenCode.

### Git

**Ubuntu/Debian:**
```bash
sudo apt-get install git
```

**Arch Linux:**
```bash
sudo pacman -S git
```

**Fedora:**
```bash
sudo dnf install git
```

**macOS:**
```bash
xcode-select --install
```

**Windows (via WSL):**
Use [WSL](https://learn.microsoft.com/en-us/windows/wsl/install), then install git with your Linux distribution's package manager

---

## Development Setup (Contributing)

If you want to contribute to autospec or build from source, you'll need additional tools.

### Go (1.21+)

**Ubuntu/Debian:**
```bash
# Download from https://go.dev/dl/ or use snap
sudo snap install go --classic
```

**Arch Linux:**
```bash
sudo pacman -S go
```

**Fedora:**
```bash
sudo dnf install golang
```

**macOS:**
```bash
brew install go
```

**Verify:**
```bash
go version
```

### golangci-lint (for linting)

**All platforms (via Go):**
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

**Arch Linux (AUR):**
```bash
yay -S golangci-lint
# or
paru -S golangci-lint
```

**Fedora:**
```bash
sudo dnf install golangci-lint
```

**macOS:**
```bash
brew install golangci-lint
```

**Verify:**
```bash
golangci-lint --version
```

### Claude Code CLI (Alternative Install Methods)

**Native installer (recommended for all Linux):**
```bash
curl -fsSL https://claude.ai/install.sh | bash
```

**Arch Linux (AUR):**
```bash
yay -S claude-code
# or
paru -S claude-code
```

**Via npm (requires Node.js):**
```bash
# First install Node.js
# Arch: sudo pacman -S nodejs npm
# Fedora: sudo dnf install nodejs npm
# Ubuntu: sudo apt install nodejs npm

npm install -g @anthropic-ai/claude-code
```

### Build & Test

Once dependencies are installed:

```bash
make build    # Build the binary
make test     # Run tests
make lint     # Run linters
make fmt      # Format code
```

## Note on GitHub SpecKit

autospec was originally inspired by [GitHub SpecKit](https://github.com/github/spec-kit), but is now a **fully standalone tool**. The command templates in `internal/commands/templates/` were initially generated using SpecKit's tooling, but autospec now embeds and manages these commands independently.

**You do NOT need to install SpecKit** - autospec includes everything needed out of the box.
