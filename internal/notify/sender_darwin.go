//go:build darwin

package notify

import (
	"fmt"
	"os/exec"
)

const (
	// DefaultMacOSSound is the default notification sound on macOS
	DefaultMacOSSound = "/System/Library/Sounds/Glass.aiff"
)

// darwinSender implements Sender for macOS using osascript and afplay
type darwinSender struct {
	visualAvailable bool
	soundAvailable  bool
}

// newDarwinSender creates a new macOS notification sender
func newDarwinSender() Sender {
	return &darwinSender{
		visualAvailable: toolAvailable("osascript"),
		soundAvailable:  toolAvailable("afplay"),
	}
}

// newLinuxSender returns a no-op sender on darwin
func newLinuxSender() Sender {
	return &noopSender{}
}

// newWindowsSender returns a no-op sender on darwin
func newWindowsSender() Sender {
	return &noopSender{}
}

// SendVisual sends a visual notification using osascript
//
// TEST COVERAGE BLOCKED: Executes osascript; requires macOS.
func (s *darwinSender) SendVisual(n Notification) error {
	if !s.visualAvailable {
		return nil // graceful degradation
	}

	// Build AppleScript command for display notification
	script := fmt.Sprintf(`display notification %q with title %q`, n.Message, n.Title)

	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// SendSound plays a sound using afplay
//
// TEST COVERAGE BLOCKED: Executes afplay; requires macOS audio subsystem.
func (s *darwinSender) SendSound(soundFile string) error {
	if !s.soundAvailable {
		return nil // graceful degradation
	}

	// Validate custom sound file if provided
	validatedFile := ValidateSoundFile(soundFile)

	// Use default sound if no valid custom file
	if validatedFile == "" {
		validatedFile = DefaultMacOSSound
	}

	cmd := exec.Command("afplay", validatedFile)
	return cmd.Run()
}

// VisualAvailable returns true if osascript is available
func (s *darwinSender) VisualAvailable() bool {
	return s.visualAvailable
}

// SoundAvailable returns true if afplay is available
func (s *darwinSender) SoundAvailable() bool {
	return s.soundAvailable
}
