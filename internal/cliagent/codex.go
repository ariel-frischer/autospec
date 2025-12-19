package cliagent

// Codex implements the Agent interface for OpenAI Codex CLI.
// Command: codex exec <prompt>
type Codex struct {
	BaseAgent
}

// NewCodex creates a new Codex CLI agent.
func NewCodex() *Codex {
	return &Codex{
		BaseAgent: BaseAgent{
			AgentName:   "codex",
			Cmd:         "codex",
			VersionFlag: "--version",
			AgentCaps: Caps{
				Automatable: true,
				PromptDelivery: PromptDelivery{
					Method: PromptMethodSubcommand,
					Flag:   "exec",
				},
				// exec mode is inherently autonomous, no extra flag needed
				AutonomousFlag: "",
				RequiredEnv:    []string{"OPENAI_API_KEY"},
				OptionalEnv:    []string{},
			},
		},
	}
}
