// Package cli tests the checklist command registration, examples, and flag validation.
// Related: internal/cli/checklist.go
// Tags: cli, checklist, command, validation
package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChecklistCmdRegistration(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "checklist [optional-prompt]" {
			found = true
			break
		}
	}
	assert.True(t, found, "checklist command should be registered")
}

func TestChecklistCmdExamples(t *testing.T) {
	examples := []string{
		"autospec checklist",
		"Focus on security",
		"accessibility",
	}

	for _, example := range examples {
		assert.Contains(t, checklistCmd.Example, example)
	}
}

func TestChecklistCmdLongDescription(t *testing.T) {
	keywords := []string{
		"checklist",
		"spec.yaml",
		"checklists/",
	}

	for _, keyword := range keywords {
		assert.Contains(t, checklistCmd.Long, keyword)
	}
}

func TestChecklistCmdAcceptsOptionalPrompt(t *testing.T) {
	// Command should accept arbitrary args (for optional prompt)
	assert.Contains(t, checklistCmd.Use, "[optional-prompt]")
}

func TestChecklistCmd_MaxRetriesFlag(t *testing.T) {
	// checklist should have max-retries flag
	flag := checklistCmd.Flags().Lookup("max-retries")
	require.NotNil(t, flag, "max-retries flag should exist")
	assert.Equal(t, "r", flag.Shorthand, "max-retries should have shorthand 'r'")
	assert.Equal(t, "0", flag.DefValue, "max-retries should default to 0")
}

func TestChecklistCmd_InheritedFlags(t *testing.T) {
	// checklist should inherit skip-preflight from root
	f := rootCmd.PersistentFlags().Lookup("skip-preflight")
	require.NotNil(t, f)

	// checklist should inherit config from root
	f = rootCmd.PersistentFlags().Lookup("config")
	require.NotNil(t, f)
}
