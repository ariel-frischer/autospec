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

3. **Verify system performance**:
   ```bash
   # Check if system is under load
   top

   # Check network connectivity
   ping api.anthropic.com
   ```

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

Or create a wrapper function:
```bash
run_with_timeout() {
    local timeout=$1
    shift
    AUTOSPEC_TIMEOUT=$timeout "$@"
}

run_with_timeout 300 autospec specify "feature"
run_with_timeout 1800 autospec implement
```

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

**Solutions**: Allow commands in `~/.claude/settings.json`: `{"permissions":{"allow":["Bash(mkdir:*)", "Edit", "Write", "Read"]}}`. Or add `--dangerously-skip-permissions` to `claude_args`—enable Claude's sandbox first (`/sandbox`, uses [bubblewrap](https://github.com/containers/bubblewrap) on Linux). **WARNING**: bypasses ALL safety checks—never use with API keys/credentials/production data.

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
   - **Windows**: PowerShell (standard on all versions)

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

#### Windows
```powershell
# Test system beep
[Console]::Beep(800, 200)
```

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
autospec full "feature"
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

### Windows PowerShell execution policy

**Problem**: Toast notifications fail on Windows.

**Error**: `Running scripts is disabled on this system`

**Solutions**:
```powershell
# Option 1: Run as administrator and enable scripts
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser

# Option 2: autospec uses -ExecutionPolicy Bypass, which should work
# If still failing, check for group policy restrictions
Get-ExecutionPolicy -List
```

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
autospec --debug full "test"
```

**Solutions**:
- If notifications are causing delays, switch to `type: visual` (sound playback can take longer)
- Disable notifications for time-critical operations:
  ```bash
  AUTOSPEC_NOTIFICATIONS_ENABLED=false autospec full "feature"
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
