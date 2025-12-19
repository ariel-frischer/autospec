package workflow

import (
	"errors"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/notify"
	"github.com/stretchr/testify/assert"
)

// mockNotifySender implements notify.Sender for testing
type mockNotifySender struct {
	visualCalls []notify.Notification
	soundCalls  []string
}

func (m *mockNotifySender) SendVisual(n notify.Notification) error {
	m.visualCalls = append(m.visualCalls, n)
	return nil
}

func (m *mockNotifySender) SendSound(soundFile string) error {
	m.soundCalls = append(m.soundCalls, soundFile)
	return nil
}

func (m *mockNotifySender) VisualAvailable() bool {
	return true
}

func (m *mockNotifySender) SoundAvailable() bool {
	return true
}

func newTestHandler(sender *mockNotifySender) *notify.Handler {
	cfg := notify.NotificationConfig{
		Enabled:           true,
		Type:              notify.OutputBoth,
		OnCommandComplete: true,
		OnStageComplete:   true,
		OnError:           true,
	}
	return notify.NewHandlerWithSender(cfg, sender)
}

func TestNewNotifyDispatcher(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		handler     *notify.Handler
		wantHas     bool
		description string
	}{
		"with handler": {
			handler:     newTestHandler(&mockNotifySender{}),
			wantHas:     true,
			description: "should report handler when provided",
		},
		"nil handler": {
			handler:     nil,
			wantHas:     false,
			description: "should report no handler when nil",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dispatcher := NewNotifyDispatcher(tc.handler)
			assert.NotNil(t, dispatcher)
			assert.Equal(t, tc.wantHas, dispatcher.HasHandler(), tc.description)
		})
	}
}

func TestNotifyDispatcher_OnStageComplete(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		handler   *notify.Handler
		stageName string
		success   bool
	}{
		"nil handler no panic": {
			handler:   nil,
			stageName: "specify",
			success:   true,
		},
		"success notification": {
			handler:   newTestHandler(&mockNotifySender{}),
			stageName: "plan",
			success:   true,
		},
		"failure notification": {
			handler:   newTestHandler(&mockNotifySender{}),
			stageName: "tasks",
			success:   false,
		},
		"empty stage name": {
			handler:   newTestHandler(&mockNotifySender{}),
			stageName: "",
			success:   true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dispatcher := NewNotifyDispatcher(tc.handler)
			// OnStageComplete should not panic regardless of inputs
			assert.NotPanics(t, func() {
				dispatcher.OnStageComplete(tc.stageName, tc.success)
			})
		})
	}
}

func TestNotifyDispatcher_OnError(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		handler   *notify.Handler
		stageName string
		err       error
	}{
		"nil handler no panic": {
			handler:   nil,
			stageName: "specify",
			err:       errors.New("test error"),
		},
		"with error": {
			handler:   newTestHandler(&mockNotifySender{}),
			stageName: "plan",
			err:       errors.New("validation failed"),
		},
		"nil error": {
			handler:   newTestHandler(&mockNotifySender{}),
			stageName: "tasks",
			err:       nil,
		},
		"empty stage name with error": {
			handler:   newTestHandler(&mockNotifySender{}),
			stageName: "",
			err:       errors.New("some error"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dispatcher := NewNotifyDispatcher(tc.handler)
			// OnError should not panic regardless of inputs
			assert.NotPanics(t, func() {
				dispatcher.OnError(tc.stageName, tc.err)
			})
		})
	}
}

func TestNotifyDispatcher_HasHandler(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		handler *notify.Handler
		want    bool
	}{
		"nil handler": {
			handler: nil,
			want:    false,
		},
		"non-nil handler": {
			handler: newTestHandler(&mockNotifySender{}),
			want:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dispatcher := NewNotifyDispatcher(tc.handler)
			assert.Equal(t, tc.want, dispatcher.HasHandler())
		})
	}
}

func TestNotifyDispatcher_Handler(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		handler *notify.Handler
		wantNil bool
	}{
		"nil handler returns nil": {
			handler: nil,
			wantNil: true,
		},
		"non-nil handler returns handler": {
			handler: newTestHandler(&mockNotifySender{}),
			wantNil: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dispatcher := NewNotifyDispatcher(tc.handler)
			got := dispatcher.Handler()
			if tc.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.Equal(t, tc.handler, got)
			}
		})
	}
}

func TestNotifyDispatcher_NilSafety(t *testing.T) {
	t.Parallel()

	// This test verifies the core requirement: nil handler should not panic
	dispatcher := NewNotifyDispatcher(nil)

	// All methods should be safe to call with nil handler
	assert.NotPanics(t, func() {
		dispatcher.OnStageComplete("any-stage", true)
		dispatcher.OnStageComplete("any-stage", false)
		dispatcher.OnError("any-stage", errors.New("error"))
		dispatcher.OnError("any-stage", nil)
		_ = dispatcher.HasHandler()
		_ = dispatcher.Handler()
	})
}

func TestNotifyDispatcher_DisabledHandler(t *testing.T) {
	t.Parallel()

	// Create handler with notifications disabled
	cfg := notify.NotificationConfig{
		Enabled:         false,
		OnStageComplete: true,
		OnError:         true,
	}
	handler := notify.NewHandler(cfg)
	dispatcher := NewNotifyDispatcher(handler)

	// Methods should not panic even with disabled handler
	assert.NotPanics(t, func() {
		dispatcher.OnStageComplete("stage", true)
		dispatcher.OnError("stage", errors.New("error"))
	})

	// Handler should still be accessible
	assert.True(t, dispatcher.HasHandler())
	assert.NotNil(t, dispatcher.Handler())
}

func TestNotifyDispatcher_Integration(t *testing.T) {
	// Integration test showing the component works with real handler
	// Note: This test doesn't verify actual notification dispatch
	// because that depends on TTY and CI detection
	t.Parallel()

	cfg := notify.NotificationConfig{
		Enabled:              true,
		Type:                 notify.OutputBoth,
		OnCommandComplete:    true,
		OnStageComplete:      true,
		OnError:              true,
		OnLongRunning:        false,
		LongRunningThreshold: 2 * time.Minute,
	}
	handler := notify.NewHandler(cfg)
	dispatcher := NewNotifyDispatcher(handler)

	// Verify component is correctly wired
	assert.True(t, dispatcher.HasHandler())
	assert.Equal(t, handler, dispatcher.Handler())

	// These calls won't actually send notifications in test environment
	// (CI detection disables notifications) but they verify no panics
	dispatcher.OnStageComplete("specify", true)
	dispatcher.OnStageComplete("plan", false)
	dispatcher.OnError("tasks", errors.New("validation error"))
}
