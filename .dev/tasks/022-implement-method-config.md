# 22. Default Implement Method Configuration - DONE

## Summary

Add a configuration option `implement_method` that sets the default execution mode for `autospec implement`. Change the default from "single-session" to "phases" for better cost efficiency and context isolation.

## Problem

Currently, `autospec implement` (no flags) runs all tasks in a single Claude session, similar to GitHub SpecKit's approach. This causes:

1. **Context pollution**: Earlier task discussions contaminate later task reasoning
2. **LLM degradation**: Performance measurably decreases as context accumulates
3. **Higher costs**: Each API turn bills the entire conversation context (up to 80%+ higher costs on large specs)
4. **Harder debugging**: When something fails, the entire session's context is involved

See [docs/faq.md](../../docs/faq.md#why-use---phases-or---tasks-instead-of-running-everything-in-one-session) and [docs/research/claude-opus-4.5-context-performance.md](../../docs/research/claude-opus-4.5-context-performance.md) for detailed analysis.

## Solution

Add `implement_method` config option with three valid values:
- `single-session` - All tasks in one Claude session (legacy behavior)
- `phases` - Each phase in separate session (NEW DEFAULT)
- `tasks` - Each task in separate session (maximum isolation)

## Implementation

### Config Changes

```yaml
# .autospec/config.yml
implement_method: phases  # Options: single-session, phases, tasks
```

### Files Modified

1. `internal/config/config.go` - Add `ImplementMethod` field to Configuration struct
2. `internal/config/defaults.go` - Set default to "phases"
3. `internal/config/validate.go` - Add validation for valid values
4. `internal/cli/implement.go` - Use config default when no execution mode flags provided

### Behavior

- Config sets default, CLI flags override
- `autospec implement` → uses `implement_method` from config (default: phases)
- `autospec implement --phases` → runs phases (explicit)
- `autospec implement --tasks` → runs tasks (explicit override)
- Backward compatible: users can set `implement_method: single-session` to restore old behavior

## Spec Command

```bash
autospec specify "Add implement_method config option. See .dev/tasks/022-implement-method-config.md for full details."
```
