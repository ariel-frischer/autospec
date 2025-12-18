package cli

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/ariel-frischer/autospec/internal/history"
	"github.com/ariel-frischer/autospec/internal/lifecycle"
	"github.com/ariel-frischer/autospec/internal/notify"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/ariel-frischer/autospec/internal/workflow"
	"github.com/spf13/cobra"
)

var implementCmd = &cobra.Command{
	Use:     "implement [spec-name-or-prompt]",
	Aliases: []string{"impl", "i"},
	Short:   "Execute the implementation stage for the current spec (impl, i)",
	Long: `Execute the /autospec.implement command for the current specification.

The implement command will:
- Auto-detect the current spec from git branch or most recent spec
- Execute the implementation workflow based on tasks.yaml
- Track progress and validate task completion
- Support resuming from where it left off with --resume flag

Execution Modes:
- Default (no flags): Uses implement_method from config (default: 'phases')
- --phases: Run each phase in a separate Claude session (fresh context per phase)
- --phase N: Run only phase N in a fresh Claude session
- --from-phase N: Run phases N through end, each in a fresh session
- --tasks: Run each task in a separate Claude session (finest granularity)
- --from-task T003: Start task-level execution from a specific task ID
- --single-session: Run all tasks in one Claude session (legacy mode)

The default execution mode can be configured in config.yml:
  implement_method: phases     # Each phase in separate session (default)
  implement_method: tasks      # Each task in separate session
  implement_method: single-session  # All tasks in one session (legacy)

CLI flags always override the config setting. Environment variable
AUTOSPEC_IMPLEMENT_METHOD can also be used to set the default.

The --phases mode provides benefits for large implementations:
- Fresh context per phase reduces attention degradation
- Lower token usage per session
- Natural recovery points if execution fails
- Clearer progress visibility (Phase X/Y displayed)

The --tasks mode provides maximum context isolation:
- Each task gets a completely fresh Claude session
- Ideal for complex or long-running tasks
- Finest-grained recovery points
- Can combine with --from-task to resume from specific task`,
	Example: `  # Auto-detect spec and implement
  autospec implement

  # Resume interrupted implementation
  autospec implement --resume

  # Implement a specific spec by name
  autospec implement 003-my-feature

  # Provide prompt guidance for implementation
  autospec implement "Focus on error handling first"

  # Run each phase in a separate Claude session
  autospec implement --phases

  # Run only phase 3
  autospec implement --phase 3

  # Resume from phase 3 onwards
  autospec implement --from-phase 3

  # Run each task in a separate Claude session
  autospec implement --tasks

  # Resume task execution from a specific task
  autospec implement --tasks --from-task T003

  # Run all tasks in a single Claude session (legacy mode)
  autospec implement --single-session`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse args to distinguish between spec-name and prompt
		specName, prompt := parseImplementArgs(args)

		// Get flags
		configPath, _ := cmd.Flags().GetString("config")
		skipPreflight, _ := cmd.Flags().GetBool("skip-preflight")
		maxRetries, _ := cmd.Flags().GetInt("max-retries")
		resume, _ := cmd.Flags().GetBool("resume")

		// Get phase execution flags
		runAllPhases, _ := cmd.Flags().GetBool("phases")
		singlePhase, _ := cmd.Flags().GetInt("phase")
		fromPhase, _ := cmd.Flags().GetInt("from-phase")

		// Get task execution flags
		taskMode, _ := cmd.Flags().GetBool("tasks")
		fromTask, _ := cmd.Flags().GetString("from-task")

		// Get single-session flag
		singleSession, _ := cmd.Flags().GetBool("single-session")

		// Validate phase flag values
		if singlePhase < 0 {
			cliErr := clierrors.NewArgumentError("--phase must be a positive integer")
			clierrors.PrintError(cliErr)
			return cliErr
		}
		if fromPhase < 0 {
			cliErr := clierrors.NewArgumentError("--from-phase must be a positive integer")
			clierrors.PrintError(cliErr)
			return cliErr
		}

		// Load configuration
		cfg, err := config.Load(configPath)
		if err != nil {
			cliErr := clierrors.ConfigParseError(configPath, err)
			clierrors.PrintError(cliErr)
			return cliErr
		}

		// Override skip-preflight from flag if set
		if cmd.Flags().Changed("skip-preflight") {
			cfg.SkipPreflight = skipPreflight
		}

		// Override max-retries from flag if set
		if cmd.Flags().Changed("max-retries") {
			cfg.MaxRetries = maxRetries
		}

		// Resolve execution mode based on flags and config
		anyFlagsChanged := cmd.Flags().Changed("phases") ||
			cmd.Flags().Changed("tasks") ||
			cmd.Flags().Changed("phase") ||
			cmd.Flags().Changed("from-phase") ||
			cmd.Flags().Changed("from-task") ||
			cmd.Flags().Changed("single-session")

		execMode := resolveExecutionMode(
			ExecutionModeFlags{
				PhasesFlag:        runAllPhases,
				TasksFlag:         taskMode,
				SingleSessionFlag: singleSession,
				PhaseFlag:         singlePhase,
				FromPhaseFlag:     fromPhase,
				FromTaskFlag:      fromTask,
			},
			anyFlagsChanged,
			cfg.ImplementMethod,
		)
		runAllPhases = execMode.RunAllPhases
		taskMode = execMode.TaskMode

		// Check if constitution exists (required for implement)
		constitutionCheck := workflow.CheckConstitutionExists()
		if !constitutionCheck.Exists {
			fmt.Fprint(os.Stderr, constitutionCheck.ErrorMessage)
			return NewExitError(ExitInvalidArguments)
		}

		// Auto-detect spec directory for prerequisite validation
		metadata, err := spec.DetectCurrentSpec(cfg.SpecsDir)
		if err != nil {
			return fmt.Errorf("failed to detect current spec: %w\n\nRun 'autospec specify' to create a new spec first", err)
		}
		PrintSpecInfo(metadata)

		// Validate tasks.yaml exists (required for implement stage)
		prereqResult := workflow.ValidateStagePrerequisites(workflow.StageImplement, metadata.Directory)
		if !prereqResult.Valid {
			fmt.Fprint(os.Stderr, prereqResult.ErrorMessage)
			return NewExitError(ExitInvalidArguments)
		}

		// Create notification handler and history logger
		notifHandler := notify.NewHandler(cfg.Notifications)
		historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)
		historySpecName := fmt.Sprintf("%s-%s", metadata.Number, metadata.Name)

		// Wrap command execution with lifecycle for timing, notification, and history
		// Use RunWithHistoryContext to support context cancellation (e.g., Ctrl+C)
		return lifecycle.RunWithHistoryContext(cmd.Context(), notifHandler, historyLogger, "implement", historySpecName, func(_ context.Context) error {
			// Create workflow orchestrator
			orch := workflow.NewWorkflowOrchestrator(cfg)
			orch.Executor.NotificationHandler = notifHandler

			// Build phase execution options
			phaseOpts := workflow.PhaseExecutionOptions{
				RunAllPhases: runAllPhases,
				SinglePhase:  singlePhase,
				FromPhase:    fromPhase,
				TaskMode:     taskMode,
				FromTask:     fromTask,
			}

			// Execute implement stage with optional prompt and phase options
			if err := orch.ExecuteImplement(specName, prompt, resume, phaseOpts); err != nil {
				return fmt.Errorf("implement stage failed: %w", err)
			}

			return nil
		})
	},
}

// specNamePattern matches spec names like "003-feature-name" or "42-answer"
var specNamePattern = regexp.MustCompile(`^\d+-[a-z0-9-]+$`)

// parseImplementArgs parses command arguments to distinguish between spec names and prompts.
// Returns the spec name (if first arg matches NNN-name pattern) and any remaining prompt text.
func parseImplementArgs(args []string) (specName, prompt string) {
	if len(args) == 0 {
		return "", ""
	}

	if specNamePattern.MatchString(args[0]) {
		specName = args[0]
		if len(args) > 1 {
			prompt = strings.Join(args[1:], " ")
		}
	} else {
		prompt = strings.Join(args, " ")
	}
	return specName, prompt
}

// ExecutionModeFlags holds the flag values for execution mode resolution
type ExecutionModeFlags struct {
	PhasesFlag        bool
	TasksFlag         bool
	SingleSessionFlag bool
	PhaseFlag         int
	FromPhaseFlag     int
	FromTaskFlag      string
}

// ExecutionModeResult holds the resolved execution mode
type ExecutionModeResult struct {
	RunAllPhases bool
	TaskMode     bool
	SinglePhase  int
	FromPhase    int
	FromTask     string
}

// resolveExecutionMode determines the execution mode based on CLI flags and config.
// CLI flags take precedence over config settings.
func resolveExecutionMode(flags ExecutionModeFlags, flagsChanged bool, configMethod string) ExecutionModeResult {
	result := ExecutionModeResult{
		RunAllPhases: flags.PhasesFlag,
		TaskMode:     flags.TasksFlag,
		SinglePhase:  flags.PhaseFlag,
		FromPhase:    flags.FromPhaseFlag,
		FromTask:     flags.FromTaskFlag,
	}

	// If --single-session flag is explicitly set, ensure phase/task modes are disabled
	if flags.SingleSessionFlag {
		result.RunAllPhases = false
		result.TaskMode = false
		return result
	}

	// Apply config default execution mode when no execution mode flags are provided
	if !flagsChanged && configMethod != "" {
		switch configMethod {
		case "phases":
			result.RunAllPhases = true
		case "tasks":
			result.TaskMode = true
		case "single-session":
			// Legacy behavior: no phase/task mode (default state)
		}
	}

	return result
}

func init() {
	implementCmd.GroupID = GroupCoreStages
	rootCmd.AddCommand(implementCmd)

	// Command-specific flags
	implementCmd.Flags().Bool("resume", false, "Resume implementation from where it left off")
	implementCmd.Flags().IntP("max-retries", "r", 0, "Override max retry attempts (0 = use config)")

	// Phase execution flags
	implementCmd.Flags().Bool("phases", false, "Run each phase in a separate Claude session (fresh context per phase)")
	implementCmd.Flags().Int("phase", 0, "Run only a specific phase number (e.g., --phase 3)")
	implementCmd.Flags().Int("from-phase", 0, "Start execution from a specific phase (e.g., --from-phase 3)")

	// Task execution flags
	implementCmd.Flags().Bool("tasks", false, "Run each task in a separate Claude session (finest granularity)")
	implementCmd.Flags().String("from-task", "", "Start execution from a specific task ID (e.g., --from-task T003)")

	// Single-session flag (legacy mode)
	implementCmd.Flags().Bool("single-session", false, "Run all tasks in one Claude session (legacy mode)")

	// Mark phase flags as mutually exclusive
	implementCmd.MarkFlagsMutuallyExclusive("phases", "phase", "from-phase")

	// Mark task flags as mutually exclusive with phase flags
	// --tasks cannot be used with any phase-level flags
	implementCmd.MarkFlagsMutuallyExclusive("tasks", "phases")
	implementCmd.MarkFlagsMutuallyExclusive("tasks", "phase")
	implementCmd.MarkFlagsMutuallyExclusive("tasks", "from-phase")

	// Mark single-session as mutually exclusive with all other execution modes
	implementCmd.MarkFlagsMutuallyExclusive("single-session", "phases")
	implementCmd.MarkFlagsMutuallyExclusive("single-session", "phase")
	implementCmd.MarkFlagsMutuallyExclusive("single-session", "from-phase")
	implementCmd.MarkFlagsMutuallyExclusive("single-session", "tasks")
}
