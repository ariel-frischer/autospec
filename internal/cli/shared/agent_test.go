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
		args              []string
		initialModel      string
		wantModel         string
		wantModelOverride string
	}{
		"absent model leaves configured model unchanged": {
			args:         []string{},
			initialModel: "configured",
			wantModel:    "configured",
		},
		"present model writes transient override": {
			args:              []string{"--model", "cli-model"},
			initialModel:      "configured",
			wantModel:         "configured",
			wantModelOverride: "cli-model",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := commandWithModelFlags()
			require.NoError(t, cmd.ParseFlags(tt.args))
			cfg := &config.Configuration{Model: tt.initialModel}

			ApplyModelOverride(cmd, cfg)

			assert.Equal(t, tt.wantModel, cfg.Model)
			assert.Equal(t, tt.wantModelOverride, cfg.ModelOverride)
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
