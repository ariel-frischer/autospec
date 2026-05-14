// Package health provides dependency health checks for autospec. It validates that
// required external tools (Claude CLI) are available and properly configured,
// returning structured reports used by the 'autospec doctor' command.
// Note: Git CLI check was removed since go-git library is used for core operations.
package health

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ariel-frischer/autospec/internal/build"
	"github.com/ariel-frischer/autospec/internal/claude"
	"github.com/ariel-frischer/autospec/internal/cliagent"
	initpkg "github.com/ariel-frischer/autospec/internal/init"
)

// Source values for permission checks indicating where permissions were found.
const (
	SourceGlobal  = "global"
	SourceProject = "project"
	SourceLegacy  = "legacy"
)

// CheckResult represents the result of a single health check
type CheckResult struct {
	Name    string
	Passed  bool
	Message string
	// Source indicates where permissions were found for permission checks.
	// Values: "global", "project", "legacy", or empty for non-permission checks.
	Source string
}

// HealthReport contains all health check results
type HealthReport struct {
	Checks       []CheckResult
	AgentChecks  []cliagent.AgentStatus
	Passed       bool
	AgentsPassed bool
}

// RunHealthChecks runs all health checks and returns a report.
// Note: Git CLI check was removed since go-git library is used for core operations.
func RunHealthChecks() *HealthReport {
	report := &HealthReport{
		Checks:       make([]CheckResult, 0),
		AgentChecks:  make([]cliagent.AgentStatus, 0),
		Passed:       true,
		AgentsPassed: true,
	}

	// Check Claude CLI
	claudeCheck := CheckClaudeCLI()
	report.Checks = append(report.Checks, claudeCheck)
	if !claudeCheck.Passed {
		report.Passed = false
	}

	// Check Claude settings
	settingsCheck := CheckClaudeSettings()
	report.Checks = append(report.Checks, settingsCheck)
	if !settingsCheck.Passed {
		report.Passed = false
	}

	// Check registered agents (production builds only check production agents)
	allAgentChecks := cliagent.Doctor()
	report.AgentChecks = filterAgentChecks(allAgentChecks)
	for _, status := range report.AgentChecks {
		if !status.Valid {
			report.AgentsPassed = false
			break
		}
	}

	return report
}

// filterAgentChecks returns only production agents (claude, opencode).
// Other agents are hidden for now even in dev builds.
func filterAgentChecks(allChecks []cliagent.AgentStatus) []cliagent.AgentStatus {
	// TODO: Uncomment to show all agents in dev builds when ready
	// if build.IsDevBuild() {
	// 	return allChecks
	// }

	// Only include production agents (claude, opencode)
	prodAgents := make(map[string]bool)
	for _, name := range build.ProductionAgents() {
		prodAgents[name] = true
	}

	filtered := make([]cliagent.AgentStatus, 0, len(prodAgents))
	for _, status := range allChecks {
		if prodAgents[status.Name] {
			filtered = append(filtered, status)
		}
	}
	return filtered
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

// FormatReport formats the health report for console output
func FormatReport(report *HealthReport) string {
	var output string

	// Core checks
	for _, check := range report.Checks {
		if check.Passed {
			output += fmt.Sprintf("✓ %s: %s\n", check.Name, check.Message)
		} else {
			output += fmt.Sprintf("✗ %s: %s\n", check.Name, check.Message)
		}
	}

	// Agent checks
	if len(report.AgentChecks) > 0 {
		output += "\nCLI Agents:\n"
		for _, status := range report.AgentChecks {
			output += FormatAgentStatus(status)
		}
	}

	return output
}

// FormatAgentStatus formats a single agent status for console output
func FormatAgentStatus(status cliagent.AgentStatus) string {
	if status.Valid {
		if status.Version != "" {
			return fmt.Sprintf("  ✓ %s: installed (%s)\n", status.Name, formatAgentVersion(status))
		}
		return fmt.Sprintf("  ✓ %s: installed\n", status.Name)
	}
	if status.Error != "" {
		return fmt.Sprintf("  ○ %s: %s\n", status.Name, status.Error)
	}
	return fmt.Sprintf("  ○ %s: not available\n", status.Name)
}

func formatAgentVersion(status cliagent.AgentStatus) string {
	version := status.Version
	if !strings.HasPrefix(version, "v") && !strings.Contains(version, " ") {
		version = fmt.Sprintf("v%s", version)
	}
	if status.TestedVersion == "" {
		return version
	}
	return fmt.Sprintf("%s; tested %s", version, status.TestedVersion)
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
// It reads init.yml to determine the settings scope (global/project) and checks
// the appropriate location. Falls back to legacy behavior if init.yml is missing.
func CheckClaudeSettingsInDir(projectDir string) CheckResult {
	initPath := filepath.Join(projectDir, initpkg.DefaultPath())
	initSettings, err := initpkg.LoadFrom(initPath)

	// If init.yml exists and is valid, use scope-aware checking
	if err == nil && initpkg.IsValidScope(initSettings.SettingsScope) {
		return checkSettingsWithScope(projectDir, initSettings.SettingsScope)
	}

	// Log warning and fall back to legacy behavior
	if err != nil && initpkg.ExistsAt(initPath) {
		log.Printf("Warning: init.yml exists but is invalid: %v", err)
	}

	return checkSettingsLegacy(projectDir)
}

// checkSettingsWithScope checks permissions at the specified scope location only.
func checkSettingsWithScope(projectDir, scope string) CheckResult {
	switch scope {
	case initpkg.ScopeGlobal:
		return checkGlobalSettings()
	case initpkg.ScopeProject:
		return checkProjectSettings(projectDir)
	default:
		return checkSettingsLegacy(projectDir)
	}
}

// checkGlobalSettings checks permissions in the global Claude settings file.
func checkGlobalSettings() CheckResult {
	settings, err := claude.LoadGlobal()
	if err != nil {
		return CheckResult{
			Name:    "Claude settings",
			Passed:  false,
			Message: fmt.Sprintf("failed to load global settings: %v", err),
			Source:  SourceGlobal,
		}
	}

	return formatSettingsCheck(settings, SourceGlobal)
}

// checkProjectSettings checks permissions in the project-level Claude settings file.
func checkProjectSettings(projectDir string) CheckResult {
	settings, err := claude.Load(projectDir)
	if err != nil {
		return CheckResult{
			Name:    "Claude settings",
			Passed:  false,
			Message: fmt.Sprintf("failed to load project settings: %v", err),
			Source:  SourceProject,
		}
	}

	return formatSettingsCheck(settings, SourceProject)
}

// checkSettingsLegacy checks both global and project settings (legacy fallback).
// Logs a warning about missing init.yml and checks global first, then project.
func checkSettingsLegacy(projectDir string) CheckResult {
	log.Printf("Warning: .autospec/init.yml not found (legacy project). Run 'autospec init' to create it and track configuration properly.")

	// Check global settings first
	globalSettings, err := claude.LoadGlobal()
	if err == nil && globalSettings.HasPermission(claude.RequiredPermission) {
		return CheckResult{
			Name:    "Claude settings",
			Passed:  true,
			Message: fmt.Sprintf("%s permission configured (legacy)", claude.RequiredPermission),
			Source:  SourceLegacy,
		}
	}

	// Check project settings
	projectSettings, err := claude.Load(projectDir)
	if err == nil && projectSettings.HasPermission(claude.RequiredPermission) {
		return CheckResult{
			Name:    "Claude settings",
			Passed:  true,
			Message: fmt.Sprintf("%s permission configured (legacy)", claude.RequiredPermission),
			Source:  SourceLegacy,
		}
	}

	// Neither location has the permission
	return CheckResult{
		Name:    "Claude settings",
		Passed:  false,
		Message: fmt.Sprintf("missing %s permission (legacy) - run 'autospec init' to configure", claude.RequiredPermission),
		Source:  SourceLegacy,
	}
}

// formatSettingsCheck formats a settings check result with the given source.
func formatSettingsCheck(settings *claude.Settings, source string) CheckResult {
	if !settings.Exists() {
		return CheckResult{
			Name:    "Claude settings",
			Passed:  false,
			Message: fmt.Sprintf("settings file not found (%s)", source),
			Source:  source,
		}
	}

	if settings.CheckDenyList(claude.RequiredPermission) {
		return CheckResult{
			Name:    "Claude settings",
			Passed:  false,
			Message: fmt.Sprintf("%s is explicitly denied (%s)", claude.RequiredPermission, source),
			Source:  source,
		}
	}

	if !settings.HasPermission(claude.RequiredPermission) {
		return CheckResult{
			Name:    "Claude settings",
			Passed:  false,
			Message: fmt.Sprintf("missing %s permission (%s)", claude.RequiredPermission, source),
			Source:  source,
		}
	}

	return CheckResult{
		Name:    "Claude settings",
		Passed:  true,
		Message: fmt.Sprintf("%s permission configured (%s)", claude.RequiredPermission, source),
		Source:  source,
	}
}
