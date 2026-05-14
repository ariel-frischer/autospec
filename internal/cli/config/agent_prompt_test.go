package config

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSupportedAgents(t *testing.T) {
	t.Parallel()

	agents := GetSupportedAgents()

	// Verify we get all 6 registered agents
	require.Len(t, agents, 6, "expected 6 registered agents")

	// Build a map for easier lookup
	agentMap := make(map[string]AgentOption)
	for _, a := range agents {
		agentMap[a.Name] = a
	}

	// Verify all expected agents are present
	expectedAgents := []string{"claude", "cline", "codex", "gemini", "goose", "opencode"}
	for _, name := range expectedAgents {
		_, ok := agentMap[name]
		assert.True(t, ok, "expected agent %q to be present", name)
	}
}

func TestGetSupportedAgents_ClaudeIsRecommended(t *testing.T) {
	t.Parallel()

	agents := GetSupportedAgents()

	var claudeFound bool
	var recommendedCount int

	for _, agent := range agents {
		if agent.Recommended {
			recommendedCount++
		}
		if agent.Name == "claude" {
			claudeFound = true
			assert.True(t, agent.Recommended, "claude should be marked as Recommended")
		}
	}

	assert.True(t, claudeFound, "claude should be in the agent list")
	assert.Equal(t, 1, recommendedCount, "only claude should be Recommended")
}

func TestGetSupportedAgents_DisplayNames(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agentName       string
		wantDisplayName string
	}{
		"claude has display name": {
			agentName:       "claude",
			wantDisplayName: "Claude Code",
		},
		"cline has display name": {
			agentName:       "cline",
			wantDisplayName: "Cline",
		},
		"codex has display name": {
			agentName:       "codex",
			wantDisplayName: "Codex CLI",
		},
		"gemini has display name": {
			agentName:       "gemini",
			wantDisplayName: "Gemini CLI",
		},
		"goose has display name": {
			agentName:       "goose",
			wantDisplayName: "Goose",
		},
		"opencode has display name": {
			agentName:       "opencode",
			wantDisplayName: "OpenCode",
		},
	}

	agents := GetSupportedAgents()
	agentMap := make(map[string]AgentOption)
	for _, a := range agents {
		agentMap[a.Name] = a
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			agent, ok := agentMap[tt.agentName]
			require.True(t, ok, "agent %q not found", tt.agentName)
			assert.Equal(t, tt.wantDisplayName, agent.DisplayName)
		})
	}
}

func TestGetSupportedAgents_AlphabeticalOrder(t *testing.T) {
	t.Parallel()

	agents := GetSupportedAgents()

	// Verify agents are in alphabetical order
	for i := 1; i < len(agents); i++ {
		assert.True(t, agents[i-1].Name < agents[i].Name,
			"agents should be in alphabetical order: %s should come before %s",
			agents[i-1].Name, agents[i].Name)
	}
}

func TestGetSupportedAgentsWithDefaults(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		defaultAgents []string
		wantSelected  []string
	}{
		"empty defaults selects claude": {
			defaultAgents: nil,
			wantSelected:  []string{"claude"},
		},
		"empty slice defaults selects claude": {
			defaultAgents: []string{},
			wantSelected:  []string{"claude"},
		},
		"single agent default": {
			defaultAgents: []string{"gemini"},
			wantSelected:  []string{"gemini"},
		},
		"multiple agent defaults": {
			defaultAgents: []string{"claude", "cline"},
			wantSelected:  []string{"claude", "cline"},
		},
		"unknown agents are ignored": {
			defaultAgents: []string{"unknown", "claude", "nonexistent"},
			wantSelected:  []string{"claude"},
		},
		"all agents selected": {
			defaultAgents: []string{"claude", "cline", "codex", "gemini", "goose", "opencode"},
			wantSelected:  []string{"claude", "cline", "codex", "gemini", "goose", "opencode"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			agents := GetSupportedAgentsWithDefaults(tt.defaultAgents)

			// Collect selected agent names
			var selected []string
			for _, a := range agents {
				if a.Selected {
					selected = append(selected, a.Name)
				}
			}

			// Sort both slices for comparison (selected may not be in order)
			assert.ElementsMatch(t, tt.wantSelected, selected)
		})
	}
}

func TestToggleAgentSelections(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		initialSelected []bool
		input           string
		wantSelected    []bool
	}{
		"toggle single agent": {
			initialSelected: []bool{false, false, false},
			input:           "1",
			wantSelected:    []bool{true, false, false},
		},
		"toggle multiple agents": {
			initialSelected: []bool{false, false, false},
			input:           "1 3",
			wantSelected:    []bool{true, false, true},
		},
		"toggle off selected agent": {
			initialSelected: []bool{true, false, false},
			input:           "1",
			wantSelected:    []bool{false, false, false},
		},
		"toggle mixed": {
			initialSelected: []bool{true, false, true},
			input:           "1 2 3",
			wantSelected:    []bool{false, true, false},
		},
		"invalid number ignored": {
			initialSelected: []bool{false, false, false},
			input:           "1 99 2",
			wantSelected:    []bool{true, true, false},
		},
		"zero ignored": {
			initialSelected: []bool{false, false, false},
			input:           "0 1",
			wantSelected:    []bool{true, false, false},
		},
		"negative ignored": {
			initialSelected: []bool{false, false, false},
			input:           "-1 1",
			wantSelected:    []bool{true, false, false},
		},
		"non-numeric ignored": {
			initialSelected: []bool{false, false, false},
			input:           "abc 1 def",
			wantSelected:    []bool{true, false, false},
		},
		"empty input no change": {
			initialSelected: []bool{true, false, true},
			input:           "",
			wantSelected:    []bool{true, false, true},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			agents := make([]AgentOption, len(tt.initialSelected))
			for i, selected := range tt.initialSelected {
				agents[i] = AgentOption{
					Name:     string(rune('a' + i)), // "a", "b", "c"
					Selected: selected,
				}
			}

			toggleAgentSelections(agents, tt.input)

			for i, want := range tt.wantSelected {
				assert.Equal(t, want, agents[i].Selected,
					"agent %d: expected Selected=%v, got %v", i+1, want, agents[i].Selected)
			}
		})
	}
}

func TestGetSelectedAgentNames(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agents       []AgentOption
		wantSelected []string
	}{
		"no agents selected": {
			agents: []AgentOption{
				{Name: "a", Selected: false},
				{Name: "b", Selected: false},
			},
			wantSelected: nil,
		},
		"all agents selected": {
			agents: []AgentOption{
				{Name: "a", Selected: true},
				{Name: "b", Selected: true},
			},
			wantSelected: []string{"a", "b"},
		},
		"some agents selected": {
			agents: []AgentOption{
				{Name: "a", Selected: true},
				{Name: "b", Selected: false},
				{Name: "c", Selected: true},
			},
			wantSelected: []string{"a", "c"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := getSelectedAgentNames(tt.agents)
			assert.Equal(t, tt.wantSelected, got)
		})
	}
}

func TestDisplayAgentList(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agents       []AgentOption
		wantContains []string
	}{
		"shows selected checkbox": {
			agents: []AgentOption{
				{Name: "test", DisplayName: "Test Agent", Selected: true},
			},
			wantContains: []string{"[x]", "Test Agent"},
		},
		"shows unselected checkbox": {
			agents: []AgentOption{
				{Name: "test", DisplayName: "Test Agent", Selected: false},
			},
			wantContains: []string{"[ ]", "Test Agent"},
		},
		"shows recommended label": {
			agents: []AgentOption{
				{Name: "claude", DisplayName: "Claude Code", Recommended: true, Selected: true},
			},
			wantContains: []string{"(Recommended)", "Claude Code"},
		},
		"shows numbered list": {
			agents: []AgentOption{
				{Name: "a", DisplayName: "Agent A"},
				{Name: "b", DisplayName: "Agent B"},
			},
			wantContains: []string{"[1]", "[2]", "Agent A", "Agent B"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			displayAgentList(&buf, tt.agents)
			output := buf.String()

			for _, want := range tt.wantContains {
				assert.Contains(t, output, want)
			}
		})
	}
}

func TestPromptAgentSelection(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input        string
		wantSelected []string
	}{
		"empty input confirms default": {
			input:        "\n",
			wantSelected: []string{"claude"},
		},
		"done confirms selection": {
			input:        "done\n",
			wantSelected: []string{"claude"},
		},
		"toggle and confirm": {
			input:        "2\n\n", // Toggle cline (index 2), then confirm
			wantSelected: []string{"claude", "cline"},
		},
		"toggle off claude and confirm": {
			input:        "1\n\n", // Toggle claude off (index 1)
			wantSelected: nil,
		},
		"select multiple then confirm": {
			input:        "3 4\n\n", // Toggle codex and gemini
			wantSelected: []string{"claude", "codex", "gemini"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Get agents with Claude pre-selected
			agents := GetSupportedAgentsWithDefaults(nil)

			input := strings.NewReader(tt.input)
			var output bytes.Buffer

			selected := promptAgentSelection(input, &output, agents)

			assert.ElementsMatch(t, tt.wantSelected, selected)
		})
	}
}

func TestPromptAgentSelection_EOF(t *testing.T) {
	t.Parallel()

	// Simulate EOF (empty reader)
	agents := GetSupportedAgentsWithDefaults(nil)
	input := strings.NewReader("")
	var output bytes.Buffer

	selected := promptAgentSelection(input, &output, agents)

	// On EOF, should return current selections (claude is pre-selected)
	assert.Equal(t, []string{"claude"}, selected)
}

func TestPromptDefaultAgentSelection(t *testing.T) {
	t.Parallel()

	agents := []AgentOption{
		{Name: "claude", DisplayName: "Claude Code"},
		{Name: "codex", DisplayName: "Codex CLI"},
		{Name: "opencode", DisplayName: "OpenCode"},
	}

	tests := map[string]struct {
		input       string
		current     string
		wantDefault string
	}{
		"enter confirms current preset": {
			input:       "\n",
			current:     "codex",
			wantDefault: "codex",
		},
		"number selects default": {
			input:       "3\n",
			current:     "",
			wantDefault: "opencode",
		},
		"unknown current falls back to first selected": {
			input:       "\n",
			current:     "gemini",
			wantDefault: "claude",
		},
		"invalid input retries until valid": {
			input:       "99\n2\n",
			current:     "",
			wantDefault: "codex",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			input := strings.NewReader(tt.input)
			var output bytes.Buffer

			got := promptDefaultAgentSelection(input, &output, agents, tt.current)

			assert.Equal(t, tt.wantDefault, got)
		})
	}
}

func TestPromptDefaultAgentSelection_PrefersClaudeWhenCurrentDefaultUnset(t *testing.T) {
	t.Parallel()

	agents := []AgentOption{
		{Name: "codex", DisplayName: "Codex CLI"},
		{Name: "claude", DisplayName: "Claude Code"},
		{Name: "opencode", DisplayName: "OpenCode"},
	}
	var output bytes.Buffer

	got := promptDefaultAgentSelection(strings.NewReader("\n"), &output, agents, "")

	assert.Equal(t, "claude", got)
	assert.Contains(t, output.String(), "[2] * Claude Code")
}

func TestDefaultAgentIndex(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agents         []AgentOption
		currentDefault string
		want           int
	}{
		"current default wins": {
			agents: []AgentOption{
				{Name: "claude"},
				{Name: "codex"},
				{Name: "opencode"},
			},
			currentDefault: "opencode",
			want:           2,
		},
		"unset current default prefers claude": {
			agents: []AgentOption{
				{Name: "codex"},
				{Name: "claude"},
				{Name: "opencode"},
			},
			want: 1,
		},
		"missing claude falls back to first selected": {
			agents: []AgentOption{
				{Name: "codex"},
				{Name: "opencode"},
			},
			currentDefault: "gemini",
			want:           0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := defaultAgentIndex(tt.agents, tt.currentDefault)

			assert.Equal(t, tt.want, got)
		})
	}
}
