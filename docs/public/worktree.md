# Worktree Management

The `autospec worktree` command enables **parallel agent execution** by creating isolated filesystem directories for each feature. This allows multiple agent sessions to work simultaneously on different features without branch conflicts or file contention.

## Why Worktrees?

When running `autospec run -a`, the agent creates and checks out a new feature branch. Running multiple agents in the same directory causes:
- **Branch conflicts**: Agents fight over which branch is checked out
- **File contention**: Concurrent file edits corrupt work
- **Spec collisions**: Multiple `specs/` directories interfere

Git worktrees solve this by giving each agent its own complete working directory, all sharing the same repository.

Git still prevents the same branch from being checked out in more than one worktree at a time. Use separate branch names for separate worktrees, even when those branches are collaborating on the same autospec feature directory.

**The problem:** Standard `git worktree add` doesn't copy non-tracked directories (`.autospec/`, `.claude/`, `.codex/`, `.opencode/`) or run project setup (npm install, etc.).

**The solution:** `autospec worktree create` handles everything automatically.

## Active Feature Resolution

Autospec workflow commands choose the active feature directory in a fixed order:

1. **Explicit selection**: a command argument or `--spec <feature>` wins for that invocation.
2. **Persisted active feature**: project-local state under `.autospec/state/` can remember the selected feature directory across later commands.
3. **Branch-prefix fallback**: when no explicit or persisted feature is available, autospec falls back to the existing branch-prefix lookup such as `128-my-feature` → `specs/128-*`.

This lets a worktree on `feat/refactor-active-feature` continue work on `specs/128-persist-feature-directory` after that feature has been selected or created, without requiring the branch suffix to match the spec directory name. It does not bypass Git's same-branch worktree restriction; each worktree still needs a branch Git can check out independently.

Use `--spec` when automation must target a specific feature regardless of saved state:

```bash
autospec run -ti --spec 128-persist-feature-directory
autospec implement 128-persist-feature-directory
autospec status 128-persist-feature-directory
```

If persisted state points at a deleted or invalid directory, autospec reports the selected directory and why it cannot be used. Recover by selecting an existing feature explicitly with `--spec`, running from a matching feature branch so branch-prefix fallback can resolve a valid spec, or removing the stale project-local active feature state from `.autospec/state/` before retrying.

## Quick Start

```bash
# Create a new worktree with automatic setup
autospec worktree create feature-auth --branch feat/user-auth

# List all tracked worktrees
autospec worktree list

# Remove a worktree
autospec worktree remove feature-auth
```

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
2. Copies configured directories (for example `.autospec/`, `.claude/`, `.codex/`, `.opencode/`) to the new worktree
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
- `active`: Worktree is in active use
- `merged`: Branch has been merged
- `abandoned`: Work was abandoned
- `stale`: Worktree path no longer exists

### remove

Remove a tracked worktree with safety checks.

```bash
autospec worktree remove <name> [--force]
```

**Flags:**
- `--force, -f`: Bypass safety checks

**Safety checks:**
By default, removal is blocked if the worktree has:
- Uncommitted changes
- Unpushed commits

Use `--force` to bypass these checks when intentionally discarding work.

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

**Examples:**
```bash
# Generate a setup script
autospec worktree gen-script

# Include environment files (use with caution)
autospec worktree gen-script --include-env
```

**Security note:**
By default, the generated script excludes all environment and secret files. Use `--include-env` only if you understand the security implications and need environment files copied to worktrees.

### prune

Remove stale worktree tracking entries.

```bash
autospec worktree prune
```

This command removes tracking entries for worktrees whose paths no longer exist on disk. It's useful after manually deleting worktree directories.

**Note:** This only removes tracking entries - it does not delete any files.

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
    - .claude
    - .codex
    - .opencode
```

### Environment Variables

All configuration options can be set via environment variables with the `AUTOSPEC_WORKTREE_` prefix:

```bash
export AUTOSPEC_WORKTREE_BASE_DIR="/path/to/worktrees"
export AUTOSPEC_WORKTREE_PREFIX="wt-"
export AUTOSPEC_WORKTREE_SETUP_SCRIPT="scripts/setup.sh"
```

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
