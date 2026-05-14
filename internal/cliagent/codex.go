package cliagent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ariel-frischer/autospec/internal/commands"
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
// Project setup installs Codex skills generated from autospec command
// templates, and registers those skills in .codex/config.toml.
func (c *Codex) ConfigureProject(projectDir, specsDir string, projectLevel bool) (ConfigResult, error) {
	if !projectLevel {
		return ConfigResult{
			AlreadyConfigured: true,
		}, nil
	}

	configPath := filepath.Join(projectDir, ".codex", "config.toml")
	configExisted := fileExists(configPath)

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return ConfigResult{}, fmt.Errorf("creating codex config directory: %w", err)
	}
	if !configExisted {
		content := codexProjectConfigContent(specsDir)
		if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
			return ConfigResult{}, fmt.Errorf("writing codex config: %w", err)
		}
	}

	skillNames, allSkillsExisted, err := commands.InstallAgentSkills(projectDir, specsDir)
	if err != nil {
		return ConfigResult{}, err
	}
	if err := ensureCodexSkillsRegistered(configPath, skillNames); err != nil {
		return ConfigResult{}, fmt.Errorf("registering codex autospec skills: %w", err)
	}
	cleanupObsoleteCodexSkills(projectDir, skillNames)

	return ConfigResult{
		AlreadyConfigured: configExisted && allSkillsExisted,
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

func ensureCodexSkillsRegistered(configPath string, skillNames []string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading codex config: %w", err)
	}
	text := removeGeneratedCodexSkillBlocks(string(content), skillNames)
	if strings.TrimSpace(text) != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	for _, skillName := range skillNames {
		text += fmt.Sprintf(`
[[skills.config]]
path = ".agents/skills/%s"
enabled = true
`, skillName)
	}
	if err := os.WriteFile(configPath, []byte(text), 0o644); err != nil {
		return fmt.Errorf("writing codex config: %w", err)
	}
	return nil
}

func removeGeneratedCodexSkillBlocks(text string, skillNames []string) string {
	generatedPaths := map[string]bool{
		`.codex/skills/autospec`: true,
	}
	for _, skillName := range skillNames {
		generatedPaths[`.codex/skills/`+skillName] = true
		generatedPaths[`.agents/skills/`+skillName] = true
	}

	lines := strings.Split(text, "\n")
	var kept []string
	for i := 0; i < len(lines); {
		if strings.TrimSpace(lines[i]) != "[[skills.config]]" {
			kept = append(kept, lines[i])
			i++
			continue
		}
		j := i + 1
		for j < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[j]), "[[") {
			j++
		}
		block := strings.Join(lines[i:j], "\n")
		if codexSkillBlockUsesGeneratedPath(block, generatedPaths) {
			i = j
			continue
		}
		kept = append(kept, lines[i:j]...)
		i = j
	}
	return strings.TrimRight(strings.Join(kept, "\n"), "\n") + "\n"
}

func codexSkillBlockUsesGeneratedPath(block string, generatedPaths map[string]bool) bool {
	for path := range generatedPaths {
		if strings.Contains(block, `path = "`+path+`"`) ||
			strings.Contains(block, `path = '`+path+`'`) {
			return true
		}
	}
	return false
}

func cleanupObsoleteCodexSkills(projectDir string, skillNames []string) {
	cleanupObsoleteCodexAutospecRouter(projectDir)
	for _, skillName := range skillNames {
		cleanupObsoleteCodexCommandSkill(projectDir, skillName)
	}
}

func cleanupObsoleteCodexAutospecRouter(projectDir string) {
	path := filepath.Join(projectDir, ".codex", "skills", "autospec", "SKILL.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}
	text := string(content)
	if strings.Contains(text, "name: autospec-commands") &&
		strings.Contains(text, "## Command Routing") {
		_ = os.Remove(path)
		_ = os.Remove(filepath.Dir(path))
	}
}

func cleanupObsoleteCodexCommandSkill(projectDir, skillName string) {
	path := filepath.Join(projectDir, ".codex", "skills", skillName, "SKILL.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}
	text := string(content)
	if strings.Contains(text, "This Codex skill is generated from autospec.") &&
		strings.Contains(text, "Do not route back through") {
		_ = os.Remove(path)
		_ = os.Remove(filepath.Dir(path))
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
