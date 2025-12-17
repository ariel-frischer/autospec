package notify

import (
	"testing"
	"time"
)

func TestNotificationTypeConstants(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		notifType NotificationType
		expected  string
	}{
		"TypeSuccess": {
			notifType: TypeSuccess,
			expected:  "success",
		},
		"TypeFailure": {
			notifType: TypeFailure,
			expected:  "failure",
		},
		"TypeInfo": {
			notifType: TypeInfo,
			expected:  "info",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if string(tt.notifType) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.notifType)
			}
		})
	}
}

func TestOutputTypeConstants(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		outputType OutputType
		expected   string
	}{
		"OutputSound": {
			outputType: OutputSound,
			expected:   "sound",
		},
		"OutputVisual": {
			outputType: OutputVisual,
			expected:   "visual",
		},
		"OutputBoth": {
			outputType: OutputBoth,
			expected:   "both",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if string(tt.outputType) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.outputType)
			}
		})
	}
}

func TestValidOutputType(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		input    string
		expected bool
	}{
		"valid sound": {
			input:    "sound",
			expected: true,
		},
		"valid visual": {
			input:    "visual",
			expected: true,
		},
		"valid both": {
			input:    "both",
			expected: true,
		},
		"invalid empty": {
			input:    "",
			expected: false,
		},
		"invalid random": {
			input:    "invalid",
			expected: false,
		},
		"invalid uppercase": {
			input:    "SOUND",
			expected: false,
		},
		"invalid mixed case": {
			input:    "Both",
			expected: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := ValidOutputType(tt.input)
			if result != tt.expected {
				t.Errorf("ValidOutputType(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()
	config := DefaultConfig()

	tests := map[string]struct {
		field    string
		got      interface{}
		expected interface{}
	}{
		"Enabled defaults to false": {
			field:    "Enabled",
			got:      config.Enabled,
			expected: false,
		},
		"Type defaults to both": {
			field:    "Type",
			got:      config.Type,
			expected: OutputBoth,
		},
		"SoundFile defaults to empty": {
			field:    "SoundFile",
			got:      config.SoundFile,
			expected: "",
		},
		"OnCommandComplete defaults to true": {
			field:    "OnCommandComplete",
			got:      config.OnCommandComplete,
			expected: true,
		},
		"OnStageComplete defaults to false": {
			field:    "OnStageComplete",
			got:      config.OnStageComplete,
			expected: false,
		},
		"OnError defaults to true": {
			field:    "OnError",
			got:      config.OnError,
			expected: true,
		},
		"OnLongRunning defaults to false": {
			field:    "OnLongRunning",
			got:      config.OnLongRunning,
			expected: false,
		},
		"LongRunningThreshold defaults to 2m": {
			field:    "LongRunningThreshold",
			got:      config.LongRunningThreshold,
			expected: 2 * time.Minute,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s: got %v, expected %v", tt.field, tt.got, tt.expected)
			}
		})
	}
}

func TestNotificationConfigStruct(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		config   NotificationConfig
		checkFn  func(NotificationConfig) bool
		expected bool
	}{
		"all fields configurable": {
			config: NotificationConfig{
				Enabled:              true,
				Type:                 OutputSound,
				SoundFile:            "/custom/sound.wav",
				OnCommandComplete:    false,
				OnStageComplete:      true,
				OnError:              false,
				OnLongRunning:        true,
				LongRunningThreshold: 60 * time.Second,
			},
			checkFn: func(c NotificationConfig) bool {
				return c.Enabled == true &&
					c.Type == OutputSound &&
					c.SoundFile == "/custom/sound.wav" &&
					c.OnCommandComplete == false &&
					c.OnStageComplete == true &&
					c.OnError == false &&
					c.OnLongRunning == true &&
					c.LongRunningThreshold == 60*time.Second
			},
			expected: true,
		},
		"zero value config": {
			config: NotificationConfig{},
			checkFn: func(c NotificationConfig) bool {
				return c.Enabled == false &&
					c.Type == "" &&
					c.SoundFile == "" &&
					c.OnCommandComplete == false &&
					c.OnStageComplete == false &&
					c.OnError == false &&
					c.OnLongRunning == false &&
					c.LongRunningThreshold == 0
			},
			expected: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := tt.checkFn(tt.config)
			if result != tt.expected {
				t.Errorf("config check failed: got %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestNewNotification(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		title    string
		message  string
		notifTyp NotificationType
	}{
		"success notification": {
			title:    "autospec",
			message:  "Command completed successfully",
			notifTyp: TypeSuccess,
		},
		"failure notification": {
			title:    "autospec",
			message:  "Command failed",
			notifTyp: TypeFailure,
		},
		"info notification": {
			title:    "autospec",
			message:  "Stage 'specify' complete",
			notifTyp: TypeInfo,
		},
		"empty strings": {
			title:    "",
			message:  "",
			notifTyp: TypeInfo,
		},
		"unicode content": {
			title:    "autospec \U0001F680",
			message:  "Build complete! \u2714",
			notifTyp: TypeSuccess,
		},
		"long message": {
			title:    "autospec",
			message:  "This is a very long notification message that contains a lot of text to test how the system handles lengthy content in notification messages",
			notifTyp: TypeInfo,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			n := NewNotification(tt.title, tt.message, tt.notifTyp)

			if n.Title != tt.title {
				t.Errorf("Title: got %q, expected %q", n.Title, tt.title)
			}
			if n.Message != tt.message {
				t.Errorf("Message: got %q, expected %q", n.Message, tt.message)
			}
			if n.NotificationType != tt.notifTyp {
				t.Errorf("NotificationType: got %q, expected %q", n.NotificationType, tt.notifTyp)
			}
		})
	}
}

func TestNotificationStruct(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		notification Notification
		checkFn      func(Notification) bool
		expected     bool
	}{
		"direct struct creation": {
			notification: Notification{
				Title:            "Test",
				Message:          "Test message",
				NotificationType: TypeSuccess,
			},
			checkFn: func(n Notification) bool {
				return n.Title == "Test" && n.Message == "Test message" && n.NotificationType == TypeSuccess
			},
			expected: true,
		},
		"zero value notification": {
			notification: Notification{},
			checkFn: func(n Notification) bool {
				return n.Title == "" && n.Message == "" && n.NotificationType == ""
			},
			expected: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := tt.checkFn(tt.notification)
			if result != tt.expected {
				t.Errorf("notification check failed: got %v, expected %v", result, tt.expected)
			}
		})
	}
}
