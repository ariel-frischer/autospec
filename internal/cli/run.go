package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/ariel-frischer/autospec/internal/validation"
	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [feature-description]",
	Short: "Run selected workflow phases with flexible phase selection",
	Long: `Run selected workflow phases with flexible phase selection.

Core phase flags:
  -s, --specify    Include specify phase (requires feature description)
  -p, --plan       Include plan phase
  -t, --tasks      Include tasks phase
  -i, --implement  Include implement phase
  -a, --all        Run all core phases (equivalent to -spti)

Optional phase flags:
  -n, --constitution  Include constitution phase
  -r, --clarify       Include clarify phase
  -l, --checklist     Include checklist phase (note: -c is used for --config)
  -z, --analyze       Include analyze phase

Phases are always executed in canonical order:
  constitution -> specify -> clarify -> plan -> tasks -> checklist -> analyze -> implement`,
	Example: `  # Run all core phases for a new feature
  autospec run -a "Add user authentication"

  # Run only plan and implement on current spec
  autospec run -pi

  # Run tasks and implement on a specific spec
  autospec run -ti --spec 007-yaml-output

  # Preview what phases would run (dry run mode)
  autospec run -ti --dry-run

  # Skip confirmation prompts for CI/CD
  autospec run -ti -y`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get core phase flags
		specify, _ := cmd.Flags().GetBool("specify")
		plan, _ := cmd.Flags().GetBool("plan")
		tasks, _ := cmd.Flags().GetBool("tasks")
		implement, _ := cmd.Flags().GetBool("implement")
		all, _ := cmd.Flags().GetBool("all")

		// Get optional phase flags
		constitution, _ := cmd.Flags().GetBool("constitution")
		clarify, _ := cmd.Flags().GetBool("clarify")
		checklist, _ := cmd.Flags().GetBool("checklist")
		analyze, _ := cmd.Flags().GetBool("analyze")

		// Get other flags
		specName, _ := cmd.Flags().GetString("spec")
		skipConfirm, _ := cmd.Flags().GetBool("yes")
		configPath, _ := cmd.Flags().GetString("config")
		skipPreflight, _ := cmd.Flags().GetBool("skip-preflight")
		maxRetries, _ := cmd.Flags().GetInt("max-retries")
		resume, _ := cmd.Flags().GetBool("resume")
		debug, _ := cmd.Flags().GetBool("debug")
		progress, _ := cmd.Flags().GetBool("progress")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Build PhaseConfig from flags
		phaseConfig := workflow.NewPhaseConfig()
		if all {
			phaseConfig.SetAll() // SetAll only sets core phases (specify, plan, tasks, implement)
		} else {
			// Core phases
			phaseConfig.Specify = specify
			phaseConfig.Plan = plan
			phaseConfig.Tasks = tasks
			phaseConfig.Implement = implement
		}
		// Optional phases are always set from flags (can be combined with -a)
		phaseConfig.Constitution = constitution
		phaseConfig.Clarify = clarify
		phaseConfig.Checklist = checklist
		phaseConfig.Analyze = analyze

		// Validate at least one phase is selected
		if !phaseConfig.HasAnyPhase() {
			return fmt.Errorf("no phases selected. Use -s/-p/-t/-i flags or -a for all phases\n\nRun 'autospec run --help' for usage")
		}

		// Get feature description from args if specify phase is selected
		var featureDescription string
		if phaseConfig.Specify {
			if len(args) < 1 {
				return fmt.Errorf("feature description required when using specify phase (-s)\n\nUsage: autospec run -s \"feature description\"")
			}
			featureDescription = args[0]
		} else if len(args) > 0 {
			// If not specifying but args provided, treat as prompt
			featureDescription = args[0]
		}

		// Load configuration
		cfg, err := config.Load(configPath)
		if err != nil {
			cliErr := clierrors.ConfigParseError(configPath, err)
			clierrors.PrintError(cliErr)
			return cliErr
		}

		// Override settings from flags
		if cmd.Flags().Changed("skip-preflight") {
			cfg.SkipPreflight = skipPreflight
		}
		if cmd.Flags().Changed("max-retries") {
			cfg.MaxRetries = maxRetries
		}
		if cmd.Flags().Changed("progress") {
			cfg.ShowProgress = progress
		}

		// Resolve skip confirmations (flag > env > config)
		if skipConfirm || os.Getenv("AUTOSPEC_YES") != "" || cfg.SkipConfirmations {
			cfg.SkipConfirmations = true
		}

		// Check if constitution exists (required unless only running constitution phase)
		if !phaseConfig.Constitution || phaseConfig.Count() > 1 {
			// Either not running constitution at all, or running other phases too
			constitutionCheck := workflow.CheckConstitutionExists()
			if !constitutionCheck.Exists {
				fmt.Fprint(os.Stderr, constitutionCheck.ErrorMessage)
				return fmt.Errorf("constitution required")
			}
		}

		// Detect or validate spec name
		var specMetadata *spec.Metadata
		if !phaseConfig.Specify {
			// Need to detect or validate spec if not starting with specify
			if specName != "" {
				// Validate explicit spec exists
				specDir := filepath.Join(cfg.SpecsDir, specName)
				if _, err := os.Stat(specDir); os.IsNotExist(err) {
					return fmt.Errorf("spec not found: %s\n\nRun 'autospec specify' to create a new spec or check the spec name", specName)
				}
				specMetadata = &spec.Metadata{
					Name:      specName,
					Directory: specDir,
				}
			} else {
				// Auto-detect from git branch
				specMetadata, err = spec.DetectCurrentSpec(cfg.SpecsDir)
				if err != nil {
					return fmt.Errorf("failed to detect spec: %w\n\nUse --spec flag to specify explicitly or checkout a spec branch", err)
				}
				fmt.Printf("Detected spec: %s-%s\n", specMetadata.Number, specMetadata.Name)
			}
		}

		// Check artifact dependencies before execution
		if !phaseConfig.Specify {
			preflightResult := workflow.CheckArtifactDependencies(phaseConfig, specMetadata.Directory)
			if preflightResult.RequiresConfirmation {
				fmt.Fprint(os.Stderr, preflightResult.WarningMessage)

				if !cfg.SkipConfirmations {
					// Prompt for confirmation
					fmt.Fprint(os.Stderr, "\nDo you want to continue anyway? [y/N]: ")
					shouldContinue, promptErr := workflow.PromptUserToContinue("")
					if promptErr != nil {
						return promptErr
					}
					if !shouldContinue {
						return fmt.Errorf("operation cancelled by user")
					}
				} else {
					fmt.Fprintln(os.Stderr, "\nProceeding (skip_confirmations enabled)...")
				}
			}
		}

		// Create workflow orchestrator
		orchestrator := workflow.NewWorkflowOrchestrator(cfg)
		orchestrator.Debug = debug
		orchestrator.Executor.Debug = debug

		if debug {
			fmt.Println("[DEBUG] Debug mode enabled")
			fmt.Printf("[DEBUG] Config: %+v\n", cfg)
			fmt.Printf("[DEBUG] PhaseConfig: %+v\n", phaseConfig)
		}

		// Handle dry run mode - preview without execution
		if dryRun {
			return printDryRunPreview(phaseConfig, featureDescription, specMetadata)
		}

		// Execute phases in canonical order
		return executePhases(orchestrator, phaseConfig, featureDescription, specMetadata, resume, debug)
	},
}

// printDryRunPreview shows what would be executed without actually running
func printDryRunPreview(phaseConfig *workflow.PhaseConfig, featureDescription string, specMetadata *spec.Metadata) error {
	phases := phaseConfig.GetCanonicalOrder()

	fmt.Println("Dry Run Preview")
	fmt.Println("===============")
	fmt.Println()
	fmt.Printf("Phases to execute: %d\n", len(phases))
	fmt.Println()

	fmt.Println("Execution order:")
	for i, phase := range phases {
		fmt.Printf("  %d. %s\n", i+1, phase)
	}
	fmt.Println()

	if specMetadata != nil {
		fmt.Printf("Target spec: specs/%s-%s/\n", specMetadata.Number, specMetadata.Name)
	} else if featureDescription != "" {
		fmt.Printf("Feature description: %s\n", featureDescription)
	}
	fmt.Println()

	// Show what artifacts would be created/modified
	fmt.Println("Artifacts that would be created/modified:")
	for _, phase := range phases {
		switch phase {
		case workflow.PhaseConstitution:
			fmt.Println("  - .autospec/constitution.yaml")
		case workflow.PhaseSpecify:
			fmt.Println("  - specs/<new-spec>/spec.yaml")
		case workflow.PhaseClarify:
			fmt.Println("  - specs/*/spec.yaml (updated)")
		case workflow.PhasePlan:
			fmt.Println("  - specs/*/plan.yaml")
		case workflow.PhaseTasks:
			fmt.Println("  - specs/*/tasks.yaml")
		case workflow.PhaseChecklist:
			fmt.Println("  - specs/*/checklists/*.yaml")
		case workflow.PhaseAnalyze:
			fmt.Println("  - (analysis output, no file changes)")
		case workflow.PhaseImplement:
			fmt.Println("  - (implementation changes to codebase)")
		}
	}
	fmt.Println()
	fmt.Println("No changes made. Remove --dry-run to execute.")

	return nil
}

// phaseExecutionContext holds state during phase execution
type phaseExecutionContext struct {
	orchestrator       *workflow.WorkflowOrchestrator
	featureDescription string
	resume             bool
	specName           string
	specDir            string
	ranImplement       bool
}

// executePhases executes the selected phases in order
func executePhases(orchestrator *workflow.WorkflowOrchestrator, phaseConfig *workflow.PhaseConfig, featureDescription string, specMetadata *spec.Metadata, resume, debug bool) error {
	phases := phaseConfig.GetCanonicalOrder()
	orchestrator.Executor.TotalPhases = len(phases)

	ctx := &phaseExecutionContext{
		orchestrator:       orchestrator,
		featureDescription: featureDescription,
		resume:             resume,
	}

	if specMetadata != nil {
		ctx.specName = fmt.Sprintf("%s-%s", specMetadata.Number, specMetadata.Name)
		ctx.specDir = specMetadata.Directory
	}

	for i, phase := range phases {
		fmt.Printf("[Phase %d/%d] %s...\n", i+1, len(phases), phase)
		if err := ctx.executePhase(phase); err != nil {
			return err
		}
	}

	printWorkflowSummary(phases, ctx.specName, ctx.specDir, ctx.ranImplement)
	return nil
}

// executePhase dispatches to the appropriate phase handler
func (ctx *phaseExecutionContext) executePhase(phase workflow.Phase) error {
	switch phase {
	case workflow.PhaseSpecify:
		return ctx.executeSpecify()
	case workflow.PhasePlan:
		return ctx.executePlan()
	case workflow.PhaseTasks:
		return ctx.executeTasks()
	case workflow.PhaseImplement:
		return ctx.executeImplement()
	case workflow.PhaseConstitution:
		return ctx.executeConstitution()
	case workflow.PhaseClarify:
		return ctx.executeClarify()
	case workflow.PhaseChecklist:
		return ctx.executeChecklist()
	case workflow.PhaseAnalyze:
		return ctx.executeAnalyze()
	default:
		return fmt.Errorf("unknown phase: %s", phase)
	}
}

func (ctx *phaseExecutionContext) executeSpecify() error {
	name, err := ctx.orchestrator.ExecuteSpecify(ctx.featureDescription)
	if err != nil {
		return fmt.Errorf("specify phase failed: %w", err)
	}
	ctx.specName = name
	ctx.specDir = filepath.Join(ctx.orchestrator.SpecsDir, name)
	return nil
}

func (ctx *phaseExecutionContext) executePlan() error {
	if err := ctx.orchestrator.ExecutePlan(ctx.specName, ctx.featureDescription); err != nil {
		return fmt.Errorf("plan phase failed: %w", err)
	}
	return nil
}

func (ctx *phaseExecutionContext) executeTasks() error {
	if err := ctx.orchestrator.ExecuteTasks(ctx.specName, ctx.featureDescription); err != nil {
		return fmt.Errorf("tasks phase failed: %w", err)
	}
	return nil
}

func (ctx *phaseExecutionContext) executeImplement() error {
	phaseOpts := workflow.PhaseExecutionOptions{}
	if err := ctx.orchestrator.ExecuteImplement(ctx.specName, ctx.featureDescription, ctx.resume, phaseOpts); err != nil {
		return fmt.Errorf("implement phase failed: %w", err)
	}
	ctx.ranImplement = true
	return nil
}

func (ctx *phaseExecutionContext) executeConstitution() error {
	if err := ctx.orchestrator.ExecuteConstitution(ctx.featureDescription); err != nil {
		return fmt.Errorf("constitution phase failed: %w", err)
	}
	return nil
}

func (ctx *phaseExecutionContext) executeClarify() error {
	if err := ctx.orchestrator.ExecuteClarify(ctx.specName, ctx.featureDescription); err != nil {
		return fmt.Errorf("clarify phase failed: %w", err)
	}
	return nil
}

func (ctx *phaseExecutionContext) executeChecklist() error {
	if err := ctx.orchestrator.ExecuteChecklist(ctx.specName, ctx.featureDescription); err != nil {
		return fmt.Errorf("checklist phase failed: %w", err)
	}
	return nil
}

func (ctx *phaseExecutionContext) executeAnalyze() error {
	if err := ctx.orchestrator.ExecuteAnalyze(ctx.specName, ctx.featureDescription); err != nil {
		return fmt.Errorf("analyze phase failed: %w", err)
	}
	return nil
}

// printWorkflowSummary prints a comprehensive summary after workflow completion
func printWorkflowSummary(phases []workflow.Phase, specName, specDir string, ranImplement bool) {
	fmt.Println()

	// If implement ran, show task completion stats
	if ranImplement && specDir != "" {
		tasksPath := validation.GetTasksFilePath(specDir)
		stats, err := validation.GetTaskStats(tasksPath)
		if err == nil && stats.TotalTasks > 0 {
			fmt.Println("Task Summary:")
			fmt.Print(validation.FormatTaskSummary(stats))
			fmt.Println()
		}
	}

	// Show workflow phases completed
	fmt.Printf("Completed %d workflow phase(s): ", len(phases))
	phaseNames := make([]string, len(phases))
	for i, p := range phases {
		phaseNames[i] = string(p)
	}
	fmt.Println(joinPhaseNames(phaseNames))

	if specName != "" {
		fmt.Printf("Spec: specs/%s/\n", specName)
	}
}

// joinPhaseNames joins phase names with arrows for display
func joinPhaseNames(names []string) string {
	if len(names) == 0 {
		return ""
	}
	if len(names) == 1 {
		return names[0]
	}

	result := names[0]
	for i := 1; i < len(names); i++ {
		result += " → " + names[i]
	}
	return result
}

func init() {
	runCmd.GroupID = GroupWorkflows
	rootCmd.AddCommand(runCmd)

	// Core phase selection flags
	runCmd.Flags().BoolP("specify", "s", false, "Include specify phase")
	runCmd.Flags().BoolP("plan", "p", false, "Include plan phase")
	runCmd.Flags().BoolP("tasks", "t", false, "Include tasks phase")
	runCmd.Flags().BoolP("implement", "i", false, "Include implement phase")
	runCmd.Flags().BoolP("all", "a", false, "Run all core phases (equivalent to -spti)")

	// Optional phase selection flags
	// Note: -c is already used globally for --config, so checklist uses -l
	runCmd.Flags().BoolP("constitution", "n", false, "Include constitution phase")
	runCmd.Flags().BoolP("clarify", "r", false, "Include clarify phase")
	runCmd.Flags().BoolP("checklist", "l", false, "Include checklist phase")
	runCmd.Flags().BoolP("analyze", "z", false, "Include analyze phase")

	// Spec selection
	runCmd.Flags().String("spec", "", "Specify which spec to work with (overrides branch detection)")

	// Skip confirmation
	runCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompts")

	// Other flags (NOTE: max-retries is now long-only, -r is used for clarify)
	runCmd.Flags().Int("max-retries", 0, "Override max retry attempts (0 = use config)")
	runCmd.Flags().Bool("resume", false, "Resume implementation from where it left off")
	runCmd.Flags().Bool("progress", false, "Show progress indicators (spinners) during execution")
	runCmd.Flags().Bool("dry-run", false, "Preview what phases would run without executing")
}
