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

func TestDoctorCmd_HasRunFunc(t *testing.T) {

	// Doctor uses Run, not RunE (because it handles errors internally)
	assert.NotNil(t, doctorCmd.Run, "Doctor command should have a Run function")
}

func TestDoctorCmd_DescriptionContents(t *testing.T) {

	tests := map[string]struct {
		wantContains string
		field        string
	}{
		"short mentions health checks": {
			wantContains: "health checks",
			field:        "Short",
		},
		"long mentions Claude CLI": {
			wantContains: "Claude CLI",
			field:        "Long",
		},
		"long mentions Git": {
			wantContains: "Git",
			field:        "Long",
		},
		"example contains doctor command": {
			wantContains: "doctor",
			field:        "Example",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

			var content string
			switch tt.field {
			case "Short":
				content = doctorCmd.Short
			case "Long":
				content = doctorCmd.Long
			case "Example":
				content = doctorCmd.Example
			}

			assert.Contains(t, content, tt.wantContains)
		})
	}
}

func TestDoctorCmd_IsExecutable(t *testing.T) {

	// Doctor command should be executable (has Run function)
	// Verify Run or RunE is set
	assert.True(t, doctorCmd.Run != nil || doctorCmd.RunE != nil,
		"Doctor command should have a Run or RunE function")
}
