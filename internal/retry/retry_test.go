package retry

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadRetryState(t *testing.T) {
	tests := map[string]struct {
		setupStore    func(t *testing.T, stateDir string)
		specName      string
		phase         string
		maxRetries    int
		wantCount     int
		wantCanRetry  bool
	}{
		"new state when file doesn't exist": {
			setupStore:   func(t *testing.T, stateDir string) {},
			specName:     "001",
			phase:        "specify",
			maxRetries:   3,
			wantCount:    0,
			wantCanRetry: true,
		},
		"new state when key doesn't exist": {
			setupStore: func(t *testing.T, stateDir string) {
				state := &RetryState{
					SpecName:   "002",
					Phase:      "plan",
					Count:      1,
					MaxRetries: 3,
				}
				require.NoError(t, SaveRetryState(stateDir, state))
			},
			specName:     "001",
			phase:        "specify",
			maxRetries:   3,
			wantCount:    0,
			wantCanRetry: true,
		},
		"load existing state": {
			setupStore: func(t *testing.T, stateDir string) {
				state := &RetryState{
					SpecName:   "001",
					Phase:      "specify",
					Count:      2,
					MaxRetries: 3,
				}
				require.NoError(t, SaveRetryState(stateDir, state))
			},
			specName:     "001",
			phase:        "specify",
			maxRetries:   3,
			wantCount:    2,
			wantCanRetry: true,
		},
		"load state with updated maxRetries": {
			setupStore: func(t *testing.T, stateDir string) {
				state := &RetryState{
					SpecName:   "001",
					Phase:      "specify",
					Count:      2,
					MaxRetries: 3,
				}
				require.NoError(t, SaveRetryState(stateDir, state))
			},
			specName:     "001",
			phase:        "specify",
			maxRetries:   5, // Updated max
			wantCount:    2,
			wantCanRetry: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()
			tc.setupStore(t, stateDir)

			state, err := LoadRetryState(stateDir, tc.specName, tc.phase, tc.maxRetries)
			require.NoError(t, err)
			assert.Equal(t, tc.specName, state.SpecName)
			assert.Equal(t, tc.phase, state.Phase)
			assert.Equal(t, tc.wantCount, state.Count)
			assert.Equal(t, tc.maxRetries, state.MaxRetries)
			assert.Equal(t, tc.wantCanRetry, state.CanRetry())
		})
	}
}

func TestSaveRetryState(t *testing.T) {
	tests := map[string]struct {
		state     *RetryState
		wantCount int
	}{
		"save new state": {
			state: &RetryState{
				SpecName:   "001",
				Phase:      "specify",
				Count:      1,
				MaxRetries: 3,
			},
			wantCount: 1,
		},
		"save updated state": {
			state: &RetryState{
				SpecName:   "002",
				Phase:      "plan",
				Count:      2,
				MaxRetries: 3,
			},
			wantCount: 2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()

			// Save state
			err := SaveRetryState(stateDir, tc.state)
			require.NoError(t, err)

			// Verify file exists
			retryPath := filepath.Join(stateDir, "retry.json")
			assert.FileExists(t, retryPath)

			// Load and verify
			loaded, err := LoadRetryState(stateDir, tc.state.SpecName, tc.state.Phase, tc.state.MaxRetries)
			require.NoError(t, err)
			assert.Equal(t, tc.wantCount, loaded.Count)
		})
	}
}

func TestRetryState_CanRetry(t *testing.T) {
	tests := map[string]struct {
		count      int
		maxRetries int
		want       bool
	}{
		"can retry with count=0": {
			count:      0,
			maxRetries: 3,
			want:       true,
		},
		"can retry with count<max": {
			count:      2,
			maxRetries: 3,
			want:       true,
		},
		"cannot retry with count=max": {
			count:      3,
			maxRetries: 3,
			want:       false,
		},
		"cannot retry with count>max": {
			count:      4,
			maxRetries: 3,
			want:       false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			state := &RetryState{
				Count:      tc.count,
				MaxRetries: tc.maxRetries,
			}
			assert.Equal(t, tc.want, state.CanRetry())
		})
	}
}

func TestRetryState_Increment(t *testing.T) {
	tests := map[string]struct {
		initialCount int
		maxRetries   int
		wantCount    int
		wantErr      bool
	}{
		"increment from 0": {
			initialCount: 0,
			maxRetries:   3,
			wantCount:    1,
			wantErr:      false,
		},
		"increment from 2": {
			initialCount: 2,
			maxRetries:   3,
			wantCount:    3,
			wantErr:      false,
		},
		"error when at max": {
			initialCount: 3,
			maxRetries:   3,
			wantCount:    3,
			wantErr:      true,
		},
		"error when above max": {
			initialCount: 4,
			maxRetries:   3,
			wantCount:    4,
			wantErr:      true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			state := &RetryState{
				SpecName:   "001",
				Phase:      "specify",
				Count:      tc.initialCount,
				MaxRetries: tc.maxRetries,
			}

			beforeTime := time.Now().Add(-time.Second)
			err := state.Increment()
			afterTime := time.Now().Add(time.Second)

			if tc.wantErr {
				assert.Error(t, err)
				var exhaustedErr *RetryExhaustedError
				require.ErrorAs(t, err, &exhaustedErr)
				assert.Equal(t, "001", exhaustedErr.SpecName)
				assert.Equal(t, "specify", exhaustedErr.Phase)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantCount, state.Count)
				assert.True(t, state.LastAttempt.After(beforeTime))
				assert.True(t, state.LastAttempt.Before(afterTime))
			}
		})
	}
}

func TestRetryState_Reset(t *testing.T) {
	state := &RetryState{
		SpecName:    "001",
		Phase:       "specify",
		Count:       3,
		LastAttempt: time.Now(),
		MaxRetries:  3,
	}

	state.Reset()

	assert.Equal(t, 0, state.Count)
	assert.True(t, state.LastAttempt.IsZero())
	assert.Equal(t, "001", state.SpecName)
	assert.Equal(t, "specify", state.Phase)
	assert.Equal(t, 3, state.MaxRetries)
}

func TestIncrementRetryCount(t *testing.T) {
	tests := map[string]struct {
		initialState  *RetryState
		maxRetries    int
		wantCount     int
		wantErr       bool
		wantCanRetry  bool
	}{
		"increment new state": {
			initialState: nil,
			maxRetries:   3,
			wantCount:    1,
			wantErr:      false,
			wantCanRetry: true,
		},
		"increment existing state": {
			initialState: &RetryState{
				SpecName:   "001",
				Phase:      "specify",
				Count:      1,
				MaxRetries: 3,
			},
			maxRetries:   3,
			wantCount:    2,
			wantErr:      false,
			wantCanRetry: true,
		},
		"error when at max": {
			initialState: &RetryState{
				SpecName:   "001",
				Phase:      "specify",
				Count:      3,
				MaxRetries: 3,
			},
			maxRetries:   3,
			wantCount:    3,
			wantErr:      true,
			wantCanRetry: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()

			// Set up initial state if provided
			if tc.initialState != nil {
				require.NoError(t, SaveRetryState(stateDir, tc.initialState))
			}

			// Increment
			state, err := IncrementRetryCount(stateDir, "001", "specify", tc.maxRetries)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantCount, state.Count)
				assert.Equal(t, tc.wantCanRetry, state.CanRetry())

				// Verify persistence
				loaded, err := LoadRetryState(stateDir, "001", "specify", tc.maxRetries)
				require.NoError(t, err)
				assert.Equal(t, tc.wantCount, loaded.Count)
			}
		})
	}
}

func TestResetRetryCount(t *testing.T) {
	t.Run("reset existing state", func(t *testing.T) {
		stateDir := t.TempDir()

		// Create initial state
		state := &RetryState{
			SpecName:    "001",
			Phase:       "specify",
			Count:       3,
			LastAttempt: time.Now(),
			MaxRetries:  3,
		}
		require.NoError(t, SaveRetryState(stateDir, state))

		// Reset
		err := ResetRetryCount(stateDir, "001", "specify")
		require.NoError(t, err)

		// Verify reset
		loaded, err := LoadRetryState(stateDir, "001", "specify", 3)
		require.NoError(t, err)
		assert.Equal(t, 0, loaded.Count)
		assert.True(t, loaded.LastAttempt.IsZero())
	})

	t.Run("reset non-existent state", func(t *testing.T) {
		stateDir := t.TempDir()

		// Reset should not error even if state doesn't exist
		err := ResetRetryCount(stateDir, "999", "nonexistent")
		assert.NoError(t, err)
	})
}

func TestAtomicWrite(t *testing.T) {
	// This test verifies that SaveRetryState uses atomic write (temp + rename)
	stateDir := t.TempDir()

	state := &RetryState{
		SpecName:   "001",
		Phase:      "specify",
		Count:      1,
		MaxRetries: 3,
	}

	err := SaveRetryState(stateDir, state)
	require.NoError(t, err)

	// Verify temp file doesn't exist
	tmpPath := filepath.Join(stateDir, "retry.json.tmp")
	_, err = os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(err), "temp file should not exist after atomic write")

	// Verify final file exists
	finalPath := filepath.Join(stateDir, "retry.json")
	assert.FileExists(t, finalPath)
}

func TestMultipleStates(t *testing.T) {
	// Test that multiple retry states can coexist in the same store
	stateDir := t.TempDir()

	states := []*RetryState{
		{SpecName: "001", Phase: "specify", Count: 1, MaxRetries: 3},
		{SpecName: "001", Phase: "plan", Count: 2, MaxRetries: 3},
		{SpecName: "002", Phase: "specify", Count: 0, MaxRetries: 3},
	}

	// Save all states
	for _, state := range states {
		require.NoError(t, SaveRetryState(stateDir, state))
	}

	// Verify all states can be loaded
	for _, expected := range states {
		loaded, err := LoadRetryState(stateDir, expected.SpecName, expected.Phase, expected.MaxRetries)
		require.NoError(t, err)
		assert.Equal(t, expected.Count, loaded.Count)
	}
}

func TestRetryExhaustedError(t *testing.T) {
	err := &RetryExhaustedError{
		SpecName:   "001",
		Phase:      "specify",
		Count:      3,
		MaxRetries: 3,
	}

	assert.Equal(t, 2, err.ExitCode())
	assert.Contains(t, err.Error(), "001:specify")
	assert.Contains(t, err.Error(), "3/3")
}
