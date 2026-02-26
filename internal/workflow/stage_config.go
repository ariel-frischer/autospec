package workflow

// StageConfig represents the user's selected stages for execution.
// It determines which workflow stages (specify, plan, tasks, implement)
// and optional stages (constitution, clarify, checklist, analyze)
// will be executed during a run.
type StageConfig struct {
	// Core workflow stages
	Specify   bool
	Plan      bool
	Tasks     bool
	Implement bool

	// Optional stages
	Constitution bool
	Clarify      bool
	Checklist    bool
	Analyze      bool
}

// NewStageConfig creates a new StageConfig with all stages disabled.
func NewStageConfig() *StageConfig {
	return &StageConfig{}
}

// NewStageConfigAll creates a new StageConfig with all core stages enabled.
func NewStageConfigAll() *StageConfig {
	return &StageConfig{
		Specify:   true,
		Plan:      true,
		Tasks:     true,
		Implement: true,
	}
}

// HasAnyStage returns true if any stage (core or optional) is selected.
func (sc *StageConfig) HasAnyStage() bool {
	return sc.Specify || sc.Plan || sc.Tasks || sc.Implement ||
		sc.Constitution || sc.Clarify || sc.Checklist || sc.Analyze
}

// GetSelectedStages returns a slice of selected stages in canonical order.
// The canonical order is always: constitution -> specify -> clarify -> plan -> tasks -> checklist -> analyze -> implement.
func (sc *StageConfig) GetSelectedStages() []Stage {
	stages := make([]Stage, 0, 8)
	if sc.Constitution {
		stages = append(stages, StageConstitution)
	}
	if sc.Specify {
		stages = append(stages, StageSpecify)
	}
	if sc.Clarify {
		stages = append(stages, StageClarify)
	}
	if sc.Plan {
		stages = append(stages, StagePlan)
	}
	if sc.Tasks {
		stages = append(stages, StageTasks)
	}
	if sc.Checklist {
		stages = append(stages, StageChecklist)
	}
	if sc.Analyze {
		stages = append(stages, StageAnalyze)
	}
	if sc.Implement {
		stages = append(stages, StageImplement)
	}
	return stages
}

// GetCanonicalOrder is an alias for GetSelectedStages that returns stages
// in the canonical execution order:
// constitution -> specify -> clarify -> plan -> tasks -> checklist -> analyze -> implement
// This ensures stages always execute in the correct order regardless of
// the order in which flags were specified.
func (sc *StageConfig) GetCanonicalOrder() []Stage {
	return sc.GetSelectedStages()
}

// SetAll enables all core stages.
func (sc *StageConfig) SetAll() {
	sc.Specify = true
	sc.Plan = true
	sc.Tasks = true
	sc.Implement = true
}

// Count returns the number of selected stages (core and optional).
func (sc *StageConfig) Count() int {
	count := 0
	// Core stages
	if sc.Specify {
		count++
	}
	if sc.Plan {
		count++
	}
	if sc.Tasks {
		count++
	}
	if sc.Implement {
		count++
	}
	// Optional stages
	if sc.Constitution {
		count++
	}
	if sc.Clarify {
		count++
	}
	if sc.Checklist {
		count++
	}
	if sc.Analyze {
		count++
	}
	return count
}

// ArtifactDependency defines the relationship between a stage and its
// required/produced artifacts.
type ArtifactDependency struct {
	Stage    Stage
	Requires []string // Artifacts required before this stage can run
	Produces []string // Artifacts created by this stage
}

// artifactDependencies is the complete dependency map for all stages.
// It defines what each stage requires as input and produces as output.
var artifactDependencies = map[Stage]ArtifactDependency{
	// Core workflow stages
	StageSpecify: {
		Stage:    StageSpecify,
		Requires: []string{}, // Specify has no prerequisites
		Produces: []string{"spec.yaml"},
	},
	StagePlan: {
		Stage:    StagePlan,
		Requires: []string{"spec.yaml"},
		Produces: []string{"plan.yaml"},
	},
	StageTasks: {
		Stage:    StageTasks,
		Requires: []string{"plan.yaml"},
		Produces: []string{"tasks.yaml"},
	},
	StageImplement: {
		Stage:    StageImplement,
		Requires: []string{"tasks.yaml"},
		Produces: []string{}, // Implement modifies existing files, doesn't create new artifacts
	},

	// Optional stages
	StageConstitution: {
		Stage:    StageConstitution,
		Requires: []string{}, // Constitution has no prerequisites
		Produces: []string{}, // Constitution modifies .autospec/constitution.yaml
	},
	StageClarify: {
		Stage:    StageClarify,
		Requires: []string{"spec.yaml"}, // Clarify refines an existing spec
		Produces: []string{},            // Clarify updates spec.yaml in place
	},
	StageChecklist: {
		Stage:    StageChecklist,
		Requires: []string{"spec.yaml"}, // Checklist validates spec requirements
		Produces: []string{},            // Checklist creates checklist files in checklists/ dir
	},
	StageAnalyze: {
		Stage:    StageAnalyze,
		Requires: []string{"spec.yaml", "plan.yaml", "tasks.yaml"}, // Analyze validates all artifacts
		Produces: []string{},                                       // Analyze outputs analysis report
	},
}

// GetArtifactDependencies returns the complete dependency map for all stages.
func GetArtifactDependencies() map[Stage]ArtifactDependency {
	// Return a copy to prevent modification of the internal map
	result := make(map[Stage]ArtifactDependency, len(artifactDependencies))
	for k, v := range artifactDependencies {
		result[k] = v
	}
	return result
}

// GetArtifactDependency returns the artifact dependency for a specific stage.
func GetArtifactDependency(stage Stage) ArtifactDependency {
	return artifactDependencies[stage]
}

// GetRequiredArtifacts returns the required artifacts for a stage.
func GetRequiredArtifacts(stage Stage) []string {
	if dep, ok := artifactDependencies[stage]; ok {
		return dep.Requires
	}
	return []string{}
}

// GetProducedArtifacts returns the artifacts produced by a stage.
func GetProducedArtifacts(stage Stage) []string {
	if dep, ok := artifactDependencies[stage]; ok {
		return dep.Produces
	}
	return []string{}
}

// GetAllRequiredArtifacts returns all artifacts required by the selected stages,
// excluding artifacts that will be produced by earlier selected stages.
func (sc *StageConfig) GetAllRequiredArtifacts() []string {
	required := make(map[string]bool)
	produced := make(map[string]bool)

	// Iterate through stages in canonical order
	for _, stage := range sc.GetCanonicalOrder() {
		dep := artifactDependencies[stage]

		// Add requirements that won't be produced by earlier stages
		for _, req := range dep.Requires {
			if !produced[req] {
				required[req] = true
			}
		}

		// Mark what this stage produces
		for _, prod := range dep.Produces {
			produced[prod] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(required))
	for artifact := range required {
		result = append(result, artifact)
	}
	return result
}
