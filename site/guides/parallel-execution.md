---
title: Sequential & Parallel Execution
parent: Guides
nav_order: 3
---

# Sequential and Parallel Execution Guide

This guide covers running multiple autospec workflows either sequentially in one working directory or manually in parallel with separate git worktrees.

## Understanding Branch Behavior

{: .warning }
When you run `autospec specify` or `autospec run -a`, autospec creates a new feature branch and checks it out.

```bash
# Before: on main branch
autospec specify "Add user authentication"
# After: now on branch 001-user-authentication
```

That means:
- Running multiple `autospec specify` commands sequentially switches branches each time.
- You should not run multiple autospec processes in the same working directory.
- Parallel feature work needs separate working directories, typically git worktrees.

## Sequential Execution

Use sequential execution when features depend on each other, modify overlapping code, or need review between steps.

```bash
autospec run -a "Add user authentication" && \
autospec run -a "Add user profile page" && \
autospec run -a "Add password reset flow"
```

For review checkpoints:

```bash
autospec prep "Add user authentication"
# Review specs/001-user-authentication/
autospec implement

autospec prep "Add user profile page"
# Review specs/002-user-profile-page/
autospec implement
```

## Parallel Execution with Git Worktrees

Git worktrees allow multiple working directories to share one repository. Each worktree has its own checked-out branch, which lets separate agent sessions work on independent features without file contention.

{: .note }
For automated worktree management with project setup, see [Worktree Management](worktree).

Manual setup:

```bash
cd ~/projects/myapp

git worktree add ../myapp-auth -b feature/auth
git worktree add ../myapp-profile -b feature/profile
git worktree add ../myapp-search -b feature/search
```

Directory layout:

```text
~/projects/
├── myapp/
├── myapp-auth/
├── myapp-profile/
└── myapp-search/
```

Run each workflow from its own terminal:

```bash
cd ~/projects/myapp-auth
autospec run -a "Add user authentication with OAuth"
```

```bash
cd ~/projects/myapp-profile
autospec run -a "Add user profile page with avatar upload"
```

```bash
cd ~/projects/myapp-search
autospec run -a "Add full-text search"
```

You can also launch independent workflows in the background:

```bash
cd ~/projects/myapp-auth && autospec run -a "Add user auth" &
cd ~/projects/myapp-profile && autospec run -a "Add profile page" &
cd ~/projects/myapp-search && autospec run -a "Add search" &
wait
```

## Worktree Practices

- Use one branch per worktree. Git prevents checking out the same branch in multiple worktrees.
- Keep worktrees as siblings of the main repository, not nested inside it.
- Use descriptive directory and branch names.
- Merge one feature at a time from the main worktree.
- Remove worktrees after merging.

```bash
git worktree list
git worktree remove ../myapp-auth
git worktree prune
```

## Merging Parallel Features

After each feature finishes:

```bash
cd ~/projects/myapp
git checkout main

git merge feature/auth
git merge feature/profile
git merge feature/search

git worktree remove ../myapp-auth
git worktree remove ../myapp-profile
git worktree remove ../myapp-search
```

If branches conflict, merge them one at a time and use the related `specs/` artifacts as context for resolving intent.

## Comparison

| Approach | Parallel | Disk Usage | Complexity | Best For |
|----------|----------|------------|------------|----------|
| Sequential | No | Low | Simple | Dependent features, review-heavy workflows |
| Git worktrees | Yes | Medium | Moderate | Independent features in one repository |
| Full clones | Yes | High | Simple | Complete isolation or different remotes |

## Troubleshooting

If autospec tries to create a branch that already exists:

```bash
git branch -a | grep feature-name
git branch -d 001-feature-name
```

If a worktree shows as prunable:

```bash
git worktree prune
git worktree remove --force /path/to/worktree
```

Each worktree has its own `specs/` directory. Copy specs manually when you need to reuse context across worktrees.

## See Also

- [Worktree Management](worktree)
- [CLI Reference](../reference/cli)
- [Troubleshooting](troubleshooting)
