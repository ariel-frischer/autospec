# Troubleshooting Guide

## Common Issues

### Timeout Issues

#### Commands timeout unexpectedly

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

#### Timeout not being respected

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

#### Need different timeouts for different commands

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

### Configuration Issues

#### Config file not loading

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

#### Environment variable not working

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

### API Billing Issues

#### Accidentally using API credits instead of subscription

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

3. **If using custom_agent, check for explicit API key**:
   ```yaml
   # Remove or leave empty the ANTHROPIC_API_KEY in custom_agent.env
   custom_agent:
     env:
       ANTHROPIC_API_KEY: ""  # Empty = use subscription
   ```

#### Want to use API credits instead of subscription

**Problem**: You want to use API billing, but autospec keeps using subscription.

**Solution**: Disable subscription mode:

```yaml
# In .autospec/config.yml or ~/.config/autospec/config.yml
use_subscription: false
```

Then ensure `ANTHROPIC_API_KEY` is set in your shell:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

### Workflow Execution Issues

#### Retry limit exhausted (exit code 2)

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

#### Validation failed (exit code 1)

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

#### Missing dependencies (exit code 4)

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

#### Claude permission denied / command blocked

**Problem**: Claude blocks commands (can't respond to approval prompts).

**Solutions**: Allow commands in `~/.claude/settings.json`: `{"permissions":{"allow":["Bash(mkdir:*)", "Edit", "Write", "Read"]}}`. Or configure `custom_agent` with `--dangerously-skip-permissions` in the args—enable Claude's sandbox first (`/sandbox`, uses [bubblewrap](https://github.com/containers/bubblewrap) on Linux). **WARNING**: bypasses ALL safety checks—never use with API keys/credentials/production data.

### Prerequisite Validation Errors

Autospec validates that required artifacts exist before executing commands. This prevents wasted API costs when prerequisites are missing.

#### Missing constitution error

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

The constitution defines your project's coding standards, architectural principles, and guidelines. It must exist before running any other stage commands.

#### Missing spec.yaml error

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

#### Missing plan.yaml error

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

#### Missing tasks.yaml error

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

#### Multiple missing artifacts (analyze command)

**Problem**: `analyze` command lists multiple missing files.

**Symptoms**:
```
Error: Missing required artifacts:
  - spec.yaml
  - plan.yaml
  - tasks.yaml

Run the following commands to create them:
  autospec specify
  autospec plan
  autospec tasks
```

**Solution**:
```bash
# Run the full prep workflow to create all artifacts
autospec prep "feature description"

# Or run stages individually
autospec specify "feature"
autospec plan
autospec tasks
```

#### Stage Dependency Diagram

The following diagram shows which artifacts each stage requires and produces:

```
constitution ──> constitution.yaml
      ↓
   specify ────> spec.yaml
      ↓
    plan ──────> plan.yaml
      ↓
    tasks ─────> tasks.yaml
      ↓
  implement

Optional stages:
- clarify:   requires spec.yaml
- checklist: requires spec.yaml
- analyze:   requires spec.yaml, plan.yaml, tasks.yaml
```

#### Understanding run command validation

The `run` command performs "smart" validation. It only checks for artifacts that won't be created by earlier stages in your selection:

| Command | What's Validated | Why |
|---------|-----------------|-----|
| `autospec run -spt` | Constitution only | `specify` creates spec.yaml, `plan` creates plan.yaml |
| `autospec run -pti` | spec.yaml | `plan` needs spec.yaml, but produces plan.yaml |
| `autospec run -ti` | plan.yaml | `tasks` needs plan.yaml, produces tasks.yaml |
| `autospec run -i` | tasks.yaml | `implement` needs tasks.yaml |
| `autospec run -a` | Constitution only | Full chain produces all artifacts |

**Tip**: Use `autospec run -spt` to go from nothing to tasks.yaml in one command.

### Blocked Tasks Workflow

When Claude encounters a task it can't complete, it marks the task as `Blocked` with a `blocked_reason`. This section covers how to handle blocked tasks effectively.

#### Understanding blocked tasks

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

#### Workflow for resolving blocked tasks

**Recommended flow:**

```
autospec implement
       ↓
Task T015 gets blocked
       ↓
autospec st  (see what's blocked and why)
       ↓
claude  (interactive session - work on T015 with Claude)
       ↓
Either:
  • You/Claude fix it   → autospec task complete T015
  • Need to retry       → autospec task unblock T015
       ↓
autospec implement  (continues with remaining tasks)
```

**Key insight:** Blocked tasks are better suited for interactive Claude sessions, not automated retries. The blocking often requires:
- Back-and-forth dialogue
- Human judgment or clarification
- Manual fixes or environment changes

#### Commands for managing blocked tasks

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

#### Using blocked_reason vs notes

Both fields help document why a task is blocked:

| Field | Purpose | Example |
|-------|---------|---------|
| `blocked_reason` | Brief explanation of the blocker | "Need human review - edge cases unclear" |
| `notes` | Detailed context, attempts made, questions | "Tried X approach, failed because Y. Questions: 1) How should empty input be handled? 2) Is retry logic correct?" |

**Example in tasks.yaml:**
```yaml
- id: "T015"
  title: "Implement retry logic"
  status: "Blocked"
  blocked_reason: "Uncertain about edge cases, need human review"
  notes: |
    Attempted implementation but unsure about:
    - What should happen with empty input?
    - Is the retry logic correct for timeouts?
    - Tests pass but coverage seems low

    Tried exponential backoff but unclear if 30s max is correct.
```

#### Common blocking scenarios

| Scenario | blocked_reason example | Resolution |
|----------|----------------------|------------|
| External dependency | "Waiting for auth service API spec" | Wait for dependency, then unblock |
| Too complex | "Task too large, needs breakdown" | Split task in tasks.yaml, unblock |
| Needs clarification | "Unclear requirement - multiple valid approaches" | Discuss in interactive Claude, then complete |
| Environment issue | "Database not running locally" | Fix environment, then unblock |
| Partial progress | "80% done, stuck on edge case" | Interactive Claude session to finish |

#### Iterating with Claude on blocked tasks

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

#### Preventing wasted retries

Blocked tasks should NOT be automatically retried - they need human intervention. The workflow is:

1. **Don't** just rerun `autospec implement` hoping it works
2. **Do** check `autospec st` to understand why it's blocked
3. **Do** resolve the underlying issue (interactive session, manual fix, wait for dependency)
4. **Then** unblock and continue

### Performance Issues

#### Commands running very slowly

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

#### Validation functions taking too long

**Problem**: Validation steps are slow (<10ms requirement not met).

**Diagnostics**:
```bash
# Run with debug logging
autospec --debug plan

# Profile Go code (for developers)
go test -bench=. -cpuprofile=cpu.prof ./internal/validation/
```

**Solutions**:
- Report performance regression as issue
- Check if validation files are extremely large
- Verify disk I/O is not bottleneck

### Git Integration Issues

#### Spec detection fails

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

#### Branch name doesn't match spec

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

### Output and Logging Issues

#### No output during execution

**Problem**: Command runs but shows no progress.

**Solutions**:
```bash
# Enable verbose output
autospec --verbose prep "feature"

# Enable debug logging
autospec --debug prep "feature"

# Check if output is being redirected
autospec prep "feature" 2>&1 | tee output.log
```

#### Error messages unclear

**Problem**: Error messages don't provide enough context.

**Solutions**:
```bash
# Run with debug mode
autospec --debug <command>

# Check logs
cat ~/.autospec/state/*.log

# Review retry state
cat ~/.autospec/state/retry.json | jq .
```

## Debugging Techniques

### Enable Debug Mode

```bash
# Global debug flag
autospec --debug prep "feature"

# Per-command debugging
autospec -d plan
```

### Check Configuration

```bash
# View all loaded configuration
autospec config show

# Check specific values
autospec config show | grep timeout
autospec config show | grep max_retries
```

### Inspect State

```bash
# Retry state
cat ~/.autospec/state/retry.json | jq .

# Config files
cat .autospec/config.yml
cat ~/.config/autospec/config.yml
```

### Test Individual Components

```bash
# Test Claude CLI directly
claude --version
echo "test" | claude

# Test config loading
autospec config show

# Verify commands are installed
ls .claude/commands/autospec.*.md
```

### Capture Full Output

```bash
# Capture stdout and stderr
autospec prep "feature" 2>&1 | tee full-output.log

# Capture with timestamps
autospec prep "feature" 2>&1 | ts '[%Y-%m-%d %H:%M:%S]' | tee output.log
```

## Getting Help

### Information to Include When Reporting Issues

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

### Where to Get Help

- **Documentation**: Check [CLAUDE.md](../CLAUDE.md) and [docs/](.)
- **Issues**: Report bugs at repository issues page
- **Logs**: Check `~/.autospec/state/` for state files

## Notification Issues

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

#### macOS
```bash
# Check if afplay is available
which afplay

# Test default sound
afplay /System/Library/Sounds/Glass.aiff
```

#### Linux
```bash
# Check if paplay is available (PulseAudio/PipeWire)
which paplay

# Test a sound file (must provide custom file)
paplay /path/to/your/sound.wav
```

**Note**: Linux has no default notification sound. You must configure `sound_file` for audio notifications.

### Custom sound file not playing

**Problem**: Configured custom sound file doesn't play.

**Diagnostics**:
```bash
# Check if file exists
ls -la /path/to/your/sound.wav

# Check file extension is supported
# Supported: .wav, .mp3, .aiff, .aif, .ogg, .flac, .m4a
```

**Solutions**:
1. Verify the file path is absolute
2. Ensure file format is supported
3. Check file permissions (must be readable)
4. If file is invalid, autospec falls back to system default (or no sound on Linux)

### Notifications in CI/CD pipelines

**Problem**: Notifications trigger unexpectedly in CI environment.

**Behavior**: autospec automatically detects CI environments and disables notifications. The following environment variables are checked:
- `CI`
- `GITHUB_ACTIONS`
- `GITLAB_CI`
- `CIRCLECI`
- `TRAVIS`
- `JENKINS_URL`
- `BUILDKITE`
- `DRONE`
- `TEAMCITY_VERSION`
- `TF_BUILD` (Azure DevOps)
- `BITBUCKET_PIPELINES`
- `CODEBUILD_BUILD_ID` (AWS CodeBuild)
- And others

**Solution**: If you need to force enable notifications in CI (not recommended):
```bash
# Temporarily unset CI variable
unset CI
autospec all "feature"
```

### Notifications in headless/SSH sessions

**Problem**: No notifications when running over SSH or in headless mode.

**Behavior**: autospec checks for TTY availability. Non-interactive sessions skip notifications to avoid errors.

**Diagnostics**:
```bash
# Check if session is interactive
if [ -t 0 ]; then echo "Interactive"; else echo "Non-interactive"; fi
```

**Solutions**:
1. For SSH sessions that need notifications, use `ssh -t` for pseudo-terminal allocation
2. On headless servers, notifications are intentionally skipped (no display to show them)

### Notifications not appearing on Linux

**Problem**: Linux desktop doesn't show notifications.

**Solutions**:

1. **Install notify-send**:
   ```bash
   # Debian/Ubuntu
   sudo apt install libnotify-bin

   # Fedora
   sudo dnf install libnotify

   # Arch
   sudo pacman -S libnotify
   ```

2. **Verify notification daemon is running**:
   ```bash
   # Check for notification daemon
   pgrep -l notification
   # or
   pgrep -l dunst  # if using dunst
   ```

3. **Check display environment**:
   ```bash
   # For X11
   export DISPLAY=:0

   # For Wayland
   # Ensure WAYLAND_DISPLAY is set by your compositor
   ```

### Notification timeout/latency

**Problem**: Command execution seems slow due to notifications.

**Behavior**: Notifications dispatch asynchronously with a 100ms timeout. They should not block command execution.

**Diagnostics**:
```bash
# Run with debug to see notification timing
autospec --debug all "test"
```

**Solutions**:
- If notifications are causing delays, switch to `type: visual` (sound playback can take longer)
- Disable notifications for time-critical operations:
  ```bash
  AUTOSPEC_NOTIFICATIONS_ENABLED=false autospec all "feature"
  ```

## Quick Reference

### Exit Codes

| Code | Meaning | What to Do |
|------|---------|------------|
| 0 | Success | Nothing, all good |
| 1 | Validation failed | Check validation errors, retry |
| 2 | Retry exhausted | Reset retry state or fix root cause |
| 3 | Invalid arguments | Check command syntax |
| 4 | Missing dependencies | Run `autospec doctor` |
| 5 | Timeout | Increase timeout or break down task |

### Common Commands

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

### Configuration Locations

```
~/.config/autospec/config.yml  # Global config (XDG compliant)
.autospec/config.yml           # Local config (project)
~/.autospec/state/             # State files
AUTOSPEC_*                     # Environment variables
```
