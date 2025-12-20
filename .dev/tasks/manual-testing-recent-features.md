# Manual Testing Plan: Recent Features

Testing plan for features from specs 064-067.

## Features to Test

1. **autospec init** (067-agent-init-config) - Agent selection during initialization
2. **autospec worktree** (064-worktree-management) - Worktree create/list/remove/prune
3. **autospec worktree gen-script** (066-worktree-setup-generator) - Setup script generation

---

## 1. autospec init (067-agent-init-config)

### Test Cases

| # | Test | Command | Expected Result |
|---|------|---------|-----------------|
| 1.1 | Agent selection prompt | `autospec init` | Multi-select appears with claude pre-selected + "(Recommended)" |
| 1.2 | Toggle selections | Space key in prompt | Agents toggle on/off |
| 1.3 | Claude permissions | Select claude → complete | `.claude/settings.local.json` has required permissions |
| 1.4 | Preference persistence | Run init twice | Second run has previous selections pre-checked |
| 1.5 | Skip agents flag | `autospec init --no-agents` | No agent prompt, completes without configuring agents |
| 1.6 | No agents selected | Deselect all in prompt | Warning about manual agent configuration |
| 1.7 | Idempotency | Run init multiple times | Permissions not duplicated, config not corrupted |
| 1.8 | Custom specs_dir | Set `specs_dir: features` | Permissions use `Write(features/**)` instead |

### Expected Claude Permissions

After selecting claude, `.claude/settings.local.json` should contain:
- `Bash(autospec:*)`
- `Write(.autospec/**)`
- `Edit(.autospec/**)`
- `Write(specs/**)`
- `Edit(specs/**)`

### Test Script (Fresh Directory)

```bash
# Create isolated test environment
cd /tmp && rm -rf test-init && mkdir test-init && cd test-init
git init && echo "# Test" > README.md && git add . && git commit -m "init"

# Test 1: Basic init with agent selection (interactive)
autospec init
# Expected: Multi-select prompt with claude pre-selected

# Verify Claude settings
cat .claude/settings.local.json | jq .

# Verify config saved agent preferences
cat .autospec/config.yml | grep -A5 default_agents

# Test 2: Run init again - should have previous selections
autospec init
# Expected: Previously selected agents are pre-checked

# Test 3: --no-agents flag
rm -rf .autospec .claude
autospec init --no-agents
# Expected: No prompt, warning about no agents configured

# Cleanup
cd /tmp && rm -rf test-init
```

---

## 2. Worktree Commands (064-worktree-management)

### Test Cases

| # | Test | Command | Expected Result |
|---|------|---------|-----------------|
| 2.1 | Create worktree | `autospec worktree create test-wt --branch feat/test` | Worktree created, .autospec/ copied |
| 2.2 | List worktrees | `autospec worktree list` | Table with name, path, branch, status, age |
| 2.3 | Empty list | (no worktrees) `autospec worktree list` | "No worktrees tracked" message |
| 2.4 | Remove clean worktree | `autospec worktree remove test-wt` | Worktree removed |
| 2.5 | Remove with changes | Make changes → remove | Error about uncommitted changes |
| 2.6 | Force remove | `autospec worktree remove test-wt --force` | Removed despite changes |
| 2.7 | Prune stale | Delete dir manually → prune | Stale entry removed |
| 2.8 | Custom path | `--path /tmp/foo-wt` | Created at custom path |

### Test Script

```bash
# From autospec repo
cd /home/ari/repos/autospec-architecture-1

# Test 2.1: Create worktree
autospec worktree create wt-test-alpha --branch test/wt-alpha
# Expected: Worktree created, .autospec/ copied

# Verify .autospec was copied
ls ../wt-test-alpha/.autospec/

# Test 2.2: List worktrees
autospec worktree list
# Expected: Table showing wt-test-alpha with branch, status, age

# Test 2.1b: Create second worktree
autospec worktree create wt-test-beta --branch test/wt-beta

# Test 2.2b: List multiple
autospec worktree list
# Expected: Both worktrees shown

# Test 2.5: Remove with uncommitted changes
echo "test" > ../wt-test-alpha/test.txt
autospec worktree remove wt-test-alpha
# Expected: Error about uncommitted changes

# Test 2.6: Force remove
autospec worktree remove wt-test-alpha --force
# Expected: Removed despite changes

# Test 2.4: Remove clean worktree
autospec worktree remove wt-test-beta
# Expected: Removed (was clean)

# Test 2.7: Prune stale entries
autospec worktree create wt-test-stale --branch test/wt-stale
rm -rf ../wt-test-stale  # Delete manually
autospec worktree prune
# Expected: Stale entry removed, count shown

# Final cleanup - remove any test branches
git branch -D test/wt-alpha test/wt-beta test/wt-stale 2>/dev/null || true
```

---

## 3. Worktree Setup Script Generator (066-worktree-setup-generator)

### Test Cases

| # | Test | Command | Expected Result |
|---|------|---------|-----------------|
| 3.1 | Generate script | `autospec worktree gen-script` | `.autospec/scripts/setup-worktree.sh` created |
| 3.2 | Script executable | Check permissions | Script has 755 permissions |
| 3.3 | Package manager detection | Check script content | Contains `go mod download` (Go project) |
| 3.4 | Excludes secrets | Check script | Does NOT copy `.env*`, `credentials.*` |
| 3.5 | Always includes .autospec | Check script | Always copies `.autospec/` |
| 3.6 | Include env flag | `--include-env` | Script includes .env files + warning |

### Test Script

```bash
cd /home/ari/repos/autospec-architecture-1

# Test 3.1: Generate setup script
autospec worktree gen-script
# Expected: Claude generates script

# Test 3.2: Check permissions
ls -la .autospec/scripts/setup-worktree.sh
# Expected: -rwxr-xr-x (755)

# Test 3.3-3.5: Review script content
cat .autospec/scripts/setup-worktree.sh
# Look for:
# - go mod download (Go project detection)
# - .autospec/ copy command
# - NO .env copies by default
# - Proper shebang (#!/bin/bash)

# Test 3.6: Include env flag (if supported)
autospec worktree gen-script --include-env
# Expected: Warning about security, .env included in script
```

---

## 4. Config Changes Verification

```bash
# Check default_agents field works
autospec config get default_agents

# Check worktree config options exist
cat .autospec/config.yml | grep -A10 worktree || echo "No worktree config yet"

# Verify agent_preset is the current way (not deprecated claude_cmd)
autospec config get agent_preset
```

---

## End-to-End Test Sequence

Run these in order for full coverage:

```bash
# 1. Fresh init test (isolated)
cd /tmp && rm -rf e2e-test && mkdir e2e-test && cd e2e-test
git init && echo "# E2E" > README.md && git add . && git commit -m "init"
autospec init  # Select claude
cat .claude/settings.local.json
cd /tmp && rm -rf e2e-test

# 2. Worktree cycle (in autospec repo)
cd /home/ari/repos/autospec-architecture-1
autospec worktree create wt-e2e --branch test/e2e
autospec worktree list
ls ../wt-e2e/.autospec/
autospec worktree remove wt-e2e --force
git branch -D test/e2e 2>/dev/null || true

# 3. Setup script (in autospec repo)
autospec worktree gen-script
cat .autospec/scripts/setup-worktree.sh
```

---

## Cleanup Commands

If anything goes wrong, use these to reset:

```bash
# Remove test worktrees
autospec worktree prune

# Remove test branches
git branch | grep test/ | xargs -r git branch -D

# Remove generated setup script
rm -f .autospec/scripts/setup-worktree.sh

# Clear worktree state
rm -f .autospec/state/worktrees.yaml
```

---

## Status

- [ ] 1.1 Agent selection prompt
- [ ] 1.2 Toggle selections
- [ ] 1.3 Claude permissions
- [ ] 1.4 Preference persistence
- [ ] 1.5 Skip agents flag
- [ ] 1.6 No agents selected
- [ ] 1.7 Idempotency
- [ ] 1.8 Custom specs_dir
- [ ] 2.1 Create worktree
- [ ] 2.2 List worktrees
- [ ] 2.3 Empty list
- [ ] 2.4 Remove clean worktree
- [ ] 2.5 Remove with changes
- [ ] 2.6 Force remove
- [ ] 2.7 Prune stale
- [ ] 2.8 Custom path
- [ ] 3.1 Generate script
- [ ] 3.2 Script executable
- [ ] 3.3 Package manager detection
- [ ] 3.4 Excludes secrets
- [ ] 3.5 Always includes .autospec
- [ ] 3.6 Include env flag
