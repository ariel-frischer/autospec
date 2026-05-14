# CLI Agent Configuration

autospec supports multiple CLI-based AI coding agents through a unified agent abstraction layer. This allows you to use your preferred agent while maintaining compatibility with the same workflow commands.

## Supported Agents

### Currently Supported

| Agent | Binary | Description | Status |
|-------|--------|-------------|--------|
| `claude` | `claude` | Anthropic's Claude Code CLI (default) | ✅ Supported; smoke-tested with 2.1.139 |
| `codex` | `codex` | OpenAI Codex CLI | ✅ Supported; smoke-tested with 0.130.0 |
| `opencode` | `opencode` | OpenCode AI coding CLI | ✅ Supported; smoke-tested with 1.14.46 |

### Experimental Agents (Untested)

| Agent | Binary | Description | Status |
|-------|--------|-------------|--------|
| `cline` | `cline` | Cline VSCode extension CLI | ⚠️ Untested |
| `gemini` | `gemini` | Google Gemini CLI | ⚠️ Untested |
| `goose` | `goose` | Goose AI CLI | ⚠️ Untested |

These agents have code-level support (agent abstraction, command building, doctor checks) but have not been tested with real binaries. They may require adjustments. Please [report issues](https://github.com/ariel-frischer/autospec/issues) if you try them.

### Custom Agents

You can configure any CLI tool as an agent using a command template with `{{PROMPT}}` placeholder.

## Configuration

### Using a Preset Agent

Set the `agent_preset` field in your configuration file:

```yaml
# .autospec/config.yml
agent_preset: claude
```

Or in user-level config:

```yaml
# ~/.config/autospec/config.yml
agent_preset: gemini
```

### Using a Custom Agent Command

For agents not built-in, or for custom configurations:

```yaml
# .autospec/config.yml
custom_agent_cmd: "my-agent run --prompt {{PROMPT}} --mode headless"
```

The `{{PROMPT}}` placeholder is replaced with the actual prompt at execution time. The placeholder can appear anywhere in the command template.

### CLI Flag Override

Override the configured agent for a single command execution:

```bash
# Use gemini for this run only
autospec run -a "Add user auth" --agent gemini

# Use codex for a full run
autospec run -a --agent codex "Add user auth"

# Use cline for implementation
autospec implement --agent cline
```

Available for all workflow commands: `run`, `prep`, `specify`, `plan`, `tasks`, `implement`.

## Configuration Priority

When determining which agent to use, autospec follows this priority order:

1. **CLI flag** (`--agent`): Highest priority, single-command override
2. **custom_agent**: Project or user-level custom command configuration
3. **agent_preset**: Project or user-level preset name
4. **Default**: Falls back to `claude` agent (hardcoded)

> **Note**: When `agent_preset` is empty (`""`), autospec always uses `claude` as the default agent. This is a hardcoded fallback, not configurable via `default_agents`.

### `agent_preset` vs `default_agents`

These two config fields serve different purposes:

| Field | Purpose | Used When |
|-------|---------|-----------|
| `agent_preset` | Selects which agent runs commands | Runtime (every command) |
| `default_agents` | Pre-selects checkboxes in `autospec init` prompt | Initialization prompt defaults |

**Example config:**

```yaml
# This agent runs your commands:
agent_preset: opencode

# These are remembered selections for next `autospec init`:
default_agents:
  - claude
  - opencode
```

If `agent_preset` is empty, **claude is used regardless of what's in `default_agents`**. Interactive `autospec init` sets `agent_preset` automatically when one agent is selected, or asks which selected agent should be the execution default when multiple agents are selected. Non-interactive `autospec init --ai claude,codex,opencode` uses the first selected agent as the execution default.

## Environment Configuration

Override agent settings via environment variables:

```bash
# Set agent preset
export AUTOSPEC_AGENT_PRESET=gemini

# Set custom agent command
export AUTOSPEC_CUSTOM_AGENT_CMD="my-agent --prompt {{PROMPT}}"
```

Environment variables take precedence over config file values.

## Auto-Commit Configuration

When enabled, autospec instructs the agent to update .gitignore, stage appropriate files, and create a conventional commit message after workflow completion.

### Configuration

```yaml
# ~/.config/autospec/config.yml or .autospec/config.yml

# Enable auto-commit
auto_commit: true

# Default: auto-commit disabled
auto_commit: false
```

### Environment Variable

Override via environment:

```bash
export AUTOSPEC_AUTO_COMMIT=true   # Enable
export AUTOSPEC_AUTO_COMMIT=false  # Disable
```

### CLI Flags

Override for a single command:

```bash
# Enable auto-commit for this run
autospec implement --auto-commit

# Disable auto-commit for this run (overrides config)
autospec implement --no-auto-commit
```

The flags are mutually exclusive and available on all workflow commands: `run`, `prep`, `specify`, `plan`, `tasks`, `implement`.

### What the Agent Does

When auto-commit is enabled, the agent is instructed to:

1. **Update .gitignore**: Identify ignorable files (node_modules, __pycache__, .tmp, build artifacts, IDE files) and add them to .gitignore
2. **Stage files**: Stage appropriate files for version control, excluding temporary files and dependencies
3. **Create commit**: Create a commit message in conventional commit format: `type(scope): description` where scope is determined by the files/components changed

### Failure Handling

- If the auto-commit process fails (e.g., git add fails, .gitignore write fails), the workflow still succeeds (exit 0)
- A warning is logged to stderr describing the failure
- This ensures that implementation work is never lost due to commit failures

### Migration Notice

On the first workflow run after upgrading to a version with auto-commit enabled by default, a one-time notice is displayed explaining the new behavior. This notice is persisted to state and will not be shown again.

## Claude Subscription Mode

By default, autospec forces Claude to use your **subscription (Pro/Max)** instead of API credits. This protects users from accidentally burning API credits when they have `ANTHROPIC_API_KEY` set in their shell for other purposes.

### How It Works

| Setting | Behavior |
|---------|----------|
| `use_subscription: true` (default) | Forces `ANTHROPIC_API_KEY=""` at execution → uses subscription |
| `use_subscription: false` | Uses shell's `ANTHROPIC_API_KEY` → uses API credits |

### Configuration

```yaml
# ~/.config/autospec/config.yml or .autospec/config.yml

# Default: use subscription (recommended - no API charges)
use_subscription: true

# Override: use API credits instead
use_subscription: false
```

### Cost Display Note

When using subscription mode (`use_subscription: true`), Claude Code still displays cost information in its output:

```
Cost: $0.5014
Tokens: in=2 out=4558 cache_read=284417
```

**This cost is informational only** — it shows what the tokens *would* cost at API rates, but you are not actually charged this amount. With a subscription (Pro/Max), you pay a flat monthly fee and token usage counts against rate limits, not billing.

### Using API Mode

If you specifically want to use API billing:

1. Set `use_subscription: false` in your config
2. Ensure `ANTHROPIC_API_KEY` is set in your shell environment

```yaml
# Enable API mode
use_subscription: false
```

Or with a custom agent:

```yaml
custom_agent:
  command: claude
  args: ["-p", "{{PROMPT}}"]
  env:
    ANTHROPIC_API_KEY: "sk-ant-..."  # Explicit API key
```

## Agent Requirements

Each agent has specific requirements:

| Agent | Binary in PATH | Environment Variables | Status |
|-------|----------------|----------------------|--------|
| `claude` | `claude` | - (uses subscription by default) | ✅ Supported |
| `codex` | `codex` | - (ChatGPT login or API auth via Codex CLI) | ✅ Supported |
| `opencode` | `opencode` | - | ✅ Supported |
| `cline` | `cline` | - | ⚠️ Untested |
| `gemini` | `gemini` | `GEMINI_API_KEY` | ⚠️ Untested |
| `goose` | `goose` | - | ⚠️ Untested |

Use `autospec doctor` to verify agent availability and configuration.

## Checking Agent Status

The `autospec doctor` command shows the status of available agents.

**Production builds** check supported agents (claude, codex, opencode):

```bash
$ autospec doctor

✓ Claude CLI: Claude CLI found
✓ Git: Git found
✓ Claude settings: Bash(autospec:*) permission configured

CLI Agents:
  ✓ claude: installed (v2.0.76)
  ✓ codex: installed (codex-cli 0.130.0; tested 0.130.0)
  ✓ opencode: installed (v1.0.223)
```

**Dev builds** check all registered agents:

```bash
$ autospec doctor

CLI Agents:
  ✓ claude: installed (v2.0.76)
  ○ cline: not found in PATH
  ✓ codex: installed (codex-cli 0.130.0; tested 0.130.0)
  ○ gemini: not found in PATH
  ○ goose: not found in PATH
  ✓ opencode: installed (v1.0.223)
```

## Agent Configuration

There are two ways to configure which agent to use:

### Using a Preset

Use `agent_preset` to select a built-in agent:

```yaml
# Use the claude agent preset
agent_preset: claude
```

### Using a Custom Agent

Use `custom_agent` for full control over the command:

```yaml
# Custom agent configuration
custom_agent:
  command: claude
  args:
    - -p
    - --verbose
    - --output-format
    - stream-json
    - "{{PROMPT}}"
```

You can also use shell commands for pipelines:

```yaml
custom_agent:
  command: sh
  args:
    - -c
    - "claude -p {{PROMPT}} | tee output.log"
```

## Custom Agent Examples

### Using a Custom Model with Claude

```yaml
custom_agent_cmd: "claude --model claude-3-opus {{PROMPT}}"
```

### Piping Output Through a Filter

```yaml
custom_agent_cmd: "claude -p {{PROMPT}} | grep -v DEBUG"
```

### Using SSH to Run on Remote Machine

```yaml
custom_agent_cmd: "ssh build-server 'claude -p {{PROMPT}}'"
```

### Using Docker Container

```yaml
custom_agent_cmd: "docker run --rm ai-agent run {{PROMPT}}"
```

## Codex Configuration

Codex is a supported agent for autospec's non-interactive workflows.

### Invocation Pattern

autospec sends rendered prompt text to Codex using:

```bash
codex exec --json "<rendered autospec prompt>"
```

Use Codex for one command with:

```bash
autospec run -a --agent codex "Add user auth"
autospec implement --agent codex
```

### Authentication

Codex authentication is handled by the Codex CLI itself. autospec does not require `OPENAI_API_KEY`; Codex can use ChatGPT login or API credentials configured through Codex.

Useful environment variables:

| Variable | Purpose |
|----------|---------|
| `OPENAI_API_KEY` | Optional API authentication for Codex |
| `OPENAI_BASE_URL` | Optional API-compatible base URL override |
| `CODEX_HOME` | Optional Codex home/config directory override |

### Settings

Codex reads user config from `~/.codex/config.toml`. Project-level initialization with `autospec init --project --ai codex` creates `.codex/config.toml` as safe project metadata and registers project-local shared skills under `.agents/skills/autospec-*/SKILL.md`.

Codex supports `--sandbox`, `--ask-for-approval`, and `--dangerously-bypass-approvals-and-sandbox` in `codex exec`. autospec maps `skip_permissions: true` to `--dangerously-bypass-approvals-and-sandbox`.

autospec uses compact Codex output by default. It runs `codex exec --json`, parses Codex JSONL events, and shows color-coded concise summaries for agent messages, command executions, file changes, and useful reasoning/tool events. Set `codex_output.color: false` to disable ANSI color, or `codex_output.mode: full` to restore Codex's native terminal transcript. Codex can also write the final assistant message with `codex exec -o <file>`. autospec validates generated autospec artifacts after Codex exits.

Codex does not support autospec's Claude/OpenCode command-template directories. Instead, autospec generates shared Agent Skills from each embedded `autospec.*` prompt. In interactive Codex, use `$autospec-specify "Add user auth"`, `$autospec-plan`, `$autospec-tasks`, `$autospec-implement`, `$autospec-constitution`, `$autospec-clarify`, `$autospec-checklist`, or `$autospec-analyze`.

See [Codex Settings](./codex-settings.md) for details.

## OpenCode Configuration

OpenCode is a fully supported agent with its own configuration patterns that differ from Claude Code.

### Command Directory Structure

| Agent | Command Directory | Note |
|-------|-------------------|------|
| Claude | `.claude/commands/` | Plural "commands" |
| OpenCode | `.opencode/command/` | Singular "command" |

When you run `autospec init --ai opencode`, command templates are installed to `.opencode/command/autospec.*.md` and shared skills are installed to `.agents/skills/autospec-*/SKILL.md`.

### Invocation Pattern

OpenCode uses a different command invocation pattern than Claude:

| Agent | Invocation Pattern |
|-------|-------------------|
| Claude | `claude -p "<rendered autospec prompt>"` |
| Codex | `codex exec "<rendered autospec prompt>"` |
| OpenCode | `opencode run "prompt" --command autospec.specify` |

Key differences:
- OpenCode uses `run` subcommand (not `-p` flag)
- Command name is passed via `--command` flag at the end
- Non-interactive execution is the default with `run`

### Model Configuration

OpenCode supports multiple AI providers. For the best experience with Anthropic models, use OAuth authentication with your Claude Max/Pro subscription instead of API keys.

#### Authentication Setup

1. Run `opencode` to start the interactive interface
2. Use `/login` or `/connect` command
3. Select **Anthropic** from the provider list
4. Complete browser-based OAuth authentication

This stores credentials in `~/.local/share/opencode/auth.json` and allows you to use your Claude Max/Pro subscription without API charges.

> **Warning**: Be careful using `ANTHROPIC_API_KEY` in your shell environment. API usage can become costly quickly. OAuth authentication with your Max/Pro subscription is recommended for most users.

#### Configuration Files

OpenCode uses two configuration locations:

| Location | Scope | Priority |
|----------|-------|----------|
| `~/.config/opencode/opencode.json` | User-level (all projects) | Lower |
| `opencode.json` (project root) | Project-level | Higher |

Project-level settings override user-level settings.

#### Setting Opus 4.5 as Default Model

Create or update your configuration file:

**Project-level** (`opencode.json` in project root):

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "anthropic/claude-opus-4-5-20251101",
  "agent": {
    "build": {
      "model": "anthropic/claude-opus-4-5-20251101"
    },
    "plan": {
      "model": "anthropic/claude-opus-4-5-20251101"
    }
  }
}
```

**User-level** (`~/.config/opencode/opencode.json`):

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "anthropic/claude-opus-4-5-20251101",
  "agent": {
    "build": {
      "model": "anthropic/claude-opus-4-5-20251101"
    },
    "plan": {
      "model": "anthropic/claude-opus-4-5-20251101"
    }
  }
}
```

The model format is `provider/model-id`. For Anthropic OAuth, use `anthropic/` prefix.

#### Available Models

Common Anthropic models:

| Model | ID | Notes |
|-------|-----|-------|
| Claude Opus 4.5 (pinned) | `anthropic/claude-opus-4-5-20251101` | Recommended for production |
| Claude Opus 4.5 (latest) | `anthropic/claude-opus-4-5-latest` | Dev/testing only, auto-updates |
| Claude Sonnet 4 | `anthropic/claude-sonnet-4-20250514` | |
| Claude Haiku 4 | `anthropic/claude-haiku-4-20250514` | |

> **Note**: Use date-pinned versions (e.g., `-20251101`) for production to ensure consistent behavior. The `-latest` alias auto-updates and may cause unexpected changes.

Use `/models` in OpenCode to list all available models for your authenticated providers.

### Permission Configuration

OpenCode uses `opencode.json` at the project root (not in `.opencode/`) for permission configuration:

```json
{
  "permission": {
    "bash": {
      "autospec *": "allow"
    }
  }
}
```

When you run `autospec init --ai opencode`, this permission is automatically added to allow autospec commands to run without manual approval.

**Permission levels:**
- `allow`: Command runs without prompting
- `ask`: User is prompted for approval
- `deny`: Command is blocked

**Glob patterns:** The `*` in `autospec *` matches any arguments, so `autospec run`, `autospec update-task`, etc. are all allowed.

### Using OpenCode as Default Agent

Set OpenCode as your default agent in configuration:

```yaml
# .autospec/config.yml or ~/.config/autospec/config.yml
agent_preset: opencode
```

Or via environment variable:

```bash
export AUTOSPEC_AGENT_PRESET=opencode
```

### Multi-Agent Initialization

Initialize a project for one or more supported agents:

```bash
# Initialize for supported agents
autospec init --ai claude,codex,opencode

# Initialize for Codex only
autospec init --ai codex

# Initialize for OpenCode only
autospec init --ai opencode

# Interactive selection (shows multi-select checklist)
autospec init
```

### Constitution File

OpenCode uses the same constitution file hierarchy as other agents:

1. **AGENTS.md** (primary) - Universal agent instructions
2. **OPENCODE.md** (fallback) - OpenCode-specific instructions if AGENTS.md is missing
3. **CLAUDE.md** (legacy fallback) - For backward compatibility

Command templates reference `AGENTS.md` as the constitution source. If your project only has `CLAUDE.md`, consider creating `AGENTS.md` for multi-agent support.

## Agent Capabilities

All agents expose their capabilities through the agent abstraction:

| Capability | Description |
|------------|-------------|
| Automatable | Supports headless/non-interactive execution |
| Interactive | Supports interactive prompts (not used by autospec) |
| Streaming | Supports real-time output streaming |

Currently, autospec requires automatable agents for all workflow commands.

## Troubleshooting

### Agent Not Found

If `autospec doctor` shows an agent as "not found in PATH":

1. Verify the agent binary is installed
2. Ensure the binary is in your system PATH
3. Try running the agent directly: `which claude` or `claude --version`

### Missing Environment Variables

Some agents require API keys or configuration:

```bash
# For Gemini
export GEMINI_API_KEY=your-api-key
```

Codex does not require `OPENAI_API_KEY`; it can use ChatGPT login or API auth managed by the Codex CLI. `OPENAI_API_KEY` remains optional for API billing.

### Custom Agent Template Issues

If your custom agent command isn't working:

1. Verify `{{PROMPT}}` placeholder is present in the template
2. Test the command manually with a simple prompt
3. Check shell quoting and escaping

```bash
# Test custom command manually
my-agent run --prompt "test prompt"
```

### Agent Validation Failed

If agent validation fails, check:

1. Binary exists and is executable
2. Required environment variables are set
3. Agent can run with `--version` or similar flag
