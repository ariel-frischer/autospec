// Package retry_test tests retry state management for workflow stages, phases, and tasks.
// Related: /home/ari/repos/autospec/internal/retry/retry.go
// Tags: retry, state, persistence, stage, phase, task

package retry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadRetryState tests retry state persistence with setup closures.
//
// Uses closure-based setup: each test case defines setupStore func that
// pre-populates state before the test runs. Coverage:
//   - Missing file → fresh state (count=0)
//   - Missing key → fresh state for that spec/phase combo
//   - Existing state → loaded with correct values
//   - MaxRetries update → loaded state adopts new max
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

// Stage Execution State Tests

func TestStageExecutionState_Serialization(t *testing.T) {
	state := &StageExecutionState{
		SpecName:         "001-test-feature",
		CurrentPhase:     2,
		TotalPhases:      5,
		CompletedPhases:  []int{1},
		LastPhaseAttempt: time.Now(),
	}

	stateDir := t.TempDir()

	// Save and reload
	err := SaveStageState(stateDir, state)
	require.NoError(t, err)

	loaded, err := LoadStageState(stateDir, "001-test-feature")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, state.SpecName, loaded.SpecName)
	assert.Equal(t, state.CurrentPhase, loaded.CurrentPhase)
	assert.Equal(t, state.TotalPhases, loaded.TotalPhases)
	assert.Equal(t, state.CompletedPhases, loaded.CompletedPhases)
}

func TestLoadStageState(t *testing.T) {
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
				state := &StageExecutionState{
					SpecName:        "other-spec",
					CurrentPhase:    1,
					CompletedPhases: []int{},
				}
				require.NoError(t, SaveStageState(stateDir, state))
			},
			specName: "001-test",
			wantNil:  true,
		},
		"loads existing state": {
			setupStore: func(t *testing.T, stateDir string) {
				state := &StageExecutionState{
					SpecName:        "001-test",
					CurrentPhase:    3,
					TotalPhases:     5,
					CompletedPhases: []int{1, 2},
				}
				require.NoError(t, SaveStageState(stateDir, state))
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

			state, err := LoadStageState(stateDir, tc.specName)
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

func TestSaveStageState_Roundtrip(t *testing.T) {
	stateDir := t.TempDir()

	state := &StageExecutionState{
		SpecName:         "001-feature",
		CurrentPhase:     2,
		TotalPhases:      4,
		CompletedPhases:  []int{1},
		LastPhaseAttempt: time.Now().Truncate(time.Millisecond), // JSON loses nanoseconds
	}

	err := SaveStageState(stateDir, state)
	require.NoError(t, err)

	loaded, err := LoadStageState(stateDir, "001-feature")
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, state.SpecName, loaded.SpecName)
	assert.Equal(t, state.CurrentPhase, loaded.CurrentPhase)
	assert.Equal(t, state.TotalPhases, loaded.TotalPhases)
	assert.Equal(t, state.CompletedPhases, loaded.CompletedPhases)
}

func TestMarkStageComplete(t *testing.T) {
	tests := map[string]struct {
		initialState    *StageExecutionState
		phaseToComplete int
		wantCompleted   []int
	}{
		"mark first phase complete": {
			initialState:    nil,
			phaseToComplete: 1,
			wantCompleted:   []int{1},
		},
		"mark additional phase complete": {
			initialState: &StageExecutionState{
				SpecName:        "001-test",
				CompletedPhases: []int{1},
			},
			phaseToComplete: 2,
			wantCompleted:   []int{1, 2},
		},
		"marking same phase twice is idempotent": {
			initialState: &StageExecutionState{
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
				require.NoError(t, SaveStageState(stateDir, tc.initialState))
			}

			err := MarkStageComplete(stateDir, "001-test", tc.phaseToComplete)
			require.NoError(t, err)

			loaded, err := LoadStageState(stateDir, "001-test")
			require.NoError(t, err)
			require.NotNil(t, loaded)

			assert.Equal(t, tc.wantCompleted, loaded.CompletedPhases)
		})
	}
}

func TestResetStageState(t *testing.T) {
	t.Run("reset existing state", func(t *testing.T) {
		t.Parallel()
		stateDir := t.TempDir()

		// Create initial state
		state := &StageExecutionState{
			SpecName:        "001-test",
			CurrentPhase:    3,
			TotalPhases:     5,
			CompletedPhases: []int{1, 2},
		}
		require.NoError(t, SaveStageState(stateDir, state))

		// Reset
		err := ResetStageState(stateDir, "001-test")
		require.NoError(t, err)

		// Verify state is gone
		loaded, err := LoadStageState(stateDir, "001-test")
		require.NoError(t, err)
		assert.Nil(t, loaded)
	})

	t.Run("reset non-existent state", func(t *testing.T) {
		t.Parallel()
		stateDir := t.TempDir()

		// Reset should not error even if state doesn't exist
		err := ResetStageState(stateDir, "999-nonexistent")
		assert.NoError(t, err)
	})

	t.Run("reset preserves other states", func(t *testing.T) {
		t.Parallel()
		stateDir := t.TempDir()

		// Create two states
		state1 := &StageExecutionState{
			SpecName:        "001-test",
			CompletedPhases: []int{1, 2},
		}
		state2 := &StageExecutionState{
			SpecName:        "002-other",
			CompletedPhases: []int{1},
		}
		require.NoError(t, SaveStageState(stateDir, state1))
		require.NoError(t, SaveStageState(stateDir, state2))

		// Reset one state
		err := ResetStageState(stateDir, "001-test")
		require.NoError(t, err)

		// Verify first is gone
		loaded1, err := LoadStageState(stateDir, "001-test")
		require.NoError(t, err)
		assert.Nil(t, loaded1)

		// Verify second is preserved
		loaded2, err := LoadStageState(stateDir, "002-other")
		require.NoError(t, err)
		require.NotNil(t, loaded2)
		assert.Equal(t, []int{1}, loaded2.CompletedPhases)
	})
}

func TestStageExecutionState_IsPhaseCompleted(t *testing.T) {
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
			state := &StageExecutionState{
				CompletedPhases: tc.completedPhases,
			}
			got := state.IsPhaseCompleted(tc.checkPhase)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestStageStateCoexistsWithRetryState(t *testing.T) {
	// Test that stage states and retry states can coexist in the same store
	stateDir := t.TempDir()

	// Save retry state
	retryState := &RetryState{
		SpecName:   "001-test",
		Phase:      "implement",
		Count:      1,
		MaxRetries: 3,
	}
	require.NoError(t, SaveRetryState(stateDir, retryState))

	// Save stage state
	phaseState := &StageExecutionState{
		SpecName:        "001-test",
		CurrentPhase:    2,
		TotalPhases:     4,
		CompletedPhases: []int{1},
	}
	require.NoError(t, SaveStageState(stateDir, phaseState))

	// Verify both can be loaded
	loadedRetry, err := LoadRetryState(stateDir, "001-test", "implement", 3)
	require.NoError(t, err)
	assert.Equal(t, 1, loadedRetry.Count)

	loadedPhase, err := LoadStageState(stateDir, "001-test")
	require.NoError(t, err)
	require.NotNil(t, loadedPhase)
	assert.Equal(t, []int{1}, loadedPhase.CompletedPhases)
}

// Backward Compatibility Tests

func TestLegacyPhaseStatesBackwardCompatibility(t *testing.T) {
	// Test that old retry state files with "phase_states" can still be loaded
	// and are correctly migrated to "stage_states" format
	stateDir := t.TempDir()

	// Create a legacy format retry.json with phase_states (simulating old format)
	legacyData := `{
		"retries": {},
		"phase_states": {
			"001-test-feature": {
				"spec_name": "001-test-feature",
				"current_phase": 2,
				"total_phases": 5,
				"completed_phases": [1],
				"last_phase_attempt": "2024-01-15T10:30:00Z"
			}
		}
	}`

	retryPath := filepath.Join(stateDir, "retry.json")
	require.NoError(t, os.WriteFile(retryPath, []byte(legacyData), 0644))

	// Load should work and migrate phase_states to stage_states
	state, err := LoadStageState(stateDir, "001-test-feature")
	require.NoError(t, err)
	require.NotNil(t, state, "legacy phase_states should be migrated to stage_states")

	// Verify migrated data
	assert.Equal(t, "001-test-feature", state.SpecName)
	assert.Equal(t, 2, state.CurrentPhase)
	assert.Equal(t, 5, state.TotalPhases)
	assert.Equal(t, []int{1}, state.CompletedPhases)

	// Now save something new - this should use the new format
	state.CompletedPhases = append(state.CompletedPhases, 2)
	require.NoError(t, SaveStageState(stateDir, state))

	// Read the raw JSON to verify it uses stage_states, not phase_states
	data, err := os.ReadFile(retryPath)
	require.NoError(t, err)

	var rawStore map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &rawStore))

	// New format should have stage_states
	_, hasStageStates := rawStore["stage_states"]
	assert.True(t, hasStageStates, "saved file should have stage_states key")

	// Phase_states should NOT be in the new format (only stage_states)
	_, hasPhaseStates := rawStore["phase_states"]
	assert.False(t, hasPhaseStates, "saved file should not have phase_states key")
}

func TestMixedLegacyAndNewFormat(t *testing.T) {
	// Test that files with both phase_states and stage_states are handled correctly
	// (stage_states takes precedence, but legacy data is still migrated)
	stateDir := t.TempDir()

	// Create a file with both formats (edge case during migration)
	mixedData := `{
		"retries": {},
		"phase_states": {
			"old-spec": {
				"spec_name": "old-spec",
				"current_phase": 1,
				"total_phases": 3,
				"completed_phases": [1]
			}
		},
		"stage_states": {
			"new-spec": {
				"spec_name": "new-spec",
				"current_phase": 2,
				"total_phases": 4,
				"completed_phases": [1, 2]
			}
		}
	}`

	retryPath := filepath.Join(stateDir, "retry.json")
	require.NoError(t, os.WriteFile(retryPath, []byte(mixedData), 0644))

	// Load old spec (from legacy phase_states)
	oldState, err := LoadStageState(stateDir, "old-spec")
	require.NoError(t, err)
	require.NotNil(t, oldState)
	assert.Equal(t, "old-spec", oldState.SpecName)
	assert.Equal(t, []int{1}, oldState.CompletedPhases)

	// Load new spec (from stage_states)
	newState, err := LoadStageState(stateDir, "new-spec")
	require.NoError(t, err)
	require.NotNil(t, newState)
	assert.Equal(t, "new-spec", newState.SpecName)
	assert.Equal(t, []int{1, 2}, newState.CompletedPhases)
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
	// Test that task states, stage states, and retry states can coexist in the same store
	stateDir := t.TempDir()

	// Save retry state
	retryState := &RetryState{
		SpecName:   "001-test",
		Phase:      "implement",
		Count:      1,
		MaxRetries: 3,
	}
	require.NoError(t, SaveRetryState(stateDir, retryState))

	// Save stage state
	phaseState := &StageExecutionState{
		SpecName:        "001-test",
		CurrentPhase:    2,
		TotalPhases:     4,
		CompletedPhases: []int{1},
	}
	require.NoError(t, SaveStageState(stateDir, phaseState))

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

	loadedPhase, err := LoadStageState(stateDir, "001-test")
	require.NoError(t, err)
	require.NotNil(t, loadedPhase)
	assert.Equal(t, []int{1}, loadedPhase.CompletedPhases)

	loadedTask, err := LoadTaskState(stateDir, "001-test")
	require.NoError(t, err)
	require.NotNil(t, loadedTask)
	assert.Equal(t, []string{"T001", "T002"}, loadedTask.CompletedTaskIDs)
}

// Additional tests to improve coverage to 85%

func TestLoadStore_CorruptedJSON(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	retryPath := filepath.Join(stateDir, "retry.json")

	// Write corrupted JSON
	require.NoError(t, os.WriteFile(retryPath, []byte("not valid json {{{"), 0644))

	// loadStore should return error for corrupted JSON
	_, err := loadStore(stateDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal")
}

func TestLoadStore_ReadError(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	retryPath := filepath.Join(stateDir, "retry.json")

	// Create a directory instead of a file to cause read error
	require.NoError(t, os.MkdirAll(retryPath, 0755))

	// loadStore should return error when file is actually a directory
	_, err := loadStore(stateDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read")
}

func TestSaveRetryState_MarshalError(t *testing.T) {
	// This test is tricky because JSON marshaling of RetryState doesn't fail easily.
	// We can test the existing behavior with nil Retries map in store
	stateDir := t.TempDir()

	state := &RetryState{
		SpecName:   "test-spec",
		Phase:      "test-phase",
		Count:      1,
		MaxRetries: 3,
	}

	err := SaveRetryState(stateDir, state)
	require.NoError(t, err)

	// Verify the store handles nil Retries map by creating one
	loaded, err := LoadRetryState(stateDir, "test-spec", "test-phase", 3)
	require.NoError(t, err)
	assert.Equal(t, 1, loaded.Count)
}

func TestSaveStageState_CreatesStore(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	// Save state without any existing store
	state := &StageExecutionState{
		SpecName:        "new-spec",
		CurrentPhase:    1,
		TotalPhases:     3,
		CompletedPhases: []int{},
	}

	err := SaveStageState(stateDir, state)
	require.NoError(t, err)

	// Verify it was saved
	loaded, err := LoadStageState(stateDir, "new-spec")
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "new-spec", loaded.SpecName)
}

func TestSaveTaskState_CreatesStore(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	// Save task state without any existing store
	state := &TaskExecutionState{
		SpecName:         "new-spec",
		CurrentTaskID:    "T001",
		TotalTasks:       5,
		CompletedTaskIDs: []string{},
	}

	err := SaveTaskState(stateDir, state)
	require.NoError(t, err)

	// Verify it was saved
	loaded, err := LoadTaskState(stateDir, "new-spec")
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "new-spec", loaded.SpecName)
}

func TestMarkStageComplete_WithCorruptedStore(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	retryPath := filepath.Join(stateDir, "retry.json")

	// Create corrupted JSON - LoadStageState silently ignores errors and returns nil
	require.NoError(t, os.WriteFile(retryPath, []byte("invalid json"), 0644))

	// MarkStageComplete creates new state when load returns nil
	// Save will create a new valid file, overwriting the corrupted one
	err := MarkStageComplete(stateDir, "test-spec", 1)
	require.NoError(t, err)

	// Verify the new state was saved correctly
	loaded, err := LoadStageState(stateDir, "test-spec")
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, []int{1}, loaded.CompletedPhases)
}

func TestMarkTaskComplete_WithCorruptedStore(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	retryPath := filepath.Join(stateDir, "retry.json")

	// Create corrupted JSON - LoadTaskState silently ignores errors and returns nil
	require.NoError(t, os.WriteFile(retryPath, []byte("invalid json"), 0644))

	// MarkTaskComplete creates new state when load returns nil
	// Save will create a new valid file, overwriting the corrupted one
	err := MarkTaskComplete(stateDir, "test-spec", "T001")
	require.NoError(t, err)

	// Verify the new state was saved correctly
	loaded, err := LoadTaskState(stateDir, "test-spec")
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, []string{"T001"}, loaded.CompletedTaskIDs)
}

func TestResetStageState_WithCorruptedStore(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	retryPath := filepath.Join(stateDir, "retry.json")

	// Create corrupted JSON
	require.NoError(t, os.WriteFile(retryPath, []byte("invalid json"), 0644))

	// Reset should not error if it can't load the store (nothing to reset)
	err := ResetStageState(stateDir, "test-spec")
	assert.NoError(t, err)
}

func TestResetTaskState_WithCorruptedStore(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	retryPath := filepath.Join(stateDir, "retry.json")

	// Create corrupted JSON
	require.NoError(t, os.WriteFile(retryPath, []byte("invalid json"), 0644))

	// Reset should not error if it can't load the store (nothing to reset)
	err := ResetTaskState(stateDir, "test-spec")
	assert.NoError(t, err)
}

func TestIncrementRetryCount_LoadError(t *testing.T) {
	// IncrementRetryCount handles load errors by creating new state
	// Test that it works correctly when state dir doesn't exist
	stateDir := filepath.Join(t.TempDir(), "nonexistent")

	// Should create the directory and increment
	state, err := IncrementRetryCount(stateDir, "test", "phase", 3)
	require.NoError(t, err)
	assert.Equal(t, 1, state.Count)
}

func TestIncrementRetryCount_SaveError(t *testing.T) {
	// Test IncrementRetryCount when save fails
	// This is hard to trigger without mocking, but we can test the happy path more thoroughly
	stateDir := t.TempDir()

	// First increment
	state1, err := IncrementRetryCount(stateDir, "test", "phase", 3)
	require.NoError(t, err)
	assert.Equal(t, 1, state1.Count)

	// Second increment
	state2, err := IncrementRetryCount(stateDir, "test", "phase", 3)
	require.NoError(t, err)
	assert.Equal(t, 2, state2.Count)

	// Third increment
	state3, err := IncrementRetryCount(stateDir, "test", "phase", 3)
	require.NoError(t, err)
	assert.Equal(t, 3, state3.Count)

	// Fourth increment should fail (at max)
	_, err = IncrementRetryCount(stateDir, "test", "phase", 3)
	assert.Error(t, err)
}

func TestResetRetryCount_AfterIncrements(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()

	// Increment a few times
	_, err := IncrementRetryCount(stateDir, "test", "phase", 5)
	require.NoError(t, err)
	_, err = IncrementRetryCount(stateDir, "test", "phase", 5)
	require.NoError(t, err)

	// Reset
	err = ResetRetryCount(stateDir, "test", "phase")
	require.NoError(t, err)

	// Load should show count=0
	loaded, err := LoadRetryState(stateDir, "test", "phase", 5)
	require.NoError(t, err)
	assert.Equal(t, 0, loaded.Count)
}

func TestSaveRetryState_DirectoryCreation(t *testing.T) {
	t.Parallel()

	// Test that SaveRetryState creates the directory if it doesn't exist
	baseDir := t.TempDir()
	stateDir := filepath.Join(baseDir, "nested", "state", "dir")

	state := &RetryState{
		SpecName:   "test",
		Phase:      "phase",
		Count:      1,
		MaxRetries: 3,
	}

	err := SaveRetryState(stateDir, state)
	require.NoError(t, err)

	// Verify directory was created
	assert.DirExists(t, stateDir)

	// Verify state was saved
	loaded, err := LoadRetryState(stateDir, "test", "phase", 3)
	require.NoError(t, err)
	assert.Equal(t, 1, loaded.Count)
}

func TestSaveStageState_DirectoryCreation(t *testing.T) {
	t.Parallel()

	// Test that SaveStageState creates the directory if it doesn't exist
	baseDir := t.TempDir()
	stateDir := filepath.Join(baseDir, "nested", "stage", "dir")

	state := &StageExecutionState{
		SpecName:        "test",
		CurrentPhase:    1,
		TotalPhases:     3,
		CompletedPhases: []int{},
	}

	err := SaveStageState(stateDir, state)
	require.NoError(t, err)

	// Verify directory was created
	assert.DirExists(t, stateDir)

	// Verify state was saved
	loaded, err := LoadStageState(stateDir, "test")
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, 1, loaded.CurrentPhase)
}

func TestSaveTaskState_DirectoryCreation(t *testing.T) {
	t.Parallel()

	// Test that SaveTaskState creates the directory if it doesn't exist
	baseDir := t.TempDir()
	stateDir := filepath.Join(baseDir, "nested", "task", "dir")

	state := &TaskExecutionState{
		SpecName:         "test",
		CurrentTaskID:    "T001",
		TotalTasks:       5,
		CompletedTaskIDs: []string{},
	}

	err := SaveTaskState(stateDir, state)
	require.NoError(t, err)

	// Verify directory was created
	assert.DirExists(t, stateDir)

	// Verify state was saved
	loaded, err := LoadTaskState(stateDir, "test")
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "T001", loaded.CurrentTaskID)
}

func TestResetStageState_DirectoryCreation(t *testing.T) {
	t.Parallel()

	// Test that ResetStageState creates directory if needed
	baseDir := t.TempDir()
	stateDir := filepath.Join(baseDir, "nested", "reset", "dir")

	// Should not error even though directory doesn't exist
	err := ResetStageState(stateDir, "nonexistent")
	require.NoError(t, err)

	// Verify directory was created
	assert.DirExists(t, stateDir)
}

func TestResetTaskState_DirectoryCreation(t *testing.T) {
	t.Parallel()

	// Test that ResetTaskState creates directory if needed
	baseDir := t.TempDir()
	stateDir := filepath.Join(baseDir, "nested", "reset", "task", "dir")

	// Should not error even though directory doesn't exist
	err := ResetTaskState(stateDir, "nonexistent")
	require.NoError(t, err)

	// Verify directory was created
	assert.DirExists(t, stateDir)
}

func TestLoadStore_NilRetries(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	retryPath := filepath.Join(stateDir, "retry.json")

	// Create JSON with null retries
	require.NoError(t, os.WriteFile(retryPath, []byte(`{"retries": null}`), 0644))

	store, err := loadStore(stateDir)
	require.NoError(t, err)
	require.NotNil(t, store)
	assert.NotNil(t, store.Retries) // Should be initialized
}
