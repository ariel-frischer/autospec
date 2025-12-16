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
		setupStore   func(t *testing.T, stateDir string)
		specName     string
		phase        string
		maxRetries   int
		wantCount    int
		wantCanRetry bool
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
			t.Parallel()
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
			t.Parallel()
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
		initialState *RetryState
		maxRetries   int
		wantCount    int
		wantErr      bool
		wantCanRetry bool
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
		t.Parallel()
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
		t.Parallel()
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

// Phase Execution State Tests

func TestPhaseExecutionState_Serialization(t *testing.T) {
	state := &PhaseExecutionState{
		SpecName:         "001-test-feature",
		CurrentPhase:     2,
		TotalPhases:      5,
		CompletedPhases:  []int{1},
		LastPhaseAttempt: time.Now(),
	}

	stateDir := t.TempDir()

	// Save and reload
	err := SavePhaseState(stateDir, state)
	require.NoError(t, err)

	loaded, err := LoadPhaseState(stateDir, "001-test-feature")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, state.SpecName, loaded.SpecName)
	assert.Equal(t, state.CurrentPhase, loaded.CurrentPhase)
	assert.Equal(t, state.TotalPhases, loaded.TotalPhases)
	assert.Equal(t, state.CompletedPhases, loaded.CompletedPhases)
}

func TestLoadPhaseState(t *testing.T) {
	tests := map[string]struct {
		setupStore func(t *testing.T, stateDir string)
		specName   string
		wantNil    bool
	}{
		"returns nil when file doesn't exist": {
			setupStore: func(t *testing.T, stateDir string) {},
			specName:   "001-test",
			wantNil:    true,
		},
		"returns nil when spec not in store": {
			setupStore: func(t *testing.T, stateDir string) {
				state := &PhaseExecutionState{
					SpecName:        "other-spec",
					CurrentPhase:    1,
					CompletedPhases: []int{},
				}
				require.NoError(t, SavePhaseState(stateDir, state))
			},
			specName: "001-test",
			wantNil:  true,
		},
		"loads existing state": {
			setupStore: func(t *testing.T, stateDir string) {
				state := &PhaseExecutionState{
					SpecName:        "001-test",
					CurrentPhase:    3,
					TotalPhases:     5,
					CompletedPhases: []int{1, 2},
				}
				require.NoError(t, SavePhaseState(stateDir, state))
			},
			specName: "001-test",
			wantNil:  false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()
			tc.setupStore(t, stateDir)

			state, err := LoadPhaseState(stateDir, tc.specName)
			require.NoError(t, err)

			if tc.wantNil {
				assert.Nil(t, state)
			} else {
				assert.NotNil(t, state)
				assert.Equal(t, tc.specName, state.SpecName)
			}
		})
	}
}

func TestSavePhaseState_Roundtrip(t *testing.T) {
	stateDir := t.TempDir()

	state := &PhaseExecutionState{
		SpecName:         "001-feature",
		CurrentPhase:     2,
		TotalPhases:      4,
		CompletedPhases:  []int{1},
		LastPhaseAttempt: time.Now().Truncate(time.Millisecond), // JSON loses nanoseconds
	}

	err := SavePhaseState(stateDir, state)
	require.NoError(t, err)

	loaded, err := LoadPhaseState(stateDir, "001-feature")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, state.SpecName, loaded.SpecName)
	assert.Equal(t, state.CurrentPhase, loaded.CurrentPhase)
	assert.Equal(t, state.TotalPhases, loaded.TotalPhases)
	assert.Equal(t, state.CompletedPhases, loaded.CompletedPhases)
}

func TestMarkPhaseComplete(t *testing.T) {
	tests := map[string]struct {
		initialState    *PhaseExecutionState
		phaseToComplete int
		wantCompleted   []int
	}{
		"mark first phase complete": {
			initialState:    nil,
			phaseToComplete: 1,
			wantCompleted:   []int{1},
		},
		"mark additional phase complete": {
			initialState: &PhaseExecutionState{
				SpecName:        "001-test",
				CompletedPhases: []int{1},
			},
			phaseToComplete: 2,
			wantCompleted:   []int{1, 2},
		},
		"marking same phase twice is idempotent": {
			initialState: &PhaseExecutionState{
				SpecName:        "001-test",
				CompletedPhases: []int{1, 2},
			},
			phaseToComplete: 2,
			wantCompleted:   []int{1, 2},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()

			if tc.initialState != nil {
				require.NoError(t, SavePhaseState(stateDir, tc.initialState))
			}

			err := MarkPhaseComplete(stateDir, "001-test", tc.phaseToComplete)
			require.NoError(t, err)

			loaded, err := LoadPhaseState(stateDir, "001-test")
			require.NoError(t, err)
			require.NotNil(t, loaded)

			assert.Equal(t, tc.wantCompleted, loaded.CompletedPhases)
		})
	}
}

func TestResetPhaseState(t *testing.T) {
	t.Run("reset existing state", func(t *testing.T) {
		t.Parallel()
		stateDir := t.TempDir()

		// Create initial state
		state := &PhaseExecutionState{
			SpecName:        "001-test",
			CurrentPhase:    3,
			TotalPhases:     5,
			CompletedPhases: []int{1, 2},
		}
		require.NoError(t, SavePhaseState(stateDir, state))

		// Reset
		err := ResetPhaseState(stateDir, "001-test")
		require.NoError(t, err)

		// Verify state is gone
		loaded, err := LoadPhaseState(stateDir, "001-test")
		require.NoError(t, err)
		assert.Nil(t, loaded)
	})

	t.Run("reset non-existent state", func(t *testing.T) {
		t.Parallel()
		stateDir := t.TempDir()

		// Reset should not error even if state doesn't exist
		err := ResetPhaseState(stateDir, "999-nonexistent")
		assert.NoError(t, err)
	})

	t.Run("reset preserves other states", func(t *testing.T) {
		t.Parallel()
		stateDir := t.TempDir()

		// Create two states
		state1 := &PhaseExecutionState{
			SpecName:        "001-test",
			CompletedPhases: []int{1, 2},
		}
		state2 := &PhaseExecutionState{
			SpecName:        "002-other",
			CompletedPhases: []int{1},
		}
		require.NoError(t, SavePhaseState(stateDir, state1))
		require.NoError(t, SavePhaseState(stateDir, state2))

		// Reset one state
		err := ResetPhaseState(stateDir, "001-test")
		require.NoError(t, err)

		// Verify first is gone
		loaded1, err := LoadPhaseState(stateDir, "001-test")
		require.NoError(t, err)
		assert.Nil(t, loaded1)

		// Verify second is preserved
		loaded2, err := LoadPhaseState(stateDir, "002-other")
		require.NoError(t, err)
		require.NotNil(t, loaded2)
		assert.Equal(t, []int{1}, loaded2.CompletedPhases)
	})
}

func TestPhaseExecutionState_IsPhaseCompleted(t *testing.T) {
	tests := map[string]struct {
		completedPhases []int
		checkPhase      int
		want            bool
	}{
		"phase is completed": {
			completedPhases: []int{1, 2, 3},
			checkPhase:      2,
			want:            true,
		},
		"phase is not completed": {
			completedPhases: []int{1, 2},
			checkPhase:      3,
			want:            false,
		},
		"empty completed list": {
			completedPhases: []int{},
			checkPhase:      1,
			want:            false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			state := &PhaseExecutionState{
				CompletedPhases: tc.completedPhases,
			}
			got := state.IsPhaseCompleted(tc.checkPhase)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestPhaseStateCoexistsWithRetryState(t *testing.T) {
	// Test that phase states and retry states can coexist in the same store
	stateDir := t.TempDir()

	// Save retry state
	retryState := &RetryState{
		SpecName:   "001-test",
		Phase:      "implement",
		Count:      1,
		MaxRetries: 3,
	}
	require.NoError(t, SaveRetryState(stateDir, retryState))

	// Save phase state
	phaseState := &PhaseExecutionState{
		SpecName:        "001-test",
		CurrentPhase:    2,
		TotalPhases:     4,
		CompletedPhases: []int{1},
	}
	require.NoError(t, SavePhaseState(stateDir, phaseState))

	// Verify both can be loaded
	loadedRetry, err := LoadRetryState(stateDir, "001-test", "implement", 3)
	require.NoError(t, err)
	assert.Equal(t, 1, loadedRetry.Count)

	loadedPhase, err := LoadPhaseState(stateDir, "001-test")
	require.NoError(t, err)
	require.NotNil(t, loadedPhase)
	assert.Equal(t, []int{1}, loadedPhase.CompletedPhases)
}

// Task Execution State Tests

func TestTaskExecutionState_Serialization(t *testing.T) {
	state := &TaskExecutionState{
		SpecName:         "001-test-feature",
		CurrentTaskID:    "T002",
		TotalTasks:       10,
		CompletedTaskIDs: []string{"T001"},
		LastTaskAttempt:  time.Now(),
	}

	stateDir := t.TempDir()

	// Save and reload
	err := SaveTaskState(stateDir, state)
	require.NoError(t, err)

	loaded, err := LoadTaskState(stateDir, "001-test-feature")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, state.SpecName, loaded.SpecName)
	assert.Equal(t, state.CurrentTaskID, loaded.CurrentTaskID)
	assert.Equal(t, state.TotalTasks, loaded.TotalTasks)
	assert.Equal(t, state.CompletedTaskIDs, loaded.CompletedTaskIDs)
}

func TestLoadTaskState(t *testing.T) {
	tests := map[string]struct {
		setupStore func(t *testing.T, stateDir string)
		specName   string
		wantNil    bool
	}{
		"returns nil when file doesn't exist": {
			setupStore: func(t *testing.T, stateDir string) {},
			specName:   "001-test",
			wantNil:    true,
		},
		"returns nil when spec not in store": {
			setupStore: func(t *testing.T, stateDir string) {
				state := &TaskExecutionState{
					SpecName:         "other-spec",
					CurrentTaskID:    "T001",
					CompletedTaskIDs: []string{},
				}
				require.NoError(t, SaveTaskState(stateDir, state))
			},
			specName: "001-test",
			wantNil:  true,
		},
		"loads existing state": {
			setupStore: func(t *testing.T, stateDir string) {
				state := &TaskExecutionState{
					SpecName:         "001-test",
					CurrentTaskID:    "T003",
					TotalTasks:       5,
					CompletedTaskIDs: []string{"T001", "T002"},
				}
				require.NoError(t, SaveTaskState(stateDir, state))
			},
			specName: "001-test",
			wantNil:  false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()
			tc.setupStore(t, stateDir)

			state, err := LoadTaskState(stateDir, tc.specName)
			require.NoError(t, err)

			if tc.wantNil {
				assert.Nil(t, state)
			} else {
				assert.NotNil(t, state)
				assert.Equal(t, tc.specName, state.SpecName)
			}
		})
	}
}

func TestSaveTaskState_Roundtrip(t *testing.T) {
	stateDir := t.TempDir()

	state := &TaskExecutionState{
		SpecName:         "001-feature",
		CurrentTaskID:    "T002",
		TotalTasks:       4,
		CompletedTaskIDs: []string{"T001"},
		LastTaskAttempt:  time.Now().Truncate(time.Millisecond), // JSON loses nanoseconds
	}

	err := SaveTaskState(stateDir, state)
	require.NoError(t, err)

	loaded, err := LoadTaskState(stateDir, "001-feature")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, state.SpecName, loaded.SpecName)
	assert.Equal(t, state.CurrentTaskID, loaded.CurrentTaskID)
	assert.Equal(t, state.TotalTasks, loaded.TotalTasks)
	assert.Equal(t, state.CompletedTaskIDs, loaded.CompletedTaskIDs)
}

func TestMarkTaskComplete(t *testing.T) {
	tests := map[string]struct {
		initialState   *TaskExecutionState
		taskToComplete string
		wantCompleted  []string
	}{
		"mark first task complete": {
			initialState:   nil,
			taskToComplete: "T001",
			wantCompleted:  []string{"T001"},
		},
		"mark additional task complete": {
			initialState: &TaskExecutionState{
				SpecName:         "001-test",
				CompletedTaskIDs: []string{"T001"},
			},
			taskToComplete: "T002",
			wantCompleted:  []string{"T001", "T002"},
		},
		"marking same task twice is idempotent": {
			initialState: &TaskExecutionState{
				SpecName:         "001-test",
				CompletedTaskIDs: []string{"T001", "T002"},
			},
			taskToComplete: "T002",
			wantCompleted:  []string{"T001", "T002"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stateDir := t.TempDir()

			if tc.initialState != nil {
				require.NoError(t, SaveTaskState(stateDir, tc.initialState))
			}

			err := MarkTaskComplete(stateDir, "001-test", tc.taskToComplete)
			require.NoError(t, err)

			loaded, err := LoadTaskState(stateDir, "001-test")
			require.NoError(t, err)
			require.NotNil(t, loaded)

			assert.Equal(t, tc.wantCompleted, loaded.CompletedTaskIDs)
		})
	}
}

func TestResetTaskState(t *testing.T) {
	t.Run("reset existing state", func(t *testing.T) {
		t.Parallel()
		stateDir := t.TempDir()

		// Create initial state
		state := &TaskExecutionState{
			SpecName:         "001-test",
			CurrentTaskID:    "T003",
			TotalTasks:       5,
			CompletedTaskIDs: []string{"T001", "T002"},
		}
		require.NoError(t, SaveTaskState(stateDir, state))

		// Reset
		err := ResetTaskState(stateDir, "001-test")
		require.NoError(t, err)

		// Verify state is gone
		loaded, err := LoadTaskState(stateDir, "001-test")
		require.NoError(t, err)
		assert.Nil(t, loaded)
	})

	t.Run("reset non-existent state", func(t *testing.T) {
		t.Parallel()
		stateDir := t.TempDir()

		// Reset should not error even if state doesn't exist
		err := ResetTaskState(stateDir, "999-nonexistent")
		assert.NoError(t, err)
	})

	t.Run("reset preserves other states", func(t *testing.T) {
		t.Parallel()
		stateDir := t.TempDir()

		// Create two states
		state1 := &TaskExecutionState{
			SpecName:         "001-test",
			CompletedTaskIDs: []string{"T001", "T002"},
		}
		state2 := &TaskExecutionState{
			SpecName:         "002-other",
			CompletedTaskIDs: []string{"T001"},
		}
		require.NoError(t, SaveTaskState(stateDir, state1))
		require.NoError(t, SaveTaskState(stateDir, state2))

		// Reset one state
		err := ResetTaskState(stateDir, "001-test")
		require.NoError(t, err)

		// Verify first is gone
		loaded1, err := LoadTaskState(stateDir, "001-test")
		require.NoError(t, err)
		assert.Nil(t, loaded1)

		// Verify second is preserved
		loaded2, err := LoadTaskState(stateDir, "002-other")
		require.NoError(t, err)
		require.NotNil(t, loaded2)
		assert.Equal(t, []string{"T001"}, loaded2.CompletedTaskIDs)
	})
}

func TestTaskExecutionState_IsTaskCompleted(t *testing.T) {
	tests := map[string]struct {
		completedTasks []string
		checkTask      string
		want           bool
	}{
		"task is completed": {
			completedTasks: []string{"T001", "T002", "T003"},
			checkTask:      "T002",
			want:           true,
		},
		"task is not completed": {
			completedTasks: []string{"T001", "T002"},
			checkTask:      "T003",
			want:           false,
		},
		"empty completed list": {
			completedTasks: []string{},
			checkTask:      "T001",
			want:           false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			state := &TaskExecutionState{
				CompletedTaskIDs: tc.completedTasks,
			}
			got := state.IsTaskCompleted(tc.checkTask)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestTaskStateCoexistsWithOtherStates(t *testing.T) {
	// Test that task states, phase states, and retry states can coexist in the same store
	stateDir := t.TempDir()

	// Save retry state
	retryState := &RetryState{
		SpecName:   "001-test",
		Phase:      "implement",
		Count:      1,
		MaxRetries: 3,
	}
	require.NoError(t, SaveRetryState(stateDir, retryState))

	// Save phase state
	phaseState := &PhaseExecutionState{
		SpecName:        "001-test",
		CurrentPhase:    2,
		TotalPhases:     4,
		CompletedPhases: []int{1},
	}
	require.NoError(t, SavePhaseState(stateDir, phaseState))

	// Save task state
	taskState := &TaskExecutionState{
		SpecName:         "001-test",
		CurrentTaskID:    "T003",
		TotalTasks:       10,
		CompletedTaskIDs: []string{"T001", "T002"},
	}
	require.NoError(t, SaveTaskState(stateDir, taskState))

	// Verify all three can be loaded
	loadedRetry, err := LoadRetryState(stateDir, "001-test", "implement", 3)
	require.NoError(t, err)
	assert.Equal(t, 1, loadedRetry.Count)

	loadedPhase, err := LoadPhaseState(stateDir, "001-test")
	require.NoError(t, err)
	require.NotNil(t, loadedPhase)
	assert.Equal(t, []int{1}, loadedPhase.CompletedPhases)

	loadedTask, err := LoadTaskState(stateDir, "001-test")
	require.NoError(t, err)
	require.NotNil(t, loadedTask)
	assert.Equal(t, []string{"T001", "T002"}, loadedTask.CompletedTaskIDs)
}
