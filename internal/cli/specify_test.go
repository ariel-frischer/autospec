// Package cli_test tests the specify command for generating spec.yaml from feature descriptions with notification support.
// Related: internal/cli/stages/specify.go
// Tags: cli, specify, command, workflow, specification, lifecycle, notifications
package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getSpecifyCmd finds the specify command from rootCmd
func getSpecifyCmd() *cobra.Command {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "specify <feature-description>" {
			return cmd
		}
	}
	return nil
}

func TestSpecifyCmdRegistration(t *testing.T) {
	cmd := getSpecifyCmd()
	assert.NotNil(t, cmd, "specify command should be registered")
}

func TestSpecifyCmdAliases(t *testing.T) {
	cmd := getSpecifyCmd()
	require.NotNil(t, cmd, "specify command must exist")
	expectedAliases := []string{"spec", "s"}
	assert.Equal(t, expectedAliases, cmd.Aliases)
}

func TestSpecifyCmdRequiresAtLeastOneArg(t *testing.T) {
	cmd := getSpecifyCmd()
	require.NotNil(t, cmd, "specify command must exist")

	// Should require at least 1 arg
	err := cmd.Args(cmd, []string{})
	assert.Error(t, err)

	err = cmd.Args(cmd, []string{"feature description"})
	assert.NoError(t, err)

	// Multiple args should work (they get joined)
	err = cmd.Args(cmd, []string{"feature", "description", "here"})
	assert.NoError(t, err)
}

func TestSpecifyCmdFlags(t *testing.T) {
	cmd := getSpecifyCmd()
	require.NotNil(t, cmd, "specify command must exist")

	// max-retries flag should exist
	f := cmd.Flags().Lookup("max-retries")
	require.NotNil(t, f)
	assert.Equal(t, "r", f.Shorthand)
	assert.Equal(t, "0", f.DefValue)
}

func TestSpecifyCmdExamples(t *testing.T) {
	cmd := getSpecifyCmd()
	require.NotNil(t, cmd, "specify command must exist")

	examples := []string{
		"autospec specify",
		"authentication",
		"dark mode",
	}

	for _, example := range examples {
		assert.Contains(t, cmd.Example, example)
	}
}

func TestSpecifyCmdLongDescription(t *testing.T) {
	cmd := getSpecifyCmd()
	require.NotNil(t, cmd, "specify command must exist")

	keywords := []string{
		"specification",
		"spec.yaml",
		"feature description",
	}

	for _, keyword := range keywords {
		assert.Contains(t, cmd.Long, keyword)
	}
}

func TestSpecifyCmd_InheritedFlags(t *testing.T) {
	// Should inherit skip-preflight from root
	f := rootCmd.PersistentFlags().Lookup("skip-preflight")
	require.NotNil(t, f)

	// Should inherit config from root
	f = rootCmd.PersistentFlags().Lookup("config")
	require.NotNil(t, f)
}

func TestSpecifyCmd_MaxRetriesDefault(t *testing.T) {
	cmd := getSpecifyCmd()
	require.NotNil(t, cmd, "specify command must exist")

	// Default should be 0 (use config)
	f := cmd.Flags().Lookup("max-retries")
	require.NotNil(t, f)
	assert.Equal(t, "0", f.DefValue)
}

// TestSpecifyCmd_NotificationIntegration is a regression test to ensure
// the specify command has notification support via the lifecycle wrapper.
// This test reads the source file and verifies the lifecycle patterns are present.
//
// Background: The specify command was refactored to use lifecycle.Run() wrapper
// which handles timing and notification dispatch automatically.
func TestSpecifyCmd_NotificationIntegration(t *testing.T) {
	// Get the directory containing this test file
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get test file path")
	baseDir := filepath.Dir(thisFile)

	// Read the specify.go source file (now in stages subpackage)
	sourceFile := filepath.Join(baseDir, "stages/specify.go")
	content, err := os.ReadFile(sourceFile)
	require.NoError(t, err, "failed to read stages/specify.go source file")

	source := string(content)

	t.Run("imports lifecycle package", func(t *testing.T) {
		assert.True(t,
			strings.Contains(source, `"github.com/ariel-frischer/autospec/internal/lifecycle"`),
			"specify.go must import the lifecycle package")
	})

	t.Run("imports notify package", func(t *testing.T) {
		assert.True(t,
			strings.Contains(source, `"github.com/ariel-frischer/autospec/internal/notify"`),
			"specify.go must import the notify package")
	})

	t.Run("creates notification handler", func(t *testing.T) {
		assert.True(t,
			strings.Contains(source, "notify.NewHandler"),
			"specify.go must create a notification handler with notify.NewHandler")
	})

	t.Run("sets notification handler on executor", func(t *testing.T) {
		assert.True(t,
			strings.Contains(source, "NotificationHandler = notifHandler") ||
				strings.Contains(source, "Executor.NotificationHandler"),
			"specify.go must set the notification handler on the executor")
	})

	t.Run("uses lifecycle.Run wrapper", func(t *testing.T) {
		usesLifecycle := strings.Contains(source, "lifecycle.Run(") ||
			strings.Contains(source, "lifecycle.RunWithHistory(")
		assert.True(t, usesLifecycle,
			"specify.go must use lifecycle wrapper for timing, notification, and history")
	})
}

// TestAllCommandsHaveNotificationSupport is a comprehensive regression test
// to ensure all workflow commands use the lifecycle wrapper for notifications.
//
// All commands MUST use lifecycle.Run() wrapper which handles:
// - Timing (start time, duration calculation)
// - Notification dispatch (OnCommandComplete with correct parameters)
// - Panic recovery for handlers
//
// This test enforces the lifecycle.Run() pattern and fails if commands
// use direct OnCommandComplete calls (legacy boilerplate pattern).
func TestAllCommandsHaveNotificationSupport(t *testing.T) {
	t.Parallel()

	// Get the directory containing this test file
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get test file path")
	baseDir := filepath.Dir(thisFile)

	// Commands that should have notification support via lifecycle.Run()
	// Key is command name, value is relative path from internal/cli/
	// Note: Only specify, plan, tasks, implement were moved to stages/
	// Others remain in the main cli package
	commandFiles := map[string]string{
		"specify":      "stages/specify.go",
		"prep":         "prep.go",
		"run":          "run.go",
		"implement":    "stages/implement.go",
		"all":          "all.go",
		"clarify":      "clarify.go",
		"analyze":      "analyze.go",
		"plan":         "stages/plan.go",
		"tasks":        "stages/tasks.go",
		"checklist":    "checklist.go",
		"constitution": "constitution.go",
	}

	for cmdName, fileName := range commandFiles {
		t.Run(cmdName, func(t *testing.T) {
			t.Parallel()
			fullPath := filepath.Join(baseDir, fileName)
			content, err := os.ReadFile(fullPath)
			require.NoError(t, err, "failed to read %s", fileName)
			source := string(content)

			// All commands must import lifecycle package
			t.Run("lifecycle import", func(t *testing.T) {
				assert.True(t,
					strings.Contains(source, `"github.com/ariel-frischer/autospec/internal/lifecycle"`),
					"%s must import lifecycle package", fileName)
			})

			// All commands must import notify and create a handler
			t.Run("notify import", func(t *testing.T) {
				assert.True(t,
					strings.Contains(source, `"github.com/ariel-frischer/autospec/internal/notify"`),
					"%s must import notify package", fileName)
			})

			t.Run("handler creation", func(t *testing.T) {
				assert.True(t,
					strings.Contains(source, "notify.NewHandler"),
					"%s must create notification handler", fileName)
			})

			// Commands MUST use lifecycle wrapper (Run, RunWithHistory, RunWithContext, or RunWithHistoryContext)
			t.Run("uses lifecycle wrapper", func(t *testing.T) {
				usesLifecycle := strings.Contains(source, "lifecycle.Run(") ||
					strings.Contains(source, "lifecycle.RunWithContext(") ||
					strings.Contains(source, "lifecycle.RunWithHistory(") ||
					strings.Contains(source, "lifecycle.RunWithHistoryContext(")
				assert.True(t, usesLifecycle,
					"%s must use lifecycle wrapper (Run, RunWithHistory, RunWithContext, or RunWithHistoryContext)", fileName)
			})

			// Commands must NOT use direct OnCommandComplete calls (legacy boilerplate)
			// This ensures they're using the lifecycle wrapper instead
			t.Run("no legacy boilerplate", func(t *testing.T) {
				hasDirectCall := strings.Contains(source, ".OnCommandComplete(")
				assert.False(t, hasDirectCall,
					"%s should not call OnCommandComplete directly - use lifecycle.Run() instead", fileName)
			})

			// Commands MUST import history package and create history logger
			t.Run("history import", func(t *testing.T) {
				assert.True(t,
					strings.Contains(source, `"github.com/ariel-frischer/autospec/internal/history"`),
					"%s must import history package", fileName)
			})

			t.Run("history logger creation", func(t *testing.T) {
				assert.True(t,
					strings.Contains(source, "history.NewWriter("),
					"%s must create history logger with history.NewWriter()", fileName)
			})
		})
	}
}
