// Package validation provides validation functions for autospec artifacts.
package validation

import (
	"fmt"
	"strings"
)

// PhaseDisplayInfo contains information formatted for display in pre/post-phase messages.
type PhaseDisplayInfo struct {
	// PhaseNumber is the current phase number (1-based)
	PhaseNumber int
	// TotalPhases is the total number of phases
	TotalPhases int
	// Title is the phase title from tasks.yaml
	Title string
	// TaskIDs is the list of task IDs in this phase
	TaskIDs []string
	// CompletedCount is the number of tasks with Completed status
	CompletedCount int
	// BlockedCount is the number of tasks with Blocked status
	BlockedCount int
	// PendingCount is the number of tasks with Pending or InProgress status
	PendingCount int
}

// TaskIDsString returns task IDs as a comma-separated string.
func (p *PhaseDisplayInfo) TaskIDsString() string {
	return strings.Join(p.TaskIDs, ", ")
}

// FormatPhaseHeader formats the pre-phase summary header.
// Output format: '[Phase N/M] Title\n  -> X tasks: T001, T002, ...\n  -> Status: X completed, Y blocked, Z pending'
func FormatPhaseHeader(info PhaseDisplayInfo) string {
	var sb strings.Builder

	// Header line
	sb.WriteString(fmt.Sprintf("[Phase %d/%d] %s\n", info.PhaseNumber, info.TotalPhases, info.Title))

	// Task count and IDs
	taskCount := len(info.TaskIDs)
	if taskCount == 0 {
		sb.WriteString("  -> 0 tasks\n")
	} else {
		sb.WriteString(fmt.Sprintf("  -> %d tasks: %s\n", taskCount, info.TaskIDsString()))
	}

	// Status breakdown
	sb.WriteString(fmt.Sprintf("  -> Status: %d completed, %d blocked, %d pending",
		info.CompletedCount, info.BlockedCount, info.PendingCount))

	return sb.String()
}

// FormatPhaseCompletion formats the post-phase completion message.
// Output format: 'Phase N complete (X/Y tasks completed, Z blocked)'
// Omits blocked count if zero.
func FormatPhaseCompletion(phaseNumber int, completed int, total int, blocked int) string {
	if blocked > 0 {
		return fmt.Sprintf("Phase %d complete (%d/%d tasks completed, %d blocked)",
			phaseNumber, completed, total, blocked)
	}
	return fmt.Sprintf("Phase %d complete (%d/%d tasks completed)",
		phaseNumber, completed, total)
}

// BuildPhaseDisplayInfo creates a PhaseDisplayInfo from a PhaseInfo and task list.
// This bridges the existing PhaseInfo struct with the display-specific format.
func BuildPhaseDisplayInfo(phaseInfo PhaseInfo, totalPhases int, taskIDs []string) PhaseDisplayInfo {
	// Calculate pending as total minus completed and blocked
	pendingCount := phaseInfo.TotalTasks - phaseInfo.CompletedTasks - phaseInfo.BlockedTasks
	if pendingCount < 0 {
		pendingCount = 0
	}

	return PhaseDisplayInfo{
		PhaseNumber:    phaseInfo.Number,
		TotalPhases:    totalPhases,
		Title:          phaseInfo.Title,
		TaskIDs:        taskIDs,
		CompletedCount: phaseInfo.CompletedTasks,
		BlockedCount:   phaseInfo.BlockedTasks,
		PendingCount:   pendingCount,
	}
}
