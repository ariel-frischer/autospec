# Arch 10: Targeted Test Coverage Improvement

**Location:** Multiple packages (CLI subpackages, notify, completion, retry)
**Impact:** MEDIUM - Improves confidence without adding abstraction complexity
**Effort:** MEDIUM
**Dependencies:** arch-1, arch-2 COMPLETED (provides existing interfaces to leverage)

## Coverage Baseline (2025-12-18)

### Critical Gaps (0% coverage - MUST FIX)

| Package | Current | Target | Priority |
|---------|---------|--------|----------|
| `internal/cli/admin` | 0% | 85% | HIGH |
| `internal/cli/config` | 0% | 85% | HIGH |
| `internal/cli/shared` | 0% | 85% | HIGH |
| `internal/cli/stages` | 0% | 85% | HIGH |
| `internal/cli/util` | 0% | 85% | HIGH |

### Below Threshold (<85%)

| Package | Current | Target | Priority |
|---------|---------|--------|----------|
| `internal/cli` | 35.8% | 85% | HIGH |
| `internal/notify` | 56.9% | 85% | MEDIUM |
| `internal/completion` | 65.2% | 85% | LOW |
| `internal/retry` | 83.4% | 85% | LOW |
| `internal/health` | 83.3% | 85% | LOW |

### Already Meeting Threshold (no action needed)

| Package | Current |
|---------|---------|
| `internal/validation` | 91.2% |
| `internal/config` | 92.1% |
| `internal/workflow` | 86.4% |
| `internal/yaml` | 94.4% |
| `internal/spec` | 94.3% |
| `internal/lifecycle` | 94.0% |
| `internal/errors` | 96.7% |
| `internal/clean` | 96.4% |
| `internal/uninstall` | 90.5% |
| `internal/commands` | 88.0% |
| `internal/claude` | 88.9% |
| `internal/agent` | 86.7% |
| `internal/progress` | 88.6% |
| `internal/history` | 85.7% |
| `internal/git` | 85.0% |

### Excluded from Coverage

| Package | Reason |
|---------|--------|
| `cmd/autospec` | Entry point only, nothing to test |
| `internal/testutil` | Test utilities, not production code |
| `integration` | Integration tests, not subject to coverage |
| `mocks/scripts` | Test fixtures |

## Approach

### Step 1: CLI Subpackages (0% → 85%)

These command packages need tests for:
- Flag parsing and validation
- Error handling paths
- Command execution flow (mock the underlying workflow/orchestrator)

```go
// Example: internal/cli/stages/specify_test.go
func TestSpecifyCommand_Flags(t *testing.T) {
    t.Parallel()
    tests := map[string]struct {
        args    []string
        wantErr string
    }{
        "missing description": {
            args:    []string{"specify"},
            wantErr: "description required",
        },
        "valid description": {
            args:    []string{"specify", "add user auth"},
            wantErr: "",
        },
    }
    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            t.Parallel()
            cmd := NewSpecifyCmd()
            cmd.SetArgs(tt.args)
            // Test flag parsing without executing
        })
    }
}
```

### Step 2: CLI Main Package (35.8% → 85%)

Focus on:
- Root command setup
- Subcommand registration
- Global flag handling
- Error formatting

### Step 3: Notify Package (56.9% → 85%)

Test:
- Different notification backends (sound, visual)
- Error handling when notifications fail
- Configuration parsing

### Step 4: Completion Package (65.2% → 85%)

Test:
- Shell completion generation (bash, zsh, fish, powershell)
- Dynamic completion functions

### Step 5: Remaining Packages (83% → 85%)

Minor additions to:
- `internal/retry` - edge cases in state persistence
- `internal/health` - error paths in dependency checks

## Test Patterns

### For CLI Commands (use Cobra test utilities)

```go
func TestCommand_Execute(t *testing.T) {
    t.Parallel()

    // Create command with mocked dependencies
    cmd := NewMyCmd()
    cmd.SetOut(io.Discard)
    cmd.SetErr(io.Discard)
    cmd.SetArgs([]string{"--flag", "value"})

    // For commands that call orchestrator, inject mock
    err := cmd.Execute()

    // Assert
}
```

### For Notify Package (test without side effects)

```go
func TestHandler_OnCommandComplete(t *testing.T) {
    t.Parallel()
    tests := map[string]struct {
        cfg     config.NotificationConfig
        success bool
    }{
        "sound enabled success": {
            cfg:     config.NotificationConfig{Sound: true},
            success: true,
        },
        "sound disabled": {
            cfg:     config.NotificationConfig{Sound: false},
            success: true,
        },
    }
    // ...
}
```

## Acceptance Criteria

- [ ] `internal/cli/admin` reaches 85% coverage
- [ ] `internal/cli/config` reaches 85% coverage
- [ ] `internal/cli/shared` reaches 85% coverage
- [ ] `internal/cli/stages` reaches 85% coverage
- [ ] `internal/cli/util` reaches 85% coverage
- [ ] `internal/cli` reaches 85% coverage
- [ ] `internal/notify` reaches 85% coverage
- [ ] `internal/completion` reaches 85% coverage
- [ ] `internal/retry` reaches 85% coverage
- [ ] `internal/health` reaches 85% coverage
- [ ] **No new interfaces added** (use existing mocks/stubs)
- [ ] **No changes to production code** (tests only)
- [ ] All existing tests pass
- [ ] `make test`, `make fmt`, `make lint`, `make build` all pass

## Non-Functional Requirements

- Tests use map-based table-driven pattern with t.Parallel()
- All errors wrapped with context
- Functions under 40 lines
- Prefer Cobra test utilities for CLI tests
- Use t.TempDir() for file I/O tests
- Use existing arch-1/2 interfaces for mocking execution

## Anti-Patterns to Avoid

1. **Don't add interfaces just for mocking** - Use real objects or existing mocks
2. **Don't mock file systems** - Use t.TempDir() with real files
3. **Don't modify production code** - Tests only in this task
4. **Don't test implementation details** - Test behavior
5. **Don't over-mock Cobra** - Test actual command execution where possible

## Files to Create

| File | Purpose |
|------|---------|
| `internal/cli/admin/commands_test.go` | Admin command tests |
| `internal/cli/admin/completion_test.go` | Completion command tests |
| `internal/cli/admin/uninstall_test.go` | Uninstall command tests |
| `internal/cli/config/config_test.go` | Config command tests |
| `internal/cli/config/doctor_test.go` | Doctor command tests |
| `internal/cli/config/init_test.go` | Init command tests |
| `internal/cli/config/migrate_test.go` | Migrate command tests |
| `internal/cli/shared/types_test.go` | Shared types tests |
| `internal/cli/stages/specify_test.go` | Specify command tests |
| `internal/cli/stages/plan_test.go` | Plan command tests |
| `internal/cli/stages/tasks_test.go` | Tasks command tests |
| `internal/cli/stages/implement_test.go` | Implement command tests |
| `internal/cli/stages/clarify_test.go` | Clarify command tests |
| `internal/cli/stages/analyze_test.go` | Analyze command tests |
| `internal/cli/stages/checklist_test.go` | Checklist command tests |
| `internal/cli/stages/constitution_test.go` | Constitution command tests |
| `internal/cli/util/status_test.go` | Status command tests |
| `internal/cli/util/history_test.go` | History command tests |
| `internal/cli/util/version_test.go` | Version command tests |
| `internal/cli/util/clean_test.go` | Clean command tests |
| `internal/notify/handler_test.go` | Additional notify tests |
| `internal/completion/completion_test.go` | Additional completion tests |

## Command

```bash
autospec specify "$(cat .dev/tasks/arch/arch-10-targeted-test-coverage.md)"
```
