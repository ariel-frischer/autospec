package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGetContextFilePath(t *testing.T) {
	// Save current directory
	origDir, err := os.Getwd()
	require.NoError(t, err)

	t.Run("creates directory and returns correct path", func(t *testing.T) {
		// Use temp directory as working directory
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		path, err := GetContextFilePath(3)
		require.NoError(t, err)

		// Should be in .autospec/context/
		assert.Contains(t, path, "phase-3.yaml")

		// Directory should exist
		contextDir := filepath.Dir(path)
		info, err := os.Stat(contextDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("returns different paths for different phases", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		path1, err := GetContextFilePath(1)
		require.NoError(t, err)

		path2, err := GetContextFilePath(2)
		require.NoError(t, err)

		assert.NotEqual(t, path1, path2)
		assert.Contains(t, path1, "phase-1.yaml")
		assert.Contains(t, path2, "phase-2.yaml")
	})
}

func TestBuildPhaseContext(t *testing.T) {
	t.Run("builds context from spec, plan, and tasks files", func(t *testing.T) {
		specDir := t.TempDir()

		// Create spec.yaml
		specContent := `feature:
  branch: "test-branch"
  status: "Draft"
user_stories:
  - id: "US-001"
    title: "Test Story"
`
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte(specContent), 0644))

		// Create plan.yaml
		planContent := `plan:
  branch: "test-branch"
summary: |
  Test plan summary
technical_context:
  language: "Go"
`
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte(planContent), 0644))

		// Create tasks.yaml
		tasksContent := `phases:
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
`
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "tasks.yaml"), []byte(tasksContent), 0644))

		// Build context for phase 1
		ctx, err := BuildPhaseContext(specDir, 1, 2)
		require.NoError(t, err)

		assert.Equal(t, 1, ctx.Phase)
		assert.Equal(t, 2, ctx.TotalPhases)
		assert.Equal(t, specDir, ctx.SpecDir)

		// Verify spec content was loaded
		assert.NotNil(t, ctx.Spec)
		feature, ok := ctx.Spec["feature"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "test-branch", feature["branch"])

		// Verify plan content was loaded
		assert.NotNil(t, ctx.Plan)
		plan, ok := ctx.Plan["plan"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "test-branch", plan["branch"])

		// Verify only phase 1 tasks were included
		assert.Len(t, ctx.Tasks, 2)
		assert.Equal(t, "T001", ctx.Tasks[0]["id"])
		assert.Equal(t, "T002", ctx.Tasks[1]["id"])
	})

	t.Run("extracts only tasks for specified phase", func(t *testing.T) {
		specDir := t.TempDir()

		// Create minimal spec.yaml
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte("feature:\n  branch: test\n"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte("plan:\n  branch: test\n"), 0644))

		// Create tasks.yaml with multiple phases
		tasksContent := `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        title: Task 1
  - number: 2
    title: Phase 2
    tasks:
      - id: T002
        title: Task 2
      - id: T003
        title: Task 3
  - number: 3
    title: Phase 3
    tasks:
      - id: T004
        title: Task 4
`
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "tasks.yaml"), []byte(tasksContent), 0644))

		// Build context for phase 2
		ctx, err := BuildPhaseContext(specDir, 2, 3)
		require.NoError(t, err)

		assert.Equal(t, 2, ctx.Phase)
		assert.Len(t, ctx.Tasks, 2)
		assert.Equal(t, "T002", ctx.Tasks[0]["id"])
		assert.Equal(t, "T003", ctx.Tasks[1]["id"])
	})

	t.Run("returns error for missing spec.yaml", func(t *testing.T) {
		specDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte("plan:\n  branch: test\n"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "tasks.yaml"), []byte("phases: []\n"), 0644))

		_, err := BuildPhaseContext(specDir, 1, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "spec.yaml")
	})

	t.Run("returns error for missing plan.yaml", func(t *testing.T) {
		specDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte("feature:\n  branch: test\n"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "tasks.yaml"), []byte("phases: []\n"), 0644))

		_, err := BuildPhaseContext(specDir, 1, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "plan.yaml")
	})

	t.Run("returns error for missing tasks.yaml", func(t *testing.T) {
		specDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte("feature:\n  branch: test\n"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte("plan:\n  branch: test\n"), 0644))

		_, err := BuildPhaseContext(specDir, 1, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tasks.yaml")
	})

	t.Run("returns error for invalid phase number", func(t *testing.T) {
		specDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte("feature:\n  branch: test\n"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte("plan:\n  branch: test\n"), 0644))
		tasksContent := `phases:
  - number: 1
    title: Phase 1
    tasks: []
`
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "tasks.yaml"), []byte(tasksContent), 0644))

		_, err := BuildPhaseContext(specDir, 99, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "phase 99")
	})

	t.Run("handles empty phase tasks", func(t *testing.T) {
		specDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte("feature:\n  branch: test\n"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte("plan:\n  branch: test\n"), 0644))
		tasksContent := `phases:
  - number: 1
    title: Empty Phase
    tasks: []
`
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "tasks.yaml"), []byte(tasksContent), 0644))

		ctx, err := BuildPhaseContext(specDir, 1, 1)
		require.NoError(t, err)
		assert.Len(t, ctx.Tasks, 0)
	})
}

func TestWriteContextFile(t *testing.T) {
	// Save current directory
	origDir, err := os.Getwd()
	require.NoError(t, err)

	t.Run("writes valid YAML file", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		ctx := &PhaseContext{
			Phase:       2,
			TotalPhases: 5,
			SpecDir:     "specs/test-feature",
			Spec: map[string]interface{}{
				"feature": map[string]interface{}{
					"branch": "test-branch",
				},
			},
			Plan: map[string]interface{}{
				"summary": "Test plan",
			},
			Tasks: []map[string]interface{}{
				{
					"id":     "T001",
					"title":  "Test Task",
					"status": "Pending",
				},
			},
		}

		path, err := WriteContextFile(ctx)
		require.NoError(t, err)

		// Verify file exists
		content, err := os.ReadFile(path)
		require.NoError(t, err)

		// Verify header comment
		assert.True(t, strings.HasPrefix(string(content), "# Auto-generated"))

		// Verify it's valid YAML
		var parsed PhaseContext
		// Skip the header comment for parsing
		yamlContent := strings.SplitN(string(content), "\n\n", 2)
		require.Len(t, yamlContent, 2)
		err = yaml.Unmarshal([]byte(yamlContent[1]), &parsed)
		require.NoError(t, err)

		assert.Equal(t, 2, parsed.Phase)
		assert.Equal(t, 5, parsed.TotalPhases)
		assert.Equal(t, "specs/test-feature", parsed.SpecDir)
	})

	t.Run("returns path to created file", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		ctx := &PhaseContext{
			Phase:       3,
			TotalPhases: 7,
			SpecDir:     "specs/test",
			Spec:        map[string]interface{}{},
			Plan:        map[string]interface{}{},
			Tasks:       []map[string]interface{}{},
		}

		path, err := WriteContextFile(ctx)
		require.NoError(t, err)

		assert.Contains(t, path, "phase-3.yaml")
		_, err = os.Stat(path)
		require.NoError(t, err)
	})
}

func TestCleanupContextFile(t *testing.T) {
	t.Run("removes existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test-context.yaml")
		require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

		// File should exist
		_, err := os.Stat(testFile)
		require.NoError(t, err)

		// Cleanup
		err = CleanupContextFile(testFile)
		require.NoError(t, err)

		// File should not exist
		_, err = os.Stat(testFile)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("handles file-not-found gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonexistent := filepath.Join(tmpDir, "nonexistent.yaml")

		err := CleanupContextFile(nonexistent)
		assert.NoError(t, err, "should not return error for non-existent file")
	})
}

func TestContainsLine(t *testing.T) {
	tests := map[string]struct {
		content string
		line    string
		want    bool
	}{
		"line exists": {
			content: "line1\nline2\nline3",
			line:    "line2",
			want:    true,
		},
		"line does not exist": {
			content: "line1\nline2\nline3",
			line:    "line4",
			want:    false,
		},
		"partial match should not match": {
			content: "line1\nline2longer\nline3",
			line:    "line2",
			want:    false,
		},
		"empty content": {
			content: "",
			line:    "test",
			want:    false,
		},
		"empty line matches empty line": {
			content: "line1\n\nline3",
			line:    "",
			want:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := containsLine(tc.content, tc.line)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := map[string]struct {
		content   string
		wantLines []string
	}{
		"multiple lines": {
			content:   "line1\nline2\nline3",
			wantLines: []string{"line1", "line2", "line3"},
		},
		"single line": {
			content:   "line1",
			wantLines: []string{"line1"},
		},
		"empty string": {
			content:   "",
			wantLines: nil,
		},
		"trailing newline": {
			content:   "line1\nline2\n",
			wantLines: []string{"line1", "line2"},
		},
		"empty lines": {
			content:   "line1\n\nline3",
			wantLines: []string{"line1", "", "line3"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := splitLines(tc.content)
			assert.Equal(t, tc.wantLines, got)
		})
	}
}

func TestExtractTasksForPhase(t *testing.T) {
	t.Run("extracts tasks for valid phase", func(t *testing.T) {
		tasksFile := map[string]interface{}{
			"phases": []interface{}{
				map[string]interface{}{
					"number": 1,
					"title":  "Phase 1",
					"tasks": []interface{}{
						map[string]interface{}{"id": "T001", "title": "Task 1"},
						map[string]interface{}{"id": "T002", "title": "Task 2"},
					},
				},
				map[string]interface{}{
					"number": 2,
					"title":  "Phase 2",
					"tasks": []interface{}{
						map[string]interface{}{"id": "T003", "title": "Task 3"},
					},
				},
			},
		}

		tasks, err := extractTasksForPhase(tasksFile, 1)
		require.NoError(t, err)
		assert.Len(t, tasks, 2)
		assert.Equal(t, "T001", tasks[0]["id"])
		assert.Equal(t, "T002", tasks[1]["id"])
	})

	t.Run("returns error for phase not found", func(t *testing.T) {
		tasksFile := map[string]interface{}{
			"phases": []interface{}{
				map[string]interface{}{
					"number": 1,
					"title":  "Phase 1",
					"tasks":  []interface{}{},
				},
			},
		}

		_, err := extractTasksForPhase(tasksFile, 99)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "phase 99")
	})

	t.Run("returns error for missing phases field", func(t *testing.T) {
		tasksFile := map[string]interface{}{}

		_, err := extractTasksForPhase(tasksFile, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "phases field")
	})

	t.Run("handles float64 phase numbers from YAML", func(t *testing.T) {
		// YAML sometimes parses integers as float64
		tasksFile := map[string]interface{}{
			"phases": []interface{}{
				map[string]interface{}{
					"number": float64(1),
					"title":  "Phase 1",
					"tasks": []interface{}{
						map[string]interface{}{"id": "T001"},
					},
				},
			},
		}

		tasks, err := extractTasksForPhase(tasksFile, 1)
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
	})
}

func BenchmarkBuildPhaseContext(b *testing.B) {
	// Setup test files
	specDir := b.TempDir()
	require.NoError(b, os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte("feature:\n  branch: test\n"), 0644))
	require.NoError(b, os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte("plan:\n  branch: test\n"), 0644))
	tasksContent := `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        title: Task 1
      - id: T002
        title: Task 2
`
	require.NoError(b, os.WriteFile(filepath.Join(specDir, "tasks.yaml"), []byte(tasksContent), 0644))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = BuildPhaseContext(specDir, 1, 1)
	}
}

func BenchmarkWriteContextFile(b *testing.B) {
	// Save current directory
	origDir, err := os.Getwd()
	require.NoError(b, err)
	tmpDir := b.TempDir()
	require.NoError(b, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	ctx := &PhaseContext{
		Phase:       1,
		TotalPhases: 3,
		SpecDir:     "specs/test",
		Spec: map[string]interface{}{
			"feature": map[string]interface{}{"branch": "test"},
		},
		Plan: map[string]interface{}{
			"summary": "Test plan",
		},
		Tasks: []map[string]interface{}{
			{"id": "T001", "title": "Task 1"},
			{"id": "T002", "title": "Task 2"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Phase = i%10 + 1 // Vary phase number to create different files
		path, _ := WriteContextFile(ctx)
		_ = CleanupContextFile(path)
	}
}
