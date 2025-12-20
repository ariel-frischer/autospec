# Worktree Management Command

A new `autospec worktree` command for managing git worktrees with project-aware setup automation.

---

## Problem Statement

When running multiple `autospec all` commands in parallel across worktrees:
1. `.autospec/` directory is not git-tracked, so `git worktree add` doesn't copy it
2. `.claude/commands/` may also be untracked
3. Project-specific setup (npm install, etc.) must be run manually
4. No central tracking of which worktrees exist and their status

Users currently must manually:
```bash
git worktree add ../project-feature feat/feature-name
cp -r .autospec ../project-feature/
cp -r .claude ../project-feature/
cd ../project-feature && autospec init && npm install
```

---

## Proposed Solution

### Command Structure

```bash
autospec worktree <subcommand> [options]
```

| Subcommand | Description |
|------------|-------------|
| `create <name>` | Create worktree with setup |
| `list` | Show all tracked worktrees |
| `remove <name>` | Remove worktree cleanly |
| `setup [path]` | Run setup on existing worktree |
| `prune` | Remove stale worktree entries |

### Usage Examples

```bash
# Create worktree for a feature
autospec worktree create zoom --branch feat/zoom-engine
# Output: Created worktree 'zoom' at ../autospec-zoom on branch feat/zoom-engine
#         Running setup script... done.

# Create with custom path
autospec worktree create zoom --branch feat/zoom --path /tmp/zoom-dev

# List all worktrees
autospec worktree list
# NAME     PATH                 BRANCH              STATUS    CREATED
# zoom     ../autospec-zoom     feat/zoom-engine    active    2h ago
# portal   ../autospec-portal   feat/portal         merged    1d ago

# Remove worktree (with safety)
autospec worktree remove zoom
# Output: Worktree 'zoom' has uncommitted changes. Use --force to remove anyway.

autospec worktree remove zoom --force
# Output: Removed worktree 'zoom'

# Run setup on manually-created worktree
autospec worktree setup ../my-worktree

# Clean up stale entries
autospec worktree prune
# Removed 2 stale worktree entries
```

---

## Configuration

### Project Setup Script

Projects define a setup script that runs after worktree creation:

```bash
# .autospec/scripts/worktree-setup.sh
#!/bin/bash
# Called with: $1=worktree_path, $2=worktree_name, $3=branch_name
set -e

WORKTREE_PATH="$1"
WORKTREE_NAME="$2"
BRANCH_NAME="$3"

# Copy non-git-tracked configs
cp -r .autospec "$WORKTREE_PATH/"
cp -r .claude "$WORKTREE_PATH/" 2>/dev/null || true

# Run autospec init if needed
(cd "$WORKTREE_PATH" && autospec init --quiet)

# Project-specific setup
if [ -f "$WORKTREE_PATH/package.json" ]; then
    (cd "$WORKTREE_PATH" && npm install)
fi

if [ -f "$WORKTREE_PATH/go.mod" ]; then
    (cd "$WORKTREE_PATH" && go mod download)
fi

echo "Setup complete for worktree '$WORKTREE_NAME'"
```

### Config Options

```yaml
# .autospec/config.yml
worktree:
  base_dir: "../"              # Default: parent of repo
  prefix: "wt-"                # Worktree directory prefix
  setup_script: ".autospec/scripts/worktree-setup.sh"
  auto_setup: true             # Run setup after create
  track_status: true           # Track in worktrees.yaml
```

---

## State Management

### Worktree State File

Tracked worktrees stored in `.autospec/state/worktrees.yaml`:

```yaml
version: "1.0"

worktrees:
  - name: "zoom"
    path: "/home/user/projects/autospec-zoom"
    branch: "feat/zoom-engine"
    created_at: "2025-01-15T10:00:00Z"
    status: "active"
    setup_completed: true
    last_accessed: "2025-01-15T14:30:00Z"

  - name: "portal"
    path: "/home/user/projects/autospec-portal"
    branch: "feat/portal"
    created_at: "2025-01-14T08:00:00Z"
    status: "merged"
    setup_completed: true
    merged_at: "2025-01-15T09:00:00Z"
```

### Status Values

| Status | Description |
|--------|-------------|
| `active` | Worktree exists and in use |
| `merged` | Branch merged, worktree may be removed |
| `abandoned` | User marked as not needed |
| `stale` | Worktree path no longer exists |

---

## Package Architecture

```
internal/
└── worktree/
    ├── manager.go      # Core CRUD operations
    ├── state.go        # State persistence
    ├── setup.go        # Setup script execution
    └── types.go        # Struct definitions

internal/cli/orchestration/
└── worktree.go         # CLI command implementation
```

### Key Types

```go
// internal/worktree/types.go
package worktree

type Worktree struct {
    Name           string    `yaml:"name"`
    Path           string    `yaml:"path"`
    Branch         string    `yaml:"branch"`
    CreatedAt      time.Time `yaml:"created_at"`
    Status         Status    `yaml:"status"`
    SetupCompleted bool      `yaml:"setup_completed"`
    LastAccessed   time.Time `yaml:"last_accessed,omitempty"`
    MergedAt       time.Time `yaml:"merged_at,omitempty"`
}

type Status string

const (
    StatusActive    Status = "active"
    StatusMerged    Status = "merged"
    StatusAbandoned Status = "abandoned"
    StatusStale     Status = "stale"
)
```

### Manager Interface

```go
// internal/worktree/manager.go
package worktree

type Manager interface {
    // Create creates a new worktree and runs setup
    Create(name, branch string, opts CreateOptions) (*Worktree, error)

    // List returns all tracked worktrees
    List() ([]Worktree, error)

    // Get retrieves a worktree by name
    Get(name string) (*Worktree, error)

    // Remove deletes a worktree and its tracking entry
    Remove(name string, force bool) error

    // Setup runs the setup script on a worktree
    Setup(wt *Worktree) error

    // Prune removes stale worktree entries
    Prune() ([]string, error)

    // UpdateStatus updates a worktree's status
    UpdateStatus(name string, status Status) error
}

type CreateOptions struct {
    Path      string // Custom path (default: baseDir/prefix+name)
    SkipSetup bool   // Skip running setup script
}
```

---

## Implementation Details

### Create Worktree Flow

```go
func (m *manager) Create(name, branch string, opts CreateOptions) (*Worktree, error) {
    // 1. Validate name doesn't exist
    if existing, _ := m.Get(name); existing != nil {
        return nil, fmt.Errorf("worktree %q already exists", name)
    }

    // 2. Determine path
    path := opts.Path
    if path == "" {
        path = filepath.Join(m.cfg.BaseDir, m.cfg.Prefix+name)
    }

    // 3. Create git worktree
    //    git worktree add <path> -b <branch> OR
    //    git worktree add <path> <existing-branch>
    if err := m.gitWorktreeAdd(path, branch); err != nil {
        return nil, fmt.Errorf("git worktree add: %w", err)
    }

    // 4. Create state entry
    wt := &Worktree{
        Name:      name,
        Path:      path,
        Branch:    branch,
        CreatedAt: time.Now(),
        Status:    StatusActive,
    }

    // 5. Run setup (unless skipped)
    if !opts.SkipSetup && m.cfg.AutoSetup {
        if err := m.Setup(wt); err != nil {
            // Log but don't fail - worktree exists
            m.logger.Warn("setup script failed", "error", err)
        } else {
            wt.SetupCompleted = true
        }
    }

    // 6. Save state
    if err := m.state.Add(wt); err != nil {
        return nil, fmt.Errorf("saving state: %w", err)
    }

    return wt, nil
}
```

### Remove Safety Checks

```go
func (m *manager) Remove(name string, force bool) error {
    wt, err := m.Get(name)
    if err != nil {
        return err
    }

    // Check for uncommitted changes
    if !force {
        if hasChanges, err := m.hasUncommittedChanges(wt.Path); err != nil {
            return fmt.Errorf("checking for changes: %w", err)
        } else if hasChanges {
            return fmt.Errorf("worktree %q has uncommitted changes; use --force", name)
        }

        // Check for unpushed commits
        if hasUnpushed, err := m.hasUnpushedCommits(wt.Path); err != nil {
            return fmt.Errorf("checking for unpushed commits: %w", err)
        } else if hasUnpushed {
            return fmt.Errorf("worktree %q has unpushed commits; use --force", name)
        }
    }

    // Remove git worktree
    if err := m.gitWorktreeRemove(wt.Path, force); err != nil {
        return fmt.Errorf("git worktree remove: %w", err)
    }

    // Remove from state
    return m.state.Remove(name)
}
```

---

## CLI Implementation

```go
// internal/cli/orchestration/worktree.go
package orchestration

var worktreeCmd = &cobra.Command{
    Use:   "worktree",
    Short: "Manage git worktrees with project-aware setup",
    Long:  `Create, list, and remove git worktrees with automatic project configuration.`,
}

var worktreeCreateCmd = &cobra.Command{
    Use:   "create <name>",
    Short: "Create a new worktree",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        name := args[0]
        branch, _ := cmd.Flags().GetString("branch")
        path, _ := cmd.Flags().GetString("path")
        skipSetup, _ := cmd.Flags().GetBool("skip-setup")

        mgr := worktree.NewManager(cfg)
        wt, err := mgr.Create(name, branch, worktree.CreateOptions{
            Path:      path,
            SkipSetup: skipSetup,
        })
        if err != nil {
            return err
        }

        fmt.Printf("Created worktree '%s' at %s on branch %s\n",
            wt.Name, wt.Path, wt.Branch)
        if wt.SetupCompleted {
            fmt.Println("Setup completed successfully.")
        }
        return nil
    },
}

var worktreeListCmd = &cobra.Command{
    Use:   "list",
    Short: "List all tracked worktrees",
    RunE: func(cmd *cobra.Command, args []string) error {
        mgr := worktree.NewManager(cfg)
        worktrees, err := mgr.List()
        if err != nil {
            return err
        }

        if len(worktrees) == 0 {
            fmt.Println("No worktrees tracked.")
            return nil
        }

        // Table output
        w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
        fmt.Fprintln(w, "NAME\tPATH\tBRANCH\tSTATUS\tCREATED")
        for _, wt := range worktrees {
            age := humanize.Time(wt.CreatedAt)
            fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
                wt.Name, wt.Path, wt.Branch, wt.Status, age)
        }
        return w.Flush()
    },
}
```

---

## Testing Strategy

### Unit Tests

```go
// internal/worktree/manager_test.go
func TestManager_Create(t *testing.T) {
    t.Parallel()

    tests := map[string]struct {
        name    string
        branch  string
        opts    CreateOptions
        wantErr bool
    }{
        "basic create": {
            name:   "test-wt",
            branch: "feat/test",
        },
        "custom path": {
            name:   "custom",
            branch: "feat/custom",
            opts:   CreateOptions{Path: "/tmp/custom-wt"},
        },
        "duplicate name fails": {
            name:    "existing",
            branch:  "feat/dup",
            wantErr: true,
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            t.Parallel()
            // ... test implementation
        })
    }
}
```

### Integration Tests

```go
func TestWorktreeIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Create temp git repo
    repoDir := t.TempDir()
    initGitRepo(t, repoDir)

    // Test full workflow
    mgr := worktree.NewManager(worktree.Config{
        BaseDir: repoDir,
        Prefix:  "wt-",
    })

    // Create
    wt, err := mgr.Create("test", "feat/test", worktree.CreateOptions{})
    require.NoError(t, err)
    assert.DirExists(t, wt.Path)

    // List
    list, err := mgr.List()
    require.NoError(t, err)
    assert.Len(t, list, 1)

    // Remove
    err = mgr.Remove("test", false)
    require.NoError(t, err)
    assert.NoDirExists(t, wt.Path)
}
```

---

## Dependencies

- None on other new commands
- Uses existing `internal/config` for configuration
- Uses standard `os/exec` for git commands

---

## Implementation Phases

### Phase 1: Core Commands
1. Implement `Manager` interface with git operations
2. Implement state persistence
3. Add `create`, `list`, `remove` subcommands
4. Unit tests for all operations

### Phase 2: Setup Script
1. Implement setup script execution
2. Add default setup script template
3. Add `setup` and `prune` subcommands

### Phase 3: Integration
1. CLI integration with cobra
2. Tab completion for worktree names
3. Integration tests

---

## Quick Start Command

```bash
autospec specify "$(cat .dev/tasks/worktree-management-command.md)"
```
