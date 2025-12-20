# Mock Testing Infrastructure

This directory contains mock infrastructure for testing autospec without making real Claude CLI calls or modifying the actual git repository state.

## Directory Structure

```
mocks/
├── README.md           # This file
├── scripts/            # Mock shell scripts
│   ├── mock-claude.sh  # Simulates Claude CLI behavior
│   └── generate-mock-artifacts.sh  # Creates test artifacts
└── fixtures/           # Static test data files
    ├── valid-spec.yaml     # Valid spec artifact
    ├── valid-plan.yaml     # Valid plan artifact
    ├── valid-tasks.yaml    # Valid tasks artifact
    ├── invalid-*.yaml      # Invalid artifacts for error testing
    └── partial-*.yaml      # Partial artifacts for edge cases
```

## Purpose

The mock infrastructure serves three critical purposes:

1. **No Real Claude Calls**: Tests can verify workflow behavior without incurring API costs or requiring network access
2. **Git Isolation**: Tests can manipulate git state without polluting the actual repository branches
3. **Deterministic Testing**: Mocks provide consistent, reproducible responses for reliable test assertions

## Components

### mock-claude.sh

A shell script that simulates the Claude CLI with configurable behavior.

**Environment Variables:**

| Variable | Description | Default |
|----------|-------------|---------|
| `MOCK_RESPONSE_FILE` | Path to file containing the response to return | Empty response |
| `MOCK_CALL_LOG` | Path to log file for recording calls | No logging |
| `MOCK_EXIT_CODE` | Exit code to return | 0 |
| `MOCK_DELAY` | Seconds to delay before responding (for timeout testing) | 0 |

**Example Usage:**

```bash
# Configure mock to return specific response
export MOCK_RESPONSE_FILE="/tmp/mock-response.yaml"
export MOCK_CALL_LOG="/tmp/claude-calls.log"
export MOCK_EXIT_CODE=0

# Run tests with mock claude
AUTOSPEC_CUSTOM_AGENT_CMD="./mocks/scripts/mock-claude.sh {{PROMPT}}" go test ./internal/workflow/...

# Verify calls made
cat /tmp/claude-calls.log
```

### generate-mock-artifacts.sh

Creates valid autospec artifacts (spec.yaml, plan.yaml, tasks.yaml) for testing.

**Arguments:**

```bash
./mocks/scripts/generate-mock-artifacts.sh [OPTIONS]

Options:
  -o, --output DIR     Output directory (default: temp dir, printed to stdout)
  -f, --feature NAME   Feature name (default: "test-feature")
  -t, --tasks COUNT    Number of tasks to generate (default: 3)
  -h, --help           Show help
```

**Example Usage:**

```bash
# Generate artifacts in temp directory
OUTPUT_DIR=$(./mocks/scripts/generate-mock-artifacts.sh -f "my-feature" -t 5)

# Verify artifacts are valid
./bin/autospec artifact "$OUTPUT_DIR/spec.yaml"
./bin/autospec artifact "$OUTPUT_DIR/plan.yaml"
./bin/autospec artifact "$OUTPUT_DIR/tasks.yaml"
```

### fixtures/

Pre-built YAML fixtures for common test scenarios:

| File | Description |
|------|-------------|
| `valid-spec.yaml` | Complete, valid spec artifact |
| `valid-plan.yaml` | Complete, valid plan artifact linked to valid-spec |
| `valid-tasks.yaml` | Complete, valid tasks artifact with sample tasks |
| `invalid-spec-missing-feature.yaml` | Spec missing required feature section |
| `invalid-plan-bad-reference.yaml` | Plan with invalid spec reference |
| `invalid-tasks-orphan.yaml` | Tasks with dependencies on non-existent tasks |
| `partial-spec-minimal.yaml` | Minimal valid spec (only required fields) |
| `partial-tasks-empty-phases.yaml` | Tasks with no tasks in phases |

## Go Test Helpers

The mock infrastructure integrates with Go test helpers in `internal/testutil/`:

### Git Isolation

```go
import "github.com/ariel-frischer/autospec/internal/testutil"

func TestWorkflow(t *testing.T) {
    // Creates isolated git repo, restores original dir on cleanup
    cleanup := testutil.WithIsolatedGitRepo(t)
    defer cleanup()

    // Test code runs in temp git repo
    // Original branch/repo unchanged after test
}
```

### Mock Executor

```go
import "github.com/ariel-frischer/autospec/internal/testutil"

func TestWorkflowExecution(t *testing.T) {
    // Build mock with fluent API
    mock := testutil.NewMockExecutorBuilder().
        WithResponse("spec-response.yaml").
        ThenResponse("plan-response.yaml").
        ThenFail(errors.New("simulated failure")).
        Build()

    // Inject mock into workflow
    orch := workflow.NewOrchestrator(cfg, workflow.WithExecutor(mock))

    // Verify calls after test
    calls := mock.GetCalls()
    assert.Equal(t, 3, len(calls))
}
```

## Best Practices

1. **Always use mocks for workflow tests** - Never call real Claude CLI in tests
2. **Use git isolation for any test that modifies git state** - Prevents branch pollution
3. **Prefer fixtures over generated artifacts** - Fixtures are deterministic
4. **Verify mock calls** - Assert that expected commands were invoked
5. **Test timeouts with MOCK_DELAY** - Ensure timeout handling works correctly
6. **Test failures with MOCK_EXIT_CODE** - Verify error handling paths

## Adding New Fixtures

When adding new fixtures:

1. Create the YAML file in `mocks/fixtures/`
2. Ensure it passes validation: `./bin/autospec artifact mocks/fixtures/new-fixture.yaml`
3. Document the fixture's purpose in this README
4. Add tests that use the fixture

## Troubleshooting

### Mock not being used

Ensure `AUTOSPEC_CUSTOM_AGENT_CMD` environment variable is set to the mock script path with `{{PROMPT}}` placeholder before running tests.

### Git isolation not working

Verify the test calls `testutil.WithIsolatedGitRepo(t)` at the start and defers the cleanup function.

### Fixtures failing validation

Run `./bin/autospec artifact <file>` to see detailed validation errors. Ensure the `_meta` section is present and correct.
