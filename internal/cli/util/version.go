package util

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	// Version information - set via ldflags during build
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// ASCII logo for autospec - minimal block style
// Both lines are exactly 33 display characters wide
var logo = []string{
	"▄▀█ █ █ ▀█▀ █▀█ █▀ █▀█ █▀▀ █▀▀",
	"█▀█ █▄█  █  █▄█ ▄█ █▀▀ ██▄ █▄▄",
}

// logoDisplayWidth is the visual width of the logo (for centering)
const logoDisplayWidth = 33

// Box drawing characters
const (
	boxTopLeft     = "╭"
	boxTopRight    = "╮"
	boxBottomLeft  = "╰"
	boxBottomRight = "╯"
	boxHorizontal  = "─"
	boxVertical    = "│"
)

var versionPlain bool

var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Display version information (v)",
	Long:    "Display version, commit, build date, and Go version information for autospec",
	Example: `  # Show version info
  autospec version

  # Plain output (for scripts)
  autospec version --plain`,
	Run: func(cmd *cobra.Command, args []string) {
		if versionPlain {
			printPlainVersion()
		} else {
			printPrettyVersion()
		}
	},
}

func init() {
	versionCmd.GroupID = shared.GroupGettingStarted
	versionCmd.Flags().BoolVar(&versionPlain, "plain", false, "Plain output without formatting")
}

// printPlainVersion prints a simple version output for scripting
func printPlainVersion() {
	fmt.Printf("autospec %s\n", Version)
	fmt.Printf("commit: %s\n", Commit)
	fmt.Printf("built: %s\n", BuildDate)
	fmt.Printf("go: %s\n", runtime.Version())
	fmt.Printf("platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

// SourceURL is the project source URL
const SourceURL = "https://github.com/ariel-frischer/autospec"

var sauceCmd = &cobra.Command{
	Use:   "sauce",
	Short: "Display the source URL",
	Long:  "Display the source URL for the autospec project",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println(SourceURL)
	},
}

// getTerminalWidth returns the terminal width, defaulting to 80 if unavailable
func getTerminalWidth() int {
	if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && width > 0 {
		return width
	}
	return 80
}

// centerText centers text within a given width
func centerText(text string, width int) string {
	textLen := len([]rune(text))
	if textLen >= width {
		return text
	}
	padding := (width - textLen) / 2
	return strings.Repeat(" ", padding) + text
}

// printPrettyVersion prints a styled version output with logo and box
func printPrettyVersion() {
	termWidth := getTerminalWidth()

	// Color setup
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()
	white := color.New(color.FgWhite, color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	// Print logo centered (use fixed display width for unicode block chars)
	fmt.Println()
	logoPadding := (termWidth - logoDisplayWidth) / 2
	for _, line := range logo {
		fmt.Println(cyan(strings.Repeat(" ", logoPadding) + line))
	}
	fmt.Println()

	// Tagline
	tagline := "Spec-Driven Development Automation"
	fmt.Println(dim(centerText(tagline, termWidth)))
	fmt.Println()

	// Build version info content
	info := []struct {
		label string
		value string
	}{
		{"Version", Version},
		{"Commit", truncateCommit(Commit)},
		{"Built", BuildDate},
		{"Go", runtime.Version()},
		{"Platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)},
	}

	// Calculate box width (minimum 40, max 60)
	boxWidth := 44
	if termWidth < 50 {
		boxWidth = termWidth - 6
	}
	contentWidth := boxWidth - 4 // Account for borders and padding

	// Print box centered
	boxPadding := (termWidth - boxWidth) / 2
	pad := strings.Repeat(" ", boxPadding)

	// Top border
	fmt.Println(pad + boxTopLeft + strings.Repeat(boxHorizontal, boxWidth-2) + boxTopRight)

	// Empty line
	fmt.Println(pad + boxVertical + strings.Repeat(" ", boxWidth-2) + boxVertical)

	// Content lines
	for _, item := range info {
		label := yellow(fmt.Sprintf("%12s", item.label))
		value := white(item.value)
		line := fmt.Sprintf("  %s    %s", label, value)
		// Pad to fill the box
		lineLen := 12 + 4 + len(item.value) + 2 // label width + spacing + value + margin
		if lineLen < contentWidth {
			line += strings.Repeat(" ", contentWidth-lineLen)
		}
		fmt.Println(pad + boxVertical + " " + line + " " + boxVertical)
	}

	// Empty line
	fmt.Println(pad + boxVertical + strings.Repeat(" ", boxWidth-2) + boxVertical)

	// Bottom border
	fmt.Println(pad + boxBottomLeft + strings.Repeat(boxHorizontal, boxWidth-2) + boxBottomRight)
	fmt.Println()
}

// truncateCommit shortens commit hash if it's too long
func truncateCommit(commit string) string {
	if len(commit) > 8 {
		return commit[:8]
	}
	return commit
}
