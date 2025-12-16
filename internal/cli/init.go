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
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize autospec configuration and commands",
	Long: `Initialize autospec with everything needed to get started.

This command:
  1. Installs command templates to .claude/commands/ (automatic)
  2. Creates user-level configuration at ~/.config/autospec/config.yml

If config already exists, it is left unchanged (use --force to overwrite).

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

  # Overwrite existing config with defaults
  autospec init --force`,
	RunE: runInit,
}

func init() {
	initCmd.GroupID = GroupGettingStarted
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolP("project", "p", false, "Create project-level config (.autospec/config.yml)")
	initCmd.Flags().BoolP("force", "f", false, "Overwrite existing config with defaults")
	// Keep --global as hidden alias for backward compatibility
	initCmd.Flags().BoolP("global", "g", false, "Deprecated: use default behavior instead (creates user-level config)")
	initCmd.Flags().MarkHidden("global")
}

func runInit(cmd *cobra.Command, args []string) error {
	project, _ := cmd.Flags().GetBool("project")
	force, _ := cmd.Flags().GetBool("force")
	out := cmd.OutOrStdout()

	if err := installCommandTemplates(out); err != nil {
		return err
	}

	if err := initializeConfig(out, project, force); err != nil {
		return err
	}

	constitutionExists := handleConstitution(out)
	checkGitignore(out)
	printSummary(out, constitutionExists)
	return nil
}

// installCommandTemplates installs command templates and prints status
func installCommandTemplates(out io.Writer) error {
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
	return nil
}

// initializeConfig creates or updates config file
func initializeConfig(out io.Writer, project, force bool) error {
	configPath, err := getConfigPath(project)
	if err != nil {
		return err
	}

	configExists := fileExistsCheck(configPath)

	if configExists && !force {
		fmt.Fprintf(out, "✓ Config: exists at %s\n", configPath)
		return nil
	}

	if err := writeDefaultConfig(configPath); err != nil {
		return err
	}

	if configExists {
		fmt.Fprintf(out, "✓ Config: overwritten at %s\n", configPath)
	} else {
		fmt.Fprintf(out, "✓ Config: created at %s\n", configPath)
	}
	return nil
}

// getConfigPath returns the appropriate config path based on project flag
func getConfigPath(project bool) (string, error) {
	if project {
		return config.ProjectConfigPath(), nil
	}
	configPath, err := config.UserConfigPath()
	if err != nil {
		return "", fmt.Errorf("failed to get user config path: %w", err)
	}
	return configPath, nil
}

// writeDefaultConfig writes the default configuration to the given path
func writeDefaultConfig(configPath string) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	defaults := config.GetDefaults()
	data, err := yaml.Marshal(defaults)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	header := "# Autospec Configuration\n# See 'autospec config show' for all available options\n\n"
	if err := os.WriteFile(configPath, []byte(header+string(data)), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
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

func promptYesNo(cmd *cobra.Command, question string) bool {
	fmt.Fprintf(cmd.OutOrStdout(), "%s [y/N]: ", question)

	reader := bufio.NewReader(cmd.InOrStdin())
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	return answer == "y" || answer == "yes"
}

// handleConstitution checks for existing constitution and copies it if needed.
// Returns true if constitution exists (either copied or already present).
func handleConstitution(out io.Writer) bool {
	// Autospec paths (where we want the constitution)
	autospecPaths := []string{
		".autospec/memory/constitution.yaml",
		".autospec/memory/constitution.yml",
	}

	// Legacy specify paths (source for migration)
	legacyPaths := []string{
		".specify/memory/constitution.yaml",
		".specify/memory/constitution.yml",
	}

	// Check if any autospec constitution already exists
	for _, path := range autospecPaths {
		if _, err := os.Stat(path); err == nil {
			fmt.Fprintf(out, "✓ Constitution: found at %s\n", path)
			return true
		}
	}

	// Check if any legacy specify constitution exists
	for _, legacyPath := range legacyPaths {
		if _, err := os.Stat(legacyPath); err == nil {
			// Copy legacy constitution to autospec location (prefer .yaml)
			destPath := autospecPaths[0]
			if err := copyConstitution(legacyPath, destPath); err != nil {
				fmt.Fprintf(out, "⚠ Constitution: failed to copy from %s: %v\n", legacyPath, err)
				return false
			}
			fmt.Fprintf(out, "✓ Constitution: copied from %s → %s\n", legacyPath, destPath)
			return true
		}
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
