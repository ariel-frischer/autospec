// Package util tests the view command implementation.
// Related: internal/cli/util/view.go
// Tags: util, cli, view, dashboard

package util

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveLimit(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		flagValue   int
		configValue int
		want        int
	}{
		"flag value takes precedence over config": {
			flagValue:   10,
			configValue: 5,
			want:        10,
		},
		"config value used when flag is zero": {
			flagValue:   0,
			configValue: 7,
			want:        7,
		},
		"default of 5 when both are zero": {
			flagValue:   0,
			configValue: 0,
			want:        5,
		},
		"negative flag falls back to config": {
			flagValue:   -1,
			configValue: 3,
			want:        3,
		},
		"negative config falls back to default": {
			flagValue:   0,
			configValue: -2,
			want:        5,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := resolveLimit(tt.flagValue, tt.configValue)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestComputeDashboardStats(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		summaries []SpecSummary
		want      DashboardStats
	}{
		"empty summaries": {
			summaries: []SpecSummary{},
			want:      DashboardStats{TotalSpecs: 0},
		},
		"all completed by status": {
			summaries: []SpecSummary{
				{Name: "spec-1", Status: "Completed"},
				{Name: "spec-2", Status: "completed"},
				{Name: "spec-3", Status: "Done"},
			},
			want: DashboardStats{
				TotalSpecs:     3,
				CompletedCount: 3,
			},
		},
		"all skipped": {
			summaries: []SpecSummary{
				{Name: "spec-1", Status: "Rejected"},
				{Name: "spec-2", Status: "Skipped"},
			},
			want: DashboardStats{
				TotalSpecs:   2,
				SkippedCount: 2,
			},
		},
		"all in progress": {
			summaries: []SpecSummary{
				{Name: "spec-1", Status: "Draft"},
				{Name: "spec-2", Status: "In Progress"},
				{Name: "spec-3", Status: "Review"},
				{Name: "spec-4", Status: "Unknown"},
			},
			want: DashboardStats{
				TotalSpecs:      4,
				InProgressCount: 4,
			},
		},
		"mixed statuses": {
			summaries: []SpecSummary{
				{Name: "spec-1", Status: "Completed"},
				{Name: "spec-2", Status: "In Progress"},
				{Name: "spec-3", Status: "Rejected"},
				{Name: "spec-4", Status: "Draft"},
			},
			want: DashboardStats{
				TotalSpecs:      4,
				CompletedCount:  1,
				InProgressCount: 2,
				SkippedCount:    1,
			},
		},
		"completed by task progress": {
			summaries: []SpecSummary{
				{Name: "spec-1", Status: "Draft", CompletedTasks: 10, TotalTasks: 10},
			},
			want: DashboardStats{
				TotalSpecs:     1,
				CompletedCount: 1,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := computeDashboardStats(tt.summaries)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsCompletedStatus(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		statusLower string
		completed   int
		total       int
		want        bool
	}{
		"completed status": {
			statusLower: "completed",
			want:        true,
		},
		"done status": {
			statusLower: "done",
			want:        true,
		},
		"complete status": {
			statusLower: "complete",
			want:        true,
		},
		"100% task completion": {
			statusLower: "draft",
			completed:   5,
			total:       5,
			want:        true,
		},
		"partial completion": {
			statusLower: "draft",
			completed:   3,
			total:       5,
			want:        false,
		},
		"zero tasks": {
			statusLower: "draft",
			completed:   0,
			total:       0,
			want:        false,
		},
		"draft status": {
			statusLower: "draft",
			want:        false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := isCompletedStatus(tt.statusLower, tt.completed, tt.total)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsSkippedStatus(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		statusLower string
		want        bool
	}{
		"rejected status":    {statusLower: "rejected", want: true},
		"skipped status":     {statusLower: "skipped", want: true},
		"completed status":   {statusLower: "completed", want: false},
		"draft status":       {statusLower: "draft", want: false},
		"in progress status": {statusLower: "in progress", want: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := isSkippedStatus(tt.statusLower)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input  string
		maxLen int
		want   string
	}{
		"short string unchanged": {
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		"exact length unchanged": {
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		"long string truncated": {
			input:  "hello world",
			maxLen: 8,
			want:   "hello...",
		},
		"empty string": {
			input:  "",
			maxLen: 10,
			want:   "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestScanSpecsDir(t *testing.T) {
	t.Parallel()

	t.Run("empty directory", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		specsDir := filepath.Join(tmpDir, "specs")
		require.NoError(t, os.MkdirAll(specsDir, 0755))

		summaries, err := scanSpecsDir(specsDir)
		require.NoError(t, err)
		assert.Empty(t, summaries)
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		t.Parallel()
		summaries, err := scanSpecsDir("/nonexistent/path")
		require.NoError(t, err)
		assert.Nil(t, summaries)
	})

	t.Run("directory with valid spec", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		specsDir := filepath.Join(tmpDir, "specs")
		specDir := filepath.Join(specsDir, "001-test-spec")
		require.NoError(t, os.MkdirAll(specDir, 0755))

		specContent := `feature:
  status: "Draft"
`
		require.NoError(t, os.WriteFile(
			filepath.Join(specDir, "spec.yaml"),
			[]byte(specContent),
			0644,
		))

		summaries, err := scanSpecsDir(specsDir)
		require.NoError(t, err)
		require.Len(t, summaries, 1)
		assert.Equal(t, "001-test-spec", summaries[0].Name)
		assert.Equal(t, "Draft", summaries[0].Status)
	})

	t.Run("directory without spec.yaml skipped", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		specsDir := filepath.Join(tmpDir, "specs")
		specDir := filepath.Join(specsDir, "001-no-spec")
		require.NoError(t, os.MkdirAll(specDir, 0755))

		// Create plan.yaml but no spec.yaml
		require.NoError(t, os.WriteFile(
			filepath.Join(specDir, "plan.yaml"),
			[]byte("summary: test"),
			0644,
		))

		summaries, err := scanSpecsDir(specsDir)
		require.NoError(t, err)
		assert.Empty(t, summaries)
	})

	t.Run("sorted by modification time", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		specsDir := filepath.Join(tmpDir, "specs")

		// Create two specs with different modification times
		spec1Dir := filepath.Join(specsDir, "001-older")
		spec2Dir := filepath.Join(specsDir, "002-newer")
		require.NoError(t, os.MkdirAll(spec1Dir, 0755))
		require.NoError(t, os.MkdirAll(spec2Dir, 0755))

		specContent := `feature:
  status: "Draft"
`
		spec1Path := filepath.Join(spec1Dir, "spec.yaml")
		spec2Path := filepath.Join(spec2Dir, "spec.yaml")

		require.NoError(t, os.WriteFile(spec1Path, []byte(specContent), 0644))
		// Set older time for spec1
		oldTime := time.Now().Add(-24 * time.Hour)
		require.NoError(t, os.Chtimes(spec1Path, oldTime, oldTime))

		require.NoError(t, os.WriteFile(spec2Path, []byte(specContent), 0644))

		summaries, err := scanSpecsDir(specsDir)
		require.NoError(t, err)
		require.Len(t, summaries, 2)
		// Newer should come first
		assert.Equal(t, "002-newer", summaries[0].Name)
		assert.Equal(t, "001-older", summaries[1].Name)
	})
}

func TestGetSpecSummary(t *testing.T) {
	t.Parallel()

	t.Run("valid spec with all artifacts", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		specDir := filepath.Join(tmpDir, "001-test-spec")
		require.NoError(t, os.MkdirAll(specDir, 0755))

		specContent := `feature:
  status: "In Progress"
`
		planContent := `summary: "Test plan"
`
		tasksContent := `phases:
  - number: 1
    title: "Setup"
    tasks:
      - id: "T1"
        title: "Task 1"
        status: "Completed"
      - id: "T2"
        title: "Task 2"
        status: "Pending"
`
		require.NoError(t, os.WriteFile(
			filepath.Join(specDir, "spec.yaml"),
			[]byte(specContent),
			0644,
		))
		require.NoError(t, os.WriteFile(
			filepath.Join(specDir, "plan.yaml"),
			[]byte(planContent),
			0644,
		))
		require.NoError(t, os.WriteFile(
			filepath.Join(specDir, "tasks.yaml"),
			[]byte(tasksContent),
			0644,
		))

		summary, err := getSpecSummary(specDir, "001-test-spec")
		require.NoError(t, err)
		assert.Equal(t, "001-test-spec", summary.Name)
		assert.Equal(t, "In Progress", summary.Status)
		assert.Equal(t, 1, summary.CompletedTasks)
		assert.Equal(t, 2, summary.TotalTasks)
		assert.Equal(t, "1/2 tasks", summary.TaskProgress)
		assert.Contains(t, summary.ArtifactsPresent, "spec.yaml")
		assert.Contains(t, summary.ArtifactsPresent, "plan.yaml")
		assert.Contains(t, summary.ArtifactsPresent, "tasks.yaml")
	})

	t.Run("spec without tasks.yaml", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		specDir := filepath.Join(tmpDir, "001-no-tasks")
		require.NoError(t, os.MkdirAll(specDir, 0755))

		specContent := `feature:
  status: "Draft"
`
		require.NoError(t, os.WriteFile(
			filepath.Join(specDir, "spec.yaml"),
			[]byte(specContent),
			0644,
		))

		summary, err := getSpecSummary(specDir, "001-no-tasks")
		require.NoError(t, err)
		assert.Equal(t, "no tasks", summary.TaskProgress)
		assert.Equal(t, 0, summary.TotalTasks)
	})

	t.Run("missing spec.yaml returns error", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		specDir := filepath.Join(tmpDir, "001-missing")
		require.NoError(t, os.MkdirAll(specDir, 0755))

		_, err := getSpecSummary(specDir, "001-missing")
		assert.Error(t, err)
	})

	t.Run("malformed spec.yaml returns parse error status", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		specDir := filepath.Join(tmpDir, "001-malformed")
		require.NoError(t, os.MkdirAll(specDir, 0755))

		malformedContent := `feature: [invalid yaml`
		require.NoError(t, os.WriteFile(
			filepath.Join(specDir, "spec.yaml"),
			[]byte(malformedContent),
			0644,
		))

		summary, err := getSpecSummary(specDir, "001-malformed")
		require.NoError(t, err)
		assert.Equal(t, "parse error", summary.Status)
	})
}

func TestParseSpecStatus(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yamlContent string
		want        string
	}{
		"valid status": {
			yamlContent: `feature:
  status: "Draft"
`,
			want: "Draft",
		},
		"empty status": {
			yamlContent: `feature:
  status: ""
`,
			want: "Unknown",
		},
		"missing status field": {
			yamlContent: `feature:
  name: "Test"
`,
			want: "Unknown",
		},
		"malformed yaml": {
			yamlContent: `feature: [invalid`,
			want:        "parse error",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			specPath := filepath.Join(tmpDir, "spec.yaml")
			require.NoError(t, os.WriteFile(specPath, []byte(tt.yamlContent), 0644))

			got := parseSpecStatus(specPath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDetectArtifacts(t *testing.T) {
	t.Parallel()

	t.Run("all artifacts present", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "spec.yaml"), []byte(""), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "plan.yaml"), []byte(""), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "tasks.yaml"), []byte(""), 0644))

		artifacts := detectArtifacts(tmpDir)
		assert.Len(t, artifacts, 3)
		assert.Contains(t, artifacts, "spec.yaml")
		assert.Contains(t, artifacts, "plan.yaml")
		assert.Contains(t, artifacts, "tasks.yaml")
	})

	t.Run("only spec.yaml present", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "spec.yaml"), []byte(""), 0644))

		artifacts := detectArtifacts(tmpDir)
		assert.Len(t, artifacts, 1)
		assert.Contains(t, artifacts, "spec.yaml")
	})

	t.Run("empty directory", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		artifacts := detectArtifacts(tmpDir)
		assert.Empty(t, artifacts)
	})
}

func TestGetTaskProgress(t *testing.T) {
	t.Parallel()

	t.Run("valid tasks.yaml", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		tasksContent := `phases:
  - number: 1
    title: "Setup"
    tasks:
      - id: "T1"
        title: "Task 1"
        status: "Completed"
      - id: "T2"
        title: "Task 2"
        status: "Completed"
      - id: "T3"
        title: "Task 3"
        status: "Pending"
`
		require.NoError(t, os.WriteFile(
			filepath.Join(tmpDir, "tasks.yaml"),
			[]byte(tasksContent),
			0644,
		))

		completed, total, progress := getTaskProgress(tmpDir)
		assert.Equal(t, 2, completed)
		assert.Equal(t, 3, total)
		assert.Equal(t, "2/3 tasks", progress)
	})

	t.Run("no tasks.yaml", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		completed, total, progress := getTaskProgress(tmpDir)
		assert.Equal(t, 0, completed)
		assert.Equal(t, 0, total)
		assert.Equal(t, "no tasks", progress)
	})

	t.Run("empty phases", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		tasksContent := `phases: []
`
		require.NoError(t, os.WriteFile(
			filepath.Join(tmpDir, "tasks.yaml"),
			[]byte(tasksContent),
			0644,
		))

		completed, total, progress := getTaskProgress(tmpDir)
		assert.Equal(t, 0, completed)
		assert.Equal(t, 0, total)
		assert.Equal(t, "0 tasks", progress)
	})
}

func TestGetLatestModTime(t *testing.T) {
	t.Parallel()

	t.Run("returns latest file time", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		// Create files with different mod times
		file1 := filepath.Join(tmpDir, "file1.txt")
		file2 := filepath.Join(tmpDir, "file2.txt")

		require.NoError(t, os.WriteFile(file1, []byte("old"), 0644))
		oldTime := time.Now().Add(-24 * time.Hour)
		require.NoError(t, os.Chtimes(file1, oldTime, oldTime))

		require.NoError(t, os.WriteFile(file2, []byte("new"), 0644))

		latest := getLatestModTime(tmpDir)
		// Latest should be recent (within last minute)
		assert.True(t, time.Since(latest) < time.Minute)
	})

	t.Run("empty directory returns zero time", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		latest := getLatestModTime(tmpDir)
		assert.True(t, latest.IsZero())
	})

	t.Run("skips subdirectories", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		// Create a subdirectory
		subDir := filepath.Join(tmpDir, "subdir")
		require.NoError(t, os.MkdirAll(subDir, 0755))

		// Create file with old time
		file1 := filepath.Join(tmpDir, "file1.txt")
		require.NoError(t, os.WriteFile(file1, []byte("old"), 0644))
		oldTime := time.Now().Add(-24 * time.Hour)
		require.NoError(t, os.Chtimes(file1, oldTime, oldTime))

		latest := getLatestModTime(tmpDir)
		// Should use file1 time, not subdir time
		assert.True(t, time.Since(latest) > 23*time.Hour)
	})
}
