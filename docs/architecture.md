# Architecture Overview

Deep dive into autospec's system design, component structure, and execution patterns.

## Table of Contents

- [Component Overview](#component-overview)
- [System Architecture](#system-architecture)
- [Package Structure](#package-structure)
- [Execution Flow](#execution-flow)
- [Key Patterns](#key-patterns)
- [Integration Points](#integration-points)

## Component Overview

autospec is built as a modular Go application with clear separation of concerns across several packages:

```mermaid
graph TB
    CLI[CLI Layer<br/>internal/cli] --> Workflow[Workflow Orchestration<br/>internal/workflow]
    CLI --> Config[Configuration<br/>internal/config]
    Workflow --> Executor[Phase Executor<br/>internal/workflow]
    Workflow --> Validation[Validation<br/>internal/validation]
    Workflow --> Retry[Retry Management<br/>internal/retry]
    Executor --> Claude[Claude Integration<br/>internal/workflow]
    CLI --> Health[Health Checks<br/>internal/health]
    CLI --> Spec[Spec Detection<br/>internal/spec]
    CLI --> Git[Git Integration<br/>internal/git]
    Workflow --> Progress[Progress Display<br/>internal/progress]

    classDef primary fill:#e1f5ff,stroke:#0066cc,stroke-width:2px
    classDef secondary fill:#f0f0f0,stroke:#666,stroke-width:1px

    class CLI,Workflow,Executor primary
    class Config,Validation,Retry,Claude,Health,Spec,Git,Progress secondary
```

### 1. CLI Layer (internal/cli/)

User-facing Cobra commands (internal/cli/root.go:1): Parse args, load config, invoke workflows, display results

### 2. Workflow Orchestration (internal/workflow/)

Multi-phase execution with validation and retry (internal/workflow/workflow.go:1): Execute phases in order, validate outputs, retry on failure

### 3. Configuration (internal/config/)

Hierarchical config loading with koanf (internal/config/config.go:1): Priority Env > Local > Global > Defaults

### 4. Validation (internal/validation/)

Fast validation (<10ms) for artifacts (internal/validation/validation.go:1): Validate file existence, parse tasks, generate prompts

### 5. Retry Management (internal/retry/)

Persistent retry state (internal/retry/retry.go:1): Track retries per spec:phase in `~/.autospec/state/retry.json`

### 6. Spec Detection (internal/spec/)

Auto-detect current spec (internal/spec/spec.go:1): Check git branch pattern or most recent specs/ directory

### 7. Git Integration (internal/git/)

Git helpers (internal/git/git.go:1): Check repo status, get branch name

### 8. Health Checks (internal/health/)

Dependency verification (internal/health/health.go:1): Verify Claude CLI, check directory access, validate config

### 9. Progress Display (internal/progress/)

Real-time feedback (internal/progress/display.go:1): Spinner indicators for long-running operations

## System Architecture

High-level architecture showing data flow and component interactions:

```mermaid
flowchart TB
    User[User] -->|Command + Args| CLI[CLI Layer]
    CLI -->|Load| Config[Configuration]
    Config -->|Settings| CLI
    CLI -->|Execute| Orchestrator[Workflow Orchestrator]

    Orchestrator -->|Run Checks| Preflight[Preflight Checks]
    Preflight -->|Verify| Health[Health Checks]
    Health -->|Status| Preflight
    Preflight -->|OK/Fail| Orchestrator

    Orchestrator -->|Execute Phase| Executor[Phase Executor]
    Executor -->|Load State| Retry[Retry Manager]
    Retry -->|State| Executor

    Executor -->|Command| Claude[Claude Integration]
    Claude -->|API/CLI| ClaudeSvc[Claude Service]
    ClaudeSvc -->|Response| Claude
    Claude -->|Output| Executor

    Executor -->|Validate| Validation[Validation Functions]
    Validation -->|Parse| Files[Output Files]
    Files -->|Status| Validation
    Validation -->|Pass/Fail| Executor

    Executor -->|Update State| Retry
    Retry -->|Persist| StateFile[~/.autospec/state/retry.json]

    Executor -->|Result| Orchestrator
    Orchestrator -->|Status| CLI
    CLI -->|Output + Exit Code| User

    CLI -->|Detect Spec| SpecDetect[Spec Detection]
    SpecDetect -->|Branch| GitInt[Git Integration]
    GitInt -->|Branch Name| SpecDetect
    SpecDetect -->|Spec Info| CLI

    classDef primary fill:#e1f5ff,stroke:#0066cc,stroke-width:2px
    classDef secondary fill:#f0f0f0,stroke:#666,stroke-width:1px
    classDef external fill:#fff4e6,stroke:#ff9800,stroke-width:2px

    class User,CLI,Orchestrator,Executor primary
    class Config,Preflight,Health,Retry,Validation,SpecDetect,GitInt secondary
    class Claude,ClaudeSvc,Files,StateFile external
```

## Package Structure

Detailed breakdown of internal package organization:

```
internal/
├── cli/                    # Cobra-based CLI commands
│   ├── root.go            # Root command + global flags
│   ├── full.go            # autospec full command
│   ├── prep.go            # autospec prep command
│   ├── specify.go         # autospec specify command
│   ├── plan.go            # autospec plan command
│   ├── tasks.go           # autospec tasks command
│   ├── implement.go       # autospec implement command
│   ├── doctor.go          # autospec doctor command
│   ├── status.go          # autospec status command
│   ├── config.go          # autospec config command
│   ├── init.go            # autospec init command
│   └── version.go         # autospec version command
│
├── workflow/              # Workflow orchestration
│   ├── workflow.go        # WorkflowOrchestrator
│   ├── executor.go        # Executor (phase execution)
│   ├── claude.go          # ClaudeExecutor (API/CLI)
│   └── preflight.go       # PreflightChecks
│
├── config/                # Configuration management
│   ├── config.go          # Load config (koanf)
│   └── defaults.go        # Default values
│
├── validation/            # Validation functions
│   ├── validation.go      # File validation
│   ├── tasks.go           # Task parsing
│   ├── prompt.go          # Prompt generation
│   └── docs_test.go       # Documentation tests
│
├── retry/                 # Retry state management
│   └── retry.go           # RetryState (persistent)
│
├── spec/                  # Spec detection
│   └── spec.go            # DetectCurrentSpec()
│
├── git/                   # Git integration
│   └── git.go             # Git helpers
│
├── health/                # Health checks
│   └── health.go          # Dependency verification
│
└── progress/              # Progress indicators
    └── progress.go        # Spinner display
```

## Execution Flow

### Complete Workflow Execution

Sequence diagram showing full workflow execution with retry logic:

```mermaid
sequenceDiagram
    participant U as User
    participant CLI as CLI Command
    participant O as Orchestrator
    participant E as Executor
    participant C as Claude
    participant V as Validation
    participant R as Retry State

    U->>CLI: autospec full "feature"
    CLI->>O: RunFullWorkflow()

    Note over O: Phase 1: Specify
    O->>E: ExecutePhase(specify)
    E->>R: LoadState(001-feature:specify)
    R-->>E: retries=0, max=3
    E->>C: Execute("/autospec.specify ...")
    C->>C: Call Claude API/CLI
    C-->>E: spec.yaml created
    E->>V: ValidateSpec(spec.yaml)
    alt Validation Pass
        V-->>E: OK
        E->>R: ResetRetries(001-feature:specify)
        E-->>O: Success
    else Validation Fail
        V-->>E: Error
        E->>R: IncrementRetries(001-feature:specify)
        alt Retries Remaining
            E->>C: Retry with continuation prompt
        else Max Retries Exhausted
            E-->>O: Fail (exit code 2)
        end
    end

    Note over O: Phase 2: Plan
    O->>E: ExecutePhase(plan)
    E->>R: LoadState(001-feature:plan)
    E->>C: Execute("/autospec.plan")
    C-->>E: plan.yaml created
    E->>V: ValidatePlan(plan.yaml)
    V-->>E: OK
    E->>R: ResetRetries(001-feature:plan)
    E-->>O: Success

    Note over O: Phase 3: Tasks
    O->>E: ExecutePhase(tasks)
    E->>C: Execute("/autospec.tasks")
    C-->>E: tasks.yaml created
    E->>V: ValidateTasks(tasks.yaml)
    V-->>E: OK
    E-->>O: Success

    Note over O: Phase 4: Implement
    O->>E: ExecutePhase(implement)
    E->>C: Execute("/autospec.implement")
    C-->>E: Implementation complete
    E->>V: ValidateTasksComplete(tasks.yaml)
    V-->>E: OK
    E-->>O: Success

    O-->>CLI: All Phases Complete
    CLI-->>U: Success (exit code 0)
```

### Phase Execution with Retry

Detailed flowchart showing retry logic within a single phase:

```mermaid
flowchart TD
    Start([Start Phase Execution]) --> LoadState[Load Retry State]
    LoadState --> CheckRetries{Retries < Max?}
    CheckRetries -->|Yes| ExecuteCmd[Execute Command]
    CheckRetries -->|No| FailExhausted[Fail: Retries Exhausted]

    ExecuteCmd --> Validate[Validate Output]
    Validate --> ValidationOK{Validation Pass?}

    ValidationOK -->|Yes| ResetRetries[Reset Retry Count]
    ResetRetries --> Success([Success: Return 0])

    ValidationOK -->|No| IncrementRetries[Increment Retry Count]
    IncrementRetries --> SaveState[Save Retry State]
    SaveState --> CheckRetries

    FailExhausted --> ExitCode2([Fail: Exit Code 2])

    style Start fill:#90EE90
    style Success fill:#90EE90
    style FailExhausted fill:#FFB6C1
    style ExitCode2 fill:#FFB6C1
    style ValidationOK fill:#FFE4B5
    style CheckRetries fill:#FFE4B5
```

## Key Patterns

### 1. Retry Pattern

**Implementation**: internal/workflow/executor.go:50

**How it works**:
```go
func (e *Executor) ExecutePhase(specName, phase, command string, validateFn func() error) error {
    state := e.retryManager.LoadState(specName, phase)

    for state.Count < state.MaxRetries {
        err := e.executeCommand(command)
        if err != nil {
            state.Count++
            e.retryManager.SaveState(state)
            continue
        }

        err = validateFn()
        if err == nil {
            e.retryManager.ResetRetries(specName, phase)
            return nil
        }

        state.Count++
        e.retryManager.SaveState(state)
    }

    return ErrRetriesExhausted
}
```

**Benefits**:
- Resilient to transient failures
- Persistent state across executions
- Configurable retry limits
- Clear failure reporting

### 2. Configuration Layering

**Implementation**: internal/config/config.go:1

**Priority Order**:
1. Environment variables (`AUTOSPEC_*`)
2. Local config (`.autospec/config.yml`)
3. Global config (`~/.config/autospec/config.yml`)
4. Defaults (hardcoded)

**Example**:
```bash
# Priority 1: Environment variable
export AUTOSPEC_MAX_RETRIES=5

# Priority 2: Local config
echo 'max_retries: 3' > .autospec/config.yml

# Priority 3: Global config
echo 'max_retries: 2' > ~/.config/autospec/config.yml

# Result: max_retries = 5 (environment wins)
```

### 3. Spec Detection

**Implementation**: internal/spec/spec.go:1

**Strategy**:
```go
func DetectCurrentSpec() (*SpecMetadata, error) {
    // Strategy 1: Git branch name
    branch := git.CurrentBranch()
    if matches := specPattern.FindStringSubmatch(branch); matches != nil {
        return parseSpecFromBranch(matches), nil
    }

    // Strategy 2: Most recent specs/ directory
    dirs := findSpecDirs("./specs")
    mostRecent := sortByModTime(dirs)[0]
    return parseSpecFromDir(mostRecent), nil
}
```

**Benefits**:
- Zero friction for users
- Works with git workflows
- Falls back to directory scan

### 4. Exit Code Conventions

**Implementation**: All commands follow consistent exit codes

| Code | Meaning | When Used |
|------|---------|-----------|
| 0 | Success | All operations completed successfully |
| 1 | Validation failed | Output artifact validation failed (retryable) |
| 2 | Retry limit exhausted | Max retries reached without success |
| 3 | Invalid arguments | User provided invalid command arguments |
| 4 | Missing dependencies | Required dependencies (claude, git) not found |
| 5 | Command timeout | Operation exceeded configured timeout |

**Benefits**:
- Scriptable workflows
- CI/CD integration
- Clear error identification

## Integration Points

### Claude Integration

**Methods**:
1. **CLI Mode** (default): Execute `claude` command via shell
2. **API Mode**: Direct API calls using API key
3. **Custom Mode**: User-defined command with `{{PROMPT}}` placeholder

**Configuration**:
```yaml
claude_cmd: claude
custom_claude_cmd: "claude -p {{PROMPT}} | process-output"
use_api_key: false
```

**Prompt Injection**:
All phase commands support optional guidance text:
```bash
autospec plan "Focus on security best practices"
# Executes: claude -p "/autospec.plan \"Focus on security best practices\""
```

### File System

**Directories**:
- `./specs/NNN-feature-name/`: Feature specifications and artifacts
- `~/.config/autospec/`: Global configuration (XDG compliant)
- `~/.autospec/state/`: State directory
- `.autospec/`: Local project configuration

**Files**:
- `spec.yaml`: Feature specification
- `plan.yaml`: Technical plan
- `tasks.yaml`: Task breakdown
- `~/.config/autospec/config.yml`: Global configuration
- `.autospec/config.yml`: Local configuration
- `~/.autospec/state/retry.json`: Retry state

### External Tools

**Required**:
- Claude CLI: For workflow execution
- Git (optional): For branch-based spec detection

## Performance Characteristics

**Design Goals**:
- Sub-second validation checks (<1s)
- Validation functions <10ms
- Minimal memory footprint
- Fast startup time

**Benchmarks**:
- File validation: ~1-2ms (internal/validation/validation_bench_test.go)
- Task parsing: ~5-8ms (internal/validation/tasks_test.go)
- Config loading: ~10-15ms (internal/config/config.go)

## Error Handling

**Philosophy**: Fail fast, fail clearly, provide actionable guidance

**Patterns**:
1. **Validation Errors**: Include file paths, expected vs actual, retry suggestions
2. **Dependency Errors**: Include installation/fix instructions
3. **Timeout Errors**: Include timeout duration, config hints
4. **Retry Exhausted**: Include last error, retry count, state file location

**Example**:
```
Error: Validation failed: spec file not found
Expected: specs/001-dark-mode/spec.yaml
Retry: 2/3 (state: ~/.autospec/state/retry.json)
Hint: Run 'autospec specify "feature description"' to create spec
```

## Extension Points

Areas designed for future extension:

1. **Custom Validators**: Add new validation functions in internal/validation/
2. **Additional Commands**: Add new CLI commands in internal/cli/
3. **Alternative Executors**: Implement ClaudeExecutor interface for new backends
4. **Custom Health Checks**: Extend health check framework
5. **Progress Reporters**: Implement alternative progress display formats

## Further Reading

- **[Quick Start Guide](./quickstart.md)**: Get started with basic usage
- **[Command Reference](./reference.md)**: Complete command and configuration documentation
- **[Troubleshooting](./troubleshooting.md)**: Common issues and solutions
- **[CLAUDE.md](../CLAUDE.md)**: Detailed development guidelines for contributors
