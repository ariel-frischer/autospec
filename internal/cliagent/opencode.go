package cliagent

// OpenCode implements the Agent interface for OpenCode CLI.
// Command: opencode run <prompt>
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
					Method: PromptMethodSubcommand,
					Flag:   "run",
				},
				// run subcommand is inherently non-interactive
				AutonomousFlag: "",
				RequiredEnv:    []string{},
				OptionalEnv:    []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GEMINI_API_KEY"},
			},
		},
	}
}
