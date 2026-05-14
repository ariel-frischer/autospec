package cliagent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
				AutonomousFlag: "--dangerously-bypass-approvals-and-sandbox",
				RequiredEnv:    []string{},
				OptionalEnv:    []string{"OPENAI_API_KEY", "CODEX_HOME", "OPENAI_BASE_URL"},
			},
		},
	}
}

// ConfigureProject implements Configurator for Codex.
// Codex does not use Claude/OpenCode slash-command template directories.
// Project setup installs a Codex skill that maps interactive /autospec.*
// requests back to autospec CLI commands, and registers that skill in
// .codex/config.toml.
func (c *Codex) ConfigureProject(projectDir, specsDir string, projectLevel bool) (ConfigResult, error) {
	if !projectLevel {
		return ConfigResult{
			AlreadyConfigured: true,
		}, nil
	}

	configPath := filepath.Join(projectDir, ".codex", "config.toml")
	configExisted := fileExists(configPath)
	skillPath := filepath.Join(projectDir, ".codex", "skills", "autospec", "SKILL.md")
	skillExisted := fileExists(skillPath)

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return ConfigResult{}, fmt.Errorf("creating codex config directory: %w", err)
	}
	if !configExisted {
		content := codexProjectConfigContent(specsDir)
		if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
			return ConfigResult{}, fmt.Errorf("writing codex config: %w", err)
		}
	}
	if err := ensureCodexSkillRegistered(configPath); err != nil {
		return ConfigResult{}, fmt.Errorf("registering codex autospec skill: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		return ConfigResult{}, fmt.Errorf("creating codex skill directory: %w", err)
	}
	if err := os.WriteFile(skillPath, []byte(codexAutospecSkillContent(specsDir)), 0o644); err != nil {
		return ConfigResult{}, fmt.Errorf("writing codex autospec skill: %w", err)
	}

	return ConfigResult{
		AlreadyConfigured: configExisted && skillExisted,
		SettingsFilePath:  configPath,
	}, nil
}

func codexProjectConfigContent(specsDir string) string {
	specsDir = strings.TrimSpace(specsDir)
	if specsDir == "" {
		specsDir = "specs"
	}
	return fmt.Sprintf(`# autospec project metadata for Codex CLI.
# autospec stores feature workflow artifacts in %q.
# Runtime autonomy is controlled by autospec skip_permissions and Codex CLI flags.
`, specsDir)
}

func ensureCodexSkillRegistered(configPath string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading codex config: %w", err)
	}
	text := string(content)
	if strings.Contains(text, `path = ".codex/skills/autospec"`) ||
		strings.Contains(text, `path = './.codex/skills/autospec'`) ||
		strings.Contains(text, `path = ".\.codex\skills\autospec"`) {
		return nil
	}
	if strings.TrimSpace(text) != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += `
[[skills.config]]
path = ".codex/skills/autospec"
enabled = true
`
	if err := os.WriteFile(configPath, []byte(text), 0o644); err != nil {
		return fmt.Errorf("writing codex config: %w", err)
	}
	return nil
}

func codexAutospecSkillContent(specsDir string) string {
	specsDir = strings.TrimSpace(specsDir)
	if specsDir == "" {
		specsDir = "specs"
	}
	return fmt.Sprintf(`---
name: autospec-commands
description: Use when the user invokes /autospec.* commands, asks to run autospec stages, or wants spec-driven autospec workflow automation.
---

# Autospec Codex Skill

This project uses autospec for spec-driven development. Treat user text like
"/autospec.specify", "/autospec.plan", "/autospec.tasks",
"/autospec.implement", "/autospec.constitution", "/autospec.clarify",
"/autospec.analyze", and "/autospec.checklist" as autospec command aliases,
not as prose requests to complete manually.

## Command Routing

When the user invokes one of these aliases in an interactive Codex session, run
the matching autospec CLI command from the repository root:

| User input | Run |
|------------|-----|
| "/autospec.specify \"desc\"" | "autospec specify \"desc\"" |
| "/autospec.plan" | "autospec plan" |
| "/autospec.tasks" | "autospec tasks" |
| "/autospec.implement" | "autospec implement" |
| "/autospec.constitution \"guidance\"" | "autospec constitution \"guidance\"" |
| "/autospec.clarify" | "autospec clarify" |
| "/autospec.analyze" | "autospec analyze" |
| "/autospec.checklist" | "autospec checklist" |

Pass quoted user arguments through unchanged. Do not hand-generate "spec.yaml",
"plan.yaml", or "tasks.yaml" for an interactive slash-style request unless
autospec itself launches you with a rendered stage prompt.

## Rendered Autospec Prompts

When autospec launches Codex through "codex exec", the prompt is already
rendered and may include sections such as "## User Input" and "## Outline". In
that case, follow the rendered instructions directly and write the requested
autospec artifact under "%s/".

## Workflow

Core stages are:

"constitution -> specify -> plan -> tasks -> implement"

The project stores workflow artifacts under "%s/".

Always let the autospec CLI validate artifacts. If a command fails, read the
error and fix the underlying artifact or prerequisite before reporting success.
`, specsDir, specsDir)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
