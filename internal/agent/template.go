package agent

import (
	_ "embed"
	"time"
)

// defaultTemplate is the embedded default agent file template.
// This is used when creating new agent context files.
//
//go:embed template.md
var defaultTemplate string

// GetDefaultTemplate returns the default agent file template content.
func GetDefaultTemplate() string {
	return defaultTemplate
}

// GetTimestamp returns the current timestamp formatted for the agent file.
func GetTimestamp() string {
	return time.Now().Format("2006-01-02")
}
