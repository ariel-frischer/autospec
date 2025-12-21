# CLI Agent Configuration

autospec supports multiple CLI-based AI coding agents through a unified agent abstraction layer. This allows you to use your preferred agent while maintaining compatibility with the same workflow commands.

## Supported Agents

### Tier 1 Agents (Built-in)

| Agent | Binary | Description |
|-------|--------|-------------|
| `claude` | `claude` | Anthropic's Claude Code CLI (default) |
| `cline` | `cline` | Cline VSCode extension CLI |
| `gemini` | `gemini` | Google Gemini CLI |
| `codex` | `codex` | OpenAI Codex CLI |
| `opencode` | `opencode` | OpenCode CLI |
| `goose` | `goose` | Goose AI CLI |

All built-in agents support headless/automated execution suitable for CI/CD pipelines.

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

# Use codex for planning only
autospec plan --agent codex

# Use cline for implementation
autospec implement --agent cline
```

Available for all workflow commands: `run`, `prep`, `specify`, `plan`, `tasks`, `implement`.

## Configuration Priority

When determining which agent to use, autospec follows this priority order:

1. **CLI flag** (`--agent`): Highest priority, single-command override
2. **custom_agent**: Project or user-level custom command configuration
3. **agent_preset**: Project or user-level preset name
4. **Default**: Falls back to `claude` agent

## Environment Configuration

Override agent settings via environment variables:

```bash
# Set agent preset
export AUTOSPEC_AGENT_PRESET=gemini

# Set custom agent command
export AUTOSPEC_CUSTOM_AGENT_CMD="my-agent --prompt {{PROMPT}}"
```

Environment variables take precedence over config file values.

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

| Agent | Binary in PATH | Environment Variables |
|-------|----------------|----------------------|
| `claude` | `claude` | - (uses subscription by default) |
| `cline` | `cline` | - |
| `gemini` | `gemini` | `GOOGLE_API_KEY` |
| `codex` | `codex` | `OPENAI_API_KEY` |
| `opencode` | `opencode` | - |
| `goose` | `goose` | - |

Use `autospec doctor` to verify agent availability and configuration.

## Checking Agent Status

The `autospec doctor` command shows the status of all registered agents:

```bash
$ autospec doctor

Dependencies:
  Git: installed
  Claude CLI: installed

CLI Agents:
  claude: installed (v1.0.5)
  cline: not found in PATH
  codex: missing OPENAI_API_KEY environment variable
  gemini: installed (v0.8.2)
  goose: not found in PATH
  opencode: installed (v2.1.0)
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
export GOOGLE_API_KEY=your-api-key

# For Codex
export OPENAI_API_KEY=your-api-key
```

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
