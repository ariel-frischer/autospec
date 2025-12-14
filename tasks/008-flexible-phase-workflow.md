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

### Root-Level Phase Flags

```bash
# Phase flags directly on root command (no subcommand needed)
autospec -s -p -t -i "feature description"

# Short flags can be combined (like tar -xzvf)
autospec -spi "feature description"    # specify → plan → implement
autospec -sp "feature description"     # specify → plan
autospec -pti                          # plan → tasks → implement (auto-detect spec)

# Long flags available for clarity
autospec --specify --plan --tasks --implement "feature"

# Shortcut for all phases
autospec -a "feature description"      # same as -spti
autospec --all "feature description"
```

**Pros:**
- Shortest possible command
- Intuitive for Unix users (similar to tar, chmod flags)
- Concise: `-spi` vs `--specify --plan --implement`
- Easy to remember: s=specify, p=plan, t=tasks, i=implement, a=all
- `-a` / `--all` for the common "run everything" case

### Command Specification

```bash
autospec [phase-flags] [feature-description]

Phase Flags:
  -s, --specify     Include specify phase (requires feature description)
  -p, --plan        Include plan phase
  -t, --tasks       Include tasks phase
  -i, --implement   Include implement phase
  -a, --all         Include all phases (equivalent to -spti)

Other Flags:
  -r, --resume      Resume implementation from last checkpoint
  -y, --yes         Skip confirmation prompts
  --spec            Explicitly specify spec name (e.g., --spec 007-feature)
  --max-retries     Maximum retry attempts (default: 3)

Examples:
  autospec -a "Add authentication"      # All phases (full workflow)
  autospec -spti "Add authentication"   # Same as above, explicit
  autospec -spt "Add feature"           # Specify, plan, tasks (no implement)
  autospec -spi "Add feature"           # Skip tasks phase
  autospec -sp "Add feature"            # Just specify and plan
  autospec -pi                          # Plan and implement (existing spec)
  autospec -ti                          # Tasks and implement (existing spec)
  autospec -i                           # Just implement
  autospec -i --spec 007-feature        # Implement specific spec
```

### Execution Order

Phases always execute in canonical order regardless of flag order:
1. specify (if -s or -a)
2. plan (if -p or -a)
3. tasks (if -t or -a)
4. implement (if -i or -a)

This prevents user confusion and ensures correct artifact dependencies.

### Naming Discussion: `full` vs `all` vs `-a`

| Option | Command | Pros | Cons |
|--------|---------|------|------|
| Current `full` | `autospec full "feature"` | Explicit subcommand | Longer, redundant with `-a` |
| Rename to `all` | `autospec all "feature"` | Clearer meaning | Still a subcommand |
| Flag only | `autospec -a "feature"` | Shortest, consistent | Less discoverable |
| Both | `autospec -a` + `autospec all` | Flexibility | Redundancy |

**Recommendation**: Keep `-a` flag as primary, deprecate `full` subcommand (or keep as alias for discoverability).

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
autospec -pi -y

# Or set in config
{
  "skip_confirmations": true
}

# Or environment variable
AUTOSPEC_YES=1 autospec -pi
```

---

## Implementation Architecture

### New/Modified Files

```
internal/cli/
├── root.go             # Add phase flags to root command
├── phases.go           # Phase ordering and validation logic
├── phases_test.go      # Tests for phase logic
├── preflight.go        # Branch-aware prerequisite checking
└── preflight_test.go   # Tests for preflight logic
```

### Phase Configuration

```go
// internal/cli/phases.go

type PhaseConfig struct {
    Specify   bool
    Plan      bool
    Tasks     bool
    Implement bool
    All       bool  // -a flag, expands to all phases
}

// Normalize expands -a flag to all phases
func (p *PhaseConfig) Normalize() {
    if p.All {
        p.Specify = true
        p.Plan = true
        p.Tasks = true
        p.Implement = true
    }
}

// HasAnyPhase returns true if at least one phase is selected
func (p *PhaseConfig) HasAnyPhase() bool {
    return p.Specify || p.Plan || p.Tasks || p.Implement || p.All
}

// GetExecutionOrder returns ordered list of phases to execute
func (p *PhaseConfig) GetExecutionOrder() []workflow.Phase {
    p.Normalize()
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
```

### Root Command Integration

```go
// internal/cli/root.go (additions)

func init() {
    // Existing global flags...

    // Phase flags (combinable short flags on root)
    rootCmd.Flags().BoolP("specify", "s", false, "Include specify phase")
    rootCmd.Flags().BoolP("plan", "p", false, "Include plan phase")
    rootCmd.Flags().BoolP("tasks", "t", false, "Include tasks phase")
    rootCmd.Flags().BoolP("implement", "i", false, "Include implement phase")
    rootCmd.Flags().BoolP("all", "a", false, "Include all phases (equivalent to -spti)")

    // Phase-related flags
    rootCmd.Flags().BoolP("resume", "r", false, "Resume from checkpoint")
    rootCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompts")
    rootCmd.Flags().String("spec", "", "Explicitly specify spec name")
}

var rootCmd = &cobra.Command{
    Use:   "autospec [flags] [feature-description]",
    Short: "SpecKit workflow automation",
    Long: `Autospec orchestrates SpecKit workflow phases.

Use phase flags to run custom combinations:
  -s, --specify     Include specify phase
  -p, --plan        Include plan phase
  -t, --tasks       Include tasks phase
  -i, --implement   Include implement phase
  -a, --all         All phases (equivalent to -spti)

Examples:
  autospec -a "Add authentication"    # Full workflow
  autospec -spi "Add feature"         # Skip tasks
  autospec -pi                        # Plan + implement (existing spec)
  autospec -i --spec 007-feature      # Implement specific spec`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Check if any phase flags are set
        phases := getPhaseConfig(cmd)

        if !phases.HasAnyPhase() {
            // No phase flags - show help or run default subcommand
            return cmd.Help()
        }

        // Phase flags present - run phase workflow
        return runPhaseWorkflow(cmd, args, phases)
    },
}

func runPhaseWorkflow(cmd *cobra.Command, args []string, phases PhaseConfig) error {
    phases.Normalize()

    // Get feature description (required if specify phase)
    var featureDesc string
    if phases.Specify {
        if len(args) == 0 {
            return fmt.Errorf("feature description required with -s/--specify")
        }
        featureDesc = strings.Join(args, " ")
    }

    // Run preflight checks (branch-aware)
    specName, _ := cmd.Flags().GetString("spec")
    preflight := checkPrerequisites(phases, specsDir, specName)

    if preflight.Error != nil {
        return preflight.Error
    }

    // Handle warnings with confirmation
    skipConfirm, _ := cmd.Flags().GetBool("yes")
    if !skipConfirm && preflight.NeedsConfirmation {
        if !displayWarningsAndConfirm(preflight) {
            return fmt.Errorf("aborted by user")
        }
    }

    // Execute phases
    return executePhases(phases, featureDesc, preflight.SpecName)
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

### Existing Subcommands

Existing subcommands continue to work unchanged:

| Subcommand | Equivalent Flags | Status |
|------------|------------------|--------|
| `autospec full "feature"` | `autospec -a "feature"` | Keep or deprecate |
| `autospec workflow "feature"` | `autospec -spt "feature"` | Keep or deprecate |
| `autospec specify "feature"` | `autospec -s "feature"` | Keep (single-phase convenience) |
| `autospec plan` | `autospec -p` | Keep (single-phase convenience) |
| `autospec tasks` | `autospec -t` | Keep (single-phase convenience) |
| `autospec implement` | `autospec -i` | Keep (single-phase convenience) |

### Deprecation Strategy

**Option A: Keep All (Recommended for now)**
- Subcommands remain for discoverability
- Flags are the "power user" interface
- No breaking changes

**Option B: Deprecate Multi-Phase Subcommands**
- Deprecate `full` and `workflow` (replaced by `-a` and `-spt`)
- Keep single-phase subcommands for simplicity
- Print deprecation warning when used

### Migration Path

1. Add phase flags to root command
2. Document flag-based approach as primary
3. Keep subcommands working (no breaking changes)
4. Consider deprecation warnings in future version

---

## Tasks

### Phase 1: Core Infrastructure

- [ ] T001 Create `internal/cli/phases.go` with PhaseConfig struct
- [ ] T002 Write tests for phase ordering logic in `phases_test.go`
- [ ] T003 Implement `Normalize()` method (expand -a to all phases)
- [ ] T004 Implement `GetExecutionOrder()` method
- [ ] T005 Implement `HasAnyPhase()` validation method

### Phase 2: Preflight Checks (Branch-Aware)

- [ ] T006 Create `internal/cli/preflight.go` with PreflightResult struct
- [ ] T007 Write tests for preflight logic in `preflight_test.go`
- [ ] T008 Implement `checkPrerequisites()` with git branch detection
- [ ] T009 Implement artifact existence checking (spec.md, plan.md, tasks.md)
- [ ] T010 Implement warning generation for missing prerequisites
- [ ] T011 Implement hint generation based on missing artifacts
- [ ] T012 Handle Case 1: spec branch with all artifacts (no warning)
- [ ] T013 Handle Case 2: spec branch with missing artifacts (warn + confirm)
- [ ] T014 Handle Case 3: non-spec branch without -s (error with suggestions)
- [ ] T015 Handle Case 4: spec branch with no artifacts (warn fresh start)
- [ ] T016 Handle Case 5: detached HEAD / no git (fallback to recent spec)

### Phase 3: Root Command Phase Flags

- [ ] T017 Add phase flags to root command in `root.go` (-s, -p, -t, -i, -a)
- [ ] T018 Write tests for flag parsing in `root_test.go`
- [ ] T019 Implement `getPhaseConfig()` helper to extract flags
- [ ] T020 Add `runPhaseWorkflow()` function for phase execution
- [ ] T021 Add validation (at least one phase flag required when no subcommand)
- [ ] T022 Add feature description requirement when -s is used
- [ ] T023 Add `--spec` flag to explicitly specify spec name
- [ ] T024 Update root command RunE to detect phase flags vs subcommands

### Phase 4: Confirmation Flow

- [ ] T025 Implement `confirmContinue()` function with stdin handling
- [ ] T026 Write tests for confirmation logic
- [ ] T027 Add `--yes` / `-y` flag to skip confirmation
- [ ] T028 Add `AUTOSPEC_YES` environment variable support
- [ ] T029 Add `skip_confirmations` config option in `internal/config/`
- [ ] T030 Implement `displayWarningsAndConfirm()` with formatted output
- [ ] T031 Display phase execution plan before confirmation

### Phase 5: Execution Integration

- [ ] T032 Implement `executePhases()` function calling WorkflowOrchestrator
- [ ] T033 Integrate preflight checks before execution
- [ ] T034 Implement sequential phase execution respecting canonical order
- [ ] T035 Handle spec auto-detection for non-specify phases
- [ ] T036 Create spec directory when -s is used on new branch
- [ ] T037 Add progress display support (reuse existing progress system)

### Phase 6: Testing & Polish

- [ ] T038 Write integration tests for Case 1 (all artifacts exist)
- [ ] T039 Write integration tests for Case 2 (missing artifacts)
- [ ] T040 Write integration tests for Case 3 (non-spec branch)
- [ ] T041 Write integration tests for Case 4 (fresh spec branch)
- [ ] T042 Write integration tests for Case 5 (no git)
- [ ] T043 Test flag combinations: -a, -spi, -sp, -pi, -ti, -spti
- [ ] T044 Test --spec flag with explicit spec name
- [ ] T045 Update CLI help text and examples in root.go
- [ ] T046 Update CLAUDE.md with new phase flag documentation

---

## Summary

**Command**: `autospec -spi "feature"` (no subcommand needed)

**Key Benefits:**
1. Shortest possible: `autospec -a "feature"` for full workflow
2. Flexible: Any phase combination with `-s`, `-p`, `-t`, `-i`
3. Branch-aware: Smart detection of current spec from git branch
4. Safe: Context-aware warnings when prerequisites missing
5. User-friendly: y/N confirmation with skip options (`-y`, `AUTOSPEC_YES`, config)
6. Helpful: Clear error messages with actionable suggestions
7. Backward compatible: Existing subcommands unchanged

**Total Tasks**: 46

| Phase | Tasks | Focus |
|-------|-------|-------|
| 1. Core Infrastructure | 5 | PhaseConfig struct, ordering, -a expansion |
| 2. Preflight Checks | 11 | Branch-aware detection, 5 cases |
| 3. Root Command Flags | 8 | Add flags to root, validation |
| 4. Confirmation Flow | 7 | y/N prompts, skip options |
| 5. Execution Integration | 6 | Orchestrator integration |
| 6. Testing & Polish | 9 | Integration tests, docs |

**Complexity**: Medium-High (branch-aware detection, root command modification)
