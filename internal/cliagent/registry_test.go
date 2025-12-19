package cliagent

import (
	"context"
	"errors"
	"os/exec"
	"sync"
	"testing"
)

// mockAgent implements Agent for testing.
type mockAgent struct {
	name        string
	version     string
	versionErr  error
	validateErr error
	caps        Caps
}

func (m *mockAgent) Name() string                { return m.name }
func (m *mockAgent) Version() (string, error)    { return m.version, m.versionErr }
func (m *mockAgent) Validate() error             { return m.validateErr }
func (m *mockAgent) Capabilities() Caps          { return m.caps }

func (m *mockAgent) BuildCommand(_ string, _ ExecOptions) (*exec.Cmd, error) {
	return exec.Command("echo", "mock"), nil
}

func (m *mockAgent) Execute(_ context.Context, _ string, _ ExecOptions) (*Result, error) {
	return &Result{ExitCode: 0}, nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agents    []*mockAgent
		getName   string
		wantFound bool
	}{
		"get registered agent": {
			agents:    []*mockAgent{{name: "test"}},
			getName:   "test",
			wantFound: true,
		},
		"get unregistered agent": {
			agents:    []*mockAgent{{name: "other"}},
			getName:   "notfound",
			wantFound: false,
		},
		"overwrite existing agent": {
			agents: []*mockAgent{
				{name: "dup", version: "1.0"},
				{name: "dup", version: "2.0"},
			},
			getName:   "dup",
			wantFound: true,
		},
		"empty registry": {
			agents:    nil,
			getName:   "anything",
			wantFound: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			reg := NewRegistry()
			for _, a := range tt.agents {
				reg.Register(a)
			}
			got := reg.Get(tt.getName)
			if (got != nil) != tt.wantFound {
				t.Errorf("Get(%q) found=%v, want found=%v", tt.getName, got != nil, tt.wantFound)
			}
		})
	}
}

func TestRegistry_List(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agents []string
		want   []string
	}{
		"empty registry": {
			agents: nil,
			want:   []string{},
		},
		"single agent": {
			agents: []string{"alpha"},
			want:   []string{"alpha"},
		},
		"multiple agents sorted": {
			agents: []string{"charlie", "alpha", "bravo"},
			want:   []string{"alpha", "bravo", "charlie"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			reg := NewRegistry()
			for _, a := range tt.agents {
				reg.Register(&mockAgent{name: a})
			}
			got := reg.List()
			if len(got) != len(tt.want) {
				t.Fatalf("List() len=%d, want len=%d", len(got), len(tt.want))
			}
			for i, name := range got {
				if name != tt.want[i] {
					t.Errorf("List()[%d]=%q, want %q", i, name, tt.want[i])
				}
			}
		})
	}
}

func TestRegistry_Available(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agents []*mockAgent
		want   []string
	}{
		"all valid": {
			agents: []*mockAgent{
				{name: "alpha", validateErr: nil},
				{name: "bravo", validateErr: nil},
			},
			want: []string{"alpha", "bravo"},
		},
		"some invalid": {
			agents: []*mockAgent{
				{name: "valid", validateErr: nil},
				{name: "invalid", validateErr: errors.New("not installed")},
			},
			want: []string{"valid"},
		},
		"all invalid": {
			agents: []*mockAgent{
				{name: "a", validateErr: errors.New("err")},
				{name: "b", validateErr: errors.New("err")},
			},
			want: []string{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			reg := NewRegistry()
			for _, a := range tt.agents {
				reg.Register(a)
			}
			got := reg.Available()
			if len(got) != len(tt.want) {
				t.Fatalf("Available() len=%d, want len=%d", len(got), len(tt.want))
			}
			for i, agent := range got {
				if agent.Name() != tt.want[i] {
					t.Errorf("Available()[%d].Name()=%q, want %q", i, agent.Name(), tt.want[i])
				}
			}
		})
	}
}

func TestRegistry_Automatable(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agents []*mockAgent
		want   []string
	}{
		"automatable and valid": {
			agents: []*mockAgent{
				{name: "auto", caps: Caps{Automatable: true}},
				{name: "notauto", caps: Caps{Automatable: false}},
			},
			want: []string{"auto"},
		},
		"automatable but invalid": {
			agents: []*mockAgent{
				{name: "auto", caps: Caps{Automatable: true}, validateErr: errors.New("err")},
			},
			want: []string{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			reg := NewRegistry()
			for _, a := range tt.agents {
				reg.Register(a)
			}
			got := reg.Automatable()
			if len(got) != len(tt.want) {
				t.Fatalf("Automatable() len=%d, want len=%d", len(got), len(tt.want))
			}
			for i, agent := range got {
				if agent.Name() != tt.want[i] {
					t.Errorf("Automatable()[%d].Name()=%q, want %q", i, agent.Name(), tt.want[i])
				}
			}
		})
	}
}

func TestRegistry_MustGet_Panics(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGet did not panic for unregistered agent")
		}
	}()
	reg.MustGet("nonexistent")
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3)

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			reg.Register(&mockAgent{name: "agent"})
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = reg.Get("agent")
		}()
	}

	// Concurrent list
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = reg.List()
		}()
	}

	wg.Wait()
}
