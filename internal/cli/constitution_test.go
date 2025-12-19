// Package cli tests the constitution command registration, examples, and flag validation.
// Related: internal/cli/constitution.go
// Tags: cli, constitution, command, validation
package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstitutionCmdRegistration(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "constitution [optional-prompt]" {
			found = true
			break
		}
	}
	assert.True(t, found, "constitution command should be registered")
}

func TestConstitutionCmdExamples(t *testing.T) {
	examples := []string{
		"autospec constitution",
		"Focus on security",
		"test-driven development",
	}

	for _, example := range examples {
		assert.Contains(t, constitutionCmd.Example, example)
	}
}

func TestConstitutionCmdLongDescription(t *testing.T) {
	keywords := []string{
		"constitution",
		".autospec/constitution.yaml",
		"principles",
	}

	for _, keyword := range keywords {
		assert.Contains(t, constitutionCmd.Long, keyword)
	}
}

func TestConstitutionCmdAcceptsOptionalPrompt(t *testing.T) {
	// Command should accept arbitrary args (for optional prompt)
	assert.Contains(t, constitutionCmd.Use, "[optional-prompt]")
}

func TestConstitutionCmd_MaxRetriesFlag(t *testing.T) {
	// constitution should have max-retries flag
	flag := constitutionCmd.Flags().Lookup("max-retries")
	require.NotNil(t, flag, "max-retries flag should exist")
	assert.Equal(t, "r", flag.Shorthand, "max-retries should have shorthand 'r'")
	assert.Equal(t, "0", flag.DefValue, "max-retries should default to 0")
}

func TestConstitutionCmd_InheritedFlags(t *testing.T) {
	// constitution should inherit skip-preflight from root
	f := rootCmd.PersistentFlags().Lookup("skip-preflight")
	require.NotNil(t, f)

	// constitution should inherit config from root
	f = rootCmd.PersistentFlags().Lookup("config")
	require.NotNil(t, f)
}
