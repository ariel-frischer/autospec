package errors

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormatError(t *testing.T) {
	t.Run("nil error returns empty string", func(t *testing.T) {
		t.Parallel()
		result := FormatError(nil)
		if result != "" {
			t.Errorf("Expected empty string, got %q", result)
		}
	})

	t.Run("basic error formatting", func(t *testing.T) {
		t.Parallel()
		err := &CLIError{
			Category: Argument,
			Message:  "test message",
		}

		result := FormatError(err)

		if !strings.Contains(result, "Argument Error") {
			t.Error("Expected output to contain 'Argument Error'")
		}
		if !strings.Contains(result, "test message") {
			t.Error("Expected output to contain 'test message'")
		}
	})

	t.Run("error with usage", func(t *testing.T) {
		t.Parallel()
		err := &CLIError{
			Category: Argument,
			Message:  "missing arg",
			Usage:    "cmd <arg>",
		}

		result := FormatError(err)

		if !strings.Contains(result, "Usage:") {
			t.Error("Expected output to contain 'Usage:'")
		}
		if !strings.Contains(result, "cmd <arg>") {
			t.Error("Expected output to contain usage string")
		}
	})

	t.Run("error with remediation", func(t *testing.T) {
		t.Parallel()
		err := &CLIError{
			Category:    Argument,
			Message:     "error",
			Remediation: []string{"step 1", "step 2"},
		}

		result := FormatError(err)

		if !strings.Contains(result, "To fix this:") {
			t.Error("Expected output to contain 'To fix this:'")
		}
		if !strings.Contains(result, "step 1") {
			t.Error("Expected output to contain 'step 1'")
		}
		if !strings.Contains(result, "step 2") {
			t.Error("Expected output to contain 'step 2'")
		}
	})
}

func TestFormatErrorPlain(t *testing.T) {
	t.Run("nil error returns empty string", func(t *testing.T) {
		t.Parallel()
		result := FormatErrorPlain(nil)
		if result != "" {
			t.Errorf("Expected empty string, got %q", result)
		}
	})

	t.Run("basic formatting without colors", func(t *testing.T) {
		t.Parallel()
		err := &CLIError{
			Category:    Configuration,
			Message:     "config error",
			Remediation: []string{"fix it"},
		}

		result := FormatErrorPlain(err)

		// Should contain text without ANSI escape codes
		if !strings.Contains(result, "Configuration Error") {
			t.Error("Expected output to contain 'Configuration Error'")
		}
		if !strings.Contains(result, "config error") {
			t.Error("Expected output to contain 'config error'")
		}
	})
}

func TestPrintError(t *testing.T) {
	// PrintError writes to stderr, but we can't easily capture that
	// This just verifies it doesn't panic
	err := &CLIError{
		Category: Runtime,
		Message:  "test",
	}
	PrintError(err) // Should not panic
	PrintError(nil) // Should not panic
}

func TestFprintError(t *testing.T) {
	t.Run("nil error does nothing", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		FprintError(&buf, nil)

		if buf.Len() != 0 {
			t.Errorf("Expected no output for nil error, got %q", buf.String())
		}
	})

	t.Run("writes error to buffer", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		err := &CLIError{
			Category: Prerequisite,
			Message:  "missing file",
		}

		FprintError(&buf, err)

		if !strings.Contains(buf.String(), "missing file") {
			t.Error("Expected buffer to contain error message")
		}
	})
}

func TestFormatSimpleError(t *testing.T) {
	t.Run("nil error returns empty string", func(t *testing.T) {
		t.Parallel()
		result := FormatSimpleError(nil, Runtime)
		if result != "" {
			t.Errorf("Expected empty string, got %q", result)
		}
	})

	t.Run("formats regular error", func(t *testing.T) {
		t.Parallel()
		err := &testError{}
		result := FormatSimpleError(err, Runtime)

		if !strings.Contains(result, "Runtime Error") {
			t.Error("Expected output to contain 'Runtime Error'")
		}
		if !strings.Contains(result, "test error") {
			t.Error("Expected output to contain the error message")
		}
	})
}
