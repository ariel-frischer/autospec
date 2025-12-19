package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/ariel-frischer/autospec/internal/history"
	"github.com/ariel-frischer/autospec/internal/lifecycle"
	"github.com/ariel-frischer/autospec/internal/notify"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/ariel-frischer/autospec/internal/validation"
	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [feature-description]",
	Short: "Run selected workflow stages with flexible stage selection",
	Long: `Run selected workflow stages with flexible stage selection.

Core stage flags:
  -s, --specify    Include specify stage (requires feature description)
  -p, --plan       Include plan stage
  -t, --tasks      Include tasks stage
  -i, --implement  Include implement stage
  -a, --all        Run all core stages (equivalent to -spti)

Optional stage flags:
  -n, --constitution  Include constitution stage
  -r, --clarify       Include clarify stage
  -l, --checklist     Include checklist stage (note: -c is used for --config)
  -z, --analyze       Include analyze stage

Stages are always executed in canonical order:
  constitution -> specify -> clarify -> plan -> tasks -> checklist -> analyze -> implement`,
	Example: `  # Run all core stages for a new feature
  autospec run -a "Add user authentication"

  # Run only plan and implement on current spec
  autospec run -pi

  # Run tasks and implement on a specific spec
  autospec run -ti --spec 007-yaml-output

  # Preview what stages would run (dry run mode)
  autospec run -ti --dry-run

  # Skip confirmation prompts for CI/CD
  autospec run -ti -y`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get core stage flags
		specify, _ := cmd.Flags().GetBool("specify")
		plan, _ := cmd.Flags().GetBool("plan")
		tasks, _ := cmd.Flags().GetBool("tasks")
		implement, _ := cmd.Flags().GetBool("implement")
		all, _ := cmd.Flags().GetBool("all")

		// Get optional stage flags
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
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Build StageConfig from flags
		stageConfig := workflow.NewStageConfig()
		if all {
			stageConfig.SetAll() // SetAll only sets core stages (specify, plan, tasks, implement)
		} else {
			// Core stages
			stageConfig.Specify = specify
			stageConfig.Plan = plan
			stageConfig.Tasks = tasks
			stageConfig.Implement = implement
		}
		// Optional stages are always set from flags (can be combined with -a)
		stageConfig.Constitution = constitution
		stageConfig.Clarify = clarify
		stageConfig.Checklist = checklist
		stageConfig.Analyze = analyze

		// Validate at least one stage is selected
		if !stageConfig.HasAnyStage() {
			return fmt.Errorf("no stages selected. Use -s/-p/-t/-i flags or -a for all stages\n\nRun 'autospec run --help' for usage")
		}

		// Get feature description from args if specify stage is selected
		var featureDescription string
		if stageConfig.Specify {
			if len(args) < 1 {
				return fmt.Errorf("feature description required when using specify stage (-s)\n\nUsage: autospec run -s \"feature description\"")
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

		// Resolve skip confirmations (flag > env > config)
		if skipConfirm || os.Getenv("AUTOSPEC_YES") != "" || cfg.SkipConfirmations {
			cfg.SkipConfirmations = true
		}

		// Check if constitution exists (required unless only running constitution stage)
		if !stageConfig.Constitution || stageConfig.Count() > 1 {
			// Either not running constitution at all, or running other stages too
			constitutionCheck := workflow.CheckConstitutionExists()
			if !constitutionCheck.Exists {
				fmt.Fprint(os.Stderr, constitutionCheck.ErrorMessage)
				return fmt.Errorf("constitution required")
			}
		}

		// Detect or validate spec name
		var specMetadata *spec.Metadata
		if !stageConfig.Specify {
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
				PrintSpecInfo(specMetadata)
			}
		}

		// Check artifact dependencies before execution - hard fail if missing
		// These are artifacts that no earlier selected stage will produce
		if !stageConfig.Specify {
			preflightResult := workflow.CheckArtifactDependencies(stageConfig, specMetadata.Directory)
			if len(preflightResult.MissingArtifacts) > 0 {
				fmt.Fprint(os.Stderr, preflightResult.WarningMessage)
				return NewExitError(ExitInvalidArguments)
			}
		}

		// Create workflow orchestrator
		orchestrator := workflow.NewWorkflowOrchestrator(cfg)
		orchestrator.Debug = debug
		orchestrator.Executor.Debug = debug

		if debug {
			fmt.Println("[DEBUG] Debug mode enabled")
			fmt.Printf("[DEBUG] Config: %+v\n", cfg)
			fmt.Printf("[DEBUG] StageConfig: %+v\n", stageConfig)
		}

		// Handle dry run mode - preview without execution
		if dryRun {
			return printDryRunPreview(stageConfig, featureDescription, specMetadata)
		}

		// Create history logger
		historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)

		// Execute stages in canonical order with context for cancellation support
		// Pass 'all' flag as isFullWorkflow to control description propagation
		return executeStages(cmd.Context(), orchestrator, stageConfig, featureDescription, specMetadata, resume, debug, cfg.ImplementMethod, all, historyLogger)
	},
}

// printDryRunPreview shows what would be executed without actually running
func printDryRunPreview(stageConfig *workflow.StageConfig, featureDescription string, specMetadata *spec.Metadata) error {
	stages := stageConfig.GetCanonicalOrder()

	fmt.Println("Dry Run Preview")
	fmt.Println("===============")
	fmt.Println()
	fmt.Printf("Stages to execute: %d\n", len(stages))
	fmt.Println()

	fmt.Println("Execution order:")
	for i, stage := range stages {
		fmt.Printf("  %d. %s\n", i+1, stage)
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
	for _, stage := range stages {
		switch stage {
		case workflow.StageConstitution:
			fmt.Println("  - .autospec/constitution.yaml")
		case workflow.StageSpecify:
			fmt.Println("  - specs/<new-spec>/spec.yaml")
		case workflow.StageClarify:
			fmt.Println("  - specs/*/spec.yaml (updated)")
		case workflow.StagePlan:
			fmt.Println("  - specs/*/plan.yaml")
		case workflow.StageTasks:
			fmt.Println("  - specs/*/tasks.yaml")
		case workflow.StageChecklist:
			fmt.Println("  - specs/*/checklists/*.yaml")
		case workflow.StageAnalyze:
			fmt.Println("  - (analysis output, no file changes)")
		case workflow.StageImplement:
			fmt.Println("  - (implementation changes to codebase)")
		}
	}
	fmt.Println()
	fmt.Println("No changes made. Remove --dry-run to execute.")

	return nil
}

// stageExecutionContext holds state during stage execution
type stageExecutionContext struct {
	orchestrator        *workflow.WorkflowOrchestrator
	notificationHandler *notify.Handler
	featureDescription  string
	// isFullWorkflow is true when -a flag was used, indicating description should only
	// go to specify stage. When true, plan/tasks/implement receive empty prompts to
	// ensure they work from structured artifacts rather than raw feature descriptions.
	isFullWorkflow  bool
	resume          bool
	implementMethod string
	specName        string
	specDir         string
	ranImplement    bool
}

// executeStages executes the selected stages in order
// isFullWorkflow indicates whether -a flag was used (all core stages), which affects
// how featureDescription is propagated: only to specify when true, to all stages when false.
func executeStages(cmdCtx context.Context, orchestrator *workflow.WorkflowOrchestrator, stageConfig *workflow.StageConfig, featureDescription string, specMetadata *spec.Metadata, resume, debug bool, implementMethod string, isFullWorkflow bool, historyLogger *history.Writer) error {
	stages := stageConfig.GetCanonicalOrder()
	orchestrator.Executor.TotalStages = len(stages)

	// Create notification handler from config
	notifHandler := notify.NewHandler(orchestrator.Config.Notifications)
	orchestrator.Executor.NotificationHandler = notifHandler

	ctx := &stageExecutionContext{
		orchestrator:        orchestrator,
		notificationHandler: notifHandler,
		featureDescription:  featureDescription,
		isFullWorkflow:      isFullWorkflow,
		resume:              resume,
		implementMethod:     implementMethod,
	}

	if specMetadata != nil {
		ctx.specName = fmt.Sprintf("%s-%s", specMetadata.Number, specMetadata.Name)
		ctx.specDir = specMetadata.Directory
	}

	// Wrap stage execution with lifecycle for timing, notification, and history
	// Use RunWithHistoryContext to support context cancellation (e.g., Ctrl+C)
	// Note: spec name may be empty if starting with specify stage
	return lifecycle.RunWithHistoryContext(cmdCtx, notifHandler, historyLogger, "run", ctx.specName, func(_ context.Context) error {
		for i, stage := range stages {
			fmt.Printf("[Stage %d/%d] %s...\n", i+1, len(stages), stage)
			if err := ctx.executeStage(stage); err != nil {
				return fmt.Errorf("executing stage %s: %w", stage, err)
			}
		}

		printWorkflowSummary(stages, ctx.specName, ctx.specDir, ctx.ranImplement)
		return nil
	})
}

// executeStage dispatches to the appropriate stage handler
func (ctx *stageExecutionContext) executeStage(stage workflow.Stage) error {
	switch stage {
	case workflow.StageSpecify:
		return ctx.executeSpecify()
	case workflow.StagePlan:
		return ctx.executePlan()
	case workflow.StageTasks:
		return ctx.executeTasks()
	case workflow.StageImplement:
		return ctx.executeImplement()
	case workflow.StageConstitution:
		return ctx.executeConstitution()
	case workflow.StageClarify:
		return ctx.executeClarify()
	case workflow.StageChecklist:
		return ctx.executeChecklist()
	case workflow.StageAnalyze:
		return ctx.executeAnalyze()
	default:
		return fmt.Errorf("unknown stage: %s", stage)
	}
}

func (ctx *stageExecutionContext) executeSpecify() error {
	name, err := ctx.orchestrator.ExecuteSpecify(ctx.featureDescription)
	if err != nil {
		return fmt.Errorf("specify stage failed: %w", err)
	}
	ctx.specName = name
	ctx.specDir = filepath.Join(ctx.orchestrator.SpecsDir, name)
	return nil
}

func (ctx *stageExecutionContext) executePlan() error {
	// When running full workflow (-a), pass empty prompt so plan works from spec.yaml artifacts.
	// When running individual stages, pass the user's hint/description to the stage.
	prompt := ctx.featureDescription
	if ctx.isFullWorkflow {
		prompt = ""
	}
	if err := ctx.orchestrator.ExecutePlan(ctx.specName, prompt); err != nil {
		return fmt.Errorf("plan stage failed: %w", err)
	}
	return nil
}

func (ctx *stageExecutionContext) executeTasks() error {
	// When running full workflow (-a), pass empty prompt so tasks works from plan.yaml artifacts.
	// When running individual stages, pass the user's hint/description to the stage.
	prompt := ctx.featureDescription
	if ctx.isFullWorkflow {
		prompt = ""
	}
	if err := ctx.orchestrator.ExecuteTasks(ctx.specName, prompt); err != nil {
		return fmt.Errorf("tasks stage failed: %w", err)
	}
	return nil
}

func (ctx *stageExecutionContext) executeImplement() error {
	// Build phase options from config's implement_method setting
	phaseOpts := workflow.PhaseExecutionOptions{}
	switch ctx.implementMethod {
	case "phases":
		phaseOpts.RunAllPhases = true
	case "tasks":
		phaseOpts.TaskMode = true
	case "single-session":
		// Legacy behavior: no phase/task mode (default state)
	}
	// When running full workflow (-a), pass empty prompt so implement works from tasks.yaml artifacts.
	// When running individual stages, pass the user's hint/description to the stage.
	prompt := ctx.featureDescription
	if ctx.isFullWorkflow {
		prompt = ""
	}
	if err := ctx.orchestrator.ExecuteImplement(ctx.specName, prompt, ctx.resume, phaseOpts); err != nil {
		return fmt.Errorf("implement stage failed: %w", err)
	}
	ctx.ranImplement = true
	return nil
}

func (ctx *stageExecutionContext) executeConstitution() error {
	if err := ctx.orchestrator.ExecuteConstitution(ctx.featureDescription); err != nil {
		return fmt.Errorf("constitution stage failed: %w", err)
	}
	return nil
}

func (ctx *stageExecutionContext) executeClarify() error {
	if err := ctx.orchestrator.ExecuteClarify(ctx.specName, ctx.featureDescription); err != nil {
		return fmt.Errorf("clarify stage failed: %w", err)
	}
	return nil
}

func (ctx *stageExecutionContext) executeChecklist() error {
	if err := ctx.orchestrator.ExecuteChecklist(ctx.specName, ctx.featureDescription); err != nil {
		return fmt.Errorf("checklist stage failed: %w", err)
	}
	return nil
}

func (ctx *stageExecutionContext) executeAnalyze() error {
	if err := ctx.orchestrator.ExecuteAnalyze(ctx.specName, ctx.featureDescription); err != nil {
		return fmt.Errorf("analyze stage failed: %w", err)
	}
	return nil
}

// printWorkflowSummary prints a comprehensive summary after workflow completion
func printWorkflowSummary(stages []workflow.Stage, specName, specDir string, ranImplement bool) {
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

	// Show workflow stages completed
	fmt.Printf("Completed %d workflow stage(s): ", len(stages))
	stageNames := make([]string, len(stages))
	for i, s := range stages {
		stageNames[i] = string(s)
	}
	fmt.Println(joinStageNames(stageNames))

	if specName != "" {
		fmt.Printf("Spec: specs/%s/\n", specName)
	}
}

// joinStageNames joins stage names with arrows for display
func joinStageNames(names []string) string {
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

	// Core stage selection flags
	runCmd.Flags().BoolP("specify", "s", false, "Include specify stage")
	runCmd.Flags().BoolP("plan", "p", false, "Include plan stage")
	runCmd.Flags().BoolP("tasks", "t", false, "Include tasks stage")
	runCmd.Flags().BoolP("implement", "i", false, "Include implement stage")
	runCmd.Flags().BoolP("all", "a", false, "Run all core stages (equivalent to -spti)")

	// Optional stage selection flags
	// Note: -c is already used globally for --config, so checklist uses -l
	runCmd.Flags().BoolP("constitution", "n", false, "Include constitution stage")
	runCmd.Flags().BoolP("clarify", "r", false, "Include clarify stage")
	runCmd.Flags().BoolP("checklist", "l", false, "Include checklist stage")
	runCmd.Flags().BoolP("analyze", "z", false, "Include analyze stage")

	// Spec selection
	runCmd.Flags().String("spec", "", "Specify which spec to work with (overrides branch detection)")

	// Skip confirmation
	runCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompts")

	// Other flags (NOTE: max-retries is now long-only, -r is used for clarify)
	runCmd.Flags().Int("max-retries", 0, "Override max retry attempts (overrides config when set)")
	runCmd.Flags().Bool("resume", false, "Resume implementation from where it left off")
	runCmd.Flags().Bool("dry-run", false, "Preview what stages would run without executing")
}
