package shared

import (
	"testing"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyModelOverride(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args               []string
		initialModel       string
		initialEffort      string
		wantModel          string
		wantModelOverride  string
		wantEffort         string
		wantEffortOverride string
	}{
		"absent model leaves configured model unchanged": {
			args:          []string{},
			initialModel:  "configured",
			initialEffort: "medium",
			wantModel:     "configured",
			wantEffort:    "medium",
		},
		"present model writes transient override": {
			args:              []string{"--model", "cli-model"},
			initialModel:      "configured",
			wantModel:         "configured",
			wantModelOverride: "cli-model",
		},
		"present reasoning effort writes transient override": {
			args:               []string{"--reasoning-effort", "xhigh"},
			initialEffort:      "medium",
			wantEffort:         "medium",
			wantEffortOverride: "xhigh",
		},
		"reasoning effort shorthand writes transient override": {
			args:               []string{"-e", "high"},
			wantEffortOverride: "high",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := commandWithModelFlags()
			require.NoError(t, cmd.ParseFlags(tt.args))
			cfg := &config.Configuration{
				Model:           tt.initialModel,
				ReasoningEffort: tt.initialEffort,
			}

			ApplyModelOverride(cmd, cfg)

			assert.Equal(t, tt.wantModel, cfg.Model)
			assert.Equal(t, tt.wantModelOverride, cfg.ModelOverride)
			assert.Equal(t, tt.wantEffort, cfg.ReasoningEffort)
			assert.Equal(t, tt.wantEffortOverride, cfg.ReasoningEffortOverride)
		})
	}
}

func commandWithModelFlags() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	AddModelFlag(cmd)
	return cmd
}

func TestResolveOpenCodeAgent(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		flagValue   string
		configValue string
		want        string
	}{
		"cli flag takes priority over config": {
			flagValue:   "plan",
			configValue: "build",
			want:        "plan",
		},
		"config used when no flag": {
			configValue: "build",
			want:        "build",
		},
		"empty when neither flag nor config": {
			want: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{Use: "test"}
			AddOpenCodeAgentFlag(cmd)
			if tt.flagValue != "" {
				require.NoError(t, cmd.ParseFlags([]string{"--" + OpenCodeAgentFlagName, tt.flagValue}))
			}
			cfg := &config.Configuration{OpenCodeAgent: tt.configValue, AgentPreset: "opencode"}

			got := ResolveOpenCodeAgent(cmd, cfg)

			assert.Equal(t, tt.want, got)
		})
	}
}
