package cliagent

import (
	"strings"
	"testing"
)

// TestAllAgentsRegistered verifies that all Tier 1 agents are registered.
func TestAllAgentsRegistered(t *testing.T) {
	t.Parallel()

	expected := []string{"claude", "cline", "codex", "gemini", "goose", "opencode"}
	registered := List()

	if len(registered) != len(expected) {
		t.Errorf("expected %d agents registered, got %d: %v", len(expected), len(registered), registered)
	}

	for _, name := range expected {
		if Get(name) == nil {
			t.Errorf("agent %q should be registered but was not found", name)
		}
	}
}

// TestAgentInterface verifies all agents implement the Agent interface correctly.
func TestAgentInterface(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agent       Agent
		wantName    string
		wantCmd     string
		wantMethod  PromptMethod
		wantFlag    string
		wantAutonom string
	}{
		"claude": {
			agent:       NewClaude(),
			wantName:    "claude",
			wantCmd:     "claude",
			wantMethod:  PromptMethodArg,
			wantFlag:    "-p",
			wantAutonom: "--dangerously-skip-permissions",
		},
		"cline": {
			agent:       NewCline(),
			wantName:    "cline",
			wantCmd:     "cline",
			wantMethod:  PromptMethodPositional,
			wantFlag:    "",
			wantAutonom: "-Y",
		},
		"gemini": {
			agent:       NewGemini(),
			wantName:    "gemini",
			wantCmd:     "gemini",
			wantMethod:  PromptMethodArg,
			wantFlag:    "-p",
			wantAutonom: "--yolo",
		},
		"codex": {
			agent:       NewCodex(),
			wantName:    "codex",
			wantCmd:     "codex",
			wantMethod:  PromptMethodSubcommand,
			wantFlag:    "exec",
			wantAutonom: "",
		},
		"opencode": {
			agent:       NewOpenCode(),
			wantName:    "opencode",
			wantCmd:     "opencode",
			wantMethod:  PromptMethodSubcommand,
			wantFlag:    "run",
			wantAutonom: "",
		},
		"goose": {
			agent:       NewGoose(),
			wantName:    "goose",
			wantCmd:     "goose",
			wantMethod:  PromptMethodSubcommandArg,
			wantFlag:    "run",
			wantAutonom: "--no-session",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := tt.agent.Name(); got != tt.wantName {
				t.Errorf("Name() = %q, want %q", got, tt.wantName)
			}

			caps := tt.agent.Capabilities()
			if !caps.Automatable {
				t.Error("Automatable should be true for Tier 1 agents")
			}
			if caps.PromptDelivery.Method != tt.wantMethod {
				t.Errorf("PromptDelivery.Method = %q, want %q", caps.PromptDelivery.Method, tt.wantMethod)
			}
			if caps.PromptDelivery.Flag != tt.wantFlag {
				t.Errorf("PromptDelivery.Flag = %q, want %q", caps.PromptDelivery.Flag, tt.wantFlag)
			}
			if caps.AutonomousFlag != tt.wantAutonom {
				t.Errorf("AutonomousFlag = %q, want %q", caps.AutonomousFlag, tt.wantAutonom)
			}
		})
	}
}

// TestBuildCommand verifies that BuildCommand produces correct CLI invocations.
func TestBuildCommand(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agent    Agent
		prompt   string
		opts     ExecOptions
		wantArgs []string
		wantEnv  string // Check for specific env var in autonomous mode
	}{
		"claude basic": {
			agent:    NewClaude(),
			prompt:   "fix the bug",
			opts:     ExecOptions{},
			wantArgs: []string{"-p", "fix the bug", "--verbose", "--output-format", "stream-json"},
		},
		"claude autonomous": {
			agent:    NewClaude(),
			prompt:   "fix the bug",
			opts:     ExecOptions{Autonomous: true},
			wantArgs: []string{"-p", "fix the bug", "--verbose", "--output-format", "stream-json", "--dangerously-skip-permissions"},
		},
		"cline basic": {
			agent:    NewCline(),
			prompt:   "refactor module",
			opts:     ExecOptions{},
			wantArgs: []string{"refactor module"},
		},
		"cline autonomous": {
			agent:    NewCline(),
			prompt:   "refactor module",
			opts:     ExecOptions{Autonomous: true},
			wantArgs: []string{"refactor module", "-Y"},
		},
		"gemini basic": {
			agent:    NewGemini(),
			prompt:   "analyze code",
			opts:     ExecOptions{},
			wantArgs: []string{"-p", "analyze code"},
		},
		"gemini autonomous": {
			agent:    NewGemini(),
			prompt:   "analyze code",
			opts:     ExecOptions{Autonomous: true},
			wantArgs: []string{"-p", "analyze code", "--yolo"},
		},
		"codex basic": {
			agent:    NewCodex(),
			prompt:   "fix tests",
			opts:     ExecOptions{},
			wantArgs: []string{"exec", "fix tests"},
		},
		"codex autonomous (no extra flag)": {
			agent:    NewCodex(),
			prompt:   "fix tests",
			opts:     ExecOptions{Autonomous: true},
			wantArgs: []string{"exec", "fix tests"},
		},
		"opencode basic": {
			agent:    NewOpenCode(),
			prompt:   "update deps",
			opts:     ExecOptions{},
			wantArgs: []string{"run", "update deps"},
		},
		"goose basic": {
			agent:    NewGoose(),
			prompt:   "add feature",
			opts:     ExecOptions{},
			wantArgs: []string{"run", "-t", "add feature"},
		},
		"goose autonomous": {
			agent:    NewGoose(),
			prompt:   "add feature",
			opts:     ExecOptions{Autonomous: true},
			wantArgs: []string{"run", "-t", "add feature", "--no-session"},
			wantEnv:  "GOOSE_MODE=auto",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd, err := tt.agent.BuildCommand(tt.prompt, tt.opts)
			if err != nil {
				t.Fatalf("BuildCommand() error = %v", err)
			}

			// Skip command name, compare args
			gotArgs := cmd.Args[1:]
			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("args = %v, want %v", gotArgs, tt.wantArgs)
				return
			}
			for i, arg := range gotArgs {
				if arg != tt.wantArgs[i] {
					t.Errorf("args[%d] = %q, want %q", i, arg, tt.wantArgs[i])
				}
			}

			// Check for autonomous env var if expected
			if tt.wantEnv != "" {
				found := false
				for _, e := range cmd.Env {
					if e == tt.wantEnv {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected env var %q in command environment", tt.wantEnv)
				}
			}
		})
	}
}

// TestGoosePromptFlag verifies Goose uses the correct prompt flag for subcommand-arg.
func TestGoosePromptFlag(t *testing.T) {
	t.Parallel()

	agent := NewGoose()
	caps := agent.Capabilities()

	if caps.PromptDelivery.PromptFlag != "-t" {
		t.Errorf("Goose PromptFlag = %q, want %q", caps.PromptDelivery.PromptFlag, "-t")
	}
}

// TestClaudeRequiredEnv verifies Claude has no required env vars.
// Claude works with either subscription (Pro/Max) or API key, so ANTHROPIC_API_KEY is optional.
func TestClaudeRequiredEnv(t *testing.T) {
	t.Parallel()

	agent := NewClaude()
	caps := agent.Capabilities()

	if len(caps.RequiredEnv) != 0 {
		t.Errorf("Claude RequiredEnv = %v, want [] (empty - no required env vars)", caps.RequiredEnv)
	}

	// ANTHROPIC_API_KEY should be in OptionalEnv
	hasAPIKey := false
	for _, env := range caps.OptionalEnv {
		if env == "ANTHROPIC_API_KEY" {
			hasAPIKey = true
			break
		}
	}
	if !hasAPIKey {
		t.Errorf("Claude OptionalEnv = %v, want to contain ANTHROPIC_API_KEY", caps.OptionalEnv)
	}
}

// TestGeminiRequiredEnv verifies Gemini requires GEMINI_API_KEY.
func TestGeminiRequiredEnv(t *testing.T) {
	t.Parallel()

	agent := NewGemini()
	caps := agent.Capabilities()

	if len(caps.RequiredEnv) != 1 || caps.RequiredEnv[0] != "GEMINI_API_KEY" {
		t.Errorf("Gemini RequiredEnv = %v, want [GEMINI_API_KEY]", caps.RequiredEnv)
	}
}

// TestCodexRequiredEnv verifies Codex requires OPENAI_API_KEY.
func TestCodexRequiredEnv(t *testing.T) {
	t.Parallel()

	agent := NewCodex()
	caps := agent.Capabilities()

	if len(caps.RequiredEnv) != 1 || caps.RequiredEnv[0] != "OPENAI_API_KEY" {
		t.Errorf("Codex RequiredEnv = %v, want [OPENAI_API_KEY]", caps.RequiredEnv)
	}
}

// TestAgentNamesLowercase verifies all agent names are lowercase.
func TestAgentNamesLowercase(t *testing.T) {
	t.Parallel()

	for _, name := range List() {
		if name != strings.ToLower(name) {
			t.Errorf("agent name %q should be lowercase", name)
		}
	}
}
