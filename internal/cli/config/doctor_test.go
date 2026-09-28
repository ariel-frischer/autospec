// Package config tests CLI configuration commands for autospec.
// Related: internal/cli/config/doctor.go
// Tags: config, cli, doctor, health

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDoctorCmd_Structure(t *testing.T) {
	assert.Equal(t, "doctor", doctorCmd.Use)
	assert.NotEmpty(t, doctorCmd.Short)
	assert.NotEmpty(t, doctorCmd.Long)
	assert.NotEmpty(t, doctorCmd.Example)
}

func TestDoctorCmd_Aliases(t *testing.T) {
	aliases := doctorCmd.Aliases
	assert.Contains(t, aliases, "doc", "Should have 'doc' alias")
}

func TestDoctorCmd_GroupID(t *testing.T) {
	// doctorCmd should be in the configuration group
	assert.Equal(t, "configuration", doctorCmd.GroupID)
}

func TestDoctorCmd_IsExecutable(t *testing.T) {
	// Doctor command should be executable (has Run function)
	// Verify Run or RunE is set
	assert.True(t, doctorCmd.Run != nil || doctorCmd.RunE != nil,
		"Doctor command should have a Run or RunE function")
}
