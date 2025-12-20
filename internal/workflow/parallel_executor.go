// Package workflow provides parallel task execution functionality.
// ParallelExecutor handles concurrent task execution across waves.
package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ariel-frischer/autospec/internal/dag"
	"github.com/ariel-frischer/autospec/internal/worktree"
	"golang.org/x/sync/errgroup"
)

// ProgressCallback is called when task status changes.
type ProgressCallback func(waveNum int, taskID string, status dag.TaskStatus, progressLine string)

// ParallelExecutor orchestrates concurrent task execution across waves.
type ParallelExecutor struct {
	maxParallel     int                  // Maximum concurrent Claude sessions
	graph           *dag.DependencyGraph // Task dependency graph
	worktreeManager worktree.Manager     // Git worktree manager for isolation (optional)
	dagRoot         string               // Branch name where worktree changes merge
	worktreeDir     string               // Directory for worktrees (default: .worktrees)
	repoRoot        string               // Repository root path
	failedTasks     map[string]error     // Tasks that failed execution
	skippedTasks    map[string]string    // Tasks skipped due to failed dependencies
	worktreePaths   map[string]string    // TaskID -> worktree path mapping
	progressCb      ProgressCallback     // Callback for progress updates
	mu              sync.Mutex           // Protects failedTasks, skippedTasks, worktreePaths

	// Dependencies injected for testing
	taskRunner TaskRunner // Interface for running individual tasks
	debug      bool       // Enable debug logging
}

// TaskRunner defines the interface for executing individual tasks.
// This abstraction allows for testing without spawning real Claude sessions.
type TaskRunner interface {
	RunTask(ctx context.Context, taskID, specName, tasksPath string) error
}

// ParallelTaskResult represents the outcome of a single task execution.
type ParallelTaskResult struct {
	TaskID       string        // Task identifier
	Success      bool          // Whether task completed successfully
	Error        error         // Error message if failed
	Duration     time.Duration // Execution time
	WorktreePath string        // Path to worktree if used (empty otherwise)
	Skipped      bool          // True if task was skipped due to failed dependency
	SkipReason   string        // Reason for skipping
}

// WaveResult represents the outcome of executing a wave.
type WaveResult struct {
	WaveNumber int                            // Wave number (1-indexed)
	Results    map[string]*ParallelTaskResult // Task ID to result mapping
	Status     dag.WaveStatus                 // Final wave status
	Duration   time.Duration                  // Total wave execution time
}

// ParallelExecutorOption is a functional option for configuring ParallelExecutor.
type ParallelExecutorOption func(*ParallelExecutor)

// WithMaxParallel sets the maximum number of concurrent tasks.
func WithMaxParallel(n int) ParallelExecutorOption {
	return func(pe *ParallelExecutor) {
		pe.maxParallel = n
	}
}

// WithWorktreeManager enables worktree isolation.
func WithWorktreeManager(wm worktree.Manager) ParallelExecutorOption {
	return func(pe *ParallelExecutor) {
		pe.worktreeManager = wm
	}
}

// WithDAGRoot sets the branch name for worktree merges.
func WithDAGRoot(branch string) ParallelExecutorOption {
	return func(pe *ParallelExecutor) {
		pe.dagRoot = branch
	}
}

// WithTaskRunner sets the task runner implementation.
func WithTaskRunner(tr TaskRunner) ParallelExecutorOption {
	return func(pe *ParallelExecutor) {
		pe.taskRunner = tr
	}
}

// WithParallelDebug enables debug logging.
func WithParallelDebug(debug bool) ParallelExecutorOption {
	return func(pe *ParallelExecutor) {
		pe.debug = debug
	}
}

// WithRepoRoot sets the repository root path.
func WithRepoRoot(repoRoot string) ParallelExecutorOption {
	return func(pe *ParallelExecutor) {
		pe.repoRoot = repoRoot
	}
}

// WithWorktreeDir sets the worktree directory name.
func WithWorktreeDir(dir string) ParallelExecutorOption {
	return func(pe *ParallelExecutor) {
		pe.worktreeDir = dir
	}
}

// WithProgressCallback sets the progress callback function.
func WithProgressCallback(cb ProgressCallback) ParallelExecutorOption {
	return func(pe *ParallelExecutor) {
		pe.progressCb = cb
	}
}

// NewParallelExecutor creates a new ParallelExecutor with the given options.
func NewParallelExecutor(graph *dag.DependencyGraph, opts ...ParallelExecutorOption) *ParallelExecutor {
	pe := &ParallelExecutor{
		maxParallel:   4, // Default
		graph:         graph,
		worktreeDir:   ".worktrees",
		failedTasks:   make(map[string]error),
		skippedTasks:  make(map[string]string),
		worktreePaths: make(map[string]string),
	}

	for _, opt := range opts {
		opt(pe)
	}

	return pe
}

// ExecuteWaves executes all waves in order, running tasks within each wave concurrently.
// Returns results for all waves and any error that occurred.
func (pe *ParallelExecutor) ExecuteWaves(ctx context.Context, specName, tasksPath string) ([]WaveResult, error) {
	waves := pe.graph.Waves()
	if len(waves) == 0 {
		return nil, nil
	}

	results := make([]WaveResult, 0, len(waves))

	for _, wave := range waves {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		waveResult, err := pe.executeWave(ctx, wave, specName, tasksPath)
		results = append(results, *waveResult)

		if err != nil {
			return results, fmt.Errorf("executing wave %d: %w", wave.Number, err)
		}
	}

	return results, nil
}

// executeWave executes all tasks in a single wave concurrently.
func (pe *ParallelExecutor) executeWave(ctx context.Context, wave dag.ExecutionWave, specName, tasksPath string) (*WaveResult, error) {
	startTime := time.Now()
	result := &WaveResult{
		WaveNumber: wave.Number,
		Results:    make(map[string]*ParallelTaskResult),
		Status:     dag.WaveRunning,
	}

	// Filter out tasks to skip
	tasksToRun, skipped := pe.filterTasksToRun(wave.TaskIDs)
	for taskID, reason := range skipped {
		result.Results[taskID] = &ParallelTaskResult{
			TaskID:     taskID,
			Success:    false,
			Skipped:    true,
			SkipReason: reason,
		}
	}

	if len(tasksToRun) == 0 {
		result.Status = dag.WaveCompleted
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Execute tasks concurrently with semaphore limiting
	taskResults := pe.runTasksConcurrently(ctx, tasksToRun, specName, tasksPath)

	// Collect results
	allSuccess := true
	for _, tr := range taskResults {
		result.Results[tr.TaskID] = tr
		if !tr.Success && !tr.Skipped {
			allSuccess = false
			pe.recordFailedTask(tr.TaskID, tr.Error)
		}
	}

	if allSuccess {
		result.Status = dag.WaveCompleted
	} else {
		result.Status = dag.WavePartialFailed
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// filterTasksToRun checks dependencies and returns tasks that can run.
func (pe *ParallelExecutor) filterTasksToRun(taskIDs []string) ([]string, map[string]string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	toRun := make([]string, 0, len(taskIDs))
	skipped := make(map[string]string)

	for _, taskID := range taskIDs {
		node := pe.graph.GetNode(taskID)
		if node == nil {
			continue
		}

		// Check if any dependency failed
		failedDep := ""
		for _, depID := range node.Dependencies {
			if _, failed := pe.failedTasks[depID]; failed {
				failedDep = depID
				break
			}
			if _, skippedDep := pe.skippedTasks[depID]; skippedDep {
				failedDep = depID
				break
			}
		}

		if failedDep != "" {
			reason := fmt.Sprintf("Skipped: dependency %s failed", failedDep)
			skipped[taskID] = reason
			pe.skippedTasks[taskID] = reason
		} else {
			toRun = append(toRun, taskID)
		}
	}

	return toRun, skipped
}

// runTasksConcurrently runs tasks with semaphore-limited concurrency.
func (pe *ParallelExecutor) runTasksConcurrently(ctx context.Context, taskIDs []string, specName, tasksPath string) []*ParallelTaskResult {
	results := make([]*ParallelTaskResult, 0, len(taskIDs))
	resultsChan := make(chan *ParallelTaskResult, len(taskIDs))

	// Create errgroup with limited concurrency
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(pe.maxParallel)

	for _, taskID := range taskIDs {
		taskID := taskID // Capture for goroutine
		g.Go(func() error {
			result := pe.executeTask(ctx, taskID, specName, tasksPath)
			resultsChan <- result
			return nil // Don't propagate errors to allow other tasks to continue
		})
	}

	// Wait for all tasks and close channel
	go func() {
		_ = g.Wait()
		close(resultsChan)
	}()

	// Collect results
	for result := range resultsChan {
		results = append(results, result)
	}

	return results
}

// executeTask runs a single task and returns the result.
func (pe *ParallelExecutor) executeTask(ctx context.Context, taskID, specName, tasksPath string) *ParallelTaskResult {
	startTime := time.Now()
	result := &ParallelTaskResult{
		TaskID: taskID,
	}

	// Find current wave for progress reporting
	waveNum := pe.graph.GetWaveForTask(taskID)

	// Update graph status and report progress
	_ = pe.graph.SetNodeStatus(taskID, dag.StatusRunning)
	pe.reportProgress(waveNum, taskID, dag.StatusRunning)

	if pe.taskRunner == nil {
		result.Error = fmt.Errorf("no task runner configured")
		result.Success = false
		_ = pe.graph.SetNodeStatus(taskID, dag.StatusFailed)
		pe.reportProgress(waveNum, taskID, dag.StatusFailed)
		result.Duration = time.Since(startTime)
		return result
	}

	// Create worktree if enabled
	worktreePath, err := pe.createWorktree(taskID)
	if err != nil {
		result.Error = err
		result.Success = false
		_ = pe.graph.SetNodeStatus(taskID, dag.StatusFailed)
		pe.reportProgress(waveNum, taskID, dag.StatusFailed)
		result.Duration = time.Since(startTime)
		return result
	}
	result.WorktreePath = worktreePath

	// Execute the task
	err = pe.taskRunner.RunTask(ctx, taskID, specName, tasksPath)
	result.Duration = time.Since(startTime)

	if err != nil {
		result.Error = err
		result.Success = false
		_ = pe.graph.SetNodeStatus(taskID, dag.StatusFailed)
		pe.reportProgress(waveNum, taskID, dag.StatusFailed)
	} else {
		result.Success = true
		_ = pe.graph.SetNodeStatus(taskID, dag.StatusCompleted)
		pe.reportProgress(waveNum, taskID, dag.StatusCompleted)
	}

	return result
}

// reportProgress calls the progress callback if set.
func (pe *ParallelExecutor) reportProgress(waveNum int, taskID string, status dag.TaskStatus) {
	if pe.progressCb == nil {
		return
	}
	progressLine := pe.graph.RenderProgress(waveNum)
	pe.progressCb(waveNum, taskID, status, progressLine)
}

// recordFailedTask records a task as failed for dependency checking.
func (pe *ParallelExecutor) recordFailedTask(taskID string, err error) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.failedTasks[taskID] = err
}

// FailedTasks returns a copy of the failed tasks map.
func (pe *ParallelExecutor) FailedTasks() map[string]error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	result := make(map[string]error, len(pe.failedTasks))
	for k, v := range pe.failedTasks {
		result[k] = v
	}
	return result
}

// SkippedTasks returns a copy of the skipped tasks map.
func (pe *ParallelExecutor) SkippedTasks() map[string]string {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	result := make(map[string]string, len(pe.skippedTasks))
	for k, v := range pe.skippedTasks {
		result[k] = v
	}
	return result
}

// DryRun outputs the execution plan without running any tasks.
func (pe *ParallelExecutor) DryRun() []dag.ExecutionWave {
	return pe.graph.Waves()
}

// GetWaveStats returns statistics about the execution waves.
func (pe *ParallelExecutor) GetWaveStats() dag.WaveStats {
	return pe.graph.GetWaveStats()
}

// MaxParallel returns the configured maximum parallel tasks.
func (pe *ParallelExecutor) MaxParallel() int {
	return pe.maxParallel
}

// Graph returns the underlying dependency graph.
func (pe *ParallelExecutor) Graph() *dag.DependencyGraph {
	return pe.graph
}

// UseWorktrees returns true if worktree isolation is enabled.
func (pe *ParallelExecutor) UseWorktrees() bool {
	return pe.worktreeManager != nil
}

// createWorktree creates a worktree for a task if worktree mode is enabled.
// Returns the worktree path (or empty string if not using worktrees).
func (pe *ParallelExecutor) createWorktree(taskID string) (string, error) {
	if pe.worktreeManager == nil {
		return "", nil
	}

	// Create worktree path: .worktrees/<task-id>/
	worktreePath := filepath.Join(pe.repoRoot, pe.worktreeDir, taskID)

	// Create the worktree using the manager
	// Branch name is the task ID for easy identification
	wt, err := pe.worktreeManager.Create(taskID, taskID, worktreePath)
	if err != nil {
		return "", fmt.Errorf("creating worktree for task %s: %w", taskID, err)
	}

	pe.mu.Lock()
	pe.worktreePaths[taskID] = wt.Path
	pe.mu.Unlock()

	return wt.Path, nil
}

// mergeWorktree merges changes from a task's worktree into DAG-ROOT.
func (pe *ParallelExecutor) mergeWorktree(taskID string) error {
	if pe.worktreeManager == nil {
		return nil
	}

	pe.mu.Lock()
	wtPath, exists := pe.worktreePaths[taskID]
	pe.mu.Unlock()

	if !exists {
		return nil // No worktree for this task
	}

	// Update worktree status to merged
	if err := pe.worktreeManager.UpdateStatus(taskID, worktree.StatusMerged); err != nil {
		return fmt.Errorf("updating worktree status: %w", err)
	}

	// Note: Actual git merge is handled separately as it may require
	// user intervention for conflicts. The worktree package handles this.
	_ = wtPath // Used for potential merge operations

	return nil
}

// cleanupWorktree removes a task's worktree after successful merge.
func (pe *ParallelExecutor) cleanupWorktree(taskID string) error {
	if pe.worktreeManager == nil {
		return nil
	}

	pe.mu.Lock()
	_, exists := pe.worktreePaths[taskID]
	pe.mu.Unlock()

	if !exists {
		return nil
	}

	// Remove the worktree (force=false to preserve uncommitted work)
	if err := pe.worktreeManager.Remove(taskID, false); err != nil {
		return fmt.Errorf("removing worktree for task %s: %w", taskID, err)
	}

	pe.mu.Lock()
	delete(pe.worktreePaths, taskID)
	pe.mu.Unlock()

	return nil
}

// getWorktreeDir returns the worktree directory for a task, creating it if needed.
func (pe *ParallelExecutor) getWorktreeDir() string {
	if pe.repoRoot == "" {
		if wd, err := os.Getwd(); err == nil {
			return filepath.Join(wd, pe.worktreeDir)
		}
		return pe.worktreeDir
	}
	return filepath.Join(pe.repoRoot, pe.worktreeDir)
}

// MergeWaveWorktrees sequentially merges all worktrees from a completed wave.
// Returns an error if any merge fails (typically due to conflicts).
func (pe *ParallelExecutor) MergeWaveWorktrees(waveResult *WaveResult) error {
	if pe.worktreeManager == nil {
		return nil
	}

	for taskID, result := range waveResult.Results {
		// Only merge successful tasks
		if !result.Success || result.Skipped {
			continue
		}

		if err := pe.mergeWorktree(taskID); err != nil {
			return fmt.Errorf("merging worktree for task %s: %w", taskID, err)
		}
	}

	return nil
}

// CleanupWaveWorktrees removes all worktrees from a merged wave.
func (pe *ParallelExecutor) CleanupWaveWorktrees(waveResult *WaveResult) error {
	if pe.worktreeManager == nil {
		return nil
	}

	var lastErr error
	for taskID, result := range waveResult.Results {
		// Only cleanup successful merged tasks
		if !result.Success || result.Skipped {
			continue
		}

		if err := pe.cleanupWorktree(taskID); err != nil {
			lastErr = err // Continue cleanup, but track error
		}
	}

	return lastErr
}
