package errors

import (
	"testing"
)

func TestErrorCategoryString(t *testing.T) {
	tests := map[string]struct {
		category ErrorCategory
		expected string
	}{
		"Argument":       {category: Argument, expected: "Argument Error"},
		"Configuration":  {category: Configuration, expected: "Configuration Error"},
		"Prerequisite":   {category: Prerequisite, expected: "Prerequisite Error"},
		"Runtime":        {category: Runtime, expected: "Runtime Error"},
		"Unknown":        {category: ErrorCategory(99), expected: "Error"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result := test.category.String()
			if result != test.expected {
				t.Errorf("Expected %q, got %q", test.expected, result)
			}
		})
	}
}

func TestCLIErrorError(t *testing.T) {
	err := &CLIError{
		Category: Argument,
		Message:  "test error message",
	}

	if err.Error() != "test error message" {
		t.Errorf("Expected 'test error message', got %q", err.Error())
	}
}

func TestNewArgumentError(t *testing.T) {
	err := NewArgumentError("missing argument", "provide the argument", "see --help")

	if err.Category != Argument {
		t.Errorf("Expected Argument category, got %v", err.Category)
	}
	if err.Message != "missing argument" {
		t.Errorf("Expected message 'missing argument', got %q", err.Message)
	}
	if len(err.Remediation) != 2 {
		t.Errorf("Expected 2 remediation steps, got %d", len(err.Remediation))
	}
}

func TestNewArgumentErrorWithUsage(t *testing.T) {
	err := NewArgumentErrorWithUsage("invalid arg", "command <arg>", "use correct syntax")

	if err.Category != Argument {
		t.Errorf("Expected Argument category, got %v", err.Category)
	}
	if err.Usage != "command <arg>" {
		t.Errorf("Expected usage 'command <arg>', got %q", err.Usage)
	}
}

func TestNewConfigError(t *testing.T) {
	err := NewConfigError("config error", "check config file")

	if err.Category != Configuration {
		t.Errorf("Expected Configuration category, got %v", err.Category)
	}
}

func TestNewPrerequisiteError(t *testing.T) {
	err := NewPrerequisiteError("missing file", "create the file")

	if err.Category != Prerequisite {
		t.Errorf("Expected Prerequisite category, got %v", err.Category)
	}
}

func TestNewRuntimeError(t *testing.T) {
	err := NewRuntimeError("execution failed", "try again")

	if err.Category != Runtime {
		t.Errorf("Expected Runtime category, got %v", err.Category)
	}
}

func TestWrap(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		t.Parallel()
		result := Wrap(nil, Runtime)
		if result != nil {
			t.Error("Expected nil for nil input")
		}
	})

	t.Run("wraps error with category", func(t *testing.T) {
		t.Parallel()
		original := &CLIError{Message: "original error"}
		result := Wrap(original, Runtime, "fix it")

		if result.Category != Runtime {
			t.Errorf("Expected Runtime category, got %v", result.Category)
		}
		if len(result.Remediation) != 1 {
			t.Errorf("Expected 1 remediation step, got %d", len(result.Remediation))
		}
	})
}

func TestWrapWithMessage(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		t.Parallel()
		result := WrapWithMessage(nil, Runtime, "wrapper")
		if result != nil {
			t.Error("Expected nil for nil input")
		}
	})

	t.Run("wraps error with message", func(t *testing.T) {
		t.Parallel()
		original := &CLIError{Message: "inner"}
		result := WrapWithMessage(original, Runtime, "outer")

		if result.Category != Runtime {
			t.Errorf("Expected Runtime category, got %v", result.Category)
		}
		// Message should contain both outer and inner
		if result.Message != "outer: inner" {
			t.Errorf("Expected 'outer: inner', got %q", result.Message)
		}
	})
}

func TestIsCLIError(t *testing.T) {
	t.Run("returns true for CLIError", func(t *testing.T) {
		t.Parallel()
		err := NewArgumentError("test")
		if !IsCLIError(err) {
			t.Error("Expected true for CLIError")
		}
	})

	t.Run("returns false for other errors", func(t *testing.T) {
		t.Parallel()
		err := &testError{}
		if IsCLIError(err) {
			t.Error("Expected false for non-CLIError")
		}
	})
}

func TestAsCLIError(t *testing.T) {
	t.Run("returns CLIError for CLIError", func(t *testing.T) {
		t.Parallel()
		original := NewArgumentError("test")
		result := AsCLIError(original)
		if result != original {
			t.Error("Expected same CLIError")
		}
	})

	t.Run("returns nil for other errors", func(t *testing.T) {
		t.Parallel()
		err := &testError{}
		result := AsCLIError(err)
		if result != nil {
			t.Error("Expected nil for non-CLIError")
		}
	})
}

// testError is a helper for testing non-CLIError errors
type testError struct{}

func (e *testError) Error() string { return "test error" }
