// autospec - Spec-Driven Development Automation
// Author: Ariel Frischer
// Source: https://github.com/ariel-frischer/autospec

// Package workflow provides workflow orchestration for the autospec CLI.
// This file contains the WorkflowOrchestrator which coordinates between specialized
// executor components (StageExecutor, PhaseExecutor, TaskExecutor) for different workflow stages.
// Related: internal/workflow/stage_executor.go, internal/workflow/phase_executor.go, internal/workflow/task_executor.go
// Tags: workflow, orchestrator, coordination, delegation
package workflow

import (
	"fmt"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/ariel-frischer/autospec/internal/validation"
)

// WorkflowOrchestrator manages the complete specify → plan → tasks workflow.
// It coordinates between specialized executor components for different workflow stages.
// The orchestrator contains only coordination and delegation logic - all execution
// logic is delegated to the injected executor interfaces.
//
// Design: The orchestrator follows the Strategy pattern, delegating execution to
// specialized executor types. This enables:
// - Isolated unit testing with mock executors
// - Single responsibility (coordination only)
// - Easy extension for new execution strategies
type WorkflowOrchestrator struct {
	// Executor is the underlying command executor for Claude CLI invocations.
	Executor *Executor
	// Config holds the application configuration.
	Config *config.Configuration
	// SpecsDir is the base directory for spec storage (e.g., "specs/").
	SpecsDir string
	// SkipPreflight disables pre-flight checks when true.
	SkipPreflight bool
	// Debug enables debug logging when true.
	Debug bool
	// PreflightChecker is injectable for testing (nil uses default).
	PreflightChecker PreflightChecker

	// Executor interfaces for dependency injection.
	// These are always set by constructors - never nil during normal operation.
	stageExecutor StageExecutorInterface // Handles specify, plan, tasks stages
	phaseExecutor PhaseExecutorInterface // Handles phase-based implementation
	taskExecutor  TaskExecutorInterface  // Handles task-level implementation
}

// debugLog prints a debug message if debug mode is enabled
func (w *WorkflowOrchestrator) debugLog(format string, args ...interface{}) {
	if w.Debug {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// NewWorkflowOrchestrator creates a new workflow orchestrator from configuration.
// This constructor creates default implementations for all executor interfaces,
// ensuring the orchestrator always delegates to specialized executors.
// For dependency injection (testing), use NewWorkflowOrchestratorWithExecutors.
//
// Component wiring:
// - ClaudeExecutor implements ClaudeRunner interface for command execution
// - ProgressController wraps nil display (CLI commands don't provide progress display)
// - NotifyDispatcher wraps nil handler (CLI commands set handler via deprecated field)
//
// Note: CLI commands typically set Executor.NotificationHandler after construction.
// The Executor methods support both new controllers and deprecated fields via fallback.
func NewWorkflowOrchestrator(cfg *config.Configuration) *WorkflowOrchestrator {
	// Create ClaudeExecutor as ClaudeRunner interface implementation
	claude := &ClaudeExecutor{
		ClaudeCmd:       cfg.ClaudeCmd,
		ClaudeArgs:      cfg.ClaudeArgs,
		CustomClaudeCmd: cfg.CustomClaudeCmd,
		Timeout:         cfg.Timeout,
	}

	// Create ProgressController with nil display (no-op, CLI commands don't use progress display)
	progressCtrl := NewProgressController(nil)

	// Create NotifyDispatcher with nil handler (CLI commands set handler via deprecated field)
	notifyDispatch := NewNotifyDispatcher(nil)

	executor := &Executor{
		Claude:      claude,
		StateDir:    cfg.StateDir,
		SpecsDir:    cfg.SpecsDir,
		MaxRetries:  cfg.MaxRetries,
		TotalStages: 3,     // Default to 3 stages (specify, plan, tasks)
		Debug:       false, // Will be set by CLI command
		Progress:    progressCtrl,
		Notify:      notifyDispatch,
	}

	// Create default executor implementations
	stageExec := NewStageExecutor(executor, cfg.SpecsDir, false)
	phaseExec := NewPhaseExecutor(executor, cfg.SpecsDir, false)
	taskExec := NewTaskExecutor(executor, cfg.SpecsDir, false)

	return &WorkflowOrchestrator{
		Executor:      executor,
		Config:        cfg,
		SpecsDir:      cfg.SpecsDir,
		SkipPreflight: cfg.SkipPreflight,
		stageExecutor: stageExec,
		phaseExecutor: phaseExec,
		taskExecutor:  taskExec,
	}
}

// ExecutorOptions holds optional executor interfaces for dependency injection.
// All fields are optional; nil values cause the orchestrator to use default implementations.
type ExecutorOptions struct {
	StageExecutor StageExecutorInterface
	PhaseExecutor PhaseExecutorInterface
	TaskExecutor  TaskExecutorInterface
}

// NewWorkflowOrchestratorWithExecutors creates a workflow orchestrator with injected executors.
// This constructor enables dependency injection for testing and modular composition.
// Pass nil for any executor to use the default implementation created by NewWorkflowOrchestrator.
//
// Example usage for testing:
//
//	mockStage := &MockStageExecutor{}
//	orch := NewWorkflowOrchestratorWithExecutors(cfg, ExecutorOptions{
//	    StageExecutor: mockStage,
//	})
func NewWorkflowOrchestratorWithExecutors(cfg *config.Configuration, opts ExecutorOptions) *WorkflowOrchestrator {
	orch := NewWorkflowOrchestrator(cfg)
	// Override with provided executors, keeping defaults for nil values
	if opts.StageExecutor != nil {
		orch.stageExecutor = opts.StageExecutor
	}
	if opts.PhaseExecutor != nil {
		orch.phaseExecutor = opts.PhaseExecutor
	}
	if opts.TaskExecutor != nil {
		orch.taskExecutor = opts.TaskExecutor
	}
	return orch
}

// RunCompleteWorkflow executes the full specify → plan → tasks workflow
func (w *WorkflowOrchestrator) RunCompleteWorkflow(featureDescription string) error {
	if err := w.runPreflightIfNeeded(); err != nil {
		return fmt.Errorf("preflight checks failed: %w", err)
	}

	specName, err := w.executeSpecifyPlanTasks(featureDescription, 3)
	if err != nil {
		return fmt.Errorf("executing specify-plan-tasks workflow: %w", err)
	}

	fmt.Println("Workflow completed successfully!")
	fmt.Printf("Spec: specs/%s/\n", specName)
	fmt.Println("Next: autospec implement")

	return nil
}

// RunFullWorkflow executes the complete specify → plan → tasks → implement workflow
func (w *WorkflowOrchestrator) RunFullWorkflow(featureDescription string, resume bool) error {
	// Set total stages for full workflow
	w.Executor.TotalStages = 4

	if err := w.runPreflightIfNeeded(); err != nil {
		return fmt.Errorf("preflight checks failed: %w", err)
	}

	// Execute specify → plan → tasks stages
	specName, err := w.executeSpecifyPlanTasks(featureDescription, 4)
	if err != nil {
		return fmt.Errorf("executing specify-plan-tasks workflow: %w", err)
	}

	// Execute implement stage
	if err := w.executeImplementStage(specName, featureDescription, resume); err != nil {
		return fmt.Errorf("executing implement stage: %w", err)
	}

	// Print success summary
	w.printFullWorkflowSummary(specName)
	return nil
}

// runPreflightIfNeeded runs preflight checks if enabled
func (w *WorkflowOrchestrator) runPreflightIfNeeded() error {
	if ShouldRunPreflightChecks(w.SkipPreflight) {
		return w.runPreflightChecks()
	}
	return nil
}

// executeSpecifyPlanTasks runs specify, plan, and tasks stages sequentially.
// Delegates to StageExecutor for all stage execution.
func (w *WorkflowOrchestrator) executeSpecifyPlanTasks(featureDescription string, totalStages int) (string, error) {
	// Stage 1: Specify
	fmt.Printf("[Stage 1/%d] Specify...\n", totalStages)
	fmt.Printf("Executing: /autospec.specify \"%s\"\n", featureDescription)

	specName, err := w.stageExecutor.ExecuteSpecify(featureDescription)
	if err != nil {
		return "", fmt.Errorf("specify stage failed: %w", err)
	}
	fmt.Printf("✓ Created specs/%s/spec.yaml\n\n", specName)

	// Stage 2: Plan
	fmt.Printf("[Stage 2/%d] Plan...\n", totalStages)
	fmt.Println("Executing: /autospec.plan")

	if err := w.stageExecutor.ExecutePlan(specName, ""); err != nil {
		return "", fmt.Errorf("plan stage failed: %w", err)
	}
	fmt.Printf("✓ Created specs/%s/plan.yaml\n\n", specName)

	// Stage 3: Tasks
	fmt.Printf("[Stage 3/%d] Tasks...\n", totalStages)
	fmt.Println("Executing: /autospec.tasks")

	if err := w.stageExecutor.ExecuteTasks(specName, ""); err != nil {
		return "", fmt.Errorf("tasks stage failed: %w", err)
	}
	fmt.Printf("✓ Created specs/%s/tasks.yaml\n\n", specName)

	return specName, nil
}

// executeImplementStage runs the implement stage with resume support.
// Delegates to PhaseExecutor.ExecuteDefault for execution.
func (w *WorkflowOrchestrator) executeImplementStage(specName, featureDescription string, resume bool) error {
	fmt.Println("[Stage 4/4] Implement...")
	specDir := filepath.Join(w.SpecsDir, specName)
	return w.phaseExecutor.ExecuteDefault(specName, specDir, "", resume)
}

// printFullWorkflowSummary prints the completion summary for full workflow
func (w *WorkflowOrchestrator) printFullWorkflowSummary(specName string) {
	fmt.Println("\n✓ All tasks completed!")
	fmt.Println()

	specDir := filepath.Join(w.SpecsDir, specName)
	tasksPath := validation.GetTasksFilePath(specDir)
	stats, statsErr := validation.GetTaskStats(tasksPath)
	if statsErr == nil && stats.TotalTasks > 0 {
		fmt.Println("Task Summary:")
		fmt.Print(validation.FormatTaskSummary(stats))
		fmt.Println()
	}

	// Mark spec as completed
	markSpecCompletedAndPrint(specDir)

	fmt.Println("Completed 4 workflow stage(s): specify → plan → tasks → implement")
	fmt.Printf("Spec: specs/%s/\n", specName)
	w.debugLog("RunFullWorkflow exiting normally")
}

// runPreflightChecks runs pre-flight validation and handles user interaction.
// Uses the injected PreflightChecker if present, otherwise uses the default implementation.
func (w *WorkflowOrchestrator) runPreflightChecks() error {
	fmt.Println("Running pre-flight checks...")

	// Use injected checker or default
	checker := w.getPreflightChecker()

	result, err := checker.RunChecks()
	if err != nil {
		return fmt.Errorf("pre-flight checks failed: %w", err)
	}

	if !result.Passed {
		if len(result.FailedChecks) > 0 {
			for _, check := range result.FailedChecks {
				fmt.Printf("✗ %s\n", check)
			}
		}

		if result.WarningMessage != "" {
			// Prompt user to continue
			shouldContinue, err := checker.PromptUser(result.WarningMessage)
			if err != nil {
				return fmt.Errorf("prompting user to continue: %w", err)
			}
			if !shouldContinue {
				return fmt.Errorf("pre-flight checks failed, user aborted")
			}
		} else {
			// Critical failures (missing CLI tools)
			return fmt.Errorf("pre-flight checks failed")
		}
	} else {
		fmt.Println("✓ claude CLI found")
		fmt.Println("✓ specify CLI found")
		fmt.Println("✓ .claude/commands/ directory exists")
		fmt.Println("✓ .autospec/ directory exists")
	}

	fmt.Println()
	return nil
}

// getPreflightChecker returns the injected PreflightChecker or a default one.
// This ensures nil-safety: existing code works unchanged with nil checker.
func (w *WorkflowOrchestrator) getPreflightChecker() PreflightChecker {
	if w.PreflightChecker != nil {
		return w.PreflightChecker
	}
	return NewDefaultPreflightChecker()
}

// resolveSpecName resolves the spec name from argument or auto-detection.
func (w *WorkflowOrchestrator) resolveSpecName(specNameArg string) (string, error) {
	if specNameArg != "" {
		return specNameArg, nil
	}

	// Auto-detect current spec
	metadata, err := spec.DetectCurrentSpec(w.SpecsDir)
	if err != nil {
		return "", fmt.Errorf("detecting current spec: %w", err)
	}

	return fmt.Sprintf("%s-%s", metadata.Number, metadata.Name), nil
}

// ExecuteSpecify runs only the specify stage.
// Delegates to the StageExecutor for execution.
func (w *WorkflowOrchestrator) ExecuteSpecify(featureDescription string) (string, error) {
	fmt.Printf("Executing: /autospec.specify \"%s\"\n", featureDescription)

	specName, err := w.stageExecutor.ExecuteSpecify(featureDescription)
	if err != nil {
		return "", err
	}

	fmt.Printf("✓ Created specs/%s/spec.yaml\n\n", specName)
	fmt.Println("Next: autospec plan")

	return specName, nil
}

// ExecutePlan runs only the plan stage for a detected or specified spec.
// Delegates to the StageExecutor for execution.
func (w *WorkflowOrchestrator) ExecutePlan(specNameArg string, prompt string) error {
	specName, err := w.resolveSpecName(specNameArg)
	if err != nil {
		return fmt.Errorf("resolving spec name: %w", err)
	}

	if prompt != "" {
		fmt.Printf("Executing: /autospec.plan \"%s\"\n", prompt)
	} else {
		fmt.Println("Executing: /autospec.plan")
	}

	if err := w.stageExecutor.ExecutePlan(specName, prompt); err != nil {
		return fmt.Errorf("executing plan stage: %w", err)
	}

	fmt.Printf("✓ Created specs/%s/plan.yaml\n\n", specName)
	fmt.Println("Next: autospec tasks")

	return nil
}

// ExecuteTasks runs only the tasks stage for a detected or specified spec.
// Delegates to the StageExecutor for execution.
func (w *WorkflowOrchestrator) ExecuteTasks(specNameArg string, prompt string) error {
	specName, err := w.resolveSpecName(specNameArg)
	if err != nil {
		return fmt.Errorf("resolving spec name: %w", err)
	}

	if prompt != "" {
		fmt.Printf("Executing: /autospec.tasks \"%s\"\n", prompt)
	} else {
		fmt.Println("Executing: /autospec.tasks")
	}

	if err := w.stageExecutor.ExecuteTasks(specName, prompt); err != nil {
		return fmt.Errorf("executing tasks stage: %w", err)
	}

	fmt.Printf("✓ Created specs/%s/tasks.yaml\n\n", specName)
	fmt.Println("Next: autospec implement")

	return nil
}

// ExecuteImplement runs the implementation stage with optional prompt
func (w *WorkflowOrchestrator) ExecuteImplement(specNameArg string, prompt string, resume bool, phaseOpts PhaseExecutionOptions) error {
	var specName string
	var metadata *spec.Metadata
	var err error

	if specNameArg != "" {
		specName = specNameArg
		// Load metadata for this spec
		metadata, err = spec.GetSpecMetadata(w.SpecsDir, specName)
		if err != nil {
			return fmt.Errorf("failed to load spec metadata: %w", err)
		}
	} else {
		// Auto-detect current spec
		metadata, err = spec.DetectCurrentSpec(w.SpecsDir)
		if err != nil {
			return fmt.Errorf("failed to detect current spec: %w", err)
		}
		// Use full spec directory name (e.g., "003-command-timeout")
		specName = fmt.Sprintf("%s-%s", metadata.Number, metadata.Name)
	}

	// Dispatch to appropriate execution mode based on phase options
	switch phaseOpts.Mode() {
	case ModeAllTasks:
		return w.ExecuteImplementWithTasks(specName, metadata, prompt, phaseOpts.FromTask)
	case ModeAllPhases:
		return w.ExecuteImplementWithPhases(specName, metadata, prompt, resume)
	case ModeSinglePhase:
		return w.ExecuteImplementSinglePhase(specName, metadata, prompt, phaseOpts.SinglePhase)
	case ModeFromPhase:
		return w.ExecuteImplementFromPhase(specName, metadata, prompt, phaseOpts.FromPhase)
	default:
		// Default mode: single session (backward compatible)
		return w.executeImplementDefault(specName, metadata, prompt, resume)
	}
}

// executeImplementDefault executes implementation in a single Claude session (backward compatible).
// Delegates to PhaseExecutor.ExecuteDefault for execution.
func (w *WorkflowOrchestrator) executeImplementDefault(specName string, metadata *spec.Metadata, prompt string, resume bool) error {
	return w.phaseExecutor.ExecuteDefault(specName, metadata.Directory, prompt, resume)
}

// ExecuteImplementWithPhases runs each phase in a separate Claude session.
// Delegates to PhaseExecutor for execution.
func (w *WorkflowOrchestrator) ExecuteImplementWithPhases(specName string, metadata *spec.Metadata, prompt string, resume bool) error {
	tasksPath := validation.GetTasksFilePath(filepath.Join(w.SpecsDir, specName))
	phases, err := validation.GetPhaseInfo(tasksPath)
	if err != nil {
		return fmt.Errorf("getting phase info: %w", err)
	}
	if len(phases) == 0 {
		return fmt.Errorf("no phases found in tasks.yaml")
	}
	firstIncomplete, _, err := validation.GetFirstIncompletePhase(tasksPath)
	if err != nil {
		return fmt.Errorf("checking phase completion: %w", err)
	}
	if firstIncomplete == 0 {
		fmt.Println("✓ All phases already complete!")
		return nil
	}
	if firstIncomplete > 1 {
		fmt.Printf("Phases 1-%d complete, starting from phase %d\n\n", firstIncomplete-1, firstIncomplete)
	}
	return w.phaseExecutor.ExecutePhaseLoop(specName, tasksPath, phases, firstIncomplete, len(phases), prompt)
}

// ExecuteImplementSinglePhase runs only a specific phase. Delegates to PhaseExecutor.
func (w *WorkflowOrchestrator) ExecuteImplementSinglePhase(specName string, metadata *spec.Metadata, prompt string, phaseNumber int) error {
	tasksPath := validation.GetTasksFilePath(filepath.Join(w.SpecsDir, specName))
	totalPhases, err := validation.GetTotalPhases(tasksPath)
	if err != nil {
		return fmt.Errorf("getting total phases: %w", err)
	}
	if phaseNumber < 1 || phaseNumber > totalPhases {
		return fmt.Errorf("phase %d is out of range (valid: 1-%d)", phaseNumber, totalPhases)
	}
	return w.phaseExecutor.ExecuteSinglePhase(specName, phaseNumber, prompt)
}

// ExecuteImplementFromPhase runs phases starting from the specified phase. Delegates to PhaseExecutor.
func (w *WorkflowOrchestrator) ExecuteImplementFromPhase(specName string, metadata *spec.Metadata, prompt string, startPhase int) error {
	tasksPath := validation.GetTasksFilePath(filepath.Join(w.SpecsDir, specName))
	totalPhases, err := validation.GetTotalPhases(tasksPath)
	if err != nil {
		return fmt.Errorf("getting total phases: %w", err)
	}
	if startPhase < 1 || startPhase > totalPhases {
		return fmt.Errorf("phase %d is out of range (valid: 1-%d)", startPhase, totalPhases)
	}
	phases, err := validation.GetPhaseInfo(tasksPath)
	if err != nil {
		return fmt.Errorf("getting phase info: %w", err)
	}
	fmt.Printf("Starting from phase %d of %d\n\n", startPhase, totalPhases)
	return w.phaseExecutor.ExecutePhaseLoop(specName, tasksPath, phases, startPhase, totalPhases, prompt)
}

// ExecuteImplementWithTasks runs each task in a separate Claude session.
// Delegates to TaskExecutor for execution.
func (w *WorkflowOrchestrator) ExecuteImplementWithTasks(specName string, metadata *spec.Metadata, prompt string, fromTask string) error {
	specDir := filepath.Join(w.SpecsDir, specName)
	tasksPath := validation.GetTasksFilePath(specDir)

	orderedTasks, startIdx, totalTasks, err := w.taskExecutor.PrepareTaskExecution(tasksPath, fromTask)
	if err != nil {
		return fmt.Errorf("preparing task execution: %w", err)
	}

	if startIdx > 0 {
		fmt.Printf("Starting from task %s (task %d of %d)\n\n", fromTask, startIdx+1, totalTasks)
	}

	return w.taskExecutor.ExecuteTaskLoop(specName, tasksPath, orderedTasks, startIdx, totalTasks, prompt)
}

// markSpecCompletedAndPrint marks the spec as completed and prints the result.
// This is a package-level function used by executors for consistent completion marking.
func markSpecCompletedAndPrint(specDir string) {
	result, err := spec.MarkSpecCompleted(specDir)
	if err != nil {
		fmt.Printf("Warning: could not update spec.yaml status: %v\n", err)
		return
	}

	if result.Updated {
		fmt.Printf("Updated spec.yaml: %s → %s\n", result.PreviousStatus, result.NewStatus)
	}
}

// ExecuteConstitution runs the constitution stage with optional prompt.
// Delegates to StageExecutor for execution.
func (w *WorkflowOrchestrator) ExecuteConstitution(prompt string) error {
	return w.stageExecutor.ExecuteConstitution(prompt)
}

// ExecuteClarify runs the clarify stage with optional prompt.
// Delegates to StageExecutor for execution.
func (w *WorkflowOrchestrator) ExecuteClarify(specNameArg string, prompt string) error {
	specName, err := w.resolveSpecName(specNameArg)
	if err != nil {
		return fmt.Errorf("resolving spec name: %w", err)
	}
	return w.stageExecutor.ExecuteClarify(specName, prompt)
}

// ExecuteChecklist runs the checklist stage with optional prompt.
// Delegates to StageExecutor for execution.
func (w *WorkflowOrchestrator) ExecuteChecklist(specNameArg string, prompt string) error {
	specName, err := w.resolveSpecName(specNameArg)
	if err != nil {
		return fmt.Errorf("resolving spec name: %w", err)
	}
	return w.stageExecutor.ExecuteChecklist(specName, prompt)
}

// ExecuteAnalyze runs the analyze stage with optional prompt.
// Delegates to StageExecutor for execution.
func (w *WorkflowOrchestrator) ExecuteAnalyze(specNameArg string, prompt string) error {
	specName, err := w.resolveSpecName(specNameArg)
	if err != nil {
		return fmt.Errorf("resolving spec name: %w", err)
	}
	return w.stageExecutor.ExecuteAnalyze(specName, prompt)
}
