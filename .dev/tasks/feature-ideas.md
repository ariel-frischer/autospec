# Feature Ideas for autospec CLI

Based on analysis of the codebase, common Go CLI patterns, and comparison with mature CLIs like kubectl, docker, and gh.

---

## Quick Start Commands

Copy-paste any of these to start specifying a feature:

### High Priority

```bash
# 0. Enhance Status Command (HIGH PRIORITY) - DONE!
autospec specify "Fix and enhance 'autospec status' command. BUGS to fix: (1) Line 55 hardcodes 'Status: In Progress' - never checks actual completion, (2) Line 58 looks for tasks.md but should be tasks.yaml, (3) Errors and exits if tasks.yaml missing instead of showing partial status. NEW BEHAVIOR: Show artifact table (spec.yaml, plan.yaml, tasks.yaml) with EXISTS (✓/✗) and LAST_MODIFIED columns. If tasks.yaml exists: show concise task stats like 'Tasks: 12 total | 8 completed | 3 in-progress | 1 pending' and calculate overall status (Pending/In Progress/Complete) from task counts. If tasks.yaml missing: show 'Recommendations:' section with specific autospec commands like 'Run: autospec tasks' or 'Run: autospec run -pti' based on which artifacts exist. Add --json flag for scripting. Add 'st' alias."

# 1. Self-Update Command
autospec specify "Add 'autospec update' command that checks GitHub releases for newer versions and self-updates the binary. Support flags: --check (check only, no update), --version v1.2.3 (specific version). Use go-github-selfupdate library. Show changelog summary after update. Handle permission errors gracefully with sudo hint."

# 2. Output Format Flag
autospec specify "Add global --output/-o flag supporting json, yaml, table, plain formats. Create shared output formatter in internal/output/. Retrofit status, doctor, config show, and list commands to use formatter. Default to table for interactive, json when piped."

# 3. Command Aliases - DONE!
autospec specify "Add short aliases to all major commands: specify→spec/s, plan→p, tasks→t, implement→impl/i, status→st, doctor→doc, constitution→const, clarify→cl, checklist→chk, analyze→az. Use Cobra's Aliases field. Update help text to show aliases."

# 4. Shell Completion Installer - DONE!
autospec specify "Add 'autospec completion install [bash|zsh|fish|powershell]' subcommand that auto-detects shell from \$SHELL, writes completion script to appropriate location (~/.bashrc, ~/.zshrc, fish config), creates backup before modifying rc files, and provides manual instructions as fallback."

# 5. History/Audit Log - DONE
autospec specify "Add 'autospec history' command that logs all command executions to ~/.autospec/history.yaml with timestamp, command, spec, exit code, duration. Support flags: --spec NAME (filter), --clear (clear history), --limit N (show last N). Limit storage to configurable max entries."

# 6. Diff/Preview Mode
autospec specify "Add --diff flag to plan/tasks commands showing changes from previous version. Add --preview flag to implement for dry-run showing expected changes. Store artifact snapshots before/after in .autospec/snapshots/. Use go-diff library for color-coded output."

# 7. Artifact Validation Command (HIGH PRIORITY) DONE
autospec specify "Add 'autospec artifact [type] [path]' command that validates YAML artifacts against their schemas. Types: spec, plan, tasks. Validates: (1) valid YAML syntax, (2) required fields present for artifact type, (3) field types correct (strings, lists, enums), (4) cross-references valid (e.g. task dependencies exist). Output: success message with artifact summary OR detailed error with line numbers and hints. Add --schema flag to print expected schema for artifact type. Add --fix flag to auto-fix common issues (missing optional fields, formatting). Then update all .autospec/commands/autospec*.md slash command templates to call 'autospec artifact TYPE PATH' at the end of each phase to validate output before completing."

# 27. Phase-Based Task Execution (HIGH PRIORITY) - DONE!
autospec specify "Add --phases flag to implement command that runs each task phase in a separate Claude session. PROBLEM: 28 tasks across 8 phases in single session causes context pollution, attention degradation, high token usage, no recovery points. KEY INSIGHT: tasks.yaml already has phases array - just need orchestration change, no schema work. DEFAULT BEHAVIOR: 'autospec implement' (no flags) = single session, backward compatible. NEW FLAGS: --phases (run all phases sequentially, each in fresh Claude session), --phase 2 (run only phase 2), --from-phase 3 (run phases 3+ sequentially, each in fresh session). EXECUTION WITH --phases: for each phase in tasks.yaml, start fresh Claude session, run /autospec.implement with phase number filter, validate all phase tasks completed, end session (context cleared), repeat for next phase. IMPLEMENTATION: modify /autospec.implement slash command to accept phase number, update ExecuteImplement() to loop through phases with fresh Claude sessions when --phases set, track phase completion in retry state, add phase-level validation between transitions, parse existing phases array from tasks.yaml. BENEFITS: fresh context per phase, better accuracy on smaller batches, lower token usage, natural recovery points, clearer progress display (Phase 2/8: Git Extensions)."

# 28. Task-Level Execution (HIGH PRIORITY) - DONE
autospec specify "Add --tasks flag to implement command that runs each individual task in a separate Claude session. PROBLEM: Even phase-level batching (--phases) may not be granular enough - phases with 5+ tasks still accumulate context, and some complex tasks benefit from complete isolation. KEY INSIGHT: tasks.yaml has individual task entries with IDs - just need orchestration to run one task per session. RELATION TO --phases: --tasks is the most granular execution mode (1 task = 1 session), --phases is mid-level (1 phase = 1 session), default is coarse (all tasks = 1 session). DEFAULT BEHAVIOR: 'autospec implement' (no flags) = single session, backward compatible. NEW FLAG: --tasks (run all tasks sequentially, each in fresh Claude session). EXECUTION WITH --tasks: for each task in tasks.yaml (respecting dependency order), start fresh Claude session, run /autospec.implement with single task ID filter, validate task completed, end session (context cleared), repeat for next task. IMPLEMENTATION: modify /autospec.implement slash command to accept single task ID filter, update ExecuteImplement() to loop through tasks with fresh Claude sessions when --tasks set, respect task dependencies (skip if dependency not completed), track individual task completion in retry state, add task-level validation between transitions. BENEFITS: maximum context isolation per task, ideal for complex/long-running tasks, finest-grained recovery points, easiest to debug failures, can combine with --from-task to resume from specific task. FUTURE: --tasks could support parallel execution for independent tasks (no shared dependencies), but sequential is the MVP."
```

### Medium Priority

```bash
# 8. Config Profiles
autospec specify "Add config profile system: 'autospec config profiles' (list), 'autospec config use NAME' (switch), 'autospec config create NAME' (create from current). Store in ~/.config/autospec/profiles/. Add global --profile flag to use specific profile for single command."

# 9. Spec Templates
autospec specify "Add template system: 'autospec template list', 'autospec template use NAME', 'autospec template save NAME' (save current spec as template), 'autospec template import FILE'. Store in ~/.config/autospec/templates/. Include default templates for api-endpoint, bug-fix, refactor."

# 10. Man Page Generation
autospec specify "Add 'autospec docs man' command using Cobra's built-in doc.GenManTree() to generate man pages to ./man/ directory. Add 'autospec docs install' to copy to system man path. Add 'make docs' target to Makefile."

# 11. Watch Mode
autospec specify "Add 'autospec watch PHASE' command that monitors relevant files and re-runs phase on changes. Use fsnotify library. watch plan monitors spec.yaml, watch tasks monitors plan.yaml, watch implement monitors tasks.yaml. Include debounce for rapid changes and --interval flag."

# 12. Spec List Command
autospec specify "Add 'autospec list' command showing all specs in table format with columns: NUM, NAME, STATUS (planned/in-progress/complete), TASKS (done/total), LAST_MODIFIED. Support flags: --status (include status), --sort=date|name|status, --filter=PATTERN, --json."

# 13. Enhanced Progress Bars
autospec specify "Enhance progress display during implement phase using progressbar library. Show percentage, current task ID and description, elapsed time. Parse tasks.yaml completion status for real progress. Fall back to spinner when task count unknown."

# 14. Interactive Mode
autospec specify "Add 'autospec interactive' or 'autospec -i' for guided wizard mode using charmbracelet/huh library. Prompt for: action (specify/plan/tasks/implement), feature description, optional phases (clarify/checklist/analyze), confirmation before execution."
```

### Lower Priority

```bash
# 15. Plugin System
autospec specify "Add plugin system: 'autospec plugin install NAME', 'autospec plugin list', 'autospec plugin remove NAME'. Plugins are Go binaries in ~/.config/autospec/plugins/ following naming convention autospec-PLUGINNAME. Auto-discover and register as subcommands."

# 16. Context/Workspace Switching
autospec specify "Add workspace context: 'autospec context set NAME PATH', 'autospec context use NAME', 'autospec context list', 'autospec context current'. Store in ~/.config/autospec/contexts.yaml. Context switches working directory and loads project config."

# 17. Export/Import Specs
autospec specify "Add 'autospec export SPEC-NAME --output FILE.tar.gz' to bundle spec directory with all artifacts. Add 'autospec import FILE.tar.gz' to extract into specs/. Support --all flag to export all specs. Preserve directory structure and metadata."

# 18. Webhook/Event System
autospec specify "Add hook system: 'autospec hook add EVENT COMMAND', 'autospec hook list', 'autospec hook remove EVENT'. Events: phase-start, phase-complete, workflow-complete, error. Store in config. Execute hooks with environment variables for spec, phase, status."

# 19. Retry Resume Enhancement
autospec specify "Enhance resume functionality: 'autospec resume' continues last failed workflow from exact failure point, 'autospec resume SPEC-NAME' for specific spec. Store workflow state in retry.json including completed phases. Show what will be resumed before executing."

# 20. Timing/Performance Stats
autospec specify "Add 'autospec stats' command showing phase execution timing. Track in ~/.autospec/stats.json: phase, spec, duration, timestamp, success/fail. Display table with columns: Phase, Runs, Avg Time, Last Run. Support --spec NAME filter."

# 21. Spec Archive
autospec specify "Add 'autospec archive SPEC-NAME' to move completed spec to specs/.archive/ preserving structure. Add 'autospec archive --list' to show archived specs. Add 'autospec unarchive SPEC-NAME' to restore. Update list command to exclude archived by default."
```

### Config Commands

```bash
# Config Get/Set
autospec specify "Add 'autospec config get KEY' and 'autospec config set KEY VALUE' commands for individual config values. Support dot notation for nested keys. Validate values against schema. Add 'autospec config unset KEY' to remove override and fall back to default."

# Config Edit
autospec specify "Add 'autospec config edit' command that opens config file in \$EDITOR (default: vim). Support --project flag for .autospec/config.yml, --user for ~/.config/autospec/config.yml. Create file from defaults if doesn't exist. Validate after save."

# Config Diff
autospec specify "Add 'autospec config diff' command showing differences between project config, user config, and defaults. Use color-coded diff output. Show which values come from which source."
```

### Global Flags

```bash
# Quiet Flag
autospec specify "Add global --quiet/-q flag that suppresses non-essential output (banners, progress, hints). Only show errors and final results. Useful for scripting. Implement via shared output package that checks quiet flag."

# Log Level Flag
autospec specify "Add global --log-level flag accepting debug, info, warn, error. Default to info. Debug shows internal state, timing, config resolution. Integrate with existing --debug flag (alias for --log-level=debug)."
```

---

## High Priority (High Value, Reasonable Effort)

### 0. Enhance Status Command (HIGH PRIORITY)
**Why:** Current `status` command is broken and outdated. It should work at any workflow stage and guide users on next steps.

**Bugs in `internal/cli/status.go`:**
```go
// Line 55 - HARDCODED! Never checks actual completion
fmt.Printf("Status: In Progress\n\n")

// Line 58 - WRONG EXTENSION! Should be tasks.yaml
tasksPath := fmt.Sprintf("%s/tasks.md", metadata.Directory)
```

**Current problems:**
- **BUG:** Status always says "In Progress" even when complete
- **BUG:** Looks for `tasks.md` instead of `tasks.yaml`
- **BUG:** Errors and exits if tasks file missing (should show partial status)
- No recommendations on what to run next
- No artifact existence check
- No --json output flag

```bash
autospec status               # Current branch's spec
autospec st 003-auth          # Specific spec (alias)
autospec status --json        # For scripting
```

**New output (no spec yet):**
```
Spec: 014-new-feature
Branch: 014-new-feature

ARTIFACT      EXISTS    LAST MODIFIED
spec.yaml     ✗         -
plan.yaml     ✗         -
tasks.yaml    ✗         -

Recommendations:
  → Run: autospec specify "your feature description"
  → Or full workflow: autospec run -spti "your feature"
```

**New output (spec only):**
```
Spec: 014-new-feature

ARTIFACT      EXISTS    LAST MODIFIED
spec.yaml     ✓         2 hours ago
plan.yaml     ✗         -
tasks.yaml    ✗         -

Recommendations:
  → Run: autospec plan
  → Or continue workflow: autospec run -pti
```

**New output (tasks exist, in progress):**
```
Spec: 003-user-authentication
Status: In Progress

ARTIFACT      EXISTS    LAST MODIFIED
spec.yaml     ✓         2 hours ago
plan.yaml     ✓         1 hour ago
tasks.yaml    ✓         30 min ago

Tasks: 15 total | 8 completed | 2 in-progress | 5 pending

Recommendations:
  → Continue: autospec implement
```

**New output (complete):**
```
Spec: 013-uninstall-command
Status: Complete ✓

ARTIFACT      EXISTS    LAST MODIFIED
spec.yaml     ✓         Dec 15
plan.yaml     ✓         Dec 15
tasks.yaml    ✓         Dec 15

Tasks: 12 total | 12 completed | 0 in-progress | 0 pending
```

**Implementation:**
- Check file existence with `os.Stat()` before parsing
- Update from `tasks.md` to `tasks.yaml`
- Parse tasks.yaml to count by status (Pending/InProgress/Completed/Blocked)
- Calculate overall status: Complete if all done, In Progress if any in-progress or some completed, Pending if none started
- Decision tree for recommendations based on artifact presence
- Add alias via Cobra's `Aliases: []string{"st"}`
- Add `--json` flag

**Effort:** Low-Medium (1-2 days)

---

### 1. Self-Update Command
**Why:** Users shouldn't need to manually download new releases.

```bash
autospec update              # Check and update to latest
autospec update --check      # Check for updates only
autospec update v1.2.3       # Update to specific version
```

**Implementation:**
- Use `github.com/rhysd/go-github-selfupdate` or similar
- Check GitHub releases for new versions
- Download and replace binary (with permission handling)
- Show changelog summary

**Effort:** Medium (2-3 days)

---

### 2. Output Format Flag (--output/-o)
**Why:** Enable scripting and integration with other tools.

```bash
autospec status --output json
autospec status -o yaml
autospec config show -o table   # Default
autospec doctor -o json
```

**Implementation:**
- Add global `--output` flag (json, yaml, table, plain)
- Create shared output formatter in `internal/output/`
- Retrofit existing commands to use formatter

**Effort:** Medium (2-3 days)

---

### 3. Command Aliases
**Why:** Power users want shorter commands.

```go
// In each command definition
var planCmd = &cobra.Command{
    Use:     "plan",
    Aliases: []string{"p"},
    // ...
}
```

**Suggested aliases:**
| Command | Alias |
|---------|-------|
| specify | spec, s |
| plan | p |
| tasks | t |
| implement | impl, i |
| status | st |
| doctor | doc |

**Effort:** Low (1 day)

---

### 4. Shell Completion Installation Helper
**Why:** Completion is generated but users don't know how to install it.

```bash
autospec completion install bash   # Add to ~/.bashrc
autospec completion install zsh    # Add to ~/.zshrc
autospec completion install fish   # Add to fish config
```

**Implementation:**
- Detect shell from $SHELL
- Write completion script to appropriate location
- Add source line to rc file (with backup)
- Provide manual instructions as fallback

**Effort:** Low (1-2 days)

---

### 5. History/Audit Log
**Why:** Track what commands were run and when for debugging.

```bash
autospec history                   # Show recent commands
autospec history --spec 003-auth   # Filter by spec
autospec history --clear           # Clear history
```

**Storage:** `~/.autospec/history.json`

**Implementation:**
- Log command, timestamp, spec, exit code
- Limit to last N entries (configurable)
- Optional: include command duration

**Effort:** Medium (2 days)

---

### 6. Diff/Preview Mode
**Why:** See what would change before committing to implementation.

```bash
autospec plan --diff              # Show diff from last plan
autospec implement --preview      # Dry-run showing expected changes
```

**Implementation:**
- Store artifact snapshots before/after
- Use `github.com/sergi/go-diff` or similar
- Color-coded diff output

**Effort:** Medium-High (3-4 days)

---

### 7. Artifact Validation Command (HIGH PRIORITY)
**Why:** Claude-generated YAML artifacts may have missing fields, invalid syntax, or schema violations. Validating artifacts immediately after generation catches issues early and provides actionable feedback. Integrating validation into slash commands creates a closed-loop quality system.

```bash
autospec artifact spec specs/003-auth/spec.yaml      # Validate spec artifact
autospec artifact plan specs/003-auth/plan.yaml      # Validate plan artifact
autospec artifact tasks specs/003-auth/tasks.yaml    # Validate tasks artifact
autospec artifact spec --schema                       # Print expected spec schema
autospec artifact plan --fix specs/003-auth/plan.yaml # Auto-fix common issues
```

**Output (success):**
```
✓ Valid spec.yaml
  Feature: User Authentication
  Stories: 5 user stories defined
  Requirements: 12 requirements across 3 categories
```

**Output (failure):**
```
✗ Invalid spec.yaml (3 errors)

  Line 12: Missing required field 'acceptance_criteria' in story US-001
  Line 28: Invalid status 'inprogress' - must be one of: Draft, Review, Approved
  Line 45: Requirement R-003 references non-existent story 'US-999'

Hints:
  → Add acceptance_criteria list to each user story
  → Valid status values: Draft, Review, Approved
  → Run with --fix to auto-correct formatting issues
```

**Schema definitions (per artifact type):**

| Artifact | Required Fields | Validations |
|----------|----------------|-------------|
| spec.yaml | feature, description, user_stories, requirements | Stories have acceptance_criteria, requirements have priority |
| plan.yaml | overview, phases, components | Phases have deliverables, components reference valid files |
| tasks.yaml | tasks[] | Each task has id, description, status (enum), dependencies exist |

**Integration with slash commands:**
Update each `.autospec/commands/autospec*.md` template to include validation:
```markdown
<!-- At end of autospec.plan.md -->
After generating plan.yaml, validate it:
\`\`\`bash
autospec artifact plan specs/{{SPEC_NAME}}/plan.yaml
\`\`\`
If validation fails, fix the issues before completing.
```

**Implementation:**
- Create `internal/artifact/` package with schema definitions
- Use `gopkg.in/yaml.v3` for parsing with line numbers
- Validate against JSON Schema or custom Go validators
- Return structured errors with line numbers and hints
- Add to existing validation infrastructure in `internal/validation/`

**Effort:** Medium (2-3 days)

---

## Medium Priority (Good Value, Moderate Effort)

### 8. Config Profiles
**Why:** Different settings for different contexts (work vs personal, fast vs thorough).

```bash
autospec config use fast           # Switch to "fast" profile
autospec config profiles           # List available profiles
autospec config create work        # Create new profile
autospec --profile thorough run -a "feature"
```

**Storage:** `~/.config/autospec/profiles/`

**Effort:** Medium (2-3 days)

---

### 9. Spec Templates
**Why:** Reuse common spec patterns.

```bash
autospec template list                    # List available templates
autospec template use api-endpoint        # Create spec from template
autospec template save my-template        # Save current spec as template
autospec template import ./template.yaml  # Import from file
```

**Storage:** `~/.config/autospec/templates/`

**Effort:** Medium (2-3 days)

---

### 10. Man Page Generation
**Why:** Unix convention for documentation.

```bash
autospec docs man           # Generate man pages to ./man/
autospec docs install       # Install man pages to system
```

**Implementation:**
- Cobra has built-in `doc.GenManTree()`
- Add to Makefile: `make docs`

**Effort:** Low (1 day)

---

### 11. Watch Mode
**Why:** Automatically re-run on file changes during development.

```bash
autospec watch plan         # Re-run plan when spec.yaml changes
autospec watch tasks        # Re-run tasks when plan.yaml changes
```

**Implementation:**
- Use `github.com/fsnotify/fsnotify`
- Watch relevant files based on phase
- Debounce rapid changes

**Effort:** Medium (2 days)

---

### 12. Spec List Command
**Why:** Quick overview of all specs without leaving terminal.

```bash
autospec list                      # List all specs
autospec list --status             # Include completion status
autospec list --sort=date          # Sort options
autospec list --filter="auth"      # Filter by name
```

**Output:**
```
NUM  NAME                  STATUS      TASKS  LAST MODIFIED
001  initial-setup         complete    5/5    2024-01-15
002  go-binary-migration   in-progress 8/12   2024-01-20
003  auth-feature          planned     0/15   2024-01-22
```

**Effort:** Low-Medium (1-2 days)

---

### 13. Progress Bars (Enhanced)
**Why:** Better UX for long-running operations.

```bash
autospec implement
# Output:
# [████████░░░░░░░░] 47% Implementing... (T008: Add validation)
```

**Implementation:**
- Use `github.com/schollz/progressbar/v3` or enhance existing spinner
- Parse task completion to show real progress
- Show current task being worked on

**Effort:** Medium (2 days)

---

### 14. Interactive Mode
**Why:** Guided workflow for new users.

```bash
autospec interactive        # or `autospec -i`
# Prompts:
# ? What do you want to do? [specify/plan/tasks/implement]
# ? Enter feature description:
# ? Include clarify phase? [Y/n]
```

**Implementation:**
- Use `github.com/AlecAivazis/survey/v2` or `github.com/charmbracelet/huh`
- Wizard-style flow

**Effort:** Medium (2-3 days)

---

## Lower Priority (Nice to Have)

### 15. Plugin System
**Why:** Extensibility for custom workflows.

```bash
autospec plugin install my-plugin
autospec plugin list
autospec my-custom-command    # From plugin
```

**Consideration:** High complexity, evaluate demand first.

**Effort:** High (1-2 weeks)

---

### 16. Context/Workspace Switching
**Why:** Work on multiple projects easily.

```bash
autospec context set project-a /path/to/project-a
autospec context use project-a
autospec context list
```

**Effort:** Medium (2-3 days)

---

### 17. Export/Import Specs
**Why:** Share specs between projects or team members.

```bash
autospec export 003-auth --output auth-spec.tar.gz
autospec import auth-spec.tar.gz
autospec export --all --output backup.tar.gz
```

**Effort:** Medium (2 days)

---

### 18. Webhook/Event System
**Why:** CI/CD integration and notifications.

```bash
autospec hook add on-complete "curl https://webhook.example.com"
autospec hook list
autospec hook remove on-complete
```

**Events:** phase-start, phase-complete, workflow-complete, error

**Effort:** Medium-High (3-4 days)

---

### 19. Retry Resume
**Why:** Continue from exact failure point after fixing issues.

```bash
autospec resume              # Resume last failed workflow
autospec resume 003-auth     # Resume specific spec
```

**Note:** Partial support exists via retry state; this would enhance it.

**Effort:** Medium (2 days)

---

### 20. Timing/Performance Stats
**Why:** Track how long phases take for optimization.

```bash
autospec stats                     # Show timing stats
autospec stats 003-auth            # For specific spec
# Output:
# Phase       Runs  Avg Time  Last Run
# specify     3     45s       2024-01-20
# plan        2     120s      2024-01-20
# tasks       1     90s       2024-01-21
```

**Effort:** Low-Medium (1-2 days)

---

### 21. Spec Archive/Cleanup
**Why:** Manage completed specs without deleting them.

```bash
autospec archive 001-initial-setup  # Move to specs/.archive/
autospec archive --list             # List archived specs
autospec unarchive 001-initial      # Restore from archive
```

**Effort:** Low (1 day)

---

## Config Commands to Add

### Config Get/Set
```bash
autospec config get timeout
autospec config set timeout 3600
autospec config set max_retries 5
autospec config unset custom_claude_cmd
```

### Config Edit
```bash
autospec config edit           # Open in $EDITOR
autospec config edit --project # Edit project config
```

### Config Diff
```bash
autospec config diff           # Show diff between project and user config
```

---

## Global Flags to Consider

| Flag | Description |
|------|-------------|
| `--quiet/-q` | Suppress non-essential output |
| `--no-color` | Disable colored output |
| `--profile` | Use specific config profile |
| `--log-level` | Set log level (debug, info, warn, error) |

---

## Implementation Notes

### Recommended Libraries

| Feature | Library |
|---------|---------|
| Self-update | `github.com/rhysd/go-github-selfupdate` |
| Progress bars | `github.com/schollz/progressbar/v3` |
| Interactive prompts | `github.com/charmbracelet/huh` |
| File watching | `github.com/fsnotify/fsnotify` |
| Diff | `github.com/sergi/go-diff` |
| Man pages | `github.com/spf13/cobra/doc` |

### Priority Matrix

```
                    High Value
                        │
    [Self-Update]       │    [Output Format]
    [Aliases]           │    [Shell Install]
                        │
Low Effort ─────────────┼───────────────── High Effort
                        │
    [Man Pages]         │    [Plugin System]
    [Spec List]         │    [Interactive Mode]
                        │
                    Low Value
```

### Quick Wins (< 1 day each)

1. Command aliases
2. Man page generation
3. `--quiet` flag
4. `config get/set` commands
5. Spec archive command

---

## Implementation Phase Robustness

These improvements address issues discovered when Claude stops mid-implementation.

### 22. Smart Retry with Remaining Task List (HIGH PRIORITY)

**Why:** When Claude stops mid-implementation (context limits, attention issues), the retry lacks targeted guidance.

**Problem:** Current retry just re-runs `/autospec.implement` without telling Claude what's already done.

**Proposed Solution:**
```go
// After Claude "finishes" but validation fails:
if !stats.IsComplete() {
    remainingIDs := getRemainingTaskIDs(stats)
    retryPrompt := fmt.Sprintf("Continue implementation. Remaining tasks: %s",
        strings.Join(remainingIDs, ", "))
    // Trigger retry with this specific prompt
}
```

**Benefits:**
- Claude gets explicit list of remaining tasks
- More focused continuation rather than starting from scratch
- Could include task titles for better context

**Effort:** Medium (2 days)

---

### 23. Blocked Task Detection

**Why:** Retrying blocked tasks is wasteful - they need human intervention.

**Proposed Solution:**
```go
if stats.BlockedTasks > 0 {
    blockedIDs := getBlockedTaskIDs(stats)
    return &RetryDecision{
        ShouldRetry: false,
        Reason:      "blocked tasks require human intervention",
        BlockedTasks: blockedIDs,
        Suggestion:  "Review blocked tasks and update their status before retrying",
    }
}
```

**Decision Matrix:**

| Scenario | Action |
|----------|--------|
| Tasks "Pending" or "InProgress" | Auto-retry with continuation prompt |
| Tasks "Blocked" | Stop and ask human for input |
| Claude errors/timeout | Retry up to max_retries |
| All tasks "Completed" | Success |

**Effort:** Low-Medium (1-2 days)

---

### 24. Continuation Prompt Generation

**Why:** When retrying, Claude may not understand what was already done.

**Proposed Solution:**
```go
func GenerateContinuationPrompt(stats *TaskStats) string {
    var prompt strings.Builder
    prompt.WriteString("Resume implementation from where you left off.\n\n")

    prompt.WriteString("Completed tasks (DO NOT redo these):\n")
    for _, id := range getCompletedTaskIDs(stats) {
        prompt.WriteString(fmt.Sprintf("- %s\n", id))
    }

    prompt.WriteString("\nRemaining tasks:\n")
    for _, id := range getRemainingTaskIDs(stats) {
        prompt.WriteString(fmt.Sprintf("- %s\n", id))
    }

    if stats.InProgressTasks > 0 {
        prompt.WriteString("\nNote: Task currently in-progress - verify completion before moving on.\n")
    }

    return prompt.String()
}
```

**Effort:** Low (1 day)

---

### 25. Phase-Level Checkpoints

**Why:** If autospec crashes or user kills the process, there's no way to resume exactly where Claude left off.

**Proposed Solution:**
- Save checkpoint after each completed phase
- Allow `autospec implement --from-phase=5` to resume from specific phase
- Store last-executed task ID in retry state

**Effort:** Medium (2-3 days)

---

### 26. Max Context Warning

**Why:** Large implementations can exceed Claude's context limits.

**Proposed Solution:**
- Track approximate token usage during implementation
- Warn user when approaching context limits
- Suggest breaking into smaller batches

**Effort:** Medium (2 days)

---

### 27. Phase-Based Task Execution (HIGH PRIORITY)

**Why:** Currently Claude executes all tasks in tasks.yaml in a single long conversation. This causes:
- Context pollution from earlier task work affecting later tasks
- Reduced accuracy as conversation grows (attention degradation)
- Higher token usage from maintaining large conversation history
- No natural breakpoints for recovery if something fails mid-implementation

**Problem:** A tasks.yaml with 28 tasks across 8 phases forces Claude into one massive session where early context (file reads, decisions, errors) pollutes the working memory for later unrelated tasks.

**Key insight:** tasks.yaml already has phases defined - we just need to change the orchestration to run each phase in a separate Claude session.

**Current behavior (remains default):**
```
autospec implement
  → Single Claude session
  → Execute all 28 tasks
  → Context grows continuously
  → Early work pollutes later task context
```

**New behavior with --phases flag:**
```
autospec implement --phases
  ↓
  Phase 1: Setup (3 tasks)
    → New Claude session
    → Execute T001, T002, T003
    → Validate phase completion
    → Session ends (context cleared)
  ↓
  Phase 2: Git Package Extensions (4 tasks)
    → New Claude session (fresh context!)
    → Execute T004, T005, T006, T007
    → Validate phase completion
    → Session ends
  ↓
  ... continues for each phase
```

**Benefits of --phases:**
1. **Fresh context per phase** - No pollution from unrelated earlier work
2. **Better accuracy** - Claude focuses on smaller task batches
3. **Lower token usage** - Shorter conversations = fewer tokens
4. **Natural recovery points** - If phase fails, retry just that phase
5. **Clearer progress** - Can show "Phase 2/8: Git Package Extensions"

**CLI flags:**
```bash
autospec implement                    # Single session (default, backward compatible)
autospec implement --phases           # Run all phases sequentially, each in fresh session
autospec implement --phase 2          # Run only phase 2
autospec implement --from-phase 3     # Run phases 3+ sequentially, each in fresh session
```

**Implementation:**
- Modify `/autospec.implement` slash command to accept phase number filter
- Update `ExecuteImplement()` to loop through phases with fresh Claude sessions
- Track phase completion in retry state (not just task completion)
- Add phase-level validation between phase transitions
- Parse existing `phases` array from tasks.yaml to get phase boundaries

**Effort:** Medium (2-3 days)

---

## References

- [Cobra CLI Best Practices](https://cobra.dev/)
- [Go CLI Mastery](https://dev.to/tavernetech/go-cli-mastery-crafting-developer-tools-that-dont-suck-3p53)
- [12-Factor CLI Apps](https://medium.com/@jdxcode/12-factor-cli-apps-dd3c227a0e46)
