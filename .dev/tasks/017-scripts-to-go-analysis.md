# Analysis: Shell Scripts vs Go Commands

**Date:** 2025-12-15
**Status:** Recommendation
**Decision:** Migrate to Go commands

## Executive Summary

The shell scripts in `.autospec/scripts/` should be migrated to autospec CLI commands. Go is well-suited for cross-platform CLI tools and does NOT require heavy dependencies. The scripts duplicate functionality already present in Go packages, and Claude Code can call `autospec <command> --json` just as easily as shell scripts.

## Current Architecture

### Scripts Present (5 total)

| Script | Lines | Purpose |
|--------|-------|---------|
| `common.sh` | 200 | Utility functions (repo root, branch parsing, path resolution) |
| `check-prerequisites.sh` | 199 | Validates prerequisites, outputs JSON with feature paths |
| `create-new-feature.sh` | 309 | Creates feature branches and directories |
| `setup-plan.sh` | 62 | Copies plan template to feature directory |
| `update-agent-context.sh` | 800 | Updates CLAUDE.md and other agent context files |

### How Scripts Are Used

```
User runs: autospec specify "Add feature X"
    ↓
Go CLI calls: claude -p "/autospec.specify \"Add feature X\""
    ↓
Claude CLI interprets /autospec.specify command
    ↓
Claude AI reads .claude/commands/autospec.specify.md
    ↓
Claude AI executes bash commands in the prompt:
    .autospec/scripts/create-new-feature.sh --json ...
    ↓
Claude AI parses JSON output and generates YAML artifacts
```

**Key insight:** The scripts are called by Claude AI during slash command execution, NOT by the Go CLI directly.

## Analysis: Go vs Shell Scripts

### Cross-Platform Concerns

**Does Go require "hefty packages" for cross-platform?**

**NO.** This is a misconception. Go is ideal for cross-platform CLI tools:

| Operation | Go Approach | Packages Required |
|-----------|-------------|-------------------|
| File operations | `os`, `io`, `filepath` | Standard library |
| Git operations | `os/exec` calling git | Standard library |
| JSON output | `encoding/json` | Standard library |
| String processing | `strings`, `regexp` | Standard library |
| Directory traversal | `filepath.Walk` | Standard library |
| Path handling | `filepath.Join` | Standard library |

**Cross-platform is automatic:**
- `filepath.Join()` handles path separators (/ vs \)
- Git CLI works identically on all platforms
- Go binaries are self-contained

### Existing Go Code Overlap

The scripts duplicate functionality that already exists in Go:

| Script Function | Existing Go Equivalent |
|-----------------|------------------------|
| `get_repo_root()` | `git.GetRepositoryRoot()` in `internal/git/git.go` |
| `get_current_branch()` | `git.GetCurrentBranch()` in `internal/git/git.go` |
| `find_feature_dir_by_prefix()` | `spec.GetSpecDirectory()` in `internal/spec/spec.go` |
| `check_feature_branch()` | `spec.DetectCurrentSpec()` in `internal/spec/spec.go` |
| `find_artifact()` | `validation.GetTasksFilePath()` pattern |

### Benefits of Go Migration

| Benefit | Impact |
|---------|--------|
| **Single source of truth** | No logic duplication between scripts and Go |
| **Type safety** | Compile-time error checking |
| **Better testing** | Go unit tests vs shell testing (difficult) |
| **Structured error handling** | Error types vs exit codes |
| **No shell escaping issues** | Go handles arguments safely |
| **Simpler distribution** | No script embedding/copying required |
| **Easier maintenance** | One language, one codebase |

### Why Scripts Were Originally Used

The scripts exist because:
1. Claude Code executes shell commands during slash command interpretation
2. Scripts provide structured JSON output for Claude to parse
3. Historical evolution from shell-based to Go-based tooling

However, Claude can call ANY CLI tool, not just shell scripts. The slash commands can be updated to call `autospec <command> --json` instead.

## Recommendation

**Migrate to Go commands.** The shell scripts should be replaced with autospec CLI commands.

### Proposed New Commands

| Command | Replaces | Effort |
|---------|----------|--------|
| `autospec new-feature` | `create-new-feature.sh` | Medium |
| `autospec prereqs` | `check-prerequisites.sh` | Low |
| (defer) | `update-agent-context.sh` | High |
| (trivial) | `setup-plan.sh` | Trivial |

### Migration Priority

```
Priority 1 (HIGH): create-new-feature.sh → autospec new-feature
  - Core workflow, most frequently used
  - Creates branches, directories, outputs JSON
  - Building blocks exist in internal/git/ and internal/spec/

Priority 2 (HIGH): check-prerequisites.sh → autospec prereqs
  - Simple validation logic
  - Already have validation.go and spec.go building blocks
  - Low effort, high impact

Priority 3 (LOW): setup-plan.sh → autospec setup-plan
  - Rarely used
  - Just copies a template file
  - Trivial implementation

Priority 4 (DEFER): update-agent-context.sh
  - 800 lines of complex text manipulation
  - Optional feature (agent context files)
  - Keep script until higher priority items complete
```

### Slash Command Updates

After Go commands exist, update `.claude/commands/autospec.specify.md`:

```markdown
# Before:
.autospec/scripts/create-new-feature.sh --json --short-name "<name>" "$ARGUMENTS"

# After:
autospec new-feature --json --short-name "<name>" "$ARGUMENTS"
```

### Implementation Outline

**`autospec new-feature` command:**
```go
// internal/cli/new_feature.go

// Functionality:
// 1. Parse feature description
// 2. Generate branch name (stop word filtering, truncation)
// 3. Get next feature number from git branches + specs/
// 4. Create git branch (if git repo)
// 5. Create specs/<NNN-name>/ directory
// 6. Output JSON: {BRANCH_NAME, SPEC_FILE, FEATURE_NUM, ...}

// Reuse:
// - internal/git/ for git operations
// - internal/spec/ for spec directory handling
// - encoding/json for output
```

**`autospec prereqs` command:**
```go
// internal/cli/prereqs.go

// Functionality:
// 1. Detect current spec (reuse spec.DetectCurrentSpec)
// 2. Check required files exist (spec.yaml, plan.yaml, tasks.yaml)
// 3. List available documents
// 4. Output JSON: {FEATURE_DIR, FEATURE_SPEC, IMPL_PLAN, TASKS, ...}

// Reuse:
// - internal/spec/ for detection
// - internal/validation/ for file checks
```

## Conclusion

Go is the right choice for autospec's CLI commands. The concern about "hefty packages" is unfounded - Go's standard library provides everything needed for cross-platform CLI development. Migrating from shell scripts to Go commands will:

1. Eliminate code duplication
2. Improve type safety and testability
3. Simplify distribution (no script embedding)
4. Provide better error handling
5. Maintain the same Claude integration (Claude calls `autospec` instead of scripts)

**Recommended action:** Create `autospec new-feature` and `autospec prereqs` commands, update slash commands to use them, and deprecate `.autospec/scripts/` over time.
