package workflow

import (
	"testing"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestResolveWorkflowModelSelection(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cfg        config.Configuration
		input      ModelSelectionInput
		wantValue  string
		wantSource ModelSelectionSource
	}{
		"cli model wins over stage and top-level values": {
			cfg: config.Configuration{
				Model:         "top-level-model",
				Models:        config.StageModels{Plan: "stage-model"},
				ModelOverride: "cli-model",
			},
			input:      ModelSelectionInput{Agent: "opencode", Stage: StagePlan},
			wantValue:  "cli-model",
			wantSource: ModelSourceCLI,
		},
		"stage model wins over top-level value": {
			cfg: config.Configuration{
				Model:  "top-level-model",
				Models: config.StageModels{Plan: "stage-model"},
			},
			input:      ModelSelectionInput{Agent: "claude", Stage: StagePlan},
			wantValue:  "stage-model",
			wantSource: ModelSourceStage,
		},
		"empty matching stage falls back to top-level value": {
			cfg: config.Configuration{
				Model:  "top-level-model",
				Models: config.StageModels{Specify: "neighbor-model"},
			},
			input:      ModelSelectionInput{Agent: "codex", Stage: StagePlan},
			wantValue:  "top-level-model",
			wantSource: ModelSourceConfig,
		},
		"matching stage is isolated from neighboring stages": {
			cfg: config.Configuration{
				Models: config.StageModels{
					Plan:  "plan-model",
					Tasks: "tasks-model",
				},
			},
			input:      ModelSelectionInput{Agent: "opencode", Stage: StagePlan},
			wantValue:  "plan-model",
			wantSource: ModelSourceStage,
		},
		"empty default source when no model configured": {
			cfg:        config.Configuration{},
			input:      ModelSelectionInput{Agent: "opencode", Stage: StageImplement},
			wantValue:  "",
			wantSource: ModelSourceDefault,
		},
		"unsupported agent ignores configured model": {
			cfg: config.Configuration{
				Model:         "top-level-model",
				Models:        config.StageModels{Plan: "stage-model"},
				ModelOverride: "cli-model",
			},
			input:      ModelSelectionInput{Agent: "gemini", Stage: StagePlan},
			wantValue:  "",
			wantSource: ModelSourceDefault,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			originalConfig := tt.cfg
			got := ResolveWorkflowModelSelection(tt.cfg, tt.input)

			assert.Equal(t, tt.wantValue, got.Value)
			assert.Equal(t, tt.wantSource, got.Source)
			assert.Equal(t, tt.input.Stage, got.Stage)
			assert.Equal(t, tt.input.Agent, got.Agent)
			assert.Equal(t, originalConfig, tt.cfg)
		})
	}
}
