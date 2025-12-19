// Package util tests the status command implementation.
// Related: internal/cli/util/status.go
// Tags: util, cli, status, commands

package util

import (
	"os"
	"path/filepath"
	"testing"

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
	require.NoError(t, os.WriteFile(tasksPath, []byte(tasksContent), 0644))

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
	require.NoError(t, os.WriteFile(tasksPath, []byte(tasksContent), 0644))

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
	require.NoError(t, os.WriteFile(tasksPath, []byte(tasksContent), 0644))

	// Call displayBlockedTasks
	// Should handle empty blocked_reason gracefully
	displayBlockedTasks(tasksPath)
}
