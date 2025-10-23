package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Configuration represents the autospec CLI tool configuration
type Configuration struct {
	ClaudeCmd       string   `koanf:"claude_cmd" validate:"required"`
	ClaudeArgs      []string `koanf:"claude_args"`
	UseAPIKey       bool     `koanf:"use_api_key"`
	CustomClaudeCmd string   `koanf:"custom_claude_cmd"`
	SpecifyCmd      string   `koanf:"specify_cmd" validate:"required"`
	MaxRetries      int      `koanf:"max_retries" validate:"min=1,max=10"`
	SpecsDir        string   `koanf:"specs_dir" validate:"required"`
	StateDir        string   `koanf:"state_dir" validate:"required"`
	SkipPreflight   bool     `koanf:"skip_preflight"`
	Timeout         int      `koanf:"timeout" validate:"min=1,max=3600"`
}

// Load loads configuration from global, local, and environment sources
// Priority: Environment variables > Local config > Global config > Defaults
func Load(localConfigPath string) (*Configuration, error) {
	k := koanf.New(".")

	// Apply defaults first
	defaults := GetDefaults()
	for key, value := range defaults {
		k.Set(key, value)
	}

	// Load global config if it exists
	homeDir, err := os.UserHomeDir()
	if err == nil {
		globalPath := filepath.Join(homeDir, ".autospec", "config.json")
		if _, err := os.Stat(globalPath); err == nil {
			if err := k.Load(file.Provider(globalPath), json.Parser()); err != nil {
				return nil, fmt.Errorf("failed to load global config: %w", err)
			}
		}
	}

	// Load local config if it exists
	if localConfigPath != "" {
		if _, err := os.Stat(localConfigPath); err == nil {
			if err := k.Load(file.Provider(localConfigPath), json.Parser()); err != nil {
				return nil, fmt.Errorf("failed to load local config: %w", err)
			}
		}
	}

	// Override with environment variables (highest priority)
	k.Load(env.Provider("AUTOSPEC_", ".", envTransform), nil)

	// Unmarshal into struct
	var cfg Configuration
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Validate custom_claude_cmd if specified
	if cfg.CustomClaudeCmd != "" && !strings.Contains(cfg.CustomClaudeCmd, "{{PROMPT}}") {
		return nil, fmt.Errorf("custom_claude_cmd must contain {{PROMPT}} placeholder")
	}

	// Expand home directory in paths
	cfg.StateDir = expandHomePath(cfg.StateDir)
	cfg.SpecsDir = expandHomePath(cfg.SpecsDir)

	return &cfg, nil
}

// envTransform converts environment variable names to config keys
// Example: AUTOSPEC_MAX_RETRIES -> max_retries
func envTransform(s string) string {
	return strings.Replace(strings.ToLower(strings.TrimPrefix(s, "AUTOSPEC_")), "_", "_", -1)
}

// expandHomePath expands ~ to the user's home directory
func expandHomePath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(homeDir, path[2:])
		}
	}
	return path
}
