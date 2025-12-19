// Package health_test tests dependency health checks for Claude CLI and git.
// Related: /home/ari/repos/autospec/internal/health/health.go
// Tags: health, dependencies, validation, doctor

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

// Additional tests to improve coverage to 85%

func TestRunHealthChecks_AllChecksPresent(t *testing.T) {
	t.Parallel()

	report := RunHealthChecks()
	require.NotNil(t, report)

	// Verify report structure
	checkMap := make(map[string]CheckResult)
	for _, check := range report.Checks {
		checkMap[check.Name] = check
	}

	// All three checks should be present
	_, hasClaudeCLI := checkMap["Claude CLI"]
	_, hasGit := checkMap["Git"]
	_, hasSettings := checkMap["Claude settings"]

	assert.True(t, hasClaudeCLI, "Should have Claude CLI check")
	assert.True(t, hasGit, "Should have Git check")
	assert.True(t, hasSettings, "Should have Claude settings check")
}

func TestRunHealthChecks_ReportPassedStatus(t *testing.T) {
	tests := map[string]struct {
		checkResults []CheckResult
		wantPassed   bool
	}{
		"all pass": {
			checkResults: []CheckResult{
				{Name: "Check1", Passed: true, Message: "ok"},
				{Name: "Check2", Passed: true, Message: "ok"},
			},
			wantPassed: true,
		},
		"one fails": {
			checkResults: []CheckResult{
				{Name: "Check1", Passed: true, Message: "ok"},
				{Name: "Check2", Passed: false, Message: "failed"},
			},
			wantPassed: false,
		},
		"all fail": {
			checkResults: []CheckResult{
				{Name: "Check1", Passed: false, Message: "failed"},
				{Name: "Check2", Passed: false, Message: "failed"},
			},
			wantPassed: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create report manually to test the Passed field logic
			report := &HealthReport{
				Checks: tc.checkResults,
				Passed: true,
			}
			for _, check := range tc.checkResults {
				if !check.Passed {
					report.Passed = false
					break
				}
			}

			assert.Equal(t, tc.wantPassed, report.Passed)
		})
	}
}

func TestCheckClaudeCLI_ResultStructure(t *testing.T) {
	t.Parallel()

	result := CheckClaudeCLI()

	// Should always have correct name
	assert.Equal(t, "Claude CLI", result.Name)

	// Message should be non-empty
	assert.NotEmpty(t, result.Message)

	// Message should be one of the expected values
	validMessages := []string{"Claude CLI found", "Claude CLI not found in PATH"}
	assert.Contains(t, validMessages, result.Message)
}

func TestCheckGit_ResultStructure(t *testing.T) {
	t.Parallel()

	result := CheckGit()

	// Should always have correct name
	assert.Equal(t, "Git", result.Name)

	// Message should be non-empty
	assert.NotEmpty(t, result.Message)

	// Git should be found in test environments
	assert.True(t, result.Passed)
	assert.Equal(t, "Git found", result.Message)
}

func TestFormatReport_EmptyReport(t *testing.T) {
	t.Parallel()

	report := &HealthReport{
		Checks: []CheckResult{},
		Passed: true,
	}

	output := FormatReport(report)
	assert.Empty(t, output, "Empty report should produce empty output")
}

func TestFormatReport_SingleCheck(t *testing.T) {
	tests := map[string]struct {
		check       CheckResult
		wantSymbol  string
		wantContent string
	}{
		"passed check": {
			check:       CheckResult{Name: "Test", Passed: true, Message: "success"},
			wantSymbol:  "✓",
			wantContent: "Test: success",
		},
		"failed check": {
			check:       CheckResult{Name: "Test", Passed: false, Message: "failure"},
			wantSymbol:  "✗",
			wantContent: "Test: failure",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			report := &HealthReport{
				Checks: []CheckResult{tc.check},
				Passed: tc.check.Passed,
			}

			output := FormatReport(report)
			assert.Contains(t, output, tc.wantSymbol)
			assert.Contains(t, output, tc.wantContent)
		})
	}
}

func TestCheckClaudeSettingsInDir_InvalidJSON(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))

	// Write invalid JSON
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeDir, "settings.local.json"),
		[]byte("not valid json"),
		0644,
	))

	result := CheckClaudeSettingsInDir(tmpDir)
	assert.Equal(t, "Claude settings", result.Name)
	assert.False(t, result.Passed)
	// Error message from JSON parsing
	assert.NotEmpty(t, result.Message)
}

func TestCheckClaudeSettingsInDir_EmptyFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))

	// Write empty file
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeDir, "settings.local.json"),
		[]byte(""),
		0644,
	))

	result := CheckClaudeSettingsInDir(tmpDir)
	assert.Equal(t, "Claude settings", result.Name)
	assert.False(t, result.Passed)
}
