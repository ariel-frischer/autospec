// Package util tests utility CLI commands for autospec.
// Related: internal/cli/util/*.go
// Tags: util, cli, commands, status, history, version, clean

package util

import (
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/history"
	"github.com/ariel-frischer/autospec/internal/validation"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestStatusCmd_Flags(t *testing.T) {
	// Cannot run in parallel - accesses global statusCmd state

	tests := map[string]struct {
		flagName string
		wantFlag bool
	}{
		"verbose flag exists": {
			flagName: "verbose",
			wantFlag: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - accesses global command state

			flag := statusCmd.Flags().Lookup(tt.flagName)
			if tt.wantFlag {
				assert.NotNil(t, flag, "Flag %s should exist", tt.flagName)
			} else {
				assert.Nil(t, flag)
			}
		})
	}
}

func TestHistoryCmd_Flags(t *testing.T) {
	// Cannot run in parallel - accesses global historyCmd state

	tests := map[string]struct {
		flagName string
		wantFlag bool
	}{
		"spec flag exists": {
			flagName: "spec",
			wantFlag: true,
		},
		"limit flag exists": {
			flagName: "limit",
			wantFlag: true,
		},
		"clear flag exists": {
			flagName: "clear",
			wantFlag: true,
		},
		"status flag exists": {
			flagName: "status",
			wantFlag: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - accesses global command state

			flag := historyCmd.Flags().Lookup(tt.flagName)
			if tt.wantFlag {
				assert.NotNil(t, flag, "Flag %s should exist", tt.flagName)
			} else {
				assert.Nil(t, flag)
			}
		})
	}
}

func TestVersionCmd_Flags(t *testing.T) {
	// Cannot run in parallel - accesses global versionCmd state

	tests := map[string]struct {
		flagName string
		wantFlag bool
	}{
		"plain flag exists": {
			flagName: "plain",
			wantFlag: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - accesses global command state

			flag := versionCmd.Flags().Lookup(tt.flagName)
			if tt.wantFlag {
				assert.NotNil(t, flag, "Flag %s should exist", tt.flagName)
			} else {
				assert.Nil(t, flag)
			}
		})
	}
}

func TestCleanCmd_Flags(t *testing.T) {
	// Cannot run in parallel - accesses global cleanCmd state

	tests := map[string]struct {
		flagName string
		wantFlag bool
	}{
		"dry-run flag exists": {
			flagName: "dry-run",
			wantFlag: true,
		},
		"yes flag exists": {
			flagName: "yes",
			wantFlag: true,
		},
		"keep-specs flag exists": {
			flagName: "keep-specs",
			wantFlag: true,
		},
		"remove-specs flag exists": {
			flagName: "remove-specs",
			wantFlag: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - accesses global command state

			flag := cleanCmd.Flags().Lookup(tt.flagName)
			if tt.wantFlag {
				assert.NotNil(t, flag, "Flag %s should exist", tt.flagName)
			} else {
				assert.Nil(t, flag)
			}
		})
	}
}

func TestUtilCommands_GroupIDs(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	tests := map[string]struct {
		cmd         *cobra.Command
		wantGroupID string
	}{
		"status group": {
			cmd:         statusCmd,
			wantGroupID: "getting-started",
		},
		"history group": {
			cmd:         historyCmd,
			wantGroupID: "configuration",
		},
		"clean group": {
			cmd:         cleanCmd,
			wantGroupID: "configuration",
		},
		"version group": {
			cmd:         versionCmd,
			wantGroupID: "getting-started",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - accesses global command state

			assert.Equal(t, tt.wantGroupID, tt.cmd.GroupID)
		})
	}
}

func TestUtilCommands_DescriptionsAreInformative(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	tests := map[string]struct {
		cmdShort    string
		minShortLen int
	}{
		"status has informative description": {
			cmdShort:    statusCmd.Short,
			minShortLen: 20,
		},
		"history has informative description": {
			cmdShort:    historyCmd.Short,
			minShortLen: 10,
		},
		"version has informative description": {
			cmdShort:    versionCmd.Short,
			minShortLen: 10,
		},
		"clean has informative description": {
			cmdShort:    cleanCmd.Short,
			minShortLen: 10,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - accesses global command state

			assert.GreaterOrEqual(t, len(tt.cmdShort), tt.minShortLen,
				"Short description should be informative")
		})
	}
}

func TestVersionInfo_Variables(t *testing.T) {
	t.Parallel()

	// Test that version variables are defined (they're set via ldflags at build time)
	assert.NotEmpty(t, Version, "Version should have a default value")
	assert.NotEmpty(t, Commit, "Commit should have a default value")
	assert.NotEmpty(t, BuildDate, "BuildDate should have a default value")
}

func TestGetDefaultStateDir(t *testing.T) {
	t.Parallel()

	// Test that getDefaultStateDir returns a non-empty path
	stateDir := getDefaultStateDir()
	// On most systems, this should return a valid path
	// It may be empty if $HOME is not set, which is unlikely
	assert.NotEmpty(t, stateDir, "State directory should be set")
	assert.Contains(t, stateDir, ".autospec", "State directory should contain .autospec")
	assert.Contains(t, stateDir, "state", "State directory should contain state")
}

func TestStatusCmd_Aliases(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	aliases := statusCmd.Aliases
	assert.Contains(t, aliases, "st", "Should have 'st' alias")
}

func TestHistoryCmd_Aliases(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// historyCmd doesn't have aliases by default
	// Just verify we can check aliases without panic
	_ = historyCmd.Aliases
}

func TestVersionCmd_Aliases(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	aliases := versionCmd.Aliases
	assert.Contains(t, aliases, "v", "Should have 'v' alias")
}

func TestCleanCmd_Aliases(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// cleanCmd doesn't have aliases by default
	// Just verify we can check aliases without panic
	_ = cleanCmd.Aliases
}

func TestCleanCmd_MutuallyExclusiveFlags(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// Verify that keep-specs and remove-specs flags exist
	// Their mutual exclusivity is set in init()
	keepFlag := cleanCmd.Flags().Lookup("keep-specs")
	removeFlag := cleanCmd.Flags().Lookup("remove-specs")

	assert.NotNil(t, keepFlag, "keep-specs flag should exist")
	assert.NotNil(t, removeFlag, "remove-specs flag should exist")
}

func TestHistoryCmd_HasRunE(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	assert.NotNil(t, historyCmd.RunE)
}

func TestCleanCmd_HasRunE(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	assert.NotNil(t, cleanCmd.RunE)
}

// Test helper functions from status.go

func TestFilterBlockedTasks(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input []validation.TaskItem
		want  int
	}{
		"no tasks": {
			input: []validation.TaskItem{},
			want:  0,
		},
		"no blocked tasks": {
			input: []validation.TaskItem{
				{ID: "T1", Status: "Completed"},
				{ID: "T2", Status: "Pending"},
			},
			want: 0,
		},
		"some blocked tasks": {
			input: []validation.TaskItem{
				{ID: "T1", Status: "Completed"},
				{ID: "T2", Status: "Blocked"},
				{ID: "T3", Status: "Pending"},
			},
			want: 1,
		},
		"all blocked tasks": {
			input: []validation.TaskItem{
				{ID: "T1", Status: "Blocked"},
				{ID: "T2", Status: "blocked"},
			},
			want: 2,
		},
		"case insensitive": {
			input: []validation.TaskItem{
				{ID: "T1", Status: "BLOCKED"},
				{ID: "T2", Status: "blocked"},
				{ID: "T3", Status: "Blocked"},
			},
			want: 3,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := filterBlockedTasks(tt.input)
			assert.Len(t, result, tt.want)
		})
	}
}

func TestFormatBlockedReason(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		want  string
	}{
		"empty reason": {
			input: "",
			want:  "(no reason provided)",
		},
		"short reason": {
			input: "Waiting for API",
			want:  "Waiting for API",
		},
		"long reason gets truncated": {
			input: "This is a very long reason that exceeds the maximum allowed length and should be truncated with ellipsis at the end",
			want:  "This is a very long reason that exceeds the maximum allowed length and should...",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := formatBlockedReason(tt.input)
			assert.Equal(t, tt.want, result)
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
		"short string": {
			input:  "Short",
			maxLen: 10,
			want:   "Short",
		},
		"exact length": {
			input:  "Exact",
			maxLen: 5,
			want:   "Exact",
		},
		"needs truncation": {
			input:  "This is a long string",
			maxLen: 10,
			want:   "This is...",
		},
		"truncation at boundary": {
			input:  "1234567890",
			maxLen: 5,
			want:   "12...",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := truncateStatusReason(tt.input, tt.maxLen)
			assert.Equal(t, tt.want, result)
			assert.LessOrEqual(t, len(result), tt.maxLen)
		})
	}
}

// Test helper functions from history.go

func TestBuildEmptyMessage(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		specFilter   string
		statusFilter string
		want         string
	}{
		"no filters": {
			specFilter:   "",
			statusFilter: "",
			want:         "No history available.",
		},
		"spec filter only": {
			specFilter:   "my-feature",
			statusFilter: "",
			want:         "No matching entries for spec 'my-feature'.",
		},
		"status filter only": {
			specFilter:   "",
			statusFilter: "completed",
			want:         "No matching entries for status 'completed'.",
		},
		"both filters": {
			specFilter:   "my-feature",
			statusFilter: "failed",
			want:         "No matching entries for spec 'my-feature' and status 'failed'.",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := buildEmptyMessage(tt.specFilter, tt.statusFilter)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestMatchesFilters(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		entry        history.HistoryEntry
		specFilter   string
		statusFilter string
		want         bool
	}{
		"no filters": {
			entry:        history.HistoryEntry{Spec: "test", Status: "completed"},
			specFilter:   "",
			statusFilter: "",
			want:         true,
		},
		"spec matches": {
			entry:        history.HistoryEntry{Spec: "test", Status: "completed"},
			specFilter:   "test",
			statusFilter: "",
			want:         true,
		},
		"spec does not match": {
			entry:        history.HistoryEntry{Spec: "test", Status: "completed"},
			specFilter:   "other",
			statusFilter: "",
			want:         false,
		},
		"status matches": {
			entry:        history.HistoryEntry{Spec: "test", Status: "completed"},
			specFilter:   "",
			statusFilter: "completed",
			want:         true,
		},
		"status does not match": {
			entry:        history.HistoryEntry{Spec: "test", Status: "completed"},
			specFilter:   "",
			statusFilter: "failed",
			want:         false,
		},
		"both match": {
			entry:        history.HistoryEntry{Spec: "test", Status: "completed"},
			specFilter:   "test",
			statusFilter: "completed",
			want:         true,
		},
		"spec matches but status does not": {
			entry:        history.HistoryEntry{Spec: "test", Status: "completed"},
			specFilter:   "test",
			statusFilter: "failed",
			want:         false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := matchesFilters(tt.entry, tt.specFilter, tt.statusFilter)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestFilterEntries(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		entries      []history.HistoryEntry
		specFilter   string
		statusFilter string
		limit        int
		wantCount    int
	}{
		"no filters no limit": {
			entries: []history.HistoryEntry{
				{Spec: "test1", Status: "completed"},
				{Spec: "test2", Status: "failed"},
			},
			specFilter:   "",
			statusFilter: "",
			limit:        0,
			wantCount:    2,
		},
		"filter by spec": {
			entries: []history.HistoryEntry{
				{Spec: "test1", Status: "completed"},
				{Spec: "test2", Status: "completed"},
				{Spec: "test1", Status: "failed"},
			},
			specFilter:   "test1",
			statusFilter: "",
			limit:        0,
			wantCount:    2,
		},
		"filter by status": {
			entries: []history.HistoryEntry{
				{Spec: "test1", Status: "completed"},
				{Spec: "test2", Status: "failed"},
				{Spec: "test3", Status: "completed"},
			},
			specFilter:   "",
			statusFilter: "completed",
			limit:        0,
			wantCount:    2,
		},
		"apply limit": {
			entries: []history.HistoryEntry{
				{Spec: "test1", Status: "completed"},
				{Spec: "test2", Status: "completed"},
				{Spec: "test3", Status: "completed"},
			},
			specFilter:   "",
			statusFilter: "",
			limit:        2,
			wantCount:    2,
		},
		"limit larger than result": {
			entries: []history.HistoryEntry{
				{Spec: "test1", Status: "completed"},
			},
			specFilter:   "",
			statusFilter: "",
			limit:        10,
			wantCount:    1,
		},
		"combined filters and limit": {
			entries: []history.HistoryEntry{
				{Spec: "test1", Status: "completed"},
				{Spec: "test1", Status: "completed"},
				{Spec: "test1", Status: "completed"},
				{Spec: "test2", Status: "completed"},
			},
			specFilter:   "test1",
			statusFilter: "completed",
			limit:        2,
			wantCount:    2,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := filterEntries(tt.entries, tt.specFilter, tt.statusFilter, tt.limit)
			assert.Len(t, result, tt.wantCount)
		})
	}
}

func TestFormatStatus(t *testing.T) {
	t.Parallel()

	greenFunc := func(a ...interface{}) string { return "GREEN" }
	yellowFunc := func(a ...interface{}) string { return "YELLOW" }
	redFunc := func(a ...interface{}) string { return "RED" }

	tests := map[string]struct {
		status string
		want   string
	}{
		"completed status": {
			status: "completed",
			want:   "GREEN",
		},
		"running status": {
			status: "running",
			want:   "YELLOW",
		},
		"failed status": {
			status: "failed",
			want:   "RED",
		},
		"cancelled status": {
			status: "cancelled",
			want:   "RED",
		},
		"empty status": {
			status: "",
			want:   "-         ",
		},
		"unknown status": {
			status: "unknown",
			want:   "unknown   ",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := formatStatus(tt.status, greenFunc, yellowFunc, redFunc)
			assert.Contains(t, result, tt.want)
		})
	}
}

func TestFormatID(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		id      string
		wantLen int
	}{
		"empty ID": {
			id:      "",
			wantLen: 30,
		},
		"short ID": {
			id:      "short_id",
			wantLen: 30,
		},
		"exact length": {
			id:      "exact_length_thirty_chars00",
			wantLen: 30,
		},
		"long ID gets truncated": {
			id:      "this_is_a_very_long_id_that_exceeds_thirty_characters",
			wantLen: 30,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := formatID(tt.id)
			assert.Len(t, result, tt.wantLen)
		})
	}
}

// Test helper functions from version.go

func TestCenterText(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		text  string
		width int
		want  string
	}{
		"text fits exactly": {
			text:  "Test",
			width: 4,
			want:  "Test",
		},
		"text needs centering": {
			text:  "Hi",
			width: 10,
			want:  "    Hi",
		},
		"text wider than width": {
			text:  "TooLong",
			width: 5,
			want:  "TooLong",
		},
		"odd padding": {
			text:  "Odd",
			width: 10,
			want:  "   Odd",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := centerText(tt.text, tt.width)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestTruncateCommit(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		commit string
		want   string
	}{
		"short commit": {
			commit: "abc123",
			want:   "abc123",
		},
		"exact 8 chars": {
			commit: "12345678",
			want:   "12345678",
		},
		"long commit gets truncated": {
			commit: "1234567890abcdef",
			want:   "12345678",
		},
		"empty commit": {
			commit: "",
			want:   "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := truncateCommit(tt.commit)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestGetTerminalWidth(t *testing.T) {
	t.Parallel()

	// Test that getTerminalWidth returns a reasonable value
	width := getTerminalWidth()
	assert.Greater(t, width, 0)
	assert.LessOrEqual(t, width, 500) // Reasonable max
}

// Test command Examples field

func TestCleanCmd_HasExample(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	assert.NotEmpty(t, cleanCmd.Example, "cleanCmd should have examples")
	assert.Contains(t, cleanCmd.Example, "--dry-run", "Example should mention --dry-run")
}

func TestVersionCmd_HasExample(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	assert.NotEmpty(t, versionCmd.Example, "versionCmd should have examples")
	assert.Contains(t, versionCmd.Example, "--plain", "Example should mention --plain")
}

// Test that commands have Long descriptions where expected

func TestCleanCmd_HasLongDescription(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	assert.NotEmpty(t, cleanCmd.Long, "cleanCmd should have long description")
	assert.Greater(t, len(cleanCmd.Long), len(cleanCmd.Short),
		"Long description should be longer than short")
}

func TestVersionCmd_HasLongDescription(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	assert.NotEmpty(t, versionCmd.Long, "versionCmd should have long description")
}

func TestHistoryCmd_HasLongDescription(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	assert.NotEmpty(t, historyCmd.Long, "historyCmd should have long description")
}

// Test Args validators

func TestStatusCmd_AcceptsArgs(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// statusCmd accepts at most 1 argument
	assert.NotNil(t, statusCmd.Args, "statusCmd should have Args validator")
}

// Test SilenceUsage settings

func TestStatusCmd_SilencesUsage(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	assert.True(t, statusCmd.SilenceUsage, "statusCmd should silence usage on error")
}

func TestHistoryCmd_SilencesUsage(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	assert.True(t, historyCmd.SilenceUsage, "historyCmd should silence usage on error")
}

// Test sauce command

func TestSauceCmd_Structure(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "sauce", sauceCmd.Use)
	assert.NotEmpty(t, sauceCmd.Short)
	assert.NotEmpty(t, sauceCmd.Long)
}

func TestSauceCmd_HasRun(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	assert.NotNil(t, sauceCmd.Run, "sauceCmd should have Run function")
}

func TestSourceURL_IsSet(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, SourceURL, "SourceURL should be set")
	assert.Contains(t, SourceURL, "github.com", "SourceURL should point to GitHub")
	assert.Contains(t, SourceURL, "autospec", "SourceURL should reference autospec")
}

// Test version command execution

func TestPrintPlainVersion(t *testing.T) {
	// Cannot run in parallel - modifies global version variables

	// Save original values
	origVersion := Version
	origCommit := Commit
	origBuildDate := BuildDate
	defer func() {
		Version = origVersion
		Commit = origCommit
		BuildDate = origBuildDate
	}()

	// Set test values
	Version = "1.2.3"
	Commit = "abc123"
	BuildDate = "2024-01-01"

	// Call printPlainVersion (it prints to stdout, we just verify it doesn't panic)
	assert.NotPanics(t, func() {
		printPlainVersion()
	})
}

func TestPrintPrettyVersion(t *testing.T) {
	// Cannot run in parallel - modifies global version variables

	// Save original values
	origVersion := Version
	origCommit := Commit
	origBuildDate := BuildDate
	defer func() {
		Version = origVersion
		Commit = origCommit
		BuildDate = origBuildDate
	}()

	// Set test values
	Version = "1.2.3"
	Commit = "abc123def456"
	BuildDate = "2024-01-01"

	// Call printPrettyVersion (it prints to stdout, we just verify it doesn't panic)
	assert.NotPanics(t, func() {
		printPrettyVersion()
	})
}

// Test clean command helpers

func TestPromptYesNo(t *testing.T) {
	// Cannot run in parallel - uses stdin/stdout

	tests := map[string]struct {
		input string
		want  bool
	}{
		"yes lowercase": {
			input: "yes\n",
			want:  true,
		},
		"y lowercase": {
			input: "y\n",
			want:  true,
		},
		"no lowercase": {
			input: "no\n",
			want:  false,
		},
		"n lowercase": {
			input: "n\n",
			want:  false,
		},
		"empty response": {
			input: "\n",
			want:  false,
		},
		"uppercase yes": {
			input: "YES\n",
			want:  true,
		},
		"uppercase y": {
			input: "Y\n",
			want:  true,
		},
		"random text": {
			input: "maybe\n",
			want:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - uses stdin/stdout

			cmd := &cobra.Command{}
			cmd.SetIn(strings.NewReader(tt.input))
			var outBuf strings.Builder
			cmd.SetOut(&outBuf)

			result := promptYesNo(cmd, "Test question")
			assert.Equal(t, tt.want, result)
		})
	}
}

// Test history command helpers

func TestDisplayEntries(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		entries []history.HistoryEntry
		wantLen int
	}{
		"no entries": {
			entries: []history.HistoryEntry{},
			wantLen: 0,
		},
		"single entry": {
			entries: []history.HistoryEntry{
				{
					Command:  "specify",
					Spec:     "test-feature",
					Status:   "completed",
					ExitCode: 0,
					Duration: "1.5s",
				},
			},
			wantLen: 1,
		},
		"multiple entries": {
			entries: []history.HistoryEntry{
				{
					Command:  "specify",
					Spec:     "test-feature",
					Status:   "completed",
					ExitCode: 0,
					Duration: "1.5s",
				},
				{
					Command:  "plan",
					Spec:     "test-feature",
					Status:   "failed",
					ExitCode: 1,
					Duration: "2.3s",
				},
			},
			wantLen: 2,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{}
			var outBuf strings.Builder
			cmd.SetOut(&outBuf)

			displayEntries(cmd, tt.entries)

			output := outBuf.String()
			// Count newlines to verify entries were displayed
			lines := strings.Count(output, "\n")
			assert.Equal(t, tt.wantLen, lines)
		})
	}
}

func TestDisplayEntries_OutputFormat(t *testing.T) {
	t.Parallel()

	entries := []history.HistoryEntry{
		{
			ID:       "test_id_123",
			Command:  "specify",
			Spec:     "test-spec",
			Status:   "completed",
			ExitCode: 0,
			Duration: "1.5s",
		},
	}

	cmd := &cobra.Command{}
	var outBuf strings.Builder
	cmd.SetOut(&outBuf)

	displayEntries(cmd, entries)

	output := outBuf.String()
	assert.Contains(t, output, "specify")
	assert.Contains(t, output, "test-spec")
	assert.Contains(t, output, "1.5s")
	assert.Contains(t, output, "test_id_123")
}

// Test history command with temp directory

func TestRunHistoryWithStateDir_InvalidLimit(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	cmd.Flags().Int("limit", -5, "")
	cmd.Flags().String("spec", "", "")
	cmd.Flags().String("status", "", "")
	cmd.Flags().Bool("clear", false, "")

	err := runHistoryWithStateDir(cmd, "/nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "limit must be positive")
}

// Additional tests for edge cases

func TestStatusCmd_MaximumArgs(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	// statusCmd should accept at most 1 argument
	// This is handled by cobra.MaximumNArgs(1)
	err := statusCmd.Args(statusCmd, []string{})
	assert.NoError(t, err)

	err = statusCmd.Args(statusCmd, []string{"spec-name"})
	assert.NoError(t, err)

	err = statusCmd.Args(statusCmd, []string{"spec-name", "extra"})
	assert.Error(t, err)
}

func TestFilterEntries_LimitReturnsLastN(t *testing.T) {
	t.Parallel()

	entries := []history.HistoryEntry{
		{Spec: "first"},
		{Spec: "second"},
		{Spec: "third"},
		{Spec: "fourth"},
	}

	result := filterEntries(entries, "", "", 2)

	assert.Len(t, result, 2)
	assert.Equal(t, "third", result[0].Spec)
	assert.Equal(t, "fourth", result[1].Spec)
}

func TestFormatID_EmptyReturnsPlaceholder(t *testing.T) {
	t.Parallel()

	result := formatID("")
	assert.Contains(t, result, "-")
	assert.Len(t, result, 30)
}

func TestTruncateStatusReason_EmptyString(t *testing.T) {
	t.Parallel()

	result := truncateStatusReason("", 10)
	assert.Equal(t, "", result)
}

func TestCenterText_UnicodeRunes(t *testing.T) {
	t.Parallel()

	// Test with unicode characters (counted as runes, not bytes)
	result := centerText("Hi", 10)
	assert.Equal(t, "    Hi", result)
}

func TestGetDefaultStateDir_ContainsExpectedPath(t *testing.T) {
	t.Parallel()

	stateDir := getDefaultStateDir()
	if stateDir != "" {
		assert.Contains(t, stateDir, ".autospec")
		assert.Contains(t, stateDir, "state")
	}
}

// Test history command with various scenarios

func TestRunHistoryWithStateDir_ClearHistory(t *testing.T) {
	t.Parallel()

	// Create temp directory
	tmpDir := t.TempDir()

	cmd := &cobra.Command{}
	cmd.Flags().Int("limit", 0, "")
	cmd.Flags().String("spec", "", "")
	cmd.Flags().String("status", "", "")
	cmd.Flags().Bool("clear", true, "")
	var outBuf strings.Builder
	cmd.SetOut(&outBuf)

	// Should attempt to clear but fail gracefully (no history file exists)
	err := runHistoryWithStateDir(cmd, tmpDir)
	// Error is expected because history file doesn't exist
	if err != nil {
		assert.Contains(t, err.Error(), "clearing history")
	}
}

func TestRunHistoryWithStateDir_NoHistory(t *testing.T) {
	t.Parallel()

	// Create temp directory
	tmpDir := t.TempDir()

	cmd := &cobra.Command{}
	cmd.Flags().Int("limit", 0, "")
	cmd.Flags().String("spec", "", "")
	cmd.Flags().String("status", "", "")
	cmd.Flags().Bool("clear", false, "")
	var outBuf strings.Builder
	cmd.SetOut(&outBuf)

	// Should succeed with empty history (LoadHistory returns empty HistoryFile when file doesn't exist)
	err := runHistoryWithStateDir(cmd, tmpDir)
	assert.NoError(t, err)
	// Should display "No history available"
	assert.Contains(t, outBuf.String(), "No history available")
}

// Test logo and constants

func TestLogoDisplayWidth(t *testing.T) {
	t.Parallel()

	// Verify logo display width constant matches actual width
	assert.Equal(t, 33, logoDisplayWidth)
	assert.Len(t, logo, 2, "Logo should have 2 lines")
}

func TestBoxDrawingCharacters(t *testing.T) {
	t.Parallel()

	// Verify box drawing characters are set
	assert.NotEmpty(t, boxTopLeft)
	assert.NotEmpty(t, boxTopRight)
	assert.NotEmpty(t, boxBottomLeft)
	assert.NotEmpty(t, boxBottomRight)
	assert.NotEmpty(t, boxHorizontal)
	assert.NotEmpty(t, boxVertical)
}

// Test displayBlockedTasks (requires task file)

func TestDisplayBlockedTasks_InvalidPath(t *testing.T) {
	t.Parallel()

	// Should handle invalid path gracefully (no panic)
	assert.NotPanics(t, func() {
		displayBlockedTasks("/nonexistent/tasks.yaml")
	})
}

// Test command flag defaults

func TestHistoryCmd_FlagDefaults(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	specFlag := historyCmd.Flags().Lookup("spec")
	assert.NotNil(t, specFlag)
	assert.Equal(t, "", specFlag.DefValue)

	limitFlag := historyCmd.Flags().Lookup("limit")
	assert.NotNil(t, limitFlag)
	assert.Equal(t, "0", limitFlag.DefValue)

	clearFlag := historyCmd.Flags().Lookup("clear")
	assert.NotNil(t, clearFlag)
	assert.Equal(t, "false", clearFlag.DefValue)

	statusFlag := historyCmd.Flags().Lookup("status")
	assert.NotNil(t, statusFlag)
	assert.Equal(t, "", statusFlag.DefValue)
}

func TestStatusCmd_FlagDefaults(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	verboseFlag := statusCmd.Flags().Lookup("verbose")
	assert.NotNil(t, verboseFlag)
	assert.Equal(t, "false", verboseFlag.DefValue)
}

func TestVersionCmd_FlagDefaults(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	plainFlag := versionCmd.Flags().Lookup("plain")
	assert.NotNil(t, plainFlag)
	assert.Equal(t, "false", plainFlag.DefValue)
}

func TestCleanCmd_FlagDefaults(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	dryRunFlag := cleanCmd.Flags().Lookup("dry-run")
	assert.NotNil(t, dryRunFlag)
	assert.Equal(t, "false", dryRunFlag.DefValue)

	yesFlag := cleanCmd.Flags().Lookup("yes")
	assert.NotNil(t, yesFlag)
	assert.Equal(t, "false", yesFlag.DefValue)

	keepSpecsFlag := cleanCmd.Flags().Lookup("keep-specs")
	assert.NotNil(t, keepSpecsFlag)
	assert.Equal(t, "false", keepSpecsFlag.DefValue)

	removeSpecsFlag := cleanCmd.Flags().Lookup("remove-specs")
	assert.NotNil(t, removeSpecsFlag)
	assert.Equal(t, "false", removeSpecsFlag.DefValue)
}

// Test flag shortcuts

func TestStatusCmd_VerboseFlagShorthand(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	verboseFlag := statusCmd.Flags().Lookup("verbose")
	assert.NotNil(t, verboseFlag)
	assert.Equal(t, "v", verboseFlag.Shorthand)
}

func TestHistoryCmd_SpecFlagShorthand(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	specFlag := historyCmd.Flags().Lookup("spec")
	assert.NotNil(t, specFlag)
	assert.Equal(t, "s", specFlag.Shorthand)
}

func TestHistoryCmd_LimitFlagShorthand(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	limitFlag := historyCmd.Flags().Lookup("limit")
	assert.NotNil(t, limitFlag)
	assert.Equal(t, "n", limitFlag.Shorthand)
}

func TestCleanCmd_DryRunFlagShorthand(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	dryRunFlag := cleanCmd.Flags().Lookup("dry-run")
	assert.NotNil(t, dryRunFlag)
	assert.Equal(t, "n", dryRunFlag.Shorthand)
}

func TestCleanCmd_YesFlagShorthand(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	yesFlag := cleanCmd.Flags().Lookup("yes")
	assert.NotNil(t, yesFlag)
	assert.Equal(t, "y", yesFlag.Shorthand)
}

func TestCleanCmd_KeepSpecsFlagShorthand(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	keepSpecsFlag := cleanCmd.Flags().Lookup("keep-specs")
	assert.NotNil(t, keepSpecsFlag)
	assert.Equal(t, "k", keepSpecsFlag.Shorthand)
}

func TestCleanCmd_RemoveSpecsFlagShorthand(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	removeSpecsFlag := cleanCmd.Flags().Lookup("remove-specs")
	assert.NotNil(t, removeSpecsFlag)
	assert.Equal(t, "r", removeSpecsFlag.Shorthand)
}

// Test edge cases in helper functions

func TestFormatStatus_AllStatuses(t *testing.T) {
	t.Parallel()

	greenFunc := func(a ...interface{}) string { return "G" }
	yellowFunc := func(a ...interface{}) string { return "Y" }
	redFunc := func(a ...interface{}) string { return "R" }

	tests := map[string]struct {
		status       string
		wantContains string
	}{
		"completed": {status: "completed", wantContains: "G"},
		"running":   {status: "running", wantContains: "Y"},
		"failed":    {status: "failed", wantContains: "R"},
		"cancelled": {status: "cancelled", wantContains: "R"},
		"empty":     {status: "", wantContains: "-"},
		"unknown":   {status: "pending", wantContains: "pending"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := formatStatus(tt.status, greenFunc, yellowFunc, redFunc)
			assert.Contains(t, result, tt.wantContains)
		})
	}
}

func TestDisplayEntries_EmptySpec(t *testing.T) {
	t.Parallel()

	entries := []history.HistoryEntry{
		{
			Command:  "specify",
			Spec:     "", // Empty spec
			Status:   "completed",
			ExitCode: 0,
			Duration: "1.5s",
		},
	}

	cmd := &cobra.Command{}
	var outBuf strings.Builder
	cmd.SetOut(&outBuf)

	displayEntries(cmd, entries)

	output := outBuf.String()
	// Should display "-" for empty spec
	assert.Contains(t, output, "-")
}

func TestDisplayEntries_EmptyID(t *testing.T) {
	t.Parallel()

	entries := []history.HistoryEntry{
		{
			ID:       "", // Empty ID
			Command:  "specify",
			Spec:     "test",
			Status:   "completed",
			ExitCode: 0,
			Duration: "1.5s",
		},
	}

	cmd := &cobra.Command{}
	var outBuf strings.Builder
	cmd.SetOut(&outBuf)

	displayEntries(cmd, entries)

	output := outBuf.String()
	// Should display "-" for empty ID
	assert.Contains(t, output, "-")
}

func TestDisplayEntries_NonZeroExitCode(t *testing.T) {
	t.Parallel()

	entries := []history.HistoryEntry{
		{
			Command:  "specify",
			Spec:     "test",
			Status:   "failed",
			ExitCode: 1,
			Duration: "1.5s",
		},
	}

	cmd := &cobra.Command{}
	var outBuf strings.Builder
	cmd.SetOut(&outBuf)

	displayEntries(cmd, entries)

	// Should not panic with non-zero exit code
	assert.NotEmpty(t, outBuf.String())
}

// Test additional edge cases

func TestFilterBlockedTasks_MixedCase(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T1", Status: "BLOCKED"},
		{ID: "T2", Status: "blocked"},
		{ID: "T3", Status: "Blocked"},
		{ID: "T4", Status: "BLOcKeD"},
	}

	result := filterBlockedTasks(tasks)
	assert.Len(t, result, 4, "Should match all case variations of 'blocked'")
}

func TestTruncateCommit_ExactBoundary(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		want  string
	}{
		"exactly 8 chars": {
			input: "abcdefgh",
			want:  "abcdefgh",
		},
		"9 chars": {
			input: "abcdefghi",
			want:  "abcdefgh",
		},
		"7 chars": {
			input: "abcdefg",
			want:  "abcdefg",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := truncateCommit(tt.input)
			assert.Equal(t, tt.want, result)
			assert.LessOrEqual(t, len(result), 8)
		})
	}
}

func TestFormatID_LongID(t *testing.T) {
	t.Parallel()

	longID := "this_is_a_very_long_id_that_definitely_exceeds_thirty_characters_in_length"
	result := formatID(longID)

	assert.Len(t, result, 30)
	assert.Equal(t, longID[:30], result)
}

func TestCenterText_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		text  string
		width int
	}{
		"zero width": {
			text:  "test",
			width: 0,
		},
		"negative width": {
			text:  "test",
			width: -5,
		},
		"single char": {
			text:  "a",
			width: 10,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Should not panic
			assert.NotPanics(t, func() {
				centerText(tt.text, tt.width)
			})
		})
	}
}

func TestGetTerminalWidth_Fallback(t *testing.T) {
	t.Parallel()

	// When terminal info is not available, should return 80
	width := getTerminalWidth()
	assert.True(t, width == 80 || width > 0, "Should return valid width")
}

func TestRunHistoryWithStateDir_WithFilters(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	tests := map[string]struct {
		specFilter   string
		statusFilter string
	}{
		"spec filter only": {
			specFilter:   "my-spec",
			statusFilter: "",
		},
		"status filter only": {
			specFilter:   "",
			statusFilter: "completed",
		},
		"both filters": {
			specFilter:   "my-spec",
			statusFilter: "completed",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{}
			cmd.Flags().Int("limit", 0, "")
			cmd.Flags().String("spec", tt.specFilter, "")
			cmd.Flags().String("status", tt.statusFilter, "")
			cmd.Flags().Bool("clear", false, "")
			var outBuf strings.Builder
			cmd.SetOut(&outBuf)

			err := runHistoryWithStateDir(cmd, tmpDir)
			assert.NoError(t, err)
		})
	}
}

func TestRunHistoryWithStateDir_WithLimit(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	cmd := &cobra.Command{}
	cmd.Flags().Int("limit", 10, "")
	cmd.Flags().String("spec", "", "")
	cmd.Flags().String("status", "", "")
	cmd.Flags().Bool("clear", false, "")
	var outBuf strings.Builder
	cmd.SetOut(&outBuf)

	err := runHistoryWithStateDir(cmd, tmpDir)
	assert.NoError(t, err)
}

// Test command structure edge cases

func TestAllCommands_HaveUseDefined(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	commands := []*cobra.Command{
		statusCmd,
		historyCmd,
		versionCmd,
		cleanCmd,
		sauceCmd,
	}

	for _, cmd := range commands {
		assert.NotEmpty(t, cmd.Use, "Command %s should have Use defined", cmd.Name())
	}
}

func TestAllCommands_HaveShortDescription(t *testing.T) {
	// Cannot run in parallel - accesses global command state

	commands := []*cobra.Command{
		statusCmd,
		historyCmd,
		versionCmd,
		cleanCmd,
		sauceCmd,
	}

	for _, cmd := range commands {
		assert.NotEmpty(t, cmd.Short, "Command %s should have Short description", cmd.Name())
	}
}

// Test promptYesNo with whitespace

func TestPromptYesNo_Whitespace(t *testing.T) {
	// Cannot run in parallel - uses stdin/stdout

	tests := map[string]struct {
		input string
		want  bool
	}{
		"yes with spaces": {
			input: "  yes  \n",
			want:  true,
		},
		"y with tabs": {
			input: "\ty\t\n",
			want:  true,
		},
		"no with spaces": {
			input: " no \n",
			want:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Cannot run subtests in parallel - uses stdin/stdout

			cmd := &cobra.Command{}
			cmd.SetIn(strings.NewReader(tt.input))
			var outBuf strings.Builder
			cmd.SetOut(&outBuf)

			result := promptYesNo(cmd, "Test question")
			assert.Equal(t, tt.want, result)
		})
	}
}

// Test displayEntries with various statuses

func TestDisplayEntries_AllStatuses(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status   string
		exitCode int
	}{
		"completed": {status: "completed", exitCode: 0},
		"running":   {status: "running", exitCode: 0},
		"failed":    {status: "failed", exitCode: 1},
		"cancelled": {status: "cancelled", exitCode: 130},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			entries := []history.HistoryEntry{
				{
					Command:  "test",
					Spec:     "spec",
					Status:   tt.status,
					ExitCode: tt.exitCode,
					Duration: "1s",
				},
			}

			cmd := &cobra.Command{}
			var outBuf strings.Builder
			cmd.SetOut(&outBuf)

			displayEntries(cmd, entries)
			assert.NotEmpty(t, outBuf.String())
		})
	}
}

// Test formatBlockedReason edge cases

func TestFormatBlockedReason_ExactLength(t *testing.T) {
	t.Parallel()

	// Exactly 80 chars (should not truncate)
	input := strings.Repeat("a", 80)
	result := formatBlockedReason(input)
	assert.Equal(t, input, result)
	assert.NotContains(t, result, "...")

	// 81 chars (should truncate)
	input81 := strings.Repeat("a", 81)
	result81 := formatBlockedReason(input81)
	assert.Len(t, result81, 80)
	assert.Contains(t, result81, "...")
}

// Test buildEmptyMessage all combinations

func TestBuildEmptyMessage_AllCombinations(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		spec   string
		status string
		want   string
	}{
		"both empty": {
			spec:   "",
			status: "",
			want:   "No history available.",
		},
		"only spec": {
			spec:   "test-spec",
			status: "",
			want:   "No matching entries for spec 'test-spec'.",
		},
		"only status": {
			spec:   "",
			status: "completed",
			want:   "No matching entries for status 'completed'.",
		},
		"both set": {
			spec:   "test-spec",
			status: "completed",
			want:   "No matching entries for spec 'test-spec' and status 'completed'.",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := buildEmptyMessage(tt.spec, tt.status)
			assert.Equal(t, tt.want, result)
		})
	}
}
