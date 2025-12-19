// Package cli_test tests the migrate-md-to-yaml command for converting legacy Markdown artifacts to YAML format.
// Related: internal/cli/config/migrate_mdtoyaml.go
// Tags: cli, migrate, markdown, yaml, conversion, legacy
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getMigrateCmd finds the migrate command from rootCmd
func getMigrateCmd() *cobra.Command {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "migrate" {
			return cmd
		}
	}
	return nil
}

// getMigrateMdToYamlCmd finds the "migrate md-to-yaml" subcommand
func getMigrateMdToYamlCmd() *cobra.Command {
	migrateCmd := getMigrateCmd()
	if migrateCmd == nil {
		return nil
	}
	for _, cmd := range migrateCmd.Commands() {
		if cmd.Use == "md-to-yaml <path>" {
			return cmd
		}
	}
	return nil
}

func TestMigrateCmdRegistration(t *testing.T) {
	cmd := getMigrateCmd()
	assert.NotNil(t, cmd, "migrate command should be registered")
}

func TestMigrateMdToYamlCmdRegistration(t *testing.T) {
	cmd := getMigrateMdToYamlCmd()
	assert.NotNil(t, cmd, "migrate md-to-yaml subcommand should be registered")
}

func TestMigrateMdToYaml_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a spec.md file
	specMd := `# Feature Specification: Test

## Description

Test spec.

## User Stories

### US-001: Test Story (P1)

**As a** user
**I want** to test
**So that** it works

## Requirements

### Functional Requirements

- FR-001: System works
`
	specPath := filepath.Join(tmpDir, "spec.md")
	err := os.WriteFile(specPath, []byte(specMd), 0644)
	require.NoError(t, err)

	cmd := getMigrateMdToYamlCmd()
	require.NotNil(t, cmd, "migrate md-to-yaml command must exist")

	// Run migration
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err = cmd.RunE(cmd, []string{specPath})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Converted")
	assert.Contains(t, output, "spec.yaml")

	// Verify YAML file was created
	yamlPath := filepath.Join(tmpDir, "spec.yaml")
	_, err = os.Stat(yamlPath)
	assert.NoError(t, err, "spec.yaml should exist")
}

func TestMigrateMdToYaml_Directory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create spec.md and plan.md
	specMd := `# Feature Specification: Test

## Description

Test.

## User Stories

### US-001: Test (P1)

**As a** user
**I want** to test
**So that** it works

## Requirements

### Functional Requirements

- FR-001: Works
`
	planMd := `# Implementation Plan: Test

**Branch**: test

## Summary

Test plan.
`
	err := os.WriteFile(filepath.Join(tmpDir, "spec.md"), []byte(specMd), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "plan.md"), []byte(planMd), 0644)
	require.NoError(t, err)

	cmd := getMigrateMdToYamlCmd()
	require.NotNil(t, cmd, "migrate md-to-yaml command must exist")

	// Run migration
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err = cmd.RunE(cmd, []string{tmpDir})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Done:")

	// Verify files were created
	_, err = os.Stat(filepath.Join(tmpDir, "spec.yaml"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(tmpDir, "plan.yaml"))
	assert.NoError(t, err)
}

func TestMigrateMdToYaml_NonExistentPath(t *testing.T) {
	cmd := getMigrateMdToYamlCmd()
	require.NotNil(t, cmd, "migrate md-to-yaml command must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.RunE(cmd, []string{"/nonexistent/path"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMigrateMdToYaml_NonMarkdownFile(t *testing.T) {
	tmpDir := t.TempDir()
	txtPath := filepath.Join(tmpDir, "readme.txt")
	err := os.WriteFile(txtPath, []byte("content"), 0644)
	require.NoError(t, err)

	cmd := getMigrateMdToYamlCmd()
	require.NotNil(t, cmd, "migrate md-to-yaml command must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err = cmd.RunE(cmd, []string{txtPath})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a markdown file")
}

func TestMigrateMdToYaml_UnknownArtifactType(t *testing.T) {
	tmpDir := t.TempDir()
	unknownPath := filepath.Join(tmpDir, "readme.md")
	err := os.WriteFile(unknownPath, []byte("# README\n\nContent"), 0644)
	require.NoError(t, err)

	cmd := getMigrateMdToYamlCmd()
	require.NotNil(t, cmd, "migrate md-to-yaml command must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err = cmd.RunE(cmd, []string{unknownPath})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown artifact type")
}
