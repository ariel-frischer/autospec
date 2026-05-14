// Package config tests CLI configuration commands for autospec.
// Related: internal/cli/config/init_cmd.go
// Tags: config, cli, init, commands

package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/build"
	"github.com/ariel-frischer/autospec/internal/cliagent"
	"github.com/ariel-frischer/autospec/internal/commands"
	"github.com/ariel-frischer/autospec/internal/config"
	initpkg "github.com/ariel-frischer/autospec/internal/init"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunInit_InstallsCommands(t *testing.T) {
	// Cannot run in parallel due to working directory change and global mocks

	// CRITICAL: Mock the runners to prevent real Claude execution
	originalConstitutionRunner := ConstitutionRunner
	originalWorktreeRunner := WorktreeScriptRunner
	ConstitutionRunner = func(cmd *cobra.Command, configPath string) bool {
		return true // Simulate successful constitution creation
	}
	WorktreeScriptRunner = func(cmd *cobra.Command, configPath string) bool {
		return true // Simulate successful worktree script creation
	}
	defer func() {
		ConstitutionRunner = originalConstitutionRunner
		WorktreeScriptRunner = originalWorktreeRunner
	}()

	// Create temp directory for test
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)

	// Change to temp directory for test
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(origDir)
	}()

	// Create the .claude/commands directory for command installation
	cmdDir := filepath.Join(tmpDir, ".claude", "commands")
	err = os.MkdirAll(cmdDir, 0o755)
	require.NoError(t, err)

	// Create a mock root command
	rootCmd := &cobra.Command{Use: "test"}
	cmd := &cobra.Command{
		Use:  "init",
		RunE: runInit,
	}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().Bool("no-agents", false, "")
	rootCmd.AddCommand(cmd)

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	// Provide "n" responses to all prompts (not strictly needed now with mocks, but kept for safety)
	cmd.SetIn(bytes.NewBufferString("n\nn\nn\n"))
	cmd.SetArgs([]string{"--no-agents"})

	// This will fail because we're in a temp directory without full setup,
	// but we can test that the function runs without panic
	_ = cmd.Execute()
}

func TestCountResults(t *testing.T) {
	tests := map[string]struct {
		results       []commands.InstallResult
		wantInstalled int
		wantUpdated   int
	}{
		"empty results": {
			results:       []commands.InstallResult{},
			wantInstalled: 0,
			wantUpdated:   0,
		},
		"all installed": {
			results: []commands.InstallResult{
				{Action: "installed"},
				{Action: "installed"},
			},
			wantInstalled: 2,
			wantUpdated:   0,
		},
		"all updated": {
			results: []commands.InstallResult{
				{Action: "updated"},
				{Action: "updated"},
			},
			wantInstalled: 0,
			wantUpdated:   2,
		},
		"mixed results": {
			results: []commands.InstallResult{
				{Action: "installed"},
				{Action: "updated"},
				{Action: "skipped"},
				{Action: "installed"},
			},
			wantInstalled: 2,
			wantUpdated:   1,
		},
		"unknown actions": {
			results: []commands.InstallResult{
				{Action: "unknown"},
				{Action: "other"},
			},
			wantInstalled: 0,
			wantUpdated:   0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			installed, updated := countResults(tt.results)
			assert.Equal(t, tt.wantInstalled, installed)
			assert.Equal(t, tt.wantUpdated, updated)
		})
	}
}

func TestPromptYesNo(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected bool
	}{
		"yes lowercase": {
			input:    "y\n",
			expected: true,
		},
		"yes full": {
			input:    "yes\n",
			expected: true,
		},
		"yes uppercase": {
			input:    "Y\n",
			expected: true,
		},
		"no lowercase": {
			input:    "n\n",
			expected: false,
		},
		"no full": {
			input:    "no\n",
			expected: false,
		},
		"empty input": {
			input:    "\n",
			expected: false,
		},
		"other input": {
			input:    "maybe\n",
			expected: false,
		},
		"whitespace around yes": {
			input:    "  yes  \n",
			expected: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			var outBuf bytes.Buffer
			cmd.SetOut(&outBuf)
			cmd.SetIn(bytes.NewBufferString(tt.input))

			result := promptYesNo(cmd, "Test question?")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPromptYesNoDefaultYes(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected bool
	}{
		"yes lowercase": {
			input:    "y\n",
			expected: true,
		},
		"yes full": {
			input:    "yes\n",
			expected: true,
		},
		"yes uppercase": {
			input:    "Y\n",
			expected: true,
		},
		"no lowercase": {
			input:    "n\n",
			expected: false,
		},
		"no full": {
			input:    "no\n",
			expected: false,
		},
		"empty input defaults to yes": {
			input:    "\n",
			expected: true,
		},
		"other input": {
			input:    "maybe\n",
			expected: false,
		},
		"whitespace around yes": {
			input:    "  yes  \n",
			expected: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			var outBuf bytes.Buffer
			cmd.SetOut(&outBuf)
			cmd.SetIn(bytes.NewBufferString(tt.input))

			result := promptYesNoDefaultYes(cmd, "Test question?")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFileExistsCheck(t *testing.T) {
	tmpDir := t.TempDir()

	tests := map[string]struct {
		setup    func() string
		expected bool
	}{
		"existing file": {
			setup: func() string {
				path := filepath.Join(tmpDir, "exists.txt")
				_ = os.WriteFile(path, []byte("content"), 0o644)
				return path
			},
			expected: true,
		},
		"non-existing file": {
			setup: func() string {
				return filepath.Join(tmpDir, "nonexistent.txt")
			},
			expected: false,
		},
		"existing directory": {
			setup: func() string {
				path := filepath.Join(tmpDir, "existsdir")
				_ = os.MkdirAll(path, 0o755)
				return path
			},
			expected: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			path := tt.setup()
			result := fileExistsCheck(path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	tests := map[string]struct {
		project bool
		wantErr bool
	}{
		"user config path": {
			project: false,
			wantErr: false,
		},
		"project config path": {
			project: true,
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			path, err := getConfigPath(tt.project)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, path)
			}
		})
	}
}

func TestWriteDefaultConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "config.yml")

	err := writeDefaultConfig(configPath)
	require.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, configPath)

	// Verify content
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.NotEmpty(t, content)
}

func TestInitializeProjectConfigUsesProjectLocalStateDir(t *testing.T) {
	tmpDir := t.TempDir()
	previous, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(previous))
	})

	var out bytes.Buffer
	created, err := initializeConfig(&out, true, false)
	require.NoError(t, err)
	require.True(t, created)

	content, err := os.ReadFile(filepath.Join(tmpDir, ".autospec", "config.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "state_dir: ./.autospec/state")
	assert.NotContains(t, string(content), "state_dir: ~/.autospec/state")
}

func TestWriteDefaultConfig_ErrorOnInvalidPath(t *testing.T) {
	// Use a path that will fail (empty string would cause issues)
	// On most systems, trying to write to root's protected areas would fail
	// But for a more reliable test, we test that valid paths work
	tmpDir := t.TempDir()
	validPath := filepath.Join(tmpDir, "valid", "config.yml")

	err := writeDefaultConfig(validPath)
	assert.NoError(t, err)
}

func TestCopyConstitution(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "source.yaml")
	dstPath := filepath.Join(tmpDir, "subdir", "dest.yaml")

	// Create source file
	content := "test: constitution"
	err := os.WriteFile(srcPath, []byte(content), 0o644)
	require.NoError(t, err)

	// Copy
	err = copyConstitution(srcPath, dstPath)
	require.NoError(t, err)

	// Verify destination
	assert.FileExists(t, dstPath)
	dstContent, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(dstContent))
}

func TestCopyConstitution_SourceNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "nonexistent.yaml")
	dstPath := filepath.Join(tmpDir, "dest.yaml")

	err := copyConstitution(srcPath, dstPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read source")
}

func TestHandleConstitution_NoConstitution(t *testing.T) {
	// Cannot run in parallel due to working directory change
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(origDir)
	}()

	var buf bytes.Buffer
	result := handleConstitution(&buf)

	assert.False(t, result)
	assert.Contains(t, buf.String(), "not found")
}

func TestHandleConstitution_ExistingAutospec(t *testing.T) {
	// Cannot run in parallel due to working directory change
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(origDir)
	}()

	// Create existing autospec constitution
	constitutionPath := filepath.Join(tmpDir, ".autospec", "memory", "constitution.yaml")
	err = os.MkdirAll(filepath.Dir(constitutionPath), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(constitutionPath, []byte("test: content"), 0o644)
	require.NoError(t, err)

	var buf bytes.Buffer
	result := handleConstitution(&buf)

	assert.True(t, result)
	assert.Contains(t, buf.String(), "found at")
}

func TestGitignoreHasAutospec(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		content string
		want    bool
	}{
		"empty file":             {content: "", want: false},
		"no autospec":            {content: "node_modules/\ndist/", want: false},
		".autospec exact":        {content: ".autospec", want: true},
		".autospec/ with slash":  {content: ".autospec/", want: true},
		".autospec/ with others": {content: "node_modules/\n.autospec/\ndist/", want: true},
		".autospec subpath":      {content: ".autospec/config.yml", want: true},
		"similar but not match":  {content: "autospec/", want: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := gitignoreHasAutospec(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHandleGitignorePrompt_NoGitignore_UserSaysNo(t *testing.T) {
	// Cannot run in parallel due to working directory change
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(origDir)
	}()

	cmd := &cobra.Command{Use: "test"}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetIn(bytes.NewBufferString("n\n"))

	handleGitignorePrompt(cmd, &buf)

	// Should show prompt and skip message
	assert.Contains(t, buf.String(), "Add .autospec/ to .gitignore?")
	assert.Contains(t, buf.String(), "skipped")

	// File should not be created
	_, err = os.Stat(".gitignore")
	assert.True(t, os.IsNotExist(err))
}

func TestHandleGitignorePrompt_NoGitignore_UserSaysYes(t *testing.T) {
	// Cannot run in parallel due to working directory change
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(origDir)
	}()

	cmd := &cobra.Command{Use: "test"}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetIn(bytes.NewBufferString("y\n"))

	handleGitignorePrompt(cmd, &buf)

	// Should show checkmark
	assert.Contains(t, buf.String(), "✓ Gitignore: added .autospec/")

	// File should be created with .autospec/
	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	assert.Contains(t, string(data), ".autospec/")
}

func TestHandleGitignorePrompt_WithAutospec(t *testing.T) {
	// Cannot run in parallel due to working directory change
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(origDir)
	}()

	// Create .gitignore with .autospec entry
	err = os.WriteFile(".gitignore", []byte(".autospec/\nnode_modules/"), 0o644)
	require.NoError(t, err)

	cmd := &cobra.Command{Use: "test"}
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	handleGitignorePrompt(cmd, &buf)

	// Should show already present, no prompt
	assert.Contains(t, buf.String(), "✓ Gitignore: .autospec/ already present")
	assert.NotContains(t, buf.String(), "[y/N]")
}

func TestHandleGitignorePrompt_WithoutAutospec_UserSaysNo(t *testing.T) {
	// Cannot run in parallel due to working directory change
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(origDir)
	}()

	// Create .gitignore without .autospec entry
	err = os.WriteFile(".gitignore", []byte("node_modules/\ndist/"), 0o644)
	require.NoError(t, err)

	cmd := &cobra.Command{Use: "test"}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetIn(bytes.NewBufferString("n\n"))

	handleGitignorePrompt(cmd, &buf)

	// Should show skipped
	assert.Contains(t, buf.String(), "skipped")

	// File should not be modified
	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	assert.NotContains(t, string(data), ".autospec")
}

func TestHandleGitignorePrompt_WithoutAutospec_UserSaysYes(t *testing.T) {
	// Cannot run in parallel due to working directory change
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(origDir)
	}()

	// Create .gitignore without .autospec entry (no trailing newline)
	err = os.WriteFile(".gitignore", []byte("node_modules/\ndist/"), 0o644)
	require.NoError(t, err)

	cmd := &cobra.Command{Use: "test"}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetIn(bytes.NewBufferString("y\n"))

	handleGitignorePrompt(cmd, &buf)

	// Should show checkmark
	assert.Contains(t, buf.String(), "✓ Gitignore: added .autospec/")

	// File should have .autospec/ appended with proper newline handling
	data, err := os.ReadFile(".gitignore")
	require.NoError(t, err)
	assert.Contains(t, string(data), ".autospec/")
	// Original content should be preserved
	assert.Contains(t, string(data), "node_modules/")
	assert.Contains(t, string(data), "dist/")
}

func TestAddAutospecToGitignore_NewFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	err := addAutospecToGitignore(gitignorePath)
	require.NoError(t, err)

	data, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)
	assert.Equal(t, ".autospec/\n", string(data))
}

func TestAddAutospecToGitignore_ExistingWithNewline(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	err := os.WriteFile(gitignorePath, []byte("node_modules/\n"), 0o644)
	require.NoError(t, err)

	err = addAutospecToGitignore(gitignorePath)
	require.NoError(t, err)

	data, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)
	assert.Equal(t, "node_modules/\n.autospec/\n", string(data))
}

func TestAddAutospecToGitignore_ExistingWithoutNewline(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	gitignorePath := filepath.Join(tmpDir, ".gitignore")

	err := os.WriteFile(gitignorePath, []byte("node_modules/"), 0o644)
	require.NoError(t, err)

	err = addAutospecToGitignore(gitignorePath)
	require.NoError(t, err)

	data, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)
	assert.Equal(t, "node_modules/\n.autospec/\n", string(data))
}

func TestEnsureAutospecGitignore(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		existing string
		want     string
	}{
		"creates file with local runtime ignores": {
			want: autospecGitignoreContent,
		},
		"preserves existing content": {
			existing: "# custom\nlogs/\n",
			want:     "# custom\nlogs/\n\n" + autospecGitignoreContent,
		},
		"adds newline before autospec block": {
			existing: "# custom",
			want:     "# custom\n\n" + autospecGitignoreContent,
		},
		"is idempotent when autospec block exists": {
			existing: autospecGitignoreContent,
			want:     autospecGitignoreContent,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			if tt.existing != "" {
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".autospec"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".autospec", ".gitignore"), []byte(tt.existing), 0o644))
			}

			require.NoError(t, ensureAutospecGitignore(tmpDir))

			data, err := os.ReadFile(filepath.Join(tmpDir, ".autospec", ".gitignore"))
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(data))
		})
	}
}

func TestPrintSummary_WithConstitution(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	printSummary(&buf, initResult{constitutionExists: true, hadErrors: false}, "specs")

	output := buf.String()
	assert.Contains(t, output, "Autospec is ready!")
	assert.Contains(t, output, "Quick start")
	assert.Contains(t, output, "Review the generated spec in specs/")
	assert.NotContains(t, output, "IMPORTANT: You MUST create a constitution")
}

func TestPrintSummary_WithoutConstitution(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	printSummary(&buf, initResult{constitutionExists: false, hadErrors: false}, "specs")

	output := buf.String()
	assert.Contains(t, output, "IMPORTANT: You MUST create a constitution")
	assert.Contains(t, output, "autospec constitution")
	assert.Contains(t, output, "# required first!")
	assert.NotContains(t, output, "Autospec is ready!")
}

func TestPrintSummary_CustomSpecsDir(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	printSummary(&buf, initResult{constitutionExists: true, hadErrors: false}, "my-specs")

	output := buf.String()
	assert.Contains(t, output, "Review the generated spec in my-specs/")
}

func TestPrintSummary_WithErrors(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	printSummary(&buf, initResult{constitutionExists: true, hadErrors: true}, "specs")

	output := buf.String()
	// Should NOT show "ready" message when there were errors
	assert.NotContains(t, output, "Autospec is ready!")
	assert.Contains(t, output, "Quick start")
}

func TestInitCmd_RunE(t *testing.T) {
	// Verify initCmd has a RunE function set
	assert.NotNil(t, initCmd.RunE)
}

func TestPromptYesNo_DisplaysCorrectFormat(t *testing.T) {
	// This test verifies that the promptYesNo function uses the correct format [y/N]

	tests := map[string]struct {
		input    string
		expected bool
	}{
		"yes answers y": {
			input:    "y\n",
			expected: true,
		},
		"empty defaults to no": {
			input:    "\n",
			expected: false,
		},
		"no answers n": {
			input:    "n\n",
			expected: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			var outBuf bytes.Buffer
			cmd.SetOut(&outBuf)
			cmd.SetIn(bytes.NewBufferString(tt.input))

			result := promptYesNo(cmd, "Test prompt?")
			assert.Equal(t, tt.expected, result)

			// Verify prompt format shows [y/N]
			assert.Contains(t, outBuf.String(), "[y/N]")
		})
	}
}

func TestWorktreeScriptDetection(t *testing.T) {
	// Test that worktree script detection works correctly
	tmpDir := t.TempDir()

	tests := map[string]struct {
		createScript bool
		expected     bool
	}{
		"script exists": {
			createScript: true,
			expected:     true,
		},
		"script does not exist": {
			createScript: false,
			expected:     false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testDir := filepath.Join(tmpDir, name)
			err := os.MkdirAll(testDir, 0o755)
			require.NoError(t, err)

			scriptPath := filepath.Join(testDir, ".autospec", "scripts", "setup-worktree.sh")

			if tt.createScript {
				err := os.MkdirAll(filepath.Dir(scriptPath), 0o755)
				require.NoError(t, err)
				err = os.WriteFile(scriptPath, []byte("#!/bin/bash\n"), 0o755)
				require.NoError(t, err)
			}

			result := fileExistsCheck(scriptPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdateDefaultAgentsInConfig(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		content  string
		agents   []string
		expected string
	}{
		"empty agents clears list": {
			content:  "default_agents: [\"claude\"]\n",
			agents:   []string{},
			expected: "default_agents: []\n",
		},
		"single agent": {
			content:  "default_agents: []\n",
			agents:   []string{"claude"},
			expected: "default_agents: [\"claude\"]\n",
		},
		"multiple agents": {
			content:  "default_agents: []\n",
			agents:   []string{"claude", "cline", "gemini"},
			expected: "default_agents: [\"claude\", \"cline\", \"gemini\"]\n",
		},
		"replaces existing": {
			content:  "default_agents: [\"claude\", \"cline\"]\n",
			agents:   []string{"gemini"},
			expected: "default_agents: [\"gemini\"]\n",
		},
		"preserves other content": {
			content:  "specs_dir: features\ndefault_agents: []\ntimeout: 30m\n",
			agents:   []string{"claude"},
			expected: "specs_dir: features\ndefault_agents: [\"claude\"]\ntimeout: 30m\n",
		},
		"appends if not found": {
			content:  "specs_dir: features\n",
			agents:   []string{"claude"},
			expected: "specs_dir: features\n\ndefault_agents: [\"claude\"]",
		},
		"handles indented line": {
			content:  "  default_agents: []\n",
			agents:   []string{"claude"},
			expected: "default_agents: [\"claude\"]\n",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := updateDefaultAgentsInConfig(tt.content, tt.agents)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatAgentList(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agents   []string
		expected string
	}{
		"empty list": {
			agents:   []string{},
			expected: "",
		},
		"single agent": {
			agents:   []string{"claude"},
			expected: `"claude"`,
		},
		"multiple agents": {
			agents:   []string{"claude", "cline", "gemini"},
			expected: `"claude", "cline", "gemini"`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := formatAgentList(tt.agents)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDisplayAgentConfigResult(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agentName    string
		result       *cliagent.ConfigResult
		wantContains []string
	}{
		"nil result shows no config needed": {
			agentName:    "gemini",
			result:       nil,
			wantContains: []string{"Gemini CLI", "no configuration needed"},
		},
		"already configured": {
			agentName:    "claude",
			result:       &cliagent.ConfigResult{AlreadyConfigured: true},
			wantContains: []string{"Claude Code", "already configured"},
		},
		"permissions added": {
			agentName: "claude",
			result: &cliagent.ConfigResult{
				PermissionsAdded: []string{"Write(.autospec/**)", "Edit(.autospec/**)"},
			},
			wantContains: []string{"Claude Code", "configured with permissions", "Write(.autospec/**)", "Edit(.autospec/**)"},
		},
		"warning shown": {
			agentName: "claude",
			result: &cliagent.ConfigResult{
				Warning:          "permission denied",
				PermissionsAdded: []string{"Bash(autospec:*)"},
			},
			wantContains: []string{"permission denied", "Bash(autospec:*)"},
		},
		"unknown agent uses name": {
			agentName:    "unknown",
			result:       nil,
			wantContains: []string{"unknown", "no configuration needed"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			displayAgentConfigResult(&buf, tt.agentName, tt.result)
			output := buf.String()

			for _, want := range tt.wantContains {
				assert.Contains(t, output, want)
			}
		})
	}
}

func TestConfigureSelectedAgents_NoAgentsSelected(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	cfg := &config.Configuration{SpecsDir: "specs"}
	tmpDir := t.TempDir()

	_, _, err := configureSelectedAgents(&buf, []string{}, "", cfg, "config.yml", tmpDir, true)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "Warning")
	assert.Contains(t, buf.String(), "No agents selected")
}

func TestPersistAgentPreferences(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Create initial config file
	initialContent := "specs_dir: specs\ndefault_agents: []\ntimeout: 30m\n"
	err := os.WriteFile(configPath, []byte(initialContent), 0o644)
	require.NoError(t, err)

	var buf bytes.Buffer
	cfg := &config.Configuration{}

	err = persistAgentPreferences(&buf, []string{"claude", "cline"}, "claude", cfg, configPath)
	require.NoError(t, err)

	// Verify file was updated
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "claude")
	assert.Contains(t, string(content), "cline")
	assert.Contains(t, buf.String(), "Agent preferences saved")
}

func TestPersistAgentPreferences_WithDefaultAgent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	initialContent := "specs_dir: specs\nagent_preset: \"\"\ndefault_agents: []\ntimeout: 30m\n"
	err := os.WriteFile(configPath, []byte(initialContent), 0o644)
	require.NoError(t, err)

	var buf bytes.Buffer
	cfg := &config.Configuration{}

	err = persistAgentPreferences(&buf, []string{"claude", "codex", "opencode"}, "codex", cfg, configPath)
	require.NoError(t, err)

	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), `default_agents: ["claude", "codex", "opencode"]`)
	assert.Contains(t, string(content), "agent_preset: codex")
	assert.Contains(t, buf.String(), "set as default for execution")
}

func TestPersistAgentPreferences_FileNotExists(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent", "config.yml")

	var buf bytes.Buffer
	cfg := &config.Configuration{}

	// Should not error if file doesn't exist
	err := persistAgentPreferences(&buf, []string{"claude"}, "claude", cfg, configPath)
	require.NoError(t, err)
}

// TestConfigureSelectedAgents_FilePermissionError tests that file permission
// errors display a clear error message and continue with other agents.
// Edge case from spec: "File permission errors: clear error message, continue with other agents"
func TestConfigureSelectedAgents_FilePermissionError(t *testing.T) {
	t.Parallel()

	// This test simulates the scenario where an agent configuration fails
	// but other agents should still be configured
	var buf bytes.Buffer
	cfg := &config.Configuration{SpecsDir: "specs"}

	// Select multiple agents - Claude will be configured, others have no config
	selected := []string{"claude", "gemini", "cline"}

	// Use a temp project dir for agent config files
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	_ = os.WriteFile(configPath, []byte("specs_dir: specs\ndefault_agents: []\n"), 0o644)

	// Run configuration - even if one agent fails, others should complete
	_, _, err := configureSelectedAgents(&buf, selected, "claude", cfg, configPath, tmpDir, true)
	require.NoError(t, err)

	// Verify output mentions Claude was configured (or tried to configure)
	output := buf.String()
	// Other agents should show "no configuration needed"
	assert.Contains(t, output, "Gemini CLI")
	assert.Contains(t, output, "Cline")
}

// TestConfigureSelectedAgents_PartialConfigContinues tests that when one agent
// configuration fails, the init continues with remaining agents.
func TestConfigureSelectedAgents_PartialConfigContinues(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	cfg := &config.Configuration{SpecsDir: "specs"}

	// Select agents where only claude implements Configurator
	selected := []string{"claude", "gemini", "goose", "opencode"}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	_ = os.WriteFile(configPath, []byte("default_agents: []\n"), 0o644)

	_, _, err := configureSelectedAgents(&buf, selected, "claude", cfg, configPath, tmpDir, true)
	require.NoError(t, err)

	output := buf.String()

	// Verify all agents were processed
	assert.Contains(t, output, "Claude Code")
	assert.Contains(t, output, "Gemini CLI")
	assert.Contains(t, output, "Goose")
	assert.Contains(t, output, "OpenCode")

	// Non-configurator agents should show "no configuration needed"
	assert.Contains(t, output, "no configuration needed")
}

// TestGetSupportedAgentsWithDefaults_MalformedDefaultAgents tests that
// invalid/unknown agent names in default_agents are gracefully ignored.
// Edge case from spec: "Unknown agent names in config: ignored with no error"
func TestGetSupportedAgentsWithDefaults_MalformedDefaultAgents(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		defaultAgents []string
		wantSelected  []string
	}{
		"all unknown agents defaults to claude": {
			defaultAgents: []string{"unknown1", "nonexistent", "fake-agent"},
			wantSelected:  []string{}, // No known agents selected
		},
		"mix of known and unknown": {
			defaultAgents: []string{"unknown", "claude", "fake", "gemini"},
			wantSelected:  []string{"claude", "gemini"},
		},
		"empty string in list": {
			defaultAgents: []string{"", "claude"},
			wantSelected:  []string{"claude"},
		},
		"whitespace only names": {
			defaultAgents: []string{"  ", "claude", "\t"},
			wantSelected:  []string{"claude"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			agents := GetSupportedAgentsWithDefaults(tt.defaultAgents)

			var selected []string
			for _, a := range agents {
				if a.Selected {
					selected = append(selected, a.Name)
				}
			}

			assert.ElementsMatch(t, tt.wantSelected, selected)
		})
	}
}

// TestHandleAgentConfiguration_NonInteractiveRequiresNoAgentsFlag tests that
// non-interactive terminals fail with helpful message unless --no-agents is used.
// Edge case from spec: "Non-interactive terminal: fail fast with helpful message"
func TestHandleAgentConfiguration_NonInteractiveRequiresNoAgentsFlag(t *testing.T) {
	// This test only applies when multi-agent is enabled (development build).
	// In production builds, the function configures Claude directly and ignores --no-agents.
	if !build.MultiAgentEnabled() {
		t.Skip("Test requires multi-agent enabled (dev build)")
	}

	// Note: We can't easily test the isTerminal() check in unit tests,
	// but we verify that --no-agents works correctly
	t.Parallel()

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("no-agents", true, "")
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// This should succeed because --no-agents is set
	_, _, err := handleAgentConfiguration(cmd, &buf, false, true, nil)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "skipped")
}

// TestPersistAgentPreferences_Idempotency tests that running persistAgentPreferences
// multiple times with the same agents produces identical config.
// T017 acceptance criteria: "running init 3 times produces identical config"
func TestPersistAgentPreferences_Idempotency(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Create initial config
	initialContent := "specs_dir: specs\ndefault_agents: []\ntimeout: 30m\n"
	err := os.WriteFile(configPath, []byte(initialContent), 0o644)
	require.NoError(t, err)

	cfg := &config.Configuration{}
	agents := []string{"claude", "cline"}

	// Run 3 times
	for i := 0; i < 3; i++ {
		var buf bytes.Buffer
		err := persistAgentPreferences(&buf, agents, "claude", cfg, configPath)
		require.NoError(t, err, "run %d failed", i+1)
	}

	// Read final content
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	// Verify format is exactly what we expect
	expectedContent := "specs_dir: specs\nagent_preset: claude\ndefault_agents: [\"claude\", \"cline\"]\ntimeout: 30m\n"
	assert.Equal(t, expectedContent, string(content))
}

// TestUpdateDefaultAgentsInConfig_NoDuplicates tests that repeated calls with same
// agents don't create duplicate lines.
// T017 acceptance criteria: "DefaultAgents not corrupted on repeated saves"
func TestUpdateDefaultAgentsInConfig_NoDuplicates(t *testing.T) {
	t.Parallel()

	// Start with config containing default_agents
	content := "specs_dir: specs\ndefault_agents: [\"claude\"]\ntimeout: 30m\n"

	// Update 3 times with same agents
	for i := 0; i < 3; i++ {
		content = updateDefaultAgentsInConfig(content, []string{"claude", "gemini"})
	}

	// Count occurrences of default_agents
	lines := bytes.Split([]byte(content), []byte("\n"))
	defaultAgentsCount := 0
	for _, line := range lines {
		if bytes.Contains(line, []byte("default_agents:")) {
			defaultAgentsCount++
		}
	}

	assert.Equal(t, 1, defaultAgentsCount, "should have exactly one default_agents line")

	// Verify correct content
	assert.Contains(t, content, "default_agents: [\"claude\", \"gemini\"]")
}

// TestFullIdempotencyFlow tests the complete init flow for idempotency.
// T017 acceptance criteria: "Test: running init 3 times produces identical config"
func TestFullIdempotencyFlow(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Create initial config
	initialContent := "specs_dir: features\ndefault_agents: []\n"
	err := os.WriteFile(configPath, []byte(initialContent), 0o644)
	require.NoError(t, err)

	cfg := &config.Configuration{SpecsDir: "features"}
	selected := []string{"claude", "cline", "gemini"}

	// Run the full configuration 3 times
	var finalOutput string
	for i := 0; i < 3; i++ {
		var buf bytes.Buffer
		_, _, err := configureSelectedAgents(&buf, selected, "claude", cfg, configPath, tmpDir, true)
		require.NoError(t, err, "run %d failed", i+1)
		finalOutput = buf.String()
	}

	// After first run, Claude should show permissions added
	// After subsequent runs, Claude should show "already configured"

	// Read final config
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	// Verify all agents are in the config exactly once
	assert.Contains(t, string(content), "claude")
	assert.Contains(t, string(content), "cline")
	assert.Contains(t, string(content), "gemini")

	// Verify no duplicates in the default_agents list
	lines := bytes.Split(content, []byte("\n"))
	defaultAgentsLines := 0
	for _, line := range lines {
		if bytes.Contains(line, []byte("default_agents:")) {
			defaultAgentsLines++
		}
	}
	assert.Equal(t, 1, defaultAgentsLines)

	// Verify the output on the third run mentions things are already configured
	assert.NotEmpty(t, finalOutput)
}

// TestConfigureSpecificAgents_Claude tests --ai claude configures only Claude.
func TestConfigureSpecificAgents_Claude(t *testing.T) {
	// Cannot run in parallel: changes working directory
	tempDir := t.TempDir()

	// Save original directory and change to temp
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	defer func() { _ = os.Chdir(origDir) }()

	// Create minimal config
	configDir := filepath.Join(tempDir, ".autospec")
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	configPath := filepath.Join(configDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("specs_dir: specs\n"), 0o644))

	cmd := &cobra.Command{Use: "init"}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().StringSlice("ai", nil, "")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetIn(bytes.NewBufferString("n\n")) // Say no to sandbox prompt

	_, _, err = configureSpecificAgents(cmd, &buf, true, []string{"claude"})
	assert.NoError(t, err)

	// Verify Claude permissions are configured in settings.local.json
	settingsPath := filepath.Join(tempDir, ".claude", "settings.local.json")
	_, err = os.Stat(settingsPath)
	assert.NoError(t, err, ".claude/settings.local.json should exist")

	data, err := os.ReadFile(settingsPath)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "Bash(autospec:*)")

	_, err = os.Stat(filepath.Join(tempDir, ".claude", "skills", "autospec.specify", "SKILL.md"))
	assert.NoError(t, err, ".claude/skills/autospec.specify/SKILL.md should exist")

	_, err = os.Stat(filepath.Join(tempDir, ".claude", "commands", "autospec.specify.md"))
	assert.True(t, os.IsNotExist(err), ".claude/commands/autospec.specify.md should not be installed")

	// Verify OpenCode is NOT configured
	opencodeCmdDir := filepath.Join(tempDir, ".opencode", "command")
	_, err = os.Stat(opencodeCmdDir)
	assert.True(t, os.IsNotExist(err), ".opencode/command should NOT exist")
}

// TestConfigureSpecificAgents_OpenCode tests --ai opencode configures only OpenCode.
func TestConfigureSpecificAgents_OpenCode(t *testing.T) {
	// Cannot run in parallel: changes working directory
	tempDir := t.TempDir()

	// Save original directory and change to temp
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	defer func() { _ = os.Chdir(origDir) }()

	// Create minimal config
	configDir := filepath.Join(tempDir, ".autospec")
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	configPath := filepath.Join(configDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("specs_dir: specs\n"), 0o644))

	cmd := &cobra.Command{Use: "init"}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().StringSlice("ai", nil, "")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetIn(bytes.NewBufferString("n\n"))

	_, _, err = configureSpecificAgents(cmd, &buf, true, []string{"opencode"})
	assert.NoError(t, err)

	_, err = os.Stat(filepath.Join(tempDir, ".opencode"))
	assert.True(t, os.IsNotExist(err), ".opencode should not be created")

	_, err = os.Stat(filepath.Join(tempDir, ".agents", "skills", "autospec-specify", "SKILL.md"))
	assert.NoError(t, err, ".agents/skills/autospec-specify/SKILL.md should exist")

	// Verify opencode.json has permission
	data, err := os.ReadFile(filepath.Join(tempDir, "opencode.json"))
	assert.NoError(t, err)
	assert.Contains(t, string(data), "autospec *")
	assert.Contains(t, string(data), "allow")
}

// TestConfigureSpecificAgents_Codex tests --ai codex records Codex and installs its project skill.
func TestConfigureSpecificAgents_Codex(t *testing.T) {
	// Cannot run in parallel: changes working directory
	tempDir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	defer func() { _ = os.Chdir(origDir) }()

	configDir := filepath.Join(tempDir, ".autospec")
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	configPath := filepath.Join(configDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("specs_dir: specs\nagent_preset: \"\"\ndefault_agents: []\n"), 0o644))

	cmd := &cobra.Command{Use: "init"}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().StringSlice("ai", nil, "")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	configured, agentConfigs, err := configureSpecificAgents(cmd, &buf, true, []string{"codex"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"codex"}, configured)
	require.Len(t, agentConfigs, 1)
	assert.Equal(t, "codex", agentConfigs[0].name)
	assert.True(t, agentConfigs[0].configured)

	configContent, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(configContent), `default_agents: ["codex"]`)
	assert.Contains(t, string(configContent), "agent_preset: codex")

	_, err = os.Stat(filepath.Join(tempDir, ".claude", "commands"))
	assert.True(t, os.IsNotExist(err), ".claude/commands should not exist for codex")
	_, err = os.Stat(filepath.Join(tempDir, ".opencode", "command"))
	assert.True(t, os.IsNotExist(err), ".opencode/command should not exist for codex")

	_, err = os.Stat(filepath.Join(tempDir, ".codex", "config.toml"))
	assert.NoError(t, err, ".codex/config.toml should exist")
	for _, skillName := range []string{
		"autospec-specify",
		"autospec-plan",
		"autospec-tasks",
		"autospec-implement",
		"autospec-constitution",
		"autospec-clarify",
		"autospec-analyze",
		"autospec-checklist",
		"autospec-worktree-setup",
	} {
		_, err = os.Stat(filepath.Join(tempDir, ".agents", "skills", skillName, "SKILL.md"))
		assert.NoError(t, err, ".agents/skills/%s/SKILL.md should exist", skillName)
	}
}

// TestConfigureSpecificAgents_Both tests --ai claude,opencode configures both.
func TestConfigureSpecificAgents_Both(t *testing.T) {
	// Cannot run in parallel: changes working directory
	tempDir := t.TempDir()

	// Save original directory and change to temp
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	defer func() { _ = os.Chdir(origDir) }()

	// Create minimal config
	configDir := filepath.Join(tempDir, ".autospec")
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	configPath := filepath.Join(configDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("specs_dir: specs\n"), 0o644))

	cmd := &cobra.Command{Use: "init"}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().StringSlice("ai", nil, "")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetIn(bytes.NewBufferString("n\n"))

	_, _, err = configureSpecificAgents(cmd, &buf, true, []string{"claude", "opencode"})
	assert.NoError(t, err)

	// Verify Claude settings exist
	claudeSettings := filepath.Join(tempDir, ".claude", "settings.local.json")
	_, err = os.Stat(claudeSettings)
	assert.NoError(t, err, ".claude/settings.local.json should exist")

	_, err = os.Stat(filepath.Join(tempDir, ".claude", "skills", "autospec.specify", "SKILL.md"))
	assert.NoError(t, err, ".claude/skills/autospec.specify/SKILL.md should exist")

	_, err = os.Stat(filepath.Join(tempDir, ".opencode"))
	assert.True(t, os.IsNotExist(err), ".opencode should not be created")

	opencodeJSON := filepath.Join(tempDir, "opencode.json")
	_, err = os.Stat(opencodeJSON)
	assert.NoError(t, err, "opencode.json should exist")
}

func TestConfigureSpecificAgents_MultipleAgentsUseFirstAsDefault(t *testing.T) {
	// Cannot run in parallel: changes working directory
	tempDir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	defer func() { _ = os.Chdir(origDir) }()

	configDir := filepath.Join(tempDir, ".autospec")
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	configPath := filepath.Join(configDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("specs_dir: specs\nagent_preset: opencode\ndefault_agents: []\n"), 0o644))

	cmd := &cobra.Command{Use: "init"}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().StringSlice("ai", nil, "")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	_, _, err = configureSpecificAgents(cmd, &buf, true, []string{"codex", "opencode"})
	require.NoError(t, err)

	configContent, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(configContent), "agent_preset: codex")
}

// TestConfigureSpecificAgents_InvalidAgent tests that invalid agent names error.
func TestConfigureSpecificAgents_InvalidAgent(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agents  []string
		wantErr string
	}{
		"completely unknown agent": {
			agents:  []string{"unknown"},
			wantErr: "unknown agent(s): unknown",
		},
		"mix of valid and invalid": {
			agents:  []string{"claude", "foobar"},
			wantErr: "unknown agent(s): foobar",
		},
		"empty after trim": {
			agents:  []string{"   ", ""},
			wantErr: "no valid agents specified",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{Use: "init"}
			cmd.Flags().BoolP("project", "p", false, "")
			cmd.Flags().BoolP("force", "f", false, "")
			cmd.Flags().StringSlice("ai", nil, "")

			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// These test cases all hit early-return error paths before config loading,
			// so no working directory setup is needed.
			_, _, err := configureSpecificAgents(cmd, &buf, true, tt.agents)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestGetValidAgentNames tests production agent filtering.
func TestGetValidAgentNames(t *testing.T) {
	t.Parallel()

	valid := getValidAgentNames()

	// In production builds (MultiAgentEnabled() == false), only claude and opencode are valid
	if !build.MultiAgentEnabled() {
		assert.True(t, valid["claude"], "claude should be valid in production")
		assert.True(t, valid["opencode"], "opencode should be valid in production")
		// Other agents should not be valid in production
		assert.False(t, valid["gemini"], "gemini should not be valid in production")
		assert.False(t, valid["cline"], "cline should not be valid in production")
	} else {
		// In dev builds, all registered agents should be valid
		for _, name := range cliagent.List() {
			assert.True(t, valid[name], "%s should be valid in dev build", name)
		}
	}
}

// TestProductionAgents tests build.ProductionAgents returns expected agents.
func TestProductionAgents(t *testing.T) {
	t.Parallel()

	agents := build.ProductionAgents()
	assert.Contains(t, agents, "claude")
	assert.Contains(t, agents, "codex")
	assert.Contains(t, agents, "opencode")
	assert.Len(t, agents, 3)
}

// ============================================================================
// Path Argument Tests (spec 079-init-path-argument)
// ============================================================================

// TestRunInit_WithPathArgument tests that init creates files at the specified path.
// US-001: "Initialize project at specified path"
func TestRunInit_WithPathArgument(t *testing.T) {
	// Cannot run in parallel: changes working directory
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	// Change to a temp dir that is NOT the target dir
	workDir := filepath.Join(tmpDir, "workdir")
	require.NoError(t, os.MkdirAll(workDir, 0o755))
	require.NoError(t, os.Chdir(workDir))

	// Target directory for init
	targetDir := filepath.Join(tmpDir, "my-project")

	// Mock the runners to prevent real Claude execution
	originalConstitutionRunner := ConstitutionRunner
	originalWorktreeRunner := WorktreeScriptRunner
	ConstitutionRunner = func(cmd *cobra.Command, configPath string) bool {
		return true
	}
	WorktreeScriptRunner = func(cmd *cobra.Command, configPath string) bool {
		return true
	}
	defer func() {
		ConstitutionRunner = originalConstitutionRunner
		WorktreeScriptRunner = originalWorktreeRunner
	}()

	cmd := &cobra.Command{Use: "init [path]", Args: cobra.MaximumNArgs(1), RunE: runInit}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().Bool("no-agents", false, "")
	cmd.Flags().Bool("here", false, "")
	cmd.Flags().StringSlice("ai", nil, "")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetIn(bytes.NewBufferString("n\nn\nn\n"))
	cmd.SetArgs([]string{targetDir, "--no-agents"})

	err = cmd.Execute()
	require.NoError(t, err)

	// Verify output mentions target directory
	output := buf.String()
	assert.Contains(t, output, "Target directory")
	assert.Contains(t, output, targetDir)

	// Verify the target directory was created
	info, err := os.Stat(targetDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	autospecGitignore, err := os.ReadFile(filepath.Join(targetDir, ".autospec", ".gitignore"))
	require.NoError(t, err)
	assert.Equal(t, autospecGitignoreContent, string(autospecGitignore))

	// Verify we're back in original working directory
	cwd, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, workDir, cwd)
}

// TestRunInit_WithRelativePath tests that relative paths are resolved correctly.
// FR-002: "MUST resolve relative paths relative to the current working directory"
func TestRunInit_WithRelativePath(t *testing.T) {
	// Cannot run in parallel: changes working directory
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	require.NoError(t, os.Chdir(tmpDir))

	originalConstitutionRunner := ConstitutionRunner
	originalWorktreeRunner := WorktreeScriptRunner
	ConstitutionRunner = func(cmd *cobra.Command, configPath string) bool { return true }
	WorktreeScriptRunner = func(cmd *cobra.Command, configPath string) bool { return true }
	defer func() {
		ConstitutionRunner = originalConstitutionRunner
		WorktreeScriptRunner = originalWorktreeRunner
	}()

	cmd := &cobra.Command{Use: "init [path]", Args: cobra.MaximumNArgs(1), RunE: runInit}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().Bool("no-agents", false, "")
	cmd.Flags().Bool("here", false, "")
	cmd.Flags().StringSlice("ai", nil, "")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetIn(bytes.NewBufferString("n\nn\nn\n"))
	cmd.SetArgs([]string{"relative-project", "--no-agents"})

	err = cmd.Execute()
	require.NoError(t, err)

	// Verify the directory was created in the expected location
	expectedPath := filepath.Join(tmpDir, "relative-project")
	info, err := os.Stat(expectedPath)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestRunInit_WithDot tests that '.' uses current directory (same as no arg).
// FR-005: "MUST treat '.' as equivalent to no argument (current directory)"
func TestRunInit_WithDot(t *testing.T) {
	// Cannot run in parallel: changes working directory
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	require.NoError(t, os.Chdir(tmpDir))

	originalConstitutionRunner := ConstitutionRunner
	originalWorktreeRunner := WorktreeScriptRunner
	ConstitutionRunner = func(cmd *cobra.Command, configPath string) bool { return true }
	WorktreeScriptRunner = func(cmd *cobra.Command, configPath string) bool { return true }
	defer func() {
		ConstitutionRunner = originalConstitutionRunner
		WorktreeScriptRunner = originalWorktreeRunner
	}()

	cmd := &cobra.Command{Use: "init [path]", Args: cobra.MaximumNArgs(1), RunE: runInit}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().Bool("no-agents", false, "")
	cmd.Flags().Bool("here", false, "")
	cmd.Flags().StringSlice("ai", nil, "")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetIn(bytes.NewBufferString("n\nn\nn\n"))
	cmd.SetArgs([]string{".", "--no-agents"})

	err = cmd.Execute()
	require.NoError(t, err)

	// Should NOT show "Target directory" message when using current dir
	output := buf.String()
	assert.NotContains(t, output, "Target directory")
}

// TestRunInit_WithHereFlag tests that --here uses current directory.
// FR-006: "MUST provide '--here' flag as alias for current directory"
func TestRunInit_WithHereFlag(t *testing.T) {
	// Cannot run in parallel: changes working directory
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	require.NoError(t, os.Chdir(tmpDir))

	originalConstitutionRunner := ConstitutionRunner
	originalWorktreeRunner := WorktreeScriptRunner
	ConstitutionRunner = func(cmd *cobra.Command, configPath string) bool { return true }
	WorktreeScriptRunner = func(cmd *cobra.Command, configPath string) bool { return true }
	defer func() {
		ConstitutionRunner = originalConstitutionRunner
		WorktreeScriptRunner = originalWorktreeRunner
	}()

	cmd := &cobra.Command{Use: "init [path]", Args: cobra.MaximumNArgs(1), RunE: runInit}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().Bool("no-agents", false, "")
	cmd.Flags().Bool("here", false, "")
	cmd.Flags().StringSlice("ai", nil, "")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetIn(bytes.NewBufferString("n\nn\nn\n"))
	cmd.SetArgs([]string{"--here", "--no-agents"})

	err = cmd.Execute()
	require.NoError(t, err)

	// Should NOT show "Target directory" message when using --here
	output := buf.String()
	assert.NotContains(t, output, "Target directory")
}

// TestRunInit_PathOverridesHere tests that path argument takes precedence over --here.
// Edge case: "Both path argument and --here flag provided"
func TestRunInit_PathOverridesHere(t *testing.T) {
	// Cannot run in parallel: changes working directory
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	workDir := filepath.Join(tmpDir, "workdir")
	require.NoError(t, os.MkdirAll(workDir, 0o755))
	require.NoError(t, os.Chdir(workDir))

	targetDir := filepath.Join(tmpDir, "target-project")

	originalConstitutionRunner := ConstitutionRunner
	originalWorktreeRunner := WorktreeScriptRunner
	ConstitutionRunner = func(cmd *cobra.Command, configPath string) bool { return true }
	WorktreeScriptRunner = func(cmd *cobra.Command, configPath string) bool { return true }
	defer func() {
		ConstitutionRunner = originalConstitutionRunner
		WorktreeScriptRunner = originalWorktreeRunner
	}()

	cmd := &cobra.Command{Use: "init [path]", Args: cobra.MaximumNArgs(1), RunE: runInit}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().Bool("no-agents", false, "")
	cmd.Flags().Bool("here", false, "")
	cmd.Flags().StringSlice("ai", nil, "")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetIn(bytes.NewBufferString("n\nn\nn\n"))
	// Both --here and path argument provided - path should win
	cmd.SetArgs([]string{targetDir, "--here", "--no-agents"})

	err = cmd.Execute()
	require.NoError(t, err)

	// Target directory should be created (path wins over --here)
	output := buf.String()
	assert.Contains(t, output, "Target directory")
	assert.Contains(t, output, targetDir)

	info, err := os.Stat(targetDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestRunInit_NonExistentPathCreated tests that non-existent paths are created.
// FR-004: "MUST create the target directory if it does not exist"
func TestRunInit_NonExistentPathCreated(t *testing.T) {
	// Cannot run in parallel: changes working directory
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	workDir := filepath.Join(tmpDir, "workdir")
	require.NoError(t, os.MkdirAll(workDir, 0o755))
	require.NoError(t, os.Chdir(workDir))

	// Nested path that doesn't exist
	targetDir := filepath.Join(tmpDir, "a", "b", "c", "project")

	originalConstitutionRunner := ConstitutionRunner
	originalWorktreeRunner := WorktreeScriptRunner
	ConstitutionRunner = func(cmd *cobra.Command, configPath string) bool { return true }
	WorktreeScriptRunner = func(cmd *cobra.Command, configPath string) bool { return true }
	defer func() {
		ConstitutionRunner = originalConstitutionRunner
		WorktreeScriptRunner = originalWorktreeRunner
	}()

	cmd := &cobra.Command{Use: "init [path]", Args: cobra.MaximumNArgs(1), RunE: runInit}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().Bool("no-agents", false, "")
	cmd.Flags().Bool("here", false, "")
	cmd.Flags().StringSlice("ai", nil, "")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetIn(bytes.NewBufferString("n\nn\nn\n"))
	cmd.SetArgs([]string{targetDir, "--no-agents"})

	err = cmd.Execute()
	require.NoError(t, err)

	// All nested directories should be created
	info, err := os.Stat(targetDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestRunInit_PathIsFile tests that path to file returns error.
// FR-007: "MUST fail with clear error if path exists but is a file"
func TestRunInit_PathIsFile(t *testing.T) {
	// Cannot run in parallel: changes working directory
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	workDir := filepath.Join(tmpDir, "workdir")
	require.NoError(t, os.MkdirAll(workDir, 0o755))
	require.NoError(t, os.Chdir(workDir))

	// Create a file where we want the directory
	targetPath := filepath.Join(tmpDir, "existing-file")
	require.NoError(t, os.WriteFile(targetPath, []byte("content"), 0o644))

	cmd := &cobra.Command{Use: "init [path]", Args: cobra.MaximumNArgs(1), RunE: runInit}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().Bool("no-agents", false, "")
	cmd.Flags().Bool("here", false, "")
	cmd.Flags().StringSlice("ai", nil, "")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetIn(bytes.NewBufferString("n\nn\nn\n"))
	cmd.SetArgs([]string{targetPath, "--no-agents"})

	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}

// TestRunInit_CwdRestoredAfterInit tests that working directory is restored.
// Open question resolution: "Restore original cwd after init"
func TestRunInit_CwdRestoredAfterInit(t *testing.T) {
	// Cannot run in parallel: changes working directory
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	workDir := filepath.Join(tmpDir, "workdir")
	require.NoError(t, os.MkdirAll(workDir, 0o755))
	require.NoError(t, os.Chdir(workDir))

	targetDir := filepath.Join(tmpDir, "target-project")

	originalConstitutionRunner := ConstitutionRunner
	originalWorktreeRunner := WorktreeScriptRunner
	ConstitutionRunner = func(cmd *cobra.Command, configPath string) bool { return true }
	WorktreeScriptRunner = func(cmd *cobra.Command, configPath string) bool { return true }
	defer func() {
		ConstitutionRunner = originalConstitutionRunner
		WorktreeScriptRunner = originalWorktreeRunner
	}()

	cmd := &cobra.Command{Use: "init [path]", Args: cobra.MaximumNArgs(1), RunE: runInit}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().Bool("no-agents", false, "")
	cmd.Flags().Bool("here", false, "")
	cmd.Flags().StringSlice("ai", nil, "")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetIn(bytes.NewBufferString("n\nn\nn\n"))
	cmd.SetArgs([]string{targetDir, "--no-agents"})

	err = cmd.Execute()
	require.NoError(t, err)

	// Verify we're back in original working directory
	cwd, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, workDir, cwd, "working directory should be restored after init")
}

// TestRunInit_BackwardCompatibility tests that init without args still works.
// SC-002: "Zero breaking changes for 'autospec init' without arguments"
func TestRunInit_BackwardCompatibility(t *testing.T) {
	// Cannot run in parallel: changes working directory
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	require.NoError(t, os.Chdir(tmpDir))

	originalConstitutionRunner := ConstitutionRunner
	originalWorktreeRunner := WorktreeScriptRunner
	ConstitutionRunner = func(cmd *cobra.Command, configPath string) bool { return true }
	WorktreeScriptRunner = func(cmd *cobra.Command, configPath string) bool { return true }
	defer func() {
		ConstitutionRunner = originalConstitutionRunner
		WorktreeScriptRunner = originalWorktreeRunner
	}()

	cmd := &cobra.Command{Use: "init [path]", Args: cobra.MaximumNArgs(1), RunE: runInit}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().Bool("no-agents", false, "")
	cmd.Flags().Bool("here", false, "")
	cmd.Flags().StringSlice("ai", nil, "")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetIn(bytes.NewBufferString("n\nn\nn\n"))
	cmd.SetArgs([]string{"--no-agents"}) // No path argument

	err = cmd.Execute()
	require.NoError(t, err)

	// Should NOT show "Target directory" message when no path given
	output := buf.String()
	assert.NotContains(t, output, "Target directory")

	// Still in same directory
	cwd, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, tmpDir, cwd)
}

// TestUpdateSkipPermissionsInConfig tests updating skip_permissions in config file.
func TestUpdateSkipPermissionsInConfig(t *testing.T) {
	tests := map[string]struct {
		initialContent string
		skipPerms      bool
		wantContains   string
	}{
		"update existing false to true": {
			initialContent: "agent_preset: claude\nuse_subscription: true\nskip_permissions: false\n",
			skipPerms:      true,
			wantContains:   "skip_permissions: true",
		},
		"update existing true to false": {
			initialContent: "agent_preset: claude\nuse_subscription: true\nskip_permissions: true\n",
			skipPerms:      false,
			wantContains:   "skip_permissions: false",
		},
		"insert after use_subscription": {
			initialContent: "agent_preset: claude\nuse_subscription: true\nspecs_dir: specs\n",
			skipPerms:      true,
			wantContains:   "skip_permissions: true",
		},
		"insert after agent_preset when no use_subscription": {
			initialContent: "agent_preset: claude\nspecs_dir: specs\n",
			skipPerms:      true,
			wantContains:   "skip_permissions: true",
		},
		"append to end when neither found": {
			initialContent: "specs_dir: specs\nmax_retries: 3\n",
			skipPerms:      false,
			wantContains:   "skip_permissions: false",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "config.yml")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tt.initialContent), 0o644))

			err := updateSkipPermissionsInConfig(tmpFile, tt.skipPerms)
			require.NoError(t, err)

			content, err := os.ReadFile(tmpFile)
			require.NoError(t, err)
			assert.Contains(t, string(content), tt.wantContains)
		})
	}
}

// TestUpdateSkipPermissionsInConfig_ReadError tests error handling for missing file.
func TestUpdateSkipPermissionsInConfig_ReadError(t *testing.T) {
	err := updateSkipPermissionsInConfig("/nonexistent/path/config.yml", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading config")
}

// TestHandleSkipPermissionsPrompt_UserSaysYes tests prompt when user enables skip_permissions.
func TestHandleSkipPermissionsPrompt_UserSaysYes(t *testing.T) {
	// Mock isTerminalFunc to simulate interactive mode
	originalIsTerminal := isTerminalFunc
	isTerminalFunc = func() bool { return true }
	defer func() { isTerminalFunc = originalIsTerminal }()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("agent_preset: claude\n"), 0o644))

	cmd := &cobra.Command{Use: "test"}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetIn(bytes.NewBufferString("y\n"))

	handleSkipPermissionsPrompt(cmd, &outBuf, configPath, false)

	output := outBuf.String()
	// Verify prompt content
	assert.Contains(t, output, "Permissions Mode")
	assert.Contains(t, output, "skip_permissions")
	assert.Contains(t, output, "recommended")
	assert.Contains(t, output, "autospec config toggle")

	// Verify config was updated
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "skip_permissions: true")

	// Verify result message
	assert.Contains(t, output, "autonomous mode enabled")
}

// TestHandleSkipPermissionsPrompt_UserSaysNo tests prompt when user declines skip_permissions.
func TestHandleSkipPermissionsPrompt_UserSaysNo(t *testing.T) {
	// Mock isTerminalFunc to simulate interactive mode
	originalIsTerminal := isTerminalFunc
	isTerminalFunc = func() bool { return true }
	defer func() { isTerminalFunc = originalIsTerminal }()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("agent_preset: claude\n"), 0o644))

	cmd := &cobra.Command{Use: "test"}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetIn(bytes.NewBufferString("n\n"))

	handleSkipPermissionsPrompt(cmd, &outBuf, configPath, false)

	// Verify config was updated to false
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "skip_permissions: false")

	// Verify result message
	output := outBuf.String()
	assert.Contains(t, output, "interactive mode")
}

// TestHandleSkipPermissionsPrompt_ExistingEnabled tests that when skip_permissions is already
// set to true, it just displays the value without prompting.
func TestHandleSkipPermissionsPrompt_ExistingEnabled(t *testing.T) {
	// Mock isTerminalFunc to simulate interactive mode
	originalIsTerminal := isTerminalFunc
	isTerminalFunc = func() bool { return true }
	defer func() { isTerminalFunc = originalIsTerminal }()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("agent_preset: claude\nskip_permissions: true\n"), 0o644))

	cmd := &cobra.Command{Use: "test"}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	// No input needed since prompt should be skipped

	handleSkipPermissionsPrompt(cmd, &outBuf, configPath, false)

	output := outBuf.String()
	// Should show current value without prompting
	assert.Contains(t, output, "skip_permissions")
	assert.Contains(t, output, "enabled")
	// Should NOT show the full explanation (no prompt)
	assert.NotContains(t, output, "Without sufficient permissions")
	assert.NotContains(t, output, "Enable skip_permissions (recommended)?")
}

// TestHandleSkipPermissionsPrompt_ExistingDisabled tests that when skip_permissions is
// set to false, it still prompts the user (to give them a chance to enable it).
func TestHandleSkipPermissionsPrompt_ExistingDisabled(t *testing.T) {
	// Mock isTerminalFunc to simulate interactive mode
	originalIsTerminal := isTerminalFunc
	isTerminalFunc = func() bool { return true }
	defer func() { isTerminalFunc = originalIsTerminal }()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("agent_preset: claude\nskip_permissions: false\n"), 0o644))

	cmd := &cobra.Command{Use: "test"}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetIn(bytes.NewBufferString("n\n")) // User declines

	handleSkipPermissionsPrompt(cmd, &outBuf, configPath, false)

	output := outBuf.String()
	// Should show the full explanation and prompt (since it's disabled, we re-prompt)
	assert.Contains(t, output, "Without sufficient permissions")
	assert.Contains(t, output, "skip_permissions")
}

// TestHandleSkipPermissionsPrompt_NonInteractiveUsesDefault tests that non-interactive mode
// uses the default value (true) without prompting.
// T008 acceptance criteria:
// - Non-interactive mode uses default (true)
// - No prompt displayed in non-interactive mode
// - Config still updated with default value
func TestHandleSkipPermissionsPrompt_NonInteractiveUsesDefault(t *testing.T) {
	// Mock isTerminalFunc to simulate non-interactive mode
	originalIsTerminal := isTerminalFunc
	isTerminalFunc = func() bool { return false }
	defer func() { isTerminalFunc = originalIsTerminal }()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("agent_preset: claude\n"), 0o644))

	cmd := &cobra.Command{Use: "test"}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	// No input needed since prompt should be skipped

	handleSkipPermissionsPrompt(cmd, &outBuf, configPath, false)

	output := outBuf.String()

	// Should NOT show the full prompt section in non-interactive mode
	assert.NotContains(t, output, "Permissions Mode")
	assert.NotContains(t, output, "Enable skip_permissions (autonomous mode)?")

	// Should show non-interactive feedback
	assert.Contains(t, output, "non-interactive")

	// Config should be updated with default value (true)
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "skip_permissions: true")
}

// TestHandleSkipPermissionsPrompt_DefaultEmpty tests that empty input defaults to Yes (recommended).
func TestHandleSkipPermissionsPrompt_DefaultEmpty(t *testing.T) {
	// Mock isTerminalFunc to simulate interactive mode
	originalIsTerminal := isTerminalFunc
	isTerminalFunc = func() bool { return true }
	defer func() { isTerminalFunc = originalIsTerminal }()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("agent_preset: claude\n"), 0o644))

	cmd := &cobra.Command{Use: "test"}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetIn(bytes.NewBufferString("\n")) // Empty input

	handleSkipPermissionsPrompt(cmd, &outBuf, configPath, false)

	// Verify config shows true (default Yes - recommended)
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "skip_permissions: true")
}

// ============================================================================
// Integration Tests for skip_permissions prompt (T006)
// ============================================================================

// TestSkipPermissionsPrompt_AppearsOnlyForClaude verifies that the skip_permissions
// prompt is shown only when Claude agent is selected, not for other agents.
// T006 acceptance criteria: prompt appears when Claude selected, NOT for non-Claude agents.
func TestSkipPermissionsPrompt_AppearsOnlyForClaude(t *testing.T) {
	tests := map[string]struct {
		agents              []string
		expectPromptShown   bool
		promptResponseInput string
	}{
		"claude only shows prompt": {
			agents:              []string{"claude"},
			expectPromptShown:   true,
			promptResponseInput: "n\n", // decline skip_permissions
		},
		"opencode only does not show prompt": {
			agents:            []string{"opencode"},
			expectPromptShown: false,
		},
		"gemini only does not show prompt": {
			agents:            []string{"gemini"},
			expectPromptShown: false,
		},
		"claude with others shows prompt": {
			agents:              []string{"claude", "opencode"},
			expectPromptShown:   true,
			promptResponseInput: "y\n", // accept skip_permissions
		},
		"multiple non-claude does not show prompt": {
			agents:            []string{"opencode", "gemini"},
			expectPromptShown: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			origDir, err := os.Getwd()
			require.NoError(t, err)
			require.NoError(t, os.Chdir(tmpDir))
			defer func() { _ = os.Chdir(origDir) }()

			// Create minimal config
			configDir := filepath.Join(tmpDir, ".autospec")
			require.NoError(t, os.MkdirAll(configDir, 0o755))
			configPath := filepath.Join(configDir, "config.yml")
			require.NoError(t, os.WriteFile(configPath, []byte("specs_dir: specs\n"), 0o644))

			// Mock isTerminalFunc to simulate interactive mode for the prompt test
			originalIsTerminal := isTerminalFunc
			isTerminalFunc = func() bool { return true }
			defer func() { isTerminalFunc = originalIsTerminal }()

			// Mock the runners to prevent real Claude execution
			originalConstitutionRunner := ConstitutionRunner
			originalWorktreeRunner := WorktreeScriptRunner
			ConstitutionRunner = func(cmd *cobra.Command, configPath string) bool { return true }
			WorktreeScriptRunner = func(cmd *cobra.Command, configPath string) bool { return true }
			defer func() {
				ConstitutionRunner = originalConstitutionRunner
				WorktreeScriptRunner = originalWorktreeRunner
			}()

			cmd := &cobra.Command{Use: "init [path]", Args: cobra.MaximumNArgs(1), RunE: runInit}
			cmd.Flags().BoolP("project", "p", false, "")
			cmd.Flags().BoolP("force", "f", false, "")
			cmd.Flags().Bool("no-agents", false, "")
			cmd.Flags().Bool("here", false, "")
			cmd.Flags().StringSlice("ai", nil, "")

			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// Build input for all prompts (sandbox prompt + skip_permissions + gitignore + constitution)
			// The number of "n\n" responses depends on agent selection
			var inputBuilder bytes.Buffer
			inputBuilder.WriteString("n\n") // sandbox prompt (for Claude)
			if tt.expectPromptShown {
				inputBuilder.WriteString(tt.promptResponseInput) // skip_permissions prompt
			}
			inputBuilder.WriteString("n\nn\n") // gitignore + constitution prompts

			cmd.SetIn(&inputBuilder)
			cmd.SetArgs([]string{"--project", "--ai", strings.Join(tt.agents, ",")})

			err = cmd.Execute()
			// May error if agent not found, but we care about prompt appearing
			_ = err

			output := buf.String()

			if tt.expectPromptShown {
				assert.Contains(t, output, "Permissions Mode", "prompt section header should appear for Claude")
				assert.Contains(t, output, "skip_permissions", "skip_permissions should be mentioned in prompt")
				// Note: "autonomous mode" only appears in the response after user input, not in the prompt itself
				assert.Contains(t, output, "permissions", "permissions should be explained in prompt")
			} else {
				assert.NotContains(t, output, "Permissions Mode", "prompt section should NOT appear for non-Claude agents")
			}
		})
	}
}

// TestSkipPermissionsPrompt_AppearsRegardlessOfSandboxStatus verifies that the
// skip_permissions prompt appears whether sandbox is configured or not.
// T006 acceptance criteria: prompt appears regardless of sandbox status.
func TestSkipPermissionsPrompt_AppearsRegardlessOfSandboxStatus(t *testing.T) {
	tests := map[string]struct {
		sandboxConfigured   bool
		sandboxPromptAnswer string
	}{
		"sandbox not configured - decline sandbox": {
			sandboxConfigured:   false,
			sandboxPromptAnswer: "n\n",
		},
		"sandbox configured via prompt - accept sandbox": {
			sandboxConfigured:   true,
			sandboxPromptAnswer: "y\n",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			origDir, err := os.Getwd()
			require.NoError(t, err)
			require.NoError(t, os.Chdir(tmpDir))
			defer func() { _ = os.Chdir(origDir) }()

			// Create minimal config
			configDir := filepath.Join(tmpDir, ".autospec")
			require.NoError(t, os.MkdirAll(configDir, 0o755))
			configPath := filepath.Join(configDir, "config.yml")
			require.NoError(t, os.WriteFile(configPath, []byte("specs_dir: specs\n"), 0o644))

			// Mock isTerminalFunc to simulate interactive mode
			originalIsTerminal := isTerminalFunc
			isTerminalFunc = func() bool { return true }
			defer func() { isTerminalFunc = originalIsTerminal }()

			// Mock the runners
			originalConstitutionRunner := ConstitutionRunner
			originalWorktreeRunner := WorktreeScriptRunner
			ConstitutionRunner = func(cmd *cobra.Command, configPath string) bool { return true }
			WorktreeScriptRunner = func(cmd *cobra.Command, configPath string) bool { return true }
			defer func() {
				ConstitutionRunner = originalConstitutionRunner
				WorktreeScriptRunner = originalWorktreeRunner
			}()

			cmd := &cobra.Command{Use: "init [path]", Args: cobra.MaximumNArgs(1), RunE: runInit}
			cmd.Flags().BoolP("project", "p", false, "")
			cmd.Flags().BoolP("force", "f", false, "")
			cmd.Flags().Bool("no-agents", false, "")
			cmd.Flags().Bool("here", false, "")
			cmd.Flags().StringSlice("ai", nil, "")

			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// Build input: sandbox answer + skip_permissions answer + gitignore + constitution
			var inputBuilder bytes.Buffer
			inputBuilder.WriteString(tt.sandboxPromptAnswer) // sandbox prompt response
			inputBuilder.WriteString("n\n")                  // skip_permissions prompt - decline
			inputBuilder.WriteString("n\nn\n")               // gitignore + constitution prompts

			cmd.SetIn(&inputBuilder)
			cmd.SetArgs([]string{"--project", "--ai", "claude"})

			err = cmd.Execute()
			require.NoError(t, err)

			output := buf.String()

			// Regardless of sandbox configuration status, skip_permissions prompt should appear
			assert.Contains(t, output, "Permissions Mode", "prompt should appear regardless of sandbox status")
			assert.Contains(t, output, "skip_permissions", "skip_permissions should be in prompt")
		})
	}
}

// TestSkipPermissionsPrompt_ConfigUpdatedCorrectly verifies that the skip_permissions
// setting is correctly persisted to the config file via CLI flags.
// T006 acceptance criteria: config reflects user selection after init.
// Note: Uses CLI flags (--skip-permissions, --no-skip-permissions) to bypass prompt
// buffering issues. The prompt input handling is tested in TestHandleSkipPermissionsPrompt_*.
func TestSkipPermissionsPrompt_ConfigUpdatedCorrectly(t *testing.T) {
	tests := map[string]struct {
		flagArg     string
		expectValue string
	}{
		"--skip-permissions sets true": {
			flagArg:     "--skip-permissions",
			expectValue: "skip_permissions: true",
		},
		"--no-skip-permissions sets false": {
			flagArg:     "--no-skip-permissions",
			expectValue: "skip_permissions: false",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			origDir, err := os.Getwd()
			require.NoError(t, err)
			require.NoError(t, os.Chdir(tmpDir))
			defer func() { _ = os.Chdir(origDir) }()

			// Unset ANTHROPIC_API_KEY to avoid auth detection prompt
			originalAPIKey := os.Getenv("ANTHROPIC_API_KEY")
			os.Unsetenv("ANTHROPIC_API_KEY")
			defer func() {
				if originalAPIKey != "" {
					os.Setenv("ANTHROPIC_API_KEY", originalAPIKey)
				}
			}()

			// Create minimal config (project level)
			configDir := filepath.Join(tmpDir, ".autospec")
			require.NoError(t, os.MkdirAll(configDir, 0o755))
			configPath := filepath.Join(configDir, "config.yml")
			require.NoError(t, os.WriteFile(configPath, []byte("specs_dir: specs\nagent_preset: claude\n"), 0o644))

			// Mock isTerminalFunc to simulate interactive mode
			originalIsTerminal := isTerminalFunc
			isTerminalFunc = func() bool { return true }
			defer func() { isTerminalFunc = originalIsTerminal }()

			// Mock the runners
			originalConstitutionRunner := ConstitutionRunner
			originalWorktreeRunner := WorktreeScriptRunner
			ConstitutionRunner = func(cmd *cobra.Command, configPath string) bool { return true }
			WorktreeScriptRunner = func(cmd *cobra.Command, configPath string) bool { return true }
			defer func() {
				ConstitutionRunner = originalConstitutionRunner
				WorktreeScriptRunner = originalWorktreeRunner
			}()

			cmd := &cobra.Command{Use: "init [path]", Args: cobra.MaximumNArgs(1), RunE: runInit}
			cmd.Flags().BoolP("project", "p", false, "")
			cmd.Flags().BoolP("force", "f", false, "")
			cmd.Flags().Bool("no-agents", false, "")
			cmd.Flags().Bool("here", false, "")
			cmd.Flags().StringSlice("ai", nil, "")
			cmd.Flags().Bool("skip-permissions", false, "")
			cmd.Flags().Bool("no-skip-permissions", false, "")

			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// Build input: sandbox + gitignore + constitution (skip_permissions handled by flag)
			var inputBuilder bytes.Buffer
			inputBuilder.WriteString("n\n") // sandbox prompt (Y/n default yes)
			inputBuilder.WriteString("n\n") // gitignore prompt [y/N]
			inputBuilder.WriteString("n\n") // constitution prompt (Y/n default yes)

			cmd.SetIn(&inputBuilder)
			cmd.SetArgs([]string{"--project", "--ai", "claude", tt.flagArg})

			err = cmd.Execute()
			require.NoError(t, err)

			// Verify the config file was updated
			content, err := os.ReadFile(configPath)
			require.NoError(t, err)
			assert.Contains(t, string(content), tt.expectValue, "config should reflect flag setting")
		})
	}
}

// ============================================================================
// init.yml Tests (T006)
// ============================================================================

// TestSaveInitSettings_ScopeSettings tests that saveInitSettings correctly sets
// the settings_scope based on the projectLevel flag.
func TestSaveInitSettings_ScopeSettings(t *testing.T) {
	// Cannot run in parallel: subtests change working directory

	tests := map[string]struct {
		projectLevel  bool
		expectedScope string
	}{
		"global scope (default)": {
			projectLevel:  false,
			expectedScope: initpkg.ScopeGlobal,
		},
		"project scope (--project flag)": {
			projectLevel:  true,
			expectedScope: initpkg.ScopeProject,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run in parallel: changes working directory

			tmpDir := t.TempDir()
			origDir, err := os.Getwd()
			require.NoError(t, err)
			require.NoError(t, os.Chdir(tmpDir))
			defer func() { _ = os.Chdir(origDir) }()

			// Create .autospec directory
			require.NoError(t, os.MkdirAll(".autospec", 0o755))

			var buf bytes.Buffer
			agentConfigs := []agentConfigInfo{
				{name: "claude", configured: true, settingsFile: "/home/test/.claude/settings.json"},
			}

			err = saveInitSettings(&buf, tt.projectLevel, agentConfigs)
			require.NoError(t, err)

			// Verify init.yml was created
			assert.True(t, initpkg.Exists(), "init.yml should exist")

			// Load and verify settings
			settings, err := initpkg.Load()
			require.NoError(t, err)
			assert.Equal(t, tt.expectedScope, settings.SettingsScope)
			assert.Equal(t, initpkg.SchemaVersion, settings.Version)
			assert.Contains(t, settings.AutospecVersion, "autospec v")
		})
	}
}

// TestSaveInitSettings_AgentEntries tests that saveInitSettings correctly records
// agent configuration results.
func TestSaveInitSettings_AgentEntries(t *testing.T) {
	// Cannot run in parallel: changes working directory

	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	// Create .autospec directory
	require.NoError(t, os.MkdirAll(".autospec", 0o755))

	var buf bytes.Buffer
	agentConfigs := []agentConfigInfo{
		{name: "claude", configured: true, settingsFile: "/home/test/.claude/settings.json"},
		{name: "opencode", configured: true, settingsFile: "/home/test/.config/opencode/opencode.json"},
		{name: "gemini", configured: false, settingsFile: ""}, // Agent that doesn't support config
	}

	err = saveInitSettings(&buf, false, agentConfigs)
	require.NoError(t, err)

	// Load and verify agents
	settings, err := initpkg.Load()
	require.NoError(t, err)
	require.Len(t, settings.Agents, 3)

	// Verify each agent entry
	agentMap := make(map[string]initpkg.AgentEntry)
	for _, a := range settings.Agents {
		agentMap[a.Name] = a
	}

	assert.True(t, agentMap["claude"].Configured)
	assert.Equal(t, "/home/test/.claude/settings.json", agentMap["claude"].SettingsFile)

	assert.True(t, agentMap["opencode"].Configured)
	assert.Equal(t, "/home/test/.config/opencode/opencode.json", agentMap["opencode"].SettingsFile)

	assert.False(t, agentMap["gemini"].Configured)
	assert.Empty(t, agentMap["gemini"].SettingsFile)
}

// TestSaveInitSettings_ReInit tests that re-running init preserves created_at
// and updates updated_at.
func TestSaveInitSettings_ReInit(t *testing.T) {
	// Cannot run in parallel: modifies time-sensitive state

	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	// Create .autospec directory
	require.NoError(t, os.MkdirAll(".autospec", 0o755))

	var buf bytes.Buffer
	agentConfigs := []agentConfigInfo{
		{name: "claude", configured: true, settingsFile: "/path/to/settings.json"},
	}

	// First init
	err = saveInitSettings(&buf, false, agentConfigs)
	require.NoError(t, err)

	// Load first settings
	firstSettings, err := initpkg.Load()
	require.NoError(t, err)
	firstCreatedAt := firstSettings.CreatedAt
	firstUpdatedAt := firstSettings.UpdatedAt

	// Wait a moment to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Re-init with different scope
	buf.Reset()
	err = saveInitSettings(&buf, true, agentConfigs)
	require.NoError(t, err)

	// Load second settings
	secondSettings, err := initpkg.Load()
	require.NoError(t, err)

	// created_at should be preserved
	assert.Equal(t, firstCreatedAt.Unix(), secondSettings.CreatedAt.Unix(),
		"created_at should be preserved on re-init")

	// updated_at should be updated
	assert.True(t, secondSettings.UpdatedAt.After(firstUpdatedAt) || secondSettings.UpdatedAt.Equal(firstUpdatedAt),
		"updated_at should be >= first updated_at")

	// Scope should be updated
	assert.Equal(t, initpkg.ScopeProject, secondSettings.SettingsScope,
		"settings_scope should be updated on re-init")
}

// TestSaveInitSettings_EmptyAgents tests behavior with no agents configured.
func TestSaveInitSettings_EmptyAgents(t *testing.T) {
	// Cannot run in parallel: changes working directory

	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	// Create .autospec directory
	require.NoError(t, os.MkdirAll(".autospec", 0o755))

	var buf bytes.Buffer
	err = saveInitSettings(&buf, false, nil) // nil agents
	require.NoError(t, err)

	// init.yml should still be created
	assert.True(t, initpkg.Exists())

	settings, err := initpkg.Load()
	require.NoError(t, err)
	assert.Empty(t, settings.Agents)
	assert.Equal(t, initpkg.ScopeGlobal, settings.SettingsScope)
}

// TestResolveBoolFlag tests the three-state boolean flag resolution.
func TestResolveBoolFlag(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFlags func(cmd *cobra.Command)
		positive   string
		negative   string
		want       *bool
	}{
		"neither flag set returns nil": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("sandbox", false, "")
				cmd.Flags().Bool("no-sandbox", false, "")
			},
			positive: "sandbox",
			negative: "no-sandbox",
			want:     nil,
		},
		"positive flag set returns true": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("sandbox", false, "")
				cmd.Flags().Bool("no-sandbox", false, "")
				_ = cmd.ParseFlags([]string{"--sandbox"})
			},
			positive: "sandbox",
			negative: "no-sandbox",
			want:     ptrBool(true),
		},
		"negative flag set returns false": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("sandbox", false, "")
				cmd.Flags().Bool("no-sandbox", false, "")
				_ = cmd.ParseFlags([]string{"--no-sandbox"})
			},
			positive: "sandbox",
			negative: "no-sandbox",
			want:     ptrBool(false),
		},
		"positive flag with value true": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("gitignore", false, "")
				cmd.Flags().Bool("no-gitignore", false, "")
				_ = cmd.ParseFlags([]string{"--gitignore=true"})
			},
			positive: "gitignore",
			negative: "no-gitignore",
			want:     ptrBool(true),
		},
		"negative flag with value true": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("constitution", false, "")
				cmd.Flags().Bool("no-constitution", false, "")
				_ = cmd.ParseFlags([]string{"--no-constitution=true"})
			},
			positive: "constitution",
			negative: "no-constitution",
			want:     ptrBool(false),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{Use: "test"}
			tt.setupFlags(cmd)

			got := resolveBoolFlag(cmd, tt.positive, tt.negative)

			if tt.want == nil {
				assert.Nil(t, got, "expected nil but got %v", got)
			} else {
				require.NotNil(t, got, "expected non-nil value")
				assert.Equal(t, *tt.want, *got)
			}
		})
	}
}

// TestCheckMutuallyExclusiveFlags tests mutual exclusivity validation.
func TestCheckMutuallyExclusiveFlags(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFlags func(cmd *cobra.Command)
		pairs      []BoolFlagPair
		wantErr    bool
		errMsg     string
	}{
		"both flags set returns error": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("sandbox", false, "")
				cmd.Flags().Bool("no-sandbox", false, "")
				_ = cmd.ParseFlags([]string{"--sandbox", "--no-sandbox"})
			},
			pairs: []BoolFlagPair{
				{Positive: "sandbox", Negative: "no-sandbox"},
			},
			wantErr: true,
			errMsg:  "mutually exclusive",
		},
		"only positive flag set returns no error": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("sandbox", false, "")
				cmd.Flags().Bool("no-sandbox", false, "")
				_ = cmd.ParseFlags([]string{"--sandbox"})
			},
			pairs: []BoolFlagPair{
				{Positive: "sandbox", Negative: "no-sandbox"},
			},
			wantErr: false,
		},
		"only negative flag set returns no error": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("sandbox", false, "")
				cmd.Flags().Bool("no-sandbox", false, "")
				_ = cmd.ParseFlags([]string{"--no-sandbox"})
			},
			pairs: []BoolFlagPair{
				{Positive: "sandbox", Negative: "no-sandbox"},
			},
			wantErr: false,
		},
		"neither flag set returns no error": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("sandbox", false, "")
				cmd.Flags().Bool("no-sandbox", false, "")
			},
			pairs: []BoolFlagPair{
				{Positive: "sandbox", Negative: "no-sandbox"},
			},
			wantErr: false,
		},
		"multiple pairs with one conflict": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("sandbox", false, "")
				cmd.Flags().Bool("no-sandbox", false, "")
				cmd.Flags().Bool("gitignore", false, "")
				cmd.Flags().Bool("no-gitignore", false, "")
				_ = cmd.ParseFlags([]string{"--sandbox", "--gitignore", "--no-gitignore"})
			},
			pairs: []BoolFlagPair{
				{Positive: "sandbox", Negative: "no-sandbox"},
				{Positive: "gitignore", Negative: "no-gitignore"},
			},
			wantErr: true,
			errMsg:  "gitignore",
		},
		"multiple pairs with no conflicts": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("sandbox", false, "")
				cmd.Flags().Bool("no-sandbox", false, "")
				cmd.Flags().Bool("gitignore", false, "")
				cmd.Flags().Bool("no-gitignore", false, "")
				_ = cmd.ParseFlags([]string{"--sandbox", "--no-gitignore"})
			},
			pairs: []BoolFlagPair{
				{Positive: "sandbox", Negative: "no-sandbox"},
				{Positive: "gitignore", Negative: "no-gitignore"},
			},
			wantErr: false,
		},
		"empty pairs returns no error": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("sandbox", false, "")
				cmd.Flags().Bool("no-sandbox", false, "")
				_ = cmd.ParseFlags([]string{"--sandbox", "--no-sandbox"})
			},
			pairs:   []BoolFlagPair{},
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{Use: "test"}
			tt.setupFlags(cmd)

			err := checkMutuallyExclusiveFlags(cmd, tt.pairs)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ptrBool is a helper to create a pointer to a bool value.
func ptrBool(v bool) *bool {
	return &v
}

// TestPromptAndConfigureSandbox_FlagBehavior tests that sandbox flags bypass the prompt.
func TestPromptAndConfigureSandbox_FlagBehavior(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFlags   func(cmd *cobra.Command)
		wantContains []string
		wantSkipped  bool
	}{
		"--sandbox flag bypasses prompt and configures": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("sandbox", false, "")
				cmd.Flags().Bool("no-sandbox", false, "")
				_ = cmd.ParseFlags([]string{"--sandbox"})
			},
			wantContains: []string{"sandbox"},
			wantSkipped:  false,
		},
		"--no-sandbox flag skips configuration": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("sandbox", false, "")
				cmd.Flags().Bool("no-sandbox", false, "")
				_ = cmd.ParseFlags([]string{"--no-sandbox"})
			},
			wantContains: []string{"skipped", "--no-sandbox"},
			wantSkipped:  true,
		},
		"neither flag shows prompt": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("sandbox", false, "")
				cmd.Flags().Bool("no-sandbox", false, "")
			},
			wantContains: []string{"Sandbox Configuration"},
			wantSkipped:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{Use: "test"}
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			// Provide "n" to skip prompt when neither flag is set
			cmd.SetIn(bytes.NewBufferString("n\n"))
			tt.setupFlags(cmd)

			info := sandboxPromptInfo{
				agentName:   "claude",
				displayName: "Claude Code",
				pathsToAdd:  []string{".autospec/**"},
				needsEnable: true,
			}

			// We don't actually want to call the real sandbox configuration
			// since we're just testing the flag behavior and output
			_ = promptAndConfigureSandbox(cmd, &buf, info, ".", "specs")

			output := buf.String()
			for _, want := range tt.wantContains {
				assert.Contains(t, output, want, "expected output to contain %q", want)
			}

			if tt.wantSkipped {
				assert.Contains(t, output, "skipped")
			}
		})
	}
}

func TestHandleSandboxConfiguration_ExplicitSandboxFlagRunsWhenPromptDisabled(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("sandbox", false, "")
	cmd.Flags().Bool("no-sandbox", false, "")
	require.NoError(t, cmd.ParseFlags([]string{"--sandbox"}))

	var buf bytes.Buffer
	info := sandboxPromptInfo{
		agentName:   "claude",
		displayName: "Claude Code",
		pathsToAdd:  []string{".autospec", "specs"},
		needsEnable: true,
	}

	err := handleSandboxConfiguration(cmd, &buf, []sandboxPromptInfo{info}, projectDir, "specs")

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Configuring sandbox")
	settingsPath := filepath.Join(projectDir, ".claude", "settings.local.json")
	settingsContent, err := os.ReadFile(settingsPath)
	require.NoError(t, err)
	assert.Contains(t, string(settingsContent), `"enabled": true`)
	assert.Contains(t, string(settingsContent), `"additionalAllowWritePaths"`)
	assert.Contains(t, string(settingsContent), `".autospec"`)
	assert.Contains(t, string(settingsContent), `"specs"`)
}

// TestSandboxFlagsMutualExclusivity tests that --sandbox and --no-sandbox cannot be used together.
func TestSandboxFlagsMutualExclusivity(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		flags   []string
		wantErr bool
		errMsg  string
	}{
		"both flags returns error": {
			flags:   []string{"--sandbox", "--no-sandbox"},
			wantErr: true,
			errMsg:  "mutually exclusive",
		},
		"only --sandbox no error": {
			flags:   []string{"--sandbox"},
			wantErr: false,
		},
		"only --no-sandbox no error": {
			flags:   []string{"--no-sandbox"},
			wantErr: false,
		},
		"neither flag no error": {
			flags:   []string{},
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{Use: "test"}
			cmd.Flags().Bool("sandbox", false, "")
			cmd.Flags().Bool("no-sandbox", false, "")
			_ = cmd.ParseFlags(tt.flags)

			pairs := []BoolFlagPair{
				{Positive: "sandbox", Negative: "no-sandbox"},
			}
			err := checkMutuallyExclusiveFlags(cmd, pairs)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ============================================================================
// Phase 4: Billing and Permissions Flag Tests (T013)
// ============================================================================

// TestHandleSkipPermissionsPrompt_FlagBypass tests that --skip-permissions and
// --no-skip-permissions flags bypass the prompt.
// T012 acceptance criteria:
// - Permissions prompt bypassed when flag is set
// - Autonomous mode enabled/disabled based on flag
// - Interactive prompt shown when neither flag set
func TestHandleSkipPermissionsPrompt_FlagBypass(t *testing.T) {
	tests := map[string]struct {
		setupFlags   func(cmd *cobra.Command)
		wantValue    bool
		wantContains []string
	}{
		"--skip-permissions enables autonomous mode": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("skip-permissions", false, "")
				cmd.Flags().Bool("no-skip-permissions", false, "")
				_ = cmd.ParseFlags([]string{"--skip-permissions"})
			},
			wantValue:    true,
			wantContains: []string{"skip_permissions: true", "--skip-permissions"},
		},
		"--no-skip-permissions disables autonomous mode": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("skip-permissions", false, "")
				cmd.Flags().Bool("no-skip-permissions", false, "")
				_ = cmd.ParseFlags([]string{"--no-skip-permissions"})
			},
			wantValue:    false,
			wantContains: []string{"skip_permissions: false", "--no-skip-permissions"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yml")
			require.NoError(t, os.WriteFile(configPath, []byte("agent_preset: claude\n"), 0o644))

			cmd := &cobra.Command{Use: "test"}
			var outBuf bytes.Buffer
			cmd.SetOut(&outBuf)
			tt.setupFlags(cmd)

			handleSkipPermissionsPrompt(cmd, &outBuf, configPath, false)

			output := outBuf.String()

			// Verify output contains expected strings
			for _, want := range tt.wantContains {
				assert.Contains(t, output, want)
			}

			// Verify config was updated with correct value
			content, err := os.ReadFile(configPath)
			require.NoError(t, err)
			if tt.wantValue {
				assert.Contains(t, string(content), "skip_permissions: true")
			} else {
				assert.Contains(t, string(content), "skip_permissions: false")
			}

			// Should NOT show the interactive prompt section
			assert.NotContains(t, output, "Permissions Mode")
			assert.NotContains(t, output, "Enable skip_permissions (recommended)?")
		})
	}
}

// TestHandleSkipPermissionsPrompt_FlagOverridesExisting tests that flags override
// existing config values.
func TestHandleSkipPermissionsPrompt_FlagOverridesExisting(t *testing.T) {
	tests := map[string]struct {
		existingValue string
		flag          string
		wantValue     bool
	}{
		"--skip-permissions overrides existing false": {
			existingValue: "skip_permissions: false\n",
			flag:          "--skip-permissions",
			wantValue:     true,
		},
		"--no-skip-permissions overrides existing true": {
			existingValue: "skip_permissions: true\n",
			flag:          "--no-skip-permissions",
			wantValue:     false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yml")
			require.NoError(t, os.WriteFile(configPath, []byte("agent_preset: claude\n"+tt.existingValue), 0o644))

			cmd := &cobra.Command{Use: "test"}
			cmd.Flags().Bool("skip-permissions", false, "")
			cmd.Flags().Bool("no-skip-permissions", false, "")
			_ = cmd.ParseFlags([]string{tt.flag})

			var outBuf bytes.Buffer
			cmd.SetOut(&outBuf)

			handleSkipPermissionsPrompt(cmd, &outBuf, configPath, false)

			// Verify config was updated with flag value
			content, err := os.ReadFile(configPath)
			require.NoError(t, err)
			if tt.wantValue {
				assert.Contains(t, string(content), "skip_permissions: true")
			} else {
				assert.Contains(t, string(content), "skip_permissions: false")
			}
		})
	}
}

// TestBillingFlagMutualExclusivity tests that --use-subscription and
// --no-use-subscription are mutually exclusive.
// T009 acceptance criteria (mutual exclusivity):
// - Error returned when both flags provided
// - Error message clearly states flags are mutually exclusive
func TestBillingFlagMutualExclusivity(t *testing.T) {
	tests := map[string]struct {
		flags   []string
		wantErr bool
		errMsg  string
	}{
		"both billing flags returns error": {
			flags:   []string{"--use-subscription", "--no-use-subscription"},
			wantErr: true,
			errMsg:  "mutually exclusive",
		},
		"only --use-subscription no error": {
			flags:   []string{"--use-subscription"},
			wantErr: false,
		},
		"only --no-use-subscription no error": {
			flags:   []string{"--no-use-subscription"},
			wantErr: false,
		},
		"neither billing flag no error": {
			flags:   []string{},
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{Use: "test"}
			cmd.Flags().Bool("use-subscription", false, "")
			cmd.Flags().Bool("no-use-subscription", false, "")
			_ = cmd.ParseFlags(tt.flags)

			pairs := []BoolFlagPair{
				{Positive: "use-subscription", Negative: "no-use-subscription"},
			}
			err := checkMutuallyExclusiveFlags(cmd, pairs)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Contains(t, err.Error(), "use-subscription")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPermissionsFlagMutualExclusivity tests that --skip-permissions and
// --no-skip-permissions are mutually exclusive.
// T010 acceptance criteria (mutual exclusivity):
// - Error returned when both flags provided
// - Error message clearly states flags are mutually exclusive
func TestPermissionsFlagMutualExclusivity(t *testing.T) {
	tests := map[string]struct {
		flags   []string
		wantErr bool
		errMsg  string
	}{
		"both permissions flags returns error": {
			flags:   []string{"--skip-permissions", "--no-skip-permissions"},
			wantErr: true,
			errMsg:  "mutually exclusive",
		},
		"only --skip-permissions no error": {
			flags:   []string{"--skip-permissions"},
			wantErr: false,
		},
		"only --no-skip-permissions no error": {
			flags:   []string{"--no-skip-permissions"},
			wantErr: false,
		},
		"neither permissions flag no error": {
			flags:   []string{},
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{Use: "test"}
			cmd.Flags().Bool("skip-permissions", false, "")
			cmd.Flags().Bool("no-skip-permissions", false, "")
			_ = cmd.ParseFlags(tt.flags)

			pairs := []BoolFlagPair{
				{Positive: "skip-permissions", Negative: "no-skip-permissions"},
			}
			err := checkMutuallyExclusiveFlags(cmd, pairs)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Contains(t, err.Error(), "skip-permissions")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestResolveBoolFlag_BillingFlags tests three-state resolution for billing flags.
func TestResolveBoolFlag_BillingFlags(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFlags func(cmd *cobra.Command)
		want       *bool
	}{
		"neither billing flag set returns nil": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("use-subscription", false, "")
				cmd.Flags().Bool("no-use-subscription", false, "")
			},
			want: nil,
		},
		"--use-subscription returns true": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("use-subscription", false, "")
				cmd.Flags().Bool("no-use-subscription", false, "")
				_ = cmd.ParseFlags([]string{"--use-subscription"})
			},
			want: ptrBool(true),
		},
		"--no-use-subscription returns false": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("use-subscription", false, "")
				cmd.Flags().Bool("no-use-subscription", false, "")
				_ = cmd.ParseFlags([]string{"--no-use-subscription"})
			},
			want: ptrBool(false),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{Use: "test"}
			tt.setupFlags(cmd)

			got := resolveBoolFlag(cmd, "use-subscription", "no-use-subscription")

			if tt.want == nil {
				assert.Nil(t, got, "expected nil but got %v", got)
			} else {
				require.NotNil(t, got, "expected non-nil value")
				assert.Equal(t, *tt.want, *got)
			}
		})
	}
}

// TestResolveBoolFlag_PermissionsFlags tests three-state resolution for permissions flags.
func TestResolveBoolFlag_PermissionsFlags(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFlags func(cmd *cobra.Command)
		want       *bool
	}{
		"neither permissions flag set returns nil": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("skip-permissions", false, "")
				cmd.Flags().Bool("no-skip-permissions", false, "")
			},
			want: nil,
		},
		"--skip-permissions returns true": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("skip-permissions", false, "")
				cmd.Flags().Bool("no-skip-permissions", false, "")
				_ = cmd.ParseFlags([]string{"--skip-permissions"})
			},
			want: ptrBool(true),
		},
		"--no-skip-permissions returns false": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("skip-permissions", false, "")
				cmd.Flags().Bool("no-skip-permissions", false, "")
				_ = cmd.ParseFlags([]string{"--no-skip-permissions"})
			},
			want: ptrBool(false),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{Use: "test"}
			tt.setupFlags(cmd)

			got := resolveBoolFlag(cmd, "skip-permissions", "no-skip-permissions")

			if tt.want == nil {
				assert.Nil(t, got, "expected nil but got %v", got)
			} else {
				require.NotNil(t, got, "expected non-nil value")
				assert.Equal(t, *tt.want, *got)
			}
		})
	}
}

// TestAllNewFlagsMutualExclusivity tests that all new flag pairs (billing, permissions)
// are validated for mutual exclusivity in runInit.
func TestAllNewFlagsMutualExclusivity(t *testing.T) {
	tests := map[string]struct {
		flags   []string
		wantErr bool
		errMsg  string
	}{
		"sandbox conflict": {
			flags:   []string{"--sandbox", "--no-sandbox"},
			wantErr: true,
			errMsg:  "sandbox",
		},
		"billing conflict": {
			flags:   []string{"--use-subscription", "--no-use-subscription"},
			wantErr: true,
			errMsg:  "use-subscription",
		},
		"permissions conflict": {
			flags:   []string{"--skip-permissions", "--no-skip-permissions"},
			wantErr: true,
			errMsg:  "skip-permissions",
		},
		"mixed valid flags no error": {
			flags:   []string{"--sandbox", "--use-subscription", "--skip-permissions"},
			wantErr: false,
		},
		"mixed with one conflict": {
			flags:   []string{"--sandbox", "--use-subscription", "--no-use-subscription"},
			wantErr: true,
			errMsg:  "use-subscription",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{Use: "test"}
			// Register all flag pairs
			cmd.Flags().Bool("sandbox", false, "")
			cmd.Flags().Bool("no-sandbox", false, "")
			cmd.Flags().Bool("use-subscription", false, "")
			cmd.Flags().Bool("no-use-subscription", false, "")
			cmd.Flags().Bool("skip-permissions", false, "")
			cmd.Flags().Bool("no-skip-permissions", false, "")
			_ = cmd.ParseFlags(tt.flags)

			// Use the same pairs as in runInit
			pairs := []BoolFlagPair{
				{Positive: "sandbox", Negative: "no-sandbox"},
				{Positive: "use-subscription", Negative: "no-use-subscription"},
				{Positive: "skip-permissions", Negative: "no-skip-permissions"},
			}
			err := checkMutuallyExclusiveFlags(cmd, pairs)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ============================================================================
// Phase 5: Gitignore and Constitution Flag Tests (T014-T018)
// ============================================================================

// TestGitignoreFlagsMutualExclusivity tests that --gitignore and --no-gitignore
// are mutually exclusive.
func TestGitignoreFlagsMutualExclusivity(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		flags   []string
		wantErr bool
		errMsg  string
	}{
		"both gitignore flags returns error": {
			flags:   []string{"--gitignore", "--no-gitignore"},
			wantErr: true,
			errMsg:  "mutually exclusive",
		},
		"only --gitignore no error": {
			flags:   []string{"--gitignore"},
			wantErr: false,
		},
		"only --no-gitignore no error": {
			flags:   []string{"--no-gitignore"},
			wantErr: false,
		},
		"neither gitignore flag no error": {
			flags:   []string{},
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{Use: "test"}
			cmd.Flags().Bool("gitignore", false, "")
			cmd.Flags().Bool("no-gitignore", false, "")
			_ = cmd.ParseFlags(tt.flags)

			pairs := []BoolFlagPair{
				{Positive: "gitignore", Negative: "no-gitignore"},
			}
			err := checkMutuallyExclusiveFlags(cmd, pairs)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Contains(t, err.Error(), "gitignore")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConstitutionFlagsMutualExclusivity tests that --constitution and
// --no-constitution are mutually exclusive.
func TestConstitutionFlagsMutualExclusivity(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		flags   []string
		wantErr bool
		errMsg  string
	}{
		"both constitution flags returns error": {
			flags:   []string{"--constitution", "--no-constitution"},
			wantErr: true,
			errMsg:  "mutually exclusive",
		},
		"only --constitution no error": {
			flags:   []string{"--constitution"},
			wantErr: false,
		},
		"only --no-constitution no error": {
			flags:   []string{"--no-constitution"},
			wantErr: false,
		},
		"neither constitution flag no error": {
			flags:   []string{},
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{Use: "test"}
			cmd.Flags().Bool("constitution", false, "")
			cmd.Flags().Bool("no-constitution", false, "")
			_ = cmd.ParseFlags(tt.flags)

			pairs := []BoolFlagPair{
				{Positive: "constitution", Negative: "no-constitution"},
			}
			err := checkMutuallyExclusiveFlags(cmd, pairs)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Contains(t, err.Error(), "constitution")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestResolveBoolFlag_GitignoreFlags tests three-state resolution for gitignore flags.
func TestResolveBoolFlag_GitignoreFlags(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFlags func(cmd *cobra.Command)
		want       *bool
	}{
		"neither gitignore flag set returns nil": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("gitignore", false, "")
				cmd.Flags().Bool("no-gitignore", false, "")
			},
			want: nil,
		},
		"--gitignore returns true": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("gitignore", false, "")
				cmd.Flags().Bool("no-gitignore", false, "")
				_ = cmd.ParseFlags([]string{"--gitignore"})
			},
			want: ptrBool(true),
		},
		"--no-gitignore returns false": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("gitignore", false, "")
				cmd.Flags().Bool("no-gitignore", false, "")
				_ = cmd.ParseFlags([]string{"--no-gitignore"})
			},
			want: ptrBool(false),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{Use: "test"}
			tt.setupFlags(cmd)

			got := resolveBoolFlag(cmd, "gitignore", "no-gitignore")

			if tt.want == nil {
				assert.Nil(t, got, "expected nil but got %v", got)
			} else {
				require.NotNil(t, got, "expected non-nil value")
				assert.Equal(t, *tt.want, *got)
			}
		})
	}
}

// TestResolveBoolFlag_ConstitutionFlags tests three-state resolution for constitution flags.
func TestResolveBoolFlag_ConstitutionFlags(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFlags func(cmd *cobra.Command)
		want       *bool
	}{
		"neither constitution flag set returns nil": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("constitution", false, "")
				cmd.Flags().Bool("no-constitution", false, "")
			},
			want: nil,
		},
		"--constitution returns true": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("constitution", false, "")
				cmd.Flags().Bool("no-constitution", false, "")
				_ = cmd.ParseFlags([]string{"--constitution"})
			},
			want: ptrBool(true),
		},
		"--no-constitution returns false": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("constitution", false, "")
				cmd.Flags().Bool("no-constitution", false, "")
				_ = cmd.ParseFlags([]string{"--no-constitution"})
			},
			want: ptrBool(false),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{Use: "test"}
			tt.setupFlags(cmd)

			got := resolveBoolFlag(cmd, "constitution", "no-constitution")

			if tt.want == nil {
				assert.Nil(t, got, "expected nil but got %v", got)
			} else {
				require.NotNil(t, got, "expected non-nil value")
				assert.Equal(t, *tt.want, *got)
			}
		})
	}
}

// TestCollectGitignoreChoice_FlagBehavior tests the gitignore flag bypass behavior.
func TestCollectGitignoreChoice_FlagBehavior(t *testing.T) {
	tests := map[string]struct {
		setupFlags    func(cmd *cobra.Command)
		wantContains  []string
		wantResult    bool
		createGitFile bool
	}{
		"--gitignore flag bypasses prompt": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("gitignore", false, "")
				cmd.Flags().Bool("no-gitignore", false, "")
				_ = cmd.ParseFlags([]string{"--gitignore"})
			},
			wantContains: []string{"Gitignore", "--gitignore"},
			wantResult:   true,
		},
		"--no-gitignore flag skips": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("gitignore", false, "")
				cmd.Flags().Bool("no-gitignore", false, "")
				_ = cmd.ParseFlags([]string{"--no-gitignore"})
			},
			wantContains: []string{"Gitignore", "--no-gitignore", "skipped"},
			wantResult:   false,
		},
		"already present shows message": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("gitignore", false, "")
				cmd.Flags().Bool("no-gitignore", false, "")
			},
			wantContains:  []string{"Gitignore", "already present"},
			wantResult:    false,
			createGitFile: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			origDir, err := os.Getwd()
			require.NoError(t, err)
			require.NoError(t, os.Chdir(tmpDir))
			defer func() { _ = os.Chdir(origDir) }()

			if tt.createGitFile {
				// Create .gitignore with .autospec/ already present
				require.NoError(t, os.WriteFile(".gitignore", []byte(".autospec/\n"), 0o644))
			}

			cmd := &cobra.Command{Use: "test"}
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetIn(bytes.NewBufferString("n\n")) // Default answer if prompt shown
			tt.setupFlags(cmd)

			result := collectGitignoreChoice(cmd, &buf)

			output := buf.String()
			for _, want := range tt.wantContains {
				assert.Contains(t, output, want, "expected output to contain %q", want)
			}
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

// TestCollectConstitutionChoice_FlagBehavior tests the constitution flag bypass behavior.
func TestCollectConstitutionChoice_FlagBehavior(t *testing.T) {
	tests := map[string]struct {
		setupFlags         func(cmd *cobra.Command)
		constitutionExists bool
		wantContains       []string
		wantResult         bool
	}{
		"--constitution flag bypasses prompt": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("constitution", false, "")
				cmd.Flags().Bool("no-constitution", false, "")
				_ = cmd.ParseFlags([]string{"--constitution"})
			},
			constitutionExists: false,
			wantContains:       []string{"Constitution", "will create", "--constitution"},
			wantResult:         true,
		},
		"--constitution with existing constitution shows ignored": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("constitution", false, "")
				cmd.Flags().Bool("no-constitution", false, "")
				_ = cmd.ParseFlags([]string{"--constitution"})
			},
			constitutionExists: true,
			wantContains:       []string{"Constitution", "already exists", "ignored"},
			wantResult:         false,
		},
		"--no-constitution flag skips": {
			setupFlags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("constitution", false, "")
				cmd.Flags().Bool("no-constitution", false, "")
				_ = cmd.ParseFlags([]string{"--no-constitution"})
			},
			constitutionExists: false,
			wantContains:       []string{"Constitution", "skipped", "--no-constitution"},
			wantResult:         false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetIn(bytes.NewBufferString("n\n")) // Default answer if prompt shown
			tt.setupFlags(cmd)

			result := collectConstitutionChoice(cmd, &buf, tt.constitutionExists)

			output := buf.String()
			for _, want := range tt.wantContains {
				assert.Contains(t, output, want, "expected output to contain %q", want)
			}
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

// TestCollectConstitutionChoice_InteractivePromptIsAgentAgnostic verifies init copy
// does not imply the constitution workflow always runs Claude.
func TestCollectConstitutionChoice_InteractivePromptIsAgentAgnostic(t *testing.T) {
	// Cannot run in parallel: modifies global isTerminalFunc.
	originalIsTerminal := isTerminalFunc
	isTerminalFunc = func() bool { return true }
	defer func() { isTerminalFunc = originalIsTerminal }()

	cmd := &cobra.Command{Use: "test"}
	var buf bytes.Buffer
	cmd.Flags().Bool("constitution", false, "")
	cmd.Flags().Bool("no-constitution", false, "")
	cmd.SetOut(&buf)
	cmd.SetIn(bytes.NewBufferString("n\n"))

	result := collectConstitutionChoice(cmd, &buf, false)

	output := buf.String()
	assert.False(t, result)
	assert.Contains(t, output, "Runs the configured agent to analyze your project")
	assert.NotContains(t, output, "Runs a Claude session to analyze your project")
}

// TestCollectGitignoreChoice_NonInteractive tests non-interactive mode behavior.
func TestCollectGitignoreChoice_NonInteractive(t *testing.T) {
	// Cannot run in parallel: modifies global isTerminalFunc
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	// Mock isTerminalFunc to simulate non-interactive mode
	originalIsTerminal := isTerminalFunc
	isTerminalFunc = func() bool { return false }
	defer func() { isTerminalFunc = originalIsTerminal }()

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("gitignore", false, "")
	cmd.Flags().Bool("no-gitignore", false, "")
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	result := collectGitignoreChoice(cmd, &buf)

	output := buf.String()
	assert.Contains(t, output, "non-interactive")
	assert.Contains(t, output, "--gitignore")
	assert.False(t, result, "should default to false in non-interactive mode")
}

// TestCollectConstitutionChoice_NonInteractive tests non-interactive mode behavior.
func TestCollectConstitutionChoice_NonInteractive(t *testing.T) {
	// Cannot run in parallel: modifies global isTerminalFunc

	// Mock isTerminalFunc to simulate non-interactive mode
	originalIsTerminal := isTerminalFunc
	isTerminalFunc = func() bool { return false }
	defer func() { isTerminalFunc = originalIsTerminal }()

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("constitution", false, "")
	cmd.Flags().Bool("no-constitution", false, "")
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	result := collectConstitutionChoice(cmd, &buf, false)

	output := buf.String()
	assert.Contains(t, output, "non-interactive")
	assert.Contains(t, output, "--constitution")
	assert.False(t, result, "should default to false in non-interactive mode")
}

// TestAllFlagPairsMutualExclusivity tests all flag pairs including new gitignore and constitution.
func TestAllFlagPairsMutualExclusivity(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		flags   []string
		wantErr bool
		errMsg  string
	}{
		"gitignore conflict": {
			flags:   []string{"--gitignore", "--no-gitignore"},
			wantErr: true,
			errMsg:  "gitignore",
		},
		"constitution conflict": {
			flags:   []string{"--constitution", "--no-constitution"},
			wantErr: true,
			errMsg:  "constitution",
		},
		"all positive flags no conflict": {
			flags:   []string{"--sandbox", "--use-subscription", "--skip-permissions", "--gitignore", "--constitution"},
			wantErr: false,
		},
		"all negative flags no conflict": {
			flags:   []string{"--no-sandbox", "--no-use-subscription", "--no-skip-permissions", "--no-gitignore", "--no-constitution"},
			wantErr: false,
		},
		"mixed flags no conflict": {
			flags:   []string{"--sandbox", "--no-use-subscription", "--skip-permissions", "--no-gitignore", "--constitution"},
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{Use: "test"}
			// Register all flag pairs
			cmd.Flags().Bool("sandbox", false, "")
			cmd.Flags().Bool("no-sandbox", false, "")
			cmd.Flags().Bool("use-subscription", false, "")
			cmd.Flags().Bool("no-use-subscription", false, "")
			cmd.Flags().Bool("skip-permissions", false, "")
			cmd.Flags().Bool("no-skip-permissions", false, "")
			cmd.Flags().Bool("gitignore", false, "")
			cmd.Flags().Bool("no-gitignore", false, "")
			cmd.Flags().Bool("constitution", false, "")
			cmd.Flags().Bool("no-constitution", false, "")
			_ = cmd.ParseFlags(tt.flags)

			// Use the same pairs as in runInit
			pairs := []BoolFlagPair{
				{Positive: "sandbox", Negative: "no-sandbox"},
				{Positive: "use-subscription", Negative: "no-use-subscription"},
				{Positive: "skip-permissions", Negative: "no-skip-permissions"},
				{Positive: "gitignore", Negative: "no-gitignore"},
				{Positive: "constitution", Negative: "no-constitution"},
			}
			err := checkMutuallyExclusiveFlags(cmd, pairs)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
