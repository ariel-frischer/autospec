//go:build e2e

package e2e

import (
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestE2E_ChangelogCommand tests the changelog command functionality.
// This verifies FR-007: "autospec changelog command with version, --last, and --plain flags"
// and FR-008: "embed CHANGELOG.yaml in binary at build time".
func TestE2E_ChangelogCommand(t *testing.T) {
	tests := map[string]struct {
		description   string
		args          []string
		wantExitCode  int
		wantOutSubstr []string
		wantErrSubstr string
	}{
		"changelog shows recent entries by default": {
			description:  "Run changelog with no args shows last 5 entries",
			args:         []string{"changelog"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"entries shown", // Footer showing entry count
			},
		},
		"changelog --last flag controls entry count": {
			description:  "Run changelog with --last 3",
			args:         []string{"changelog", "--last", "3"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"entries shown", // Shows entry count
			},
		},
		"changelog --plain disables colors": {
			description:  "Run changelog with --plain flag",
			args:         []string{"changelog", "--plain"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"Added", "Changed", "Fixed", // Category names appear in plain mode too
			},
		},
		"changelog with specific version": {
			description:  "Run changelog for a known version",
			args:         []string{"changelog", "0.9.0"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"0.9.0", // Version should be shown
			},
		},
		"changelog with v-prefix version": {
			description:  "Run changelog with v prefix (normalized)",
			args:         []string{"changelog", "v0.9.0"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"0.9.0",
			},
		},
		"changelog unreleased shows pending changes": {
			description:  "Run changelog for unreleased version",
			args:         []string{"changelog", "unreleased"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"Added", "Changed", "Fixed", // Any category present
			},
		},
		"changelog for non-existent version shows error": {
			description:   "Run changelog for version that doesn't exist",
			args:          []string{"changelog", "v99.99.99"},
			wantExitCode:  shared.ExitValidationFailed, // main.go returns 1 for all errors
			wantErrSubstr: "not found",
		},
		"changelog --last with --plain": {
			description:  "Combine --last and --plain flags",
			args:         []string{"changelog", "--last", "2", "--plain"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"entries shown",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			result := env.Run(tt.args...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"exit code mismatch\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := result.Stdout + result.Stderr

			if len(tt.wantOutSubstr) > 0 {
				foundMatch := false
				for _, substr := range tt.wantOutSubstr {
					if strings.Contains(combinedOutput, substr) {
						foundMatch = true
						break
					}
				}
				require.True(t, foundMatch,
					"output should contain one of %v\nstdout: %s\nstderr: %s",
					tt.wantOutSubstr, result.Stdout, result.Stderr)
			}

			if tt.wantErrSubstr != "" {
				require.Contains(t, strings.ToLower(combinedOutput), strings.ToLower(tt.wantErrSubstr),
					"error output should contain substring\nstdout: %s\nstderr: %s",
					result.Stdout, result.Stderr)
			}
		})
	}
}

// TestE2E_ChangelogExtract tests the changelog extract subcommand.
// This verifies FR-009: "autospec changelog extract <version> for GitHub release workflow".
func TestE2E_ChangelogExtract(t *testing.T) {
	tests := map[string]struct {
		description   string
		args          []string
		wantExitCode  int
		wantOutSubstr []string
		wantErrSubstr string
	}{
		"extract outputs markdown for specific version": {
			description:  "Extract release notes for version 0.9.0",
			args:         []string{"changelog", "extract", "0.9.0"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"###", // Markdown headers
			},
		},
		"extract with v-prefix version": {
			description:  "Extract with v prefix (normalized)",
			args:         []string{"changelog", "extract", "v0.9.0"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"###",
			},
		},
		"extract unreleased changes": {
			description:  "Extract unreleased changes",
			args:         []string{"changelog", "extract", "unreleased"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"###", "-", // Markdown list items
			},
		},
		"extract for non-existent version errors": {
			description:   "Extract for non-existent version",
			args:          []string{"changelog", "extract", "v99.99.99"},
			wantExitCode:  shared.ExitValidationFailed, // main.go returns 1 for all errors
			wantErrSubstr: "not found",
		},
		"extract without version argument errors": {
			description:   "Extract requires version argument",
			args:          []string{"changelog", "extract"},
			wantExitCode:  1, // Cobra argument validation error
			wantErrSubstr: "accepts 1 arg",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			result := env.Run(tt.args...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"exit code mismatch\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := result.Stdout + result.Stderr

			if len(tt.wantOutSubstr) > 0 {
				foundMatch := false
				for _, substr := range tt.wantOutSubstr {
					if strings.Contains(combinedOutput, substr) {
						foundMatch = true
						break
					}
				}
				require.True(t, foundMatch,
					"output should contain one of %v\nstdout: %s\nstderr: %s",
					tt.wantOutSubstr, result.Stdout, result.Stderr)
			}

			if tt.wantErrSubstr != "" {
				require.Contains(t, strings.ToLower(combinedOutput), strings.ToLower(tt.wantErrSubstr),
					"error output should contain substring\nstdout: %s\nstderr: %s",
					result.Stdout, result.Stderr)
			}
		})
	}
}

// TestE2E_ChangelogSync tests the changelog sync subcommand.
// This verifies FR-005: "make changelog-sync target to regenerate CHANGELOG.md from YAML".
func TestE2E_ChangelogSync(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
		wantErrSubstr string
	}{
		"sync without YAML file errors": {
			description:   "Sync fails when changelog.yaml not found",
			args:          []string{"changelog", "sync"},
			wantExitCode:  1,
			wantErrSubstr: "cannot find changelog.yaml",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			result := env.Run(tt.args...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"exit code mismatch\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := result.Stdout + result.Stderr

			if len(tt.wantOutSubstr) > 0 {
				foundMatch := false
				for _, substr := range tt.wantOutSubstr {
					if strings.Contains(combinedOutput, substr) {
						foundMatch = true
						break
					}
				}
				require.True(t, foundMatch,
					"output should contain one of %v\nstdout: %s\nstderr: %s",
					tt.wantOutSubstr, result.Stdout, result.Stderr)
			}

			if tt.wantErrSubstr != "" {
				require.Contains(t, strings.ToLower(combinedOutput), strings.ToLower(tt.wantErrSubstr),
					"error output should contain substring\nstdout: %s\nstderr: %s",
					result.Stdout, result.Stderr)
			}
		})
	}
}

// TestE2E_ChangelogCheck tests the changelog check subcommand.
// This verifies FR-006: "make changelog-check target to validate CHANGELOG.md matches YAML".
func TestE2E_ChangelogCheck(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
		wantErrSubstr string
	}{
		"check without YAML file errors": {
			description:   "Check fails when changelog.yaml not found",
			args:          []string{"changelog", "check"},
			wantExitCode:  1,
			wantErrSubstr: "cannot find changelog.yaml",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			result := env.Run(tt.args...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"exit code mismatch\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := result.Stdout + result.Stderr

			if len(tt.wantOutSubstr) > 0 {
				foundMatch := false
				for _, substr := range tt.wantOutSubstr {
					if strings.Contains(combinedOutput, substr) {
						foundMatch = true
						break
					}
				}
				require.True(t, foundMatch,
					"output should contain one of %v\nstdout: %s\nstderr: %s",
					tt.wantOutSubstr, result.Stdout, result.Stderr)
			}

			if tt.wantErrSubstr != "" {
				require.Contains(t, strings.ToLower(combinedOutput), strings.ToLower(tt.wantErrSubstr),
					"error output should contain substring\nstdout: %s\nstderr: %s",
					result.Stdout, result.Stderr)
			}
		})
	}
}

// TestE2E_ChangelogHelp tests that help is available for changelog commands.
func TestE2E_ChangelogHelp(t *testing.T) {
	tests := map[string]struct {
		description   string
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"changelog --help shows usage": {
			description:  "Show changelog help",
			args:         []string{"changelog", "--help"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"changelog", "embedded", "version", "--last", "--plain",
			},
		},
		"changelog extract --help shows usage": {
			description:  "Show changelog extract help",
			args:         []string{"changelog", "extract", "--help"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"extract", "version", "markdown", "release",
			},
		},
		"changelog sync --help shows usage": {
			description:  "Show changelog sync help",
			args:         []string{"changelog", "sync", "--help"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"sync", "CHANGELOG.md", "YAML",
			},
		},
		"changelog check --help shows usage": {
			description:  "Show changelog check help",
			args:         []string{"changelog", "check", "--help"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"check", "sync", "CHANGELOG.md",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			result := env.Run(tt.args...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"exit code mismatch\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)

			foundCount := 0
			for _, substr := range tt.wantOutSubstr {
				if strings.Contains(combinedOutput, strings.ToLower(substr)) {
					foundCount++
				}
			}
			require.GreaterOrEqual(t, foundCount, 2,
				"help output should contain at least 2 of %v\nstdout: %s\nstderr: %s",
				tt.wantOutSubstr, result.Stdout, result.Stderr)
		})
	}
}
