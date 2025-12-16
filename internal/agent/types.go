// Package agent provides functionality for managing AI agent context files.
package agent

import (
	"fmt"
)

// AgentType represents a supported AI agent with its file path and display name.
type AgentType struct {
	ID          string // Short identifier (claude, gemini, copilot, etc.)
	FilePath    string // Relative path to context file from repo root
	DisplayName string // Human-readable name for logging
}

// PlanData represents parsed technology information from plan.yaml technical_context.
type PlanData struct {
	Language    string // Programming language from technical_context.language
	Framework   string // Framework from primary_dependencies first entry
	Database    string // Storage technology from technical_context.storage
	ProjectType string // Project type from technical_context.project_type
	Branch      string // Git branch from plan.branch
}

// UpdateResult represents the result of updating a single agent context file.
type UpdateResult struct {
	FilePath          string   // Absolute path to the updated file
	Created           bool     // True if file was created, false if updated
	TechnologiesAdded []string // List of technology entries added
	Error             error    // Error if update failed
}

// CommandOutput represents structured output for --json flag.
type CommandOutput struct {
	Success      bool           `json:"success"`       // Overall success status
	SpecName     string         `json:"specName"`      // Detected spec name
	PlanPath     string         `json:"planPath"`      // Path to plan.yaml used
	Technologies *PlanData      `json:"technologies"`  // Extracted technology information
	UpdatedFiles []UpdateResult `json:"updatedFiles"`  // List of file update results
	Errors       []string       `json:"errors"`        // List of error messages if any
}

// SupportedAgents is a map of all supported AI agent types.
// Keys are the identifiers used for --agent flag.
var SupportedAgents = map[string]AgentType{
	"claude":   {ID: "claude", FilePath: "CLAUDE.md", DisplayName: "Claude"},
	"gemini":   {ID: "gemini", FilePath: "GEMINI.md", DisplayName: "Gemini"},
	"copilot":  {ID: "copilot", FilePath: ".github/copilot-instructions.md", DisplayName: "GitHub Copilot"},
	"cursor":   {ID: "cursor", FilePath: ".cursor/rules/context.mdc", DisplayName: "Cursor"},
	"qwen":     {ID: "qwen", FilePath: ".qwen/context.md", DisplayName: "Qwen"},
	"opencode": {ID: "opencode", FilePath: ".opencode/context.md", DisplayName: "OpenCode"},
	"codex":    {ID: "codex", FilePath: "AGENTS.md", DisplayName: "Codex"},
	"windsurf": {ID: "windsurf", FilePath: ".windsurfrules", DisplayName: "Windsurf"},
	"kilocode": {ID: "kilocode", FilePath: ".kilocode/rules", DisplayName: "Kilocode"},
	"auggie":   {ID: "auggie", FilePath: ".auggie/context.md", DisplayName: "Auggie"},
	"roo":      {ID: "roo", FilePath: ".roo/rules/context.md", DisplayName: "Roo"},
	"codebuddy": {ID: "codebuddy", FilePath: ".codebuddy/context.md", DisplayName: "CodeBuddy"},
	"qoder":    {ID: "qoder", FilePath: ".qoder/context.md", DisplayName: "Qoder"},
	"amp":      {ID: "amp", FilePath: "AMP.md", DisplayName: "Amp"},
	"shai":     {ID: "shai", FilePath: ".shai/context.md", DisplayName: "Shai"},
	"q":        {ID: "q", FilePath: ".q/context.md", DisplayName: "Q"},
	"bob":      {ID: "bob", FilePath: ".bob/context.md", DisplayName: "Bob"},
}

// ErrUnknownAgent is returned when an invalid agent identifier is provided.
var ErrUnknownAgent = fmt.Errorf("unknown agent type")

// GetAgentByID returns the AgentType for the given identifier.
// Returns ErrUnknownAgent if the identifier is not recognized.
func GetAgentByID(id string) (AgentType, error) {
	if agent, ok := SupportedAgents[id]; ok {
		return agent, nil
	}

	// Build suggestion of valid agents
	validIDs := make([]string, 0, len(SupportedAgents))
	for k := range SupportedAgents {
		validIDs = append(validIDs, k)
	}

	return AgentType{}, fmt.Errorf("%w: %q. Valid agents: %v", ErrUnknownAgent, id, validIDs)
}

// GetAllAgentIDs returns a slice of all supported agent identifiers.
func GetAllAgentIDs() []string {
	ids := make([]string, 0, len(SupportedAgents))
	for k := range SupportedAgents {
		ids = append(ids, k)
	}
	return ids
}
