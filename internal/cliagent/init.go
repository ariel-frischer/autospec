package cliagent

// init registers all built-in Tier 1 agents with the default registry.
// This is called automatically when the package is imported.
func init() {
	Register(NewClaude())
	Register(NewCline())
	Register(NewGemini())
	Register(NewCodex())
	Register(NewOpenCode())
	Register(NewGoose())
}
