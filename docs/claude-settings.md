# Claude Code Settings & Sandboxing

*Last updated: 2025-12-16*

Configuration guide for Claude Code settings relevant to autospec workflows.

## Sandboxing Overview

Claude Code uses OS-level sandboxing to isolate bash commands:

- **Linux**: [bubblewrap (bwrap)](https://github.com/containers/bubblewrap)
- **macOS**: Seatbelt

When enabled, all commands run in an isolated environment with restricted filesystem and network access.

## Enabling Sandbox

### Via Settings File (Recommended)

Add to `.claude/settings.local.json` (project) or `~/.claude/settings.json` (global):

```json
{
  "sandbox": {
    "enabled": true,
    "autoAllowBashIfSandboxed": true
  }
}
```

This automatically enables sandboxing for all `claude -p` invocations, including autospec workflows.

### Via Interactive Command

In an interactive Claude session:

```
/sandbox
```

## Sandbox Configuration Options

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `sandbox.enabled` | boolean | `false` | Enable bash sandboxing |
| `sandbox.autoAllowBashIfSandboxed` | boolean | `true` | Auto-approve bash commands when sandboxed |
| `sandbox.excludedCommands` | string[] | `[]` | Commands that run outside sandbox (e.g., `["git", "docker"]`) |
| `sandbox.allowUnsandboxedCommands` | boolean | `true` | Allow `dangerouslyDisableSandbox` escape hatch |

### Network Settings

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `sandbox.network.allowUnixSockets` | string[] | `[]` | Unix sockets to allow (e.g., `["~/.ssh/agent-socket"]`) |
| `sandbox.network.allowLocalBinding` | boolean | `true` | Allow binding to local ports |

## Filesystem Isolation

By default, the sandbox provides:

- **Read-only**: Entire filesystem (except denied paths)
- **Read-write**: Current working directory and subdirectories
- **Denied**: Sensitive paths like `~/.ssh`, `~/.gnupg`, credentials files

## Network Isolation

Network access is controlled via a proxy server:

- Allowed domains can be accessed (configured via `WebFetch` permissions)
- New domain requests trigger permission prompts
- Applies to all subprocesses spawned by commands

**Limitation**: Domain filtering only—does not inspect traffic content. Broad domains like `github.com` may allow data exfiltration.

## Full Example Configuration

```json
{
  "sandbox": {
    "enabled": true,
    "autoAllowBashIfSandboxed": true,
    "excludedCommands": ["git", "docker"],
    "allowUnsandboxedCommands": false,
    "network": {
      "allowUnixSockets": [],
      "allowLocalBinding": true
    }
  },
  "permissions": {
    "allow": [
      "Bash(make:*)",
      "Bash(go:*)",
      "Bash(autospec:*)",
      "WebFetch(domain:github.com)"
    ],
    "deny": [],
    "ask": [
      "Bash(git reset:*)",
      "Bash(git revert:*)"
    ]
  }
}
```

## Using with autospec

### Automatic Configuration

Running `autospec init` automatically configures Claude Code permissions:

```bash
autospec init
# Output: Created .claude/settings.local.json with Claude Code permissions for autospec
```

This creates `.claude/settings.local.json` with the `Bash(autospec:*)` permission in the allow list.

**Behavior:**
- Creates settings file if missing
- Adds permission to existing settings without removing other configurations
- Warns if permission is explicitly denied (respects user security decisions)
- Skips if permission already configured

### Validating Configuration

Use `autospec doctor` to check Claude settings:

```bash
autospec doctor
# ✓ Claude settings: Bash(autospec:*) permission configured
```

### Recommended Setup

1. Enable sandbox in `.claude/settings.local.json`:

```json
{
  "sandbox": {
    "enabled": true,
    "autoAllowBashIfSandboxed": true
  }
}
```

2. Use `--dangerously-skip-permissions` in your config:

```yaml
# ~/.config/autospec/config.yml
claude_args:
  - -p
  - --dangerously-skip-permissions
  - --verbose
  - --output-format
  - stream-json
```

The sandbox provides OS-level isolation even when permission prompts are bypassed.

### Custom Claude Command

For piping through [cclean](https://github.com/ariel-frischer/claude-clean):

```yaml
custom_claude_cmd: "ANTHROPIC_API_KEY='' claude -p --dangerously-skip-permissions --verbose --output-format stream-json {{PROMPT}} | cclean"
```

## Manual bubblewrap Wrapper

For custom sandbox control outside Claude's built-in sandbox:

```bash
bwrap \
  --ro-bind / / \
  --bind $PWD $PWD \
  --dev /dev \
  --proc /proc \
  --tmpfs /tmp \
  --unshare-pid \
  -- claude -p --dangerously-skip-permissions "{{PROMPT}}"
```

Key flags:
- `--ro-bind / /`: Read-only root filesystem
- `--bind $PWD $PWD`: Read-write current directory
- `--tmpfs /tmp`: Isolated temp directory
- `--unshare-pid`: PID namespace isolation
- `--unshare-net`: Full network isolation (breaks most workflows)

## Security Considerations

### What Sandbox Protects Against

- Modifying files outside working directory
- Accessing sensitive config files (`~/.bashrc`, `~/.ssh/*`)
- Malicious prompt injection attacks
- Compromised dependencies

### Limitations

1. **Domain fronting**: Network filtering by domain only, not content
2. **Unix sockets**: `allowUnixSockets` can expose powerful services (e.g., Docker socket)
3. **Excluded commands**: Run outside sandbox entirely
4. **Escape hatch**: `dangerouslyDisableSandbox` bypasses sandbox (disable with `allowUnsandboxedCommands: false`)

### Best Practices

1. Start with minimal permissions, expand as needed
2. Don't allowlist broad domains unnecessarily
3. Review `excludedCommands` carefully
4. Set `allowUnsandboxedCommands: false` for stricter isolation
5. Use project-level settings (`.claude/settings.local.json`) over global

## Disabling for Enterprise

Administrators can prevent `--dangerously-skip-permissions`:

```json
{
  "permissions": {
    "disableBypassPermissionsMode": "disable"
  }
}
```

## References

- [bubblewrap GitHub](https://github.com/containers/bubblewrap)
- [Claude Code Sandboxing Docs](https://docs.anthropic.com/en/docs/claude-code/security#sandbox)
- [Arch Wiki bubblewrap Examples](https://wiki.archlinux.org/title/Bubblewrap/Examples)
