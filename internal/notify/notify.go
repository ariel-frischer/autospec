package notify

import "time"

// NotificationType represents the type of notification event
type NotificationType string

const (
	// TypeSuccess indicates a successful operation
	TypeSuccess NotificationType = "success"
	// TypeFailure indicates a failed operation
	TypeFailure NotificationType = "failure"
	// TypeInfo indicates an informational notification
	TypeInfo NotificationType = "info"
)

// OutputType represents the notification output type
type OutputType string

const (
	// OutputSound sends only an audible notification
	OutputSound OutputType = "sound"
	// OutputVisual sends only a visual notification
	OutputVisual OutputType = "visual"
	// OutputBoth sends both sound and visual notifications
	OutputBoth OutputType = "both"
)

// ValidOutputType checks if the given string is a valid output type
func ValidOutputType(s string) bool {
	switch OutputType(s) {
	case OutputSound, OutputVisual, OutputBoth:
		return true
	default:
		return false
	}
}

// NotificationConfig holds user preferences for notification behavior.
// Configuration is loaded from the config hierarchy (env > project > user > defaults).
type NotificationConfig struct {
	// Enabled is the master switch for all notifications (default: false, opt-in)
	Enabled bool `koanf:"enabled" yaml:"enabled" json:"enabled"`

	// Type specifies the notification output type: sound, visual, or both (default: both)
	Type OutputType `koanf:"type" yaml:"type" json:"type"`

	// SoundFile is an optional custom sound file path
	SoundFile string `koanf:"sound_file" yaml:"sound_file" json:"sound_file"`

	// OnCommandComplete notifies when any command finishes (default: true when enabled)
	OnCommandComplete bool `koanf:"on_command_complete" yaml:"on_command_complete" json:"on_command_complete"`

	// OnStageComplete notifies after each workflow stage (default: false)
	OnStageComplete bool `koanf:"on_stage_complete" yaml:"on_stage_complete" json:"on_stage_complete"`

	// OnError notifies on command/stage failure (default: true when enabled)
	OnError bool `koanf:"on_error" yaml:"on_error" json:"on_error"`

	// OnLongRunning notifies only if duration exceeds threshold (default: false)
	OnLongRunning bool `koanf:"on_long_running" yaml:"on_long_running" json:"on_long_running"`

	// LongRunningThreshold is the threshold for on_long_running hook (default: 30s)
	// A value of 0 or negative means "always notify"
	LongRunningThreshold time.Duration `koanf:"long_running_threshold" yaml:"long_running_threshold" json:"long_running_threshold"`

	// OnInteractiveSession notifies when an interactive stage is about to begin (default: true when enabled)
	// This alerts users to return to the terminal after automated stages complete.
	OnInteractiveSession bool `koanf:"on_interactive_session" yaml:"on_interactive_session" json:"on_interactive_session"`
}

// DefaultConfig returns a NotificationConfig with default values
func DefaultConfig() NotificationConfig {
	return NotificationConfig{
		Enabled:              false,
		Type:                 OutputBoth,
		SoundFile:            "",
		OnCommandComplete:    true,
		OnStageComplete:      false,
		OnError:              true,
		OnLongRunning:        false,
		LongRunningThreshold: 2 * time.Minute,
		OnInteractiveSession: true,
	}
}

// Notification represents a single notification event to dispatch
type Notification struct {
	// Title is the notification title (e.g., "autospec")
	Title string

	// Message is the notification body text
	Message string

	// NotificationType indicates the event type: success, failure, or info
	NotificationType NotificationType
}

// NewNotification creates a new Notification with the given parameters
func NewNotification(title, message string, notificationType NotificationType) Notification {
	return Notification{
		Title:            title,
		Message:          message,
		NotificationType: notificationType,
	}
}
