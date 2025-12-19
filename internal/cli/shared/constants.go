// Package shared provides constants and types used across CLI subpackages.
// This package has no dependencies on other CLI packages to avoid circular imports.
package shared

import "fmt"

// Command group IDs for organizing help output
const (
	GroupGettingStarted = "getting-started"
	GroupWorkflows      = "workflows"
	GroupCoreStages     = "core-stages"
	GroupOptionalStages = "optional-stages"
	GroupConfiguration  = "configuration"
	GroupInternal       = "internal"
)

// Exit codes for CLI commands
const (
	ExitSuccess           = 0
	ExitValidationFailed  = 1
	ExitRetryLimitReached = 2
	ExitInvalidArguments  = 3
	ExitMissingDependency = 4
	ExitTimeout           = 5
)

// exitError is a custom error type that carries an exit code.
type exitError struct {
	code int
}

func (e *exitError) Error() string {
	return fmt.Sprintf("exit code %d", e.code)
}

// NewExitError creates a new exit error with the given code.
func NewExitError(code int) error {
	return &exitError{code: code}
}

// ExitCode returns the exit code from an error.
func ExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	if e, ok := err.(*exitError); ok {
		return e.code
	}
	return ExitValidationFailed
}

// SpecMetadata is an interface for spec metadata that can format info.
type SpecMetadata interface {
	FormatInfo() string
}

// PrintSpecInfo prints the spec metadata if available.
func PrintSpecInfo(metadata SpecMetadata) {
	if metadata != nil {
		fmt.Println(metadata.FormatInfo())
	}
}
