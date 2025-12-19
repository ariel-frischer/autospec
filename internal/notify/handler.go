package notify

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

// Handler manages notification dispatch based on configuration and hooks.
// It wraps a Sender with configuration and provides hook methods for
// command completion, stage completion, and error notifications.
type Handler struct {
	config    NotificationConfig
	sender    Sender
	startTime time.Time
}

// NewHandler creates a new notification handler with the given configuration.
// The handler initializes with the current time as the command start time.
// If notifications are disabled in config, the handler will no-op on all calls.
func NewHandler(config NotificationConfig) *Handler {
	return &Handler{
		config:    config,
		sender:    NewSender(),
		startTime: time.Now(),
	}
}

// NewHandlerWithSender creates a handler with a custom sender (for testing).
func NewHandlerWithSender(config NotificationConfig, sender Sender) *Handler {
	return &Handler{
		config:    config,
		sender:    sender,
		startTime: time.Now(),
	}
}

// SetStartTime updates the command start time (useful for accurate duration tracking)
func (h *Handler) SetStartTime(t time.Time) {
	h.startTime = t
}

// Config returns the handler's notification configuration
func (h *Handler) Config() NotificationConfig {
	return h.config
}

// isEnabled checks if notifications should be sent.
// Returns false if notifications are disabled, running in CI, or non-interactive.
func (h *Handler) isEnabled() bool {
	if !h.config.Enabled {
		return false
	}

	// Check CI environment - auto-disable unless running interactively
	if isCI() {
		return false
	}

	// Check TTY availability for interactive mode
	if !isInteractive() {
		return false
	}

	return true
}

// isCI checks for common CI environment variables.
// Returns true if any CI-related environment variable is set.
func isCI() bool {
	ciVars := []string{
		"CI",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"CIRCLECI",
		"TRAVIS",
		"JENKINS_URL",
		"BUILDKITE",
		"DRONE",
		"TEAMCITY_VERSION",
		"TF_BUILD",            // Azure DevOps
		"BITBUCKET_PIPELINES", // Bitbucket
		"CODEBUILD_BUILD_ID",  // AWS CodeBuild
		"HEROKU_TEST_RUN_ID",  // Heroku CI
		"NETLIFY",             // Netlify
		"VERCEL",              // Vercel
		"RENDER",              // Render
		"RAILWAY_ENVIRONMENT", // Railway
	}
	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}
	return false
}

// isInteractive checks if the session is interactive (has TTY).
// Checks stdout rather than stdin because CLI tools often have stdin piped
// while stdout remains connected to the terminal.
//
// TEST COVERAGE BLOCKED: Requires real TTY; term.IsTerminal cannot be mocked
// without adding interfaces to production code.
func isInteractive() bool {
	// Check stdout first (most reliable for CLI tools)
	if term.IsTerminal(int(os.Stdout.Fd())) {
		return true
	}
	// Fall back to stderr (also commonly connected to terminal)
	if term.IsTerminal(int(os.Stderr.Fd())) {
		return true
	}
	// Finally check stdin
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// dispatch sends a notification asynchronously with a timeout.
//
// Concurrency pattern: goroutine + done channel + select with timeout.
// The 5s timeout allows audio files to play but prevents indefinite blocking.
// Notification failures are silent (logged internally, don't propagate).
// This ensures notifications never block or crash the main workflow.
func (h *Handler) dispatch(n Notification) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.sendNotification(n)
	}()

	select {
	case <-done:
		// Notification sent successfully
	case <-ctx.Done():
		// Timeout - notification took too long, but we don't block
	}
}

// sendNotification sends the notification based on configured type
func (h *Handler) sendNotification(n Notification) {
	switch h.config.Type {
	case OutputSound:
		_ = h.sender.SendSound(h.config.SoundFile)
	case OutputVisual:
		_ = h.sender.SendVisual(n)
	case OutputBoth:
		_ = h.sender.SendVisual(n)
		_ = h.sender.SendSound(h.config.SoundFile)
	}
}

// OnCommandComplete is called when an autospec command finishes.
//
// Two-level filtering:
//  1. on_long_running check: if enabled and duration < threshold, skip notification
//  2. on_command_complete check: must be enabled to send any notification
//
// Threshold of 0 or negative means "always notify" (no duration filter).
// This allows users to only be notified for long operations.
//
// TEST COVERAGE BLOCKED: isEnabled() requires TTY; dispatch() calls OS notification APIs.
func (h *Handler) OnCommandComplete(commandName string, success bool, duration time.Duration) {
	if !h.isEnabled() {
		return
	}

	// Check on_long_running first - if enabled and duration is below threshold, skip
	if h.config.OnLongRunning {
		threshold := h.config.LongRunningThreshold
		// 0 or negative threshold means "always notify"
		if threshold > 0 && duration < threshold {
			return
		}
	}

	// Only notify if on_command_complete is enabled
	if !h.config.OnCommandComplete {
		return
	}

	notifType := TypeSuccess
	status := "completed successfully"
	if !success {
		notifType = TypeFailure
		status = "failed"
	}

	n := NewNotification(
		"autospec",
		fmt.Sprintf("Command '%s' %s (%s)", commandName, status, formatDuration(duration)),
		notifType,
	)
	h.dispatch(n)
}

// OnStageComplete is called when a workflow stage finishes.
// It sends a notification if the on_stage_complete hook is enabled.
//
// TEST COVERAGE BLOCKED: isEnabled() requires TTY; dispatch() calls OS notification APIs.
func (h *Handler) OnStageComplete(stageName string, success bool) {
	if !h.isEnabled() {
		return
	}

	if !h.config.OnStageComplete {
		return
	}

	notifType := TypeSuccess
	status := "completed"
	if !success {
		notifType = TypeFailure
		status = "failed"
	}

	n := NewNotification(
		"autospec",
		fmt.Sprintf("Stage '%s' %s", stageName, status),
		notifType,
	)
	h.dispatch(n)
}

// OnError is called when a command or stage fails.
// It sends a notification if the on_error hook is enabled.
// This is separate from OnCommandComplete/OnStageComplete to allow
// error-only notifications without command/stage completion notifications.
//
// TEST COVERAGE BLOCKED: isEnabled() requires TTY; dispatch() calls OS notification APIs.
func (h *Handler) OnError(commandName string, err error) {
	if !h.isEnabled() {
		return
	}

	if !h.config.OnError {
		return
	}

	errMsg := "unknown error"
	if err != nil {
		errMsg = err.Error()
	}

	n := NewNotification(
		"autospec",
		fmt.Sprintf("Error in '%s': %s", commandName, errMsg),
		TypeFailure,
	)
	h.dispatch(n)
}

// formatDuration formats a duration for display in notifications
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}
