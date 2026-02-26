# DAG Multi-Spec Orchestration

> **⚠️ Experimental Feature (Dev Builds Only)**
>
> DAG orchestration is an experimental feature available only in development builds. It is not included in production releases. Install a dev build to use this feature.

Run multiple autospec workflows in parallel across git worktrees with dependency management.

## Overview

**DAG** stands for **Directed Acyclic Graph** — a structure where tasks have dependencies (directed edges) but no circular references (acyclic). This ensures specs execute in the correct order.

DAG orchestration is a meta-orchestrator that:
- Executes multiple specs simultaneously
- Respects dependencies between features
- Isolates each spec in its own git worktree
- Auto-merges completed specs back to your base branch

**Problem it solves**: Running 5+ features manually requires multiple terminals, tracking dependencies by hand, and manually merging each one. DAG orchestration handles all of this with a single command.

## Prerequisites

1. **Constitution required**: Run `autospec constitution` first — all workflow stages fail without `.autospec/constitution.yaml`
2. **Worktree setup script** (recommended): Run `autospec worktree gen-script` to generate a project-specific setup script that installs dependencies in each worktree

## Quick Start

```bash
# 1. Create a DAG file defining your features
cat > .autospec/dags/my-features.yaml << 'EOF'
schema_version: "1.0"

dag:
  name: "Q1 Features"

execution:
  max_parallel: 4
  timeout: "4h"
  base_branch: "main"

layers:
  - id: "L0"
    features:
      - id: "050-auth-core"
        description: "Implement JWT authentication with login/logout endpoints, token refresh, and middleware"

      - id: "051-user-model"
        description: "Create user data model with CRUD operations, password hashing, and email validation"

  - id: "L1"
    depends_on: ["L0"]
    features:
      - id: "052-user-profile"
        description: "Add user profile endpoints: view, update, avatar upload, and account deletion"
        depends_on: ["050-auth-core", "051-user-model"]
EOF

# 2. Validate the DAG
autospec dag validate .autospec/dags/my-features.yaml

# 3. Run with parallelism
autospec dag run .autospec/dags/my-features.yaml --parallel
```

## DAG File Reference

```yaml
schema_version: "1.0"  # Required, currently "1.0"

dag:
  name: "Feature Set Name"  # Human-readable name (used for branch naming)
  id: "optional-id"         # Optional explicit ID override (takes priority over name)

execution:
  max_parallel: 4      # Max specs to run simultaneously
  timeout: "2h"        # Per-spec timeout
  base_branch: "main"  # Source for worktrees, merge target
  on_conflict: "manual"  # "manual" or "agent"

layers:
  - id: "L0"           # Layer identifier
    name: "Foundation" # Optional human-readable name
    features:
      - id: "001-feature-name"  # Maps to specs/001-feature-name/
        description: "Detailed description used by autospec run -spti"
        timeout: "30m"  # Optional per-spec override
        depends_on: ["other-spec-id"]  # Optional spec-level deps

  - id: "L1"
    depends_on: ["L0"]  # All L1 specs wait for ALL L0 specs
    features:
      - id: "002-next-feature"
        description: "..."
```

## Commands

| Command | Purpose |
|---------|---------|
| `dag validate <file>` | Check DAG structure, dependencies, and ID uniqueness |
| `dag visualize <file>` | ASCII diagram of spec dependencies |
| `dag run <file>` | Execute specs (resumes automatically if interrupted) |
| `dag run <file> --parallel` | Execute specs in parallel |
| `dag run <file> --fresh` | Discard existing state and start fresh |
| `dag run <file> --only spec1,spec2` | Run only specified specs |
| `dag run <file> --only spec1 --clean` | Clean and restart specific specs |
| `dag run <file> --dry-run` | Preview execution plan |
| `dag status <file>` | Show progress for workflow |
| `dag watch <file>` | Live status table (auto-refresh) |
| `dag logs <file> <spec>` | Tail a spec's output |
| `dag list` | List all DAG runs |
| `dag merge <file>` | Merge completed specs to base |
| `dag cleanup <file>` | Remove worktrees and optionally logs |
| `dag clean-logs` | Bulk cleanup of log files |

## How It Works

```
dag.yaml
    │
    ▼
┌─────────────────┐
│ Parse & Validate│ ← Check deps, detect cycles
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Create Worktrees│ ← One per spec (../wt-001-feature/)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Execute Specs   │ ← Run autospec run -spti in each
│ (parallel/seq)  │   Respect dependencies
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Merge & Cleanup │ ← Auto-merge to base branch
└─────────────────┘
```

## Branch and Worktree Naming

DAG orchestration creates git branches and worktrees with human-readable names based on the DAG identity.

### Branch Format

```
dag/<dag-id>/<spec-id>
```

Examples:
- `dag/gitstats-cli-v1/050-auth-core`
- `dag/q1-features/051-user-model`

### Worktree Format

```
dag-<dag-id>-<spec-id>
```

Worktrees are created adjacent to the main repository directory.

### ID Resolution Priority

The DAG ID used in branch and worktree names is resolved in this order:

1. **`dag.id`** — If explicitly set, used directly (also slugified for git-branch safety)
2. **`dag.name`** — Slugified to lowercase, hyphen-separated format
3. **Workflow filename** — Fallback when neither id nor name is set

Examples:
| dag.id | dag.name | Workflow File | Resolved ID |
|--------|----------|---------------|-------------|
| `v1` | `GitStats CLI` | `workflow.yaml` | `v1` |
| — | `GitStats CLI v1` | `workflow.yaml` | `gitstats-cli-v1` |
| — | — | `features/v1.yaml` | `v1` |

### ID Immutability

The resolved ID is **locked in the state file** at the first run. If you later modify `dag.name` or `dag.id` in a way that would produce a different resolved ID, you'll get an error:

```
Error: DAG ID mismatch detected for workflow.yaml

  Current resolved ID:  new-feature-name
  Stored ID in state:   old-feature-name
  Original DAG name:    Old Feature Name

This can happen when dag.name or dag.id is modified after the first run.
Continuing would orphan existing branches and worktrees.

To resolve this, choose one of these options:
  1. Revert your dag.name/dag.id changes to match the original
  2. Use --fresh to start a new run (old branches/worktrees will be cleaned up)
```

This prevents accidentally orphaning branches and worktrees when the DAG name changes.

### Collision Handling

If a branch name collides with an existing branch from a **different** DAG, a 4-character hash suffix is automatically appended:

```
dag/mydag/200-spec-a8f3
```

This ensures unique branch names even when two DAGs resolve to similar identifiers.

## Dependency Types

### Layer Dependencies

All specs in a layer wait for ALL specs in dependent layers:

```yaml
layers:
  - id: "L0"
    features: [A, B]
  - id: "L1"
    depends_on: ["L0"]  # C and D wait for BOTH A and B
    features: [C, D]
```

### Spec Dependencies (Additive)

Fine-grained control within or across layers:

```yaml
layers:
  - id: "L0"
    features:
      - id: "A"
      - id: "B"
        depends_on: ["A"]  # B waits for A specifically
```

## Dynamic Spec Creation

Specs are created on-the-fly if they don't exist:

- If `specs/<id>/` exists → Resume from current stage
- If `specs/<id>/` doesn't exist → Create using `description` field

This means you can define 10 features in a DAG and run them all without manually creating specs first.

## Validating DAG Files

The `dag validate` command checks DAG files for common issues:

```bash
# Validate a single DAG file
autospec dag validate .autospec/dags/my-features.yaml

# Validate all DAGs in a directory for ID uniqueness
autospec dag validate .autospec/dags/
```

### Validation Checks

| Check | Severity | Description |
|-------|----------|-------------|
| Schema validation | Error | Required fields, valid structure |
| Dependency cycles | Error | Circular dependencies between specs |
| Missing spec refs | Error | References to non-existent specs |
| Duplicate resolved IDs | Error | Two DAGs resolve to the same ID |
| Duplicate names | Warning | Two DAGs have the same `dag.name` (IDs may differ) |

### Duplicate ID Detection

When validating a directory of DAG files, `dag validate` detects duplicate resolved IDs:

```
Error: duplicate resolved DAG ID "q1-features": first in q1-release.yaml, also in quarterly.yaml
```

This prevents runtime conflicts when multiple DAGs would create overlapping branches.

Duplicate `dag.name` values produce a warning (not an error) since names are for display only—the resolved ID is what matters:

```
Warning: duplicate DAG name "Q1 Features": first in q1-release.yaml, also in quarterly.yaml (IDs may differ)
```

## Layer Staging

When running multi-layer DAGs, layer staging ensures each layer has access to code from previous layers.

### How It Works

1. **Layer 0** specs branch from the base branch (typically `main`)
2. When Layer 0 completes, all specs merge into a staging branch (`dag/<dag-id>/stage-L0`)
3. **Layer 1** specs branch from the Layer 0 staging branch
4. This continues for each layer, creating a chain of staging branches

```
main ─────────────────────────────────────────────▶
  │
  ├─ 050-auth-core ─────┐
  │                     │
  └─ 051-user-model ────┼──▶ dag/mydag/stage-L0 ──┐
                        │                          │
                        │    ├─ 052-user-profile ──┼──▶ dag/mydag/stage-L1
                        │    │                     │
                        │    └─ 053-api-docs ──────┘
                        │
                        └──▶ (final merge to main)
```

### Configuration

```yaml
dag:
  automerge: true   # Auto-merge specs to staging as they complete (default: false)
```

### CLI Flags

| Flag | Description |
|------|-------------|
| `--automerge` | Enable auto-merge to staging (overrides config) |
| `--no-automerge` | Disable auto-merge (batch merge at layer end) |
| `--no-layer-staging` | Disable layer staging entirely (legacy mode) |

### Final Merge

When layer staging is enabled, `dag merge` merges only the **final staging branch** to main:

```bash
autospec dag merge .autospec/dags/my-workflow.yaml
# Merges dag/mydag/stage-L1 → main (single merge commit)
```

## Configuration

DAG settings in `.autospec/config.yml`:

```yaml
dag:
  on_conflict: "manual"     # Default conflict handling
  max_spec_retries: 0       # Auto-retry failed specs
  max_log_size: "50MB"      # Max log file size per spec
  automerge: false          # Auto-merge specs to staging as they complete
  autocommit: true          # Verify/retry commits after spec completion
  autocommit_retries: 1     # Number of commit retry attempts

worktree:
  base_dir: ""              # Parent directory for worktrees
  prefix: ""                # Worktree directory prefix
  setup_script: ""          # Custom setup script path
  setup_timeout: "5m"       # Setup script timeout
  copy_dirs: .autospec,.claude,.opencode  # Dirs to copy
```

## Conflict Handling

When merging completed specs:

**`on_conflict: manual`** (default)
- Pause merge, output context for manual resolution
- Run `dag merge --continue` after fixing

**`on_conflict: agent`**
- Spawn agent to resolve conflicts automatically
- Merge continues after resolution

## Best Practices

1. **Right-size your specs**: Each should be 3-15 tasks, 2-8 hours (see [task-sizing.md](./task-sizing.md))
2. **Genuine dependencies**: Only add deps where truly needed
3. **Detailed descriptions**: Include specific commands, flags, behaviors
4. **Start sequential**: Use `--dry-run` first, then sequential, then parallel
5. **Monitor with watch**: Use `dag watch` in a separate terminal

## Example Use Cases

- **Feature release**: Ship 5 features for a version, some dependent
- **Refactoring**: Break large refactor into independent chunks
- **Onboarding project**: Build a new app feature-by-feature
- **Tech debt**: Parallelize multiple cleanup tasks

## See Also

- [Task Sizing Guide](./task-sizing.md) — Right-size specs for optimal results
- [Parallel Execution](./parallel-execution.md) — Parallel task execution within a spec
- [Worktree Management](./worktree.md) — Git worktree configuration
