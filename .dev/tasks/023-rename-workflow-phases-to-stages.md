# 23. Rename Workflow Phases to Stages

## Summary

Rename the high-level workflow "phases" (specify, plan, tasks, implement) to "stages" to eliminate naming confusion with task "phases" in tasks.yaml.

## Problem

The term "phases" is overloaded in two distinct contexts:

1. **Workflow phases** - High-level stages: specify → plan → tasks → implement
2. **Task phases** - Numbered groupings in tasks.yaml (e.g., "Phase 1: Setup", "Phase 2: Core Auth")

This causes confusion because:
- CLI flags `--phases`, `--phase N`, `--from-phase N` refer to task phases, not workflow phases
- Documentation uses "phase" for both concepts inconsistently
- Code has `Phase` type for workflow AND `Phase` struct for task groupings
- Recent features (specs 020, 021, 022) heavily use both concepts, making code hard to follow

## Solution

Rename workflow phases to **stages** because:
- "Stage" feels higher-level (stages of a rocket, stages of development)
- "Phase" feels like subdivisions within a stage
- This creates a clear hierarchy: workflow stages contain implementation phases

### Mental Model

```
Workflow Stages (high-level):
  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
  │   SPECIFY   │ → │    PLAN     │ → │   TASKS     │ → │  IMPLEMENT  │
  │   stage     │    │   stage     │    │   stage     │    │   stage     │
  └─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
                                                                  │
                                                                  ▼
                                              Implementation Phases (sub-divisions):
                                              ┌──────────┬──────────┬──────────┐
                                              │ Phase 1  │ Phase 2  │ Phase 3  │
                                              │  Setup   │Core Auth │ Testing  │
                                              └──────────┴──────────┴──────────┘
```

### Resulting CLI

```bash
# Workflow STAGES (high-level)
autospec run -s        # Include specify stage
autospec run -p        # Include plan stage
autospec run -spti     # All stages

# Implementation PHASES (within implement stage)
autospec implement --phases        # Run each phase separately
autospec implement --phase 3       # Run only phase 3
autospec implement --from-phase 3  # Start from phase 3
```

The `--phases`, `--phase`, `--from-phase` flags stay the same - they already refer to the correct concept (task phases).

## Implementation

### Code Changes

1. `internal/workflow/executor.go`
   - Rename `type Phase string` to `type Stage string`
   - Rename constants: `PhaseSpecify` → `StageSpecify`, `PhasePlan` → `StagePlan`, etc.

2. `internal/workflow/phase_config.go` → `internal/workflow/stage_config.go`
   - Rename `PhaseConfig` → `StageConfig`
   - Update `GetCanonicalOrder()` to return stages
   - Update `ArtifactDependency` comments

3. `internal/workflow/workflow.go`
   - Update method signatures and variable names

4. `internal/workflow/preflight.go`
   - Update artifact dependency messages

5. `internal/cli/*.go`
   - Update help text to use "stage" for workflow concepts
   - Keep "phase" in implement.go for task phases

6. `internal/retry/retry.go`
   - Rename `MarkPhaseComplete` → `MarkStageComplete` (if it refers to workflow)

### Documentation Changes

1. `CLAUDE.md` - Update terminology throughout
2. `README.md` - Update terminology throughout
3. `.claude/commands/*.md` - Update references
4. `docs/*.md` - Update all documentation

### Key Distinction

| Concept | Term | Example | CLI |
|---------|------|---------|-----|
| Workflow stages | "stage" | specify, plan, tasks, implement | `-s`, `-p`, `-t`, `-i` |
| Task groupings | "phase" | Phase 1: Setup, Phase 2: Core Auth | `--phases`, `--phase N` |

## Spec Command

```bash
autospec specify "Rename workflow phases to stages. See .dev/tasks/023-rename-workflow-phases-to-stages.md for full details."
```
