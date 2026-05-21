package cli

import (
	"testing"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowModelFlagsRegistered(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cmd *cobra.Command
	}{
		"run has model flag": {
			cmd: runCmd,
		},
		"prep has model flag": {
			cmd: prepCmd,
		},
		"constitution has model flag": {
			cmd: constitutionCmd,
		},
		"clarify has model flag": {
			cmd: clarifyCmd,
		},
		"checklist has model flag": {
			cmd: checklistCmd,
		},
		"analyze has model flag": {
			cmd: analyzeCmd,
		},
		"all has model flag": {
			cmd: allCmd,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			flag := tt.cmd.Flags().Lookup(shared.ModelFlagName)
			require.NotNil(t, flag, "model flag should be registered")
			assert.Equal(t, "string", flag.Value.Type())
			assert.Contains(t, flag.Usage, "model")
			assert.Contains(t, flag.Usage, "workflow")
		})
	}
}
