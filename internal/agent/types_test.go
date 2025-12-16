package agent

import (
	"errors"
	"testing"
)

func TestGetAgentByID(t *testing.T) {
	tests := []struct {
		name      string
		agentID   string
		wantAgent AgentType
		wantErr   bool
	}{
		{
			name:    "valid agent claude",
			agentID: "claude",
			wantAgent: AgentType{
				ID:          "claude",
				FilePath:    "CLAUDE.md",
				DisplayName: "Claude",
			},
			wantErr: false,
		},
		{
			name:    "valid agent gemini",
			agentID: "gemini",
			wantAgent: AgentType{
				ID:          "gemini",
				FilePath:    "GEMINI.md",
				DisplayName: "Gemini",
			},
			wantErr: false,
		},
		{
			name:    "valid agent copilot",
			agentID: "copilot",
			wantAgent: AgentType{
				ID:          "copilot",
				FilePath:    ".github/copilot-instructions.md",
				DisplayName: "GitHub Copilot",
			},
			wantErr: false,
		},
		{
			name:    "valid agent cursor",
			agentID: "cursor",
			wantAgent: AgentType{
				ID:          "cursor",
				FilePath:    ".cursor/rules/context.mdc",
				DisplayName: "Cursor",
			},
			wantErr: false,
		},
		{
			name:    "valid agent windsurf",
			agentID: "windsurf",
			wantAgent: AgentType{
				ID:          "windsurf",
				FilePath:    ".windsurfrules",
				DisplayName: "Windsurf",
			},
			wantErr: false,
		},
		{
			name:      "unknown agent",
			agentID:   "unknown",
			wantAgent: AgentType{},
			wantErr:   true,
		},
		{
			name:      "empty agent id",
			agentID:   "",
			wantAgent: AgentType{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAgentByID(tt.agentID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetAgentByID(%q) expected error, got nil", tt.agentID)
				}
				if !errors.Is(err, ErrUnknownAgent) {
					t.Errorf("GetAgentByID(%q) error should wrap ErrUnknownAgent, got %v", tt.agentID, err)
				}
				return
			}

			if err != nil {
				t.Errorf("GetAgentByID(%q) unexpected error: %v", tt.agentID, err)
				return
			}

			if got.ID != tt.wantAgent.ID {
				t.Errorf("GetAgentByID(%q) ID = %q, want %q", tt.agentID, got.ID, tt.wantAgent.ID)
			}
			if got.FilePath != tt.wantAgent.FilePath {
				t.Errorf("GetAgentByID(%q) FilePath = %q, want %q", tt.agentID, got.FilePath, tt.wantAgent.FilePath)
			}
			if got.DisplayName != tt.wantAgent.DisplayName {
				t.Errorf("GetAgentByID(%q) DisplayName = %q, want %q", tt.agentID, got.DisplayName, tt.wantAgent.DisplayName)
			}
		})
	}
}

func TestGetAgentByID_AllSupportedAgents(t *testing.T) {
	// Verify all 17 agents from FR-004 are supported
	expectedAgents := []string{
		"claude", "gemini", "copilot", "cursor", "qwen", "opencode", "codex",
		"windsurf", "kilocode", "auggie", "roo", "codebuddy", "qoder",
		"amp", "shai", "q", "bob",
	}

	if len(expectedAgents) != 17 {
		t.Errorf("Expected 17 agents per FR-004, but test has %d agents listed", len(expectedAgents))
	}

	for _, id := range expectedAgents {
		agent, err := GetAgentByID(id)
		if err != nil {
			t.Errorf("GetAgentByID(%q) returned error: %v", id, err)
			continue
		}
		if agent.ID != id {
			t.Errorf("GetAgentByID(%q) returned agent with ID %q", id, agent.ID)
		}
		if agent.FilePath == "" {
			t.Errorf("GetAgentByID(%q) returned agent with empty FilePath", id)
		}
		if agent.DisplayName == "" {
			t.Errorf("GetAgentByID(%q) returned agent with empty DisplayName", id)
		}
	}
}

func TestGetAllAgentIDs(t *testing.T) {
	ids := GetAllAgentIDs()

	// Should return all 17 agents
	if len(ids) != 17 {
		t.Errorf("GetAllAgentIDs() returned %d agents, want 17", len(ids))
	}

	// All returned IDs should be valid
	for _, id := range ids {
		if _, err := GetAgentByID(id); err != nil {
			t.Errorf("GetAllAgentIDs() returned invalid ID %q: %v", id, err)
		}
	}
}

func TestSupportedAgentsCount(t *testing.T) {
	// Verify we have exactly 17 supported agents per FR-004
	if len(SupportedAgents) != 17 {
		t.Errorf("SupportedAgents has %d entries, want 17 per FR-004", len(SupportedAgents))
	}
}

func TestPlanData(t *testing.T) {
	// Test that PlanData fields can be accessed
	pd := PlanData{
		Language:    "Go 1.25.1",
		Framework:   "Cobra CLI v1.10.1",
		Database:    "None",
		ProjectType: "cli",
		Branch:      "017-update-agent-context-go",
	}

	if pd.Language != "Go 1.25.1" {
		t.Errorf("PlanData.Language = %q, want %q", pd.Language, "Go 1.25.1")
	}
	if pd.Framework != "Cobra CLI v1.10.1" {
		t.Errorf("PlanData.Framework = %q, want %q", pd.Framework, "Cobra CLI v1.10.1")
	}
	if pd.Database != "None" {
		t.Errorf("PlanData.Database = %q, want %q", pd.Database, "None")
	}
	if pd.ProjectType != "cli" {
		t.Errorf("PlanData.ProjectType = %q, want %q", pd.ProjectType, "cli")
	}
	if pd.Branch != "017-update-agent-context-go" {
		t.Errorf("PlanData.Branch = %q, want %q", pd.Branch, "017-update-agent-context-go")
	}
}

func TestUpdateResult(t *testing.T) {
	// Test UpdateResult structure
	result := UpdateResult{
		FilePath:          "/path/to/CLAUDE.md",
		Created:           true,
		TechnologiesAdded: []string{"Go 1.25.1", "Cobra CLI v1.10.1"},
		Error:             nil,
	}

	if result.FilePath != "/path/to/CLAUDE.md" {
		t.Errorf("UpdateResult.FilePath = %q, want %q", result.FilePath, "/path/to/CLAUDE.md")
	}
	if !result.Created {
		t.Error("UpdateResult.Created should be true")
	}
	if len(result.TechnologiesAdded) != 2 {
		t.Errorf("UpdateResult.TechnologiesAdded has %d items, want 2", len(result.TechnologiesAdded))
	}
	if result.Error != nil {
		t.Errorf("UpdateResult.Error should be nil, got %v", result.Error)
	}
}

func TestCommandOutput(t *testing.T) {
	// Test CommandOutput structure
	output := CommandOutput{
		Success:  true,
		SpecName: "017-update-agent-context-go",
		PlanPath: "specs/017-update-agent-context-go/plan.yaml",
		Technologies: &PlanData{
			Language:    "Go 1.25.1",
			Framework:   "Cobra CLI v1.10.1",
			Database:    "None",
			ProjectType: "cli",
			Branch:      "017-update-agent-context-go",
		},
		UpdatedFiles: []UpdateResult{
			{FilePath: "CLAUDE.md", Created: false, TechnologiesAdded: []string{"Go 1.25.1"}},
		},
		Errors: nil,
	}

	if !output.Success {
		t.Error("CommandOutput.Success should be true")
	}
	if output.SpecName != "017-update-agent-context-go" {
		t.Errorf("CommandOutput.SpecName = %q, want %q", output.SpecName, "017-update-agent-context-go")
	}
	if output.Technologies == nil {
		t.Error("CommandOutput.Technologies should not be nil")
	}
	if len(output.UpdatedFiles) != 1 {
		t.Errorf("CommandOutput.UpdatedFiles has %d items, want 1", len(output.UpdatedFiles))
	}
}
