# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Testing
```bash
# Run all tests (60+ bats-core tests)
./tests/run-all-tests.sh

# Run specific test suite
bats tests/lib/validation-lib.bats
bats tests/scripts/workflow-validate.bats
bats tests/hooks/stop-speckit-implement.bats

# Run with verbose output
bats -t tests/integration.bats
```

### Development
```bash
# Run workflow validation for a feature
./scripts/speckit-workflow-validate.sh <feature-name>

# Check implementation status
./scripts/speckit-implement-validate.sh <feature-name>

# Run with debug output
SPECKIT_DEBUG=true ./scripts/speckit-workflow-validate.sh <feature-name>

# Dry run mode
./scripts/speckit-workflow-validate.sh --dry-run <feature-name>
```

### Linting
```bash
# Validate all bash scripts
shellcheck scripts/**/*.sh

# Check for shellcheck errors (required before PR)
find scripts -name "*.sh" -exec shellcheck {} \;
```

## Architecture Overview

Auto Claude SpecKit is a validation and enforcement system for SpecKit workflows. It operates through four architectural layers:

### 1. Validation Library (`scripts/lib/speckit-validation-lib.sh`)

The foundation providing reusable validation functions:
- **Exit Codes**: Standardized codes (0=success, 1=failed, 2=exhausted, 3=invalid, 4=missing deps)
- **Retry State Management**: Persistent retry tracking in `/tmp` (not in-memory)
- **File Validation**: Check artifact existence (spec.md, plan.md, tasks.md)
- **Task Counting**: Parse tasks.md to count unchecked tasks and detect phase completion
- **Continuation Prompts**: Generate context-aware prompts for incomplete work

All scripts source this library and use its functions for consistent behavior.

### 2. Workflow Scripts (`scripts/`)

Orchestrate complete SpecKit workflows with validation:
- **speckit-workflow-validate.sh**: Runs specify → plan → tasks with automatic retry
- **speckit-implement-validate.sh**: Validates implementation progress and generates continuation prompts

These scripts call SpecKit commands, validate outputs, and retry up to 3 times on failure.

### 3. Hook Scripts (`scripts/hooks/`)

Integrate with Claude Code's hook system to enforce completeness:
- **stop-speckit-specify.sh**: Blocks stopping until spec.md exists
- **stop-speckit-plan.sh**: Blocks stopping until plan.md exists
- **stop-speckit-tasks.sh**: Blocks stopping until tasks.md exists
- **stop-speckit-implement.sh**: Blocks stopping until all tasks checked
- **stop-speckit-clarify.sh**: Blocks stopping until clarifications captured

Hooks validate artifacts, increment retry counts, and either `allow_stop` or `block_stop` based on validation results and retry exhaustion.

### 4. SpecKit Commands (`.claude/commands/speckit.*.md`)

Slash commands that execute SpecKit workflows:
- `/speckit.specify`: Create feature specification
- `/speckit.plan`: Generate implementation plan
- `/speckit.tasks`: Generate task breakdown
- `/speckit.implement`: Execute implementation
- `/speckit.clarify`: Identify and resolve underspecified areas
- `/speckit.analyze`: Cross-artifact consistency analysis
- `/speckit.checklist`: Generate custom checklists
- `/speckit.constitution`: Update project constitution

These commands produce artifacts in `specs/<feature-name>/` that the validation layer checks.

## Key Concepts

### Retry State Management

Retry counts are persisted to `/tmp/speckit-retry-<spec>-<command>` files (not environment variables):
- Scripts increment retry count on failure
- Hooks check retry count to decide block vs. allow
- Retry limit: 3 by default (configurable via `SPECKIT_MAX_RETRIES`)
- After exhaustion, hooks allow stop to prevent infinite loops

### Hook Behavior

Hooks follow a specific decision flow:
1. Detect current spec from git branch or directory
2. Check if required artifact exists
3. If exists: `allow_stop` (exit 0)
4. If missing: Check retry count
5. If retries exhausted: `allow_stop` with warning
6. If retries remain: `block_stop` with continuation prompt (exit 1)

This ensures validation without creating unbreakable deadlocks.

### Exit Code Conventions

All scripts use standardized exit codes from the validation library:
- `0`: Success
- `1`: Validation failed (retryable)
- `2`: Retry limit exhausted
- `3`: Invalid arguments
- `4`: Missing dependencies

These codes support programmatic composition and CI/CD integration.

### Spec Detection

Scripts automatically detect the current spec from:
1. Git branch name (e.g., `002-speckit-validation-hooks` → `002`)
2. Directory name pattern in `specs/` (e.g., `specs/002-*/`)
3. Most recently modified spec directory

This allows validation to work without explicit spec name arguments.

## Constitution Principles

Development in this repository follows the constitution at `.specify/memory/constitution.md`:

1. **Validation-First**: All workflow transitions must be validated; automatic retry for failures
2. **Hook-Based Enforcement**: Quality gates via Claude Code hooks while maintaining developer agency
3. **Test-First Development** (NON-NEGOTIABLE): 60+ test minimum; tests written before implementation
4. **Performance Standards**: Sub-second validation (<1s); targets: workflow ~0.22s, implementation ~0.15s, hooks ~0.08s
5. **Idempotency & Retry Logic**: All scripts idempotent; configurable retry limits; persistent state

Before making changes:
- Write failing tests first
- Run `./tests/run-all-tests.sh` to verify tests pass
- Run `shellcheck` on modified scripts
- Verify performance targets (<1s for validation operations)
- Check constitution compliance

## SpecKit Workflow Integration

This tool integrates with the SpecKit feature development workflow:

```
User: /speckit.specify "feature description"
  ↓ produces specs/<feature>/spec.md
  ↓ validated by scripts/hooks/stop-speckit-specify.sh

User: /speckit.plan
  ↓ produces specs/<feature>/plan.md, research.md, data-model.md, contracts/
  ↓ validated by scripts/hooks/stop-speckit-plan.sh

User: /speckit.tasks
  ↓ produces specs/<feature>/tasks.md
  ↓ validated by scripts/hooks/stop-speckit-tasks.sh

User: /speckit.implement
  ↓ executes tasks from tasks.md
  ↓ validated by scripts/hooks/stop-speckit-implement.sh (checks all tasks checked)
```

Each validation hook ensures the artifact exists and is complete before allowing workflow progression.

## Testing Architecture

Tests are organized in `tests/` using bats-core:

- **tests/lib/**: Unit tests for validation library functions
- **tests/scripts/**: Tests for workflow and implementation scripts
- **tests/hooks/**: Tests for stop hook behavior
- **tests/integration.bats**: End-to-end workflow tests
- **tests/quickstart-validation.bats**: Validates README examples work
- **tests/fixtures/**: Mock artifacts (spec.md, tasks.md, etc.)
- **tests/mocks/**: Mock external commands
- **tests/test_helper.bash**: Shared test utilities

Key testing principle: All functionality has corresponding tests. Test coverage must not decrease below 60+ tests baseline.

## Important Implementation Details

### Prerequisites

Required dependencies (checked at runtime):
- Bash 4.0+
- jq 1.6+ (JSON parsing for configuration)
- git (spec detection from branches)
- grep, sed (text processing)
- `specify` CLI tool (SpecKit templates)

Scripts validate dependencies using `check_dependencies` from the validation library and exit with code 4 if missing.

### Configuration

Environment variables control behavior:
- `SPECKIT_RETRY_LIMIT`: Max retry attempts (default: 2, meaning 3 total tries)
- `SPECKIT_SPECS_DIR`: Spec directory location (default: `./specs`)
- `SPECKIT_DEBUG`: Enable debug logging (default: `false`)
- `SPECKIT_DRY_RUN`: Dry run mode (default: `false`)
- `SPECKIT_VALIDATION_TIMEOUT`: Validation timeout in seconds (default: 5)

### Performance Optimization

Validation speed is critical (<1s target):
- Use `grep -q` for existence checks (don't capture output)
- Use `find -maxdepth 1` to limit directory traversal
- Cache spec detection results within script runs
- Avoid subshells where possible
- Use `set -euo pipefail` for fast failure

Performance regressions beyond 5 seconds require immediate attention per constitution.

### Hook Integration

To enable hooks in Claude Code:
1. Copy `.claude/spec-workflow-settings.json` template
2. Update `{{HOOK_SCRIPT}}` placeholders with absolute paths to hook scripts
3. Launch Claude with: `claude --settings .claude/your-settings.json`

Hooks receive payload via stdin (reserved for future use) and output decisions to stdout.

## Working with SpecKit Templates

Templates are in `.specify/templates/`:
- **spec-template.md**: Feature specification structure
- **plan-template.md**: Implementation plan structure
- **tasks-template.md**: Task breakdown structure
- **checklist-template.md**: Custom checklist structure
- **agent-file-template.md**: AI agent context structure

These templates use `[PLACEHOLDER]` tokens that SpecKit commands fill in. When modifying templates:
1. Preserve placeholder format: `[ALL_CAPS_IDENTIFIER]`
2. Update corresponding command in `.claude/commands/speckit.*.md`
3. Verify constitution alignment (e.g., Constitution Check in plan-template.md)
4. Test with actual SpecKit command execution

## Common Patterns

### Adding a New Validation Hook

1. Create hook script in `scripts/hooks/stop-speckit-<command>.sh`
2. Source validation library: `source "$SCRIPT_DIR/../lib/speckit-validation-lib.sh"`
3. Implement validation logic using library functions
4. Add corresponding test in `tests/hooks/stop-speckit-<command>.bats`
5. Update `.claude/spec-workflow-settings.json` template with new hook

### Adding a New Validation Function

1. Add function to `scripts/lib/speckit-validation-lib.sh`
2. Document usage in function comments
3. Write unit tests in `tests/lib/validation-lib.bats`
4. Run full test suite to verify no regressions
5. Update this CLAUDE.md if it's a major new capability

### Debugging Validation Failures

1. Enable debug mode: `SPECKIT_DEBUG=true ./scripts/...`
2. Check retry state files: `ls -la /tmp/speckit-retry-*`
3. Manually run validation function from library
4. Check artifact existence: `find specs -name "spec.md"`
5. Verify git root detection: `git rev-parse --show-toplevel`
