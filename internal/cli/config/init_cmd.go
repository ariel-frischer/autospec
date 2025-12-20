package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/cliagent"
	"github.com/ariel-frischer/autospec/internal/commands"
	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/history"
	"github.com/ariel-frischer/autospec/internal/lifecycle"
	"github.com/ariel-frischer/autospec/internal/notify"
	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
	initCmd.GroupID = shared.GroupGettingStarted
	initCmd.Flags().BoolP("project", "p", false, "Create project-level config (.autospec/config.yml)")
	initCmd.Flags().BoolP("force", "f", false, "Overwrite existing config with defaults")
	initCmd.Flags().Bool("no-agents", false, "Skip agent configuration prompt")
	// Keep --global as hidden alias for backward compatibility
	initCmd.Flags().BoolP("global", "g", false, "Deprecated: use default behavior instead (creates user-level config)")
	initCmd.Flags().MarkHidden("global")
}

func runInit(cmd *cobra.Command, args []string) error {
	project, _ := cmd.Flags().GetBool("project")
	force, _ := cmd.Flags().GetBool("force")
	noAgents, _ := cmd.Flags().GetBool("no-agents")
	out := cmd.OutOrStdout()

	if err := installCommandTemplates(out); err != nil {
		return fmt.Errorf("installing command templates: %w", err)
	}

	if err := initializeConfig(out, project, force); err != nil {
		return fmt.Errorf("initializing config: %w", err)
	}

	// Handle agent selection and configuration
	if err := handleAgentConfiguration(cmd, out, project, noAgents); err != nil {
		return fmt.Errorf("configuring agents: %w", err)
	}

	constitutionExists := handleConstitution(out)
	checkGitignore(out)

	// If no constitution, prompt user to create one
	if !constitutionExists {
		if promptYesNoDefaultYes(cmd, "\nWould you like to create a constitution now?") {
			configPath, _ := cmd.Flags().GetString("config")
			if runConstitutionFromInit(cmd, configPath) {
				constitutionExists = true
			}
		}
	}

	// Prompt for worktree setup script generation if not already present
	worktreeScriptPath := filepath.Join(".autospec", "scripts", "setup-worktree.sh")
	if !fileExistsCheck(worktreeScriptPath) {
		if promptYesNo(cmd, "\nDo you intend to use git worktrees for parallelization?") {
			configPath, _ := cmd.Flags().GetString("config")
			runWorktreeGenScriptFromInit(cmd, configPath)
		}
	} else {
		fmt.Fprintf(out, "✓ Worktree setup script: already exists at %s\n", worktreeScriptPath)
	}

	printSummary(out, constitutionExists)
	return nil
}

// handleAgentConfiguration handles the agent selection and configuration flow.
// If noAgents is true, the prompt is skipped. In non-interactive mode without
// --no-agents, it returns an error with a helpful message.
func handleAgentConfiguration(cmd *cobra.Command, out io.Writer, project, noAgents bool) error {
	if noAgents {
		fmt.Fprintln(out, "⏭ Agent configuration: skipped (--no-agents)")
		return nil
	}

	// Check if stdin is a terminal
	if !isTerminal() {
		return fmt.Errorf("agent selection requires an interactive terminal; " +
			"use --no-agents for non-interactive environments")
	}

	// Load config to get DefaultAgents for pre-selection
	configPath, err := getConfigPath(project)
	if err != nil {
		return fmt.Errorf("getting config path: %w", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		// Continue with empty defaults if config load fails
		cfg = &config.Configuration{}
	}

	// Get agents with defaults pre-selected
	agents := GetSupportedAgentsWithDefaults(cfg.DefaultAgents)

	// Run agent selection prompt
	selected := promptAgentSelection(cmd.InOrStdin(), out, agents)

	// Configure selected agents and save preferences
	return configureSelectedAgents(out, selected, cfg, configPath)
}

// configureSelectedAgents configures each selected agent and persists preferences.
func configureSelectedAgents(out io.Writer, selected []string, cfg *config.Configuration, configPath string) error {
	if len(selected) == 0 {
		fmt.Fprintln(out, "⚠ Warning: No agents selected. You may need to configure agent permissions manually.")
		return nil
	}

	specsDir := cfg.SpecsDir
	if specsDir == "" {
		specsDir = "specs"
	}

	// Configure each selected agent
	for _, agentName := range selected {
		agent := cliagent.Get(agentName)
		if agent == nil {
			continue
		}

		result, err := cliagent.Configure(agent, ".", specsDir)
		if err != nil {
			fmt.Fprintf(out, "⚠ %s: configuration failed: %v\n", agentDisplayNames[agentName], err)
			continue
		}

		displayAgentConfigResult(out, agentName, result)
	}

	// Persist selected agents to config
	return persistAgentPreferences(out, selected, cfg, configPath)
}

// displayAgentConfigResult displays the configuration result for an agent.
func displayAgentConfigResult(out io.Writer, agentName string, result *cliagent.ConfigResult) {
	displayName := agentDisplayNames[agentName]
	if displayName == "" {
		displayName = agentName
	}

	if result == nil {
		fmt.Fprintf(out, "✓ %s: no configuration needed\n", displayName)
		return
	}

	if result.Warning != "" {
		fmt.Fprintf(out, "⚠ %s: %s\n", displayName, result.Warning)
	}

	if result.AlreadyConfigured {
		fmt.Fprintf(out, "✓ %s: already configured\n", displayName)
		return
	}

	if len(result.PermissionsAdded) > 0 {
		fmt.Fprintf(out, "✓ %s: configured with permissions:\n", displayName)
		for _, perm := range result.PermissionsAdded {
			fmt.Fprintf(out, "    - %s\n", perm)
		}
	}
}

// persistAgentPreferences saves the selected agents to config for future init runs.
func persistAgentPreferences(out io.Writer, selected []string, cfg *config.Configuration, configPath string) error {
	// Update config with new agent preferences
	cfg.DefaultAgents = selected

	// Read existing config file to preserve formatting and comments
	existingContent, err := os.ReadFile(configPath)
	if err != nil {
		// Config doesn't exist yet, nothing to update
		return nil
	}

	// Update the default_agents line in the config file
	newContent := updateDefaultAgentsInConfig(string(existingContent), selected)

	if err := os.WriteFile(configPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("saving agent preferences: %w", err)
	}

	fmt.Fprintf(out, "✓ Agent preferences saved to %s\n", configPath)
	return nil
}

// updateDefaultAgentsInConfig updates the default_agents line in the config content.
func updateDefaultAgentsInConfig(content string, agents []string) string {
	lines := strings.Split(content, "\n")
	found := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "default_agents:") {
			// Replace the line with new agent list
			if len(agents) == 0 {
				lines[i] = "default_agents: []"
			} else {
				lines[i] = fmt.Sprintf("default_agents: [%s]", formatAgentList(agents))
			}
			found = true
			break
		}
	}

	if !found && len(agents) > 0 {
		// Append default_agents at the end if not found
		lines = append(lines, fmt.Sprintf("default_agents: [%s]", formatAgentList(agents)))
	}

	return strings.Join(lines, "\n")
}

// formatAgentList formats a list of agent names for YAML output.
func formatAgentList(agents []string) string {
	quoted := make([]string, len(agents))
	for i, agent := range agents {
		quoted[i] = fmt.Sprintf("%q", agent)
	}
	return strings.Join(quoted, ", ")
}

// isTerminal returns true if stdin is connected to a terminal.
func isTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
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
		return fmt.Errorf("getting config path: %w", err)
	}

	configExists := fileExistsCheck(configPath)

	if configExists && !force {
		fmt.Fprintf(out, "✓ Config: exists at %s\n", configPath)
		return nil
	}

	if err := writeDefaultConfig(configPath); err != nil {
		return fmt.Errorf("writing default config: %w", err)
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

	template := config.GetDefaultConfigTemplate()
	if err := os.WriteFile(configPath, []byte(template), 0644); err != nil {
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

// promptYesNoDefaultYes prompts the user with a question that defaults to yes.
// Empty input (just pressing Enter) returns true.
func promptYesNoDefaultYes(cmd *cobra.Command, question string) bool {
	fmt.Fprintf(cmd.OutOrStdout(), "%s (Y/n): ", question)

	reader := bufio.NewReader(cmd.InOrStdin())
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	// Default to yes: empty string or explicit yes
	return answer == "" || answer == "y" || answer == "yes"
}

// runConstitutionFromInit executes the constitution workflow.
// Returns true if constitution was created successfully.
func runConstitutionFromInit(cmd *cobra.Command, configPath string) bool {
	out := cmd.OutOrStdout()

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(out, "⚠ Failed to load config: %v\n", err)
		return false
	}

	// Create notification handler and history logger
	notifHandler := notify.NewHandler(cfg.Notifications)
	historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)

	fmt.Fprintf(out, "\n")

	// Run constitution with lifecycle wrapper
	err = lifecycle.RunWithHistory(notifHandler, historyLogger, "constitution", "", func() error {
		orch := workflow.NewWorkflowOrchestrator(cfg)
		orch.Executor.NotificationHandler = notifHandler
		return orch.ExecuteConstitution("")
	})

	if err != nil {
		fmt.Fprintf(out, "\n⚠ Constitution creation failed: %v\n", err)
		return false
	}

	fmt.Fprintf(out, "\n✓ Constitution created successfully\n")
	return true
}

// runWorktreeGenScriptFromInit executes the worktree gen-script workflow.
// This generates a project-specific setup script for git worktrees.
func runWorktreeGenScriptFromInit(cmd *cobra.Command, configPath string) {
	out := cmd.OutOrStdout()

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(out, "⚠ Failed to load config: %v\n", err)
		return
	}

	notifHandler := notify.NewHandler(cfg.Notifications)
	historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)

	fmt.Fprintf(out, "\n")

	err = lifecycle.RunWithHistory(notifHandler, historyLogger, "worktree-gen-script", "", func() error {
		orch := workflow.NewWorkflowOrchestrator(cfg)
		orch.Executor.NotificationHandler = notifHandler

		fmt.Fprintf(out, "Generating worktree setup script...\n\n")
		if err := orch.Executor.Claude.Execute("/autospec.worktree-setup"); err != nil {
			return fmt.Errorf("generating worktree setup script: %w", err)
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(out, "\n⚠ Worktree script generation failed: %v\n", err)
		return
	}

	fmt.Fprintf(out, "\n✓ Worktree setup script generated at .autospec/scripts/setup-worktree.sh\n")
	fmt.Fprintf(out, "  Use 'autospec worktree create <branch>' to create worktrees with auto-setup.\n")
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
