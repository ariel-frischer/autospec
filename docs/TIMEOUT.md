# Command Timeout Configuration

## Overview

The autospec CLI supports configurable timeouts for Claude command execution to prevent indefinite hangs in automated workflows. When a command exceeds the configured timeout, it is automatically terminated and returns exit code 5.

## Quick Start

### Set Timeout via Environment Variable

```bash
# Set 10-minute timeout for all commands
export AUTOSPEC_TIMEOUT=600

# Run commands with timeout
autospec prep "Add user authentication"
autospec plan
autospec implement
```

### Set Timeout via Configuration File

**Local config** (project-specific):
```bash
mkdir -p .autospec
cat > .autospec/config.yml << EOF
timeout: 600
EOF
```

**Global config** (all projects):
```bash
mkdir -p ~/.config/autospec
cat > ~/.config/autospec/config.yml << EOF
timeout: 1800
EOF
```

## Configuration Details

### Valid Timeout Values

| Value | Meaning | Common Use Cases |
|-------|---------|------------------|
| `0` | No timeout | Development, interactive use, debugging |
| `30` | 30 seconds | Quick commands, testing timeout behavior |
| `300` | 5 minutes | Most workflow commands |
| `600` | 10 minutes | Complex planning/implementation |
| `1800` | 30 minutes | Large features |
| `2400` | 40 minutes (default) | Full workflows |
| `3600` | 1 hour | Very complex tasks |
| `86400` | 24 hours | Extremely long-running operations |
| `604800` | 7 days (maximum) | Extended background processing |

### Configuration Priority

Configuration sources are applied in this order (highest to lowest priority):

1. **Environment Variable**: `AUTOSPEC_TIMEOUT`
2. **Local Config**: `.autospec/config.yml` (current directory)
3. **Global Config**: `~/.config/autospec/config.yml` (home directory)
4. **Default**: `2400` (40 minutes)

### Example: Multiple Configuration Sources

```bash
# Global config: 30 minutes
echo 'timeout: 1800' > ~/.config/autospec/config.yml

# Local config: 10 minutes (overrides global)
echo 'timeout: 600' > .autospec/config.yml

# Environment variable: 5 minutes (overrides all)
export AUTOSPEC_TIMEOUT=300

# Result: timeout = 300 seconds
```

## Usage Examples

### Basic Usage

```bash
# Default: no timeout
autospec prep "feature"

# With 5-minute timeout
AUTOSPEC_TIMEOUT=300 autospec prep "feature"

# With 30-minute timeout
AUTOSPEC_TIMEOUT=1800 autospec implement
```

### Different Timeouts per Command

```bash
#!/bin/bash
# Quick commands: 5 minutes
AUTOSPEC_TIMEOUT=300 autospec specify "Add user profile feature"

# Planning: 10 minutes
AUTOSPEC_TIMEOUT=600 autospec plan

# Implementation: 30 minutes
AUTOSPEC_TIMEOUT=1800 autospec implement
```

### CI/CD Pipeline

```bash
#!/bin/bash
set -e

# Prevent indefinite hangs in CI
export AUTOSPEC_TIMEOUT=600  # 10 minutes max

autospec prep "feature description"
exit_code=$?

if [ $exit_code -eq 5 ]; then
    echo "ERROR: Command timed out after 10 minutes"
    echo "Consider:"
    echo "  1. Breaking the feature into smaller pieces"
    echo "  2. Increasing the timeout for this specific feature"
    echo "  3. Investigating why the command is taking so long"
    exit 1
fi
```

### Retry with Increased Timeout

```bash
#!/bin/bash
timeout=300

for attempt in 1 2 3; do
    echo "Attempt $attempt with ${timeout}s timeout"

    AUTOSPEC_TIMEOUT=$timeout autospec prep "complex feature"
    exit_code=$?

    if [ $exit_code -eq 5 ]; then
        echo "Timed out. Doubling timeout and retrying..."
        timeout=$((timeout * 2))
    elif [ $exit_code -eq 0 ]; then
        echo "Success!"
        exit 0
    else
        echo "Failed with exit code $exit_code"
        exit $exit_code
    fi
done

echo "All retries exhausted"
exit 1
```

## Timeout Behavior

### What Happens When a Timeout Occurs

1. **Process Termination**: The running command is sent a `SIGKILL` signal
2. **Immediate Stop**: The process cannot ignore this signal and terminates immediately
3. **Error Return**: A `TimeoutError` is returned with details
4. **Exit Code 5**: The CLI exits with code 5 (specific to timeouts)
5. **Helpful Message**: Error message includes:
   - Timeout duration
   - Command that timed out
   - Suggestions for increasing the timeout

### Example Timeout Output

```
Error: command timed out after 5m0s: claude /autospec.implement (hint: increase timeout in config)

To increase the timeout, set AUTOSPEC_TIMEOUT environment variable or update config.yml:
  export AUTOSPEC_TIMEOUT=600  # 10 minutes
  or edit .autospec/config.yml and set "timeout: 600"
```

## Verification and Testing

### Check Current Timeout Configuration

```bash
# View all configuration including timeout
autospec config show

# Check environment variable
echo $AUTOSPEC_TIMEOUT

# Check local config
cat .autospec/config.yml

# Check global config
cat ~/.config/autospec/config.yml
```

### Test Timeout Behavior

```bash
# Test with very short timeout (should timeout quickly)
AUTOSPEC_TIMEOUT=1 autospec specify "test feature"

# Expected: Times out after ~1 second with exit code 5
echo "Exit code: $?"  # Should be 5
```

### Disable Timeout Temporarily

```bash
# Disable timeout for debugging
AUTOSPEC_TIMEOUT=0 autospec prep "feature"

# Or unset the environment variable
unset AUTOSPEC_TIMEOUT
autospec prep "feature"
```

## Best Practices

### 1. Start Conservative

Begin with a reasonable timeout (e.g., 10 minutes) and adjust based on actual command duration:

```bash
# Track how long commands actually take
time autospec prep "feature"

# Set timeout to 2x the typical duration
# If commands typically take 5 minutes, set timeout to 10 minutes
export AUTOSPEC_TIMEOUT=600
```

### 2. Use Different Timeouts for Different Environments

**Development** (generous or no timeout):
```bash
# ~/.bashrc or ~/.zshrc
export AUTOSPEC_TIMEOUT=0  # No timeout for interactive work
```

**CI/CD** (strict timeout):
```bash
# .github/workflows/ci.yml or similar
env:
  AUTOSPEC_TIMEOUT: 600  # 10 minutes max
```

### 3. Document Project-Specific Timeouts

In your project README:

```markdown
## Autospec Configuration

This project's workflows typically complete in:
- Specify: 2-3 minutes
- Plan: 5-8 minutes
- Tasks: 2-3 minutes
- Implement: 15-20 minutes

**Recommended timeout**: 1800 seconds (30 minutes)

```bash
echo 'timeout: 1800' > .autospec/config.yml
```
```

### 4. Monitor and Adjust

Keep track of timeout occurrences:

```bash
# In CI/CD, log timeout information
autospec prep "feature" 2>&1 | tee autospec.log
if [ $? -eq 5 ]; then
    echo "TIMEOUT OCCURRED" >> timeout-incidents.log
    date >> timeout-incidents.log
fi
```

## Exit Codes

The autospec CLI uses standardized exit codes:

| Code | Meaning | Action |
|------|---------|--------|
| 0 | Success | Continue workflow |
| 1 | Validation failed | Retry possible |
| 2 | Retry limit exhausted | Manual intervention needed |
| 3 | Invalid arguments | Fix command syntax |
| 4 | Missing dependencies | Install required tools |
| **5** | **Command timeout** | **Increase timeout or investigate** |

### Handling Timeout Exit Code in Scripts

```bash
autospec prep "feature"
case $? in
    0)
        echo "Success"
        ;;
    5)
        echo "Command timed out - increase timeout or break into smaller tasks"
        ;;
    *)
        echo "Other error occurred"
        ;;
esac
```

## Troubleshooting

### Issue: Commands timeout unexpectedly

**Symptoms**: Exit code 5, error message about timeout

**Solutions**:
1. **Increase timeout**:
   ```bash
   export AUTOSPEC_TIMEOUT=1800  # Try 30 minutes
   ```

2. **Check what's taking so long**:
   - Large feature descriptions may take longer to process
   - Complex planning phases need more time
   - Implementation of large features can be time-consuming

3. **Break down the work**:
   - Split large features into smaller, focused tasks
   - Implement incrementally

### Issue: Timeout not being respected

**Symptoms**: Commands run longer than configured timeout

**Diagnostics**:
```bash
# 1. Verify timeout is set
autospec config show | grep timeout

# 2. Check environment variable
echo $AUTOSPEC_TIMEOUT

# 3. Verify binary version
autospec version
```

**Solutions**:
- Ensure `AUTOSPEC_TIMEOUT` environment variable is set correctly (all caps)
- Check that config file YAML is valid
- Reload shell session or restart terminal

### Issue: Want no timeout for development

**Solution**:
```bash
# Method 1: Set to 0
export AUTOSPEC_TIMEOUT=0

# Method 2: Unset the variable
unset AUTOSPEC_TIMEOUT

# Method 3: Update config
echo 'timeout: 0' > .autospec/config.yml
```

## Advanced Configuration

### Per-Command Timeouts

While the CLI doesn't support per-command timeouts directly, you can work around this:

```bash
#!/bin/bash
run_with_timeout() {
    local timeout=$1
    shift
    AUTOSPEC_TIMEOUT=$timeout "$@"
}

# Different timeouts for different commands
run_with_timeout 300 autospec specify "feature"    # 5 min
run_with_timeout 600 autospec plan                 # 10 min
run_with_timeout 1800 autospec implement           # 30 min
```

### Conditional Timeouts

```bash
#!/bin/bash
if [ "$CI" = "true" ]; then
    # Strict timeout in CI
    export AUTOSPEC_TIMEOUT=600
else
    # No timeout in development
    export AUTOSPEC_TIMEOUT=0
fi
```

### Dynamic Timeout Based on Feature Size

```bash
#!/bin/bash
feature_description="$1"
word_count=$(echo "$feature_description" | wc -w)

# Longer descriptions get more time
if [ $word_count -gt 50 ]; then
    timeout=1800  # 30 minutes
elif [ $word_count -gt 20 ]; then
    timeout=900   # 15 minutes
else
    timeout=300   # 5 minutes
fi

AUTOSPEC_TIMEOUT=$timeout autospec prep "$feature_description"
```

## See Also

- [Troubleshooting Guide](./troubleshooting.md)
- [CLAUDE.md](../CLAUDE.md) - Development documentation
- [Configuration Reference](../CLAUDE.md#configuration)
