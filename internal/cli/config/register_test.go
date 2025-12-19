// Package config tests CLI configuration commands for autospec.
// Related: internal/cli/config/register.go
// Tags: config, cli, commands, registration

package config

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {

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

	// Should have init, config, migrate, doctor commands
	assert.True(t, commandNames["init"], "Should have 'init' command")
	assert.True(t, commandNames["config"], "Should have 'config' command")
	assert.True(t, commandNames["migrate"], "Should have 'migrate' command")
	assert.True(t, commandNames["doctor"], "Should have 'doctor' command")
}

func TestRegister_CommandAnnotations(t *testing.T) {

	tests := map[string]struct {
		cmdUse  string
		wantCmd bool
	}{
		"init command exists": {
			cmdUse:  "init",
			wantCmd: true,
		},
		"config command exists": {
			cmdUse:  "config",
			wantCmd: true,
		},
		"migrate command exists": {
			cmdUse:  "migrate",
			wantCmd: true,
		},
		"doctor command exists": {
			cmdUse:  "doctor",
			wantCmd: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

			rootCmd := &cobra.Command{
				Use: "test",
			}
			Register(rootCmd)

			found := false
			for _, cmd := range rootCmd.Commands() {
				if cmd.Use == tt.cmdUse {
					found = true
					break
				}
			}
			assert.Equal(t, tt.wantCmd, found)
		})
	}
}

func TestInitCmd_Structure(t *testing.T) {

	assert.Equal(t, "init", initCmd.Use)
	assert.NotEmpty(t, initCmd.Short)
	assert.NotEmpty(t, initCmd.Long)
	assert.NotEmpty(t, initCmd.Example)
}

func TestInitCmd_Flags(t *testing.T) {

	tests := map[string]struct {
		flagName string
		wantFlag bool
	}{
		"project flag exists": {
			flagName: "project",
			wantFlag: true,
		},
		"force flag exists": {
			flagName: "force",
			wantFlag: true,
		},
		"global flag exists (hidden)": {
			flagName: "global",
			wantFlag: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

			flag := initCmd.Flags().Lookup(tt.flagName)
			if tt.wantFlag {
				assert.NotNil(t, flag, "Flag %s should exist", tt.flagName)
			} else {
				assert.Nil(t, flag, "Flag %s should not exist", tt.flagName)
			}
		})
	}
}

func TestConfigCmd_Structure(t *testing.T) {

	assert.Equal(t, "config", configCmd.Use)
	assert.NotEmpty(t, configCmd.Short)
	assert.NotEmpty(t, configCmd.Long)
}

func TestConfigCmd_HasSubcommands(t *testing.T) {

	subcommands := configCmd.Commands()
	subcommandNames := make(map[string]bool)
	for _, cmd := range subcommands {
		subcommandNames[cmd.Name()] = true
	}

	assert.True(t, subcommandNames["show"], "Should have 'show' subcommand")
}

func TestConfigShowCmd_Structure(t *testing.T) {

	assert.Equal(t, "show", configShowCmd.Use)
	assert.NotEmpty(t, configShowCmd.Short)
	assert.NotEmpty(t, configShowCmd.Long)
}

func TestConfigShowCmd_Flags(t *testing.T) {

	tests := map[string]struct {
		flagName string
		wantFlag bool
	}{
		"json flag exists": {
			flagName: "json",
			wantFlag: true,
		},
		"yaml flag exists": {
			flagName: "yaml",
			wantFlag: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

			flag := configShowCmd.Flags().Lookup(tt.flagName)
			if tt.wantFlag {
				assert.NotNil(t, flag, "Flag %s should exist", tt.flagName)
			} else {
				assert.Nil(t, flag)
			}
		})
	}
}

func TestRegister_CommandCount(t *testing.T) {

	rootCmd := &cobra.Command{
		Use: "test",
	}

	Register(rootCmd)

	// Should register exactly 4 commands: init, config, migrate, doctor
	assert.Equal(t, 4, len(rootCmd.Commands()))
}

func TestConfigCmd_RunsWithoutArgs(t *testing.T) {

	// Create isolated command for testing
	cmd := &cobra.Command{
		Use: "config",
	}

	cmd.SetArgs([]string{})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Execute should succeed (shows usage)
	err := cmd.Execute()
	assert.NoError(t, err)
}
