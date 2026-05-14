// Package clean_test tests autospec artifact cleanup and directory management.
// Related: /home/ari/repos/autospec/internal/clean/clean.go
// Tags: clean, cleanup, artifacts, directories

package clean

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindAutospecFiles_NoFiles(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	require.NoError(t, os.Chdir(tmpDir))

	targets, err := FindAutospecFiles(false)
	require.NoError(t, err)
	assert.Empty(t, targets)
}

func TestFindAutospecFiles_AllTargets(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	require.NoError(t, os.Chdir(tmpDir))

	// Create .autospec directory
	require.NoError(t, os.MkdirAll(".autospec", 0o755))

	// Create specs directory
	require.NoError(t, os.MkdirAll("specs", 0o755))

	// Create .claude/commands directory with legacy autospec files
	require.NoError(t, os.MkdirAll(".claude/commands", 0o755))
	require.NoError(t, os.WriteFile(".claude/commands/autospec.plan.md", []byte("test"), 0o644))
	require.NoError(t, os.WriteFile(".claude/commands/autospec.tasks.md", []byte("test"), 0o644))

	// Create .claude/skills directory with autospec skills
	require.NoError(t, os.MkdirAll(".claude/skills/autospec.plan", 0o755))
	require.NoError(t, os.WriteFile(".claude/skills/autospec.plan/SKILL.md", []byte("test"), 0o644))
	require.NoError(t, os.MkdirAll(".claude/skills/autospec.tasks", 0o755))
	require.NoError(t, os.WriteFile(".claude/skills/autospec.tasks/SKILL.md", []byte("test"), 0o644))

	targets, err := FindAutospecFiles(false)
	require.NoError(t, err)

	// Should find 6 targets: .autospec, specs, 2 command files, and 2 skill dirs
	assert.Len(t, targets, 6)

	// Verify target types
	var dirs, files int
	for _, t := range targets {
		if t.Type == TypeDirectory {
			dirs++
		} else {
			files++
		}
	}
	assert.Equal(t, 4, dirs, "should have 4 directories")
	assert.Equal(t, 2, files, "should have 2 files")
}

func TestFindAutospecFiles_KeepSpecs(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	require.NoError(t, os.Chdir(tmpDir))

	// Create .autospec and specs directories
	require.NoError(t, os.MkdirAll(".autospec", 0o755))
	require.NoError(t, os.MkdirAll("specs", 0o755))

	// With keepSpecs=true, specs should not be included
	targets, err := FindAutospecFiles(true)
	require.NoError(t, err)

	assert.Len(t, targets, 1)
	assert.Equal(t, ".autospec", targets[0].Path)
}

func TestFindAutospecFiles_OnlyAutospecDir(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	require.NoError(t, os.Chdir(tmpDir))

	// Only create .autospec directory
	require.NoError(t, os.MkdirAll(".autospec", 0o755))

	targets, err := FindAutospecFiles(false)
	require.NoError(t, err)

	assert.Len(t, targets, 1)
	assert.Equal(t, ".autospec", targets[0].Path)
	assert.Equal(t, TypeDirectory, targets[0].Type)
	assert.NotEmpty(t, targets[0].Description)
}

func TestFindAutospecFiles_OnlySpecs(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	require.NoError(t, os.Chdir(tmpDir))

	// Only create specs directory
	require.NoError(t, os.MkdirAll("specs", 0o755))

	targets, err := FindAutospecFiles(false)
	require.NoError(t, err)

	assert.Len(t, targets, 1)
	assert.Equal(t, "specs", targets[0].Path)
}

func TestFindAutospecFiles_OnlyCommandFiles(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	require.NoError(t, os.Chdir(tmpDir))

	// Create .claude/commands with only autospec files
	require.NoError(t, os.MkdirAll(".claude/commands", 0o755))
	require.NoError(t, os.WriteFile(".claude/commands/autospec.plan.md", []byte("test"), 0o644))
	require.NoError(t, os.WriteFile(".claude/commands/autospec.implement.md", []byte("test"), 0o644))
	// Non-autospec file should not be included
	require.NoError(t, os.WriteFile(".claude/commands/custom.md", []byte("test"), 0o644))

	targets, err := FindAutospecFiles(false)
	require.NoError(t, err)

	// Should only find autospec*.md files, not custom.md
	assert.Len(t, targets, 2)
	for _, target := range targets {
		assert.Contains(t, target.Path, "autospec")
		assert.Equal(t, TypeFile, target.Type)
	}
}

func TestFindAutospecFiles_OnlyClaudeSkills(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	require.NoError(t, os.Chdir(tmpDir))

	require.NoError(t, os.MkdirAll(".claude/skills/autospec.plan", 0o755))
	require.NoError(t, os.WriteFile(".claude/skills/autospec.plan/SKILL.md", []byte("test"), 0o644))
	require.NoError(t, os.MkdirAll(".claude/skills/autospec.implement", 0o755))
	require.NoError(t, os.WriteFile(".claude/skills/autospec.implement/SKILL.md", []byte("test"), 0o644))
	require.NoError(t, os.MkdirAll(".claude/skills/custom", 0o755))
	require.NoError(t, os.WriteFile(".claude/skills/custom/SKILL.md", []byte("test"), 0o644))

	targets, err := FindAutospecFiles(false)
	require.NoError(t, err)

	assert.Len(t, targets, 2)
	for _, target := range targets {
		assert.Contains(t, target.Path, filepath.Join(".claude", "skills", "autospec."))
		assert.Equal(t, TypeDirectory, target.Type)
	}
}

func TestRemoveFiles_Success(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	require.NoError(t, os.Chdir(tmpDir))

	// Create files to remove
	require.NoError(t, os.MkdirAll(".autospec/state", 0o755))
	require.NoError(t, os.WriteFile(".autospec/config.yml", []byte("test"), 0o644))
	require.NoError(t, os.MkdirAll(".claude/commands", 0o755))
	require.NoError(t, os.WriteFile(".claude/commands/autospec.plan.md", []byte("test"), 0o644))
	require.NoError(t, os.MkdirAll(".claude/skills/autospec.plan", 0o755))
	require.NoError(t, os.WriteFile(".claude/skills/autospec.plan/SKILL.md", []byte("test"), 0o644))

	targets := []CleanTarget{
		{Path: ".autospec", Type: TypeDirectory, Description: "test"},
		{Path: ".claude/commands/autospec.plan.md", Type: TypeFile, Description: "test"},
		{Path: ".claude/skills/autospec.plan", Type: TypeDirectory, Description: "test"},
	}

	results := RemoveFiles(targets)

	assert.Len(t, results, 3)
	for _, result := range results {
		assert.True(t, result.Success)
		assert.Nil(t, result.Error)
	}

	// Verify files are gone
	_, err := os.Stat(".autospec")
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(".claude/commands/autospec.plan.md")
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(".claude/skills/autospec.plan")
	assert.True(t, os.IsNotExist(err))
}

func TestRemoveFiles_PartialFailure(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	require.NoError(t, os.Chdir(tmpDir))

	// Create one file that exists
	require.NoError(t, os.MkdirAll(".autospec", 0o755))

	targets := []CleanTarget{
		{Path: ".autospec", Type: TypeDirectory, Description: "exists"},
		{Path: "nonexistent-file.txt", Type: TypeFile, Description: "does not exist"},
	}

	results := RemoveFiles(targets)

	assert.Len(t, results, 2)

	// First should succeed
	assert.True(t, results[0].Success)
	assert.Nil(t, results[0].Error)

	// Second should fail (file doesn't exist)
	assert.False(t, results[1].Success)
	assert.NotNil(t, results[1].Error)
}

func TestRemoveFiles_DirectoryWithContents(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	require.NoError(t, os.Chdir(tmpDir))

	// Create directory with nested content
	require.NoError(t, os.MkdirAll(".autospec/memory/nested", 0o755))
	require.NoError(t, os.WriteFile(".autospec/memory/constitution.yaml", []byte("test"), 0o644))
	require.NoError(t, os.WriteFile(".autospec/memory/nested/file.txt", []byte("test"), 0o644))

	targets := []CleanTarget{
		{Path: ".autospec", Type: TypeDirectory, Description: "test"},
	}

	results := RemoveFiles(targets)

	assert.Len(t, results, 1)
	assert.True(t, results[0].Success)

	// Verify entire tree is removed
	_, err := os.Stat(".autospec")
	assert.True(t, os.IsNotExist(err))
}

func TestRemoveFiles_CleansUpEmptyDirs(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	require.NoError(t, os.Chdir(tmpDir))

	// Create .claude/commands with only autospec files
	require.NoError(t, os.MkdirAll(".claude/commands", 0o755))
	require.NoError(t, os.WriteFile(".claude/commands/autospec.plan.md", []byte("test"), 0o644))

	targets := []CleanTarget{
		{Path: ".claude/commands/autospec.plan.md", Type: TypeFile, Description: "test"},
	}

	results := RemoveFiles(targets)

	assert.Len(t, results, 1)
	assert.True(t, results[0].Success)

	// Verify empty directories are cleaned up
	_, err := os.Stat(".claude/commands")
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(".claude")
	assert.True(t, os.IsNotExist(err))
}

func TestRemoveFiles_PreservesNonEmptyDirs(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	require.NoError(t, os.Chdir(tmpDir))

	// Create .claude/commands with both autospec and non-autospec files
	require.NoError(t, os.MkdirAll(".claude/commands", 0o755))
	require.NoError(t, os.WriteFile(".claude/commands/autospec.plan.md", []byte("test"), 0o644))
	require.NoError(t, os.WriteFile(".claude/commands/custom.md", []byte("test"), 0o644))

	targets := []CleanTarget{
		{Path: ".claude/commands/autospec.plan.md", Type: TypeFile, Description: "test"},
	}

	results := RemoveFiles(targets)

	assert.Len(t, results, 1)
	assert.True(t, results[0].Success)

	// .claude/commands should still exist (has other files)
	_, err := os.Stat(".claude/commands")
	assert.NoError(t, err)

	// custom.md should still exist
	_, err = os.Stat(".claude/commands/custom.md")
	assert.NoError(t, err)
}

func TestCleanTarget_TypeConstants(t *testing.T) {
	assert.Equal(t, TargetType("directory"), TypeDirectory)
	assert.Equal(t, TargetType("file"), TypeFile)
}

func TestGetSpecsTarget_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	require.NoError(t, os.Chdir(tmpDir))

	// Create specs directory
	require.NoError(t, os.MkdirAll("specs", 0o755))

	target, exists := GetSpecsTarget()
	assert.True(t, exists)
	assert.Equal(t, "specs", target.Path)
	assert.Equal(t, TypeDirectory, target.Type)
	assert.NotEmpty(t, target.Description)
}

func TestGetSpecsTarget_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	require.NoError(t, os.Chdir(tmpDir))

	// No specs directory
	target, exists := GetSpecsTarget()
	assert.False(t, exists)
	assert.Empty(t, target.Path)
}

func TestFindAutospecFiles_MixedScenario(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	require.NoError(t, os.Chdir(tmpDir))

	// Create a realistic project structure
	require.NoError(t, os.MkdirAll(".autospec/memory", 0o755))
	require.NoError(t, os.MkdirAll(".autospec/scripts", 0o755))
	require.NoError(t, os.WriteFile(".autospec/config.yml", []byte("claude_cmd: claude"), 0o644))
	require.NoError(t, os.WriteFile(".autospec/memory/constitution.yaml", []byte("principles:"), 0o644))

	require.NoError(t, os.MkdirAll("specs/001-feature", 0o755))
	require.NoError(t, os.WriteFile("specs/001-feature/spec.yaml", []byte("feature:"), 0o644))

	require.NoError(t, os.MkdirAll(".claude/commands", 0o755))
	require.NoError(t, os.WriteFile(".claude/commands/autospec.plan.md", []byte("plan"), 0o644))
	require.NoError(t, os.WriteFile(".claude/commands/autospec.tasks.md", []byte("tasks"), 0o644))
	require.NoError(t, os.WriteFile(".claude/commands/autospec.implement.md", []byte("implement"), 0o644))
	require.NoError(t, os.WriteFile(".claude/commands/custom-command.md", []byte("custom"), 0o644))

	// Also create some non-autospec files
	require.NoError(t, os.MkdirAll("src", 0o755))
	require.NoError(t, os.WriteFile("src/main.go", []byte("package main"), 0o644))

	targets, err := FindAutospecFiles(false)
	require.NoError(t, err)

	// Should find: .autospec (dir), specs (dir), and 3 autospec command files
	assert.Len(t, targets, 5)

	// Count by type
	var pathsFound []string
	for _, target := range targets {
		pathsFound = append(pathsFound, target.Path)
	}

	assert.Contains(t, pathsFound, ".autospec")
	assert.Contains(t, pathsFound, "specs")
	assert.Contains(t, pathsFound, filepath.Join(".claude", "commands", "autospec.plan.md"))
	assert.Contains(t, pathsFound, filepath.Join(".claude", "commands", "autospec.tasks.md"))
	assert.Contains(t, pathsFound, filepath.Join(".claude", "commands", "autospec.implement.md"))

	// Should NOT include custom-command.md or src
	for _, target := range targets {
		assert.NotContains(t, target.Path, "custom-command")
		assert.NotContains(t, target.Path, "src")
	}
}
