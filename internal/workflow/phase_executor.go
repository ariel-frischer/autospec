// Package workflow provides phase execution functionality.
// PhaseExecutor handles phase-based implementation execution.
// Related: internal/workflow/orchestrator.go, internal/workflow/interfaces.go (interface definition)
// Tags: workflow, phase-executor, implementation, phases
package workflow

import (
	"fmt"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/commands"
	"github.com/ariel-frischer/autospec/internal/prereqs"
	"github.com/ariel-frischer/autospec/internal/validation"
)

// PhaseExecutor handles phase-based implementation execution.
// It implements PhaseExecutorInterface to enable dependency injection and testing.
// Each phase represents a logical grouping of tasks that are executed together
// in a single Claude session with phase-specific context.
type PhaseExecutor struct {
	executor *Executor // Underlying executor for Claude command execution
	specsDir string    // Base directory for spec storage (e.g., "specs/")
	debug    bool      // Enable debug logging
}

// NewPhaseExecutor creates a new PhaseExecutor with the given dependencies.
// executor: required, handles actual command execution with retry logic
// specsDir: required, base directory where spec directories are located
// debug: optional, enables verbose logging for troubleshooting
func NewPhaseExecutor(executor *Executor, specsDir string, debug bool) *PhaseExecutor {
	return &PhaseExecutor{
		executor: executor,
		specsDir: specsDir,
		debug:    debug,
	}
}

// debugLog prints a debug message if debug mode is enabled.
func (p *PhaseExecutor) debugLog(format string, args ...interface{}) {
	if p.debug {
		fmt.Printf("[DEBUG][PhaseExecutor] "+format+"\n", args...)
	}
}

// ExecutePhaseLoop iterates through phases from startPhase to totalPhases.
// Each phase runs in a separate Claude session with phase-specific context.
// specName: the spec directory name (e.g., "003-command-timeout")
// tasksPath: path to tasks.yaml file
// phases: slice of PhaseInfo containing phase metadata
// startPhase: 1-based phase number to start from
// totalPhases: total number of phases
// prompt: optional custom prompt to pass to each phase
func (p *PhaseExecutor) ExecutePhaseLoop(specName, tasksPath string, phases []validation.PhaseInfo, startPhase, totalPhases int, prompt string) error {
	p.debugLog("ExecutePhaseLoop called: spec=%s, startPhase=%d, totalPhases=%d", specName, startPhase, totalPhases)
	specDir := filepath.Join(p.specsDir, specName)

	for _, phase := range phases {
		if phase.Number < startPhase {
			continue
		}

		if err := p.executeAndVerifyPhase(specName, tasksPath, phase, totalPhases, prompt); err != nil {
			return fmt.Errorf("executing phase %d: %w", phase.Number, err)
		}
	}

	p.printPhasesSummary(tasksPath, specDir)
	return nil
}

// ExecuteSinglePhase runs a specific phase in isolation.
// specName: the spec directory name
// phaseNumber: 1-based phase number to execute
// prompt: optional custom prompt
func (p *PhaseExecutor) ExecuteSinglePhase(specName string, phaseNumber int, prompt string) error {
	p.debugLog("ExecuteSinglePhase called: spec=%s, phaseNumber=%d", specName, phaseNumber)
	return p.executeSinglePhaseSession(specName, phaseNumber, prompt)
}

// executeAndVerifyPhase executes a single phase and verifies completion.
func (p *PhaseExecutor) executeAndVerifyPhase(specName, tasksPath string, phase validation.PhaseInfo, totalPhases int, prompt string) error {
	taskIDs := p.getTaskIDsForPhase(tasksPath, phase.Number)
	displayInfo := validation.BuildPhaseDisplayInfo(phase, totalPhases, taskIDs)
	fmt.Println(validation.FormatPhaseHeader(displayInfo))

	if err := p.executeSinglePhaseSession(specName, phase.Number, prompt); err != nil {
		return fmt.Errorf("phase %d failed: %w", phase.Number, err)
	}

	updatedPhase := p.getUpdatedPhaseInfo(tasksPath, phase.Number)

	complete, verifyErr := validation.IsPhaseComplete(tasksPath, phase.Number)
	if verifyErr != nil {
		return fmt.Errorf("failed to verify phase %d completion: %w", phase.Number, verifyErr)
	}

	if !complete {
		fmt.Printf("\n⚠ Phase %d has incomplete tasks. Run 'autospec implement --phase %d' to continue.\n", phase.Number, phase.Number)
		return fmt.Errorf("phase %d did not complete all tasks", phase.Number)
	}

	p.printPhaseCompletion(phase.Number, updatedPhase)
	fmt.Println()
	return nil
}

// executeSinglePhaseSession executes a single phase in a fresh Claude session.
func (p *PhaseExecutor) executeSinglePhaseSession(specName string, phaseNumber int, prompt string) error {
	specDir := filepath.Join(p.specsDir, specName)
	tasksPath := validation.GetTasksFilePath(specDir)

	// Get total phases for context
	totalPhases, err := validation.GetTotalPhases(tasksPath)
	if err != nil {
		return fmt.Errorf("failed to get total phases: %w", err)
	}

	// Check for edge cases before building context
	if shouldSkip, skipErr := p.checkPhaseSkipConditions(tasksPath, phaseNumber); skipErr != nil {
		return skipErr
	} else if shouldSkip {
		return nil
	}

	// Build and write context file
	contextFilePath, err := p.buildAndWritePhaseContext(specDir, phaseNumber, totalPhases)
	if err != nil {
		return err
	}
	defer CleanupContextFile(contextFilePath)

	// Check gitignore status (only warn, don't block)
	EnsureContextDirGitignored()

	// Build and execute command
	command, err := p.buildPhaseCommand(specName, phaseNumber, contextFilePath, prompt)
	if err != nil {
		return fmt.Errorf("building phase command: %w", err)
	}
	fmt.Printf("Executing: autospec.implement --phase %d\n", phaseNumber)

	return p.executePhaseWithValidation(specName, phaseNumber, command)
}

// checkPhaseSkipConditions checks if a phase should be skipped.
// Returns (shouldSkip, error).
func (p *PhaseExecutor) checkPhaseSkipConditions(tasksPath string, phaseNumber int) (bool, error) {
	phaseTasks, err := validation.GetTasksForPhase(tasksPath, phaseNumber)
	if err != nil {
		return false, fmt.Errorf("failed to get tasks for phase %d: %w", phaseNumber, err)
	}

	// Edge case: Empty phase
	if len(phaseTasks) == 0 {
		fmt.Printf("  -> Phase %d has 0 tasks, skipping execution\n", phaseNumber)
		return true, nil
	}

	// Edge case: All tasks already completed or blocked
	if p.allTasksCompletedOrBlocked(phaseTasks, phaseNumber) {
		return true, nil
	}

	return false, nil
}

// allTasksCompletedOrBlocked checks if all tasks in a phase are completed or blocked.
func (p *PhaseExecutor) allTasksCompletedOrBlocked(phaseTasks []validation.TaskItem, phaseNumber int) bool {
	completedCount := 0
	for _, task := range phaseTasks {
		status := task.Status
		if status == "Completed" || status == "completed" || status == "Done" || status == "done" {
			completedCount++
		} else if status != "Blocked" && status != "blocked" {
			return false // Found a task that's neither completed nor blocked
		}
	}

	if completedCount == len(phaseTasks) || completedCount > 0 {
		fmt.Printf("  -> All %d tasks in phase %d already completed, skipping execution\n", completedCount, phaseNumber)
		return true
	}

	return false
}

// buildAndWritePhaseContext builds and writes the phase context file.
func (p *PhaseExecutor) buildAndWritePhaseContext(specDir string, phaseNumber, totalPhases int) (string, error) {
	phaseCtx, err := BuildPhaseContext(specDir, phaseNumber, totalPhases)
	if err != nil {
		return "", fmt.Errorf("failed to build phase context for phase %d: %w", phaseNumber, err)
	}

	contextFilePath, err := WriteContextFile(phaseCtx)
	if err != nil {
		return "", fmt.Errorf("failed to write context file: %w", err)
	}

	return contextFilePath, nil
}

// buildPhaseCommand renders the implement template and appends phase flags.
func (p *PhaseExecutor) buildPhaseCommand(specName string, phaseNumber int, contextFilePath, prompt string) (string, error) {
	rendered, err := p.computeAndRenderImplementCommand(specName)
	if err != nil {
		return "", err
	}
	flags := fmt.Sprintf("--phase %d --context-file %s", phaseNumber, contextFilePath)
	if prompt != "" {
		return fmt.Sprintf("%s\n\n## Flags\n\n%s\n\n## User Input\n\n%s", rendered, flags, prompt), nil
	}
	return fmt.Sprintf("%s\n\n## Flags\n\n%s", rendered, flags), nil
}

// computeAndRenderImplementCommand gets and renders the implement template.
func (p *PhaseExecutor) computeAndRenderImplementCommand(specName string) (string, error) {
	content, err := commands.GetTemplate("autospec.implement")
	if err != nil {
		return "", fmt.Errorf("loading template autospec.implement: %w", err)
	}

	opts := prereqs.Options{
		SpecsDir:          p.specsDir,
		FeatureIdentifier: specName,
		RequireTasks:      true,
	}
	if p.executor != nil {
		opts.StateDir = p.executor.StateDir
	}
	ctx, err := prereqs.ComputeContext(opts)
	if err != nil {
		return "", fmt.Errorf("computing prereqs context: %w", err)
	}

	rendered, err := commands.RenderAndValidate("autospec.implement", content, ctx)
	if err != nil {
		return "", fmt.Errorf("rendering template: %w", err)
	}

	return string(rendered), nil
}

// executePhaseWithValidation executes the phase command with validation.
func (p *PhaseExecutor) executePhaseWithValidation(specName string, phaseNumber int, command string) error {
	result, err := p.executor.ExecuteStage(
		specName,
		StageImplement,
		command,
		func(specDir string) error {
			tasksPath := validation.GetTasksFilePath(specDir)
			complete, err := validation.IsPhaseComplete(tasksPath, phaseNumber)
			if err != nil {
				return fmt.Errorf("checking phase %d completion: %w", phaseNumber, err)
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
		return fmt.Errorf("executing phase %d session: %w", phaseNumber, err)
	}

	return nil
}

// getTaskIDsForPhase returns task IDs for a given phase.
func (p *PhaseExecutor) getTaskIDsForPhase(tasksPath string, phaseNumber int) []string {
	phaseTasks, taskErr := validation.GetTasksForPhase(tasksPath, phaseNumber)
	taskIDs := make([]string, 0, len(phaseTasks))
	if taskErr == nil {
		for _, t := range phaseTasks {
			taskIDs = append(taskIDs, t.ID)
		}
	}
	return taskIDs
}

// getUpdatedPhaseInfo re-reads phase info to get updated task counts.
func (p *PhaseExecutor) getUpdatedPhaseInfo(tasksPath string, phaseNumber int) *validation.PhaseInfo {
	updatedPhases, rereadErr := validation.GetPhaseInfo(tasksPath)
	if rereadErr == nil {
		for _, ph := range updatedPhases {
			if ph.Number == phaseNumber {
				return &ph
			}
		}
	}
	return nil
}

// printPhaseCompletion prints the phase completion message.
func (p *PhaseExecutor) printPhaseCompletion(phaseNumber int, updatedPhase *validation.PhaseInfo) {
	if updatedPhase != nil {
		fmt.Println(validation.FormatPhaseCompletion(phaseNumber, updatedPhase.CompletedTasks, updatedPhase.TotalTasks, updatedPhase.BlockedTasks))
	} else {
		fmt.Printf("✓ Phase %d complete\n", phaseNumber)
	}
}

// printPhasesSummary prints the final phase execution summary and marks spec as completed.
func (p *PhaseExecutor) printPhasesSummary(tasksPath, specDir string) {
	fmt.Println("✓ All phases completed!")
	fmt.Println()
	stats, statsErr := validation.GetTaskStats(tasksPath)
	if statsErr == nil && stats.TotalTasks > 0 {
		fmt.Println("Task Summary:")
		fmt.Print(validation.FormatTaskSummary(stats))
	}

	// Mark spec as completed
	markSpecCompletedAndPrint(specDir)
}

// ExecuteDefault runs all implementation in a single Claude session.
// This is the default behavior when no --phases, --tasks, or --phase flags are specified.
func (p *PhaseExecutor) ExecuteDefault(specName, specDir, prompt string, resume bool) error {
	p.debugLog("ExecuteDefault called: spec=%s, resume=%v", specName, resume)

	// Check progress
	fmt.Printf("Progress: checking tasks...\n\n")

	// Build command with optional prompt and resume flag
	command, err := p.buildDefaultCommand(specName, prompt, resume)
	if err != nil {
		return fmt.Errorf("building implement command: %w", err)
	}
	p.printExecuting("/autospec.implement", prompt)

	result, err := p.executor.ExecuteStage(
		specName,
		StageImplement,
		command,
		func(sd string) error {
			tasksPath := validation.GetTasksFilePath(sd)
			return p.executor.ValidateTasksComplete(tasksPath)
		},
	)
	if err != nil {
		if result.Exhausted {
			fmt.Println("\nImplementation paused.")
			fmt.Println("To resume: autospec implement --resume")
			return fmt.Errorf("implementation stage exhausted retries: %w", err)
		}
		return fmt.Errorf("implementation failed: %w", err)
	}

	// Show task completion stats
	fmt.Println("\n✓ All tasks completed!")
	fmt.Println()
	tasksPath := validation.GetTasksFilePath(specDir)
	stats, statsErr := validation.GetTaskStats(tasksPath)
	if statsErr == nil && stats.TotalTasks > 0 {
		fmt.Println("Task Summary:")
		fmt.Print(validation.FormatTaskSummary(stats))
	}

	return nil
}

// buildDefaultCommand renders the implement template for default mode.
func (p *PhaseExecutor) buildDefaultCommand(specName, prompt string, resume bool) (string, error) {
	rendered, err := p.computeAndRenderImplementCommand(specName)
	if err != nil {
		return "", err
	}

	var flags string
	if resume {
		flags = "--resume"
	}

	if flags != "" && prompt != "" {
		return fmt.Sprintf("%s\n\n## Flags\n\n%s\n\n## User Input\n\n%s", rendered, flags, prompt), nil
	}
	if flags != "" {
		return fmt.Sprintf("%s\n\n## Flags\n\n%s", rendered, flags), nil
	}
	if prompt != "" {
		return fmt.Sprintf("%s\n\n## User Input\n\n%s", rendered, prompt), nil
	}
	return rendered, nil
}

// printExecuting prints the executing message for a command.
func (p *PhaseExecutor) printExecuting(baseCmd, prompt string) {
	if prompt != "" {
		fmt.Printf("Executing: %s \"%s\"\n", baseCmd, prompt)
	} else {
		fmt.Printf("Executing: %s\n", baseCmd)
	}
}

// Compile-time interface compliance check.
var _ PhaseExecutorInterface = (*PhaseExecutor)(nil)
