# 24. CLI Prerequisite Validation

## Summary

Add prerequisite artifact validation to ALL CLI commands before invoking Claude. This provides fast, cheap validation and clear error messages instead of wasting API costs on predictable failures.

## Problem

Currently, CLI commands do NOT consistently validate that required artifacts exist before invoking Claude:

### Core Stage Commands
- `specify` requires constitution but doesn't check (plan/tasks do check constitution)
- `plan` requires constitution + `spec.yaml` - only checks constitution
- `tasks` requires constitution + `plan.yaml` - only checks constitution
- `implement` requires constitution + `tasks.yaml` - doesn't check either

### Optional Stage Commands
- `clarify` requires constitution + `spec.yaml` (refines existing spec)
- `checklist` requires constitution + `spec.yaml` (validates spec requirements)
- `analyze` requires constitution + `spec.yaml` + `plan.yaml` + `tasks.yaml` (validates all artifacts)

### Run Command with Stage Flags
- `autospec run -s` (specify only) - should check constitution
- `autospec run -p` (plan only) - should check constitution + spec.yaml
- `autospec run -pti` (plan, tasks, implement) - should check constitution + spec.yaml (subsequent stages produce their own prereqs)
- `autospec run -ti` (tasks, implement) - should check constitution + plan.yaml

The infrastructure exists (`artifactDependencies` map in `stage_config.go`, `prereqs` command) but isn't used consistently by CLI commands.

### Current Behavior (Without Validation)

```
User runs: autospec plan
├── CLI starts Claude process (~2-5s)
├── Claude loads context, reads template
├── Claude attempts to read spec.yaml → fails
├── Error after 10-30 seconds
└── API cost incurred for predictable failure
```

### Desired Behavior (With Validation)

```
User runs: autospec plan
├── CLI checks spec.yaml exists (~1ms)
├── Error: "spec.yaml not found. Run 'autospec specify' first"
└── No Claude invocation, no cost, instant feedback
```

## Analysis

### Two Execution Paths, Different Contexts

| Aspect | CLI Path (`autospec plan`) | Slash Command Path (`/autospec.plan`) |
|--------|---------------------------|---------------------------------------|
| User context | Terminal, scripts, CI/CD | Inside active Claude session |
| Claude state | Not running, must start | Already running, context loaded |
| Feedback speed | Critical - user waiting | Less critical - conversational |
| Cost of failure | High (startup + API) | Low (already in session) |
| Recovery | Run command again | Claude can explain/assist |

### Can Claude Meaningfully Recover From Missing Artifacts?

| Scenario | Reality |
|----------|---------|
| Spec in different location | `DetectCurrentSpec` already handles this |
| Create spec inline | Violates single-responsibility; use `autospec run` |
| User wants Claude to explain | That's a docs/help issue, not workflow |

**Conclusion**: No practical case where bypassing validation adds value.

### Should There Be a `--skip-prereqs` Flag?

**No.** Key distinction:
- `--skip-preflight` skips OUTPUT validation (retryable)
- Prerequisites are INPUT validation (not retryable - artifact truly missing)

If someone needs to bypass, they can use slash commands directly in Claude Code.

## Solution

Add prerequisite validation to `WorkflowOrchestrator` methods before invoking Claude, including constitution checks for all stages.

### Prerequisite Matrix

**Rule: ALL stages require constitution (except `constitution` command itself)**

#### Core Stages

| Command | Constitution | spec.yaml | plan.yaml | tasks.yaml |
|---------|-------------|-----------|-----------|------------|
| `constitution` | - | - | - | - |
| `specify` | Required | - | - | - |
| `plan` | Required | Required | - | - |
| `tasks` | Required | - | Required | - |
| `implement` | Required | - | - | Required |

#### Optional Stages

| Command | Constitution | spec.yaml | plan.yaml | tasks.yaml |
|---------|-------------|-----------|-----------|------------|
| `clarify` | Required | Required | - | - |
| `checklist` | Required | Required | - | - |
| `analyze` | Required | Required | Required | Required |

#### Run Command Combinations

| Command | Constitution | spec.yaml | plan.yaml | tasks.yaml |
|---------|-------------|-----------|-----------|------------|
| `run -s` | Required | - | - | - |
| `run -p` | Required | Required | - | - |
| `run -t` | Required | - | Required | - |
| `run -i` | Required | - | - | Required |
| `run -sp` | Required | - | - | - |
| `run -spt` | Required | - | - | - |
| `run -spti` / `run -a` | Required | - | - | - |
| `run -pti` | Required | Required | - | - |
| `run -ti` | Required | - | Required | - |

Note: When stages are chained (e.g., `-spt`), only the FIRST stage's artifact prereqs need checking - subsequent stages will have their prereqs created by earlier stages.

### Implementation

1. Add `validatePrerequisites(specDir string, stage Stage) error` helper to `WorkflowOrchestrator`
2. Add `validateConstitution() error` helper (or reuse existing `CheckConstitutionExists()`)
3. Use existing `GetRequiredArtifacts(stage)` from `stage_config.go`
4. For `run` command with flags: use `StageConfig.GetAllRequiredArtifacts()` which already excludes artifacts produced by earlier selected stages
5. Check each required artifact exists
6. Return clear error with actionable suggestion
7. Call validation at start of all Execute* methods

### Files to Modify

1. `internal/workflow/workflow.go` - Add `validatePrerequisites` helper and call from Execute* methods
2. `internal/workflow/workflow_test.go` - Add tests for prerequisite validation
3. `internal/cli/specify.go` - Add constitution check (currently missing)
4. `internal/cli/implement.go` - Add constitution + tasks.yaml check (currently missing both)
5. `internal/cli/clarify.go` - Add constitution + spec.yaml check
6. `internal/cli/checklist.go` - Add constitution + spec.yaml check
7. `internal/cli/analyze.go` - Add constitution + all artifacts check
8. `internal/cli/run.go` - Add prerequisite validation before executing selected stages

### Error Messages

```
# Missing constitution (all stages except constitution itself)
Error: Project constitution not found at .autospec/memory/constitution.yaml
Run 'autospec constitution' first to create project principles.

# Missing spec.yaml before plan/clarify/checklist/analyze
Error: spec.yaml not found in specs/001-my-feature/
Run 'autospec specify' first to create the specification.

# Missing plan.yaml before tasks/analyze
Error: plan.yaml not found in specs/001-my-feature/
Run 'autospec plan' first to create the implementation plan.

# Missing tasks.yaml before implement/analyze
Error: tasks.yaml not found in specs/001-my-feature/
Run 'autospec tasks' first to generate the task breakdown.

# Multiple missing artifacts (analyze)
Error: Missing required artifacts in specs/001-my-feature/:
  - plan.yaml (run 'autospec plan')
  - tasks.yaml (run 'autospec tasks')
```

## Consistency

This approach is consistent with existing patterns:
- Constitution check already exists in `plan.go`, `tasks.go` - extend to all other stages
- Uses existing `artifactDependencies` infrastructure in `stage_config.go`
- `StageConfig.GetAllRequiredArtifacts()` already handles the "chain" logic for `run` command
- Matches GitHub SpecKit's prerequisite shell script approach

## Implementation Notes

Consider centralizing validation in `WorkflowOrchestrator` rather than duplicating in each CLI command:
- Add `ValidateStagePrerequisites(specName string, stages []Stage) error` method
- Handles constitution check + artifact checks in one place
- CLI commands call this before invoking Claude
- Easier to maintain and test

## Spec Command

```bash
autospec specify "Add prerequisite validation to all CLI commands. All stages (except constitution) must check constitution exists. Core stages (plan, tasks, implement) and optional stages (clarify, checklist, analyze) must check their required artifacts exist before invoking Claude. See .dev/tasks/024-cli-prerequisite-validation.md for full details."
```
