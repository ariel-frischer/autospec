// autospec - Spec-Driven Development Automation
// Author: Ariel Frischer
// Source: https://github.com/ariel-frischer/autospec

// Package config provides hierarchical configuration management for autospec using koanf.
// Configuration is loaded with priority: environment variables > project config (.autospec/config.yml)
// > user config (~/.config/autospec/config.yml) > defaults. It supports both YAML and legacy JSON
// formats, with migration utilities for transitioning from JSON to YAML.
package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ariel-frischer/autospec/internal/cliagent"
	"github.com/ariel-frischer/autospec/internal/notify"
	"github.com/ariel-frischer/autospec/internal/worktree"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	yamlv3 "gopkg.in/yaml.v3"
)

// ConfigSource tracks where a configuration value came from
type ConfigSource string

const (
	SourceDefault ConfigSource = "default"
	SourceUser    ConfigSource = "user"
	SourceProject ConfigSource = "project"
	SourceEnv     ConfigSource = "env"
	SourceFlag    ConfigSource = "flag"
)

// Configuration represents the autospec CLI tool configuration
type Configuration struct {
	// AgentPreset selects a built-in agent by name (e.g., "claude", "gemini", "cline").
	// Can be set via AUTOSPEC_AGENT_PRESET env var.
	AgentPreset string `koanf:"agent_preset"`

	// CustomAgent provides structured configuration for custom agents.
	// Takes precedence over agent_preset.
	// Example:
	//   custom_agent:
	//     command: "claude"
	//     args: ["-p", "--verbose", "{{PROMPT}}"]
	//     env:
	//       ANTHROPIC_API_KEY: ""
	//     post_processor: "cclean"
	CustomAgent *cliagent.CustomAgentConfig `koanf:"custom_agent"`

	// UseSubscription forces Claude to use subscription (Pro/Max) instead of API credits.
	// When true, ANTHROPIC_API_KEY is set to empty string at execution time,
	// and validation is skipped for this environment variable.
	// Default: true (protects users from accidental API charges).
	// Can be set via AUTOSPEC_USE_SUBSCRIPTION env var.
	UseSubscription bool `koanf:"use_subscription"`

	MaxRetries        int    `koanf:"max_retries"`
	SpecsDir          string `koanf:"specs_dir"`
	StateDir          string `koanf:"state_dir"`
	SkipPreflight     bool   `koanf:"skip_preflight"`
	Timeout           int    `koanf:"timeout"`
	SkipConfirmations bool   `koanf:"skip_confirmations"` // Skip confirmation prompts (can also be set via AUTOSPEC_YES env var)
	// ImplementMethod sets the default execution mode for the implement command.
	// Valid values: "single-session" (legacy), "phases" (default), "tasks"
	// Can be overridden by CLI flags (--phases, --tasks) or env var AUTOSPEC_IMPLEMENT_METHOD
	ImplementMethod string `koanf:"implement_method"`

	// Notifications configures notification preferences for command and stage completion.
	// Supports sound, visual, or both notification types across macOS, Linux, and Windows.
	// Environment variable support via AUTOSPEC_NOTIFICATIONS_* prefix.
	Notifications notify.NotificationConfig `koanf:"notifications"`

	// MaxHistoryEntries sets the maximum number of command history entries to retain.
	// Oldest entries are pruned when this limit is exceeded.
	// Default: 500. Can be set via AUTOSPEC_MAX_HISTORY_ENTRIES env var.
	MaxHistoryEntries int `koanf:"max_history_entries"`

	// ViewLimit sets the number of recent specs displayed by the view command.
	// Default: 5. Can be set via AUTOSPEC_VIEW_LIMIT env var.
	ViewLimit int `koanf:"view_limit"`

	// Worktree configures worktree management settings.
	// Used by the 'autospec worktree' command for creating and managing git worktrees.
	Worktree *worktree.WorktreeConfig `koanf:"worktree"`

	// DefaultAgents stores the list of agent names to pre-select in future init prompts.
	// Set during 'autospec init' when user selects agents for configuration.
	// Can be set via AUTOSPEC_DEFAULT_AGENTS env var (comma-separated).
	DefaultAgents []string `koanf:"default_agents,omitempty"`

	// OutputStyle controls how stream-json output is formatted for display.
	// Valid values: default, compact, minimal, plain, raw
	// Default: "default" (box-drawing characters with colors)
	// Can be set via AUTOSPEC_OUTPUT_STYLE env var or --output-style CLI flag.
	OutputStyle string `koanf:"output_style"`

	// SkipPermissionsNoticeShown tracks whether the user has seen the security notice
	// about --dangerously-skip-permissions. Set to true after first workflow run.
	// This is a user-level config field only (not shown in project config).
	// Can be set via AUTOSPEC_SKIP_PERMISSIONS_NOTICE_SHOWN env var.
	SkipPermissionsNoticeShown bool `koanf:"skip_permissions_notice_shown"`

	// AutoCommit enables automatic git commit creation after workflow completion.
	// When true, instructions are injected into the agent prompt to:
	// - Update .gitignore with common ignorable patterns
	// - Stage appropriate files for version control
	// - Create a commit with conventional commit message format
	// Default: false. Can be set via AUTOSPEC_AUTO_COMMIT env var.
	AutoCommit bool `koanf:"auto_commit"`

	// AutoCommitSource tracks where the AutoCommit value came from.
	// Used to determine if the user explicitly configured auto-commit.
	// Set during config loading, not persisted.
	AutoCommitSource ConfigSource `koanf:"-"`
}

// LoadOptions configures how configuration is loaded
type LoadOptions struct {
	// ProjectConfigPath overrides the project config path (default: .autospec/config.yml)
	ProjectConfigPath string
	// UserConfigPath overrides the user config path (default: ~/.config/autospec/config.yml)
	// Useful for testing to provide a mock user config
	UserConfigPath string
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

	if err := loadUserConfig(k, opts.UserConfigPath, warningWriter, opts.SkipWarnings); err != nil {
		return nil, err
	}

	if err := loadProjectConfig(k, opts.ProjectConfigPath, warningWriter, opts.SkipWarnings); err != nil {
		return nil, err
	}

	if err := loadEnvironmentConfig(k); err != nil {
		return nil, err
	}

	cfg, err := finalizeConfig(k)
	if err != nil {
		return nil, err
	}

	// Track AutoCommit source for migration notice
	cfg.AutoCommitSource = detectAutoCommitSource(opts)

	return cfg, nil
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

// loadUserConfig loads user-level config (YAML preferred, legacy JSON supported).
// If customPath is provided, it uses that path exclusively (for testing).
// Otherwise: Priority: YAML (~/.config/autospec/config.yml) > JSON (~/.autospec/config.json).
// Warns if both exist (YAML used, JSON ignored) or if only legacy JSON exists.
func loadUserConfig(k *koanf.Koanf, customPath string, warningWriter io.Writer, skipWarnings bool) error {
	// If custom path provided, use it exclusively (for testing)
	if customPath != "" {
		if fileExists(customPath) {
			if err := loadYAMLConfig(k, customPath, "user"); err != nil {
				return fmt.Errorf("loading user YAML config: %w", err)
			}
		}
		return nil
	}

	userYAMLPath, _ := UserConfigPath()
	legacyUserPath, _ := LegacyUserConfigPath()

	userYAMLExists := fileExists(userYAMLPath)
	legacyUserExists := fileExists(legacyUserPath)

	if userYAMLExists {
		if err := loadYAMLConfig(k, userYAMLPath, "user"); err != nil {
			return fmt.Errorf("loading user YAML config: %w", err)
		}
		warnLegacyExists(warningWriter, legacyUserPath, userYAMLPath, legacyUserExists, skipWarnings, "--user")
	} else if legacyUserExists {
		if err := loadLegacyJSONConfig(k, legacyUserPath, "user", warningWriter, skipWarnings, "--user"); err != nil {
			return fmt.Errorf("loading legacy user JSON config: %w", err)
		}
	}
	return nil
}

// loadProjectConfig loads project-level config (YAML preferred, legacy JSON supported).
// Supports custom path override (for testing). Falls back to legacy JSON with warning.
// Same priority/warning logic as loadUserConfig.
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
			return fmt.Errorf("loading project YAML config: %w", err)
		}
		warnLegacyExists(warningWriter, legacyProjectPath, projectYAMLPath, legacyProjectExists, skipWarnings, "--project")
	} else if legacyProjectExists {
		if err := loadLegacyJSONConfig(k, legacyProjectPath, "project", warningWriter, skipWarnings, "--project"); err != nil {
			return fmt.Errorf("loading legacy project JSON config: %w", err)
		}
	}
	return nil
}

// loadYAMLConfig validates and loads a YAML config file
func loadYAMLConfig(k *koanf.Koanf, path, configType string) error {
	if err := ValidateYAMLSyntax(path); err != nil {
		return fmt.Errorf("validating YAML syntax for %s config: %w", configType, err)
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

// detectAutoCommitSource determines where the auto_commit setting came from.
// Checks in priority order: env > project > user > default.
func detectAutoCommitSource(opts LoadOptions) ConfigSource {
	// Check environment variable first (highest priority)
	if os.Getenv("AUTOSPEC_AUTO_COMMIT") != "" {
		return SourceEnv
	}

	// Check project config
	projectPath := opts.ProjectConfigPath
	if projectPath == "" {
		projectPath = ProjectConfigPath()
	}
	if configContainsKey(projectPath, "auto_commit") {
		return SourceProject
	}

	// Check user config
	userPath := opts.UserConfigPath
	if userPath == "" {
		userPath, _ = UserConfigPath()
	}
	if configContainsKey(userPath, "auto_commit") {
		return SourceUser
	}

	return SourceDefault
}

// configContainsKey checks if a YAML config file contains a specific key.
// Returns false if file doesn't exist or key is not present.
func configContainsKey(path, key string) bool {
	if !fileExists(path) {
		return false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	// Simple check: look for the key in the file content
	// This is a basic implementation that works for top-level keys
	var content map[string]interface{}
	if err := yamlv3.Unmarshal(data, &content); err != nil {
		return false
	}

	_, exists := content[key]
	return exists
}

// finalizeConfig unmarshals, validates, and applies final transformations
func finalizeConfig(k *koanf.Koanf) (*Configuration, error) {
	return finalizeConfigWithWarnings(k, os.Stderr, false)
}

// finalizeConfigWithWarnings unmarshals and optionally warns about deprecations
func finalizeConfigWithWarnings(k *koanf.Koanf, warningWriter io.Writer, skipWarnings bool) (*Configuration, error) {
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

// GetAgent returns a CLI agent based on configuration priority.
// Priority: custom_agent > agent_preset > default (claude).
// Returns error if the selected agent is invalid or not found in registry.
func (c *Configuration) GetAgent() (cliagent.Agent, error) {
	// Highest priority: structured custom_agent config
	if c.CustomAgent.IsValid() {
		return cliagent.NewCustomAgentFromConfig(*c.CustomAgent)
	}

	// Second priority: agent_preset (built-in agent by name)
	if c.AgentPreset != "" {
		agent := cliagent.Get(c.AgentPreset)
		if agent == nil {
			return nil, fmt.Errorf("unknown agent preset %q; available: %v", c.AgentPreset, cliagent.List())
		}
		return agent, nil
	}

	// Default: use claude agent from registry
	agent := cliagent.Get("claude")
	if agent == nil {
		return nil, fmt.Errorf("default agent 'claude' not registered")
	}
	return agent, nil
}
