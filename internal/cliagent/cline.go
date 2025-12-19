package cliagent

// Cline implements the Agent interface for Cline CLI.
// Command: cline <prompt> [-Y] (YOLO mode)
type Cline struct {
	BaseAgent
}

// NewCline creates a new Cline agent.
func NewCline() *Cline {
	return &Cline{
		BaseAgent: BaseAgent{
			AgentName:   "cline",
			Cmd:         "cline",
			VersionFlag: "--version",
			AgentCaps: Caps{
				Automatable: true,
				PromptDelivery: PromptDelivery{
					Method: PromptMethodPositional,
				},
				AutonomousFlag: "-Y",
				RequiredEnv:    []string{},
				OptionalEnv:    []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY"},
			},
		},
	}
}
