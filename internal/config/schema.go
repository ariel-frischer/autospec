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
	"claude_cmd": {
		Path:        "claude_cmd",
		Type:        TypeString,
		Description: "Path to the Claude CLI executable",
		Default:     "claude",
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
	"output_style": {
		Path:          "output_style",
		Type:          TypeEnum,
		AllowedValues: []string{"default", "compact", "minimal", "plain", "raw"},
		Description:   "Output formatting style for Claude stream-json display",
		Default:       "default",
	},
	"skip_permissions_notice_shown": {
		Path:        "skip_permissions_notice_shown",
		Type:        TypeBool,
		Description: "Whether the --dangerously-skip-permissions security notice has been shown",
		Default:     false,
	},
	"auto_commit": {
		Path:        "auto_commit",
		Type:        TypeBool,
		Description: "Enable automatic git commit creation after workflow completion",
		Default:     false,
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
