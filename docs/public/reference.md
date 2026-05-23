# Command Reference

Complete reference for autospec commands, configuration options, exit codes, and file locations.

## CLI Commands

All commands support global flags: `--config`, `--specs-dir`, `--debug`, `--verbose`, `--output-style`

- `--output-style <style>`: Output formatting style (`default`, `compact`, `minimal`, `plain`, `raw`)

### autospec all

Execute complete workflow: specify → plan → tasks → implement

**Syntax**: `autospec all "<feature description>" [flags]`

**Description**: Creates specification, generates plan and tasks, then executes implementation in a single command.

**Flags**:
- `--skip-preflight`: Skip dependency health checks
- `--timeout <seconds>`: Command timeout (0=infinite, 1-604800)
- `--max-retries <count>`: Maximum retry attempts (0-10, default: 0)
- `--agent <name>`: Override agent for this run (see [CLI Agents](#cli-agents))
- `--model <model>`: Override the workflow agent model for this run
- `--opencode-agent <name>`: OpenCode sub-agent to use (e.g., `Build`, `Plan`); see [OpenCode Agents](https://opencode.ai/docs/agents)
- `--auto-commit`: Enable automatic git commit after workflow completion
- `--no-auto-commit`: Disable automatic git commit (overrides config)

**Examples**:
```bash
autospec all "Add user authentication with OAuth"
autospec all "Add dark mode toggle" --timeout 600
autospec all "Export data to CSV" --skip-preflight
autospec all "Add caching" --agent gemini

# With auto-commit enabled
autospec all "Add feature" --auto-commit

# With auto-commit disabled (overrides config)
autospec all "Add feature" --no-auto-commit
```

**Exit Codes**: 0 (success), 1 (validation failed), 2 (retries exhausted), 3 (invalid args), 4 (missing deps), 5 (timeout)

### autospec prep

Prepare for implementation: specify → plan → tasks (no implementation)

**Syntax**: `autospec prep "<feature description>" [flags]`

**Description**: Creates specification and generates plan/tasks for review before implementation.

**Flags**: Same as `autospec all` (including `--auto-commit` and `--no-auto-commit`)

**Examples**:
```bash
autospec prep "Add user profile page"
autospec prep "Implement caching layer" --max-retries 5
autospec prep "Add payments" --auto-commit
```

**Exit Codes**: 0 (success), 1 (validation failed), 2 (retries exhausted), 3 (invalid args), 4 (missing deps), 5 (timeout)

### autospec run

Run selected workflow stages with flexible stage selection

**Syntax**: `autospec run [feature-description] [flags]`

**Description**: Flexible workflow command that lets you pick any combination of stages to run. Stages always execute in canonical order regardless of flag order.

**Core Stage Flags**:
- `-s, --specify`: Include specify stage (requires feature description)
- `-p, --plan`: Include plan stage
- `-t, --tasks`: Include tasks stage
- `-i, --implement`: Include implement stage
- `-a, --all`: Run all core stages (equivalent to `-spti`)

**Optional Stage Flags**:
- `-n, --constitution`: Include constitution stage
- `-r, --clarify`: Include clarify stage
- `-l, --checklist`: Include checklist stage
- `-z, --analyze`: Include analyze stage

**Other Flags**:
- `--spec <name>`: Target a specific spec for this invocation. This explicit selection overrides persisted active feature state and branch detection.
- `-y, --yes`: Skip confirmation prompts
- `--resume`: Resume implementation from where it left off
- `--dry-run`: Preview what stages would run without executing
- `--max-retries <count>`: Override max retry attempts
- `--agent <name>`: Override agent for this run
- `--model <model>`: Override the workflow agent model for this run
- `--opencode-agent <name>`: OpenCode sub-agent to use (e.g., `Build`, `Plan`)
- `--auto-commit` / `--no-auto-commit`: Override auto-commit config

**Canonical Stage Order**: constitution → specify → clarify → plan → tasks → checklist → analyze → implement

**Feature Selection**: When a workflow command needs an existing feature directory, autospec resolves it in this order: explicit command selection (`--spec` or positional spec argument), persisted project-local active feature state, then branch-prefix fallback. Commands that create or select a feature may update the persisted active feature so later `plan`, `tasks`, `implement`, `status`, `prereqs`, and artifact lookups can target the same directory from a differently named branch. If persisted state points at a deleted spec directory, autospec ignores that stale selection and continues to branch-prefix fallback.

**Examples**:
```bash
# Run all core stages for a new feature
autospec run -a "Add user authentication"

# Run only plan and implement on current spec
autospec run -pi

# Run tasks and implement on a specific spec
autospec run -ti --spec 007-yaml-output

# Preview what stages would run (dry run mode)
autospec run -ti --dry-run

# Skip confirmation prompts for CI/CD
autospec run -ti -y

# Include optional stages
autospec run -a -n "Add feature"         # constitution + all core stages
autospec run -pi -l                      # plan + checklist + implement
autospec run -s -r "Add feature"         # specify + clarify
```

**Exit Codes**: 0 (success), 1 (validation failed), 2 (retries exhausted), 3 (invalid args), 4 (missing deps), 5 (timeout)

### autospec specify

Create feature specification from natural language description

**Syntax**: `autospec specify "<feature description>" ["<guidance>"] [flags]`

**Alias**: `autospec spec`, `autospec s`

**Description**: Generate detailed specification with requirements, acceptance criteria, and success metrics.

**Flags**: Same as `autospec all` (including `--auto-commit` and `--no-auto-commit`)

**Examples**:
```bash
autospec specify "Add real-time notifications"
autospec specify "Add API rate limiting" "Focus on security"
autospec specify "Add webhooks" --auto-commit
```

**Exit Codes**: 0 (success), 1 (validation failed), 2 (retries exhausted), 3 (invalid args), 4 (missing deps), 5 (timeout)

### autospec plan

Generate technical implementation plan from specification

**Syntax**: `autospec plan ["<guidance>"] [flags]`

**Alias**: `autospec p`

**Description**: Create technical plan with architecture, file structure, and design decisions.

**Flags**: Same as `autospec all` (including `--auto-commit` and `--no-auto-commit`)

When no explicit spec is provided, `autospec plan` uses the persisted active feature if one exists; otherwise it uses branch-prefix fallback.

**Examples**:
```bash
autospec plan
autospec plan "Prioritize performance and scalability"
autospec plan --timeout 300
autospec plan --auto-commit
```

**Exit Codes**: 0 (success), 1 (validation failed), 2 (retries exhausted), 3 (invalid args), 4 (missing deps), 5 (timeout)

### autospec tasks

Generate task breakdown from implementation plan

**Syntax**: `autospec tasks ["<guidance>"] [flags]`

**Alias**: `autospec t`

**Description**: Break down plan into ordered, actionable tasks with dependencies.

**Flags**: Same as `autospec all` (including `--auto-commit` and `--no-auto-commit`)

When no explicit spec is provided, `autospec tasks` uses the persisted active feature if one exists; otherwise it uses branch-prefix fallback.

**Examples**:
```bash
autospec tasks
autospec tasks "Break into small incremental steps"
autospec tasks --auto-commit
```

**Exit Codes**: 0 (success), 1 (validation failed), 2 (retries exhausted), 3 (invalid args), 4 (missing deps), 5 (timeout)

### autospec implement

Execute implementation phase using tasks breakdown

**Syntax**: `autospec implement [<spec-name>] ["<guidance>"] [flags]`

**Alias**: `autospec impl`, `autospec i`

**Description**: Execute tasks with the configured agent, validating progress. Supports multiple execution modes for context isolation.

**Flags**:
- `--phases`: Run each phase in a separate agent session (fresh context per phase)
- `--phase <N>`: Run only the specified phase number
- `--from-phase <N>`: Run phases N and onwards, each in separate session
- `--tasks`: Run each task in a separate agent session (maximum context isolation)
- `--from-task <ID>`: Resume from specific task ID
- `--resume`: Resume implementation from where it left off
- `--single-session`: Run all tasks in one agent session (legacy mode)
- `--auto-commit`: Enable automatic git commit after workflow completion
- `--no-auto-commit`: Disable automatic git commit (overrides config)
- `--agent <name>`: Override agent for this run
- `--opencode-agent <name>`: OpenCode sub-agent to use (e.g., `Build`, `Plan`)
- Plus all flags from `autospec all`

When no positional spec is provided, `autospec implement` uses the persisted active feature if one exists; otherwise it uses branch-prefix fallback. If persisted state points at a deleted spec directory, branch-prefix fallback is used. A positional spec argument is explicit selection and takes precedence over persisted state.

**Execution Modes**:

| Mode | Flag | Sessions | Use Case |
|------|------|----------|----------|
| Phase-level | (default) | 1 per phase | Balanced cost/context |
| Task-level | `--tasks` | 1 per task | Large specs, maximum isolation |
| Single-session | `--single-session` | 1 | Small specs, quick iterations |

**Examples**:
```bash
# Default: phase-level isolation (1 session per phase)
autospec implement
autospec implement 001-dark-mode
autospec implement --phase 2             # Run only phase 2
autospec implement --from-phase 3        # Run phases 3+ sequentially

# Task-level isolation (maximum granularity)
autospec implement --tasks               # Each task in separate session
autospec implement --from-task T005      # Resume from task T005

# Single-session (all tasks in one session)
autospec implement --single-session

# With guidance
autospec implement --phases "Focus on tests first"
```

**Exit Codes**: 0 (success), 1 (validation failed), 2 (retries exhausted), 3 (invalid args), 4 (missing deps), 5 (timeout)

### autospec constitution

Create or update the project constitution

**Syntax**: `autospec constitution [optional-prompt] [flags]`

**Alias**: `autospec const`

**Description**: Generate or update `.autospec/constitution.yaml` which defines project principles and guidelines. Required before running any other workflow stage.

**Flags**:
- `--max-retries <count>`: Override max retry attempts

**Examples**:
```bash
autospec constitution
autospec constitution "Focus on test-driven development"
```

**Exit Codes**: 0 (success), 1 (validation failed), 3 (invalid args)

### autospec clarify

Refine the specification by asking clarification questions

**Syntax**: `autospec clarify [optional-prompt] [flags]`

**Alias**: `autospec cl`

**Description**: Identify underspecified areas in the current spec and encode clarifications back into `spec.yaml`.

**Flags**: Global flags only.

**Examples**:
```bash
autospec clarify
autospec clarify "Focus on edge cases in auth flow"
```

**Exit Codes**: 0 (success), 1 (validation failed), 3 (invalid args)

### autospec checklist

Generate a quality validation checklist

**Syntax**: `autospec checklist [optional-prompt] [flags]`

**Alias**: `autospec chk`

**Description**: Generate a YAML checklist for validating feature quality based on the current spec.

**Flags**:
- `--max-retries <count>`: Override max retry attempts

**Examples**:
```bash
autospec checklist
autospec checklist "Include accessibility checks"
```

**Exit Codes**: 0 (success), 1 (validation failed), 3 (invalid args)

### autospec analyze

Perform cross-artifact consistency analysis

**Syntax**: `autospec analyze [optional-prompt] [flags]`

**Alias**: `autospec az`

**Description**: Analyze consistency and quality across spec.yaml, plan.yaml, and tasks.yaml artifacts.

**Flags**: Global flags only.

**Examples**:
```bash
autospec analyze
autospec analyze "Check for gaps in test coverage plan"
```

**Exit Codes**: 0 (success), 1 (validation failed), 3 (invalid args)

### autospec doctor

Run health checks and verify dependencies

**Syntax**: `autospec doctor [flags]`

**Alias**: `autospec doc`

**Description**: Verify configured CLI agents, authentication/configuration, and directories are accessible. When `.autospec/init.yml` indicates global scope was used during init, doctor checks global agent settings instead of project-level ones.

**Flags**: None (uses global flags only)

**Examples**:
```bash
autospec doctor
autospec doctor --debug
```

**Exit Codes**: 0 (all checks passed), 4 (dependencies missing)

### autospec history

View command execution history

**Syntax**: `autospec history [flags]`

**Description**: Display a log of all autospec command executions with timestamp, unique ID, status, command name, spec, exit code, and duration.

**Automatic Logging**: All workflow commands are automatically logged to history:
- Core stages: `specify`, `plan`, `tasks`, `implement`
- Optional stages: `clarify`, `analyze`, `checklist`, `constitution`
- Workflows: `run`, `prep`, `all`

**Two-Phase Logging**: History entries are written **immediately when commands start** (with status `running`) and updated when commands complete. This ensures:
- Running commands are visible in history
- No history data is lost if a command crashes or is interrupted
- Each entry has a unique, memorable ID for tracking

**Flags**:
- `-s, --spec <name>`: Filter by spec name
- `-n, --limit <count>`: Limit to last N entries (most recent)
- `--status <value>`: Filter by status (`running`, `completed`, `failed`, `cancelled`)
- `--clear`: Clear all history

**Output Format**:
```
TIMESTAMP            ID                              STATUS      COMMAND       SPEC              EXIT  DURATION
2024-01-15 10:30:00  brave_fox_20240115_103000       completed   specify       -                 0     2m30s
2024-01-15 10:35:00  calm_river_20240115_103500      completed   plan          001-test-feature  0     1m15s
2024-01-15 10:40:00  swift_falcon_20240115_104000    failed      tasks         001-test-feature  1     45s
2024-01-15 10:45:00  gentle_owl_20240115_104500      running     implement     001-test-feature  0
```

**Columns**:
- **ID**: Unique identifier in `adjective_noun_YYYYMMDD_HHMMSS` format (memorable and sortable)
- **STATUS**: Current state with color coding:
  - Green: `completed` (successful execution)
  - Yellow: `running` (currently executing)
  - Red: `failed` (error occurred) or `cancelled` (user interrupted)
  - `-`: Old entries without status (backward compatibility)

Note: Commands that create new specs (`specify`, `prep`, `all`, `run -s`) log with an empty spec name since the spec doesn't exist yet when the command starts.

**Examples**:
```bash
# View all history
autospec history

# View last 10 entries
autospec history -n 10

# Filter by spec name
autospec history --spec 001-feature

# Filter by status (see running commands)
autospec history --status running

# Filter by failed commands
autospec history --status failed

# Combine filters
autospec history --spec 001-feature --status completed

# Clear all history
autospec history --clear
```

**Exit Codes**: 0 (success), 3 (invalid arguments, e.g., negative limit)

**File Location**: `~/.autospec/state/history.yaml`

**Storage Limit**: History is automatically pruned to `max_history_entries` (default: 500). Oldest entries are removed first when the limit is exceeded. See [Configuration](#max_history_entries) to customize.

### autospec status

Check current feature status and progress

**Syntax**: `autospec status [spec-name] [flags]`

**Alias**: `autospec st`

**Description**: Display detected spec, which artifact files exist (spec.yaml, plan.yaml, tasks.yaml), task completion progress, and risk summary (if plan.yaml contains risks).

Without a `spec-name`, status reports the currently resolved active feature. Resolution uses persisted project-local active feature state before falling back to the current branch prefix. A `spec-name` argument is explicit selection and overrides persisted state.

**Flags**:
- `-v, --verbose`: Show phase-by-phase breakdown

**Examples**:
```bash
autospec status              # Current spec status
autospec st                  # Short alias
autospec st -v               # Verbose with phase details
autospec status 003-feature  # Specific spec
```

**Output**:
```
015-artifact-validation
  artifacts: [spec.yaml plan.yaml tasks.yaml]
  risks: 3 total (1 high, 2 medium)
  25/38 tasks completed (66%)
  7/10 task phases completed
  (1 in progress)
```

**Exit Codes**: 0 (success), 3 (invalid args)

### autospec view

Display dashboard overview of all specs in the project

**Syntax**: `autospec view [flags]`

**Description**: Shows project-wide spec statistics, recent specs with task progress, and completed specs in a single dashboard view.

**Flags**:
- `-l, --limit <count>`: Number of recent specs to display (default: from config or 5)

**Output Sections**:
1. **Dashboard Header**: Total specs, in-progress count, completed count, skipped count
2. **Recent Specs**: Top N most recently modified specs with status and task progress
3. **Completed Specs**: All specs with Completed status or 100% task completion

**Examples**:
```bash
autospec view                  # Show dashboard with default limit (5)
autospec view --limit 10       # Show top 10 recent specs
autospec view -l 3             # Short flag for limit
```

**Output**:
```
Spec Dashboard
----------------------------------------
Total specs:   48
In progress:   10
Completed:     37
Skipped:       1

Recent Specs (top 5)
----------------------------------------
  063-view-dashboard             Draft
    Progress: 4/18 tasks
  058-config-set-command         Completed
    Progress: 18/18 tasks
  057-fix-description-propaga... Completed
    Progress: 10/10 tasks

Completed Specs
----------------------------------------
  058-config-set-command         18/18 tasks
  057-fix-description-propaga... 10/10 tasks
```

**Status Categories**:
- **In Progress**: Draft, In Progress, Review, or any non-completed/non-skipped status
- **Completed**: Completed status OR 100% task completion
- **Skipped**: Rejected or Skipped status

**Exit Codes**: 0 (success)

### autospec config

Manage configuration settings

**Syntax**: `autospec config <subcommand> [flags]`

**Subcommands**:
- `show`: Display current configuration
- `set <key> <value>`: Set configuration value
- `get <key>`: Get configuration value
- `toggle <key>`: Toggle boolean configuration value
- `keys`: List all available configuration keys
- `sync`: Sync configuration with current schema (adds new options, removes deprecated)

**Examples**:
```bash
autospec config show
autospec config set max_retries 5
autospec config get timeout
autospec config toggle notifications.enabled
autospec config keys
autospec config sync --dry-run    # Preview changes
autospec config sync              # Apply changes
autospec config sync --project    # Sync project config
```

**Note**: Configuration is automatically synced when running `autospec update`. New configuration options are added with their default values, and deprecated options are removed.

**Exit Codes**: 0 (success), 3 (invalid args)

### autospec init

Initialize configuration files and directories

**Syntax**: `autospec init [path] [flags]`

**Description**: Set up autospec with everything needed to get started:
1. Installs agent-specific skills for selected agents
2. Creates configuration at `~/.config/autospec/config.yml`
3. Creates `.autospec/init.yml` to track initialization settings (scope, agent, version)
4. Creates `.autospec/.gitignore` for local runtime files
5. Prompts for agent selection and configuration
6. Optionally creates project constitution
7. Optionally generates worktree setup script

If config already exists, it is left unchanged (use `--force` to overwrite).

**Path Argument**: If provided, initializes the project at the specified path instead of the current directory:
- **Relative paths**: resolved against current directory (e.g., `my-project`)
- **Absolute paths**: used as-is (e.g., `/home/user/project`)
- **Tilde paths**: expanded to home directory (e.g., `~/projects/new`)
- **Non-existent paths**: created automatically with standard permissions

**Flags**:
- `--project, -p`: Create project-level config (`.autospec/config.yml`)
- `--force, -f`: Overwrite existing configuration with defaults
- `--no-agents`: Skip agent configuration prompt (for non-interactive environments)
- `--here`: Initialize in current directory (same as `init .`)
- `--ai <agents>`: Configure specific agent(s), comma-separated (e.g., `--ai claude,codex,opencode`)

**Non-Interactive Flags** (for CI/CD and automation):

| Flag | Positive | Negative | Effect |
|------|----------|----------|--------|
| Sandbox | `--sandbox` | `--no-sandbox` | Enable/skip Claude sandbox configuration |
| Billing | `--use-subscription` | `--no-use-subscription` | Use subscription billing vs API key |
| Permissions | `--skip-permissions` | `--no-skip-permissions` | Enable/disable autonomous mode |
| Gitignore | `--gitignore` | `--no-gitignore` | Add/skip adding .autospec/ to root .gitignore |
| Constitution | `--constitution` | `--no-constitution` | Create/skip project constitution |

**Mutual Exclusivity**: Each positive/negative flag pair is mutually exclusive. Using both (e.g., `--sandbox --no-sandbox`) returns an error:
```
Error: flags --sandbox and --no-sandbox are mutually exclusive
```

**Non-Interactive Mode**: When running without a TTY (e.g., in CI/CD), init validates that all required flags are provided. If flags are missing, an error lists which ones are needed:
```
Error: non-interactive mode requires all prompt flags to be set

Missing flags (use positive or negative form):
  - sandbox configuration: --sandbox or --no-sandbox
  - billing preference: --use-subscription or --no-use-subscription
  - permissions mode: --skip-permissions or --no-skip-permissions
  - gitignore modification: --gitignore or --no-gitignore
  - constitution creation: --constitution or --no-constitution
```

**Agent Selection**: During initialization, you'll be prompted to select which CLI agents to configure. If you select more than one agent, init prompts for the default execution agent and saves it to `agent_preset`. Claude installs project skills under `.claude/skills/autospec.*/` so existing `/autospec.specify`-style invocations continue to work. OpenCode installs shared skills under `.agents/skills/` and configures `opencode.json` permissions; autospec workflow runs send rendered prompt text through `opencode run`. Codex records project metadata in `.codex/config.toml` and registers shared skills under `.agents/skills/`. Your selections are saved to `default_agents` in config to pre-select checkboxes in future `autospec init` runs.

> **Note**: `default_agents` remembers init prompt selections. `agent_preset` controls which agent actually runs commands and defaults to `claude` when empty. See `docs/public/agents.md` for details.

**Examples**:
```bash
# Interactive mode
autospec init                        # Interactive setup in current directory
autospec init /path/to/project       # Initialize at specific absolute path
autospec init ~/projects/my-app      # Initialize with tilde expansion
autospec init my-new-project         # Initialize at relative path (creates if needed)
autospec init .                      # Explicitly initialize in current directory
autospec init --here                 # Same as init .
autospec init --project              # Create project-level config
autospec init --force                # Overwrite existing config with defaults
autospec init /path/to/project --project  # Path + project config

# Non-interactive mode (CI/CD friendly)
autospec init --no-agents            # Skip agent prompts

# Fully non-interactive CI/CD setup (all prompts bypassed)
autospec init --ai claude \
  --sandbox \
  --no-use-subscription \
  --skip-permissions \
  --gitignore \
  --constitution

# Codex setup
autospec init --ai codex \
  --no-sandbox \
  --skip-permissions \
  --gitignore \
  --constitution

# Minimal non-interactive setup (skip optional features)
autospec init --ai claude \
  --no-sandbox \
  --no-use-subscription \
  --no-skip-permissions \
  --no-gitignore \
  --no-constitution

# Production-ready setup with subscription billing
autospec init --ai claude \
  --sandbox \
  --use-subscription \
  --skip-permissions \
  --gitignore \
  --constitution
```

**Working Directory**: When a path is provided, autospec changes to that directory for initialization and then restores the original working directory when complete. All operations (constitution workflow, agent configuration) operate on the specified path.

**Exit Codes**: 0 (success), 3 (invalid args - e.g., path is a file)

### autospec update-agent-context

Update AI agent context files with technology information from plan.yaml

**Syntax**: `autospec update-agent-context [flags]`

**Description**: Updates AI agent context files (CLAUDE.md, GEMINI.md, etc.) with technology information extracted from the current feature's plan.yaml file. Updates the Active Technologies and Recent Changes sections.

**Flags**:
- `--agent <name>`: Update only the specified agent's context file (e.g., claude, gemini, copilot, cursor)
- `--json`: Output results as JSON for programmatic consumption

**Supported Agents**: claude, gemini, copilot, cursor, qwen, opencode, codex, windsurf, kilocode, auggie, roo, codebuddy, qoder, amp, shai, q, bob

**Examples**:
```bash
autospec update-agent-context                    # Update all existing agent files
autospec update-agent-context --agent claude     # Update only CLAUDE.md
autospec update-agent-context --agent cursor     # Create/update Cursor context file
autospec update-agent-context --json             # JSON output for integration
```

**Exit Codes**: 0 (success), 1 (validation failed), 3 (invalid args)

### autospec artifact

Validate YAML artifacts against their schemas

**Syntax**: `autospec artifact <path>` or `autospec artifact <type> <path>`

**Description**: Validates artifacts against their schemas, checking required fields, types, enums, and cross-references (e.g., task dependencies).

Path-based validation uses the path you provide. Type-only lookup, such as printing a schema or resolving a current artifact type without a path, uses the same active feature resolution order as workflow commands: explicit selection, persisted active feature, then branch-prefix fallback.

**Supported Types**:
- `spec` - Feature specification (spec.yaml)
- `plan` - Implementation plan (plan.yaml)
- `tasks` - Task breakdown (tasks.yaml)
- `analysis` - Cross-artifact analysis (analysis.yaml)
- `checklist` - Feature quality checklist (checklists/*.yaml)
- `constitution` - Project constitution (constitution.yaml)

**Flags**:
- `--schema` - Print the expected schema for an artifact type
- `--fix` - Auto-fix common issues (missing optional fields, formatting)

**Examples**:
```bash
# Path-only (preferred) - type inferred from filename
autospec artifact specs/001-feature/spec.yaml
autospec artifact specs/001-feature/plan.yaml
autospec artifact specs/001-feature/tasks.yaml
autospec artifact .autospec/constitution.yaml

# Checklist requires explicit type (filename varies)
autospec artifact checklist specs/001-feature/checklists/ux.yaml

# Show schema
autospec artifact spec --schema

# Auto-fix issues
autospec artifact specs/001-feature/plan.yaml --fix
```

**Exit Codes**: 0 (valid), 1 (validation failed), 3 (invalid args)

### autospec yaml check

Validate YAML syntax

**Syntax**: `autospec yaml check <file>`

**Description**: Quick syntax validation without schema checking. Use `autospec artifact` for full schema validation.

**Examples**:
```bash
autospec yaml check specs/001-feature/spec.yaml
```

**Exit Codes**: 0 (valid syntax), 1 (syntax error)

### autospec render-command

Render a command template with current feature context

**Syntax**: `autospec render-command <command-name> [flags]`

**Description**: Preview autospec slash command templates with pre-computed feature context. Useful for debugging, verifying context detection, and piping rendered prompts to external tools.

**Flags**:
- `-o, --output <file>`: Output file path (default: stdout)

**Available Commands**: `autospec.specify`, `autospec.plan`, `autospec.tasks`, `autospec.implement`, `autospec.checklist`, `autospec.clarify`, `autospec.analyze`, `autospec.constitution`, `autospec.worktree-setup`

**Examples**:
```bash
# Preview the plan command for current feature
autospec render-command autospec.plan

# Save rendered command to a file
autospec render-command autospec.tasks --output /tmp/tasks-prompt.md

# Pipe to clipboard (macOS)
autospec render-command autospec.implement | pbcopy
```

**Exit Codes**: 0 (success), 1 (render failed), 3 (invalid args)

See [render-command documentation](render-command.md) for detailed usage.

### autospec version

Display version information

**Syntax**: `autospec version`

**Alias**: `autospec v`

**Description**: Show autospec version number and build info.

**Examples**:
```bash
autospec version
```

**Exit Codes**: 0 (success)

### autospec ck

Check if an update is available

**Syntax**: `autospec ck [flags]`

**Alias**: `autospec check`

**Description**: Check if a newer version of autospec is available on GitHub releases.

**Flags**:
- `--plain`: Plain output without formatting (key-value pairs for scripting)

**Examples**:
```bash
autospec ck              # Check for updates (colored output)
autospec ck --plain      # Plain output for scripts
autospec check           # Using the longer alias
```

**Exit Codes**: 0 (success), 1 (network error)

### autospec update-task

Update the status of a task in tasks.yaml

**Syntax**: `autospec update-task <task-id> <status>`

**Description**: Programmatically update task status. Used internally by the implementation workflow and available for manual task management.

**Examples**:
```bash
autospec update-task T001 completed
autospec update-task T003 in_progress
```

**Exit Codes**: 0 (success), 3 (invalid args)

### autospec task

Manage tasks within tasks.yaml

**Syntax**: `autospec task <subcommand> [flags]`

**Subcommands**:
- `list`: List tasks with optional status filters
- `block <task-id>`: Block a task with a reason
- `unblock <task-id>`: Unblock a task and set its status

**Flags** (list):
- `--blocked`: Show only blocked tasks
- `--pending`: Show only pending tasks
- `--in-progress`: Show only in-progress tasks
- `--completed`: Show only completed tasks

**Flags** (block):
- `-r, --reason <text>`: Reason for blocking (required)

**Flags** (unblock):
- `-s, --status <status>`: Status to set after unblocking (default: `Pending`)

**Examples**:
```bash
autospec task list
autospec task list --pending
autospec task block T003 --reason "Waiting on API design"
autospec task unblock T003 --status InProgress
```

**Exit Codes**: 0 (success), 3 (invalid args)

### autospec new-feature

Create a new feature branch and directory

**Syntax**: `autospec new-feature <feature_description> [flags]`

**Description**: Create a new numbered spec directory and feature branch without running any workflow stages.

**Flags**:
- `--json`: Output as JSON
- `--short-name <name>`: Custom short name for the branch (2-4 words)
- `--number <N>`: Specify branch number manually (overrides auto-detection)
- `--no-fetch`: Skip fetching from remote repositories

**Examples**:
```bash
autospec new-feature "Add user authentication"
autospec new-feature "Add caching" --short-name "add-cache"
autospec new-feature "Add logging" --number 042
```

**Exit Codes**: 0 (success), 3 (invalid args)

### autospec prereqs

Check prerequisites for workflow stages

**Syntax**: `autospec prereqs [flags]`

**Description**: Validate that required artifacts exist for the current spec before running workflow stages.

When no explicit feature is provided, `autospec prereqs` checks the persisted active feature first and then falls back to branch-prefix detection. If persisted state points at a deleted spec directory, branch-prefix fallback is used.

**Flags**:
- `--json`: Output as JSON
- `--require-spec`: Check for spec.yaml
- `--require-plan`: Check for plan.yaml
- `--require-tasks`: Check for tasks.yaml
- `--include-tasks`: Include task status in output
- `--paths-only`: Output only file paths

**Examples**:
```bash
autospec prereqs
autospec prereqs --require-plan --json
```

**Exit Codes**: 0 (success), 1 (prerequisites missing)

### autospec clean

Remove autospec files from the project

**Syntax**: `autospec clean [flags]`

**Description**: Remove autospec configuration, state, and optionally spec files from the project.

**Flags**:
- `-n, --dry-run`: Preview what would be removed
- `-y, --yes`: Skip confirmation prompt
- `-k, --keep-specs`: Keep spec directories (remove only config)
- `-r, --remove-specs`: Remove spec directories too

**Examples**:
```bash
autospec clean --dry-run        # Preview cleanup
autospec clean --yes            # Clean without prompts
autospec clean --remove-specs   # Also remove specs/
```

**Exit Codes**: 0 (success), 1 (failed)

### autospec migrate

Migrate artifacts between formats

**Syntax**: `autospec migrate`

**Description**: Migrate legacy configuration and artifact formats to current versions.

**Exit Codes**: 0 (success), 1 (failed)

### autospec commands

Manage autospec command templates

**Syntax**: `autospec commands`

**Description**: List and manage the slash command templates installed in agent command directories.

**Exit Codes**: 0 (success)

### autospec uninstall

Completely remove autospec from the system

**Syntax**: `autospec uninstall [flags]`

**Description**: Remove all autospec files including config, state, and agent integrations.

**Flags**:
- `-n, --dry-run`: Preview what would be removed
- `-y, --yes`: Skip confirmation prompt

**Examples**:
```bash
autospec uninstall --dry-run    # Preview removal
autospec uninstall --yes        # Remove without prompts
```

**Exit Codes**: 0 (success), 1 (failed)

## CLI Agents

autospec supports multiple CLI-based AI coding agents. The `--agent` flag is available on all workflow commands to override the configured agent for a single execution.

### Available Agents

| Agent | Binary | Description |
|-------|--------|-------------|
| `claude` | `claude` | Anthropic's Claude Code CLI (default) |
| `cline` | `cline` | Cline VSCode extension CLI |
| `gemini` | `gemini` | Google Gemini CLI |
| `codex` | `codex` | OpenAI Codex CLI |
| `opencode` | `opencode` | OpenCode CLI |
| `goose` | `goose` | Goose AI CLI |

### Agent Override Examples

```bash
# Use gemini for all stages
autospec run -a "Add caching" --agent gemini

# Use cline for planning
autospec plan --agent cline

# Use codex for implementation
autospec implement --agent codex

# Use codex for a complete workflow
autospec run -a --agent codex "Add caching"
```

### Agent Status

Check agent availability with `autospec doctor`:

```bash
$ autospec doctor

CLI Agents:
  ✓ claude: installed (v1.0.5)
  ○ cline: not found in PATH
  ✓ gemini: installed (v0.8.2)
```

See [CLI Agent Configuration](./agents.md) for detailed documentation on agent configuration, custom agents, and migration from legacy settings.

### autospec worktree

Manage git worktrees with project-aware setup automation

**Syntax**: `autospec worktree <subcommand> [flags]`

**Description**: Create and manage git worktrees with automatic copying of non-tracked directories (`.autospec/`, `.agents/`, `.claude/`) and execution of project-specific setup scripts.

**Subcommands**:
- `create <name> --branch <branch> [--path <path>]`: Create new worktree
- `list`: List all tracked worktrees
- `remove <name> [--force]`: Remove a worktree
- `setup <path> [--track]`: Run setup on existing worktree
- `prune`: Remove stale tracking entries
- `gen-script`: Generate a project-specific worktree setup script (`--include-env` to include .env files)

**Examples**:
```bash
# Create a new worktree
autospec worktree create feature-auth --branch feat/user-auth

# Create at custom location
autospec worktree create zoom --branch feat/zoom --path /tmp/zoom-dev

# List all worktrees
autospec worktree list

# Remove (with safety checks)
autospec worktree remove feature-auth

# Force remove (bypass checks)
autospec worktree remove feature-auth --force

# Setup existing worktree
autospec worktree setup ../my-worktree --track

# Clean up stale entries
autospec worktree prune
```

**Exit Codes**: 0 (success), 1 (operation failed), 3 (invalid args)

See [docs/worktree.md](worktree.md) for detailed documentation.

## Configuration Options

Configuration sources (priority order): Environment variables > Local config > Global config > Defaults

### agent_preset

**Type**: string
**Default**: `"claude"`
**Description**: Name of the built-in agent to use for workflow execution

**Available presets**: `claude`, `cline`, `gemini`, `codex`, `opencode`, `goose`

**Example**:
```yaml
agent_preset: gemini
```

**Environment**: `AUTOSPEC_AGENT_PRESET`

See [CLI Agent Configuration](./agents.md) for detailed agent documentation.

### use_subscription

**Type**: boolean
**Default**: `true`
**Description**: Force Claude to use subscription (Pro/Max) instead of API credits. When enabled, `ANTHROPIC_API_KEY` is set to empty at execution time, preventing accidental API charges.

**Example**:
```yaml
# Default: use subscription mode (recommended)
use_subscription: true

# Disable to use API credits instead
use_subscription: false
```

**Environment**: `AUTOSPEC_USE_SUBSCRIPTION`

**Note**: This setting protects users from accidentally burning API credits when they have `ANTHROPIC_API_KEY` set in their shell for other purposes. Set to `false` only if you specifically want to use API billing.

### model

**Type**: string
**Default**: `""`
**Description**: Default model passed to autospec workflow stages for supported CLI agents. Applies when the active workflow agent is Claude, Codex, or OpenCode and no `--model` override is provided.

**Example**:
```yaml
agent_preset: codex
model: gpt-5.4
```

**Environment**: `AUTOSPEC_MODEL`

For one command invocation, pass `--model <model>`. Precedence is `--model`, then top-level `model`, then the agent CLI default.

### skip_permissions

**Type**: boolean
**Default**: `true`
**Description**: Enable autonomous mode for supported agents. Claude receives `--dangerously-skip-permissions`; Codex receives `--dangerously-bypass-approvals-and-sandbox`; OpenCode continues to rely on its `run` mode and configured permissions.

**Example**:
```bash
autospec config set skip_permissions false  # require agent approvals/sandbox behavior
autospec config set skip_permissions true   # restore unattended autonomous mode
```

**Environment**: `AUTOSPEC_SKIP_PERMISSIONS`

**Note**: `autospec init` defaults this setting to enabled for unattended workflow execution. For Claude, enable Claude's sandbox first (`/sandbox` in Claude Code) for OS-level isolation. For Codex, this maps to yolo mode (`--dangerously-bypass-approvals-and-sandbox`); set `skip_permissions: false` if you want Codex sandbox and approval behavior controlled by Codex config. See [Claude Settings](./claude-settings.md) and [Codex Settings](./codex-settings.md) for security details.

### custom_agent_cmd

**Type**: string
**Default**: `""` (not set)
**Description**: Custom agent command template with `{{PROMPT}}` placeholder. Takes precedence over `agent_preset`.

**Example**:
```yaml
custom_agent_cmd: "my-agent run --prompt {{PROMPT}} --mode headless"
```

**Environment**: `AUTOSPEC_CUSTOM_AGENT_CMD`

### max_retries

**Type**: integer
**Default**: `0` (disabled)
**Range**: 0-10
**Description**: Maximum retry attempts on validation failure. Set to 0 to disable automatic retries.

**Example**:
```yaml
max_retries: 5
```

**Environment**: `AUTOSPEC_MAX_RETRIES`

### specs_dir

**Type**: string
**Default**: `"./specs"`
**Description**: Directory for feature specifications

**Example**:
```yaml
specs_dir: /path/to/specs
```

**Environment**: `AUTOSPEC_SPECS_DIR`

### state_dir

**Type**: string
**Default**: `"~/.autospec/state"`
**Description**: Directory for persistent state (retry tracking)

**Example**:
```yaml
state_dir: ~/.autospec/state
```

**Environment**: `AUTOSPEC_STATE_DIR`

### timeout

**Type**: integer
**Default**: `2400` (40 minutes)
**Range**: 0 or 1-604800 (7 days in seconds)
**Description**: Command execution timeout in seconds

**Example**:
```yaml
timeout: 600
```

**Environment**: `AUTOSPEC_TIMEOUT`

**Behavior**:
- `0`: No timeout (infinite wait)
- `1-604800`: Timeout after specified seconds
- Commands exceeding timeout return exit code 5

### skip_preflight

**Type**: boolean
**Default**: `false`
**Description**: Skip pre-flight dependency checks

**Example**:
```yaml
skip_preflight: true
```

**Environment**: `AUTOSPEC_SKIP_PREFLIGHT`

### implement_method

**Type**: string (enum)
**Default**: `"phases"`
**Values**: `"phases"` | `"tasks"` | `"single-session"`
**Description**: Default execution method for the implement command

**Example**:
```yaml
implement_method: tasks  # Each task in separate agent session
```

**Environment**: `AUTOSPEC_IMPLEMENT_METHOD`

**Behavior**:
- `phases`: Each phase runs in separate session (fresh context per phase) — **default**
- `tasks`: Each task runs in separate session (maximum context isolation)
- `single-session`: All tasks in single agent session (legacy)

**Note**: CLI flags (`--phases`, `--tasks`, `--single-session`) override this config setting.

### max_history_entries

**Type**: integer
**Default**: `500`
**Description**: Maximum number of command history entries to retain. Oldest entries are pruned when this limit is exceeded.

**Example**:
```yaml
max_history_entries: 1000
```

**Environment**: `AUTOSPEC_MAX_HISTORY_ENTRIES`

### view_limit

**Type**: integer
**Default**: `5`
**Description**: Number of recent specs to display in the view command dashboard

**Example**:
```yaml
view_limit: 10
```

**Environment**: `AUTOSPEC_VIEW_LIMIT`

**Note**: Can be overridden by the `--limit` flag on the `autospec view` command.

### auto_commit

**Type**: boolean
**Default**: `false`
**Description**: Enable automatic git commit creation after workflow completion. When enabled, the agent receives instructions to update .gitignore with common patterns, stage appropriate files, and create a conventional commit message.

**Example**:
```yaml
auto_commit: true   # Enable auto-commit
auto_commit: false  # Disable auto-commit (default)
```

**Environment**: `AUTOSPEC_AUTO_COMMIT`

**Behavior**:
- When enabled, the agent is instructed to:
  1. Identify and add ignorable files/folders (node_modules, __pycache__, .tmp, build artifacts) to .gitignore
  2. Stage appropriate files for version control (excluding temporary files and dependencies)
  3. Create a commit message in conventional commit format: `type(scope): description`
- The `--auto-commit` flag enables this for a single command
- The `--no-auto-commit` flag disables this for a single command (overrides config)
- Flags are mutually exclusive

**Migration Notice**: On first workflow run after upgrading, a one-time notice about auto-commit is displayed. This notice is shown once per user and persisted to state.

**Failure Handling**: If the auto-commit process fails (e.g., git add fails, .gitignore write fails), the workflow still succeeds (exit 0) and a warning is logged to stderr.

### enable_risk_assessment

**Type**: boolean
**Default**: `false`
**Description**: Controls whether risk assessment instructions are injected into the plan stage prompt. When enabled, the generated `plan.yaml` will include a `risks` section documenting potential implementation risks and mitigations.

**Example**:
```yaml
enable_risk_assessment: false  # Disabled by default
enable_risk_assessment: true   # Enable risk documentation in plan.yaml
```

**Environment**: `AUTOSPEC_ENABLE_RISK_ASSESSMENT`

**Behavior**:
- When disabled (default), plan generation skips the `risks` section to reduce cognitive overhead for simple features
- When enabled, the agent receives instructions to document:
  - Technical risks (dependencies, performance, scalability, security)
  - Integration risks (third-party APIs, data migration, system compatibility)
  - Operational risks (deployment, monitoring, maintenance complexity)
  - Schedule risks (complexity underestimation, external blockers)
- Each risk includes: description, likelihood (low/medium/high), impact (low/medium/high), and optional mitigation strategy
- For trivial features, an empty `risks: []` array is acceptable

**Use Cases**:
- Enable for complex features with significant technical unknowns
- Enable for projects with strict risk management requirements
- Keep disabled for simple bug fixes or small enhancements

### skip_confirmations

**Type**: boolean
**Default**: `false`
**Description**: Skip interactive confirmation prompts. Useful for CI/CD pipelines.

**Environment**: `AUTOSPEC_SKIP_CONFIRMATIONS`

### skip_permissions

**Type**: boolean
**Default**: `true`
**Description**: Enable autonomous mode for supported agents. Claude uses `--dangerously-skip-permissions`; Codex uses `--dangerously-bypass-approvals-and-sandbox`.

**Environment**: `AUTOSPEC_SKIP_PERMISSIONS`

### cclean

**Type**: object
**Description**: Configuration for cclean output formatting.

#### cclean.verbose

**Type**: boolean
**Default**: `false`
**Description**: Enable verbose output with usage stats and tool IDs (`-V` flag)

#### cclean.line_numbers

**Type**: boolean
**Default**: `false`
**Description**: Show line numbers in formatted output (`-n` flag)

#### cclean.style

**Type**: string (enum)
**Default**: `"default"`
**Values**: `"default"` | `"compact"` | `"minimal"` | `"plain"`
**Description**: Output formatting style for cclean (`-s` flag)

### codex_output

**Type**: object
**Description**: Configuration for automated Codex output formatting. Applies only to built-in Codex non-interactive runs.

#### codex_output.mode

**Type**: string (enum)
**Default**: `"compact"`
**Values**: `"compact"` | `"full"`
**Description**: `compact` runs `codex exec --json` and displays concise JSONL event summaries. `full` preserves Codex's native terminal output.

**Environment**: `AUTOSPEC_CODEX_OUTPUT_MODE`

#### codex_output.max_lines_per_message

**Type**: integer
**Default**: `40`
**Description**: Maximum lines shown for each compact Codex output block before autospec prints a truncation marker.

**Environment**: `AUTOSPEC_CODEX_OUTPUT_MAX_LINES_PER_MESSAGE`

#### codex_output.color

**Type**: boolean
**Default**: `true`
**Description**: Enable ANSI color in compact Codex output.

**Environment**: `AUTOSPEC_CODEX_OUTPUT_COLOR`

### worktree

**Type**: object
**Description**: Git worktree management configuration.

#### worktree.base_dir

**Type**: string
**Default**: `""` (uses default location)
**Description**: Parent directory for new worktrees

#### worktree.prefix

**Type**: string
**Default**: `""`
**Description**: Directory name prefix for worktrees

#### worktree.setup_script

**Type**: string
**Default**: `""`
**Description**: Path to setup script relative to repo root. Runs after worktree creation.

#### worktree.auto_setup

**Type**: boolean
**Default**: `true`
**Description**: Run setup script automatically on worktree creation

#### worktree.track_status

**Type**: boolean
**Default**: `true`
**Description**: Persist worktree state for status tracking

#### worktree.copy_dirs

**Type**: list
**Default**: `[.autospec, .agents, .claude]`
**Description**: Non-tracked directories to copy to new worktrees (for example, autospec state and agent configuration directories)

#### worktree.setup_timeout

**Type**: duration
**Default**: `"5m"`
**Description**: Maximum duration for setup script execution

### verification

**Type**: object
**Default**: `{ level: "basic", mutation_threshold: 0.8, coverage_threshold: 0.85, complexity_max: 10 }`
**Description**: Configuration for verification depth and quality thresholds

#### verification.level

**Type**: string (enum)
**Default**: `"basic"`
**Values**: `"basic"` | `"enhanced"` | `"full"`
**Description**: Verification tier that controls which features are enabled by default

**Level Feature Sets**:

| Level | Adversarial Review | Contracts | Property Tests | Metamorphic Tests |
|-------|-------------------|-----------|----------------|-------------------|
| `basic` | disabled | disabled | disabled | disabled |
| `enhanced` | disabled | **enabled** | disabled | disabled |
| `full` | **enabled** | **enabled** | **enabled** | **enabled** |

**Example**:
```yaml
verification:
  level: enhanced  # Enable contracts verification by default
```

**Environment**: `AUTOSPEC_VERIFICATION_LEVEL`

#### verification.adversarial_review

**Type**: boolean (optional)
**Default**: Based on level (see table above)
**Description**: Toggle for adversarial review feature. Explicit value overrides level default.

**Example**:
```yaml
verification:
  level: basic
  adversarial_review: true  # Enable despite basic level
```

**Environment**: `AUTOSPEC_VERIFICATION_ADVERSARIAL_REVIEW`

#### verification.contracts

**Type**: boolean (optional)
**Default**: Based on level (see table above)
**Description**: Toggle for contracts verification. Explicit value overrides level default.

**Example**:
```yaml
verification:
  level: enhanced
  contracts: false  # Disable despite enhanced level
```

**Environment**: `AUTOSPEC_VERIFICATION_CONTRACTS`

#### verification.property_tests

**Type**: boolean (optional)
**Default**: Based on level (see table above)
**Description**: Toggle for property-based testing. Explicit value overrides level default.

**Example**:
```yaml
verification:
  level: basic
  property_tests: true  # Enable property tests
```

**Environment**: `AUTOSPEC_VERIFICATION_PROPERTY_TESTS`

#### verification.metamorphic_tests

**Type**: boolean (optional)
**Default**: Based on level (see table above)
**Description**: Toggle for metamorphic testing. Explicit value overrides level default.

**Example**:
```yaml
verification:
  level: full
  metamorphic_tests: false  # Disable metamorphic tests
```

**Environment**: `AUTOSPEC_VERIFICATION_METAMORPHIC_TESTS`

#### verification.mutation_threshold

**Type**: float64
**Default**: `0.8`
**Range**: 0.0-1.0
**Description**: Minimum mutation score threshold for quality gates

**Example**:
```yaml
verification:
  mutation_threshold: 0.9  # Require 90% mutation score
```

**Environment**: `AUTOSPEC_VERIFICATION_MUTATION_THRESHOLD`

#### verification.coverage_threshold

**Type**: float64
**Default**: `0.85`
**Range**: 0.0-1.0
**Description**: Minimum code coverage threshold for quality gates

**Example**:
```yaml
verification:
  coverage_threshold: 0.95  # Require 95% coverage
```

**Environment**: `AUTOSPEC_VERIFICATION_COVERAGE_THRESHOLD`

#### verification.complexity_max

**Type**: integer
**Default**: `10`
**Range**: Positive integer
**Description**: Maximum cyclomatic complexity allowed per function

**Example**:
```yaml
verification:
  complexity_max: 15  # Allow slightly higher complexity
```

**Environment**: `AUTOSPEC_VERIFICATION_COMPLEXITY_MAX`

#### verification.ears_requirements

**Type**: boolean (optional)
**Default**: Based on level (`basic` = disabled, `enhanced`/`full` = enabled)
**Description**: Enable EARS (Easy Approach to Requirements Syntax) requirements in spec.yaml. Explicit value overrides level default.

**Example**:
```yaml
verification:
  level: basic
  ears_requirements: true  # Enable EARS despite basic level
```

**Environment**: `AUTOSPEC_VERIFICATION_EARS_REQUIREMENTS`

### Full Verification Configuration Example

```yaml
# Project config: .autospec/config.yml
verification:
  level: enhanced              # Use enhanced verification tier
  adversarial_review: true     # Override: enable adversarial review
  contracts: true              # Use level default (enabled for enhanced)
  property_tests: false        # Keep disabled
  metamorphic_tests: false     # Keep disabled
  mutation_threshold: 0.85     # Require 85% mutation score
  coverage_threshold: 0.90     # Require 90% coverage
  complexity_max: 10           # Max cyclomatic complexity
```

### Feature Toggle Resolution Order

Feature toggles follow this resolution order (highest to lowest priority):
1. **Explicit toggle**: Value set directly (`adversarial_review: true`)
2. **Level preset**: Default for selected level (see table above)
3. **Default**: `false` if neither explicit nor level preset applies

**Examples**:
- `level: basic` with no explicit toggle → all features disabled
- `level: basic` with `property_tests: true` → only property tests enabled
- `level: full` with `contracts: false` → all features except contracts enabled

### notifications

**Type**: object
**Default**: `{ enabled: false, type: "both", ... }`
**Description**: Configuration for desktop notifications when commands complete

#### notifications.enabled

**Type**: boolean
**Default**: `false`
**Description**: Master switch for all notifications (opt-in)

**Example**:
```yaml
notifications:
  enabled: true
```

**Environment**: `AUTOSPEC_NOTIFICATIONS_ENABLED`

#### notifications.type

**Type**: string (enum)
**Default**: `"both"`
**Values**: `"sound"` | `"visual"` | `"both"`
**Description**: Type of notification to send

**Example**:
```yaml
notifications:
  enabled: true
  type: visual  # Only show desktop notification, no sound
```

**Environment**: `AUTOSPEC_NOTIFICATIONS_TYPE`

#### notifications.sound_file

**Type**: string
**Default**: `""` (uses system default)
**Description**: Custom sound file path for audio notifications

**Supported formats**: `.wav`, `.mp3`, `.aiff`, `.aif`, `.ogg`, `.flac`, `.m4a`

**Example**:
```yaml
notifications:
  enabled: true
  type: sound
  sound_file: /path/to/custom/notification.wav
```

**Environment**: `AUTOSPEC_NOTIFICATIONS_SOUND_FILE`

**Notes**:
- If the file doesn't exist, falls back to system default sound
- macOS default: `/System/Library/Sounds/Glass.aiff`
- Linux: No default sound (requires custom file)

#### notifications.on_command_complete

**Type**: boolean
**Default**: `true` (when notifications enabled)
**Description**: Notify when any autospec command finishes

**Example**:
```yaml
notifications:
  enabled: true
  on_command_complete: true
```

**Environment**: `AUTOSPEC_NOTIFICATIONS_ON_COMMAND_COMPLETE`

#### notifications.on_stage_complete

**Type**: boolean
**Default**: `false`
**Description**: Notify after each workflow stage (specify, plan, tasks, implement)

**Example**:
```yaml
notifications:
  enabled: true
  on_stage_complete: true  # Get notified after each stage
```

**Environment**: `AUTOSPEC_NOTIFICATIONS_ON_STAGE_COMPLETE`

#### notifications.on_error

**Type**: boolean
**Default**: `true` (when notifications enabled)
**Description**: Notify when a command or stage fails

**Example**:
```yaml
notifications:
  enabled: true
  on_error: true
```

**Environment**: `AUTOSPEC_NOTIFICATIONS_ON_ERROR`

#### notifications.on_long_running

**Type**: boolean
**Default**: `false`
**Description**: Only notify if command duration exceeds threshold

**Example**:
```yaml
notifications:
  enabled: true
  on_long_running: true
  long_running_threshold: 60s  # Only notify if command takes > 60 seconds
```

**Environment**: `AUTOSPEC_NOTIFICATIONS_ON_LONG_RUNNING`

#### notifications.long_running_threshold

**Type**: duration
**Default**: `30s`
**Description**: Threshold for `on_long_running` hook. Set to 0 for "always notify".

**Example**:
```yaml
notifications:
  enabled: true
  on_long_running: true
  long_running_threshold: 5m  # 5 minutes
```

**Environment**: `AUTOSPEC_NOTIFICATIONS_LONG_RUNNING_THRESHOLD`

### Full Notification Configuration Example

```yaml
# Project config: .autospec/config.yml
notifications:
  enabled: true              # Master switch - must be true
  type: both                 # "sound", "visual", or "both"
  sound_file: ""             # Optional custom sound file path
  on_command_complete: true  # Notify when command finishes
  on_stage_complete: false   # Notify after each stage
  on_error: true             # Notify on failures
  on_long_running: false     # Only notify for long commands
  long_running_threshold: 2m  # Threshold for on_long_running
```

### Hook Combinations

Hooks are composable - enable multiple to customize notification behavior:

| Use Case | Configuration |
|----------|---------------|
| Notify on completion only | `on_command_complete: true`, others: false |
| Notify on errors only | `on_error: true`, `on_command_complete: false` |
| Notify per stage | `on_stage_complete: true` |
| Notify for long tasks | `on_long_running: true`, `long_running_threshold: 60s` |
| Full notifications | All hooks enabled |

**Notes**:
- Multiple hooks can fire for the same event (e.g., command completes with error after long time)
- Each enabled hook fires independently
- Notifications are disabled automatically in CI environments
- Notifications are skipped in non-interactive sessions (no TTY)

## Exit Codes

Standardized exit codes for programmatic composition and CI/CD integration:

| Code | Meaning | Description | Action |
|------|---------|-------------|--------|
| 0 | Success | All operations completed successfully | Continue workflow |
| 1 | Validation Failed | Output artifact validation failed | Retry or inspect error |
| 2 | Retries Exhausted | Max retry limit reached without success | Reset retry state or fix issue |
| 3 | Invalid Arguments | User provided invalid command arguments | Check command syntax |
| 4 | Missing Dependencies | Required dependencies not found | Install the configured agent CLI or other deps |
| 5 | Command Timeout | Operation exceeded configured timeout | Increase timeout or optimize |

**Examples**:
```bash
# Check exit code in bash
autospec prep "feature"
if [ $? -eq 0 ]; then
    echo "Success"
elif [ $? -eq 2 ]; then
    echo "Retries exhausted, resetting state"
    rm ~/.autospec/state/retry.json
fi

# Use in CI/CD
autospec all "feature" || exit 1
```

## Prerequisite Validation

Before executing any stage command, autospec validates that required artifacts exist. This provides immediate feedback when prerequisites are missing, avoiding wasted API costs and time.

### Constitution Requirement

All stage commands (except `constitution` itself) require a project constitution:

| Command | Requires Constitution |
|---------|----------------------|
| `specify` | Yes |
| `plan` | Yes |
| `tasks` | Yes |
| `implement` | Yes |
| `clarify` | Yes |
| `checklist` | Yes |
| `analyze` | Yes |
| `constitution` | No (creates it) |

If constitution is missing, you'll see:
```
Error: Project constitution not found.

A constitution is required before running any workflow stages.
The constitution defines your project's principles and guidelines.

To create a constitution, run:
  autospec constitution
```

### Artifact Prerequisites

Each command validates that its required artifacts exist in the spec directory:

| Command | Required Artifacts | Remediation |
|---------|-------------------|-------------|
| `specify` | (none) | - |
| `plan` | `spec.yaml` | Run `autospec specify` first |
| `tasks` | `plan.yaml` | Run `autospec plan` first |
| `implement` | `tasks.yaml` | Run `autospec tasks` first |
| `clarify` | `spec.yaml` | Run `autospec specify` first |
| `checklist` | `spec.yaml` | Run `autospec specify` first |
| `analyze` | `spec.yaml`, `plan.yaml`, `tasks.yaml` | Run missing stages first |

**Example error**:
```
Error: spec.yaml not found.

Run 'autospec specify' first to create this file.
```

### Run Command Smart Validation

The `run` command performs "smart" validation - it only checks for artifacts that won't be produced by earlier selected stages:

| Flags | Validates | Reason |
|-------|-----------|--------|
| `-spt` | constitution only | `specify` produces `spec.yaml`, `plan` produces `plan.yaml` |
| `-pti` | `spec.yaml` | `plan` needs `spec.yaml`, but produces `plan.yaml`; `tasks` produces `tasks.yaml` |
| `-ti` | `plan.yaml` | `tasks` needs `plan.yaml`, produces `tasks.yaml` |
| `-i` | `tasks.yaml` | `implement` needs `tasks.yaml` |
| `-a` | constitution only | Full chain (`-spti`) produces all intermediate artifacts |

This allows running `autospec run -spt` without having `spec.yaml` present, since `specify` will create it.

### Exit Code for Missing Prerequisites

Missing prerequisites return exit code **3** (`ExitInvalidArguments`), the same code used for other argument validation failures.

```bash
# Check if prerequisite validation failed
autospec plan
if [ $? -eq 3 ]; then
    echo "Missing prerequisites - run autospec specify first"
fi
```

## File Locations

### Configuration Files

| File | Purpose | Priority |
|------|---------|----------|
| `~/.config/autospec/config.yml` | Global configuration (XDG compliant) | 3 (after env, local) |
| `.autospec/config.yml` | Local project configuration | 2 (after env) |

### State Files

| File | Purpose |
|------|---------|
| `~/.autospec/state/retry.json` | Persistent retry state tracking |
| `~/.autospec/state/history.yaml` | Command execution history log |

### Specification Directories

| Directory | Purpose |
|-----------|---------|
| `./specs/` | Feature specifications (default) |
| `./specs/NNN-feature-name/` | Individual feature directory |
| `./specs/NNN-feature-name/spec.yaml` | Feature specification |
| `./specs/NNN-feature-name/plan.yaml` | Technical plan |
| `./specs/NNN-feature-name/tasks.yaml` | Task breakdown |

**Naming Convention**: `NNN-feature-name` where NNN is a 3-digit number (e.g., `001-dark-mode`, `042-api-auth`)

## Advanced Patterns

### Prompt Injection

All phase commands support optional guidance text to direct the configured agent's execution:

```bash
# Plan with specific focus
autospec plan "Prioritize security and performance"

# Tasks with specific constraints
autospec tasks "Break into very small incremental steps"

# Implement with specific guidance
autospec implement "Focus on completing tests first"
autospec implement 003-feature "Document all public APIs"
```

**How It Works**:
- Guidance text appended to slash command
- Full command displayed before execution
- Works with custom commands using `{{PROMPT}}` placeholder

### Custom Command Templates

Use `custom_agent` for complex pipelines:

```yaml
custom_agent:
  command: sh
  args:
    - -c
    - "claude -p {{PROMPT}} | tee logs/$(date +%s).log | grep -v DEBUG"
```

**Placeholders**:
- `{{PROMPT}}`: Replaced with the rendered autospec prompt text

### Retry State Management

Manually inspect or reset retry state:

```bash
# View retry state
cat ~/.autospec/state/retry.json

# Reset retry state for specific spec:phase
jq 'del(.retries["001-feature:specify"])' ~/.autospec/state/retry.json > tmp && mv tmp ~/.autospec/state/retry.json

# Reset all retry state
rm ~/.autospec/state/retry.json
```

### CI/CD Integration

Use exit codes for automated workflows:

```yaml
# GitHub Actions example
- name: Run autospec prep
  run: |
    autospec prep "feature" || exit 1

- name: Check status
  run: autospec status
```

### Timeout Configuration

Configure different timeouts for different operations:

```bash
# Short timeout for quick operations
AUTOSPEC_TIMEOUT=60 autospec doctor

# Long timeout for complex workflows
AUTOSPEC_TIMEOUT=3600 autospec all "complex feature"

# No timeout (default)
AUTOSPEC_TIMEOUT=0 autospec prep "feature"
```

## Further Reading

- **[Quick Start Guide](./quickstart.md)**: Installation and first workflow
- **[Architecture Overview](./architecture.md)**: System design and components
- **[CLI Agent Configuration](./agents.md)**: Multi-agent support and custom agents
- **[Troubleshooting](./troubleshooting.md)**: Common issues and solutions
