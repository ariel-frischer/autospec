package util

import (
	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/spf13/cobra"
)

// ckCmd is the command for checking if an update is available.
var ckCmd = &cobra.Command{
	Use:     "ck",
	Aliases: []string{"check"},
	Short:   "Check if an update is available",
	Long:    "Check if a newer version of autospec is available on GitHub releases.",
	Example: `  # Check for available updates
  autospec ck

  # Using the longer alias
  autospec check`,
	RunE: runCheck,
}

func init() {
	ckCmd.GroupID = shared.GroupGettingStarted
}

// runCheck executes the update check command.
func runCheck(cmd *cobra.Command, args []string) error {
	// TODO: Implement update check logic in Phase 2
	return nil
}
