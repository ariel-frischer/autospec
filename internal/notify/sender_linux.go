//go:build linux

package notify

import (
	"os"
	"os/exec"
)

// linuxSender implements Sender for Linux using notify-send and paplay
type linuxSender struct {
	visualAvailable bool
	soundAvailable  bool
}

// newLinuxSender creates a new Linux notification sender
func newLinuxSender() Sender {
	return &linuxSender{
		visualAvailable: toolAvailable("notify-send") && hasDisplay(),
		soundAvailable:  toolAvailable("paplay"),
	}
}

// newDarwinSender returns a no-op sender on linux
func newDarwinSender() Sender {
	return &noopSender{}
}

// newWindowsSender returns a no-op sender on linux
func newWindowsSender() Sender {
	return &noopSender{}
}

// hasDisplay checks if a display environment is available
//
// TEST COVERAGE BLOCKED: Checks env vars set by display server; not available in CI.
func hasDisplay() bool {
	// Check for X11 display
	if os.Getenv("DISPLAY") != "" {
		return true
	}
	// Check for Wayland display
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return true
	}
	return false
}

// SendVisual sends a visual notification using notify-send
//
// TEST COVERAGE BLOCKED: Executes notify-send; requires DISPLAY/WAYLAND_DISPLAY env.
func (s *linuxSender) SendVisual(n Notification) error {
	if !s.visualAvailable {
		return nil // graceful degradation
	}

	// Map notification type to urgency
	urgency := "normal"
	if n.NotificationType == TypeFailure {
		urgency = "critical"
	}

	cmd := exec.Command("notify-send", "-u", urgency, n.Title, n.Message)
	return cmd.Run()
}

// SendSound plays a sound using paplay
//
// TEST COVERAGE BLOCKED: Executes paplay; requires audio subsystem.
func (s *linuxSender) SendSound(soundFile string) error {
	if !s.soundAvailable {
		return nil // graceful degradation
	}

	// Validate custom sound file if provided
	validatedFile := ValidateSoundFile(soundFile)

	// No default sound on Linux, skip if no valid custom file
	if validatedFile == "" {
		return nil // no sound to play, skip silently
	}

	cmd := exec.Command("paplay", validatedFile)
	return cmd.Run()
}

// VisualAvailable returns true if notify-send is available and display is present
func (s *linuxSender) VisualAvailable() bool {
	return s.visualAvailable
}

// SoundAvailable returns true if paplay is available
func (s *linuxSender) SoundAvailable() bool {
	return s.soundAvailable
}
