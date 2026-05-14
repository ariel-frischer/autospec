---
title: Worktree Management
parent: Guides
nav_order: 4
---

# Worktree Management

The `autospec worktree` command enables **parallel agent execution** by creating isolated filesystem directories for each feature. This allows multiple agent sessions to work simultaneously on different features without branch conflicts or file contention.

## Why Worktrees?

When running `autospec run -a`, the agent creates and checks out a new feature branch. Running multiple agents in the same directory causes:
- **Branch conflicts**: Agents fight over which branch is checked out
- **File contention**: Concurrent file edits corrupt work
- **Spec collisions**: Multiple `specs/` directories interfere

Git worktrees solve this by giving each agent its own complete working directory, all sharing the same repository.

**The problem:** Standard `git worktree add` doesn't copy non-tracked directories (`.autospec/`, `.agents/`, `.claude/`) or run project setup (npm install, etc.).

**The solution:** `autospec worktree create` handles everything automatically.

---

## Quick Start

```bash
# Create a new worktree with automatic setup
autospec worktree create feature-auth --branch feat/user-auth

# List all tracked worktrees
autospec worktree list

# Remove a worktree
autospec worktree remove feature-auth
```

---

## Commands

### create

Create a new git worktree with automatic project configuration.

```bash
autospec worktree create <name> --branch <branch> [--path <path>]
```

**Flags:**
- `--branch, -b` (required): Branch name for the worktree
- `--path, -p` (optional): Custom path for the worktree

**What it does:**
1. Creates a new git worktree using `git worktree add`
2. Copies configured directories (for example `.autospec/`, `.agents/`, `.claude/`) to the new worktree
3. Runs the project setup script if configured
4. Tracks the worktree in `.autospec/state/worktrees.yaml`

**Examples:**
```bash
# Create with a new branch
autospec worktree create zoom --branch feat/zoom-engine

# Create at a custom location
autospec worktree create zoom --branch feat/zoom --path /tmp/zoom-dev
```

### list

List all tracked worktrees with their status.

```bash
autospec worktree list
```

**Output columns:**
- **NAME**: The worktree identifier
- **PATH**: Filesystem path to the worktree
- **BRANCH**: Git branch checked out
- **STATUS**: Current state (active, merged, abandoned, stale)
- **CREATED**: When the worktree was created

**Status meanings:**

| Status | Meaning |
|--------|---------|
| `active` | Worktree is in active use |
| `merged` | Branch has been merged |
| `abandoned` | Work was abandoned |
| `stale` | Worktree path no longer exists |

### remove

Remove a tracked worktree with safety checks.

```bash
autospec worktree remove <name> [--force]
```

**Flags:**
- `--force, -f`: Bypass safety checks

{: .warning }
By default, removal is blocked if the worktree has uncommitted changes or unpushed commits. Use `--force` only when intentionally discarding work.

**Examples:**
```bash
# Safe removal (checks for uncommitted/unpushed work)
autospec worktree remove feature-auth

# Force removal (bypasses safety checks)
autospec worktree remove feature-auth --force
```

### setup

Run project setup on an existing worktree.

```bash
autospec worktree setup <path> [--track]
```

**Flags:**
- `--track`: Add the worktree to tracking state

Use this command to configure worktrees that weren't created with `autospec worktree create`, such as manually created git worktrees.

**Example:**
```bash
# Setup an existing worktree
autospec worktree setup ../my-worktree

# Setup and track
autospec worktree setup ../my-worktree --track
```

### gen-script

Generate a project-specific worktree setup script using the configured agent.

```bash
autospec worktree gen-script [--include-env]
```

**Flags:**
- `--include-env`: Include `.env` files in the copy list (security warning displayed)

**What it does:**
1. Analyzes your project to detect package managers and configuration
2. Generates a customized `setup-worktree.sh` script using the configured agent
3. Saves the script to `.autospec/scripts/setup-worktree.sh` (executable)

**Generated script behavior:**
- Copies essential directories such as `.autospec/`, agent config directories, and `.vscode/` (if present)
- Excludes secrets by default: `.env*`, `credentials.*`, `*.pem`, `*.key`
- Runs package manager install commands instead of copying dependencies:
  - `npm install` / `yarn install` / `pnpm install` for Node.js projects
  - `go mod download` for Go projects
  - `pip install -r requirements.txt` or `poetry install` for Python projects
  - `cargo build` for Rust projects

{: .note }
By default, the generated script excludes all environment and secret files. Use `--include-env` only if you understand the security implications.

### prune

Remove stale worktree tracking entries.

```bash
autospec worktree prune
```

This command removes tracking entries for worktrees whose paths no longer exist on disk. It's useful after manually deleting worktree directories.

{: .note }
This only removes tracking entries - it does not delete any files.

---

## Configuration

Add worktree configuration to your `.autospec/config.yml`:

```yaml
worktree:
  # Parent directory for new worktrees (default: parent of repo)
  base_dir: ""

  # Directory name prefix (e.g., "wt-" creates "wt-feature-name")
  prefix: ""

  # Path to setup script relative to repo
  setup_script: ""

  # Run setup automatically on create (default: true)
  auto_setup: true

  # Persist worktree state (default: true)
  track_status: true

  # Non-tracked directories to copy
  copy_dirs:
    - .autospec
    - .agents
    - .claude
```

### Environment Variables

All configuration options can be set via environment variables with the `AUTOSPEC_WORKTREE_` prefix:

```bash
export AUTOSPEC_WORKTREE_BASE_DIR="/path/to/worktrees"
export AUTOSPEC_WORKTREE_PREFIX="wt-"
export AUTOSPEC_WORKTREE_SETUP_SCRIPT="scripts/setup.sh"
```

---

## Setup Script

The setup script receives the following information:

**Arguments:**
1. `$1` - Worktree path
2. `$2` - Worktree name
3. `$3` - Branch name

**Environment Variables:**
- `WORKTREE_PATH` - Absolute path to the worktree
- `WORKTREE_NAME` - Name of the worktree
- `WORKTREE_BRANCH` - Branch checked out
- `SOURCE_REPO` - Path to the source repository

**Example setup script:**

```bash
#!/bin/bash
set -e

cd "$WORKTREE_PATH"

echo "Setting up: $WORKTREE_NAME ($WORKTREE_BRANCH)"

# Install dependencies
npm install

# Copy local environment
cp "$SOURCE_REPO/.env.local" .env.local 2>/dev/null || true

echo "Setup complete!"
```

---

## State File

Worktree state is persisted to `.autospec/state/worktrees.yaml`:

```yaml
version: "1.0.0"
worktrees:
  - name: feature-auth
    path: /home/user/repos/project-feature-auth
    branch: feat/user-auth
    status: active
    created_at: 2024-01-15T10:30:00Z
    setup_completed: true
    last_accessed: 2024-01-15T14:20:00Z
```

---

## Troubleshooting

### "worktree already exists"

A worktree with that name is already tracked. Use a different name or remove the existing worktree first.

### "uncommitted changes" or "unpushed commits"

The worktree has work that hasn't been committed or pushed. Either commit/push your changes, or use `--force` to discard them.

### "path does not exist"

For `setup` command: Verify the path is correct and the worktree exists.

For tracking entries: Run `autospec worktree prune` to clean up stale entries.

### Setup script failed

Check the script output for errors. You can retry setup with:
```bash
autospec worktree setup /path/to/worktree
```

### Worktree shows as "stale"

The worktree directory was deleted outside of autospec. Run `autospec worktree prune` to remove the tracking entry.

---

## See Also

- [Sequential & Parallel Execution](parallel-execution) - Running multiple workflows with git worktrees
- [CLI Reference](../reference/cli) - Complete command documentation
- [Configuration](../reference/configuration) - Project and user configuration
