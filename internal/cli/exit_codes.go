package cli

import (
	"github.com/ariel-frischer/autospec/internal/cli/shared"
)

// Exit codes for the autospec CLI (re-exported from shared)
// These codes support programmatic composition and CI/CD integration
const (
	// ExitSuccess indicates successful command execution
	ExitSuccess = shared.ExitSuccess

	// ExitValidationFailed indicates validation failed (retryable)
	ExitValidationFailed = shared.ExitValidationFailed

	// ExitRetryExhausted indicates retry limit was exhausted
	ExitRetryExhausted = shared.ExitRetryLimitReached

	// ExitInvalidArguments indicates invalid command arguments
	ExitInvalidArguments = shared.ExitInvalidArguments

	// ExitMissingDependencies indicates required dependencies are missing
	ExitMissingDependencies = shared.ExitMissingDependency

	// ExitTimeout indicates command execution timed out
	ExitTimeout = shared.ExitTimeout
)

// NewExitError creates a new exit error with the given code (re-exported from shared).
func NewExitError(code int) error {
	return shared.NewExitError(code)
}

// ExitCode returns the exit code from an error (re-exported from shared).
func ExitCode(err error) int {
	return shared.ExitCode(err)
}
