// autospec - Spec-Driven Development Automation
// Author: Ariel Frischer
// Source: https://github.com/ariel-frischer/autospec

package workflow

import (
	"fmt"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/progress"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/ariel-frischer/autospec/internal/validation"
)

// WorkflowOrchestrator manages the complete specify → plan → tasks workflow
type WorkflowOrchestrator struct {
	Executor      *Executor
	Config        *config.Configuration
	SpecsDir      string
	SkipPreflight bool
	Debug         bool // Enable debug logging
}

// debugLog prints a debug message if debug mode is enabled
func (w *WorkflowOrchestrator) debugLog(format string, args ...interface{}) {
	if w.Debug {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// NewWorkflowOrchestrator creates a new workflow orchestrator from configuration
func NewWorkflowOrchestrator(cfg *config.Configuration) *WorkflowOrchestrator {
	claude := &ClaudeExecutor{
		ClaudeCmd:       cfg.ClaudeCmd,
		ClaudeArgs:      cfg.ClaudeArgs,
		UseAPIKey:       cfg.UseAPIKey,
		CustomClaudeCmd: cfg.CustomClaudeCmd,
		Timeout:         cfg.Timeout,
	}

	// Detect terminal capabilities and create progress display (only if enabled)
	var progressDisplay *progress.ProgressDisplay
	if cfg.ShowProgress {
		caps := progress.DetectTerminalCapabilities()
		if caps.IsTTY {
			progressDisplay = progress.NewProgressDisplay(caps)
		}
	}

	executor := &Executor{
		Claude:          claude,
		StateDir:        cfg.StateDir,
		SpecsDir:        cfg.SpecsDir,
		MaxRetries:      cfg.MaxRetries,
		ProgressDisplay: progressDisplay,
		TotalPhases:     3,     // Default to 3 phases (specify, plan, tasks)
		Debug:           false, // Will be set by CLI command
	}

	return &WorkflowOrchestrator{
		Executor:      executor,
		Config:        cfg,
		SpecsDir:      cfg.SpecsDir,
		SkipPreflight: cfg.SkipPreflight,
	}
}

// RunCompleteWorkflow executes the full specify → plan → tasks workflow
func (w *WorkflowOrchestrator) RunCompleteWorkflow(featureDescription string) error {
	// Run pre-flight checks
	if ShouldRunPreflightChecks(w.SkipPreflight) {
		if err := w.runPreflightChecks(); err != nil {
			return err
		}
	}

	// Phase 1: Specify
	fmt.Println("[Phase 1/3] Specify...")
	fmt.Printf("Executing: /autospec.specify \"%s\"\n", featureDescription)

	specName, err := w.executeSpecify(featureDescription)
	if err != nil {
		return fmt.Errorf("specify phase failed: %w", err)
	}

	fmt.Printf("✓ Created specs/%s/spec.yaml\n\n", specName)

	// Phase 2: Plan
	fmt.Println("[Phase 2/3] Plan...")
	fmt.Println("Executing: /autospec.plan")

	if err := w.executePlan(specName, ""); err != nil {
		return fmt.Errorf("plan phase failed: %w", err)
	}

	fmt.Printf("✓ Created specs/%s/plan.yaml\n\n", specName)

	// Phase 3: Tasks
	fmt.Println("[Phase 3/3] Tasks...")
	fmt.Println("Executing: /autospec.tasks")

	if err := w.executeTasks(specName, ""); err != nil {
		return fmt.Errorf("tasks phase failed: %w", err)
	}

	fmt.Printf("✓ Created specs/%s/tasks.yaml\n\n", specName)

	// Success!
	fmt.Println("Workflow completed successfully!")
	fmt.Printf("Spec: specs/%s/\n", specName)
	fmt.Println("Next: autospec implement")

	return nil
}

// RunFullWorkflow executes the complete specify → plan → tasks → implement workflow
func (w *WorkflowOrchestrator) RunFullWorkflow(featureDescription string, resume bool) error {
	// Set total phases for full workflow
	w.Executor.TotalPhases = 4

	// Run pre-flight checks
	if ShouldRunPreflightChecks(w.SkipPreflight) {
		if err := w.runPreflightChecks(); err != nil {
			return err
		}
	}

	// Phase 1: Specify
	fmt.Println("[Phase 1/4] Specify...")
	fmt.Printf("Executing: /autospec.specify \"%s\"\n", featureDescription)

	specName, err := w.executeSpecify(featureDescription)
	if err != nil {
		return fmt.Errorf("specify phase failed: %w", err)
	}

	fmt.Printf("✓ Created specs/%s/spec.yaml\n\n", specName)

	// Phase 2: Plan
	fmt.Println("[Phase 2/4] Plan...")
	fmt.Println("Executing: /autospec.plan")

	if err := w.executePlan(specName, ""); err != nil {
		return fmt.Errorf("plan phase failed: %w", err)
	}

	fmt.Printf("✓ Created specs/%s/plan.yaml\n\n", specName)

	// Phase 3: Tasks
	fmt.Println("[Phase 3/4] Tasks...")
	fmt.Println("Executing: /autospec.tasks")

	if err := w.executeTasks(specName, ""); err != nil {
		return fmt.Errorf("tasks phase failed: %w", err)
	}

	fmt.Printf("✓ Created specs/%s/tasks.yaml\n\n", specName)

	// Phase 4: Implement
	fmt.Println("[Phase 4/4] Implement...")
	fmt.Println("Executing: /autospec.implement")
	w.debugLog("Starting implement phase for spec: %s", specName)

	command := "/autospec.implement"
	if resume {
		command += " --resume"
		w.debugLog("Resume flag enabled")
	}

	w.debugLog("Calling ExecutePhase with command: %s", command)
	result, err := w.Executor.ExecutePhase(
		specName,
		PhaseImplement,
		command,
		func(specDir string) error {
			w.debugLog("Running validation function for spec dir: %s", specDir)
			tasksPath := validation.GetTasksFilePath(specDir)
			w.debugLog("Validating tasks at: %s", tasksPath)
			validationErr := w.Executor.ValidateTasksComplete(tasksPath)
			w.debugLog("Validation result: %v", validationErr)
			return validationErr
		},
	)
	w.debugLog("ExecutePhase returned - result: %+v, err: %v", result, err)

	if err != nil {
		w.debugLog("Implement phase failed with error: %v", err)
		if result.Exhausted {
			w.debugLog("Retries exhausted")
			// Generate continuation prompt
			fmt.Println("\nImplementation paused.")
			fmt.Printf("To resume: autospec full \"%s\" --resume\n", featureDescription)
			return fmt.Errorf("implementation phase exhausted retries: %w", err)
		}
		return fmt.Errorf("implementation failed: %w", err)
	}

	// Success!
	w.debugLog("Implement phase completed successfully")
	fmt.Println("\n✓ All tasks completed!")
	fmt.Println()

	// Show task completion stats
	specDir := filepath.Join(w.SpecsDir, specName)
	tasksPath := validation.GetTasksFilePath(specDir)
	stats, statsErr := validation.GetTaskStats(tasksPath)
	if statsErr == nil && stats.TotalTasks > 0 {
		fmt.Println("Task Summary:")
		fmt.Print(validation.FormatTaskSummary(stats))
		fmt.Println()
	}

	fmt.Println("Completed 4 workflow phase(s): specify → plan → tasks → implement")
	fmt.Printf("Spec: specs/%s/\n", specName)
	w.debugLog("RunFullWorkflow exiting normally")

	return nil
}

// runPreflightChecks runs pre-flight validation and handles user interaction
func (w *WorkflowOrchestrator) runPreflightChecks() error {
	fmt.Println("Running pre-flight checks...")

	result, err := RunPreflightChecks()
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
			shouldContinue, err := PromptUserToContinue(result.WarningMessage)
			if err != nil {
				return err
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

// executeSpecify executes the /autospec.specify command and returns the spec name
func (w *WorkflowOrchestrator) executeSpecify(featureDescription string) (string, error) {
	command := fmt.Sprintf("/autospec.specify \"%s\"", featureDescription)

	// Execute with validation and retry
	result, err := w.Executor.ExecutePhase(
		"", // Spec name not known yet
		PhaseSpecify,
		command,
		func(specDir string) error {
			// After specify, we need to detect the newly created spec
			// For now, just check if any spec.md was created
			return nil
		},
	)

	if err != nil && !result.Exhausted {
		// Retry if not exhausted
		return "", fmt.Errorf("specify failed after %d attempts: %w", result.RetryCount, err)
	}

	// Detect the newly created spec
	metadata, err := spec.DetectCurrentSpec(w.SpecsDir)
	if err != nil {
		return "", fmt.Errorf("failed to detect created spec: %w", err)
	}

	// Validate spec.md exists
	if err := w.Executor.ValidateSpec(metadata.Directory); err != nil {
		return "", err
	}

	// Return full spec directory name (e.g., "003-command-timeout")
	return fmt.Sprintf("%s-%s", metadata.Number, metadata.Name), nil
}

// executePlan executes the /autospec.plan command with optional prompt
func (w *WorkflowOrchestrator) executePlan(specName string, prompt string) error {
	command := "/autospec.plan"
	if prompt != "" {
		command = fmt.Sprintf("/autospec.plan \"%s\"", prompt)
	}
	specDir := filepath.Join(w.SpecsDir, specName)

	result, err := w.Executor.ExecutePhase(
		specName,
		PhasePlan,
		command,
		w.Executor.ValidatePlan,
	)

	if err != nil {
		if result.Exhausted {
			return fmt.Errorf("plan phase exhausted retries: %w", err)
		}
		return fmt.Errorf("plan failed after %d attempts: %w", result.RetryCount, err)
	}

	// Also check for research.md (optional but usually created)
	researchPath := filepath.Join(specDir, "research.md")
	if _, statErr := filepath.Glob(researchPath); statErr == nil {
		// Research file exists, that's good
	}

	return nil
}

// executeTasks executes the /autospec.tasks command with optional prompt
func (w *WorkflowOrchestrator) executeTasks(specName string, prompt string) error {
	command := "/autospec.tasks"
	if prompt != "" {
		command = fmt.Sprintf("/autospec.tasks \"%s\"", prompt)
	}

	result, err := w.Executor.ExecutePhase(
		specName,
		PhaseTasks,
		command,
		w.Executor.ValidateTasks,
	)

	if err != nil {
		if result.Exhausted {
			return fmt.Errorf("tasks phase exhausted retries: %w", err)
		}
		return fmt.Errorf("tasks failed after %d attempts: %w", result.RetryCount, err)
	}

	return nil
}

// ExecuteSpecify runs only the specify phase
func (w *WorkflowOrchestrator) ExecuteSpecify(featureDescription string) (string, error) {
	fmt.Printf("Executing: /autospec.specify \"%s\"\n", featureDescription)

	specName, err := w.executeSpecify(featureDescription)
	if err != nil {
		return "", err
	}

	fmt.Printf("✓ Created specs/%s/spec.yaml\n\n", specName)
	fmt.Println("Next: autospec plan")

	return specName, nil
}

// ExecutePlan runs only the plan phase for a detected or specified spec
func (w *WorkflowOrchestrator) ExecutePlan(specNameArg string, prompt string) error {
	var specName string
	var err error

	if specNameArg != "" {
		specName = specNameArg
	} else {
		// Auto-detect current spec
		metadata, err := spec.DetectCurrentSpec(w.SpecsDir)
		if err != nil {
			return fmt.Errorf("failed to detect current spec: %w", err)
		}
		// Use full spec directory name (e.g., "003-command-timeout")
		specName = fmt.Sprintf("%s-%s", metadata.Number, metadata.Name)
		fmt.Printf("Detected spec: %s\n", specName)
	}

	if prompt != "" {
		fmt.Printf("Executing: /autospec.plan \"%s\"\n", prompt)
	} else {
		fmt.Println("Executing: /autospec.plan")
	}

	if err = w.executePlan(specName, prompt); err != nil {
		return err
	}

	fmt.Printf("✓ Created specs/%s/plan.yaml\n\n", specName)
	fmt.Println("Next: autospec tasks")

	return nil
}

// ExecuteTasks runs only the tasks phase for a detected or specified spec
func (w *WorkflowOrchestrator) ExecuteTasks(specNameArg string, prompt string) error {
	var specName string
	var err error

	if specNameArg != "" {
		specName = specNameArg
	} else {
		// Auto-detect current spec
		metadata, err := spec.DetectCurrentSpec(w.SpecsDir)
		if err != nil {
			return fmt.Errorf("failed to detect current spec: %w", err)
		}
		// Use full spec directory name (e.g., "003-command-timeout")
		specName = fmt.Sprintf("%s-%s", metadata.Number, metadata.Name)
		fmt.Printf("Detected spec: %s\n", specName)
	}

	if prompt != "" {
		fmt.Printf("Executing: /autospec.tasks \"%s\"\n", prompt)
	} else {
		fmt.Println("Executing: /autospec.tasks")
	}

	if err = w.executeTasks(specName, prompt); err != nil {
		return err
	}

	fmt.Printf("✓ Created specs/%s/tasks.yaml\n\n", specName)
	fmt.Println("Next: autospec implement")

	return nil
}

// ExecuteImplement runs the implementation phase with optional prompt
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
		fmt.Printf("Detected spec: %s\n", specName)
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

// executeImplementDefault executes implementation in a single Claude session (backward compatible)
func (w *WorkflowOrchestrator) executeImplementDefault(specName string, metadata *spec.Metadata, prompt string, resume bool) error {
	// Check progress
	fmt.Printf("Progress: checking tasks...\n\n")

	// Build command with optional prompt
	command := "/autospec.implement"
	if resume {
		command += " --resume"
	}
	if prompt != "" {
		command = fmt.Sprintf("/autospec.implement \"%s\"", prompt)
		if resume {
			// If both resume and prompt, append resume after prompt
			command = fmt.Sprintf("/autospec.implement --resume \"%s\"", prompt)
		}
	}

	if prompt != "" {
		fmt.Printf("Executing: /autospec.implement \"%s\"\n", prompt)
	} else {
		fmt.Println("Executing: /autospec.implement")
	}

	result, err := w.Executor.ExecutePhase(
		specName,
		PhaseImplement,
		command,
		func(specDir string) error {
			tasksPath := validation.GetTasksFilePath(specDir)
			return w.Executor.ValidateTasksComplete(tasksPath)
		},
	)

	if err != nil {
		if result.Exhausted {
			// Generate continuation prompt
			fmt.Println("\nImplementation paused.")
			fmt.Println("To resume: autospec implement --resume")
			return fmt.Errorf("implementation phase exhausted retries: %w", err)
		}
		return fmt.Errorf("implementation failed: %w", err)
	}

	// Show task completion stats
	fmt.Println("\n✓ All tasks completed!")
	fmt.Println()
	tasksPath := validation.GetTasksFilePath(metadata.Directory)
	stats, statsErr := validation.GetTaskStats(tasksPath)
	if statsErr == nil && stats.TotalTasks > 0 {
		fmt.Println("Task Summary:")
		fmt.Print(validation.FormatTaskSummary(stats))
	}

	return nil
}

// ExecuteImplementWithPhases runs each phase in a separate Claude session
func (w *WorkflowOrchestrator) ExecuteImplementWithPhases(specName string, metadata *spec.Metadata, prompt string, resume bool) error {
	specDir := filepath.Join(w.SpecsDir, specName)
	tasksPath := validation.GetTasksFilePath(specDir)

	// Get phase information
	phases, err := validation.GetPhaseInfo(tasksPath)
	if err != nil {
		return fmt.Errorf("failed to get phase info: %w", err)
	}

	if len(phases) == 0 {
		return fmt.Errorf("no phases found in tasks.yaml")
	}

	totalPhases := len(phases)

	// Find the first incomplete phase
	firstIncomplete, _, err := validation.GetFirstIncompletePhase(tasksPath)
	if err != nil {
		return fmt.Errorf("failed to check phase completion: %w", err)
	}

	// If all phases are complete, nothing to do
	if firstIncomplete == 0 {
		fmt.Println("✓ All phases already complete!")
		return nil
	}

	// Display skip message if starting from a later phase
	if firstIncomplete > 1 {
		fmt.Printf("Phases 1-%d complete, starting from phase %d\n\n", firstIncomplete-1, firstIncomplete)
	}

	// Execute each phase starting from the first incomplete
	for _, phase := range phases {
		if phase.Number < firstIncomplete {
			continue // Skip already complete phases
		}

		// Get tasks for this phase to display task IDs
		phaseTasks, taskErr := validation.GetTasksForPhase(tasksPath, phase.Number)
		taskIDs := make([]string, 0, len(phaseTasks))
		if taskErr == nil {
			for _, t := range phaseTasks {
				taskIDs = append(taskIDs, t.ID)
			}
		}

		// Display pre-phase summary with task count, IDs, and status
		displayInfo := validation.BuildPhaseDisplayInfo(phase, totalPhases, taskIDs)
		fmt.Println(validation.FormatPhaseHeader(displayInfo))

		// Execute this phase
		err := w.executeSinglePhaseSession(specName, phase.Number, prompt)
		if err != nil {
			return fmt.Errorf("phase %d failed: %w", phase.Number, err)
		}

		// Re-read phase info to get updated task counts
		updatedPhases, rereadErr := validation.GetPhaseInfo(tasksPath)
		var updatedPhase *validation.PhaseInfo
		if rereadErr == nil {
			for _, p := range updatedPhases {
				if p.Number == phase.Number {
					updatedPhase = &p
					break
				}
			}
		}

		// Verify phase completion
		complete, verifyErr := validation.IsPhaseComplete(tasksPath, phase.Number)
		if verifyErr != nil {
			return fmt.Errorf("failed to verify phase %d completion: %w", phase.Number, verifyErr)
		}

		if !complete {
			fmt.Printf("\n⚠ Phase %d has incomplete tasks. Run 'autospec implement --phase %d' to continue.\n", phase.Number, phase.Number)
			return fmt.Errorf("phase %d did not complete all tasks", phase.Number)
		}

		// Display completion message with task counts
		if updatedPhase != nil {
			fmt.Println(validation.FormatPhaseCompletion(phase.Number, updatedPhase.CompletedTasks, updatedPhase.TotalTasks, updatedPhase.BlockedTasks))
		} else {
			fmt.Printf("✓ Phase %d complete\n", phase.Number)
		}
		fmt.Println()
	}

	// Show final summary
	fmt.Println("✓ All phases completed!")
	fmt.Println()
	stats, statsErr := validation.GetTaskStats(tasksPath)
	if statsErr == nil && stats.TotalTasks > 0 {
		fmt.Println("Task Summary:")
		fmt.Print(validation.FormatTaskSummary(stats))
	}

	return nil
}

// ExecuteImplementSinglePhase runs only a specific phase
func (w *WorkflowOrchestrator) ExecuteImplementSinglePhase(specName string, metadata *spec.Metadata, prompt string, phaseNumber int) error {
	specDir := filepath.Join(w.SpecsDir, specName)
	tasksPath := validation.GetTasksFilePath(specDir)

	// Get total phases for validation
	totalPhases, err := validation.GetTotalPhases(tasksPath)
	if err != nil {
		return fmt.Errorf("failed to get total phases: %w", err)
	}

	if phaseNumber < 1 || phaseNumber > totalPhases {
		return fmt.Errorf("phase %d is out of range (valid: 1-%d)", phaseNumber, totalPhases)
	}

	// Get phase info
	phases, err := validation.GetPhaseInfo(tasksPath)
	if err != nil {
		return fmt.Errorf("failed to get phase info: %w", err)
	}

	var phaseInfo *validation.PhaseInfo
	for _, p := range phases {
		if p.Number == phaseNumber {
			phaseInfo = &p
			break
		}
	}

	if phaseInfo == nil {
		return fmt.Errorf("phase %d not found", phaseNumber)
	}

	// Get tasks for this phase to display task IDs
	phaseTasks, taskErr := validation.GetTasksForPhase(tasksPath, phaseNumber)
	taskIDs := make([]string, 0, len(phaseTasks))
	if taskErr == nil {
		for _, t := range phaseTasks {
			taskIDs = append(taskIDs, t.ID)
		}
	}

	// Display pre-phase summary with task count, IDs, and status
	displayInfo := validation.BuildPhaseDisplayInfo(*phaseInfo, totalPhases, taskIDs)
	fmt.Println(validation.FormatPhaseHeader(displayInfo))

	// Execute the specific phase
	err = w.executeSinglePhaseSession(specName, phaseNumber, prompt)
	if err != nil {
		return fmt.Errorf("phase %d failed: %w", phaseNumber, err)
	}

	// Re-read phase info to get updated task counts
	updatedPhases, rereadErr := validation.GetPhaseInfo(tasksPath)
	var updatedPhase *validation.PhaseInfo
	if rereadErr == nil {
		for _, p := range updatedPhases {
			if p.Number == phaseNumber {
				updatedPhase = &p
				break
			}
		}
	}

	// Verify phase completion
	complete, verifyErr := validation.IsPhaseComplete(tasksPath, phaseNumber)
	if verifyErr != nil {
		return fmt.Errorf("failed to verify phase %d completion: %w", phaseNumber, verifyErr)
	}

	if complete {
		if updatedPhase != nil {
			fmt.Println(validation.FormatPhaseCompletion(phaseNumber, updatedPhase.CompletedTasks, updatedPhase.TotalTasks, updatedPhase.BlockedTasks))
		} else {
			fmt.Printf("✓ Phase %d complete\n", phaseNumber)
		}
	} else {
		fmt.Printf("⚠ Phase %d has incomplete tasks\n", phaseNumber)
	}

	return nil
}

// ExecuteImplementFromPhase runs phases starting from the specified phase
func (w *WorkflowOrchestrator) ExecuteImplementFromPhase(specName string, metadata *spec.Metadata, prompt string, startPhase int) error {
	specDir := filepath.Join(w.SpecsDir, specName)
	tasksPath := validation.GetTasksFilePath(specDir)

	// Get total phases for validation
	totalPhases, err := validation.GetTotalPhases(tasksPath)
	if err != nil {
		return fmt.Errorf("failed to get total phases: %w", err)
	}

	if startPhase < 1 || startPhase > totalPhases {
		return fmt.Errorf("phase %d is out of range (valid: 1-%d)", startPhase, totalPhases)
	}

	// Get phase info
	phases, err := validation.GetPhaseInfo(tasksPath)
	if err != nil {
		return fmt.Errorf("failed to get phase info: %w", err)
	}

	fmt.Printf("Starting from phase %d of %d\n\n", startPhase, totalPhases)

	// Execute each phase starting from startPhase
	for _, phase := range phases {
		if phase.Number < startPhase {
			continue // Skip phases before startPhase
		}

		// Get tasks for this phase to display task IDs
		phaseTasks, taskErr := validation.GetTasksForPhase(tasksPath, phase.Number)
		taskIDs := make([]string, 0, len(phaseTasks))
		if taskErr == nil {
			for _, t := range phaseTasks {
				taskIDs = append(taskIDs, t.ID)
			}
		}

		// Display pre-phase summary with task count, IDs, and status
		displayInfo := validation.BuildPhaseDisplayInfo(phase, totalPhases, taskIDs)
		fmt.Println(validation.FormatPhaseHeader(displayInfo))

		// Execute this phase
		err := w.executeSinglePhaseSession(specName, phase.Number, prompt)
		if err != nil {
			return fmt.Errorf("phase %d failed: %w", phase.Number, err)
		}

		// Re-read phase info to get updated task counts
		updatedPhases, rereadErr := validation.GetPhaseInfo(tasksPath)
		var updatedPhase *validation.PhaseInfo
		if rereadErr == nil {
			for _, p := range updatedPhases {
				if p.Number == phase.Number {
					updatedPhase = &p
					break
				}
			}
		}

		// Verify phase completion
		complete, verifyErr := validation.IsPhaseComplete(tasksPath, phase.Number)
		if verifyErr != nil {
			return fmt.Errorf("failed to verify phase %d completion: %w", phase.Number, verifyErr)
		}

		if !complete {
			fmt.Printf("\n⚠ Phase %d has incomplete tasks. Run 'autospec implement --phase %d' to continue.\n", phase.Number, phase.Number)
			return fmt.Errorf("phase %d did not complete all tasks", phase.Number)
		}

		// Display completion message with task counts
		if updatedPhase != nil {
			fmt.Println(validation.FormatPhaseCompletion(phase.Number, updatedPhase.CompletedTasks, updatedPhase.TotalTasks, updatedPhase.BlockedTasks))
		} else {
			fmt.Printf("✓ Phase %d complete\n", phase.Number)
		}
		fmt.Println()
	}

	// Show final summary
	fmt.Println("✓ All phases completed!")
	fmt.Println()
	stats, statsErr := validation.GetTaskStats(tasksPath)
	if statsErr == nil && stats.TotalTasks > 0 {
		fmt.Println("Task Summary:")
		fmt.Print(validation.FormatTaskSummary(stats))
	}

	return nil
}

// ExecuteImplementWithTasks runs each task in a separate Claude session
func (w *WorkflowOrchestrator) ExecuteImplementWithTasks(specName string, metadata *spec.Metadata, prompt string, fromTask string) error {
	specDir := filepath.Join(w.SpecsDir, specName)
	tasksPath := validation.GetTasksFilePath(specDir)

	// Get all tasks from tasks.yaml
	allTasks, err := validation.GetAllTasks(tasksPath)
	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	if len(allTasks) == 0 {
		return fmt.Errorf("no tasks found in tasks.yaml")
	}

	// Get tasks in dependency order
	orderedTasks, err := validation.GetTasksInDependencyOrder(allTasks)
	if err != nil {
		return fmt.Errorf("failed to order tasks by dependencies: %w", err)
	}

	totalTasks := len(orderedTasks)

	// Find starting index based on fromTask
	startIdx := 0
	if fromTask != "" {
		found := false
		for i, task := range orderedTasks {
			if task.ID == fromTask {
				startIdx = i
				found = true
				break
			}
		}
		if !found {
			// List available task IDs
			taskIDs := make([]string, len(orderedTasks))
			for i, t := range orderedTasks {
				taskIDs[i] = t.ID
			}
			return fmt.Errorf("task %s not found in tasks.yaml (available: %v)", fromTask, taskIDs)
		}

		// Validate that fromTask's dependencies are met
		fromTaskItem, _ := validation.GetTaskByID(allTasks, fromTask)
		met, unmetDeps := validation.ValidateTaskDependenciesMet(*fromTaskItem, allTasks)
		if !met {
			return fmt.Errorf("cannot start from task %s: dependencies not met (%v)", fromTask, unmetDeps)
		}
	}

	// Display skip message if starting from a later task
	if startIdx > 0 {
		fmt.Printf("Starting from task %s (task %d of %d)\n\n", fromTask, startIdx+1, totalTasks)
	}

	// Execute each task starting from startIdx
	completedCount := 0
	for i := startIdx; i < len(orderedTasks); i++ {
		task := orderedTasks[i]

		// Skip already completed tasks
		if task.Status == "Completed" || task.Status == "completed" {
			fmt.Printf("✓ Task %d/%d: %s - %s (already completed)\n", i+1, totalTasks, task.ID, task.Title)
			completedCount++
			continue
		}

		// Skip blocked tasks
		if task.Status == "Blocked" || task.Status == "blocked" {
			fmt.Printf("⚠ Task %d/%d: %s - %s (blocked)\n", i+1, totalTasks, task.ID, task.Title)
			continue
		}

		fmt.Printf("[Task %d/%d] %s - %s\n", i+1, totalTasks, task.ID, task.Title)

		// Validate dependencies before executing
		// Re-read tasks to get fresh status
		freshTasks, err := validation.GetAllTasks(tasksPath)
		if err != nil {
			return fmt.Errorf("failed to refresh tasks: %w", err)
		}

		met, unmetDeps := validation.ValidateTaskDependenciesMet(task, freshTasks)
		if !met {
			fmt.Printf("⚠ Skipping task %s: dependencies not met (%v)\n", task.ID, unmetDeps)
			continue
		}

		// Execute this task in a fresh Claude session
		err = w.executeSingleTaskSession(specName, task.ID, task.Title, prompt)
		if err != nil {
			return fmt.Errorf("task %s failed: %w", task.ID, err)
		}

		// Verify task completion by re-reading tasks.yaml
		freshTasks, err = validation.GetAllTasks(tasksPath)
		if err != nil {
			return fmt.Errorf("failed to verify task completion: %w", err)
		}

		freshTask, err := validation.GetTaskByID(freshTasks, task.ID)
		if err != nil {
			return fmt.Errorf("failed to find task %s after execution: %w", task.ID, err)
		}

		if freshTask.Status != "Completed" && freshTask.Status != "completed" {
			fmt.Printf("\n⚠ Task %s did not complete (status: %s). Run 'autospec implement --tasks --from-task %s' to retry.\n", task.ID, freshTask.Status, task.ID)
			return fmt.Errorf("task %s did not complete after execution (status: %s)", task.ID, freshTask.Status)
		}

		fmt.Printf("✓ Task %s complete\n\n", task.ID)
		completedCount++
	}

	// Show final summary
	fmt.Println("✓ All tasks processed!")
	fmt.Println()
	stats, statsErr := validation.GetTaskStats(tasksPath)
	if statsErr == nil && stats.TotalTasks > 0 {
		fmt.Println("Task Summary:")
		fmt.Print(validation.FormatTaskSummary(stats))
	}

	return nil
}

// executeSingleTaskSession executes a single task in a fresh Claude session
func (w *WorkflowOrchestrator) executeSingleTaskSession(specName, taskID, taskTitle, prompt string) error {
	// Build command with task filter
	command := fmt.Sprintf("/autospec.implement --task %s", taskID)
	if prompt != "" {
		command = fmt.Sprintf("/autospec.implement --task %s \"%s\"", taskID, prompt)
	}

	fmt.Printf("Executing: %s\n", command)

	result, err := w.Executor.ExecutePhase(
		specName,
		PhaseImplement,
		command,
		func(specDir string) error {
			// For task execution, we validate the specific task is completed
			tasksPath := validation.GetTasksFilePath(specDir)
			allTasks, err := validation.GetAllTasks(tasksPath)
			if err != nil {
				return err
			}

			task, err := validation.GetTaskByID(allTasks, taskID)
			if err != nil {
				return err
			}

			if task.Status != "Completed" && task.Status != "completed" {
				return fmt.Errorf("task %s not completed (status: %s)", taskID, task.Status)
			}
			return nil
		},
	)

	if err != nil {
		if result.Exhausted {
			fmt.Printf("\nTask %s paused.\n", taskID)
			fmt.Printf("To resume: autospec implement --tasks --from-task %s\n", taskID)
			return fmt.Errorf("task %s exhausted retries: %w", taskID, err)
		}
		return err
	}

	return nil
}

// executeSinglePhaseSession executes a single phase in a fresh Claude session
func (w *WorkflowOrchestrator) executeSinglePhaseSession(specName string, phaseNumber int, prompt string) error {
	specDir := filepath.Join(w.SpecsDir, specName)
	tasksPath := validation.GetTasksFilePath(specDir)

	// Get total phases for context
	totalPhases, err := validation.GetTotalPhases(tasksPath)
	if err != nil {
		return fmt.Errorf("failed to get total phases: %w", err)
	}

	// Check for edge cases before building context
	phaseTasks, err := validation.GetTasksForPhase(tasksPath, phaseNumber)
	if err != nil {
		return fmt.Errorf("failed to get tasks for phase %d: %w", phaseNumber, err)
	}

	// Edge case: Empty phase - display '0 tasks' and skip execution
	if len(phaseTasks) == 0 {
		fmt.Printf("  -> Phase %d has 0 tasks, skipping execution\n", phaseNumber)
		return nil
	}

	// Edge case: All tasks in phase already completed
	allCompleted := true
	completedCount := 0
	for _, task := range phaseTasks {
		statusLower := task.Status
		if statusLower == "Completed" || statusLower == "completed" || statusLower == "Done" || statusLower == "done" {
			completedCount++
		} else if statusLower != "Blocked" && statusLower != "blocked" {
			allCompleted = false
		}
	}

	// If all tasks are either completed or blocked, skip execution
	if allCompleted {
		fmt.Printf("  -> All %d tasks in phase %d already completed, skipping execution\n", completedCount, phaseNumber)
		return nil
	}

	// Build phase context with spec, plan, and phase-specific tasks
	phaseCtx, err := BuildPhaseContext(specDir, phaseNumber, totalPhases)
	if err != nil {
		return fmt.Errorf("failed to build phase context for phase %d: %w", phaseNumber, err)
	}

	// Write context file
	contextFilePath, err := WriteContextFile(phaseCtx)
	if err != nil {
		return fmt.Errorf("failed to write context file: %w", err)
	}

	// Ensure context file is cleaned up after execution
	defer CleanupContextFile(contextFilePath)

	// Check gitignore status (only warn, don't block)
	EnsureContextDirGitignored()

	// Build command with phase filter and context file
	command := fmt.Sprintf("/autospec.implement --phase %d --context-file %s", phaseNumber, contextFilePath)
	if prompt != "" {
		command = fmt.Sprintf("/autospec.implement --phase %d --context-file %s \"%s\"", phaseNumber, contextFilePath, prompt)
	}

	fmt.Printf("Executing: %s\n", command)

	result, err := w.Executor.ExecutePhase(
		specName,
		PhaseImplement,
		command,
		func(specDir string) error {
			// For phased execution, we validate the specific phase
			tasksPath := validation.GetTasksFilePath(specDir)
			complete, err := validation.IsPhaseComplete(tasksPath, phaseNumber)
			if err != nil {
				return err
			}
			if !complete {
				return fmt.Errorf("phase %d has incomplete tasks", phaseNumber)
			}
			return nil
		},
	)

	if err != nil {
		if result.Exhausted {
			fmt.Printf("\nPhase %d paused.\n", phaseNumber)
			fmt.Printf("To resume: autospec implement --phase %d\n", phaseNumber)
			return fmt.Errorf("phase %d exhausted retries: %w", phaseNumber, err)
		}
		return err
	}

	return nil
}

// ExecuteConstitution runs the constitution phase with optional prompt
// Constitution creates or updates the project constitution file
func (w *WorkflowOrchestrator) ExecuteConstitution(prompt string) error {
	// Build command with optional prompt
	command := "/autospec.constitution"
	if prompt != "" {
		command = fmt.Sprintf("/autospec.constitution \"%s\"", prompt)
	}

	if prompt != "" {
		fmt.Printf("Executing: /autospec.constitution \"%s\"\n", prompt)
	} else {
		fmt.Println("Executing: /autospec.constitution")
	}

	// Constitution phase doesn't require spec detection - it works at project level
	result, err := w.Executor.ExecutePhase(
		"", // No spec name needed for constitution
		PhaseConstitution,
		command,
		func(specDir string) error {
			// Constitution doesn't produce tracked artifacts
			// It modifies .autospec/memory/constitution.yaml
			return nil
		},
	)

	if err != nil {
		if result.Exhausted {
			return fmt.Errorf("constitution phase exhausted retries: %w", err)
		}
		return fmt.Errorf("constitution failed: %w", err)
	}

	fmt.Println("\n✓ Constitution updated!")
	return nil
}

// ExecuteClarify runs the clarify phase with optional prompt
// Clarify refines the specification by asking targeted clarification questions
func (w *WorkflowOrchestrator) ExecuteClarify(specNameArg string, prompt string) error {
	var specName string
	var err error

	if specNameArg != "" {
		specName = specNameArg
	} else {
		// Auto-detect current spec
		metadata, err := spec.DetectCurrentSpec(w.SpecsDir)
		if err != nil {
			return fmt.Errorf("failed to detect current spec: %w", err)
		}
		specName = fmt.Sprintf("%s-%s", metadata.Number, metadata.Name)
		fmt.Printf("Detected spec: %s\n", specName)
	}

	// Build command with optional prompt
	command := "/autospec.clarify"
	if prompt != "" {
		command = fmt.Sprintf("/autospec.clarify \"%s\"", prompt)
	}

	if prompt != "" {
		fmt.Printf("Executing: /autospec.clarify \"%s\"\n", prompt)
	} else {
		fmt.Println("Executing: /autospec.clarify")
	}

	result, err := w.Executor.ExecutePhase(
		specName,
		PhaseClarify,
		command,
		func(specDir string) error {
			// Clarify updates spec.yaml in place - just verify it still exists
			return validation.ValidateSpecFile(specDir)
		},
	)

	if err != nil {
		if result.Exhausted {
			return fmt.Errorf("clarify phase exhausted retries: %w", err)
		}
		return fmt.Errorf("clarify failed: %w", err)
	}

	fmt.Printf("\n✓ Clarification complete for specs/%s/\n", specName)
	return nil
}

// ExecuteChecklist runs the checklist phase with optional prompt
// Checklist generates a custom checklist for the current feature
func (w *WorkflowOrchestrator) ExecuteChecklist(specNameArg string, prompt string) error {
	var specName string
	var err error

	if specNameArg != "" {
		specName = specNameArg
	} else {
		// Auto-detect current spec
		metadata, err := spec.DetectCurrentSpec(w.SpecsDir)
		if err != nil {
			return fmt.Errorf("failed to detect current spec: %w", err)
		}
		specName = fmt.Sprintf("%s-%s", metadata.Number, metadata.Name)
		fmt.Printf("Detected spec: %s\n", specName)
	}

	// Build command with optional prompt
	command := "/autospec.checklist"
	if prompt != "" {
		command = fmt.Sprintf("/autospec.checklist \"%s\"", prompt)
	}

	if prompt != "" {
		fmt.Printf("Executing: /autospec.checklist \"%s\"\n", prompt)
	} else {
		fmt.Println("Executing: /autospec.checklist")
	}

	result, err := w.Executor.ExecutePhase(
		specName,
		PhaseChecklist,
		command,
		func(specDir string) error {
			// Checklist creates files in checklists/ directory
			// For now, just verify the command completed successfully
			return nil
		},
	)

	if err != nil {
		if result.Exhausted {
			return fmt.Errorf("checklist phase exhausted retries: %w", err)
		}
		return fmt.Errorf("checklist failed: %w", err)
	}

	fmt.Printf("\n✓ Checklist generated for specs/%s/\n", specName)
	return nil
}

// ExecuteAnalyze runs the analyze phase with optional prompt
// Analyze performs cross-artifact consistency and quality analysis
func (w *WorkflowOrchestrator) ExecuteAnalyze(specNameArg string, prompt string) error {
	var specName string
	var err error

	if specNameArg != "" {
		specName = specNameArg
	} else {
		// Auto-detect current spec
		metadata, err := spec.DetectCurrentSpec(w.SpecsDir)
		if err != nil {
			return fmt.Errorf("failed to detect current spec: %w", err)
		}
		specName = fmt.Sprintf("%s-%s", metadata.Number, metadata.Name)
		fmt.Printf("Detected spec: %s\n", specName)
	}

	// Build command with optional prompt
	command := "/autospec.analyze"
	if prompt != "" {
		command = fmt.Sprintf("/autospec.analyze \"%s\"", prompt)
	}

	if prompt != "" {
		fmt.Printf("Executing: /autospec.analyze \"%s\"\n", prompt)
	} else {
		fmt.Println("Executing: /autospec.analyze")
	}

	result, err := w.Executor.ExecutePhase(
		specName,
		PhaseAnalyze,
		command,
		func(specDir string) error {
			// Analyze outputs analysis report
			// For now, just verify the command completed successfully
			return nil
		},
	)

	if err != nil {
		if result.Exhausted {
			return fmt.Errorf("analyze phase exhausted retries: %w", err)
		}
		return fmt.Errorf("analyze failed: %w", err)
	}

	fmt.Printf("\n✓ Analysis complete for specs/%s/\n", specName)
	return nil
}
