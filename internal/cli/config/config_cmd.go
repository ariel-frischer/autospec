package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
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

func init() {
	configCmd.GroupID = shared.GroupConfiguration

	// Add subcommands
	configCmd.AddCommand(configShowCmd)

	// Show command flags
	configShowCmd.Flags().Bool("json", false, "Output in JSON format")
	configShowCmd.Flags().Bool("yaml", true, "Output in YAML format (default)")
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
		"custom_claude_cmd":  cfg.CustomClaudeCmd,
		"max_retries":        cfg.MaxRetries,
		"specs_dir":          cfg.SpecsDir,
		"state_dir":          cfg.StateDir,
		"skip_preflight":     cfg.SkipPreflight,
		"timeout":            cfg.Timeout,
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

// fileExistsCheck returns true if the file exists
func fileExistsCheck(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
