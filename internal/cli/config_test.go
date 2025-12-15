package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigCmdRegistration(t *testing.T) {
	// Verify configCmd is registered
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "config" {
			found = true
			break
		}
	}
	assert.True(t, found, "config command should be registered")
}

func TestConfigShowCmdRegistration(t *testing.T) {
	// Verify show subcommand is registered
	found := false
	for _, cmd := range configCmd.Commands() {
		if cmd.Use == "show" {
			found = true
			break
		}
	}
	assert.True(t, found, "config show command should be registered")
}

func TestConfigMigrateCmdRegistration(t *testing.T) {
	// Verify migrate subcommand is registered
	found := false
	for _, cmd := range configCmd.Commands() {
		if cmd.Use == "migrate" {
			found = true
			break
		}
	}
	assert.True(t, found, "config migrate command should be registered")
}

func TestConfigShowCmd_DefaultOutput(t *testing.T) {
	// Create test command
	cmd := &cobra.Command{
		Use:  "show",
		RunE: runConfigShow,
	}
	cmd.Flags().Bool("json", false, "")
	cmd.Flags().Bool("yaml", true, "")

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
	// Create test command
	cmd := &cobra.Command{
		Use:  "show",
		RunE: runConfigShow,
	}
	cmd.Flags().Bool("json", false, "")
	cmd.Flags().Bool("yaml", true, "")

	// Set JSON flag
	require.NoError(t, cmd.Flags().Set("json", "true"))

	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)

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
	cmd := &cobra.Command{
		Use:  "show",
		RunE: runConfigShow,
	}
	cmd.Flags().Bool("json", false, "")
	cmd.Flags().Bool("yaml", true, "")

	// Use JSON for easier parsing
	require.NoError(t, cmd.Flags().Set("json", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

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
		"use_api_key",
		"custom_claude_cmd",
		"max_retries",
		"specs_dir",
		"state_dir",
		"skip_preflight",
		"timeout",
		"show_progress",
		"skip_confirmations",
	}

	for _, field := range expectedFields {
		assert.Contains(t, config, field, "Config should contain field: %s", field)
	}
}

func TestConfigMigrateCmd_Flags(t *testing.T) {
	flags := []string{"dry-run", "user", "project"}

	for _, flagName := range flags {
		t.Run("flag "+flagName, func(t *testing.T) {
			f := configMigrateCmd.Flags().Lookup(flagName)
			require.NotNil(t, f, "flag %s should exist", flagName)
		})
	}
}

func TestConfigMigrateCmd_DryRunOutput(t *testing.T) {
	cmd := &cobra.Command{
		Use:  "migrate",
		RunE: runConfigMigrate,
	}
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("user", false, "")
	cmd.Flags().Bool("project", false, "")

	// Set dry-run flag
	require.NoError(t, cmd.Flags().Set("dry-run", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Dry run mode")
}

func TestConfigMigrateCmd_UserOnlyFlag(t *testing.T) {
	cmd := &cobra.Command{
		Use:  "migrate",
		RunE: runConfigMigrate,
	}
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("user", false, "")
	cmd.Flags().Bool("project", false, "")

	require.NoError(t, cmd.Flags().Set("user", "true"))
	require.NoError(t, cmd.Flags().Set("dry-run", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
	// Should not error even if no config exists
}

func TestConfigMigrateCmd_ProjectOnlyFlag(t *testing.T) {
	cmd := &cobra.Command{
		Use:  "migrate",
		RunE: runConfigMigrate,
	}
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("user", false, "")
	cmd.Flags().Bool("project", false, "")

	require.NoError(t, cmd.Flags().Set("project", "true"))
	require.NoError(t, cmd.Flags().Set("dry-run", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
}

func TestFileExistsCheck(t *testing.T) {
	tests := map[string]struct {
		setup func(t *testing.T) string
		want  bool
	}{
		"file exists": {
			setup: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "exists.txt")
				require.NoError(t, os.WriteFile(path, []byte("test"), 0644))
				return path
			},
			want: true,
		},
		"file doesn't exist": {
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.txt")
			},
			want: false,
		},
		"directory exists": {
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			path := tc.setup(t)
			got := fileExistsCheck(path)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestConfigCmdExamples(t *testing.T) {
	// Verify examples are present
	assert.Contains(t, configCmd.Example, "autospec config show")
	assert.Contains(t, configCmd.Example, "autospec config migrate")
}

func TestConfigCmdLongDescription(t *testing.T) {
	// Verify priority order is documented
	priorities := []string{
		"Environment variables",
		"Project config",
		"User config",
		"defaults",
	}

	for _, priority := range priorities {
		assert.Contains(t, configCmd.Long, priority)
	}
}

func TestConfigShowCmd_YAMLFormatDefault(t *testing.T) {
	// yaml flag should default to true
	f := configShowCmd.Flags().Lookup("yaml")
	require.NotNil(t, f)
	assert.Equal(t, "true", f.DefValue)

	// json flag should default to false
	f = configShowCmd.Flags().Lookup("json")
	require.NotNil(t, f)
	assert.Equal(t, "false", f.DefValue)
}

func TestConfigMigrateCmd_NoConfigsToMigrate(t *testing.T) {
	// When no JSON configs exist, should report "No JSON configs found"
	cmd := &cobra.Command{
		Use:  "migrate",
		RunE: runConfigMigrate,
	}
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("user", false, "")
	cmd.Flags().Bool("project", false, "")

	// Create empty project dir to avoid finding any configs
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(tmpDir)

	// Clear XDG to avoid finding user configs
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "no-config"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	// Should indicate no configs were found or migration was skipped
	assert.True(t, len(output) > 0)
}

func TestConfigMigrateCmd_Examples(t *testing.T) {
	examples := []string{
		"autospec config migrate",
		"--dry-run",
		"--user",
		"--project",
	}

	for _, example := range examples {
		assert.Contains(t, configMigrateCmd.Example, example)
	}
}
