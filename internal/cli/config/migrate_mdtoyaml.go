package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/yaml"
	"github.com/spf13/cobra"
)

var migrateMdToYamlCmd = &cobra.Command{
	Use:   "md-to-yaml <path>",
	Short: "Convert markdown artifacts to YAML",
	Long: `Convert markdown spec artifacts to YAML format.

The path can be:
- A single markdown file (e.g., spec.md)
- A directory containing markdown files (e.g., specs/007-feature/)

Supported artifact types:
- spec.md → spec.yaml
- plan.md → plan.yaml
- tasks.md → tasks.yaml
- checklist.md → checklist.yaml
- analysis.md → analysis.yaml
- constitution.md → constitution.yaml

Existing YAML files are preserved (not overwritten).

Example:
  autospec migrate md-to-yaml specs/007-feature/spec.md
  autospec migrate md-to-yaml specs/007-feature/`,
	Args: cobra.ExactArgs(1),
	RunE: runMigrateMdToYaml,
}

var migrateForce bool

func init() {
	migrateCmd.AddCommand(migrateMdToYamlCmd)
	migrateMdToYamlCmd.Flags().BoolVarP(&migrateForce, "force", "f", false, "Overwrite existing YAML files")
}

func runMigrateMdToYaml(cmd *cobra.Command, args []string) error {
	path := args[0]

	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path not found: %s", path)
	}

	if info.IsDir() {
		return migrateDirectory(cmd, path)
	}

	return migrateFile(cmd, path)
}

func migrateFile(cmd *cobra.Command, mdPath string) error {
	// Check if it's a markdown file
	ext := filepath.Ext(mdPath)
	if ext != ".md" {
		return fmt.Errorf("not a markdown file: %s", mdPath)
	}

	// Detect artifact type
	filename := filepath.Base(mdPath)
	artifactType := yaml.DetectArtifactType(filename)
	if artifactType == "unknown" {
		return fmt.Errorf("unknown artifact type: %s (expected spec.md, plan.md, tasks.md, etc.)", filename)
	}

	yamlPath, err := yaml.MigrateFile(mdPath)
	if err != nil {
		return fmt.Errorf("migrating %s to YAML: %w", mdPath, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Converted %s → %s\n", mdPath, yamlPath)
	return nil
}

func migrateDirectory(cmd *cobra.Command, dir string) error {
	fmt.Fprintf(cmd.OutOrStdout(), "Migrating markdown files in %s...\n\n", dir)

	migrated, errors := yaml.MigrateDirectory(dir)

	// Report successes
	if len(migrated) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Converted:")
		for _, path := range migrated {
			fmt.Fprintf(cmd.OutOrStdout(), "  ✓ %s\n", path)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Report errors
	if len(errors) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Skipped:")
		for _, err := range errors {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %v\n", err)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	if len(migrated) == 0 && len(errors) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No markdown artifacts found to migrate.")
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Done: %d converted, %d skipped\n", len(migrated), len(errors))

	return nil
}
