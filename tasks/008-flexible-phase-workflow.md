# Flexible Phase Workflow Command

**Feature**: Allow users to run custom combinations of SpecKit phases in any order with safety warnings.

## Current State Analysis

### Existing Commands
| Command | Phases | Use Case |
|---------|--------|----------|
| `autospec full "feature"` | specify → plan → tasks → implement | Complete workflow |
| `autospec workflow "feature"` | specify → plan → tasks | Planning only |
| `autospec specify "feature"` | specify | Single phase |
| `autospec plan` | plan | Single phase |
| `autospec tasks` | tasks | Single phase |
| `autospec implement` | implement | Single phase |

### Gap
Users cannot easily run custom combinations like:
- `specify → plan → implement` (skip tasks)
- `plan → implement` (resume from existing spec)
- `specify → plan` (minimal planning)

---

## Recommended Command Structure

### Option 1: Phase Flags (Recommended)

```bash
# New 'run' command with phase flags
autospec run -s -p -t -i "feature description"

# Short flags can be combined (like tar -xzvf)
autospec run -spi "feature description"    # specify → plan → implement
autospec run -sp "feature description"     # specify → plan
autospec run -pti                          # plan → tasks → implement (auto-detect spec)

# Long flags available for clarity
autospec run --specify --plan --tasks --implement "feature"
```

**Pros:**
- Intuitive for Unix users (similar to tar, chmod flags)
- Concise: `-spi` vs `--specify --plan --implement`
- Easy to remember: s=specify, p=plan, t=tasks, i=implement
- Flexible ordering in flags (execution always follows canonical order)

**Cons:**
- New command to learn (`run`)
- Slightly more typing than current `full` for common case

### Option 2: Comma-Separated Phases

```bash
autospec run specify,plan,implement "feature"
autospec run s,p,i "feature"
```

**Pros:**
- Explicit phase list
- Could support arbitrary ordering

**Cons:**
- More typing
- Parsing complexity
- Less familiar pattern

### Option 3: Root-Level Flags

```bash
autospec -spi "feature"
autospec --specify --plan --implement "feature"
```

**Pros:**
- Shortest possible command
- No subcommand needed

**Cons:**
- Conflicts with existing subcommand structure
- May confuse users expecting `autospec specify`

---

## Recommendation: Option 1 (Phase Flags)

### Command Specification

```bash
autospec run [flags] [feature-description]

Flags:
  -s, --specify     Include specify phase (requires feature description)
  -p, --plan        Include plan phase
  -t, --tasks       Include tasks phase
  -i, --implement   Include implement phase
  -r, --resume      Resume implementation from last checkpoint
  --max-retries     Maximum retry attempts (default: 3)

Examples:
  autospec run -spti "Add authentication"   # Full workflow (same as 'full')
  autospec run -spt "Add feature"           # Same as 'workflow'
  autospec run -spi "Add feature"           # Skip tasks phase
  autospec run -sp "Add feature"            # Just specify and plan
  autospec run -pi                          # Plan and implement (existing spec)
  autospec run -ti                          # Tasks and implement (existing spec)
  autospec run -i                           # Just implement (same as 'implement')
```

### Execution Order

Phases always execute in canonical order regardless of flag order:
1. specify (if -s)
2. plan (if -p)
3. tasks (if -t)
4. implement (if -i)

This prevents user confusion and ensures correct artifact dependencies.

---

## Safety Warnings & Confirmations (Branch-Aware)

The warning system uses git branch detection and artifact checking to provide context-aware guidance.

### Case Matrix

| Case | Branch | Artifacts | Behavior |
|------|--------|-----------|----------|
| 1 | Spec branch (e.g., `007-feature`) | All exist | No warning, run immediately |
| 2 | Spec branch | Some missing | Warn with specifics, y/N |
| 3 | Non-spec branch (`main`, `develop`) | N/A | Error: no spec detected, suggest `-s` or checkout |
| 4 | Spec branch | None exist | Warn: fresh spec, confirm phases |
| 5 | Detached HEAD / no git | N/A | Fall back to most recent spec dir |

### Case 1: On Spec Branch, Artifacts Exist

```
$ git branch
* 007-yaml-structured-output

$ autospec run -pi

→ Detected spec: 007-yaml-structured-output
→ Found: spec.md ✓, plan.md ✓, tasks.md ✓

Phases to execute: plan → implement

→ Executing: plan phase...
```

No warning needed - artifacts exist, user knows what they're doing.

### Case 2: On Spec Branch, Some Artifacts Missing

```
$ git branch
* 007-yaml-structured-output

$ autospec run -ti

→ Detected spec: 007-yaml-structured-output

⚠️  Warning: Missing prerequisite artifacts:
    • plan.md not found (required for tasks phase)

    Hint: Consider running with -p to generate plan first,
          or create plan.md manually.

Phases to execute: tasks → implement

Continue? [y/N]: _
```

### Case 3: On Non-Spec Branch

```
$ git branch
* main

$ autospec run -pi

✗ Error: No spec detected from branch 'main'

  Options:
    1. Include -s flag to create a new spec:
       autospec run -spi "Your feature description"

    2. Checkout an existing spec branch:
       git checkout 007-yaml-structured-output

    3. Specify a spec explicitly:
       autospec run -pi --spec 007-yaml-structured-output
```

This is an error, not a warning - we can't proceed without knowing which spec.

### Case 4: On Spec Branch, No Artifacts (Fresh Start)

```
$ git branch
* 008-new-feature

$ autospec run -pti

→ Detected spec: 008-new-feature

⚠️  Warning: No artifacts found for this spec.
    • spec.md not found
    • plan.md not found
    • tasks.md not found

    This appears to be a fresh spec. Consider starting with -s:
    autospec run -spti "Your feature description"

Phases to execute: plan → tasks → implement

Continue anyway? [y/N]: _
```

### Case 5: Detached HEAD / No Git

```
$ autospec run -pi

→ No git branch detected, using most recent spec: 007-yaml-structured-output
→ Found: spec.md ✓, plan.md ✓

Phases to execute: plan → implement

→ Executing: plan phase...
```

Falls back gracefully to existing behavior.

### Artifact Dependency Map

```
specify  →  creates spec.md
plan     →  requires spec.md, creates plan.md
tasks    →  requires plan.md, creates tasks.md
implement → requires tasks.md
```

### Warning Logic (Pseudocode)

```go
func checkPrerequisites(phases PhaseConfig, specsDir string) *PreflightResult {
    result := &PreflightResult{}

    // Step 1: Detect spec from git branch
    spec, err := spec.DetectCurrentSpec(specsDir)
    if err != nil {
        // Case 3: No spec detected
        if phases.Specify {
            // OK - they're creating a new spec
            result.NeedsNewSpec = true
            return result
        }
        result.Error = fmt.Errorf("no spec detected from branch '%s'", gitBranch())
        result.Suggestions = []string{
            "Include -s flag to create a new spec",
            "Checkout an existing spec branch",
            "Specify a spec explicitly with --spec",
        }
        return result
    }

    result.SpecName = spec.Name
    result.SpecDir = spec.Directory

    // Step 2: Check which artifacts exist
    result.HasSpec = fileExists(spec.Directory, "spec.md")
    result.HasPlan = fileExists(spec.Directory, "plan.md")
    result.HasTasks = fileExists(spec.Directory, "tasks.md")

    // Step 3: Check for missing prerequisites based on requested phases
    if phases.Plan && !phases.Specify && !result.HasSpec {
        result.AddWarning("spec.md not found (required for plan phase)")
        result.AddHint("Consider running with -s to generate spec first")
    }

    if phases.Tasks && !phases.Plan && !result.HasPlan {
        result.AddWarning("plan.md not found (required for tasks phase)")
        result.AddHint("Consider running with -p to generate plan first")
    }

    if phases.Implement && !phases.Tasks && !result.HasTasks {
        result.AddWarning("tasks.md not found (required for implement phase)")
        result.AddHint("Consider running with -t to generate tasks first")
    }

    // Step 4: Determine if confirmation needed
    // Case 1: All good, no warnings
    // Case 2: Has warnings, needs confirmation
    // Case 4: No artifacts at all, needs confirmation
    result.NeedsConfirmation = len(result.Warnings) > 0

    return result
}
```

### Skip Confirmation

```bash
# Skip with -y flag
autospec run -pi -y

# Or set in config
{
  "skip_confirmations": true
}

# Or environment variable
AUTOSPEC_YES=1 autospec run -pi
```

---

## Implementation Architecture

### New Files

```
internal/cli/
├── run.go              # New 'run' command
├── run_test.go         # Tests for run command
└── phases.go           # Phase ordering and validation logic
```

### Phase Configuration

```go
// internal/cli/phases.go

type PhaseConfig struct {
    Specify   bool
    Plan      bool
    Tasks     bool
    Implement bool
}

// Returns ordered list of phases to execute
func (p *PhaseConfig) GetExecutionOrder() []workflow.Phase {
    var phases []workflow.Phase
    if p.Specify {
        phases = append(phases, workflow.PhaseSpecify)
    }
    if p.Plan {
        phases = append(phases, workflow.PhasePlan)
    }
    if p.Tasks {
        phases = append(phases, workflow.PhaseTasks)
    }
    if p.Implement {
        phases = append(phases, workflow.PhaseImplement)
    }
    return phases
}

// Checks for missing prerequisites and returns warnings
func (p *PhaseConfig) GetWarnings(specsDir, specName string) []string {
    var warnings []string

    // Plan without specify (and no existing spec)
    if p.Plan && !p.Specify {
        if !specFileExists(specsDir, specName) {
            warnings = append(warnings,
                "Running plan without specify. No spec.md exists.")
        }
    }

    // Tasks without plan (and no existing plan)
    if p.Tasks && !p.Plan {
        if !planFileExists(specsDir, specName) {
            warnings = append(warnings,
                "Running tasks without plan. No plan.md exists.")
        }
    }

    // Implement without tasks (and no existing tasks)
    if p.Implement && !p.Tasks {
        if !tasksFileExists(specsDir, specName) {
            warnings = append(warnings,
                "Running implement without tasks. No tasks.md exists.")
        }
    }

    return warnings
}
```

### Run Command

```go
// internal/cli/run.go

var runCmd = &cobra.Command{
    Use:   "run [feature-description]",
    Short: "Run custom phase combinations",
    Long: `Execute a custom combination of SpecKit phases.

Phases are always executed in canonical order:
  specify → plan → tasks → implement

Use short flags for concise commands:
  -s  Include specify phase
  -p  Include plan phase
  -t  Include tasks phase
  -i  Include implement phase

Examples:
  autospec run -spti "Add auth"    # Full workflow
  autospec run -spi "Add feature"  # Skip tasks
  autospec run -pi                 # Plan + implement (existing spec)`,
    RunE: runPhases,
}

func init() {
    rootCmd.AddCommand(runCmd)

    // Phase flags (combinable short flags)
    runCmd.Flags().BoolP("specify", "s", false, "Include specify phase")
    runCmd.Flags().BoolP("plan", "p", false, "Include plan phase")
    runCmd.Flags().BoolP("tasks", "t", false, "Include tasks phase")
    runCmd.Flags().BoolP("implement", "i", false, "Include implement phase")

    // Standard flags
    runCmd.Flags().BoolP("resume", "r", false, "Resume from checkpoint")
    runCmd.Flags().Int("max-retries", 0, "Maximum retry attempts")
    runCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompts")
}

func runPhases(cmd *cobra.Command, args []string) error {
    // Build phase config from flags
    config := PhaseConfig{
        Specify:   getBoolFlag(cmd, "specify"),
        Plan:      getBoolFlag(cmd, "plan"),
        Tasks:     getBoolFlag(cmd, "tasks"),
        Implement: getBoolFlag(cmd, "implement"),
    }

    // Validate at least one phase selected
    if !config.HasAnyPhase() {
        return fmt.Errorf("at least one phase flag required (-s, -p, -t, -i)")
    }

    // Get feature description (required if specify phase)
    var featureDesc string
    if config.Specify {
        if len(args) == 0 {
            return fmt.Errorf("feature description required with --specify")
        }
        featureDesc = strings.Join(args, " ")
    }

    // Check for warnings
    skipConfirm, _ := cmd.Flags().GetBool("yes")
    warnings := config.GetWarnings(specsDir, autoDetectedSpec)

    if len(warnings) > 0 && !skipConfirm {
        // Display warnings
        for _, w := range warnings {
            fmt.Printf("⚠️  Warning: %s\n", w)
        }

        // Show planned phases
        fmt.Printf("\nPhases to execute: %s\n\n",
            formatPhaseList(config.GetExecutionOrder()))

        // Prompt for confirmation
        if !confirmContinue() {
            return fmt.Errorf("aborted by user")
        }
    }

    // Execute phases
    return executePhases(config, featureDesc)
}
```

### Confirmation Prompt

```go
func confirmContinue() bool {
    reader := bufio.NewReader(os.Stdin)
    fmt.Print("Continue? [y/N]: ")

    input, _ := reader.ReadString('\n')
    input = strings.TrimSpace(strings.ToLower(input))

    return input == "y" || input == "yes"
}
```

---

## Backward Compatibility

### Existing Commands Remain

The existing commands continue to work unchanged:
- `autospec full` = `autospec run -spti`
- `autospec workflow` = `autospec run -spt`
- `autospec specify` = `autospec run -s`
- `autospec plan` = `autospec run -p`
- `autospec tasks` = `autospec run -t`
- `autospec implement` = `autospec run -i`

### Migration Path

1. Add `run` command alongside existing commands
2. Update documentation to highlight `run` for custom workflows
3. Keep existing commands as convenient aliases

---

## Alternative Shortcuts (Optional)

For very common patterns, consider additional aliases:

```bash
# Could add these if users request them
autospec quick "feature"     # -spi (skip tasks)
autospec validate "feature"  # -sp (specify + plan only, for review)
```

---

## Tasks

### Phase 1: Core Infrastructure

- [ ] T001 Create `internal/cli/phases.go` with PhaseConfig struct
- [ ] T002 Write tests for phase ordering logic in `phases_test.go`
- [ ] T003 Implement `GetExecutionOrder()` method
- [ ] T004 Implement `HasAnyPhase()` validation method

### Phase 2: Preflight Checks (Branch-Aware)

- [ ] T005 Create `internal/cli/preflight.go` with PreflightResult struct
- [ ] T006 Write tests for preflight logic in `preflight_test.go`
- [ ] T007 Implement `checkPrerequisites()` with git branch detection
- [ ] T008 Implement artifact existence checking (spec.md, plan.md, tasks.md)
- [ ] T009 Implement warning generation for missing prerequisites
- [ ] T010 Implement hint generation based on missing artifacts
- [ ] T011 Handle Case 1: spec branch with all artifacts (no warning)
- [ ] T012 Handle Case 2: spec branch with missing artifacts (warn + confirm)
- [ ] T013 Handle Case 3: non-spec branch without -s (error with suggestions)
- [ ] T014 Handle Case 4: spec branch with no artifacts (warn fresh start)
- [ ] T015 Handle Case 5: detached HEAD / no git (fallback to recent spec)

### Phase 3: Run Command

- [ ] T016 Create `internal/cli/run.go` with command definition
- [ ] T017 Write tests for flag parsing in `run_test.go`
- [ ] T018 Implement flag parsing for -s, -p, -t, -i (combinable short flags)
- [ ] T019 Add validation (at least one phase required)
- [ ] T020 Add feature description requirement when -s is used
- [ ] T021 Add `--spec` flag to explicitly specify spec name

### Phase 4: Confirmation Flow

- [ ] T022 Implement `confirmContinue()` function with stdin handling
- [ ] T023 Write tests for confirmation logic
- [ ] T024 Add `--yes` / `-y` flag to skip confirmation
- [ ] T025 Add `AUTOSPEC_YES` environment variable support
- [ ] T026 Add `skip_confirmations` config option
- [ ] T027 Display formatted warning messages with hints
- [ ] T028 Display phase execution plan before confirmation

### Phase 5: Execution Integration

- [ ] T029 Integrate preflight checks before execution
- [ ] T030 Integrate with WorkflowOrchestrator for phase execution
- [ ] T031 Implement sequential phase execution respecting order
- [ ] T032 Handle spec auto-detection for non-specify phases
- [ ] T033 Create spec directory when -s is used on new branch
- [ ] T034 Add progress display support (reuse existing progress system)

### Phase 6: Testing & Polish

- [ ] T035 Write integration tests for Case 1 (all artifacts exist)
- [ ] T036 Write integration tests for Case 2 (missing artifacts)
- [ ] T037 Write integration tests for Case 3 (non-spec branch)
- [ ] T038 Write integration tests for Case 4 (fresh spec branch)
- [ ] T039 Write integration tests for Case 5 (no git)
- [ ] T040 Test common flag combinations (-spi, -sp, -pi, -ti, -spti)
- [ ] T041 Update CLI help text and examples
- [ ] T042 Update CLAUDE.md with new command documentation

---

## Summary

**Recommended Approach**: `autospec run -spi "feature"`

**Key Benefits:**
1. Concise: Combined short flags (`-spi`) are quick to type
2. Flexible: Any phase combination supported
3. Branch-aware: Smart detection of current spec from git branch
4. Safe: Context-aware warnings when prerequisites missing
5. User-friendly: y/N confirmation with skip option (`-y`, `AUTOSPEC_YES`, config)
6. Helpful: Clear error messages with actionable suggestions
7. Backward compatible: Existing commands unchanged

**Total Tasks**: 42

| Phase | Tasks | Focus |
|-------|-------|-------|
| 1. Core Infrastructure | 4 | PhaseConfig struct, ordering |
| 2. Preflight Checks | 11 | Branch-aware detection, 5 cases |
| 3. Run Command | 6 | Flag parsing, validation |
| 4. Confirmation Flow | 7 | y/N prompts, skip options |
| 5. Execution Integration | 6 | Orchestrator integration |
| 6. Testing & Polish | 8 | Integration tests, docs |

**Complexity**: Medium-High (branch-aware detection adds significant logic)
