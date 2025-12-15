package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// ConfigSource tracks where a configuration value came from
type ConfigSource string

const (
	SourceDefault ConfigSource = "default"
	SourceUser    ConfigSource = "user"
	SourceProject ConfigSource = "project"
	SourceEnv     ConfigSource = "env"
)

// Configuration represents the autospec CLI tool configuration
type Configuration struct {
	ClaudeCmd         string   `koanf:"claude_cmd"`
	ClaudeArgs        []string `koanf:"claude_args"`
	UseAPIKey         bool     `koanf:"use_api_key"`
	CustomClaudeCmd   string   `koanf:"custom_claude_cmd"`
	MaxRetries        int      `koanf:"max_retries"`
	SpecsDir          string   `koanf:"specs_dir"`
	StateDir          string   `koanf:"state_dir"`
	SkipPreflight     bool     `koanf:"skip_preflight"`
	Timeout           int      `koanf:"timeout"`
	ShowProgress      bool     `koanf:"show_progress"`      // Show progress indicators (spinners) during execution
	SkipConfirmations bool     `koanf:"skip_confirmations"` // Skip confirmation prompts (can also be set via AUTOSPEC_YES env var)
}

// LoadOptions configures how configuration is loaded
type LoadOptions struct {
	// ProjectConfigPath overrides the project config path (default: .autospec/config.yml)
	ProjectConfigPath string
	// WarningWriter receives deprecation warnings (default: os.Stderr)
	WarningWriter io.Writer
	// SkipWarnings suppresses deprecation warnings
	SkipWarnings bool
}

// Load loads configuration from user, project, and environment sources.
// Priority: Environment variables > Project config > User config > Defaults
//
// New YAML config paths:
//   - User config: ~/.config/autospec/config.yml (XDG compliant)
//   - Project config: .autospec/config.yml
//
// Legacy JSON config paths (deprecated, triggers migration warning):
//   - User config: ~/.autospec/config.json
//   - Project config: .autospec/config.json
func Load(projectConfigPath string) (*Configuration, error) {
	return LoadWithOptions(LoadOptions{ProjectConfigPath: projectConfigPath})
}

// LoadWithOptions loads configuration with custom options
func LoadWithOptions(opts LoadOptions) (*Configuration, error) {
	k := koanf.New(".")
	warningWriter := opts.WarningWriter
	if warningWriter == nil {
		warningWriter = os.Stderr
	}

	// Apply defaults first
	defaults := GetDefaults()
	for key, value := range defaults {
		k.Set(key, value)
	}

	// Load user-level config (new YAML location first, then legacy JSON)
	userYAMLPath, _ := UserConfigPath()
	legacyUserPath, _ := LegacyUserConfigPath()

	userYAMLExists := fileExists(userYAMLPath)
	legacyUserExists := fileExists(legacyUserPath)

	if userYAMLExists {
		// Validate YAML syntax first
		if err := ValidateYAMLSyntax(userYAMLPath); err != nil {
			return nil, err
		}
		if err := k.Load(file.Provider(userYAMLPath), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("failed to load user config %s: %w", userYAMLPath, err)
		}
		// Warn if legacy JSON also exists
		if legacyUserExists && !opts.SkipWarnings {
			fmt.Fprintf(warningWriter, "Warning: Legacy JSON config found at %s (ignored, using %s)\n", legacyUserPath, userYAMLPath)
			fmt.Fprintf(warningWriter, "  Run 'autospec config migrate --user' to remove the legacy file.\n\n")
		}
	} else if legacyUserExists {
		// Load legacy JSON and warn about migration
		if err := k.Load(file.Provider(legacyUserPath), json.Parser()); err != nil {
			return nil, fmt.Errorf("failed to load legacy user config %s: %w", legacyUserPath, err)
		}
		if !opts.SkipWarnings {
			fmt.Fprintf(warningWriter, "Warning: Using deprecated JSON config at %s\n", legacyUserPath)
			fmt.Fprintf(warningWriter, "  Run 'autospec config migrate --user' to migrate to YAML format.\n\n")
		}
	}

	// Load project-level config (new YAML location first, then legacy JSON)
	projectYAMLPath := ProjectConfigPath()
	if opts.ProjectConfigPath != "" {
		projectYAMLPath = opts.ProjectConfigPath
	}
	legacyProjectPath := LegacyProjectConfigPath()

	projectYAMLExists := fileExists(projectYAMLPath)
	legacyProjectExists := fileExists(legacyProjectPath)

	if projectYAMLExists {
		// Validate YAML syntax first
		if err := ValidateYAMLSyntax(projectYAMLPath); err != nil {
			return nil, err
		}
		if err := k.Load(file.Provider(projectYAMLPath), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("failed to load project config %s: %w", projectYAMLPath, err)
		}
		// Warn if legacy JSON also exists
		if legacyProjectExists && !opts.SkipWarnings {
			fmt.Fprintf(warningWriter, "Warning: Legacy JSON config found at %s (ignored, using %s)\n", legacyProjectPath, projectYAMLPath)
			fmt.Fprintf(warningWriter, "  Run 'autospec config migrate --project' to remove the legacy file.\n\n")
		}
	} else if legacyProjectExists {
		// Load legacy JSON and warn about migration
		if err := k.Load(file.Provider(legacyProjectPath), json.Parser()); err != nil {
			return nil, fmt.Errorf("failed to load legacy project config %s: %w", legacyProjectPath, err)
		}
		if !opts.SkipWarnings {
			fmt.Fprintf(warningWriter, "Warning: Using deprecated JSON config at %s\n", legacyProjectPath)
			fmt.Fprintf(warningWriter, "  Run 'autospec config migrate --project' to migrate to YAML format.\n\n")
		}
	}

	// Override with environment variables (highest priority)
	if err := k.Load(env.Provider("AUTOSPEC_", ".", envTransform), nil); err != nil {
		return nil, fmt.Errorf("failed to load environment config: %w", err)
	}

	// Unmarshal into struct
	var cfg Configuration
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration values
	if err := ValidateConfigValues(&cfg, "config"); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// Expand home directory in paths
	cfg.StateDir = expandHomePath(cfg.StateDir)
	cfg.SpecsDir = expandHomePath(cfg.SpecsDir)

	// Handle AUTOSPEC_YES as an alias for skip_confirmations
	if os.Getenv("AUTOSPEC_YES") != "" {
		cfg.SkipConfirmations = true
	}

	return &cfg, nil
}

// fileExists returns true if the file exists and is readable
func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
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
