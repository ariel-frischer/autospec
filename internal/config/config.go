// autospec - Spec-Driven Development Automation
// Author: Ariel Frischer
// Source: https://github.com/ariel-frischer/autospec

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
	// ImplementMethod sets the default execution mode for the implement command.
	// Valid values: "single-session" (legacy), "phases" (default), "tasks"
	// Can be overridden by CLI flags (--phases, --tasks) or env var AUTOSPEC_IMPLEMENT_METHOD
	ImplementMethod string `koanf:"implement_method"`
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
	warningWriter := getWarningWriter(opts.WarningWriter)

	loadDefaults(k)

	if err := loadUserConfig(k, warningWriter, opts.SkipWarnings); err != nil {
		return nil, err
	}

	if err := loadProjectConfig(k, opts.ProjectConfigPath, warningWriter, opts.SkipWarnings); err != nil {
		return nil, err
	}

	if err := loadEnvironmentConfig(k); err != nil {
		return nil, err
	}

	return finalizeConfig(k)
}

// getWarningWriter returns the warning writer or defaults to stderr
func getWarningWriter(w io.Writer) io.Writer {
	if w == nil {
		return os.Stderr
	}
	return w
}

// loadDefaults applies default configuration values
func loadDefaults(k *koanf.Koanf) {
	defaults := GetDefaults()
	for key, value := range defaults {
		k.Set(key, value)
	}
}

// loadUserConfig loads user-level config (YAML preferred, legacy JSON supported)
func loadUserConfig(k *koanf.Koanf, warningWriter io.Writer, skipWarnings bool) error {
	userYAMLPath, _ := UserConfigPath()
	legacyUserPath, _ := LegacyUserConfigPath()

	userYAMLExists := fileExists(userYAMLPath)
	legacyUserExists := fileExists(legacyUserPath)

	if userYAMLExists {
		if err := loadYAMLConfig(k, userYAMLPath, "user"); err != nil {
			return err
		}
		warnLegacyExists(warningWriter, legacyUserPath, userYAMLPath, legacyUserExists, skipWarnings, "--user")
	} else if legacyUserExists {
		if err := loadLegacyJSONConfig(k, legacyUserPath, "user", warningWriter, skipWarnings, "--user"); err != nil {
			return err
		}
	}
	return nil
}

// loadProjectConfig loads project-level config (YAML preferred, legacy JSON supported)
func loadProjectConfig(k *koanf.Koanf, customPath string, warningWriter io.Writer, skipWarnings bool) error {
	projectYAMLPath := ProjectConfigPath()
	if customPath != "" {
		projectYAMLPath = customPath
	}
	legacyProjectPath := LegacyProjectConfigPath()

	projectYAMLExists := fileExists(projectYAMLPath)
	legacyProjectExists := fileExists(legacyProjectPath)

	if projectYAMLExists {
		if err := loadYAMLConfig(k, projectYAMLPath, "project"); err != nil {
			return err
		}
		warnLegacyExists(warningWriter, legacyProjectPath, projectYAMLPath, legacyProjectExists, skipWarnings, "--project")
	} else if legacyProjectExists {
		if err := loadLegacyJSONConfig(k, legacyProjectPath, "project", warningWriter, skipWarnings, "--project"); err != nil {
			return err
		}
	}
	return nil
}

// loadYAMLConfig validates and loads a YAML config file
func loadYAMLConfig(k *koanf.Koanf, path, configType string) error {
	if err := ValidateYAMLSyntax(path); err != nil {
		return err
	}
	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		return fmt.Errorf("failed to load %s config %s: %w", configType, path, err)
	}
	return nil
}

// loadLegacyJSONConfig loads legacy JSON and warns about migration
func loadLegacyJSONConfig(k *koanf.Koanf, path, configType string, warningWriter io.Writer, skipWarnings bool, migrateFlag string) error {
	if err := k.Load(file.Provider(path), json.Parser()); err != nil {
		return fmt.Errorf("failed to load legacy %s config %s: %w", configType, path, err)
	}
	if !skipWarnings {
		fmt.Fprintf(warningWriter, "Warning: Using deprecated JSON config at %s\n", path)
		fmt.Fprintf(warningWriter, "  Run 'autospec config migrate %s' to migrate to YAML format.\n\n", migrateFlag)
	}
	return nil
}

// warnLegacyExists warns if legacy JSON exists alongside new YAML
func warnLegacyExists(warningWriter io.Writer, legacyPath, yamlPath string, legacyExists, skipWarnings bool, migrateFlag string) {
	if legacyExists && !skipWarnings {
		fmt.Fprintf(warningWriter, "Warning: Legacy JSON config found at %s (ignored, using %s)\n", legacyPath, yamlPath)
		fmt.Fprintf(warningWriter, "  Run 'autospec config migrate %s' to remove the legacy file.\n\n", migrateFlag)
	}
}

// loadEnvironmentConfig loads environment variable overrides
func loadEnvironmentConfig(k *koanf.Koanf) error {
	if err := k.Load(env.Provider("AUTOSPEC_", ".", envTransform), nil); err != nil {
		return fmt.Errorf("failed to load environment config: %w", err)
	}
	return nil
}

// finalizeConfig unmarshals, validates, and applies final transformations
func finalizeConfig(k *koanf.Koanf) (*Configuration, error) {
	var cfg Configuration
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := ValidateConfigValues(&cfg, "config"); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	cfg.StateDir = expandHomePath(cfg.StateDir)
	cfg.SpecsDir = expandHomePath(cfg.SpecsDir)

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
