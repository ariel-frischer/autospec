---
layout: default
title: Configuration
parent: Reference
nav_order: 2
---

# Configuration
{: .no_toc }

Configuration options, file locations, profiles, model selection, reasoning settings, and runtime policies.
{: .fs-6 .fw-300 }

<details open markdown="block">
  <summary>
    Table of contents
  </summary>
  {: .text-delta }
1. TOC
{:toc}
</details>

---

## Configuration Priority

Configuration sources (priority order): Environment variables > Local config > Global config > Defaults

## Core Options

### Named profiles

Named profiles are selected per invocation with `--profile`:

```bash
autospec run -a --profile cheap "Add a feature"
autospec config show --profile cheap
autospec config profiles
autospec config create cheap
```

The user profile is loaded first and the project profile with the same name is
loaded afterward. Environment variables remain the highest-priority overrides.
Profile names use 1-64 letters, numbers, hyphens, or underscores. `--config`
and `--profile` cannot be combined. Use `autospec config profiles` to list
profiles and `autospec config create NAME [--force]` to save the current
effective configuration.

### agent_preset

**Type**: string
**Default**: `""` (uses the default agent; the repository project config selects `jcode`)
**Description**: Name of the built-in agent to use for workflow execution

**Available presets**: `claude`, `cline`, `gemini`, `codex`, `jcode`, `opencode`, `goose`

**Example**:
```yaml
agent_preset: gemini
```

**Environment**: `AUTOSPEC_AGENT_PRESET`

See [CLI Agent Configuration](./agents.md) for detailed agent documentation.

### Native jcode lifecycle settings

### jcode.runner

**Default**: `exec`

Selects the jcode implementation. The default and empty value invoke
`jcode run --quiet` through the installed CLI. Use `custom` with `jcode.binary`
for an alternate executable, or `sdk` explicitly for native SDK lifecycle
behavior. Native SDK settings do not override an unset runner.

When `agent_preset: jcode`, the `jcode` settings control runtime ownership:

| Key | Default | Values / meaning |
| --- | --- | --- |
| `jcode.mode` | `connect` | `connect`, `private`, or `auto` |
| `jcode.socket_path` | empty | Existing API socket, or SDK environment discovery |
| `jcode.binary` | empty | Private runtime executable, defaulting to `jcode` on `PATH` |
| `jcode.home` | empty | Persistent private home, or SDK-owned temporary state |
| `jcode.inherit_logins` | `false` | Whether private launches inherit local jcode logins |
| `jcode.startup_timeout` | `30s` | Maximum private startup duration |
| `jcode.cleanup_timeout` | `30s` | Maximum private cleanup duration |
| `jcode.startup_command` | empty | Optional private/auto-only launcher |
| `jcode.reconnect_attempts` | `2` | Shared bridge reconnect limit, 0-10 |
| `jcode.restart_attempts` | `1` | Run-owned private restart limit, 0-10 |
| `jcode.retry_delay` | `250ms` | Delay between recovery attempts, 0-5m |
| `jcode.session_profile` | empty | Experimental SDK-only named session profile; maps to `CreateSessionOptions.Profile` |
| `jcode.max_turns` | `0` (unset) | Experimental SDK-only positive turn limit; maps to `SendOptions.MaxTurns` |
| `jcode.token_budget` | `0` (unset) | Experimental SDK-only positive token limit; maps to `SendOptions.TokenBudget` |
| `jcode.deadline` | empty | Experimental SDK-only RFC3339 deadline with explicit `Z` or numeric UTC offset; maps to `SendOptions.Deadline` |

The four experimental session controls apply only when `jcode.runner: sdk` is
selected explicitly. They never alter the official `exec` invocation or the
default runner. Omitted controls preserve the SDK defaults. A syntactically
valid past deadline reaches the SDK unchanged so its actionable runtime error is
preserved.

Connect mode never starts or stops a shared daemon. Auto mode prefers a healthy
shared bridge and falls back to a private SDK-owned runtime. Only a runtime
created by the current run may be restarted or cleaned up.

For a disposable built-binary smoke check, use a temporary repository and
temporary `JCODE_HOME`/runtime directory, configure `mode: private` with a
cheap profile, and run the built binary with a short timeout. Accept either a
validated artifact or a bounded actionable failure, then assert that the
temporary home, socket, process, and workspace are gone. Do not point this
check at the developer's shared daemon.

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
**Description**: Default model passed to autospec workflow stages for supported agents. For native jcode, Autospec sends this as a non-secret session setting while jcode retains provider and authentication ownership.

**Example**:
```yaml
agent_preset: jcode
model: gpt-5.4
```

**Environment**: `AUTOSPEC_MODEL`

For one command invocation, pass `--model <model>`. Precedence is `--model`, then the current stage's `models.<stage>` value, then top-level `model`, then the agent CLI default.

### models.&lt;stage&gt;

**Type**: string
**Default**: `""`
**Description**: Optional workflow model for one stage. Supported keys are `models.constitution`, `models.specify`, `models.clarify`, `models.plan`, `models.tasks`, `models.checklist`, `models.analyze`, and `models.implement`.

**Example**:
```yaml
model: provider/default-model
models:
  constitution: provider/constitution-model
  specify: provider/specify-model
  clarify: provider/clarify-model
  plan: provider/plan-model
  tasks: provider/tasks-model
  checklist: provider/checklist-model
  analyze: provider/analyze-model
  implement: provider/implement-model
```

**Environment**: `AUTOSPEC_MODELS_<STAGE>` (for example, `AUTOSPEC_MODELS_CONSTITUTION` through `AUTOSPEC_MODELS_IMPLEMENT`)

Every generated stage default is empty. An empty or absent stage value falls back to top-level `model`, then to the selected agent's default. CLI `--model` remains the highest-priority, invocation-scoped override.

### reasoning_effort

**Type**: string
**Default**: `""`
**Description**: Default reasoning effort passed to supported workflow agents. Autospec forwards this as Codex's `model_reasoning_effort` override or as a native jcode session setting, while the selected agent validates provider compatibility.

**Example**:
```yaml
agent_preset: jcode
model: gpt-5.6-terra
reasoning_effort: high
```

**Environment**: `AUTOSPEC_REASONING_EFFORT`

For one command invocation, pass `-e <effort>` or `--reasoning-effort <effort>`. Precedence is the CLI flag, then a stage-specific value, then top-level `reasoning_effort`, then the Codex model default. Current Codex models use `low`, `medium`, `high`, `xhigh`, and, where supported, `max` or `ultra`.

### reasoning_efforts.&lt;stage&gt;

**Type**: string
**Default**: `""`
**Description**: Optional Codex reasoning effort for one workflow stage. Supported stage keys are `constitution`, `specify`, `clarify`, `plan`, `tasks`, `checklist`, `analyze`, and `implement`.

**Example**:
```yaml
reasoning_effort: medium
reasoning_efforts:
  specify: low
  plan: high
  tasks: medium
  implement: xhigh
```

**Environment**: `AUTOSPEC_REASONING_EFFORTS_<STAGE>`

Precedence is the CLI `-e`/`--reasoning-effort` override, then the current stage's `reasoning_efforts` value, then top-level `reasoning_effort`, then the Codex model default.

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
