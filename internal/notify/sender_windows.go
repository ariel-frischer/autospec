//go:build windows

package notify

import (
	"fmt"
	"os/exec"
)

// windowsSender implements Sender for Windows using PowerShell
type windowsSender struct {
	visualAvailable bool
	soundAvailable  bool
}

// newWindowsSender creates a new Windows notification sender
func newWindowsSender() Sender {
	return &windowsSender{
		visualAvailable: toolAvailable("powershell"),
		soundAvailable:  toolAvailable("powershell"),
	}
}

// newDarwinSender returns a no-op sender on windows
func newDarwinSender() Sender {
	return &noopSender{}
}

// newLinuxSender returns a no-op sender on windows
func newLinuxSender() Sender {
	return &noopSender{}
}

// SendVisual sends a toast notification using PowerShell
//
// TEST COVERAGE BLOCKED: Executes PowerShell; requires Windows.
func (s *windowsSender) SendVisual(n Notification) error {
	if !s.visualAvailable {
		return nil // graceful degradation
	}

	// PowerShell script to show a toast notification using BurntToast or built-in method
	script := fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$textNodes = $template.GetElementsByTagName('text')
$textNodes.Item(0).AppendChild($template.CreateTextNode('%s')) | Out-Null
$textNodes.Item(1).AppendChild($template.CreateTextNode('%s')) | Out-Null
$toast = [Windows.UI.Notifications.ToastNotification]::new($template)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('autospec').Show($toast)
`, escapeForPowerShell(n.Title), escapeForPowerShell(n.Message))

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-NoProfile", "-Command", script)
	return cmd.Run()
}

// SendSound plays a sound using PowerShell
//
// TEST COVERAGE BLOCKED: Executes PowerShell; requires Windows audio.
func (s *windowsSender) SendSound(soundFile string) error {
	if !s.soundAvailable {
		return nil // graceful degradation
	}

	// Validate custom sound file if provided
	validatedFile := ValidateSoundFile(soundFile)

	var script string
	if validatedFile == "" {
		// Default system beep
		script = "[Console]::Beep(800, 200)"
	} else {
		// Play custom sound file
		script = fmt.Sprintf(`
$player = New-Object System.Media.SoundPlayer
$player.SoundLocation = '%s'
$player.PlaySync()
`, escapeForPowerShell(validatedFile))
	}

	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-NoProfile", "-Command", script)
	return cmd.Run()
}

// VisualAvailable returns true if PowerShell is available
func (s *windowsSender) VisualAvailable() bool {
	return s.visualAvailable
}

// SoundAvailable returns true if PowerShell is available
func (s *windowsSender) SoundAvailable() bool {
	return s.soundAvailable
}

// escapeForPowerShell escapes special characters for PowerShell strings
func escapeForPowerShell(s string) string {
	// Escape single quotes by doubling them
	result := ""
	for _, c := range s {
		if c == '\'' {
			result += "''"
		} else if c == '`' || c == '$' {
			result += "`" + string(c)
		} else {
			result += string(c)
		}
	}
	return result
}
