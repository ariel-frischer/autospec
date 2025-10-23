# Research: Go Binary Migration

**Feature**: Go Binary Migration (002-go-binary-migration)
**Date**: 2025-10-22

This document consolidates research findings to resolve NEEDS CLARIFICATION items from the Technical Context section of plan.md.

---

## 1. Go CLI Framework Selection

### Decision: **Cobra** (spf13/cobra)

### Rationale
- Industry standard used by kubectl, docker, hugo, GitHub CLI
- 181,299+ projects depend on it
- Binary size: 3,420 KB stripped (only 36 KB larger than Kong alternative)
- Meets all requirements: subcommands, flag parsing, auto-help, <50ms startup
- Active maintenance (v1.9.1, September 2025)
- Extensive documentation and community support
- Historical dependency bloat issues resolved in 2022

### Alternatives Considered
1. **Kong** (alecthomas/kong) - 3,384 KB binary, cleaner API, minimal deps
   - Rejected: Smaller community (2.8K vs 41.7K stars), less enterprise adoption
   - Good alternative if code elegance prioritized over battle-testing
2. **urfave/cli** - 3,684 KB binary, simple API
   - Rejected: Larger binary, fewer features than Cobra, less elegant than Kong
3. **go-flags** - 2,304 KB binary (smallest)
   - Rejected: Less sophisticated than major frameworks

### Implementation Notes
```bash
# Quick start
go install github.com/spf13/cobra-cli@latest
cobra-cli init autospec
cobra-cli add init
cobra-cli add workflow
```

---

## 2. Configuration Management Library

### Decision: **Koanf** (github.com/knadh/koanf) + **go-playground/validator**

### Rationale
- Binary overhead: Only 334 KB (Koanf 300 KB + validator 34 KB)
- Viper alternative adds 9.4 MB overhead (313% larger than baseline)
- Modular architecture: only import what you use
- Preserves JSON case sensitivity (Viper forces lowercase)
- Explicit config merging with clear precedence
- Fast config loading (<1ms typical)
- Active maintenance, modern design

### Alternatives Considered
1. **Viper** (spf13/viper) - Native Cobra integration, feature-rich
   - Rejected: 12 MB binary (vs 2.9 MB for Koanf), 55 total dependencies, breaks JSON spec with forced lowercase keys
2. **envconfig** - Extremely lightweight (~100 KB)
   - Rejected: No JSON file support, single source only (env vars)
3. **Native encoding/json** - Zero dependencies, smallest binary
   - Rejected: Requires 200-300 LOC manual implementation for merging, override logic, validation

### Implementation Pattern
```go
k := koanf.New(".")

// 1. Load global config
k.Load(file.Provider("~/.autospec/config.json"), json.Parser())

// 2. Override with local config
k.Load(file.Provider(".autospec/config.json"), json.Parser())

// 3. Override with env vars (highest priority)
k.Load(env.Provider("AUTOSPEC_", ".", transformFunc), nil)

// Unmarshal and validate
var cfg Config
k.Unmarshal("", &cfg)
validate.Struct(cfg)
```

### Override Precedence (Highest to Lowest)
1. Environment variables (`AUTOSPEC_*`)
2. Local config (`.autospec/config.json`)
3. Global config (`~/.autospec/config.json`)
4. Struct defaults

---

## 3. Git Operations Library

### Decision: **os/exec** calling git binary directly

### Rationale
- Zero binary size overhead (stdlib only)
- Extremely simple: 10-20 lines of code for required operations
- Fastest for basic operations (git rev-parse 2.5x faster than alternatives)
- No external Go dependencies
- Battle-tested: official Go tooling uses this approach
- Easy to test with interfaces

### Alternatives Considered
1. **go-git** (github.com/go-git/go-git/v5) - Pure Go implementation
   - Rejected: 2-4 MB overhead, 57+ dependencies, recent CVEs (CVE-2025-21614, CVE-2025-21613), overkill for 2 simple queries
   - Use case: Complex git operations (clone, push, commit) or environments without git binary
2. **git2go** - libgit2 bindings
   - Rejected: Requires CGO (violates cross-platform requirement), complex builds
3. **Lightweight wrappers** (go-git-cmd-wrapper, go-gitcmd) - Clean API around exec
   - Rejected: Additional dependency (10-50 KB) for marginal benefit

### Implementation
```go
// Get current branch
cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
output, err := cmd.Output()
branch := strings.TrimSpace(string(output))

// Get repository root
cmd := exec.Command("git", "rev-parse", "--show-toplevel")
output, err := cmd.Output()
root := strings.TrimSpace(string(output))

// Check if in git repo
cmd := exec.Command("git", "rev-parse", "--git-dir")
isRepo := cmd.Run() == nil
```

### Trade-off
Requires git binary installed on target systems, but acceptable because:
- Tool is git-aware (reads branch names)
- Users almost certainly have git installed
- Graceful error messaging: "git not found - please install git"

---

## 4. Testing Framework and Best Practices

### Decision: **Hybrid Approach**
- Standard `testing` package (foundation)
- **testify/assert** for readable assertions
- **testify/require** for fatal assertions
- **testscript** for CLI-specific integration tests

### Rationale
- Standard library provides excellent built-in support (coverage, parallel, benchmarks)
- Testify adds expressiveness without replacing core framework
- testscript offers CLI-specific DSL (derived from Go's own tool testing)
- Idiomatic Go patterns (table-driven tests)
- Achieves 60+ test baseline efficiently

### Testing Strategy

#### Test Structure
```
auto-claude-speckit/
├── internal/
│   ├── validation/
│   │   ├── validation_test.go          # Unit tests (white-box)
│   │   ├── validation_external_test.go # Black-box API tests
│   │   └── validation_bench_test.go    # Benchmarks
│   ├── retry/
│   └── workflow/
├── integration/
│   ├── workflow_test.go                # End-to-end workflow tests
│   └── testdata/
│       └── scripts/                    # testscript files
└── cmd/speckit/testdata/scripts/       # CLI testscript tests
```

#### Test Distribution (63-80 tests total)
1. **Unit Tests** (35-40 tests): Validation, retry, file ops, task parsing, spec detection
2. **CLI Tests with testscript** (15-20 tests): Argument parsing, output validation, error scenarios
3. **Integration Tests** (8-12 tests): Complete workflows, hook behavior, retry scenarios
4. **Benchmarks** (5-8 tests): Validation, workflow, hooks, file operations

#### Table-Driven Test Pattern
```go
func TestValidateSpec(t *testing.T) {
    tests := map[string]struct {
        specName string
        wantErr  bool
    }{
        "valid spec exists": {specName: "001", wantErr: false},
        "missing spec":      {specName: "999", wantErr: true},
    }

    for name, tc := range tests {
        t.Run(name, func(t *testing.T) {
            t.Parallel()
            err := ValidateSpec(tc.specName)
            if (err != nil) != tc.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tc.wantErr)
            }
        })
    }
}
```

#### CLI Testing with testscript
```
# testdata/scripts/validate.txt
exec autospec validate 001
stdout 'Validation passed'
! stderr .

! exec autospec validate 999
stderr 'spec.md not found'
```

#### Mocking External Commands
Use TestMain hijacking pattern:
```go
func TestMain(m *testing.M) {
    behavior := os.Getenv("TEST_MOCK_BEHAVIOR")
    switch behavior {
    case "":
        os.Exit(m.Run())
    case "claudeSuccess":
        mockClaudeSuccess()
    case "specifySuccess":
        mockSpecifySuccess()
    }
}

func TestRunClaude(t *testing.T) {
    testExe, _ := os.Executable()
    commander := &Commander{ClaudeExe: testExe}
    t.Setenv("TEST_MOCK_BEHAVIOR", "claudeSuccess")
    output, err := commander.RunClaude("--format", "json")
    require.NoError(t, err)
}
```

#### Mocking File System
- Read operations: `testing/fstest.MapFS`
- Write operations: `t.TempDir()` (auto-cleanup)

#### Performance Benchmarking
```go
func BenchmarkValidateSpec(b *testing.B) {
    tmpDir := b.TempDir()
    createTestSpec(tmpDir, "001")

    for b.Loop() { // Go 1.24+ automatic timing
        _ = ValidateSpec("001")
    }
}
```

Run with: `go test -bench=. -benchmem -count=10`
Compare with: `benchstat old.txt new.txt`

### Test Coverage
- Built-in: `go test -cover ./...`
- HTML report: `go test -coverprofile=cover.out && go tool cover -html=cover.out`

### Libraries
**Essential:**
- `testing` (stdlib)
- `github.com/stretchr/testify/assert`
- `github.com/stretchr/testify/require`
- `github.com/rogpeppe/go-internal/testscript`

**Optional:**
- `golang.org/x/tools/cmd/benchstat` (statistical benchmark comparison)
- `testing/fstest` (stdlib, file system mocking)

---

## 5. Additional Best Practices

### Binary Size Optimization
- Use `go build -ldflags="-s -w"` to strip debug info
- Target: <15 MB (all frameworks meet this)
- Baseline Go binary: ~2 MB
- With Cobra + Koanf + testify: ~4-5 MB

### Performance Monitoring
- Startup time: Measured with `time ./autospec --version`
- Validation time: Benchmarks with `go test -bench=BenchmarkValidate`
- Targets: Startup <50ms, validation <100ms, status <1s

### Cross-Platform Builds
```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o autospec-linux-amd64
GOOS=linux GOARCH=arm64 go build -o autospec-linux-arm64

# macOS
GOOS=darwin GOARCH=amd64 go build -o autospec-darwin-amd64
GOOS=darwin GOARCH=arm64 go build -o autospec-darwin-arm64

# Windows
GOOS=windows GOARCH=amd64 go build -o autospec-windows-amd64.exe
```

### Error Handling
- Use standardized exit codes (0=success, 1=failed, 2=exhausted, 3=invalid, 4=missing deps)
- Provide actionable error messages
- Gracefully handle missing dependencies (claude, specify, git)

---

## Summary of Decisions

| Component | Choice | Binary Overhead | Rationale |
|-----------|--------|----------------|-----------|
| CLI Framework | Cobra | 3.4 MB | Industry standard, battle-tested, extensive features |
| Configuration | Koanf + validator | 334 KB | Lightweight, modular, preserves JSON spec |
| Git Operations | os/exec | 0 KB | Simple, fast, no dependencies |
| Testing | testing + testify + testscript | Dev-only | Idiomatic Go, achieves 60+ tests, CLI-specific DSL |

**Total Binary Size Estimate**: 4-5 MB (well under 15 MB requirement)
**Performance**: All choices support <50ms startup, <1s validation targets
**Maintenance**: All choices have active development and community support

---

## References

1. [Cobra vs Kong vs urfave/cli Comparison](https://github.com/gschauer/go-cli-comparison)
2. [Koanf vs Viper Wiki](https://github.com/knadh/koanf/wiki/Comparison-with-spf13-viper)
3. [go-git Security Advisories](https://github.com/go-git/go-git/security/advisories)
4. [Go Wiki: TableDrivenTests](https://go.dev/wiki/TableDrivenTests)
5. [testscript Documentation](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript)
6. [Mock programs to test os/exec](https://abhinavg.net/2022/05/15/hijack-testmain/)
7. [Benchmarking in Go Guide](https://betterstack.com/community/guides/scaling-go/golang-benchmarking/)
