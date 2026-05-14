package cliagent

import (
	"fmt"

	"github.com/ariel-frischer/autospec/internal/commands"
	"github.com/ariel-frischer/autospec/internal/opencode"
)

// OpenCode implements the Agent interface for OpenCode CLI.
// Command: opencode run <prompt> --command <command-name>
type OpenCode struct {
	BaseAgent
}

// NewOpenCode creates a new OpenCode agent.
func NewOpenCode() *OpenCode {
	return &OpenCode{
		BaseAgent: BaseAgent{
			AgentName:   "opencode",
			Cmd:         "opencode",
			VersionFlag: "--version",
			AgentCaps: Caps{
				Automatable: true,
				PromptDelivery: PromptDelivery{
					Method:          PromptMethodSubcommandWithFlag,
					Flag:            "run",
					CommandFlag:     "--command",
					InteractiveFlag: "--prompt",
					ContextFileFlag: "-f",
				},
				// run subcommand is inherently non-interactive
				AutonomousFlag: "",
				RequiredEnv:    []string{},
				OptionalEnv:    []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GEMINI_API_KEY"},
			},
		},
	}
}

// ConfigureProject implements the Configurator interface for OpenCode.
// It configures the OpenCode agent for autospec:
//   - Installs command templates to .opencode/command/
//   - Adds 'autospec *': 'allow' bash permission to opencode.json
//   - Sets edit permission to 'allow' for file editing
//
// The projectLevel parameter determines where permissions are configured:
//   - false (default): writes to global config (~/.config/opencode/opencode.json)
//   - true: writes to project-level config (./opencode.json)
//
// This method is idempotent - calling it multiple times produces the same result.
func (o *OpenCode) ConfigureProject(projectDir, specsDir string, projectLevel bool) (ConfigResult, error) {
	// Install command templates (always project-level)
	if _, err := commands.InstallTemplatesForAgent("opencode", projectDir); err != nil {
		return ConfigResult{}, fmt.Errorf("installing opencode commands: %w", err)
	}
	if _, _, err := commands.InstallAgentSkills(projectDir, specsDir); err != nil {
		return ConfigResult{}, fmt.Errorf("installing shared autospec skills: %w", err)
	}

	// Configure opencode.json permissions
	var settings *opencode.Settings
	var err error
	var configLocation string

	if projectLevel {
		settings, err = opencode.Load(projectDir)
		configLocation = "project"
	} else {
		settings, err = opencode.LoadGlobal()
		configLocation = "global"
	}
	if err != nil {
		return ConfigResult{}, fmt.Errorf("loading opencode %s settings: %w", configLocation, err)
	}

	if settings.HasRequiredPermission() {
		return ConfigResult{
			AlreadyConfigured: true,
			SettingsFilePath:  settings.FilePath(),
		}, nil
	}

	// Check for explicit deny
	var warning string
	if settings.IsPermissionDenied() {
		warning = fmt.Sprintf("permission '%s' or edit is explicitly denied in %s opencode.json", opencode.RequiredPattern, configLocation)
	}

	// Add bash permission for autospec commands
	settings.AddBashPermission(opencode.RequiredPattern, opencode.PermissionAllow)

	// Set edit permission to allow (OpenCode uses simple string, not patterns)
	settings.SetEditPermission(opencode.PermissionAllow)

	if err := settings.Save(); err != nil {
		return ConfigResult{}, fmt.Errorf("saving opencode %s settings: %w", configLocation, err)
	}

	// Build list of permissions added
	permissionsAdded := []string{
		fmt.Sprintf("Bash(%s)", opencode.RequiredPattern),
		"Edit(allow)",
	}

	return ConfigResult{
		PermissionsAdded: permissionsAdded,
		Warning:          warning,
		SettingsFilePath: settings.FilePath(),
	}, nil
}
