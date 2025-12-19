package workflow

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PreflightChecker is an interface for running preflight checks with testable injection.
// This allows mocking preflight behavior in tests without requiring real system dependencies.
type PreflightChecker interface {
	// RunChecks runs all preflight validations and returns the result.
	// Returns non-nil PreflightResult on success (even if checks fail).
	RunChecks() (*PreflightResult, error)

	// PromptUser prompts the user to continue despite warnings.
	// Returns true if user wants to continue, false otherwise.
	// Must handle EOF gracefully by returning (false, nil).
	PromptUser(warningMessage string) (bool, error)
}

// DefaultPreflightChecker is the default implementation of PreflightChecker
// that uses the system's actual preflight checks and stdin for user prompts.
type DefaultPreflightChecker struct{}

// RunChecks implements PreflightChecker.RunChecks using the actual RunPreflightChecks function.
func (d *DefaultPreflightChecker) RunChecks() (*PreflightResult, error) {
	return RunPreflightChecks()
}

// PromptUser implements PreflightChecker.PromptUser using the actual PromptUserToContinue function.
func (d *DefaultPreflightChecker) PromptUser(warningMessage string) (bool, error) {
	return PromptUserToContinue(warningMessage)
}

// NewDefaultPreflightChecker creates a new DefaultPreflightChecker.
func NewDefaultPreflightChecker() *DefaultPreflightChecker {
	return &DefaultPreflightChecker{}
}

// PreflightCheck represents a pre-flight validation check
type PreflightCheck struct {
	Name        string
	Description string
	Check       func() error
}

// PreflightResult contains the results of pre-flight validation
type PreflightResult struct {
	Passed               bool
	FailedChecks         []string
	MissingDirs          []string
	GitRoot              string
	CanContinue          bool
	WarningMessage       string
	DetectedSpec         string            // Auto-detected or user-specified spec name
	MissingArtifacts     []string          // List of missing prerequisite files
	InvalidArtifacts     map[string]string // Map of artifact name to validation error message
	Warnings             []string          // Warning messages for user
	RequiresConfirmation bool              // Whether user confirmation is needed
}

// RunPreflightChecks runs all pre-flight validation checks
// Performance contract: <100ms
func RunPreflightChecks() (*PreflightResult, error) {
	result := &PreflightResult{
		Passed:       true,
		FailedChecks: make([]string, 0),
		MissingDirs:  make([]string, 0),
	}

	// Check 1: Verify claude CLI is in PATH
	if err := checkCommandExists("claude"); err != nil {
		result.Passed = false
		result.FailedChecks = append(result.FailedChecks, "claude CLI not found in PATH")
	}

	// Check 2: Verify .claude/commands/ directory exists
	if _, err := os.Stat(".claude/commands"); os.IsNotExist(err) {
		result.MissingDirs = append(result.MissingDirs, ".claude/commands/")
	}

	// Check 3: Verify .autospec/ directory exists
	if _, err := os.Stat(".autospec"); os.IsNotExist(err) {
		result.MissingDirs = append(result.MissingDirs, ".autospec/")
	}

	// Get git root for helpful error messages
	if root, err := getGitRoot(); err == nil {
		result.GitRoot = root
	}

	// If directories are missing, generate warning
	if len(result.MissingDirs) > 0 {
		result.Passed = false
		result.WarningMessage = generateMissingDirsWarning(result.MissingDirs, result.GitRoot)
	}

	return result, nil
}

// checkCommandExists verifies that a command is available in PATH
func checkCommandExists(command string) error {
	_, err := exec.LookPath(command)
	if err != nil {
		return fmt.Errorf("%s not found in PATH", command)
	}
	return nil
}

// getGitRoot returns the git repository root directory
func getGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// generateMissingDirsWarning generates a helpful warning message for missing directories
func generateMissingDirsWarning(missingDirs []string, gitRoot string) string {
	var sb strings.Builder

	sb.WriteString("WARNING: Project not initialized with autospec\n\n")
	sb.WriteString("Missing directories:\n")
	for _, dir := range missingDirs {
		sb.WriteString(fmt.Sprintf("  - %s (required for autospec)\n", dir))
	}
	sb.WriteString("\n")

	if gitRoot != "" {
		sb.WriteString(fmt.Sprintf("Git repository root: %s\n\n", gitRoot))
		sb.WriteString("Recommended setup:\n")
		sb.WriteString(fmt.Sprintf("  cd %s\n", gitRoot))
		sb.WriteString("  autospec init\n")
	} else {
		sb.WriteString("Recommended setup:\n")
		sb.WriteString("  autospec init\n")
	}

	return sb.String()
}

// PromptUserToContinueWithReader prompts the user to continue despite pre-flight failures.
// Reader injection pattern: accepts io.Reader for testability (vs hardcoded os.Stdin).
// EOF handling: returns (false, nil) instead of error for graceful termination.
// Response parsing: case-insensitive, accepts "y" or "yes".
func PromptUserToContinueWithReader(warningMessage string, reader io.Reader) (bool, error) {
	// Print warning
	fmt.Fprint(os.Stderr, warningMessage)
	fmt.Fprintf(os.Stderr, "\nDo you want to continue anyway? [y/N]: ")

	// Read user input
	bufReader := bufio.NewReader(reader)
	response, err := bufReader.ReadString('\n')
	if err != nil {
		// Handle EOF gracefully - treat as declining to continue
		if err == io.EOF {
			return false, nil
		}
		return false, fmt.Errorf("reading user input: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
}

// PromptUserToContinue prompts the user to continue despite pre-flight failures
// Returns true if user wants to continue, false otherwise
func PromptUserToContinue(warningMessage string) (bool, error) {
	return PromptUserToContinueWithReader(warningMessage, os.Stdin)
}

// ShouldRunPreflightChecks determines if pre-flight checks should be run
// Checks are skipped in CI/CD environments or if explicitly disabled
func ShouldRunPreflightChecks(skipPreflight bool) bool {
	if skipPreflight {
		return false
	}

	// Check if running in CI/CD environment
	ciEnvVars := []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI"}
	for _, envVar := range ciEnvVars {
		if os.Getenv(envVar) != "" {
			return false
		}
	}

	return true
}

// CheckDependencies checks if all required dependencies are installed
// Returns nil if all dependencies are available
func CheckDependencies() error {
	deps := []string{"claude", "git"}
	var missing []string

	for _, dep := range deps {
		if err := checkCommandExists(dep); err != nil {
			missing = append(missing, dep)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required dependencies: %s", strings.Join(missing, ", "))
	}

	return nil
}

// CheckProjectStructure verifies the project has the expected directory structure
func CheckProjectStructure() error {
	requiredDirs := []string{".claude/commands", ".autospec"}
	var missing []string

	for _, dir := range requiredDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			missing = append(missing, dir)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required directories: %s", strings.Join(missing, ", "))
	}

	return nil
}

// CheckSpecDirectory verifies a spec directory exists and has expected structure
func CheckSpecDirectory(specDir string) error {
	if _, err := os.Stat(specDir); os.IsNotExist(err) {
		return fmt.Errorf("spec directory not found: %s", specDir)
	}

	// Check if it's actually a directory
	info, err := os.Stat(specDir)
	if err != nil {
		return fmt.Errorf("error accessing spec directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("spec path is not a directory: %s", specDir)
	}

	return nil
}

// FindSpecsDirectory finds the specs directory, checking both relative and absolute paths
func FindSpecsDirectory(specsDir string) (string, error) {
	// Try as-is
	if _, err := os.Stat(specsDir); err == nil {
		absPath, _ := filepath.Abs(specsDir)
		return absPath, nil
	}

	// Try from git root if in a git repo
	if gitRoot, err := getGitRoot(); err == nil {
		path := filepath.Join(gitRoot, specsDir)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("specs directory not found: %s", specsDir)
}

// CheckArtifactDependencies checks if required artifacts exist for the selected stages.
// Performs two-level validation: (1) file existence, (2) schema validity.
// Both missing and invalid artifacts cause Passed=false since the workflow cannot proceed.
// Returns PreflightResult with MissingArtifacts and InvalidArtifacts populated.
func CheckArtifactDependencies(stageConfig *StageConfig, specDir string) *PreflightResult {
	result := &PreflightResult{
		Passed:           true,
		MissingArtifacts: make([]string, 0),
		InvalidArtifacts: make(map[string]string),
		Warnings:         make([]string, 0),
	}

	// Get all required artifacts for the selected stages
	requiredArtifacts := stageConfig.GetAllRequiredArtifacts()

	// Check each required artifact - existence AND schema validity
	for _, artifact := range requiredArtifacts {
		artifactPath := filepath.Join(specDir, artifact)
		if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
			result.MissingArtifacts = append(result.MissingArtifacts, artifact)
			continue
		}

		// File exists - validate its schema
		if validationErr := validateArtifactSchema(artifact, specDir); validationErr != nil {
			result.InvalidArtifacts[artifact] = validationErr.Error()
		}
	}

	// If any artifacts are missing or invalid, this is a hard error
	if len(result.MissingArtifacts) > 0 || len(result.InvalidArtifacts) > 0 {
		result.RequiresConfirmation = true // Keep for backward compat in tests
		result.Passed = false
		result.WarningMessage = GeneratePrerequisiteError(stageConfig, result.MissingArtifacts, result.InvalidArtifacts)
	}

	return result
}

// validateArtifactSchema validates the schema of an artifact file.
// Returns nil if valid, error with details if invalid.
func validateArtifactSchema(artifact, specDir string) error {
	switch artifact {
	case "spec.yaml":
		return ValidateSpecSchema(specDir)
	case "plan.yaml":
		return ValidatePlanSchema(specDir)
	case "tasks.yaml":
		return ValidateTasksSchema(specDir)
	default:
		// Unknown artifact type - skip schema validation
		return nil
	}
}

// GeneratePrerequisiteError generates a human-readable error message for missing prerequisites.
//
// Message structure:
//  1. List missing artifacts (files that don't exist)
//  2. List invalid artifacts (files that exist but fail schema validation)
//  3. Show which stages require which problematic artifacts (triple-nested loop)
//  4. Suggest remediation commands based on artifact type
//
// The stage→artifact mapping helps users understand why the check failed.
func GeneratePrerequisiteError(stageConfig *StageConfig, missingArtifacts []string, invalidArtifacts map[string]string) string {
	var sb strings.Builder

	formatMissingArtifacts(&sb, missingArtifacts)
	formatInvalidArtifacts(&sb, invalidArtifacts)
	formatStageArtifactMapping(&sb, stageConfig, missingArtifacts, invalidArtifacts)
	formatMissingRemediation(&sb, missingArtifacts)
	formatInvalidRemediation(&sb, invalidArtifacts)

	return sb.String()
}

// formatMissingArtifacts writes the missing artifacts section
func formatMissingArtifacts(sb *strings.Builder, missingArtifacts []string) {
	if len(missingArtifacts) == 0 {
		return
	}
	sb.WriteString("\nError: Missing required prerequisite artifacts:\n")
	for _, artifact := range missingArtifacts {
		sb.WriteString(fmt.Sprintf("  - %s\n", artifact))
	}
}

// formatInvalidArtifacts writes the invalid artifacts section
func formatInvalidArtifacts(sb *strings.Builder, invalidArtifacts map[string]string) {
	if len(invalidArtifacts) == 0 {
		return
	}
	sb.WriteString("\nError: Invalid artifact schemas:\n")
	for artifact, errMsg := range invalidArtifacts {
		sb.WriteString(fmt.Sprintf("  - %s: %s\n", artifact, errMsg))
	}
}

// formatStageArtifactMapping writes which stages require which artifacts
func formatStageArtifactMapping(sb *strings.Builder, stageConfig *StageConfig, missingArtifacts []string, invalidArtifacts map[string]string) {
	allProblematic := collectProblematicArtifacts(missingArtifacts, invalidArtifacts)
	sb.WriteString("\nThe following stages require these artifacts:\n")
	for _, stage := range stageConfig.GetSelectedStages() {
		for _, req := range GetRequiredArtifacts(stage) {
			for _, problematic := range allProblematic {
				if req == problematic {
					sb.WriteString(fmt.Sprintf("  - %s requires %s\n", stage, req))
				}
			}
		}
	}
}

// collectProblematicArtifacts combines missing and invalid artifact names
func collectProblematicArtifacts(missingArtifacts []string, invalidArtifacts map[string]string) []string {
	result := make([]string, 0, len(missingArtifacts)+len(invalidArtifacts))
	result = append(result, missingArtifacts...)
	for artifact := range invalidArtifacts {
		result = append(result, artifact)
	}
	return result
}

// formatMissingRemediation writes suggestions for generating missing artifacts
func formatMissingRemediation(sb *strings.Builder, missingArtifacts []string) {
	if len(missingArtifacts) == 0 {
		return
	}
	sb.WriteString("\nRun earlier stages first to generate the required artifacts:\n")
	if containsArtifact(missingArtifacts, "spec.yaml") {
		sb.WriteString("  autospec run -s \"feature description\"  # Generate spec.yaml\n")
	}
	if containsArtifact(missingArtifacts, "plan.yaml") {
		sb.WriteString("  autospec run -p                         # Generate plan.yaml\n")
	}
	if containsArtifact(missingArtifacts, "tasks.yaml") {
		sb.WriteString("  autospec run -t                         # Generate tasks.yaml\n")
	}
}

// formatInvalidRemediation writes suggestions for fixing invalid artifacts
func formatInvalidRemediation(sb *strings.Builder, invalidArtifacts map[string]string) {
	if len(invalidArtifacts) == 0 {
		return
	}
	sb.WriteString("\nFix the schema errors by re-running the stage that produces each artifact:\n")
	for artifact := range invalidArtifacts {
		switch artifact {
		case "spec.yaml":
			sb.WriteString("  autospec run -s \"feature description\"  # Regenerate spec.yaml\n")
		case "plan.yaml":
			sb.WriteString("  autospec run -p                         # Regenerate plan.yaml\n")
		case "tasks.yaml":
			sb.WriteString("  autospec run -t                         # Regenerate tasks.yaml\n")
		}
	}
}

// GeneratePrerequisiteWarning is an alias for GeneratePrerequisiteError for backward compatibility.
// Deprecated: Use GeneratePrerequisiteError instead.
func GeneratePrerequisiteWarning(stageConfig *StageConfig, missingArtifacts []string, invalidArtifacts map[string]string) string {
	return GeneratePrerequisiteError(stageConfig, missingArtifacts, invalidArtifacts)
}

// containsArtifact checks if an artifact is in the list
func containsArtifact(artifacts []string, artifact string) bool {
	for _, a := range artifacts {
		if a == artifact {
			return true
		}
	}
	return false
}

// ConstitutionPaths contains all valid paths for the autospec constitution file (in priority order)
var ConstitutionPaths = []string{
	".autospec/memory/constitution.yaml",
	".autospec/memory/constitution.yml",
	".specify/memory/constitution.yaml",
	".specify/memory/constitution.yml",
}

// ConstitutionCheckResult contains the result of constitution validation
type ConstitutionCheckResult struct {
	Exists       bool
	Path         string
	ErrorMessage string
}

// CheckConstitutionExists checks if the constitution file exists.
// This is a required project-level artifact that must exist before
// running any workflow stages (specify, plan, tasks, implement).
// Checks paths in ConstitutionPaths order (.yaml and .yml extensions supported)
func CheckConstitutionExists() *ConstitutionCheckResult {
	result := &ConstitutionCheckResult{}

	// Check all valid constitution paths in priority order
	for _, path := range ConstitutionPaths {
		if _, err := os.Stat(path); err == nil {
			result.Exists = true
			result.Path = path
			return result
		}
	}

	// Constitution not found
	result.Exists = false
	result.ErrorMessage = generateConstitutionMissingError()
	return result
}

// generateConstitutionMissingError generates the error message for missing constitution
func generateConstitutionMissingError() string {
	var sb strings.Builder

	sb.WriteString("\nError: Project constitution not found.\n\n")
	sb.WriteString("A constitution is required before running any workflow stages.\n")
	sb.WriteString("The constitution defines your project's principles and guidelines.\n\n")
	sb.WriteString("To create a constitution, run:\n")
	sb.WriteString("  autospec constitution\n\n")
	sb.WriteString("Or if you have an existing constitution at .specify/memory/constitution.yaml,\n")
	sb.WriteString("run 'autospec init' to copy it to .autospec/memory/constitution.yaml\n")

	return sb.String()
}

// PrerequisiteValidationResult contains the result of prerequisite validation for a stage.
type PrerequisiteValidationResult struct {
	Valid            bool              // Whether all prerequisites are satisfied
	MissingArtifacts []string          // List of missing artifact file names
	InvalidArtifacts map[string]string // Map of artifact -> validation error message
	ErrorMessage     string            // User-friendly error with remediation suggestions
}

// ValidateStagePrerequisites validates that all required artifacts exist for a stage.
// It checks the artifacts defined in artifactDependencies for the given stage.
// Returns a PrerequisiteValidationResult indicating if validation passed and any missing files.
func ValidateStagePrerequisites(stage Stage, specDir string) *PrerequisiteValidationResult {
	result := &PrerequisiteValidationResult{
		Valid:            true,
		MissingArtifacts: make([]string, 0),
		InvalidArtifacts: make(map[string]string),
	}

	// Get required artifacts for this specific stage
	requiredArtifacts := GetRequiredArtifacts(stage)

	// Check each required artifact exists and has valid schema
	for _, artifact := range requiredArtifacts {
		artifactPath := filepath.Join(specDir, artifact)
		if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
			result.MissingArtifacts = append(result.MissingArtifacts, artifact)
			continue
		}

		// Validate schema for existing artifacts
		if validationErr := validateArtifactSchema(artifact, specDir); validationErr != nil {
			result.InvalidArtifacts[artifact] = validationErr.Error()
		}
	}

	// Generate error message if any artifacts are missing or invalid
	if len(result.MissingArtifacts) > 0 || len(result.InvalidArtifacts) > 0 {
		result.Valid = false
		result.ErrorMessage = GenerateArtifactValidationError(result.MissingArtifacts, result.InvalidArtifacts)
	}

	return result
}

// GenerateArtifactMissingError generates a user-friendly error message for missing artifacts.
// It includes the missing file names and remediation commands for each.
func GenerateArtifactMissingError(missingArtifacts []string) string {
	var sb strings.Builder

	if len(missingArtifacts) == 1 {
		artifact := missingArtifacts[0]
		sb.WriteString(fmt.Sprintf("\nError: %s not found.\n\n", artifact))
		sb.WriteString(fmt.Sprintf("Run '%s' first to create this file.\n", GetRemediationCommand(artifact)))
	} else {
		sb.WriteString("\nError: Missing required artifacts:\n")
		for _, artifact := range missingArtifacts {
			sb.WriteString(fmt.Sprintf("  - %s\n", artifact))
		}
		sb.WriteString("\nRun the following commands to create them:\n")
		for _, artifact := range missingArtifacts {
			sb.WriteString(fmt.Sprintf("  %s\n", GetRemediationCommand(artifact)))
		}
	}

	return sb.String()
}

// GenerateArtifactValidationError creates a user-friendly error message for missing and invalid artifacts.
func GenerateArtifactValidationError(missingArtifacts []string, invalidArtifacts map[string]string) string {
	var sb strings.Builder

	// Handle missing artifacts
	if len(missingArtifacts) > 0 {
		if len(missingArtifacts) == 1 {
			artifact := missingArtifacts[0]
			sb.WriteString(fmt.Sprintf("\nError: %s not found.\n\n", artifact))
			sb.WriteString(fmt.Sprintf("Run '%s' first to create this file.\n", GetRemediationCommand(artifact)))
		} else {
			sb.WriteString("\nError: Missing required artifacts:\n")
			for _, artifact := range missingArtifacts {
				sb.WriteString(fmt.Sprintf("  - %s\n", artifact))
			}
			sb.WriteString("\nRun the following commands to create them:\n")
			for _, artifact := range missingArtifacts {
				sb.WriteString(fmt.Sprintf("  %s\n", GetRemediationCommand(artifact)))
			}
		}
	}

	// Handle invalid artifacts
	if len(invalidArtifacts) > 0 {
		if len(missingArtifacts) > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("\nError: Invalid artifact schema:\n")
		for artifact, errMsg := range invalidArtifacts {
			sb.WriteString(fmt.Sprintf("  - %s: %s\n", artifact, errMsg))
		}
		sb.WriteString("\nFix the schema errors or regenerate the artifact.\n")
	}

	return sb.String()
}

// GetRemediationCommand returns the autospec command that creates the given artifact.
func GetRemediationCommand(artifact string) string {
	commands := map[string]string{
		"constitution.yaml": "autospec constitution",
		"spec.yaml":         "autospec specify",
		"plan.yaml":         "autospec plan",
		"tasks.yaml":        "autospec tasks",
	}

	if cmd, ok := commands[artifact]; ok {
		return cmd
	}
	return fmt.Sprintf("autospec (unknown artifact: %s)", artifact)
}
