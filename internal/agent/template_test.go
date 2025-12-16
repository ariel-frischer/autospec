package agent

import (
	"strings"
	"testing"
	"time"
)

func TestGetDefaultTemplate(t *testing.T) {
	template := GetDefaultTemplate()

	// Verify template is not empty
	if template == "" {
		t.Error("GetDefaultTemplate() returned empty string")
	}

	// Verify template contains required sections (FR-003)
	requiredSections := []string{
		"## Active Technologies",
		"## Recent Changes",
		"**Last updated**",
	}

	for _, section := range requiredSections {
		if !strings.Contains(template, section) {
			t.Errorf("GetDefaultTemplate() missing required section: %q", section)
		}
	}
}

func TestGetDefaultTemplate_ProjectSections(t *testing.T) {
	template := GetDefaultTemplate()

	// Verify template contains useful project sections
	expectedSections := []string{
		"# Project Context",
		"## Project Overview",
		"## Directory Structure",
		"## Development Guidelines",
	}

	for _, section := range expectedSections {
		if !strings.Contains(template, section) {
			t.Errorf("GetDefaultTemplate() missing expected section: %q", section)
		}
	}
}

func TestGetTimestamp(t *testing.T) {
	timestamp := GetTimestamp()

	// Verify timestamp format is YYYY-MM-DD
	_, err := time.Parse("2006-01-02", timestamp)
	if err != nil {
		t.Errorf("GetTimestamp() returned invalid format: %q, error: %v", timestamp, err)
	}

	// Verify timestamp is today's date
	today := time.Now().Format("2006-01-02")
	if timestamp != today {
		t.Errorf("GetTimestamp() = %q, want %q", timestamp, today)
	}
}

func TestGetDefaultTemplate_NotBlankSections(t *testing.T) {
	template := GetDefaultTemplate()

	// Verify template has placeholder comments in sections
	// This ensures users know where to add content
	if !strings.Contains(template, "<!--") {
		t.Error("GetDefaultTemplate() should contain comment placeholders for user guidance")
	}
}
