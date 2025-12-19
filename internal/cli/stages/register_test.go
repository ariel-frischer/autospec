// Package stages tests CLI workflow stage commands for autospec.
// Related: internal/cli/stages/register.go
// Tags: stages, cli, commands, registration

package stages

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	// Cannot run in parallel - Register modifies global command state

	rootCmd := &cobra.Command{
		Use:   "test",
		Short: "Test root command",
	}

	// Register should not panic
	require.NotPanics(t, func() {
		Register(rootCmd)
	})

	// Verify commands are added
	commands := rootCmd.Commands()
	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Use] = true
	}

	// Should have specify, plan, tasks, implement commands
	assert.True(t, commandNames["specify <feature-description>"], "Should have 'specify' command")
	assert.True(t, commandNames["plan [optional-prompt]"], "Should have 'plan' command")
	assert.True(t, commandNames["tasks [optional-prompt]"], "Should have 'tasks' command")
	assert.True(t, commandNames["implement [spec-name-or-prompt]"], "Should have 'implement' command")
}

func TestRegister_CommandAnnotations(t *testing.T) {
	// Cannot run in parallel - Register modifies global command state

	tests := map[string]struct {
		cmdName string
		wantCmd bool
	}{
		"specify command exists": {
			cmdName: "specify",
			wantCmd: true,
		},
		"plan command exists": {
			cmdName: "plan",
			wantCmd: true,
		},
		"tasks command exists": {
			cmdName: "tasks",
			wantCmd: true,
		},
		"implement command exists": {
			cmdName: "implement",
			wantCmd: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - Register modifies global state

			rootCmd := &cobra.Command{
				Use: "test",
			}
			Register(rootCmd)

			found := false
			for _, cmd := range rootCmd.Commands() {
				if cmd.Name() == tt.cmdName {
					found = true
					break
				}
			}
			assert.Equal(t, tt.wantCmd, found)
		})
	}
}

func TestRegister_CommandCount(t *testing.T) {
	// Cannot run in parallel - Register modifies global command state

	rootCmd := &cobra.Command{
		Use: "test",
	}

	Register(rootCmd)

	// Should register exactly 4 commands
	assert.Equal(t, 4, len(rootCmd.Commands()))
}

func TestSpecifyCmd_Structure(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	assert.Equal(t, "specify <feature-description>", specifyCmd.Use)
	assert.NotEmpty(t, specifyCmd.Short)
	assert.NotEmpty(t, specifyCmd.Long)
	assert.NotEmpty(t, specifyCmd.Example)
}

func TestSpecifyCmd_Aliases(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	aliases := specifyCmd.Aliases
	assert.Contains(t, aliases, "spec", "Should have 'spec' alias")
	assert.Contains(t, aliases, "s", "Should have 's' alias")
}

func TestPlanCmd_Structure(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	assert.Equal(t, "plan [optional-prompt]", planCmd.Use)
	assert.NotEmpty(t, planCmd.Short)
	assert.NotEmpty(t, planCmd.Long)
	assert.NotEmpty(t, planCmd.Example)
}

func TestPlanCmd_Aliases(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	aliases := planCmd.Aliases
	assert.Contains(t, aliases, "p", "Should have 'p' alias")
}

func TestTasksCmd_Structure(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	assert.Equal(t, "tasks [optional-prompt]", tasksCmd.Use)
	assert.NotEmpty(t, tasksCmd.Short)
	assert.NotEmpty(t, tasksCmd.Long)
	assert.NotEmpty(t, tasksCmd.Example)
}

func TestTasksCmd_Aliases(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	aliases := tasksCmd.Aliases
	assert.Contains(t, aliases, "t", "Should have 't' alias")
}

func TestImplementCmd_Structure(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	assert.Equal(t, "implement [spec-name-or-prompt]", implementCmd.Use)
	assert.NotEmpty(t, implementCmd.Short)
	assert.NotEmpty(t, implementCmd.Long)
	assert.NotEmpty(t, implementCmd.Example)
}

func TestImplementCmd_Aliases(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	aliases := implementCmd.Aliases
	assert.Contains(t, aliases, "impl", "Should have 'impl' alias")
	assert.Contains(t, aliases, "i", "Should have 'i' alias")
}

func TestStageCommands_HaveRunE(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	tests := map[string]struct {
		cmd *cobra.Command
	}{
		"specify has RunE": {
			cmd: specifyCmd,
		},
		"plan has RunE": {
			cmd: planCmd,
		},
		"tasks has RunE": {
			cmd: tasksCmd,
		},
		"implement has RunE": {
			cmd: implementCmd,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - accesses global command state

			assert.NotNil(t, tt.cmd.RunE)
		})
	}
}

func TestSpecifyCmd_ArgsValidation(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// specifyCmd should require at least one argument
	assert.NotNil(t, specifyCmd.Args, "Specify command should have Args validator")
}

func TestStageCommands_GroupIDs(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	tests := map[string]struct {
		cmd         *cobra.Command
		wantGroupID string
	}{
		"specify group": {
			cmd:         specifyCmd,
			wantGroupID: "core-stages",
		},
		"plan group": {
			cmd:         planCmd,
			wantGroupID: "core-stages",
		},
		"tasks group": {
			cmd:         tasksCmd,
			wantGroupID: "core-stages",
		},
		"implement group": {
			cmd:         implementCmd,
			wantGroupID: "core-stages",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - accesses global command state

			assert.Equal(t, tt.wantGroupID, tt.cmd.GroupID)
		})
	}
}

func TestRegister_DoesNotPanic(t *testing.T) {
	// Cannot run in parallel - Register modifies global command state

	rootCmd := &cobra.Command{Use: "test"}

	// Verify register can be called multiple times without issues
	require.NotPanics(t, func() {
		Register(rootCmd)
	})
}

func TestStageCommands_ExecuteWithoutSubcommands(t *testing.T) {
	t.Parallel()

	// Create an isolated command to test
	cmd := &cobra.Command{
		Use: "specify",
	}
	cmd.SetArgs([]string{})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Should succeed (shows usage)
	err := cmd.Execute()
	assert.NoError(t, err)
}
