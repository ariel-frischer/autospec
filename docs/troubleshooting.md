# Troubleshooting Guide

## Common Issues

### Timeout Issues

#### Commands timeout unexpectedly

**Problem**: Commands fail with exit code 5 and timeout error message.

```
Error: command timed out after 5m0s: claude /autospec.workflow ...
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

**Solutions**: Allow commands in `~/.claude/settings.json`: `{"permissions":{"allow":["Bash(mkdir:*)", "Edit", "Write", "Read"]}}`. Or add `--dangerously-skip-permissions` to `claude_args` (sandbox recommended). **WARNING**: bypasses ALL safety checks—never use with API keys/credentials/production data.

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
autospec --verbose workflow "feature"

# Enable debug logging
autospec --debug workflow "feature"

# Check if output is being redirected
autospec workflow "feature" 2>&1 | tee output.log
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
autospec --debug workflow "feature"

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
autospec workflow "feature" 2>&1 | tee full-output.log

# Capture with timestamps
autospec workflow "feature" 2>&1 | ts '[%Y-%m-%d %H:%M:%S]' | tee output.log
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
