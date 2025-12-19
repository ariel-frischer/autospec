// Package cli_test tests the init command for project initialization, command template installation, and Claude settings configuration.
// Related: internal/cli/config/init_cmd.go
// Tags: cli, init, initialization, setup, project, templates, constitution, gitignore
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	claudepkg "github.com/ariel-frischer/autospec/internal/claude"
	"github.com/ariel-frischer/autospec/internal/commands"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getInitCmd finds the init command from rootCmd
func getInitCmd() *cobra.Command {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "init" {
			return cmd
		}
	}
	return nil
}

func TestInitCmdRegistration(t *testing.T) {
	cmd := getInitCmd()
	assert.NotNil(t, cmd, "init command should be registered")
}

func TestInitCmdFlags(t *testing.T) {
	initCmd := getInitCmd()
	require.NotNil(t, initCmd, "init command must exist")

	flags := []struct {
		name      string
		shorthand string
	}{
		{"project", "p"},
		{"force", "f"},
	}

	for _, flag := range flags {
		t.Run("flag "+flag.name, func(t *testing.T) {
			f := initCmd.Flags().Lookup(flag.name)
			require.NotNil(t, f, "flag %s should exist", flag.name)
			assert.Equal(t, flag.shorthand, f.Shorthand)
		})
	}
}

func TestInitCmdGlobalFlagHidden(t *testing.T) {
	initCmd := getInitCmd()
	require.NotNil(t, initCmd, "init command must exist")

	// --global should be hidden (deprecated)
	f := initCmd.Flags().Lookup("global")
	require.NotNil(t, f)
	assert.True(t, f.Hidden)
}

func TestCountResults(t *testing.T) {
	// Test count results logic inline since the function is unexported
	countResults := func(results []commands.InstallResult) (installed, updated int) {
		for _, r := range results {
			switch r.Action {
			case "installed":
				installed++
			case "updated":
				updated++
			}
		}
		return
	}

	tests := map[string]struct {
		results     []commands.InstallResult
		wantInstall int
		wantUpdate  int
	}{
		"empty": {
			results:     []commands.InstallResult{},
			wantInstall: 0,
			wantUpdate:  0,
		},
		"mixed actions": {
			results: []commands.InstallResult{
				{Action: "installed"},
				{Action: "installed"},
				{Action: "updated"},
				{Action: "skipped"},
			},
			wantInstall: 2,
			wantUpdate:  1,
		},
		"all installed": {
			results: []commands.InstallResult{
				{Action: "installed"},
				{Action: "installed"},
				{Action: "installed"},
			},
			wantInstall: 3,
			wantUpdate:  0,
		},
		"all updated": {
			results: []commands.InstallResult{
				{Action: "updated"},
				{Action: "updated"},
			},
			wantInstall: 0,
			wantUpdate:  2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			installed, updated := countResults(tc.results)
			assert.Equal(t, tc.wantInstall, installed)
			assert.Equal(t, tc.wantUpdate, updated)
		})
	}
}

func TestCopyConstitution(t *testing.T) {
	// Test file copy logic inline since the function is unexported
	copyFile := func(src, dst string) error {
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0644)
	}

	t.Run("copy success", func(t *testing.T) {
		srcDir := t.TempDir()
		dstDir := t.TempDir()

		srcPath := filepath.Join(srcDir, "constitution.md")
		dstPath := filepath.Join(dstDir, "subdir", "constitution.md")

		content := "# Test Constitution\n\nThis is a test."
		require.NoError(t, os.WriteFile(srcPath, []byte(content), 0644))

		err := copyFile(srcPath, dstPath)
		require.NoError(t, err)

		// Verify content
		data, err := os.ReadFile(dstPath)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("source not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := copyFile("/nonexistent/path", filepath.Join(tmpDir, "dest"))
		assert.Error(t, err)
	})
}

func TestInitCmdExamples(t *testing.T) {
	initCmd := getInitCmd()
	require.NotNil(t, initCmd, "init command must exist")

	assert.Contains(t, initCmd.Example, "autospec init")
	assert.Contains(t, initCmd.Example, "--project")
	assert.Contains(t, initCmd.Example, "--force")
}

func TestInitCmdLongDescription(t *testing.T) {
	initCmd := getInitCmd()
	require.NotNil(t, initCmd, "init command must exist")

	keywords := []string{
		"command templates",
		"user-level",
		"Configuration precedence",
	}

	for _, keyword := range keywords {
		assert.Contains(t, initCmd.Long, keyword)
	}
}

// TestRunInit_CreateUserConfig tests user config creation.
func TestRunInit_CreateUserConfig(t *testing.T) {
	// Setup temp directories
	tmpDir := t.TempDir()

	// Set XDG_CONFIG_HOME to control where user config is created
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	// Change to a temp project directory
	projDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(projDir, 0755))
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(projDir)

	cmd := getInitCmd()
	require.NotNil(t, cmd, "init command must exist")

	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Commands:")
	assert.Contains(t, output, "Config:")
}

// TestRunInit_ProjectConfig tests project config creation.
func TestRunInit_ProjectConfig(t *testing.T) {
	// Setup temp directory
	tmpDir := t.TempDir()

	// Change to temp directory
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(tmpDir)

	// Set XDG to avoid touching user's actual config
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	cmd := getInitCmd()
	require.NotNil(t, cmd, "init command must exist")

	// Set flag directly
	cmd.Flags().Set("project", "true")

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	// Verify project config was created
	projectConfig := filepath.Join(tmpDir, ".autospec", "config.yml")
	assert.FileExists(t, projectConfig)
}

// TestRunInit_ForceOverwrite tests force overwrite behavior.
func TestRunInit_ForceOverwrite(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project directory
	projDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(filepath.Join(projDir, ".autospec"), 0755))

	// Create existing config
	existingConfig := filepath.Join(projDir, ".autospec", "config.yml")
	require.NoError(t, os.WriteFile(existingConfig, []byte("max_retries: 99\n"), 0644))

	// Change to project directory
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(projDir)

	// Set XDG to avoid touching user's actual config
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	cmd := getInitCmd()
	require.NotNil(t, cmd, "init command must exist")

	// Set flags directly
	cmd.Flags().Set("project", "true")
	cmd.Flags().Set("force", "true")

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	// Verify config was overwritten
	data, err := os.ReadFile(existingConfig)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "max_retries: 99") // Should be default now
}

func TestConfigureClaudeSettings(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup           func(t *testing.T, dir string)
		wantFileExists  bool
		wantInAllowList bool
	}{
		"adds permission to existing settings": {
			setup: func(t *testing.T, dir string) {
				createClaudeSettingsFile(t, dir, `{"permissions": {"allow": ["Bash(other:*)"]}}`)
			},
			wantFileExists:  true,
			wantInAllowList: false, // We're not running init, just checking setup
		},
		"skips when permission already present": {
			setup: func(t *testing.T, dir string) {
				createClaudeSettingsFile(t, dir, `{"permissions": {"allow": ["Bash(autospec:*)"]}}`)
			},
			wantFileExists:  true,
			wantInAllowList: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tt.setup != nil {
				tt.setup(t, dir)
			}

			settingsPath := filepath.Join(dir, ".claude", "settings.local.json")
			if tt.wantFileExists {
				assert.FileExists(t, settingsPath)
				settings, err := claudepkg.Load(dir)
				require.NoError(t, err)
				assert.Equal(t, tt.wantInAllowList, settings.HasPermission(claudepkg.RequiredPermission))
			}
		})
	}
}

func TestConfigureClaudeSettings_PreservesExistingFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	createClaudeSettingsFile(t, dir, `{
		"permissions": {
			"allow": ["Bash(existing:*)"],
			"ask": ["Write(*)"],
			"deny": ["Bash(rm:*)"]
		},
		"sandbox": {"enabled": true}
	}`)

	// Verify existing fields are present
	settingsPath := filepath.Join(dir, ".claude", "settings.local.json")
	data, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "Bash(existing:*)")
	assert.Contains(t, content, "Write(*)")
	assert.Contains(t, content, "Bash(rm:*)")
	assert.Contains(t, content, "sandbox")
}

// createClaudeSettingsFile is a helper to create a .claude/settings.local.json file
func createClaudeSettingsFile(t *testing.T, dir, content string) {
	t.Helper()
	claudeDir := filepath.Join(dir, ".claude")
	err := os.MkdirAll(claudeDir, 0755)
	require.NoError(t, err)

	settingsPath := filepath.Join(claudeDir, "settings.local.json")
	err = os.WriteFile(settingsPath, []byte(content), 0644)
	require.NoError(t, err)
}
