package workflow

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PreflightCheck represents a pre-flight validation check
type PreflightCheck struct {
	Name        string
	Description string
	Check       func() error
}

// PreflightResult contains the results of pre-flight validation
type PreflightResult struct {
	Passed         bool
	FailedChecks   []string
	MissingDirs    []string
	GitRoot        string
	CanContinue    bool
	WarningMessage string
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

	// Check 2: Verify specify CLI is in PATH
	if err := checkCommandExists("specify"); err != nil {
		result.Passed = false
		result.FailedChecks = append(result.FailedChecks, "specify CLI not found in PATH")
	}

	// Check 3: Verify .claude/commands/ directory exists
	if _, err := os.Stat(".claude/commands"); os.IsNotExist(err) {
		result.MissingDirs = append(result.MissingDirs, ".claude/commands/")
	}

	// Check 4: Verify .specify/ directory exists
	if _, err := os.Stat(".specify"); os.IsNotExist(err) {
		result.MissingDirs = append(result.MissingDirs, ".specify/")
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

	sb.WriteString("WARNING: Project not initialized with SpecKit\n\n")
	sb.WriteString("Missing directories:\n")
	for _, dir := range missingDirs {
		sb.WriteString(fmt.Sprintf("  - %s (required for SpecKit)\n", dir))
	}
	sb.WriteString("\n")

	if gitRoot != "" {
		sb.WriteString(fmt.Sprintf("Git repository root: %s\n\n", gitRoot))
		sb.WriteString("Recommended setup:\n")
		sb.WriteString(fmt.Sprintf("  cd %s\n", gitRoot))
		sb.WriteString("  specify init . --ai claude --force\n")
	} else {
		sb.WriteString("Recommended setup:\n")
		sb.WriteString("  specify init . --ai claude --force\n")
	}

	return sb.String()
}

// PromptUserToContinue prompts the user to continue despite pre-flight failures
// Returns true if user wants to continue, false otherwise
func PromptUserToContinue(warningMessage string) (bool, error) {
	// Print warning
	fmt.Fprint(os.Stderr, warningMessage)
	fmt.Fprintf(os.Stderr, "\nDo you want to continue anyway? [y/N]: ")

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
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
	deps := []string{"claude", "specify", "git"}
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
	requiredDirs := []string{".claude/commands", ".specify"}
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
