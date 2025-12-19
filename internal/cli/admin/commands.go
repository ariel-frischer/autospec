package admin

import (
	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/spf13/cobra"
)

var commandsCmd = &cobra.Command{
	Use:   "commands",
	Short: "Manage autospec command templates",
	Long:  `Commands for installing, checking, and viewing autospec command templates.`,
}

func init() {
	commandsCmd.GroupID = shared.GroupInternal
}
