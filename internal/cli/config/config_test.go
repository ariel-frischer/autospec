// Package config tests CLI configuration commands for autospec.
// Related: internal/cli/config/config_cmd.go
// Tags: config, cli, show

package config

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRunConfigShow_YAMLOutput(t *testing.T) {

	// Create isolated command
	cmd := &cobra.Command{
		Use:  "show",
		RunE: runConfigShow,
	}
	cmd.Flags().Bool("json", false, "Output in JSON format")
	cmd.Flags().Bool("yaml", true, "Output in YAML format")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Configuration Sources")
	// YAML output should have key: value format
	assert.Contains(t, output, "claude_cmd:")
}

func TestRunConfigShow_JSONOutput(t *testing.T) {

	// Create isolated command
	cmd := &cobra.Command{
		Use:  "show",
		RunE: runConfigShow,
	}
	cmd.Flags().Bool("json", false, "Output in JSON format")
	cmd.Flags().Bool("yaml", true, "Output in YAML format")
	_ = cmd.Flags().Set("json", "true")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Configuration Sources")
	// JSON output should have braces
	assert.Contains(t, output, "{")
	assert.Contains(t, output, "}")
}

func TestConfigShowCmd_OutputFormats(t *testing.T) {

	tests := map[string]struct {
		jsonFlag bool
		wantYAML bool
	}{
		"yaml output by default": {
			jsonFlag: false,
			wantYAML: true,
		},
		"json output when flag set": {
			jsonFlag: true,
			wantYAML: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

			// Create a fresh command for each test
			cmd := &cobra.Command{
				Use:  "show",
				RunE: runConfigShow,
			}
			cmd.Flags().Bool("json", false, "Output in JSON format")
			cmd.Flags().Bool("yaml", true, "Output in YAML format")

			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			if tt.jsonFlag {
				_ = cmd.Flags().Set("json", "true")
			}

			err := cmd.Execute()
			assert.NoError(t, err)

			output := buf.String()
			if tt.wantYAML {
				assert.Contains(t, output, "claude_cmd:")
			} else {
				assert.Contains(t, output, "{")
			}
		})
	}
}

func TestConfigCmd_SubcommandExecution(t *testing.T) {

	// Verify that config command has subcommands properly set up
	subcommands := configCmd.Commands()

	// Should have show subcommand
	found := make(map[string]bool)
	for _, cmd := range subcommands {
		found[cmd.Name()] = true
	}

	assert.True(t, found["show"], "Should have show subcommand")
}

func TestConfigShowCmd_HasRunE(t *testing.T) {

	assert.NotNil(t, configShowCmd.RunE)
}
