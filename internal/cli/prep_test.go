package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepCmdRegistration(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "prep <feature-description>" {
			found = true
			break
		}
	}
	assert.True(t, found, "prep command should be registered")
}

func TestPrepCmdRequiresExactlyOneArg(t *testing.T) {
	// Should require exactly 1 arg
	err := prepCmd.Args(prepCmd, []string{})
	assert.Error(t, err)

	err = prepCmd.Args(prepCmd, []string{"feature description"})
	assert.NoError(t, err)

	err = prepCmd.Args(prepCmd, []string{"arg1", "arg2"})
	assert.Error(t, err)
}

func TestPrepCmdFlags(t *testing.T) {
	// max-retries flag should exist
	f := prepCmd.Flags().Lookup("max-retries")
	require.NotNil(t, f)
	assert.Equal(t, "r", f.Shorthand)
	assert.Equal(t, "0", f.DefValue)
}

func TestPrepCmdExamples(t *testing.T) {
	examples := []string{
		"autospec prep",
		"Add user authentication",
		"Refactor database",
	}

	for _, example := range examples {
		assert.Contains(t, prepCmd.Example, example)
	}
}

func TestPrepCmdLongDescription(t *testing.T) {
	keywords := []string{
		"pre-flight",
		"specify",
		"plan",
		"tasks",
		"validate",
		"retry",
	}

	for _, keyword := range keywords {
		assert.Contains(t, prepCmd.Long, keyword)
	}
}

func TestPrepCmd_ExcludesImplement(t *testing.T) {
	// The prep command description should NOT mention implement in the steps
	// because prep excludes the implement phase (but it can mention "implementation" in context)
	assert.NotContains(t, prepCmd.Long, "Execute /autospec.implement")
	assert.Contains(t, prepCmd.Short, "specify")
	assert.Contains(t, prepCmd.Short, "plan")
	assert.Contains(t, prepCmd.Short, "tasks")
}

func TestPrepCmd_InheritedFlags(t *testing.T) {
	// Should inherit skip-preflight from root
	f := rootCmd.PersistentFlags().Lookup("skip-preflight")
	require.NotNil(t, f)

	// Should inherit config from root
	f = rootCmd.PersistentFlags().Lookup("config")
	require.NotNil(t, f)
}

func TestPrepCmd_MaxRetriesDefault(t *testing.T) {
	// Default should be 0 (use config)
	f := prepCmd.Flags().Lookup("max-retries")
	require.NotNil(t, f)
	assert.Equal(t, "0", f.DefValue)
}
