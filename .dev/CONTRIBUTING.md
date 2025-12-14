# Contributing to Auto Claude SpecKit

## Development Setup

1. Clone the repo and checkout `dev` branch:
   ```bash
   git clone git@github.com:ariel-frischer/auto-claude-speckit.git
   cd auto-claude-speckit
   git checkout dev
   ```

2. Install git hooks:
   ```bash
   ./.dev/scripts/setup-hooks.sh
   ```

3. Build and test:
   ```bash
   make build
   make test
   ```

## Branch Workflow

- **`main`** - Stable release branch (no `.dev/` files)
- **`dev`** - Development branch (has `.dev/` files)

### Rules

| Action | Allowed |
|--------|---------|
| Merge `dev` -> `main` | Yes |
| Rebase `dev` from `main` | Yes (preferred) |
| Merge `main` -> `dev` | No (use rebase) |

### Why?

The `dev` branch contains `.dev/` files (docs, scripts, specs) that shouldn't exist on `main`. Using rebase instead of merge keeps history clean and avoids conflicts with these files.

### Syncing dev with main

After a release, sync `dev` with `main`:

```bash
git checkout dev
git rebase main
git push origin dev --force-with-lease
```

## Git Hooks

Install hooks after cloning:

```bash
./.dev/scripts/setup-hooks.sh
```

### pre-merge-commit

Prevents accidentally merging `main` into `dev` branches. Suggests using `git rebase main` instead.

To bypass (if you really need to):
```bash
git merge --no-verify main
```

### post-merge

Auto-cleans `.dev/` directory when merging to `main`. Runs automatically after `git merge dev` on main.

## Releasing

Releases are made from `main`:

```bash
git checkout main
git merge dev        # post-merge hook auto-removes .dev/
git push origin main
make patch           # or make minor/major
```

The `post-merge` hook automatically removes `.dev/` from the merge commit. CI will build and publish binaries automatically.

Alternatively, use the autobump commands directly:

```bash
make patch   # Bump v0.0.X
make minor   # Bump v0.X.0
make major   # Bump vX.0.0
```
