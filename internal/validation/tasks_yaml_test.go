package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTasksYAML(t *testing.T) {
	tests := map[string]struct {
		content     string
		wantPhases  int
		wantTasks   int
		wantErr     bool
		errContains string
	}{
		"valid tasks with multiple phases": {
			content: `_meta:
  version: "1.0"
  generator: autospec
phases:
  - number: 1
    title: Setup
    tasks:
      - id: T001
        title: Create schema
        status: Completed
  - number: 2
    title: Implementation
    tasks:
      - id: T002
        title: Add API endpoints
        status: Pending
      - id: T003
        title: Write tests
        status: InProgress
`,
			wantPhases: 2,
			wantTasks:  3,
			wantErr:    false,
		},
		"valid tasks with nested structure": {
			content: `phases:
  - number: 1
    title: Phase One
    purpose: Setup
    tasks:
      - id: T001
        title: Task 1
        status: Pending
        type: implementation
        parallel: false
        dependencies: []
        acceptance_criteria:
          - Criterion 1
          - Criterion 2
`,
			wantPhases: 1,
			wantTasks:  1,
			wantErr:    false,
		},
		"empty phases list": {
			content: `_meta:
  version: "1.0"
phases: []
`,
			wantPhases: 0,
			wantTasks:  0,
			wantErr:    false,
		},
		"invalid yaml syntax": {
			content:     `phases:\n  - number: 1\n  title: bad indent`,
			wantErr:     true,
			errContains: "parse",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create temp file
			dir := t.TempDir()
			tasksPath := filepath.Join(dir, "tasks.yaml")
			require.NoError(t, os.WriteFile(tasksPath, []byte(tc.content), 0644))

			// Parse
			tasks, err := ParseTasksYAML(tasksPath)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, tasks)
			assert.Equal(t, tc.wantPhases, len(tasks.Phases))

			// Count total tasks
			totalTasks := 0
			for _, phase := range tasks.Phases {
				totalTasks += len(phase.Tasks)
			}
			assert.Equal(t, tc.wantTasks, totalTasks)
		})
	}
}

func TestParseTasksYAML_FileNotFound(t *testing.T) {
	_, err := ParseTasksYAML("/nonexistent/path/tasks.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read")
}

func TestParseTasksYAML_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	tasksPath := filepath.Join(dir, "tasks.yaml")
	require.NoError(t, os.WriteFile(tasksPath, []byte(""), 0644))

	tasks, err := ParseTasksYAML(tasksPath)
	require.NoError(t, err)
	assert.Equal(t, 0, len(tasks.Phases))
}

func TestGetTaskStats_YAML(t *testing.T) {
	tests := map[string]struct {
		content          string
		wantTotal        int
		wantCompleted    int
		wantInProgress   int
		wantPending      int
		wantBlocked      int
		wantPhases       int
		wantCompletedPhs int
	}{
		"mixed statuses": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        status: Completed
      - id: T002
        status: InProgress
      - id: T003
        status: Pending
      - id: T004
        status: Blocked
`,
			wantTotal:        4,
			wantCompleted:    1,
			wantInProgress:   1,
			wantPending:      1,
			wantBlocked:      1,
			wantPhases:       1,
			wantCompletedPhs: 0,
		},
		"all completed phase": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        status: Completed
      - id: T002
        status: Done
      - id: T003
        status: Complete
`,
			wantTotal:        3,
			wantCompleted:    3,
			wantInProgress:   0,
			wantPending:      0,
			wantBlocked:      0,
			wantPhases:       1,
			wantCompletedPhs: 1,
		},
		"multiple phases mixed completion": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        status: Completed
  - number: 2
    title: Phase 2
    tasks:
      - id: T002
        status: Pending
      - id: T003
        status: Completed
`,
			wantTotal:        3,
			wantCompleted:    2,
			wantInProgress:   0,
			wantPending:      1,
			wantBlocked:      0,
			wantPhases:       2,
			wantCompletedPhs: 1,
		},
		"status variations": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        status: in_progress
      - id: T002
        status: wip
      - id: T003
        status: in-progress
`,
			wantTotal:        3,
			wantCompleted:    0,
			wantInProgress:   3,
			wantPending:      0,
			wantBlocked:      0,
			wantPhases:       1,
			wantCompletedPhs: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			tasksPath := filepath.Join(dir, "tasks.yaml")
			require.NoError(t, os.WriteFile(tasksPath, []byte(tc.content), 0644))

			stats, err := GetTaskStats(tasksPath)

			require.NoError(t, err)
			require.NotNil(t, stats)
			assert.Equal(t, tc.wantTotal, stats.TotalTasks)
			assert.Equal(t, tc.wantCompleted, stats.CompletedTasks)
			assert.Equal(t, tc.wantInProgress, stats.InProgressTasks)
			assert.Equal(t, tc.wantPending, stats.PendingTasks)
			assert.Equal(t, tc.wantBlocked, stats.BlockedTasks)
			assert.Equal(t, tc.wantPhases, stats.TotalPhases)
			assert.Equal(t, tc.wantCompletedPhs, stats.CompletedPhases)
		})
	}
}

func TestGetTaskStats_MarkdownFallback(t *testing.T) {
	content := `# Tasks

## Phase 1
- [x] Task 1
- [ ] Task 2

## Phase 2
- [x] Task 3
- [x] Task 4
`

	dir := t.TempDir()
	tasksPath := filepath.Join(dir, "tasks.md")
	require.NoError(t, os.WriteFile(tasksPath, []byte(content), 0644))

	stats, err := GetTaskStats(tasksPath)

	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, 4, stats.TotalTasks)
	assert.Equal(t, 3, stats.CompletedTasks)
	assert.Equal(t, 1, stats.PendingTasks)
	assert.Equal(t, 2, stats.TotalPhases)
	assert.Equal(t, 1, stats.CompletedPhases) // Only Phase 2 is complete
}

func TestTaskStats_CompletionPercentage(t *testing.T) {
	tests := map[string]struct {
		stats *TaskStats
		want  float64
	}{
		"all complete": {
			stats: &TaskStats{TotalTasks: 10, CompletedTasks: 10},
			want:  100.0,
		},
		"half complete": {
			stats: &TaskStats{TotalTasks: 10, CompletedTasks: 5},
			want:  50.0,
		},
		"none complete": {
			stats: &TaskStats{TotalTasks: 10, CompletedTasks: 0},
			want:  0.0,
		},
		"no tasks": {
			stats: &TaskStats{TotalTasks: 0, CompletedTasks: 0},
			want:  100.0, // Empty is considered complete
		},
		"one third": {
			stats: &TaskStats{TotalTasks: 3, CompletedTasks: 1},
			want:  33.333333333333336,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.stats.CompletionPercentage()
			assert.InDelta(t, tc.want, got, 0.001)
		})
	}
}

func TestTaskStats_IsComplete(t *testing.T) {
	tests := map[string]struct {
		stats *TaskStats
		want  bool
	}{
		"all complete": {
			stats: &TaskStats{TotalTasks: 5, CompletedTasks: 5},
			want:  true,
		},
		"partially complete": {
			stats: &TaskStats{TotalTasks: 5, CompletedTasks: 3},
			want:  false,
		},
		"none complete": {
			stats: &TaskStats{TotalTasks: 5, CompletedTasks: 0},
			want:  false,
		},
		"no tasks": {
			stats: &TaskStats{TotalTasks: 0, CompletedTasks: 0},
			want:  false, // Empty is not considered complete
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.stats.IsComplete()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestPhaseStats(t *testing.T) {
	content := `phases:
  - number: 1
    title: Complete Phase
    tasks:
      - id: T001
        status: Completed
      - id: T002
        status: Completed
  - number: 2
    title: Incomplete Phase
    tasks:
      - id: T003
        status: Completed
      - id: T004
        status: Pending
`

	dir := t.TempDir()
	tasksPath := filepath.Join(dir, "tasks.yaml")
	require.NoError(t, os.WriteFile(tasksPath, []byte(content), 0644))

	stats, err := GetTaskStats(tasksPath)
	require.NoError(t, err)
	require.Len(t, stats.PhaseStats, 2)

	// Check first phase (complete)
	phase1 := stats.PhaseStats[0]
	assert.Equal(t, 1, phase1.Number)
	assert.Equal(t, "Complete Phase", phase1.Title)
	assert.Equal(t, 2, phase1.TotalTasks)
	assert.Equal(t, 2, phase1.CompletedTasks)
	assert.True(t, phase1.IsComplete)

	// Check second phase (incomplete)
	phase2 := stats.PhaseStats[1]
	assert.Equal(t, 2, phase2.Number)
	assert.Equal(t, "Incomplete Phase", phase2.Title)
	assert.Equal(t, 2, phase2.TotalTasks)
	assert.Equal(t, 1, phase2.CompletedTasks)
	assert.False(t, phase2.IsComplete)
}

func TestFormatTaskSummary(t *testing.T) {
	tests := map[string]struct {
		stats           *TaskStats
		wantContains    []string
		wantNotContains []string
	}{
		"basic summary": {
			stats: &TaskStats{
				TotalTasks:      10,
				CompletedTasks:  5,
				TotalPhases:     2,
				CompletedPhases: 1,
			},
			wantContains: []string{
				"5/10 tasks completed",
				"50%",
				"1/2 task phases completed",
			},
			wantNotContains: []string{
				"in progress",
				"blocked",
			},
		},
		"with in-progress tasks": {
			stats: &TaskStats{
				TotalTasks:      10,
				CompletedTasks:  5,
				InProgressTasks: 2,
				TotalPhases:     2,
				CompletedPhases: 1,
			},
			wantContains: []string{
				"5/10 tasks completed",
				"2 in progress",
			},
		},
		"with blocked tasks": {
			stats: &TaskStats{
				TotalTasks:      10,
				CompletedTasks:  5,
				BlockedTasks:    1,
				TotalPhases:     2,
				CompletedPhases: 1,
			},
			wantContains: []string{
				"5/10 tasks completed",
				"1 blocked",
			},
		},
		"with both in-progress and blocked": {
			stats: &TaskStats{
				TotalTasks:      10,
				CompletedTasks:  5,
				InProgressTasks: 2,
				BlockedTasks:    1,
				TotalPhases:     2,
				CompletedPhases: 1,
			},
			wantContains: []string{
				"2 in progress",
				"1 blocked",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			output := FormatTaskSummary(tc.stats)

			for _, want := range tc.wantContains {
				assert.Contains(t, output, want)
			}
			for _, notWant := range tc.wantNotContains {
				assert.NotContains(t, output, notWant)
			}
		})
	}
}

func TestGetTaskStats_EmptyPhases(t *testing.T) {
	content := `phases:
  - number: 1
    title: Empty Phase
    tasks: []
  - number: 2
    title: Another Empty Phase
    tasks: []
`

	dir := t.TempDir()
	tasksPath := filepath.Join(dir, "tasks.yaml")
	require.NoError(t, os.WriteFile(tasksPath, []byte(content), 0644))

	stats, err := GetTaskStats(tasksPath)

	require.NoError(t, err)
	assert.Equal(t, 0, stats.TotalTasks)
	assert.Equal(t, 0, stats.CompletedTasks)
	assert.Equal(t, 2, stats.TotalPhases)
	// Empty phases are technically "complete" (no uncompleted tasks)
	assert.Equal(t, 0, stats.CompletedPhases) // But IsComplete requires TotalTasks > 0
}

func TestTasksYAML_MetaFields(t *testing.T) {
	content := `_meta:
  version: "1.0"
  generator: autospec
  generator_version: "0.5.0"
  created: "2024-01-15T10:30:00Z"
  artifact_type: tasks
tasks:
  branch: feature/test
  created: "2024-01-15"
  spec_path: specs/001-test/spec.yaml
  plan_path: specs/001-test/plan.yaml
summary:
  total_tasks: 10
  total_phases: 3
  parallel_opportunities: 2
  estimated_complexity: medium
phases: []
`

	dir := t.TempDir()
	tasksPath := filepath.Join(dir, "tasks.yaml")
	require.NoError(t, os.WriteFile(tasksPath, []byte(content), 0644))

	tasks, err := ParseTasksYAML(tasksPath)
	require.NoError(t, err)

	// Verify meta fields
	assert.Equal(t, "1.0", tasks.Meta.Version)
	assert.Equal(t, "autospec", tasks.Meta.Generator)
	assert.Equal(t, "0.5.0", tasks.Meta.GeneratorVersion)
	assert.Equal(t, "tasks", tasks.Meta.ArtifactType)

	// Verify tasks info
	assert.Equal(t, "feature/test", tasks.Tasks.Branch)
	assert.Equal(t, "specs/001-test/spec.yaml", tasks.Tasks.SpecPath)
	assert.Equal(t, "specs/001-test/plan.yaml", tasks.Tasks.PlanPath)

	// Verify summary
	assert.Equal(t, 10, tasks.Summary.TotalTasks)
	assert.Equal(t, 3, tasks.Summary.TotalPhases)
	assert.Equal(t, 2, tasks.Summary.ParallelOpportunities)
	assert.Equal(t, "medium", tasks.Summary.EstimatedComplexity)
}

func TestTaskItem_Fields(t *testing.T) {
	content := `phases:
  - number: 1
    title: Test Phase
    story_reference: US-001
    tasks:
      - id: T001
        title: Test Task
        status: Pending
        type: implementation
        parallel: true
        story_id: US-001
        file_path: src/main.go
        dependencies:
          - T000
        acceptance_criteria:
          - Criterion 1
          - Criterion 2
`

	dir := t.TempDir()
	tasksPath := filepath.Join(dir, "tasks.yaml")
	require.NoError(t, os.WriteFile(tasksPath, []byte(content), 0644))

	tasks, err := ParseTasksYAML(tasksPath)
	require.NoError(t, err)
	require.Len(t, tasks.Phases, 1)
	require.Len(t, tasks.Phases[0].Tasks, 1)

	task := tasks.Phases[0].Tasks[0]
	assert.Equal(t, "T001", task.ID)
	assert.Equal(t, "Test Task", task.Title)
	assert.Equal(t, "Pending", task.Status)
	assert.Equal(t, "implementation", task.Type)
	assert.True(t, task.Parallel)
	assert.Equal(t, "US-001", task.StoryID)
	assert.Equal(t, "src/main.go", task.FilePath)
	assert.Equal(t, []string{"T000"}, task.Dependencies)
	assert.Equal(t, []string{"Criterion 1", "Criterion 2"}, task.AcceptanceCriteria)

	// Verify phase fields
	phase := tasks.Phases[0]
	assert.Equal(t, 1, phase.Number)
	assert.Equal(t, "Test Phase", phase.Title)
	assert.Equal(t, "US-001", phase.StoryReference)
}

// Tests for GetPhaseInfo function
func TestGetPhaseInfo(t *testing.T) {
	tests := map[string]struct {
		content    string
		wantPhases int
		validate   func(t *testing.T, phases []PhaseInfo)
	}{
		"mixed task statuses": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        status: Completed
      - id: T002
        status: InProgress
      - id: T003
        status: Pending
      - id: T004
        status: Blocked
`,
			wantPhases: 1,
			validate: func(t *testing.T, phases []PhaseInfo) {
				require.Len(t, phases, 1)
				assert.Equal(t, 1, phases[0].Number)
				assert.Equal(t, "Phase 1", phases[0].Title)
				assert.Equal(t, 4, phases[0].TotalTasks)
				assert.Equal(t, 1, phases[0].CompletedTasks)
				assert.Equal(t, 1, phases[0].BlockedTasks)
				assert.Equal(t, 2, phases[0].ActionableTasks) // InProgress + Pending
			},
		},
		"all completed phase": {
			content: `phases:
  - number: 1
    title: Complete Phase
    tasks:
      - id: T001
        status: Completed
      - id: T002
        status: Done
      - id: T003
        status: Complete
`,
			wantPhases: 1,
			validate: func(t *testing.T, phases []PhaseInfo) {
				require.Len(t, phases, 1)
				assert.Equal(t, 3, phases[0].CompletedTasks)
				assert.Equal(t, 0, phases[0].BlockedTasks)
				assert.Equal(t, 0, phases[0].ActionableTasks)
				assert.True(t, phases[0].IsComplete())
			},
		},
		"completed and blocked only": {
			content: `phases:
  - number: 1
    title: Done Phase
    tasks:
      - id: T001
        status: Completed
      - id: T002
        status: Blocked
      - id: T003
        status: Completed
`,
			wantPhases: 1,
			validate: func(t *testing.T, phases []PhaseInfo) {
				require.Len(t, phases, 1)
				assert.Equal(t, 2, phases[0].CompletedTasks)
				assert.Equal(t, 1, phases[0].BlockedTasks)
				assert.Equal(t, 0, phases[0].ActionableTasks)
				assert.True(t, phases[0].IsComplete()) // No actionable tasks = complete
			},
		},
		"multiple phases": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        status: Completed
      - id: T002
        status: Completed
  - number: 2
    title: Phase 2
    tasks:
      - id: T003
        status: Pending
  - number: 3
    title: Phase 3
    tasks:
      - id: T004
        status: Blocked
`,
			wantPhases: 3,
			validate: func(t *testing.T, phases []PhaseInfo) {
				require.Len(t, phases, 3)
				// Phase 1: all completed
				assert.True(t, phases[0].IsComplete())
				// Phase 2: has pending
				assert.False(t, phases[1].IsComplete())
				assert.Equal(t, 1, phases[1].ActionableTasks)
				// Phase 3: only blocked
				assert.True(t, phases[2].IsComplete())
			},
		},
		"empty phases": {
			content: `phases:
  - number: 1
    title: Empty Phase
    tasks: []
`,
			wantPhases: 1,
			validate: func(t *testing.T, phases []PhaseInfo) {
				require.Len(t, phases, 1)
				assert.Equal(t, 0, phases[0].TotalTasks)
				assert.True(t, phases[0].IsComplete()) // Empty = complete
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			tasksPath := filepath.Join(dir, "tasks.yaml")
			require.NoError(t, os.WriteFile(tasksPath, []byte(tc.content), 0644))

			phases, err := GetPhaseInfo(tasksPath)
			require.NoError(t, err)
			assert.Len(t, phases, tc.wantPhases)

			if tc.validate != nil {
				tc.validate(t, phases)
			}
		})
	}
}

func TestIsPhaseComplete(t *testing.T) {
	tests := map[string]struct {
		content      string
		phaseNum     int
		wantComplete bool
		wantErr      bool
	}{
		"completed phase": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        status: Completed
`,
			phaseNum:     1,
			wantComplete: true,
		},
		"incomplete phase": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        status: Pending
`,
			phaseNum:     1,
			wantComplete: false,
		},
		"blocked only is complete": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        status: Blocked
`,
			phaseNum:     1,
			wantComplete: true,
		},
		"mixed completed and blocked is complete": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        status: Completed
      - id: T002
        status: Blocked
`,
			phaseNum:     1,
			wantComplete: true,
		},
		"phase not found": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        status: Completed
`,
			phaseNum: 99,
			wantErr:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			tasksPath := filepath.Join(dir, "tasks.yaml")
			require.NoError(t, os.WriteFile(tasksPath, []byte(tc.content), 0644))

			complete, err := IsPhaseComplete(tasksPath, tc.phaseNum)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantComplete, complete)
		})
	}
}

func TestGetActionablePhases(t *testing.T) {
	content := `phases:
  - number: 1
    title: Complete Phase
    tasks:
      - id: T001
        status: Completed
  - number: 2
    title: Actionable Phase
    tasks:
      - id: T002
        status: Pending
  - number: 3
    title: Blocked Phase
    tasks:
      - id: T003
        status: Blocked
  - number: 4
    title: Another Actionable
    tasks:
      - id: T004
        status: InProgress
`

	dir := t.TempDir()
	tasksPath := filepath.Join(dir, "tasks.yaml")
	require.NoError(t, os.WriteFile(tasksPath, []byte(content), 0644))

	actionable, err := GetActionablePhases(tasksPath)
	require.NoError(t, err)

	// Should only return phases 2 and 4
	require.Len(t, actionable, 2)
	assert.Equal(t, 2, actionable[0].Number)
	assert.Equal(t, "Actionable Phase", actionable[0].Title)
	assert.Equal(t, 4, actionable[1].Number)
	assert.Equal(t, "Another Actionable", actionable[1].Title)
}

func TestGetFirstIncompletePhase(t *testing.T) {
	tests := map[string]struct {
		content   string
		wantPhase int
		wantNil   bool
	}{
		"first phase incomplete": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        status: Pending
  - number: 2
    title: Phase 2
    tasks:
      - id: T002
        status: Completed
`,
			wantPhase: 1,
		},
		"second phase incomplete": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        status: Completed
  - number: 2
    title: Phase 2
    tasks:
      - id: T002
        status: Pending
`,
			wantPhase: 2,
		},
		"all phases complete": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        status: Completed
  - number: 2
    title: Phase 2
    tasks:
      - id: T002
        status: Blocked
`,
			wantPhase: 0,
			wantNil:   true,
		},
		"skip completed and blocked": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        status: Completed
  - number: 2
    title: Phase 2
    tasks:
      - id: T002
        status: Blocked
  - number: 3
    title: Phase 3
    tasks:
      - id: T003
        status: InProgress
`,
			wantPhase: 3,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			tasksPath := filepath.Join(dir, "tasks.yaml")
			require.NoError(t, os.WriteFile(tasksPath, []byte(tc.content), 0644))

			phaseNum, phaseInfo, err := GetFirstIncompletePhase(tasksPath)
			require.NoError(t, err)

			assert.Equal(t, tc.wantPhase, phaseNum)
			if tc.wantNil {
				assert.Nil(t, phaseInfo)
			} else {
				assert.NotNil(t, phaseInfo)
				assert.Equal(t, tc.wantPhase, phaseInfo.Number)
			}
		})
	}
}

func TestGetTotalPhases(t *testing.T) {
	tests := map[string]struct {
		content    string
		wantPhases int
	}{
		"multiple phases": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks: []
  - number: 2
    title: Phase 2
    tasks: []
  - number: 3
    title: Phase 3
    tasks: []
`,
			wantPhases: 3,
		},
		"single phase": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks: []
`,
			wantPhases: 1,
		},
		"no phases": {
			content:    `phases: []`,
			wantPhases: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			tasksPath := filepath.Join(dir, "tasks.yaml")
			require.NoError(t, os.WriteFile(tasksPath, []byte(tc.content), 0644))

			total, err := GetTotalPhases(tasksPath)
			require.NoError(t, err)
			assert.Equal(t, tc.wantPhases, total)
		})
	}
}

func TestPhaseInfo_IsComplete(t *testing.T) {
	tests := map[string]struct {
		info PhaseInfo
		want bool
	}{
		"no actionable tasks": {
			info: PhaseInfo{
				TotalTasks:      3,
				CompletedTasks:  2,
				BlockedTasks:    1,
				ActionableTasks: 0,
			},
			want: true,
		},
		"has actionable tasks": {
			info: PhaseInfo{
				TotalTasks:      3,
				CompletedTasks:  1,
				BlockedTasks:    0,
				ActionableTasks: 2,
			},
			want: false,
		},
		"empty phase": {
			info: PhaseInfo{
				TotalTasks:      0,
				CompletedTasks:  0,
				BlockedTasks:    0,
				ActionableTasks: 0,
			},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.info.IsComplete()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetAllTasks(t *testing.T) {
	content := `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        title: Task 1
        status: Completed
      - id: T002
        title: Task 2
        status: Pending
  - number: 2
    title: Phase 2
    tasks:
      - id: T003
        title: Task 3
        status: InProgress
`

	dir := t.TempDir()
	tasksPath := filepath.Join(dir, "tasks.yaml")
	require.NoError(t, os.WriteFile(tasksPath, []byte(content), 0644))

	tasks, err := GetAllTasks(tasksPath)
	require.NoError(t, err)
	require.Len(t, tasks, 3)

	assert.Equal(t, "T001", tasks[0].ID)
	assert.Equal(t, "T002", tasks[1].ID)
	assert.Equal(t, "T003", tasks[2].ID)
}

func TestGetTaskByID(t *testing.T) {
	tasks := []TaskItem{
		{ID: "T001", Title: "Task 1", Status: "Completed"},
		{ID: "T002", Title: "Task 2", Status: "Pending"},
		{ID: "T003", Title: "Task 3", Status: "InProgress"},
	}

	tests := map[string]struct {
		id        string
		wantTitle string
		wantErr   bool
	}{
		"find first task": {
			id:        "T001",
			wantTitle: "Task 1",
			wantErr:   false,
		},
		"find middle task": {
			id:        "T002",
			wantTitle: "Task 2",
			wantErr:   false,
		},
		"find last task": {
			id:        "T003",
			wantTitle: "Task 3",
			wantErr:   false,
		},
		"task not found": {
			id:      "T999",
			wantErr: true,
		},
		"case sensitive - lowercase fails": {
			id:      "t001",
			wantErr: true,
		},
		"empty id": {
			id:      "",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			task, err := GetTaskByID(tasks, tc.id)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
				return
			}

			require.NoError(t, err)
			require.NotNil(t, task)
			assert.Equal(t, tc.wantTitle, task.Title)
		})
	}
}

func TestGetTasksInDependencyOrder(t *testing.T) {
	tests := map[string]struct {
		tasks       []TaskItem
		wantOrder   []string
		wantErr     bool
		errContains string
	}{
		"no dependencies - original order": {
			tasks: []TaskItem{
				{ID: "T001", Dependencies: nil},
				{ID: "T002", Dependencies: nil},
				{ID: "T003", Dependencies: nil},
			},
			wantOrder: []string{"T001", "T002", "T003"},
		},
		"simple linear dependency": {
			tasks: []TaskItem{
				{ID: "T001", Dependencies: nil},
				{ID: "T002", Dependencies: []string{"T001"}},
				{ID: "T003", Dependencies: []string{"T002"}},
			},
			wantOrder: []string{"T001", "T002", "T003"},
		},
		"reverse order with dependencies": {
			tasks: []TaskItem{
				{ID: "T003", Dependencies: []string{"T002"}},
				{ID: "T002", Dependencies: []string{"T001"}},
				{ID: "T001", Dependencies: nil},
			},
			wantOrder: []string{"T001", "T002", "T003"},
		},
		"multiple dependencies - fan-in": {
			tasks: []TaskItem{
				{ID: "T001", Dependencies: nil},
				{ID: "T002", Dependencies: nil},
				{ID: "T003", Dependencies: []string{"T001", "T002"}},
			},
			wantOrder: []string{"T001", "T002", "T003"},
		},
		"complex dependency graph": {
			tasks: []TaskItem{
				{ID: "T005", Dependencies: []string{"T003", "T004"}},
				{ID: "T004", Dependencies: []string{"T002"}},
				{ID: "T003", Dependencies: []string{"T001"}},
				{ID: "T002", Dependencies: []string{"T001"}},
				{ID: "T001", Dependencies: nil},
			},
			wantOrder: []string{"T001", "T003", "T002", "T004", "T005"},
		},
		"circular dependency - self": {
			tasks: []TaskItem{
				{ID: "T001", Dependencies: []string{"T001"}},
			},
			wantErr:     true,
			errContains: "circular dependency",
		},
		"circular dependency - two tasks": {
			tasks: []TaskItem{
				{ID: "T001", Dependencies: []string{"T002"}},
				{ID: "T002", Dependencies: []string{"T001"}},
			},
			wantErr:     true,
			errContains: "circular dependency",
		},
		"circular dependency - three tasks": {
			tasks: []TaskItem{
				{ID: "T001", Dependencies: []string{"T003"}},
				{ID: "T002", Dependencies: []string{"T001"}},
				{ID: "T003", Dependencies: []string{"T002"}},
			},
			wantErr:     true,
			errContains: "circular dependency",
		},
		"missing dependency - skipped silently": {
			tasks: []TaskItem{
				{ID: "T001", Dependencies: nil},
				{ID: "T002", Dependencies: []string{"T999"}}, // T999 doesn't exist
			},
			wantOrder: []string{"T001", "T002"},
		},
		"empty task list": {
			tasks:     []TaskItem{},
			wantOrder: []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := GetTasksInDependencyOrder(tc.tasks)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.Len(t, result, len(tc.wantOrder))

			// Extract IDs from result
			gotOrder := make([]string, len(result))
			for i, task := range result {
				gotOrder[i] = task.ID
			}

			assert.Equal(t, tc.wantOrder, gotOrder)
		})
	}
}

// Tests for GetTasksForPhase function
func TestGetTasksForPhase(t *testing.T) {
	tests := map[string]struct {
		content     string
		phaseNumber int
		wantTaskIDs []string
		wantErr     bool
		errContains string
	}{
		"valid phase returns correct tasks": {
			content: `phases:
  - number: 1
    title: Setup
    tasks:
      - id: T001
        title: Task 1
        status: Pending
      - id: T002
        title: Task 2
        status: Completed
  - number: 2
    title: Implementation
    tasks:
      - id: T003
        title: Task 3
        status: Pending
`,
			phaseNumber: 1,
			wantTaskIDs: []string{"T001", "T002"},
		},
		"second phase returns only its tasks": {
			content: `phases:
  - number: 1
    title: Setup
    tasks:
      - id: T001
        title: Task 1
        status: Pending
  - number: 2
    title: Implementation
    tasks:
      - id: T002
        title: Task 2
        status: Pending
      - id: T003
        title: Task 3
        status: InProgress
      - id: T004
        title: Task 4
        status: Completed
`,
			phaseNumber: 2,
			wantTaskIDs: []string{"T002", "T003", "T004"},
		},
		"empty phase returns empty slice": {
			content: `phases:
  - number: 1
    title: Empty Phase
    tasks: []
  - number: 2
    title: Non-empty
    tasks:
      - id: T001
        title: Task 1
        status: Pending
`,
			phaseNumber: 1,
			wantTaskIDs: []string{},
		},
		"invalid phase number returns error": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        title: Task 1
        status: Pending
`,
			phaseNumber: 99,
			wantErr:     true,
			errContains: "phase 99 not found",
		},
		"negative phase number returns error": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        title: Task 1
        status: Pending
`,
			phaseNumber: -1,
			wantErr:     true,
			errContains: "not found",
		},
		"preserves task fields": {
			content: `phases:
  - number: 1
    title: Test Phase
    tasks:
      - id: T001
        title: Test Task
        status: Pending
        type: implementation
        parallel: true
        story_id: US-001
        file_path: src/main.go
        dependencies:
          - T000
        acceptance_criteria:
          - Criterion 1
`,
			phaseNumber: 1,
			wantTaskIDs: []string{"T001"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			tasksPath := filepath.Join(dir, "tasks.yaml")
			require.NoError(t, os.WriteFile(tasksPath, []byte(tc.content), 0644))

			tasks, err := GetTasksForPhase(tasksPath, tc.phaseNumber)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.Len(t, tasks, len(tc.wantTaskIDs))

			// Verify task IDs match
			gotIDs := make([]string, len(tasks))
			for i, task := range tasks {
				gotIDs[i] = task.ID
			}
			assert.Equal(t, tc.wantTaskIDs, gotIDs)
		})
	}
}

func TestGetTasksForPhase_FileErrors(t *testing.T) {
	t.Run("file not found", func(t *testing.T) {
		_, err := GetTasksForPhase("/nonexistent/path/tasks.yaml", 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read")
	})

	t.Run("invalid yaml", func(t *testing.T) {
		dir := t.TempDir()
		tasksPath := filepath.Join(dir, "tasks.yaml")
		require.NoError(t, os.WriteFile(tasksPath, []byte("{{invalid yaml"), 0644))

		_, err := GetTasksForPhase(tasksPath, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse")
	})
}

func BenchmarkGetTasksForPhase(b *testing.B) {
	// Create a realistic tasks.yaml with multiple phases
	content := `phases:
  - number: 1
    title: Setup
    tasks:
      - id: T001
        title: Task 1
        status: Completed
      - id: T002
        title: Task 2
        status: Completed
  - number: 2
    title: Implementation
    tasks:
      - id: T003
        title: Task 3
        status: Pending
      - id: T004
        title: Task 4
        status: InProgress
      - id: T005
        title: Task 5
        status: Pending
  - number: 3
    title: Testing
    tasks:
      - id: T006
        title: Task 6
        status: Pending
      - id: T007
        title: Task 7
        status: Pending
`
	dir := b.TempDir()
	tasksPath := filepath.Join(dir, "tasks.yaml")
	if err := os.WriteFile(tasksPath, []byte(content), 0644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetTasksForPhase(tasksPath, 2)
	}
}

func TestValidateTaskDependenciesMet(t *testing.T) {
	allTasks := []TaskItem{
		{ID: "T001", Status: "Completed"},
		{ID: "T002", Status: "Done"},
		{ID: "T003", Status: "Pending"},
		{ID: "T004", Status: "InProgress"},
		{ID: "T005", Status: "Blocked"},
	}

	tests := map[string]struct {
		task      TaskItem
		wantMet   bool
		wantUnmet []string
	}{
		"no dependencies - always met": {
			task:      TaskItem{ID: "T006", Dependencies: nil},
			wantMet:   true,
			wantUnmet: nil,
		},
		"empty dependencies - always met": {
			task:      TaskItem{ID: "T006", Dependencies: []string{}},
			wantMet:   true,
			wantUnmet: nil,
		},
		"single completed dependency": {
			task:      TaskItem{ID: "T006", Dependencies: []string{"T001"}},
			wantMet:   true,
			wantUnmet: nil,
		},
		"single done dependency": {
			task:      TaskItem{ID: "T006", Dependencies: []string{"T002"}},
			wantMet:   true,
			wantUnmet: nil,
		},
		"single pending dependency - unmet": {
			task:      TaskItem{ID: "T006", Dependencies: []string{"T003"}},
			wantMet:   false,
			wantUnmet: []string{"T003"},
		},
		"single in-progress dependency - unmet": {
			task:      TaskItem{ID: "T006", Dependencies: []string{"T004"}},
			wantMet:   false,
			wantUnmet: []string{"T004"},
		},
		"single blocked dependency - unmet": {
			task:      TaskItem{ID: "T006", Dependencies: []string{"T005"}},
			wantMet:   false,
			wantUnmet: []string{"T005"},
		},
		"multiple completed dependencies - all met": {
			task:      TaskItem{ID: "T006", Dependencies: []string{"T001", "T002"}},
			wantMet:   true,
			wantUnmet: nil,
		},
		"mixed dependencies - some unmet": {
			task:      TaskItem{ID: "T006", Dependencies: []string{"T001", "T003", "T004"}},
			wantMet:   false,
			wantUnmet: []string{"T003", "T004"},
		},
		"non-existent dependency": {
			task:      TaskItem{ID: "T006", Dependencies: []string{"T999"}},
			wantMet:   false,
			wantUnmet: []string{"T999 (not found)"},
		},
		"mixed with non-existent": {
			task:      TaskItem{ID: "T006", Dependencies: []string{"T001", "T999", "T003"}},
			wantMet:   false,
			wantUnmet: []string{"T999 (not found)", "T003"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			met, unmet := ValidateTaskDependenciesMet(tc.task, allTasks)

			assert.Equal(t, tc.wantMet, met)
			assert.Equal(t, tc.wantUnmet, unmet)
		})
	}
}
