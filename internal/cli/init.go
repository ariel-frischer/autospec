package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ariel-frischer/autospec/internal/commands"
	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize autospec configuration, commands, and scripts",
	Long: `Initialize autospec with everything needed to get started.

This command:
  1. Installs command templates to .claude/commands/ (automatic)
  2. Installs helper scripts to .autospec/scripts/ (automatic)
  3. Creates user-level configuration at ~/.config/autospec/config.yml

By default, creates user-level config which applies to all your projects.
Use --project to create project-specific config that overrides user settings.

Configuration precedence (highest to lowest):
  1. Environment variables (AUTOSPEC_*)
  2. Project config (.autospec/config.yml)
  3. User config (~/.config/autospec/config.yml)
  4. Built-in defaults`,
	Example: `  # Initialize with user-level config (recommended for first-time setup)
  autospec init

  # Create project-specific config (overrides user config)
  autospec init --project

  # Overwrite existing config without prompting
  autospec init --force`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolP("project", "p", false, "Create project-level config (.autospec/config.yml)")
	initCmd.Flags().BoolP("force", "f", false, "Overwrite existing config without prompting")
	// Keep --global as hidden alias for backward compatibility
	initCmd.Flags().BoolP("global", "g", false, "Deprecated: use default behavior instead (creates user-level config)")
	initCmd.Flags().MarkHidden("global")
}

func runInit(cmd *cobra.Command, args []string) error {
	project, _ := cmd.Flags().GetBool("project")
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
	var configPath string
	var configLabel string

	if project {
		// Project-level config
		configPath = config.ProjectConfigPath()
		configLabel = "project"
	} else {
		// User-level config (default)
		var err error
		configPath, err = config.UserConfigPath()
		if err != nil {
			return fmt.Errorf("failed to get user config path: %w", err)
		}
		configLabel = "user"
	}

	configExists := false
	var existingConfig map[string]interface{}
	if data, err := os.ReadFile(configPath); err == nil {
		configExists = true
		yaml.Unmarshal(data, &existingConfig)
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
			// Still handle constitution even if config unchanged
			constitutionExists := handleConstitution(out)
			printSummary(out, constitutionExists)
			return nil
		}

		// Interactive config update
		if err := updateConfigInteractiveYAML(cmd, configPath, existingConfig); err != nil {
			return err
		}
		fmt.Fprintf(out, "✓ Config: updated\n")
	} else {
		// Create new config with defaults
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		defaults := config.GetDefaults()
		data, err := yaml.Marshal(defaults)
		if err != nil {
			return fmt.Errorf("failed to serialize config: %w", err)
		}

		// Add a header comment to the YAML file
		header := "# Autospec Configuration\n# See 'autospec config show' for all available options\n\n"
		if err := os.WriteFile(configPath, []byte(header+string(data)), 0644); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		if configExists {
			fmt.Fprintf(out, "✓ Config: overwritten at %s\n", configPath)
		} else {
			fmt.Fprintf(out, "✓ Config: created at %s\n", configPath)
		}
	}

	// Step 4: Handle constitution
	constitutionExists := handleConstitution(out)

	// Step 5: Check .gitignore for .autospec
	checkGitignore(out)

	printSummary(out, constitutionExists)
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

func updateConfigInteractiveYAML(cmd *cobra.Command, configPath string, existing map[string]interface{}) error {
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

	// Write updated config as YAML
	data, err := yaml.Marshal(existing)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	header := "# Autospec Configuration\n# See 'autospec config show' for all available options\n\n"
	return os.WriteFile(configPath, []byte(header+string(data)), 0644)
}

// handleConstitution checks for existing constitution and copies it if needed.
// Returns true if constitution exists (either copied or already present).
func handleConstitution(out io.Writer) bool {
	autospecConstitution := workflow.ConstitutionPath
	legacyConstitution := workflow.LegacyConstitutionPath

	// Check if autospec constitution already exists
	if _, err := os.Stat(autospecConstitution); err == nil {
		fmt.Fprintf(out, "✓ Constitution: found at %s\n", autospecConstitution)
		return true
	}

	// Check if legacy specify constitution exists
	if _, err := os.Stat(legacyConstitution); err == nil {
		// Copy legacy constitution to autospec location
		if err := copyConstitution(legacyConstitution, autospecConstitution); err != nil {
			fmt.Fprintf(out, "⚠ Constitution: failed to copy from %s: %v\n", legacyConstitution, err)
			return false
		}
		fmt.Fprintf(out, "✓ Constitution: copied from %s → %s\n", legacyConstitution, autospecConstitution)
		return true
	}

	// No constitution found
	fmt.Fprintf(out, "⚠ Constitution: not found\n")
	return false
}

// copyConstitution copies the constitution file from src to dst
func copyConstitution(src, dst string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Read source file
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}

	// Write to destination
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("failed to write destination: %w", err)
	}

	return nil
}

// checkGitignore checks if .gitignore exists and contains .autospec entry.
// If not, prints a recommendation to add it.
func checkGitignore(out io.Writer) {
	gitignorePath := ".gitignore"

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		// .gitignore doesn't exist - no recommendation needed
		return
	}

	content := string(data)
	// Check for .autospec or .autospec/ in the file
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == ".autospec" || line == ".autospec/" || strings.HasPrefix(line, ".autospec/") {
			// Already has .autospec entry
			return
		}
	}

	// .gitignore exists but doesn't have .autospec
	fmt.Fprintf(out, "\n💡 Recommendation: Consider adding .autospec/ to your .gitignore\n")
	fmt.Fprintf(out, "   This prevents accidental commit of local configuration and state files.\n")
}

func printSummary(out io.Writer, constitutionExists bool) {
	fmt.Fprintf(out, "\n")

	if !constitutionExists {
		fmt.Fprintf(out, "⚠ IMPORTANT: You MUST create a constitution before using autospec.\n")
		fmt.Fprintf(out, "Run the following command to get started:\n\n")
		fmt.Fprintf(out, "  autospec constitution\n\n")
		fmt.Fprintf(out, "The constitution defines your project's principles and guidelines.\n")
		fmt.Fprintf(out, "Without it, workflow commands (specify, plan, tasks, implement) will fail.\n\n")
	}

	fmt.Fprintf(out, "Quick start:\n")
	fmt.Fprintf(out, "  1. autospec specify \"Add user authentication\"\n")
	fmt.Fprintf(out, "  2. Review the generated spec\n")
	fmt.Fprintf(out, "  # -pti is short for --plan --tasks --implement\n")
	fmt.Fprintf(out, "  3. autospec run -pti\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "Or run all steps at once (specify → plan → tasks → implement):\n")
	fmt.Fprintf(out, "  autospec all \"Add user authentication\"\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "Run 'autospec doctor' to verify dependencies.\n")
	fmt.Fprintf(out, "Run 'autospec --help' for all commands.\n")
}
