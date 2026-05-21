package stages

import (
	"testing"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoCommitFlagsRegistered(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cmd *cobra.Command
	}{
		"specify has auto-commit flags": {
			cmd: specifyCmd,
		},
		"plan has auto-commit flags": {
			cmd: planCmd,
		},
		"tasks has auto-commit flags": {
			cmd: tasksCmd,
		},
		"implement has auto-commit flags": {
			cmd: implementCmd,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Verify --auto-commit flag is registered
			autoCommitFlag := tt.cmd.Flags().Lookup(shared.AutoCommitFlagName)
			assert.NotNil(t, autoCommitFlag, "auto-commit flag should be registered")
			assert.Equal(t, "bool", autoCommitFlag.Value.Type())

			// Verify --no-auto-commit flag is registered
			noAutoCommitFlag := tt.cmd.Flags().Lookup(shared.NoAutoCommitFlagName)
			assert.NotNil(t, noAutoCommitFlag, "no-auto-commit flag should be registered")
			assert.Equal(t, "bool", noAutoCommitFlag.Value.Type())
		})
	}
}

func TestAutoCommitFlagsHelpText(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cmd *cobra.Command
	}{
		"specify auto-commit help text": {
			cmd: specifyCmd,
		},
		"plan auto-commit help text": {
			cmd: planCmd,
		},
		"tasks auto-commit help text": {
			cmd: tasksCmd,
		},
		"implement auto-commit help text": {
			cmd: implementCmd,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			autoCommitFlag := tt.cmd.Flags().Lookup(shared.AutoCommitFlagName)
			assert.NotEmpty(t, autoCommitFlag.Usage, "auto-commit flag should have help text")

			noAutoCommitFlag := tt.cmd.Flags().Lookup(shared.NoAutoCommitFlagName)
			assert.NotEmpty(t, noAutoCommitFlag.Usage, "no-auto-commit flag should have help text")
		})
	}
}

func TestModelFlagsRegistered(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cmd *cobra.Command
	}{
		"specify has model flag": {
			cmd: specifyCmd,
		},
		"plan has model flag": {
			cmd: planCmd,
		},
		"tasks has model flag": {
			cmd: tasksCmd,
		},
		"implement has model flag": {
			cmd: implementCmd,
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
