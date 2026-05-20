package workflow

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestGetContextFilePath tests the context file path creation.
// NOTE: This test cannot use t.Parallel() because it uses os.Chdir() which modifies
// global state (the current working directory). Parallel subtests would race for
// the working directory.
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

// TestBuildPhaseContext tests phase context assembly from multiple YAML sources.
//
// The context builder aggregates data from 3 files (spec.yaml, plan.yaml, tasks.yaml)
// and filters tasks to only include those for the requested phase number.
// Tests verify:
//   - Content from all 3 files is loaded and accessible
//   - Phase filtering extracts only matching tasks (phase 1 → T001,T002; phase 2 → T003)
//   - Missing files are handled gracefully
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
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte(specContent), 0o644))

		// Create plan.yaml
		planContent := `plan:
  branch: "test-branch"
summary: |
  Test plan summary
technical_context:
  language: "Go"
`
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte(planContent), 0o644))

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
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "tasks.yaml"), []byte(tasksContent), 0o644))

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
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte("feature:\n  branch: test\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte("plan:\n  branch: test\n"), 0o644))

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
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "tasks.yaml"), []byte(tasksContent), 0o644))

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
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte("plan:\n  branch: test\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "tasks.yaml"), []byte("phases: []\n"), 0o644))

		_, err := BuildPhaseContext(specDir, 1, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "spec.yaml")
	})

	t.Run("returns error for missing plan.yaml", func(t *testing.T) {
		specDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte("feature:\n  branch: test\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "tasks.yaml"), []byte("phases: []\n"), 0o644))

		_, err := BuildPhaseContext(specDir, 1, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "plan.yaml")
	})

	t.Run("returns error for missing tasks.yaml", func(t *testing.T) {
		specDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte("feature:\n  branch: test\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte("plan:\n  branch: test\n"), 0o644))

		_, err := BuildPhaseContext(specDir, 1, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tasks.yaml")
	})

	t.Run("returns error for invalid phase number", func(t *testing.T) {
		specDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte("feature:\n  branch: test\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte("plan:\n  branch: test\n"), 0o644))
		tasksContent := `phases:
  - number: 1
    title: Phase 1
    tasks: []
`
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "tasks.yaml"), []byte(tasksContent), 0o644))

		_, err := BuildPhaseContext(specDir, 99, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "phase 99")
	})

	t.Run("handles empty phase tasks", func(t *testing.T) {
		specDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte("feature:\n  branch: test\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte("plan:\n  branch: test\n"), 0o644))
		tasksContent := `phases:
  - number: 1
    title: Empty Phase
    tasks: []
`
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "tasks.yaml"), []byte(tasksContent), 0o644))

		ctx, err := BuildPhaseContext(specDir, 1, 1)
		require.NoError(t, err)
		assert.Len(t, ctx.Tasks, 0)
	})
}

// TestWriteContextFile tests the context file writing.
// NOTE: This test cannot use t.Parallel() because it uses os.Chdir() which modifies
// global state (the current working directory). Parallel subtests would race for
// the working directory.
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
		require.NoError(t, os.WriteFile(testFile, []byte("test"), 0o644))

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

	t.Run("handles permission error gracefully", func(t *testing.T) {
		// Skip on Windows where permissions work differently
		if os.Getenv("GOOS") == "windows" {
			t.Skip("skipping permission test on Windows")
		}

		tmpDir := t.TempDir()

		// Create a read-only directory with a file inside
		readOnlyDir := filepath.Join(tmpDir, "readonly")
		require.NoError(t, os.MkdirAll(readOnlyDir, 0o755))

		testFile := filepath.Join(readOnlyDir, "locked-file.yaml")
		require.NoError(t, os.WriteFile(testFile, []byte("test"), 0o644))

		// Make directory read-only to prevent file deletion
		require.NoError(t, os.Chmod(readOnlyDir, 0o555))

		// Ensure cleanup happens even if test fails
		t.Cleanup(func() {
			_ = os.Chmod(readOnlyDir, 0o755)
		})

		// Attempt to cleanup should return an error
		err := CleanupContextFile(testFile)
		assert.Error(t, err, "should return error when file cannot be removed")
		assert.Contains(t, err.Error(), "removing context file")
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
			t.Parallel()
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
			t.Parallel()
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

// TestEnsureContextDirGitignored tests the gitignore handling for context directory.
// NOTE: This test cannot use t.Parallel() because it uses os.Chdir() which modifies
// global state (the current working directory). Parallel subtests would race for
// the working directory.
func TestEnsureContextDirGitignored(t *testing.T) {
	// Save current directory
	origDir, err := os.Getwd()
	require.NoError(t, err)

	tests := map[string]struct {
		gitignoreContent string
		expectWarning    bool
	}{
		"exact context path": {
			gitignoreContent: ".autospec/context/\n",
			expectWarning:    false,
		},
		"context path without trailing slash": {
			gitignoreContent: ".autospec/context\n",
			expectWarning:    false,
		},
		"parent directory with trailing slash": {
			gitignoreContent: ".autospec/\n",
			expectWarning:    false,
		},
		"parent directory without trailing slash": {
			gitignoreContent: ".autospec\n",
			expectWarning:    false,
		},
		"parent with wildcard": {
			gitignoreContent: ".autospec/*\n",
			expectWarning:    false,
		},
		"globstar pattern": {
			gitignoreContent: ".autospec/**/context\n",
			expectWarning:    false,
		},
		"unrelated patterns only": {
			gitignoreContent: "node_modules/\n*.log\n",
			expectWarning:    true,
		},
		"empty gitignore": {
			gitignoreContent: "",
			expectWarning:    true,
		},
		"mixed patterns with parent": {
			gitignoreContent: "node_modules/\n.autospec/\n*.log\n",
			expectWarning:    false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			require.NoError(t, os.Chdir(tmpDir))
			defer func() { _ = os.Chdir(origDir) }()

			// Create .gitignore with test content
			require.NoError(t, os.WriteFile(".gitignore", []byte(tc.gitignoreContent), 0o644))

			// Capture stderr to check for warning
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			EnsureContextDirGitignored()

			w.Close()
			os.Stderr = oldStderr

			outBytes, _ := io.ReadAll(r)
			output := string(outBytes)

			if tc.expectWarning {
				assert.Contains(t, output, "Warning:", "expected warning for gitignore content: %q", tc.gitignoreContent)
			} else {
				assert.Empty(t, output, "expected no warning for gitignore content: %q", tc.gitignoreContent)
			}
		})
	}

	t.Run("no gitignore file", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		EnsureContextDirGitignored()

		w.Close()
		os.Stderr = oldStderr

		outBytes, _ := io.ReadAll(r)
		output := string(outBytes)

		assert.Contains(t, output, ".gitignore not found")
	})
}

func TestContextMetaYAMLSerialization(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		meta          ContextMeta
		wantFields    []string
		wantBoolValue bool
	}{
		"serializes all fields with correct YAML tags": {
			meta: ContextMeta{
				PhaseArtifactsBundled: true,
				BundledArtifacts:      []string{"spec.yaml", "plan.yaml", "tasks.yaml"},
				HasGovernance:         true,
				GovernanceFile:        ".autospec/constitution.yaml",
				HasChecklists:         false,
				SkipReads:             []string{"specs/test/spec.yaml"},
			},
			wantFields: []string{
				"phase_artifacts_bundled:",
				"bundled_artifacts:",
				"has_governance:",
				"governance_file:",
				"has_checklists:",
				"skip_reads:",
			},
			wantBoolValue: true,
		},
		"serializes with has_checklists true": {
			meta: ContextMeta{
				PhaseArtifactsBundled: true,
				BundledArtifacts:      []string{"spec.yaml"},
				HasGovernance:         false,
				HasChecklists:         true,
				SkipReads:             []string{},
			},
			wantFields: []string{
				"phase_artifacts_bundled: true",
				"has_governance: false",
				"has_checklists: true",
			},
			wantBoolValue: true,
		},
		"serializes empty slices correctly": {
			meta: ContextMeta{
				PhaseArtifactsBundled: false,
				BundledArtifacts:      []string{},
				HasGovernance:         false,
				HasChecklists:         false,
				SkipReads:             []string{},
			},
			wantFields: []string{
				"phase_artifacts_bundled: false",
				"bundled_artifacts: []",
				"has_governance: false",
				"has_checklists: false",
				"skip_reads: []",
			},
			wantBoolValue: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data, err := yaml.Marshal(tt.meta)
			require.NoError(t, err)

			yamlStr := string(data)
			for _, wantField := range tt.wantFields {
				assert.Contains(t, yamlStr, wantField,
					"expected YAML to contain %q", wantField)
			}
		})
	}
}

func TestPhaseContextMetaAppearsFirst(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		ctx          PhaseContext
		wantFirstKey string
	}{
		"_context_meta appears at top of YAML output": {
			ctx: PhaseContext{
				ContextMeta: ContextMeta{
					PhaseArtifactsBundled: true,
					BundledArtifacts:      []string{"spec.yaml", "plan.yaml", "tasks.yaml"},
					HasGovernance:         false,
					HasChecklists:         false,
					SkipReads:             []string{"specs/test/spec.yaml"},
				},
				Phase:       1,
				TotalPhases: 3,
				SpecDir:     "specs/test",
				Spec:        map[string]interface{}{"feature": "test"},
				Plan:        map[string]interface{}{"summary": "test"},
				Tasks:       []map[string]interface{}{},
			},
			wantFirstKey: "_context_meta:",
		},
		"context meta with all fields populated": {
			ctx: PhaseContext{
				ContextMeta: ContextMeta{
					PhaseArtifactsBundled: true,
					BundledArtifacts:      []string{"spec.yaml", "plan.yaml", "tasks.yaml (phase-filtered)"},
					HasGovernance:         true,
					GovernanceFile:        ".autospec/constitution.yaml",
					HasChecklists:         true,
					SkipReads:             []string{"specs/feature/spec.yaml", "specs/feature/plan.yaml", "specs/feature/tasks.yaml"},
				},
				Phase:       2,
				TotalPhases: 5,
				SpecDir:     "specs/feature",
				Spec:        nil,
				Plan:        nil,
				Tasks:       nil,
			},
			wantFirstKey: "_context_meta:",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data, err := yaml.Marshal(tt.ctx)
			require.NoError(t, err)

			yamlStr := string(data)
			lines := strings.Split(yamlStr, "\n")

			// Find first non-empty line
			var firstLine string
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					firstLine = line
					break
				}
			}

			assert.Equal(t, tt.wantFirstKey, firstLine,
				"expected first YAML key to be %q but got %q", tt.wantFirstKey, firstLine)
		})
	}
}

func TestBuildContextMeta(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		specDirSuffix             string
		wantPhaseArtifactsBundled bool
		wantBundledArtifacts      []string
	}{
		"sets phase_artifacts_bundled to true": {
			specDirSuffix:             filepath.Join("specs", "test-feature"),
			wantPhaseArtifactsBundled: true,
			wantBundledArtifacts: []string{
				"spec.yaml",
				"plan.yaml",
				"tasks.yaml (phase-filtered)",
			},
		},
		"uses correct paths for nested spec directory": {
			specDirSuffix:             filepath.Join("specs", "nested", "feature"),
			wantPhaseArtifactsBundled: true,
			wantBundledArtifacts: []string{
				"spec.yaml",
				"plan.yaml",
				"tasks.yaml (phase-filtered)",
			},
		},
		"handles simple directory name": {
			specDirSuffix:             "feature",
			wantPhaseArtifactsBundled: true,
			wantBundledArtifacts: []string{
				"spec.yaml",
				"plan.yaml",
				"tasks.yaml (phase-filtered)",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			specDir := filepath.Join(t.TempDir(), tt.specDirSuffix)
			meta := buildContextMeta(specDir)

			assert.Equal(t, tt.wantPhaseArtifactsBundled, meta.PhaseArtifactsBundled,
				"PhaseArtifactsBundled should be true for all generated contexts")

			assert.Equal(t, tt.wantBundledArtifacts, meta.BundledArtifacts,
				"BundledArtifacts should contain expected three items")

			wantSkipReads := []string{
				filepath.Join(specDir, "spec.yaml"),
				filepath.Join(specDir, "plan.yaml"),
				filepath.Join(specDir, "tasks.yaml"),
			}
			assert.Equal(t, wantSkipReads, meta.SkipReads,
				"SkipReads should contain correct paths for spec directory")
		})
	}
}

func TestBuildContextMetaIncludesGovernanceWhenPresent(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, ".autospec"), 0o755))
	governanceFile := filepath.Join(tmpDir, ".autospec", "constitution.yaml")
	require.NoError(t, os.WriteFile(governanceFile,
		[]byte("constitution:\n  name: Test\n"), 0o644))

	meta := buildContextMeta(filepath.Join(tmpDir, "specs", "test-feature"))

	assert.True(t, meta.HasGovernance)
	assert.Equal(t, governanceFile, meta.GovernanceFile)
	assert.Contains(t, meta.BundledArtifacts, "constitution.yaml")
	assert.Contains(t, meta.SkipReads, governanceFile)
}

func TestBuildPhaseContextPopulatesContextMeta(t *testing.T) {
	t.Parallel()

	t.Run("BuildPhaseContext includes populated ContextMeta", func(t *testing.T) {
		t.Parallel()

		specDir := t.TempDir()

		// Create spec.yaml
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.yaml"),
			[]byte("feature:\n  branch: test\n"), 0o644))

		// Create plan.yaml
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "plan.yaml"),
			[]byte("plan:\n  branch: test\n"), 0o644))

		// Create tasks.yaml
		tasksContent := `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        title: Task 1
`
		require.NoError(t, os.WriteFile(filepath.Join(specDir, "tasks.yaml"),
			[]byte(tasksContent), 0o644))

		ctx, err := BuildPhaseContext(specDir, 1, 1)
		require.NoError(t, err)

		// Verify ContextMeta is populated
		assert.True(t, ctx.ContextMeta.PhaseArtifactsBundled,
			"PhaseArtifactsBundled should be true")

		assert.Len(t, ctx.ContextMeta.BundledArtifacts, 3,
			"BundledArtifacts should have three items")
		assert.Contains(t, ctx.ContextMeta.BundledArtifacts, "spec.yaml")
		assert.Contains(t, ctx.ContextMeta.BundledArtifacts, "plan.yaml")
		assert.Contains(t, ctx.ContextMeta.BundledArtifacts, "tasks.yaml (phase-filtered)")

		assert.Len(t, ctx.ContextMeta.SkipReads, 3,
			"SkipReads should have three items")
		assert.Contains(t, ctx.ContextMeta.SkipReads, filepath.Join(specDir, "spec.yaml"))
		assert.Contains(t, ctx.ContextMeta.SkipReads, filepath.Join(specDir, "plan.yaml"))
		assert.Contains(t, ctx.ContextMeta.SkipReads, filepath.Join(specDir, "tasks.yaml"))
	})
}

func TestBuildPhaseContextBundlesGovernanceWhenPresent(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, ".autospec"), 0o755))
	governanceFile := filepath.Join(tmpDir, ".autospec", "constitution.yaml")
	require.NoError(t, os.WriteFile(governanceFile,
		[]byte("constitution:\n  name: Test Governance\n"), 0o644))

	specDir := filepath.Join(tmpDir, "specs", "test-feature")
	require.NoError(t, os.MkdirAll(specDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.yaml"),
		[]byte("feature:\n  branch: test\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "plan.yaml"),
		[]byte("plan:\n  branch: test\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "tasks.yaml"), []byte(`phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        title: Task 1
`), 0o644))

	ctx, err := BuildPhaseContext(specDir, 1, 1)
	require.NoError(t, err)

	assert.True(t, ctx.ContextMeta.HasGovernance)
	assert.Equal(t, governanceFile, ctx.ContextMeta.GovernanceFile)
	assert.Contains(t, ctx.ContextMeta.SkipReads, governanceFile)
	assert.NotEmpty(t, ctx.Governance)
}

func TestCheckChecklistsExist(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFunc func(specDir string) error
		want      bool
	}{
		"returns false when checklists directory does not exist": {
			setupFunc: func(_ string) error {
				// No setup needed - empty directory
				return nil
			},
			want: false,
		},
		"returns true when checklists directory exists": {
			setupFunc: func(specDir string) error {
				return os.Mkdir(filepath.Join(specDir, "checklists"), 0o755)
			},
			want: true,
		},
		"returns false when checklists is a file not a directory": {
			setupFunc: func(specDir string) error {
				return os.WriteFile(filepath.Join(specDir, "checklists"), []byte("not a directory"), 0o644)
			},
			want: false,
		},
		"returns true when checklists directory is empty": {
			setupFunc: func(specDir string) error {
				return os.Mkdir(filepath.Join(specDir, "checklists"), 0o755)
			},
			want: true,
		},
		"returns true when checklists directory has files": {
			setupFunc: func(specDir string) error {
				checklistsDir := filepath.Join(specDir, "checklists")
				if err := os.Mkdir(checklistsDir, 0o755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(checklistsDir, "ux.yaml"), []byte("items: []"), 0o644)
			},
			want: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			specDir := t.TempDir()
			if tt.setupFunc != nil {
				require.NoError(t, tt.setupFunc(specDir))
			}

			got := checkChecklistsExist(specDir)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildPhaseContextHasChecklistsIntegration(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		hasChecklistsDir bool
		want             bool
	}{
		"HasChecklists is false when checklists directory does not exist": {
			hasChecklistsDir: false,
			want:             false,
		},
		"HasChecklists is true when checklists directory exists": {
			hasChecklistsDir: true,
			want:             true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			specDir := t.TempDir()

			// Create required artifact files
			require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.yaml"),
				[]byte("feature:\n  branch: test\n"), 0o644))
			require.NoError(t, os.WriteFile(filepath.Join(specDir, "plan.yaml"),
				[]byte("plan:\n  branch: test\n"), 0o644))
			tasksContent := `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        title: Task 1
`
			require.NoError(t, os.WriteFile(filepath.Join(specDir, "tasks.yaml"),
				[]byte(tasksContent), 0o644))

			// Optionally create checklists directory
			if tt.hasChecklistsDir {
				require.NoError(t, os.Mkdir(filepath.Join(specDir, "checklists"), 0o755))
			}

			ctx, err := BuildPhaseContext(specDir, 1, 1)
			require.NoError(t, err)

			assert.Equal(t, tt.want, ctx.ContextMeta.HasChecklists,
				"HasChecklists should be %v when checklists directory exists=%v",
				tt.want, tt.hasChecklistsDir)
		})
	}
}

func BenchmarkCheckChecklistsExist(b *testing.B) {
	specDir := b.TempDir()

	b.Run("directory does not exist", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = checkChecklistsExist(specDir)
		}
	})

	// Create checklists directory for next benchmark
	checklistsDir := filepath.Join(specDir, "checklists")
	require.NoError(b, os.Mkdir(checklistsDir, 0o755))

	b.Run("directory exists", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = checkChecklistsExist(specDir)
		}
	})
}

func BenchmarkBuildPhaseContext(b *testing.B) {
	// Setup test files
	specDir := b.TempDir()
	require.NoError(b, os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte("feature:\n  branch: test\n"), 0o644))
	require.NoError(b, os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte("plan:\n  branch: test\n"), 0o644))
	tasksContent := `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        title: Task 1
      - id: T002
        title: Task 2
`
	require.NoError(b, os.WriteFile(filepath.Join(specDir, "tasks.yaml"), []byte(tasksContent), 0o644))

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
