package shared

import (
	"testing"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigRejectsExplicitConfigAndProfile(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.PersistentFlags().String("config", ".autospec/config.yml", "config")
	cmd.PersistentFlags().String("profile", "", "profile")
	require.NoError(t, cmd.ParseFlags([]string{"--config", "custom.yml", "--profile", "cheap"}))

	_, err := LoadConfig(cmd, "custom.yml")
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot be combined")
}

func TestLoadConfigWithoutProfileUsesExistingLoader(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.PersistentFlags().String("profile", "", "profile")

	cfg, err := LoadConfig(cmd, "")
	require.NoError(t, err)
	require.Equal(t, config.GetDefaults()["specs_dir"], cfg.SpecsDir)
}
