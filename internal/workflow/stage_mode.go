package workflow

// StageMode represents the execution mode for a workflow stage.
// Interactive mode runs without -p flag, allowing user conversation.
// Automated mode uses -p flag and stream-json output for unattended execution.
type StageMode int

const (
	// StageModeAutomated indicates the stage runs with -p flag and stream-json output.
	// Used for file-modifying stages: specify, plan, tasks, implement, constitution, checklist.
	StageModeAutomated StageMode = iota

	// StageModeInteractive indicates the stage runs without -p flag.
	// Used for recommendation-focused stages: analyze, clarify.
	StageModeInteractive
)

// interactiveStages defines which stages run in interactive mode.
// Interactive stages are recommendation-focused and benefit from user conversation.
var interactiveStages = map[Stage]bool{
	StageAnalyze: true,
	StageClarify: true,
}

// IsInteractive returns true if the given stage should run in interactive mode.
// Interactive stages (analyze, clarify) skip -p flag and --output-format stream-json
// to allow multi-turn conversation with the user.
func IsInteractive(stage Stage) bool {
	return interactiveStages[stage]
}
