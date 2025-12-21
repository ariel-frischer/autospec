# Auto-Commit Injection Refactor

Improve how auto-commit instructions are injected into workflow commands. Current approach shows verbose text to users and conflates system instructions with user arguments.

---

## Problem Statement

When running `autospec run -pti` or ANY workflow commands with auto-commit enabled:

1. **Verbose output pollution**: Full auto-commit instructions (~90 lines) are displayed to the user as part of the command execution output
2. **Conflation of concerns**: Auto-commit instructions are appended to the command string (e.g., `/autospec.plan "args"\n\n[huge block of text]`), treating system instructions as user input
3. **Over-verbose instructions**: The 90-line instruction block is excessive—Claude/agents already understand conventional commits, git status, and gitignore patterns
4. **Poor UX**: Users see a wall of instructional text before the actual workflow begins

**Current flow:**
```
User runs: autospec run -pti
→ Executor builds: /autospec.plan "feature description"
→ InjectAutoCommitInstructions() appends 90 lines
→ Agent receives: /autospec.plan "feature description"\n\n## Auto-Commit Instructions\n[huge block]
→ User sees ALL of this in terminal output
```

---

## Goals

1. **Separate system instructions from user arguments** in command templates
2. **Compact output** - show `[Injected: AutoCommit]` or similar, not full text
3. **Minimal instructions** - agents know conventions; just remind the essential steps
4. **Consistent injection pattern** for any future injectable instructions (not just auto-commit)

---

## Proposed Solution

### Phase 1: Command Template System Instructions Argument

Add a dedicated `$SYSTEM_INSTRUCTIONS` placeholder to command templates (or a separate mechanism for system-level directives).

**Option A: Template placeholder**
```markdown
# internal/commands/autospec.plan.md
---
description: Generate YAML implementation plan
version: "1.0.0"
---

{{#if SYSTEM_INSTRUCTIONS}}
## System Directives
{{{SYSTEM_INSTRUCTIONS}}}
{{/if}}

## User Input
```text
$ARGUMENTS
```
```

**Option B: Agent-level system prompt injection**
Instead of modifying templates, inject system instructions at the agent execution layer (separate from the prompt). This keeps templates clean and project-agnostic.

**Recommendation:** Option B is cleaner—command templates in `internal/commands/` should remain project-agnostic per constitution. System instructions are an execution concern, not a template concern.

### Phase 2: Output Compaction

When displaying the command being executed, detect and compact injected instructions:

**Current output:**
```
→ Executing: claude -p /autospec.plan

## Auto-Commit Instructions

After completing your implementation work, create a clean git commit following these steps:

### Step 1: Update .gitignore
[... 80 more lines ...]
```

**Proposed output:**
```
→ Executing: claude -p /autospec.plan
  [+AutoCommit]
```

Or for verbose mode (`-v`):
```
→ Executing: claude -p /autospec.plan
  [+AutoCommit: post-work git commit with conventional format]
```

**Implementation:**
- Add a `DisplayHint` field to injectable instructions
- In output formatting, detect instructions by marker/prefix and replace with compact form
- Keep full instructions in debug logs

### Phase 3: Minimal Instructions

Replace the 90-line instruction block with ~15 lines. Agents understand:
- Conventional commit format (type(scope): message)
- gitignore patterns
- Git workflow

**Current instructions (90 lines):**
- Lists every possible gitignore pattern
- Explains conventional commit from scratch
- Provides example commands

**Proposed instructions (~10 lines):**
```
## Auto-Commit

After completing implementation:

1. Check for changes:
   git status --short

2. If untracked files exist that should be ignored, update .gitignore

3. Stage and commit:
   git add -A
   git commit -m "type(scope): description"

Skip if no changes or detached HEAD.
```

**Rationale:**
- `git status --short` first (identify the situation before acting)
- Assume agent knows gitignore patterns (it does)
- Assume agent knows conventional commit format (it does)
- Just remind the workflow, don't teach from scratch

---

## Implementation Plan

### Task 1: Refactor Instruction Injection

**Files:**
- `internal/workflow/autocommit.go` - Simplify instructions
- `internal/workflow/executor.go` - Modify `InjectAutoCommitInstructions()`

**Changes:**
1. Replace verbose instructions with minimal version
2. Add injection metadata (name, display hint)

```go
type InjectableInstruction struct {
    Name        string  // "AutoCommit"
    DisplayHint string  // "post-work git commit with conventional format"
    Content     string  // The actual instructions
}

func InjectInstructions(command string, instructions []InjectableInstruction) string {
    // Build injection block with markers for detection
}
```

### Task 2: Output Compaction

**Files:**
- `internal/workflow/executor.go` - Compact output before display
- `internal/workflow/claude.go` - Agent execution output handling

**Changes:**
1. Before displaying command to user, detect instruction blocks
2. Replace with `[+Name]` or `[+Name: hint]` format
3. Keep full content for actual agent execution

```go
const instructionMarkerStart = "<!-- AUTOSPEC_INJECT:"
const instructionMarkerEnd = "-->"

func CompactInstructionsForDisplay(command string) string {
    // Detect markers, extract name, replace block with [+Name]
}
```

### Task 3: Update Internal Commands (If Needed)

If Option A (template placeholder) is chosen:
- Add `$SYSTEM_INSTRUCTIONS` handling to template processing
- Update affected templates

If Option B (agent-level):
- No template changes needed
- Update agent execution to handle system instructions separately

---

## Files to Modify

| File | Changes |
|------|---------|
| `internal/workflow/autocommit.go` | Simplify `autoCommitInstructions` constant to ~15 lines |
| `internal/workflow/executor.go` | Add output compaction, refactor injection with markers |
| `internal/workflow/claude.go` | Pass-through (instructions stay in prompt) |
| `internal/workflow/stage_executor.go` | No changes (calls executor) |

---

## Testing

1. **Unit tests:**
   - `TestAutoCommitInstructionsMinimal` - Verify new instructions are <20 lines
   - `TestCompactInstructionsForDisplay` - Verify marker detection and replacement
   - `TestInjectInstructionsFull` - Verify full injection still works for agent

2. **Integration tests:**
   - Run `autospec run -pti` and verify output is compact
   - Run with `--verbose` and verify display hint appears
   - Verify agent still receives full instructions

---

## Non-Goals

- **Removing auto-commit feature**: Just improving injection/display
- **Making templates project-aware**: Templates stay agnostic
- **Supporting arbitrary injection hooks**: Focus on auto-commit for now; pattern enables future expansion

---

## Success Criteria

1. User output shows `[+AutoCommit]` instead of 90 lines
2. Instructions are ≤20 lines
3. First action in instructions is `git status --short`
4. No regressions in actual commit behavior
5. Works for all workflow commands (specify, plan, tasks, implement, etc.)

---

## Related

- `internal/workflow/autocommit.go` - Current implementation
- `internal/workflow/executor.go:565-577` - Current injection point
- `internal/cli/shared/auto_commit.go` - Flag handling
- `.dev/tasks/core-feature-improvements.md` - Pattern for incremental improvements
