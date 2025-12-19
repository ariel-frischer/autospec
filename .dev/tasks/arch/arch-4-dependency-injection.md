# Arch 4: Complete Dependency Injection (SKIPPED)

**Status:** SKIPPED
**Skipped:** 2025-12-18
**Superseded by:** arch-10 (Targeted Test Coverage)

## Skip Reason

After analysis, arch-1/2 already solved the core testability problem via executor interfaces:
- `ClaudeRunner` - mock command execution
- `StageExecutorInterface`, `PhaseExecutorInterface`, `TaskExecutorInterface` - mock delegation
- `ProgressController`, `NotifyDispatcher` - mock side effects

Adding ConfigProvider, ArtifactValidator, and RetryStateStore interfaces would be abstraction for abstraction's sake:
- **ConfigProvider is unnecessary** - `config.Configuration` is already a simple struct; you can construct one with any test values
- **ArtifactValidator is misplaced** - validation should be tested in its own package, not mocked in orchestrator
- **RetryStateStore has marginal value** - temp directories work fine for file I/O tests

The 7-field `OrchestratorDeps` was a code smell suggesting over-engineering, not a solution.

---

## Original Proposal (for reference)

**Location:** Multiple packages (workflow, config, validation, retry)
**Impact:** MEDIUM - Completes testability improvements started in arch-1/2
**Effort:** LOW-MEDIUM (reduced scope due to prior work)

### Proposed Interfaces (NOT IMPLEMENTED)

```go
// ConfigProvider - SKIPPED: config.Configuration is already a simple struct
type ConfigProvider interface {
    GetClaudeCmd() string
    GetMaxRetries() int
    // ...
}

// ArtifactValidator - SKIPPED: test validation package directly
type ArtifactValidator interface {
    ValidateSpec(path string) *validation.Result
    // ...
}

// RetryStateStore - SKIPPED: use t.TempDir() for file tests
type RetryStateStore interface {
    LoadState(specName, phase string, maxRetries int) (*retry.RetryState, error)
    // ...
}
```

### Why This Seemed Like a Good Idea

- "Complete the pattern" started in arch-1/2
- More interfaces = more mockability = better tests (right?)

### Why It Was Wrong

- More interfaces ≠ better tests
- Coverage comes from writing tests, not from adding mocking capability
- Integration tests with real files are often simpler and more realistic
- The existing arch-1/2 interfaces cover the actual execution paths that need mocking
