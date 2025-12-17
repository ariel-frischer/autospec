package lifecycle

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// mockHandler records notification calls for testing.
type mockHandler struct {
	mu             sync.Mutex
	commandCalls   []commandCall
	stageCalls     []stageCall
	shouldPanic    bool
	panicOnCommand bool
	panicOnStage   bool
}

type commandCall struct {
	name     string
	success  bool
	duration time.Duration
}

type stageCall struct {
	name    string
	success bool
}

func (m *mockHandler) OnCommandComplete(name string, success bool, duration time.Duration) {
	if m.shouldPanic || m.panicOnCommand {
		panic("handler panic")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commandCalls = append(m.commandCalls, commandCall{name, success, duration})
}

func (m *mockHandler) OnStageComplete(name string, success bool) {
	if m.shouldPanic || m.panicOnStage {
		panic("handler panic")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stageCalls = append(m.stageCalls, stageCall{name, success})
}

func (m *mockHandler) getCommandCalls() []commandCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]commandCall{}, m.commandCalls...)
}

func (m *mockHandler) getStageCalls() []stageCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]stageCall{}, m.stageCalls...)
}

func TestRun(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	tests := map[string]struct {
		handler     *mockHandler
		fn          func() error
		wantErr     error
		wantSuccess bool
		wantCalls   int
	}{
		"success": {
			handler:     &mockHandler{},
			fn:          func() error { return nil },
			wantErr:     nil,
			wantSuccess: true,
			wantCalls:   1,
		},
		"failure": {
			handler:     &mockHandler{},
			fn:          func() error { return errTest },
			wantErr:     errTest,
			wantSuccess: false,
			wantCalls:   1,
		},
		"nil handler": {
			handler:     nil,
			fn:          func() error { return nil },
			wantErr:     nil,
			wantSuccess: true,
			wantCalls:   0,
		},
		"handler panics": {
			handler:     &mockHandler{panicOnCommand: true},
			fn:          func() error { return errTest },
			wantErr:     errTest,
			wantSuccess: false,
			wantCalls:   0, // panic prevents recording
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := Run(tt.handler, "test-cmd", tt.fn)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Run() error = %v, want %v", err, tt.wantErr)
			}

			if tt.handler == nil {
				return
			}

			calls := tt.handler.getCommandCalls()
			if len(calls) != tt.wantCalls {
				t.Errorf("got %d calls, want %d", len(calls), tt.wantCalls)
				return
			}

			if tt.wantCalls > 0 {
				if calls[0].name != "test-cmd" {
					t.Errorf("got name %q, want %q", calls[0].name, "test-cmd")
				}
				if calls[0].success != tt.wantSuccess {
					t.Errorf("got success %v, want %v", calls[0].success, tt.wantSuccess)
				}
				if calls[0].duration <= 0 {
					t.Error("duration should be positive")
				}
			}
		})
	}
}

func TestRunWithContext(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		ctx         context.Context
		fn          func(context.Context) error
		wantErr     error
		wantSuccess bool
		fnCalled    bool
	}{
		"valid context success": {
			ctx:         context.Background(),
			fn:          func(ctx context.Context) error { return nil },
			wantErr:     nil,
			wantSuccess: true,
			fnCalled:    true,
		},
		"valid context failure": {
			ctx:         context.Background(),
			fn:          func(ctx context.Context) error { return errors.New("fail") },
			wantErr:     errors.New("fail"),
			wantSuccess: false,
			fnCalled:    true,
		},
		"cancelled context": {
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			fn:          func(ctx context.Context) error { return nil },
			wantErr:     context.Canceled,
			wantSuccess: false,
			fnCalled:    false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			handler := &mockHandler{}
			fnWasCalled := false
			wrappedFn := func(ctx context.Context) error {
				fnWasCalled = true
				return tt.fn(ctx)
			}

			err := RunWithContext(tt.ctx, handler, "test-cmd", wrappedFn)

			if tt.wantErr != nil {
				if err == nil || (err.Error() != tt.wantErr.Error() && !errors.Is(err, tt.wantErr)) {
					t.Errorf("RunWithContext() error = %v, want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("RunWithContext() unexpected error = %v", err)
			}

			if fnWasCalled != tt.fnCalled {
				t.Errorf("fn called = %v, want %v", fnWasCalled, tt.fnCalled)
			}

			calls := handler.getCommandCalls()
			if len(calls) != 1 {
				t.Fatalf("got %d calls, want 1", len(calls))
			}

			if calls[0].success != tt.wantSuccess {
				t.Errorf("got success %v, want %v", calls[0].success, tt.wantSuccess)
			}
		})
	}
}

func TestRunWithContextNilHandler(t *testing.T) {
	t.Parallel()

	err := RunWithContext(context.Background(), nil, "test", func(ctx context.Context) error {
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunStage(t *testing.T) {
	t.Parallel()

	errTest := errors.New("stage error")

	tests := map[string]struct {
		handler     *mockHandler
		fn          func() error
		wantErr     error
		wantSuccess bool
		wantCalls   int
	}{
		"success": {
			handler:     &mockHandler{},
			fn:          func() error { return nil },
			wantErr:     nil,
			wantSuccess: true,
			wantCalls:   1,
		},
		"failure": {
			handler:     &mockHandler{},
			fn:          func() error { return errTest },
			wantErr:     errTest,
			wantSuccess: false,
			wantCalls:   1,
		},
		"nil handler": {
			handler:     nil,
			fn:          func() error { return nil },
			wantErr:     nil,
			wantSuccess: true,
			wantCalls:   0,
		},
		"handler panics": {
			handler:     &mockHandler{panicOnStage: true},
			fn:          func() error { return errTest },
			wantErr:     errTest,
			wantSuccess: false,
			wantCalls:   0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := RunStage(tt.handler, "test-stage", tt.fn)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("RunStage() error = %v, want %v", err, tt.wantErr)
			}

			if tt.handler == nil {
				return
			}

			calls := tt.handler.getStageCalls()
			if len(calls) != tt.wantCalls {
				t.Errorf("got %d calls, want %d", len(calls), tt.wantCalls)
				return
			}

			if tt.wantCalls > 0 {
				if calls[0].name != "test-stage" {
					t.Errorf("got name %q, want %q", calls[0].name, "test-stage")
				}
				if calls[0].success != tt.wantSuccess {
					t.Errorf("got success %v, want %v", calls[0].success, tt.wantSuccess)
				}
			}
		})
	}
}

func TestRunDurationAccuracy(t *testing.T) {
	t.Parallel()

	handler := &mockHandler{}
	sleepDuration := 10 * time.Millisecond

	_ = Run(handler, "sleep-cmd", func() error {
		time.Sleep(sleepDuration)
		return nil
	})

	calls := handler.getCommandCalls()
	if len(calls) != 1 {
		t.Fatalf("got %d calls, want 1", len(calls))
	}

	// Allow 5ms tolerance for timing variance
	if calls[0].duration < sleepDuration {
		t.Errorf("duration %v less than sleep %v", calls[0].duration, sleepDuration)
	}
	if calls[0].duration > sleepDuration+5*time.Millisecond {
		t.Errorf("duration %v too much greater than sleep %v", calls[0].duration, sleepDuration)
	}
}

// mockLogger records history calls for testing two-phase logging.
type mockLogger struct {
	mu           sync.Mutex
	startCalls   []startCall
	updateCalls  []updateCall
	historyCalls []historyCall // kept for backward compat tests
	shouldPanic  bool
	nextID       int
}

type startCall struct {
	command string
	spec    string
	id      string
}

type updateCall struct {
	id       string
	exitCode int
	status   string
	duration time.Duration
}

type historyCall struct {
	command  string
	spec     string
	exitCode int
	duration time.Duration
}

func (m *mockLogger) LogCommand(command, spec string, exitCode int, duration time.Duration) {
	if m.shouldPanic {
		panic("logger panic")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.historyCalls = append(m.historyCalls, historyCall{command, spec, exitCode, duration})
}

func (m *mockLogger) WriteStart(command, spec string) (string, error) {
	if m.shouldPanic {
		panic("logger panic")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	id := "mock_id_" + string(rune('0'+m.nextID))
	m.startCalls = append(m.startCalls, startCall{command, spec, id})
	return id, nil
}

func (m *mockLogger) UpdateComplete(id string, exitCode int, status string, duration time.Duration) error {
	if m.shouldPanic {
		panic("logger panic")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateCalls = append(m.updateCalls, updateCall{id, exitCode, status, duration})
	return nil
}

func (m *mockLogger) getStartCalls() []startCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]startCall{}, m.startCalls...)
}

func (m *mockLogger) getUpdateCalls() []updateCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]updateCall{}, m.updateCalls...)
}

func (m *mockLogger) getHistoryCalls() []historyCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]historyCall{}, m.historyCalls...)
}

func TestRunWithHistory(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	tests := map[string]struct {
		handler      *mockHandler
		logger       *mockLogger
		spec         string
		fn           func() error
		wantErr      error
		wantSuccess  bool
		wantStatus   string
		wantExitCode int
	}{
		"success": {
			handler:      &mockHandler{},
			logger:       &mockLogger{},
			spec:         "test-spec",
			fn:           func() error { return nil },
			wantErr:      nil,
			wantSuccess:  true,
			wantStatus:   StatusCompleted,
			wantExitCode: 0,
		},
		"failure": {
			handler:      &mockHandler{},
			logger:       &mockLogger{},
			spec:         "test-spec",
			fn:           func() error { return errTest },
			wantErr:      errTest,
			wantSuccess:  false,
			wantStatus:   StatusFailed,
			wantExitCode: 1,
		},
		"empty spec": {
			handler:      &mockHandler{},
			logger:       &mockLogger{},
			spec:         "",
			fn:           func() error { return nil },
			wantErr:      nil,
			wantSuccess:  true,
			wantStatus:   StatusCompleted,
			wantExitCode: 0,
		},
		"nil handler": {
			handler:      nil,
			logger:       &mockLogger{},
			spec:         "test-spec",
			fn:           func() error { return nil },
			wantErr:      nil,
			wantSuccess:  true,
			wantStatus:   StatusCompleted,
			wantExitCode: 0,
		},
		"nil logger": {
			handler:      &mockHandler{},
			logger:       nil,
			spec:         "test-spec",
			fn:           func() error { return nil },
			wantErr:      nil,
			wantSuccess:  true,
			wantStatus:   "",
			wantExitCode: 0,
		},
		"both nil": {
			handler:      nil,
			logger:       nil,
			spec:         "test-spec",
			fn:           func() error { return nil },
			wantErr:      nil,
			wantSuccess:  true,
			wantStatus:   "",
			wantExitCode: 0,
		},
		"logger panics": {
			handler:      &mockHandler{},
			logger:       &mockLogger{shouldPanic: true},
			spec:         "test-spec",
			fn:           func() error { return errTest },
			wantErr:      errTest,
			wantSuccess:  false,
			wantStatus:   "",
			wantExitCode: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := RunWithHistory(tt.handler, tt.logger, "test-cmd", tt.spec, tt.fn)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("RunWithHistory() error = %v, want %v", err, tt.wantErr)
			}

			// Check handler was called (if not nil)
			if tt.handler != nil {
				calls := tt.handler.getCommandCalls()
				if len(calls) != 1 {
					t.Errorf("handler got %d calls, want 1", len(calls))
				} else {
					if calls[0].name != "test-cmd" {
						t.Errorf("handler got name %q, want %q", calls[0].name, "test-cmd")
					}
					if calls[0].success != tt.wantSuccess {
						t.Errorf("handler got success %v, want %v", calls[0].success, tt.wantSuccess)
					}
				}
			}

			// Check two-phase logger calls (if not nil and not panicking)
			if tt.logger != nil && !tt.logger.shouldPanic {
				// Verify WriteStart was called
				startCalls := tt.logger.getStartCalls()
				if len(startCalls) != 1 {
					t.Errorf("logger got %d start calls, want 1", len(startCalls))
				} else {
					if startCalls[0].command != "test-cmd" {
						t.Errorf("start call got command %q, want %q", startCalls[0].command, "test-cmd")
					}
					if startCalls[0].spec != tt.spec {
						t.Errorf("start call got spec %q, want %q", startCalls[0].spec, tt.spec)
					}
				}

				// Verify UpdateComplete was called
				updateCalls := tt.logger.getUpdateCalls()
				if len(updateCalls) != 1 {
					t.Errorf("logger got %d update calls, want 1", len(updateCalls))
				} else {
					if updateCalls[0].status != tt.wantStatus {
						t.Errorf("update call got status %q, want %q", updateCalls[0].status, tt.wantStatus)
					}
					if updateCalls[0].exitCode != tt.wantExitCode {
						t.Errorf("update call got exitCode %d, want %d", updateCalls[0].exitCode, tt.wantExitCode)
					}
					if updateCalls[0].duration <= 0 {
						t.Error("update call duration should be positive")
					}
				}
			}
		})
	}
}

func TestRunWithHistoryContext_TwoPhaseLogging(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	tests := map[string]struct {
		ctx          context.Context
		fn           func(context.Context) error
		wantErr      error
		wantStatus   string
		wantExitCode int
	}{
		"success": {
			ctx:          context.Background(),
			fn:           func(ctx context.Context) error { return nil },
			wantErr:      nil,
			wantStatus:   StatusCompleted,
			wantExitCode: 0,
		},
		"failure": {
			ctx:          context.Background(),
			fn:           func(ctx context.Context) error { return errTest },
			wantErr:      errTest,
			wantStatus:   StatusFailed,
			wantExitCode: 1,
		},
		"context already cancelled": {
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			fn:           func(ctx context.Context) error { return nil },
			wantErr:      context.Canceled,
			wantStatus:   StatusCancelled,
			wantExitCode: 1,
		},
		"context cancelled during execution": {
			ctx: context.Background(),
			fn: func(ctx context.Context) error {
				return context.Canceled
			},
			wantErr:      context.Canceled,
			wantStatus:   StatusCancelled,
			wantExitCode: 1,
		},
		"context deadline exceeded": {
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 0)
				cancel()
				return ctx
			}(),
			fn:           func(ctx context.Context) error { return nil },
			wantErr:      context.DeadlineExceeded,
			wantStatus:   StatusCancelled,
			wantExitCode: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			handler := &mockHandler{}
			logger := &mockLogger{}

			err := RunWithHistoryContext(tt.ctx, handler, logger, "test-cmd", "test-spec", tt.fn)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("RunWithHistoryContext() error = %v, want %v", err, tt.wantErr)
			}

			// Verify WriteStart was called
			startCalls := logger.getStartCalls()
			if len(startCalls) != 1 {
				t.Errorf("logger got %d start calls, want 1", len(startCalls))
			} else {
				if startCalls[0].command != "test-cmd" {
					t.Errorf("start call got command %q, want %q", startCalls[0].command, "test-cmd")
				}
			}

			// Verify UpdateComplete was called with correct status
			updateCalls := logger.getUpdateCalls()
			if len(updateCalls) != 1 {
				t.Errorf("logger got %d update calls, want 1", len(updateCalls))
			} else {
				if updateCalls[0].status != tt.wantStatus {
					t.Errorf("update call got status %q, want %q", updateCalls[0].status, tt.wantStatus)
				}
				if updateCalls[0].exitCode != tt.wantExitCode {
					t.Errorf("update call got exitCode %d, want %d", updateCalls[0].exitCode, tt.wantExitCode)
				}
			}
		})
	}
}

func TestRunWithHistoryContext_NilLogger(t *testing.T) {
	t.Parallel()

	handler := &mockHandler{}
	err := RunWithHistoryContext(context.Background(), handler, nil, "test-cmd", "test-spec", func(ctx context.Context) error {
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Handler should still be called
	calls := handler.getCommandCalls()
	if len(calls) != 1 {
		t.Errorf("handler got %d calls, want 1", len(calls))
	}
}

func TestDetermineStatusAndCode(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		err          error
		wantStatus   string
		wantExitCode int
	}{
		"nil error": {
			err:          nil,
			wantStatus:   StatusCompleted,
			wantExitCode: 0,
		},
		"regular error": {
			err:          errors.New("some error"),
			wantStatus:   StatusFailed,
			wantExitCode: 1,
		},
		"context canceled": {
			err:          context.Canceled,
			wantStatus:   StatusCancelled,
			wantExitCode: 1,
		},
		"context deadline exceeded": {
			err:          context.DeadlineExceeded,
			wantStatus:   StatusCancelled,
			wantExitCode: 1,
		},
		"wrapped context canceled": {
			err:          errors.Join(errors.New("wrapped"), context.Canceled),
			wantStatus:   StatusCancelled,
			wantExitCode: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			status, exitCode := determineStatusAndCode(tt.err)

			if status != tt.wantStatus {
				t.Errorf("determineStatusAndCode() status = %q, want %q", status, tt.wantStatus)
			}
			if exitCode != tt.wantExitCode {
				t.Errorf("determineStatusAndCode() exitCode = %d, want %d", exitCode, tt.wantExitCode)
			}
		})
	}
}
