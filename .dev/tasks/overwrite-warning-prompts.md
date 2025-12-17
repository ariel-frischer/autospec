# Artifact Overwrite Warning Prompts

Add confirmation prompts before overwriting existing artifacts (plan.yaml, tasks.yaml).

## Problem

Current behavior silently overwrites artifacts without any warning:
- `autospec plan` overwrites existing plan.yaml without confirmation
- `autospec tasks` overwrites existing tasks.yaml without confirmation
- `autospec run -p` / `autospec run -t` same issue
- Users may lose work if they accidentally run a command on an existing spec

Note: `autospec specify` creates NEW spec directories (001, 002, etc.) so doesn't have this issue.

## Proposed Solution

### Prompt Behavior

When an artifact already exists, show a confirmation prompt:

```
plan.yaml already exists (last modified: 2 hours ago)
? Overwrite existing plan.yaml? [y/N]
```

### CLI Flags

| Flag | Behavior |
|------|----------|
| `--yes` / `-y` | Skip prompts, always overwrite (for CI/CD) |
| `--no-overwrite` | Fail if artifact exists (safe mode) |
| (default) | Prompt user for confirmation |

### Detection Logic

```go
// Before executing stage
if artifactExists(artifactPath) {
    if cfg.NoOverwrite {
        return fmt.Errorf("%s already exists (use --yes to overwrite)", artifactPath)
    }
    if !cfg.Yes && isInteractive() {
        if !promptOverwrite(artifactPath) {
            return fmt.Errorf("aborted: %s not overwritten", artifactPath)
        }
    }
}
```

### Interactive Detection

Check if running in interactive terminal:
```go
func isInteractive() bool {
    return term.IsTerminal(int(os.Stdin.Fd()))
}
```

Non-interactive (piped, CI) defaults to NOT overwriting unless `--yes` is passed.

## Tasks

- [ ] Add `--yes`/`-y` flag to plan, tasks, run commands
- [ ] Add `--no-overwrite` flag to plan, tasks, run commands
- [ ] Create prompt helper in `internal/cli/prompt.go`
  - `PromptOverwrite(artifactPath string) bool`
  - Use existing `PromptUserToContinue` pattern from preflight.go
- [ ] Update `internal/cli/plan.go` to check for existing plan.yaml
- [ ] Update `internal/cli/tasks.go` to check for existing tasks.yaml
- [ ] Update `internal/cli/run.go` to pass flags through
- [ ] Handle non-interactive mode (default to abort without --yes)
- [ ] Update docs/reference.md with new flags
- [ ] Add tests for overwrite prompt logic
