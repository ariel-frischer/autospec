---
title: Troubleshooting
parent: Guides
nav_order: 1
---

# Troubleshooting Guide

Common issues and their solutions when working with autospec.

## Quick reference

| Exit Code | Meaning | What to Do |
|-----------|---------|------------|
| 0 | Success | Nothing, all good |
| 1 | Validation failed | Check validation errors, retry |
| 2 | Retry exhausted | Reset retry state or fix root cause |
| 3 | Invalid arguments | Check command syntax |
| 4 | Missing dependencies | Run `autospec doctor` |
| 5 | Timeout | Increase timeout or break down task |

---

## Timeout issues

### Commands timeout unexpectedly

**Problem**: Commands fail with exit code 5 and timeout error message.

```
Error: command timed out after 5m0s: claude /autospec.prep ...
```

**Causes**:
- Timeout is set too low for the operation
- Large or complex features take longer than expected
- Network issues or slow API responses

**Solutions**:

1. **Increase the timeout**:
   ```bash
   # Quick fix: Set higher timeout
   export AUTOSPEC_TIMEOUT=1800  # 30 minutes

   # Or in config file
   echo 'timeout: 1800' > .autospec/config.yml
   ```

2. **Check feature complexity**:
   - Large feature descriptions take longer to process
   - Break down into smaller, focused features
   - Implement incrementally

3. **Verify system performance**: Run `top` to check system load, `ping api.anthropic.com` to check network connectivity.

### Timeout not being respected

**Problem**: Commands run longer than the configured timeout.

**Diagnostics**:
```bash
# 1. Check current timeout setting
autospec config show | grep timeout

# 2. Verify environment variable (must be all caps)
echo $AUTOSPEC_TIMEOUT

# 3. Check config files
cat .autospec/config.yml
cat ~/.config/autospec/config.yml

# 4. Verify autospec version
autospec version
```

**Solutions**:
- Ensure environment variable is `AUTOSPEC_TIMEOUT` (not `autospec_timeout`)
- Check config YAML is valid: `cat .autospec/config.yml`
- Reload shell: `exec $SHELL` or restart terminal
- Reinstall if necessary: `make install`

### Need different timeouts for different commands

**Problem**: Some commands need more time than others.

**Solution**: Use environment variable per command:
```bash
#!/bin/bash
# Specify: 5 minutes
AUTOSPEC_TIMEOUT=300 autospec specify "feature"

# Plan: 10 minutes
AUTOSPEC_TIMEOUT=600 autospec plan

# Implement: 30 minutes
AUTOSPEC_TIMEOUT=1800 autospec implement
```

Or create a wrapper: `run_with_timeout() { local t=$1; shift; AUTOSPEC_TIMEOUT=$t "$@"; }`

---

## Configuration issues

### Config file not loading

**Problem**: Changes to config.yml have no effect.

**Diagnostics**:
```bash
# 1. Check config syntax
cat .autospec/config.yml

# 2. Verify config location
ls -la .autospec/config.yml
ls -la ~/.config/autospec/config.yml

# 3. Test config loading
autospec config show
```

**Solutions**:
- Ensure YAML is valid syntax
- Check file permissions: `chmod 644 .autospec/config.yml`
- Verify config is in correct location
- Remember: environment variables override config files

### Environment variable not working

**Problem**: `AUTOSPEC_TIMEOUT` doesn't seem to apply.

**Diagnostics**:
```bash
# Check if variable is set
env | grep AUTOSPEC

# Check case sensitivity (must be uppercase)
echo $AUTOSPEC_TIMEOUT
echo $autospec_timeout  # This won't work!
```

**Solutions**:
```bash
# Correct: uppercase
export AUTOSPEC_TIMEOUT=600

# Incorrect: lowercase
export autospec_timeout=600  # Won't work!

# Persist in shell rc file
echo 'export AUTOSPEC_TIMEOUT=600' >> ~/.bashrc
source ~/.bashrc
```

---

## API billing issues

### Accidentally using API credits instead of subscription

**Problem**: Commands are consuming API credits instead of using your Claude Pro/Max subscription.

**Cause**: You have `ANTHROPIC_API_KEY` set in your shell environment (perhaps for other tools), and autospec is using it instead of your subscription.

**Solution**: autospec defaults to subscription mode (`use_subscription: true`), which should prevent this. If you're still seeing API charges:

1. **Verify subscription mode is enabled**:
   ```bash
   autospec config show | grep use_subscription
   ```

2. **Ensure you haven't disabled it**:
   ```yaml
   # Check .autospec/config.yml and ~/.config/autospec/config.yml
   # This should be true (or not set, as true is the default)
   use_subscription: true
   ```

### Want to use API credits instead of subscription

**Problem**: You want to use API billing, but autospec keeps using subscription.

**Solution**: Disable subscription mode:

```yaml
# In .autospec/config.yml or ~/.config/autospec/config.yml
use_subscription: false
```

Then ensure `ANTHROPIC_API_KEY` is set in your shell.

---

## Workflow execution issues

### Retry limit exhausted (exit code 2)

**Problem**: Command fails repeatedly and exhausts retries.

**Symptoms**:
```
retry limit exhausted
Exit code: 2
```

**Solutions**:

1. **Check retry state**:
   ```bash
   cat ~/.autospec/state/retry.json | jq .
   ```

2. **Reset retry count**:
   ```bash
   # Manual reset
   rm ~/.autospec/state/retry.json

   # Or reset specific phase (if implemented)
   autospec reset specify
   ```

3. **Increase max retries**:
   ```bash
   autospec --max-retries 5 workflow "feature"
   ```

4. **Fix underlying issue**:
   - Review error messages from previous attempts
   - Check if validation is failing
   - Verify dependencies are installed

### Validation failed (exit code 1)

**Problem**: Generated files don't pass validation.

**Common Causes**:
- spec.yaml missing or incomplete
- plan.yaml missing required sections
- tasks.yaml improperly formatted

**Solutions**:
```bash
# Check what's missing
ls specs/FEATURE-NAME/

# Run validation manually
autospec status

# Re-run the failed phase
autospec plan  # or specify, tasks, etc.
```

### Missing dependencies (exit code 4)

**Problem**: Required tools not found.

**Symptoms**:
```
missing dependencies
Exit code: 4
```

**Solutions**:
```bash
# Run doctor to diagnose
autospec doctor

# Install Claude CLI - see https://claude.ai/download
```

### Claude permission denied / command blocked

**Problem**: Claude blocks commands (can't respond to approval prompts).

**Solutions**: Allow commands in `~/.claude/settings.json`:
```json
{
  "permissions": {
    "allow": ["Bash(mkdir:*)", "Edit", "Write", "Read"]
  }
}
```

Or add `--dangerously-skip-permissions` to `claude_args`--enable Claude's sandbox first (`/sandbox`, uses [bubblewrap](https://github.com/containers/bubblewrap) on Linux).

{: .warning }
> **WARNING**: Bypasses ALL safety checks--never use with API keys/credentials/production data.

---

## Prerequisite validation errors

Autospec validates that required artifacts exist before executing commands. This prevents wasted API costs when prerequisites are missing.

### Missing constitution error

**Problem**: Command fails immediately with constitution error.

**Symptoms**:
```
Error: Project constitution not found.

A constitution is required before running any workflow stages.
The constitution defines your project's principles and guidelines.

To create a constitution, run:
  autospec constitution
```

**Solution**:
```bash
# Create a constitution for your project
autospec constitution
```

### Missing spec.yaml error

**Problem**: Commands like `plan`, `clarify`, or `checklist` fail because spec.yaml is missing.

**Symptoms**:
```
Error: spec.yaml not found.

Run 'autospec specify' first to create this file.
```

**Solution**:
```bash
# Create the specification first
autospec specify "your feature description"

# Or run the full prep workflow
autospec prep "your feature description"
```

### Missing plan.yaml error

**Problem**: `tasks` command fails because plan.yaml is missing.

**Symptoms**:
```
Error: plan.yaml not found.

Run 'autospec plan' first to create this file.
```

**Solution**:
```bash
# Create the plan first
autospec plan

# Or run spec and plan together
autospec run -sp
```

### Missing tasks.yaml error

**Problem**: `implement` command fails because tasks.yaml is missing.

**Symptoms**:
```
Error: tasks.yaml not found.

Run 'autospec tasks' first to create this file.
```

**Solution**:
```bash
# Create tasks first
autospec tasks

# Or run the full prep workflow
autospec prep "feature"

# Or run spec, plan, and tasks together
autospec run -spt
```

### Stage dependency diagram

The following diagram shows which artifacts each stage requires and produces:

```
constitution ──> constitution.yaml
      │
   specify ────> spec.yaml
      │
    plan ──────> plan.yaml
      │
    tasks ─────> tasks.yaml
      │
  implement

Optional stages:
- clarify:   requires spec.yaml
- checklist: requires spec.yaml
- analyze:   requires spec.yaml, plan.yaml, tasks.yaml
```

### Understanding run command validation

The `run` command performs "smart" validation. It only checks for artifacts that won't be created by earlier stages in your selection:

| Command | What's Validated | Why |
|---------|-----------------|-----|
| `autospec run -spt` | Constitution only | `specify` creates spec.yaml, `plan` creates plan.yaml |
| `autospec run -pti` | spec.yaml | `plan` needs spec.yaml, but produces plan.yaml |
| `autospec run -ti` | plan.yaml | `tasks` needs plan.yaml, produces tasks.yaml |
| `autospec run -i` | tasks.yaml | `implement` needs tasks.yaml |
| `autospec run -a` | Constitution only | Full chain produces all artifacts |

{: .tip }
> Use `autospec run -spt` to go from nothing to tasks.yaml in one command.

---

## Blocked tasks workflow

When Claude encounters a task it can't complete, it marks the task as `Blocked` with a `blocked_reason`.

### Understanding blocked tasks

**What happens:**
1. `autospec implement` runs
2. Claude hits a wall (too complex, needs clarification, external dependency)
3. Claude marks task as `Blocked` with `blocked_reason` and optionally `notes`
4. Implementation stops or continues with other tasks

**Check blocked tasks:**
```bash
# See status including blocked tasks
autospec st

# Output shows:
# Tasks: 15 total | 10 completed | 2 in-progress | 1 pending | 2 blocked
#
# Blocked Tasks:
#   T015: Implement complex parsing logic
#         → Uncertain about edge cases, need human review
```

### Workflow for resolving blocked tasks

**Recommended flow:**

```
autospec implement
       │
Task T015 gets blocked
       │
autospec st  (see what's blocked and why)
       │
claude  (interactive session - work on T015 with Claude)
       │
Either:
  - You/Claude fix it   → autospec task complete T015
  - Need to retry       → autospec task unblock T015
       │
autospec implement  (continues with remaining tasks)
```

{: .note }
> Blocked tasks are better suited for interactive Claude sessions, not automated retries. The blocking often requires back-and-forth dialogue, human judgment or clarification, and manual fixes or environment changes.

### Commands for managing blocked tasks

```bash
# View blocked tasks with reasons
autospec st

# Unblock a task (sets to Pending for retry)
autospec task unblock T015

# Mark as complete (if you fixed it manually)
autospec task complete T015

# Block a task with reason (useful for manual blocking)
autospec task block T015 --reason "Waiting for API spec from backend team"
```

### Using blocked_reason vs notes

Both fields help document why a task is blocked:

| Field | Purpose | Example |
|-------|---------|---------|
| `blocked_reason` | Brief explanation of the blocker | "Need human review - edge cases unclear" |
| `notes` | Detailed context, attempts made, questions | "Tried X approach, failed because Y. Questions: 1) How should empty input be handled?" |

### Common blocking scenarios

| Scenario | blocked_reason example | Resolution |
|----------|----------------------|------------|
| External dependency | "Waiting for auth service API spec" | Wait for dependency, then unblock |
| Too complex | "Task too large, needs breakdown" | Split task in tasks.yaml, unblock |
| Needs clarification | "Unclear requirement - multiple valid approaches" | Discuss in interactive Claude, then complete |
| Environment issue | "Database not running locally" | Fix environment, then unblock |
| Partial progress | "80% done, stuck on edge case" | Interactive Claude session to finish |

### Iterating with Claude on blocked tasks

For complex blocked tasks, use interactive Claude:

```bash
# Start interactive session
claude

# In the session, reference the blocked task:
> Look at T015 in specs/my-feature/tasks.yaml.
> It's blocked because of edge case uncertainty.
> Let's work through this together.
```

After resolving:
```bash
# If Claude completed the task in interactive mode
autospec task complete T015

# If you want autospec to retry it
autospec task unblock T015
autospec implement
```

---

## Performance issues

### Commands running very slowly

**Problem**: Commands take much longer than expected, even without timing out.

**Diagnostics**:
```bash
# Time a command
time autospec specify "test"

# Check system resources
top
df -h  # Check disk space
free -h  # Check memory
```

**Solutions**:

1. **Check network**:
   ```bash
   # Test API connectivity
   curl -I https://api.anthropic.com
   ```

2. **Reduce workload**:
   - Use smaller feature descriptions
   - Break large features into smaller ones
   - Implement incrementally

3. **Check for system issues**:
   - High CPU/memory usage
   - Low disk space
   - Network congestion

---

## Git integration issues

### Spec detection fails

**Problem**: autospec can't detect current spec from git branch.

**Symptoms**:
```
failed to detect current spec
```

**Solutions**:
```bash
# 1. Check git branch name
git branch --show-current

# 2. Branch should match pattern: NNN-feature-name
git checkout -b 004-my-feature

# 3. Or specify spec name explicitly
autospec plan 003-timeout

# 4. Check if in git repo
git status
```

### Branch name doesn't match spec

**Problem**: Branch is `003-feature` but spec directory is `003-command-timeout`.

**Solution**:
- Rename branch to match spec directory:
  ```bash
  git branch -m 003-command-timeout
  ```
- Or explicitly specify spec name:
  ```bash
  autospec plan 003-command-timeout
  ```

---

## Notification issues

### Notifications not appearing

**Problem**: Notifications are enabled but don't appear.

**Diagnostics**:
```bash
# 1. Check if notifications are enabled in config
autospec config show | grep -A10 notifications

# 2. Verify you're in an interactive session
tty && echo "Interactive" || echo "Non-interactive"

# 3. Check for CI environment variables
env | grep -E "^(CI|GITHUB_ACTIONS|GITLAB_CI|JENKINS)="
```

**Solutions**:

1. **Ensure notifications are enabled**:
   ```yaml
   # .autospec/config.yml
   notifications:
     enabled: true
   ```

2. **Check platform-specific tools**:
   - **macOS**: `osascript` and `afplay` (standard on all versions)
   - **Linux**: Install `notify-send` (`sudo apt install libnotify-bin` or equivalent)

3. **Verify display environment (Linux)**:
   ```bash
   echo $DISPLAY    # X11
   echo $WAYLAND_DISPLAY  # Wayland
   ```
   At least one must be set for notifications to work.

### No sound notifications

**Problem**: Visual notifications work but no sound plays.

**Platform-specific solutions**:

**macOS**:
```bash
# Check if afplay is available
which afplay

# Test default sound
afplay /System/Library/Sounds/Glass.aiff
```

**Linux**:
```bash
# Check if paplay is available (PulseAudio/PipeWire)
which paplay

# Test a sound file (must provide custom file)
paplay /path/to/your/sound.wav
```

{: .note }
> Linux has no default notification sound. You must configure `sound_file` for audio notifications.

---

## Debugging techniques

### Enable debug mode

```bash
# Global debug flag
autospec --debug prep "feature"

# Per-command debugging
autospec -d plan
```

### Check configuration

```bash
# View all loaded configuration
autospec config show

# Check specific values
autospec config show | grep timeout
autospec config show | grep max_retries
```

### Inspect state

```bash
# Retry state
cat ~/.autospec/state/retry.json | jq .

# Config files
cat .autospec/config.yml
cat ~/.config/autospec/config.yml
```

### Capture full output

```bash
# Capture stdout and stderr
autospec prep "feature" 2>&1 | tee full-output.log

# Capture with timestamps
autospec prep "feature" 2>&1 | ts '[%Y-%m-%d %H:%M:%S]' | tee output.log
```

---

## Getting help

### Information to include when reporting issues

1. **Version information**:
   ```bash
   autospec version
   ```

2. **Configuration**:
   ```bash
   autospec config show
   ```

3. **Environment**:
   ```bash
   echo "OS: $(uname -s)"
   echo "Go version: $(go version)"
   echo "Shell: $SHELL"
   env | grep AUTOSPEC
   ```

4. **Error output**:
   - Full error message
   - Exit code
   - Command that failed

5. **Reproduction steps**:
   - Exact commands run
   - Expected vs actual behavior

### Where to get help

- **Documentation**: This site and [CLAUDE.md](https://github.com/ariel-frischer/autospec/blob/main/CLAUDE.md)
- **Issues**: [Report bugs on GitHub](https://github.com/ariel-frischer/autospec/issues)
- **Logs**: Check `~/.autospec/state/` for state files

---

## Common commands reference

```bash
# Check status
autospec doctor
autospec status
autospec config show

# Reset state
rm ~/.autospec/state/retry.json

# Debug
autospec --debug <command>
autospec --verbose <command>

# Timeout control
export AUTOSPEC_TIMEOUT=600
AUTOSPEC_TIMEOUT=0 autospec <command>  # Disable timeout
```

### Configuration locations

```
~/.config/autospec/config.yml  # Global config (XDG compliant)
.autospec/config.yml           # Local config (project)
~/.autospec/state/             # State files
AUTOSPEC_*                     # Environment variables
```

---

## See Also

- [FAQ](faq) - Frequently asked questions
- [CLI Reference](/autospec/reference/cli) - Complete command documentation
- [Configuration Reference](/autospec/reference/configuration) - All configuration options
- [Architecture Internals](/autospec/architecture/internals) - How spec detection and retry work
