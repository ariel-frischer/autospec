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
	JcodeModeAuto    JcodeMode = "auto"
)

const redactedJcodeValue = "[redacted]"

const (
	defaultJcodeReconnectAttempts = 2
	defaultJcodeRestartAttempts   = 1
	defaultJcodeRetryDelay        = 250 * time.Millisecond
	maxJcodeRecoveryAttempts      = 10
	maxJcodeRetryDelay            = 5 * time.Minute
)

// JcodeConfig contains runtime and SDK settings for the native jcode agent.
type JcodeConfig struct {
	Mode              JcodeMode     `yaml:"mode,omitempty" koanf:"mode"`
	SocketPath        string        `yaml:"socket_path,omitempty" koanf:"socket_path"`
	Binary            string        `yaml:"binary,omitempty" koanf:"binary"`
	Home              string        `yaml:"home,omitempty" koanf:"home"`
	InheritLogins     bool          `yaml:"inherit_logins" koanf:"inherit_logins"`
	StartupTimeout    time.Duration `yaml:"startup_timeout,omitempty" koanf:"startup_timeout"`
	CleanupTimeout    time.Duration `yaml:"cleanup_timeout,omitempty" koanf:"cleanup_timeout"`
	StartupCommand    string        `yaml:"startup_command,omitempty" koanf:"startup_command"`
	ReconnectAttempts int           `yaml:"reconnect_attempts,omitempty" koanf:"reconnect_attempts"`
	RestartAttempts   int           `yaml:"restart_attempts,omitempty" koanf:"restart_attempts"`
	RetryDelay        time.Duration `yaml:"retry_delay,omitempty" koanf:"retry_delay"`
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
	if c.StartupCommand != "" {
		c.StartupCommand = redactedJcodeValue
	}
	return c
}

func validateJcodeConfig(c JcodeConfig, filePath string) error {
	if c.Mode != "" && c.Mode != JcodeModeConnect && c.Mode != JcodeModePrivate && c.Mode != JcodeModeAuto {
		return &ValidationError{FilePath: filePath, Field: "jcode.mode", Message: "must be one of: connect, private, auto"}
	}
	if c.StartupTimeout < 0 {
		return &ValidationError{FilePath: filePath, Field: "jcode.startup_timeout", Message: "must be positive when set"}
	}
	if c.CleanupTimeout < 0 {
		return &ValidationError{FilePath: filePath, Field: "jcode.cleanup_timeout", Message: "must be positive when set"}
	}
	mode := c.Mode
	if mode == "" {
		mode = JcodeModeConnect
	}
	if c.StartupCommand != "" && mode == JcodeModeConnect {
		return &ValidationError{FilePath: filePath, Field: "jcode.startup_command", Message: "is only valid for private or auto mode"}
	}
	if c.StartupCommand != "" && strings.TrimSpace(c.StartupCommand) == "" {
		return &ValidationError{FilePath: filePath, Field: "jcode.startup_command", Message: "must not be blank"}
	}
	if strings.ContainsAny(c.StartupCommand, ";&|$`\n\r") {
		return &ValidationError{FilePath: filePath, Field: "jcode.startup_command", Message: "must not contain shell metacharacters"}
	}
	if c.ReconnectAttempts < 0 || c.ReconnectAttempts > maxJcodeRecoveryAttempts {
		return &ValidationError{FilePath: filePath, Field: "jcode.reconnect_attempts", Message: "must be between 0 and 10"}
	}
	if c.RestartAttempts < 0 || c.RestartAttempts > maxJcodeRecoveryAttempts {
		return &ValidationError{FilePath: filePath, Field: "jcode.restart_attempts", Message: "must be between 0 and 10"}
	}
	if c.RetryDelay < 0 || c.RetryDelay > maxJcodeRetryDelay {
		return &ValidationError{FilePath: filePath, Field: "jcode.retry_delay", Message: "must be between 0 and 5m"}
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
