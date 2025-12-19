package workflow

import (
	"errors"
	"testing"

	"github.com/ariel-frischer/autospec/internal/progress"
	"github.com/stretchr/testify/assert"
)

func TestNewProgressController(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		display     *progress.ProgressDisplay
		wantHas     bool
		description string
	}{
		"with display": {
			display:     progress.NewProgressDisplay(progress.TerminalCapabilities{IsTTY: false}),
			wantHas:     true,
			description: "should report display when provided",
		},
		"nil display": {
			display:     nil,
			wantHas:     false,
			description: "should report no display when nil",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			controller := NewProgressController(tc.display)
			assert.NotNil(t, controller)
			assert.Equal(t, tc.wantHas, controller.HasDisplay(), tc.description)
		})
	}
}

func TestProgressController_StartStage(t *testing.T) {
	t.Parallel()

	validStageInfo := progress.StageInfo{
		Name:        "test",
		Number:      1,
		TotalStages: 4,
		Status:      progress.StageInProgress,
		RetryCount:  0,
		MaxRetries:  3,
	}

	tests := map[string]struct {
		display   *progress.ProgressDisplay
		stageInfo progress.StageInfo
		wantErr   bool
	}{
		"nil display returns nil": {
			display:   nil,
			stageInfo: validStageInfo,
			wantErr:   false,
		},
		"valid display and stage info": {
			display:   progress.NewProgressDisplay(progress.TerminalCapabilities{IsTTY: false}),
			stageInfo: validStageInfo,
			wantErr:   false,
		},
		"invalid stage info with display": {
			display: progress.NewProgressDisplay(progress.TerminalCapabilities{IsTTY: false}),
			stageInfo: progress.StageInfo{
				Name:        "", // invalid - empty name
				Number:      0,  // invalid - zero
				TotalStages: 4,
				Status:      progress.StageInProgress,
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			controller := NewProgressController(tc.display)
			err := controller.StartStage(tc.stageInfo)

			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "starting stage display")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProgressController_CompleteStage(t *testing.T) {
	t.Parallel()

	validStageInfo := progress.StageInfo{
		Name:        "test",
		Number:      1,
		TotalStages: 4,
		Status:      progress.StageCompleted,
		RetryCount:  0,
		MaxRetries:  3,
	}

	tests := map[string]struct {
		display   *progress.ProgressDisplay
		stageInfo progress.StageInfo
		wantErr   bool
	}{
		"nil display returns nil": {
			display:   nil,
			stageInfo: validStageInfo,
			wantErr:   false,
		},
		"valid display and stage info": {
			display:   progress.NewProgressDisplay(progress.TerminalCapabilities{IsTTY: false}),
			stageInfo: validStageInfo,
			wantErr:   false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			controller := NewProgressController(tc.display)
			err := controller.CompleteStage(tc.stageInfo)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProgressController_FailStage(t *testing.T) {
	t.Parallel()

	validStageInfo := progress.StageInfo{
		Name:        "test",
		Number:      1,
		TotalStages: 4,
		Status:      progress.StageFailed,
		RetryCount:  0,
		MaxRetries:  3,
	}

	tests := map[string]struct {
		display   *progress.ProgressDisplay
		stageInfo progress.StageInfo
		err       error
	}{
		"nil display no panic": {
			display:   nil,
			stageInfo: validStageInfo,
			err:       errors.New("test error"),
		},
		"valid display with error": {
			display:   progress.NewProgressDisplay(progress.TerminalCapabilities{IsTTY: false}),
			stageInfo: validStageInfo,
			err:       errors.New("stage failed"),
		},
		"nil error": {
			display:   progress.NewProgressDisplay(progress.TerminalCapabilities{IsTTY: false}),
			stageInfo: validStageInfo,
			err:       nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			controller := NewProgressController(tc.display)
			// FailStage should not panic regardless of inputs
			assert.NotPanics(t, func() {
				controller.FailStage(tc.stageInfo, tc.err)
			})
		})
	}
}

func TestProgressController_StopSpinner(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		display *progress.ProgressDisplay
	}{
		"nil display no panic": {
			display: nil,
		},
		"valid display": {
			display: progress.NewProgressDisplay(progress.TerminalCapabilities{IsTTY: false}),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			controller := NewProgressController(tc.display)
			// StopSpinner should not panic regardless of state
			assert.NotPanics(t, func() {
				controller.StopSpinner()
			})
		})
	}
}

func TestProgressController_HasDisplay(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		display *progress.ProgressDisplay
		want    bool
	}{
		"nil display": {
			display: nil,
			want:    false,
		},
		"non-nil display": {
			display: progress.NewProgressDisplay(progress.TerminalCapabilities{IsTTY: false}),
			want:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			controller := NewProgressController(tc.display)
			assert.Equal(t, tc.want, controller.HasDisplay())
		})
	}
}
