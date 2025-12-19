// Package workflow provides progress controller for stage display management.
// Related: internal/workflow/executor.go, internal/progress/display.go
// Tags: workflow, progress, display, separation-of-concerns
package workflow

import (
	"fmt"

	"github.com/ariel-frischer/autospec/internal/progress"
)

// ProgressController manages stage progress display operations.
// It wraps a ProgressDisplay instance and provides nil-safe methods
// that become no-ops when the display is nil.
//
// Design rationale: Extracted from Executor to separate display concerns
// from command execution. This enables independent testing of progress
// display without requiring actual command execution.
type ProgressController struct {
	display *progress.ProgressDisplay
}

// NewProgressController creates a new ProgressController with the given display.
// The display may be nil, in which case all methods become no-ops.
func NewProgressController(display *progress.ProgressDisplay) *ProgressController {
	return &ProgressController{
		display: display,
	}
}

// StartStage begins displaying progress for a stage.
// Returns nil if display is nil (no-op for tests without progress display).
// Errors are wrapped with context describing the operation.
func (p *ProgressController) StartStage(info progress.StageInfo) error {
	if p.display == nil {
		return nil
	}

	if err := p.display.StartStage(info); err != nil {
		return fmt.Errorf("starting stage display: %w", err)
	}
	return nil
}

// CompleteStage marks a stage as completed in the progress display.
// Returns nil if display is nil (no-op for tests without progress display).
// Errors are wrapped with context describing the operation.
func (p *ProgressController) CompleteStage(info progress.StageInfo) error {
	if p.display == nil {
		return nil
	}

	if err := p.display.CompleteStage(info); err != nil {
		return fmt.Errorf("completing stage display: %w", err)
	}
	return nil
}

// FailStage marks a stage as failed with an error in the progress display.
// This method does not return an error as failure display is best-effort.
// No-op if display is nil (safe for tests without progress display).
func (p *ProgressController) FailStage(info progress.StageInfo, err error) {
	if p.display == nil {
		return
	}

	// FailStage returns error but we treat display errors as non-fatal
	_ = p.display.FailStage(info, err)
}

// StopSpinner stops the spinner without showing completion/failure status.
// This is useful when pausing progress display during interactive output.
// No-op if display is nil.
func (p *ProgressController) StopSpinner() {
	if p.display == nil {
		return
	}
	p.display.StopSpinner()
}

// HasDisplay returns true if a progress display is configured.
// This can be used to conditionally log messages when no display is available.
func (p *ProgressController) HasDisplay() bool {
	return p.display != nil
}
