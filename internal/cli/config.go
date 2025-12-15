package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage autospec configuration",
	Long: `Manage autospec configuration settings.

Configuration is loaded with the following priority (highest to lowest):
  1. Environment variables (AUTOSPEC_*)
  2. Project config (.autospec/config.yml)
  3. User config (~/.config/autospec/config.yml)
  4. Built-in defaults`,
	Example: `  # Show current configuration
  autospec config show

  # Show configuration as JSON
  autospec config show --json

  # Migrate legacy JSON config to YAML
  autospec config migrate

  # Preview migration without making changes
  autospec config migrate --dry-run

  # Initialize configuration
  autospec init`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current effective configuration",
	Long: `Display the current effective configuration values.

Shows the merged result of defaults, user config, project config, and
environment variables. Use --json or --yaml to control output format.`,
	Example: `  # Show configuration in YAML format (default)
  autospec config show

  # Show configuration in JSON format
  autospec config show --json`,
	RunE: runConfigShow,
}

var configMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate JSON configuration to YAML format",
	Long: `Migrate legacy JSON configuration files to the new YAML format.

By default, migrates both user-level and project-level configurations.
Use --user or --project to migrate only one level.

The original JSON files are renamed to .bak after successful migration.`,
	Example: `  # Migrate all JSON configs to YAML
  autospec config migrate

  # Preview what would be migrated
  autospec config migrate --dry-run

  # Migrate user config only
  autospec config migrate --user

  # Migrate project config only
  autospec config migrate --project`,
	RunE: runConfigMigrate,
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Add subcommands
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configMigrateCmd)

	// Show command flags
	configShowCmd.Flags().Bool("json", false, "Output in JSON format")
	configShowCmd.Flags().Bool("yaml", true, "Output in YAML format (default)")

	// Migrate command flags
	configMigrateCmd.Flags().Bool("dry-run", false, "Preview migration without making changes")
	configMigrateCmd.Flags().Bool("user", false, "Migrate user-level config only")
	configMigrateCmd.Flags().Bool("project", false, "Migrate project-level config only")
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	useJSON, _ := cmd.Flags().GetBool("json")

	// Load configuration with warnings suppressed
	cfg, err := config.LoadWithOptions(config.LoadOptions{
		SkipWarnings: true,
	})
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Convert to map for output
	configMap := map[string]interface{}{
		"claude_cmd":         cfg.ClaudeCmd,
		"claude_args":        cfg.ClaudeArgs,
		"use_api_key":        cfg.UseAPIKey,
		"custom_claude_cmd":  cfg.CustomClaudeCmd,
		"max_retries":        cfg.MaxRetries,
		"specs_dir":          cfg.SpecsDir,
		"state_dir":          cfg.StateDir,
		"skip_preflight":     cfg.SkipPreflight,
		"timeout":            cfg.Timeout,
		"show_progress":      cfg.ShowProgress,
		"skip_confirmations": cfg.SkipConfirmations,
	}

	// Show config paths
	userPath, _ := config.UserConfigPath()
	projectPath := config.ProjectConfigPath()

	fmt.Fprintf(out, "# Configuration Sources\n")
	fmt.Fprintf(out, "# User config:    %s\n", userPath)
	fmt.Fprintf(out, "# Project config: %s\n", projectPath)
	fmt.Fprintf(out, "\n")

	if useJSON {
		data, err := json.MarshalIndent(configMap, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to serialize config: %w", err)
		}
		fmt.Fprintln(out, string(data))
	} else {
		data, err := yaml.Marshal(configMap)
		if err != nil {
			return fmt.Errorf("failed to serialize config: %w", err)
		}
		fmt.Fprint(out, string(data))
	}

	return nil
}

func runConfigMigrate(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	userOnly, _ := cmd.Flags().GetBool("user")
	projectOnly, _ := cmd.Flags().GetBool("project")

	// Default to migrating both if neither flag is set
	migrateUser := !projectOnly || userOnly
	migrateProject := !userOnly || projectOnly

	if dryRun {
		fmt.Fprintln(out, "Dry run mode - no changes will be made")
		fmt.Fprintln(out)
	}

	var migrated, skipped int

	// Migrate user config
	if migrateUser {
		result, err := config.MigrateUserConfig(dryRun)
		if err != nil {
			return fmt.Errorf("failed to migrate user config: %w", err)
		}

		if result.Success {
			fmt.Fprintf(out, "✓ %s\n", result.Message)
			migrated++

			// Remove legacy file if not dry run
			if !dryRun {
				if err := config.RemoveLegacyConfig(result.SourcePath, dryRun); err != nil {
					fmt.Fprintf(out, "  Warning: failed to backup legacy file: %v\n", err)
				} else {
					fmt.Fprintf(out, "  Legacy file backed up to %s.bak\n", result.SourcePath)
				}
			}
		} else {
			fmt.Fprintf(out, "- %s\n", result.Message)
			skipped++
		}
	}

	// Migrate project config
	if migrateProject {
		result, err := config.MigrateProjectConfig(dryRun)
		if err != nil {
			return fmt.Errorf("failed to migrate project config: %w", err)
		}

		if result.Success {
			fmt.Fprintf(out, "✓ %s\n", result.Message)
			migrated++

			// Remove legacy file if not dry run
			if !dryRun {
				if err := config.RemoveLegacyConfig(result.SourcePath, dryRun); err != nil {
					fmt.Fprintf(out, "  Warning: failed to backup legacy file: %v\n", err)
				} else {
					fmt.Fprintf(out, "  Legacy file backed up to %s.bak\n", result.SourcePath)
				}
			}
		} else {
			fmt.Fprintf(out, "- %s\n", result.Message)
			skipped++
		}
	}

	// Summary
	fmt.Fprintln(out)
	if migrated > 0 {
		if dryRun {
			fmt.Fprintf(out, "Would migrate %d config file(s)\n", migrated)
		} else {
			fmt.Fprintf(out, "Migrated %d config file(s)\n", migrated)
		}
	}

	if migrated == 0 && skipped > 0 {
		fmt.Fprintln(out, "No JSON configs found to migrate.")
		// Check if YAML configs exist
		if userPath, _ := config.UserConfigPath(); fileExistsCheck(userPath) {
			fmt.Fprintf(out, "User config already exists at: %s\n", userPath)
		}
		if fileExistsCheck(config.ProjectConfigPath()) {
			fmt.Fprintf(out, "Project config already exists at: %s\n", config.ProjectConfigPath())
		}
	}

	return nil
}

// fileExistsCheck returns true if the file exists
func fileExistsCheck(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
