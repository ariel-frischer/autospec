package config

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
		"use_api_key":        false,
		"custom_claude_cmd":  "",
		"max_retries":        3,
		"specs_dir":          "./specs",
		"state_dir":          "~/.autospec/state",
		"skip_preflight":     false,
		"timeout":            2400,  // 40 minutes default
		"show_progress":      false, // Progress indicators off by default (professional)
		"skip_confirmations": false, // Confirmation prompts enabled by default
	}
}
