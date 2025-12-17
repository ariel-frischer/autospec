package workflow

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/progress"
	"github.com/ariel-frischer/autospec/internal/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClaudeExecutor implements a mock for testing
type mockClaudeExecutor struct {
	executeErr   error
	executeCalls []string
}

func (m *mockClaudeExecutor) Execute(prompt string) error {
	m.executeCalls = append(m.executeCalls, prompt)
	return m.executeErr
}

func (m *mockClaudeExecutor) FormatCommand(prompt string) string {
	return "claude " + prompt
}

func TestGetStageNumber(t *testing.T) {
	tests := map[string]struct {
		stage Stage
		want  int
	}{
		"constitution stage": {stage: StageConstitution, want: 1},
		"specify stage":      {stage: StageSpecify, want: 2},
		"clarify stage":      {stage: StageClarify, want: 3},
		"plan stage":         {stage: StagePlan, want: 4},
		"tasks stage":        {stage: StageTasks, want: 5},
		"checklist stage":    {stage: StageChecklist, want: 6},
		"analyze stage":      {stage: StageAnalyze, want: 7},
		"implement stage":    {stage: StageImplement, want: 8},
		"unknown stage":      {stage: Stage("unknown"), want: 0},
		"empty stage":        {stage: Stage(""), want: 0},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			executor := &Executor{}
			got := executor.getStageNumber(tc.stage)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBuildStageInfo(t *testing.T) {
	tests := map[string]struct {
		stage       Stage
		retryCount  int
		maxRetries  int
		totalStages int
		wantName    string
		wantNumber  int
	}{
		"specify stage no retries": {
			stage:       StageSpecify,
			retryCount:  0,
			maxRetries:  3,
			totalStages: 4,
			wantName:    "specify",
			wantNumber:  2,
		},
		"plan stage with retries": {
			stage:       StagePlan,
			retryCount:  2,
			maxRetries:  3,
			totalStages: 4,
			wantName:    "plan",
			wantNumber:  4,
		},
		"implement stage max retries": {
			stage:       StageImplement,
			retryCount:  3,
			maxRetries:  3,
			totalStages: 8,
			wantName:    "implement",
			wantNumber:  8,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			executor := &Executor{
				MaxRetries:  tc.maxRetries,
				TotalStages: tc.totalStages,
			}

			info := executor.buildStageInfo(tc.stage, tc.retryCount)

			assert.Equal(t, tc.wantName, info.Name)
			assert.Equal(t, tc.wantNumber, info.Number)
			assert.Equal(t, tc.totalStages, info.TotalStages)
			assert.Equal(t, tc.retryCount, info.RetryCount)
			assert.Equal(t, tc.maxRetries, info.MaxRetries)
		})
	}
}

func TestExecuteStage_Success(t *testing.T) {
	stateDir := t.TempDir()
	specsDir := t.TempDir()

	// Create spec directory with required file
	specDir := filepath.Join(specsDir, "001-test")
	require.NoError(t, os.MkdirAll(specDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.md"), []byte("# Test Spec"), 0644))

	executor := &Executor{
		Claude: &ClaudeExecutor{
			ClaudeCmd:  "echo",
			ClaudeArgs: []string{},
		},
		StateDir:   stateDir,
		SpecsDir:   specsDir,
		MaxRetries: 3,
	}

	// Validation function that always succeeds
	validateFunc := func(dir string) error {
		return nil
	}

	result, err := executor.ExecuteStage("001-test", StageSpecify, "/test.command", validateFunc)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, StageSpecify, result.Stage)
	assert.Equal(t, 0, result.RetryCount)
	assert.False(t, result.Exhausted)
}

func TestExecuteStage_ValidationFailure(t *testing.T) {
	stateDir := t.TempDir()
	specsDir := t.TempDir()

	executor := &Executor{
		Claude: &ClaudeExecutor{
			ClaudeCmd:  "echo",
			ClaudeArgs: []string{},
		},
		StateDir:   stateDir,
		SpecsDir:   specsDir,
		MaxRetries: 3,
	}

	// Validation function that always fails
	validateFunc := func(dir string) error {
		return errors.New("validation failed: missing spec.md")
	}

	result, err := executor.ExecuteStage("001-test", StageSpecify, "/test.command", validateFunc)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
	assert.False(t, result.Success)
	assert.Equal(t, 1, result.RetryCount) // Should have incremented
}

func TestExecuteStage_RetryExhausted(t *testing.T) {
	stateDir := t.TempDir()
	specsDir := t.TempDir()

	// Pre-set retry count to max so next failure returns exhausted error
	state := &retry.RetryState{
		SpecName:   "001-test",
		Phase:      "specify",
		Count:      3,
		MaxRetries: 3,
	}
	require.NoError(t, retry.SaveRetryState(stateDir, state))

	executor := &Executor{
		Claude: &ClaudeExecutor{
			ClaudeCmd:  "echo",
			ClaudeArgs: []string{},
		},
		StateDir:   stateDir,
		SpecsDir:   specsDir,
		MaxRetries: 3,
	}

	// Validation function that always fails
	validateFunc := func(dir string) error {
		return errors.New("validation failed")
	}

	result, err := executor.ExecuteStage("001-test", StageSpecify, "/test.command", validateFunc)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "exhausted")
	assert.False(t, result.Success)
	assert.True(t, result.Exhausted)
}

func TestExecuteStage_ResetsRetryOnSuccess(t *testing.T) {
	stateDir := t.TempDir()
	specsDir := t.TempDir()

	// Pre-set retry count
	state := &retry.RetryState{
		SpecName:   "001-test",
		Phase:      "specify",
		Count:      2,
		MaxRetries: 3,
	}
	require.NoError(t, retry.SaveRetryState(stateDir, state))

	executor := &Executor{
		Claude: &ClaudeExecutor{
			ClaudeCmd:  "echo",
			ClaudeArgs: []string{},
		},
		StateDir:   stateDir,
		SpecsDir:   specsDir,
		MaxRetries: 3,
	}

	// Validation function that succeeds
	validateFunc := func(dir string) error {
		return nil
	}

	result, err := executor.ExecuteStage("001-test", StageSpecify, "/test.command", validateFunc)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, 0, result.RetryCount)

	// Verify retry state was reset
	loaded, err := retry.LoadRetryState(stateDir, "001-test", "specify", 3)
	require.NoError(t, err)
	assert.Equal(t, 0, loaded.Count)
}

func TestExecuteWithRetry_Success(t *testing.T) {
	executor := &Executor{
		Claude: &ClaudeExecutor{
			ClaudeCmd:  "echo",
			ClaudeArgs: []string{"success"},
		},
	}

	err := executor.ExecuteWithRetry("/test.command", 3)
	assert.NoError(t, err)
}

func TestExecuteWithRetry_AllAttemptsFail(t *testing.T) {
	executor := &Executor{
		Claude: &ClaudeExecutor{
			ClaudeCmd:  "false", // Command that always fails
			ClaudeArgs: []string{},
		},
	}

	err := executor.ExecuteWithRetry("/test.command", 2)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "all 2 attempts failed")
}

func TestGetRetryState(t *testing.T) {
	stateDir := t.TempDir()

	// Save initial state
	state := &retry.RetryState{
		SpecName:   "001-test",
		Phase:      "plan",
		Count:      1,
		MaxRetries: 3,
	}
	require.NoError(t, retry.SaveRetryState(stateDir, state))

	executor := &Executor{
		StateDir:   stateDir,
		MaxRetries: 3,
	}

	loaded, err := executor.GetRetryState("001-test", StagePlan)

	require.NoError(t, err)
	assert.Equal(t, 1, loaded.Count)
	assert.Equal(t, "001-test", loaded.SpecName)
	assert.Equal(t, "plan", loaded.Phase)
}

func TestResetStage(t *testing.T) {
	stateDir := t.TempDir()

	// Save initial state with non-zero count
	state := &retry.RetryState{
		SpecName:   "001-test",
		Phase:      "tasks",
		Count:      2,
		MaxRetries: 3,
	}
	require.NoError(t, retry.SaveRetryState(stateDir, state))

	executor := &Executor{
		StateDir:   stateDir,
		MaxRetries: 3,
	}

	err := executor.ResetStage("001-test", StageTasks)
	require.NoError(t, err)

	// Verify reset
	loaded, err := retry.LoadRetryState(stateDir, "001-test", "tasks", 3)
	require.NoError(t, err)
	assert.Equal(t, 0, loaded.Count)
}

func TestValidateSpec(t *testing.T) {
	t.Run("spec exists", func(t *testing.T) {
		t.Parallel()
		specDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.md"), []byte("# Spec"), 0644))

		executor := &Executor{}
		err := executor.ValidateSpec(specDir)
		assert.NoError(t, err)
	})

	t.Run("spec missing", func(t *testing.T) {
		t.Parallel()
		specDir := t.TempDir()

		executor := &Executor{}
		err := executor.ValidateSpec(specDir)
		assert.Error(t, err)
	})
}

func TestValidatePlan(t *testing.T) {
	t.Run("plan exists", func(t *testing.T) {
		t.Parallel()
		specDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "plan.md"), []byte("# Plan"), 0644))

		executor := &Executor{}
		err := executor.ValidatePlan(specDir)
		assert.NoError(t, err)
	})

	t.Run("plan missing", func(t *testing.T) {
		t.Parallel()
		specDir := t.TempDir()

		executor := &Executor{}
		err := executor.ValidatePlan(specDir)
		assert.Error(t, err)
	})
}

func TestValidateTasks(t *testing.T) {
	t.Run("tasks exists", func(t *testing.T) {
		t.Parallel()
		specDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "tasks.md"), []byte("# Tasks"), 0644))

		executor := &Executor{}
		err := executor.ValidateTasks(specDir)
		assert.NoError(t, err)
	})

	t.Run("tasks missing", func(t *testing.T) {
		t.Parallel()
		specDir := t.TempDir()

		executor := &Executor{}
		err := executor.ValidateTasks(specDir)
		assert.Error(t, err)
	})
}

func TestValidateTasksComplete(t *testing.T) {
	tests := map[string]struct {
		content string
		wantErr bool
	}{
		"all tasks complete": {
			content: `# Tasks
- [x] Task 1
- [x] Task 2
- [x] Task 3
`,
			wantErr: false,
		},
		"some tasks incomplete": {
			content: `# Tasks
- [x] Task 1
- [ ] Task 2
- [x] Task 3
`,
			wantErr: true,
		},
		"all tasks incomplete": {
			content: `# Tasks
- [ ] Task 1
- [ ] Task 2
`,
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tasksPath := filepath.Join(dir, "tasks.md")
			require.NoError(t, os.WriteFile(tasksPath, []byte(tc.content), 0644))

			executor := &Executor{}
			err := executor.ValidateTasksComplete(tasksPath)

			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "tasks remain")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDebugLog(t *testing.T) {
	t.Run("debug disabled does not print", func(t *testing.T) {
		t.Parallel()
		executor := &Executor{Debug: false}
		// Should not panic and should not print
		executor.debugLog("test message %s", "arg")
	})

	t.Run("debug enabled prints", func(t *testing.T) {
		t.Parallel()
		executor := &Executor{Debug: true}
		// Should not panic - we can't easily capture stdout in this test
		// but we verify it doesn't crash
		executor.debugLog("test message %s", "arg")
	})
}

// TestHandleExecutionFailure tests the handleExecutionFailure method (T010)
func TestHandleExecutionFailure(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		initialRetryCount int
		maxRetries        int
		wantRetryCount    int
		wantExhausted     bool
		wantErrorContains string
	}{
		"first failure increments retry count": {
			initialRetryCount: 0,
			maxRetries:        3,
			wantRetryCount:    1,
			wantExhausted:     false,
			wantErrorContains: "command execution failed",
		},
		"second failure increments retry count": {
			initialRetryCount: 1,
			maxRetries:        3,
			wantRetryCount:    2,
			wantExhausted:     false,
			wantErrorContains: "command execution failed",
		},
		"max retry limit reached returns exhausted": {
			initialRetryCount: 3,
			maxRetries:        3,
			wantRetryCount:    3,
			wantExhausted:     true,
			wantErrorContains: "retry limit exhausted",
		},
		"one before max increments to max": {
			initialRetryCount: 2,
			maxRetries:        3,
			wantRetryCount:    3,
			wantExhausted:     false,
			wantErrorContains: "command execution failed",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()

			// Pre-set retry state if needed
			if tc.initialRetryCount > 0 {
				state := &retry.RetryState{
					SpecName:   "test-spec",
					Phase:      "specify",
					Count:      tc.initialRetryCount,
					MaxRetries: tc.maxRetries,
				}
				require.NoError(t, retry.SaveRetryState(stateDir, state))
			}

			executor := &Executor{
				StateDir:            stateDir,
				MaxRetries:          tc.maxRetries,
				ProgressDisplay:     nil, // Use nil to skip display calls
				NotificationHandler: nil, // Use nil to skip notification calls
			}

			// Load or create retry state
			retryState, err := retry.LoadRetryState(stateDir, "test-spec", "specify", tc.maxRetries)
			require.NoError(t, err)

			result := &StageResult{
				Stage: StageSpecify,
			}

			stageInfo := progress.StageInfo{
				Name:        "specify",
				Number:      1,
				TotalStages: 4,
			}

			execErr := errors.New("claude execution error")

			// Call handleExecutionFailure
			returnErr := executor.handleExecutionFailure(result, retryState, stageInfo, execErr)

			// Verify error message
			assert.Error(t, returnErr)
			assert.Contains(t, returnErr.Error(), tc.wantErrorContains)

			// Verify result state
			if tc.wantExhausted {
				assert.True(t, result.Exhausted)
			}
			assert.Equal(t, tc.wantRetryCount, result.RetryCount)

			// Verify result.Error is set correctly
			assert.Contains(t, result.Error.Error(), "command execution failed")
		})
	}
}

// TestHandleValidationFailure tests the handleValidationFailure method
func TestHandleValidationFailure(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		initialRetryCount int
		maxRetries        int
		wantRetryCount    int
		wantExhausted     bool
		wantErrorContains string
	}{
		"first validation failure increments count": {
			initialRetryCount: 0,
			maxRetries:        3,
			wantRetryCount:    1,
			wantExhausted:     false,
			wantErrorContains: "validation failed",
		},
		"max retry limit reached on validation": {
			initialRetryCount: 3,
			maxRetries:        3,
			wantRetryCount:    3,
			wantExhausted:     true,
			wantErrorContains: "validation failed and retry exhausted",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()

			// Pre-set retry state if needed
			if tc.initialRetryCount > 0 {
				state := &retry.RetryState{
					SpecName:   "test-spec",
					Phase:      "plan",
					Count:      tc.initialRetryCount,
					MaxRetries: tc.maxRetries,
				}
				require.NoError(t, retry.SaveRetryState(stateDir, state))
			}

			executor := &Executor{
				StateDir:            stateDir,
				MaxRetries:          tc.maxRetries,
				ProgressDisplay:     nil,
				NotificationHandler: nil,
			}

			// Load or create retry state
			retryState, err := retry.LoadRetryState(stateDir, "test-spec", "plan", tc.maxRetries)
			require.NoError(t, err)

			result := &StageResult{
				Stage: StagePlan,
			}

			stageInfo := progress.StageInfo{
				Name:        "plan",
				Number:      2,
				TotalStages: 4,
			}

			validationErr := errors.New("schema validation failed")

			// Call handleValidationFailure
			returnErr := executor.handleValidationFailure(result, retryState, stageInfo, validationErr)

			// Verify error message
			assert.Error(t, returnErr)
			assert.Contains(t, returnErr.Error(), tc.wantErrorContains)

			// Verify result state
			if tc.wantExhausted {
				assert.True(t, result.Exhausted)
			}
			assert.Equal(t, tc.wantRetryCount, result.RetryCount)

			// Verify result.Error is set correctly
			assert.Contains(t, result.Error.Error(), "validation failed")
		})
	}
}

// TestHandleRetryIncrement tests the handleRetryIncrement method
func TestHandleRetryIncrement(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		initialCount    int
		maxRetries      int
		exhaustedMsg    string
		wantCount       int
		wantExhausted   bool
		wantErrContains string
	}{
		"increment from zero": {
			initialCount:    0,
			maxRetries:      3,
			exhaustedMsg:    "test exhausted",
			wantCount:       1,
			wantExhausted:   false,
			wantErrContains: "original",
		},
		"increment to max": {
			initialCount:    2,
			maxRetries:      3,
			exhaustedMsg:    "test exhausted",
			wantCount:       3,
			wantExhausted:   false,
			wantErrContains: "original",
		},
		"increment past max returns exhausted": {
			initialCount:    3,
			maxRetries:      3,
			exhaustedMsg:    "test exhausted",
			wantCount:       3,
			wantExhausted:   true,
			wantErrContains: "test exhausted",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()

			// Pre-set retry state
			state := &retry.RetryState{
				SpecName:   "test-spec",
				Phase:      "specify",
				Count:      tc.initialCount,
				MaxRetries: tc.maxRetries,
			}
			require.NoError(t, retry.SaveRetryState(stateDir, state))

			executor := &Executor{
				StateDir:   stateDir,
				MaxRetries: tc.maxRetries,
			}

			// Load retry state
			retryState, err := retry.LoadRetryState(stateDir, "test-spec", "specify", tc.maxRetries)
			require.NoError(t, err)

			originalErr := errors.New("original error")

			result := &StageResult{
				Stage: StageSpecify,
				Error: originalErr, // Set Error since handleRetryIncrement returns it on success
			}

			// Call handleRetryIncrement
			returnedResult, returnErr := executor.handleRetryIncrement(result, retryState, originalErr, tc.exhaustedMsg)

			// Verify result
			assert.Equal(t, tc.wantCount, returnedResult.RetryCount)
			if tc.wantExhausted {
				assert.True(t, returnedResult.Exhausted)
				assert.Contains(t, returnErr.Error(), tc.exhaustedMsg)
			} else {
				assert.False(t, returnedResult.Exhausted)
				// When not exhausted, the function returns result.Error
				assert.Contains(t, returnErr.Error(), "original")
			}
		})
	}
}

// TestCompleteStageSuccessNoNotify tests the completeStageSuccessNoNotify method
func TestCompleteStageSuccessNoNotify(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	// Pre-set retry state with non-zero count
	state := &retry.RetryState{
		SpecName:   "test-spec",
		Phase:      "specify",
		Count:      2,
		MaxRetries: 3,
	}
	require.NoError(t, retry.SaveRetryState(stateDir, state))

	executor := &Executor{
		StateDir:        stateDir,
		MaxRetries:      3,
		ProgressDisplay: nil, // Use nil to skip display calls
	}

	result := &StageResult{
		Stage:   StageSpecify,
		Success: true,
	}

	stageInfo := progress.StageInfo{
		Name:        "specify",
		Number:      1,
		TotalStages: 4,
	}

	// Call completeStageSuccessNoNotify
	executor.completeStageSuccessNoNotify(result, stageInfo, "test-spec", StageSpecify)

	// Verify retry count was reset
	loaded, err := retry.LoadRetryState(stateDir, "test-spec", "specify", 3)
	require.NoError(t, err)
	assert.Equal(t, 0, loaded.Count)
}

// TestHandleExecutionFailure_NilHandlers tests behavior with nil handlers
func TestHandleExecutionFailure_NilHandlers(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	executor := &Executor{
		StateDir:            stateDir,
		MaxRetries:          3,
		ProgressDisplay:     nil, // nil handler
		NotificationHandler: nil, // nil handler
	}

	retryState, err := retry.LoadRetryState(stateDir, "test-spec", "specify", 3)
	require.NoError(t, err)

	result := &StageResult{
		Stage: StageSpecify,
	}

	stageInfo := progress.StageInfo{
		Name: "specify",
	}

	execErr := errors.New("test error")

	// Should not panic with nil handlers
	returnErr := executor.handleExecutionFailure(result, retryState, stageInfo, execErr)

	assert.Error(t, returnErr)
	assert.Contains(t, returnErr.Error(), "command execution failed")
}

// TestErrorMessageFormatting tests error message formatting in failure handlers
func TestErrorMessageFormatting(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		originalErr    error
		handler        string
		wantErrPrefix  string
	}{
		"execution failure wraps error": {
			originalErr:   errors.New("connection timeout"),
			handler:       "execution",
			wantErrPrefix: "command execution failed",
		},
		"validation failure wraps error": {
			originalErr:   errors.New("schema mismatch"),
			handler:       "validation",
			wantErrPrefix: "validation failed",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()

			executor := &Executor{
				StateDir:   stateDir,
				MaxRetries: 3,
			}

			retryState, err := retry.LoadRetryState(stateDir, "test-spec", "specify", 3)
			require.NoError(t, err)

			result := &StageResult{Stage: StageSpecify}
			stageInfo := progress.StageInfo{Name: "specify"}

			var returnErr error
			switch tc.handler {
			case "execution":
				returnErr = executor.handleExecutionFailure(result, retryState, stageInfo, tc.originalErr)
			case "validation":
				returnErr = executor.handleValidationFailure(result, retryState, stageInfo, tc.originalErr)
			}

			// Verify error formatting
			assert.Error(t, returnErr)
			assert.Contains(t, result.Error.Error(), tc.wantErrPrefix)

			// Verify original error is wrapped
			assert.ErrorIs(t, result.Error, tc.originalErr)
		})
	}
}
