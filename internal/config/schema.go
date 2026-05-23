package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ConfigValueType defines the expected type for a configuration value.
type ConfigValueType int

const (
	TypeBool ConfigValueType = iota
	TypeInt
	TypeFloat
	TypeDuration
	TypeString
	TypeEnum
)

// String returns the string representation of ConfigValueType.
func (t ConfigValueType) String() string {
	switch t {
	case TypeBool:
		return "bool"
	case TypeInt:
		return "int"
	case TypeFloat:
		return "float"
	case TypeDuration:
		return "duration"
	case TypeString:
		return "string"
	case TypeEnum:
		return "enum"
	default:
		return "unknown"
	}
}

// ConfigKeySchema defines a known configuration key with its expected type and validation rules.
type ConfigKeySchema struct {
	Path          string          // Dotted key path (e.g., "notifications.enabled")
	Type          ConfigValueType // Expected value type for validation
	AllowedValues []string        // Valid values for enum types (empty for non-enums)
	Description   string          // Human-readable description for help text
	Default       interface{}     // Default value
}

// KnownKeys is the registry of all known configuration keys with their schemas.
var KnownKeys = map[string]ConfigKeySchema{
	"agent_preset": {
		Path:          "agent_preset",
		Type:          TypeEnum,
		AllowedValues: []string{"", "claude", "gemini", "cline", "codex", "opencode", "goose"},
		Description:   "Built-in agent preset to use",
		Default:       "",
	},
	"model": {
		Path:        "model",
		Type:        TypeString,
		Description: "Default model for workflow agent execution",
		Default:     "",
	},
	"opencode_agent": {
		Path:        "opencode_agent",
		Type:        TypeString,
		Description: "OpenCode sub-agent to use (build, plan, explore, etc.)",
		Default:     "",
	},
	"use_subscription": {
		Path:        "use_subscription",
		Type:        TypeBool,
		Description: "Force subscription mode (no API charges)",
		Default:     true,
	},
	"max_retries": {
		Path:        "max_retries",
		Type:        TypeInt,
		Description: "Maximum number of retry attempts",
		Default:     0,
	},
	"timeout": {
		Path:        "timeout",
		Type:        TypeInt,
		Description: "Timeout in seconds for Claude operations",
		Default:     2400,
	},
	"specs_dir": {
		Path:        "specs_dir",
		Type:        TypeString,
		Description: "Directory for spec files",
		Default:     "./specs",
	},
	"state_dir": {
		Path:        "state_dir",
		Type:        TypeString,
		Description: "Directory for state files",
		Default:     "~/.autospec/state",
	},
	"skip_preflight": {
		Path:        "skip_preflight",
		Type:        TypeBool,
		Description: "Skip preflight checks",
		Default:     false,
	},
	"skip_confirmations": {
		Path:        "skip_confirmations",
		Type:        TypeBool,
		Description: "Skip confirmation prompts",
		Default:     false,
	},
	"implement_method": {
		Path:          "implement_method",
		Type:          TypeEnum,
		AllowedValues: []string{"single-session", "phases", "tasks"},
		Description:   "Default execution mode for implement command",
		Default:       "phases",
	},
	"max_history_entries": {
		Path:        "max_history_entries",
		Type:        TypeInt,
		Description: "Maximum number of command history entries to retain",
		Default:     500,
	},
	"view_limit": {
		Path:        "view_limit",
		Type:        TypeInt,
		Description: "Number of recent specs to display in view command",
		Default:     5,
	},
	"default_agents": {
		Path:        "default_agents",
		Type:        TypeString, // Actually a list, but we handle as string for simplicity
		Description: "Agents to pre-select in 'autospec init' prompt",
		Default:     "",
	},
	"enable_risk_assessment": {
		Path:        "enable_risk_assessment",
		Type:        TypeBool,
		Description: "Enable risk assessment in plan generation",
		Default:     false,
	},
	"notifications.enabled": {
		Path:        "notifications.enabled",
		Type:        TypeBool,
		Description: "Enable or disable all notifications",
		Default:     false,
	},
	"notifications.type": {
		Path:          "notifications.type",
		Type:          TypeEnum,
		AllowedValues: []string{"sound", "visual", "both"},
		Description:   "Notification output type",
		Default:       "both",
	},
	"notifications.sound_file": {
		Path:        "notifications.sound_file",
		Type:        TypeString,
		Description: "Custom sound file path for notifications",
		Default:     "",
	},
	"notifications.on_command_complete": {
		Path:        "notifications.on_command_complete",
		Type:        TypeBool,
		Description: "Notify when command completes",
		Default:     true,
	},
	"notifications.on_stage_complete": {
		Path:        "notifications.on_stage_complete",
		Type:        TypeBool,
		Description: "Notify after each workflow stage",
		Default:     false,
	},
	"notifications.on_error": {
		Path:        "notifications.on_error",
		Type:        TypeBool,
		Description: "Notify on command or stage failure",
		Default:     true,
	},
	"notifications.on_long_running": {
		Path:        "notifications.on_long_running",
		Type:        TypeBool,
		Description: "Notify only if duration exceeds threshold",
		Default:     false,
	},
	"notifications.long_running_threshold": {
		Path:        "notifications.long_running_threshold",
		Type:        TypeDuration,
		Description: "Threshold for long-running notifications (e.g., 2m, 1h30m)",
		Default:     "2m",
	},
	"auto_commit": {
		Path:        "auto_commit",
		Type:        TypeBool,
		Description: "Enable automatic git commit creation after workflow completion",
		Default:     false,
	},
	"skip_permissions_notice_shown": {
		Path:        "skip_permissions_notice_shown",
		Type:        TypeBool,
		Description: "Whether the security notice about autonomous agent permissions has been shown",
		Default:     false,
	},
	"worktree.base_dir": {
		Path:        "worktree.base_dir",
		Type:        TypeString,
		Description: "Parent directory for new worktrees",
		Default:     "",
	},
	"worktree.prefix": {
		Path:        "worktree.prefix",
		Type:        TypeString,
		Description: "Directory name prefix for worktrees",
		Default:     "",
	},
	"worktree.setup_script": {
		Path:        "worktree.setup_script",
		Type:        TypeString,
		Description: "Path to setup script relative to repo",
		Default:     "",
	},
	"worktree.auto_setup": {
		Path:        "worktree.auto_setup",
		Type:        TypeBool,
		Description: "Run setup automatically on worktree create",
		Default:     true,
	},
	"worktree.track_status": {
		Path:        "worktree.track_status",
		Type:        TypeBool,
		Description: "Persist worktree state",
		Default:     true,
	},
	"worktree.copy_dirs": {
		Path:        "worktree.copy_dirs",
		Type:        TypeString, // Actually a list, but we handle as string for simplicity
		Description: "Non-tracked directories to copy to worktrees",
		Default:     ".autospec,.agents,.claude",
	},
	"worktree.setup_timeout": {
		Path:        "worktree.setup_timeout",
		Type:        TypeString, // Duration string like "5m", "30s"
		Description: "Maximum duration for setup script execution",
		Default:     "5m",
	},
	"cclean.verbose": {
		Path:        "cclean.verbose",
		Type:        TypeBool,
		Description: "Enable verbose output with usage stats and tool IDs (-V flag)",
		Default:     false,
	},
	"cclean.line_numbers": {
		Path:        "cclean.line_numbers",
		Type:        TypeBool,
		Description: "Show line numbers in formatted output (-n flag)",
		Default:     false,
	},
	"cclean.style": {
		Path:          "cclean.style",
		Type:          TypeEnum,
		AllowedValues: []string{"default", "compact", "minimal", "plain"},
		Description:   "Output formatting style for cclean (-s flag)",
		Default:       "default",
	},
	"codex_output.mode": {
		Path:          "codex_output.mode",
		Type:          TypeEnum,
		AllowedValues: []string{"compact", "full"},
		Description:   "Output mode for automated Codex runs",
		Default:       "compact",
	},
	"codex_output.max_lines_per_message": {
		Path:        "codex_output.max_lines_per_message",
		Type:        TypeInt,
		Description: "Maximum displayed lines per compact Codex output block",
		Default:     40,
	},
	"codex_output.color": {
		Path:        "codex_output.color",
		Type:        TypeBool,
		Description: "Enable ANSI color in compact Codex output",
		Default:     true,
	},
	"skip_permissions": {
		Path:        "skip_permissions",
		Type:        TypeBool,
		Description: "Enable autonomous mode for supported agents",
		Default:     true,
	},
	"verification.level": {
		Path:          "verification.level",
		Type:          TypeEnum,
		AllowedValues: []string{"basic", "enhanced", "full"},
		Description:   "Verification depth tier (basic, enhanced, full)",
		Default:       "basic",
	},
	"verification.adversarial_review": {
		Path:        "verification.adversarial_review",
		Type:        TypeBool,
		Description: "Enable adversarial review feature (overrides level default)",
		Default:     nil,
	},
	"verification.contracts": {
		Path:        "verification.contracts",
		Type:        TypeBool,
		Description: "Enable contracts verification feature (overrides level default)",
		Default:     nil,
	},
	"verification.property_tests": {
		Path:        "verification.property_tests",
		Type:        TypeBool,
		Description: "Enable property-based tests feature (overrides level default)",
		Default:     nil,
	},
	"verification.metamorphic_tests": {
		Path:        "verification.metamorphic_tests",
		Type:        TypeBool,
		Description: "Enable metamorphic tests feature (overrides level default)",
		Default:     nil,
	},
	"verification.mutation_threshold": {
		Path:        "verification.mutation_threshold",
		Type:        TypeFloat,
		Description: "Minimum mutation score threshold (0.0-1.0)",
		Default:     0.8,
	},
	"verification.coverage_threshold": {
		Path:        "verification.coverage_threshold",
		Type:        TypeFloat,
		Description: "Minimum code coverage threshold (0.0-1.0)",
		Default:     0.85,
	},
	"verification.complexity_max": {
		Path:        "verification.complexity_max",
		Type:        TypeInt,
		Description: "Maximum cyclomatic complexity allowed (positive integer)",
		Default:     10,
	},
	"verification.ears_requirements": {
		Path:        "verification.ears_requirements",
		Type:        TypeBool,
		Description: "Enable EARS requirements in spec.yaml (basic=disabled, enhanced/full=enabled by default)",
		Default:     nil,
	},
}

// ErrUnknownKey is returned when trying to access an unknown configuration key.
type ErrUnknownKey struct {
	Key string
}

func (e ErrUnknownKey) Error() string {
	return "unknown configuration key: " + e.Key
}

// GetKeySchema returns the schema for a known configuration key.
// Returns ErrUnknownKey if the key is not in the registry.
func GetKeySchema(path string) (ConfigKeySchema, error) {
	schema, ok := KnownKeys[path]
	if !ok {
		return ConfigKeySchema{}, ErrUnknownKey{Key: path}
	}
	return schema, nil
}

// InferType determines the ConfigValueType from a string value.
// Order of inference: bool literals -> integers -> durations -> string fallback.
func InferType(value string) ConfigValueType {
	if value == "true" || value == "false" {
		return TypeBool
	}
	if _, err := strconv.Atoi(value); err == nil {
		return TypeInt
	}
	if _, err := time.ParseDuration(value); err == nil {
		return TypeDuration
	}
	return TypeString
}

// ParsedValue represents a configuration value after type inference and validation.
type ParsedValue struct {
	Raw    string      // Original string input from user
	Parsed interface{} // Value converted to correct type
	Type   ConfigValueType
}

// ValidateValue validates a value against the schema for a given key.
// Returns the parsed value or an error with details about what's wrong.
func ValidateValue(key, value string) (ParsedValue, error) {
	schema, err := GetKeySchema(key)
	if err != nil {
		return ParsedValue{}, err
	}
	return validateAgainstSchema(schema, value)
}

// validateAgainstSchema validates a value against a specific schema.
func validateAgainstSchema(schema ConfigKeySchema, value string) (ParsedValue, error) {
	switch schema.Type {
	case TypeBool:
		return parseBoolValue(value)
	case TypeInt:
		return parseIntValue(value)
	case TypeFloat:
		return parseFloatValue(value)
	case TypeDuration:
		return parseDurationValue(value)
	case TypeEnum:
		return parseEnumValue(schema, value)
	case TypeString:
		return ParsedValue{Raw: value, Parsed: value, Type: TypeString}, nil
	default:
		return ParsedValue{}, fmt.Errorf("unsupported type: %v", schema.Type)
	}
}

// parseBoolValue parses and validates a boolean value.
func parseBoolValue(value string) (ParsedValue, error) {
	switch strings.ToLower(value) {
	case "true":
		return ParsedValue{Raw: value, Parsed: true, Type: TypeBool}, nil
	case "false":
		return ParsedValue{Raw: value, Parsed: false, Type: TypeBool}, nil
	default:
		return ParsedValue{}, fmt.Errorf("invalid boolean: %q (expected true or false)", value)
	}
}

// parseIntValue parses and validates an integer value.
func parseIntValue(value string) (ParsedValue, error) {
	n, err := strconv.Atoi(value)
	if err != nil {
		return ParsedValue{}, fmt.Errorf("invalid integer: %q", value)
	}
	return ParsedValue{Raw: value, Parsed: n, Type: TypeInt}, nil
}

// parseFloatValue parses and validates a float value.
func parseFloatValue(value string) (ParsedValue, error) {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return ParsedValue{}, fmt.Errorf("invalid float: %q", value)
	}
	return ParsedValue{Raw: value, Parsed: f, Type: TypeFloat}, nil
}

// parseDurationValue parses and validates a duration value.
func parseDurationValue(value string) (ParsedValue, error) {
	d, err := time.ParseDuration(value)
	if err != nil {
		return ParsedValue{}, fmt.Errorf("invalid duration: %q (examples: 5m, 1h30m, 10s)", value)
	}
	return ParsedValue{Raw: value, Parsed: d.String(), Type: TypeDuration}, nil
}

// parseEnumValue validates a value against allowed enum options.
func parseEnumValue(schema ConfigKeySchema, value string) (ParsedValue, error) {
	for _, allowed := range schema.AllowedValues {
		if value == allowed {
			return ParsedValue{Raw: value, Parsed: value, Type: TypeEnum}, nil
		}
	}
	return ParsedValue{}, fmt.Errorf(
		"invalid value: %q (valid options: %s)",
		value,
		strings.Join(schema.AllowedValues, ", "),
	)
}
