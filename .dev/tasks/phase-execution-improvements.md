# Phase Execution Improvements Analysis

> **Status Update (2025-12-16)**: Task-level execution (`--tasks` flag) has been implemented in spec 021-task-level-execution. The context injection optimization (Option A) described below remains a future enhancement.

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

#### Option A: Pre-extract Full Phase Context (Recommended)

Have Go code extract ALL context (spec, plan, AND phase-specific tasks) and inject it directly so Claude doesn't need to read ANY files:

**What gets injected:**
1. **spec.yaml** - Full content (user stories, requirements, acceptance criteria)
2. **plan.yaml** - Full content (technical context, architecture decisions, phases overview)
3. **tasks.yaml (phase-specific)** - Only tasks for phase N with their dependencies

**Implementation approach:**

```go
// In executeSinglePhaseSession()
func buildPhaseContext(specDir string, phaseNumber int) (*PhaseContext, error) {
    // Read and include full spec.yaml
    specContent, _ := os.ReadFile(filepath.Join(specDir, "spec.yaml"))

    // Read and include full plan.yaml
    planContent, _ := os.ReadFile(filepath.Join(specDir, "plan.yaml"))

    // Extract ONLY phase N tasks from tasks.yaml
    phaseTasks := extractPhaseTasksYAML(filepath.Join(specDir, "tasks.yaml"), phaseNumber)

    return &PhaseContext{
        Spec:       specContent,
        Plan:       planContent,
        PhaseTasks: phaseTasks,
        PhaseNum:   phaseNumber,
    }, nil
}

// Write to temp file for Claude to consume
contextFile := writeContextFile(phaseContext)
command := fmt.Sprintf("/autospec.implement --phase %d --context-file %s", phaseNumber, contextFile)
```

**Context file format (YAML):**

```yaml
# Auto-generated phase context - DO NOT read spec.yaml, plan.yaml, or tasks.yaml
# All required context is embedded below

phase: 3
total_phases: 7

spec:
  feature: "Add user authentication"
  user_stories:
    - id: US001
      description: "As a user, I want to log in..."
  # ... full spec.yaml content embedded

plan:
  summary: "Implement OAuth2 authentication flow..."
  technical_context:
    architecture_decisions:
      - "Use JWT tokens for session management"
    # ... full plan.yaml content embedded

tasks:
  # ONLY phase 3 tasks included
  - id: T008
    title: "Implement login endpoint"
    status: Pending
    dependencies: [T007]
  - id: T009
    title: "Add JWT token generation"
    status: Pending
    dependencies: [T008]
```

**Benefits:**

| Benefit | Impact |
|---------|--------|
| **Zero file reads** | Claude doesn't waste time/tokens on Read tool calls |
| **Faster startup** | Eliminates ~5-10 seconds of file reading per phase |
| **Reduced errors** | No chance of reading wrong files or parsing errors |
| **Simpler slash command** | Remove all "REQUIRED: Read X" instructions |
| **Focused context** | Claude sees only relevant tasks, reducing confusion |

**Cost analysis (from research):**

While token savings from reducing initial context are marginal (~5%), the TIME savings are significant:
- Current: Claude makes 3-4 Read tool calls per phase (~5-10 sec each)
- With injection: Zero Read calls needed
- **Per-phase time savings: ~15-30 seconds**
- **For 10-phase spec: ~2.5-5 minutes saved**

**Cons:**
- More Go code to maintain
- Need to update slash command to parse context file
- Context file must be cleaned up after execution

**Slash command changes:**

```markdown
# Current (lines 52-59, 97-108):
3. **Load and analyze the implementation context**:
   - **REQUIRED**: Read tasks.yaml for the complete task list
   - **REQUIRED**: Read plan.yaml for technical_context...
   - **REQUIRED**: Read spec.yaml for user_stories...

# New:
3. **Load context from injected file**:
   - Parse --context-file argument to get context path
   - Read the single context file (contains spec, plan, and phase tasks)
   - DO NOT read spec.yaml, plan.yaml, or tasks.yaml directly
```

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

5. **Pre-extract full phase context (Option A)** - 3-4 hours
   - Create `internal/workflow/context.go` with `PhaseContext` struct
   - Implement `BuildPhaseContext()` to bundle spec + plan + phase tasks
   - Implement `WriteContextFile()` to create temp YAML file
   - Add cleanup logic to remove temp files after execution
   - Modify `executeSinglePhaseSession()` to generate and pass context file
   - Update `.claude/commands/autospec.implement.md` to:
     - Parse `--context-file` argument
     - Read single context file instead of 3 separate files
     - Remove "REQUIRED: Read X" instructions for phase mode

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
| `.claude/commands/autospec.implement.md` | Parse `--context-file`, remove file read instructions |
| `internal/workflow/context.go` (new) | `PhaseContext`, `BuildPhaseContext()`, `WriteContextFile()` |

### New File: `internal/workflow/context.go`

```go
package workflow

import (
    "fmt"
    "os"
    "path/filepath"

    "gopkg.in/yaml.v3"
)

// PhaseContext contains all context needed for a single phase execution
type PhaseContext struct {
    Phase       int                    `yaml:"phase"`
    TotalPhases int                    `yaml:"total_phases"`
    SpecDir     string                 `yaml:"spec_dir"`
    Spec        map[string]interface{} `yaml:"spec"`
    Plan        map[string]interface{} `yaml:"plan"`
    Tasks       []TaskItem             `yaml:"tasks"`
}

// BuildPhaseContext extracts context for a specific phase
func BuildPhaseContext(specDir string, phaseNumber int, totalPhases int) (*PhaseContext, error) {
    ctx := &PhaseContext{
        Phase:       phaseNumber,
        TotalPhases: totalPhases,
        SpecDir:     specDir,
    }

    // Load full spec.yaml
    specPath := filepath.Join(specDir, "spec.yaml")
    specData, err := os.ReadFile(specPath)
    if err != nil {
        return nil, fmt.Errorf("reading spec.yaml: %w", err)
    }
    if err := yaml.Unmarshal(specData, &ctx.Spec); err != nil {
        return nil, fmt.Errorf("parsing spec.yaml: %w", err)
    }

    // Load full plan.yaml
    planPath := filepath.Join(specDir, "plan.yaml")
    planData, err := os.ReadFile(planPath)
    if err != nil {
        return nil, fmt.Errorf("reading plan.yaml: %w", err)
    }
    if err := yaml.Unmarshal(planData, &ctx.Plan); err != nil {
        return nil, fmt.Errorf("parsing plan.yaml: %w", err)
    }

    // Load ONLY phase-specific tasks
    tasksPath := filepath.Join(specDir, "tasks.yaml")
    tasks, err := GetTasksForPhase(tasksPath, phaseNumber)
    if err != nil {
        return nil, fmt.Errorf("extracting phase tasks: %w", err)
    }
    ctx.Tasks = tasks

    return ctx, nil
}

// WriteContextFile writes phase context to a temporary file
func WriteContextFile(ctx *PhaseContext) (string, error) {
    data, err := yaml.Marshal(ctx)
    if err != nil {
        return "", fmt.Errorf("marshaling context: %w", err)
    }

    // Add header comment
    header := []byte("# Auto-generated phase context - DO NOT read spec.yaml, plan.yaml, or tasks.yaml directly\n# All required context is embedded below\n\n")
    data = append(header, data...)

    // Write to temp file in state directory
    tmpFile, err := os.CreateTemp("", fmt.Sprintf("autospec-phase-%d-*.yaml", ctx.Phase))
    if err != nil {
        return "", fmt.Errorf("creating temp file: %w", err)
    }
    defer tmpFile.Close()

    if _, err := tmpFile.Write(data); err != nil {
        return "", fmt.Errorf("writing context file: %w", err)
    }

    return tmpFile.Name(), nil
}
```

---

## Success Metrics

After implementation:

1. **Phase start shows**: Total tasks, task IDs, current status breakdown
2. **Phase complete shows**: Completed/blocked counts, not just checkmark
3. **Zero file reads**: Claude reads 1 context file instead of 3 separate artifact files
4. **Time savings**: ~15-30 seconds per phase from eliminated file read operations
5. **Total time saved**: ~2.5-5 minutes for a 10-phase spec

### Cost vs Time Analysis

| Metric | Without Option A | With Option A | Improvement |
|--------|-----------------|---------------|-------------|
| **File reads per phase** | 3-4 | 1 | 75% fewer |
| **Read tool calls** | ~4 per phase | 1 per phase | 75% fewer |
| **Time per phase startup** | ~20-30 sec | ~5-10 sec | 50-66% faster |
| **Token cost** | baseline | ~5% lower | marginal |
| **Total time (10 phases)** | ~4-5 min reading | ~1-2 min reading | ~3 min saved |

**Key insight**: The primary benefit of Option A is **TIME savings** (eliminating redundant Read tool calls), not token cost reduction. Token savings from context size reduction are marginal (~5%) because conversation accumulation dominates costs.

---

## Extension: Task-Level Context Injection

> **Note**: Task-level execution (`--tasks`, `--from-task`, `--task` flags) was implemented in spec 021-task-level-execution. The context injection optimization below is a **future enhancement** to reduce file reads per task session.

The same context injection approach applies to `--tasks` mode (per-task execution). Each task session would receive:

```yaml
# Auto-generated task context
task:
  id: T008
  title: "Implement login endpoint"
  status: Pending
  dependencies: [T007]
  description: "Create POST /api/auth/login endpoint..."
  acceptance_criteria:
    - "Returns JWT on valid credentials"
    - "Returns 401 on invalid credentials"

# Full spec and plan still included (needed for context)
spec:
  feature: "Add user authentication"
  # ... full spec.yaml

plan:
  summary: "Implement OAuth2 authentication..."
  # ... full plan.yaml

# Dependency tasks included for reference (status only)
dependency_tasks:
  - id: T007
    title: "Set up auth database schema"
    status: Completed
```

**Benefits for task-level execution:**
- Even more focused context (single task vs 4-5 tasks per phase)
- Dependency status clearly visible without reading full tasks.yaml
- Consistent approach across phase and task execution modes
