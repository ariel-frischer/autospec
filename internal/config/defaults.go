package config

import "time"

// GetDefaultConfigTemplate returns a fully commented config template
// that helps users understand all available options
func GetDefaultConfigTemplate() string {
	return `# Autospec Configuration
# See 'autospec --help' for command reference

# Claude CLI settings
claude_cmd: claude                    # Claude CLI command
claude_args:                          # Arguments passed to Claude CLI
  - -p
  - --verbose
  - --output-format
  - stream-json
custom_claude_cmd: ""                 # Custom command (overrides claude_cmd + claude_args)

# Workflow settings
max_retries: 0                        # Max retry attempts per stage (0-10)
specs_dir: ./specs                    # Directory for feature specs
state_dir: ~/.autospec/state          # Directory for state files
skip_preflight: false                 # Skip preflight checks
timeout: 2400                         # Timeout in seconds (40 min default, 0 = no timeout)
skip_confirmations: false             # Skip confirmation prompts
implement_method: phases              # Default: phases | tasks | single-session

# History settings
max_history_entries: 500              # Max command history entries to retain

# Notifications (all platforms)
notifications:
  enabled: false                      # Enable notifications (opt-in)
  type: both                          # sound | visual | both
  sound_file: ""                      # Custom sound file path (empty = system default)
  on_command_complete: true           # Notify when command finishes
  on_stage_complete: false            # Notify on each stage completion
  on_error: true                      # Notify on failures
  on_long_running: false              # Enable duration-based notifications
  long_running_threshold: 30s         # Threshold for long-running notification
`
}

// GetDefaults returns the default configuration values
func GetDefaults() map[string]interface{} {
	return map[string]interface{}{
		"claude_cmd": "claude",
		"claude_args": []string{
			"-p",
			"--verbose",
			"--output-format",
			"stream-json",
		},
		"custom_claude_cmd":  "",
		"max_retries":        0,
		"specs_dir":          "./specs",
		"state_dir":          "~/.autospec/state",
		"skip_preflight":     false,
		"timeout":            2400,  // 40 minutes default
		"skip_confirmations": false, // Confirmation prompts enabled by default
		// implement_method: Default to "phases" for cost-efficient execution with context isolation.
		// This changes the legacy behavior (single-session) to run each phase in a separate Claude session.
		// Valid values: "single-session", "phases", "tasks"
		"implement_method": "phases",
		// notifications: Notification settings for command and stage completion.
		// Disabled by default (opt-in). When enabled, defaults to both sound and visual notifications.
		"notifications": map[string]interface{}{
			"enabled":                false,                       // Disabled by default (opt-in)
			"type":                   "both",                      // Both sound and visual when enabled
			"sound_file":             "",                          // Use system default sound
			"on_command_complete":    true,                        // Notify when command finishes (default when enabled)
			"on_stage_complete":      false,                       // Don't notify on each stage by default
			"on_error":               true,                        // Notify on failures (default when enabled)
			"on_long_running":        false,                       // Don't use duration threshold by default
			"long_running_threshold": (30 * time.Second).String(), // 30 seconds threshold
		},
		// max_history_entries: Maximum number of command history entries to retain.
		// Oldest entries are pruned when this limit is exceeded.
		"max_history_entries": 500,
	}
}
