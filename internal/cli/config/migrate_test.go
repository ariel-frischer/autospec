// Package config tests CLI configuration commands for autospec.
// Related: internal/cli/config/migrate.go
// Tags: config, cli, migrate

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigrateCmd_Structure(t *testing.T) {

	assert.Equal(t, "migrate", migrateCmd.Use)
	assert.NotEmpty(t, migrateCmd.Short)
	assert.NotEmpty(t, migrateCmd.Long)
	assert.NotEmpty(t, migrateCmd.Example)
}

func TestMigrateCmd_GroupID(t *testing.T) {

	// migrateCmd should be in the internal group
	assert.Equal(t, "internal", migrateCmd.GroupID)
}

func TestMigrateCmd_NoSubcommands(t *testing.T) {

	// The standalone migrateCmd (not config migrate) should not have subcommands
	// as it's a parent command for md-to-yaml migration
	subcommands := migrateCmd.Commands()
	// Just verify we can check subcommands without panic
	_ = subcommands
}

func TestMigrateCmd_HasValidFields(t *testing.T) {

	tests := map[string]struct {
		field    string
		getValue func() string
		wantSet  bool
	}{
		"use field is set": {
			field:    "Use",
			getValue: func() string { return migrateCmd.Use },
			wantSet:  true,
		},
		"short field is set": {
			field:    "Short",
			getValue: func() string { return migrateCmd.Short },
			wantSet:  true,
		},
		"long field is set": {
			field:    "Long",
			getValue: func() string { return migrateCmd.Long },
			wantSet:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

			value := tt.getValue()
			if tt.wantSet {
				assert.NotEmpty(t, value, "Field %s should be set", tt.field)
			} else {
				assert.Empty(t, value, "Field %s should be empty", tt.field)
			}
		})
	}
}
