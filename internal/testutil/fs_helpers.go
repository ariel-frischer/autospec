// Package testutil provides test utilities and helpers for autospec tests.
package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// CreateTempSpec creates a valid spec.yaml file in a temp directory for testing.
// Returns the spec directory path. Cleanup is handled via t.Cleanup.
func CreateTempSpec(t *testing.T, specsDir, specName string) string {
	t.Helper()

	specDir := filepath.Join(specsDir, specName)
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("failed to create spec directory: %v", err)
	}

	specContent := fmt.Sprintf(`feature:
  branch: "%s"
  created: "2025-01-01"
  status: "Draft"
  input: "test feature"

user_stories:
  - id: "US-001"
    title: "Test user story"
    priority: "P1"
    as_a: "developer"
    i_want: "to test"
    so_that: "tests pass"
    why_this_priority: "Testing"
    independent_test: "Run tests"
    acceptance_scenarios:
      - given: "a test"
        when: "running"
        then: "it passes"

requirements:
  functional:
    - id: "FR-001"
      description: "Test requirement"
      testable: true
      acceptance_criteria: "Test passes"

  non_functional:
    - id: "NFR-001"
      category: "code_quality"
      description: "Code quality"
      measurable_target: "High quality"

success_criteria:
  measurable_outcomes:
    - id: "SC-001"
      description: "Test passes"
      metric: "Pass rate"
      target: "100%%"

key_entities: []
edge_cases: []
assumptions: []
constraints: []
out_of_scope: []

_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "spec"
`, specName)

	specPath := filepath.Join(specDir, "spec.yaml")
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write spec.yaml: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(specDir)
	})

	return specDir
}

// CreateTempPlan creates a valid plan.yaml file in the spec directory for testing.
// Returns the plan file path. Cleanup is handled via t.Cleanup.
func CreateTempPlan(t *testing.T, specDir string) string {
	t.Helper()

	planContent := `plan:
  branch: "test-feature"
  created: "2025-01-01"
  spec_path: "specs/test-feature/spec.yaml"

summary: |
  Test implementation plan.

technical_context:
  language: "Go"
  framework: "None"
  primary_dependencies: []
  storage: "None"
  testing:
    framework: "Go testing"
    approach: "Unit tests"
  target_platform: "Linux, macOS, Windows"
  project_type: "cli"
  performance_goals: "Fast"
  constraints: []
  scale_scope: "Small"

constitution_check:
  constitution_path: ".autospec/memory/constitution.yaml"
  gates: []

research_findings:
  decisions: []

data_model:
  entities: []

api_contracts:
  endpoints: []

project_structure:
  documentation: []
  source_code: []
  tests: []

implementation_phases:
  - phase: 1
    name: "Implementation"
    goal: "Implement feature"
    deliverables: []

risks: []
open_questions: []

_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "plan"
`

	planPath := filepath.Join(specDir, "plan.yaml")
	if err := os.WriteFile(planPath, []byte(planContent), 0644); err != nil {
		t.Fatalf("failed to write plan.yaml: %v", err)
	}

	return planPath
}

// CreateTempTasks creates a valid tasks.yaml file in the spec directory for testing.
// Returns the tasks file path. Cleanup is handled via t.Cleanup.
func CreateTempTasks(t *testing.T, specDir string, opts ...TasksOption) string {
	t.Helper()

	config := &tasksConfig{
		totalTasks:   1,
		taskStatus:   "Pending",
		totalPhases:  1,
		phaseTitle:   "Test Phase",
		taskTitle:    "Test Task",
		taskID:       "T001",
		dependencies: []string{},
	}

	for _, opt := range opts {
		opt(config)
	}

	tasksContent := fmt.Sprintf(`tasks:
  branch: "test-feature"
  created: "2025-01-01"
  spec_path: "specs/test-feature/spec.yaml"
  plan_path: "specs/test-feature/plan.yaml"

summary:
  total_tasks: %d
  total_phases: %d
  parallel_opportunities: 0
  estimated_complexity: "low"

phases:
  - number: 1
    title: "%s"
    purpose: "Testing"
    tasks:
      - id: "%s"
        title: "%s"
        status: "%s"
        type: "implementation"
        parallel: false
        story_id: "US-001"
        file_path: "test.go"
        dependencies: %s
        acceptance_criteria:
          - "Test passes"

dependencies:
  user_story_order: []
  phase_order: []

parallel_execution: []

implementation_strategy:
  mvp_scope:
    phases: [1]
    description: "MVP"
    validation: "Tests pass"
  incremental_delivery: []

_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "tasks"
`, config.totalTasks, config.totalPhases, config.phaseTitle, config.taskID,
		config.taskTitle, config.taskStatus, formatDependencies(config.dependencies))

	tasksPath := filepath.Join(specDir, "tasks.yaml")
	if err := os.WriteFile(tasksPath, []byte(tasksContent), 0644); err != nil {
		t.Fatalf("failed to write tasks.yaml: %v", err)
	}

	return tasksPath
}

// tasksConfig holds configuration for CreateTempTasks
type tasksConfig struct {
	totalTasks   int
	taskStatus   string
	totalPhases  int
	phaseTitle   string
	taskTitle    string
	taskID       string
	dependencies []string
}

// TasksOption is a functional option for CreateTempTasks
type TasksOption func(*tasksConfig)

// WithTaskStatus sets the task status
func WithTaskStatus(status string) TasksOption {
	return func(c *tasksConfig) {
		c.taskStatus = status
	}
}

// WithTotalTasks sets the total task count
func WithTotalTasks(count int) TasksOption {
	return func(c *tasksConfig) {
		c.totalTasks = count
	}
}

// WithTaskID sets the task ID
func WithTaskID(id string) TasksOption {
	return func(c *tasksConfig) {
		c.taskID = id
	}
}

// WithTaskTitle sets the task title
func WithTaskTitle(title string) TasksOption {
	return func(c *tasksConfig) {
		c.taskTitle = title
	}
}

// WithPhaseTitle sets the phase title
func WithPhaseTitle(title string) TasksOption {
	return func(c *tasksConfig) {
		c.phaseTitle = title
	}
}

// WithDependencies sets the task dependencies
func WithDependencies(deps []string) TasksOption {
	return func(c *tasksConfig) {
		c.dependencies = deps
	}
}

// formatDependencies formats a string slice as YAML array
func formatDependencies(deps []string) string {
	if len(deps) == 0 {
		return "[]"
	}
	result := "["
	for i, dep := range deps {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf(`"%s"`, dep)
	}
	return result + "]"
}

// CreateTempDir creates a temporary directory with cleanup.
func CreateTempDir(t *testing.T, prefix string) string {
	t.Helper()

	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	return dir
}

// WriteFile writes content to a file, creating parent directories if needed.
func WriteFile(t *testing.T, path, content string) {
	t.Helper()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

// FileExists checks if a file exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ReadFile reads file content, failing the test on error.
func ReadFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}

	return string(content)
}

// apiKeyEnvVars lists all environment variables that could enable API calls.
// Clearing these makes it impossible for tests to accidentally make real API requests.
var apiKeyEnvVars = []string{
	// Anthropic/Claude
	"ANTHROPIC_API_KEY",
	"CLAUDE_API_KEY",
	// OpenAI
	"OPENAI_API_KEY",
	// Google/Gemini
	"GEMINI_API_KEY",
	"GOOGLE_API_KEY",
	// Generic
	"API_KEY",
}

// ClearAPIKeys clears all API key environment variables to prevent accidental
// real API calls during tests. Returns a cleanup function to restore original values.
// Usage: defer testutil.ClearAPIKeys(t)()
func ClearAPIKeys(t *testing.T) func() {
	t.Helper()

	// Save original values
	originals := make(map[string]string)
	for _, key := range apiKeyEnvVars {
		if val, exists := os.LookupEnv(key); exists {
			originals[key] = val
		}
	}

	// Clear all API keys
	for _, key := range apiKeyEnvVars {
		t.Setenv(key, "")
	}

	// Return cleanup function
	return func() {
		for key, val := range originals {
			os.Setenv(key, val)
		}
	}
}
