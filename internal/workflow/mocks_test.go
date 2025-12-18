// Package workflow tests mock implementations for ClaudeExecutor, PreflightChecker, and Executor interfaces.
// Related: internal/workflow/claude.go, internal/workflow/preflight.go, internal/workflow/interfaces.go
// Tags: workflow, mocks, testing, executor, preflight, test-doubles
package workflow

import (
	"errors"
	"fmt"
	"io"

	"github.com/ariel-frischer/autospec/internal/validation"
)

// MockClaudeExecutor is a mock implementation of ClaudeExecutor for testing.
// It records method calls and allows configuring return values and errors.
type MockClaudeExecutor struct {
	// Configuration
	ExecuteError     error
	StreamError      error
	FormatCmdFunc    func(string) string
	ExecuteFunc      func(string) error
	StreamFunc       func(string, io.Writer, io.Writer) error
	ExecuteDelay     int // Number of times to fail before succeeding
	executeCallCount int

	// Call tracking
	ExecuteCalls    []string
	StreamCalls     []StreamCall
	FormatCmdCalls  []string
	SpecKitCmdCalls []string
}

// StreamCall records a call to StreamCommand
type StreamCall struct {
	Prompt string
	Stdout io.Writer
	Stderr io.Writer
}

// NewMockClaudeExecutor creates a new mock executor with default behavior
func NewMockClaudeExecutor() *MockClaudeExecutor {
	return &MockClaudeExecutor{
		ExecuteCalls:    make([]string, 0),
		StreamCalls:     make([]StreamCall, 0),
		FormatCmdCalls:  make([]string, 0),
		SpecKitCmdCalls: make([]string, 0),
	}
}

// WithExecuteError configures the mock to return an error on Execute
func (m *MockClaudeExecutor) WithExecuteError(err error) *MockClaudeExecutor {
	m.ExecuteError = err
	return m
}

// WithStreamError configures the mock to return an error on StreamCommand
func (m *MockClaudeExecutor) WithStreamError(err error) *MockClaudeExecutor {
	m.StreamError = err
	return m
}

// WithExecuteFunc configures a custom execute function
func (m *MockClaudeExecutor) WithExecuteFunc(fn func(string) error) *MockClaudeExecutor {
	m.ExecuteFunc = fn
	return m
}

// WithExecuteDelay configures the mock to fail N times before succeeding
func (m *MockClaudeExecutor) WithExecuteDelay(count int) *MockClaudeExecutor {
	m.ExecuteDelay = count
	return m
}

// Execute records the call and returns configured error
func (m *MockClaudeExecutor) Execute(prompt string) error {
	m.ExecuteCalls = append(m.ExecuteCalls, prompt)
	m.executeCallCount++

	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(prompt)
	}

	if m.ExecuteDelay > 0 && m.executeCallCount <= m.ExecuteDelay {
		return fmt.Errorf("mock execute failure %d/%d", m.executeCallCount, m.ExecuteDelay)
	}

	return m.ExecuteError
}

// FormatCommand records the call and returns formatted command
func (m *MockClaudeExecutor) FormatCommand(prompt string) string {
	m.FormatCmdCalls = append(m.FormatCmdCalls, prompt)

	if m.FormatCmdFunc != nil {
		return m.FormatCmdFunc(prompt)
	}

	return "claude " + prompt
}

// ExecuteSpecKitCommand records the call and delegates to Execute
func (m *MockClaudeExecutor) ExecuteSpecKitCommand(command string) error {
	m.SpecKitCmdCalls = append(m.SpecKitCmdCalls, command)
	return m.Execute(command)
}

// StreamCommand records the call and returns configured error
func (m *MockClaudeExecutor) StreamCommand(prompt string, stdout, stderr io.Writer) error {
	m.StreamCalls = append(m.StreamCalls, StreamCall{
		Prompt: prompt,
		Stdout: stdout,
		Stderr: stderr,
	})

	if m.StreamFunc != nil {
		return m.StreamFunc(prompt, stdout, stderr)
	}

	return m.StreamError
}

// Reset clears all recorded calls
func (m *MockClaudeExecutor) Reset() {
	m.ExecuteCalls = make([]string, 0)
	m.StreamCalls = make([]StreamCall, 0)
	m.FormatCmdCalls = make([]string, 0)
	m.SpecKitCmdCalls = make([]string, 0)
	m.executeCallCount = 0
}

// AssertExecuteCalled checks if Execute was called with the given prompt
func (m *MockClaudeExecutor) AssertExecuteCalled(prompt string) bool {
	for _, call := range m.ExecuteCalls {
		if call == prompt {
			return true
		}
	}
	return false
}

// ExecuteCallCount returns the number of times Execute was called
func (m *MockClaudeExecutor) ExecuteCallCount() int {
	return len(m.ExecuteCalls)
}

// Common test errors
var (
	ErrMockExecute        = errors.New("mock execute error")
	ErrMockValidation     = errors.New("mock validation error")
	ErrMockTimeout        = errors.New("mock timeout error")
	ErrMockRetryExhausted = errors.New("mock retry exhausted")
)

// mockPreflightChecker is a test double implementing PreflightChecker interface.
// It allows configuring return values and tracking call counts for verification.
type mockPreflightChecker struct {
	// Configuration for RunChecks
	RunChecksResult *PreflightResult
	RunChecksError  error

	// Configuration for PromptUser
	PromptUserResult bool
	PromptUserError  error

	// Call tracking
	RunChecksCalled    bool
	RunChecksCallCount int
	PromptUserCalled   bool
	PromptUserCalls    []string // Stores warning messages passed to PromptUser
}

// newMockPreflightChecker creates a new mockPreflightChecker with default success behavior.
func newMockPreflightChecker() *mockPreflightChecker {
	return &mockPreflightChecker{
		RunChecksResult: &PreflightResult{
			Passed:       true,
			FailedChecks: make([]string, 0),
			MissingDirs:  make([]string, 0),
		},
		PromptUserResult: true,
		PromptUserCalls:  make([]string, 0),
	}
}

// RunChecks implements PreflightChecker.RunChecks for testing.
func (m *mockPreflightChecker) RunChecks() (*PreflightResult, error) {
	m.RunChecksCalled = true
	m.RunChecksCallCount++
	return m.RunChecksResult, m.RunChecksError
}

// PromptUser implements PreflightChecker.PromptUser for testing.
func (m *mockPreflightChecker) PromptUser(warningMessage string) (bool, error) {
	m.PromptUserCalled = true
	m.PromptUserCalls = append(m.PromptUserCalls, warningMessage)
	return m.PromptUserResult, m.PromptUserError
}

// WithRunChecksResult configures the result returned by RunChecks.
func (m *mockPreflightChecker) WithRunChecksResult(result *PreflightResult) *mockPreflightChecker {
	m.RunChecksResult = result
	return m
}

// WithRunChecksError configures RunChecks to return an error.
func (m *mockPreflightChecker) WithRunChecksError(err error) *mockPreflightChecker {
	m.RunChecksError = err
	return m
}

// WithPromptUserResult configures the result returned by PromptUser.
func (m *mockPreflightChecker) WithPromptUserResult(result bool) *mockPreflightChecker {
	m.PromptUserResult = result
	return m
}

// WithPromptUserError configures PromptUser to return an error.
func (m *mockPreflightChecker) WithPromptUserError(err error) *mockPreflightChecker {
	m.PromptUserError = err
	return m
}

// WithFailedChecks configures RunChecks to return failed checks with a warning.
func (m *mockPreflightChecker) WithFailedChecks(failedChecks []string, warningMessage string) *mockPreflightChecker {
	m.RunChecksResult = &PreflightResult{
		Passed:         false,
		FailedChecks:   failedChecks,
		MissingDirs:    make([]string, 0),
		WarningMessage: warningMessage,
	}
	return m
}

// WithMissingDirs configures RunChecks to return missing directories with a warning.
func (m *mockPreflightChecker) WithMissingDirs(missingDirs []string, warningMessage string) *mockPreflightChecker {
	m.RunChecksResult = &PreflightResult{
		Passed:         false,
		FailedChecks:   make([]string, 0),
		MissingDirs:    missingDirs,
		WarningMessage: warningMessage,
	}
	return m
}

// Reset clears all call tracking state.
func (m *mockPreflightChecker) Reset() {
	m.RunChecksCalled = false
	m.RunChecksCallCount = 0
	m.PromptUserCalled = false
	m.PromptUserCalls = make([]string, 0)
}

// =============================================================================
// Mock Executor Interface Implementations
// =============================================================================

// MockStageExecutor is a mock implementation of StageExecutorInterface for testing.
type MockStageExecutor struct {
	// Return values
	SpecifyResult     string
	SpecifyError      error
	PlanError         error
	TasksError        error
	ConstitutionError error
	ClarifyError      error
	ChecklistError    error
	AnalyzeError      error

	// Call tracking
	SpecifyCalls      []string // Feature descriptions
	PlanCalls         []PlanCall
	TasksCalls        []TasksCall
	ConstitutionCalls []string // Prompts
	ClarifyCalls      []ClarifyCall
	ChecklistCalls    []ChecklistCall
	AnalyzeCalls      []AnalyzeCall
}

// PlanCall records a call to ExecutePlan.
type PlanCall struct {
	SpecNameArg string
	Prompt      string
}

// TasksCall records a call to ExecuteTasks.
type TasksCall struct {
	SpecNameArg string
	Prompt      string
}

// ClarifyCall records a call to ExecuteClarify.
type ClarifyCall struct {
	SpecName string
	Prompt   string
}

// ChecklistCall records a call to ExecuteChecklist.
type ChecklistCall struct {
	SpecName string
	Prompt   string
}

// AnalyzeCall records a call to ExecuteAnalyze.
type AnalyzeCall struct {
	SpecName string
	Prompt   string
}

// NewMockStageExecutor creates a new MockStageExecutor with default success behavior.
func NewMockStageExecutor() *MockStageExecutor {
	return &MockStageExecutor{
		SpecifyResult:     "001-test-feature",
		SpecifyCalls:      make([]string, 0),
		PlanCalls:         make([]PlanCall, 0),
		TasksCalls:        make([]TasksCall, 0),
		ConstitutionCalls: make([]string, 0),
		ClarifyCalls:      make([]ClarifyCall, 0),
		ChecklistCalls:    make([]ChecklistCall, 0),
		AnalyzeCalls:      make([]AnalyzeCall, 0),
	}
}

// ExecuteSpecify implements StageExecutorInterface.
func (m *MockStageExecutor) ExecuteSpecify(featureDescription string) (string, error) {
	m.SpecifyCalls = append(m.SpecifyCalls, featureDescription)
	return m.SpecifyResult, m.SpecifyError
}

// ExecutePlan implements StageExecutorInterface.
func (m *MockStageExecutor) ExecutePlan(specNameArg string, prompt string) error {
	m.PlanCalls = append(m.PlanCalls, PlanCall{SpecNameArg: specNameArg, Prompt: prompt})
	return m.PlanError
}

// ExecuteTasks implements StageExecutorInterface.
func (m *MockStageExecutor) ExecuteTasks(specNameArg string, prompt string) error {
	m.TasksCalls = append(m.TasksCalls, TasksCall{SpecNameArg: specNameArg, Prompt: prompt})
	return m.TasksError
}

// ExecuteConstitution implements StageExecutorInterface.
func (m *MockStageExecutor) ExecuteConstitution(prompt string) error {
	m.ConstitutionCalls = append(m.ConstitutionCalls, prompt)
	return m.ConstitutionError
}

// ExecuteClarify implements StageExecutorInterface.
func (m *MockStageExecutor) ExecuteClarify(specName string, prompt string) error {
	m.ClarifyCalls = append(m.ClarifyCalls, ClarifyCall{SpecName: specName, Prompt: prompt})
	return m.ClarifyError
}

// ExecuteChecklist implements StageExecutorInterface.
func (m *MockStageExecutor) ExecuteChecklist(specName string, prompt string) error {
	m.ChecklistCalls = append(m.ChecklistCalls, ChecklistCall{SpecName: specName, Prompt: prompt})
	return m.ChecklistError
}

// ExecuteAnalyze implements StageExecutorInterface.
func (m *MockStageExecutor) ExecuteAnalyze(specName string, prompt string) error {
	m.AnalyzeCalls = append(m.AnalyzeCalls, AnalyzeCall{SpecName: specName, Prompt: prompt})
	return m.AnalyzeError
}

// Compile-time interface compliance check.
var _ StageExecutorInterface = (*MockStageExecutor)(nil)

// MockPhaseExecutor is a mock implementation of PhaseExecutorInterface for testing.
type MockPhaseExecutor struct {
	// Return values
	PhaseLoopError   error
	SinglePhaseError error
	DefaultError     error

	// Call tracking
	PhaseLoopCalls   []PhaseLoopCall
	SinglePhaseCalls []SinglePhaseCall
	DefaultCalls     []DefaultCall
}

// PhaseLoopCall records a call to ExecutePhaseLoop.
type PhaseLoopCall struct {
	SpecName    string
	TasksPath   string
	Phases      []validation.PhaseInfo
	StartPhase  int
	TotalPhases int
	Prompt      string
}

// SinglePhaseCall records a call to ExecuteSinglePhase.
type SinglePhaseCall struct {
	SpecName    string
	PhaseNumber int
	Prompt      string
}

// DefaultCall records a call to ExecuteDefault.
type DefaultCall struct {
	SpecName string
	SpecDir  string
	Prompt   string
	Resume   bool
}

// NewMockPhaseExecutor creates a new MockPhaseExecutor with default success behavior.
func NewMockPhaseExecutor() *MockPhaseExecutor {
	return &MockPhaseExecutor{
		PhaseLoopCalls:   make([]PhaseLoopCall, 0),
		SinglePhaseCalls: make([]SinglePhaseCall, 0),
		DefaultCalls:     make([]DefaultCall, 0),
	}
}

// ExecutePhaseLoop implements PhaseExecutorInterface.
func (m *MockPhaseExecutor) ExecutePhaseLoop(specName, tasksPath string, phases []validation.PhaseInfo, startPhase, totalPhases int, prompt string) error {
	m.PhaseLoopCalls = append(m.PhaseLoopCalls, PhaseLoopCall{
		SpecName:    specName,
		TasksPath:   tasksPath,
		Phases:      phases,
		StartPhase:  startPhase,
		TotalPhases: totalPhases,
		Prompt:      prompt,
	})
	return m.PhaseLoopError
}

// ExecuteSinglePhase implements PhaseExecutorInterface.
func (m *MockPhaseExecutor) ExecuteSinglePhase(specName string, phaseNumber int, prompt string) error {
	m.SinglePhaseCalls = append(m.SinglePhaseCalls, SinglePhaseCall{
		SpecName:    specName,
		PhaseNumber: phaseNumber,
		Prompt:      prompt,
	})
	return m.SinglePhaseError
}

// ExecuteDefault implements PhaseExecutorInterface.
func (m *MockPhaseExecutor) ExecuteDefault(specName, specDir, prompt string, resume bool) error {
	m.DefaultCalls = append(m.DefaultCalls, DefaultCall{
		SpecName: specName,
		SpecDir:  specDir,
		Prompt:   prompt,
		Resume:   resume,
	})
	return m.DefaultError
}

// Compile-time interface compliance check.
var _ PhaseExecutorInterface = (*MockPhaseExecutor)(nil)

// MockTaskExecutor is a mock implementation of TaskExecutorInterface for testing.
type MockTaskExecutor struct {
	// Return values
	TaskLoopError     error
	SingleTaskError   error
	PrepareResult     []validation.TaskItem
	PrepareStartIdx   int
	PrepareTotalTasks int
	PrepareError      error

	// Call tracking
	TaskLoopCalls   []TaskLoopCall
	SingleTaskCalls []SingleTaskCall
	PrepareCalls    []PrepareCall
}

// TaskLoopCall records a call to ExecuteTaskLoop.
type TaskLoopCall struct {
	SpecName     string
	TasksPath    string
	OrderedTasks []validation.TaskItem
	StartIdx     int
	TotalTasks   int
	Prompt       string
}

// SingleTaskCall records a call to ExecuteSingleTask.
type SingleTaskCall struct {
	SpecName  string
	TaskID    string
	TaskTitle string
	Prompt    string
}

// PrepareCall records a call to PrepareTaskExecution.
type PrepareCall struct {
	TasksPath string
	FromTask  string
}

// NewMockTaskExecutor creates a new MockTaskExecutor with default success behavior.
func NewMockTaskExecutor() *MockTaskExecutor {
	return &MockTaskExecutor{
		PrepareResult:   make([]validation.TaskItem, 0),
		TaskLoopCalls:   make([]TaskLoopCall, 0),
		SingleTaskCalls: make([]SingleTaskCall, 0),
		PrepareCalls:    make([]PrepareCall, 0),
	}
}

// ExecuteTaskLoop implements TaskExecutorInterface.
func (m *MockTaskExecutor) ExecuteTaskLoop(specName, tasksPath string, orderedTasks []validation.TaskItem, startIdx, totalTasks int, prompt string) error {
	m.TaskLoopCalls = append(m.TaskLoopCalls, TaskLoopCall{
		SpecName:     specName,
		TasksPath:    tasksPath,
		OrderedTasks: orderedTasks,
		StartIdx:     startIdx,
		TotalTasks:   totalTasks,
		Prompt:       prompt,
	})
	return m.TaskLoopError
}

// ExecuteSingleTask implements TaskExecutorInterface.
func (m *MockTaskExecutor) ExecuteSingleTask(specName, taskID, taskTitle, prompt string) error {
	m.SingleTaskCalls = append(m.SingleTaskCalls, SingleTaskCall{
		SpecName:  specName,
		TaskID:    taskID,
		TaskTitle: taskTitle,
		Prompt:    prompt,
	})
	return m.SingleTaskError
}

// PrepareTaskExecution implements TaskExecutorInterface.
func (m *MockTaskExecutor) PrepareTaskExecution(tasksPath string, fromTask string) ([]validation.TaskItem, int, int, error) {
	m.PrepareCalls = append(m.PrepareCalls, PrepareCall{
		TasksPath: tasksPath,
		FromTask:  fromTask,
	})
	return m.PrepareResult, m.PrepareStartIdx, m.PrepareTotalTasks, m.PrepareError
}

// Compile-time interface compliance check.
var _ TaskExecutorInterface = (*MockTaskExecutor)(nil)
