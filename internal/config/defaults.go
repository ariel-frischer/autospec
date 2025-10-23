package config

// GetDefaults returns the default configuration values
func GetDefaults() map[string]interface{} {
	return map[string]interface{}{
		"claude_cmd": "claude",
		"claude_args": []string{
			"-p",
			"--dangerously-skip-permissions",
			"--verbose",
			"--output-format",
			"stream-json",
		},
		"use_api_key":       false,
		"custom_claude_cmd": "",
		"specify_cmd":       "specify",
		"max_retries":       3,
		"specs_dir":         "./specs",
		"state_dir":         "~/.autospec/state",
		"skip_preflight":    false,
		"timeout":           0,
		"show_progress":     false, // Progress indicators off by default (professional)
	}
}
