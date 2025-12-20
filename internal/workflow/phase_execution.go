package workflow

// PhaseExecutionMode represents the type of phase execution
type PhaseExecutionMode int

const (
	// ModeDefault executes all tasks in a single Claude session (backward compatible)
	ModeDefault PhaseExecutionMode = iota
	// ModeAllPhases executes each phase in a separate Claude session
	ModeAllPhases
	// ModeSinglePhase executes only a specific phase
	ModeSinglePhase
	// ModeFromPhase executes from a specific phase to the end
	ModeFromPhase
	// ModeAllTasks executes each task in a separate Claude session
	ModeAllTasks
	// ModeParallel executes independent tasks concurrently using DAG-based wave scheduling
	ModeParallel
)

// PhaseExecutionOptions contains configuration for phase-based execution
type PhaseExecutionOptions struct {
	// RunAllPhases indicates --phases flag was set (run each phase in separate session)
	RunAllPhases bool
	// SinglePhase is the specific phase to run (--phase N, 0 = not set)
	SinglePhase int
	// FromPhase is the starting phase (--from-phase N, 0 = not set)
	FromPhase int
	// TaskMode indicates --tasks flag was set (run each task in separate session)
	TaskMode bool
	// FromTask is the task ID to start from (--from-task TXXX, empty = not set)
	FromTask string
	// ParallelMode indicates --parallel flag was set (DAG-based concurrent execution)
	ParallelMode bool
	// MaxParallel is the maximum number of concurrent Claude sessions (default 4)
	MaxParallel int
	// UseWorktrees indicates --worktrees flag was set (git worktree isolation)
	UseWorktrees bool
	// DryRun indicates --dry-run flag was set (preview execution plan only)
	DryRun bool
	// SkipConfirmation indicates --yes flag was set (bypass confirmation prompts)
	SkipConfirmation bool
}

// Mode determines the execution mode from the options
func (o *PhaseExecutionOptions) Mode() PhaseExecutionMode {
	if o.ParallelMode {
		return ModeParallel
	}
	if o.TaskMode {
		return ModeAllTasks
	}
	if o.RunAllPhases {
		return ModeAllPhases
	}
	if o.SinglePhase > 0 {
		return ModeSinglePhase
	}
	if o.FromPhase > 0 {
		return ModeFromPhase
	}
	return ModeDefault
}

// IsDefault returns true if no phase flags were set
func (o *PhaseExecutionOptions) IsDefault() bool {
	return o.Mode() == ModeDefault
}

// PhaseExecutionResult contains the result of a phase execution
type PhaseExecutionResult struct {
	PhaseNumber    int
	PhaseTitle     string
	Success        bool
	TasksCompleted int
	TasksTotal     int
	Error          error
}

// PhaseExecutionSummary contains summary of all phases executed
type PhaseExecutionSummary struct {
	TotalPhases     int
	CompletedPhases int
	SkippedPhases   int
	FailedPhases    int
	Results         []PhaseExecutionResult
}
