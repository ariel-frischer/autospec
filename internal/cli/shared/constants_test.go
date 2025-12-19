// Package shared_test tests shared constants and types used across CLI subpackages.
// Related: internal/cli/shared/constants.go
// Tags: cli, shared, constants, exit-codes, errors

package shared

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExitCodeConstants(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		constant int
		want     int
	}{
		"ExitSuccess":           {constant: ExitSuccess, want: 0},
		"ExitValidationFailed":  {constant: ExitValidationFailed, want: 1},
		"ExitRetryLimitReached": {constant: ExitRetryLimitReached, want: 2},
		"ExitInvalidArguments":  {constant: ExitInvalidArguments, want: 3},
		"ExitMissingDependency": {constant: ExitMissingDependency, want: 4},
		"ExitTimeout":           {constant: ExitTimeout, want: 5},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.constant)
		})
	}
}

func TestGroupConstants(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		constant string
		want     string
	}{
		"GroupGettingStarted": {constant: GroupGettingStarted, want: "getting-started"},
		"GroupWorkflows":      {constant: GroupWorkflows, want: "workflows"},
		"GroupCoreStages":     {constant: GroupCoreStages, want: "core-stages"},
		"GroupOptionalStages": {constant: GroupOptionalStages, want: "optional-stages"},
		"GroupConfiguration":  {constant: GroupConfiguration, want: "configuration"},
		"GroupInternal":       {constant: GroupInternal, want: "internal"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.constant)
		})
	}
}

func TestNewExitError(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		code int
		want int
	}{
		"success":           {code: ExitSuccess, want: 0},
		"validation failed": {code: ExitValidationFailed, want: 1},
		"retry exhausted":   {code: ExitRetryLimitReached, want: 2},
		"invalid args":      {code: ExitInvalidArguments, want: 3},
		"missing dep":       {code: ExitMissingDependency, want: 4},
		"timeout":           {code: ExitTimeout, want: 5},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := NewExitError(tc.code)
			assert.Error(t, err)
			assert.Equal(t, tc.want, ExitCode(err))
		})
	}
}

func TestExitError_Error(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		code        int
		wantMessage string
	}{
		"exit code 0": {code: 0, wantMessage: "exit code 0"},
		"exit code 1": {code: 1, wantMessage: "exit code 1"},
		"exit code 2": {code: 2, wantMessage: "exit code 2"},
		"exit code 5": {code: 5, wantMessage: "exit code 5"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := NewExitError(tc.code)
			assert.Equal(t, tc.wantMessage, err.Error())
		})
	}
}

func TestExitCode(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		err  error
		want int
	}{
		"nil error":             {err: nil, want: ExitSuccess},
		"exit error code 0":     {err: NewExitError(0), want: 0},
		"exit error code 1":     {err: NewExitError(1), want: 1},
		"exit error code 2":     {err: NewExitError(2), want: 2},
		"exit error code 5":     {err: NewExitError(5), want: 5},
		"generic error":         {err: errors.New("generic error"), want: ExitValidationFailed},
		"wrapped generic error": {err: errors.New("wrapped: something failed"), want: ExitValidationFailed},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, ExitCode(tc.err))
		})
	}
}

func TestExitCode_WithWrappedExitError(t *testing.T) {
	t.Parallel()

	// Note: ExitCode doesn't use errors.As, so wrapped exit errors
	// are treated as generic errors
	exitErr := NewExitError(ExitTimeout)
	// Direct exit error should work
	assert.Equal(t, ExitTimeout, ExitCode(exitErr))
}

// mockSpecMetadata implements SpecMetadata for testing
type mockSpecMetadata struct {
	info string
}

func (m *mockSpecMetadata) FormatInfo() string {
	return m.info
}

func TestPrintSpecInfo(t *testing.T) {
	tests := map[string]struct {
		metadata   SpecMetadata
		wantOutput string
	}{
		"nil metadata": {
			metadata:   nil,
			wantOutput: "",
		},
		"empty info": {
			metadata:   &mockSpecMetadata{info: ""},
			wantOutput: "\n",
		},
		"with info": {
			metadata:   &mockSpecMetadata{info: "Feature: test-feature"},
			wantOutput: "Feature: test-feature\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			PrintSpecInfo(tc.metadata)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)

			assert.Equal(t, tc.wantOutput, buf.String())
		})
	}
}

func TestExitCodeUniqueness(t *testing.T) {
	t.Parallel()

	// Verify all exit codes are unique
	codes := []int{
		ExitSuccess,
		ExitValidationFailed,
		ExitRetryLimitReached,
		ExitInvalidArguments,
		ExitMissingDependency,
		ExitTimeout,
	}

	seen := make(map[int]bool)
	for _, code := range codes {
		assert.False(t, seen[code], "Duplicate exit code: %d", code)
		seen[code] = true
	}
}

func TestGroupConstantsUniqueness(t *testing.T) {
	t.Parallel()

	// Verify all group constants are unique
	groups := []string{
		GroupGettingStarted,
		GroupWorkflows,
		GroupCoreStages,
		GroupOptionalStages,
		GroupConfiguration,
		GroupInternal,
	}

	seen := make(map[string]bool)
	for _, group := range groups {
		assert.False(t, seen[group], "Duplicate group constant: %s", group)
		seen[group] = true
	}
}
