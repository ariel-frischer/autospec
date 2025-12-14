# Shell Completion for Auto Claude SpecKit

Auto Claude SpecKit provides built-in shell completion support for `bash`, `zsh`, `fish`, and `powershell` using Cobra's completion system.

## Features

- **Automatic command completion**: Tab-complete all commands (`full`, `workflow`, `specify`, `plan`, `tasks`, `implement`, etc.)
- **Flag completion**: Complete command flags (e.g., `--max-retries`, `--debug`, `--specs-dir`)
- **Stays in sync**: Automatically updates as commands change
- **Multiple shells**: Works with bash, zsh, fish, and powershell

## Zsh Setup (Recommended)

### 1. Generate Completion File

```bash
# Create completions directory
mkdir -p ~/.zsh_completions

# Generate completion file
autospec completion zsh > ~/.zsh_completions/_autospec
```

### 2. Configure Your Shell

Add these lines to your `~/.zshrc` (before any existing `compinit` call):

```zsh
# Add custom completions directory to fpath
fpath=(~/.zsh_completions $fpath)

# Initialize completion system (if not already present)
autoload -U compinit
compinit
```

**Important**: The `fpath` line must appear **before** `compinit`. If you already have `compinit` in your `.zshrc`, add the `fpath` line above it.

### 3. Reload Your Shell

```bash
exec zsh
```

### 4. Test It

```bash
autospec <tab>
```

You should see all available subcommands with descriptions. If you have `fzf` installed, you'll get fuzzy filtering!

## Bash Setup

### 1. Generate Completion File

```bash
# System-wide (requires root)
autospec completion bash | sudo tee /etc/bash_completion.d/autospec > /dev/null

# User-specific
mkdir -p ~/.bash_completions
autospec completion bash > ~/.bash_completions/autospec
```

### 2. Configure Your Shell

Add to your `~/.bashrc`:

```bash
# Load autospec completion
if [ -f ~/.bash_completions/autospec ]; then
    source ~/.bash_completions/autospec
fi
```

### 3. Reload Your Shell

```bash
source ~/.bashrc
```

## Fish Setup

### 1. Generate Completion File

```bash
autospec completion fish > ~/.config/fish/completions/autospec.fish
```

### 2. Reload Completions

```bash
source ~/.config/fish/config.fish
# or just start a new shell
```

## PowerShell Setup

### 1. Generate Completion Script

```powershell
autospec completion powershell | Out-String | Invoke-Expression
```

### 2. Add to Profile (Persistent)

```powershell
# Add to your PowerShell profile
autospec completion powershell >> $PROFILE
```

## Verification

After setup, test completion works:

```bash
# Type and press TAB
autospec <TAB>

# Should show:
# full       -- Run complete workflow: specify → plan → tasks → implement
# workflow   -- Run workflow: specify → plan → tasks (no implementation)
# specify    -- Generate feature specification
# plan       -- Generate implementation plan
# tasks      -- Generate task list
# implement  -- Execute implementation
# doctor     -- Check dependencies
# status     -- Show current spec status
# config     -- Manage configuration
# init       -- Initialize configuration
# version    -- Show version information
# completion -- Generate completion script
# help       -- Help about any command
```

## Advanced Features

### Flag Completion

Complete flags for any command:

```bash
autospec workflow --<TAB>

# Shows:
# --max-retries    -- Maximum number of retry attempts
# --dry-run        -- Show what would be executed without running
# --skip-preflight -- Skip pre-flight dependency checks
# --debug          -- Enable debug logging
# --specs-dir      -- Specs directory path
# --config         -- Config file path
```

### Subcommand Completion

```bash
autospec config <TAB>

# Shows:
# show   -- Display current configuration
```

### Context-Aware Completion

Some shells provide intelligent completion based on context. For example, in zsh with fuzzy completion:

```bash
autospec imp<TAB>
# Fuzzy matches to "implement"
```

## Troubleshooting

### Completion Not Working

**Check if completion file exists:**

```bash
# Zsh
ls -la ~/.zsh_completions/_autospec

# Bash
ls -la ~/.bash_completions/autospec

# Fish
ls -la ~/.config/fish/completions/autospec.fish
```

**Verify fpath (zsh only):**

```bash
echo $fpath | grep zsh_completions
```

Should show `~/.zsh_completions` in the path.

**Rebuild completion cache (zsh only):**

```bash
rm -f ~/.zcompdump*
exec zsh
```

**Check compinit is called (zsh only):**

```bash
grep -n compinit ~/.zshrc
```

Make sure `fpath` modification appears before `compinit`.

### Completion Shows Old Commands

Regenerate the completion file:

```bash
# Zsh
autospec completion zsh > ~/.zsh_completions/_autospec

# Reload
exec zsh
```

### Permission Issues

Make sure completion files are readable:

```bash
chmod 644 ~/.zsh_completions/_autospec
```

## Updating Completions

When you update `autospec`, regenerate completion files:

```bash
# Zsh
autospec completion zsh > ~/.zsh_completions/_autospec
exec zsh

# Bash
autospec completion bash > ~/.bash_completions/autospec
source ~/.bashrc

# Fish
autospec completion fish > ~/.config/fish/completions/autospec.fish
```

## How It Works

Auto Claude SpecKit uses [Cobra's built-in completion system](https://github.com/spf13/cobra/blob/main/shell_completions.md), which automatically generates completion scripts from the command structure.

When you run `autospec completion <shell>`, Cobra:

1. Analyzes all registered commands and flags
2. Generates shell-specific completion code
3. Outputs a completion script for your shell

This means:
- Completions are always accurate and up-to-date
- No manual maintenance required
- New commands automatically get completion support
- Supports multiple shells with zero extra work

## Platform-Specific Notes

### macOS

On macOS, bash is often outdated (v3.x). For best results:

1. Install modern bash via Homebrew: `brew install bash`
2. Or use zsh (default shell since macOS Catalina)

### Windows

PowerShell completion requires PowerShell 5.0 or later.

For Windows Terminal users, add to your profile:
```powershell
if (Get-Command autospec -ErrorAction SilentlyContinue) {
    autospec completion powershell | Out-String | Invoke-Expression
}
```

### Linux

Most Linux distributions use bash or zsh by default. Follow the respective setup instructions above.

## See Also

- [Cobra Shell Completions Documentation](https://github.com/spf13/cobra/blob/main/shell_completions.md)
- [Zsh Completion System](https://zsh.sourceforge.io/Doc/Release/Completion-System.html)
- CLI Usage: `autospec --help`
