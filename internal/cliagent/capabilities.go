package cliagent

// PromptMethod defines how a prompt is passed to the agent CLI.
type PromptMethod string

const (
	// PromptMethodArg passes the prompt via a flag (e.g., "-p", "--message").
	// Example: claude -p "fix the bug"
	PromptMethodArg PromptMethod = "arg"

	// PromptMethodPositional passes the prompt as a positional argument.
	// Example: cline "fix the bug"
	PromptMethodPositional PromptMethod = "positional"

	// PromptMethodSubcommand uses a subcommand with positional prompt.
	// Example: codex exec "fix the bug"
	PromptMethodSubcommand PromptMethod = "subcommand"

	// PromptMethodSubcommandArg uses a subcommand with a flag for the prompt.
	// Example: goose run -t "fix the bug"
	PromptMethodSubcommandArg PromptMethod = "subcommand-arg"

	// PromptMethodTemplate uses {{PROMPT}} placeholder expansion.
	// Example: aider --message {{PROMPT}}
	PromptMethodTemplate PromptMethod = "template"
)

// PromptDelivery describes how to pass prompts to an agent CLI.
type PromptDelivery struct {
	// Method specifies the prompt passing pattern.
	Method PromptMethod

	// Flag is the primary flag or subcommand name.
	// For PromptMethodArg: the flag (e.g., "-p", "--message")
	// For PromptMethodSubcommand/SubcommandArg: the subcommand (e.g., "exec", "run")
	Flag string

	// PromptFlag is the secondary flag for the prompt after the subcommand.
	// Only used with PromptMethodSubcommandArg (e.g., "-t" for "goose run -t").
	PromptFlag string
}

// Caps contains self-describing feature flags for agent discovery and automation.
type Caps struct {
	// Automatable indicates whether the agent can run fully headless without user input.
	// Required for autospec automation.
	Automatable bool

	// PromptDelivery describes how to pass prompts to this agent.
	PromptDelivery PromptDelivery

	// AutonomousFlag is the CLI flag to skip confirmations (e.g., "--dangerously-skip-permissions").
	// Empty string if not needed or if autonomous mode is the default.
	AutonomousFlag string

	// AutonomousEnv contains environment variables required for autonomous mode.
	// Example: {"GOOSE_MODE": "auto"}
	AutonomousEnv map[string]string

	// RequiredEnv lists environment variable names required for the agent (API keys, etc.).
	// Validation fails if any are missing.
	RequiredEnv []string

	// OptionalEnv lists optional environment variables.
	// Informational only - validation does not fail if missing.
	OptionalEnv []string
}
