package admin

import (
	"fmt"
	"strings"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/completion"
	"github.com/spf13/cobra"
)

// rootCmdRef holds a reference to the root command for completion generation.
// This is set during Register() to avoid circular imports.
var rootCmdRef *cobra.Command

// completionCmd is our custom completion command that includes the install subcommand
var completionCmd = &cobra.Command{
	Use:   "completion [command]",
	Short: "Generate shell completion scripts or install completions",
	Long: `Generate shell completion scripts for autospec or install them automatically.

Use the subcommands to generate completion scripts for various shells,
or use 'install' to automatically configure your shell.`,
}

var completionInstallCmd = &cobra.Command{
	Use:   "install [bash|zsh|fish|powershell]",
	Short: "Install shell completions for autospec",
	Long: `Install shell completions for autospec.

This command auto-detects your shell from the $SHELL environment variable
and installs completions appropriately:

  - Bash: Appends sourcing block to ~/.bashrc
  - Zsh: Appends sourcing block to ~/.zshrc
  - Fish: Writes completion file to ~/.config/fish/completions/
  - PowerShell: Appends sourcing block to $PROFILE

A backup is created before modifying any rc file (with .autospec-backup-TIMESTAMP suffix).

Use the --manual flag to display manual installation instructions without
modifying any files.`,
	Example: `  # Auto-detect shell and install
  autospec completion install

  # Install for a specific shell
  autospec completion install bash
  autospec completion install zsh
  autospec completion install fish
  autospec completion install powershell

  # Show manual installation instructions
  autospec completion install --manual
  autospec completion install bash --manual`,
	Args:      cobra.MaximumNArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	RunE:      runCompletionInstall,
}

// Shell-specific generation commands (matching Cobra's default behavior)
var completionBashCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generate the autocompletion script for bash",
	Long: `Generate the autocompletion script for the bash shell.

To load completions in your current shell session:

	source <(autospec completion bash)

To load completions for every new session, execute once:

#### Linux:

	autospec completion bash > /etc/bash_completion.d/autospec

#### macOS:

	autospec completion bash > $(brew --prefix)/etc/bash_completion.d/autospec

You will need to start a new shell for this setup to take effect.
`,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmdRef.GenBashCompletionV2(cmd.OutOrStdout(), true)
	},
}

var completionZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generate the autocompletion script for zsh",
	Long: `Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(autospec completion zsh)

To load completions for every new session, execute once:

#### Linux:

	autospec completion zsh > "${fpath[1]}/_autospec"

#### macOS:

	autospec completion zsh > $(brew --prefix)/share/zsh/site-functions/_autospec

You will need to start a new shell for this setup to take effect.
`,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmdRef.GenZshCompletion(cmd.OutOrStdout())
	},
}

var completionFishCmd = &cobra.Command{
	Use:   "fish",
	Short: "Generate the autocompletion script for fish",
	Long: `Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	autospec completion fish | source

To load completions for every new session, execute once:

	autospec completion fish > ~/.config/fish/completions/autospec.fish

You will need to start a new shell for this setup to take effect.
`,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmdRef.GenFishCompletion(cmd.OutOrStdout(), true)
	},
}

var completionPowershellCmd = &cobra.Command{
	Use:   "powershell",
	Short: "Generate the autocompletion script for powershell",
	Long: `Generate the autocompletion script for powershell.

To load completions in your current shell session:

	autospec completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.
`,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmdRef.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
	},
}

var manualFlag bool

func init() {
	completionCmd.GroupID = shared.GroupConfiguration

	// Add shell-specific generation commands
	completionCmd.AddCommand(completionBashCmd)
	completionCmd.AddCommand(completionZshCmd)
	completionCmd.AddCommand(completionFishCmd)
	completionCmd.AddCommand(completionPowershellCmd)

	// Add the install subcommand
	completionCmd.AddCommand(completionInstallCmd)

	completionInstallCmd.Flags().BoolVar(&manualFlag, "manual", false, "Show manual installation instructions without modifying files")
}

func runCompletionInstall(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()

	// Determine the shell
	var shell completion.Shell
	var err error

	if len(args) > 0 {
		// Explicit shell argument
		shellArg := strings.ToLower(args[0])
		if !completion.IsValidShell(shellArg) {
			supportedShells := make([]string, len(completion.SupportedShells()))
			for i, s := range completion.SupportedShells() {
				supportedShells[i] = string(s)
			}
			return fmt.Errorf("unknown shell: %s\nSupported shells: %s", shellArg, strings.Join(supportedShells, ", "))
		}
		shell = completion.Shell(shellArg)
	} else {
		// Auto-detect shell
		shell, err = completion.DetectShell()
		if err != nil {
			// If detection fails, show help with all manual instructions
			fmt.Fprintln(out, "Could not auto-detect shell:", err)
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Please specify a shell explicitly:")
			fmt.Fprintln(out, "  autospec completion install bash")
			fmt.Fprintln(out, "  autospec completion install zsh")
			fmt.Fprintln(out, "  autospec completion install fish")
			fmt.Fprintln(out, "  autospec completion install powershell")
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Or use --manual for installation instructions:")
			fmt.Fprintln(out, "  autospec completion install --manual")
			return nil
		}
		fmt.Fprintf(out, "Detected shell: %s\n", shell)
	}

	// Handle --manual flag
	if manualFlag {
		fmt.Fprintln(out, completion.GetManualInstructions(shell))
		return nil
	}

	// Perform installation
	result, err := completion.Install(shell)
	if err != nil {
		// Check for permission error
		if completion.IsPermissionError(err) {
			fmt.Fprintf(out, "Error: %v\n\n", err)
			fmt.Fprintln(out, "Automatic installation failed. Here are manual instructions:")
			fmt.Fprintln(out)
			fmt.Fprintln(out, completion.GetManualInstructions(shell))
			return nil
		}
		return fmt.Errorf("installing shell completion: %w", err)
	}

	// Display result
	fmt.Fprintln(out, result.Message)

	return nil
}
