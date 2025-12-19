package workflow

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

// TestExecuteStage_ValidationFailure tests the retry exhaustion path.
//
// Scenario: Validation always fails → exhausts all 3 retries → returns exhausted error.
// Verifies the full retry loop executes MaxRetries times before giving up,
// and that result.Exhausted is true with correct RetryCount.
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
	assert.Contains(t, err.Error(), "retry exhausted")
	assert.False(t, result.Success)
	assert.Equal(t, 3, result.RetryCount) // After exhausting all retries (MaxRetries=3)
	assert.True(t, result.Exhausted)      // Retries exhausted
}

// TestExecuteStage_RetryExhausted tests pre-exhausted retry state handling.
//
// Scenario: Retry state already at max (3/3) before execution → first failure
// immediately returns exhausted error without attempting more retries.
// Differs from TestExecuteStage_ValidationFailure which starts at 0 retries.
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

// TestExecuteStage_ResetsRetryOnSuccess verifies retry count resets on success.
//
// Scenario: Pre-existing retry count (2/3) → validation succeeds → retry count
// resets to 0. This ensures the next run starts fresh after recovery.
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

func TestFormatRetryContext(t *testing.T) {
	t.Parallel()

	// Tests verify that FormatRetryContext:
	// - Returns only "RETRY X/Y" when no validation errors (no instructions injected)
	// - Returns "RETRY X/Y" + errors + retryInstructions when validation errors present
	// This ensures first-run executions don't waste tokens on retry instructions.

	tests := map[string]struct {
		attemptNum          int
		maxRetries          int
		validationErrors    []string
		wantPrefix          string // The expected content before instructions
		wantHasInstructions bool   // Whether retryInstructions should be appended
	}{
		"no errors - no instructions": {
			attemptNum:          2,
			maxRetries:          3,
			validationErrors:    nil,
			wantPrefix:          "RETRY 2/3",
			wantHasInstructions: false,
		},
		"empty errors slice - no instructions": {
			attemptNum:          1,
			maxRetries:          3,
			validationErrors:    []string{},
			wantPrefix:          "RETRY 1/3",
			wantHasInstructions: false,
		},
		"single error - includes instructions": {
			attemptNum:          2,
			maxRetries:          3,
			validationErrors:    []string{"missing required field: feature.branch"},
			wantPrefix:          "RETRY 2/3\nSchema validation failed:\n- missing required field: feature.branch",
			wantHasInstructions: true,
		},
		"multiple errors - includes instructions": {
			attemptNum:          1,
			maxRetries:          5,
			validationErrors:    []string{"error one", "error two", "error three"},
			wantPrefix:          "RETRY 1/5\nSchema validation failed:\n- error one\n- error two\n- error three",
			wantHasInstructions: true,
		},
		"exactly 10 errors - includes instructions": {
			attemptNum: 2,
			maxRetries: 3,
			validationErrors: []string{
				"error 1", "error 2", "error 3", "error 4", "error 5",
				"error 6", "error 7", "error 8", "error 9", "error 10",
			},
			wantPrefix:          "RETRY 2/3\nSchema validation failed:\n- error 1\n- error 2\n- error 3\n- error 4\n- error 5\n- error 6\n- error 7\n- error 8\n- error 9\n- error 10",
			wantHasInstructions: true,
		},
		"more than 10 errors truncated - includes instructions": {
			attemptNum: 3,
			maxRetries: 3,
			validationErrors: []string{
				"error 1", "error 2", "error 3", "error 4", "error 5",
				"error 6", "error 7", "error 8", "error 9", "error 10",
				"error 11", "error 12",
			},
			wantPrefix:          "RETRY 3/3\nSchema validation failed:\n- error 1\n- error 2\n- error 3\n- error 4\n- error 5\n- error 6\n- error 7\n- error 8\n- error 9\n- error 10\n...and 2 more errors",
			wantHasInstructions: true,
		},
		"15 errors shows truncation - includes instructions": {
			attemptNum: 1,
			maxRetries: 5,
			validationErrors: []string{
				"e1", "e2", "e3", "e4", "e5", "e6", "e7", "e8", "e9", "e10",
				"e11", "e12", "e13", "e14", "e15",
			},
			wantPrefix:          "RETRY 1/5\nSchema validation failed:\n- e1\n- e2\n- e3\n- e4\n- e5\n- e6\n- e7\n- e8\n- e9\n- e10\n...and 5 more errors",
			wantHasInstructions: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := FormatRetryContext(tc.attemptNum, tc.maxRetries, tc.validationErrors)

			if tc.wantHasInstructions {
				// Should start with the error info prefix
				assert.True(t, strings.HasPrefix(got, tc.wantPrefix),
					"output should start with error info:\ngot: %s\nwant prefix: %s", got, tc.wantPrefix)
				// Should contain the retry instructions
				assert.Contains(t, got, "## Retry Instructions",
					"output with errors should include retry instructions header")
				assert.Contains(t, got, "Common Schema Errors and Fixes",
					"output with errors should include fix guidance")
			} else {
				// Should be exactly the prefix with no instructions
				assert.Equal(t, tc.wantPrefix, got,
					"output without errors should be just RETRY X/Y, no instructions")
				assert.NotContains(t, got, "## Retry Instructions",
					"output without errors should NOT include retry instructions")
			}
		})
	}
}

// TestRetryInstructionsContent verifies the retryInstructions constant contains
// the required elements for proper retry handling guidance.
func TestRetryInstructionsContent(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		expectedContent string
		description     string
	}{
		"has retry header": {
			expectedContent: "## Retry Instructions",
			description:     "should have the main heading",
		},
		"has retry indicator guidance": {
			expectedContent: "RETRY X/Y",
			description:     "should explain the retry format",
		},
		"has error parsing guidance": {
			expectedContent: "starting with \"- \"",
			description:     "should explain how errors are formatted",
		},
		"has missing field fix": {
			expectedContent: "missing required field",
			description:     "should document how to fix missing fields",
		},
		"has invalid enum fix": {
			expectedContent: "invalid enum value",
			description:     "should document how to fix enum errors",
		},
		"has type mismatch fix": {
			expectedContent: "invalid type for",
			description:     "should document how to fix type errors",
		},
		"has pattern mismatch fix": {
			expectedContent: "does not match pattern",
			description:     "should document how to fix pattern errors",
		},
		"has dot notation explanation": {
			expectedContent: "dot notation",
			description:     "should explain field path notation",
		},
		"has array index explanation": {
			expectedContent: "indices start at 0",
			description:     "should explain array indexing",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, retryInstructions, tc.expectedContent,
				"%s: retryInstructions should contain %q", tc.description, tc.expectedContent)
		})
	}
}

// TestRetryInstructionsNotEmpty ensures the constant is defined and has content.
func TestRetryInstructionsNotEmpty(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, retryInstructions, "retryInstructions constant must not be empty")
	// Should be approximately 30-50 lines of content (1000-2500 chars)
	assert.Greater(t, len(retryInstructions), 1000,
		"retryInstructions should be substantial (>1000 chars)")
	assert.Less(t, len(retryInstructions), 3000,
		"retryInstructions should be concise (<3000 chars)")
}

func TestBuildRetryCommand(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		command      string
		retryContext string
		originalArgs string
		want         string
	}{
		"no context no args": {
			command:      "/autospec.plan",
			retryContext: "",
			originalArgs: "",
			want:         "/autospec.plan",
		},
		"no context with args": {
			command:      "/autospec.specify",
			retryContext: "",
			originalArgs: "add user auth",
			want:         "/autospec.specify add user auth",
		},
		"context only no original args": {
			command:      "/autospec.plan",
			retryContext: "RETRY 2/3\nSchema validation failed:\n- missing field",
			originalArgs: "",
			want:         "/autospec.plan RETRY 2/3\nSchema validation failed:\n- missing field",
		},
		"context with original args": {
			command:      "/autospec.specify",
			retryContext: "RETRY 1/3\nSchema validation failed:\n- error",
			originalArgs: "feature description",
			want:         "/autospec.specify RETRY 1/3\nSchema validation failed:\n- error\n\nfeature description",
		},
		"multiline original args": {
			command:      "/autospec.implement",
			retryContext: "RETRY 2/3",
			originalArgs: "--phase 1\n--verbose",
			want:         "/autospec.implement RETRY 2/3\n\n--phase 1\n--verbose",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := BuildRetryCommand(tc.command, tc.retryContext, tc.originalArgs)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestExtractValidationErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		err  error
		want []string
	}{
		"nil error": {
			err:  nil,
			want: nil,
		},
		"single line error no bullets": {
			err:  errors.New("file not found"),
			want: []string{"file not found"},
		},
		"standard validation error format": {
			err:  errors.New("schema validation failed for spec.yaml:\n- missing required field: feature.branch\n- invalid enum value"),
			want: []string{"missing required field: feature.branch", "invalid enum value"},
		},
		"multiple bullet errors": {
			err:  errors.New("validation failed:\n- error one\n- error two\n- error three"),
			want: []string{"error one", "error two", "error three"},
		},
		"mixed content with bullets": {
			err:  errors.New("header line\n- bullet one\nregular line\n- bullet two"),
			want: []string{"bullet one", "bullet two"},
		},
		"whitespace handling": {
			err:  errors.New("  - trimmed error  \n  - another error  "),
			want: []string{"trimmed error", "another error"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := ExtractValidationErrors(tc.err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestHandleValidationFailure_StoresValidationErrors(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	executor := &Executor{
		Claude: &ClaudeExecutor{
			ClaudeCmd:  "echo",
			ClaudeArgs: []string{},
		},
		StateDir:   stateDir,
		MaxRetries: 3,
	}

	result := &StageResult{
		Stage:   StageSpecify,
		Success: false,
	}

	retryState := &retry.RetryState{
		SpecName:   "test-spec",
		Phase:      "specify",
		Count:      0,
		MaxRetries: 3,
	}

	stageInfo := executor.buildStageInfo(StageSpecify, 0)

	// Create a validation error in the expected format
	validationErr := errors.New("schema validation failed for spec.yaml:\n- missing required field: feature.branch\n- invalid enum value: status")

	// Call handleValidationFailure
	_ = executor.handleValidationFailure(result, retryState, stageInfo, validationErr)

	// Verify validation errors were extracted and stored
	assert.NotNil(t, result.ValidationErrors)
	assert.Len(t, result.ValidationErrors, 2)
	assert.Contains(t, result.ValidationErrors, "missing required field: feature.branch")
	assert.Contains(t, result.ValidationErrors, "invalid enum value: status")
}

// TestExecuteStage_RetryLoopActuallyRetries verifies that the retry loop
// actually executes multiple times when validation fails.
// This test prevents regression of the bug where the loop was missing.
func TestExecuteStage_RetryLoopActuallyRetries(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	specsDir := t.TempDir()

	// Track how many times validation is called
	validationCallCount := 0

	executor := &Executor{
		Claude: &ClaudeExecutor{
			ClaudeCmd:  "echo",
			ClaudeArgs: []string{"success"},
		},
		StateDir:   stateDir,
		SpecsDir:   specsDir,
		MaxRetries: 2, // Allow 2 retries (3 total attempts)
	}

	// Validation function that always fails
	validateFunc := func(dir string) error {
		validationCallCount++
		return errors.New("schema validation failed for spec.yaml:\n- missing required field: requirements")
	}

	result, err := executor.ExecuteStage("001-test", StageSpecify, "/test.command", validateFunc)

	// Should have an error (exhausted)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exhausted")
	assert.True(t, result.Exhausted)

	// Verify the loop ran 3 times (1 initial + 2 retries)
	assert.Equal(t, 3, validationCallCount, "validation should be called 3 times (1 initial + 2 retries)")
}

// TestExecuteStage_ValidationSuccessOnRetry verifies that if validation
// succeeds on a retry attempt, the function returns success.
func TestExecuteStage_ValidationSuccessOnRetry(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	specsDir := t.TempDir()

	callCount := 0
	executor := &Executor{
		Claude: &ClaudeExecutor{
			ClaudeCmd:  "echo",
			ClaudeArgs: []string{"success"},
		},
		StateDir:   stateDir,
		SpecsDir:   specsDir,
		MaxRetries: 2,
	}

	// Validation function that fails first time, succeeds second time
	validateFunc := func(dir string) error {
		callCount++
		if callCount == 1 {
			return errors.New("schema validation failed for spec.yaml:\n- missing field")
		}
		return nil // Success on second attempt
	}

	result, err := executor.ExecuteStage("001-test", StageSpecify, "/test.command", validateFunc)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.False(t, result.Exhausted)
	assert.Equal(t, 2, callCount, "validation should be called twice (1 initial + 1 retry that succeeds)")
}

// TestExecuteStage_MaxRetriesZeroNoRetries verifies that with max_retries=0,
// no retries happen and the function returns error on first failure.
func TestExecuteStage_MaxRetriesZeroNoRetries(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	specsDir := t.TempDir()

	validationCallCount := 0
	executor := &Executor{
		Claude: &ClaudeExecutor{
			ClaudeCmd:  "echo",
			ClaudeArgs: []string{"success"},
		},
		StateDir:   stateDir,
		SpecsDir:   specsDir,
		MaxRetries: 0, // No retries allowed
	}

	// Validation function that always fails
	validateFunc := func(dir string) error {
		validationCallCount++
		return errors.New("validation failed")
	}

	result, err := executor.ExecuteStage("001-test", StageSpecify, "/test.command", validateFunc)

	require.Error(t, err)
	assert.True(t, result.Exhausted)
	assert.Equal(t, 1, validationCallCount, "validation should only be called once with max_retries=0")
}

// TestExecuteStage_RetryCountMatchesConfig verifies that the number of
// retries matches the max_retries configuration.
func TestExecuteStage_RetryCountMatchesConfig(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		maxRetries         int
		expectedValidCalls int
	}{
		"max_retries=0 means 1 attempt":  {maxRetries: 0, expectedValidCalls: 1},
		"max_retries=1 means 2 attempts": {maxRetries: 1, expectedValidCalls: 2},
		"max_retries=2 means 3 attempts": {maxRetries: 2, expectedValidCalls: 3},
		"max_retries=3 means 4 attempts": {maxRetries: 3, expectedValidCalls: 4},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()
			specsDir := t.TempDir()
			validationCallCount := 0

			executor := &Executor{
				Claude: &ClaudeExecutor{
					ClaudeCmd:  "echo",
					ClaudeArgs: []string{"success"},
				},
				StateDir:   stateDir,
				SpecsDir:   specsDir,
				MaxRetries: tc.maxRetries,
			}

			validateFunc := func(dir string) error {
				validationCallCount++
				return errors.New("always fails")
			}

			result, err := executor.ExecuteStage("001-test", StageSpecify, "/test.command", validateFunc)

			require.Error(t, err)
			assert.True(t, result.Exhausted)
			assert.Equal(t, tc.expectedValidCalls, validationCallCount,
				"with max_retries=%d, validation should be called %d times",
				tc.maxRetries, tc.expectedValidCalls)
		})
	}
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

// TestHandleValidationFailureRetryBehavior tests the handleValidationFailure method retry behavior
func TestHandleValidationFailureRetryBehavior(t *testing.T) {
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
		originalErr   error
		handler       string
		wantErrPrefix string
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

// TestStartProgressDisplay_EdgeCases tests edge cases for the startProgressDisplay method.
// Specifically tests: nil ProgressDisplay handling and error path with warning output.
func TestStartProgressDisplay_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		progressDisplay *progress.ProgressDisplay
		stageInfo       progress.StageInfo
		desc            string
	}{
		"nil ProgressDisplay does not panic": {
			progressDisplay: nil,
			stageInfo: progress.StageInfo{
				Name:        "test",
				Number:      1,
				TotalStages: 3,
			},
			desc: "nil ProgressDisplay should be handled gracefully without panic",
		},
		"valid ProgressDisplay with valid stage": {
			progressDisplay: progress.NewProgressDisplay(progress.TerminalCapabilities{
				IsTTY:         false,
				SupportsColor: false,
			}),
			stageInfo: progress.StageInfo{
				Name:        "test",
				Number:      1,
				TotalStages: 3,
			},
			desc: "valid ProgressDisplay should work correctly",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			executor := &Executor{
				ProgressDisplay: tt.progressDisplay,
			}

			// This should not panic
			assert.NotPanics(t, func() {
				executor.startProgressDisplay(tt.stageInfo)
			}, tt.desc)
		})
	}
}

// TestExecuteStage_WithMockClaudeRunner tests ExecuteStage using the mock ClaudeRunner interface.
// This verifies that the Executor properly delegates to the ClaudeRunner interface,
// enabling unit testing without actual Claude CLI invocations.
func TestExecuteStage_WithMockClaudeRunner(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		mockErr       error
		validateErr   error
		wantSuccess   bool
		wantErr       bool
		wantErrMsg    string
		wantCallCount int
	}{
		"success case - mock returns nil, stage completes": {
			mockErr:       nil,
			validateErr:   nil,
			wantSuccess:   true,
			wantErr:       false,
			wantCallCount: 1,
		},
		"failure case - mock returns error, error propagated": {
			mockErr:       errors.New("claude execution failed"),
			validateErr:   nil,
			wantSuccess:   false,
			wantErr:       true,
			wantErrMsg:    "retry limit exhausted", // MaxRetries=0 means immediate exhaustion
			wantCallCount: 1,
		},
		"validation failure - mock succeeds but validation fails": {
			mockErr:       nil,
			validateErr:   errors.New("schema validation failed"),
			wantSuccess:   false,
			wantErr:       true,
			wantErrMsg:    "validation failed",
			wantCallCount: 1, // MaxRetries=0, so only 1 call
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()
			specsDir := t.TempDir()

			// Create mock that implements ClaudeRunner interface
			mock := &mockClaudeExecutor{
				executeErr: tc.mockErr,
			}

			executor := &Executor{
				Claude:     mock, // Using interface injection
				StateDir:   stateDir,
				SpecsDir:   specsDir,
				MaxRetries: 0, // No retries for simple tests
			}

			validateFunc := func(dir string) error {
				return tc.validateErr
			}

			result, err := executor.ExecuteStage("001-test", StageSpecify, "/test.command", validateFunc)

			// Verify error expectation
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrMsg)
			} else {
				require.NoError(t, err)
			}

			// Verify result
			assert.Equal(t, tc.wantSuccess, result.Success)

			// Verify mock was called correct number of times
			assert.Len(t, mock.executeCalls, tc.wantCallCount,
				"mock should be called %d time(s)", tc.wantCallCount)

			// Verify the command was passed to mock
			if tc.wantCallCount > 0 {
				assert.Contains(t, mock.executeCalls[0], "/test.command")
			}
		})
	}
}

// TestExecuteStage_MockRetryBehavior tests retry behavior with mock ClaudeRunner.
// Verifies that retries work correctly when validation fails and then succeeds.
func TestExecuteStage_MockRetryBehavior(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		failUntilAttempt int  // Validation fails until this attempt (1-based)
		maxRetries       int  // Max retries allowed
		wantSuccess      bool // Expected final result
		wantCallCount    int  // Expected number of Execute calls
	}{
		"retry succeeds on second attempt": {
			failUntilAttempt: 1, // Fail first, succeed second
			maxRetries:       2,
			wantSuccess:      true,
			wantCallCount:    2,
		},
		"retry succeeds on third attempt": {
			failUntilAttempt: 2, // Fail first two, succeed third
			maxRetries:       3,
			wantSuccess:      true,
			wantCallCount:    3,
		},
		"all retries exhausted": {
			failUntilAttempt: 10, // Always fail
			maxRetries:       2,
			wantSuccess:      false,
			wantCallCount:    3, // 1 initial + 2 retries
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()
			specsDir := t.TempDir()

			// Create mock
			mock := &mockClaudeExecutor{
				executeErr: nil, // Execute always succeeds
			}

			executor := &Executor{
				Claude:     mock,
				StateDir:   stateDir,
				SpecsDir:   specsDir,
				MaxRetries: tc.maxRetries,
			}

			// Track validation attempts
			validationAttempt := 0
			validateFunc := func(dir string) error {
				validationAttempt++
				if validationAttempt <= tc.failUntilAttempt {
					return errors.New("schema validation failed:\n- missing field")
				}
				return nil
			}

			result, _ := executor.ExecuteStage("001-test", StageSpecify, "/test.command", validateFunc)

			// Verify success state
			assert.Equal(t, tc.wantSuccess, result.Success)

			// Verify Execute was called correct number of times
			assert.Len(t, mock.executeCalls, tc.wantCallCount,
				"mock should be called %d time(s)", tc.wantCallCount)
		})
	}
}

// TestExecuteWithRetry_MockClaudeRunner tests ExecuteWithRetry with mock ClaudeRunner.
// This verifies simplified retry logic without stage tracking.
func TestExecuteWithRetry_MockClaudeRunner(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		failUntilCall int  // Execute fails until this call (1-based)
		maxAttempts   int  // Max attempts allowed
		wantErr       bool // Expected error
		wantCallCount int  // Expected Execute calls
	}{
		"succeeds first try": {
			failUntilCall: 0, // Never fail
			maxAttempts:   3,
			wantErr:       false,
			wantCallCount: 1,
		},
		"succeeds after one failure": {
			failUntilCall: 1, // Fail first, succeed second
			maxAttempts:   3,
			wantErr:       false,
			wantCallCount: 2,
		},
		"all attempts fail": {
			failUntilCall: 10, // Always fail
			maxAttempts:   2,
			wantErr:       true,
			wantCallCount: 2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			callCount := 0
			mock := &mockClaudeExecutor{}

			// Override Execute behavior to track calls and conditionally fail
			executor := &Executor{
				Claude: &conditionalMockRunner{
					failUntilCall: tc.failUntilCall,
					callCount:     &callCount,
				},
			}

			err := executor.ExecuteWithRetry("/test.command", tc.maxAttempts)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "all")
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.wantCallCount, callCount,
				"Execute should be called %d time(s)", tc.wantCallCount)

			// Verify mock captures the pattern (not used here since we use conditionalMockRunner)
			_ = mock
		})
	}
}

// conditionalMockRunner is a ClaudeRunner that fails conditionally based on call count.
type conditionalMockRunner struct {
	failUntilCall int
	callCount     *int
}

func (c *conditionalMockRunner) Execute(prompt string) error {
	*c.callCount++
	if *c.callCount <= c.failUntilCall {
		return errors.New("mock execution error")
	}
	return nil
}

func (c *conditionalMockRunner) FormatCommand(prompt string) string {
	return "mock-claude " + prompt
}

// TestExecutor_ClaudeRunnerInterface verifies that Executor.Claude accepts ClaudeRunner interface.
// This is a compile-time check that the interface is correctly typed.
func TestExecutor_ClaudeRunnerInterface(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		runner      ClaudeRunner
		description string
	}{
		"accepts mockClaudeExecutor": {
			runner:      &mockClaudeExecutor{},
			description: "mock implementation should satisfy ClaudeRunner",
		},
		"accepts conditionalMockRunner": {
			runner:      &conditionalMockRunner{callCount: new(int)},
			description: "conditional mock should satisfy ClaudeRunner",
		},
		"accepts real ClaudeExecutor": {
			runner:      &ClaudeExecutor{ClaudeCmd: "echo"},
			description: "real implementation should satisfy ClaudeRunner",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create executor with the runner - this verifies interface compatibility
			executor := &Executor{
				Claude:   tc.runner,
				StateDir: t.TempDir(),
				SpecsDir: t.TempDir(),
			}

			// Verify the runner is accessible
			assert.NotNil(t, executor.Claude, tc.description)

			// Verify FormatCommand works through the interface
			cmd := executor.Claude.FormatCommand("test")
			assert.NotEmpty(t, cmd, "FormatCommand should return non-empty string")
		})
	}
}
