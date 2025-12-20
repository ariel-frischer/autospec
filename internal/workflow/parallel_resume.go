// Package workflow provides parallel execution resume functionality.
package workflow

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ResumeOption represents user's choice for handling interrupted execution.
type ResumeOption int

const (
	// ResumeRetry retries failed tasks and continues from where it left off.
	ResumeRetry ResumeOption = iota
	// ResumeSkipWave skips the current wave and continues to the next.
	ResumeSkipWave
	// ResumeAbort aborts the entire execution.
	ResumeAbort
	// ResumeReset clears state and starts from scratch.
	ResumeReset
)

// String returns the string representation of a ResumeOption.
func (r ResumeOption) String() string {
	switch r {
	case ResumeRetry:
		return "Retry"
	case ResumeSkipWave:
		return "Skip Wave"
	case ResumeAbort:
		return "Abort"
	case ResumeReset:
		return "Reset"
	default:
		return "Unknown"
	}
}

// PromptResumeOption prompts the user to choose how to handle an interrupted execution.
// Returns the chosen option.
func PromptResumeOption(state *ParallelExecutionState) (ResumeOption, error) {
	fmt.Println()
	fmt.Println("Previous parallel execution was interrupted:")
	fmt.Printf("  Spec: %s\n", state.SpecName)
	fmt.Printf("  Started: %s\n", state.StartedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Progress: Wave %d/%d\n", state.CurrentWave, state.TotalWaves)

	// Count task statuses
	completed := len(state.GetCompletedTasks())
	failed := len(state.FailedTasks)
	skipped := len(state.SkippedTasks)
	pending := len(state.GetPendingTasks())

	fmt.Printf("  Tasks: %d completed, %d failed, %d skipped, %d pending\n",
		completed, failed, skipped, pending)
	fmt.Println()

	fmt.Println("How would you like to proceed?")
	fmt.Println("  [R] Retry - Retry failed tasks and continue from where it left off")
	fmt.Println("  [W] Skip Wave - Skip current wave and continue to the next")
	fmt.Println("  [S] Start Fresh - Clear state and start from the beginning")
	fmt.Println("  [A] Abort - Cancel and exit")
	fmt.Println()
	fmt.Print("Choice [R/W/S/A] (default: R): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return ResumeAbort, fmt.Errorf("reading input: %w", err)
	}

	input = strings.TrimSpace(strings.ToUpper(input))
	if input == "" {
		input = "R" // Default
	}

	switch input {
	case "R", "RETRY":
		return ResumeRetry, nil
	case "W", "WAVE", "SKIP":
		return ResumeSkipWave, nil
	case "S", "START", "FRESH", "RESET":
		return ResumeReset, nil
	case "A", "ABORT", "QUIT", "EXIT", "Q":
		return ResumeAbort, nil
	default:
		fmt.Printf("Unknown option '%s', defaulting to Retry\n", input)
		return ResumeRetry, nil
	}
}

// ShouldPromptResume checks if we should prompt for resume based on state.
func ShouldPromptResume(state *ParallelExecutionState) bool {
	if state == nil {
		return false
	}
	// Don't prompt if already completed
	if state.IsComplete() {
		return false
	}
	// Don't prompt if no progress was made
	if state.CurrentWave == 0 {
		return false
	}
	return true
}

// ApplyResumeOption applies the user's resume choice to the state.
// Returns the wave number to start from, or an error if abort was chosen.
func ApplyResumeOption(option ResumeOption, state *ParallelExecutionState, stateDir string) (int, error) {
	switch option {
	case ResumeRetry:
		// Start from the interrupted wave
		return state.GetResumeWave(), nil

	case ResumeSkipWave:
		// Mark current wave tasks as skipped and move to next
		skipWave := state.CurrentWave
		if skipWave > 0 && skipWave <= state.TotalWaves {
			for taskID, status := range state.TaskStatuses {
				if status == "running" || status == "pending" {
					state.RecordTaskSkipped(taskID, fmt.Sprintf("skipped wave %d", skipWave))
				}
			}
			state.CompleteWave(skipWave, 0, 0, len(state.SkippedTasks))
		}
		return skipWave + 1, nil

	case ResumeReset:
		// Delete state and start fresh
		if err := DeleteParallelState(stateDir, state.SpecName); err != nil {
			return 0, fmt.Errorf("clearing state: %w", err)
		}
		return 1, nil

	case ResumeAbort:
		return 0, fmt.Errorf("aborted by user")

	default:
		return 0, fmt.Errorf("unknown resume option: %v", option)
	}
}
