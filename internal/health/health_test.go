package health

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, 2, len(report.Checks), "Should have 2 health checks")

	// Verify all checks are present
	checkNames := make(map[string]bool)
	for _, check := range report.Checks {
		checkNames[check.Name] = true
	}

	assert.True(t, checkNames["Claude CLI"], "Should check Claude CLI")
	assert.True(t, checkNames["Git"], "Should check Git")
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
				"✓ Claude CLI found",
				"✓ Git found",
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
				"✗ Error: Claude CLI not found in PATH",
				"✓ Git found",
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
				"✗ Error: Claude CLI not found in PATH",
				"✗ Error: Git not found in PATH",
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
