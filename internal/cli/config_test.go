// Package cli_test tests the config command for displaying configuration settings.
// Related: internal/cli/config/config_cmd.go
// Tags: cli, config, configuration, settings, yaml, json
package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getConfigCmd finds the config command from rootCmd
func getConfigCmd() *cobra.Command {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "config" {
			return cmd
		}
	}
	return nil
}

// getConfigShowCmd finds the "config show" subcommand
func getConfigShowCmd() *cobra.Command {
	configCmd := getConfigCmd()
	if configCmd == nil {
		return nil
	}
	for _, cmd := range configCmd.Commands() {
		if cmd.Use == "show" {
			return cmd
		}
	}
	return nil
}

func TestConfigCmdRegistration(t *testing.T) {
	cmd := getConfigCmd()
	assert.NotNil(t, cmd, "config command should be registered")
}

func TestConfigShowCmdRegistration(t *testing.T) {
	cmd := getConfigShowCmd()
	assert.NotNil(t, cmd, "config show command should be registered")
}

func TestConfigShowCmd_DefaultOutput(t *testing.T) {
	cmd := getConfigShowCmd()
	require.NotNil(t, cmd, "config show command must exist")

	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()

	// Should contain config sources header
	assert.Contains(t, output, "Configuration Sources")

	// Should contain key config fields in YAML format (default)
	assert.Contains(t, output, "claude_cmd:")
	assert.Contains(t, output, "max_retries:")
	assert.Contains(t, output, "specs_dir:")
}

func TestConfigShowCmd_JSONOutput(t *testing.T) {
	cmd := getConfigShowCmd()
	require.NotNil(t, cmd, "config show command must exist")

	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Set flag directly
	cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()

	// Should contain config sources header
	assert.Contains(t, output, "Configuration Sources")

	// Extract JSON portion (after the header comments)
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	var jsonLines []byte
	for _, line := range lines {
		if len(line) > 0 && line[0] != '#' {
			jsonLines = append(jsonLines, line...)
			jsonLines = append(jsonLines, '\n')
		}
	}

	// Should be valid JSON
	var config map[string]interface{}
	err = json.Unmarshal(jsonLines, &config)
	require.NoError(t, err, "Output should contain valid JSON")

	// Verify expected fields
	assert.Contains(t, config, "claude_cmd")
	assert.Contains(t, config, "max_retries")
	assert.Contains(t, config, "specs_dir")
	assert.Contains(t, config, "state_dir")
}

func TestConfigShowCmd_AllFields(t *testing.T) {
	cmd := getConfigShowCmd()
	require.NotNil(t, cmd, "config show command must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// Set flag directly
	cmd.Flags().Set("json", "true")

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	// Extract and parse JSON
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	var jsonLines []byte
	for _, line := range lines {
		if len(line) > 0 && line[0] != '#' {
			jsonLines = append(jsonLines, line...)
			jsonLines = append(jsonLines, '\n')
		}
	}

	var config map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonLines, &config))

	// All expected fields should be present
	expectedFields := []string{
		"claude_cmd",
		"claude_args",
		"custom_claude_cmd",
		"max_retries",
		"specs_dir",
		"state_dir",
		"skip_preflight",
		"timeout",
		"skip_confirmations",
	}

	for _, field := range expectedFields {
		assert.Contains(t, config, field, "Config should contain field: %s", field)
	}
}

func TestConfigCmdExamples(t *testing.T) {
	cmd := getConfigCmd()
	require.NotNil(t, cmd, "config command must exist")

	// Verify examples are present
	assert.Contains(t, cmd.Example, "autospec config show")
}

func TestConfigCmdLongDescription(t *testing.T) {
	cmd := getConfigCmd()
	require.NotNil(t, cmd, "config command must exist")

	// Verify priority order is documented
	priorities := []string{
		"Environment variables",
		"Project config",
		"User config",
		"defaults",
	}

	for _, priority := range priorities {
		assert.Contains(t, cmd.Long, priority)
	}
}

func TestConfigShowCmd_YAMLFormatDefault(t *testing.T) {
	cmd := getConfigShowCmd()
	require.NotNil(t, cmd, "config show command must exist")

	// yaml flag should default to true
	f := cmd.Flags().Lookup("yaml")
	require.NotNil(t, f)
	assert.Equal(t, "true", f.DefValue)

	// json flag should default to false
	f = cmd.Flags().Lookup("json")
	require.NotNil(t, f)
	assert.Equal(t, "false", f.DefValue)
}
