// Package config tests CLI configuration commands for autospec.
// Related: internal/cli/config/config_cmd.go
// Tags: config, cli, show, migrate

package config

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRunConfigShow_YAMLOutput(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
			t.Parallel()

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

func TestRunConfigMigrate_DryRun(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{
		Use:  "migrate",
		RunE: runConfigMigrate,
	}
	cmd.Flags().Bool("dry-run", true, "")
	cmd.Flags().Bool("user", false, "")
	cmd.Flags().Bool("project", false, "")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Dry run mode")
}

func TestRunConfigMigrate_UserOnly(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{
		Use:  "migrate",
		RunE: runConfigMigrate,
	}
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("user", true, "")
	cmd.Flags().Bool("project", false, "")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	// Should succeed even with no files to migrate
	assert.NoError(t, err)
}

func TestRunConfigMigrate_ProjectOnly(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{
		Use:  "migrate",
		RunE: runConfigMigrate,
	}
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("user", false, "")
	cmd.Flags().Bool("project", true, "")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	// Should succeed even with no files to migrate
	assert.NoError(t, err)
}

func TestPrintMigrationSummary_Migrated(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		migrated int
		skipped  int
		dryRun   bool
		wantMsg  string
	}{
		"dry run with migration": {
			migrated: 1,
			skipped:  0,
			dryRun:   true,
			wantMsg:  "Would migrate",
		},
		"actual migration": {
			migrated: 1,
			skipped:  0,
			dryRun:   false,
			wantMsg:  "Migrated 1",
		},
		"no migration needed": {
			migrated: 0,
			skipped:  1,
			dryRun:   false,
			wantMsg:  "No JSON configs found",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			printMigrationSummary(&buf, tt.migrated, tt.skipped, tt.dryRun)

			output := buf.String()
			assert.Contains(t, output, tt.wantMsg)
		})
	}
}

func TestConfigCmd_SubcommandExecution(t *testing.T) {
	t.Parallel()

	// Verify that config command has subcommands properly set up
	subcommands := configCmd.Commands()

	// Should have show and migrate subcommands
	found := make(map[string]bool)
	for _, cmd := range subcommands {
		found[cmd.Name()] = true
	}

	assert.True(t, found["show"], "Should have show subcommand")
	assert.True(t, found["migrate"], "Should have migrate subcommand")
}

func TestConfigShowCmd_HasRunE(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, configShowCmd.RunE)
}

func TestConfigMigrateCmd_HasRunE(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, configMigrateCmd.RunE)
}
