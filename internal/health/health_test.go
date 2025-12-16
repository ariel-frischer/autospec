package health

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckClaudeCLI tests the Claude CLI health check
func TestCheckClaudeCLI(t *testing.T) {
	result := CheckClaudeCLI()
	assert.NotNil(t, result)
	assert.Equal(t, "Claude CLI", result.Name)
	// Note: This test will pass/fail based on whether claude is actually installed
	// In a real environment, claude should be available
}

// TestCheckGit tests the Git health check
func TestCheckGit(t *testing.T) {
	result := CheckGit()
	assert.NotNil(t, result)
	assert.Equal(t, "Git", result.Name)
	// Git should always be available in development environments
	assert.True(t, result.Passed, "Git should be installed")
	assert.Equal(t, "Git found", result.Message)
}

// TestRunHealthChecks tests running all health checks
func TestRunHealthChecks(t *testing.T) {
	report := RunHealthChecks()
	assert.NotNil(t, report)
	assert.Equal(t, 3, len(report.Checks), "Should have 3 health checks")

	// Verify all checks are present
	checkNames := make(map[string]bool)
	for _, check := range report.Checks {
		checkNames[check.Name] = true
	}

	assert.True(t, checkNames["Claude CLI"], "Should check Claude CLI")
	assert.True(t, checkNames["Git"], "Should check Git")
	assert.True(t, checkNames["Claude settings"], "Should check Claude settings")
}

// TestFormatReport tests the report formatting
func TestFormatReport(t *testing.T) {
	tests := map[string]struct {
		report   *HealthReport
		expected []string
	}{
		"All checks pass": {
			report: &HealthReport{
				Checks: []CheckResult{
					{Name: "Claude CLI", Passed: true, Message: "Claude CLI found"},
					{Name: "Git", Passed: true, Message: "Git found"},
				},
				Passed: true,
			},
			expected: []string{
				"✓ Claude CLI: Claude CLI found",
				"✓ Git: Git found",
			},
		},
		"One check fails": {
			report: &HealthReport{
				Checks: []CheckResult{
					{Name: "Claude CLI", Passed: false, Message: "Claude CLI not found in PATH"},
					{Name: "Git", Passed: true, Message: "Git found"},
				},
				Passed: false,
			},
			expected: []string{
				"✗ Claude CLI: Claude CLI not found in PATH",
				"✓ Git: Git found",
			},
		},
		"All checks fail": {
			report: &HealthReport{
				Checks: []CheckResult{
					{Name: "Claude CLI", Passed: false, Message: "Claude CLI not found in PATH"},
					{Name: "Git", Passed: false, Message: "Git not found in PATH"},
				},
				Passed: false,
			},
			expected: []string{
				"✗ Claude CLI: Claude CLI not found in PATH",
				"✗ Git: Git not found in PATH",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			output := FormatReport(tt.report)
			for _, expected := range tt.expected {
				assert.Contains(t, output, expected, "Output should contain: %s", expected)
			}
		})
	}
}

// TestFormatReportStructure tests the structure of formatted output
func TestFormatReportStructure(t *testing.T) {
	report := &HealthReport{
		Checks: []CheckResult{
			{Name: "Test 1", Passed: true, Message: "Test 1 passed"},
			{Name: "Test 2", Passed: false, Message: "Test 2 failed"},
		},
		Passed: false,
	}

	output := FormatReport(report)

	// Should have newlines
	assert.True(t, strings.Contains(output, "\n"), "Output should contain newlines")

	// Should have checkmarks for passed tests
	assert.True(t, strings.Contains(output, "✓"), "Output should contain checkmarks")

	// Should have error markers for failed tests
	assert.True(t, strings.Contains(output, "✗"), "Output should contain error markers")
}

// TestCheckClaudeSettingsInDir tests Claude settings health check with various scenarios
func TestCheckClaudeSettingsInDir(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFunc       func(t *testing.T, dir string)
		expectedPassed  bool
		expectedMessage string
	}{
		"passes with correct settings": {
			setupFunc: func(t *testing.T, dir string) {
				claudeDir := filepath.Join(dir, ".claude")
				require.NoError(t, os.MkdirAll(claudeDir, 0755))
				settingsContent := `{
					"permissions": {
						"allow": ["Bash(autospec:*)"]
					}
				}`
				require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(settingsContent), 0644))
			},
			expectedPassed:  true,
			expectedMessage: "Bash(autospec:*) permission configured",
		},
		"fails with missing settings file": {
			setupFunc:       func(t *testing.T, dir string) {},
			expectedPassed:  false,
			expectedMessage: ".claude/settings.local.json not found (run 'autospec init' to configure)",
		},
		"fails with missing permission": {
			setupFunc: func(t *testing.T, dir string) {
				claudeDir := filepath.Join(dir, ".claude")
				require.NoError(t, os.MkdirAll(claudeDir, 0755))
				settingsContent := `{
					"permissions": {
						"allow": ["Bash(other:*)"]
					}
				}`
				require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(settingsContent), 0644))
			},
			expectedPassed:  false,
			expectedMessage: "missing Bash(autospec:*) permission (run 'autospec init' to fix)",
		},
		"fails with denied permission": {
			setupFunc: func(t *testing.T, dir string) {
				claudeDir := filepath.Join(dir, ".claude")
				require.NoError(t, os.MkdirAll(claudeDir, 0755))
				settingsContent := `{
					"permissions": {
						"deny": ["Bash(autospec:*)"]
					}
				}`
				require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(settingsContent), 0644))
			},
			expectedPassed:  false,
			expectedMessage: "is explicitly denied",
		},
		"fails with empty allow list": {
			setupFunc: func(t *testing.T, dir string) {
				claudeDir := filepath.Join(dir, ".claude")
				require.NoError(t, os.MkdirAll(claudeDir, 0755))
				settingsContent := `{
					"permissions": {
						"allow": []
					}
				}`
				require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(settingsContent), 0644))
			},
			expectedPassed:  false,
			expectedMessage: "missing Bash(autospec:*) permission",
		},
		"passes with multiple permissions including autospec": {
			setupFunc: func(t *testing.T, dir string) {
				claudeDir := filepath.Join(dir, ".claude")
				require.NoError(t, os.MkdirAll(claudeDir, 0755))
				settingsContent := `{
					"permissions": {
						"allow": ["Bash(git:*)", "Bash(autospec:*)", "Read(*)"]
					}
				}`
				require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(settingsContent), 0644))
			},
			expectedPassed:  true,
			expectedMessage: "Bash(autospec:*) permission configured",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create temp directory for this test
			tmpDir := t.TempDir()
			tt.setupFunc(t, tmpDir)

			result := CheckClaudeSettingsInDir(tmpDir)

			assert.Equal(t, "Claude settings", result.Name)
			assert.Equal(t, tt.expectedPassed, result.Passed, "Expected Passed=%v, got %v", tt.expectedPassed, result.Passed)
			assert.Contains(t, result.Message, tt.expectedMessage, "Expected message to contain %q", tt.expectedMessage)
		})
	}
}

// TestRunHealthChecksIncludesClaudeSettings verifies Claude settings check is included
func TestRunHealthChecksIncludesClaudeSettings(t *testing.T) {
	report := RunHealthChecks()
	assert.NotNil(t, report)
	assert.GreaterOrEqual(t, len(report.Checks), 3, "Should have at least 3 health checks (Claude CLI, Git, Claude settings)")

	// Verify Claude settings check is present
	hasClaudeSettings := false
	for _, check := range report.Checks {
		if check.Name == "Claude settings" {
			hasClaudeSettings = true
			break
		}
	}
	assert.True(t, hasClaudeSettings, "Should include Claude settings check")
}
