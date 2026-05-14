# Internals Guide

This document explains autospec's internal systems that power workflow execution. Understanding these systems helps debug issues and optimize your workflow.

## Table of Contents

- [Spec Detection](#spec-detection)
- [Validation System](#validation-system)
- [Retry and Error Handling](#retry-and-error-handling)
  - [Schema Validation on Retry](#schema-validation-on-retry)
  - [Retry Context Format](#retry-context-format)
  - [Command Template Handling](#command-template-handling)
- [Phase Context Injection](#phase-context-injection)

---

## Spec Detection

autospec automatically detects which feature spec you're working on. This eliminates the need to specify the spec directory for every command.

### Detection Methods

Detection uses the following priority order:

| Priority | Method | Description |
|----------|--------|-------------|
| 1 | Explicit | User provides `--spec` flag or spec identifier |
| 2 | Environment | `SPECIFY_FEATURE` environment variable |
| 3 | Git Branch | Branch name matches pattern `NNN-feature-name` |
| 4 | Fallback | Most recently modified directory in `specs/` |

### Git Branch Detection

If you're on a branch named `002-user-authentication`, autospec looks for a matching directory `specs/002-user-authentication/`. The pattern must match:

```
^(\d{3})-(.+)$
```

Examples:
- `002-user-auth` → `specs/002-user-auth/`
- `015-api-refactor` → `specs/015-api-refactor/`
- `feature/login` → Does not match, falls back to recent directory

### Fallback Detection

When git branch detection fails, autospec finds the most recently modified directory in `specs/`. This works well when actively developing a feature since the spec files are frequently updated.

### Viewing Detected Spec

Run `autospec st` to see which spec was detected and how:

```bash
$ autospec st
✓ Using spec: specs/002-user-auth (via git branch)
```

Detection methods shown:
- `via git branch` - Matched current git branch name
- `fallback - most recent` - Used most recently modified directory
- `via SPECIFY_FEATURE env` - Used environment variable
- `explicitly specified` - User provided spec identifier

### Overriding Detection

Force a specific spec using:

```bash
# By full directory name
autospec implement --spec 002-user-auth

# By number only
autospec implement --spec 002

# By feature name only
autospec implement --spec user-auth

# Via environment variable
SPECIFY_FEATURE=002-user-auth autospec implement
```

---

## Validation System

autospec validates artifacts before proceeding to the next workflow stage. This prevents wasted effort when required files are missing or malformed.

### What Gets Validated

| Stage | Required Artifacts | Validation |
|-------|-------------------|------------|
| `plan` | `spec.yaml` or `spec.md` | File exists |
| `tasks` | `plan.yaml` or `plan.md` | File exists |
| `implement` | `tasks.yaml` or `tasks.md` | File exists |
| All YAML files | - | Valid YAML syntax |

### Performance Contract

All validation functions execute in under 10ms. This ensures validation never becomes a bottleneck.

### Validation Errors

When validation fails, you'll see a clear error with remediation steps:

```
Error: spec file not found in specs/002-feature - run 'autospec specify <description>' to create it
```

Common validation errors:

| Error | Cause | Fix |
|-------|-------|-----|
| `spec file not found` | Missing spec.yaml/spec.md | Run `autospec specify "description"` |
| `plan file not found` | Missing plan.yaml/plan.md | Run `autospec plan` |
| `tasks file not found` | Missing tasks.yaml/tasks.md | Run `autospec tasks` |
| `failed to parse ... YAML` | Invalid YAML syntax | Check file for syntax errors |

### Exit Codes

Validation failures return specific exit codes:

| Code | Meaning | Retryable |
|------|---------|-----------|
| 0 | Success | - |
| 1 | Validation failed | Yes |
| 3 | Invalid arguments | No |
| 4 | Missing dependencies | No |

---

## Retry and Error Handling

autospec tracks retry attempts per stage to prevent infinite loops when Claude encounters persistent issues.

### How Retries Work

1. **Tracking**: Retry counts are stored per `spec:stage` combination
2. **Increment**: Count increases each time a stage fails validation
3. **Reset**: Count resets to zero when a stage succeeds
4. **Exhaustion**: After reaching `max_retries`, autospec exits with code 2

### Retry State Storage

State persists to `~/.autospec/state/retry.json`:

```json
{
  "retries": {
    "002-user-auth:implement": {
      "spec_name": "002-user-auth",
      "phase": "implement",
      "count": 2,
      "last_attempt": "2024-01-15T10:30:00Z",
      "max_retries": 3
    }
  },
  "stage_states": {
    "002-user-auth": {
      "spec_name": "002-user-auth",
      "current_phase": 3,
      "total_phases": 7,
      "completed_phases": [1, 2],
      "last_phase_attempt": "2024-01-15T10:30:00Z"
    }
  },
  "task_states": {
    "002-user-auth": {
      "spec_name": "002-user-auth",
      "current_task_id": "T005",
      "completed_task_ids": ["T001", "T002", "T003", "T004"],
      "total_tasks": 12,
      "last_task_attempt": "2024-01-15T10:30:00Z"
    }
  }
}
```

### Configuring Max Retries

Set in config file or environment:

```yaml
# .autospec/config.yml
max_retries: 5  # Default is 3
```

```bash
# Environment variable
AUTOSPEC_MAX_RETRIES=5 autospec implement
```

### When Retries Trigger

Retries increment when:
- Claude's output fails validation (missing expected file, invalid YAML)
- **Schema validation fails** (missing required fields, invalid types, invalid enum values)
- A stage doesn't produce the expected artifact
- Claude exits without completing the requested work

Retries do NOT increment for:
- User cancellation (Ctrl+C)
- Timeout (has its own handling)
- Missing dependencies (exit code 4)

### Schema Validation on Retry

When a stage fails due to schema validation errors, the orchestrator captures those errors and injects them into the next agent invocation. This gives the configured agent specific error context to fix the schema issues.

**How it works:**

1. The configured agent generates an artifact (spec.yaml, plan.yaml, or tasks.yaml)
2. Orchestrator validates the artifact against its schema using existing validators
3. If validation fails, errors are formatted into a retry context
4. Retry context is prepended to `$ARGUMENTS` for the next attempt
5. Claude receives specific errors to fix

**Validation performed per stage:**

| Stage | Artifact | Validator |
|-------|----------|-----------|
| `specify` | `spec.yaml` | `ValidateSpecSchema()` |
| `plan` | `plan.yaml` | `ValidatePlanSchema()` |
| `tasks` | `tasks.yaml` | `ValidateTasksSchema()` |

### Retry Context Format

When validation fails, the retry context follows this standardized format:

```text
RETRY X/Y
Schema validation failed:
- error message 1
- error message 2
- ...

<original arguments if any>
```

**Format details:**

| Component | Description |
|-----------|-------------|
| `RETRY X/Y` | X = current attempt number (1-based), Y = max retries |
| `Schema validation failed:` | Header indicating validation errors follow |
| `- error message` | Each validation error on its own line, prefixed with `- ` |
| Blank line | Separates retry context from original arguments |
| Original arguments | User's original input (if any) |

**Example with multiple errors:**

```text
RETRY 2/3
Schema validation failed:
- missing required field: feature.branch
- invalid enum value for user_stories[0].priority: expected one of [P1, P2, P3]
- invalid type for requirements.functional[0].testable: expected bool, got string

Create a user authentication feature
```

**Error truncation:**

If there are more than 10 validation errors, the list is truncated:

```text
RETRY 2/3
Schema validation failed:
- error 1
- error 2
- ...
- error 10
- ...and 5 more errors
```

### Command Template Handling

Each command template (`autospec.specify.md`, `autospec.plan.md`, `autospec.tasks.md`) includes a "Retry Context" section documenting how Claude should:

1. Detect retry context by checking if `$ARGUMENTS` starts with `RETRY X/Y`
2. Parse the validation errors
3. Fix the specific schema errors in the regenerated artifact
4. Preserve the original user intent from arguments after the blank line
5. Re-validate using `autospec artifact` before completing

### Inspecting Retry State

View current state:

```bash
cat ~/.autospec/state/retry.json | jq .
```

Check specific spec:

```bash
cat ~/.autospec/state/retry.json | jq '.retries["002-user-auth:implement"]'
```

### Resetting Retry State

When retry limit is exhausted (exit code 2), you need to fix the issue and reset:

**Reset all state for a spec:**

```bash
# Delete the state file entries manually
# Or delete the entire file to reset everything:
rm ~/.autospec/state/retry.json
```

**Programmatic reset (from Go code):**

```go
retry.ResetRetryCount(stateDir, specName, stage)
retry.ResetStageState(stateDir, specName)
retry.ResetTaskState(stateDir, specName)
```

### Exit Code 2: Retry Exhausted

When you see exit code 2:

1. **Check the error**: What stage failed? What was the validation error?
2. **Fix the issue**: Common causes:
   - Claude wrote malformed YAML
   - Required file wasn't created
   - File was created in wrong location
3. **Reset state**: Remove retry entry from `~/.autospec/state/retry.json`
4. **Retry**: Run the command again

Example workflow:

```bash
# Command fails with exit code 2
$ autospec implement
Error: retry limit exhausted for 002-user-auth:implement (3/3 attempts)

# Check what went wrong
$ cat specs/002-user-auth/tasks.yaml  # Maybe malformed?

# Fix the issue manually or regenerate
$ autospec tasks  # Regenerate tasks.yaml

# Reset retry state
$ cat ~/.autospec/state/retry.json | jq 'del(.retries["002-user-auth:implement"])' > /tmp/retry.json
$ mv /tmp/retry.json ~/.autospec/state/retry.json

# Try again
$ autospec implement
```

### Phase/Task Execution State

For `--phases` and `--tasks` modes, autospec tracks which phases/tasks completed:

**Phase tracking (`--phases`):**
- `completed_phases`: Array of phase numbers that finished successfully
- Used to skip already-completed phases on resume
- View with: `autospec st`

**Task tracking (`--tasks`):**
- `completed_task_ids`: Array of task IDs (T001, T002, etc.) that finished
- Used to skip completed tasks on resume
- Resume from specific task: `--from-task T005`

---

## Phase Context Injection

When running `autospec implement --phases`, each phase executes in a separate agent session. Phase context injection bundles all required information into a single file, eliminating redundant file reads.

### The Problem It Solves

Without context injection, each phase session:
1. The agent reads `spec.yaml` (2-5 seconds)
2. The agent reads `plan.yaml` (2-5 seconds)
3. The agent reads `tasks.yaml` (2-5 seconds)
4. The agent reads `.autospec/constitution.yaml` when governance exists
5. The agent filters to find current phase tasks

This adds 10-20 seconds per phase. For a 10-phase spec, that's 2-3 minutes of wasted time.

### How It Works

1. **Before phase execution**: autospec builds a `PhaseContext` struct containing:
   - Full `spec.yaml` content
   - Full `plan.yaml` content
   - Project constitution governance content, when present
   - Only the tasks for the current phase (filtered from `tasks.yaml`)
   - `_context_meta` with `skip_reads`, checklist presence, and governance bundle metadata
   - Phase number and total phase count

2. **Context file creation**: Written to `.autospec/context/phase-{N}.yaml`

3. **Passed to the agent**: The slash command receives `--context-file` argument

4. **Cleanup**: Context file deleted after phase completes (success or failure)

### Context File Structure

```yaml
# Auto-generated phase context file
# DO NOT edit this file manually

_context_meta:
  phase_artifacts_bundled: true
  bundled_artifacts:
    - spec.yaml
    - plan.yaml
    - tasks.yaml (phase-filtered)
    - constitution.yaml
  has_governance: true
  governance_file: .autospec/constitution.yaml
  has_checklists: false
  skip_reads:
    - specs/002-user-auth/spec.yaml
    - specs/002-user-auth/plan.yaml
    - specs/002-user-auth/tasks.yaml
    - .autospec/constitution.yaml

phase: 3
total_phases: 7
spec_dir: specs/002-user-auth

spec:
  feature:
    branch: "002-user-auth"
    status: "In Progress"
  # ... full spec.yaml content

plan:
  approach:
    overview: "Implement OAuth2 authentication..."
  # ... full plan.yaml content

governance:
  constitution:
    name: "Project Constitution"
  # ... full constitution.yaml content when present

tasks:
  - id: T008
    title: "Create auth middleware"
    status: pending
  - id: T009
    title: "Add session management"
    status: pending
  # Only tasks for phase 3
```

### Context File Location

Files are stored in `.autospec/context/`:

```
.autospec/
  context/
    phase-1.yaml
    phase-2.yaml
    phase-3.yaml
```

If `.autospec/` is not writable, falls back to system temp directory with a warning.

### Gitignore Requirement

The context directory should be gitignored. autospec warns if it's not:

```
Warning: '.autospec/context/' not found in .gitignore
```

Add to `.gitignore`:

```gitignore
.autospec/context/
```

Or the parent directory (which autospec also recognizes):

```gitignore
.autospec/
```

### Benefits

| Metric | Without Injection | With Injection |
|--------|-------------------|----------------|
| Time to first task | 15-25 seconds | 3-8 seconds |
| File reads per phase | 3 | 0 |
| Agent context used | Variable | Minimal |
| Task focus | All tasks visible | Only phase tasks |

### Focused Context

The agent only sees tasks for the current phase. This:
- Reduces cognitive load
- Prevents cross-phase confusion
- Keeps the agent focused on the immediate work
- Reduces context token usage

### Debugging Context Issues

View the generated context file before it's cleaned up:

```bash
# Run with --dry-run to create context without executing
# (if available, otherwise run and cancel quickly)

# Or check the file during execution
cat .autospec/context/phase-3.yaml
```

If context files persist after execution, they can be safely deleted:

```bash
rm -rf .autospec/context/
```

---

## Related Documentation

- [Reference](reference.md) - Complete CLI command reference
- [Troubleshooting](troubleshooting.md) - Common issues and solutions
- [Architecture](architecture.md) - System design overview
