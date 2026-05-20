package workflow

import (
	"os"
	"testing"

	"github.com/ariel-frischer/autospec/internal/cliagent"
	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenCodeModelForStage(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cfg   config.OpenCodeConfig
		stage Stage
		want  string
	}{
		"stage model wins over default": {
			cfg: config.OpenCodeConfig{
				Model: "anthropic/default",
				Models: config.OpenCodeStageModels{
					Plan: "anthropic/plan",
				},
			},
			stage: StagePlan,
			want:  "anthropic/plan",
		},
		"default model used when stage empty": {
			cfg: config.OpenCodeConfig{
				Model: "anthropic/default",
			},
			stage: StageTasks,
			want:  "anthropic/default",
		},
		"empty when no default or stage model": {
			cfg:   config.OpenCodeConfig{},
			stage: StageImplement,
			want:  "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, OpenCodeModelForStage(tt.cfg, tt.stage))
		})
	}
}

func TestClaudeExecutorExecuteWithExtraArgsPassesOpenCodeModel(t *testing.T) {
	t.Parallel()

	agent := cliagent.NewOpenCode()
	executor := &ClaudeExecutor{
		Agent:   agent,
		Timeout: 5,
	}

	commandPath := writeSuccessfulCommand(t)
	agent.Cmd = commandPath

	err := executor.ExecuteWithExtraArgs("rendered prompt", []string{"--model", "anthropic/claude-opus-4-5-latest"})
	require.NoError(t, err)

	formatted := executor.FormatCommandWithExtraArgs("rendered prompt", []string{"--model", "anthropic/claude-opus-4-5-latest"})
	assert.Contains(t, formatted, "run rendered prompt --model anthropic/claude-opus-4-5-latest")
}

func writeSuccessfulCommand(t *testing.T) string {
	t.Helper()

	path := t.TempDir() + "/success"
	err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755)
	require.NoError(t, err)
	return path
}
