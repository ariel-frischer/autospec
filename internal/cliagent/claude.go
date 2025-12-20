package cliagent

import (
	"fmt"

	"github.com/ariel-frischer/autospec/internal/claude"
)

// Claude implements the Agent interface for Claude Code CLI.
// Command: claude -p <prompt> [--dangerously-skip-permissions]
type Claude struct {
	BaseAgent
}

// NewClaude creates a new Claude Code agent.
// Note: ANTHROPIC_API_KEY is optional - Claude works with subscription (Pro/Max) when not set.
// The use_subscription config option (default: true) forces subscription mode by clearing any API key.
func NewClaude() *Claude {
	return &Claude{
		BaseAgent: BaseAgent{
			AgentName:   "claude",
			Cmd:         "claude",
			VersionFlag: "--version",
			AgentCaps: Caps{
				Automatable: true,
				PromptDelivery: PromptDelivery{
					Method: PromptMethodArg,
					Flag:   "-p",
				},
				AutonomousFlag: "--dangerously-skip-permissions",
				RequiredEnv:    []string{}, // No required env - works with subscription or API
				OptionalEnv:    []string{"ANTHROPIC_API_KEY", "CLAUDE_MODEL"},
				// DefaultArgs enables stream-json output for better terminal parsing.
				// --verbose is required with stream-json or Claude will error.
				DefaultArgs: []string{"--verbose", "--output-format", "stream-json"},
			},
		},
	}
}

// ConfigureProject implements the Configurator interface for Claude.
// It configures .claude/settings.local.json with required permissions for autospec:
//   - Bash(autospec:*) - run autospec commands
//   - Write(.autospec/**) - write to .autospec directory
//   - Edit(.autospec/**) - edit files in .autospec directory
//   - Write({specsDir}/**) - write to specs directory
//   - Edit({specsDir}/**) - edit files in specs directory
//
// This method is idempotent - calling it multiple times produces the same result.
func (c *Claude) ConfigureProject(projectDir, specsDir string) (ConfigResult, error) {
	settings, err := claude.Load(projectDir)
	if err != nil {
		return ConfigResult{}, fmt.Errorf("loading claude settings: %w", err)
	}

	permissions := buildClaudePermissions(specsDir)

	// Check for deny list conflicts before adding permissions
	warning := checkDenyConflicts(settings, permissions)

	added := settings.AddPermissions(permissions)

	if len(added) == 0 {
		return ConfigResult{
			AlreadyConfigured: true,
			Warning:           warning,
		}, nil
	}

	if err := settings.Save(); err != nil {
		return ConfigResult{}, fmt.Errorf("saving claude settings: %w", err)
	}

	return ConfigResult{
		PermissionsAdded: added,
		Warning:          warning,
	}, nil
}

// buildClaudePermissions generates the list of permissions required for autospec.
func buildClaudePermissions(specsDir string) []string {
	return []string{
		"Bash(autospec:*)",
		"Write(.autospec/**)",
		"Edit(.autospec/**)",
		fmt.Sprintf("Write(%s/**)", specsDir),
		fmt.Sprintf("Edit(%s/**)", specsDir),
	}
}

// checkDenyConflicts checks if any required permissions are in the deny list.
// Returns a warning message if conflicts are found, empty string otherwise.
func checkDenyConflicts(settings *claude.Settings, permissions []string) string {
	var denied []string
	for _, perm := range permissions {
		if settings.CheckDenyList(perm) {
			denied = append(denied, perm)
		}
	}

	if len(denied) == 0 {
		return ""
	}

	if len(denied) == 1 {
		return fmt.Sprintf("permission %s is explicitly denied in settings", denied[0])
	}
	return fmt.Sprintf("permissions %v are explicitly denied in settings", denied)
}

// GetSandboxPaths returns the paths required for Claude sandbox write access.
// These are paths Claude needs to write to during workflow execution.
// Note: autospec's own state/config (~/.autospec/, ~/.config/autospec/) are NOT
// included - those are written by autospec itself, not by Claude.
func (c *Claude) GetSandboxPaths(specsDir string) []string {
	return []string{
		".autospec",
		specsDir,
	}
}

// GetSandboxDiff returns the sandbox configuration difference without applying it.
// Use this to show the user what would be changed before prompting for confirmation.
func (c *Claude) GetSandboxDiff(projectDir, specsDir string) (*claude.SandboxConfig, error) {
	settings, err := claude.Load(projectDir)
	if err != nil {
		return nil, fmt.Errorf("loading claude settings: %w", err)
	}

	requiredPaths := c.GetSandboxPaths(specsDir)
	diff := settings.GetSandboxConfigDiff(requiredPaths)
	return &diff, nil
}

// ConfigureSandbox implements the SandboxConfigurator interface for Claude.
// It enables sandbox and adds required write paths to the sandbox configuration.
func (c *Claude) ConfigureSandbox(projectDir, specsDir string) (SandboxResult, error) {
	settings, err := claude.Load(projectDir)
	if err != nil {
		return SandboxResult{}, fmt.Errorf("loading claude settings: %w", err)
	}

	requiredPaths := c.GetSandboxPaths(specsDir)
	diff := settings.GetSandboxConfigDiff(requiredPaths)

	wasEnabled := diff.Enabled
	needsEnable := !wasEnabled
	needsPaths := len(diff.PathsToAdd) > 0

	// Already fully configured: sandbox enabled and all paths present
	if !needsEnable && !needsPaths {
		return SandboxResult{
			AlreadyConfigured: true,
			ExistingPaths:     diff.ExistingPaths,
			SandboxEnabled:    true,
		}, nil
	}

	// Enable sandbox if not already enabled
	if needsEnable {
		settings.EnableSandbox()
	}

	// Add required paths
	var added []string
	if needsPaths {
		added = settings.AddWritePaths(diff.PathsToAdd)
	}

	if err := settings.Save(); err != nil {
		return SandboxResult{}, fmt.Errorf("saving claude settings: %w", err)
	}

	return SandboxResult{
		PathsAdded:        added,
		ExistingPaths:     diff.ExistingPaths,
		SandboxEnabled:    true,
		SandboxWasEnabled: needsEnable,
	}, nil
}
