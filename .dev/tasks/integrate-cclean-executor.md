# Integrate cclean Library into Executor

**Created:** 2025-12-20
**Status:** Planning

## Problem

Current approach requires users to configure `post_processor: "cclean"` in their config to get readable output from Claude's `--output-format stream-json`. This is awkward:

1. Requires external binary in PATH
2. Uses shell piping which adds complexity
3. Users must manually configure it
4. Post-processor runs as separate process

## Solution

Integrate cclean's `parser` and `display` packages directly into autospec's executor. When Claude is invoked with `--output-format stream-json` and `-p` flag, automatically parse and display output using cclean library.

## Benefits

1. **Zero config** - Works out of the box, no post_processor needed
2. **Better performance** - No shell piping, direct Go function calls
3. **Simpler default config** - Remove post_processor from recommended setup
4. **Library reuse** - Use cclean as library, not external process

## Design

### Detection

In executor, detect when Claude is invoked with stream-json output format:
- Check if args contain `--output-format stream-json` or `-o stream-json`
- Only apply when using `-p` flag (headless mode)

### Integration Point

`internal/workflow/executor.go` - where Claude command is executed

### Output Handling

When stream-json detected:
1. Capture stdout line-by-line (JSONL format)
2. Parse each line with `parser.StreamMessage`
3. Display with `display.DisplayMessage()` using configured style
4. Strip system reminders with `parser.StripSystemReminders()`

### Configuration

Add optional config for display style:
```yaml
# ~/.config/autospec/config.yml
output_style: default  # default | compact | minimal | plain
```

Default to `default` style (box-drawing, colors).

### Fallback

If parsing fails for a line, print raw line to avoid data loss.

## Implementation Tasks

1. **Add output_style config field**
   - `internal/config/config.go` - add `OutputStyle` field
   - `internal/config/defaults.go` - add default value

2. **Create stream-json display wrapper**
   - `internal/workflow/display.go` - new file
   - Wrap cclean display with autospec config integration
   - Handle line-by-line JSONL parsing

3. **Integrate into executor**
   - `internal/workflow/executor.go`
   - Detect stream-json mode
   - Route stdout through display wrapper instead of direct passthrough

4. **Update default config template**
   - Remove post_processor from recommended setup
   - Simplify to just command + args

5. **Update documentation**
   - `docs/claude-settings.md`
   - `docs/cclean.md`
   - `site/reference/configuration.md`
   - `README.md`

6. **Tests**
   - Unit tests for stream-json detection
   - Unit tests for display wrapper
   - Integration test with mock Claude output

## New Recommended Config

```yaml
custom_agent:
  command: "claude"
  args:
    - "-p"
    - "--dangerously-skip-permissions"
    - "--verbose"
    - "--output-format"
    - "stream-json"
    - "{{PROMPT}}"
```

No `post_processor` needed - cclean display is automatic.

## Open Questions

1. Should we support disabling cclean display? (e.g., `output_style: raw`)
2. Should display style be configurable per-command or only global?
3. How to handle non-stream-json output modes? (pass through unchanged)

## Dependencies

- `github.com/ariel-frischer/claude-clean v0.2.0` (already in go.mod)
