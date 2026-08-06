package cliagent

// ParseSessionSettings extracts the generic workflow arguments used by the
// executor and maps them to native-agent session settings.
func ParseSessionSettings(args []string) (model, reasoning string) {
	for i := 0; i+1 < len(args); i++ {
		switch args[i] {
		case "--model":
			model = args[i+1]
		case "-c":
			const prefix = "model_reasoning_effort="
			if len(args[i+1]) > len(prefix) && args[i+1][:len(prefix)] == prefix {
				reasoning = args[i+1][len(prefix):]
			}
		}
	}
	return model, reasoning
}
