package cliagent

// Claude implements the Agent interface for Claude Code CLI.
// Command: claude -p <prompt> [--dangerously-skip-permissions]
type Claude struct {
	BaseAgent
}

// NewClaude creates a new Claude Code agent.
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
				RequiredEnv:    []string{"ANTHROPIC_API_KEY"},
				OptionalEnv:    []string{"CLAUDE_MODEL"},
			},
		},
	}
}
