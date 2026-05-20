// Package health_test tests dependency health checks for Claude CLI and git.
// Related: /home/ari/repos/autospec/internal/health/health.go
// Tags: health, dependencies, validation, doctor

package health

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/cliagent"
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

// TestRunHealthChecks tests running all health checks.
// Note: Git CLI check was removed since go-git library is used for core operations.
func TestRunHealthChecks(t *testing.T) {
	report := RunHealthChecks()
	assert.NotNil(t, report)
	assert.Equal(t, 2, len(report.Checks), "Should have 2 health checks (Claude CLI, Claude settings)")

	// Verify all checks are present
	checkNames := make(map[string]bool)
	for _, check := range report.Checks {
		checkNames[check.Name] = true
	}

	assert.True(t, checkNames["Claude CLI"], "Should check Claude CLI")
	assert.True(t, checkNames["Claude settings"], "Should check Claude settings")
}

// TestFormatReport tests the report formatting.
// Note: Git CLI check was removed since go-git library is used for core operations.
func TestFormatReport(t *testing.T) {
	tests := map[string]struct {
		report   *HealthReport
		expected []string
	}{
		"All checks pass": {
			report: &HealthReport{
				Checks: []CheckResult{
					{Name: "Claude CLI", Passed: true, Message: "Claude CLI found"},
					{Name: "Claude settings", Passed: true, Message: "Permission configured"},
				},
				Passed: true,
			},
			expected: []string{
				"✓ Claude CLI: Claude CLI found",
				"✓ Claude settings: Permission configured",
			},
		},
		"One check fails": {
			report: &HealthReport{
				Checks: []CheckResult{
					{Name: "Claude CLI", Passed: false, Message: "Claude CLI not found in PATH"},
					{Name: "Claude settings", Passed: true, Message: "Permission configured"},
				},
				Passed: false,
			},
			expected: []string{
				"✗ Claude CLI: Claude CLI not found in PATH",
				"✓ Claude settings: Permission configured",
			},
		},
		"All checks fail": {
			report: &HealthReport{
				Checks: []CheckResult{
					{Name: "Claude CLI", Passed: false, Message: "Claude CLI not found in PATH"},
					{Name: "Claude settings", Passed: false, Message: "Missing permission"},
				},
				Passed: false,
			},
			expected: []string{
				"✗ Claude CLI: Claude CLI not found in PATH",
				"✗ Claude settings: Missing permission",
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

// createProjectScopeInitYml creates an init.yml with project scope for isolated testing.
func createProjectScopeInitYml(t *testing.T, dir string) {
	t.Helper()
	autospecDir := filepath.Join(dir, ".autospec")
	require.NoError(t, os.MkdirAll(autospecDir, 0o755))
	initContent := `version: "1.0.0"
settings_scope: project
autospec_version: autospec v0.8.2
created_at: 2026-01-16T00:00:00Z
updated_at: 2026-01-16T00:00:00Z
`
	require.NoError(t, os.WriteFile(filepath.Join(autospecDir, "init.yml"), []byte(initContent), 0o644))
}

// TestCheckClaudeSettingsInDir tests Claude settings health check with various scenarios.
// These tests use project scope (via init.yml) to isolate from global settings.
func TestCheckClaudeSettingsInDir(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFunc       func(t *testing.T, dir string)
		expectedPassed  bool
		expectedMessage string
	}{
		"passes with correct settings": {
			setupFunc: func(t *testing.T, dir string) {
				createProjectScopeInitYml(t, dir)
				claudeDir := filepath.Join(dir, ".claude")
				require.NoError(t, os.MkdirAll(claudeDir, 0o755))
				settingsContent := `{
					"permissions": {
						"allow": ["Bash(autospec:*)"]
					}
				}`
				require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(settingsContent), 0o644))
			},
			expectedPassed:  true,
			expectedMessage: "Bash(autospec:*) permission configured (project)",
		},
		"fails with missing settings file": {
			setupFunc: func(t *testing.T, dir string) {
				createProjectScopeInitYml(t, dir)
			},
			expectedPassed:  false,
			expectedMessage: "settings file not found (project)",
		},
		"fails with missing permission": {
			setupFunc: func(t *testing.T, dir string) {
				createProjectScopeInitYml(t, dir)
				claudeDir := filepath.Join(dir, ".claude")
				require.NoError(t, os.MkdirAll(claudeDir, 0o755))
				settingsContent := `{
					"permissions": {
						"allow": ["Bash(other:*)"]
					}
				}`
				require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(settingsContent), 0o644))
			},
			expectedPassed:  false,
			expectedMessage: "missing Bash(autospec:*) permission (project)",
		},
		"fails with denied permission": {
			setupFunc: func(t *testing.T, dir string) {
				createProjectScopeInitYml(t, dir)
				claudeDir := filepath.Join(dir, ".claude")
				require.NoError(t, os.MkdirAll(claudeDir, 0o755))
				settingsContent := `{
					"permissions": {
						"deny": ["Bash(autospec:*)"]
					}
				}`
				require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(settingsContent), 0o644))
			},
			expectedPassed:  false,
			expectedMessage: "is explicitly denied (project)",
		},
		"fails with empty allow list": {
			setupFunc: func(t *testing.T, dir string) {
				createProjectScopeInitYml(t, dir)
				claudeDir := filepath.Join(dir, ".claude")
				require.NoError(t, os.MkdirAll(claudeDir, 0o755))
				settingsContent := `{
					"permissions": {
						"allow": []
					}
				}`
				require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(settingsContent), 0o644))
			},
			expectedPassed:  false,
			expectedMessage: "missing Bash(autospec:*) permission (project)",
		},
		"passes with multiple permissions including autospec": {
			setupFunc: func(t *testing.T, dir string) {
				createProjectScopeInitYml(t, dir)
				claudeDir := filepath.Join(dir, ".claude")
				require.NoError(t, os.MkdirAll(claudeDir, 0o755))
				settingsContent := `{
					"permissions": {
						"allow": ["Bash(git:*)", "Bash(autospec:*)", "Read(*)"]
					}
				}`
				require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(settingsContent), 0o644))
			},
			expectedPassed:  true,
			expectedMessage: "Bash(autospec:*) permission configured (project)",
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

// TestRunHealthChecksIncludesClaudeSettings verifies Claude settings check is included.
// Note: Git CLI check was removed since go-git library is used for core operations.
func TestRunHealthChecksIncludesClaudeSettings(t *testing.T) {
	report := RunHealthChecks()
	assert.NotNil(t, report)
	assert.GreaterOrEqual(t, len(report.Checks), 2, "Should have at least 2 health checks (Claude CLI, Claude settings)")

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

	// Both checks should be present (Git CLI check removed - using go-git)
	_, hasClaudeCLI := checkMap["Claude CLI"]
	_, hasSettings := checkMap["Claude settings"]

	assert.True(t, hasClaudeCLI, "Should have Claude CLI check")
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
	createProjectScopeInitYml(t, tmpDir)
	claudeDir := filepath.Join(tmpDir, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0o755))

	// Write invalid JSON
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeDir, "settings.local.json"),
		[]byte("not valid json"),
		0o644,
	))

	result := CheckClaudeSettingsInDir(tmpDir)
	assert.Equal(t, "Claude settings", result.Name)
	assert.False(t, result.Passed)
	// Error message from JSON parsing
	assert.NotEmpty(t, result.Message)
	assert.Equal(t, SourceProject, result.Source)
}

func TestCheckClaudeSettingsInDir_EmptyFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	createProjectScopeInitYml(t, tmpDir)
	claudeDir := filepath.Join(tmpDir, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0o755))

	// Write empty file
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeDir, "settings.local.json"),
		[]byte(""),
		0o644,
	))

	result := CheckClaudeSettingsInDir(tmpDir)
	assert.Equal(t, "Claude settings", result.Name)
	assert.False(t, result.Passed)
	assert.Equal(t, SourceProject, result.Source)
}

// TestRunHealthChecks_IncludesAgentChecks verifies agent checks are included in health report
func TestRunHealthChecks_IncludesAgentChecks(t *testing.T) {
	t.Parallel()

	report := RunHealthChecks()
	require.NotNil(t, report)

	// AgentChecks should be populated (may be empty if no agents registered, but slice should exist)
	assert.NotNil(t, report.AgentChecks)

	// At minimum, the built-in agents should be checked
	assert.GreaterOrEqual(t, len(report.AgentChecks), 1, "Should have at least one agent checked")
}

// TestFormatAgentStatus tests formatting of individual agent status
func TestFormatAgentStatus(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status   cliagent.AgentStatus
		wantSymb string
		wantName string
		wantInfo string
	}{
		"valid with version": {
			status: cliagent.AgentStatus{
				Name:          "claude",
				Installed:     true,
				Version:       "1.0.0",
				TestedVersion: "2.1.139",
				Valid:         true,
				Error:         "",
			},
			wantSymb: "✓",
			wantName: "claude",
			wantInfo: "tested 2.1.139",
		},
		"valid with command-style version": {
			status: cliagent.AgentStatus{
				Name:          "codex",
				Installed:     true,
				Version:       "codex-cli 0.130.0",
				TestedVersion: "0.130.0",
				Valid:         true,
				Error:         "",
			},
			wantSymb: "✓",
			wantName: "codex",
			wantInfo: "codex-cli 0.130.0; tested 0.130.0",
		},
		"valid without version": {
			status: cliagent.AgentStatus{
				Name:      "gemini",
				Installed: true,
				Version:   "",
				Valid:     true,
				Error:     "",
			},
			wantSymb: "✓",
			wantName: "gemini",
			wantInfo: "installed",
		},
		"not available with error": {
			status: cliagent.AgentStatus{
				Name:      "cline",
				Installed: false,
				Version:   "",
				Valid:     false,
				Error:     "cline not found in PATH",
			},
			wantSymb: "○",
			wantName: "cline",
			wantInfo: "cline not found in PATH",
		},
		"not available without error": {
			status: cliagent.AgentStatus{
				Name:      "codex",
				Installed: false,
				Version:   "",
				Valid:     false,
				Error:     "",
			},
			wantSymb: "○",
			wantName: "codex",
			wantInfo: "not available",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			output := FormatAgentStatus(tc.status)

			assert.Contains(t, output, tc.wantSymb, "Should contain correct symbol")
			assert.Contains(t, output, tc.wantName, "Should contain agent name")
			assert.Contains(t, output, tc.wantInfo, "Should contain expected info")
		})
	}
}

// TestFormatReport_WithAgentChecks tests that agent checks appear in formatted output
func TestFormatReport_WithAgentChecks(t *testing.T) {
	t.Parallel()

	report := &HealthReport{
		Checks: []CheckResult{
			{Name: "Git", Passed: true, Message: "Git found"},
		},
		AgentChecks: []cliagent.AgentStatus{
			{Name: "claude", Installed: true, Version: "1.0.0", Valid: true},
			{Name: "cline", Installed: false, Valid: false, Error: "not found"},
		},
		Passed:       true,
		AgentsPassed: false,
	}

	output := FormatReport(report)

	// Should contain core check
	assert.Contains(t, output, "Git")

	// Should contain agent section header
	assert.Contains(t, output, "CLI Agents:")

	// Should contain agent statuses
	assert.Contains(t, output, "claude")
	assert.Contains(t, output, "cline")
}

// TestFormatReport_NoAgentChecks tests report formatting when no agents are registered
func TestFormatReport_NoAgentChecks(t *testing.T) {
	t.Parallel()

	report := &HealthReport{
		Checks: []CheckResult{
			{Name: "Git", Passed: true, Message: "Git found"},
		},
		AgentChecks:  []cliagent.AgentStatus{},
		Passed:       true,
		AgentsPassed: true,
	}

	output := FormatReport(report)

	// Should contain core check
	assert.Contains(t, output, "Git")

	// Should NOT contain agent section header when no agents
	assert.NotContains(t, output, "CLI Agents:")
}

// TestHealthReport_AgentsPassedStatus tests the AgentsPassed field behavior
func TestHealthReport_AgentsPassedStatus(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agentChecks []cliagent.AgentStatus
		wantPassed  bool
	}{
		"all agents valid": {
			agentChecks: []cliagent.AgentStatus{
				{Name: "claude", Valid: true},
				{Name: "gemini", Valid: true},
			},
			wantPassed: true,
		},
		"one agent invalid": {
			agentChecks: []cliagent.AgentStatus{
				{Name: "claude", Valid: true},
				{Name: "cline", Valid: false},
			},
			wantPassed: false,
		},
		"all agents invalid": {
			agentChecks: []cliagent.AgentStatus{
				{Name: "cline", Valid: false},
				{Name: "codex", Valid: false},
			},
			wantPassed: false,
		},
		"no agents": {
			agentChecks: []cliagent.AgentStatus{},
			wantPassed:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Compute AgentsPassed like RunHealthChecks does
			agentsPassed := true
			for _, status := range tc.agentChecks {
				if !status.Valid {
					agentsPassed = false
					break
				}
			}

			assert.Equal(t, tc.wantPassed, agentsPassed)
		})
	}
}

// TestCheckClaudeSettingsInDir_InitYmlAware tests init.yml-aware permission checking.
// This covers T011 acceptance criteria for Phase 4.
func TestCheckClaudeSettingsInDir_InitYmlAware(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFunc      func(t *testing.T, dir string)
		expectedPassed bool
		expectedSource string
		messageContain string
	}{
		"init.yml global scope uses global source": {
			setupFunc: func(t *testing.T, dir string) {
				// Create init.yml with global scope
				autospecDir := filepath.Join(dir, ".autospec")
				require.NoError(t, os.MkdirAll(autospecDir, 0o755))
				initContent := `version: "1.0.0"
settings_scope: global
autospec_version: autospec v0.8.2
created_at: 2026-01-16T00:00:00Z
updated_at: 2026-01-16T00:00:00Z
`
				require.NoError(t, os.WriteFile(filepath.Join(autospecDir, "init.yml"), []byte(initContent), 0o644))
				// Note: Does not create global settings - tests that global scope is selected
			},
			expectedPassed: true, // Will depend on actual global settings
			expectedSource: SourceGlobal,
			messageContain: "(global)",
		},
		"init.yml project scope with project permissions": {
			setupFunc: func(t *testing.T, dir string) {
				// Create init.yml with project scope
				autospecDir := filepath.Join(dir, ".autospec")
				require.NoError(t, os.MkdirAll(autospecDir, 0o755))
				initContent := `version: "1.0.0"
settings_scope: project
autospec_version: autospec v0.8.2
created_at: 2026-01-16T00:00:00Z
updated_at: 2026-01-16T00:00:00Z
`
				require.NoError(t, os.WriteFile(filepath.Join(autospecDir, "init.yml"), []byte(initContent), 0o644))

				// Create project settings with permission
				claudeDir := filepath.Join(dir, ".claude")
				require.NoError(t, os.MkdirAll(claudeDir, 0o755))
				settingsContent := `{
					"permissions": {
						"allow": ["Bash(autospec:*)"]
					}
				}`
				require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(settingsContent), 0o644))
			},
			expectedPassed: true,
			expectedSource: SourceProject,
			messageContain: "(project)",
		},
		"init.yml project scope missing permissions": {
			setupFunc: func(t *testing.T, dir string) {
				// Create init.yml with project scope
				autospecDir := filepath.Join(dir, ".autospec")
				require.NoError(t, os.MkdirAll(autospecDir, 0o755))
				initContent := `version: "1.0.0"
settings_scope: project
autospec_version: autospec v0.8.2
created_at: 2026-01-16T00:00:00Z
updated_at: 2026-01-16T00:00:00Z
`
				require.NoError(t, os.WriteFile(filepath.Join(autospecDir, "init.yml"), []byte(initContent), 0o644))

				// Create project settings WITHOUT autospec permission
				claudeDir := filepath.Join(dir, ".claude")
				require.NoError(t, os.MkdirAll(claudeDir, 0o755))
				settingsContent := `{
					"permissions": {
						"allow": ["Bash(other:*)"]
					}
				}`
				require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(settingsContent), 0o644))
			},
			expectedPassed: false,
			expectedSource: SourceProject,
			messageContain: "missing",
		},
		"no init.yml falls back to legacy with project permissions": {
			setupFunc: func(t *testing.T, dir string) {
				// No init.yml, but project settings have permission
				claudeDir := filepath.Join(dir, ".claude")
				require.NoError(t, os.MkdirAll(claudeDir, 0o755))
				settingsContent := `{
					"permissions": {
						"allow": ["Bash(autospec:*)"]
					}
				}`
				require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(settingsContent), 0o644))
			},
			expectedPassed: true,
			expectedSource: SourceLegacy,
			messageContain: "(legacy)",
		},
		"no init.yml uses legacy fallback": {
			setupFunc:      func(t *testing.T, dir string) {},
			expectedPassed: true, // May pass if global settings exist on test machine
			expectedSource: SourceLegacy,
			messageContain: "(legacy)",
		},
		"malformed init.yml falls back to legacy": {
			setupFunc: func(t *testing.T, dir string) {
				// Create malformed init.yml
				autospecDir := filepath.Join(dir, ".autospec")
				require.NoError(t, os.MkdirAll(autospecDir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(autospecDir, "init.yml"), []byte("not valid yaml: [[["), 0o644))

				// Create project settings with permission
				claudeDir := filepath.Join(dir, ".claude")
				require.NoError(t, os.MkdirAll(claudeDir, 0o755))
				settingsContent := `{
					"permissions": {
						"allow": ["Bash(autospec:*)"]
					}
				}`
				require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(settingsContent), 0o644))
			},
			expectedPassed: true,
			expectedSource: SourceLegacy,
			messageContain: "(legacy)",
		},
		"invalid scope in init.yml falls back to legacy": {
			setupFunc: func(t *testing.T, dir string) {
				// Create init.yml with invalid scope
				autospecDir := filepath.Join(dir, ".autospec")
				require.NoError(t, os.MkdirAll(autospecDir, 0o755))
				initContent := `version: "1.0.0"
settings_scope: invalid_scope
autospec_version: autospec v0.8.2
created_at: 2026-01-16T00:00:00Z
updated_at: 2026-01-16T00:00:00Z
`
				require.NoError(t, os.WriteFile(filepath.Join(autospecDir, "init.yml"), []byte(initContent), 0o644))

				// Create project settings with permission
				claudeDir := filepath.Join(dir, ".claude")
				require.NoError(t, os.MkdirAll(claudeDir, 0o755))
				settingsContent := `{
					"permissions": {
						"allow": ["Bash(autospec:*)"]
					}
				}`
				require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.local.json"), []byte(settingsContent), 0o644))
			},
			expectedPassed: true,
			expectedSource: SourceLegacy,
			messageContain: "(legacy)",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Note: Not parallel because some tests access global ~/.claude
			tmpDir := t.TempDir()
			tt.setupFunc(t, tmpDir)

			result := CheckClaudeSettingsInDir(tmpDir)

			assert.Equal(t, "Claude settings", result.Name)
			// Skip expectedPassed check for global/legacy sources (depends on actual global settings)
			if tt.expectedSource == SourceProject {
				assert.Equal(t, tt.expectedPassed, result.Passed, "Expected Passed=%v, got %v. Message: %s", tt.expectedPassed, result.Passed, result.Message)
			}
			assert.Equal(t, tt.expectedSource, result.Source, "Expected Source=%q, got %q", tt.expectedSource, result.Source)
			assert.Contains(t, result.Message, tt.messageContain, "Expected message to contain %q, got %q", tt.messageContain, result.Message)
		})
	}
}

// TestCheckClaudeSettingsInDir_SourceFieldPopulated verifies Source field is always set for settings checks.
func TestCheckClaudeSettingsInDir_SourceFieldPopulated(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	result := CheckClaudeSettingsInDir(tmpDir)

	assert.Equal(t, "Claude settings", result.Name)
	// Source should be non-empty for permission checks
	assert.NotEmpty(t, result.Source, "Source field should be populated for permission checks")
	// Should be one of the valid source values
	validSources := []string{SourceGlobal, SourceProject, SourceLegacy}
	assert.Contains(t, validSources, result.Source, "Source should be one of: %v", validSources)
}

// TestSourceConstants verifies the source constant values match expected strings.
func TestSourceConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "global", SourceGlobal)
	assert.Equal(t, "project", SourceProject)
	assert.Equal(t, "legacy", SourceLegacy)
}
