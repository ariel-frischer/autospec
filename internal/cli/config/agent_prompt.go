package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/ariel-frischer/autospec/internal/build"
	"github.com/ariel-frischer/autospec/internal/cliagent"
	"golang.org/x/term"
)

// AgentOption represents an agent displayed in the multi-select prompt.
// Used by promptAgentSelection to show available agents with selection state.
type AgentOption struct {
	// Name is the unique identifier for the agent (e.g., "claude", "gemini").
	Name string

	// DisplayName is the human-readable name shown in prompts.
	DisplayName string

	// Recommended indicates whether this agent should be pre-selected by default.
	// Only Claude is recommended.
	Recommended bool

	// Selected indicates whether the agent is currently selected in the prompt.
	Selected bool
}

// agentDisplayNames maps agent names to their human-readable display names.
var agentDisplayNames = map[string]string{
	"claude":   "Claude Code",
	"cline":    "Cline",
	"codex":    "Codex CLI",
	"gemini":   "Gemini CLI",
	"goose":    "Goose",
	"opencode": "OpenCode",
}

// GetSupportedAgents returns supported agents as AgentOptions.
// In production builds, only production agents (claude, opencode) are returned.
// In dev builds, all registered agents are returned.
// Claude is marked as Recommended by default. Agents are returned in
// alphabetical order by name for consistent display.
func GetSupportedAgents() []AgentOption {
	var agentNames []string
	if build.IsDevBuild() {
		agentNames = cliagent.List()
	} else {
		agentNames = build.ProductionAgents()
	}

	options := make([]AgentOption, 0, len(agentNames))

	for _, name := range agentNames {
		displayName := agentDisplayNames[name]
		if displayName == "" {
			// Fallback: capitalize first letter
			displayName = strings.ToUpper(name[:1]) + name[1:]
		}

		options = append(options, AgentOption{
			Name:        name,
			DisplayName: displayName,
			Recommended: name == "claude",
			Selected:    false,
		})
	}

	// Sort alphabetically by name for consistent display
	sort.Slice(options, func(i, j int) bool {
		return options[i].Name < options[j].Name
	})

	return options
}

// GetSupportedAgentsWithDefaults returns agents with selections pre-applied.
// If defaultAgents is empty, only Claude is pre-selected (as recommended).
// Otherwise, agents in defaultAgents are pre-selected.
// Unknown agent names in defaultAgents are ignored.
func GetSupportedAgentsWithDefaults(defaultAgents []string) []AgentOption {
	options := GetSupportedAgents()

	if len(defaultAgents) == 0 {
		// No defaults configured - pre-select recommended (Claude)
		for i := range options {
			if options[i].Recommended {
				options[i].Selected = true
			}
		}
		return options
	}

	// Build a set of default agent names for O(1) lookup
	defaultSet := make(map[string]bool)
	for _, name := range defaultAgents {
		defaultSet[name] = true
	}

	// Pre-select agents that are in the default set
	for i := range options {
		if defaultSet[options[i].Name] {
			options[i].Selected = true
		}
	}

	return options
}

// promptAgentSelection displays an interactive multi-select prompt for agent selection.
// When connected to a terminal, it uses an interactive UI with arrow key navigation
// and space bar to toggle selections. Otherwise, it falls back to text-based input.
//
// Returns the list of selected agent names.
//
// Parameters:
//   - r: Reader for user input (typically os.Stdin)
//   - w: Writer for output (typically os.Stdout)
//   - agents: List of agent options to display
func promptAgentSelection(r io.Reader, w io.Writer, agents []AgentOption) []string {
	// Try interactive mode if stdin is a terminal
	if f, ok := r.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		return runInteractiveSelect(f, w, agents)
	}

	// Fallback to text-based input for non-terminals (tests, pipes)
	return runTextBasedSelect(r, w, agents)
}

// runInteractiveSelect provides arrow-key navigation and space-bar selection.
func runInteractiveSelect(f *os.File, w io.Writer, agents []AgentOption) []string {
	fd := int(f.Fd())

	// Save terminal state and switch to raw mode
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Fall back to text-based if raw mode fails
		return runTextBasedSelect(f, w, agents)
	}
	defer term.Restore(fd, oldState)

	cursor := 0
	buf := make([]byte, 3)
	menuLines := len(agents) + 3 // agents + header + instructions + blank line

	// Initial render
	renderInteractiveMenu(w, agents, cursor)

	for {
		// Read keypress
		n, err := f.Read(buf)
		if err != nil || n == 0 {
			break
		}

		switch {
		case buf[0] == 13 || buf[0] == 10: // Enter
			clearMenu(w, menuLines)
			return getSelectedAgentNames(agents)

		case buf[0] == ' ': // Space - toggle selection
			agents[cursor].Selected = !agents[cursor].Selected

		case buf[0] == 3 || buf[0] == 4: // Ctrl+C or Ctrl+D
			clearMenu(w, menuLines)
			return getSelectedAgentNames(agents)

		case n == 3 && buf[0] == 27 && buf[1] == 91: // Arrow keys
			switch buf[2] {
			case 65: // Up
				if cursor > 0 {
					cursor--
				}
			case 66: // Down
				if cursor < len(agents)-1 {
					cursor++
				}
			}

		case buf[0] == 'k' || buf[0] == 'K': // vim up
			if cursor > 0 {
				cursor--
			}

		case buf[0] == 'j' || buf[0] == 'J': // vim down
			if cursor < len(agents)-1 {
				cursor++
			}

		default:
			continue // Don't redraw for unhandled keys
		}

		// Move cursor back to start of menu and redraw
		moveUp(w, menuLines)
		renderInteractiveMenu(w, agents, cursor)
	}

	return getSelectedAgentNames(agents)
}

// renderInteractiveMenu draws the menu with cursor highlight.
// Uses \r\n for line endings because raw mode doesn't auto-convert \n.
func renderInteractiveMenu(w io.Writer, agents []AgentOption, cursor int) {
	fmt.Fprint(w, "Select AI coding agents to configure:\r\n")
	fmt.Fprint(w, "(↑/↓ move, Space select, Enter confirm)\r\n")
	fmt.Fprint(w, "\r\n")

	for i, agent := range agents {
		checkbox := "[ ]"
		if agent.Selected {
			checkbox = "[x]"
		}

		label := agent.DisplayName
		if agent.Recommended {
			label += " (Recommended)"
		}

		// Highlight current cursor position with inverse video
		if i == cursor {
			fmt.Fprintf(w, "  \x1b[7m %s %s \x1b[0m\x1b[K\r\n", checkbox, label)
		} else {
			fmt.Fprintf(w, "   %s %s\x1b[K\r\n", checkbox, label)
		}
	}
}

// moveUp moves cursor up n lines.
func moveUp(w io.Writer, n int) {
	fmt.Fprintf(w, "\x1b[%dA\r", n)
}

// clearMenu clears the menu area by overwriting with blank lines.
func clearMenu(w io.Writer, lines int) {
	moveUp(w, lines)
	for range lines {
		fmt.Fprint(w, "\x1b[K\r\n") // Clear line and move down
	}
	moveUp(w, lines)
}

// runTextBasedSelect is the fallback for non-interactive input.
func runTextBasedSelect(r io.Reader, w io.Writer, agents []AgentOption) []string {
	scanner := bufio.NewScanner(r)

	for {
		displayAgentList(w, agents)

		fmt.Fprint(w, "\nToggle selections (space-separated numbers), or press Enter when done: ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		if input == "" || strings.ToLower(input) == "done" {
			break
		}

		toggleAgentSelections(agents, input)
	}

	return getSelectedAgentNames(agents)
}

// promptDefaultAgentSelection asks which selected agent should be used for execution.
func promptDefaultAgentSelection(r io.Reader, w io.Writer, agents []AgentOption, currentDefault string) string {
	if len(agents) == 0 {
		return ""
	}

	defaultIndex := defaultAgentIndex(agents, currentDefault)
	scanner := bufio.NewScanner(r)

	for {
		displayDefaultAgentList(w, agents, defaultIndex)
		fmt.Fprintf(w, "\nChoose default agent for execution [1-%d] (Enter for %d): ", len(agents), defaultIndex+1)

		if !scanner.Scan() {
			return agents[defaultIndex].Name
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			return agents[defaultIndex].Name
		}

		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(agents) {
			fmt.Fprintln(w, "Invalid selection.")
			continue
		}

		return agents[num-1].Name
	}
}

func displayDefaultAgentList(w io.Writer, agents []AgentOption, defaultIndex int) {
	fmt.Fprintln(w, "\nChoose the default agent for autospec execution:")
	fmt.Fprintln(w)

	for i, agent := range agents {
		marker := " "
		if i == defaultIndex {
			marker = "*"
		}

		fmt.Fprintf(w, "  [%d] %s %s\n", i+1, marker, agent.DisplayName)
	}
}

func defaultAgentIndex(agents []AgentOption, currentDefault string) int {
	for i, agent := range agents {
		if agent.Name == currentDefault {
			return i
		}
	}

	return 0
}

func selectedAgentOptions(agents []AgentOption, selected []string) []AgentOption {
	selectedSet := make(map[string]bool, len(selected))
	for _, name := range selected {
		selectedSet[name] = true
	}

	options := make([]AgentOption, 0, len(selected))
	for _, agent := range agents {
		if selectedSet[agent.Name] {
			options = append(options, agent)
		}
	}

	return options
}

// displayAgentList prints the numbered list of agents with selection state.
func displayAgentList(w io.Writer, agents []AgentOption) {
	fmt.Fprintln(w, "\nSelect AI coding agents to configure:")
	fmt.Fprintln(w)

	for i, agent := range agents {
		checkbox := "[ ]"
		if agent.Selected {
			checkbox = "[x]"
		}

		label := agent.DisplayName
		if agent.Recommended {
			label += " (Recommended)"
		}

		fmt.Fprintf(w, "  [%d] %s %s\n", i+1, checkbox, label)
	}
}

// toggleAgentSelections parses the input string and toggles agent selections.
// Input format: space-separated numbers (1-indexed).
// Invalid numbers are silently ignored.
func toggleAgentSelections(agents []AgentOption, input string) {
	parts := strings.Fields(input)

	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			continue // Ignore non-numeric input
		}

		// Convert to 0-indexed and validate range
		idx := num - 1
		if idx >= 0 && idx < len(agents) {
			agents[idx].Selected = !agents[idx].Selected
		}
	}
}

// getSelectedAgentNames returns the names of all selected agents.
func getSelectedAgentNames(agents []AgentOption) []string {
	var selected []string
	for _, agent := range agents {
		if agent.Selected {
			selected = append(selected, agent.Name)
		}
	}
	return selected
}
