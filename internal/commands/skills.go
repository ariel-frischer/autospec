package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// SharedSkillsDir is the project-local skill directory supported by Codex,
// OpenCode, and compatible future agents.
const SharedSkillsDir = ".agents/skills"

// ClaudeSkillsDir is Claude Code's project skill directory.
const ClaudeSkillsDir = ".claude/skills"

// InstallAgentSkills installs generated autospec Agent Skills to .agents/skills.
func InstallAgentSkills(projectDir, specsDir string) ([]string, bool, error) {
	return installSkills(projectDir, specsDir, SharedSkillsDir, CommandSkillName, CommandSkillContent)
}

// InstallClaudeSkills installs generated autospec skills to .claude/skills.
func InstallClaudeSkills(projectDir, specsDir string) ([]string, bool, error) {
	return installSkills(projectDir, specsDir, ClaudeSkillsDir, ClaudeSkillDirName, ClaudeSkillContent)
}

func installSkills(
	projectDir, specsDir, skillsDir string,
	dirName func(string) string,
	content func(CommandTemplate, string) string,
) ([]string, bool, error) {
	templates, err := ListTemplates()
	if err != nil {
		return nil, false, fmt.Errorf("listing autospec templates: %w", err)
	}

	allExisted := true
	var skillNames []string
	for _, tpl := range templates {
		if !strings.HasPrefix(tpl.Name, "autospec.") {
			continue
		}

		skillName := dirName(tpl.Name)
		skillNames = append(skillNames, skillName)
		skillPath := filepath.Join(projectDir, skillsDir, skillName, "SKILL.md")
		if _, err := os.Stat(skillPath); err != nil {
			allExisted = false
		}

		if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
			return nil, false, fmt.Errorf("creating agent skill directory: %w", err)
		}
		if err := os.WriteFile(skillPath, []byte(content(tpl, specsDir)), 0o644); err != nil {
			return nil, false, fmt.Errorf("writing agent skill %s: %w", skillName, err)
		}
	}

	sort.Strings(skillNames)
	return skillNames, allExisted, nil
}

// CommandSkillName converts an autospec command name to the Agent Skill name.
func CommandSkillName(commandName string) string {
	return strings.ReplaceAll(commandName, ".", "-")
}

// ClaudeSkillDirName preserves existing Claude slash command names like /autospec.specify.
func ClaudeSkillDirName(commandName string) string {
	return commandName
}

// CommandSkillContent converts an autospec command template to Agent Skills format.
func CommandSkillContent(tpl CommandTemplate, specsDir string) string {
	description := strings.TrimSpace(tpl.Description)
	if description == "" {
		description = "Run the " + tpl.Name + " autospec workflow prompt."
	}

	skillName := CommandSkillName(tpl.Name)
	body := string(StripFrontmatter(tpl.Content))
	body = RewriteAutospecSlashCommandsForSkills(body)

	return fmt.Sprintf(`---
name: %s
description: %s
---

# %s

This Agent Skill is generated from %s. When the user invokes "$%s" or "%s",
load and follow these instructions directly. Treat the text after the skill or
command name as "$ARGUMENTS". Do not route back through "autospec %s"; this skill
is the prompt for the stage.

Project specs directory: %s

%s`, skillName, strconv.Quote(description), skillName, tpl.Name, skillName, "/"+tpl.Name, strings.TrimPrefix(tpl.Name, "autospec."), specsDir, body)
}

// ClaudeSkillContent converts an autospec command template to Claude Code skill format.
func ClaudeSkillContent(tpl CommandTemplate, specsDir string) string {
	content := CommandSkillContent(tpl, specsDir)
	return strings.Replace(content, "description:", "disable-model-invocation: true\ndescription:", 1)
}

// RewriteAutospecSlashCommandsForSkills rewrites autospec slash-command references
// to generated skill references.
func RewriteAutospecSlashCommandsForSkills(body string) string {
	for _, commandName := range GetAutospecCommandNames() {
		body = strings.ReplaceAll(body, "/"+commandName, "$"+CommandSkillName(commandName))
	}
	return body
}
