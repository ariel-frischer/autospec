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

// JcodeRunner selects the implementation used for jcode execution.
type JcodeRunner string

// JcodeMCPTools controls how MCP tools are exposed to the model.
type JcodeMCPTools string

const (
	JcodeRunnerExec   JcodeRunner = "exec"
	JcodeRunnerSDK    JcodeRunner = "sdk"
	JcodeRunnerCustom JcodeRunner = "custom"

	JcodeMCPToolsAuto     JcodeMCPTools = "auto"
	JcodeMCPToolsEager    JcodeMCPTools = "eager"
	JcodeMCPToolsDeferred JcodeMCPTools = "deferred"
)

const redactedJcodeValue = "[redacted]"

const (
	defaultJcodeReconnectAttempts = 2
	defaultJcodeRestartAttempts   = 1
	defaultJcodeRetryDelay        = 250 * time.Millisecond
	maxJcodeRecoveryAttempts      = 10
	maxJcodeRetryDelay            = 5 * time.Minute
)

// JcodeConfig contains official CLI options and opt-in SDK runtime settings.
type JcodeConfig struct {
	// Runner selects the production CLI runner or an explicit compatibility path.
	Runner            JcodeRunner   `yaml:"runner,omitempty" koanf:"runner"`
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
	Provider          string        `yaml:"provider,omitempty" koanf:"provider"`
	ProviderProfile   string        `yaml:"provider_profile,omitempty" koanf:"provider_profile"`
	SessionProfile    string        `yaml:"session_profile,omitempty" koanf:"session_profile"`
	MaxTurns          int           `yaml:"max_turns,omitempty" koanf:"max_turns"`
	TokenBudget       int           `yaml:"token_budget,omitempty" koanf:"token_budget"`
	Deadline          string        `yaml:"deadline,omitempty" koanf:"deadline"`
	Trace             bool          `yaml:"trace" koanf:"trace"`
	ToolProfile       string        `yaml:"tool_profile,omitempty" koanf:"tool_profile"`
	Tools             string        `yaml:"tools,omitempty" koanf:"tools"`
	DisabledTools     string        `yaml:"disabled_tools,omitempty" koanf:"disabled_tools"`
	DisableBaseTools  bool          `yaml:"disable_base_tools" koanf:"disable_base_tools"`
	MCPTools          JcodeMCPTools `yaml:"mcp_tools,omitempty" koanf:"mcp_tools"`
	MCPToolsThreshold int           `yaml:"mcp_tools_token_threshold,omitempty" koanf:"mcp_tools_token_threshold"`
}

// EffectiveRunner returns the CLI-compatible runner unless explicitly changed.
func (c JcodeConfig) EffectiveRunner() JcodeRunner {
	if c.Runner == "" {
		return JcodeRunnerExec
	}
	return c.Runner
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
	for _, validate := range []func(JcodeConfig, string) error{
		validateJcodeEnums,
		validateJcodeLifecycle,
		validateJcodeSDKSessionControls,
		validateJcodeExecOptions,
		validateJcodePaths,
	} {
		if err := validate(c, filePath); err != nil {
			return err
		}
	}
	return nil
}

func validateJcodeSDKSessionControls(c JcodeConfig, filePath string) error {
	if c.SessionProfile != "" && strings.TrimSpace(c.SessionProfile) == "" {
		return &ValidationError{FilePath: filePath, Field: "jcode.session_profile", Message: "must be non-blank when set"}
	}
	for _, limit := range []struct {
		field string
		value int
	}{
		{field: "jcode.max_turns", value: c.MaxTurns},
		{field: "jcode.token_budget", value: c.TokenBudget},
	} {
		if limit.value < 0 {
			return &ValidationError{FilePath: filePath, Field: limit.field, Message: "must be positive when set"}
		}
	}
	return validateJcodeDeadline(c.Deadline, filePath)
}

func validateJcodeDeadline(value, filePath string) error {
	if value == "" {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, value); err == nil && hasExplicitRFC3339Offset(value) {
		return nil
	}
	return &ValidationError{
		FilePath: filePath,
		Field:    "jcode.deadline",
		Message:  "must be an RFC3339 timestamp with an explicit UTC offset (Z or +HH:MM/-HH:MM)",
	}
}

func hasExplicitRFC3339Offset(value string) bool {
	if strings.HasSuffix(value, "Z") {
		return true
	}
	if len(value) < 6 || value[len(value)-3] != ':' {
		return false
	}
	sign := value[len(value)-6]
	return sign == '+' || sign == '-'
}

func validateJcodeEnums(c JcodeConfig, filePath string) error {
	if c.Runner != "" && c.Runner != JcodeRunnerExec && c.Runner != JcodeRunnerSDK && c.Runner != JcodeRunnerCustom {
		return &ValidationError{FilePath: filePath, Field: "jcode.runner", Message: "must be one of: exec, sdk, custom"}
	}
	if c.Mode != "" && c.Mode != JcodeModeConnect && c.Mode != JcodeModePrivate && c.Mode != JcodeModeAuto {
		return &ValidationError{FilePath: filePath, Field: "jcode.mode", Message: "must be one of: connect, private, auto"}
	}
	if c.MCPTools != "" && c.MCPTools != JcodeMCPToolsAuto && c.MCPTools != JcodeMCPToolsEager && c.MCPTools != JcodeMCPToolsDeferred {
		return &ValidationError{FilePath: filePath, Field: "jcode.mcp_tools", Message: "must be one of: auto, eager, deferred"}
	}
	return nil
}

func validateJcodeLifecycle(c JcodeConfig, filePath string) error {
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
	return nil
}

func validateJcodeExecOptions(c JcodeConfig, filePath string) error {
	if c.MCPToolsThreshold < 0 {
		return &ValidationError{FilePath: filePath, Field: "jcode.mcp_tools_token_threshold", Message: "must be positive when set"}
	}
	values := []struct{ field, value string }{
		{field: "jcode.provider", value: c.Provider},
		{field: "jcode.provider_profile", value: c.ProviderProfile},
		{field: "jcode.tool_profile", value: c.ToolProfile},
	}
	for _, value := range values {
		if err := validateJcodeValue(value.field, value.value, filePath); err != nil {
			return err
		}
	}
	for _, list := range []struct{ field, value string }{
		{field: "jcode.tools", value: c.Tools},
		{field: "jcode.disabled_tools", value: c.DisabledTools},
	} {
		if err := validateJcodeList(list.field, list.value, filePath); err != nil {
			return err
		}
	}
	return nil
}

func validateJcodePaths(c JcodeConfig, filePath string) error {
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

func validateJcodeList(field, value, filePath string) error {
	if value == "" {
		return nil
	}
	for _, item := range strings.Split(value, ",") {
		if err := validateJcodeValue(field, item, filePath); err != nil {
			return err
		}
	}
	return nil
}

func validateJcodeValue(field, value, filePath string) error {
	if value == "" || (strings.TrimSpace(value) == value && !strings.ContainsAny(value, ",;&|$`\n\r")) {
		return nil
	}
	return &ValidationError{FilePath: filePath, Field: field, Message: "must be non-blank and contain no commas or shell metacharacters"}
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
