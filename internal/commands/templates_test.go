// Package commands_test tests template installation, versioning, and frontmatter parsing.
// Related: /home/ari/repos/autospec/internal/commands/templates.go
// Tags: commands, templates, versioning, installation, frontmatter

package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTemplates(t *testing.T) {
	templates, err := ListTemplates()
	require.NoError(t, err)
	assert.NotEmpty(t, templates, "should have embedded templates")

	// Check for expected templates
	names := make([]string, len(templates))
	for i, tpl := range templates {
		names[i] = tpl.Name
	}
	assert.Contains(t, names, "autospec.specify", "should include specify")
	assert.Contains(t, names, "autospec.plan", "should include plan")
	assert.Contains(t, names, "autospec.tasks", "should include tasks")
}

func TestListTemplates_HasContent(t *testing.T) {
	templates, err := ListTemplates()
	require.NoError(t, err)

	for _, tpl := range templates {
		assert.NotEmpty(t, tpl.Content, "%s should have content", tpl.Name)
		assert.NotEmpty(t, tpl.Description, "%s should have description", tpl.Name)
		assert.NotEmpty(t, tpl.Version, "%s should have version", tpl.Name)
	}
}

func TestGetTemplateInfo(t *testing.T) {
	info, err := GetTemplateInfo("autospec.specify")
	require.NoError(t, err)

	assert.Equal(t, "autospec.specify", info.Name)
	assert.NotEmpty(t, info.Description)
	assert.NotEmpty(t, info.Version)
	assert.NotEmpty(t, info.Content)
}

func TestGetTemplateInfo_NotFound(t *testing.T) {
	_, err := GetTemplateInfo("nonexistent")
	assert.Error(t, err)
}

func TestInstallTemplates(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".claude", "commands")

	results, err := InstallTemplates(targetDir)
	require.NoError(t, err)
	assert.NotEmpty(t, results)

	// Verify files were created
	for _, result := range results {
		if result.Action == "installed" || result.Action == "updated" {
			_, err := os.Stat(result.Path)
			assert.NoError(t, err, "file should exist: %s", result.Path)
		}
	}

	// Verify specific template was installed
	specifyPath := filepath.Join(targetDir, "autospec.specify.md")
	content, err := os.ReadFile(specifyPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "description:")
}

func TestInstallTemplates_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".claude", "commands")

	// First install
	results1, err := InstallTemplates(targetDir)
	require.NoError(t, err)

	// Second install
	results2, err := InstallTemplates(targetDir)
	require.NoError(t, err)

	// Should have same number of results
	assert.Equal(t, len(results1), len(results2))
}

func TestCheckVersions_NoInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".claude", "commands")

	// Check without installing first
	mismatches, err := CheckVersions(targetDir)
	require.NoError(t, err)

	// All templates should show as needing install
	assert.NotEmpty(t, mismatches)
	for _, m := range mismatches {
		assert.Equal(t, "install", m.Action, "%s should need install", m.CommandName)
		assert.Empty(t, m.InstalledVersion)
		assert.NotEmpty(t, m.EmbeddedVersion)
	}
}

func TestCheckVersions_AllCurrent(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".claude", "commands")

	// Install templates first
	_, err := InstallTemplates(targetDir)
	require.NoError(t, err)

	// Check versions
	mismatches, err := CheckVersions(targetDir)
	require.NoError(t, err)

	// All should be current (empty mismatches)
	assert.Empty(t, mismatches, "should have no mismatches after install")
}

func TestParseTemplateFrontmatter(t *testing.T) {
	content := []byte(`---
description: Test description
version: "1.2.3"
---

# Content here
`)

	desc, version, err := ParseTemplateFrontmatter(content)
	require.NoError(t, err)
	assert.Equal(t, "Test description", desc)
	assert.Equal(t, "1.2.3", version)
}

func TestParseTemplateFrontmatter_NoFrontmatter(t *testing.T) {
	content := []byte(`# No frontmatter here`)

	_, _, err := ParseTemplateFrontmatter(content)
	assert.Error(t, err, "should error without frontmatter")
}

func TestParseTemplateFrontmatter_Variations(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		content     string
		wantDesc    string
		wantVersion string
		wantErr     bool
	}{
		"valid frontmatter": {
			content: `---
description: Test description
version: "1.2.3"
---

# Content`,
			wantDesc:    "Test description",
			wantVersion: "1.2.3",
			wantErr:     false,
		},
		"frontmatter not closed": {
			content: `---
description: Test
`,
			wantErr: true,
		},
		"no frontmatter markers": {
			content: `description: Test
version: "1.0.0"`,
			wantErr: true,
		},
		"empty frontmatter": {
			content: `---
---
# Content`,
			wantDesc:    "",
			wantVersion: "",
			wantErr:     false,
		},
		"only description": {
			content: `---
description: Only description here
---`,
			wantDesc:    "Only description here",
			wantVersion: "",
			wantErr:     false,
		},
		"only version": {
			content: `---
version: "2.0.0"
---`,
			wantDesc:    "",
			wantVersion: "2.0.0",
			wantErr:     false,
		},
		"invalid yaml in frontmatter": {
			content: `---
description: [invalid: yaml: syntax
---`,
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			desc, version, err := ParseTemplateFrontmatter([]byte(tt.content))

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantDesc, desc)
			assert.Equal(t, tt.wantVersion, version)
		})
	}
}

func TestGetInstalledCommands(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup       func(t *testing.T) string // returns targetDir
		wantErr     bool
		checkResult func(t *testing.T, commands []CommandInfo)
	}{
		"empty directory": {
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: false,
			checkResult: func(t *testing.T, commands []CommandInfo) {
				// Commands list reflects embedded templates, but none are installed
				assert.NotEmpty(t, commands)
				for _, cmd := range commands {
					assert.Empty(t, cmd.Version, "uninstalled command should have empty version")
				}
			},
		},
		"directory with installed commands": {
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				_, err := InstallTemplates(dir)
				require.NoError(t, err)
				return dir
			},
			wantErr: false,
			checkResult: func(t *testing.T, commands []CommandInfo) {
				assert.NotEmpty(t, commands)
				for _, cmd := range commands {
					assert.NotEmpty(t, cmd.Version, "%s should have version after install", cmd.Name)
					assert.False(t, cmd.IsOutdated, "%s should not be outdated after fresh install", cmd.Name)
				}
			},
		},
		"directory with outdated command": {
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				// Install templates first
				_, err := InstallTemplates(dir)
				require.NoError(t, err)

				// Modify one template to have different version
				specifyPath := filepath.Join(dir, "autospec.specify.md")
				content := []byte(`---
description: Modified
version: "0.0.1"
---
Old content`)
				err = os.WriteFile(specifyPath, content, 0o644)
				require.NoError(t, err)

				return dir
			},
			wantErr: false,
			checkResult: func(t *testing.T, commands []CommandInfo) {
				assert.NotEmpty(t, commands)
				foundOutdated := false
				for _, cmd := range commands {
					if cmd.Name == "autospec.specify" {
						assert.True(t, cmd.IsOutdated, "modified command should be outdated")
						assert.Equal(t, "0.0.1", cmd.Version)
						foundOutdated = true
					}
				}
				assert.True(t, foundOutdated, "should find the outdated command")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			targetDir := tt.setup(t)
			commands, err := GetInstalledCommands(targetDir)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			tt.checkResult(t, commands)
		})
	}
}

func TestGetDefaultCommandsDir(t *testing.T) {
	t.Parallel()

	dir := GetDefaultCommandsDir()
	assert.Equal(t, filepath.Join(".claude", "commands"), dir)
}

func TestCommandExists(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup       func(t *testing.T) string
		commandName string
		want        bool
	}{
		"command exists": {
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "test-cmd.md")
				err := os.WriteFile(path, []byte("content"), 0o644)
				require.NoError(t, err)
				return dir
			},
			commandName: "test-cmd",
			want:        true,
		},
		"command does not exist": {
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			commandName: "nonexistent",
			want:        false,
		},
		"empty directory": {
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			commandName: "any-command",
			want:        false,
		},
		"directory with other files": {
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				// Create a file with different name
				path := filepath.Join(dir, "other-cmd.md")
				err := os.WriteFile(path, []byte("content"), 0o644)
				require.NoError(t, err)
				return dir
			},
			commandName: "test-cmd",
			want:        false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			targetDir := tt.setup(t)
			got := CommandExists(targetDir, tt.commandName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetAutospecCommandNames(t *testing.T) {
	t.Parallel()

	names := GetAutospecCommandNames()

	// Should have autospec commands
	assert.NotEmpty(t, names)

	// All names should start with "autospec."
	for _, name := range names {
		assert.True(t, len(name) > 9, "name should be longer than 'autospec.'")
		assert.Equal(t, "autospec.", name[:9], "all names should start with 'autospec.'")
	}

	// Should include known commands
	assert.Contains(t, names, "autospec.specify")
	assert.Contains(t, names, "autospec.plan")
	assert.Contains(t, names, "autospec.tasks")
}

func TestCheckVersions_Variations(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup       func(t *testing.T) string
		checkResult func(t *testing.T, mismatches []VersionMismatch)
	}{
		"modified version needs update": {
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				// Install first
				_, err := InstallTemplates(dir)
				require.NoError(t, err)

				// Modify one file version
				path := filepath.Join(dir, "autospec.specify.md")
				content := []byte(`---
description: Test
version: "0.0.1"
---
content`)
				err = os.WriteFile(path, content, 0o644)
				require.NoError(t, err)
				return dir
			},
			checkResult: func(t *testing.T, mismatches []VersionMismatch) {
				// Should have one mismatch for specify
				found := false
				for _, m := range mismatches {
					if m.CommandName == "autospec.specify" {
						found = true
						assert.Equal(t, "update", m.Action)
						assert.Equal(t, "0.0.1", m.InstalledVersion)
						assert.NotEqual(t, "0.0.1", m.EmbeddedVersion)
					}
				}
				assert.True(t, found, "should find mismatch for modified command")
			},
		},
		"corrupted frontmatter needs update": {
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				// Create file with invalid frontmatter
				path := filepath.Join(dir, "autospec.specify.md")
				content := []byte(`not valid frontmatter`)
				err := os.WriteFile(path, content, 0o644)
				require.NoError(t, err)
				return dir
			},
			checkResult: func(t *testing.T, mismatches []VersionMismatch) {
				found := false
				for _, m := range mismatches {
					if m.CommandName == "autospec.specify" {
						found = true
						assert.Equal(t, "update", m.Action)
						assert.Empty(t, m.InstalledVersion, "corrupted file has no parseable version")
					}
				}
				assert.True(t, found, "should find mismatch for corrupted command")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := tt.setup(t)
			mismatches, err := CheckVersions(dir)
			require.NoError(t, err)
			tt.checkResult(t, mismatches)
		})
	}
}

func TestGetCommandsDir(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agentName string
		wantDir   string
		wantErr   bool
	}{
		"claude agent": {
			agentName: "claude",
			wantDir:   filepath.Join(".claude", "commands"),
			wantErr:   false,
		},
		"unknown agent": {
			agentName: "unknown",
			wantDir:   "",
			wantErr:   true,
		},
		"empty agent name": {
			agentName: "",
			wantDir:   "",
			wantErr:   true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotDir, err := GetCommandsDir(tt.agentName)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, gotDir)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantDir, gotDir)
		})
	}
}

func TestInstallTemplatesForAgent(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agentName   string
		wantErr     bool
		checkResult func(t *testing.T, projectDir string, results []InstallResult)
	}{
		"claude agent installs to .claude/commands": {
			agentName: "claude",
			wantErr:   false,
			checkResult: func(t *testing.T, projectDir string, results []InstallResult) {
				assert.NotEmpty(t, results)
				expectedDir := filepath.Join(projectDir, ".claude", "commands")
				for _, r := range results {
					assert.True(t, filepath.HasPrefix(r.Path, expectedDir),
						"path %s should be under %s", r.Path, expectedDir)
				}
				// Verify a specific file exists
				specifyPath := filepath.Join(expectedDir, "autospec.specify.md")
				_, err := os.Stat(specifyPath)
				assert.NoError(t, err, "autospec.specify.md should exist")
			},
		},
		"unknown agent returns error": {
			agentName: "unknown",
			wantErr:   true,
			checkResult: func(t *testing.T, projectDir string, results []InstallResult) {
				assert.Nil(t, results)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			projectDir := t.TempDir()
			results, err := InstallTemplatesForAgent(tt.agentName, projectDir)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, projectDir, results)
				}
				return
			}

			require.NoError(t, err)
			tt.checkResult(t, projectDir, results)
		})
	}
}

func TestInstallTemplatesForAgent_Idempotent(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()

	// First install
	results1, err := InstallTemplatesForAgent("claude", projectDir)
	require.NoError(t, err)
	assert.NotEmpty(t, results1)

	// Second install - should work without errors
	results2, err := InstallTemplatesForAgent("claude", projectDir)
	require.NoError(t, err)
	assert.Equal(t, len(results1), len(results2), "same number of results on reinstall")

	// All should be "updated" on second install
	for _, r := range results2 {
		assert.Equal(t, "updated", r.Action, "%s should be updated on reinstall", r.CommandName)
	}
}

func TestStripFrontmatter(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		content []byte
		want    []byte
	}{
		"valid frontmatter": {
			content: []byte(`---
description: Test description
version: "1.2.3"
---

# Content here`),
			want: []byte(`# Content here`),
		},
		"frontmatter with trailing newlines": {
			content: []byte(`---
description: Test
---


# Multiple newlines before content`),
			want: []byte(`# Multiple newlines before content`),
		},
		"no frontmatter": {
			content: []byte(`# No frontmatter here`),
			want:    []byte(`# No frontmatter here`),
		},
		"unclosed frontmatter": {
			content: []byte(`---
description: Test
no closing marker`),
			want: []byte(`---
description: Test
no closing marker`),
		},
		"empty content after frontmatter": {
			content: []byte(`---
description: Test
---
`),
			want: []byte(``),
		},
		"frontmatter with immediate content": {
			content: []byte(`---
version: "1.0.0"
---
Immediate content`),
			want: []byte(`Immediate content`),
		},
		"empty frontmatter": {
			content: []byte(`---
---
Content`),
			want: []byte(`Content`),
		},
		"content starting with dashes but not frontmatter": {
			content: []byte(`-- not frontmatter`),
			want:    []byte(`-- not frontmatter`),
		},
		"complex content after frontmatter": {
			content: []byte(`---
description: Generate YAML plan
version: "1.0.0"
---

## User Input

` + "```text" + `
$ARGUMENTS
` + "```" + `

You **MUST** consider this.`),
			want: []byte(`## User Input

` + "```text" + `
$ARGUMENTS
` + "```" + `

You **MUST** consider this.`),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := StripFrontmatter(tt.content)
			assert.Equal(t, string(tt.want), string(got))
		})
	}
}
