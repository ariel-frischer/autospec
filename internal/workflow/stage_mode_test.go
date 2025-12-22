package workflow

import "testing"

func TestIsInteractive(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stage Stage
		want  bool
	}{
		"analyze is interactive": {
			stage: StageAnalyze,
			want:  true,
		},
		"clarify is interactive": {
			stage: StageClarify,
			want:  true,
		},
		"specify is automated": {
			stage: StageSpecify,
			want:  false,
		},
		"plan is automated": {
			stage: StagePlan,
			want:  false,
		},
		"tasks is automated": {
			stage: StageTasks,
			want:  false,
		},
		"implement is automated": {
			stage: StageImplement,
			want:  false,
		},
		"constitution is automated": {
			stage: StageConstitution,
			want:  false,
		},
		"checklist is automated": {
			stage: StageChecklist,
			want:  false,
		},
		"unknown stage is automated": {
			stage: Stage("unknown"),
			want:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsInteractive(tt.stage)
			if got != tt.want {
				t.Errorf("IsInteractive(%q) = %v, want %v", tt.stage, got, tt.want)
			}
		})
	}
}

func TestStageModeConstants(t *testing.T) {
	t.Parallel()

	// Verify constants have distinct values
	if StageModeAutomated == StageModeInteractive {
		t.Error("StageModeAutomated and StageModeInteractive should have distinct values")
	}

	// Verify Automated is the zero value (default)
	var defaultMode StageMode
	if defaultMode != StageModeAutomated {
		t.Error("default StageMode should be StageModeAutomated")
	}
}
