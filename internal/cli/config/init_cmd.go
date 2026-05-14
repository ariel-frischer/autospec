package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ariel-frischer/autospec/internal/build"
	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/cliagent"
	"github.com/ariel-frischer/autospec/internal/commands"
	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/history"
	initpkg "github.com/ariel-frischer/autospec/internal/init"
	"github.com/ariel-frischer/autospec/internal/lifecycle"
	"github.com/ariel-frischer/autospec/internal/notify"
	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// sandboxPromptEnabled controls whether the sandbox configuration prompt is shown during init.
const sandboxPromptEnabled = false

// Color helper functions for init command output
var (
	cGreen   = color.New(color.FgGreen).SprintFunc()
	cYellow  = color.New(color.FgYellow).SprintFunc()
	cCyan    = color.New(color.FgCyan).SprintFunc()
	cRed     = color.New(color.FgRed).SprintFunc()
	cDim     = color.New(color.Faint).SprintFunc()
	cBold    = color.New(color.Bold).SprintFunc()
	cMagenta = color.New(color.FgMagenta).SprintFunc()
)

// printSectionHeader prints a visually distinct section header to help users focus.
// The header uses a simple line with the section name centered.
func printSectionHeader(out io.Writer, title string) {
	line := strings.Repeat("-", 10)
	fmt.Fprintf(out, "\n%s %s %s\n\n", cDim(line), cCyan(title), cDim(line))
}

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Initialize autospec configuration and commands",
	Long: `Initialize autospec with everything needed to get started.

This command:
  1. Installs Claude skills to .claude/skills/ when Claude is selected
  2. Creates user-level configuration at ~/.config/autospec/config.yml

If config already exists, it is left unchanged (use --force to overwrite).

By default, creates user-level config which applies to all your projects.
Use --project to create project-specific config that overrides user settings.

Configuration precedence (highest to lowest):
  1. Environment variables (AUTOSPEC_*)
  2. Project config (.autospec/config.yml)
  3. User config (~/.config/autospec/config.yml)
  4. Built-in defaults

Path argument:
  If provided, initializes the project at the specified path instead of
  the current directory. The path can be:
  - Relative: resolved against current directory (e.g., "my-project")
  - Absolute: used as-is (e.g., "/home/user/project")
  - Tilde: expanded to home directory (e.g., "~/projects/new")

  If the path does not exist, it will be created automatically.`,
	Example: `  # Initialize with user-level config (recommended for first-time setup)
  autospec init

  # Initialize at a specific path
  autospec init /path/to/project
  autospec init ~/projects/my-app
  autospec init my-new-project

  # Explicitly initialize in current directory
  autospec init .
  autospec init --here

  # Create project-specific config (overrides user config)
  autospec init --project

  # Overwrite existing config with defaults
  autospec init --force`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.GroupID = shared.GroupGettingStarted
	initCmd.Flags().BoolP("project", "p", false, "Create project-level config (.autospec/config.yml)")
	initCmd.Flags().BoolP("force", "f", false, "Overwrite existing config with defaults")
	initCmd.Flags().StringSlice("ai", nil, "Configure specific agents (comma-separated: claude,opencode)")
	initCmd.Flags().Bool("no-agents", false, "Skip agent configuration prompt")
	initCmd.Flags().Bool("here", false, "Initialize in current directory (same as 'init .')")
	// Keep --global as hidden alias for backward compatibility
	initCmd.Flags().BoolP("global", "g", false, "Deprecated: use default behavior instead (creates user-level config)")
	initCmd.Flags().MarkHidden("global")

	// Non-interactive flags for CI/CD and automation
	initCmd.Flags().Bool("sandbox", false, "Enable Claude sandbox configuration (skips prompt)")
	initCmd.Flags().Bool("no-sandbox", false, "Skip Claude sandbox configuration (skips prompt)")
	initCmd.Flags().Bool("use-subscription", false, "Use subscription billing (OAuth/Pro/Max) instead of API key (skips prompt)")
	initCmd.Flags().Bool("no-use-subscription", false, "Use API key billing instead of subscription (skips prompt)")
	initCmd.Flags().Bool("skip-permissions", false, "Enable autonomous mode (skip permission prompts) (skips prompt)")
	initCmd.Flags().Bool("no-skip-permissions", false, "Disable autonomous mode (more interactive) (skips prompt)")
	initCmd.Flags().Bool("gitignore", false, "Add .autospec/ to .gitignore (skips prompt)")
	initCmd.Flags().Bool("no-gitignore", false, "Skip adding .autospec/ to .gitignore (skips prompt)")
	initCmd.Flags().Bool("constitution", false, "Create project constitution (skips prompt)")
	initCmd.Flags().Bool("no-constitution", false, "Skip constitution creation (skips prompt)")
}

func runInit(cmd *cobra.Command, args []string) error {
	project, _ := cmd.Flags().GetBool("project")
	force, _ := cmd.Flags().GetBool("force")
	aiAgents, _ := cmd.Flags().GetStringSlice("ai")
	noAgents, _ := cmd.Flags().GetBool("no-agents")
	here, _ := cmd.Flags().GetBool("here")
	out := cmd.OutOrStdout()

	// Validate mutually exclusive flags before any operations
	flagPairs := []BoolFlagPair{
		{Positive: "sandbox", Negative: "no-sandbox"},
		{Positive: "use-subscription", Negative: "no-use-subscription"},
		{Positive: "skip-permissions", Negative: "no-skip-permissions"},
		{Positive: "gitignore", Negative: "no-gitignore"},
		{Positive: "constitution", Negative: "no-constitution"},
	}
	if err := checkMutuallyExclusiveFlags(cmd, flagPairs); err != nil {
		return fmt.Errorf("invalid flags: %w", err)
	}

	// Resolve target directory from path argument or --here flag
	targetDir, err := resolveTargetDirectory(args, here)
	if err != nil {
		return fmt.Errorf("resolving target directory: %w", err)
	}

	// If target directory is not current directory, change to it
	if targetDir != "" {
		originalDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}

		// Ensure target directory exists
		if err := EnsureDirectory(targetDir); err != nil {
			return fmt.Errorf("ensuring target directory: %w", err)
		}

		// Change to target directory
		if err := os.Chdir(targetDir); err != nil {
			return fmt.Errorf("changing to target directory: %w", err)
		}

		// Restore original directory when done
		defer func() {
			_ = os.Chdir(originalDir)
		}()

		fmt.Fprintf(out, "%s %s: %s\n", cGreen("✓"), cBold("Target directory"), cDim(targetDir))
	}

	// Print the banner
	shared.PrintBannerCompact(out)

	// ═══════════════════════════════════════════════════════════════════════
	// Phase 1: Fast setup (immediate file operations)
	// ═══════════════════════════════════════════════════════════════════════
	// Note: Command templates are installed per-agent in handleAgentConfiguration()
	// via cliagent.Configure(), not here. This ensures templates only go to
	// directories for agents the user actually selected.

	newConfigCreated, err := initializeConfig(out, project, force)
	if err != nil {
		return fmt.Errorf("initializing config: %w", err)
	}
	_ = newConfigCreated // Used for tracking first-time setup

	if err := ensureAutospecGitignore("."); err != nil {
		fmt.Fprintf(out, "%s Failed to save .autospec/.gitignore: %v\n", cYellow("⚠"), err)
	}

	// Handle agent selection and configuration
	selectedAgents, agentConfigs, err := handleAgentConfiguration(cmd, out, project, noAgents, aiAgents)
	if err != nil {
		return fmt.Errorf("configuring agents: %w", err)
	}

	// Create init.yml to record initialization state
	if err := saveInitSettings(out, project, agentConfigs); err != nil {
		fmt.Fprintf(out, "%s Failed to save init.yml: %v\n", cYellow("⚠"), err)
	}

	// Detect Claude auth and configure use_subscription (only if Claude was selected)
	configPath, _ := getConfigPath(project)
	if containsAgent(selectedAgents, "claude") {
		handleClaudeAuthDetection(cmd, out, configPath)
		// Prompt for skip_permissions (autonomous mode) after sandbox configuration
		handleSkipPermissionsPrompt(cmd, out, configPath, project)
	}

	// Check current state of constitution
	printSectionHeader(out, "Project Status")
	constitutionExists := handleConstitution(out)

	// ═══════════════════════════════════════════════════════════════════════
	// Phase 2: Collect all user choices (no changes applied yet)
	// ═══════════════════════════════════════════════════════════════════════
	pending := collectPendingActions(cmd, out, constitutionExists)

	// ═══════════════════════════════════════════════════════════════════════
	// Phase 3: Apply all pending changes
	// ═══════════════════════════════════════════════════════════════════════
	result := applyPendingActions(cmd, out, pending, configPath, constitutionExists)

	// Load config to get specsDir for summary
	cfg, _ := config.Load(configPath)
	specsDir := "specs"
	if cfg != nil && cfg.SpecsDir != "" {
		specsDir = cfg.SpecsDir
	}

	printSummary(out, result, specsDir)
	return nil
}

// handleAgentConfiguration handles the agent selection and configuration flow.
// If aiAgents is provided, those specific agents are configured directly.
// If noAgents is true, the prompt is skipped. In non-interactive mode without
// --no-agents or --ai, it returns an error with a helpful message.
// Returns the list of selected/configured agent names and agent configuration info.
func handleAgentConfiguration(cmd *cobra.Command, out io.Writer, project, noAgents bool, aiAgents []string) ([]string, []agentConfigInfo, error) {
	// If --ai flag was provided, validate and configure those agents directly
	if len(aiAgents) > 0 {
		return configureSpecificAgents(cmd, out, project, aiAgents)
	}

	if noAgents {
		fmt.Fprintln(out, "⏭ Agent configuration: skipped (--no-agents)")
		return nil, nil, nil
	}

	// Check if stdin is a terminal
	if !isTerminal() {
		return nil, nil, fmt.Errorf("agent selection requires an interactive terminal; " +
			"use --no-agents for non-interactive environments")
	}

	// Load config to get DefaultAgents for pre-selection
	configPath, err := getConfigPath(project)
	if err != nil {
		return nil, nil, fmt.Errorf("getting config path: %w", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		// Continue with empty defaults if config load fails
		cfg = &config.Configuration{}
	}

	// Get agents with defaults pre-selected
	agents := GetSupportedAgentsWithDefaults(cfg.DefaultAgents)

	// Section header for agent selection
	printSectionHeader(out, "Agent Selection")

	// Run agent selection prompt
	selected := promptAgentSelection(cmd.InOrStdin(), out, agents)
	defaultAgent := defaultAgentForSelection(selected, cfg.AgentPreset)
	if len(selected) > 1 {
		defaultAgent = promptDefaultAgentSelection(cmd.InOrStdin(), out, selectedAgentOptions(agents, selected), cfg.AgentPreset)
	}

	// Show selected agents feedback
	if len(selected) > 0 {
		fmt.Fprintf(out, "%s %s: %s\n", cGreen("✓"), cBold("Selected"), strings.Join(selected, ", "))
		if defaultAgent != "" {
			fmt.Fprintf(out, "%s %s: %s\n", cGreen("✓"), cBold("Default agent"), defaultAgent)
		}
	} else {
		fmt.Fprintf(out, "%s No agents selected\n", cYellow("⚠"))
	}

	// Configure selected agents and save preferences
	// Use "." as project directory for real init command
	// Pass project flag to determine whether to write to project-level or global config
	sandboxPrompts, agentConfigs, err := configureSelectedAgents(out, selected, defaultAgent, cfg, configPath, ".", project)
	if err != nil {
		return nil, nil, err
	}

	// Handle sandbox configuration prompts
	if err := handleSandboxConfiguration(cmd, out, sandboxPrompts, ".", cfg.SpecsDir); err != nil {
		return nil, nil, err
	}
	return selected, agentConfigs, nil
}

// configureSpecificAgents configures agents specified via --ai flag.
// It validates agent names against production agents in non-dev builds.
// Returns the list of successfully configured agent names and agent configuration info.
func configureSpecificAgents(cmd *cobra.Command, out io.Writer, project bool, aiAgents []string) ([]string, []agentConfigInfo, error) {
	// Validate agent names
	validAgents := getValidAgentNames()
	var invalidAgents []string
	var configuredAgents []string

	for _, agentName := range aiAgents {
		agentName = strings.TrimSpace(agentName)
		if agentName == "" {
			continue
		}

		if !validAgents[agentName] {
			invalidAgents = append(invalidAgents, agentName)
			continue
		}
		configuredAgents = append(configuredAgents, agentName)
	}

	// Report invalid agents
	if len(invalidAgents) > 0 {
		validList := build.ProductionAgents()
		return nil, nil, fmt.Errorf("unknown agent(s): %s (valid: %s)",
			strings.Join(invalidAgents, ", "),
			strings.Join(validList, ", "))
	}

	if len(configuredAgents) == 0 {
		return nil, nil, fmt.Errorf("no valid agents specified; valid agents: %s",
			strings.Join(build.ProductionAgents(), ", "))
	}

	// Load config for specsDir
	configPath, err := getConfigPath(project)
	if err != nil {
		return nil, nil, fmt.Errorf("getting config path: %w", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		cfg = &config.Configuration{}
	}

	specsDir := cfg.SpecsDir
	if specsDir == "" {
		specsDir = "specs"
	}

	// Configure each agent
	fmt.Fprintf(out, "%s %s: %s\n", cGreen("✓"), cBold("Agents"), strings.Join(configuredAgents, ", "))

	var sandboxPrompts []sandboxPromptInfo
	var agentConfigs []agentConfigInfo

	for _, agentName := range configuredAgents {
		agent := cliagent.Get(agentName)
		if agent == nil {
			fmt.Fprintf(out, "%s %s: not found\n", cYellow("⚠"), agentName)
			continue
		}

		result, err := cliagent.Configure(agent, ".", specsDir, project)
		if err != nil {
			fmt.Fprintf(out, "%s %s: configuration failed: %v\n", cYellow("⚠"), agentDisplayNames[agentName], err)
			agentConfigs = append(agentConfigs, agentConfigInfo{
				name:       agentName,
				configured: false,
			})
			continue
		}

		// result is nil when agent doesn't implement Configurator
		if result == nil {
			displayAgentConfigResult(out, agentName, nil)
			// Agent doesn't support configuration - still track but no settings file
			agentConfigs = append(agentConfigs, agentConfigInfo{
				name:       agentName,
				configured: false, // No configuration was performed
			})
			continue
		}

		displayAgentConfigResult(out, agentName, result)

		// Track agent configuration for init.yml
		agentConfigs = append(agentConfigs, agentConfigInfo{
			name:         agentName,
			configured:   true,
			settingsFile: result.SettingsFilePath,
		})

		// Check for sandbox configuration
		if info := checkSandboxConfiguration(agentName, agent, ".", specsDir); info != nil {
			sandboxPrompts = append(sandboxPrompts, *info)
		}
	}

	// Save agent preferences
	defaultAgent := configuredAgents[0]
	cfg.DefaultAgents = configuredAgents
	if err := persistAgentPreferences(out, configuredAgents, defaultAgent, cfg, configPath); err != nil {
		fmt.Fprintf(out, "%s Failed to save agent preferences: %v\n", cYellow("⚠"), err)
	}

	// Handle sandbox prompts
	if err := handleSandboxConfiguration(cmd, out, sandboxPrompts, ".", specsDir); err != nil {
		return nil, nil, err
	}
	return configuredAgents, agentConfigs, nil
}

// getValidAgentNames returns the set of valid agent names for the current build.
// In production builds, only production agents are valid.
// In dev builds, all registered agents are valid.
func getValidAgentNames() map[string]bool {
	valid := make(map[string]bool)

	if build.MultiAgentEnabled() {
		// Dev build: all registered agents are valid
		for _, name := range cliagent.List() {
			valid[name] = true
		}
	} else {
		// Production build: only production agents
		for _, name := range build.ProductionAgents() {
			valid[name] = true
		}
	}

	return valid
}

// sandboxPromptInfo holds information needed to prompt for sandbox configuration.
type sandboxPromptInfo struct {
	agentName   string
	displayName string
	pathsToAdd  []string
	existing    []string
	needsEnable bool // true if sandbox.enabled needs to be set to true
}

// agentConfigInfo holds the result of configuring a single agent.
// Used to track what was configured for init.yml creation.
type agentConfigInfo struct {
	name         string // agent name (e.g., "claude", "opencode")
	configured   bool   // true if configuration succeeded
	settingsFile string // path to settings file that was modified
}

const autospecGitignoreContent = `# Local autospec runtime files
init.yml
state/
context/
cache/
tmp/

# Legacy completion markers
SPEC_DONE
IMPLEMENTATION_DONE
`

// pendingActions holds all user choices collected during init prompts.
// Changes are applied atomically after all questions are answered.
type pendingActions struct {
	addGitignore       bool // add .autospec/ to .gitignore
	createConstitution bool // run constitution workflow
}

// initResult holds the results of the init command for final summary.
type initResult struct {
	constitutionExists bool
	hadErrors          bool
}

// configureSelectedAgents configures each selected agent and persists preferences.
// Returns a list of agents that have sandbox enabled and need configuration,
// and the configuration info for each agent (for init.yml creation).
// projectDir specifies where to write agent config files (e.g., .claude/settings.local.json).
// projectLevel determines whether to write to project-level config (true) or global config (false).
func configureSelectedAgents(out io.Writer, selected []string, defaultAgent string, cfg *config.Configuration, configPath, projectDir string, projectLevel bool) ([]sandboxPromptInfo, []agentConfigInfo, error) {
	if len(selected) == 0 {
		fmt.Fprintln(out, "⚠ Warning: No agents selected. You may need to configure agent permissions manually.")
		return nil, nil, nil
	}

	specsDir := cfg.SpecsDir
	if specsDir == "" {
		specsDir = "specs"
	}

	var sandboxPrompts []sandboxPromptInfo
	var agentConfigs []agentConfigInfo

	// Configure each selected agent
	for _, agentName := range selected {
		agent := cliagent.Get(agentName)
		if agent == nil {
			continue
		}

		result, err := cliagent.Configure(agent, projectDir, specsDir, projectLevel)
		if err != nil {
			fmt.Fprintf(out, "⚠ %s: configuration failed: %v\n", agentDisplayNames[agentName], err)
			agentConfigs = append(agentConfigs, agentConfigInfo{
				name:       agentName,
				configured: false,
			})
			continue
		}

		// result is nil when agent doesn't implement Configurator
		if result == nil {
			displayAgentConfigResult(out, agentName, nil)
			// Agent doesn't support configuration - still track but no settings file
			agentConfigs = append(agentConfigs, agentConfigInfo{
				name:       agentName,
				configured: false, // No configuration was performed
			})
			continue
		}

		displayAgentConfigResult(out, agentName, result)

		// Track agent configuration for init.yml
		agentConfigs = append(agentConfigs, agentConfigInfo{
			name:         agentName,
			configured:   true,
			settingsFile: result.SettingsFilePath,
		})

		// Check if agent supports sandbox configuration
		if info := checkSandboxConfiguration(agentName, agent, projectDir, specsDir); info != nil {
			sandboxPrompts = append(sandboxPrompts, *info)
		}
	}

	// Persist selected agents to config
	if err := persistAgentPreferences(out, selected, defaultAgent, cfg, configPath); err != nil {
		return nil, nil, err
	}

	return sandboxPrompts, agentConfigs, nil
}

// checkSandboxConfiguration checks if an agent needs sandbox configuration.
// Returns nil only if sandbox is fully configured (enabled with all required paths).
func checkSandboxConfiguration(agentName string, agent cliagent.Agent, projectDir, specsDir string) *sandboxPromptInfo {
	// Only Claude currently supports sandbox configuration
	claudeAgent, ok := agent.(*cliagent.Claude)
	if !ok {
		return nil
	}

	diff, err := claudeAgent.GetSandboxDiff(projectDir, specsDir)
	if err != nil || diff == nil {
		return nil
	}

	needsEnable := !diff.Enabled
	needsPaths := len(diff.PathsToAdd) > 0

	// Only skip if sandbox is enabled AND all paths are present
	if !needsEnable && !needsPaths {
		return nil
	}

	displayName := agentDisplayNames[agentName]
	if displayName == "" {
		displayName = agentName
	}

	return &sandboxPromptInfo{
		agentName:   agentName,
		displayName: displayName,
		pathsToAdd:  diff.PathsToAdd,
		existing:    diff.ExistingPaths,
		needsEnable: needsEnable,
	}
}

// handleSandboxConfiguration prompts for and applies sandbox configuration.
func handleSandboxConfiguration(cmd *cobra.Command, out io.Writer, prompts []sandboxPromptInfo, projectDir, specsDir string) error {
	if len(prompts) == 0 {
		return nil
	}

	sandboxFlag := resolveBoolFlag(cmd, "sandbox", "no-sandbox")
	if !sandboxPromptEnabled && sandboxFlag == nil {
		return nil
	}

	if specsDir == "" {
		specsDir = "specs"
	}

	for _, info := range prompts {
		if err := promptAndConfigureSandbox(cmd, out, info, projectDir, specsDir); err != nil {
			fmt.Fprintf(out, "⚠ %s sandbox configuration failed: %v\n", info.displayName, err)
		}
	}

	return nil
}

// promptAndConfigureSandbox displays the sandbox diff and prompts for confirmation.
// If --sandbox flag is set, configuration is applied without prompting.
// If --no-sandbox flag is set, configuration is skipped without prompting.
func promptAndConfigureSandbox(cmd *cobra.Command, out io.Writer, info sandboxPromptInfo, projectDir, specsDir string) error {
	// Check for CLI flag override
	sandboxFlag := resolveBoolFlag(cmd, "sandbox", "no-sandbox")
	if sandboxFlag != nil {
		if !*sandboxFlag {
			// --no-sandbox: skip sandbox configuration
			fmt.Fprintf(out, "%s Sandbox configuration: skipped (--no-sandbox)\n", cDim("⏭"))
			return nil
		}
		// --sandbox: apply configuration without prompting
		fmt.Fprintf(out, "%s Configuring sandbox (--sandbox flag)\n", cDim("⚙"))
		return applySandboxConfiguration(out, info, projectDir, specsDir)
	}

	// Section header for sandbox configuration
	printSectionHeader(out, "Sandbox Configuration")

	// Display the proposed changes with different messaging based on current state
	if info.needsEnable {
		fmt.Fprintf(out, "%s sandbox not enabled. Enabling sandbox improves security.\n\n", cBold(info.displayName))
	} else {
		fmt.Fprintf(out, "%s sandbox configuration detected.\n\n", cBold(info.displayName))
	}

	fmt.Fprintf(out, "Proposed changes to .claude/settings.local.json:\n\n")

	if info.needsEnable {
		fmt.Fprintf(out, "  %s: %s\n", cDim("sandbox.enabled"), cGreen("true"))
	}

	if len(info.pathsToAdd) > 0 {
		fmt.Fprintf(out, "%s:\n", cDim("sandbox.additionalAllowWritePaths"))
		for _, path := range info.pathsToAdd {
			fmt.Fprintf(out, "%s %q\n", cGreen("+"), path)
		}
	}

	if len(info.existing) > 0 {
		fmt.Fprintf(out, "\n  %s\n", cDim("(existing paths preserved)"))
	}

	fmt.Fprintf(out, "\n")

	// Prompt for confirmation (defaults to Yes)
	if !promptYesNoDefaultYes(cmd, "Configure Claude sandbox for autospec?") {
		fmt.Fprintf(out, "%s Sandbox configuration: skipped\n", cDim("⏭"))
		return nil
	}

	return applySandboxConfiguration(out, info, projectDir, specsDir)
}

// applySandboxConfiguration applies sandbox settings to the agent configuration.
// Extracted from promptAndConfigureSandbox to enable flag-based bypass.
func applySandboxConfiguration(out io.Writer, info sandboxPromptInfo, projectDir, specsDir string) error {
	agent := cliagent.Get(info.agentName)
	if agent == nil {
		return fmt.Errorf("agent %s not found", info.agentName)
	}

	result, err := cliagent.ConfigureSandbox(agent, projectDir, specsDir)
	if err != nil {
		return err
	}

	if result == nil || result.AlreadyConfigured {
		fmt.Fprintf(out, "%s %s sandbox: enabled with write paths configured\n", cGreen("✓"), cBold(info.displayName))
		return nil
	}

	// Show what was configured
	if result.SandboxWasEnabled {
		fmt.Fprintf(out, "%s %s sandbox: enabled\n", cGreen("✓"), cBold(info.displayName))
	}
	if len(result.PathsAdded) > 0 {
		fmt.Fprintf(out, "%s %s sandbox: configured with paths:\n", cGreen("✓"), cBold(info.displayName))
		for _, path := range result.PathsAdded {
			fmt.Fprintf(out, "%s %s\n", cGreen("+"), path)
		}
	}

	return nil
}

// handleClaudeAuthDetection detects Claude auth status and configures use_subscription.
// If --use-subscription or --no-use-subscription flag is set, bypasses the prompt.
// When OAuth is detected, the flag is ignored and OAuth takes precedence (with informational message).
func handleClaudeAuthDetection(cmd *cobra.Command, out io.Writer, configPath string) {
	status := cliagent.DetectClaudeAuth()

	// Section header for authentication
	printSectionHeader(out, "Authentication")

	// Show OAuth status
	if status.AuthType == cliagent.AuthTypeOAuth {
		fmt.Fprintf(out, "  %s OAuth: %s subscription\n",
			cGreen("✓"), status.SubscriptionType)
	} else {
		fmt.Fprintf(out, "  %s OAuth: not logged in\n", cDim("✗"))
	}

	// Show API key status
	if status.APIKeySet {
		fmt.Fprintf(out, "  %s API key: set in environment\n", cGreen("✓"))
	} else {
		fmt.Fprintf(out, "  %s API key: not set\n", cDim("✗"))
	}

	// Determine use_subscription value based on detection and flags
	var useSubscription bool
	var reason string

	// Check for CLI flag override
	billingFlag := resolveBoolFlag(cmd, "use-subscription", "no-use-subscription")

	switch {
	case status.AuthType == cliagent.AuthTypeOAuth:
		// OAuth detected - use subscription (flags ignored)
		useSubscription = true
		reason = fmt.Sprintf("using %s subscription, not API credits", status.SubscriptionType)
		// Inform user if they provided a conflicting flag
		if billingFlag != nil && !*billingFlag {
			fmt.Fprintf(out, "\n  %s --no-use-subscription ignored: OAuth detected, using subscription\n", cDim("ℹ"))
		}

	case billingFlag != nil:
		// Flag provided - use flag value (only when OAuth not detected)
		useSubscription = *billingFlag
		if useSubscription {
			reason = "using subscription (--use-subscription)"
		} else {
			reason = "using API credits (--no-use-subscription)"
		}

	case status.APIKeySet && status.AuthType != cliagent.AuthTypeOAuth:
		// Only API key detected - prompt user (interactive mode)
		fmt.Fprintf(out, "\n")
		fmt.Fprintf(out, "  %s You have an API key but no OAuth login.\n", cYellow("💡"))
		fmt.Fprintf(out, "     %s Use API key: charges apply per request\n", cDim("→"))
		fmt.Fprintf(out, "     %s Use OAuth: run 'claude' to login with Pro/Max subscription\n", cDim("→"))
		fmt.Fprintf(out, "\n")

		if promptYesNo(cmd, "Use API key for billing? (n = login for OAuth instead)") {
			useSubscription = false
			reason = "using API credits"
		} else {
			useSubscription = true
			reason = "will use subscription after OAuth login"
			fmt.Fprintf(out, "  %s Run %s to login before using autospec\n", cDim("→"), cCyan("'claude'"))
		}

	default:
		// No auth detected - safe default
		useSubscription = true
		reason = "safe default, no API charges"
	}

	// Update config file
	if err := updateUseSubscriptionInConfig(configPath, useSubscription); err != nil {
		fmt.Fprintf(out, "\n  %s Failed to update config: %v\n", cYellow("⚠"), err)
		return
	}

	fmt.Fprintf(out, "\n  %s use_subscription: %v %s\n",
		cGreen("→"), useSubscription, cDim("("+reason+")"))
}

// updateUseSubscriptionInConfig updates the use_subscription value in the config file.
func updateUseSubscriptionInConfig(configPath string, useSubscription bool) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	found := false
	newValue := fmt.Sprintf("use_subscription: %v", useSubscription)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "use_subscription:") {
			lines[i] = newValue
			found = true
			break
		}
	}

	if !found {
		// Find agent_preset line and insert after it
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "agent_preset:") {
				// Insert after agent_preset
				newLines := make([]string, 0, len(lines)+1)
				newLines = append(newLines, lines[:i+1]...)
				newLines = append(newLines, newValue)
				newLines = append(newLines, lines[i+1:]...)
				lines = newLines
				found = true
				break
			}
		}
	}

	if !found {
		// Append at end if neither found
		lines = append(lines, newValue)
	}

	return os.WriteFile(configPath, []byte(strings.Join(lines, "\n")), 0o644)
}

// updateSkipPermissionsInConfig updates the skip_permissions setting in the config file.
// It preserves existing config formatting by line-based editing.
func updateSkipPermissionsInConfig(configPath string, skipPermissions bool) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	found := false
	newValue := fmt.Sprintf("skip_permissions: %v", skipPermissions)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "skip_permissions:") {
			lines[i] = newValue
			found = true
			break
		}
	}

	if !found {
		// Find use_subscription line and insert after it
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "use_subscription:") {
				newLines := make([]string, 0, len(lines)+1)
				newLines = append(newLines, lines[:i+1]...)
				newLines = append(newLines, newValue)
				newLines = append(newLines, lines[i+1:]...)
				lines = newLines
				found = true
				break
			}
		}
	}

	if !found {
		// Find agent_preset line and insert after it
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "agent_preset:") {
				newLines := make([]string, 0, len(lines)+1)
				newLines = append(newLines, lines[:i+1]...)
				newLines = append(newLines, newValue)
				newLines = append(newLines, lines[i+1:]...)
				lines = newLines
				found = true
				break
			}
		}
	}

	if !found {
		// Append at end if neither found
		lines = append(lines, newValue)
	}

	return os.WriteFile(configPath, []byte(strings.Join(lines, "\n")), 0o644)
}

// handleSkipPermissionsPrompt prompts the user about the skip_permissions setting.
// This is only called when Claude is selected as an agent.
// If --skip-permissions or --no-skip-permissions flag is set, bypasses the prompt.
// In non-interactive mode without flags, it silently sets the default value (true) without prompting.
// If skip_permissions is already set to true, it just displays the value without prompting.
// If skip_permissions is false or not set, it prompts the user.
func handleSkipPermissionsPrompt(cmd *cobra.Command, out io.Writer, configPath string, project bool) {
	// Check for CLI flag override first
	permissionsFlag := resolveBoolFlag(cmd, "skip-permissions", "no-skip-permissions")

	// Load current config to check existing value
	cfg, _ := config.Load(configPath)
	currentValue := false
	if cfg != nil {
		currentValue = cfg.SkipPermissions
	}

	// Check if skip_permissions is explicitly set to true - if so, skip prompt
	alreadyEnabled := configHasKey(configPath, "skip_permissions") && currentValue

	// If flag was provided, use the flag value and bypass all prompts
	if permissionsFlag != nil {
		skipPermissions := *permissionsFlag
		if err := updateSkipPermissionsInConfig(configPath, skipPermissions); err != nil {
			fmt.Fprintf(out, "%s Failed to update skip_permissions: %v\n", cYellow("⚠"), err)
			return
		}
		if skipPermissions {
			fmt.Fprintf(out, "%s skip_permissions: true %s\n",
				cGreen("✓"), cDim("(--skip-permissions)"))
		} else {
			fmt.Fprintf(out, "%s skip_permissions: false %s\n",
				cGreen("✓"), cDim("(--no-skip-permissions)"))
		}
		return
	}

	// Handle non-interactive mode gracefully (when no flag provided)
	if !isTerminal() {
		if alreadyEnabled {
			// Already enabled, just show current value
			fmt.Fprintf(out, "%s skip_permissions: true %s\n",
				cGreen("✓"), cDim("(already enabled)"))
		} else {
			// Use default value (true) without prompting
			if err := updateSkipPermissionsInConfig(configPath, true); err != nil {
				fmt.Fprintf(out, "%s Failed to update skip_permissions: %v\n", cYellow("⚠"), err)
			} else {
				fmt.Fprintf(out, "%s skip_permissions: true %s\n",
					cGreen("✓"), cDim("(default, non-interactive)"))
			}
		}
		return
	}

	printSectionHeader(out, "Permissions Mode")

	// If already enabled, just show current value and skip prompt
	if alreadyEnabled {
		fmt.Fprintf(out, "  %s skip_permissions: %s (enabled)\n",
			cGreen("✓"), cGreen("true"))
		fmt.Fprintf(out, "  %s Change with: %s\n",
			cDim("💡"), cCyan("autospec config toggle skip_permissions"))
		return
	}

	// Display explanation for new users or those with it disabled
	fmt.Fprintf(out, "  %s Without sufficient permissions, Claude may fail mid-task.\n", cDim("→"))
	fmt.Fprintf(out, "    We recommend enabling %s to avoid permission issues.\n\n", cCyan("skip_permissions"))
	fmt.Fprintf(out, "  %s This only affects autospec Claude runs; no Claude settings files are changed.\n\n", cDim("→"))
	fmt.Fprintf(out, "  %s Change later: %s\n\n",
		cDim("💡"), cCyan("autospec config toggle skip_permissions"))

	// Prompt user (default to Yes - recommended)
	skipPermissions := promptYesNoDefaultYes(cmd, "Enable skip_permissions (recommended)?")

	// Update config file
	if err := updateSkipPermissionsInConfig(configPath, skipPermissions); err != nil {
		fmt.Fprintf(out, "\n  %s Failed to update config: %v\n", cYellow("⚠"), err)
		return
	}

	// Show result
	if skipPermissions {
		fmt.Fprintf(out, "\n  %s skip_permissions: %v %s\n",
			cGreen("→"), skipPermissions, cDim("(autonomous mode enabled)"))
	} else {
		fmt.Fprintf(out, "\n  %s skip_permissions: %v %s\n",
			cGreen("→"), skipPermissions, cDim("(interactive mode, prompts may appear)"))
	}
}

// configHasKey checks if a key is explicitly set in the config file (not just default).
func configHasKey(configPath, key string) bool {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}
	// Check if the key appears in the file (simple check for "key:")
	return strings.Contains(string(content), key+":")
}

// displayAgentConfigResult displays the configuration result for an agent.
func displayAgentConfigResult(out io.Writer, agentName string, result *cliagent.ConfigResult) {
	displayName := agentDisplayNames[agentName]
	if displayName == "" {
		displayName = agentName
	}

	if result == nil {
		fmt.Fprintf(out, "%s %s: no configuration needed\n", cGreen("✓"), cBold(displayName))
		return
	}

	if result.Warning != "" {
		fmt.Fprintf(out, "%s %s: %s\n", cYellow("⚠"), cBold(displayName), result.Warning)
	}

	if result.AlreadyConfigured {
		fmt.Fprintf(out, "%s %s: permissions already configured\n", cGreen("✓"), cBold(displayName))
		return
	}

	if len(result.PermissionsAdded) > 0 {
		fmt.Fprintf(out, "%s %s: configured with permissions:\n", cGreen("✓"), cBold(displayName))
		for _, perm := range result.PermissionsAdded {
			fmt.Fprintf(out, "    %s %s\n", cDim("-"), perm)
		}
	}
}

// persistAgentPreferences saves the selected agents to config for future init runs.
// Also updates agent_preset when an execution default is provided.
func persistAgentPreferences(out io.Writer, selected []string, defaultAgent string, cfg *config.Configuration, configPath string) error {
	// Update config with new agent preferences
	cfg.DefaultAgents = selected
	if defaultAgent != "" && !containsAgent(selected, defaultAgent) {
		defaultAgent = defaultAgentForSelection(selected, "")
	}

	// Read existing config file to preserve formatting and comments
	existingContent, err := os.ReadFile(configPath)
	if err != nil {
		// Config doesn't exist yet, nothing to update
		return nil
	}

	// Update the default_agents line in the config file
	newContent := updateDefaultAgentsInConfig(string(existingContent), selected)

	if defaultAgent != "" {
		newContent = updateAgentPresetInConfig(newContent, defaultAgent)
		cfg.AgentPreset = defaultAgent
		fmt.Fprintf(out, "%s %s: %s %s\n", cGreen("✓"), cBold("agent_preset"), defaultAgent, cDim("(set as default for execution)"))
	}

	if err := os.WriteFile(configPath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("saving agent preferences: %w", err)
	}

	fmt.Fprintf(out, "%s Agent preferences saved to %s\n", cGreen("✓"), cDim(configPath))
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

// updateAgentPresetInConfig updates the agent_preset line in the config content.
func updateAgentPresetInConfig(content, agentName string) string {
	lines := strings.Split(content, "\n")
	found := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "agent_preset:") {
			lines[i] = fmt.Sprintf("agent_preset: %s", agentName)
			found = true
			break
		}
	}

	if !found {
		// Insert after first line (usually a comment) or at start
		newLine := fmt.Sprintf("agent_preset: %s", agentName)
		if len(lines) > 0 {
			lines = append([]string{lines[0], newLine}, lines[1:]...)
		} else {
			lines = []string{newLine}
		}
	}

	return strings.Join(lines, "\n")
}

// containsAgent checks if the given agent name is in the list of agents.
func containsAgent(agents []string, name string) bool {
	for _, a := range agents {
		if a == name {
			return true
		}
	}
	return false
}

func defaultAgentForSelection(selected []string, currentDefault string) string {
	if len(selected) == 0 {
		return ""
	}

	if containsAgent(selected, currentDefault) {
		return currentDefault
	}

	return selected[0]
}

// isTerminalFunc is a function variable for terminal detection, allowing test mocking.
var isTerminalFunc = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func isTerminal() bool {
	return isTerminalFunc()
}

// BoolFlagPair represents a pair of mutually exclusive boolean flags (--flag and --no-flag).
type BoolFlagPair struct {
	Positive string // Name of the enabling flag (e.g., "sandbox")
	Negative string // Name of the disabling flag (e.g., "no-sandbox")
}

// resolveBoolFlag resolves the value of a boolean flag pair.
// Returns:
//   - nil if neither flag was explicitly set (user should be prompted)
//   - true if the positive flag was set
//   - false if the negative flag was set
//
// This enables three-state boolean logic where "unset" differs from "false".
func resolveBoolFlag(cmd *cobra.Command, positive, negative string) *bool {
	flags := cmd.Flags()

	positiveChanged := flags.Changed(positive)
	negativeChanged := flags.Changed(negative)

	// Neither flag set - return nil to indicate prompting is needed
	if !positiveChanged && !negativeChanged {
		return nil
	}

	// Positive flag was explicitly set
	if positiveChanged {
		val := true
		return &val
	}

	// Negative flag was explicitly set
	val := false
	return &val
}

// checkMutuallyExclusiveFlags validates that no flag pair has both flags set.
// Returns an error if any pair has both positive and negative flags provided.
func checkMutuallyExclusiveFlags(cmd *cobra.Command, pairs []BoolFlagPair) error {
	flags := cmd.Flags()

	for _, pair := range pairs {
		positiveChanged := flags.Changed(pair.Positive)
		negativeChanged := flags.Changed(pair.Negative)

		if positiveChanged && negativeChanged {
			return fmt.Errorf("flags --%s and --%s are mutually exclusive", pair.Positive, pair.Negative)
		}
	}

	return nil
}

// initializeConfig creates or updates config file.
// Returns true if a new config was created (for showing first-time setup info).
func initializeConfig(out io.Writer, project, force bool) (bool, error) {
	configPath, err := getConfigPath(project)
	if err != nil {
		return false, fmt.Errorf("getting config path: %w", err)
	}

	configExists := fileExistsCheck(configPath)

	if configExists && !force {
		fmt.Fprintf(out, "%s %s: exists at %s\n", cGreen("✓"), cBold("Config"), cDim(configPath))
		return false, nil
	}

	if err := writeDefaultConfigForScope(configPath, project); err != nil {
		return false, fmt.Errorf("writing default config: %w", err)
	}

	if configExists {
		fmt.Fprintf(out, "%s %s: overwritten at %s\n", cGreen("✓"), cBold("Config"), cDim(configPath))
	} else {
		fmt.Fprintf(out, "%s %s: created at %s\n", cGreen("✓"), cBold("Config"), cDim(configPath))
	}

	// Show first-time automation setup notice for new user-level configs
	if !project && !configExists {
		showAutomationSetupNotice(out, configPath)
	}

	return !configExists, nil
}

// showAutomationSetupNotice displays information about the automation setup.
// This is only shown when creating a NEW user-level config (not project, not if exists).
func showAutomationSetupNotice(out io.Writer, configPath string) {
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "%s\n", cYellow("╔══════════════════════════════════════════════════════════════════════╗"))
	fmt.Fprintf(out, "%s\n", cYellow("║                    AUTOMATION SECURITY INFO                          ║"))
	fmt.Fprintf(out, "%s\n", cYellow("╚══════════════════════════════════════════════════════════════════════╝"))
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "autospec runs Claude Code with %s by default.\n", cYellow("--dangerously-skip-permissions"))
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "%s\n", cBold("WHY THIS IS RECOMMENDED:"))
	fmt.Fprintf(out, "  Without this flag, Claude requires manual approval for every file edit,\n")
	fmt.Fprintf(out, "  shell command, and tool call - making automation impractical. Managing\n")
	fmt.Fprintf(out, "  allow/deny rules for all necessary operations is complex and error-prone.\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "%s\n", cBold("SECURITY MITIGATION:"))
	fmt.Fprintf(out, "  Enable Claude's sandbox (configured next) for OS-level protection.\n")
	fmt.Fprintf(out, "  With sandbox enabled, Claude cannot access files outside your project.\n")
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "See %s for detailed security information.\n", cDim("docs/claude-settings.md"))
	fmt.Fprintf(out, "\n")
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
	return writeDefaultConfigForScope(configPath, false)
}

func writeDefaultConfigForScope(configPath string, project bool) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	template := config.GetDefaultConfigTemplate()
	if project {
		template = strings.Replace(template, "state_dir: ~/.autospec/state", "state_dir: ./.autospec/state", 1)
	}
	if err := os.WriteFile(configPath, []byte(template), 0o644); err != nil {
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

// ConstitutionRunner is the function that runs the constitution workflow.
// It can be replaced in tests to avoid running real Claude.
// Exported for testing from other packages.
var ConstitutionRunner = runConstitutionFromInitImpl

// runConstitutionFromInit executes the constitution workflow.
// Returns true if constitution was created successfully.
func runConstitutionFromInit(cmd *cobra.Command, configPath string) bool {
	return ConstitutionRunner(cmd, configPath)
}

// runConstitutionFromInitImpl is the real implementation of constitution running.
func runConstitutionFromInitImpl(cmd *cobra.Command, configPath string) bool {
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
		shared.ApplyOutputStyle(cmd, orch)
		return orch.ExecuteConstitution("")
	})
	if err != nil {
		fmt.Fprintf(out, "\n⚠ Constitution creation failed: %v\n", err)
		return false
	}

	return true
}

// WorktreeScriptRunner is the function that runs the worktree gen-script workflow.
// It can be replaced in tests to avoid running real Claude.
// Exported for testing from other packages.
var WorktreeScriptRunner = runWorktreeGenScriptFromInitImpl

// runWorktreeGenScriptFromInit executes the worktree gen-script workflow.
// This generates a project-specific setup script for git worktrees.
// Returns true if the script was generated successfully.
func runWorktreeGenScriptFromInit(cmd *cobra.Command, configPath string) bool {
	return WorktreeScriptRunner(cmd, configPath)
}

// runWorktreeGenScriptFromInitImpl is the real implementation.
// Returns true if the script was generated successfully.
func runWorktreeGenScriptFromInitImpl(cmd *cobra.Command, configPath string) bool {
	out := cmd.OutOrStdout()

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(out, "⚠ Failed to load config: %v\n", err)
		return false
	}

	notifHandler := notify.NewHandler(cfg.Notifications)
	historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)

	fmt.Fprintf(out, "\n")

	err = lifecycle.RunWithHistory(notifHandler, historyLogger, "worktree-gen-script", "", func() error {
		orch := workflow.NewWorkflowOrchestrator(cfg)
		orch.Executor.NotificationHandler = notifHandler
		shared.ApplyOutputStyle(cmd, orch)

		fmt.Fprintf(out, "Generating worktree setup script...\n\n")
		if err := orch.Executor.Claude.Execute("/autospec.worktree-setup"); err != nil {
			return fmt.Errorf("generating worktree setup script: %w", err)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(out, "\n⚠ Worktree script generation failed: %v\n", err)
		return false
	}

	fmt.Fprintf(out, "\n✓ Worktree setup script generated at .autospec/scripts/setup-worktree.sh\n")
	fmt.Fprintf(out, "  Use 'autospec worktree create <branch>' to create worktrees with auto-setup.\n")
	return true
}

// handleConstitution checks for existing constitution and copies it if needed.
// Returns true if constitution exists (either copied or already present).
func handleConstitution(out io.Writer) bool {
	// Autospec paths (where we want the constitution)
	autospecPaths := []string{
		".autospec/constitution.yaml",
		".autospec/constitution.yml",
		".autospec/memory/constitution.yaml", // legacy
		".autospec/memory/constitution.yml",  // legacy
	}

	// Legacy specify paths (source for migration)
	legacyPaths := []string{
		".specify/memory/constitution.yaml",
		".specify/memory/constitution.yml",
	}

	// Check if any autospec constitution already exists
	for _, path := range autospecPaths {
		if _, err := os.Stat(path); err == nil {
			fmt.Fprintf(out, "%s %s: found at %s\n", cGreen("✓"), cBold("Constitution"), cDim(path))
			return true
		}
	}

	// Check if any legacy specify constitution exists
	for _, legacyPath := range legacyPaths {
		if _, err := os.Stat(legacyPath); err == nil {
			// Copy legacy constitution to autospec location (prefer .yaml)
			destPath := autospecPaths[0]
			if err := copyConstitution(legacyPath, destPath); err != nil {
				fmt.Fprintf(out, "%s %s: failed to copy from %s: %v\n", cYellow("⚠"), cBold("Constitution"), legacyPath, err)
				return false
			}
			fmt.Fprintf(out, "%s %s: copied from %s → %s\n", cGreen("✓"), cBold("Constitution"), cDim(legacyPath), cDim(destPath))
			return true
		}
	}

	// No constitution found
	fmt.Fprintf(out, "%s %s: not found\n", cYellow("⚠"), cBold("Constitution"))
	return false
}

// copyConstitution copies the constitution file from src to dst
func copyConstitution(src, dst string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Read source file
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}

	// Write to destination
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("failed to write destination: %w", err)
	}

	return nil
}

// gitignoreHasAutospec checks if .gitignore contains .autospec entry.
func gitignoreHasAutospec(content string) bool {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == ".autospec" || line == ".autospec/" || strings.HasPrefix(line, ".autospec/") {
			return true
		}
	}
	return false
}

// addAutospecToGitignore appends .autospec/ to the gitignore file.
// Creates the file if it doesn't exist.
func addAutospecToGitignore(gitignorePath string) error {
	var content string
	data, err := os.ReadFile(gitignorePath)
	if err == nil {
		content = string(data)
		// Ensure there's a newline before our entry
		if len(content) > 0 && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
	}

	content += ".autospec/\n"
	if err := os.WriteFile(gitignorePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing .gitignore: %w", err)
	}
	return nil
}

func ensureAutospecGitignore(projectDir string) error {
	gitignorePath := filepath.Join(projectDir, ".autospec", ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err == nil {
		content := string(data)
		if strings.Contains(content, autospecGitignoreContent) {
			return nil
		}
		if len(content) > 0 && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if content != "" {
			content += "\n"
		}
		content += autospecGitignoreContent
		if err := os.WriteFile(gitignorePath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing .autospec/.gitignore: %w", err)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("reading .autospec/.gitignore: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(gitignorePath), 0o755); err != nil {
		return fmt.Errorf("creating .autospec directory: %w", err)
	}
	if err := os.WriteFile(gitignorePath, []byte(autospecGitignoreContent), 0o644); err != nil {
		return fmt.Errorf("writing .autospec/.gitignore: %w", err)
	}
	return nil
}

// gitignoreNeedsUpdate checks if .autospec/ needs to be added to .gitignore.
// Returns true if gitignore doesn't exist or doesn't contain .autospec/.
func gitignoreNeedsUpdate() bool {
	data, err := os.ReadFile(".gitignore")
	if err != nil {
		return true // File doesn't exist
	}
	return !gitignoreHasAutospec(string(data))
}

// handleGitignorePrompt checks if .autospec/ is in .gitignore and prompts to add it.
// This is kept for test compatibility - the main flow uses collectPendingActions/applyPendingActions.
func handleGitignorePrompt(cmd *cobra.Command, out io.Writer) {
	gitignorePath := ".gitignore"

	data, err := os.ReadFile(gitignorePath)
	if err == nil {
		if gitignoreHasAutospec(string(data)) {
			fmt.Fprintf(out, "✓ Gitignore: .autospec/ already present\n")
			return
		}
	}

	fmt.Fprintf(out, "\n💡 Add .autospec/ to .gitignore?\n")
	fmt.Fprintf(out, "   → Recommended for shared/public/company repos (prevents config conflicts)\n")
	fmt.Fprintf(out, "   → Personal projects can leave it trackable for version history\n")

	if promptYesNo(cmd, "Add .autospec/ to .gitignore?") {
		if err := addAutospecToGitignore(gitignorePath); err != nil {
			fmt.Fprintf(out, "⚠ Failed to update .gitignore: %v\n", err)
			return
		}
		fmt.Fprintf(out, "✓ Gitignore: added .autospec/\n")
	} else {
		fmt.Fprintf(out, "⏭ Gitignore: skipped\n")
	}
}

// collectPendingActions prompts the user for all choices without applying any changes.
// Returns the collected choices for later atomic application.
// If flags are provided, they bypass the prompts. In non-interactive mode without flags,
// prompts are skipped with default values.
func collectPendingActions(cmd *cobra.Command, out io.Writer, constitutionExists bool) pendingActions {
	var pending pendingActions

	printSectionHeader(out, "Optional Setup")

	// Question 1: Gitignore
	pending.addGitignore = collectGitignoreChoice(cmd, out)

	// Question 2: Constitution (only if not exists)
	pending.createConstitution = collectConstitutionChoice(cmd, out, constitutionExists)

	return pending
}

// collectGitignoreChoice determines whether to add .autospec/ to .gitignore.
// Checks flag override first, then prompts interactively if in terminal, or uses default.
func collectGitignoreChoice(cmd *cobra.Command, out io.Writer) bool {
	// Check for CLI flag override
	gitignoreFlag := resolveBoolFlag(cmd, "gitignore", "no-gitignore")
	if gitignoreFlag != nil {
		if *gitignoreFlag {
			fmt.Fprintf(out, "%s %s: will add .autospec/ %s\n", cGreen("✓"), cBold("Gitignore"), cDim("(--gitignore)"))
		} else {
			fmt.Fprintf(out, "%s %s: skipped %s\n", cDim("⏭"), cBold("Gitignore"), cDim("(--no-gitignore)"))
		}
		return *gitignoreFlag
	}

	// Check if already present
	if !gitignoreNeedsUpdate() {
		fmt.Fprintf(out, "%s %s: .autospec/ already present\n", cGreen("✓"), cBold("Gitignore"))
		return false
	}

	// Non-interactive mode: use default (no) without prompting
	if !isTerminal() {
		fmt.Fprintf(out, "%s %s: skipped %s\n", cDim("⏭"), cBold("Gitignore"), cDim("(non-interactive, use --gitignore to enable)"))
		return false
	}

	// Interactive prompt
	fmt.Fprintf(out, "Add %s to .gitignore?\n", cBold(".autospec/"))
	fmt.Fprintf(out, "  %s %s ignore all autospec project files (public/local-only repos)\n", cGreen("y"), cDim("→"))
	fmt.Fprintf(out, "  %s %s allow versioning of autospec project files\n", cYellow("n"), cDim("→"))
	result := promptYesNo(cmd, "Add to .gitignore?")
	fmt.Fprintf(out, "\n") // Visual separation before next question
	return result
}

// collectConstitutionChoice determines whether to create a constitution.
// Checks flag override first, then prompts interactively if in terminal, or uses default.
func collectConstitutionChoice(cmd *cobra.Command, out io.Writer, constitutionExists bool) bool {
	// Check for CLI flag override
	constitutionFlag := resolveBoolFlag(cmd, "constitution", "no-constitution")
	if constitutionFlag != nil {
		if *constitutionFlag {
			if constitutionExists {
				fmt.Fprintf(out, "%s %s: already exists %s\n", cGreen("✓"), cBold("Constitution"), cDim("(--constitution ignored)"))
				return false
			}
			fmt.Fprintf(out, "%s %s: will create %s\n", cGreen("✓"), cBold("Constitution"), cDim("(--constitution)"))
			return true
		}
		fmt.Fprintf(out, "%s %s: skipped %s\n", cDim("⏭"), cBold("Constitution"), cDim("(--no-constitution)"))
		return false
	}

	// Constitution already exists: nothing to do
	if constitutionExists {
		return false
	}

	// Non-interactive mode: use default (no) without prompting
	if !isTerminal() {
		fmt.Fprintf(out, "%s %s: skipped %s\n", cDim("⏭"), cBold("Constitution"), cDim("(non-interactive, use --constitution to enable)"))
		return false
	}

	// Interactive prompt
	fmt.Fprintf(out, "%s %s (one-time setup per project)\n", cMagenta("📜"), cBold("Constitution"))
	fmt.Fprintf(out, "   %s Defines your project's coding standards and principles\n", cDim("→"))
	fmt.Fprintf(out, "   %s Required before running any autospec workflows\n", cDim("→"))
	fmt.Fprintf(out, "   %s Runs a Claude session to analyze your project\n", cDim("→"))
	return promptYesNoDefaultYes(cmd, "Create constitution?")
}

// applyPendingActions applies all collected user choices.
// Returns initResult with updated state and error tracking.
func applyPendingActions(cmd *cobra.Command, out io.Writer, pending pendingActions, configPath string, constitutionExists bool) initResult {
	result := initResult{constitutionExists: constitutionExists}

	// Apply gitignore change (fast, no Claude)
	if pending.addGitignore {
		if err := addAutospecToGitignore(".gitignore"); err != nil {
			fmt.Fprintf(out, "%s Failed to update .gitignore: %v\n", cYellow("⚠"), err)
		} else {
			fmt.Fprintf(out, "%s %s: added .autospec/\n", cGreen("✓"), cBold("Gitignore"))
		}
	}

	// Run constitution workflow (Claude session)
	if pending.createConstitution {
		printSectionHeader(out, "Running: Constitution")
		if runConstitutionFromInit(cmd, configPath) {
			result.constitutionExists = true
		} else {
			result.hadErrors = true
		}
	}

	return result
}

func printSummary(out io.Writer, result initResult, specsDir string) {
	fmt.Fprintf(out, "\n")

	// Show ready message only if constitution exists AND no errors occurred
	if result.constitutionExists && !result.hadErrors {
		fmt.Fprintf(out, "%s %s\n\n", cGreen("✓"), cBold("Autospec is ready!"))
	}

	// Show constitution warning if it doesn't exist
	if !result.constitutionExists {
		fmt.Fprintf(out, "%s %s: You MUST create a constitution before using autospec.\n", cYellow("⚠"), cBold("IMPORTANT"))
		fmt.Fprintf(out, "Run the following command first:\n\n")
		fmt.Fprintf(out, "  %s\n\n", cCyan("autospec constitution"))
		fmt.Fprintf(out, "The constitution defines your project's principles and guidelines.\n")
		fmt.Fprintf(out, "Without it, workflow commands (specify, plan, tasks, implement) will fail.\n\n")
	}

	fmt.Fprintf(out, "%s\n", cBold("Quick start:"))
	// Add step 0 if constitution doesn't exist
	if !result.constitutionExists {
		fmt.Fprintf(out, "  %s %s  %s\n", cYellow("0."), cCyan("autospec constitution"), cDim("# required first!"))
	}
	fmt.Fprintf(out, "  %s %s\n", cDim("1."), cCyan("autospec specify \"Add user authentication\""))
	fmt.Fprintf(out, "  %s Review the generated spec in %s/\n", cDim("2."), cDim(specsDir))
	fmt.Fprintf(out, "  %s %s  %s\n", cDim("3."), cCyan("autospec run -pti"), cDim("# -pti is short for --plan --tasks --implement"))
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "Or run all steps at once %s:\n", cDim("(specify → plan → tasks → implement)"))
	fmt.Fprintf(out, "  %s\n", cCyan("autospec all \"Add user authentication\""))
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "Run %s to verify dependencies.\n", cDim("'autospec doctor'"))
	fmt.Fprintf(out, "Run %s for all commands.\n", cDim("'autospec -h'"))
}

// saveInitSettings creates or updates .autospec/init.yml with initialization settings.
// This file records where agent permissions were configured (global vs project scope)
// so that `autospec doctor` knows where to check for permissions.
func saveInitSettings(out io.Writer, projectLevel bool, agentConfigs []agentConfigInfo) error {
	// Determine scope based on projectLevel flag
	scope := initpkg.ScopeGlobal
	if projectLevel {
		scope = initpkg.ScopeProject
	}

	// Build autospec version string
	autospecVersion := fmt.Sprintf("autospec v%s", build.Version)

	// Check if init.yml already exists to preserve created_at
	var settings *initpkg.Settings
	if initpkg.Exists() {
		existing, err := initpkg.Load()
		if err == nil {
			// Preserve created_at, update other fields
			settings = initpkg.NewSettings(autospecVersion)
			settings.CreatedAt = existing.CreatedAt
			settings.UpdatedAt = time.Now()
		}
	}

	// Create new settings if none loaded
	if settings == nil {
		settings = initpkg.NewSettings(autospecVersion)
	}

	settings.SettingsScope = scope

	// Convert agent config info to init.yml agent entries
	for _, info := range agentConfigs {
		settings.Agents = append(settings.Agents, initpkg.AgentEntry{
			Name:         info.name,
			Configured:   info.configured,
			SettingsFile: info.settingsFile,
		})
	}

	// Save init.yml
	if err := settings.Save(); err != nil {
		return fmt.Errorf("saving init settings: %w", err)
	}

	fmt.Fprintf(out, "%s %s: created at %s\n", cGreen("✓"), cBold("Init settings"), cDim(initpkg.DefaultPath()))
	return nil
}
