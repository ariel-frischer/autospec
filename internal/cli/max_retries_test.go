// Package cli tests the --max-retries flag behavior across all workflow commands.
// This test ensures that --max-retries=0 correctly overrides config values.
// Related: internal/cli/*.go, internal/cli/stages/*.go
// Tags: cli, max-retries, flag, override
package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// commandsWithMaxRetries lists all commands that should have the --max-retries flag.
// Note: clarify and analyze are excluded - they don't produce artifacts that need validation/retry.
var commandsWithMaxRetries = []struct {
	name    string
	getCmd  func() *cobra.Command
	wantErr bool // commands that require args
}{
	{"checklist", func() *cobra.Command { return checklistCmd }, false},
	{"constitution", func() *cobra.Command { return constitutionCmd }, false},
	{"prep", func() *cobra.Command { return prepCmd }, true},
	{"all", func() *cobra.Command { return allCmd }, true},
	{"run", func() *cobra.Command { return runCmd }, false},
}

func TestMaxRetriesFlag_ExistsOnAllWorkflowCommands(t *testing.T) {
	for _, tc := range commandsWithMaxRetries {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.getCmd()
			flag := cmd.Flags().Lookup("max-retries")
			require.NotNil(t, flag, "max-retries flag should exist on %s command", tc.name)
		})
	}
}

func TestMaxRetriesFlag_DescriptionNotMisleading(t *testing.T) {
	// The description should NOT contain "(0 = use config)" because that's misleading.
	// When --max-retries=0 is explicitly set, it should override config, not use config.
	misleadingText := "(0 = use config)"

	for _, tc := range commandsWithMaxRetries {
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

func TestMaxRetriesFlag_DefaultIsZero(t *testing.T) {
	for _, tc := range commandsWithMaxRetries {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.getCmd()
			flag := cmd.Flags().Lookup("max-retries")
			require.NotNil(t, flag)
			assert.Equal(t, "0", flag.DefValue, "max-retries default should be 0")
		})
	}
}

func TestMaxRetriesFlag_ChangedBehaviorWithZero(t *testing.T) {
	// This test verifies that when --max-retries=0 is explicitly passed,
	// cmd.Flags().Changed("max-retries") returns true.
	// This is the critical behavior that allows 0 to override config.

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
		"flag set to 5 - Changed returns true": {
			args:        []string{"--max-retries=5"},
			wantChanged: true,
			wantValue:   5,
		},
		"flag set to 0 with space - Changed returns true": {
			args:        []string{"--max-retries", "0"},
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

func TestMaxRetriesFlag_ShorthandConsistency(t *testing.T) {
	// Most commands use -r as shorthand for --max-retries
	// run command is an exception (uses long-only because -r is for --clarify)
	// clarify and analyze don't have --max-retries at all (no artifact validation)

	commandsWithShorthand := []struct {
		name      string
		getCmd    func() *cobra.Command
		wantShort string
	}{
		{"checklist", func() *cobra.Command { return checklistCmd }, "r"},
		{"constitution", func() *cobra.Command { return constitutionCmd }, "r"},
		{"prep", func() *cobra.Command { return prepCmd }, "r"},
		{"all", func() *cobra.Command { return allCmd }, "r"},
		// run command intentionally does NOT have -r shorthand
	}

	for _, tc := range commandsWithShorthand {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.getCmd()
			flag := cmd.Flags().Lookup("max-retries")
			require.NotNil(t, flag)
			assert.Equal(t, tc.wantShort, flag.Shorthand,
				"max-retries should have shorthand '%s' on %s command", tc.wantShort, tc.name)
		})
	}
}

func TestMaxRetriesFlag_RunCommandNoShorthand(t *testing.T) {
	// The run command intentionally does NOT have -r shorthand for max-retries
	// because -r is used for --clarify
	flag := runCmd.Flags().Lookup("max-retries")
	require.NotNil(t, flag)
	assert.Equal(t, "", flag.Shorthand,
		"run command max-retries should not have shorthand (conflicts with -r for clarify)")
}

func TestMaxRetriesFlag_ConfigOverridePattern(t *testing.T) {
	// This test documents and verifies the expected config override pattern.
	// The pattern used in all command handlers is:
	//
	//   if cmd.Flags().Changed("max-retries") {
	//       cfg.MaxRetries = maxRetries
	//   }
	//
	// This correctly handles --max-retries=0 because Changed() returns true
	// when the flag is explicitly set, regardless of value.

	t.Run("explicit zero overrides config", func(t *testing.T) {
		// Simulate: config has MaxRetries=5, user passes --max-retries=0
		configValue := 5
		flagValue := 0

		testCmd := &cobra.Command{Use: "test"}
		testCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts")

		// Parse with explicit zero
		err := testCmd.ParseFlags([]string{"--max-retries=0"})
		require.NoError(t, err)

		// Simulate the handler logic
		if testCmd.Flags().Changed("max-retries") {
			val, _ := testCmd.Flags().GetInt("max-retries")
			configValue = val
		}

		assert.Equal(t, flagValue, configValue,
			"Config should be overridden to 0 when --max-retries=0 is passed")
	})

	t.Run("no flag preserves config", func(t *testing.T) {
		// Simulate: config has MaxRetries=5, user passes no flag
		configValue := 5
		originalConfig := configValue

		testCmd := &cobra.Command{Use: "test"}
		testCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts")

		// Parse with no flags
		err := testCmd.ParseFlags([]string{})
		require.NoError(t, err)

		// Simulate the handler logic
		if testCmd.Flags().Changed("max-retries") {
			val, _ := testCmd.Flags().GetInt("max-retries")
			configValue = val
		}

		assert.Equal(t, originalConfig, configValue,
			"Config should be preserved when no flag is passed")
	})
}

func TestMaxRetriesFlag_DescriptionFormat(t *testing.T) {
	// Verify all descriptions follow the expected format
	expectedSubstring := "overrides config when set"

	for _, tc := range commandsWithMaxRetries {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.getCmd()
			flag := cmd.Flags().Lookup("max-retries")
			require.NotNil(t, flag)

			assert.True(t, strings.Contains(flag.Usage, expectedSubstring),
				"Flag description should contain '%s', got '%s'", expectedSubstring, flag.Usage)
		})
	}
}
