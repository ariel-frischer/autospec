# Priority Unit Tests to Add

Go unit tests only. No integration tests or scripts/ testing.

## Critical Priority

### 1. `internal/workflow/executor_test.go`

The `executor.go` file (270 lines) is the core execution engine with **zero test coverage**.

**Functions to test:**
- `ExecutePhase()` - Retry logic with 3 paths: success, execution failure, validation failure
- `ExecuteWithRetry()` - Simple retry loop
- `getPhaseNumber()` - Phase numbering logic
- `buildPhaseInfo()` - Progress tracking construction

**Test cases:**
- [ ] Successful phase execution with validation passing
- [ ] Execution failure triggers retry increment
- [ ] Validation failure triggers retry with state persistence
- [ ] Max retries exhausted returns appropriate error
- [ ] Retry state loaded correctly on startup
- [ ] Retry state reset on success
- [ ] Phase number calculation for all phases
- [ ] Progress info construction with correct phase counts

---

### 2. `internal/cli/update_task_test.go`

The `update_task.go` file (191 lines) has complex YAML node traversal with **no tests**.

**Functions to test:**
- `findAndUpdateTask()` - Recursive YAML traversal (4 node types)
- Status validation (valid statuses: Pending, InProgress, Completed, Blocked)
- YAML parsing and output

**Test cases:**
- [ ] Find task in flat task list
- [ ] Find task in nested phases structure
- [ ] Find task in deeply nested sequence/mapping
- [ ] Task not found returns appropriate error
- [ ] Invalid status rejected with error message
- [ ] YAML structure preserved after update
- [ ] Edge case: task ID appears in non-task context (ignored)

---

### 3. `internal/validation/tasks_yaml_test.go`

The `tasks_yaml.go` file (200+ lines) handles YAML task parsing with **no tests**.

**Functions to test:**
- `ParseTasksYAML()` - YAML structure parsing
- `GetTaskStats()` - Stats calculation
- `AreAllTasksComplete()` - Completion checking
- Status extraction from task nodes

**Test cases:**
- [ ] Parse valid tasks.yaml with multiple phases
- [ ] Parse tasks.yaml with nested task structure
- [ ] Count tasks by status correctly
- [ ] All tasks complete returns true
- [ ] Incomplete tasks returns false with count
- [ ] Invalid YAML returns descriptive error
- [ ] Empty tasks file handled gracefully

---

### 4. `internal/cli/all_test.go`

The `all.go` file (135 lines) orchestrates the full workflow with **no tests**.

**Functions to test:**
- `runAllWorkflow()` - Workflow orchestration logic
- Flag handling (`--yes`, `--spec`)
- Configuration override merging

**Test cases:**
- [ ] Phase config created with all phases enabled
- [ ] Skip confirmations flag propagated
- [ ] Spec name override applied
- [ ] Feature description passed to orchestrator
- [ ] Error from orchestrator propagated correctly

---

## High Priority

### 5. `internal/cli/config_test.go` (expand existing)

The `config.go` file (200+ lines) handles config management commands.

**Functions to test:**
- `showConfig()` - Configuration display
- `migrateConfig()` - Migration trigger
- JSON output formatting

**Test cases:**
- [ ] Show config outputs all fields
- [ ] Show config with `--json` produces valid JSON
- [ ] Migrate command calls migration utility
- [ ] Migrate with `--dry-run` doesn't modify files
- [ ] Migrate with `--user` targets user config
- [ ] Migrate with `--project` targets project config

---

### 6. `internal/cli/init_test.go`

The `init.go` file (400+ lines) is the largest CLI file with **no tests**.

**Functions to test:**
- User-level config initialization
- Project-level config initialization
- Path creation and file writing

**Test cases:**
- [ ] Init creates user config at XDG path
- [ ] Init with `--project` creates .autospec/config.yml
- [ ] Existing config not overwritten without `--force`
- [ ] Config template written with correct defaults
- [ ] Directory created if missing

---

### 7. `internal/cli/status_test.go`

The `status.go` file (119 lines) reports phase progress with **no tests**.

**Functions to test:**
- Phase progress calculation
- Task status aggregation
- Output formatting

**Test cases:**
- [ ] Detect completed phases correctly
- [ ] Calculate task completion percentage
- [ ] Format output with phase checkmarks
- [ ] Handle missing spec directory gracefully
- [ ] Handle missing artifacts gracefully

---

### 8. `internal/validation/prompt_test.go`

The `prompt.go` file (57 lines) generates continuation prompts.

**Functions to test:**
- `ListIncompletePhasesWithTasks()` - Phase filtering
- `GenerateContinuationPrompt()` - Prompt formatting

**Test cases:**
- [ ] Filter out completed phases
- [ ] Truncate tasks with "and X more" when > 3
- [ ] Empty phases excluded from output
- [ ] All completed returns empty prompt
- [ ] Markdown formatting correct

---

## Medium Priority

### 9. `internal/cli/analyze_test.go`

The `analyze.go` file (110 lines) checks cross-artifact consistency.

**Test cases:**
- [ ] Detect missing required artifacts
- [ ] Validate artifact dependencies
- [ ] Report consistency errors

---

### 10. `internal/cli/workflow_test.go` (create new)

The `workflow.go` file (73 lines) runs the planning workflow (no implement).

**Test cases:**
- [ ] Phase config excludes implement phase
- [ ] Feature description passed correctly
- [ ] Skip confirmations flag honored

---

### 11. Phase Command Tests (specify, plan, tasks)

These files have similar patterns and should have tests for:

**Test cases per command:**
- [ ] Prompt text appended to slash command
- [ ] Spec detection used when not provided
- [ ] Configuration overrides applied
- [ ] Max retries override respected

---

## Summary

| Priority | File | Lines | Complexity |
|----------|------|-------|------------|
| Critical | `executor.go` | 270 | High |
| Critical | `update_task.go` | 191 | High |
| Critical | `tasks_yaml.go` | 200+ | High |
| Critical | `all.go` | 135 | Medium |
| High | `config.go` | 200+ | Medium |
| High | `init.go` | 400+ | High |
| High | `status.go` | 119 | Medium |
| High | `prompt.go` | 57 | Low |
| Medium | `analyze.go` | 110 | Medium |
| Medium | `workflow.go` | 73 | Low |
| Medium | phase commands | ~85 ea | Low |

**Estimated total test files to add:** 11 new test files
**Estimated total test cases:** 60-80
