package cliagent

// Gemini implements the Agent interface for Google Gemini CLI.
// Command: gemini -p <prompt> [--yolo]
type Gemini struct {
	BaseAgent
}

// NewGemini creates a new Gemini CLI agent.
func NewGemini() *Gemini {
	return &Gemini{
		BaseAgent: BaseAgent{
			AgentName:   "gemini",
			Cmd:         "gemini",
			VersionFlag: "--version",
			AgentCaps: Caps{
				Automatable: true,
				PromptDelivery: PromptDelivery{
					Method: PromptMethodArg,
					Flag:   "-p",
				},
				AutonomousFlag: "--yolo",
				RequiredEnv:    []string{"GEMINI_API_KEY"},
				OptionalEnv:    []string{},
			},
		},
	}
}
