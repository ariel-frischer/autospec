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
