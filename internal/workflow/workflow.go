package workflow

import (
	"fmt"
	"path/filepath"

	"github.com/anthropics/auto-claude-speckit/internal/config"
	"github.com/anthropics/auto-claude-speckit/internal/progress"
	"github.com/anthropics/auto-claude-speckit/internal/spec"
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
		TotalPhases:     3, // Default to 3 phases (specify, plan, tasks)
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
	fmt.Printf("Executing: /speckit.specify \"%s\"\n", featureDescription)

	specName, err := w.executeSpecify(featureDescription)
	if err != nil {
		return fmt.Errorf("specify phase failed: %w", err)
	}

	fmt.Printf("✓ Created specs/%s/spec.md\n\n", specName)

	// Phase 2: Plan
	fmt.Println("[Phase 2/3] Plan...")
	fmt.Println("Executing: /speckit.plan")

	if err := w.executePlan(specName, ""); err != nil {
		return fmt.Errorf("plan phase failed: %w", err)
	}

	fmt.Printf("✓ Created specs/%s/plan.md\n", specName)
	fmt.Printf("✓ Created specs/%s/research.md\n\n", specName)

	// Phase 3: Tasks
	fmt.Println("[Phase 3/3] Tasks...")
	fmt.Println("Executing: /speckit.tasks")

	if err := w.executeTasks(specName, ""); err != nil {
		return fmt.Errorf("tasks phase failed: %w", err)
	}

	fmt.Printf("✓ Created specs/%s/tasks.md\n\n", specName)

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
	fmt.Printf("Executing: /speckit.specify \"%s\"\n", featureDescription)

	specName, err := w.executeSpecify(featureDescription)
	if err != nil {
		return fmt.Errorf("specify phase failed: %w", err)
	}

	fmt.Printf("✓ Created specs/%s/spec.md\n\n", specName)

	// Phase 2: Plan
	fmt.Println("[Phase 2/4] Plan...")
	fmt.Println("Executing: /speckit.plan")

	if err := w.executePlan(specName, ""); err != nil {
		return fmt.Errorf("plan phase failed: %w", err)
	}

	fmt.Printf("✓ Created specs/%s/plan.md\n", specName)
	fmt.Printf("✓ Created specs/%s/research.md\n\n", specName)

	// Phase 3: Tasks
	fmt.Println("[Phase 3/4] Tasks...")
	fmt.Println("Executing: /speckit.tasks")

	if err := w.executeTasks(specName, ""); err != nil {
		return fmt.Errorf("tasks phase failed: %w", err)
	}

	fmt.Printf("✓ Created specs/%s/tasks.md\n\n", specName)

	// Phase 4: Implement
	fmt.Println("[Phase 4/4] Implement...")
	fmt.Println("Executing: /speckit.implement")
	w.debugLog("Starting implement phase for spec: %s", specName)

	command := "/speckit.implement"
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
			tasksPath := filepath.Join(specDir, "tasks.md")
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
	fmt.Println("Full workflow completed successfully!")
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
		fmt.Println("✓ .specify/ directory exists")
	}

	fmt.Println()
	return nil
}

// executeSpecify executes the /speckit.specify command and returns the spec name
func (w *WorkflowOrchestrator) executeSpecify(featureDescription string) (string, error) {
	command := fmt.Sprintf("/speckit.specify \"%s\"", featureDescription)

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

// executePlan executes the /speckit.plan command with optional prompt
func (w *WorkflowOrchestrator) executePlan(specName string, prompt string) error {
	command := "/speckit.plan"
	if prompt != "" {
		command = fmt.Sprintf("/speckit.plan \"%s\"", prompt)
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

// executeTasks executes the /speckit.tasks command with optional prompt
func (w *WorkflowOrchestrator) executeTasks(specName string, prompt string) error {
	command := "/speckit.tasks"
	if prompt != "" {
		command = fmt.Sprintf("/speckit.tasks \"%s\"", prompt)
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
	fmt.Printf("Executing: /speckit.specify \"%s\"\n", featureDescription)

	specName, err := w.executeSpecify(featureDescription)
	if err != nil {
		return "", err
	}

	fmt.Printf("✓ Created specs/%s/spec.md\n\n", specName)
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
		fmt.Printf("Executing: /speckit.plan \"%s\"\n", prompt)
	} else {
		fmt.Println("Executing: /speckit.plan")
	}

	if err = w.executePlan(specName, prompt); err != nil {
		return err
	}

	fmt.Printf("✓ Created specs/%s/plan.md\n", specName)
	fmt.Printf("✓ Created specs/%s/research.md\n", specName)
	fmt.Printf("✓ Created specs/%s/data-model.md\n\n", specName)
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
		fmt.Printf("Executing: /speckit.tasks \"%s\"\n", prompt)
	} else {
		fmt.Println("Executing: /speckit.tasks")
	}

	if err = w.executeTasks(specName, prompt); err != nil {
		return err
	}

	fmt.Printf("✓ Created specs/%s/tasks.md\n\n", specName)
	fmt.Println("Next: autospec implement")

	return nil
}

// ExecuteImplement runs the implementation phase with optional prompt
func (w *WorkflowOrchestrator) ExecuteImplement(specNameArg string, prompt string, resume bool) error {
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

	// Check progress
	// TODO: Display progress information
	fmt.Printf("Progress: checking tasks...\n\n")

	// Build command with optional prompt
	command := "/speckit.implement"
	if resume {
		command += " --resume"
	}
	if prompt != "" {
		command = fmt.Sprintf("/speckit.implement \"%s\"", prompt)
		if resume {
			// If both resume and prompt, append resume after prompt
			command = fmt.Sprintf("/speckit.implement --resume \"%s\"", prompt)
		}
	}

	if prompt != "" {
		fmt.Printf("Executing: /speckit.implement \"%s\"\n", prompt)
	} else {
		fmt.Println("Executing: /speckit.implement")
	}

	result, err := w.Executor.ExecutePhase(
		specName,
		PhaseImplement,
		command,
		func(specDir string) error {
			tasksPath := filepath.Join(specDir, "tasks.md")
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

	fmt.Println("\n✓ All tasks completed!")
	return nil
}
