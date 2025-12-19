// Package workflow provides stage execution functionality.
// StageExecutor handles specify, plan, and tasks stage execution.
// Related: internal/workflow/orchestrator.go, internal/workflow/interfaces.go (interface definition)
// Tags: workflow, stage-executor, specify, plan, tasks
package workflow

import (
	"fmt"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/retry"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/ariel-frischer/autospec/internal/validation"
)

// StageExecutor handles specify, plan, and tasks stage execution.
// It implements StageExecutorInterface to enable dependency injection and testing.
// Each stage transforms artifacts: specify creates spec.yaml, plan creates plan.yaml,
// tasks creates tasks.yaml.
type StageExecutor struct {
	executor *Executor // Underlying executor for Claude command execution
	specsDir string    // Base directory for spec storage (e.g., "specs/")
	debug    bool      // Enable debug logging
}

// NewStageExecutor creates a new StageExecutor with the given dependencies.
// executor: required, handles actual command execution with retry logic
// specsDir: required, base directory where spec directories are created
// debug: optional, enables verbose logging for troubleshooting
func NewStageExecutor(executor *Executor, specsDir string, debug bool) *StageExecutor {
	return &StageExecutor{
		executor: executor,
		specsDir: specsDir,
		debug:    debug,
	}
}

// debugLog prints a debug message if debug mode is enabled.
func (s *StageExecutor) debugLog(format string, args ...interface{}) {
	if s.debug {
		fmt.Printf("[DEBUG][StageExecutor] "+format+"\n", args...)
	}
}

// ExecuteSpecify runs the specify stage for a feature description.
// Returns the spec name (e.g., "003-command-timeout") on success.
// The spec name is derived from the newly created spec directory.
func (s *StageExecutor) ExecuteSpecify(featureDescription string) (string, error) {
	s.debugLog("ExecuteSpecify called with description: %s", featureDescription)
	s.resetSpecifyRetryState()

	result, err := s.runSpecifyStage(featureDescription)
	if err != nil {
		return "", s.formatSpecifyError(result, err)
	}

	return s.detectAndValidateSpec()
}

// resetSpecifyRetryState clears retry state before a new specify run
func (s *StageExecutor) resetSpecifyRetryState() {
	if err := retry.ResetRetryCount(s.executor.StateDir, "", string(StageSpecify)); err != nil {
		s.debugLog("Warning: failed to reset specify retry state: %v", err)
	}
}

// runSpecifyStage executes the specify stage command
func (s *StageExecutor) runSpecifyStage(featureDescription string) (*StageResult, error) {
	command := fmt.Sprintf("/autospec.specify \"%s\"", featureDescription)
	validateFunc := MakeSpecSchemaValidatorWithDetection(s.specsDir)
	return s.executor.ExecuteStage("", StageSpecify, command, validateFunc)
}

// formatSpecifyError formats an error from the specify stage
func (s *StageExecutor) formatSpecifyError(result *StageResult, err error) error {
	totalAttempts := result.RetryCount + 1
	return fmt.Errorf("specify failed after %d total attempts (%d retries): %w",
		totalAttempts, result.RetryCount, err)
}

// detectAndValidateSpec detects and validates the newly created spec
func (s *StageExecutor) detectAndValidateSpec() (string, error) {
	metadata, err := spec.DetectCurrentSpec(s.specsDir)
	if err != nil {
		return "", fmt.Errorf("detecting created spec: %w", err)
	}
	if err := s.executor.ValidateSpec(metadata.Directory); err != nil {
		return "", fmt.Errorf("validating spec: %w", err)
	}
	specName := fmt.Sprintf("%s-%s", metadata.Number, metadata.Name)
	s.debugLog("ExecuteSpecify completed successfully: %s", specName)
	return specName, nil
}

// ExecutePlan runs the plan stage for an existing spec.
// specNameArg: spec name or empty string to auto-detect from git branch
// prompt: optional custom prompt to pass to the plan command
func (s *StageExecutor) ExecutePlan(specNameArg string, prompt string) error {
	specName, err := s.resolveSpecName(specNameArg)
	if err != nil {
		return fmt.Errorf("resolving spec name: %w", err)
	}

	s.debugLog("ExecutePlan called for spec: %s, prompt: %s", specName, prompt)

	command := s.buildPlanCommand(prompt)
	specDir := filepath.Join(s.specsDir, specName)

	result, err := s.executor.ExecuteStage(
		specName,
		StagePlan,
		command,
		ValidatePlanSchema,
	)

	if err != nil {
		totalAttempts := result.RetryCount + 1
		if result.Exhausted {
			return fmt.Errorf("plan stage exhausted retries after %d total attempts: %w",
				totalAttempts, err)
		}
		return fmt.Errorf("plan failed after %d total attempts (%d retries): %w",
			totalAttempts, result.RetryCount, err)
	}

	// Check for research.md (optional but usually created)
	researchPath := filepath.Join(specDir, "research.md")
	if _, statErr := filepath.Glob(researchPath); statErr == nil {
		s.debugLog("Research file exists at: %s", researchPath)
	}

	s.debugLog("ExecutePlan completed successfully")
	return nil
}

// ExecuteTasks runs the tasks stage for an existing spec.
// specNameArg: spec name or empty string to auto-detect from git branch
// prompt: optional custom prompt to pass to the tasks command
func (s *StageExecutor) ExecuteTasks(specNameArg string, prompt string) error {
	specName, err := s.resolveSpecName(specNameArg)
	if err != nil {
		return fmt.Errorf("resolving spec name: %w", err)
	}

	s.debugLog("ExecuteTasks called for spec: %s, prompt: %s", specName, prompt)

	command := s.buildTasksCommand(prompt)

	result, err := s.executor.ExecuteStage(
		specName,
		StageTasks,
		command,
		ValidateTasksSchema,
	)

	if err != nil {
		totalAttempts := result.RetryCount + 1
		if result.Exhausted {
			return fmt.Errorf("tasks stage exhausted retries after %d total attempts: %w",
				totalAttempts, err)
		}
		return fmt.Errorf("tasks failed after %d total attempts (%d retries): %w",
			totalAttempts, result.RetryCount, err)
	}

	s.debugLog("ExecuteTasks completed successfully")
	return nil
}

// resolveSpecName resolves the spec name from argument or auto-detection.
func (s *StageExecutor) resolveSpecName(specNameArg string) (string, error) {
	if specNameArg != "" {
		return specNameArg, nil
	}

	// Auto-detect current spec
	metadata, err := spec.DetectCurrentSpec(s.specsDir)
	if err != nil {
		return "", fmt.Errorf("detecting current spec: %w", err)
	}

	return fmt.Sprintf("%s-%s", metadata.Number, metadata.Name), nil
}

// buildPlanCommand constructs the plan command with optional prompt.
func (s *StageExecutor) buildPlanCommand(prompt string) string {
	if prompt != "" {
		return fmt.Sprintf("/autospec.plan \"%s\"", prompt)
	}
	return "/autospec.plan"
}

// buildTasksCommand constructs the tasks command with optional prompt.
func (s *StageExecutor) buildTasksCommand(prompt string) string {
	if prompt != "" {
		return fmt.Sprintf("/autospec.tasks \"%s\"", prompt)
	}
	return "/autospec.tasks"
}

// ExecuteConstitution runs the constitution stage with optional prompt.
// Constitution creates or updates the project constitution file.
func (s *StageExecutor) ExecuteConstitution(prompt string) error {
	s.debugLog("ExecuteConstitution called with prompt: %s", prompt)

	command := s.buildCommand("/autospec.constitution", prompt)
	s.printExecuting("/autospec.constitution", prompt)

	result, err := s.executor.ExecuteStage(
		"", // No spec name needed for constitution
		StageConstitution,
		command,
		func(specDir string) error { return nil }, // Constitution doesn't produce tracked artifacts
	)

	if err != nil {
		if result.Exhausted {
			return fmt.Errorf("constitution stage exhausted retries: %w", err)
		}
		return fmt.Errorf("constitution failed: %w", err)
	}

	fmt.Println("\n✓ Constitution updated!")
	return nil
}

// ExecuteClarify runs the clarify stage with optional prompt.
// Clarify refines the specification by asking targeted clarification questions.
func (s *StageExecutor) ExecuteClarify(specName string, prompt string) error {
	s.debugLog("ExecuteClarify called for spec: %s, prompt: %s", specName, prompt)

	command := s.buildCommand("/autospec.clarify", prompt)
	s.printExecuting("/autospec.clarify", prompt)

	result, err := s.executor.ExecuteStage(specName, StageClarify, command,
		func(specDir string) error { return validation.ValidateSpecFile(specDir) })

	if err != nil {
		if result.Exhausted {
			return fmt.Errorf("clarify stage exhausted retries: %w", err)
		}
		return fmt.Errorf("clarify failed: %w", err)
	}

	fmt.Printf("\n✓ Clarification complete for specs/%s/\n", specName)
	return nil
}

// ExecuteChecklist runs the checklist stage with optional prompt.
// Checklist generates a custom checklist for the current feature.
func (s *StageExecutor) ExecuteChecklist(specName string, prompt string) error {
	s.debugLog("ExecuteChecklist called for spec: %s, prompt: %s", specName, prompt)

	command := s.buildCommand("/autospec.checklist", prompt)
	s.printExecuting("/autospec.checklist", prompt)

	result, err := s.executor.ExecuteStage(specName, StageChecklist, command,
		func(specDir string) error { return nil })

	if err != nil {
		if result.Exhausted {
			return fmt.Errorf("checklist stage exhausted retries: %w", err)
		}
		return fmt.Errorf("checklist failed: %w", err)
	}

	fmt.Printf("\n✓ Checklist generated for specs/%s/\n", specName)
	return nil
}

// ExecuteAnalyze runs the analyze stage with optional prompt.
// Analyze performs cross-artifact consistency and quality analysis.
func (s *StageExecutor) ExecuteAnalyze(specName string, prompt string) error {
	s.debugLog("ExecuteAnalyze called for spec: %s, prompt: %s", specName, prompt)

	command := s.buildCommand("/autospec.analyze", prompt)
	s.printExecuting("/autospec.analyze", prompt)

	result, err := s.executor.ExecuteStage(specName, StageAnalyze, command,
		func(specDir string) error { return nil })

	if err != nil {
		if result.Exhausted {
			return fmt.Errorf("analyze stage exhausted retries: %w", err)
		}
		return fmt.Errorf("analyze failed: %w", err)
	}

	fmt.Printf("\n✓ Analysis complete for specs/%s/\n", specName)
	return nil
}

// buildCommand constructs a command with optional prompt.
func (s *StageExecutor) buildCommand(baseCmd, prompt string) string {
	if prompt != "" {
		return fmt.Sprintf("%s \"%s\"", baseCmd, prompt)
	}
	return baseCmd
}

// printExecuting prints the executing message for a command.
func (s *StageExecutor) printExecuting(baseCmd, prompt string) {
	if prompt != "" {
		fmt.Printf("Executing: %s \"%s\"\n", baseCmd, prompt)
	} else {
		fmt.Printf("Executing: %s\n", baseCmd)
	}
}

// Compile-time interface compliance check.
var _ StageExecutorInterface = (*StageExecutor)(nil)
