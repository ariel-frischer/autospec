---
layout: default
title: CLI Commands
parent: Reference
nav_order: 1
---

# CLI Commands

Complete reference for autospec commands, configuration, and workflow stages.
{: .fs-6 .fw-300 }

All commands support global flags: `--config`, `--profile`, `--specs-dir`, `--debug`, `--verbose`, `--output-style`

- `--output-style <style>`: Output formatting style (`default`, `compact`, `minimal`, `plain`, `raw`)
- `--config <path>`: Load one explicit YAML configuration file
- `--profile <name>`: Load a named profile overlay from `~/.config/autospec/profiles/<name>.yml` and/or `.autospec/profiles/<name>.yml`

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
- `-e, --reasoning-effort <effort>`: Override Codex reasoning effort for this run
- `--opencode-agent <name>`: OpenCode sub-agent to use (e.g., `build`, `plan`); see [OpenCode Agents](https://opencode.ai/docs/agents)
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
- `-e, --reasoning-effort <effort>`: Override Codex reasoning effort for this run
- `--opencode-agent <name>`: OpenCode sub-agent to use (e.g., `build`, `plan`)
- `--auto-commit` / `--no-auto-commit`: Override auto-commit config

**Canonical Stage Order**: constitution → specify → clarify → plan → tasks → checklist → analyze → implement

**Feature Selection**: When a workflow command needs an existing feature directory, autospec resolves it in this order: explicit command selection (the `--spec` flag on `run`, `all`, and `prep`, or a positional spec argument on `status` and `implement`), persisted project-local active feature state, then branch-prefix fallback. Commands that create or select a feature may update the persisted active feature so later `plan`, `tasks`, `implement`, `status`, `prereqs`, and artifact lookups can target the same directory from a differently named branch. If persisted state points at a deleted spec directory, autospec ignores that stale selection and continues to branch-prefix fallback.

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
- `--opencode-agent <name>`: OpenCode sub-agent to use (e.g., `build`, `plan`)
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

Without a `spec-name`, status reports the currently resolved active feature. Resolution uses persisted project-local active feature state before falling back to the current branch prefix. A `spec-name` argument is explicit selection and overrides persisted state. `status` has no `--spec` flag; pass the spec name as the positional argument (`autospec status 003-feature`).

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
