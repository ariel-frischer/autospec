# Backlog

Future enhancements and technical debt items.

---

## 007 - Implement Retry Loop for /speckit.implement

**Priority:** High
**Status:** Todo
**Created:** 2024-12-13

### Problem

Claude stops after completing only some tasks during `/speckit.implement`. The retry tracking system exists but there's no actual loop that re-invokes Claude with continuation context.

### Root Causes

1. **No retry loop**: `ExecutePhase()` runs Claude once, validates, increments retry count, and returns error - no loop
2. **Unused continuation prompt**: `GenerateContinuationPrompt()` is implemented but never called
3. **Low default retries**: `max_retries: 3` is insufficient for large task lists
4. **Prompt lacks persistence**: `speckit.implement.md` doesn't mandate completing ALL tasks

### Tasks

- [ ] Add retry loop in `ExecuteImplement()` that re-invokes Claude on validation failure
- [ ] Wire up `GenerateContinuationPrompt()` to provide remaining task context on retry
- [ ] Add `max_retries_implement` config option (default: 10) separate from general max_retries
- [ ] Add `implement_prompt_suffix` config option for custom completion instructions
- [ ] Enhance `speckit.implement.md` with completion mandate ("MUST complete ALL tasks")
- [ ] Handle checklist blocker (Step 2 interactive prompt) - add `--force` or `--no-checklist` flag
- [ ] Add `--until-complete` flag for infinite retries until all tasks done
- [ ] Update docs with new implement retry behavior

### Implementation Notes

**Retry loop location:** `internal/workflow/workflow.go` in `ExecuteImplement()`

```go
for {
    result, err := w.Executor.ExecutePhase(...)
    if err == nil {
        break // all tasks complete
    }
    if result.Exhausted {
        return err
    }
    // Build continuation prompt
    phases, _ := validation.ParseTasksByPhase(tasksPath)
    prompt := validation.GenerateContinuationPrompt(specDir, "implement", phases)
    command = "/speckit.implement \"" + prompt + "\""
}
```

**Prompt enhancement for `.claude/commands/speckit.implement.md`:**

```markdown
## Critical Requirements

**COMPLETION MANDATE**: You MUST complete ALL tasks in tasks.md before stopping.
Do NOT stop after completing just some tasks. Continue until every task
shows `[X]` (checked).

If you receive this command with a continuation prompt (remaining tasks listed),
focus on those specific unchecked tasks.
```

### Related Files

- `internal/workflow/workflow.go` - `ExecuteImplement()`
- `internal/workflow/executor.go` - `ExecutePhase()`
- `internal/validation/prompt.go` - `GenerateContinuationPrompt()` (unused)
- `internal/validation/tasks.go` - `ParseTasksByPhase()`
- `internal/config/defaults.go` - default config values
- `.claude/commands/speckit.implement.md` - the prompt

### References

- Analysis doc: `docs/claude-agent-sdk-evaluation.md`

---

## 008 - Parallel Spec Execution with Sandbox Isolation

**Priority:** Medium
**Status:** Research Complete
**Created:** 2024-12-13

### Summary

Enable running multiple specs in parallel using git worktrees for file isolation and `srt` (sandbox-runtime) for process isolation.

### Tasks

- [ ] Add git worktree management to Go binary (`internal/worktree/`)
- [ ] Integrate `srt` CLI for sandbox isolation
- [ ] Add `autospec parallel <spec1> <spec2> ...` command
- [ ] Add `.srt-settings.json` default config generation
- [ ] Document sandbox network allowlist requirements
- [ ] Add `--sandbox` flag to implement command

### References

- Research doc: `docs/claude-agent-sdk-evaluation.md`
- Sandbox runtime: https://github.com/anthropic-experimental/sandbox-runtime
