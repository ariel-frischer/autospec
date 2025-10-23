package progress

import (
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
)

// ProgressDisplay orchestrates the display of progress indicators
type ProgressDisplay struct {
	capabilities TerminalCapabilities
	currentPhase *PhaseInfo
	spinner      *spinner.Spinner
	symbols      ProgressSymbols
}

// NewProgressDisplay creates a new progress display with the given terminal capabilities
func NewProgressDisplay(caps TerminalCapabilities) *ProgressDisplay {
	return &ProgressDisplay{
		capabilities: caps,
		symbols:      SelectSymbols(caps),
	}
}

// StartPhase begins displaying progress for a phase
func (p *ProgressDisplay) StartPhase(phase PhaseInfo) error {
	// Validate phase info
	if err := phase.Validate(); err != nil {
		return err
	}

	p.currentPhase = &phase

	// Build the phase message
	msg := buildPhaseMessage(phase, "Running")

	if p.capabilities.IsTTY {
		// TTY mode: Start spinner animation
		p.spinner = spinner.New(
			spinner.CharSets[p.symbols.SpinnerSet],
			100*time.Millisecond,
		)
		p.spinner.Writer = os.Stderr // Write to stderr to avoid interfering with Claude's stdout
		p.spinner.Suffix = " " + msg
		p.spinner.Start()
	} else {
		// Non-interactive mode: Just print the message
		fmt.Println(msg)
	}

	return nil
}

// UpdateRetry updates the display with retry count information
func (p *ProgressDisplay) UpdateRetry(phase PhaseInfo) error {
	return p.StartPhase(phase)
}

// CompletePhase stops the spinner and displays completion status
func (p *ProgressDisplay) CompletePhase(phase PhaseInfo) error {
	// Stop spinner if running
	if p.spinner != nil {
		p.spinner.Stop()
		p.spinner = nil
	}

	// Display completion message
	mark := checkmark(p.symbols, p.capabilities.SupportsColor)
	counter := formatPhaseCounter(phase.Number, phase.TotalPhases)
	fmt.Printf("%s %s %s phase complete\n", mark, counter, capitalize(phase.Name))

	p.currentPhase = nil
	return nil
}

// FailPhase stops the spinner and displays failure status
func (p *ProgressDisplay) FailPhase(phase PhaseInfo, err error) error {
	// Stop spinner if running
	if p.spinner != nil {
		p.spinner.Stop()
		p.spinner = nil
	}

	// Display failure message
	mark := failureMark(p.symbols, p.capabilities.SupportsColor)
	counter := formatPhaseCounter(phase.Number, phase.TotalPhases)
	fmt.Printf("%s %s %s phase failed: %v\n", mark, counter, capitalize(phase.Name), err)

	p.currentPhase = nil
	return nil
}

// StopSpinner stops the spinner without showing completion/failure
// This is useful when you want to pause progress display during interactive output
func (p *ProgressDisplay) StopSpinner() {
	if p.spinner != nil {
		p.spinner.Stop()
		p.spinner = nil
	}
}
