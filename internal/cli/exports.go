package cli

import "github.com/spf13/cobra"

// RootCmd returns the root cobra command.
// This accessor allows subpackages to register their commands with the root
// without exposing the rootCmd variable directly.
func RootCmd() *cobra.Command {
	return rootCmd
}
