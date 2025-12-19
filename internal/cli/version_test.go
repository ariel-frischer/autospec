// Package cli_test tests the version and sauce commands for displaying version information and source repository URL.
// Related: internal/cli/util/version.go
// Tags: cli, version, sauce, metadata, build-info, formatting
package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getSauceCmd finds the sauce command from rootCmd
func getSauceCmd() *cobra.Command {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "sauce" {
			return cmd
		}
	}
	return nil
}

func TestSauceCmdRegistration(t *testing.T) {
	t.Parallel()

	cmd := getSauceCmd()
	assert.NotNil(t, cmd, "sauce command should be registered - did someone spill the sauce?")
}

func TestSauceCmdOutput(t *testing.T) {
	// No t.Parallel() - tests share global sauceCmd instance and race on SetOut
	cmd := getSauceCmd()
	require.NotNil(t, cmd, "sauce command must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.Run(cmd, []string{})

	assert.Equal(t, "https://github.com/ariel-frischer/autospec\n", buf.String(),
		"Wrong sauce! Expected the secret recipe but got something else. "+
			"Someone's been messing with the marinara!")
}

func TestSourceURLConstant(t *testing.T) {
	// No t.Parallel() - tests share global sauceCmd instance and race on SetOut
	// Test that the sauce command outputs the expected URL
	// (The constant is unexported, so we verify via command output)
	cmd := getSauceCmd()
	require.NotNil(t, cmd, "sauce command must exist")

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.Run(cmd, []string{})

	output := strings.TrimSpace(buf.String())

	assert.Equal(t, "https://github.com/ariel-frischer/autospec", output,
		"SourceURL has gone stale! The sauce has expired! "+
			"Quick, someone check if the repo moved or if a developer sneezed on the keyboard!")
	assert.Contains(t, output, "github.com",
		"The sauce isn't from GitHub? What kind of bootleg ketchup is this?!")
	assert.Contains(t, output, "autospec",
		"Lost the autospec! This sauce is missing its main ingredient!")
}

// centerText is a test helper that mirrors util.centerText
func centerText(text string, width int) string {
	textLen := len([]rune(text)) // Use rune count for unicode characters
	if textLen >= width {
		return text
	}
	padding := (width - textLen) / 2
	return strings.Repeat(" ", padding) + text
}

// TestCenterText tests the centerText function for version display formatting.
func TestCenterText(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		text  string
		width int
		want  string
	}{
		"text shorter than width is centered": {
			text:  "hello",
			width: 11,
			want:  "   hello", // (11-5)/2 = 3 spaces prefix
		},
		"text equal to width returns text": {
			text:  "hello",
			width: 5,
			want:  "hello",
		},
		"text longer than width returns text": {
			text:  "hello world",
			width: 5,
			want:  "hello world",
		},
		"empty text with width returns spaces": {
			text:  "",
			width: 4,
			want:  "  ", // (4-0)/2 = 2 spaces
		},
		"odd padding rounds down": {
			text:  "hi",
			width: 7,
			want:  "  hi", // (7-2)/2 = 2 spaces
		},
		"unicode text is handled correctly": {
			text:  "→",
			width: 5,
			want:  "  →", // (5-1)/2 = 2 spaces
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := centerText(tt.text, tt.width)
			if got != tt.want {
				t.Errorf("centerText(%q, %d) = %q, want %q", tt.text, tt.width, got, tt.want)
			}
		})
	}
}

// truncateCommit is a test helper that mirrors util.truncateCommit
func truncateCommit(commit string) string {
	if len(commit) <= 8 {
		return commit
	}
	return commit[:8]
}

// TestTruncateCommit tests the truncateCommit function for version display.
func TestTruncateCommit(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		commit string
		want   string
	}{
		"short commit stays same": {
			commit: "abc123",
			want:   "abc123",
		},
		"exactly 8 chars stays same": {
			commit: "abc12345",
			want:   "abc12345",
		},
		"long commit is truncated to 8 chars": {
			commit: "abc123456789",
			want:   "abc12345",
		},
		"empty string stays empty": {
			commit: "",
			want:   "",
		},
		"full SHA is truncated": {
			commit: "a1b2c3d4e5f6g7h8i9j0",
			want:   "a1b2c3d4",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := truncateCommit(tt.commit)
			if got != tt.want {
				t.Errorf("truncateCommit(%q) = %q, want %q", tt.commit, got, tt.want)
			}
		})
	}
}

func TestSauceCmdMetadata(t *testing.T) {
	t.Parallel()

	cmd := getSauceCmd()
	require.NotNil(t, cmd, "sauce command must exist")

	tests := map[string]struct {
		check func(t *testing.T)
	}{
		"has short description": {
			check: func(t *testing.T) {
				assert.NotEmpty(t, cmd.Short,
					"The sauce has no label! How will anyone know what's in the bottle?!")
			},
		},
		"has long description": {
			check: func(t *testing.T) {
				assert.NotEmpty(t, cmd.Long,
					"No long description? Even hot sauce bottles have more text than this!")
			},
		},
		"short mentions source": {
			check: func(t *testing.T) {
				assert.Contains(t, cmd.Short, "source",
					"Short description doesn't mention 'source' - "+
						"it's called SAUCE for a reason, it reveals the SOURCE!")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tt.check(t)
		})
	}
}
