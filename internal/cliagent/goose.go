package cliagent

// Goose implements the Agent interface for Goose CLI (Block/Linux Foundation).
// Command: goose run -t <prompt> [--no-session]
// Env: GOOSE_MODE=auto for autonomous mode
type Goose struct {
	BaseAgent
}

// NewGoose creates a new Goose agent.
func NewGoose() *Goose {
	return &Goose{
		BaseAgent: BaseAgent{
			AgentName:   "goose",
			Cmd:         "goose",
			VersionFlag: "--version",
			AgentCaps: Caps{
				Automatable: true,
				PromptDelivery: PromptDelivery{
					Method:     PromptMethodSubcommandArg,
					Flag:       "run",
					PromptFlag: "-t",
				},
				AutonomousFlag: "--no-session",
				AutonomousEnv: map[string]string{
					"GOOSE_MODE": "auto",
				},
				RequiredEnv: []string{},
				OptionalEnv: []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY"},
			},
		},
	}
}
