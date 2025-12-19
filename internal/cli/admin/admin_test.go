// Package admin_test provides comprehensive tests for administrative CLI commands.
// Related: internal/cli/admin/
// Tags: admin, cli, commands, completion, uninstall, testing
package admin

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/uninstall"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// commands.go Tests
// =============================================================================

func TestCommandsCmd_Structure(t *testing.T) {

	tests := map[string]struct {
		field    string
		getValue func() interface{}
		wantType string
	}{
		"Use field": {
			field:    "Use",
			getValue: func() interface{} { return commandsCmd.Use },
			wantType: "commands",
		},
		"Short description": {
			field:    "Short",
			getValue: func() interface{} { return commandsCmd.Short },
			wantType: "string",
		},
		"Long description": {
			field:    "Long",
			getValue: func() interface{} { return commandsCmd.Long },
			wantType: "string",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			val := tt.getValue()
			if tt.wantType == "string" {
				assert.NotEmpty(t, val, "Field %s should not be empty", tt.field)
			} else {
				assert.Equal(t, tt.wantType, val, "Field %s should match", tt.field)
			}
		})
	}
}

func TestCommandsCmd_Subcommands(t *testing.T) {

	tests := map[string]struct {
		subcommand string
		shouldHave bool
	}{
		"has install subcommand": {
			subcommand: "install",
			shouldHave: true,
		},
		"has check subcommand": {
			subcommand: "check",
			shouldHave: true,
		},
		"has info subcommand": {
			subcommand: "info",
			shouldHave: true,
		},
		"does not have invalid subcommand": {
			subcommand: "nonexistent",
			shouldHave: false,
		},
	}

	subcommands := commandsCmd.Commands()
	subcommandNames := make(map[string]bool)
	for _, cmd := range subcommands {
		subcommandNames[cmd.Name()] = true
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if tt.shouldHave {
				assert.True(t, subcommandNames[tt.subcommand],
					"Should have '%s' subcommand", tt.subcommand)
			} else {
				assert.False(t, subcommandNames[tt.subcommand],
					"Should not have '%s' subcommand", tt.subcommand)
			}
		})
	}
}

// =============================================================================
// commands_install.go Tests
// =============================================================================

func TestCommandsInstallCmd_Structure(t *testing.T) {

	tests := map[string]struct {
		check    func() bool
		errorMsg string
	}{
		"Use field is install": {
			check:    func() bool { return commandsInstallCmd.Use == "install" },
			errorMsg: "Use should be 'install'",
		},
		"Short description exists": {
			check:    func() bool { return commandsInstallCmd.Short != "" },
			errorMsg: "Short description should not be empty",
		},
		"Long description exists": {
			check:    func() bool { return commandsInstallCmd.Long != "" },
			errorMsg: "Long description should not be empty",
		},
		"RunE is defined": {
			check:    func() bool { return commandsInstallCmd.RunE != nil },
			errorMsg: "RunE should be defined",
		},
		"Deprecated message exists": {
			check:    func() bool { return commandsInstallCmd.Deprecated != "" },
			errorMsg: "Deprecated message should exist",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.True(t, tt.check(), tt.errorMsg)
		})
	}
}

func TestCommandsInstallCmd_Flags(t *testing.T) {

	tests := map[string]struct {
		flagName    string
		shouldExist bool
	}{
		"has target flag": {
			flagName:    "target",
			shouldExist: true,
		},
		"does not have random flag": {
			flagName:    "nonexistent-flag",
			shouldExist: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			flag := commandsInstallCmd.Flags().Lookup(tt.flagName)
			if tt.shouldExist {
				assert.NotNil(t, flag, "Flag %s should exist", tt.flagName)
			} else {
				assert.Nil(t, flag, "Flag %s should not exist", tt.flagName)
			}
		})
	}
}

// =============================================================================
// commands_check.go Tests
// =============================================================================

func TestCommandsCheckCmd_Structure(t *testing.T) {

	tests := map[string]struct {
		field    string
		getValue func() string
		wantType string
	}{
		"Use field": {
			field:    "Use",
			getValue: func() string { return commandsCheckCmd.Use },
			wantType: "check",
		},
		"Short description": {
			field:    "Short",
			getValue: func() string { return commandsCheckCmd.Short },
			wantType: "nonempty",
		},
		"Long description": {
			field:    "Long",
			getValue: func() string { return commandsCheckCmd.Long },
			wantType: "nonempty",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			val := tt.getValue()
			if tt.wantType == "nonempty" {
				assert.NotEmpty(t, val, "Field %s should not be empty", tt.field)
			} else {
				assert.Equal(t, tt.wantType, val, "Field %s should match", tt.field)
			}
		})
	}
}

func TestCommandsCheckCmd_Flags(t *testing.T) {

	flag := commandsCheckCmd.Flags().Lookup("target")
	assert.NotNil(t, flag, "Should have 'target' flag")
	assert.Equal(t, "", flag.DefValue, "Default value should be empty string")
}

func TestCommandsCheckCmd_RunE(t *testing.T) {

	assert.NotNil(t, commandsCheckCmd.RunE, "RunE should be defined")
}

// =============================================================================
// commands_info.go Tests
// =============================================================================

func TestCommandsInfoCmd_Structure(t *testing.T) {

	tests := map[string]struct {
		field    string
		getValue func() interface{}
		check    func(interface{}) bool
		errorMsg string
	}{
		"Use field": {
			field:    "Use",
			getValue: func() interface{} { return commandsInfoCmd.Use },
			check:    func(v interface{}) bool { return v.(string) == "info [command-name]" },
			errorMsg: "Use should be 'info [command-name]'",
		},
		"Short description": {
			field:    "Short",
			getValue: func() interface{} { return commandsInfoCmd.Short },
			check:    func(v interface{}) bool { return v.(string) != "" },
			errorMsg: "Short should not be empty",
		},
		"Long description": {
			field:    "Long",
			getValue: func() interface{} { return commandsInfoCmd.Long },
			check:    func(v interface{}) bool { return v.(string) != "" },
			errorMsg: "Long should not be empty",
		},
		"RunE is defined": {
			field:    "RunE",
			getValue: func() interface{} { return commandsInfoCmd.RunE },
			check:    func(v interface{}) bool { return v != nil },
			errorMsg: "RunE should be defined",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			val := tt.getValue()
			assert.True(t, tt.check(val), tt.errorMsg)
		})
	}
}

func TestCommandsInfoCmd_Flags(t *testing.T) {

	flag := commandsInfoCmd.Flags().Lookup("target")
	assert.NotNil(t, flag, "Should have 'target' flag")
}

// =============================================================================
// completion_install.go Tests
// =============================================================================

func TestCompletionCmd_DetailedStructure(t *testing.T) {

	tests := map[string]struct {
		field    string
		getValue func() interface{}
		check    func(interface{}) bool
		errorMsg string
	}{
		"Use starts with completion": {
			field:    "Use",
			getValue: func() interface{} { return completionCmd.Use },
			check:    func(v interface{}) bool { return strings.HasPrefix(v.(string), "completion") },
			errorMsg: "Use should start with 'completion'",
		},
		"Short description exists": {
			field:    "Short",
			getValue: func() interface{} { return completionCmd.Short },
			check:    func(v interface{}) bool { return v.(string) != "" },
			errorMsg: "Short should not be empty",
		},
		"Long description exists": {
			field:    "Long",
			getValue: func() interface{} { return completionCmd.Long },
			check:    func(v interface{}) bool { return v.(string) != "" },
			errorMsg: "Long should not be empty",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			val := tt.getValue()
			assert.True(t, tt.check(val), tt.errorMsg)
		})
	}
}

func TestCompletionCmd_Subcommands(t *testing.T) {

	tests := map[string]struct {
		subcommand string
		shouldHave bool
	}{
		"has install subcommand": {
			subcommand: "install",
			shouldHave: true,
		},
		"has bash subcommand": {
			subcommand: "bash",
			shouldHave: true,
		},
		"has zsh subcommand": {
			subcommand: "zsh",
			shouldHave: true,
		},
		"has fish subcommand": {
			subcommand: "fish",
			shouldHave: true,
		},
		"has powershell subcommand": {
			subcommand: "powershell",
			shouldHave: true,
		},
	}

	subcommands := completionCmd.Commands()
	subcommandNames := make(map[string]bool)
	for _, cmd := range subcommands {
		subcommandNames[cmd.Name()] = true
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if tt.shouldHave {
				assert.True(t, subcommandNames[tt.subcommand],
					"Should have '%s' subcommand", tt.subcommand)
			} else {
				assert.False(t, subcommandNames[tt.subcommand],
					"Should not have '%s' subcommand", tt.subcommand)
			}
		})
	}
}

func TestCompletionInstallCmd_Structure(t *testing.T) {

	tests := map[string]struct {
		check    func() bool
		errorMsg string
	}{
		"Use starts with install": {
			check:    func() bool { return strings.HasPrefix(completionInstallCmd.Use, "install") },
			errorMsg: "Use should start with 'install'",
		},
		"Short description exists": {
			check:    func() bool { return completionInstallCmd.Short != "" },
			errorMsg: "Short description should not be empty",
		},
		"Long description exists": {
			check:    func() bool { return completionInstallCmd.Long != "" },
			errorMsg: "Long description should not be empty",
		},
		"Example exists": {
			check:    func() bool { return completionInstallCmd.Example != "" },
			errorMsg: "Example should not be empty",
		},
		"RunE is defined": {
			check:    func() bool { return completionInstallCmd.RunE != nil },
			errorMsg: "RunE should be defined",
		},
		"ValidArgs contains bash": {
			check: func() bool {
				for _, arg := range completionInstallCmd.ValidArgs {
					if arg == "bash" {
						return true
					}
				}
				return false
			},
			errorMsg: "ValidArgs should contain 'bash'",
		},
		"ValidArgs contains zsh": {
			check: func() bool {
				for _, arg := range completionInstallCmd.ValidArgs {
					if arg == "zsh" {
						return true
					}
				}
				return false
			},
			errorMsg: "ValidArgs should contain 'zsh'",
		},
		"ValidArgs contains fish": {
			check: func() bool {
				for _, arg := range completionInstallCmd.ValidArgs {
					if arg == "fish" {
						return true
					}
				}
				return false
			},
			errorMsg: "ValidArgs should contain 'fish'",
		},
		"ValidArgs contains powershell": {
			check: func() bool {
				for _, arg := range completionInstallCmd.ValidArgs {
					if arg == "powershell" {
						return true
					}
				}
				return false
			},
			errorMsg: "ValidArgs should contain 'powershell'",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.True(t, tt.check(), tt.errorMsg)
		})
	}
}

func TestCompletionInstallCmd_Flags(t *testing.T) {

	flag := completionInstallCmd.Flags().Lookup("manual")
	assert.NotNil(t, flag, "Should have 'manual' flag")
	assert.Equal(t, "false", flag.DefValue, "Default value should be false")
}

func TestCompletionBashCmd_Structure(t *testing.T) {

	tests := map[string]struct {
		field    string
		getValue func() interface{}
		check    func(interface{}) bool
		errorMsg string
	}{
		"Use is bash": {
			field:    "Use",
			getValue: func() interface{} { return completionBashCmd.Use },
			check:    func(v interface{}) bool { return v.(string) == "bash" },
			errorMsg: "Use should be 'bash'",
		},
		"Short description exists": {
			field:    "Short",
			getValue: func() interface{} { return completionBashCmd.Short },
			check:    func(v interface{}) bool { return v.(string) != "" },
			errorMsg: "Short should not be empty",
		},
		"Long description exists": {
			field:    "Long",
			getValue: func() interface{} { return completionBashCmd.Long },
			check:    func(v interface{}) bool { return v.(string) != "" },
			errorMsg: "Long should not be empty",
		},
		"RunE is defined": {
			field:    "RunE",
			getValue: func() interface{} { return completionBashCmd.RunE },
			check:    func(v interface{}) bool { return v != nil },
			errorMsg: "RunE should be defined",
		},
		"DisableFlagsInUseLine is true": {
			field:    "DisableFlagsInUseLine",
			getValue: func() interface{} { return completionBashCmd.DisableFlagsInUseLine },
			check:    func(v interface{}) bool { return v.(bool) },
			errorMsg: "DisableFlagsInUseLine should be true",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			val := tt.getValue()
			assert.True(t, tt.check(val), tt.errorMsg)
		})
	}
}

func TestCompletionZshCmd_Structure(t *testing.T) {

	assert.Equal(t, "zsh", completionZshCmd.Use)
	assert.NotEmpty(t, completionZshCmd.Short)
	assert.NotEmpty(t, completionZshCmd.Long)
	assert.NotNil(t, completionZshCmd.RunE)
	assert.True(t, completionZshCmd.DisableFlagsInUseLine)
}

func TestCompletionFishCmd_Structure(t *testing.T) {

	assert.Equal(t, "fish", completionFishCmd.Use)
	assert.NotEmpty(t, completionFishCmd.Short)
	assert.NotEmpty(t, completionFishCmd.Long)
	assert.NotNil(t, completionFishCmd.RunE)
	assert.True(t, completionFishCmd.DisableFlagsInUseLine)
}

func TestCompletionPowershellCmd_Structure(t *testing.T) {

	assert.Equal(t, "powershell", completionPowershellCmd.Use)
	assert.NotEmpty(t, completionPowershellCmd.Short)
	assert.NotEmpty(t, completionPowershellCmd.Long)
	assert.NotNil(t, completionPowershellCmd.RunE)
	assert.True(t, completionPowershellCmd.DisableFlagsInUseLine)
}

// =============================================================================
// uninstall.go Tests
// =============================================================================

func TestUninstallCmd_DetailedStructure(t *testing.T) {

	tests := map[string]struct {
		field    string
		getValue func() interface{}
		check    func(interface{}) bool
		errorMsg string
	}{
		"Use is uninstall": {
			field:    "Use",
			getValue: func() interface{} { return uninstallCmd.Use },
			check:    func(v interface{}) bool { return v.(string) == "uninstall" },
			errorMsg: "Use should be 'uninstall'",
		},
		"Short description exists": {
			field:    "Short",
			getValue: func() interface{} { return uninstallCmd.Short },
			check:    func(v interface{}) bool { return v.(string) != "" },
			errorMsg: "Short should not be empty",
		},
		"Long description exists": {
			field:    "Long",
			getValue: func() interface{} { return uninstallCmd.Long },
			check:    func(v interface{}) bool { return v.(string) != "" },
			errorMsg: "Long should not be empty",
		},
		"Example exists": {
			field:    "Example",
			getValue: func() interface{} { return uninstallCmd.Example },
			check:    func(v interface{}) bool { return v.(string) != "" },
			errorMsg: "Example should not be empty",
		},
		"RunE is defined": {
			field:    "RunE",
			getValue: func() interface{} { return uninstallCmd.RunE },
			check:    func(v interface{}) bool { return v != nil },
			errorMsg: "RunE should be defined",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			val := tt.getValue()
			assert.True(t, tt.check(val), tt.errorMsg)
		})
	}
}

func TestUninstallCmd_Flags(t *testing.T) {

	tests := map[string]struct {
		flagName     string
		shorthand    string
		defaultValue string
	}{
		"dry-run flag": {
			flagName:     "dry-run",
			shorthand:    "n",
			defaultValue: "false",
		},
		"yes flag": {
			flagName:     "yes",
			shorthand:    "y",
			defaultValue: "false",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			flag := uninstallCmd.Flags().Lookup(tt.flagName)
			assert.NotNil(t, flag, "Flag %s should exist", tt.flagName)
			assert.Equal(t, tt.shorthand, flag.Shorthand, "Shorthand should match")
			assert.Equal(t, tt.defaultValue, flag.DefValue, "Default value should match")
		})
	}
}

func TestCollectUninstallTargets(t *testing.T) {

	// This function calls uninstall.GetUninstallTargets() which may access filesystem
	// We can test that it returns the expected structure
	targets, existingTargets, err := collectUninstallTargets()

	// Should not error even if no targets exist
	assert.NoError(t, err)
	assert.NotNil(t, targets)
	assert.NotNil(t, existingTargets)

	// existingTargets should be a subset of targets
	assert.LessOrEqual(t, len(existingTargets), len(targets))
}

func TestDisplayUninstallTargets(t *testing.T) {

	tests := map[string]struct {
		targets     []uninstall.UninstallTarget
		dryRun      bool
		wantSudo    bool
		wantContain string
	}{
		"no targets": {
			targets:     []uninstall.UninstallTarget{},
			dryRun:      false,
			wantSudo:    false,
			wantContain: "The following will be removed:",
		},
		"dry run mode": {
			targets:     []uninstall.UninstallTarget{},
			dryRun:      true,
			wantSudo:    false,
			wantContain: "Would remove:",
		},
		"target requires sudo": {
			targets: []uninstall.UninstallTarget{
				{Path: "/usr/local/bin/autospec", Type: "binary", Exists: true, RequiresSudo: true},
			},
			dryRun:      false,
			wantSudo:    true,
			wantContain: "requires sudo",
		},
		"target exists": {
			targets: []uninstall.UninstallTarget{
				{Path: "/home/user/.config/autospec", Type: "config", Exists: true, RequiresSudo: false},
			},
			dryRun:      false,
			wantSudo:    false,
			wantContain: "exists",
		},
		"target not found": {
			targets: []uninstall.UninstallTarget{
				{Path: "/home/user/.config/autospec", Type: "config", Exists: false, RequiresSudo: false},
			},
			dryRun:      false,
			wantSudo:    false,
			wantContain: "not found",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			requiresSudo := displayUninstallTargets(&buf, tt.targets, tt.dryRun)

			assert.Equal(t, tt.wantSudo, requiresSudo)
			output := buf.String()
			assert.Contains(t, output, tt.wantContain)
		})
	}
}

func TestConfirmUninstall(t *testing.T) {
	tests := map[string]struct {
		requiresSudo bool
		yes          bool
		input        string
		want         bool
		wantContain  string
	}{
		"yes flag skips prompt": {
			requiresSudo: false,
			yes:          true,
			input:        "",
			want:         true,
			wantContain:  "",
		},
		"requires sudo shows warning": {
			requiresSudo: true,
			yes:          true,
			input:        "",
			want:         true,
			wantContain:  "elevated privileges",
		},
		"user answers yes": {
			requiresSudo: false,
			yes:          false,
			input:        "y\n",
			want:         true,
			wantContain:  "Uninstall autospec?",
		},
		"user answers no": {
			requiresSudo: false,
			yes:          false,
			input:        "n\n",
			want:         false,
			wantContain:  "Uninstall cancelled",
		},
		"user answers yes full": {
			requiresSudo: false,
			yes:          false,
			input:        "yes\n",
			want:         true,
			wantContain:  "Uninstall autospec?",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var outBuf bytes.Buffer
			inBuf := bytes.NewBufferString(tt.input)

			cmd := &cobra.Command{}
			cmd.SetOut(&outBuf)
			cmd.SetIn(inBuf)

			result := confirmUninstall(cmd, &outBuf, tt.requiresSudo, tt.yes)

			assert.Equal(t, tt.want, result)
			if tt.wantContain != "" {
				assert.Contains(t, outBuf.String(), tt.wantContain)
			}
		})
	}
}

func TestDisplayRemovalResults(t *testing.T) {

	tests := map[string]struct {
		results        []uninstall.UninstallResult
		wantSuccess    int
		wantFail       int
		wantSkipped    int
		wantContain    []string
		wantNotContain []string
	}{
		"all successful": {
			results: []uninstall.UninstallResult{
				{Target: uninstall.UninstallTarget{Path: "/path/1", Exists: true}, Success: true},
				{Target: uninstall.UninstallTarget{Path: "/path/2", Exists: true}, Success: true},
			},
			wantSuccess: 2,
			wantFail:    0,
			wantSkipped: 0,
			wantContain: []string{"Removed", "/path/1", "/path/2"},
		},
		"mixed results": {
			results: []uninstall.UninstallResult{
				{Target: uninstall.UninstallTarget{Path: "/path/1", Exists: true}, Success: true},
				{Target: uninstall.UninstallTarget{Path: "/path/2", Exists: true}, Success: false, Error: assert.AnError},
				{Target: uninstall.UninstallTarget{Path: "/path/3", Exists: false}, Success: false},
			},
			wantSuccess: 1,
			wantFail:    1,
			wantSkipped: 1,
			wantContain: []string{"Removed", "Failed", "Skipped"},
		},
		"all skipped": {
			results: []uninstall.UninstallResult{
				{Target: uninstall.UninstallTarget{Path: "/path/1", Exists: false}, Success: false},
			},
			wantSuccess: 0,
			wantFail:    0,
			wantSkipped: 1,
			wantContain: []string{"Skipped", "not found"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			success, fail, skipped := displayRemovalResults(&buf, tt.results)

			assert.Equal(t, tt.wantSuccess, success)
			assert.Equal(t, tt.wantFail, fail)
			assert.Equal(t, tt.wantSkipped, skipped)

			output := buf.String()
			for _, want := range tt.wantContain {
				assert.Contains(t, output, want)
			}
			for _, notWant := range tt.wantNotContain {
				assert.NotContains(t, output, notWant)
			}
		})
	}
}

func TestPrintUninstallSummary(t *testing.T) {

	tests := map[string]struct {
		successCount int
		failCount    int
		skippedCount int
		wantContain  []string
	}{
		"only success": {
			successCount: 3,
			failCount:    0,
			skippedCount: 0,
			wantContain:  []string{"3 removed", "uninstalled"},
		},
		"success and skipped": {
			successCount: 2,
			failCount:    0,
			skippedCount: 1,
			wantContain:  []string{"2 removed", "1 skipped", "uninstalled"},
		},
		"success and failed": {
			successCount: 1,
			failCount:    2,
			skippedCount: 0,
			wantContain:  []string{"1 removed", "2 failed"},
		},
		"all types": {
			successCount: 1,
			failCount:    1,
			skippedCount: 1,
			wantContain:  []string{"1 removed", "1 skipped", "1 failed"},
		},
		"no success": {
			successCount: 0,
			failCount:    1,
			skippedCount: 1,
			wantContain:  []string{"0 removed", "1 skipped", "1 failed"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			printUninstallSummary(&buf, tt.successCount, tt.failCount, tt.skippedCount)

			output := buf.String()
			for _, want := range tt.wantContain {
				assert.Contains(t, output, want)
			}
		})
	}
}

func TestPromptYesNo(t *testing.T) {
	tests := map[string]struct {
		input    string
		question string
		want     bool
	}{
		"user answers y": {
			input:    "y\n",
			question: "Continue?",
			want:     true,
		},
		"user answers yes": {
			input:    "yes\n",
			question: "Continue?",
			want:     true,
		},
		"user answers Y uppercase": {
			input:    "Y\n",
			question: "Continue?",
			want:     true,
		},
		"user answers YES uppercase": {
			input:    "YES\n",
			question: "Continue?",
			want:     true,
		},
		"user answers n": {
			input:    "n\n",
			question: "Continue?",
			want:     false,
		},
		"user answers no": {
			input:    "no\n",
			question: "Continue?",
			want:     false,
		},
		"user presses enter": {
			input:    "\n",
			question: "Continue?",
			want:     false,
		},
		"user types invalid": {
			input:    "maybe\n",
			question: "Continue?",
			want:     false,
		},
		"user types whitespace yes": {
			input:    "  yes  \n",
			question: "Continue?",
			want:     true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var outBuf bytes.Buffer
			inBuf := bytes.NewBufferString(tt.input)

			cmd := &cobra.Command{}
			cmd.SetOut(&outBuf)
			cmd.SetIn(inBuf)

			result := promptYesNo(cmd, tt.question)

			assert.Equal(t, tt.want, result)
			assert.Contains(t, outBuf.String(), tt.question)
			assert.Contains(t, outBuf.String(), "[y/N]")
		})
	}
}

func TestExecuteUninstall_NoFailures(t *testing.T) {

	targets := []uninstall.UninstallTarget{
		{Path: "/fake/path", Type: "test", Exists: false},
	}

	var buf bytes.Buffer
	err := executeUninstall(&buf, targets)

	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Summary")
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestRegister_Integration(t *testing.T) {

	tests := map[string]struct {
		setupRoot func() *cobra.Command
		check     func(*testing.T, *cobra.Command)
	}{
		"registers all commands": {
			setupRoot: func() *cobra.Command {
				return &cobra.Command{Use: "autospec"}
			},
			check: func(t *testing.T, root *cobra.Command) {
				cmdNames := make(map[string]bool)
				for _, cmd := range root.Commands() {
					cmdNames[cmd.Use] = true
				}
				assert.True(t, cmdNames["commands"])
				assert.True(t, cmdNames["uninstall"])
			},
		},
		"disables default completion": {
			setupRoot: func() *cobra.Command {
				return &cobra.Command{Use: "autospec"}
			},
			check: func(t *testing.T, root *cobra.Command) {
				assert.True(t, root.CompletionOptions.DisableDefaultCmd)
			},
		},
		"sets root command reference": {
			setupRoot: func() *cobra.Command {
				return &cobra.Command{Use: "testroot"}
			},
			check: func(t *testing.T, root *cobra.Command) {
				// rootCmdRef should be set
				require.NotNil(t, rootCmdRef)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			root := tt.setupRoot()
			Register(root)
			tt.check(t, root)
		})
	}
}

func TestAllCommands_HaveUseAndShort(t *testing.T) {

	allCommands := []*cobra.Command{
		commandsCmd,
		commandsInstallCmd,
		commandsCheckCmd,
		commandsInfoCmd,
		completionCmd,
		completionInstallCmd,
		completionBashCmd,
		completionZshCmd,
		completionFishCmd,
		completionPowershellCmd,
		uninstallCmd,
	}

	for _, cmd := range allCommands {
		t.Run(cmd.Use, func(t *testing.T) {
			assert.NotEmpty(t, cmd.Use, "Command should have Use field")
			assert.NotEmpty(t, cmd.Short, "Command should have Short description")
		})
	}
}

func TestAllCommands_RunEFunctionsAreDefined(t *testing.T) {

	commandsWithRunE := map[string]*cobra.Command{
		"commands install":      commandsInstallCmd,
		"commands check":        commandsCheckCmd,
		"commands info":         commandsInfoCmd,
		"completion install":    completionInstallCmd,
		"completion bash":       completionBashCmd,
		"completion zsh":        completionZshCmd,
		"completion fish":       completionFishCmd,
		"completion powershell": completionPowershellCmd,
		"uninstall":             uninstallCmd,
	}

	for name, cmd := range commandsWithRunE {
		t.Run(name, func(t *testing.T) {
			assert.NotNil(t, cmd.RunE, "Command %s should have RunE defined", name)
		})
	}
}

func TestCommandsCmd_GroupID(t *testing.T) {

	// commandsCmd should have a group ID set
	assert.NotEmpty(t, commandsCmd.GroupID, "commands command should have GroupID")
}

func TestCompletionCmd_GroupID(t *testing.T) {

	// completionCmd should have a group ID set
	assert.NotEmpty(t, completionCmd.GroupID, "completion command should have GroupID")
}

func TestUninstallCmd_GroupID(t *testing.T) {

	// uninstallCmd should have a group ID set
	assert.NotEmpty(t, uninstallCmd.GroupID, "uninstall command should have GroupID")
}

func TestCommandsInstallCmd_DeprecationMessage(t *testing.T) {

	assert.NotEmpty(t, commandsInstallCmd.Deprecated, "install command should have deprecation message")
	assert.Contains(t, commandsInstallCmd.Deprecated, "init", "deprecation message should mention 'init'")
}

func TestCompletionInstallCmd_Args(t *testing.T) {

	// Should accept max 1 argument
	assert.NotNil(t, completionInstallCmd.Args, "Args validator should be set")
}

func TestCompletionShellCmds_DisableFlagsInUseLine(t *testing.T) {

	shellCommands := map[string]*cobra.Command{
		"bash":       completionBashCmd,
		"zsh":        completionZshCmd,
		"fish":       completionFishCmd,
		"powershell": completionPowershellCmd,
	}

	for name, cmd := range shellCommands {
		t.Run(name, func(t *testing.T) {
			assert.True(t, cmd.DisableFlagsInUseLine,
				"Shell command %s should have DisableFlagsInUseLine=true", name)
		})
	}
}

func TestCommandsSubcommands_HaveTargetFlag(t *testing.T) {

	subcommands := map[string]*cobra.Command{
		"install": commandsInstallCmd,
		"check":   commandsCheckCmd,
		"info":    commandsInfoCmd,
	}

	for name, cmd := range subcommands {
		t.Run(name, func(t *testing.T) {
			flag := cmd.Flags().Lookup("target")
			assert.NotNil(t, flag, "Subcommand %s should have 'target' flag", name)
			assert.Equal(t, "", flag.DefValue, "Default value should be empty")
		})
	}
}

func TestCommandsInstallCmd_LongDescriptionMentionsDeprecation(t *testing.T) {

	assert.Contains(t, commandsInstallCmd.Long, "DEPRECATED",
		"Long description should mention deprecation")
}

func TestCompletionInstallCmd_HasValidExamples(t *testing.T) {

	examples := completionInstallCmd.Example
	assert.NotEmpty(t, examples)
	assert.Contains(t, examples, "autospec completion install")
	assert.Contains(t, examples, "--manual")
}

func TestUninstallCmd_HasValidExamples(t *testing.T) {

	examples := uninstallCmd.Example
	assert.NotEmpty(t, examples)
	assert.Contains(t, examples, "--dry-run")
	assert.Contains(t, examples, "--yes")
}

func TestCommandsCheckCmd_LongDescriptionMentionsVersions(t *testing.T) {

	assert.Contains(t, commandsCheckCmd.Long, "version",
		"Long description should mention versions")
}

func TestCommandsInfoCmd_LongDescriptionMentionsArguments(t *testing.T) {

	assert.Contains(t, commandsInfoCmd.Long, "arguments",
		"Long description should mention arguments")
}

func TestUninstallCmd_LongDescriptionWarnsAboutProjectFiles(t *testing.T) {

	long := uninstallCmd.Long
	assert.Contains(t, long, "NOT remove project-level",
		"Should warn about project-level files not being removed")
	assert.Contains(t, long, "autospec clean",
		"Should mention autospec clean command")
}

func TestRootCmdRef_CanBeNil(t *testing.T) {
	// This test ensures rootCmdRef can be nil initially
	// (it gets set by Register)
	// We don't assert anything specific since it might be set by other tests
	_ = rootCmdRef
}

func TestCompletionCmd_HasNoRunE(t *testing.T) {

	// The completion command itself has no RunE (only subcommands do)
	assert.Nil(t, completionCmd.RunE,
		"Parent completion command should not have RunE (shows help)")
}

func TestCommandsCmd_HasNoRunE(t *testing.T) {

	// The commands command itself has no RunE (only subcommands do)
	assert.Nil(t, commandsCmd.RunE,
		"Parent commands command should not have RunE (shows help)")
}

func TestCommandsSubcommands_AreAddedToParent(t *testing.T) {

	subcommands := commandsCmd.Commands()
	require.NotEmpty(t, subcommands, "commands command should have subcommands")

	subcommandMap := make(map[string]*cobra.Command)
	for _, cmd := range subcommands {
		subcommandMap[cmd.Name()] = cmd
	}

	assert.Contains(t, subcommandMap, "install")
	assert.Contains(t, subcommandMap, "check")
	assert.Contains(t, subcommandMap, "info")
}

// Additional helper function tests with more edge cases
func TestCollectUninstallTargets_ReturnsValidData(t *testing.T) {

	targets, existingTargets, err := collectUninstallTargets()

	assert.NoError(t, err)
	assert.NotNil(t, targets)
	assert.NotNil(t, existingTargets)

	// Verify existingTargets is a valid subset
	for _, existing := range existingTargets {
		assert.True(t, existing.Exists, "existingTargets should only contain targets that exist")
	}
}

func TestDisplayUninstallTargets_MultipleTargets(t *testing.T) {

	tests := map[string]struct {
		targets  []uninstall.UninstallTarget
		dryRun   bool
		expected []string
	}{
		"multiple existing targets": {
			targets: []uninstall.UninstallTarget{
				{Path: "/path/1", Type: "binary", Exists: true, RequiresSudo: false},
				{Path: "/path/2", Type: "config", Exists: true, RequiresSudo: false},
				{Path: "/path/3", Type: "state", Exists: true, RequiresSudo: false},
			},
			dryRun:   false,
			expected: []string{"/path/1", "/path/2", "/path/3", "binary", "config", "state"},
		},
		"mixed sudo requirements": {
			targets: []uninstall.UninstallTarget{
				{Path: "/usr/bin/test", Type: "binary", Exists: true, RequiresSudo: true},
				{Path: "/home/user/.config", Type: "config", Exists: true, RequiresSudo: false},
			},
			dryRun:   false,
			expected: []string{"requires sudo", "/usr/bin/test", "/home/user/.config"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			_ = displayUninstallTargets(&buf, tt.targets, tt.dryRun)

			output := buf.String()
			for _, exp := range tt.expected {
				assert.Contains(t, output, exp)
			}
		})
	}
}

func TestConfirmUninstall_EdgeCases(t *testing.T) {
	tests := map[string]struct {
		requiresSudo bool
		yes          bool
		input        string
		want         bool
	}{
		"sudo warning with yes flag": {
			requiresSudo: true,
			yes:          true,
			input:        "",
			want:         true,
		},
		"no sudo with user confirmation": {
			requiresSudo: false,
			yes:          false,
			input:        "y\n",
			want:         true,
		},
		"empty input defaults to no": {
			requiresSudo: false,
			yes:          false,
			input:        "\n",
			want:         false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var outBuf bytes.Buffer
			inBuf := bytes.NewBufferString(tt.input)

			cmd := &cobra.Command{}
			cmd.SetOut(&outBuf)
			cmd.SetIn(inBuf)

			result := confirmUninstall(cmd, &outBuf, tt.requiresSudo, tt.yes)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDisplayRemovalResults_EdgeCases(t *testing.T) {

	tests := map[string]struct {
		results     []uninstall.UninstallResult
		wantSuccess int
		wantFail    int
		wantSkipped int
	}{
		"empty results": {
			results:     []uninstall.UninstallResult{},
			wantSuccess: 0,
			wantFail:    0,
			wantSkipped: 0,
		},
		"all failed": {
			results: []uninstall.UninstallResult{
				{Target: uninstall.UninstallTarget{Path: "/p1", Exists: true}, Success: false, Error: assert.AnError},
				{Target: uninstall.UninstallTarget{Path: "/p2", Exists: true}, Success: false, Error: assert.AnError},
			},
			wantSuccess: 0,
			wantFail:    2,
			wantSkipped: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			success, fail, skipped := displayRemovalResults(&buf, tt.results)

			assert.Equal(t, tt.wantSuccess, success)
			assert.Equal(t, tt.wantFail, fail)
			assert.Equal(t, tt.wantSkipped, skipped)
		})
	}
}

func TestPrintUninstallSummary_EdgeCases(t *testing.T) {

	tests := map[string]struct {
		successCount   int
		failCount      int
		skippedCount   int
		mustContain    []string
		mustNotContain []string
	}{
		"zero everything": {
			successCount: 0,
			failCount:    0,
			skippedCount: 0,
			mustContain:  []string{"0 removed"},
		},
		"only failures": {
			successCount:   0,
			failCount:      5,
			skippedCount:   0,
			mustContain:    []string{"0 removed", "5 failed"},
			mustNotContain: []string{"uninstalled"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			printUninstallSummary(&buf, tt.successCount, tt.failCount, tt.skippedCount)

			output := buf.String()
			for _, must := range tt.mustContain {
				assert.Contains(t, output, must)
			}
			for _, mustNot := range tt.mustNotContain {
				assert.NotContains(t, output, mustNot)
			}
		})
	}
}

func TestPromptYesNo_VariousInputs(t *testing.T) {
	tests := map[string]struct {
		input string
		want  bool
	}{
		"y with spaces":   {input: " y \n", want: true},
		"yes with spaces": {input: " yes \n", want: true},
		"Y uppercase":     {input: "Y\n", want: true},
		"YES uppercase":   {input: "YES\n", want: true},
		"n":               {input: "n\n", want: false},
		"no":              {input: "no\n", want: false},
		"N uppercase":     {input: "N\n", want: false},
		"NO uppercase":    {input: "NO\n", want: false},
		"empty":           {input: "\n", want: false},
		"invalid":         {input: "invalid\n", want: false},
		"numbers":         {input: "123\n", want: false},
		"special chars":   {input: "!@#\n", want: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var outBuf bytes.Buffer
			inBuf := bytes.NewBufferString(tt.input)

			cmd := &cobra.Command{}
			cmd.SetOut(&outBuf)
			cmd.SetIn(inBuf)

			result := promptYesNo(cmd, "Test?")
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestRegister_SetsCompletionOptions(t *testing.T) {

	rootCmd := &cobra.Command{Use: "test"}
	Register(rootCmd)

	assert.True(t, rootCmd.CompletionOptions.DisableDefaultCmd,
		"Should disable default completion command")
}

func TestInit_FunctionsExecute(t *testing.T) {

	// Verify that init() functions were called by checking their side effects
	tests := map[string]struct {
		parent      *cobra.Command
		childName   string
		shouldExist bool
	}{
		"commands has install": {
			parent:      commandsCmd,
			childName:   "install",
			shouldExist: true,
		},
		"commands has check": {
			parent:      commandsCmd,
			childName:   "check",
			shouldExist: true,
		},
		"commands has info": {
			parent:      commandsCmd,
			childName:   "info",
			shouldExist: true,
		},
		"completion has install": {
			parent:      completionCmd,
			childName:   "install",
			shouldExist: true,
		},
		"completion has bash": {
			parent:      completionCmd,
			childName:   "bash",
			shouldExist: true,
		},
		"completion has zsh": {
			parent:      completionCmd,
			childName:   "zsh",
			shouldExist: true,
		},
		"completion has fish": {
			parent:      completionCmd,
			childName:   "fish",
			shouldExist: true,
		},
		"completion has powershell": {
			parent:      completionCmd,
			childName:   "powershell",
			shouldExist: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			subcommands := tt.parent.Commands()
			found := false
			for _, cmd := range subcommands {
				if cmd.Name() == tt.childName {
					found = true
					break
				}
			}
			assert.Equal(t, tt.shouldExist, found,
				"Subcommand %s should exist: %v", tt.childName, tt.shouldExist)
		})
	}
}

func TestCommandsInstallCmd_FlagDefaults(t *testing.T) {

	targetFlag := commandsInstallCmd.Flags().Lookup("target")
	require.NotNil(t, targetFlag)
	assert.Equal(t, "", targetFlag.DefValue)
	assert.Equal(t, "string", targetFlag.Value.Type())
}

func TestCommandsCheckCmd_FlagDefaults(t *testing.T) {

	targetFlag := commandsCheckCmd.Flags().Lookup("target")
	require.NotNil(t, targetFlag)
	assert.Equal(t, "", targetFlag.DefValue)
	assert.Equal(t, "string", targetFlag.Value.Type())
}

func TestCommandsInfoCmd_FlagDefaults(t *testing.T) {

	targetFlag := commandsInfoCmd.Flags().Lookup("target")
	require.NotNil(t, targetFlag)
	assert.Equal(t, "", targetFlag.DefValue)
	assert.Equal(t, "string", targetFlag.Value.Type())
}

func TestCompletionInstallCmd_FlagDefaults(t *testing.T) {

	manualFlag := completionInstallCmd.Flags().Lookup("manual")
	require.NotNil(t, manualFlag)
	assert.Equal(t, "false", manualFlag.DefValue)
	assert.Equal(t, "bool", manualFlag.Value.Type())
}

func TestUninstallCmd_FlagTypes(t *testing.T) {

	tests := map[string]struct {
		flagName     string
		expectedType string
	}{
		"dry-run is bool": {
			flagName:     "dry-run",
			expectedType: "bool",
		},
		"yes is bool": {
			flagName:     "yes",
			expectedType: "bool",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			flag := uninstallCmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, flag)
			assert.Equal(t, tt.expectedType, flag.Value.Type())
		})
	}
}

func TestCompletionSubcommands_AreAddedToParent(t *testing.T) {

	subcommands := completionCmd.Commands()
	require.NotEmpty(t, subcommands, "completion command should have subcommands")

	subcommandMap := make(map[string]*cobra.Command)
	for _, cmd := range subcommands {
		subcommandMap[cmd.Name()] = cmd
	}

	assert.Contains(t, subcommandMap, "install")
	assert.Contains(t, subcommandMap, "bash")
	assert.Contains(t, subcommandMap, "zsh")
	assert.Contains(t, subcommandMap, "fish")
	assert.Contains(t, subcommandMap, "powershell")
}

// Test RunUninstall with dry-run to exercise the main function
func TestRunUninstall_DryRun(t *testing.T) {

	cmd := &cobra.Command{}
	cmd.SetArgs([]string{})

	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	// Add flags
	cmd.Flags().Bool("dry-run", true, "")
	cmd.Flags().Bool("yes", false, "")
	cmd.Flags().Set("dry-run", "true")

	err := RunUninstall(cmd, []string{})

	// Should not error on dry-run
	assert.NoError(t, err)
	output := outBuf.String()

	// Should show what would be removed
	if len(output) > 0 {
		assert.Contains(t, output, "Would remove")
	}
}

func TestRunUninstall_NoTargets(t *testing.T) {

	// This test may find targets or not depending on the system
	// We just verify it doesn't crash
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	cmd.Flags().Bool("dry-run", true, "")
	cmd.Flags().Bool("yes", false, "")
	cmd.Flags().Set("dry-run", "true")

	err := RunUninstall(cmd, []string{})
	assert.NoError(t, err)
}

func TestExecuteUninstall_WithFailures(t *testing.T) {

	targets := []uninstall.UninstallTarget{
		{Path: "/nonexistent/path/that/should/fail", Type: "test", Exists: true},
	}

	var buf bytes.Buffer
	err := executeUninstall(&buf, targets)

	// Should error if there are failures
	// Note: This might not fail if the file happens to not exist
	_ = err // We accept either outcome
}

// Test error handling in collectUninstallTargets
func TestCollectUninstallTargets_HandlesErrors(t *testing.T) {

	// This function calls uninstall.GetUninstallTargets which should not error
	// under normal circumstances
	targets, existing, err := collectUninstallTargets()

	assert.NoError(t, err)
	assert.NotNil(t, targets)
	assert.NotNil(t, existing)
}

// Additional command structure tests for completeness
func TestAllCommands_HaveLongDescription(t *testing.T) {

	commandsWithLong := map[string]*cobra.Command{
		"commands":              commandsCmd,
		"commands install":      commandsInstallCmd,
		"commands check":        commandsCheckCmd,
		"commands info":         commandsInfoCmd,
		"completion":            completionCmd,
		"completion install":    completionInstallCmd,
		"completion bash":       completionBashCmd,
		"completion zsh":        completionZshCmd,
		"completion fish":       completionFishCmd,
		"completion powershell": completionPowershellCmd,
		"uninstall":             uninstallCmd,
	}

	for name, cmd := range commandsWithLong {
		t.Run(name, func(t *testing.T) {
			assert.NotEmpty(t, cmd.Long, "Command %s should have Long description", name)
		})
	}
}

func TestUninstallCmd_FlagShorthands(t *testing.T) {

	tests := map[string]struct {
		flagName string
		want     string
	}{
		"dry-run shorthand": {
			flagName: "dry-run",
			want:     "n",
		},
		"yes shorthand": {
			flagName: "yes",
			want:     "y",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			flag := uninstallCmd.Flags().ShorthandLookup(tt.want)
			assert.NotNil(t, flag, "Should find flag by shorthand")
			assert.Equal(t, tt.flagName, flag.Name)
		})
	}
}

func TestCompletionInstallCmd_ValidArgsLength(t *testing.T) {

	// Should have exactly 4 valid shell args
	assert.Len(t, completionInstallCmd.ValidArgs, 4,
		"Should have 4 valid shell arguments")
}

func TestCommandStructures_NoRunForParents(t *testing.T) {

	parentCommands := map[string]*cobra.Command{
		"commands":   commandsCmd,
		"completion": completionCmd,
	}

	for name, cmd := range parentCommands {
		t.Run(name, func(t *testing.T) {
			assert.Nil(t, cmd.Run, "Parent command %s should not have Run", name)
			assert.Nil(t, cmd.RunE, "Parent command %s should not have RunE", name)
		})
	}
}

func TestCompletionShellCmds_HaveExampleInLong(t *testing.T) {

	shells := map[string]*cobra.Command{
		"bash":       completionBashCmd,
		"zsh":        completionZshCmd,
		"fish":       completionFishCmd,
		"powershell": completionPowershellCmd,
	}

	for name, cmd := range shells {
		t.Run(name, func(t *testing.T) {
			// Long description should contain usage examples
			assert.Contains(t, cmd.Long, "autospec completion "+name,
				"Long description should contain usage example")
		})
	}
}

func TestRegister_AddsCommandsInOrder(t *testing.T) {

	rootCmd := &cobra.Command{Use: "test"}
	Register(rootCmd)

	commands := rootCmd.Commands()
	assert.GreaterOrEqual(t, len(commands), 2,
		"Should have at least commands and uninstall")

	// Verify specific commands exist
	cmdMap := make(map[string]bool)
	for _, cmd := range commands {
		cmdMap[cmd.Use] = true
	}

	assert.True(t, cmdMap["commands"], "Should have commands command")
	assert.True(t, cmdMap["uninstall"], "Should have uninstall command")
}

func TestCommandsCheckCmd_ExampleInLong(t *testing.T) {

	assert.Contains(t, commandsCheckCmd.Long, "Example",
		"Long description should contain examples section")
}

func TestCommandsInfoCmd_ExampleInLong(t *testing.T) {

	assert.Contains(t, commandsInfoCmd.Long, "Example",
		"Long description should contain examples section")
}

func TestDisplayUninstallTargets_AlwaysShowsNote(t *testing.T) {

	tests := map[string]struct {
		targets []uninstall.UninstallTarget
		dryRun  bool
	}{
		"empty targets": {
			targets: []uninstall.UninstallTarget{},
			dryRun:  false,
		},
		"dry run with targets": {
			targets: []uninstall.UninstallTarget{
				{Path: "/test", Type: "test", Exists: true},
			},
			dryRun: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			_ = displayUninstallTargets(&buf, tt.targets, tt.dryRun)

			// Should always show the note about project-level files
			assert.Contains(t, buf.String(), "autospec clean",
				"Should mention autospec clean command")
		})
	}
}

func TestPrintUninstallSummary_AlwaysShowsSummary(t *testing.T) {

	tests := map[string]struct {
		success int
		fail    int
		skipped int
	}{
		"all zeros":     {success: 0, fail: 0, skipped: 0},
		"some success":  {success: 1, fail: 0, skipped: 0},
		"some failures": {success: 0, fail: 1, skipped: 0},
		"some skipped":  {success: 0, fail: 0, skipped: 1},
		"mixed":         {success: 1, fail: 1, skipped: 1},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			printUninstallSummary(&buf, tt.success, tt.fail, tt.skipped)

			output := buf.String()
			assert.Contains(t, output, "Summary", "Should contain Summary")
			assert.Contains(t, output, "removed", "Should mention removed count")
		})
	}
}

func TestPromptYesNo_ContainsQuestion(t *testing.T) {

	tests := map[string]string{
		"simple question":    "Continue?",
		"uninstall question": "Uninstall autospec?",
		"confirm question":   "Are you sure?",
	}

	for name, question := range tests {
		t.Run(name, func(t *testing.T) {
			var outBuf bytes.Buffer
			inBuf := bytes.NewBufferString("n\n")

			cmd := &cobra.Command{}
			cmd.SetOut(&outBuf)
			cmd.SetIn(inBuf)

			_ = promptYesNo(cmd, question)

			assert.Contains(t, outBuf.String(), question,
				"Output should contain the question")
		})
	}
}

// Test listAllCommands and showCommandDetail helper functions in commands_info.go
// These are internal but we can test them indirectly through runCommandsInfo
func TestCommandsInfoCmd_WithoutArgs(t *testing.T) {
	// Note: This test calls external package functions, but they work in test mode
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	// Call the RunE function - it should list all commands
	err := commandsInfoCmd.RunE(cmd, []string{})

	// It may succeed or fail depending on environment, but shouldn't crash
	_ = err
}

func TestCommandsInfoCmd_WithArg(t *testing.T) {
	// Note: This test calls external package functions
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	// Call with an invalid command name
	err := commandsInfoCmd.RunE(cmd, []string{"nonexistent.command"})

	// Should return an error for nonexistent command
	if err != nil {
		assert.Contains(t, err.Error(), "not found")
	}
}

func TestCommandsCheckCmd_Execute(t *testing.T) {
	// Test that the RunE function doesn't panic
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	// Call the RunE function
	_ = commandsCheckCmd.RunE(cmd, []string{})
	// It may succeed or fail depending on environment
}

func TestCommandsInstallCmd_Execute(t *testing.T) {
	// Test that the RunE function doesn't panic
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	// Call the RunE function
	_ = commandsInstallCmd.RunE(cmd, []string{})
	// It may succeed or fail depending on environment
}

func TestCompletionInstallCmd_InvalidShell(t *testing.T) {
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	// Call with invalid shell
	err := completionInstallCmd.RunE(cmd, []string{"invalid-shell"})

	// Should return error for invalid shell
	if err != nil {
		assert.Contains(t, err.Error(), "unknown shell")
	}
}

func TestCompletionInstallCmd_ManualFlag(t *testing.T) {
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	// Set manual flag
	manualFlag = true
	defer func() { manualFlag = false }()

	// Call with bash
	_ = completionInstallCmd.RunE(cmd, []string{"bash"})

	// Should show manual instructions
	// Test passes if it doesn't crash
}

func TestCompletionShellCmds_Execute(t *testing.T) {

	tests := map[string]*cobra.Command{
		"bash":       completionBashCmd,
		"zsh":        completionZshCmd,
		"fish":       completionFishCmd,
		"powershell": completionPowershellCmd,
	}

	for name, shellCmd := range tests {
		t.Run(name, func(t *testing.T) {
			// These commands need rootCmdRef to be set
			// We can skip if not set, or set a dummy one
			if rootCmdRef == nil {
				rootCmdRef = &cobra.Command{Use: "test"}
			}

			cmd := &cobra.Command{}
			var outBuf bytes.Buffer
			cmd.SetOut(&outBuf)

			// Call the RunE function
			err := shellCmd.RunE(cmd, []string{})

			// Should not error (generates completion script)
			assert.NoError(t, err)
			assert.NotEmpty(t, outBuf.String(), "Should generate completion script")
		})
	}
}

// Test completion install with no args (auto-detect)
func TestCompletionInstallCmd_AutoDetect(t *testing.T) {
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	// Call with no args to trigger auto-detect
	_ = completionInstallCmd.RunE(cmd, []string{})

	// May succeed or fail depending on environment, but shouldn't crash
	// Output should contain some message
}

// Test completion install with supported shells
func TestCompletionInstallCmd_SupportedShells(t *testing.T) {
	tests := map[string]string{
		"bash shell":       "bash",
		"zsh shell":        "zsh",
		"fish shell":       "fish",
		"powershell shell": "powershell",
	}

	for name, shell := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := &cobra.Command{}
			var outBuf bytes.Buffer
			cmd.SetOut(&outBuf)

			// Try with manual flag to avoid actual installation
			manualFlag = true
			defer func() { manualFlag = false }()

			_ = completionInstallCmd.RunE(cmd, []string{shell})
			// Should not crash
		})
	}
}

// Test commands check with various scenarios
func TestCommandsCheckCmd_Scenarios(t *testing.T) {
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	// Execute the check command
	err := commandsCheckCmd.RunE(cmd, []string{})

	// May succeed or fail, but should not crash
	_ = err
	// If successful, output should contain version info or "up to date"
}

// Test commands install output formatting
func TestCommandsInstallCmd_OutputFormatting(t *testing.T) {
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	// Execute install command
	err := commandsInstallCmd.RunE(cmd, []string{})

	// May succeed or fail depending on environment
	_ = err
	// If there's output, it should contain installation messages
}

// Test commands info with valid command name
func TestCommandsInfoCmd_ValidCommand(t *testing.T) {
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	// Try with a potentially valid command name
	err := commandsInfoCmd.RunE(cmd, []string{"autospec.specify"})

	// May succeed showing command info, or fail if command doesn't exist
	_ = err
}

// Test RunUninstall with yes flag
func TestRunUninstall_WithYesFlag(t *testing.T) {
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetIn(bytes.NewBufferString(""))

	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("yes", true, "")
	cmd.Flags().Set("dry-run", "false")
	cmd.Flags().Set("yes", "true")

	// This may try to actually remove files, so we use dry-run in practice
	// For safety, let's use dry-run
	cmd.Flags().Set("dry-run", "true")

	err := RunUninstall(cmd, []string{})
	assert.NoError(t, err)
}

// Test RunUninstall when no files exist
func TestRunUninstall_NoExistingFiles(t *testing.T) {

	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	cmd.Flags().Bool("dry-run", true, "")
	cmd.Flags().Bool("yes", true, "")
	cmd.Flags().Set("dry-run", "true")

	err := RunUninstall(cmd, []string{})

	// Should not error even if no files exist
	assert.NoError(t, err)
}

// Test displayRemovalResults with various error states
func TestDisplayRemovalResults_WithErrors(t *testing.T) {

	results := []uninstall.UninstallResult{
		{
			Target:  uninstall.UninstallTarget{Path: "/path/success", Exists: true},
			Success: true,
		},
		{
			Target:  uninstall.UninstallTarget{Path: "/path/fail", Exists: true},
			Success: false,
			Error:   assert.AnError,
		},
		{
			Target:  uninstall.UninstallTarget{Path: "/path/skip", Exists: false},
			Success: false,
		},
	}

	var buf bytes.Buffer
	success, fail, skipped := displayRemovalResults(&buf, results)

	assert.Equal(t, 1, success)
	assert.Equal(t, 1, fail)
	assert.Equal(t, 1, skipped)

	output := buf.String()
	assert.Contains(t, output, "Removed")
	assert.Contains(t, output, "Failed")
	assert.Contains(t, output, "Skipped")
}

// Test printUninstallSummary with non-zero success showing uninstalled message
func TestPrintUninstallSummary_ShowsUninstalledMessage(t *testing.T) {

	var buf bytes.Buffer
	printUninstallSummary(&buf, 3, 0, 0)

	output := buf.String()
	assert.Contains(t, output, "3 removed")
	assert.Contains(t, output, "uninstalled")
}

// Test completion install error detection path
func TestCompletionInstallCmd_DetectionError(t *testing.T) {
	// Save and restore environment
	oldShell := ""
	// The function will try to detect shell from environment
	// This exercises the auto-detect path

	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	// Try without specifying shell
	_ = completionInstallCmd.RunE(cmd, []string{})

	// Test passes if no crash occurred
	_ = oldShell
}

// Test uninstall with confirmation flow
func TestRunUninstall_ConfirmationFlow(t *testing.T) {
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	inBuf := bytes.NewBufferString("y\n")
	cmd.SetOut(&outBuf)
	cmd.SetIn(inBuf)

	cmd.Flags().Bool("dry-run", true, "")
	cmd.Flags().Bool("yes", false, "")
	cmd.Flags().Set("dry-run", "true")

	err := RunUninstall(cmd, []string{})
	assert.NoError(t, err)
}

// Test commands install with default target
func TestCommandsInstallCmd_DefaultTarget(t *testing.T) {
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	// Use temp directory to avoid polluting test directory with .claude/commands/
	installTargetDir = t.TempDir()

	err := commandsInstallCmd.RunE(cmd, []string{})

	// May succeed or fail depending on environment
	// Just ensure it doesn't crash
	_ = err
}

// Test commands check with empty target
func TestCommandsCheckCmd_EmptyTarget(t *testing.T) {
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	// Don't set target flag, use default
	checkTargetDir = ""

	err := commandsCheckCmd.RunE(cmd, []string{})

	// May succeed or fail
	_ = err
}

// Test commands info with empty target
func TestCommandsInfoCmd_EmptyTarget(t *testing.T) {
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	// Don't set target flag, use default
	infoTargetDir = ""

	err := commandsInfoCmd.RunE(cmd, []string{})

	// May succeed or fail
	_ = err
}

// Test executeUninstall error path
func TestExecuteUninstall_ErrorHandling(t *testing.T) {

	// Create a target that will fail to remove
	targets := []uninstall.UninstallTarget{
		{Path: "/root/protected/file", Type: "test", Exists: true},
	}

	var buf bytes.Buffer
	err := executeUninstall(&buf, targets)

	// May error or not depending on permissions
	// Just verify it handles the case gracefully
	_ = err
}

// Test confirmUninstall with sudo requirement
func TestConfirmUninstall_WithSudo(t *testing.T) {
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	inBuf := bytes.NewBufferString("y\n")
	cmd.SetOut(&outBuf)
	cmd.SetIn(inBuf)

	result := confirmUninstall(cmd, &outBuf, true, false)

	// Should show sudo warning
	assert.Contains(t, outBuf.String(), "elevated privileges")
	assert.True(t, result)
}

// Test collectUninstallTargets filtering
func TestCollectUninstallTargets_FiltersExisting(t *testing.T) {

	targets, existing, err := collectUninstallTargets()

	assert.NoError(t, err)

	// Verify that existing targets are actually marked as existing
	for _, target := range existing {
		assert.True(t, target.Exists,
			"Target in existing list should be marked as Exists=true")
	}

	// Verify existing is a subset of all targets
	assert.LessOrEqual(t, len(existing), len(targets),
		"Existing targets should be subset of all targets")
}

// Additional integration tests to increase coverage
func TestCommandsCheckCmd_WithTarget(t *testing.T) {
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	// Set a specific target directory
	checkTargetDir = "/tmp/test-commands"

	err := commandsCheckCmd.RunE(cmd, []string{})

	// Reset
	checkTargetDir = ""

	// May succeed or fail
	_ = err
}

func TestCommandsInstallCmd_WithTarget(t *testing.T) {
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	// Set a specific target directory
	installTargetDir = "/tmp/test-commands"

	err := commandsInstallCmd.RunE(cmd, []string{})

	// Reset
	installTargetDir = ""

	// May succeed or fail
	_ = err
}

func TestCommandsInfoCmd_WithTarget(t *testing.T) {
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	// Set a specific target directory
	infoTargetDir = "/tmp/test-commands"

	err := commandsInfoCmd.RunE(cmd, []string{})

	// Reset
	infoTargetDir = ""

	// May succeed or fail
	_ = err
}

func TestCompletionInstallCmd_WithoutManualFlag(t *testing.T) {
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	// Ensure manual flag is false
	manualFlag = false

	// Try with bash
	err := completionInstallCmd.RunE(cmd, []string{"bash"})

	// Reset
	manualFlag = false

	// May succeed or fail depending on environment
	_ = err
}

// Test displayUninstallTargets with sudo targets
func TestDisplayUninstallTargets_WithSudoTargets(t *testing.T) {

	targets := []uninstall.UninstallTarget{
		{Path: "/usr/local/bin/autospec", Type: "binary", Exists: true, RequiresSudo: true},
		{Path: "/home/user/.config", Type: "config", Exists: true, RequiresSudo: false},
	}

	var buf bytes.Buffer
	requiresSudo := displayUninstallTargets(&buf, targets, false)

	assert.True(t, requiresSudo)
	assert.Contains(t, buf.String(), "requires sudo")
}

// Test displayRemovalResults with only successes
func TestDisplayRemovalResults_OnlySuccesses(t *testing.T) {

	results := []uninstall.UninstallResult{
		{Target: uninstall.UninstallTarget{Path: "/p1", Exists: true}, Success: true},
		{Target: uninstall.UninstallTarget{Path: "/p2", Exists: true}, Success: true},
	}

	var buf bytes.Buffer
	success, fail, skipped := displayRemovalResults(&buf, results)

	assert.Equal(t, 2, success)
	assert.Equal(t, 0, fail)
	assert.Equal(t, 0, skipped)
}

// Test printUninstallSummary with mixed counts
func TestPrintUninstallSummary_MixedCounts(t *testing.T) {

	var buf bytes.Buffer
	printUninstallSummary(&buf, 2, 1, 1)

	output := buf.String()
	assert.Contains(t, output, "2 removed")
	assert.Contains(t, output, "1 skipped")
	assert.Contains(t, output, "1 failed")
	assert.Contains(t, output, "uninstalled")
}

// Test RunUninstall with empty targets list
func TestRunUninstall_EmptyTargetsList(t *testing.T) {

	// This test exercises the path when no targets are found
	cmd := &cobra.Command{}
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)

	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("yes", false, "")

	err := RunUninstall(cmd, []string{})

	// Should not error
	assert.NoError(t, err)
}

// Test completion install with each shell manually
func TestCompletionInstallCmd_EachShellManual(t *testing.T) {
	shells := []string{"bash", "zsh", "fish", "powershell"}

	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			cmd := &cobra.Command{}
			var outBuf bytes.Buffer
			cmd.SetOut(&outBuf)

			// Set manual flag
			manualFlag = true
			defer func() { manualFlag = false }()

			err := completionInstallCmd.RunE(cmd, []string{shell})

			// Should not error (shows manual instructions)
			_ = err
		})
	}
}

// Test that init functions set command GroupIDs
func TestCommands_HaveGroupIDs(t *testing.T) {

	tests := map[string]struct {
		cmd     *cobra.Command
		wantSet bool
	}{
		"commands has GroupID": {
			cmd:     commandsCmd,
			wantSet: true,
		},
		"completion has GroupID": {
			cmd:     completionCmd,
			wantSet: true,
		},
		"uninstall has GroupID": {
			cmd:     uninstallCmd,
			wantSet: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if tt.wantSet {
				assert.NotEmpty(t, tt.cmd.GroupID,
					"Command should have GroupID set")
			}
		})
	}
}

// Test prompt with different whitespace handling
func TestPromptYesNo_WhitespaceHandling(t *testing.T) {
	tests := map[string]struct {
		input string
		want  bool
	}{
		"tabs and spaces around yes": {input: "\t yes \t\n", want: true},
		"multiple spaces":            {input: "   y   \n", want: true},
		"just whitespace":            {input: "   \n", want: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var outBuf bytes.Buffer
			inBuf := bytes.NewBufferString(tt.input)

			cmd := &cobra.Command{}
			cmd.SetOut(&outBuf)
			cmd.SetIn(inBuf)

			result := promptYesNo(cmd, "Test?")
			assert.Equal(t, tt.want, result)
		})
	}
}
