# OpenCode Settings

> **Verified with:** OpenCode 1.14.46 (2026-05-13)

This document covers OpenCode configuration for use with autospec.

## Configuration Files

OpenCode uses two configuration locations:

| Location | Scope | Priority |
|----------|-------|----------|
| `~/.config/opencode/opencode.json` | User-level (all projects) | Lower |
| `opencode.json` (project root) | Project-level | Higher |

Project-level settings override user-level settings.

## opencode.json Format

The `opencode.json` file at your project root configures OpenCode behavior:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "anthropic/claude-opus-4-5-20251101",
  "permission": {
    "bash": {
      "autospec *": "allow"
    }
  },
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

### Key Fields

| Field | Type | Description |
|-------|------|-------------|
| `$schema` | string | JSON schema URL for validation |
| `model` | string | Default model in `provider/model-id` format |
| `permission` | object | Command permission rules |
| `permission.bash` | object | Bash command patterns and their permission levels |
| `agent` | object | Agent-specific model overrides |

## Permission Configuration

OpenCode requires explicit permission for bash commands. autospec needs the `autospec *` pattern allowed.

### Permission Levels

| Level | Behavior |
|-------|----------|
| `allow` | Command runs without prompting |
| `ask` | User is prompted for approval |
| `deny` | Command is blocked |

### Required Permission for autospec

```json
{
  "permission": {
    "bash": {
      "autospec *": "allow"
    }
  }
}
```

The `*` glob matches any arguments, so `autospec run`, `autospec implement`, `autospec update-task`, etc. are all allowed.

### Automatic Configuration

Running `autospec init --ai opencode` configures the required permissions:

```bash
autospec init --ai opencode           # Permissions → global (~/.config/opencode/opencode.json)
autospec init --ai opencode --project # Permissions → project (./opencode.json)

# Initialize for both Claude and OpenCode
autospec init --ai claude,opencode
```

**Default behavior:** Permissions write to global config so they apply across all projects. Use `--project` for project-specific overrides.

## Command Directory Structure

OpenCode stores command templates in a different location than Claude:

| Agent | Command Directory |
|-------|-------------------|
| Claude | `.claude/commands/` (plural) |
| OpenCode | `.opencode/command/` (singular) |

When you run `autospec init --ai opencode`, templates are installed to `.opencode/command/autospec.*.md`.

## Automatic Package Installation Behavior

> **Note**: This behavior is caused by OpenCode itself, not autospec.

When OpenCode runs, it automatically installs npm packages into your project's `.opencode/` directory:

```
.opencode/
├── command/           # autospec command templates (expected)
├── node_modules/      # npm packages (auto-created by OpenCode)
├── package.json       # depends on @opencode-ai/plugin (auto-created)
├── bun.lock           # Bun lockfile (auto-created)
└── .gitignore         # ignores the npm artifacts (auto-created)
```

### Why This Happens

OpenCode's `installDependencies()` function runs on every startup and:

1. Creates `package.json` if missing
2. Runs `bun add @opencode-ai/plugin`
3. Runs `bun install`
4. Creates `.opencode/.gitignore` to hide npm artifacts from git

This happens in every project directory where OpenCode runs, regardless of whether plugins are used.

### Git Status

OpenCode auto-creates `.opencode/.gitignore` containing:

```
node_modules
package.json
bun.lock
.gitignore
```

So the npm artifacts won't pollute your git history, but they do exist on disk.

### Mitigation Options

There is **no single flag** to fully disable this behavior. The following environment variables reduce automatic installs but don't eliminate the config-directory `bun install`:

```bash
# Add to ~/.bashrc, ~/.zshrc, or shell profile
export OPENCODE_DISABLE_DEFAULT_PLUGINS=true   # Skip builtin auth plugins
export OPENCODE_DISABLE_LSP_DOWNLOAD=true      # Skip language server installs
export OPENCODE_DISABLE_MODELS_FETCH=true      # Skip model fetching
```

**What each flag does:**

| Flag | Effect |
|------|--------|
| `OPENCODE_DISABLE_DEFAULT_PLUGINS` | Prevents `opencode-copilot-auth` and `opencode-anthropic-auth` from being installed |
| `OPENCODE_DISABLE_LSP_DOWNLOAD` | Prevents automatic language server npm installs |
| `OPENCODE_DISABLE_MODELS_FETCH` | Disables automatic model fetching |

**Limitation:** Even with all flags set, OpenCode still runs `bun install` in config directories. The only way to fully prevent this is to run a local/dev build of OpenCode (where `Installation.isLocal()` returns true).

### If You Want to Fully Prevent npm Artifacts

Options:

1. **Accept it** - The files are auto-gitignored and don't affect functionality
2. **Pre-create empty package.json** - Create `.opencode/package.json` with `{}` content; `bun install` will still run but install nothing
3. **Use Claude Code instead** - Claude Code doesn't have this behavior
4. **Request upstream fix** - Open an issue on [sst/opencode](https://github.com/sst/opencode) requesting a `OPENCODE_DISABLE_CONFIG_INSTALL` flag

## Command Invocation Patterns

OpenCode uses different invocation patterns for automated and interactive modes:

### Automated Mode (specify, plan, tasks, implement)

```bash
opencode run "feature description" --command autospec.specify
```

Pattern: `opencode run <message> --command <command-name>`

Key differences from Claude:
- Uses `run` subcommand (not `-p` flag)
- Command name is passed via `--command` flag at the end
- Message is a positional argument

### Interactive Mode (clarify, analyze)

```bash
opencode --prompt "/autospec.clarify"
```

Pattern: `opencode --prompt "<slash-command>"`

Key differences:
- No `run` subcommand
- Uses `--prompt` flag
- Slash command passed directly (not parsed)

## Model Configuration

### Model Format

Models are specified as `provider/model-id`:

| Model | ID |
|-------|-----|
| Claude Opus 4.5 (pinned) | `anthropic/claude-opus-4-5-20251101` |
| Claude Opus 4.5 (latest) | `anthropic/claude-opus-4-5-latest` |
| Claude Sonnet 4 | `anthropic/claude-sonnet-4-20250514` |
| Claude Haiku 4 | `anthropic/claude-haiku-4-20250514` |

> **Tip**: Use date-pinned versions for production. The `-latest` alias auto-updates and may cause unexpected changes.

### Setting Default Model

```json
{
  "model": "anthropic/claude-opus-4-5-20251101"
}
```

### Agent-Specific Models

Configure different models for different agent modes:

```json
{
  "agent": {
    "build": {
      "model": "anthropic/claude-opus-4-5-20251101"
    },
    "plan": {
      "model": "anthropic/claude-sonnet-4-20250514"
    }
  }
}
```

## Authentication

OpenCode supports multiple authentication methods:

### OAuth (Recommended)

OAuth with your Claude Pro/Max subscription avoids API charges:

1. Run `opencode` to start the interactive interface
2. Use `/login` or `/connect` command
3. Select **Anthropic** from the provider list
4. Complete browser-based OAuth authentication

Credentials are stored in `~/.local/share/opencode/auth.json`.

### API Key

Set the `ANTHROPIC_API_KEY` environment variable:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

> **Warning**: API usage can become costly. OAuth with Pro/Max subscription is recommended.

## Using OpenCode as Default Agent

### Via Configuration

```yaml
# .autospec/config.yml or ~/.config/autospec/config.yml
agent_preset: opencode
```

### Via Environment Variable

```bash
export AUTOSPEC_AGENT_PRESET=opencode
```

### Via CLI Flag

```bash
autospec run -a "feature" --agent opencode
autospec implement --agent opencode
```

## Checking Configuration

Use `autospec doctor` to verify OpenCode is properly configured:

```bash
$ autospec doctor

CLI Agents:
  ✓ claude: installed (v2.0.76)
  ✓ opencode: installed (v1.0.223)
```

## Troubleshooting

### "opencode.json not found"

Run `autospec init --ai opencode` to create the configuration file with required permissions.

### "Permission denied for autospec *"

Check that `opencode.json` has:

```json
{
  "permission": {
    "bash": {
      "autospec *": "allow"
    }
  }
}
```

If you see `"deny"`, change it to `"allow"`.

### Command Not Working

Verify the command pattern:

- **Automated**: `opencode run "message" --command autospec.specify`
- **Interactive**: `opencode --prompt "/autospec.clarify"`

### Model Not Found

Use `/models` in OpenCode to list available models for your authenticated providers.

## Related Documentation

- [CLI Agent Configuration](agents.md) - Full agent abstraction documentation
- [Claude Settings](claude-settings.md) - Claude Code configuration
