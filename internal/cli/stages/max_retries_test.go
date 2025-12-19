// Package stages tests the --max-retries flag behavior for stage commands.
// This test ensures that --max-retries=0 correctly overrides config values.
// Related: internal/cli/stages/*.go
// Tags: stages, cli, max-retries, flag, override
package stages

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stageCommandsWithMaxRetries lists all stage commands that should have the --max-retries flag
var stageCommandsWithMaxRetries = []struct {
	name   string
	getCmd func() *cobra.Command
}{
	{"specify", func() *cobra.Command { return specifyCmd }},
	{"plan", func() *cobra.Command { return planCmd }},
	{"tasks", func() *cobra.Command { return tasksCmd }},
	{"implement", func() *cobra.Command { return implementCmd }},
}

func TestStages_MaxRetriesFlag_ExistsOnAllCommands(t *testing.T) {
	for _, tc := range stageCommandsWithMaxRetries {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.getCmd()
			flag := cmd.Flags().Lookup("max-retries")
			require.NotNil(t, flag, "max-retries flag should exist on %s command", tc.name)
		})
	}
}

func TestStages_MaxRetriesFlag_DescriptionNotMisleading(t *testing.T) {
	// The description should NOT contain "(0 = use config)" because that's misleading.
	// When --max-retries=0 is explicitly set, it should override config, not use config.
	misleadingText := "(0 = use config)"

	for _, tc := range stageCommandsWithMaxRetries {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.getCmd()
			flag := cmd.Flags().Lookup("max-retries")
			require.NotNil(t, flag)

			assert.NotContains(t, flag.Usage, misleadingText,
				"max-retries flag description should not contain misleading text '%s'", misleadingText)
			assert.Contains(t, flag.Usage, "overrides config when set",
				"max-retries flag description should indicate it overrides config")
		})
	}
}

func TestStages_MaxRetriesFlag_DefaultIsZero(t *testing.T) {
	for _, tc := range stageCommandsWithMaxRetries {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.getCmd()
			flag := cmd.Flags().Lookup("max-retries")
			require.NotNil(t, flag)
			assert.Equal(t, "0", flag.DefValue, "max-retries default should be 0")
		})
	}
}

func TestStages_MaxRetriesFlag_HasShorthandR(t *testing.T) {
	for _, tc := range stageCommandsWithMaxRetries {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.getCmd()
			flag := cmd.Flags().Lookup("max-retries")
			require.NotNil(t, flag)
			assert.Equal(t, "r", flag.Shorthand,
				"max-retries should have shorthand 'r' on %s command", tc.name)
		})
	}
}

func TestStages_MaxRetriesFlag_ChangedBehaviorWithZero(t *testing.T) {
	// This test verifies that when --max-retries=0 is explicitly passed,
	// cmd.Flags().Changed("max-retries") returns true.
	// This is critical for allowing 0 to override config values.

	tests := map[string]struct {
		args        []string
		wantChanged bool
		wantValue   int
	}{
		"flag not set - Changed returns false": {
			args:        []string{},
			wantChanged: false,
			wantValue:   0, // default
		},
		"flag set to 0 - Changed returns true": {
			args:        []string{"--max-retries=0"},
			wantChanged: true,
			wantValue:   0,
		},
		"flag set to 3 - Changed returns true": {
			args:        []string{"--max-retries=3"},
			wantChanged: true,
			wantValue:   3,
		},
		"short flag -r 0 - Changed returns true": {
			args:        []string{"-r", "0"},
			wantChanged: true,
			wantValue:   0,
		},
		"short flag -r=0 - Changed returns true": {
			args:        []string{"-r=0"},
			wantChanged: true,
			wantValue:   0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a fresh command for testing to avoid state issues
			testCmd := &cobra.Command{
				Use: "test",
				RunE: func(cmd *cobra.Command, args []string) error {
					return nil
				},
			}
			testCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts (overrides config when set)")

			// Parse the flags
			err := testCmd.ParseFlags(tt.args)
			require.NoError(t, err)

			// Check Changed behavior
			changed := testCmd.Flags().Changed("max-retries")
			assert.Equal(t, tt.wantChanged, changed,
				"Changed() should return %v for args %v", tt.wantChanged, tt.args)

			// Check value
			val, err := testCmd.Flags().GetInt("max-retries")
			require.NoError(t, err)
			assert.Equal(t, tt.wantValue, val)
		})
	}
}

func TestStages_MaxRetriesFlag_ConfigOverridePattern(t *testing.T) {
	// This test verifies the config override pattern works correctly.
	// The pattern is:
	//
	//   if cmd.Flags().Changed("max-retries") {
	//       cfg.MaxRetries = maxRetries
	//   }

	tests := map[string]struct {
		configValue    int
		flagArgs       []string
		expectedResult int
		description    string
	}{
		"zero overrides positive config": {
			configValue:    5,
			flagArgs:       []string{"--max-retries=0"},
			expectedResult: 0,
			description:    "User wants no retries, config has 5",
		},
		"zero overrides zero config": {
			configValue:    0,
			flagArgs:       []string{"--max-retries=0"},
			expectedResult: 0,
			description:    "Explicit zero when config is already zero",
		},
		"positive overrides positive config": {
			configValue:    5,
			flagArgs:       []string{"--max-retries=10"},
			expectedResult: 10,
			description:    "User wants 10 retries, config has 5",
		},
		"no flag preserves config": {
			configValue:    5,
			flagArgs:       []string{},
			expectedResult: 5,
			description:    "No flag passed, should use config value",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testCmd := &cobra.Command{Use: "test"}
			testCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts")

			err := testCmd.ParseFlags(tt.flagArgs)
			require.NoError(t, err)

			// Simulate handler logic
			result := tt.configValue
			if testCmd.Flags().Changed("max-retries") {
				val, _ := testCmd.Flags().GetInt("max-retries")
				result = val
			}

			assert.Equal(t, tt.expectedResult, result, tt.description)
		})
	}
}

func TestStages_MaxRetriesFlag_DescriptionFormat(t *testing.T) {
	// Verify all descriptions follow the expected format
	expectedSubstring := "overrides config when set"

	for _, tc := range stageCommandsWithMaxRetries {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.getCmd()
			flag := cmd.Flags().Lookup("max-retries")
			require.NotNil(t, flag)

			assert.True(t, strings.Contains(flag.Usage, expectedSubstring),
				"Flag description should contain '%s', got '%s'", expectedSubstring, flag.Usage)
		})
	}
}
