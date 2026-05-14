package cliagent

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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

	skillNames, allSkillsExisted, err := installCodexCommandSkills(projectDir, specsDir)
	if err != nil {
		return ConfigResult{}, err
	}
	if err := ensureCodexSkillsRegistered(configPath, skillNames); err != nil {
		return ConfigResult{}, fmt.Errorf("registering codex autospec skills: %w", err)
	}
	cleanupObsoleteCodexAutospecRouter(projectDir)

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

func installCodexCommandSkills(projectDir, specsDir string) ([]string, bool, error) {
	templates, err := commands.ListTemplates()
	if err != nil {
		return nil, false, fmt.Errorf("listing autospec templates: %w", err)
	}

	allExisted := true
	var skillNames []string
	for _, tpl := range templates {
		if !strings.HasPrefix(tpl.Name, "autospec.") {
			continue
		}
		skillName := codexSkillName(tpl.Name)
		skillNames = append(skillNames, skillName)
		skillPath := filepath.Join(projectDir, ".codex", "skills", skillName, "SKILL.md")
		if !fileExists(skillPath) {
			allExisted = false
		}
		if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
			return nil, false, fmt.Errorf("creating codex skill directory: %w", err)
		}
		content := codexCommandSkillContent(tpl, specsDir)
		if err := os.WriteFile(skillPath, []byte(content), 0o644); err != nil {
			return nil, false, fmt.Errorf("writing codex skill %s: %w", skillName, err)
		}
	}
	sort.Strings(skillNames)
	return skillNames, allExisted, nil
}

func codexSkillName(commandName string) string {
	return strings.ReplaceAll(commandName, ".", "-")
}

func codexCommandSkillContent(tpl commands.CommandTemplate, specsDir string) string {
	description := strings.TrimSpace(tpl.Description)
	if description == "" {
		description = "Run the " + tpl.Name + " autospec workflow prompt."
	}
	skillName := codexSkillName(tpl.Name)
	body := string(commands.StripFrontmatter(tpl.Content))
	body = rewriteAutospecSlashCommandsForCodexSkills(body)

	return fmt.Sprintf(`---
name: %s
description: %s
---

# %s

This Codex skill is generated from %s. When the user invokes "$%s" or "%s",
load and follow these instructions directly. Treat the text after the skill or
command name as "$ARGUMENTS". Do not route back through "autospec %s"; this skill
is the prompt for the stage.

Project specs directory: %s

%s`, skillName, strconv.Quote(description), skillName, tpl.Name, skillName, "/"+tpl.Name, strings.TrimPrefix(tpl.Name, "autospec."), specsDir, body)
}

func rewriteAutospecSlashCommandsForCodexSkills(body string) string {
	for _, commandName := range autospecCommandSkillNames() {
		body = strings.ReplaceAll(body, "/"+commandName, "$"+codexSkillName(commandName))
	}
	return body
}

func autospecCommandSkillNames() []string {
	return []string{
		"autospec.analyze",
		"autospec.checklist",
		"autospec.clarify",
		"autospec.constitution",
		"autospec.implement",
		"autospec.plan",
		"autospec.specify",
		"autospec.tasks",
		"autospec.worktree-setup",
	}
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
path = ".codex/skills/%s"
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
