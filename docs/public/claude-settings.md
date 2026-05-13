# Claude Code Settings & Sandboxing

*Last updated: 2026-05-13*

> **Verified with:** Claude Code 2.1.139 (2026-05-13)

Configuration guide for Claude Code settings relevant to autospec workflows.

> **Quick Start**: Run `autospec init` to automatically configure Claude Code permissions.

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

Running `autospec init` configures Claude Code permissions and sandbox:

```bash
autospec init                # Permissions → global (~/.claude/settings.json)
autospec init --project      # Permissions → project (.claude/settings.local.json)
```

**What gets configured where:**

| Config Type | Location | Why |
|-------------|----------|-----|
| **Permissions** (`Bash(autospec:*)`, `Write`, `Edit`) | Global by default | Works across all projects |
| **Sandbox** (`additionalAllowWritePaths`) | Always project-level | Uses relative paths (`.autospec`, `specs`) |

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

2. Enable `skip_permissions` in your config:

```bash
autospec config toggle skip_permissions
# or: autospec config set skip_permissions true
```

The sandbox provides OS-level isolation even when permission prompts are bypassed.

## Sandbox vs --dangerously-skip-permissions

These are **two separate security layers** that work independently:

| Layer | What it does | Enforced by |
|-------|--------------|-------------|
| **Sandbox** | Restricts filesystem to CWD, limits network to allowed domains | OS-level (bubblewrap/seatbelt) |
| **Permission prompts** | Requires user approval for file edits, bash commands, etc. | Claude Code CLI |

### Key Interaction Behavior

| Sandbox | --dangerously-skip-permissions | Result |
|---------|-------------------------------|--------|
| Enabled | Yes | Commands auto-run but **cannot escape CWD** - safe automation |
| Enabled | No | Commands prompt for approval, stay confined to CWD |
| Disabled | Yes | Commands auto-run with **full system access** - dangerous |
| Disabled | No | Commands prompt for approval, have full system access if approved |

**Critical insight**: `--dangerously-skip-permissions` only skips the permission prompts—it does **not** bypass sandbox restrictions. When sandbox is enabled, Claude cannot:

- Edit files outside the current working directory
- Access `~/.ssh`, `~/.bashrc`, or other sensitive paths
- Connect to non-allowed network domains

This makes **sandbox + skip-permissions** a safe combination for automation: you get unattended execution while maintaining OS-level isolation.

### Recommended Full Automation Setup

```yaml
# ~/.config/autospec/config.yml
custom_agent:
  command: "claude"
  args:
    - "-p"
    - "--dangerously-skip-permissions"
    - "--verbose"
    - "--output-format"
    - "stream-json"
    - "{{PROMPT}}"
  post_processor: "cclean"
```

Combined with sandbox enabled in `.claude/settings.local.json`:

```json
{
  "sandbox": {
    "enabled": true,
    "autoAllowBashIfSandboxed": true
  }
}
```

This provides:
- Unattended automation (no permission prompts)
- OS-level filesystem isolation (can't escape project directory)
- Readable output via cclean

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
5. Use global settings for permissions (shared across projects), project-level for sandbox (relative paths)

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
