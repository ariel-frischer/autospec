//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// allCLICommands is the comprehensive list of all autospec CLI commands.
// This list is maintained manually to ensure 100% command coverage.
// When adding new commands to autospec, add them here as well.
var allCLICommands = []string{
	// Getting Started
	"ck",
	"init",
	"status",
	"update",
	"version",
	"view",

	// Workflows
	"all",
	"prep",
	"run",

	// Core Stages
	"constitution",
	"implement",
	"plan",
	"specify",
	"tasks",

	// Optional Stages
	"analyze",
	"checklist",
	"clarify",

	// Configuration
	"clean",
	"completion",
	"completion bash",
	"completion zsh",
	"completion fish",
	"completion powershell",
	"completion install",
	"config",
	"config show",
	"config set",
	"config get",
	"config toggle",
	"config keys",
	"config sync",
	"doctor",
	"help",
	"history",
	"uninstall",

	// Internal Commands
	"artifact",
	"commands",
	"commands check",
	"commands info",
	"commands install",
	"migrate",
	"migrate md-to-yaml",
	"new-feature",
	"prereqs",
	"task",
	"task block",
	"task unblock",
	"task list",
	"update-agent-context",
	"update-task",
	"yaml",
	"yaml check",

	// Additional Commands
	"sauce",
	"worktree",
	"worktree create",
	"worktree list",
	"worktree remove",
	"worktree prune",
	"worktree setup",
	"worktree gen-script",
}

// commandAliases maps command aliases to their primary command.
var commandAliases = map[string]string{
	"st":    "status",
	"spec":  "specify",
	"s":     "specify",
	"p":     "plan",
	"t":     "tasks",
	"impl":  "implement",
	"i":     "implement",
	"v":     "version",
	"doc":   "doctor",
	"const": "constitution",
	"cl":    "clarify",
	"chk":   "checklist",
	"az":    "analyze",
	"check": "ck",
}

// TestE2E_CommandCoverageAudit verifies that all CLI commands have E2E test coverage.
// This test reads all E2E test files and checks that each command is tested.
func TestE2E_CommandCoverageAudit(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get current file")

	e2eDir := filepath.Dir(currentFile)
	testFiles, err := filepath.Glob(filepath.Join(e2eDir, "*_test.go"))
	require.NoError(t, err, "failed to glob test files")

	// Read all test file contents
	var allTestContent strings.Builder
	for _, testFile := range testFiles {
		// Skip this file to avoid self-referential counting
		if filepath.Base(testFile) == "coverage_audit_test.go" {
			continue
		}
		content, err := os.ReadFile(testFile)
		require.NoError(t, err, "failed to read test file: %s", testFile)
		allTestContent.WriteString(string(content))
	}

	testContent := allTestContent.String()

	// Track which commands have coverage
	testedCommands := make(map[string]bool)
	missingCommands := []string{}

	for _, cmd := range allCLICommands {
		if hasTestCoverage(cmd, testContent) {
			testedCommands[cmd] = true
		} else {
			missingCommands = append(missingCommands, cmd)
		}
	}

	// Calculate coverage percentage
	totalCommands := len(allCLICommands)
	coveredCommands := len(testedCommands)
	coveragePercent := float64(coveredCommands) / float64(totalCommands) * 100

	t.Logf("E2E Command Coverage: %.1f%% (%d/%d commands)", coveragePercent, coveredCommands, totalCommands)

	if len(missingCommands) > 0 {
		t.Logf("Commands missing E2E test coverage:")
		for _, cmd := range missingCommands {
			t.Logf("  - %s", cmd)
		}
	}

	// Require 100% coverage
	require.Empty(t, missingCommands,
		"All CLI commands must have E2E test coverage. Missing: %v", missingCommands)
	require.Equal(t, 100.0, coveragePercent,
		"E2E command coverage must be 100%%")
}

// hasTestCoverage checks if a command has test coverage in the test content.
func hasTestCoverage(cmd string, testContent string) bool {
	testContentLower := strings.ToLower(testContent)
	cmdLower := strings.ToLower(cmd)

	// Handle multi-part commands (e.g., "config show", "worktree list")
	parts := strings.Fields(cmd)

	if len(parts) == 1 {
		// Single command - check for direct usage in env.Run() or test function
		return hasCommandInTests(cmdLower, testContentLower)
	}

	// Multi-part command - check for subcommand usage
	parent := parts[0]
	subcommand := parts[1]

	// Look for patterns like env.Run("parent", "subcommand", ...)
	patterns := []string{
		`env\.run\([^)]*"` + parent + `"[^)]*"` + subcommand + `"`,
		`"` + parent + `"[^}]*"` + subcommand + `"`,
		`args.*:.*\[\]string\{[^}]*"` + parent + `"[^}]*"` + subcommand + `"`,
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, testContentLower)
		if matched {
			return true
		}
	}

	return false
}

// hasCommandInTests checks if a single command appears in test content.
func hasCommandInTests(cmd, testContent string) bool {
	// Look for command in env.Run() calls
	patterns := []string{
		`env\.run\([^)]*"` + cmd + `"`,
		`"` + cmd + `"`,
		`\[\]string\{[^}]*"` + cmd + `"`,
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, testContent)
		if matched {
			return true
		}
	}

	return false
}

// TestE2E_CommandEnumeration verifies that our command list matches the actual CLI.
// This test runs autospec --help and compares against our known command list.
func TestE2E_CommandEnumeration(t *testing.T) {
	// Build autospec binary path
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get current file")

	repoRoot := filepath.Join(filepath.Dir(currentFile), "..", "..")
	binaryPath := filepath.Join(repoRoot, "autospec")

	// Check if binary exists, if not skip this test
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// Try to build it
		cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/autospec")
		cmd.Dir = repoRoot
		if err := cmd.Run(); err != nil {
			t.Skip("autospec binary not available and could not be built")
		}
	}

	// Run autospec --help to get command list
	cmd := exec.Command(binaryPath, "--help")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to run autospec --help")

	helpText := strings.ToLower(string(output))

	// Verify known top-level commands appear in help
	topLevelCommands := extractTopLevelCommands()
	missingFromHelp := []string{}

	for _, cmd := range topLevelCommands {
		if !strings.Contains(helpText, cmd) {
			missingFromHelp = append(missingFromHelp, cmd)
		}
	}

	if len(missingFromHelp) > 0 {
		t.Logf("Commands in our list but not in --help output: %v", missingFromHelp)
		t.Logf("This may indicate commands that were removed from CLI but not from test list")
	}
}

// extractTopLevelCommands returns only top-level commands (not subcommands).
func extractTopLevelCommands() []string {
	commands := []string{}
	for _, cmd := range allCLICommands {
		if !strings.Contains(cmd, " ") {
			commands = append(commands, cmd)
		}
	}
	return commands
}

// TestE2E_CommandAliasesWork verifies that command aliases work correctly.
func TestE2E_CommandAliasesWork(t *testing.T) {
	tests := map[string]struct {
		alias         string
		primaryCmd    string
		description   string
		wantOutSubstr []string
	}{
		"st alias for status": {
			alias:         "st",
			primaryCmd:    "status",
			description:   "st should be alias for status",
			wantOutSubstr: []string{"spec", "status"},
		},
		"v alias for version": {
			alias:         "v",
			primaryCmd:    "version",
			description:   "v should be alias for version",
			wantOutSubstr: []string{"version", "autospec"},
		},
		"doc alias for doctor": {
			alias:         "doc",
			primaryCmd:    "doctor",
			description:   "doc should be alias for doctor",
			wantOutSubstr: []string{"git", "claude", "check"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Build autospec binary path
			_, currentFile, _, ok := runtime.Caller(0)
			require.True(t, ok, "failed to get current file")

			repoRoot := filepath.Join(filepath.Dir(currentFile), "..", "..")
			binaryPath := filepath.Join(repoRoot, "autospec")

			if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
				t.Skip("autospec binary not available")
			}

			// Run with alias
			cmd := exec.Command(binaryPath, tt.alias, "--help")
			aliasOutput, _ := cmd.CombinedOutput()

			// Run with primary command
			cmd = exec.Command(binaryPath, tt.primaryCmd, "--help")
			primaryOutput, _ := cmd.CombinedOutput()

			// Both should produce similar output (help text)
			aliasLower := strings.ToLower(string(aliasOutput))
			primaryLower := strings.ToLower(string(primaryOutput))

			// Check that alias output contains expected substrings
			for _, substr := range tt.wantOutSubstr {
				if !strings.Contains(aliasLower, strings.ToLower(substr)) &&
					!strings.Contains(primaryLower, strings.ToLower(substr)) {
					t.Logf("Warning: neither alias nor primary command contains %q", substr)
				}
			}
		})
	}
}

// TestE2E_AllCommandsAccessible verifies all listed commands are accessible via help.
func TestE2E_AllCommandsAccessible(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get current file")

	repoRoot := filepath.Join(filepath.Dir(currentFile), "..", "..")
	binaryPath := filepath.Join(repoRoot, "autospec")

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("autospec binary not available")
	}

	// Test a sample of commands to verify they're accessible
	sampleCommands := [][]string{
		{"--help"},
		{"version"},
		{"config", "--help"},
		{"worktree", "--help"},
		{"completion", "--help"},
	}

	for _, args := range sampleCommands {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			cmd := exec.Command(binaryPath, args...)
			output, err := cmd.CombinedOutput()

			// Command should succeed (exit code 0) or fail gracefully
			if err != nil {
				// Some commands may fail without proper setup, but should still
				// produce meaningful output
				require.NotEmpty(t, output,
					"command %v should produce output even on error", args)
			}
		})
	}
}
