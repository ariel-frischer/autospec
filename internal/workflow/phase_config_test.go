package workflow

import (
	"testing"
)

func TestNewPhaseConfig(t *testing.T) {
	pc := NewPhaseConfig()
	if pc == nil {
		t.Fatal("NewPhaseConfig returned nil")
	}
	// Check core phases are disabled
	if pc.Specify || pc.Plan || pc.Tasks || pc.Implement {
		t.Error("NewPhaseConfig should have all core phases disabled")
	}
	// Check optional phases are disabled
	if pc.Constitution || pc.Clarify || pc.Checklist || pc.Analyze {
		t.Error("NewPhaseConfig should have all optional phases disabled")
	}
}

func TestNewPhaseConfigAll(t *testing.T) {
	pc := NewPhaseConfigAll()
	if pc == nil {
		t.Fatal("NewPhaseConfigAll returned nil")
	}
	if !pc.Specify || !pc.Plan || !pc.Tasks || !pc.Implement {
		t.Error("NewPhaseConfigAll should have all phases enabled")
	}
}

func TestHasAnyPhase(t *testing.T) {
	tests := []struct {
		name     string
		config   PhaseConfig
		expected bool
	}{
		{
			name:     "no phases selected",
			config:   PhaseConfig{},
			expected: false,
		},
		{
			name:     "only specify selected",
			config:   PhaseConfig{Specify: true},
			expected: true,
		},
		{
			name:     "only plan selected",
			config:   PhaseConfig{Plan: true},
			expected: true,
		},
		{
			name:     "only tasks selected",
			config:   PhaseConfig{Tasks: true},
			expected: true,
		},
		{
			name:     "only implement selected",
			config:   PhaseConfig{Implement: true},
			expected: true,
		},
		{
			name:     "all phases selected",
			config:   PhaseConfig{Specify: true, Plan: true, Tasks: true, Implement: true},
			expected: true,
		},
		{
			name:     "plan and implement selected",
			config:   PhaseConfig{Plan: true, Implement: true},
			expected: true,
		},
		// Optional phase tests
		{
			name:     "only constitution selected",
			config:   PhaseConfig{Constitution: true},
			expected: true,
		},
		{
			name:     "only clarify selected",
			config:   PhaseConfig{Clarify: true},
			expected: true,
		},
		{
			name:     "only checklist selected",
			config:   PhaseConfig{Checklist: true},
			expected: true,
		},
		{
			name:     "only analyze selected",
			config:   PhaseConfig{Analyze: true},
			expected: true,
		},
		{
			name:     "mixed core and optional phases",
			config:   PhaseConfig{Specify: true, Clarify: true, Checklist: true},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.config.HasAnyPhase(); got != tt.expected {
				t.Errorf("HasAnyPhase() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetSelectedPhases(t *testing.T) {
	tests := []struct {
		name     string
		config   PhaseConfig
		expected []Phase
	}{
		{
			name:     "no phases selected",
			config:   PhaseConfig{},
			expected: []Phase{},
		},
		{
			name:     "only specify",
			config:   PhaseConfig{Specify: true},
			expected: []Phase{PhaseSpecify},
		},
		{
			name:     "plan and implement",
			config:   PhaseConfig{Plan: true, Implement: true},
			expected: []Phase{PhasePlan, PhaseImplement},
		},
		{
			name:     "all core phases",
			config:   PhaseConfig{Specify: true, Plan: true, Tasks: true, Implement: true},
			expected: []Phase{PhaseSpecify, PhasePlan, PhaseTasks, PhaseImplement},
		},
		{
			name:     "tasks and implement (skipping earlier phases)",
			config:   PhaseConfig{Tasks: true, Implement: true},
			expected: []Phase{PhaseTasks, PhaseImplement},
		},
		// Optional phase tests
		{
			name:     "only constitution",
			config:   PhaseConfig{Constitution: true},
			expected: []Phase{PhaseConstitution},
		},
		{
			name:     "constitution and specify in canonical order",
			config:   PhaseConfig{Specify: true, Constitution: true},
			expected: []Phase{PhaseConstitution, PhaseSpecify},
		},
		{
			name:     "specify with clarify",
			config:   PhaseConfig{Specify: true, Clarify: true},
			expected: []Phase{PhaseSpecify, PhaseClarify},
		},
		{
			name:     "tasks with checklist and analyze",
			config:   PhaseConfig{Tasks: true, Checklist: true, Analyze: true},
			expected: []Phase{PhaseTasks, PhaseChecklist, PhaseAnalyze},
		},
		{
			name:     "full workflow with optional phases",
			config:   PhaseConfig{Constitution: true, Specify: true, Clarify: true, Plan: true, Tasks: true, Checklist: true, Analyze: true, Implement: true},
			expected: []Phase{PhaseConstitution, PhaseSpecify, PhaseClarify, PhasePlan, PhaseTasks, PhaseChecklist, PhaseAnalyze, PhaseImplement},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.config.GetSelectedPhases()
			if len(got) != len(tt.expected) {
				t.Errorf("GetSelectedPhases() returned %d phases, want %d", len(got), len(tt.expected))
				return
			}
			for i, phase := range got {
				if phase != tt.expected[i] {
					t.Errorf("GetSelectedPhases()[%d] = %v, want %v", i, phase, tt.expected[i])
				}
			}
		})
	}
}

func TestGetCanonicalOrder(t *testing.T) {
	// The canonical order must always be:
	// constitution -> specify -> clarify -> plan -> tasks -> checklist -> analyze -> implement
	// regardless of how the struct fields are set or in what order

	tests := []struct {
		name     string
		config   PhaseConfig
		expected []Phase
	}{
		{
			name:     "core phases set in reverse order",
			config:   PhaseConfig{Implement: true, Tasks: true, Plan: true, Specify: true},
			expected: []Phase{PhaseSpecify, PhasePlan, PhaseTasks, PhaseImplement},
		},
		{
			name:     "only middle phases",
			config:   PhaseConfig{Plan: true, Tasks: true},
			expected: []Phase{PhasePlan, PhaseTasks},
		},
		{
			name:     "only first and last",
			config:   PhaseConfig{Specify: true, Implement: true},
			expected: []Phase{PhaseSpecify, PhaseImplement},
		},
		// Optional phases canonical order tests - from spec US-003
		{
			name:     "-ns executes as: constitution, specify",
			config:   PhaseConfig{Constitution: true, Specify: true},
			expected: []Phase{PhaseConstitution, PhaseSpecify},
		},
		{
			name:     "-srp executes as: specify, clarify, plan",
			config:   PhaseConfig{Specify: true, Clarify: true, Plan: true},
			expected: []Phase{PhaseSpecify, PhaseClarify, PhasePlan},
		},
		{
			name:     "-tczi executes as: tasks, checklist, analyze, implement",
			config:   PhaseConfig{Tasks: true, Checklist: true, Analyze: true, Implement: true},
			expected: []Phase{PhaseTasks, PhaseChecklist, PhaseAnalyze, PhaseImplement},
		},
		{
			name:     "-icts executes as: specify, tasks, checklist, implement (regardless of flag order)",
			config:   PhaseConfig{Implement: true, Checklist: true, Tasks: true, Specify: true},
			expected: []Phase{PhaseSpecify, PhaseTasks, PhaseChecklist, PhaseImplement},
		},
		{
			name:     "all phases in canonical order",
			config:   PhaseConfig{Constitution: true, Specify: true, Clarify: true, Plan: true, Tasks: true, Checklist: true, Analyze: true, Implement: true},
			expected: []Phase{PhaseConstitution, PhaseSpecify, PhaseClarify, PhasePlan, PhaseTasks, PhaseChecklist, PhaseAnalyze, PhaseImplement},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.config.GetCanonicalOrder()
			if len(got) != len(tt.expected) {
				t.Errorf("GetCanonicalOrder() returned %d phases, want %d", len(got), len(tt.expected))
				return
			}
			for i, phase := range got {
				if phase != tt.expected[i] {
					t.Errorf("GetCanonicalOrder()[%d] = %v, want %v", i, phase, tt.expected[i])
				}
			}
		})
	}
}

func TestSetAll(t *testing.T) {
	pc := &PhaseConfig{}
	pc.SetAll()
	if !pc.Specify || !pc.Plan || !pc.Tasks || !pc.Implement {
		t.Error("SetAll should enable all phases")
	}
}

func TestCount(t *testing.T) {
	tests := []struct {
		name     string
		config   PhaseConfig
		expected int
	}{
		{
			name:     "no phases",
			config:   PhaseConfig{},
			expected: 0,
		},
		{
			name:     "one core phase",
			config:   PhaseConfig{Plan: true},
			expected: 1,
		},
		{
			name:     "two core phases",
			config:   PhaseConfig{Plan: true, Implement: true},
			expected: 2,
		},
		{
			name:     "all core phases",
			config:   PhaseConfig{Specify: true, Plan: true, Tasks: true, Implement: true},
			expected: 4,
		},
		// Optional phase count tests
		{
			name:     "one optional phase",
			config:   PhaseConfig{Constitution: true},
			expected: 1,
		},
		{
			name:     "all optional phases",
			config:   PhaseConfig{Constitution: true, Clarify: true, Checklist: true, Analyze: true},
			expected: 4,
		},
		{
			name:     "mixed core and optional phases",
			config:   PhaseConfig{Specify: true, Clarify: true, Plan: true, Checklist: true},
			expected: 4,
		},
		{
			name:     "all 8 phases",
			config:   PhaseConfig{Constitution: true, Specify: true, Clarify: true, Plan: true, Tasks: true, Checklist: true, Analyze: true, Implement: true},
			expected: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.config.Count(); got != tt.expected {
				t.Errorf("Count() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetArtifactDependencies(t *testing.T) {
	deps := GetArtifactDependencies()

	// 4 core phases + 4 optional phases = 8 total
	if len(deps) != 8 {
		t.Errorf("GetArtifactDependencies() returned %d entries, want 8", len(deps))
	}

	// Verify each phase has a dependency entry
	phases := []Phase{
		// Core phases
		PhaseSpecify, PhasePlan, PhaseTasks, PhaseImplement,
		// Optional phases
		PhaseConstitution, PhaseClarify, PhaseChecklist, PhaseAnalyze,
	}
	for _, phase := range phases {
		if _, ok := deps[phase]; !ok {
			t.Errorf("GetArtifactDependencies() missing entry for %s", phase)
		}
	}
}

func TestGetArtifactDependency(t *testing.T) {
	tests := []struct {
		phase            Phase
		expectedRequires []string
		expectedProduces []string
	}{
		// Core phases
		{
			phase:            PhaseSpecify,
			expectedRequires: []string{},
			expectedProduces: []string{"spec.yaml"},
		},
		{
			phase:            PhasePlan,
			expectedRequires: []string{"spec.yaml"},
			expectedProduces: []string{"plan.yaml"},
		},
		{
			phase:            PhaseTasks,
			expectedRequires: []string{"plan.yaml"},
			expectedProduces: []string{"tasks.yaml"},
		},
		{
			phase:            PhaseImplement,
			expectedRequires: []string{"tasks.yaml"},
			expectedProduces: []string{},
		},
		// Optional phases
		{
			phase:            PhaseConstitution,
			expectedRequires: []string{},
			expectedProduces: []string{},
		},
		{
			phase:            PhaseClarify,
			expectedRequires: []string{"spec.yaml"},
			expectedProduces: []string{},
		},
		{
			phase:            PhaseChecklist,
			expectedRequires: []string{"spec.yaml"},
			expectedProduces: []string{},
		},
		{
			phase:            PhaseAnalyze,
			expectedRequires: []string{"spec.yaml", "plan.yaml", "tasks.yaml"},
			expectedProduces: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			t.Parallel()
			dep := GetArtifactDependency(tt.phase)
			if dep.Phase != tt.phase {
				t.Errorf("GetArtifactDependency(%s).Phase = %s, want %s", tt.phase, dep.Phase, tt.phase)
			}
			if len(dep.Requires) != len(tt.expectedRequires) {
				t.Errorf("GetArtifactDependency(%s).Requires = %v, want %v", tt.phase, dep.Requires, tt.expectedRequires)
			}
			if len(dep.Produces) != len(tt.expectedProduces) {
				t.Errorf("GetArtifactDependency(%s).Produces = %v, want %v", tt.phase, dep.Produces, tt.expectedProduces)
			}
		})
	}
}

func TestGetRequiredArtifacts(t *testing.T) {
	tests := []struct {
		phase    Phase
		expected []string
	}{
		// Core phases
		{PhaseSpecify, []string{}},
		{PhasePlan, []string{"spec.yaml"}},
		{PhaseTasks, []string{"plan.yaml"}},
		{PhaseImplement, []string{"tasks.yaml"}},
		// Optional phases
		{PhaseConstitution, []string{}},
		{PhaseClarify, []string{"spec.yaml"}},
		{PhaseChecklist, []string{"spec.yaml"}},
		{PhaseAnalyze, []string{"spec.yaml", "plan.yaml", "tasks.yaml"}},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			t.Parallel()
			got := GetRequiredArtifacts(tt.phase)
			if len(got) != len(tt.expected) {
				t.Errorf("GetRequiredArtifacts(%s) = %v, want %v", tt.phase, got, tt.expected)
			}
		})
	}
}

func TestGetProducedArtifacts(t *testing.T) {
	tests := []struct {
		phase    Phase
		expected []string
	}{
		// Core phases
		{PhaseSpecify, []string{"spec.yaml"}},
		{PhasePlan, []string{"plan.yaml"}},
		{PhaseTasks, []string{"tasks.yaml"}},
		{PhaseImplement, []string{}},
		// Optional phases (none produce tracked artifacts)
		{PhaseConstitution, []string{}},
		{PhaseClarify, []string{}},
		{PhaseChecklist, []string{}},
		{PhaseAnalyze, []string{}},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			t.Parallel()
			got := GetProducedArtifacts(tt.phase)
			if len(got) != len(tt.expected) {
				t.Errorf("GetProducedArtifacts(%s) = %v, want %v", tt.phase, got, tt.expected)
			}
		})
	}
}

func TestGetAllRequiredArtifacts(t *testing.T) {
	tests := []struct {
		name     string
		config   PhaseConfig
		expected []string
	}{
		// Core phases
		{
			name:     "all core phases - no external requirements",
			config:   PhaseConfig{Specify: true, Plan: true, Tasks: true, Implement: true},
			expected: []string{}, // specify produces what plan needs, etc.
		},
		{
			name:     "only plan - requires spec.yaml",
			config:   PhaseConfig{Plan: true},
			expected: []string{"spec.yaml"},
		},
		{
			name:     "only tasks - requires plan.yaml",
			config:   PhaseConfig{Tasks: true},
			expected: []string{"plan.yaml"},
		},
		{
			name:     "only implement - requires tasks.yaml",
			config:   PhaseConfig{Implement: true},
			expected: []string{"tasks.yaml"},
		},
		{
			name:     "plan and implement - requires spec.yaml (tasks.yaml covered by plan)",
			config:   PhaseConfig{Plan: true, Implement: true},
			expected: []string{"spec.yaml", "tasks.yaml"},
		},
		{
			name:     "specify only - no requirements",
			config:   PhaseConfig{Specify: true},
			expected: []string{},
		},
		// Optional phases
		{
			name:     "constitution only - no requirements",
			config:   PhaseConfig{Constitution: true},
			expected: []string{},
		},
		{
			name:     "clarify only - requires spec.yaml",
			config:   PhaseConfig{Clarify: true},
			expected: []string{"spec.yaml"},
		},
		{
			name:     "checklist only - requires spec.yaml",
			config:   PhaseConfig{Checklist: true},
			expected: []string{"spec.yaml"},
		},
		{
			name:     "analyze only - requires spec.yaml, plan.yaml, tasks.yaml",
			config:   PhaseConfig{Analyze: true},
			expected: []string{"spec.yaml", "plan.yaml", "tasks.yaml"},
		},
		// Mixed core and optional phases
		{
			name:     "specify with clarify - no external requirements (specify produces spec.yaml)",
			config:   PhaseConfig{Specify: true, Clarify: true},
			expected: []string{},
		},
		{
			name:     "tasks with checklist and analyze - requires plan.yaml (spec.yaml for checklist/analyze)",
			config:   PhaseConfig{Tasks: true, Checklist: true, Analyze: true},
			expected: []string{"plan.yaml", "spec.yaml"},
		},
		{
			name:     "full workflow with all 8 phases - no external requirements",
			config:   PhaseConfig{Constitution: true, Specify: true, Clarify: true, Plan: true, Tasks: true, Checklist: true, Analyze: true, Implement: true},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.config.GetAllRequiredArtifacts()

			// Convert to maps for comparison (order doesn't matter)
			gotMap := make(map[string]bool)
			for _, a := range got {
				gotMap[a] = true
			}
			expectedMap := make(map[string]bool)
			for _, a := range tt.expected {
				expectedMap[a] = true
			}

			if len(gotMap) != len(expectedMap) {
				t.Errorf("GetAllRequiredArtifacts() = %v, want %v", got, tt.expected)
				return
			}
			for artifact := range expectedMap {
				if !gotMap[artifact] {
					t.Errorf("GetAllRequiredArtifacts() missing %s, got %v", artifact, got)
				}
			}
		})
	}
}
