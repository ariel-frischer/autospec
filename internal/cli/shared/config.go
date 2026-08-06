package shared

import (
	"fmt"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/spf13/cobra"
)

// LoadConfig loads the config path used by a command and applies its profile.
func LoadConfig(cmd *cobra.Command, configPath string) (*config.Configuration, error) {
	profile, _ := cmd.Flags().GetString("profile")
	if profile != "" && cmd.Flags().Changed("config") {
		return nil, fmt.Errorf("--config and --profile cannot be combined; use --config for an explicit file or --profile for a named profile")
	}

	return config.LoadWithOptions(config.LoadOptions{
		ProjectConfigPath: configPath,
		Profile:           profile,
	})
}
