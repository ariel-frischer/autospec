package shared

import (
	"testing"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// createTestCommand creates a cobra command with output-style flag for testing.
// Simulates cobra parsing by using cmd.ParseFlags to mark flags as Changed.
func createTestCommand(flagValue string) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.PersistentFlags().String("output-style", "", "")

	// Simulate parsing via command line args (this marks Changed=true)
	if flagValue != "" {
		cmd.ParseFlags([]string{"--output-style", flagValue})
	}
	return cmd
}

func TestApplyOutputStyle(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		configStyle string
		flagValue   string
		wantStyle   config.OutputStyle
	}{
		"flag takes precedence over config": {
			configStyle: "plain",
			flagValue:   "compact",
			wantStyle:   config.OutputStyleCompact,
		},
		"config used when flag not set": {
			configStyle: "minimal",
			flagValue:   "",
			wantStyle:   config.OutputStyleMinimal,
		},
		"default used when neither set": {
			configStyle: "",
			flagValue:   "",
			wantStyle:   config.OutputStyleDefault,
		},
		"raw style from flag": {
			configStyle: "default",
			flagValue:   "raw",
			wantStyle:   config.OutputStyleRaw,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create config with test style
			cfg := &config.Configuration{
				OutputStyle: tt.configStyle,
				AgentPreset: "claude",
				SpecsDir:    "./specs",
				StateDir:    "~/.autospec/state",
			}

			// Create orchestrator
			orch := workflow.NewWorkflowOrchestrator(cfg)

			// Create command with parsed flag
			cmd := createTestCommand(tt.flagValue)

			// Apply and check result
			result := ApplyOutputStyle(cmd, orch)
			assert.Equal(t, tt.wantStyle, result)
		})
	}
}

func TestApplyOutputStyle_InvalidFlagIgnored(t *testing.T) {
	t.Parallel()

	cfg := &config.Configuration{
		OutputStyle: "minimal",
		AgentPreset: "claude",
		SpecsDir:    "./specs",
		StateDir:    "~/.autospec/state",
	}

	orch := workflow.NewWorkflowOrchestrator(cfg)
	cmd := createTestCommand("invalid-style")

	// Invalid flag should fallback to config value
	result := ApplyOutputStyle(cmd, orch)
	assert.Equal(t, config.OutputStyleMinimal, result)
}
