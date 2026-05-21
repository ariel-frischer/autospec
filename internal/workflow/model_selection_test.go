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
		"cli model wins for opencode": {
			cfg: config.Configuration{
				Model:         "generic-config",
				ModelOverride: "generic-cli",
			},
			input:      ModelSelectionInput{Agent: "opencode", Stage: StagePlan},
			wantValue:  "generic-cli",
			wantSource: ModelSourceCLI,
		},
		"configured model used for claude": {
			cfg: config.Configuration{
				Model: "generic-config",
			},
			input:      ModelSelectionInput{Agent: "claude", Stage: StagePlan},
			wantValue:  "generic-config",
			wantSource: ModelSourceConfig,
		},
		"configured model used for codex": {
			cfg: config.Configuration{
				Model: "generic-config",
			},
			input:      ModelSelectionInput{Agent: "codex", Stage: StagePlan},
			wantValue:  "generic-config",
			wantSource: ModelSourceConfig,
		},
		"configured model used for opencode": {
			cfg: config.Configuration{
				Model: "generic-config",
			},
			input:      ModelSelectionInput{Agent: "opencode", Stage: StageTasks},
			wantValue:  "generic-config",
			wantSource: ModelSourceConfig,
		},
		"empty default source when no model configured": {
			cfg:        config.Configuration{},
			input:      ModelSelectionInput{Agent: "opencode", Stage: StageImplement},
			wantValue:  "",
			wantSource: ModelSourceDefault,
		},
		"unsupported agent ignores configured model": {
			cfg:        config.Configuration{Model: "generic-config", ModelOverride: "generic-cli"},
			input:      ModelSelectionInput{Agent: "gemini", Stage: StagePlan},
			wantValue:  "",
			wantSource: ModelSourceDefault,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := ResolveWorkflowModelSelection(tt.cfg, tt.input)

			assert.Equal(t, tt.wantValue, got.Value)
			assert.Equal(t, tt.wantSource, got.Source)
			assert.Equal(t, tt.input.Stage, got.Stage)
			assert.Equal(t, tt.input.Agent, got.Agent)
		})
	}
}
