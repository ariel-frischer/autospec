// Package health provides dependency health checks for autospec. It validates that
// required external tools (Claude CLI, Git) are available and properly configured,
// returning structured reports used by the 'autospec doctor' command.
package health

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ariel-frischer/autospec/internal/claude"
)

// CheckResult represents the result of a single health check
type CheckResult struct {
	Name    string
	Passed  bool
	Message string
}

// HealthReport contains all health check results
type HealthReport struct {
	Checks []CheckResult
	Passed bool
}

// RunHealthChecks runs all health checks and returns a report
func RunHealthChecks() *HealthReport {
	report := &HealthReport{
		Checks: make([]CheckResult, 0),
		Passed: true,
	}

	// Check Claude CLI
	claudeCheck := CheckClaudeCLI()
	report.Checks = append(report.Checks, claudeCheck)
	if !claudeCheck.Passed {
		report.Passed = false
	}

	// Check Git
	gitCheck := CheckGit()
	report.Checks = append(report.Checks, gitCheck)
	if !gitCheck.Passed {
		report.Passed = false
	}

	// Check Claude settings
	settingsCheck := CheckClaudeSettings()
	report.Checks = append(report.Checks, settingsCheck)
	if !settingsCheck.Passed {
		report.Passed = false
	}

	return report
}

// CheckClaudeCLI checks if the Claude CLI is available
func CheckClaudeCLI() CheckResult {
	_, err := exec.LookPath("claude")
	if err != nil {
		return CheckResult{
			Name:    "Claude CLI",
			Passed:  false,
			Message: "Claude CLI not found in PATH",
		}
	}

	return CheckResult{
		Name:    "Claude CLI",
		Passed:  true,
		Message: "Claude CLI found",
	}
}

// CheckGit checks if Git is available
func CheckGit() CheckResult {
	_, err := exec.LookPath("git")
	if err != nil {
		return CheckResult{
			Name:    "Git",
			Passed:  false,
			Message: "Git not found in PATH",
		}
	}

	return CheckResult{
		Name:    "Git",
		Passed:  true,
		Message: "Git found",
	}
}

// FormatReport formats the health report for console output
func FormatReport(report *HealthReport) string {
	var output string

	for _, check := range report.Checks {
		if check.Passed {
			output += fmt.Sprintf("✓ %s: %s\n", check.Name, check.Message)
		} else {
			output += fmt.Sprintf("✗ %s: %s\n", check.Name, check.Message)
		}
	}

	return output
}

// CheckClaudeSettings validates Claude Code settings configuration.
// Returns a health check result indicating whether the required permissions are configured.
func CheckClaudeSettings() CheckResult {
	cwd, err := os.Getwd()
	if err != nil {
		return CheckResult{
			Name:    "Claude settings",
			Passed:  false,
			Message: fmt.Sprintf("failed to get current directory: %v", err),
		}
	}

	return CheckClaudeSettingsInDir(cwd)
}

// CheckClaudeSettingsInDir validates Claude settings in the specified directory.
func CheckClaudeSettingsInDir(projectDir string) CheckResult {
	checkResult, err := claude.CheckInDir(projectDir)
	if err != nil {
		return CheckResult{
			Name:    "Claude settings",
			Passed:  false,
			Message: err.Error(),
		}
	}

	return formatClaudeCheckResult(checkResult)
}

// formatClaudeCheckResult converts a claude.SettingsCheckResult to a health.CheckResult.
func formatClaudeCheckResult(result claude.SettingsCheckResult) CheckResult {
	switch result.Status {
	case claude.StatusConfigured:
		return CheckResult{
			Name:    "Claude settings",
			Passed:  true,
			Message: fmt.Sprintf("%s permission configured", claude.RequiredPermission),
		}
	case claude.StatusMissing:
		return CheckResult{
			Name:    "Claude settings",
			Passed:  false,
			Message: ".claude/settings.local.json not found (run 'autospec init' to configure)",
		}
	case claude.StatusNeedsPermission:
		return CheckResult{
			Name:    "Claude settings",
			Passed:  false,
			Message: fmt.Sprintf("missing %s permission (run 'autospec init' to fix)", claude.RequiredPermission),
		}
	case claude.StatusDenied:
		return CheckResult{
			Name:    "Claude settings",
			Passed:  false,
			Message: fmt.Sprintf("%s is explicitly denied. Remove from permissions.deny in %s to allow autospec commands.", claude.RequiredPermission, result.FilePath),
		}
	default:
		return CheckResult{
			Name:    "Claude settings",
			Passed:  false,
			Message: "unknown Claude settings status",
		}
	}
}
