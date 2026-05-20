// Package shared provides constants and types used across CLI subpackages.
package shared

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/ariel-frischer/autospec/internal/build"
	"github.com/ariel-frischer/autospec/internal/cliagent"
	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/workflow"

	"github.com/spf13/cobra"
)

// AgentFlagName is the flag name for agent override.
const AgentFlagName = "agent"

// OpenCodeAgentFlagName is the flag name for OpenCode sub-agent override.
const OpenCodeAgentFlagName = "opencode-agent"

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

// AddOpenCodeAgentFlag adds the --opencode-agent flag to a command.
func AddOpenCodeAgentFlag(cmd *cobra.Command) {
	if !build.MultiAgentEnabled() {
		return
	}
	cmd.Flags().String(OpenCodeAgentFlagName, "", "OpenCode sub-agent to use (Plan, Build, Explore, etc.)")
}

// ResolveOpenCodeAgent resolves the OpenCode agent from CLI flag or config.
// Priority: CLI flag > config > default (empty)
func ResolveOpenCodeAgent(cmd *cobra.Command, cfg *config.Configuration) string {
	if !build.MultiAgentEnabled() {
		return ""
	}

	// Check CLI flag first
	if agent, _ := cmd.Flags().GetString(OpenCodeAgentFlagName); agent != "" {
		return agent
	}

	// Fall back to config
	return cfg.OpenCodeAgent
}

// suggested: internal/cli/shared/agent.go
func ApplyOpenCodeAgent(cmd *cobra.Command, cfg *config.Configuration, orch *workflow.WorkflowOrchestrator) {
	agent := ResolveOpenCodeAgent(cmd, cfg)
	if agent == "" {
		return
	}
	if claude, ok := orch.Executor.Claude.(*workflow.ClaudeExecutor); ok {
		claude.OpenCodeAgent = agent
	} else {
		slog.Warn("--opencode-agent flag ignored: executor is not ClaudeExecutor")
	}
}
