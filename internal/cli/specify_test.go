package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpecifyCmdRegistration(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "specify <feature-description>" {
			found = true
			break
		}
	}
	assert.True(t, found, "specify command should be registered")
}

func TestSpecifyCmdAliases(t *testing.T) {
	expectedAliases := []string{"spec", "s"}
	assert.Equal(t, expectedAliases, specifyCmd.Aliases)
}

func TestSpecifyCmdRequiresAtLeastOneArg(t *testing.T) {
	// Should require at least 1 arg
	err := specifyCmd.Args(specifyCmd, []string{})
	assert.Error(t, err)

	err = specifyCmd.Args(specifyCmd, []string{"feature description"})
	assert.NoError(t, err)

	// Multiple args should work (they get joined)
	err = specifyCmd.Args(specifyCmd, []string{"feature", "description", "here"})
	assert.NoError(t, err)
}

func TestSpecifyCmdFlags(t *testing.T) {
	// max-retries flag should exist
	f := specifyCmd.Flags().Lookup("max-retries")
	require.NotNil(t, f)
	assert.Equal(t, "r", f.Shorthand)
	assert.Equal(t, "0", f.DefValue)
}

func TestSpecifyCmdExamples(t *testing.T) {
	examples := []string{
		"autospec specify",
		"authentication",
		"dark mode",
	}

	for _, example := range examples {
		assert.Contains(t, specifyCmd.Example, example)
	}
}

func TestSpecifyCmdLongDescription(t *testing.T) {
	keywords := []string{
		"specification",
		"spec.yaml",
		"feature description",
	}

	for _, keyword := range keywords {
		assert.Contains(t, specifyCmd.Long, keyword)
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
	// Default should be 0 (use config)
	f := specifyCmd.Flags().Lookup("max-retries")
	require.NotNil(t, f)
	assert.Equal(t, "0", f.DefValue)
}

// TestSpecifyCmd_NotificationIntegration is a regression test to ensure
// the specify command has notification support. This test reads the source
// file and verifies the notification patterns are present.
//
// Background: The specify command was missing notification support while
// other commands (run, prep, implement, all) had it. This caused notifications
// to not fire when using 'autospec specify' directly.
func TestSpecifyCmd_NotificationIntegration(t *testing.T) {
	// Read the specify.go source file
	sourceFile := "specify.go"
	content, err := os.ReadFile(sourceFile)
	require.NoError(t, err, "failed to read specify.go source file")

	source := string(content)

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

	t.Run("tracks start time", func(t *testing.T) {
		assert.True(t,
			strings.Contains(source, "time.Now()") &&
				strings.Contains(source, "SetStartTime"),
			"specify.go must track command start time")
	})

	t.Run("calls OnCommandComplete", func(t *testing.T) {
		assert.True(t,
			strings.Contains(source, "OnCommandComplete"),
			"specify.go must call OnCommandComplete when finished")
	})

	t.Run("calculates duration", func(t *testing.T) {
		assert.True(t,
			strings.Contains(source, "time.Since"),
			"specify.go must calculate command duration")
	})
}

// TestAllCommandsHaveNotificationSupport is a comprehensive regression test
// to ensure all workflow commands have notification integration.
func TestAllCommandsHaveNotificationSupport(t *testing.T) {
	// Commands that should have notification support
	commandFiles := map[string]string{
		"specify":   "specify.go",
		"prep":      "prep.go",
		"run":       "run.go",
		"implement": "implement.go",
		"all":       "all.go",
	}

	requiredPatterns := []struct {
		name    string
		pattern string
	}{
		{"notify import", `"github.com/ariel-frischer/autospec/internal/notify"`},
		{"handler creation", "notify.NewHandler"},
		{"OnCommandComplete call", "OnCommandComplete"},
	}

	for cmdName, fileName := range commandFiles {
		t.Run(cmdName, func(t *testing.T) {
			content, err := os.ReadFile(fileName)
			require.NoError(t, err, "failed to read %s", fileName)
			source := string(content)

			for _, p := range requiredPatterns {
				t.Run(p.name, func(t *testing.T) {
					assert.True(t,
						strings.Contains(source, p.pattern),
						"%s must contain %s (%s)", fileName, p.name, p.pattern)
				})
			}
		})
	}
}
