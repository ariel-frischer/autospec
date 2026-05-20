// Package shared provides constants and types used across CLI subpackages.
package shared

import (
	"fmt"
	"strings"

	"github.com/ariel-frischer/autospec/internal/build"
	"github.com/ariel-frischer/autospec/internal/cliagent"
	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/spf13/cobra"
)

// AgentFlagName is the flag name for agent override.
const AgentFlagName = "agent"

// OpenCodeModelFlagName is the flag name for one-shot OpenCode model overrides.
const OpenCodeModelFlagName = "opencode-model"

// availableAgentNames returns agent names available for the current build type.
// Production builds only show production agents; dev builds show all registered agents.
func availableAgentNames() []string {
	if build.IsDevBuild() {
		return cliagent.List()
	}
	return build.ProductionAgents()
}

// AddAgentFlag adds the --agent flag to a command.
// The flag allows users to override the configured agent for a single execution.
// In production builds (multi-agent disabled), this is a no-op.
func AddAgentFlag(cmd *cobra.Command) {
	if !build.MultiAgentEnabled() {
		return // No flag in production - Claude is always used
	}
	agents := availableAgentNames()
	if build.IsDevBuild() {
		cmd.Flags().String(AgentFlagName, "", fmt.Sprintf("[DEV] Override agent (available: %s)", strings.Join(agents, ", ")))
	} else {
		cmd.Flags().String(AgentFlagName, "", fmt.Sprintf("Override agent (available: %s)", strings.Join(agents, ", ")))
	}
}

// AddOpenCodeModelFlag adds the --opencode-model flag to a command.
func AddOpenCodeModelFlag(cmd *cobra.Command) {
	cmd.Flags().String(OpenCodeModelFlagName, "", "Use an OpenCode model for this workflow run (e.g., anthropic/claude-opus-4-5-latest)")
}

// ApplyOpenCodeModelOverride applies the --opencode-model flag as a one-run override.
func ApplyOpenCodeModelOverride(cmd *cobra.Command, cfg *config.Configuration) {
	model, _ := cmd.Flags().GetString(OpenCodeModelFlagName)
	if model == "" {
		return
	}
	cfg.OpenCode.ModelOverride = model
}

// ResolveAgent resolves the agent to use based on CLI flag and config.
// Priority: CLI flag > config (agent_preset/custom_agent_cmd) > legacy fields > default (claude).
// In production builds (multi-agent disabled), always returns Claude.
func ResolveAgent(cmd *cobra.Command, cfg *config.Configuration) (cliagent.Agent, error) {
	// In production builds, always use Claude
	if !build.MultiAgentEnabled() {
		return cliagent.Get("claude"), nil
	}

	// Check for CLI flag override
	agentName, _ := cmd.Flags().GetString(AgentFlagName)
	if agentName != "" {
		agent := cliagent.Get(agentName)
		if agent == nil {
			return nil, fmt.Errorf("unknown agent %q; available: %s", agentName, strings.Join(availableAgentNames(), ", "))
		}
		return agent, nil
	}

	// Fall back to config resolution
	return cfg.GetAgent()
}

// ApplyAgentOverride updates the configuration with an agent override from CLI flag.
// This modifies the config's AgentPreset field so that workflow orchestrator picks it up.
// Returns true if an override was applied.
// In production builds (multi-agent disabled), this is a no-op.
func ApplyAgentOverride(cmd *cobra.Command, cfg *config.Configuration) (bool, error) {
	// In production builds, no agent override is possible
	if !build.MultiAgentEnabled() {
		return false, nil
	}

	agentName, _ := cmd.Flags().GetString(AgentFlagName)
	if agentName == "" {
		return false, nil
	}

	// Validate agent exists
	agent := cliagent.Get(agentName)
	if agent == nil {
		return false, fmt.Errorf("unknown agent %q; available: %s", agentName, strings.Join(availableAgentNames(), ", "))
	}

	// Override config to use this agent
	cfg.AgentPreset = agentName
	// Clear custom_agent to ensure preset takes effect
	cfg.CustomAgent = nil

	return true, nil
}
