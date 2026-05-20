# Mock Testing Patterns

This guide documents the mock testing infrastructure for autospec and provides patterns for writing tests that don't make real Claude CLI calls or pollute git state.

## Test Categories

autospec uses three test categories at different layers:

| Type | Location | Build Tag | Mock Level | Purpose |
|------|----------|-----------|------------|---------|
| **Unit** | `internal/*/` | none | Function-level | Test individual functions in isolation |
| **Integration (pkg)** | `internal/*_integration_test.go` | none | Interface mocks | Test packages with `MockExecutor` |
| **Integration (workflow)** | `tests/integration/` | `integration` | Interface mocks | Test YAML workflow end-to-end |
| **E2E** | `tests/e2e/` | `e2e` | Binary-level | Test compiled CLI with mock binary |

### Integration vs E2E

**Package-level integration tests** (`internal/workflow/integration_test.go`):
- Run with regular `go test` (no build tag)
- Test internal Go packages collaborating together
- Use `testutil.MockExecutor` to mock the Claude executor interface
- Use `testutil.GitIsolation` to test in isolated git repos
- Focus: orchestration logic, retry behavior, artifact generation

**Workflow-level integration tests** (`tests/integration/`):
- Require `go test -tags=integration`
- Test complete YAML workflow artifact generation and validation
- Focus: command template installation, YAML validation, migrations

**E2E tests** (`tests/e2e/e2e_test.go`):
- Require `go test -tags=e2e`
- Test the compiled `autospec` binary as users experience it
- Use `testutil.E2EEnv` with mock Claude binary in PATH
- Focus: CLI invocation, environment isolation, command-to-artifact chain

```bash
# Run unit + package integration tests (default)
make test

# Run workflow integration tests (separate)
go test -tags=integration ./tests/integration/...

# Run e2e tests (separate)
go test -tags=e2e ./tests/e2e/...
```

All test types run in GitHub CI on the `main` branch.

## Overview

The mock testing infrastructure enables:
- **No Real Claude Calls**: Tests verify workflow behavior without API costs or network access
- **Git Isolation**: Tests can manipulate git state without affecting the actual repository
- **Deterministic Testing**: Mocks provide consistent, reproducible responses

Default unit, package integration, and workflow tests must not invoke real
configured agents or external AI APIs. Use explicit E2E/manual testing plans for
real agent smoke tests, and keep those paths isolated from the developer's active
repository.

## Infrastructure Components

### Mock Executor (`internal/testutil/mock_executor.go`)

The `MockExecutor` provides a fluent API for configuring mock Claude CLI behavior.

#### Basic Usage

```go
import "github.com/ariel-frischer/autospec/internal/testutil"

func TestWorkflow(t *testing.T) {
    t.Parallel()

    // Create mock with fluent builder
    builder := testutil.NewMockExecutorBuilder(t)
    builder.
        WithResponse("spec created").
        ThenResponse("plan created").
        ThenError(errors.New("simulated failure"))

    mock := builder.Build()

    // Use mock in test
    err := mock.Execute("/autospec.specify")
    if err != nil {
        t.Fatal(err)
    }

    // Verify calls
    if mock.GetCallCount() != 1 {
        t.Errorf("expected 1 call, got %d", mock.GetCallCount())
    }
}
```

#### Response Sequencing

Configure different responses for sequential calls:

```go
builder := testutil.NewMockExecutorBuilder(t)
builder.
    WithResponse("first response").   // First call
    ThenResponse("second response").  // Second call
    ThenError(workflow.ErrMockExecute) // Third call fails
```

#### Delay Simulation

Test timeout handling with simulated delays:

```go
builder := testutil.NewMockExecutorBuilder(t)
builder.
    WithResponse("success").
    WithDelay(500 * time.Millisecond) // Adds delay before response
```

#### Artifact Generation

Configure mock to generate artifacts on execution:

```go
builder := testutil.NewMockExecutorBuilder(t)
builder.
    WithArtifactDir(specsDir).
    WithResponse("created").
    WithArtifactGeneration(testutil.ArtifactGenerators.Spec)

mock := builder.Build()
mock.Execute("/autospec.specify") // Creates spec.yaml in specsDir
```

Available generators:
- `testutil.ArtifactGenerators.Spec` - Creates valid spec.yaml
- `testutil.ArtifactGenerators.Plan` - Creates valid plan.yaml
- `testutil.ArtifactGenerators.Tasks` - Creates valid tasks.yaml

#### Call Verification

Verify mock was called correctly:

```go
// Get all calls
calls := mock.GetCalls()

// Get calls by method
executeCalls := mock.GetCallsByMethod("Execute")

// Assert specific call was made
mock.AssertCalled(t, "Execute", "specify") // Checks if any call contains "specify"

// Assert method was not called
mock.AssertNotCalled(t, "StreamCommand")

// Assert call count
mock.AssertCallCount(t, "Execute", 3)

// Reset for reuse
mock.Reset()
```

### Git Isolation (`internal/testutil/git_isolation.go`)

The `GitIsolation` helper creates temporary git repositories for testing.

#### Basic Usage

```go
import "github.com/ariel-frischer/autospec/internal/testutil"

func TestGitOperations(t *testing.T) {
    t.Parallel()

    // Creates temp git repo, changes to it, restores on cleanup
    gi := testutil.NewGitIsolation(t)

    // Now working in isolated temp repo
    gi.CreateBranch("test-feature", true)

    // Add files
    gi.AddFile("test.txt", "content")
    gi.CommitAll("Test commit")

    // Cleanup is automatic via t.Cleanup
}
```

#### Alternative Cleanup Pattern

For explicit cleanup control:

```go
func TestWithExplicitCleanup(t *testing.T) {
    cleanup := testutil.WithIsolatedGitRepo(t)
    defer cleanup()

    // Test code here
}
```

#### Specs Directory Setup

Set up spec directory structure:

```go
gi := testutil.NewGitIsolation(t)

// Creates specs/test-feature/ directory
specDir := gi.SetupSpecsDir("test-feature")

// Write a spec file
specPath := gi.WriteSpec(specDir) // Creates spec.yaml with valid content
```

#### Branch Verification

Verify original repository wasn't modified:

```go
gi := testutil.NewGitIsolation(t)

// ... test operations ...

// Verify original branch unchanged
gi.VerifyNoBranchPollution()
```

### Mock Claude Shell Script (`tests/mocks/scripts/mock-claude.sh`)

For integration tests that need to spawn actual processes.

#### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MOCK_RESPONSE_FILE` | Path to file containing response | Empty |
| `MOCK_CALL_LOG` | Path to log file for calls | No logging |
| `MOCK_EXIT_CODE` | Exit code to return | 0 |
| `MOCK_DELAY` | Seconds to delay | 0 |

#### Usage

```bash
# Configure mock
export MOCK_RESPONSE_FILE=/tmp/response.yaml
export MOCK_CALL_LOG=/tmp/calls.log
export MOCK_EXIT_CODE=0

# Run tests with mock claude (using custom agent configuration)
AUTOSPEC_CUSTOM_AGENT_CMD="./tests/mocks/scripts/mock-claude.sh {{PROMPT}}" go test ./...

# Verify calls
cat /tmp/calls.log
```

### Fixtures (`tests/mocks/fixtures/`)

Pre-built YAML fixtures for testing:

```
tests/mocks/fixtures/
├── valid/
│   ├── spec.yaml      # Complete, valid spec
│   ├── plan.yaml      # Valid plan linked to spec
│   └── tasks.yaml     # Valid tasks with sample tasks
├── invalid/
│   ├── spec-missing-feature.yaml
│   ├── plan-bad-reference.yaml
│   └── tasks-orphan.yaml
└── partial/
    └── ...
```

## Test Patterns

### Pattern 1: Map-Based Table Tests with Mocks

```go
func TestWorkflow(t *testing.T) {
    t.Parallel()

    tests := map[string]struct {
        setupMock   func(*testutil.MockExecutorBuilder)
        wantErr     bool
        verifyMock  func(*testing.T, *testutil.MockExecutor)
    }{
        "successful execution": {
            setupMock: func(b *testutil.MockExecutorBuilder) {
                b.WithResponse("success")
            },
            wantErr: false,
            verifyMock: func(t *testing.T, m *testutil.MockExecutor) {
                if m.GetCallCount() != 1 {
                    t.Error("expected 1 call")
                }
            },
        },
        "handles failure": {
            setupMock: func(b *testutil.MockExecutorBuilder) {
                b.WithError(errors.New("test error"))
            },
            wantErr: true,
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            t.Parallel()

            builder := testutil.NewMockExecutorBuilder(t)
            tt.setupMock(builder)
            mock := builder.Build()

            err := mock.Execute("test")

            if tt.wantErr && err == nil {
                t.Error("expected error")
            }
            if !tt.wantErr && err != nil {
                t.Errorf("unexpected error: %v", err)
            }
            if tt.verifyMock != nil {
                tt.verifyMock(t, mock)
            }
        })
    }
}
```

### Pattern 2: Isolated Git Operations

```go
func TestTaskStatus(t *testing.T) {
    t.Parallel()

    gi := testutil.NewGitIsolation(t)
    specDir := gi.SetupSpecsDir("test-feature")

    // Create tasks file
    tasksPath := filepath.Join(specDir, "tasks.yaml")
    os.WriteFile(tasksPath, []byte(tasksContent), 0644)

    // Modify and verify
    // ... test operations ...

    // Original repo unchanged (verified automatically)
}
```

### Pattern 3: Retry Behavior Testing

```go
func TestRetries(t *testing.T) {
    t.Parallel()

    builder := testutil.NewMockExecutorBuilder(t)
    builder.
        WithError(errors.New("fail 1")).
        ThenError(errors.New("fail 2")).
        ThenResponse("success")

    mock := builder.Build()

    var lastErr error
    for attempts := 0; attempts < 3; attempts++ {
        if err := mock.Execute("cmd"); err == nil {
            break
        } else {
            lastErr = err
        }
    }

    if mock.GetCallCount() != 3 {
        t.Errorf("expected 3 attempts")
    }
}
```

### Pattern 4: Artifact Validation in Isolation

```go
func TestArtifactValidation(t *testing.T) {
    t.Parallel()

    gi := testutil.NewGitIsolation(t)
    specDir := gi.SetupSpecsDir("test")

    // Use mock fixtures
    fixtureContent, _ := os.ReadFile("tests/mocks/fixtures/valid/spec.yaml")
    specPath := filepath.Join(specDir, "spec.yaml")
    os.WriteFile(specPath, fixtureContent, 0644)

    // Run validation
    err := validation.ValidateSpecFile(specDir)
    if err != nil {
        t.Errorf("validation failed: %v", err)
    }
}
```

## Best Practices

1. **Always use `t.Parallel()`** - Enable parallel test execution
2. **Use map-based table tests** - Follow Go conventions for test organization
3. **Prefer mocks over real calls** - Never call real Claude CLI in tests
4. **Use git isolation for git operations** - Prevent branch pollution
5. **Verify mock calls** - Assert expected commands were invoked
6. **Test failure paths** - Use `WithError()` to test error handling
7. **Test timeouts** - Use `WithDelay()` to test timeout behavior
8. **Use fixtures for validation** - Pre-built valid/invalid YAML files
9. **Keep tests fast** - Avoid real delays; use simulated delays only when testing timeout logic

## When to Use Mocks vs Real Integration Tests

| Scenario | Use Mocks | Use Real Integration |
|----------|-----------|---------------------|
| Unit testing workflow logic | ✓ | |
| Testing retry behavior | ✓ | |
| Testing timeout handling | ✓ | |
| Testing validation | ✓ | |
| Testing CLI argument parsing | | ✓ |
| Testing actual Claude responses | | ✓ (sparingly) |
| CI/CD pipeline tests | ✓ | |
| Local development tests | ✓ | |

## Troubleshooting

### Mock not returning expected response

Check that you're using the mock builder correctly:
```go
// Wrong - responses are consumed in order
mock.Execute("cmd1") // Gets first response
mock.Execute("cmd1") // Gets SECOND response, not first again

// Solution - add enough responses
builder.
    WithResponse("response1").
    ThenResponse("response2").
    ThenResponse("response3")
```

### Git isolation not working

Ensure you're using `NewGitIsolation` or `WithIsolatedGitRepo`:
```go
// Correct
gi := testutil.NewGitIsolation(t)

// Or
cleanup := testutil.WithIsolatedGitRepo(t)
defer cleanup()
```

### Tests failing with "file not found"

Make sure to create artifacts before testing:
```go
gi := testutil.NewGitIsolation(t)
specDir := gi.SetupSpecsDir("test")

// Create the file first
os.WriteFile(filepath.Join(specDir, "spec.yaml"), content, 0644)

// Then validate
validation.ValidateSpecFile(specDir)
```

### Mock executor not recording calls

Ensure you're using the mock returned by `Build()`:
```go
builder := testutil.NewMockExecutorBuilder(t)
builder.WithResponse("ok")
mock := builder.Build() // Use this mock, not builder

mock.Execute("cmd")
fmt.Println(mock.GetCallCount()) // Should be 1
```
