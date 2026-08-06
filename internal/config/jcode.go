package config

import (
	"strings"
	"time"
)

// JcodeMode controls whether autospec connects to or launches a jcode runtime.
type JcodeMode string

const (
	JcodeModeConnect JcodeMode = "connect"
	JcodeModePrivate JcodeMode = "private"
)

const redactedJcodeValue = "[redacted]"

// JcodeConfig contains runtime and SDK settings for the native jcode agent.
type JcodeConfig struct {
	Mode           JcodeMode     `yaml:"mode,omitempty" koanf:"mode"`
	SocketPath     string        `yaml:"socket_path,omitempty" koanf:"socket_path"`
	Binary         string        `yaml:"binary,omitempty" koanf:"binary"`
	Home           string        `yaml:"home,omitempty" koanf:"home"`
	InheritLogins  bool          `yaml:"inherit_logins" koanf:"inherit_logins"`
	StartupTimeout time.Duration `yaml:"startup_timeout,omitempty" koanf:"startup_timeout"`
	CleanupTimeout time.Duration `yaml:"cleanup_timeout,omitempty" koanf:"cleanup_timeout"`
}

// Redacted returns a copy safe for user-facing diagnostics.
func (c JcodeConfig) Redacted() JcodeConfig {
	if c.SocketPath != "" {
		c.SocketPath = redactedJcodeValue
	}
	if c.Binary != "" {
		c.Binary = redactedJcodeValue
	}
	if c.Home != "" {
		c.Home = redactedJcodeValue
	}
	return c
}

func validateJcodeConfig(c JcodeConfig, filePath string) error {
	if c.Mode != "" && c.Mode != JcodeModeConnect && c.Mode != JcodeModePrivate {
		return &ValidationError{FilePath: filePath, Field: "jcode.mode", Message: "must be one of: connect, private"}
	}
	if c.StartupTimeout < 0 {
		return &ValidationError{FilePath: filePath, Field: "jcode.startup_timeout", Message: "must be positive when set"}
	}
	if c.CleanupTimeout < 0 {
		return &ValidationError{FilePath: filePath, Field: "jcode.cleanup_timeout", Message: "must be positive when set"}
	}
	paths := []struct {
		field string
		value string
	}{
		{field: "jcode.socket_path", value: c.SocketPath},
		{field: "jcode.binary", value: c.Binary},
		{field: "jcode.home", value: c.Home},
	}
	for _, path := range paths {
		if err := validateJcodePath(path.field, path.value, filePath); err != nil {
			return err
		}
	}
	return nil
}

func validateJcodePath(field, value, filePath string) error {
	if value == "" || !strings.ContainsAny(value, ";&|$`\n\r") {
		return nil
	}
	return &ValidationError{
		FilePath: filePath,
		Field:    field,
		Message:  "must not contain shell metacharacters",
	}
}
