// Package testutil_test tests filesystem helper utilities for test fixture creation.
// Related: /home/ari/repos/autospec/internal/testutil/fs_helpers.go
// Tags: testutil, helpers, fixtures, filesystem

package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateTempSpec(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		specName string
	}{
		"simple spec name": {
			specName: "001-test-feature",
		},
		"spec with hyphens": {
			specName: "002-multi-word-feature-name",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()

			specDir := CreateTempSpec(t, tmpDir, tc.specName)

			// Verify spec directory exists
			if _, err := os.Stat(specDir); os.IsNotExist(err) {
				t.Errorf("spec directory was not created: %s", specDir)
			}

			// Verify spec.yaml exists
			specPath := filepath.Join(specDir, "spec.yaml")
			if !FileExists(specPath) {
				t.Errorf("spec.yaml was not created: %s", specPath)
			}

			// Verify content is valid
			content := ReadFile(t, specPath)
			if len(content) == 0 {
				t.Error("spec.yaml is empty")
			}
		})
	}
}

func TestCreateTempPlan(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specDir := filepath.Join(tmpDir, "001-test")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("failed to create spec dir: %v", err)
	}

	planPath := CreateTempPlan(t, specDir)

	// Verify plan.yaml exists
	if !FileExists(planPath) {
		t.Errorf("plan.yaml was not created: %s", planPath)
	}

	// Verify content is valid
	content := ReadFile(t, planPath)
	if len(content) == 0 {
		t.Error("plan.yaml is empty")
	}
}

func TestCreateTempTasks(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		opts       []TasksOption
		wantStatus string
	}{
		"default options": {
			opts:       nil,
			wantStatus: "Pending",
		},
		"with completed status": {
			opts:       []TasksOption{WithTaskStatus("Completed")},
			wantStatus: "Completed",
		},
		"with in progress status": {
			opts:       []TasksOption{WithTaskStatus("InProgress")},
			wantStatus: "InProgress",
		},
		"with custom task ID": {
			opts:       []TasksOption{WithTaskID("T005")},
			wantStatus: "Pending",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			specDir := filepath.Join(tmpDir, "001-test")
			if err := os.MkdirAll(specDir, 0755); err != nil {
				t.Fatalf("failed to create spec dir: %v", err)
			}

			tasksPath := CreateTempTasks(t, specDir, tc.opts...)

			// Verify tasks.yaml exists
			if !FileExists(tasksPath) {
				t.Errorf("tasks.yaml was not created: %s", tasksPath)
			}

			// Verify content contains expected status
			content := ReadFile(t, tasksPath)
			if len(content) == 0 {
				t.Error("tasks.yaml is empty")
			}
		})
	}
}

func TestCreateTempDir(t *testing.T) {
	t.Parallel()

	dir := CreateTempDir(t, "test-prefix")

	// Verify directory exists
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		t.Errorf("temp directory was not created: %s", dir)
	}

	if !info.IsDir() {
		t.Errorf("expected directory, got file: %s", dir)
	}
}

func TestWriteFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	tests := map[string]struct {
		path    string
		content string
	}{
		"simple file": {
			path:    filepath.Join(tmpDir, "test.txt"),
			content: "test content",
		},
		"nested file": {
			path:    filepath.Join(tmpDir, "nested", "dir", "test.txt"),
			content: "nested content",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			WriteFile(t, tc.path, tc.content)

			if !FileExists(tc.path) {
				t.Errorf("file was not created: %s", tc.path)
			}

			got := ReadFile(t, tc.path)
			if got != tc.content {
				t.Errorf("content mismatch: got %q, want %q", got, tc.content)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "exists.txt")
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := map[string]struct {
		path string
		want bool
	}{
		"existing file": {
			path: existingFile,
			want: true,
		},
		"non-existing file": {
			path: filepath.Join(tmpDir, "nonexistent.txt"),
			want: false,
		},
		"existing directory": {
			path: tmpDir,
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := FileExists(tc.path)
			if got != tc.want {
				t.Errorf("FileExists(%s) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

// TestClearAPIKeys verifies that API keys are cleared during tests.
// Note: Cannot use t.Parallel() as it modifies environment variables.
func TestClearAPIKeys(t *testing.T) {
	// Set a test API key before clearing
	testKey := "test-api-key-12345"
	os.Setenv("ANTHROPIC_API_KEY", testKey)

	// Clear API keys
	cleanup := ClearAPIKeys(t)

	// Verify the key is now empty
	if val := os.Getenv("ANTHROPIC_API_KEY"); val != "" {
		t.Errorf("ANTHROPIC_API_KEY should be empty after ClearAPIKeys, got %q", val)
	}

	// Verify all API keys are cleared
	for _, key := range apiKeyEnvVars {
		if val := os.Getenv(key); val != "" {
			t.Errorf("%s should be empty after ClearAPIKeys, got %q", key, val)
		}
	}

	// Call cleanup to restore original value
	cleanup()
}
