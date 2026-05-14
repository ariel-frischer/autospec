package config

import "fmt"

const (
	CodexOutputModeCompact = "compact"
	CodexOutputModeFull    = "full"
)

// CodexOutputConfig controls output handling for automated Codex CLI runs.
type CodexOutputConfig struct {
	// Mode controls whether autospec formats Codex JSONL output or streams
	// Codex's native transcript unchanged. Valid values: compact, full.
	Mode string `koanf:"mode" yaml:"mode"`

	// MaxLinesPerMessage caps each displayed compact output block.
	MaxLinesPerMessage int `koanf:"max_lines_per_message" yaml:"max_lines_per_message"`
}

func (c CodexOutputConfig) CompactEnabled() bool {
	return c.Mode == "" || c.Mode == CodexOutputModeCompact
}

func (c CodexOutputConfig) LineLimit() int {
	if c.MaxLinesPerMessage > 0 {
		return c.MaxLinesPerMessage
	}
	return 40
}

func validateCodexOutputConfig(cfg CodexOutputConfig, filePath string) error {
	if cfg.Mode == "" && cfg.MaxLinesPerMessage == 0 {
		return nil
	}
	if cfg.Mode != "" && cfg.Mode != CodexOutputModeCompact && cfg.Mode != CodexOutputModeFull {
		return &ValidationError{
			FilePath: filePath,
			Field:    "codex_output.mode",
			Message:  "must be one of: compact, full",
		}
	}
	if cfg.MaxLinesPerMessage < 1 {
		return &ValidationError{
			FilePath: filePath,
			Field:    "codex_output.max_lines_per_message",
			Message:  fmt.Sprintf("must be at least 1 (got %d)", cfg.MaxLinesPerMessage),
		}
	}
	return nil
}
