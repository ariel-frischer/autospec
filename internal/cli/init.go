package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/auto-claude-speckit/internal/commands"
	"github.com/anthropics/auto-claude-speckit/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize autospec configuration, commands, and scripts",
	Long: `Initialize autospec with everything needed to get started.

This command:
  1. Installs command templates to .claude/commands/ (automatic)
  2. Installs helper scripts to .autospec/scripts/ (automatic)
  3. Creates or updates configuration in .autospec/config.json

Commands and scripts are installed automatically without prompting.
For config, you'll be prompted if a config already exists.

Examples:
  autospec init              # Interactive setup
  autospec init --global     # Create global config (~/.autospec/config.json)
  autospec init --force      # Overwrite existing config without prompting`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolP("global", "g", false, "Create global config (~/.autospec/)")
	initCmd.Flags().BoolP("force", "f", false, "Overwrite existing config without prompting")
}

func runInit(cmd *cobra.Command, args []string) error {
	global, _ := cmd.Flags().GetBool("global")
	force, _ := cmd.Flags().GetBool("force")

	out := cmd.OutOrStdout()

	// Step 1: Install commands (silent, no prompt)
	cmdDir := commands.GetDefaultCommandsDir()
	cmdResults, err := commands.InstallTemplates(cmdDir)
	if err != nil {
		return fmt.Errorf("failed to install commands: %w", err)
	}

	cmdInstalled, cmdUpdated := countResults(cmdResults)
	if cmdInstalled+cmdUpdated > 0 {
		fmt.Fprintf(out, "✓ Commands: %d installed, %d updated → %s/\n", cmdInstalled, cmdUpdated, cmdDir)
	} else {
		fmt.Fprintf(out, "✓ Commands: up to date\n")
	}

	// Step 2: Install scripts (silent, no prompt)
	scriptsDir := commands.GetDefaultScriptsDir()
	scriptResults, err := commands.InstallScripts(scriptsDir)
	if err != nil {
		return fmt.Errorf("failed to install scripts: %w", err)
	}

	scriptInstalled, scriptUpdated := countScriptResults(scriptResults)
	if scriptInstalled+scriptUpdated > 0 {
		fmt.Fprintf(out, "✓ Scripts: %d installed, %d updated → %s/\n", scriptInstalled, scriptUpdated, scriptsDir)
	} else {
		fmt.Fprintf(out, "✓ Scripts: up to date\n")
	}

	// Step 3: Handle config
	configPath := ".autospec/config.json"
	configLabel := "local"
	if global {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(homeDir, ".autospec", "config.json")
		configLabel = "global"
	}

	configExists := false
	var existingConfig map[string]interface{}
	if data, err := os.ReadFile(configPath); err == nil {
		configExists = true
		json.Unmarshal(data, &existingConfig)
	}

	if configExists && !force {
		// Prompt user
		label := configLabel
		if len(label) > 0 {
			label = strings.ToUpper(label[:1]) + label[1:]
		}
		fmt.Fprintf(out, "\n%s config exists at %s\n", label, configPath)
		if !promptYesNo(cmd, "Update config?") {
			fmt.Fprintf(out, "✓ Config: unchanged\n")
			printSummary(out)
			return nil
		}

		// Interactive config update
		if err := updateConfigInteractive(cmd, configPath, existingConfig); err != nil {
			return err
		}
		fmt.Fprintf(out, "✓ Config: updated\n")
	} else {
		// Create new config with defaults
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		defaults := config.GetDefaults()
		data, _ := json.MarshalIndent(defaults, "", "  ")
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		if configExists {
			fmt.Fprintf(out, "✓ Config: overwritten at %s\n", configPath)
		} else {
			fmt.Fprintf(out, "✓ Config: created at %s\n", configPath)
		}
	}

	printSummary(out)
	return nil
}

func countResults(results []commands.InstallResult) (installed, updated int) {
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

func countScriptResults(results []commands.ScriptInstallResult) (installed, updated int) {
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

func promptYesNo(cmd *cobra.Command, question string) bool {
	fmt.Fprintf(cmd.OutOrStdout(), "%s [y/N]: ", question)

	reader := bufio.NewReader(cmd.InOrStdin())
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	return answer == "y" || answer == "yes"
}

func updateConfigInteractive(cmd *cobra.Command, configPath string, existing map[string]interface{}) error {
	out := cmd.OutOrStdout()
	defaults := config.GetDefaults()

	// Merge existing with defaults (existing takes precedence)
	for k, v := range defaults {
		if _, exists := existing[k]; !exists {
			existing[k] = v
		}
	}

	fmt.Fprintf(out, "\nCurrent settings (press Enter to keep, or type new value):\n")

	// Key settings to prompt for
	settings := []struct {
		key     string
		desc    string
		current interface{}
	}{
		{"specs_dir", "Specs directory", existing["specs_dir"]},
		{"max_retries", "Max retries (1-10)", existing["max_retries"]},
		{"timeout", "Timeout in seconds (0=disabled)", existing["timeout"]},
		{"skip_preflight", "Skip preflight checks (true/false)", existing["skip_preflight"]},
		{"show_progress", "Show progress indicators (true/false)", existing["show_progress"]},
	}

	reader := bufio.NewReader(cmd.InOrStdin())

	for _, s := range settings {
		fmt.Fprintf(out, "  %s [%v]: ", s.desc, s.current)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input != "" {
			// Parse based on type
			switch s.current.(type) {
			case bool:
				existing[s.key] = input == "true" || input == "yes" || input == "1"
			case float64, int:
				var num int
				fmt.Sscanf(input, "%d", &num)
				existing[s.key] = num
			default:
				existing[s.key] = input
			}
		}
	}

	// Write updated config
	data, _ := json.MarshalIndent(existing, "", "  ")
	return os.WriteFile(configPath, data, 0644)
}

func printSummary(out io.Writer) {
	fmt.Fprintf(out, "\nReady! Available commands:\n")
	fmt.Fprintf(out, "  /autospec.specify  - Create feature specification\n")
	fmt.Fprintf(out, "  /autospec.plan     - Generate implementation plan\n")
	fmt.Fprintf(out, "  /autospec.tasks    - Create task breakdown\n")
	fmt.Fprintf(out, "\nRun 'autospec doctor' to verify dependencies.\n")
}
