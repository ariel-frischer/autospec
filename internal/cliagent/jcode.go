package cliagent

// Jcode implements the jcode coding-agent CLI.
// Command: jcode --quiet --no-update --no-selfdev run <prompt>
type Jcode struct {
	BaseAgent
}

// NewJcode creates a new jcode agent.
func NewJcode() *Jcode {
	return &Jcode{
		BaseAgent: BaseAgent{
			AgentName:   "jcode",
			Cmd:         "jcode",
			VersionFlag: "--version",
			AgentCaps: Caps{
				Automatable: true,
				PromptDelivery: PromptDelivery{
					Method: PromptMethodSubcommand,
					Flag:   "run",
				},
				RequiredEnv:             []string{},
				ExtraArgsBeforePrompt:   true,
				DefaultArgsBeforePrompt: true,
				DefaultArgs:             []string{"--quiet", "--no-update", "--no-selfdev"},
			},
		},
	}
}
