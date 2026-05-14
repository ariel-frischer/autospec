package commands

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// frontmatter represents the YAML frontmatter in a command template.
type frontmatter struct {
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
}

// ListTemplates returns all embedded command templates with their metadata.
func ListTemplates() ([]CommandTemplate, error) {
	names, err := GetTemplateNames()
	if err != nil {
		return nil, err
	}

	var templates []CommandTemplate
	for _, name := range names {
		content, err := GetTemplate(name)
		if err != nil {
			return nil, fmt.Errorf("failed to read template %s: %w", name, err)
		}

		desc, version, err := ParseTemplateFrontmatter(content)
		if err != nil {
			// Template without valid frontmatter - use defaults
			desc = "No description"
			version = "0.0.0"
		}

		templates = append(templates, CommandTemplate{
			Name:        name,
			Description: desc,
			Version:     version,
			Content:     content,
		})
	}

	return templates, nil
}

// GetTemplateInfo returns information about a specific template.
func GetTemplateInfo(name string) (CommandTemplate, error) {
	content, err := GetTemplate(name)
	if err != nil {
		return CommandTemplate{}, fmt.Errorf("template not found: %s", name)
	}

	desc, version, err := ParseTemplateFrontmatter(content)
	if err != nil {
		desc = "No description"
		version = "0.0.0"
	}

	return CommandTemplate{
		Name:        name,
		Description: desc,
		Version:     version,
		Content:     content,
	}, nil
}

// InstallTemplates installs all embedded templates to the target directory.
// Returns a list of results describing what was done for each template.
// For backward compatibility, use InstallTemplatesForAgent for agent-specific installation.
func InstallTemplates(targetDir string) ([]InstallResult, error) {
	return installTemplatesToDir(targetDir)
}

// InstallTemplatesForAgent installs all embedded templates for the specified agent.
// It uses GetCommandsDir to determine the correct directory based on agent name.
// The projectDir is the base project directory; the agent-specific commands dir is appended.
func InstallTemplatesForAgent(agentName, projectDir string) ([]InstallResult, error) {
	cmdDir, err := GetCommandsDir(agentName)
	if err != nil {
		return nil, fmt.Errorf("getting commands dir for agent %s: %w", agentName, err)
	}
	targetDir := filepath.Join(projectDir, cmdDir)
	return installTemplatesToDir(targetDir)
}

// installTemplatesToDir is the shared implementation for template installation.
func installTemplatesToDir(targetDir string) ([]InstallResult, error) {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating directory: %w", err)
	}

	templates, err := ListTemplates()
	if err != nil {
		return nil, err
	}

	var results []InstallResult
	for _, tpl := range templates {
		filename := tpl.Name + ".md"
		targetPath := filepath.Join(targetDir, filename)

		action := "installed"
		if _, err := os.Stat(targetPath); err == nil {
			action = "updated"
		}

		if err := os.WriteFile(targetPath, tpl.Content, 0o644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", filename, err)
		}

		results = append(results, InstallResult{
			CommandName: tpl.Name,
			Action:      action,
			Path:        targetPath,
		})
	}

	return results, nil
}

// CheckVersions compares installed templates with embedded versions.
// Returns a list of templates that need updating.
func CheckVersions(targetDir string) ([]VersionMismatch, error) {
	templates, err := ListTemplates()
	if err != nil {
		return nil, err
	}

	var mismatches []VersionMismatch
	for _, tpl := range templates {
		filename := tpl.Name + ".md"
		targetPath := filepath.Join(targetDir, filename)

		installedVersion := ""
		action := "install"

		content, err := os.ReadFile(targetPath)
		if err == nil {
			// File exists, check version
			_, version, parseErr := ParseTemplateFrontmatter(content)
			if parseErr == nil {
				installedVersion = version
				if version != tpl.Version {
					action = "update"
				} else {
					// Versions match, no mismatch
					continue
				}
			} else {
				// Can't parse version, needs update
				action = "update"
			}
		}

		mismatches = append(mismatches, VersionMismatch{
			CommandName:      tpl.Name,
			InstalledVersion: installedVersion,
			EmbeddedVersion:  tpl.Version,
			Action:           action,
		})
	}

	return mismatches, nil
}

// GetInstalledCommands returns information about all installed autospec commands.
func GetInstalledCommands(targetDir string) ([]CommandInfo, error) {
	templates, err := ListTemplates()
	if err != nil {
		return nil, err
	}

	var commands []CommandInfo
	for _, tpl := range templates {
		filename := tpl.Name + ".md"
		targetPath := filepath.Join(targetDir, filename)

		info := CommandInfo{
			Name:            tpl.Name,
			Description:     tpl.Description,
			EmbeddedVersion: tpl.Version,
			InstallPath:     targetPath,
		}

		content, err := os.ReadFile(targetPath)
		if err == nil {
			// File exists
			desc, version, parseErr := ParseTemplateFrontmatter(content)
			if parseErr == nil {
				info.Version = version
				info.Description = desc
				info.IsOutdated = version != tpl.Version
			}
		}

		commands = append(commands, info)
	}

	return commands, nil
}

// ParseTemplateFrontmatter extracts description and version from YAML frontmatter.
func ParseTemplateFrontmatter(content []byte) (description, version string, err error) {
	// Check for frontmatter markers
	if !bytes.HasPrefix(content, []byte("---")) {
		return "", "", fmt.Errorf("no frontmatter found")
	}

	// Find end of frontmatter
	rest := content[3:]
	endIdx := bytes.Index(rest, []byte("\n---"))
	if endIdx == -1 {
		return "", "", fmt.Errorf("frontmatter not closed")
	}

	// Extract frontmatter content (skip leading newline)
	fmContent := rest[:endIdx]
	if len(fmContent) > 0 && fmContent[0] == '\n' {
		fmContent = fmContent[1:]
	}

	var fm frontmatter
	if err := yaml.Unmarshal(fmContent, &fm); err != nil {
		return "", "", fmt.Errorf("invalid frontmatter: %w", err)
	}

	return fm.Description, fm.Version, nil
}

// StripFrontmatter removes YAML frontmatter from template content.
// Frontmatter is delimited by "---" at the start and "\n---" as the closing marker.
// Returns the content after the closing "---" marker, trimmed of leading newlines.
// If no valid frontmatter is found, returns the original content unchanged.
func StripFrontmatter(content []byte) []byte {
	// Check for frontmatter markers
	if !bytes.HasPrefix(content, []byte("---")) {
		return content
	}

	// Find end of frontmatter
	rest := content[3:]
	endIdx := bytes.Index(rest, []byte("\n---"))
	if endIdx == -1 {
		return content
	}

	// Skip past the closing "---" and any trailing newlines
	afterFrontmatter := rest[endIdx+4:] // +4 for "\n---"

	// Trim leading newlines from the content after frontmatter
	return bytes.TrimLeft(afterFrontmatter, "\n")
}

// GetDefaultCommandsDir returns the default path for Claude commands.
func GetDefaultCommandsDir() string {
	return filepath.Join(".claude", "commands")
}

// GetCommandsDir returns the command directory path for the specified agent.
// Known agents: "claude" -> ".claude/commands"
// Unknown agents return an error.
func GetCommandsDir(agentName string) (string, error) {
	switch agentName {
	case "claude":
		return filepath.Join(".claude", "commands"), nil
	default:
		return "", fmt.Errorf("unknown agent: %s", agentName)
	}
}

// CommandExists checks if a command file exists in the target directory.
func CommandExists(targetDir, commandName string) bool {
	filename := commandName + ".md"
	targetPath := filepath.Join(targetDir, filename)
	_, err := os.Stat(targetPath)
	return err == nil
}

// GetAutospecCommandNames returns the names of all autospec commands (embedded).
func GetAutospecCommandNames() []string {
	names, err := GetTemplateNames()
	if err != nil {
		return nil
	}

	var autospecNames []string
	for _, name := range names {
		if strings.HasPrefix(name, "autospec.") {
			autospecNames = append(autospecNames, name)
		}
	}
	return autospecNames
}
