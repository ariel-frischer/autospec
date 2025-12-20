package workflow

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/dag"
	"github.com/ariel-frischer/autospec/internal/validation"
	"github.com/ariel-frischer/autospec/internal/worktree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTaskRunner implements TaskRunner for testing.
type mockTaskRunner struct {
	runCount    atomic.Int32
	failTasks   map[string]error
	taskHistory []string
	mu          chan struct{} // Mutex channel
	delay       time.Duration // Optional delay per task
}

func newMockTaskRunner() *mockTaskRunner {
	return &mockTaskRunner{
		failTasks:   make(map[string]error),
		taskHistory: []string{},
		mu:          make(chan struct{}, 1),
	}
}

func (m *mockTaskRunner) RunTask(ctx context.Context, taskID, specName, tasksPath string) error {
	m.runCount.Add(1)

	// Record task execution
	m.mu <- struct{}{}
	m.taskHistory = append(m.taskHistory, taskID)
	<-m.mu

	if m.delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(m.delay):
		}
	}

	if err, ok := m.failTasks[taskID]; ok {
		return err
	}
	return nil
}

func (m *mockTaskRunner) FailTask(taskID string, err error) {
	m.failTasks[taskID] = err
}

func (m *mockTaskRunner) RunCount() int {
	return int(m.runCount.Load())
}

func (m *mockTaskRunner) TaskHistory() []string {
	m.mu <- struct{}{}
	defer func() { <-m.mu }()
	result := make([]string, len(m.taskHistory))
	copy(result, m.taskHistory)
	return result
}

func TestNewParallelExecutor(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
	}
	g, err := dag.BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	tests := map[string]struct {
		opts            []ParallelExecutorOption
		wantMaxParallel int
	}{
		"default options": {
			opts:            nil,
			wantMaxParallel: 4,
		},
		"custom max parallel": {
			opts:            []ParallelExecutorOption{WithMaxParallel(8)},
			wantMaxParallel: 8,
		},
		"with debug": {
			opts:            []ParallelExecutorOption{WithParallelDebug(true)},
			wantMaxParallel: 4,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			pe := NewParallelExecutor(g, tt.opts...)
			assert.NotNil(t, pe)
			assert.Equal(t, tt.wantMaxParallel, pe.MaxParallel())
			assert.NotNil(t, pe.Graph())
		})
	}
}

func TestParallelExecutor_ExecuteWaves_SingleWave(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
		{ID: "T002", Dependencies: []string{}},
		{ID: "T003", Dependencies: []string{}},
	}

	g, err := dag.BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	runner := newMockTaskRunner()
	pe := NewParallelExecutor(g, WithTaskRunner(runner), WithMaxParallel(3))

	ctx := context.Background()
	results, err := pe.ExecuteWaves(ctx, "test-spec", "tasks.yaml")

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 1, results[0].WaveNumber)
	assert.Equal(t, dag.WaveCompleted, results[0].Status)
	assert.Equal(t, 3, runner.RunCount())

	// Verify all tasks ran
	assert.Len(t, results[0].Results, 3)
	for _, taskID := range []string{"T001", "T002", "T003"} {
		result, ok := results[0].Results[taskID]
		assert.True(t, ok, "missing result for %s", taskID)
		assert.True(t, result.Success)
	}
}

func TestParallelExecutor_ExecuteWaves_MultiWave(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
		{ID: "T002", Dependencies: []string{"T001"}},
		{ID: "T003", Dependencies: []string{"T002"}},
	}

	g, err := dag.BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	runner := newMockTaskRunner()
	pe := NewParallelExecutor(g, WithTaskRunner(runner))

	ctx := context.Background()
	results, err := pe.ExecuteWaves(ctx, "test-spec", "tasks.yaml")

	require.NoError(t, err)
	assert.Len(t, results, 3)

	// Verify waves executed in order
	for i, result := range results {
		assert.Equal(t, i+1, result.WaveNumber)
		assert.Equal(t, dag.WaveCompleted, result.Status)
	}

	// Verify execution order
	history := runner.TaskHistory()
	assert.Equal(t, "T001", history[0])
	assert.Equal(t, "T002", history[1])
	assert.Equal(t, "T003", history[2])
}

func TestParallelExecutor_ExecuteWaves_MaxParallelLimit(t *testing.T) {
	t.Parallel()

	// 4 independent tasks, max 2 parallel
	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
		{ID: "T002", Dependencies: []string{}},
		{ID: "T003", Dependencies: []string{}},
		{ID: "T004", Dependencies: []string{}},
	}

	g, err := dag.BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	runner := newMockTaskRunner()
	runner.delay = 10 * time.Millisecond

	pe := NewParallelExecutor(g, WithTaskRunner(runner), WithMaxParallel(2))

	ctx := context.Background()
	results, err := pe.ExecuteWaves(ctx, "test-spec", "tasks.yaml")

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 4, runner.RunCount())
}

func TestParallelExecutor_ExecuteWaves_FailureHandling(t *testing.T) {
	t.Parallel()

	// T001, T002 independent; T003 depends on T001
	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
		{ID: "T002", Dependencies: []string{}},
		{ID: "T003", Dependencies: []string{"T001"}},
	}

	g, err := dag.BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	runner := newMockTaskRunner()
	runner.FailTask("T001", errors.New("task failed"))

	pe := NewParallelExecutor(g, WithTaskRunner(runner))

	ctx := context.Background()
	results, err := pe.ExecuteWaves(ctx, "test-spec", "tasks.yaml")

	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Wave 1: T001 fails, T002 succeeds
	wave1 := results[0]
	assert.Equal(t, dag.WavePartialFailed, wave1.Status)
	assert.False(t, wave1.Results["T001"].Success)
	assert.True(t, wave1.Results["T002"].Success)

	// Wave 2: T003 skipped due to T001 failure
	wave2 := results[1]
	assert.True(t, wave2.Results["T003"].Skipped)
	assert.Contains(t, wave2.Results["T003"].SkipReason, "T001")

	// Verify failed and skipped tracking
	assert.Len(t, pe.FailedTasks(), 1)
	assert.Contains(t, pe.FailedTasks(), "T001")
	assert.Len(t, pe.SkippedTasks(), 1)
	assert.Contains(t, pe.SkippedTasks(), "T003")
}

func TestParallelExecutor_ExecuteWaves_SiblingsContinue(t *testing.T) {
	t.Parallel()

	// T001, T002, T003 all independent in wave 1
	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
		{ID: "T002", Dependencies: []string{}},
		{ID: "T003", Dependencies: []string{}},
	}

	g, err := dag.BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	runner := newMockTaskRunner()
	runner.FailTask("T002", errors.New("T002 failed"))

	pe := NewParallelExecutor(g, WithTaskRunner(runner))

	ctx := context.Background()
	results, err := pe.ExecuteWaves(ctx, "test-spec", "tasks.yaml")

	require.NoError(t, err)
	assert.Len(t, results, 1)

	// All three tasks should have run
	assert.Equal(t, 3, runner.RunCount())

	// T001 and T003 succeed, T002 fails
	assert.True(t, results[0].Results["T001"].Success)
	assert.False(t, results[0].Results["T002"].Success)
	assert.True(t, results[0].Results["T003"].Success)
}

func TestParallelExecutor_ExecuteWaves_ContextCancellation(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
		{ID: "T002", Dependencies: []string{"T001"}},
	}

	g, err := dag.BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	runner := newMockTaskRunner()
	pe := NewParallelExecutor(g, WithTaskRunner(runner))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	results, err := pe.ExecuteWaves(ctx, "test-spec", "tasks.yaml")

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, results)
}

func TestParallelExecutor_DryRun(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
		{ID: "T002", Dependencies: []string{"T001"}},
		{ID: "T003", Dependencies: []string{"T001"}},
		{ID: "T004", Dependencies: []string{"T002", "T003"}},
	}

	g, err := dag.BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	pe := NewParallelExecutor(g)
	waves := pe.DryRun()

	assert.Len(t, waves, 3)
	assert.ElementsMatch(t, []string{"T001"}, waves[0].TaskIDs)
	assert.ElementsMatch(t, []string{"T002", "T003"}, waves[1].TaskIDs)
	assert.ElementsMatch(t, []string{"T004"}, waves[2].TaskIDs)
}

func TestParallelExecutor_GetWaveStats(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
		{ID: "T002", Dependencies: []string{"T001"}},
		{ID: "T003", Dependencies: []string{"T001"}},
		{ID: "T004", Dependencies: []string{"T002", "T003"}},
	}

	g, err := dag.BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	pe := NewParallelExecutor(g)
	stats := pe.GetWaveStats()

	assert.Equal(t, 3, stats.TotalWaves)
	assert.Equal(t, 4, stats.TotalTasks)
	assert.Equal(t, 2, stats.MaxWaveSize)
	assert.Equal(t, 1, stats.MinWaveSize)
}

func TestParallelExecutor_NoTaskRunner(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
	}

	g, err := dag.BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	// No task runner configured
	pe := NewParallelExecutor(g)

	ctx := context.Background()
	results, err := pe.ExecuteWaves(ctx, "test-spec", "tasks.yaml")

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.False(t, results[0].Results["T001"].Success)
	assert.Contains(t, results[0].Results["T001"].Error.Error(), "no task runner")
}

func TestParallelExecutor_CascadeSkip(t *testing.T) {
	t.Parallel()

	// T001 -> T002 -> T003 -> T004
	// If T001 fails, all downstream tasks should be skipped
	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
		{ID: "T002", Dependencies: []string{"T001"}},
		{ID: "T003", Dependencies: []string{"T002"}},
		{ID: "T004", Dependencies: []string{"T003"}},
	}

	g, err := dag.BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	runner := newMockTaskRunner()
	runner.FailTask("T001", errors.New("root task failed"))

	pe := NewParallelExecutor(g, WithTaskRunner(runner))

	ctx := context.Background()
	results, err := pe.ExecuteWaves(ctx, "test-spec", "tasks.yaml")

	require.NoError(t, err)
	assert.Len(t, results, 4)

	// T001 failed
	assert.False(t, results[0].Results["T001"].Success)

	// T002, T003, T004 all skipped
	assert.True(t, results[1].Results["T002"].Skipped)
	assert.True(t, results[2].Results["T003"].Skipped)
	assert.True(t, results[3].Results["T004"].Skipped)

	// Only T001 actually ran
	assert.Equal(t, 1, runner.RunCount())
}

func TestParallelExecutor_EmptyGraph(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{}
	g, err := dag.BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	pe := NewParallelExecutor(g)

	ctx := context.Background()
	results, err := pe.ExecuteWaves(ctx, "test-spec", "tasks.yaml")

	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestParallelExecutor_UseWorktrees(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
	}

	g, err := dag.BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	tests := map[string]struct {
		opts          []ParallelExecutorOption
		wantWorktrees bool
	}{
		"without worktree manager": {
			opts:          nil,
			wantWorktrees: false,
		},
		"with worktree manager": {
			opts:          []ParallelExecutorOption{WithWorktreeManager(&mockWorktreeManager{})},
			wantWorktrees: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			pe := NewParallelExecutor(g, tt.opts...)
			assert.Equal(t, tt.wantWorktrees, pe.UseWorktrees())
		})
	}
}

// mockWorktreeManager implements worktree.Manager for testing.
type mockWorktreeManager struct {
	createCalls []string
	removeCalls []string
	statusCalls []string
	failCreate  bool
	failRemove  bool
	failStatus  bool
}

func (m *mockWorktreeManager) Create(name, branch, customPath string) (*worktree.Worktree, error) {
	m.createCalls = append(m.createCalls, name)
	if m.failCreate {
		return nil, errors.New("mock create error")
	}
	return &worktree.Worktree{Name: name, Path: customPath, Branch: branch}, nil
}

func (m *mockWorktreeManager) List() ([]worktree.Worktree, error) {
	return nil, nil
}

func (m *mockWorktreeManager) Get(name string) (*worktree.Worktree, error) {
	return nil, nil
}

func (m *mockWorktreeManager) Remove(name string, force bool) error {
	m.removeCalls = append(m.removeCalls, name)
	if m.failRemove {
		return errors.New("mock remove error")
	}
	return nil
}

func (m *mockWorktreeManager) Setup(path string, addToState bool) (*worktree.Worktree, error) {
	return nil, nil
}

func (m *mockWorktreeManager) Prune() (int, error) {
	return 0, nil
}

func (m *mockWorktreeManager) UpdateStatus(name string, status worktree.WorktreeStatus) error {
	m.statusCalls = append(m.statusCalls, name)
	if m.failStatus {
		return errors.New("mock status error")
	}
	return nil
}

func TestParallelExecutor_WithWorktreeManager_CreatesWorktrees(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
		{ID: "T002", Dependencies: []string{}},
	}

	g, err := dag.BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	runner := newMockTaskRunner()
	wm := &mockWorktreeManager{}

	pe := NewParallelExecutor(g,
		WithTaskRunner(runner),
		WithWorktreeManager(wm),
		WithRepoRoot("/tmp/test-repo"),
	)

	ctx := context.Background()
	results, err := pe.ExecuteWaves(ctx, "test-spec", "tasks.yaml")

	require.NoError(t, err)
	assert.Len(t, results, 1)

	// Verify worktrees were created for each task
	assert.Len(t, wm.createCalls, 2)
	assert.Contains(t, wm.createCalls, "T001")
	assert.Contains(t, wm.createCalls, "T002")
}

func TestParallelExecutor_WorktreeCreationFailure(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
	}

	g, err := dag.BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	runner := newMockTaskRunner()
	wm := &mockWorktreeManager{failCreate: true}

	pe := NewParallelExecutor(g,
		WithTaskRunner(runner),
		WithWorktreeManager(wm),
		WithRepoRoot("/tmp/test-repo"),
	)

	ctx := context.Background()
	results, err := pe.ExecuteWaves(ctx, "test-spec", "tasks.yaml")

	require.NoError(t, err) // Execution continues, but task fails
	assert.Len(t, results, 1)

	// Task should have failed due to worktree creation failure
	assert.False(t, results[0].Results["T001"].Success)
	assert.Contains(t, results[0].Results["T001"].Error.Error(), "creating worktree")

	// Task should not have run
	assert.Equal(t, 0, runner.RunCount())
}

func TestParallelExecutor_MergeWaveWorktrees(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
		{ID: "T002", Dependencies: []string{}},
	}

	g, err := dag.BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	wm := &mockWorktreeManager{}
	pe := NewParallelExecutor(g, WithWorktreeManager(wm))

	// Simulate completed wave
	waveResult := &WaveResult{
		WaveNumber: 1,
		Results: map[string]*ParallelTaskResult{
			"T001": {TaskID: "T001", Success: true},
			"T002": {TaskID: "T002", Success: true},
		},
	}

	// Store worktree paths manually (simulating createWorktree having run)
	pe.worktreePaths["T001"] = "/tmp/worktree-T001"
	pe.worktreePaths["T002"] = "/tmp/worktree-T002"

	err = pe.MergeWaveWorktrees(waveResult)
	require.NoError(t, err)

	// Verify status was updated for both tasks
	assert.Len(t, wm.statusCalls, 2)
	assert.Contains(t, wm.statusCalls, "T001")
	assert.Contains(t, wm.statusCalls, "T002")
}

func TestParallelExecutor_CleanupWaveWorktrees(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
	}

	g, err := dag.BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	wm := &mockWorktreeManager{}
	pe := NewParallelExecutor(g, WithWorktreeManager(wm))

	// Store worktree path
	pe.worktreePaths["T001"] = "/tmp/worktree-T001"

	waveResult := &WaveResult{
		WaveNumber: 1,
		Results: map[string]*ParallelTaskResult{
			"T001": {TaskID: "T001", Success: true},
		},
	}

	err = pe.CleanupWaveWorktrees(waveResult)
	require.NoError(t, err)

	// Verify worktree was removed
	assert.Len(t, wm.removeCalls, 1)
	assert.Contains(t, wm.removeCalls, "T001")

	// Verify path was cleared from map
	assert.NotContains(t, pe.worktreePaths, "T001")
}

func TestParallelExecutor_SkipsCleanupForFailedTasks(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
		{ID: "T002", Dependencies: []string{}},
	}

	g, err := dag.BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	wm := &mockWorktreeManager{}
	pe := NewParallelExecutor(g, WithWorktreeManager(wm))

	// Store worktree paths
	pe.worktreePaths["T001"] = "/tmp/worktree-T001"
	pe.worktreePaths["T002"] = "/tmp/worktree-T002"

	waveResult := &WaveResult{
		WaveNumber: 1,
		Results: map[string]*ParallelTaskResult{
			"T001": {TaskID: "T001", Success: true},
			"T002": {TaskID: "T002", Success: false, Error: errors.New("failed")},
		},
	}

	err = pe.CleanupWaveWorktrees(waveResult)
	require.NoError(t, err)

	// Only successful task should have cleanup called
	assert.Len(t, wm.removeCalls, 1)
	assert.Contains(t, wm.removeCalls, "T001")

	// Failed task should still have worktree path
	assert.Contains(t, pe.worktreePaths, "T002")
}

func TestParallelExecutor_WithWorktreeDir(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
	}

	g, err := dag.BuildFromTasks(tasks)
	require.NoError(t, err)

	pe := NewParallelExecutor(g,
		WithWorktreeDir(".custom-worktrees"),
		WithRepoRoot("/tmp/test"),
	)

	assert.Equal(t, "/tmp/test/.custom-worktrees", pe.getWorktreeDir())
}
