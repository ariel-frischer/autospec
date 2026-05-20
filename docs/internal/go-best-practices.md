# Go Best Practices for autospec

This document outlines Go best practices tailored specifically for the autospec codebase. It combines industry standards with patterns established in this project.

## Project Structure

autospec follows the [Standard Go Project Layout](https://github.com/golang-standards/project-layout). For detailed component interactions and data flow, see [Architecture Overview](./architecture.md).

```
.
├── cmd/autospec/          # Binary entry point (main.go only)
├── internal/              # Private application code
│   ├── cli/               # Cobra command definitions
│   ├── config/            # Configuration loading (koanf)
│   ├── workflow/          # Workflow orchestration, executor, Claude integration
│   ├── validation/        # Artifact validation (<10ms contract)
│   ├── retry/             # Persistent retry state
│   ├── spec/              # Spec detection logic
│   ├── git/               # Git integration helpers
│   ├── errors/            # Structured error types
│   ├── progress/          # Terminal progress indicators
│   ├── agent/             # Agent context file management
│   ├── commands/          # Embedded slash command templates
│   ├── health/            # Dependency verification
│   ├── yaml/              # YAML parsing utilities
│   ├── clean/             # Project cleanup functions
│   ├── uninstall/         # System uninstall functions
│   └── completion/        # Shell completion helpers
├── docs/                  # Documentation
├── scripts/               # Build and dev scripts
├── specs/                 # Feature specifications (project-specific)
└── tests/                 # Integration tests
```

### Key Principles

1. **`internal/` for everything** - All application code lives in `internal/` to prevent external imports
2. **`cmd/` is minimal** - Only wiring and `main()`, no business logic
3. **Package by domain** - Each package has a single, clear responsibility
4. **No circular dependencies** - If you need to import between packages, reconsider boundaries

## Coding Conventions

### Naming

```go
// Package names: short, lowercase, no underscores
package validation  // Good
package specValidation  // Bad

// Exported identifiers: CamelCase
func ValidateSpecFile(dir string) error

// Unexported identifiers: camelCase
func parseTaskLine(line string) (Task, error)

// Avoid stutter
type Config struct {}     // In package config, not ConfigConfig
type Service struct {}    // In package user, not UserService

// Interfaces: -er suffix for single-method
type Validator interface {
    Validate() error
}

// Multi-method interfaces: descriptive noun
type ArtifactValidator interface {
    Validate(path string) error
    Fix(path string) error
}
```

### Error Handling

```go
// Return error as last value
func LoadConfig(path string) (*Config, error)

// Wrap errors with context at boundaries
if err := os.ReadFile(path); err != nil {
    return fmt.Errorf("reading config %s: %w", path, err)
}

// Use project error types for structured errors (internal/errors/)
return errors.NewValidationError("spec.yaml", "missing required field: feature")

// Never panic in library code
// Reserve panic for truly unrecoverable initialization failures only
```

### Function Design

```go
// Keep functions short and focused (generally <40 lines)
// One function = one responsibility

// Accept interfaces, return concrete types
func NewExecutor(claude ClaudeExecutor, cfg *config.Config) *Executor

// Use functional options for complex construction
func NewWorkflow(opts ...WorkflowOption) *Workflow

// Context as first parameter when needed
func (e *Executor) ExecutePhase(ctx context.Context, phase Phase) error
```

## Testing

autospec uses table-driven tests with map-based test cases for clarity.

### Test File Organization

```go
// Tests live alongside code: foo.go → foo_test.go
// Benchmark tests: foo_bench_test.go (separate file for clarity)

// Use same package for white-box testing
package validation

// Or _test suffix for black-box testing
package validation_test
```

### Table-Driven Tests (Project Pattern)

```go
func TestValidateSpecFile(t *testing.T) {
    // Use map[string]struct for named test cases
    tests := map[string]struct {
        setup   func(t *testing.T) string
        wantErr bool
    }{
        "spec.yaml exists": {
            setup: func(t *testing.T) string {
                dir := t.TempDir()
                // ... setup
                return dir
            },
            wantErr: false,
        },
        "spec.yaml missing": {
            setup:   func(t *testing.T) string { return t.TempDir() },
            wantErr: true,
        },
    }

    for name, tc := range tests {
        t.Run(name, func(t *testing.T) {
            t.Parallel()  // Enable parallel execution
            specDir := tc.setup(t)
            err := ValidateSpecFile(specDir)
            if (err != nil) != tc.wantErr {
                t.Errorf("ValidateSpecFile() error = %v, wantErr %v", err, tc.wantErr)
            }
        })
    }
}
```

### Test Helpers

```go
// Use t.Helper() for test utilities
func setupTestConfig(t *testing.T) *config.Config {
    t.Helper()
    cfg, err := config.Load()
    if err != nil {
        t.Fatalf("failed to load config: %v", err)
    }
    return cfg
}

// Use t.TempDir() for test directories (auto-cleanup)
dir := t.TempDir()

// Use t.Cleanup() for resource cleanup
t.Cleanup(func() {
    os.RemoveAll(tempDir)
})
```

### Benchmark Tests

```go
// Benchmark critical paths (validation functions must be <10ms)
func BenchmarkValidateSpecFile(b *testing.B) {
    dir := setupBenchmarkDir(b)
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        _ = ValidateSpecFile(dir)
    }
}
```

### Agent and Workflow Tests

Tests for agent execution, workflow orchestration, or git behavior must be
deterministic by default. Use `testutil.MockExecutor` for agent calls,
`testutil.E2EEnv` for compiled CLI tests, and `testutil.GitIsolation` or
`t.TempDir()` for repository state. Do not call real configured agent CLIs or
external AI APIs from default unit or package integration tests.

### Running Tests

```bash
# All tests with race detection and coverage
go test -v -race -cover ./...

# Single test
go test -v -run TestValidateSpecFile ./internal/validation/

# Benchmarks
go test -bench=. -benchmem ./internal/validation/
```

## Performance Standards

autospec has strict performance contracts:

| Operation | Target |
|-----------|--------|
| Validation functions | <10ms |
| Retry state load/save | <10ms |
| Config loading | <100ms |
| Overall validation checks | <1s |

### Performance Guidelines

```go
// Avoid allocations in hot paths
// Reuse buffers and slices where possible

// Use sync.Pool for frequently allocated objects
var bufPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

// Prefer io.Reader/Writer over loading entire files
func parseYAML(r io.Reader) (*Spec, error)

// Profile before optimizing
// go test -cpuprofile cpu.prof -memprofile mem.prof
```

## Configuration

autospec uses [koanf](https://github.com/knadh/koanf) for layered configuration.

### Configuration Priority

```
Defaults → User config (~/.config/autospec/config.yml) → Project config (.autospec/config.yml) → Environment (AUTOSPEC_*)
```

### Adding New Config Fields

When adding a new configuration option, update these files:

| File | What to Add |
|------|-------------|
| `internal/config/config.go` | Field in `Configuration` struct with `koanf:"field_name"` tag |
| `internal/config/schema.go` | Entry in `KnownKeys` map with type, description, default |
| `internal/config/defaults.go` | Default YAML template and runtime default in `GetDefaults()`/default map |
| `internal/config/validate.go` | Validation rule when the field has constraints |
| Public docs | User-facing behavior, defaults, and examples when applicable |

**Important**: `config show` uses `Configuration.ToMap()` which automatically includes all `koanf`-tagged fields via reflection. No manual map updates needed.

```go
// config.go - Add field with koanf tag
type Configuration struct {
    // ... existing fields
    MyNewOption bool `koanf:"my_new_option"`
}

// schema.go - Add to KnownKeys for validation and config sync
var KnownKeys = map[string]ConfigKeySchema{
    "my_new_option": {
        Path:        "my_new_option",
        Type:        TypeBool,
        Description: "Enable my new feature",
        Default:     false,
    },
}

// defaults.go - Add default value
func defaultConfig() map[string]interface{} {
    return map[string]interface{}{
        // ... existing defaults
        "my_new_option": false,
    }
}
```

### Environment Variables

```go
// Use AUTOSPEC_ prefix for all env vars
// AUTOSPEC_DEBUG, AUTOSPEC_TIMEOUT, AUTOSPEC_SPECS_DIR

// Map env vars to config fields
k.Load(env.Provider("AUTOSPEC_", ".", func(s string) string {
    return strings.ToLower(strings.TrimPrefix(s, "AUTOSPEC_"))
}), nil)
```

## CLI Commands (Cobra)

### Command Structure

```go
var planCmd = &cobra.Command{
    Use:   "plan [prompt]",
    Short: "Execute the planning phase",
    Long:  `Execute the planning phase with optional prompt guidance.`,
    Args:  cobra.MaximumNArgs(1),
    RunE:  runPlan,
}

func init() {
    rootCmd.AddCommand(planCmd)
    planCmd.Flags().StringVarP(&specName, "spec", "s", "", "Spec name")
}

func runPlan(cmd *cobra.Command, args []string) error {
    // Business logic via workflow package, not here
    return workflow.ExecutePlan(cfg, args)
}
```

### Exit Codes

Use standardized exit codes (defined in project):

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Validation failed (retryable) |
| 2 | Retry limit exhausted |
| 3 | Invalid arguments |
| 4 | Missing dependencies |
| 5 | Command timeout |

## Dependency Management

### Guidelines

1. **Stdlib first** - Use `net/http`, `os`, `io`, `context` before external packages
2. **Minimal dependencies** - Each dependency adds maintenance burden
3. **Document dependencies** - Comment why each dependency exists in `go.mod`
4. **Audit regularly** - Check for vulnerabilities with `govulncheck`

### go.mod Best Practices

```go
// Document dependencies in go.mod
module github.com/ariel-frischer/autospec

go 1.25.1

require (
    // CLI framework for building command-line applications
    github.com/spf13/cobra v1.10.1

    // Configuration management with multiple sources
    github.com/knadh/koanf/v2 v2.3.0

    // Test-only: assertions and mocking
    github.com/stretchr/testify v1.11.1
)
```

## Concurrency

### Guidelines

```go
// Always run tests with -race flag
go test -race ./...

// Use context for cancellation
func (e *Executor) Execute(ctx context.Context) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case result := <-e.work():
        return result
    }
}

// Avoid goroutine leaks
// Always ensure goroutines can exit
done := make(chan struct{})
defer close(done)

go func() {
    select {
    case <-done:
        return
    case work := <-workChan:
        process(work)
    }
}()

// Use sync.WaitGroup for multiple goroutines
var wg sync.WaitGroup
for _, item := range items {
    wg.Add(1)
    go func(item Item) {
        defer wg.Done()
        process(item)
    }(item)
}
wg.Wait()
```

## Code Quality

### Required Before Commit

```bash
# Format code
make fmt
# or: go fmt ./...

# Run vet
go vet ./...

# Run tests
go test -race ./...
```

### Recommended Tooling

```bash
# Static analysis (golangci-lint)
golangci-lint run

# Security scanning
govulncheck ./...

# Check for unused code
staticcheck ./...
```

## Documentation

### Code Comments

```go
// Package validation provides artifact validation functions.
// All validation functions maintain a <10ms performance contract.
package validation

// ValidateSpecFile checks that spec.yaml exists and is valid.
// Returns nil if valid, error with details otherwise.
func ValidateSpecFile(dir string) error

// Internal functions don't need doc comments unless complex
func parseTaskLine(line string) (Task, error)
```

### When to Comment

- All exported types, functions, and constants
- Complex algorithms or non-obvious logic
- Performance-critical code with explanations
- **Not needed**: obvious code, unexported helpers, test code

## Common Patterns in autospec

### Validation Pattern

```go
// Validators implement this interface
type ArtifactValidator interface {
    Validate(path string) ([]ValidationError, error)
    Fix(path string) error
    Type() string
}

// Register validators
var validators = map[string]ArtifactValidator{
    "spec":  &SpecValidator{},
    "plan":  &PlanValidator{},
    "tasks": &TasksValidator{},
}
```

### Phase Execution Pattern

```go
// Phases follow this execution pattern
type Phase string

const (
    PhaseSpecify   Phase = "specify"
    PhasePlan      Phase = "plan"
    PhaseTasks     Phase = "tasks"
    PhaseImplement Phase = "implement"
)

func (e *Executor) ExecutePhase(phase Phase, validate ValidateFunc) error {
    // 1. Load retry state
    // 2. Execute command
    // 3. Run validation
    // 4. Update retry state
    // 5. Retry or return
}
```

### Spec Detection Pattern

```go
// Detection with fallback strategies
func DetectCurrentSpec() (*Metadata, error) {
    // 1. Try git branch name
    if meta := detectFromBranch(); meta != nil {
        return meta, nil
    }
    // 2. Fall back to most recent spec directory
    return detectMostRecent()
}
```

## References

### Project Documentation

- [Architecture Overview](./architecture.md) - Component interactions, data flow, and system design
- [CLAUDE.md](../../CLAUDE.md) - Detailed development guidelines and commands

### External Resources

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- [Go Proverbs](https://go-proverbs.github.io/)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
