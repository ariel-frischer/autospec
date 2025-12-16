package validation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPhaseDisplayInfo_TaskIDsString(t *testing.T) {
	tests := map[string]struct {
		taskIDs []string
		want    string
	}{
		"multiple task IDs": {
			taskIDs: []string{"T001", "T002", "T003"},
			want:    "T001, T002, T003",
		},
		"single task ID": {
			taskIDs: []string{"T001"},
			want:    "T001",
		},
		"empty task IDs": {
			taskIDs: []string{},
			want:    "",
		},
		"nil task IDs": {
			taskIDs: nil,
			want:    "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			info := PhaseDisplayInfo{TaskIDs: tc.taskIDs}
			got := info.TaskIDsString()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFormatPhaseHeader(t *testing.T) {
	tests := map[string]struct {
		info         PhaseDisplayInfo
		wantContains []string
	}{
		"basic header with tasks": {
			info: PhaseDisplayInfo{
				PhaseNumber:    1,
				TotalPhases:    5,
				Title:          "Setup Phase",
				TaskIDs:        []string{"T001", "T002", "T003"},
				CompletedCount: 1,
				BlockedCount:   0,
				PendingCount:   2,
			},
			wantContains: []string{
				"[Phase 1/5] Setup Phase",
				"-> 3 tasks: T001, T002, T003",
				"-> Status: 1 completed, 0 blocked, 2 pending",
			},
		},
		"header with no tasks": {
			info: PhaseDisplayInfo{
				PhaseNumber:    2,
				TotalPhases:    3,
				Title:          "Empty Phase",
				TaskIDs:        []string{},
				CompletedCount: 0,
				BlockedCount:   0,
				PendingCount:   0,
			},
			wantContains: []string{
				"[Phase 2/3] Empty Phase",
				"-> 0 tasks",
				"-> Status: 0 completed, 0 blocked, 0 pending",
			},
		},
		"header with all completed": {
			info: PhaseDisplayInfo{
				PhaseNumber:    3,
				TotalPhases:    3,
				Title:          "Completed Phase",
				TaskIDs:        []string{"T010", "T011"},
				CompletedCount: 2,
				BlockedCount:   0,
				PendingCount:   0,
			},
			wantContains: []string{
				"[Phase 3/3] Completed Phase",
				"-> 2 tasks: T010, T011",
				"-> Status: 2 completed, 0 blocked, 0 pending",
			},
		},
		"header with blocked tasks": {
			info: PhaseDisplayInfo{
				PhaseNumber:    1,
				TotalPhases:    2,
				Title:          "Blocked Phase",
				TaskIDs:        []string{"T001", "T002", "T003", "T004"},
				CompletedCount: 2,
				BlockedCount:   1,
				PendingCount:   1,
			},
			wantContains: []string{
				"[Phase 1/2] Blocked Phase",
				"-> 4 tasks: T001, T002, T003, T004",
				"-> Status: 2 completed, 1 blocked, 1 pending",
			},
		},
		"single task": {
			info: PhaseDisplayInfo{
				PhaseNumber:    1,
				TotalPhases:    1,
				Title:          "Single Task Phase",
				TaskIDs:        []string{"T001"},
				CompletedCount: 0,
				BlockedCount:   0,
				PendingCount:   1,
			},
			wantContains: []string{
				"[Phase 1/1] Single Task Phase",
				"-> 1 tasks: T001",
				"-> Status: 0 completed, 0 blocked, 1 pending",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := FormatPhaseHeader(tc.info)
			for _, want := range tc.wantContains {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestFormatPhaseHeader_Format(t *testing.T) {
	// Test exact output format
	info := PhaseDisplayInfo{
		PhaseNumber:    1,
		TotalPhases:    3,
		Title:          "Test Phase",
		TaskIDs:        []string{"T001", "T002"},
		CompletedCount: 1,
		BlockedCount:   0,
		PendingCount:   1,
	}

	got := FormatPhaseHeader(info)

	// Verify it has exactly 3 lines
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	// The output ends without trailing newline on last line
	assert.Len(t, lines, 3, "FormatPhaseHeader should produce 3 lines")

	// Verify line prefixes
	assert.True(t, strings.HasPrefix(lines[0], "[Phase"))
	assert.True(t, strings.HasPrefix(lines[1], "  ->"))
	assert.True(t, strings.HasPrefix(lines[2], "  ->"))
}

func TestFormatPhaseCompletion(t *testing.T) {
	tests := map[string]struct {
		phaseNumber int
		completed   int
		total       int
		blocked     int
		want        string
	}{
		"all completed no blocked": {
			phaseNumber: 1,
			completed:   5,
			total:       5,
			blocked:     0,
			want:        "Phase 1 complete (5/5 tasks completed)",
		},
		"partial completion no blocked": {
			phaseNumber: 2,
			completed:   3,
			total:       5,
			blocked:     0,
			want:        "Phase 2 complete (3/5 tasks completed)",
		},
		"with blocked tasks": {
			phaseNumber: 3,
			completed:   4,
			total:       6,
			blocked:     2,
			want:        "Phase 3 complete (4/6 tasks completed, 2 blocked)",
		},
		"zero completed with blocked": {
			phaseNumber: 1,
			completed:   0,
			total:       3,
			blocked:     3,
			want:        "Phase 1 complete (0/3 tasks completed, 3 blocked)",
		},
		"single task completed": {
			phaseNumber: 5,
			completed:   1,
			total:       1,
			blocked:     0,
			want:        "Phase 5 complete (1/1 tasks completed)",
		},
		"single blocked": {
			phaseNumber: 1,
			completed:   2,
			total:       3,
			blocked:     1,
			want:        "Phase 1 complete (2/3 tasks completed, 1 blocked)",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := FormatPhaseCompletion(tc.phaseNumber, tc.completed, tc.total, tc.blocked)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFormatPhaseCompletion_OmitsBlockedWhenZero(t *testing.T) {
	got := FormatPhaseCompletion(1, 5, 5, 0)
	assert.NotContains(t, got, "blocked", "should not contain 'blocked' when blocked count is zero")
}

func TestFormatPhaseCompletion_IncludesBlockedWhenNonZero(t *testing.T) {
	got := FormatPhaseCompletion(1, 3, 5, 2)
	assert.Contains(t, got, "blocked", "should contain 'blocked' when blocked count is non-zero")
	assert.Contains(t, got, "2 blocked")
}

func TestBuildPhaseDisplayInfo(t *testing.T) {
	tests := map[string]struct {
		phaseInfo   PhaseInfo
		totalPhases int
		taskIDs     []string
		wantPending int
	}{
		"calculates pending correctly": {
			phaseInfo: PhaseInfo{
				Number:         1,
				Title:          "Test Phase",
				TotalTasks:     5,
				CompletedTasks: 2,
				BlockedTasks:   1,
			},
			totalPhases: 3,
			taskIDs:     []string{"T001", "T002", "T003", "T004", "T005"},
			wantPending: 2, // 5 - 2 - 1 = 2
		},
		"handles all completed": {
			phaseInfo: PhaseInfo{
				Number:         2,
				Title:          "Complete Phase",
				TotalTasks:     3,
				CompletedTasks: 3,
				BlockedTasks:   0,
			},
			totalPhases: 5,
			taskIDs:     []string{"T001", "T002", "T003"},
			wantPending: 0,
		},
		"handles all blocked": {
			phaseInfo: PhaseInfo{
				Number:         1,
				Title:          "Blocked Phase",
				TotalTasks:     2,
				CompletedTasks: 0,
				BlockedTasks:   2,
			},
			totalPhases: 2,
			taskIDs:     []string{"T001", "T002"},
			wantPending: 0,
		},
		"handles empty phase": {
			phaseInfo: PhaseInfo{
				Number:         1,
				Title:          "Empty Phase",
				TotalTasks:     0,
				CompletedTasks: 0,
				BlockedTasks:   0,
			},
			totalPhases: 1,
			taskIDs:     []string{},
			wantPending: 0,
		},
		"handles negative calculation (clamps to zero)": {
			phaseInfo: PhaseInfo{
				Number:         1,
				Title:          "Inconsistent Phase",
				TotalTasks:     2,
				CompletedTasks: 2,
				BlockedTasks:   2, // More than total (shouldn't happen but handle gracefully)
			},
			totalPhases: 1,
			taskIDs:     []string{"T001", "T002"},
			wantPending: 0, // Clamped to 0
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := BuildPhaseDisplayInfo(tc.phaseInfo, tc.totalPhases, tc.taskIDs)

			assert.Equal(t, tc.phaseInfo.Number, got.PhaseNumber)
			assert.Equal(t, tc.totalPhases, got.TotalPhases)
			assert.Equal(t, tc.phaseInfo.Title, got.Title)
			assert.Equal(t, tc.taskIDs, got.TaskIDs)
			assert.Equal(t, tc.phaseInfo.CompletedTasks, got.CompletedCount)
			assert.Equal(t, tc.phaseInfo.BlockedTasks, got.BlockedCount)
			assert.Equal(t, tc.wantPending, got.PendingCount)
		})
	}
}

func BenchmarkFormatPhaseHeader(b *testing.B) {
	info := PhaseDisplayInfo{
		PhaseNumber:    3,
		TotalPhases:    7,
		Title:          "Implementation Phase",
		TaskIDs:        []string{"T010", "T011", "T012", "T013", "T014"},
		CompletedCount: 2,
		BlockedCount:   1,
		PendingCount:   2,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FormatPhaseHeader(info)
	}
}

func BenchmarkFormatPhaseCompletion(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FormatPhaseCompletion(3, 4, 5, 1)
	}
}

func BenchmarkBuildPhaseDisplayInfo(b *testing.B) {
	phaseInfo := PhaseInfo{
		Number:         3,
		Title:          "Test Phase",
		TotalTasks:     10,
		CompletedTasks: 5,
		BlockedTasks:   2,
	}
	taskIDs := []string{"T001", "T002", "T003", "T004", "T005", "T006", "T007", "T008", "T009", "T010"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildPhaseDisplayInfo(phaseInfo, 5, taskIDs)
	}
}
