// Package util tests the status command implementation.
// Related: internal/cli/util/status.go
// Tags: util, cli, status, commands

package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/stretchr/testify/require"
)

func TestDisplayBlockedTasks_WithValidTasksFile(t *testing.T) {
	t.Parallel()

	// Create temp directory
	tmpDir := t.TempDir()

	// Create a tasks.yaml file with blocked tasks
	tasksContent := `phases:
  - number: 1
    title: "Setup"
    tasks:
      - id: "T1"
        title: "Task 1"
        status: "Completed"
      - id: "T2"
        title: "Task 2 is blocked"
        status: "Blocked"
        blocked_reason: "Waiting for API access"
      - id: "T3"
        title: "Task 3 is also blocked"
        status: "Blocked"
        blocked_reason: "This is a very long reason that should be truncated because it exceeds the maximum allowed length for display purposes"
`
	tasksPath := filepath.Join(tmpDir, "tasks.yaml")
	require.NoError(t, os.WriteFile(tasksPath, []byte(tasksContent), 0o644))

	// Call displayBlockedTasks (it prints to stdout)
	// We just verify it doesn't panic
	displayBlockedTasks(tasksPath)
}

func TestDisplayBlockedTasks_NoBlockedTasks(t *testing.T) {
	t.Parallel()

	// Create temp directory
	tmpDir := t.TempDir()

	// Create a tasks.yaml file with no blocked tasks
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
	tasksPath := filepath.Join(tmpDir, "tasks.yaml")
	require.NoError(t, os.WriteFile(tasksPath, []byte(tasksContent), 0o644))

	// Call displayBlockedTasks (it prints to stdout)
	// Should handle gracefully when no blocked tasks exist
	displayBlockedTasks(tasksPath)
}

func TestDisplayBlockedTasks_EmptyBlockedReason(t *testing.T) {
	t.Parallel()

	// Create temp directory
	tmpDir := t.TempDir()

	// Create a tasks.yaml file with blocked task but no reason
	tasksContent := `phases:
  - number: 1
    title: "Setup"
    tasks:
      - id: "T1"
        title: "Task 1 is blocked"
        status: "Blocked"
`
	tasksPath := filepath.Join(tmpDir, "tasks.yaml")
	require.NoError(t, os.WriteFile(tasksPath, []byte(tasksContent), 0o644))

	// Call displayBlockedTasks
	// Should handle empty blocked_reason gracefully
	displayBlockedTasks(tasksPath)
}

func TestResolveStatusFeatureUsesPersistedState(t *testing.T) {
	tests := map[string]struct {
		persistedDir string
		removeSpec   bool
		wantSource   spec.SelectionSource
		wantErr      string
	}{
		"persisted feature resolves without positional argument": {
			persistedDir: "128-persisted-status",
			wantSource:   spec.SelectionSourcePersisted,
		},
		"stale persisted feature reports clear error": {
			persistedDir: "128-persisted-status",
			removeSpec:   true,
			wantErr:      "persisted active feature directory missing spec.yaml",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := setupStatusFeature(t, tt.persistedDir)
			if tt.removeSpec {
				path := filepath.Join(cfg.SpecsDir, tt.persistedDir, "spec.yaml")
				require.NoError(t, os.Remove(path))
			}

			got, err := resolveStatusFeature(cfg, nil)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantSource, got.Source)
			require.Equal(t, filepath.Join(cfg.SpecsDir, tt.persistedDir), got.Metadata.Directory)
		})
	}
}

func TestResolveStatusFeatureExplicitSelectionWins(t *testing.T) {
	cfg := setupStatusFeature(t, "128-persisted-status")
	writeStatusFeature(t, cfg.SpecsDir, "129-explicit-status")

	got, err := resolveStatusFeature(cfg, []string{"129-explicit-status"})

	require.NoError(t, err)
	require.Equal(t, spec.SelectionSourceExplicit, got.Source)
	require.Equal(t, filepath.Join(cfg.SpecsDir, "129-explicit-status"), got.Metadata.Directory)
}

func setupStatusFeature(t *testing.T, persistedDir string) *config.Configuration {
	t.Helper()

	root := t.TempDir()
	cfg := &config.Configuration{
		SpecsDir: filepath.Join(root, "specs"),
		StateDir: filepath.Join(root, ".autospec", "state"),
	}
	writeStatusFeature(t, cfg.SpecsDir, persistedDir)
	_, err := spec.SaveActiveFeatureState(cfg.StateDir, persistedDir, "test")
	require.NoError(t, err)
	return cfg
}

func writeStatusFeature(t *testing.T, specsDir, name string) {
	t.Helper()

	dir := filepath.Join(specsDir, name)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "spec.yaml"), []byte("feature:\n  branch: "+name+"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tasks.yaml"), []byte("tasks:\n  branch: "+name+"\n"), 0o644))
}
