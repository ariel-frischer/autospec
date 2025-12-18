---
title: Sequential & Parallel Execution
parent: Guides
nav_order: 3
---

# Sequential and Parallel Execution Guide

This guide covers strategies for running multiple autospec workflows, whether sequentially on a single branch or in parallel using git worktrees.

## Understanding Branch Behavior

{: .warning }
When you run `autospec specify` or `autospec run -a`, the command creates a new feature branch and **automatically checks it out**. This is critical to understand when planning parallel workflows.

```bash
# Before: on main branch
autospec specify "Add user authentication"
# After: now on branch 001-user-authentication
```

This automatic checkout behavior means:
- Running multiple `autospec specify` commands sequentially will switch branches each time
- You cannot run parallel `autospec` processes in the same working directory
- For parallel execution, you need separate working directories (git worktrees)

---

## Sequential Execution

For features that should be developed one after another, use shell command chaining.

### Basic Sequential Workflow

```bash
# Complete one feature fully before starting the next
autospec run -a "Add user authentication" && \
autospec run -a "Add user profile page" && \
autospec run -a "Add password reset flow"
```

### Sequential with Review Points

```bash
# Prepare specs for review, then implement separately
autospec prep "Add user authentication"
# Review specs/001-user-authentication/ artifacts
autospec implement

autospec prep "Add user profile page"
# Review specs/002-user-profile-page/ artifacts
autospec implement
```

### When to Use Sequential Execution

- Features have dependencies (feature B requires feature A)
- You want to review each feature before starting the next
- Limited compute resources
- Features modify overlapping code areas

---

## Parallel Execution with Git Worktrees

Git worktrees allow multiple working directories sharing the same repository, enabling true parallel development.

### Setting Up Worktrees

```bash
# From your main repository directory
cd ~/projects/myapp

# Create worktrees for parallel features
git worktree add ../myapp-auth feature/auth
git worktree add ../myapp-profile feature/profile
git worktree add ../myapp-search feature/search
```

**Directory structure after setup:**
```
~/projects/
├── myapp/              # Main repository
├── myapp-auth/         # Worktree for auth feature
├── myapp-profile/      # Worktree for profile feature
└── myapp-search/       # Worktree for search feature
```

### Running Parallel Workflows

```bash
# Terminal 1
cd ~/projects/myapp-auth
autospec run -a "Add user authentication with OAuth"

# Terminal 2
cd ~/projects/myapp-profile
autospec run -a "Add user profile page with avatar upload"

# Terminal 3
cd ~/projects/myapp-search
autospec run -a "Add full-text search with Elasticsearch"
```

Or launch all in background:

```bash
cd ~/projects/myapp-auth && autospec run -a "Add user auth" &
cd ~/projects/myapp-profile && autospec run -a "Add profile page" &
cd ~/projects/myapp-search && autospec run -a "Add search" &
wait  # Wait for all to complete
```

### Worktree Best Practices

{: .note }
Git prevents checking out the same branch in multiple worktrees. Each worktree must be on a unique branch.

1. **Use descriptive directory names**: Match the feature being developed
   ```bash
   # Good
   git worktree add ../myapp-oauth feature/oauth

   # Avoid
   git worktree add ../temp1 feature/oauth
   ```

2. **Keep worktrees as siblings**: Place them next to (not inside) the main repo
   ```bash
   # Good: siblings
   ~/projects/myapp/
   ~/projects/myapp-feature/

   # Bad: nested
   ~/projects/myapp/worktrees/feature/  # Avoid this
   ```

3. **Clean up when done**: Remove worktrees after merging
   ```bash
   git worktree remove ../myapp-auth
   git worktree prune  # Clean stale entries
   ```

4. **List active worktrees**: Track what's in use
   ```bash
   git worktree list
   ```

### Merging Parallel Features

After parallel features complete:

```bash
# From main repository
cd ~/projects/myapp
git checkout main

# Merge each feature
git merge feature/auth
git merge feature/profile
git merge feature/search

# Clean up worktrees
git worktree remove ../myapp-auth
git worktree remove ../myapp-profile
git worktree remove ../myapp-search
```

> **Tip: Claude Code for Merge Conflicts**
>
> Claude Code excels at resolving merge conflicts and merging branches. For best results, provide context about each branch's original purpose by sharing the `specs/` folder from each branch being merged. This gives Claude the specification context needed to make intelligent merge decisions.

---

## Advanced: Bare Repository Pattern

For heavy parallel workflows, consider a bare repository pattern:

```bash
# Clone as bare (no working directory)
git clone --bare git@github.com:user/myapp.git myapp.git

# Add worktrees from bare repo
cd myapp.git
git worktree add ../myapp-main main
git worktree add ../myapp-auth feature/auth
git worktree add ../myapp-profile feature/profile
```

**Benefits:**
- Cleaner separation between repo metadata and working directories
- Easier management of many worktrees
- Better for CI/CD pipelines

---

## Comparison Table

| Approach | Parallel | Disk Usage | Complexity | Best For |
|----------|----------|------------|------------|----------|
| Sequential (`&&`) | No | Low | Simple | Dependent features, review-heavy |
| Git Worktrees | Yes | Medium | Moderate | Independent features, time-sensitive |
| Full Clones | Yes | High | Simple | Complete isolation |

---

## Troubleshooting

### "Branch already exists" Error

If autospec tries to create a branch that exists:

```bash
# Check existing branches
git branch -a | grep feature-name

# Delete if safe to do so
git branch -d 001-feature-name
```

### Worktree Shows as "prunable"

```bash
# Remove stale worktree references
git worktree prune

# Force remove if directory was deleted
git worktree remove --force /path/to/worktree
```

### Spec Directory Conflicts

Each worktree has its own `specs/` directory. If you need to share specs:

```bash
# Copy specs between worktrees
cp -r ~/projects/myapp/specs/001-auth ~/projects/myapp-auth/specs/
```

---

## Example: Full Parallel Workflow

See [`scripts/examples/parallel-features.sh`](https://github.com/ariel-frischer/autospec/blob/main/scripts/examples/parallel-features.sh) for the complete script.

```bash
#!/bin/bash
# parallel-features.sh - Run 3 features in parallel

REPO_DIR=~/projects/myapp
FEATURES=(
    "auth:Add user authentication with OAuth"
    "profile:Add user profile with avatar upload"
    "search:Add full-text search functionality"
)

# Create worktrees
for feature in "${FEATURES[@]}"; do
    name="${feature%%:*}"
    git worktree add "${REPO_DIR}-${name}" -b "feature/${name}" 2>/dev/null || true
done

# Run autospec in parallel
pids=()
for feature in "${FEATURES[@]}"; do
    name="${feature%%:*}"
    desc="${feature#*:}"
    (
        cd "${REPO_DIR}-${name}"
        autospec run -a "${desc}"
    ) &
    pids+=($!)
done

# Wait for all to complete
for pid in "${pids[@]}"; do
    wait $pid
done

echo "All features complete!"
```

---

## See Also

- [CLI Reference](../reference/cli) - Complete command documentation
- [Configuration](../reference/configuration) - Project and user configuration
- [Troubleshooting](troubleshooting) - Common issues and solutions
