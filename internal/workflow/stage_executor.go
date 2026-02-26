// Package workflow provides stage execution functionality.
// StageExecutor handles specify, plan, and tasks stage execution.
// Related: internal/workflow/orchestrator.go, internal/workflow/interfaces.go (interface definition)
// Tags: workflow, stage-executor, specify, plan, tasks
package workflow

import (
	"fmt"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/commands"
	"github.com/ariel-frischer/autospec/internal/prereqs"
	"github.com/ariel-frischer/autospec/internal/retry"
	"github.com/ariel-frischer/autospec/internal/spec"
)

// StageExecutor handles specify, plan, and tasks stage execution.
// It implements StageExecutorInterface to enable dependency injection and testing.
// Each stage transforms artifacts: specify creates spec.yaml, plan creates plan.yaml,
// tasks creates tasks.yaml.
type StageExecutor struct {
	executor               *Executor // Underlying executor for Claude command execution
	specsDir               string    // Base directory for spec storage (e.g., "specs/")
	debug                  bool      // Enable debug logging
	enableRiskAssessment   bool      // Inject risk assessment instructions in plan command
	enableEarsRequirements bool      // Inject EARS requirements instructions in specify command
}

// StageExecutorOptions holds optional configuration for StageExecutor.
type StageExecutorOptions struct {
	Debug                  bool // Enable debug logging
	EnableRiskAssessment   bool // Inject risk assessment instructions in plan command
	EnableEarsRequirements bool // Inject EARS requirements instructions in specify command
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

// NewStageExecutorWithOptions creates a StageExecutor with additional options.
func NewStageExecutorWithOptions(executor *Executor, specsDir string, opts StageExecutorOptions) *StageExecutor {
	return &StageExecutor{
		executor:               executor,
		specsDir:               specsDir,
		debug:                  opts.Debug,
		enableRiskAssessment:   opts.EnableRiskAssessment,
		enableEarsRequirements: opts.EnableEarsRequirements,
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
	command = InjectEarsInstructions(command, s.enableEarsRequirements)
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

	command, err := s.buildRenderedPlanCommand(prompt)
	if err != nil {
		return fmt.Errorf("building plan command: %w", err)
	}
	specDir := filepath.Join(s.specsDir, specName)

	result, err := s.executor.ExecuteStage(specName, StagePlan, command, ValidatePlanSchema)
	if err != nil {
		return s.formatStageError("plan", result, err)
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

	command, err := s.buildRenderedTasksCommand(prompt)
	if err != nil {
		return fmt.Errorf("building tasks command: %w", err)
	}

	result, err := s.executor.ExecuteStage(specName, StageTasks, command, ValidateTasksSchema)
	if err != nil {
		return s.formatStageError("tasks", result, err)
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

// buildRenderedPlanCommand renders the plan template and appends optional prompt.
func (s *StageExecutor) buildRenderedPlanCommand(prompt string) (string, error) {
	rendered, err := s.computeAndRenderCommand("autospec.plan")
	if err != nil {
		return "", err
	}
	command := InjectRiskAssessment(rendered, s.enableRiskAssessment)
	if prompt != "" {
		command = fmt.Sprintf("%s\n\n## User Input\n\n%s", command, prompt)
	}
	return command, nil
}

// buildRenderedTasksCommand renders the tasks template and appends optional prompt.
func (s *StageExecutor) buildRenderedTasksCommand(prompt string) (string, error) {
	rendered, err := s.computeAndRenderCommand("autospec.tasks")
	if err != nil {
		return "", err
	}
	if prompt != "" {
		return fmt.Sprintf("%s\n\n## User Input\n\n%s", rendered, prompt), nil
	}
	return rendered, nil
}

// formatStageError formats an error from a stage execution.
func (s *StageExecutor) formatStageError(stageName string, result *StageResult, err error) error {
	totalAttempts := result.RetryCount + 1
	if result.Exhausted {
		return fmt.Errorf("%s stage exhausted retries after %d total attempts: %w",
			stageName, totalAttempts, err)
	}
	return fmt.Errorf("%s failed after %d total attempts (%d retries): %w",
		stageName, totalAttempts, result.RetryCount, err)
}

// buildRenderedAuxCommand renders an auxiliary command template (clarify, analyze, checklist).
func (s *StageExecutor) buildRenderedAuxCommand(commandName, prompt string) (string, error) {
	rendered, err := s.computeAndRenderCommand(commandName)
	if err != nil {
		return "", err
	}
	if prompt != "" {
		return fmt.Sprintf("%s\n\n## User Input\n\n%s", rendered, prompt), nil
	}
	return rendered, nil
}

// ExecuteConstitution runs the constitution stage with optional prompt.
// Constitution creates or updates the project constitution file.
func (s *StageExecutor) ExecuteConstitution(prompt string) error {
	s.debugLog("ExecuteConstitution called with prompt: %s", prompt)

	command := s.buildCommand("/autospec.constitution", prompt)
	s.printExecuting("/autospec.constitution", prompt)

	// Derive project directory from specsDir (parent of specs/)
	// specsDir is typically "specs" or an absolute path like "/tmp/xyz/specs"
	projectDir := filepath.Dir(s.specsDir)
	if projectDir == "." || projectDir == "" {
		// If specsDir is "specs", parent is "." which is the project root
		projectDir = "."
	}

	result, err := s.executor.ExecuteStage(
		"", // No spec name needed for constitution
		StageConstitution,
		command,
		func(_ string) error {
			// Validate constitution file exists and has valid schema
			return s.executor.ValidateConstitution(projectDir)
		},
	)
	if err != nil {
		if result.Exhausted {
			return fmt.Errorf("constitution stage exhausted retries: %w", err)
		}
		return fmt.Errorf("constitution failed: %w", err)
	}

	fmt.Println("\n✓ Constitution created at .autospec/constitution.yaml")
	return nil
}

// ExecuteClarify runs the clarify stage with optional prompt.
// Clarify refines the specification by asking targeted clarification questions.
// This stage runs in interactive mode (no retry loop, multi-turn conversation).
func (s *StageExecutor) ExecuteClarify(specName string, prompt string) error {
	s.debugLog("ExecuteClarify called for spec: %s, prompt: %s", specName, prompt)

	command, err := s.buildRenderedAuxCommand("autospec.clarify", prompt)
	if err != nil {
		return fmt.Errorf("building clarify command: %w", err)
	}
	s.printExecuting("/autospec.clarify", prompt)

	// ExecuteStage automatically detects interactive mode via IsInteractive(StageClarify)
	// Interactive stages skip retry loop and run without -p flag
	_, err = s.executor.ExecuteStage(specName, StageClarify, command,
		func(specDir string) error { return nil }) // No validation for interactive stages
	if err != nil {
		return fmt.Errorf("clarify session failed: %w", err)
	}

	fmt.Printf("\n✓ Clarification session complete for specs/%s/\n", specName)
	return nil
}

// ExecuteChecklist runs the checklist stage with optional prompt.
// Checklist generates a custom checklist for the current feature.
func (s *StageExecutor) ExecuteChecklist(specName string, prompt string) error {
	s.debugLog("ExecuteChecklist called for spec: %s, prompt: %s", specName, prompt)

	command, err := s.buildRenderedAuxCommand("autospec.checklist", prompt)
	if err != nil {
		return fmt.Errorf("building checklist command: %w", err)
	}
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
// This stage runs in interactive mode (no retry loop, multi-turn conversation).
func (s *StageExecutor) ExecuteAnalyze(specName string, prompt string) error {
	s.debugLog("ExecuteAnalyze called for spec: %s, prompt: %s", specName, prompt)

	command, err := s.buildRenderedAuxCommand("autospec.analyze", prompt)
	if err != nil {
		return fmt.Errorf("building analyze command: %w", err)
	}
	s.printExecuting("/autospec.analyze", prompt)

	// ExecuteStage automatically detects interactive mode via IsInteractive(StageAnalyze)
	// Interactive stages skip retry loop and run without -p flag
	_, err = s.executor.ExecuteStage(specName, StageAnalyze, command,
		func(specDir string) error { return nil })
	if err != nil {
		return fmt.Errorf("analyze session failed: %w", err)
	}

	fmt.Printf("\n✓ Analysis session complete for specs/%s/\n", specName)
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

// getOptionsForStage returns prereqs.Options based on the stage's requirements.
// Maps each stage to the files it needs to exist for template rendering.
func (s *StageExecutor) getOptionsForStage(stageName string) prereqs.Options {
	requiredVars := commands.GetRequiredVars(stageName)
	opts := prereqs.Options{
		SpecsDir: s.specsDir,
	}

	for _, v := range requiredVars {
		switch v {
		case "FeatureSpec":
			opts.RequireSpec = true
		case "ImplPlan":
			opts.RequirePlan = true
		case "TasksFile":
			opts.RequireTasks = true
		}
	}

	if len(requiredVars) == 0 {
		opts.PathsOnly = true
	}

	return opts
}

// computeAndRenderCommand gets a command template and renders it with prereqs context.
// Returns the rendered command string ready for execution.
func (s *StageExecutor) computeAndRenderCommand(commandName string) (string, error) {
	content, err := commands.GetTemplate(commandName)
	if err != nil {
		return "", fmt.Errorf("loading template %s: %w", commandName, err)
	}

	opts := s.getOptionsForStage(commandName)
	ctx, err := prereqs.ComputeContext(opts)
	if err != nil {
		return "", fmt.Errorf("computing prereqs context: %w", err)
	}

	rendered, err := commands.RenderAndValidate(commandName, content, ctx)
	if err != nil {
		return "", fmt.Errorf("rendering template: %w", err)
	}

	return string(rendered), nil
}

// Compile-time interface compliance check.
var _ StageExecutorInterface = (*StageExecutor)(nil)
