// Package workflow provides parallel execution state persistence.
package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ParallelExecutionState persists the state of a parallel execution.
// This allows resuming after interruption or crash.
type ParallelExecutionState struct {
	SpecName      string                        `json:"spec_name"`
	StartedAt     time.Time                     `json:"started_at"`
	LastUpdated   time.Time                     `json:"last_updated"`
	CurrentWave   int                           `json:"current_wave"`
	TotalWaves    int                           `json:"total_waves"`
	TaskStatuses  map[string]string             `json:"task_statuses"`  // TaskID -> status
	FailedTasks   map[string]string             `json:"failed_tasks"`   // TaskID -> error message
	SkippedTasks  map[string]string             `json:"skipped_tasks"`  // TaskID -> reason
	WaveResults   map[int]ParallelWaveStateInfo `json:"wave_results"`   // WaveNum -> info
	WorktreePaths map[string]string             `json:"worktree_paths"` // TaskID -> path
	Interrupted   bool                          `json:"interrupted"`    // True if execution was interrupted
	CompletedAt   *time.Time                    `json:"completed_at"`   // Nil if not completed
	MaxParallel   int                           `json:"max_parallel"`   // Max concurrent tasks
	UseWorktrees  bool                          `json:"use_worktrees"`  // Whether worktrees are enabled
}

// ParallelWaveStateInfo stores wave execution summary.
type ParallelWaveStateInfo struct {
	Number      int       `json:"number"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Status      string    `json:"status"` // pending, running, completed, partial_failed
	TaskCount   int       `json:"task_count"`
	Completed   int       `json:"completed"`
	Failed      int       `json:"failed"`
	Skipped     int       `json:"skipped"`
}

// stateFileName is the file name for persisted parallel execution state.
const stateFileName = "parallel-state.json"

// LoadParallelState loads the parallel execution state for a spec.
func LoadParallelState(stateDir, specName string) (*ParallelExecutionState, error) {
	statePath := filepath.Join(stateDir, specName, stateFileName)

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No state exists
		}
		return nil, fmt.Errorf("reading parallel state: %w", err)
	}

	var state ParallelExecutionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing parallel state: %w", err)
	}

	return &state, nil
}

// SaveParallelState saves the parallel execution state for a spec.
func SaveParallelState(stateDir, specName string, state *ParallelExecutionState) error {
	stateSubDir := filepath.Join(stateDir, specName)
	if err := os.MkdirAll(stateSubDir, 0755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	state.LastUpdated = time.Now()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling parallel state: %w", err)
	}

	statePath := filepath.Join(stateSubDir, stateFileName)
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("writing parallel state: %w", err)
	}

	return nil
}

// DeleteParallelState removes the parallel execution state for a spec.
func DeleteParallelState(stateDir, specName string) error {
	statePath := filepath.Join(stateDir, specName, stateFileName)
	if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing parallel state: %w", err)
	}
	return nil
}

// NewParallelExecutionState creates a new state for starting a parallel execution.
func NewParallelExecutionState(specName string, totalWaves, maxParallel int, useWorktrees bool) *ParallelExecutionState {
	return &ParallelExecutionState{
		SpecName:      specName,
		StartedAt:     time.Now(),
		LastUpdated:   time.Now(),
		CurrentWave:   0,
		TotalWaves:    totalWaves,
		TaskStatuses:  make(map[string]string),
		FailedTasks:   make(map[string]string),
		SkippedTasks:  make(map[string]string),
		WaveResults:   make(map[int]ParallelWaveStateInfo),
		WorktreePaths: make(map[string]string),
		MaxParallel:   maxParallel,
		UseWorktrees:  useWorktrees,
	}
}

// UpdateTaskStatus updates a task's status in the state.
func (s *ParallelExecutionState) UpdateTaskStatus(taskID, status string) {
	s.TaskStatuses[taskID] = status
	s.LastUpdated = time.Now()
}

// RecordTaskFailure records a failed task.
func (s *ParallelExecutionState) RecordTaskFailure(taskID, errMsg string) {
	s.FailedTasks[taskID] = errMsg
	s.TaskStatuses[taskID] = "failed"
	s.LastUpdated = time.Now()
}

// RecordTaskSkipped records a skipped task.
func (s *ParallelExecutionState) RecordTaskSkipped(taskID, reason string) {
	s.SkippedTasks[taskID] = reason
	s.TaskStatuses[taskID] = "skipped"
	s.LastUpdated = time.Now()
}

// StartWave marks a wave as started.
func (s *ParallelExecutionState) StartWave(waveNum, taskCount int) {
	s.CurrentWave = waveNum
	s.WaveResults[waveNum] = ParallelWaveStateInfo{
		Number:    waveNum,
		StartedAt: time.Now(),
		Status:    "running",
		TaskCount: taskCount,
	}
	s.LastUpdated = time.Now()
}

// CompleteWave marks a wave as completed.
func (s *ParallelExecutionState) CompleteWave(waveNum int, completed, failed, skipped int) {
	if info, ok := s.WaveResults[waveNum]; ok {
		info.CompletedAt = time.Now()
		info.Completed = completed
		info.Failed = failed
		info.Skipped = skipped
		if failed > 0 {
			info.Status = "partial_failed"
		} else {
			info.Status = "completed"
		}
		s.WaveResults[waveNum] = info
	}
	s.LastUpdated = time.Now()
}

// MarkInterrupted marks the execution as interrupted.
func (s *ParallelExecutionState) MarkInterrupted() {
	s.Interrupted = true
	s.LastUpdated = time.Now()
}

// MarkCompleted marks the execution as completed.
func (s *ParallelExecutionState) MarkCompleted() {
	now := time.Now()
	s.CompletedAt = &now
	s.LastUpdated = now
}

// IsComplete returns true if all waves are complete.
func (s *ParallelExecutionState) IsComplete() bool {
	return s.CompletedAt != nil
}

// GetResumeWave returns the wave number to resume from.
// Returns 1 if no progress was made, or the first incomplete wave.
func (s *ParallelExecutionState) GetResumeWave() int {
	for i := 1; i <= s.TotalWaves; i++ {
		if info, ok := s.WaveResults[i]; ok {
			if info.Status == "running" || info.Status == "partial_failed" {
				return i
			}
		} else {
			return i
		}
	}
	return s.TotalWaves + 1 // All done
}

// GetPendingTasks returns task IDs that are still pending (not completed, failed, or skipped).
func (s *ParallelExecutionState) GetPendingTasks() []string {
	var pending []string
	for taskID, status := range s.TaskStatuses {
		if status != "completed" && status != "failed" && status != "skipped" {
			pending = append(pending, taskID)
		}
	}
	return pending
}

// GetCompletedTasks returns task IDs that completed successfully.
func (s *ParallelExecutionState) GetCompletedTasks() []string {
	var completed []string
	for taskID, status := range s.TaskStatuses {
		if status == "completed" {
			completed = append(completed, taskID)
		}
	}
	return completed
}
