# Phase Execution Improvements Analysis

## Current Architecture Summary

### How Phase Isolation Works

The flow from CLI to Claude:

```
autospec implement --phases
    ↓
Go (workflow.go) iterates phases 1..N
    ↓
For each phase: builds command "/autospec.implement --phase N"
    ↓
Claude CLI receives: claude -p "/autospec.implement --phase N"
    ↓
$ARGUMENTS in slash command = "--phase N"
    ↓
Claude reads .claude/commands/autospec.implement.md instructions
    ↓
Instructions tell Claude to:
  - Parse --phase N argument
  - Read tasks.yaml and locate phase N
  - Execute ONLY tasks within phase N
  - Exit after phase N completion
```

**Key insight**: The phase isolation is enforced by **instructions in the slash command**, not by filtering what Claude can see. Claude reads the full `tasks.yaml` and filters based on instructions.

---

## Issue Analysis

### Issue 1: Phase Completion Summary Missing Task Counts

**Current behavior**:
```
✓ Phase 2 complete
```

**Desired behavior**:
```
✓ Phase 2 complete (4/4 tasks completed, 0 blocked)
```

**Root cause**: `workflow.go:592` just prints a static message:
```go
fmt.Printf("✓ Phase %d complete\n\n", phase.Number)
```

**Fix location**: `internal/workflow/workflow.go` lines 591-592

**Solution**: After calling `IsPhaseComplete()`, call `GetPhaseInfo()` to get detailed stats:
```go
phaseInfo, _ := validation.GetPhaseInfo(tasksPath)
for _, p := range phaseInfo {
    if p.Number == phase.Number {
        fmt.Printf("✓ Phase %d complete (%d/%d tasks, %d blocked)\n\n",
            phase.Number, p.CompletedTasks, p.TotalTasks, p.BlockedTasks)
        break
    }
}
```

---

### Issue 2: Pre-Phase Task Summary Missing

**Current behavior**:
```
[Phase 2/7] Foundational - Task Validation Infrastructure
Executing: /autospec.implement --phase 2
```

**Desired behavior**:
```
[Phase 2/7] Foundational - Task Validation Infrastructure
  → 4 tasks: T002, T003, T004, T005
  → Status: 0 completed, 0 blocked, 4 pending
Executing: /autospec.implement --phase 2
```

**Root cause**: `workflow.go:573` only shows phase title:
```go
fmt.Printf("[Phase %d/%d] %s\n", phase.Number, totalPhases, phase.Title)
```

**Fix location**: `internal/workflow/workflow.go` around line 573

**Solution**: Enhance the phase header display:
```go
// After getting phase info, before executing:
fmt.Printf("[Phase %d/%d] %s\n", phase.Number, totalPhases, phase.Title)
fmt.Printf("  → %d tasks: %s\n", phase.TotalTasks, strings.Join(getTaskIDs(phase), ", "))
fmt.Printf("  → Status: %d completed, %d blocked, %d pending\n",
    phase.CompletedTasks, phase.BlockedTasks,
    phase.TotalTasks - phase.CompletedTasks - phase.BlockedTasks)
```

---

### Issue 3: Claude Reads Entire tasks.yaml (Redundant Context)

**Current behavior**: Each phase session Claude:
1. Runs `autospec prereqs --json --require-tasks --include-tasks`
2. Reads full `tasks.yaml` (428 lines)
3. Reads full `plan.yaml`
4. Reads full `spec.yaml`
5. Filters to only work on phase N tasks

**The waste**: For a 7-phase spec, Claude reads ~1200+ lines of context 7 times instead of getting just the relevant portion.

**Root cause**: The slash command instructions (lines 52-59, 97-108) tell Claude to read these files every time:
```markdown
1. **Setup**: Run the prerequisites command to get feature paths
3. **Load and analyze the implementation context**:
   - **REQUIRED**: Read tasks.yaml for the complete task list
   - **REQUIRED**: Read plan.yaml for technical_context...
   - **REQUIRED**: Read spec.yaml for user_stories...
```

**Potential solutions**:

#### Option A: Pre-extract Phase Context (Recommended)

Have Go code extract only relevant context and pass it via a temp file or extended arguments:

```go
// In executeSinglePhaseSession()
phaseTasks := extractPhaseTasksYAML(tasksPath, phaseNumber)
contextFile := createTempContextFile(specName, phaseNumber, phaseTasks)
command := fmt.Sprintf("/autospec.implement --phase %d --context-file %s", phaseNumber, contextFile)
```

**Pros**: Minimal context per session, faster startup, less token usage
**Cons**: Requires modifying slash command to accept context file, more Go code

---

## Recommended Implementation Order

### Quick Wins (Low effort, High value)

1. **Add phase completion stats** - 15 min
   - Modify `workflow.go:591-592` to show task counts
   - Data already available from `GetPhaseInfo()`

2. **Add pre-phase task summary** - 20 min
   - Modify `workflow.go:573` to show task list and status
   - Data already available from `GetPhaseInfo()`

### Medium Effort (Higher value)

3. **Add helper functions** - 30 min
   - `GetTaskIDsForPhase(phaseNumber int) []string`
   - `FormatPhaseStats(phase PhaseInfo) string`
   - Put in `internal/validation/tasks_yaml.go`

4. **Update slash command for early filtering** - 20 min
   - Modify instructions to parse `--phase N` FIRST
   - Skip full file reads when phase specified
   - Only read phase N section from tasks.yaml

### Larger Effort (Best optimization)

5. **Pre-extract phase context** - 2-3 hours
   - Create `internal/workflow/context.go`
   - Extract only relevant tasks/spec/plan sections
   - Write to temp file or pass as argument
   - Modify slash command to use pre-extracted context

---

## Data Structures Available

From `internal/validation/tasks_yaml.go`:

```go
// PhaseInfo already has what we need
type PhaseInfo struct {
    Number         int
    Title          string
    TotalTasks     int
    CompletedTasks int
    BlockedTasks   int
    ActionableTasks int // pending tasks ready to work
}

// Functions available
GetPhaseInfo(tasksPath string) ([]PhaseInfo, error)
IsPhaseComplete(tasksPath string, phaseNumber int) (bool, error)
GetTotalPhases(tasksPath string) (int, error)
GetFirstIncompletePhase(tasksPath string) (int, *PhaseInfo, error)
```

**Missing functions to add**:
```go
GetTasksForPhase(tasksPath string, phaseNumber int) ([]TaskItem, error)
FormatPhaseHeader(phase PhaseInfo) string
FormatPhaseCompletion(phase PhaseInfo) string
```

---

## Files to Modify

| File | Changes |
|------|---------|
| `internal/workflow/workflow.go` | Add stats to phase header/completion messages |
| `internal/validation/tasks_yaml.go` | Add `GetTasksForPhase()` helper |
| `.claude/commands/autospec.implement.md` | Optional: optimize context loading |
| `internal/workflow/context.go` (new) | Optional: pre-extract context |

---

## Success Metrics

After implementation:

1. **Phase start shows**: Total tasks, task IDs, current status breakdown
2. **Phase complete shows**: Completed/blocked counts, not just checkmark
3. **Context reduction**: Each phase session reads only ~20% of current context (if Option A implemented)
4. **Time savings**: ~10-15 seconds per phase from reduced context loading
