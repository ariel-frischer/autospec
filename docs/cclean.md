# claude-clean (cclean)

*Last updated: 2025-12-20*

[claude-clean](https://github.com/ariel-frischer/claude-clean) transforms Claude Code's streaming JSON output into readable terminal output.

## Installation

claude-clean is bundled as a dependency of autospec. For standalone use:

```bash
go install github.com/ariel-frischer/claude-clean@latest
```

## CLI Usage

```bash
# Parse a JSONL conversation file
cclean output.jsonl

# Plain text output (best for piping/analysis)
cclean -s plain output.jsonl

# Available styles
cclean -s default output.jsonl   # Box-drawing characters (default)
cclean -s compact output.jsonl   # Single-line summaries
cclean -s minimal output.jsonl   # No box-drawing
cclean -s plain output.jsonl     # No colors

# With line numbers
cclean -n output.jsonl

# Verbose output (includes usage stats, tool IDs)
cclean -V output.jsonl
```

## Go Library Usage

```go
import (
    "github.com/ariel-frischer/claude-clean/parser"
    "github.com/ariel-frischer/claude-clean/display"
)

// Parse a JSONL line
var msg parser.StreamMessage
json.Unmarshal([]byte(line), &msg)

// Strip system reminders from text
clean := parser.StripSystemReminders(msg.Message.Content[0].Text)

// Display with styling
cfg := &display.Config{
    Style:       display.StyleDefault,
    Verbose:     false,
    LineNumbers: true,
}
display.DisplayMessage(&msg, 1, cfg)
```

## Display Styles

| Style | Constant | Description |
|-------|----------|-------------|
| default | `display.StyleDefault` | Box-drawing characters, full formatting |
| compact | `display.StyleCompact` | Single-line summaries |
| minimal | `display.StyleMinimal` | No box-drawing |
| plain | `display.StylePlain` | No colors, suitable for piping |

## Key Types

| Type | Package | Description |
|------|---------|-------------|
| `StreamMessage` | `parser` | Top-level message wrapper |
| `ContentBlock` | `parser` | Text, tool_use, or tool_result |
| `Config` | `display` | Output configuration |

## Use with autospec

Configure as post_processor in `~/.config/autospec/config.yml`:

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
  post_processor: "cclean"
```

This pipes Claude's stream-json output through cclean for readable terminal output during autospec workflows.

## References

- [GitHub Repository](https://github.com/ariel-frischer/claude-clean)
- [Claude Settings & Sandboxing](claude-settings.md)
