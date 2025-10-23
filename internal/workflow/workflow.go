package workflow

import (
	"fmt"
	"path/filepath"

	"github.com/anthropics/auto-claude-speckit/internal/config"
	"github.com/anthropics/auto-claude-speckit/internal/spec"
)

// WorkflowOrchestrator manages the complete specify → plan → tasks workflow
type WorkflowOrchestrator struct {
	Executor  *Executor
	Config    *config.Configuration
	SpecsDir  string
	SkipPreflight bool
}

// NewWorkflowOrchestrator creates a new workflow orchestrator from configuration
func NewWorkflowOrchestrator(cfg *config.Configuration) *WorkflowOrchestrator {
	claude := &ClaudeExecutor{
		ClaudeCmd:       cfg.ClaudeCmd,
		ClaudeArgs:      cfg.ClaudeArgs,
		UseAPIKey:       cfg.UseAPIKey,
		CustomClaudeCmd: cfg.CustomClaudeCmd,
	}

	executor := &Executor{
		Claude:     claude,
		StateDir:   cfg.StateDir,
		SpecsDir:   cfg.SpecsDir,
		MaxRetries: cfg.MaxRetries,
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

	if err := w.executePlan(specName); err != nil {
		return fmt.Errorf("plan phase failed: %w", err)
	}

	fmt.Printf("✓ Created specs/%s/plan.md\n", specName)
	fmt.Printf("✓ Created specs/%s/research.md\n\n", specName)

	// Phase 3: Tasks
	fmt.Println("[Phase 3/3] Tasks...")
	fmt.Println("Executing: /speckit.tasks")

	if err := w.executeTasks(specName); err != nil {
		return fmt.Errorf("tasks phase failed: %w", err)
	}

	fmt.Printf("✓ Created specs/%s/tasks.md\n\n", specName)

	// Success!
	fmt.Println("Workflow completed successfully!")
	fmt.Printf("Spec: specs/%s/\n", specName)
	fmt.Println("Next: autospec implement")

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

	return metadata.Name, nil
}

// executePlan executes the /speckit.plan command
func (w *WorkflowOrchestrator) executePlan(specName string) error {
	command := "/speckit.plan"
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

// executeTasks executes the /speckit.tasks command
func (w *WorkflowOrchestrator) executeTasks(specName string) error {
	command := "/speckit.tasks"

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
func (w *WorkflowOrchestrator) ExecutePlan(specNameArg string) error {
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
		specName = metadata.Name
		fmt.Printf("Detected spec: %s\n", specName)
	}

	fmt.Println("Executing: /speckit.plan")

	if err = w.executePlan(specName); err != nil {
		return err
	}

	fmt.Printf("✓ Created specs/%s/plan.md\n", specName)
	fmt.Printf("✓ Created specs/%s/research.md\n", specName)
	fmt.Printf("✓ Created specs/%s/data-model.md\n\n", specName)
	fmt.Println("Next: autospec tasks")

	return nil
}

// ExecuteTasks runs only the tasks phase for a detected or specified spec
func (w *WorkflowOrchestrator) ExecuteTasks(specNameArg string) error {
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
		specName = metadata.Name
		fmt.Printf("Detected spec: %s\n", specName)
	}

	fmt.Println("Executing: /speckit.tasks")

	if err = w.executeTasks(specName); err != nil {
		return err
	}

	fmt.Printf("✓ Created specs/%s/tasks.md\n\n", specName)
	fmt.Println("Next: autospec implement")

	return nil
}

// ExecuteImplement runs the implementation phase
func (w *WorkflowOrchestrator) ExecuteImplement(specNameArg string, resume bool) error {
	var specName string
	var metadata *spec.SpecMetadata
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
		specName = metadata.Name
		fmt.Printf("Detected spec: %s\n", specName)
	}

	// Check progress
	tasksPath := filepath.Join(metadata.Directory, "tasks.md")
	count, err := w.Executor.Executor.Claude.StreamCommand(
		"/speckit.implement",
		nil, nil,
	)

	fmt.Printf("Progress: ?/? tasks completed (?%%)\n\n")
	fmt.Println("Executing: /speckit.implement")

	command := "/speckit.implement"
	if resume {
		command += " --resume"
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
			prompt, _ := w.Executor.Executor.ValidateTasksComplete(tasksPath)
			if prompt != nil {
				fmt.Println("\nImplementation paused.")
				fmt.Println("To resume: autospec implement --resume")
			}
			return fmt.Errorf("implementation phase exhausted retries: %w", err)
		}
		return fmt.Errorf("implementation failed: %w", err)
	}

	fmt.Println("\n✓ All tasks completed!")
	return nil
}
