// Package workflow provides notification dispatcher for routing stage events.
// Related: internal/workflow/executor.go, internal/notify/handler.go
// Tags: workflow, notifications, dispatch, separation-of-concerns
package workflow

import (
	"github.com/ariel-frischer/autospec/internal/notify"
)

// NotifyDispatcher routes stage-related notifications to an optional handler.
// It wraps a notify.Handler instance and provides nil-safe methods
// that become no-ops when the handler is nil.
//
// Design rationale: Extracted from Executor to separate notification concerns
// from command execution. This enables independent testing of notification
// routing without requiring actual command execution or display updates.
type NotifyDispatcher struct {
	handler *notify.Handler
}

// NewNotifyDispatcher creates a new NotifyDispatcher with the given handler.
// The handler may be nil, in which case all methods become no-ops.
func NewNotifyDispatcher(handler *notify.Handler) *NotifyDispatcher {
	return &NotifyDispatcher{
		handler: handler,
	}
}

// OnStageComplete dispatches a stage completion notification.
// The notification includes the stage name and success/failure status.
// No-op if handler is nil (safe for tests without notifications).
func (n *NotifyDispatcher) OnStageComplete(stageName string, success bool) {
	if n.handler == nil {
		return
	}
	n.handler.OnStageComplete(stageName, success)
}

// OnError dispatches an error notification for a stage or command.
// The notification includes the stage/command name and error details.
// No-op if handler is nil (safe for tests without notifications).
func (n *NotifyDispatcher) OnError(stageName string, err error) {
	if n.handler == nil {
		return
	}
	n.handler.OnError(stageName, err)
}

// HasHandler returns true if a notification handler is configured.
// This can be used to conditionally log messages when no handler is available.
func (n *NotifyDispatcher) HasHandler() bool {
	return n.handler != nil
}

// Handler returns the underlying notification handler.
// Returns nil if no handler is configured.
// This is useful when components need direct access to the handler
// for operations not exposed through the dispatcher interface.
func (n *NotifyDispatcher) Handler() *notify.Handler {
	return n.handler
}
