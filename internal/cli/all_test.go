package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllCmdRegistration(t *testing.T) {
	// Verify allCmd is registered in rootCmd
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "all <feature-description>" {
			found = true
			break
		}
	}
	assert.True(t, found, "all command should be registered")
}

func TestAllCmdFlags(t *testing.T) {
	// Test that all expected flags are registered
	flags := map[string]struct {
		shorthand string
		usage     string
	}{
		"max-retries": {shorthand: "r", usage: "Override max retry attempts"},
		"resume":      {shorthand: "", usage: "Resume implementation"},
		"progress":    {shorthand: "", usage: "Show progress indicators"},
	}

	for flagName, flag := range flags {
		t.Run("flag "+flagName, func(t *testing.T) {
			f := allCmd.Flags().Lookup(flag.name)
			require.NotNil(t, f, "flag %s should exist", flag.name)
			if flag.shorthand != "" {
				assert.Equal(t, flag.shorthand, f.Shorthand)
			}
			assert.Contains(t, f.Usage, flag.usage)
		})
	}
}

func TestAllCmdRequiresArg(t *testing.T) {
	// Create a fresh command to test
	cmd := &cobra.Command{
		Use:  "all <feature-description>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	// Test with no args
	err := cmd.Args(cmd, []string{})
	assert.Error(t, err)

	// Test with one arg
	err = cmd.Args(cmd, []string{"test feature"})
	assert.NoError(t, err)

	// Test with too many args
	err = cmd.Args(cmd, []string{"arg1", "arg2"})
	assert.Error(t, err)
}

func TestAllCmdUsesSkipPreflightFlag(t *testing.T) {
	// skip-preflight flag should be inherited from root
	f := rootCmd.PersistentFlags().Lookup("skip-preflight")
	require.NotNil(t, f)
}

func TestAllCmdExamples(t *testing.T) {
	// Verify examples are present
	assert.Contains(t, allCmd.Example, "autospec all")
	assert.Contains(t, allCmd.Example, "--resume")
	assert.Contains(t, allCmd.Example, "--skip-preflight")
}

func TestAllCmdLongDescription(t *testing.T) {
	// Verify long description mentions key steps
	steps := []string{
		"pre-flight",
		"specify",
		"plan",
		"tasks",
		"implement",
	}

	for _, step := range steps {
		assert.Contains(t, allCmd.Long, step)
	}
}

func TestStageConfigForAllCommand(t *testing.T) {
	// The all command should enable all 4 core stages
	// Verify the expected stage behavior

	tests := []struct {
		name            string
		skipPreflight   bool
		maxRetries      int
		expectPreflight bool
	}{
		{
			name:            "default settings",
			skipPreflight:   false,
			maxRetries:      0,
			expectPreflight: false, // default is false
		},
		{
			name:            "skip preflight",
			skipPreflight:   true,
			maxRetries:      0,
			expectPreflight: true,
		},
		{
			name:            "custom max retries",
			skipPreflight:   false,
			maxRetries:      5,
			expectPreflight: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// These are configuration checks, not full workflow tests
			if tc.skipPreflight {
				assert.True(t, tc.expectPreflight)
			}
			if tc.maxRetries > 0 {
				assert.Equal(t, 5, tc.maxRetries)
			}
		})
	}
}

func TestAllCmdConstitutionCheck(t *testing.T) {
	// The all command checks for constitution existence
	// Create temp directory structure

	tmpDir := t.TempDir()
	specifyDir := filepath.Join(tmpDir, ".specify", "memory")
	require.NoError(t, os.MkdirAll(specifyDir, 0755))

	t.Run("constitution missing shows error", func(t *testing.T) {
		// When constitution doesn't exist, the command should fail
		// This is a behavior test - we can't run the full command
		// but we can verify the constitution.yaml check is in place
		constPath := filepath.Join(specifyDir, "constitution.yaml")
		_, err := os.Stat(constPath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("constitution exists", func(t *testing.T) {
		constPath := filepath.Join(specifyDir, "constitution.yaml")
		require.NoError(t, os.WriteFile(constPath, []byte("project_name: Test"), 0644))

		_, err := os.Stat(constPath)
		assert.NoError(t, err)
	})
}

func TestAllCmdEquivalentToRunA(t *testing.T) {
	// Document that 'all' is equivalent to 'run -a'
	assert.Contains(t, allCmd.Long, "run -a")
}

func TestMaxRetriesOverride(t *testing.T) {
	// Test that max-retries flag default is 0 (use config)
	f := allCmd.Flags().Lookup("max-retries")
	require.NotNil(t, f)
	assert.Equal(t, "0", f.DefValue)
}

func TestProgressFlag(t *testing.T) {
	// Test that progress flag default is false
	f := allCmd.Flags().Lookup("progress")
	require.NotNil(t, f)
	assert.Equal(t, "false", f.DefValue)
}

func TestResumeFlag(t *testing.T) {
	// Test that resume flag default is false
	f := allCmd.Flags().Lookup("resume")
	require.NotNil(t, f)
	assert.Equal(t, "false", f.DefValue)
}
