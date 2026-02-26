// Package testutil provides test utilities and helpers for autospec tests.
package testutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	// autospecBinaryPath caches the built autospec binary path.
	autospecBinaryPath string
	autospecBuildOnce  sync.Once
	autospecBuildErr   error
)

// AgentPreset defines the CLI agent to use in E2E tests.
type AgentPreset string

const (
	// AgentClaude uses the mock-claude.sh script.
	AgentClaude AgentPreset = "claude"
	// AgentOpencode uses the mock-opencode.sh script.
	AgentOpencode AgentPreset = "opencode"
)

// E2EEnv provides an isolated environment for E2E testing.
// It manages PATH isolation, temp directories, and environment sanitization
// to ensure E2E tests never invoke the real Claude or OpenCode CLI.
type E2EEnv struct {
	t               *testing.T
	tempDir         string
	binDir          string
	specsDir        string
	originalEnv     map[string]string
	cleanedUp       bool
	mockExitCode    int
	mockExitCodeSet bool
	mockCallLog     string
	mockDelay       int
	mockDelaySet    bool
	agentPreset     AgentPreset
}

// CommandResult captures the result of running an autospec command.
type CommandResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}

// NewE2EEnv creates a new E2E test environment with PATH isolation.
// The mock claude binary will be the only "claude" in PATH.
// Use WithAgentPreset() to switch to a different agent.
func NewE2EEnv(t *testing.T) *E2EEnv {
	t.Helper()

	env := &E2EEnv{
		t:           t,
		originalEnv: make(map[string]string),
		agentPreset: AgentClaude, // Default to Claude for backward compatibility
	}

	env.setup()
	t.Cleanup(env.Cleanup)

	return env
}

// NewE2EEnvWithAgent creates a new E2E test environment with the specified agent preset.
func NewE2EEnvWithAgent(t *testing.T, preset AgentPreset) *E2EEnv {
	t.Helper()

	env := &E2EEnv{
		t:           t,
		originalEnv: make(map[string]string),
		agentPreset: preset,
	}

	env.setup()
	t.Cleanup(env.Cleanup)

	return env
}

func (e *E2EEnv) setup() {
	e.t.Helper()

	// Create temp directory for this test
	tempDir, err := os.MkdirTemp("", "e2e-test-*")
	if err != nil {
		e.t.Fatalf("creating temp directory: %v", err)
	}
	e.tempDir = tempDir

	// Create bin directory for mock binaries
	e.binDir = filepath.Join(tempDir, "bin")
	if err := os.MkdirAll(e.binDir, 0o755); err != nil {
		e.t.Fatalf("creating bin directory: %v", err)
	}

	// Create specs directory
	e.specsDir = filepath.Join(tempDir, "specs")
	if err := os.MkdirAll(e.specsDir, 0o755); err != nil {
		e.t.Fatalf("creating specs directory: %v", err)
	}

	e.setupMockAgent()
	e.buildAutospec()
	e.captureAndSanitizeEnv()
}

func (e *E2EEnv) setupMockAgent() {
	e.t.Helper()

	// Always set up both mock scripts for flexibility
	e.setupMockClaude()
	e.setupMockOpencode()
}

func (e *E2EEnv) setupMockClaude() {
	e.t.Helper()

	mockPath := e.findMockScriptPath("mock-claude.sh")
	claudeLink := filepath.Join(e.binDir, "claude")

	content, err := os.ReadFile(mockPath)
	if err != nil {
		e.t.Fatalf("reading mock-claude.sh: %v", err)
	}

	if err := os.WriteFile(claudeLink, content, 0o755); err != nil {
		e.t.Fatalf("writing mock claude binary: %v", err)
	}
}

func (e *E2EEnv) setupMockOpencode() {
	e.t.Helper()

	mockPath := e.findMockScriptPath("mock-opencode.sh")
	opencodeLink := filepath.Join(e.binDir, "opencode")

	content, err := os.ReadFile(mockPath)
	if err != nil {
		e.t.Fatalf("reading mock-opencode.sh: %v", err)
	}

	if err := os.WriteFile(opencodeLink, content, 0o755); err != nil {
		e.t.Fatalf("writing mock opencode binary: %v", err)
	}
}

func (e *E2EEnv) findMockScriptPath(scriptName string) string {
	e.t.Helper()

	// Get the path to the current source file
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		e.t.Fatal("failed to determine current file location")
	}

	// Navigate from internal/testutil/ to repo root
	repoRoot := filepath.Join(filepath.Dir(currentFile), "..", "..")

	// Mock scripts are in tests/mocks/scripts/
	mockPath := filepath.Join(repoRoot, "tests", "mocks", "scripts", scriptName)
	if _, err := os.Stat(mockPath); err == nil {
		return mockPath
	}

	e.t.Fatalf("%s not found at %s", scriptName, mockPath)
	return ""
}

func (e *E2EEnv) buildAutospec() {
	e.t.Helper()

	// Build autospec binary once per test session
	autospecBuildOnce.Do(func() {
		autospecBinaryPath, autospecBuildErr = e.doBuildAutospec()
	})

	if autospecBuildErr != nil {
		e.t.Fatalf("building autospec: %v", autospecBuildErr)
	}

	// Link autospec binary to our bin directory
	autospecLink := filepath.Join(e.binDir, "autospec")
	content, err := os.ReadFile(autospecBinaryPath)
	if err != nil {
		e.t.Fatalf("reading autospec binary: %v", err)
	}

	if err := os.WriteFile(autospecLink, content, 0o755); err != nil {
		e.t.Fatalf("writing autospec binary: %v", err)
	}
}

func (e *E2EEnv) doBuildAutospec() (string, error) {
	// Get repo root
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("determining current file location")
	}
	repoRoot := filepath.Join(filepath.Dir(currentFile), "..", "..")

	// Create a temp directory for the built binary
	tmpDir, err := os.MkdirTemp("", "autospec-build-*")
	if err != nil {
		return "", fmt.Errorf("creating temp dir for build: %w", err)
	}

	binaryPath := filepath.Join(tmpDir, "autospec")

	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/autospec")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("building autospec: %w\nOutput: %s", err, output)
	}

	return binaryPath, nil
}

func (e *E2EEnv) captureAndSanitizeEnv() {
	e.t.Helper()

	// Capture original environment variables we'll modify
	sensitiveVars := []string{
		"ANTHROPIC_API_KEY",
		"CLAUDE_API_KEY",
		"OPENAI_API_KEY",
		"OPENCODE_API_KEY",
		"PATH",
	}

	for _, key := range sensitiveVars {
		if val, ok := os.LookupEnv(key); ok {
			e.originalEnv[key] = val
		}
	}
}

// Run executes an autospec command in the isolated E2E environment.
func (e *E2EEnv) Run(args ...string) CommandResult {
	e.t.Helper()

	start := time.Now()

	cmd := exec.Command(filepath.Join(e.binDir, "autospec"), args...)
	cmd.Dir = e.tempDir
	cmd.Env = e.buildIsolatedEnv()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := CommandResult{
		ExitCode: 0,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: time.Since(start),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
	}

	return result
}

func (e *E2EEnv) buildIsolatedEnv() []string {
	// Build PATH: our mock bin dir first (so mock claude takes precedence),
	// then system PATH for standard utilities (mkdir, cat, etc.)
	systemPath := os.Getenv("PATH")
	isolatedPath := e.binDir
	if systemPath != "" {
		isolatedPath = e.binDir + ":" + systemPath
	}

	env := []string{
		"PATH=" + isolatedPath,
		"HOME=" + e.tempDir,
		"MOCK_ARTIFACT_DIR=" + e.specsDir,
		"MOCK_SPEC_NAME=001-test-feature", // Default spec name matching typical branch pattern
	}

	// Add mock exit code if configured
	if e.mockExitCodeSet {
		env = append(env, fmt.Sprintf("MOCK_EXIT_CODE=%d", e.mockExitCode))
	}

	// Add mock call log if configured
	if e.mockCallLog != "" {
		env = append(env, "MOCK_CALL_LOG="+e.mockCallLog)
	}

	// Add mock delay if configured (for timeout testing)
	if e.mockDelaySet {
		env = append(env, fmt.Sprintf("MOCK_DELAY=%d", e.mockDelay))
	}

	// Add safe environment variables from original environment
	safeVars := []string{
		"TERM",
		"LANG",
		"LC_ALL",
		"TMPDIR",
		"TMP",
		"TEMP",
	}

	for _, key := range safeVars {
		if val, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+val)
		}
	}

	// Explicitly exclude API keys - they should NEVER be in the isolated env
	// This is verified by HasAPIKeyInEnv()

	return env
}

// TempDir returns the root temp directory for this test environment.
func (e *E2EEnv) TempDir() string {
	return e.tempDir
}

// SpecsDir returns the specs directory path.
func (e *E2EEnv) SpecsDir() string {
	return e.specsDir
}

// BinDir returns the bin directory containing mock and autospec binaries.
func (e *E2EEnv) BinDir() string {
	return e.binDir
}

// SetupConstitution creates a valid constitution.yaml in the test environment.
func (e *E2EEnv) SetupConstitution() {
	e.t.Helper()

	constitutionDir := filepath.Join(e.tempDir, ".autospec")
	if err := os.MkdirAll(constitutionDir, 0o755); err != nil {
		e.t.Fatalf("creating constitution directory: %v", err)
	}

	content := `constitution:
  project_name: "e2e-test-project"
  version: "1.0.0"
  ratified: "2025-01-01"
  last_amended: "2025-01-01"

preamble: "Test constitution for E2E testing."

principles:
  - name: "Test-First Development"
    id: "PRIN-001"
    category: "quality"
    priority: "NON-NEGOTIABLE"
    description: "All new code must have tests."
    rationale: "Ensures code quality"
    enforcement:
      - mechanism: "CI"
        description: "Tests run on commit"
    exceptions: []

sections:
  - name: "Code Quality"
    content: "All code must pass linting."

governance:
  amendment_process:
    - step: 1
      action: "Propose"
      requirements: "Include rationale"
  versioning_policy: "Semantic versioning"
  compliance_review:
    frequency: "quarterly"
    process: "Review"
  rules:
    - "Changes require review"

sync_impact:
  version_change: "1.0.0 -> 1.0.0"
  modified_principles: []
  added_sections: []
  removed_sections: []
  templates_requiring_updates: []
  follow_up_todos: []

_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "constitution"
`
	path := filepath.Join(constitutionDir, "constitution.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		e.t.Fatalf("writing constitution: %v", err)
	}
}

// SetupSpec creates a valid spec.yaml in the test environment.
func (e *E2EEnv) SetupSpec(specName string) string {
	e.t.Helper()

	specDir := filepath.Join(e.specsDir, specName)
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		e.t.Fatalf("creating spec directory: %v", err)
	}

	ArtifactGenerators.Spec(specDir)
	return specDir
}

// SetupPlan creates a valid plan.yaml in the test environment.
func (e *E2EEnv) SetupPlan(specName string) string {
	e.t.Helper()

	specDir := e.SetupSpec(specName) // Ensure spec exists first
	ArtifactGenerators.Plan(specDir)
	return specDir
}

// SetupTasks creates a valid tasks.yaml in the test environment.
func (e *E2EEnv) SetupTasks(specName string) string {
	e.t.Helper()

	specDir := e.SetupPlan(specName) // Ensure plan exists first
	ArtifactGenerators.Tasks(specDir)
	return specDir
}

// SetMockExitCode configures the mock agent to return a specific exit code.
// Note: This sets an internal field that will be used when Run is called.
func (e *E2EEnv) SetMockExitCode(code int) {
	e.t.Helper()
	e.mockExitCode = code
	e.mockExitCodeSet = true
}

// SetMockCallLog configures the path where mock agent invocations will be logged.
// The log file will be created by the mock script when it's invoked.
func (e *E2EEnv) SetMockCallLog(path string) {
	e.t.Helper()
	e.mockCallLog = path
}

// SetMockDelay configures a delay (in seconds) before the mock responds.
// This is used for timeout testing (FR-006, exit code 5).
func (e *E2EEnv) SetMockDelay(seconds int) {
	e.t.Helper()
	e.mockDelay = seconds
	e.mockDelaySet = true
}

// SetAgentPreset switches the agent preset used by autospec commands.
// This updates the config file in the test environment.
func (e *E2EEnv) SetAgentPreset(preset AgentPreset) {
	e.t.Helper()
	e.agentPreset = preset
	e.updateConfigAgentPreset()
}

// AgentPreset returns the current agent preset.
func (e *E2EEnv) AgentPreset() AgentPreset {
	return e.agentPreset
}

func (e *E2EEnv) updateConfigAgentPreset() {
	e.t.Helper()

	configPath := filepath.Join(e.tempDir, ".autospec", "config.yml")
	content := fmt.Sprintf(`# Autospec E2E test config
agent_preset: %s
specs_dir: specs
max_retries: 1
skip_preflight: false
`, string(e.agentPreset))

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		e.t.Fatalf("updating autospec config for agent preset: %v", err)
	}
}

// HasAPIKeyInEnv returns true if any API key is present in the environment.
// This is used to verify that E2E tests properly sanitize the environment.
// Checks for ANTHROPIC_API_KEY, CLAUDE_API_KEY, OPENAI_API_KEY, and OPENCODE_API_KEY.
func (e *E2EEnv) HasAPIKeyInEnv() bool {
	env := e.buildIsolatedEnv()
	apiKeyPrefixes := []string{
		"ANTHROPIC_API_KEY=",
		"CLAUDE_API_KEY=",
		"OPENAI_API_KEY=",
		"OPENCODE_API_KEY=",
	}
	for _, v := range env {
		for _, prefix := range apiKeyPrefixes {
			if strings.HasPrefix(v, prefix) {
				return true
			}
		}
	}
	return false
}

// HasClaudeAPIKeyInEnv returns true if ANTHROPIC_API_KEY is present in the environment.
func (e *E2EEnv) HasClaudeAPIKeyInEnv() bool {
	env := e.buildIsolatedEnv()
	for _, v := range env {
		if strings.HasPrefix(v, "ANTHROPIC_API_KEY=") {
			return true
		}
	}
	return false
}

// HasOpenAIAPIKeyInEnv returns true if OPENAI_API_KEY is present in the environment.
func (e *E2EEnv) HasOpenAIAPIKeyInEnv() bool {
	env := e.buildIsolatedEnv()
	for _, v := range env {
		if strings.HasPrefix(v, "OPENAI_API_KEY=") {
			return true
		}
	}
	return false
}

// SpecExists checks if a spec artifact file exists.
func (e *E2EEnv) SpecExists(specName string) bool {
	path := filepath.Join(e.specsDir, specName, "spec.yaml")
	_, err := os.Stat(path)
	return err == nil
}

// PlanExists checks if a plan artifact file exists.
func (e *E2EEnv) PlanExists(specName string) bool {
	path := filepath.Join(e.specsDir, specName, "plan.yaml")
	_, err := os.Stat(path)
	return err == nil
}

// TasksExists checks if a tasks artifact file exists.
func (e *E2EEnv) TasksExists(specName string) bool {
	path := filepath.Join(e.specsDir, specName, "tasks.yaml")
	_, err := os.Stat(path)
	return err == nil
}

// ChecklistExists checks if a checklist artifact file exists.
func (e *E2EEnv) ChecklistExists(specName string) bool {
	path := filepath.Join(e.specsDir, specName, "checklists", "checklist.yaml")
	_, err := os.Stat(path)
	return err == nil
}

// AnalysisExists checks if an analysis artifact file exists.
func (e *E2EEnv) AnalysisExists(specName string) bool {
	path := filepath.Join(e.specsDir, specName, "analysis.yaml")
	_, err := os.Stat(path)
	return err == nil
}

// SpecHasClarifications checks if spec.yaml contains clarifications.
func (e *E2EEnv) SpecHasClarifications(specName string) bool {
	path := filepath.Join(e.specsDir, specName, "spec.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), "clarifications:")
}

// Cleanup restores the original environment and removes temp files.
func (e *E2EEnv) Cleanup() {
	if e.cleanedUp {
		return
	}
	e.cleanedUp = true

	// Remove temp directory
	if e.tempDir != "" {
		if err := os.RemoveAll(e.tempDir); err != nil {
			e.t.Logf("note: could not remove temp directory: %v", err)
		}
	}
}

// AssertNoAPIKeys verifies that no API keys are present in the isolated environment.
// This is a helper method for safety verification tests (US-001, FR-002).
func (e *E2EEnv) AssertNoAPIKeys(t *testing.T) {
	t.Helper()
	if e.HasAPIKeyInEnv() {
		t.Fatal("API keys should not be present in isolated E2E environment")
	}
}

// AssertMockOnlyPath verifies that the PATH is properly isolated with mock binaries.
// This is a helper method for safety verification tests (US-001, FR-001).
func (e *E2EEnv) AssertMockOnlyPath(t *testing.T) {
	t.Helper()

	binDir := e.BinDir()
	tempDir := e.TempDir()

	if binDir == "" {
		t.Fatal("bin directory should be set")
	}
	if !strings.HasPrefix(binDir, tempDir) {
		t.Fatalf("bin dir %q should be within temp dir %q for isolation", binDir, tempDir)
	}

	// Verify mock binaries exist
	requiredBinaries := []string{"claude", "opencode", "autospec"}
	for _, binary := range requiredBinaries {
		path := filepath.Join(binDir, binary)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatalf("required binary %q not found in bin dir", binary)
		}
	}
}

// AssertTempStateDir verifies that state files are written to temp directory.
// This is a helper method for safety verification tests (US-001, FR-003).
func (e *E2EEnv) AssertTempStateDir(t *testing.T) {
	t.Helper()

	tempDir := e.TempDir()
	if tempDir == "" {
		t.Fatal("temp directory should be set")
	}

	// Verify HOME is not the real home directory
	realHome, err := os.UserHomeDir()
	if err == nil && realHome == tempDir {
		t.Fatal("E2EEnv temp dir should not be real home directory")
	}
}

// AssertGitIsolated verifies that git operations use an isolated temp repository.
// This is a helper method for safety verification tests (US-001, FR-004).
// Note: This method initializes a git repo if one doesn't exist.
func (e *E2EEnv) AssertGitIsolated(t *testing.T) {
	t.Helper()

	tempDir := e.TempDir()
	gitDir := filepath.Join(tempDir, ".git")

	// Initialize git repo if not already done
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		e.InitGitRepo()
	}

	// Verify .git exists in temp dir
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Fatal("git repo should be initialized in temp dir")
	}
}

// InitGitRepo initializes a git repository in the temp directory.
func (e *E2EEnv) InitGitRepo() {
	e.t.Helper()

	cmd := exec.Command("git", "init")
	cmd.Dir = e.tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		e.t.Fatalf("git init failed: %v\nOutput: %s", err, output)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = e.tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		e.t.Fatalf("git config email failed: %v\nOutput: %s", err, output)
	}

	cmd = exec.Command("git", "config", "user.name", "Test")
	cmd.Dir = e.tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		e.t.Fatalf("git config name failed: %v\nOutput: %s", err, output)
	}
}

// CreateBranch creates and checks out a new branch in the test git repo.
func (e *E2EEnv) CreateBranch(name string) {
	e.t.Helper()

	// Need an initial commit first
	readme := filepath.Join(e.tempDir, "README.md")
	if err := os.WriteFile(readme, []byte("# Test"), 0o644); err != nil {
		e.t.Fatalf("writing README: %v", err)
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = e.tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		e.t.Fatalf("git add failed: %v\nOutput: %s", err, output)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = e.tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		e.t.Fatalf("git commit failed: %v\nOutput: %s", err, output)
	}

	cmd = exec.Command("git", "checkout", "-b", name)
	cmd.Dir = e.tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		e.t.Fatalf("git checkout -b failed: %v\nOutput: %s", err, output)
	}
}

// SetupAutospecInit creates the directory structure needed by autospec preflight.
// This simulates running 'autospec init' to satisfy preflight checks.
func (e *E2EEnv) SetupAutospecInit() {
	e.t.Helper()

	// Create .claude/commands directory (needed for both agent types)
	claudeCommandsDir := filepath.Join(e.tempDir, ".claude", "commands")
	if err := os.MkdirAll(claudeCommandsDir, 0o755); err != nil {
		e.t.Fatalf("creating .claude/commands directory: %v", err)
	}

	// Create a minimal autospec config
	autospecConfigDir := filepath.Join(e.tempDir, ".autospec")
	if err := os.MkdirAll(autospecConfigDir, 0o755); err != nil {
		e.t.Fatalf("creating .autospec directory: %v", err)
	}

	// Create minimal config.yml with current agent preset
	configContent := fmt.Sprintf(`# Autospec E2E test config
agent_preset: %s
specs_dir: specs
max_retries: 1
skip_preflight: false
`, string(e.agentPreset))
	configPath := filepath.Join(autospecConfigDir, "config.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		e.t.Fatalf("writing autospec config: %v", err)
	}
}
