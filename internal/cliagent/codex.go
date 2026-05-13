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
// Codex does not use autospec slash-command template directories, so setup only
// records a minimal project-local config marker when project-level init is used.
func (c *Codex) ConfigureProject(projectDir, specsDir string, projectLevel bool) (ConfigResult, error) {
	if !projectLevel {
		return ConfigResult{
			AlreadyConfigured: true,
		}, nil
	}

	configPath := filepath.Join(projectDir, ".codex", "config.toml")
	existed := fileExists(configPath)

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return ConfigResult{}, fmt.Errorf("creating codex config directory: %w", err)
	}
	if !existed {
		content := codexProjectConfigContent(specsDir)
		if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
			return ConfigResult{}, fmt.Errorf("writing codex config: %w", err)
		}
	}

	return ConfigResult{
		AlreadyConfigured: existed,
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
