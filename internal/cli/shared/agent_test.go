package shared_test

import (
	"testing"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestResolveOpenCodeAgent(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		flagValue   string
		configValue string
		want        string
	}{
		"cli flag takes priority over config": {flagValue: "Plan", configValue: "Build", want: "Plan"},
		"config used when no flag":            {flagValue: "", configValue: "Build", want: "Build"}, "empty when neither flag nor config": {flagValue: "", configValue: "", want: ""},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmd := &cobra.Command{}
			shared.AddOpenCodeAgentFlag(cmd)
			if tt.flagValue != "" {
				cmd.ParseFlags([]string{"--" + shared.OpenCodeAgentFlagName, tt.flagValue})
			}
			cfg := &config.Configuration{OpenCodeAgent: tt.configValue, AgentPreset: "opencode"}
			got := shared.ResolveOpenCodeAgent(cmd, cfg)
			assert.Equal(t, got, tt.want)
		})
	}
}
