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

func TestInitCmdRegistration(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "init" {
			found = true
			break
		}
	}
	assert.True(t, found, "init command should be registered")
}

func TestInitCmdFlags(t *testing.T) {
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
	// --global should be hidden (deprecated)
	f := initCmd.Flags().Lookup("global")
	require.NotNil(t, f)
	assert.True(t, f.Hidden)
}

func TestCountResults(t *testing.T) {
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
	t.Run("copy success", func(t *testing.T) {
		srcDir := t.TempDir()
		dstDir := t.TempDir()

		srcPath := filepath.Join(srcDir, "constitution.md")
		dstPath := filepath.Join(dstDir, "subdir", "constitution.md")

		content := "# Test Constitution\n\nThis is a test."
		require.NoError(t, os.WriteFile(srcPath, []byte(content), 0644))

		err := copyConstitution(srcPath, dstPath)
		require.NoError(t, err)

		// Verify content
		data, err := os.ReadFile(dstPath)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("source not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := copyConstitution("/nonexistent/path", filepath.Join(tmpDir, "dest"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read")
	})
}

func TestPrintSummary(t *testing.T) {
	t.Run("with constitution", func(t *testing.T) {
		var buf bytes.Buffer
		printSummary(&buf, true)
		output := buf.String()

		assert.Contains(t, output, "Quick start")
		assert.Contains(t, output, "autospec specify")
		assert.NotContains(t, output, "IMPORTANT")
	})

	t.Run("without constitution", func(t *testing.T) {
		var buf bytes.Buffer
		printSummary(&buf, false)
		output := buf.String()

		assert.Contains(t, output, "IMPORTANT")
		assert.Contains(t, output, "autospec constitution")
		assert.Contains(t, output, "Quick start")
	})
}

func TestInitCmdExamples(t *testing.T) {
	assert.Contains(t, initCmd.Example, "autospec init")
	assert.Contains(t, initCmd.Example, "--project")
	assert.Contains(t, initCmd.Example, "--force")
}

func TestInitCmdLongDescription(t *testing.T) {
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
// NOTE: This test cannot use t.Parallel() because it uses os.Chdir() which modifies
// global state (the current working directory). Parallel subtests would race for
// the working directory.
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

	// Create command
	cmd := &cobra.Command{
		Use:  "init",
		RunE: runInit,
	}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().BoolP("global", "g", false, "")

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
// NOTE: This test cannot use t.Parallel() because it uses os.Chdir() which modifies
// global state (the current working directory). Parallel subtests would race for
// the working directory.
func TestRunInit_ProjectConfig(t *testing.T) {
	// Setup temp directory
	tmpDir := t.TempDir()

	// Change to temp directory
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(tmpDir)

	// Set XDG to avoid touching user's actual config
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	// Create command with --project flag
	cmd := &cobra.Command{
		Use:  "init",
		RunE: runInit,
	}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().BoolP("global", "g", false, "")

	require.NoError(t, cmd.Flags().Set("project", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	// Verify project config was created
	projectConfig := filepath.Join(tmpDir, ".autospec", "config.yml")
	assert.FileExists(t, projectConfig)
}

// TestRunInit_ForceOverwrite tests force overwrite behavior.
// NOTE: This test cannot use t.Parallel() because it uses os.Chdir() which modifies
// global state (the current working directory). Parallel subtests would race for
// the working directory.
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

	// Create command with --project and --force flags
	cmd := &cobra.Command{
		Use:  "init",
		RunE: runInit,
	}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().BoolP("global", "g", false, "")

	require.NoError(t, cmd.Flags().Set("project", "true"))
	require.NoError(t, cmd.Flags().Set("force", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	// Verify config was overwritten
	data, err := os.ReadFile(existingConfig)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "max_retries: 99") // Should be default now
}

// TestHandleConstitution tests constitution handling.
// NOTE: This test cannot use t.Parallel() because it uses os.Chdir() which modifies
// global state (the current working directory). Parallel subtests would race for
// the working directory.
func TestHandleConstitution(t *testing.T) {
	t.Run("no constitution found", func(t *testing.T) {
		// Use temp directory with no constitution files
		tmpDir := t.TempDir()
		origWd, _ := os.Getwd()
		defer os.Chdir(origWd)
		os.Chdir(tmpDir)

		var buf bytes.Buffer
		result := handleConstitution(&buf)

		assert.False(t, result)
		assert.Contains(t, buf.String(), "not found")
	})
}

// TestCheckGitignore tests gitignore checking.
// NOTE: This test cannot use t.Parallel() because it uses os.Chdir() which modifies
// global state (the current working directory). Parallel subtests would race for
// the working directory.
func TestCheckGitignore(t *testing.T) {
	t.Run("no gitignore file", func(t *testing.T) {
		tmpDir := t.TempDir()
		origWd, _ := os.Getwd()
		defer os.Chdir(origWd)
		os.Chdir(tmpDir)

		var buf bytes.Buffer
		checkGitignore(&buf)

		// Should not print anything if .gitignore doesn't exist
		assert.Empty(t, buf.String())
	})

	t.Run("gitignore without autospec", func(t *testing.T) {
		tmpDir := t.TempDir()
		origWd, _ := os.Getwd()
		defer os.Chdir(origWd)
		os.Chdir(tmpDir)

		// Create .gitignore without .autospec
		require.NoError(t, os.WriteFile(".gitignore", []byte("node_modules/\n*.log\n"), 0644))

		var buf bytes.Buffer
		checkGitignore(&buf)

		// Should print recommendation
		assert.Contains(t, buf.String(), "Recommendation")
		assert.Contains(t, buf.String(), ".autospec/")
	})

	t.Run("gitignore with .autospec", func(t *testing.T) {
		tmpDir := t.TempDir()
		origWd, _ := os.Getwd()
		defer os.Chdir(origWd)
		os.Chdir(tmpDir)

		// Create .gitignore with .autospec
		require.NoError(t, os.WriteFile(".gitignore", []byte("node_modules/\n.autospec/\n*.log\n"), 0644))

		var buf bytes.Buffer
		checkGitignore(&buf)

		// Should not print anything
		assert.Empty(t, buf.String())
	})

	t.Run("gitignore with .autospec no trailing slash", func(t *testing.T) {
		tmpDir := t.TempDir()
		origWd, _ := os.Getwd()
		defer os.Chdir(origWd)
		os.Chdir(tmpDir)

		// Create .gitignore with .autospec (no trailing slash)
		require.NoError(t, os.WriteFile(".gitignore", []byte(".autospec\n"), 0644))

		var buf bytes.Buffer
		checkGitignore(&buf)

		// Should not print anything
		assert.Empty(t, buf.String())
	})

	t.Run("gitignore with .autospec subdirectory pattern", func(t *testing.T) {
		tmpDir := t.TempDir()
		origWd, _ := os.Getwd()
		defer os.Chdir(origWd)
		os.Chdir(tmpDir)

		// Create .gitignore with .autospec subdirectory pattern
		require.NoError(t, os.WriteFile(".gitignore", []byte(".autospec/state/\n"), 0644))

		var buf bytes.Buffer
		checkGitignore(&buf)

		// Should not print anything - subdirectory pattern counts
		assert.Empty(t, buf.String())
	})
}

func TestConfigureClaudeSettings(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup             func(t *testing.T, dir string)
		wantMsgContains   []string
		wantMsgNotContain []string
		wantFileExists    bool
		wantInAllowList   bool
	}{
		"creates settings when missing": {
			setup: func(t *testing.T, dir string) {
				// No setup - no .claude directory
			},
			wantMsgContains: []string{"Claude settings:", "created", "permissions"},
			wantFileExists:  true,
			wantInAllowList: true,
		},
		"adds permission to existing settings": {
			setup: func(t *testing.T, dir string) {
				createClaudeSettingsFile(t, dir, `{"permissions": {"allow": ["Bash(other:*)"]}}`)
			},
			wantMsgContains: []string{"Claude settings:", "added", "Bash(autospec:*)"},
			wantFileExists:  true,
			wantInAllowList: true,
		},
		"skips when permission already present": {
			setup: func(t *testing.T, dir string) {
				createClaudeSettingsFile(t, dir, `{"permissions": {"allow": ["Bash(autospec:*)"]}}`)
			},
			wantMsgContains: []string{"Claude settings:", "already configured"},
			wantFileExists:  true,
			wantInAllowList: true,
		},
		"warns when permission is denied": {
			setup: func(t *testing.T, dir string) {
				createClaudeSettingsFile(t, dir, `{"permissions": {"deny": ["Bash(autospec:*)"]}}`)
			},
			wantMsgContains:   []string{"Warning", "Bash(autospec:*)", "deny list"},
			wantMsgNotContain: []string{"created", "added"},
			wantFileExists:    true,
			wantInAllowList:   false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tt.setup != nil {
				tt.setup(t, dir)
			}

			var buf bytes.Buffer
			configureClaudeSettings(&buf, dir)

			output := buf.String()
			for _, want := range tt.wantMsgContains {
				assert.Contains(t, output, want, "output should contain %q", want)
			}
			for _, notWant := range tt.wantMsgNotContain {
				assert.NotContains(t, output, notWant, "output should not contain %q", notWant)
			}

			settingsPath := filepath.Join(dir, ".claude", "settings.local.json")
			if tt.wantFileExists {
				assert.FileExists(t, settingsPath)

				// Check if permission is in the allow list (use claude package)
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

	var buf bytes.Buffer
	configureClaudeSettings(&buf, dir)

	// Verify all existing fields are preserved
	settingsPath := filepath.Join(dir, ".claude", "settings.local.json")
	data, err := os.ReadFile(settingsPath)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "Bash(existing:*)")
	assert.Contains(t, content, "Bash(autospec:*)")
	assert.Contains(t, content, "Write(*)")
	assert.Contains(t, content, "Bash(rm:*)")
	assert.Contains(t, content, "sandbox")
}

func TestConfigureClaudeSettings_MalformedJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	createClaudeSettingsFile(t, dir, `{invalid json}`)

	var buf bytes.Buffer
	configureClaudeSettings(&buf, dir)

	output := buf.String()
	assert.Contains(t, output, "Claude settings:")
	assert.Contains(t, output, "parsing")
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
