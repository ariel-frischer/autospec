// Package config_test tests default configuration values and template generation.
// Related: internal/config/defaults.go
// Tags: config, defaults, configuration, template, notifications
package config

import (
	"strings"
	"testing"
)

func TestGetDefaultConfigTemplate(t *testing.T) {
	t.Parallel()

	template := GetDefaultConfigTemplate()

	// Verify template is not empty
	if template == "" {
		t.Error("GetDefaultConfigTemplate() returned empty string")
	}

	// Verify key sections are present
	expectedSections := []string{
		"Claude CLI settings",
		"claude_cmd:",
		"claude_args:",
		"custom_claude_cmd:",
		"Workflow settings",
		"max_retries:",
		"specs_dir:",
		"state_dir:",
		"skip_preflight:",
		"timeout:",
		"skip_confirmations:",
		"implement_method:",
		"History settings",
		"max_history_entries:",
		"Notifications",
		"notifications:",
		"enabled:",
		"type:",
		"sound_file:",
		"on_command_complete:",
		"on_stage_complete:",
		"on_error:",
		"on_long_running:",
		"long_running_threshold:",
		"Agent initialization settings",
		"default_agents:",
	}

	for _, section := range expectedSections {
		if !strings.Contains(template, section) {
			t.Errorf("GetDefaultConfigTemplate() missing section: %s", section)
		}
	}
}

func TestGetDefaults(t *testing.T) {
	t.Parallel()

	defaults := GetDefaults()

	// Verify required keys exist
	requiredKeys := []string{
		"claude_cmd",
		"claude_args",
		"custom_claude_cmd",
		"max_retries",
		"specs_dir",
		"state_dir",
		"skip_preflight",
		"timeout",
		"skip_confirmations",
		"implement_method",
		"notifications",
		"max_history_entries",
		"default_agents",
	}

	for _, key := range requiredKeys {
		if _, ok := defaults[key]; !ok {
			t.Errorf("GetDefaults() missing required key: %s", key)
		}
	}

	// Verify specific default values
	if defaults["claude_cmd"] != "claude" {
		t.Errorf("claude_cmd default = %v, want 'claude'", defaults["claude_cmd"])
	}

	if defaults["max_retries"] != 0 {
		t.Errorf("max_retries default = %v, want 0", defaults["max_retries"])
	}

	if defaults["timeout"] != 2400 {
		t.Errorf("timeout default = %v, want 2400", defaults["timeout"])
	}

	if defaults["implement_method"] != "phases" {
		t.Errorf("implement_method default = %v, want 'phases'", defaults["implement_method"])
	}

	if defaults["max_history_entries"] != 500 {
		t.Errorf("max_history_entries default = %v, want 500", defaults["max_history_entries"])
	}

	// Verify notifications defaults
	notifications, ok := defaults["notifications"].(map[string]interface{})
	if !ok {
		t.Error("notifications should be a map")
		return
	}

	if notifications["enabled"] != false {
		t.Errorf("notifications.enabled default = %v, want false", notifications["enabled"])
	}

	if notifications["type"] != "both" {
		t.Errorf("notifications.type default = %v, want 'both'", notifications["type"])
	}

	if notifications["on_command_complete"] != true {
		t.Errorf("notifications.on_command_complete default = %v, want true", notifications["on_command_complete"])
	}

	if notifications["on_error"] != true {
		t.Errorf("notifications.on_error default = %v, want true", notifications["on_error"])
	}
}
