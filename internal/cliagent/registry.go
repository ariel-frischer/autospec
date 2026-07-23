package cliagent

import (
	"fmt"
	"sort"
	"sync"
)

// Registry is a thread-safe container for registered agents.
// It provides methods for registration, retrieval, and discovery.
type Registry struct {
	mu     sync.RWMutex
	agents map[string]Agent
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[string]Agent),
	}
}

// Register adds an agent to the registry.
// If an agent with the same name already exists, it is overwritten (last-write-wins).
func (r *Registry) Register(agent Agent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[agent.Name()] = agent
}

// Get retrieves an agent by name.
// Returns nil if the agent is not found.
func (r *Registry) Get(name string) Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.agents[name]
}

// List returns all registered agent names in alphabetical order.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.agents))
	for name := range r.agents {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Available returns agents that pass validation, in alphabetical order.
// Useful for discovering which agents are usable on this system.
func (r *Registry) Available() []Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var available []Agent
	for _, agent := range r.agents {
		if err := agent.Validate(); err == nil {
			available = append(available, agent)
		}
	}
	sort.Slice(available, func(i, j int) bool {
		return available[i].Name() < available[j].Name()
	})
	return available
}

// Automatable returns agents that support headless execution.
// Filters to only those that pass validation.
func (r *Registry) Automatable() []Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Agent
	for _, agent := range r.agents {
		if agent.Capabilities().Automatable && agent.Validate() == nil {
			result = append(result, agent)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name() < result[j].Name()
	})
	return result
}

// MustGet retrieves an agent by name or panics if not found.
// Use only when the agent is guaranteed to be registered.
func (r *Registry) MustGet(name string) Agent {
	agent := r.Get(name)
	if agent == nil {
		panic(fmt.Sprintf("cliagent: agent %q not registered", name))
	}
	return agent
}

// Default is the global registry instance.
// Built-in agents are registered here during package init.
var Default = NewRegistry()

// Register adds an agent to the default registry.
func Register(agent Agent) {
	Default.Register(agent)
}

// Get retrieves an agent from the default registry by name.
func Get(name string) Agent {
	return Default.Get(name)
}

// List returns all agent names from the default registry.
func List() []string {
	return Default.List()
}

// Available returns available agents from the default registry.
func Available() []Agent {
	return Default.Available()
}

// Automatable returns automatable agents from the default registry.
func Automatable() []Agent {
	return Default.Automatable()
}

// AgentStatus represents the diagnostic status of an agent.
type AgentStatus struct {
	Name          string
	Installed     bool
	Version       string
	TestedVersion string
	Valid         bool
	Error         string
}

// Doctor returns diagnostic status for all registered agents.
// Returns statuses in alphabetical order by agent name.
func (r *Registry) Doctor() []AgentStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	statuses := make([]AgentStatus, 0, len(r.agents))
	for _, agent := range r.agents {
		status := AgentStatus{
			Name:          agent.Name(),
			TestedVersion: TestedVersion(agent.Name()),
		}

		// Check validation (installed and env vars)
		err := agent.Validate()
		if err == nil {
			status.Installed = true
			status.Valid = true
		} else {
			status.Error = err.Error()
			// Agent may be installed but invalid (e.g., missing env vars)
			// We can't easily distinguish, so assume not installed if validation fails
			status.Installed = false
			status.Valid = false
		}

		// Try to get version if installed
		if status.Installed {
			version, verErr := agent.Version()
			if verErr == nil {
				status.Version = version
			}
		}

		statuses = append(statuses, status)
	}

	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Name < statuses[j].Name
	})
	return statuses
}

// Doctor returns diagnostic status for all agents in the default registry.
func Doctor() []AgentStatus {
	return Default.Doctor()
}

// TestedVersion returns the latest agent CLI version verified by autospec smoke tests.
func TestedVersion(agentName string) string {
	switch agentName {
	case "claude":
		return "2.1.139"
	case "codex":
		return "0.145.0-alpha.23"
	case "opencode":
		return "1.14.46"
	default:
		return ""
	}
}
