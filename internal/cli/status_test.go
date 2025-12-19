// Package cli_test tests the status command for displaying spec progress and blocked task reasons.
// Related: internal/cli/util/status.go
// Tags: cli, status, progress, tasks, blocked, filtering, verbose
package cli

import (
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/validation"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getStatusCmd finds the status command from rootCmd
func getStatusCmd() *cobra.Command {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "status [spec-name]" {
			return cmd
		}
	}
	return nil
}

func TestStatusCmdRegistration(t *testing.T) {
	cmd := getStatusCmd()
	assert.NotNil(t, cmd, "status command should be registered")
}

func TestStatusCmdFlags(t *testing.T) {
	cmd := getStatusCmd()
	require.NotNil(t, cmd, "status command must exist")

	// verbose flag
	f := cmd.Flags().Lookup("verbose")
	require.NotNil(t, f)
	assert.Equal(t, "v", f.Shorthand)
	assert.Equal(t, "false", f.DefValue)
}

func TestStatusCmdArgs(t *testing.T) {
	cmd := getStatusCmd()
	require.NotNil(t, cmd, "status command must exist")

	// Should accept 0 or 1 args
	err := cmd.Args(cmd, []string{})
	assert.NoError(t, err)

	err = cmd.Args(cmd, []string{"spec-name"})
	assert.NoError(t, err)

	err = cmd.Args(cmd, []string{"arg1", "arg2"})
	assert.Error(t, err)
}

func TestStatusCmdAlias(t *testing.T) {
	cmd := getStatusCmd()
	require.NotNil(t, cmd, "status command must exist")

	assert.Contains(t, cmd.Aliases, "st", "status command should have 'st' alias")
}

func TestStatusCmdSilenceUsage(t *testing.T) {
	cmd := getStatusCmd()
	require.NotNil(t, cmd, "status command must exist")

	assert.True(t, cmd.SilenceUsage, "status command should silence usage on errors")
}

func TestStatusCmdDefaultVerbose(t *testing.T) {
	cmd := getStatusCmd()
	require.NotNil(t, cmd, "status command must exist")

	// Default verbose should be false
	verbose, _ := cmd.Flags().GetBool("verbose")
	assert.False(t, verbose)
}

// Tests for blocked reason display in status command
// These test helper functions that mirror the unexported functions in util/status.go

// filterBlockedTasks is a test helper that mirrors util.filterBlockedTasks
func filterBlockedTasks(tasks []validation.TaskItem) []validation.TaskItem {
	var blocked []validation.TaskItem
	for _, task := range tasks {
		if strings.EqualFold(task.Status, "blocked") {
			blocked = append(blocked, task)
		}
	}
	return blocked
}

// formatBlockedReason is a test helper that mirrors util.formatBlockedReason
func formatBlockedReason(reason string) string {
	if reason == "" {
		return "(no reason provided)"
	}
	return truncateStatusReason(reason, 80)
}

// truncateStatusReason is a test helper that mirrors util.truncateStatusReason
func truncateStatusReason(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func TestFilterBlockedTasks(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		tasks   []validation.TaskItem
		wantLen int
		wantIDs []string
	}{
		"single blocked task": {
			tasks: []validation.TaskItem{
				{ID: "T001", Status: "Blocked", BlockedReason: "Waiting for API"},
				{ID: "T002", Status: "Pending"},
				{ID: "T003", Status: "Completed"},
			},
			wantLen: 1,
			wantIDs: []string{"T001"},
		},
		"multiple blocked tasks": {
			tasks: []validation.TaskItem{
				{ID: "T001", Status: "Blocked", BlockedReason: "Reason 1"},
				{ID: "T002", Status: "Blocked", BlockedReason: "Reason 2"},
				{ID: "T003", Status: "Pending"},
			},
			wantLen: 2,
			wantIDs: []string{"T001", "T002"},
		},
		"no blocked tasks": {
			tasks: []validation.TaskItem{
				{ID: "T001", Status: "Pending"},
				{ID: "T002", Status: "InProgress"},
				{ID: "T003", Status: "Completed"},
			},
			wantLen: 0,
			wantIDs: []string{},
		},
		"empty task list": {
			tasks:   []validation.TaskItem{},
			wantLen: 0,
			wantIDs: []string{},
		},
		"case insensitive blocked status": {
			tasks: []validation.TaskItem{
				{ID: "T001", Status: "blocked"},
				{ID: "T002", Status: "BLOCKED"},
				{ID: "T003", Status: "Blocked"},
			},
			wantLen: 3,
			wantIDs: []string{"T001", "T002", "T003"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := filterBlockedTasks(tc.tasks)
			assert.Len(t, got, tc.wantLen)

			var gotIDs []string
			for _, task := range got {
				gotIDs = append(gotIDs, task.ID)
			}
			assert.ElementsMatch(t, tc.wantIDs, gotIDs)
		})
	}
}

func TestFormatBlockedReason(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		reason string
		want   string
	}{
		"normal reason": {
			reason: "Waiting for API access",
			want:   "Waiting for API access",
		},
		"empty reason": {
			reason: "",
			want:   "(no reason provided)",
		},
		"long reason truncated at 80 chars": {
			reason: "This is a very long reason that exceeds 80 characters and should be truncated with an ellipsis for readability",
			want:   "This is a very long reason that exceeds 80 characters and should be truncated...",
		},
		"exactly 80 chars not truncated": {
			reason: "12345678901234567890123456789012345678901234567890123456789012345678901234567890",
			want:   "12345678901234567890123456789012345678901234567890123456789012345678901234567890",
		},
		"reason with special characters": {
			reason: "Waiting for PR #123: \"fix auth\" to be merged",
			want:   "Waiting for PR #123: \"fix auth\" to be merged",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := formatBlockedReason(tc.reason)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestTruncateStatusReason(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input  string
		maxLen int
		want   string
	}{
		"short string unchanged": {
			input:  "Short text",
			maxLen: 20,
			want:   "Short text",
		},
		"exact length unchanged": {
			input:  "ExactlyTwentyChars!x",
			maxLen: 20,
			want:   "ExactlyTwentyChars!x",
		},
		"long string truncated": {
			input:  "This is a very long string that should be truncated",
			maxLen: 30,
			want:   "This is a very long string ...",
		},
		"empty string": {
			input:  "",
			maxLen: 10,
			want:   "",
		},
		"truncate to minimum viable length": {
			input:  "Hello world",
			maxLen: 6,
			want:   "Hel...",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := truncateStatusReason(tc.input, tc.maxLen)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBlockedTasksWithMissingReason(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Status: "Blocked", BlockedReason: ""},
		{ID: "T002", Status: "Blocked", BlockedReason: "Has a reason"},
	}

	blocked := filterBlockedTasks(tasks)
	assert.Len(t, blocked, 2)

	// Task with empty reason
	reason1 := formatBlockedReason(blocked[0].BlockedReason)
	assert.Equal(t, "(no reason provided)", reason1)

	// Task with reason
	reason2 := formatBlockedReason(blocked[1].BlockedReason)
	assert.Equal(t, "Has a reason", reason2)
}
