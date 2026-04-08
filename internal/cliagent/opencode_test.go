package cliagent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/opencode"
)

func TestNewOpenCode(t *testing.T) {
	t.Parallel()
	agent := NewOpenCode()

	t.Run("agent name", func(t *testing.T) {
		t.Parallel()
		if got := agent.Name(); got != "opencode" {
			t.Errorf("Name() = %q, want %q", got, "opencode")
		}
	})

	t.Run("command", func(t *testing.T) {
		t.Parallel()
		if agent.Cmd != "opencode" {
			t.Errorf("Cmd = %q, want %q", agent.Cmd, "opencode")
		}
	})

	t.Run("version flag", func(t *testing.T) {
		t.Parallel()
		if agent.VersionFlag != "--version" {
			t.Errorf("VersionFlag = %q, want %q", agent.VersionFlag, "--version")
		}
	})

	t.Run("prompt delivery method", func(t *testing.T) {
		t.Parallel()
		caps := agent.Capabilities()
		if caps.PromptDelivery.Method != PromptMethodSubcommandWithFlag {
			t.Errorf("PromptDelivery.Method = %q, want %q",
				caps.PromptDelivery.Method, PromptMethodSubcommandWithFlag)
		}
	})

	t.Run("prompt delivery flag", func(t *testing.T) {
		t.Parallel()
		caps := agent.Capabilities()
		if caps.PromptDelivery.Flag != "run" {
			t.Errorf("PromptDelivery.Flag = %q, want %q",
				caps.PromptDelivery.Flag, "run")
		}
	})

	t.Run("prompt delivery command flag", func(t *testing.T) {
		t.Parallel()
		caps := agent.Capabilities()
		if caps.PromptDelivery.CommandFlag != "--command" {
			t.Errorf("PromptDelivery.CommandFlag = %q, want %q",
				caps.PromptDelivery.CommandFlag, "--command")
		}
	})

	t.Run("automatable", func(t *testing.T) {
		t.Parallel()
		caps := agent.Capabilities()
		if !caps.Automatable {
			t.Error("Automatable should be true")
		}
	})

	t.Run("no autonomous flag", func(t *testing.T) {
		t.Parallel()
		caps := agent.Capabilities()
		if caps.AutonomousFlag != "" {
			t.Errorf("AutonomousFlag = %q, want empty (run subcommand is inherently non-interactive)",
				caps.AutonomousFlag)
		}
	})

	t.Run("interactive flag", func(t *testing.T) {
		t.Parallel()
		caps := agent.Capabilities()
		if caps.PromptDelivery.InteractiveFlag != "--prompt" {
			t.Errorf("PromptDelivery.InteractiveFlag = %q, want %q",
				caps.PromptDelivery.InteractiveFlag, "--prompt")
		}
	})
}

func TestOpenCode_BuildCommand(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		prompt   string
		opts     ExecOptions
		wantArgs []string
	}{
		"basic prompt": {
			prompt:   "fix the bug",
			opts:     ExecOptions{},
			wantArgs: []string{"run", "fix the bug"},
		},
		"with slash command via ExtraArgs": {
			prompt:   "specify this feature",
			opts:     ExecOptions{ExtraArgs: []string{"--command", "autospec.specify"}},
			wantArgs: []string{"run", "specify this feature", "--command", "autospec.specify"},
		},
		"with plan command via ExtraArgs": {
			prompt:   "plan the implementation",
			opts:     ExecOptions{ExtraArgs: []string{"--command", "autospec.plan"}},
			wantArgs: []string{"run", "plan the implementation", "--command", "autospec.plan"},
		},
		"with implement command via ExtraArgs": {
			prompt:   "implement the feature",
			opts:     ExecOptions{ExtraArgs: []string{"--command", "autospec.implement"}},
			wantArgs: []string{"run", "implement the feature", "--command", "autospec.implement"},
		},
		"with retry prompt injection": {
			prompt: `Original task: implement feature

## Validation Errors (Retry 1/3)
- Missing required field: 'id'
- Invalid format for field 'description'

Please fix these validation errors and try again.`,
			opts: ExecOptions{ExtraArgs: []string{"--command", "autospec.specify"}},
			wantArgs: []string{"run", `Original task: implement feature

## Validation Errors (Retry 1/3)
- Missing required field: 'id'
- Invalid format for field 'description'

Please fix these validation errors and try again.`, "--command", "autospec.specify"},
		},
		"with multiple extra args": {
			prompt:   "analyze the code",
			opts:     ExecOptions{ExtraArgs: []string{"--model", "opus", "--command", "autospec.analyze"}},
			wantArgs: []string{"run", "analyze the code", "--model", "opus", "--command", "autospec.analyze"},
		},
		"with opencode-agent flag": {
			prompt:   "specify this feature",
			opts:     ExecOptions{OpenCodeAgent: "Plan", ExtraArgs: []string{"--command", "autospec.specify"}},
			wantArgs: []string{"run", "specify this feature", "--agent", "Plan", "--command", "autospec.specify"},
		},
		"with explore agent": {
			prompt:   "analyze the code",
			opts:     ExecOptions{OpenCodeAgent: "Explore", ExtraArgs: []string{"--command", "autospec.analyze"}},
			wantArgs: []string{"run", "analyze the code", "--agent", "Explore", "--command", "autospec.analyze"},
		},
		"with opencode-agent and no command flag": {
			prompt:   "run a quick task",
			opts:     ExecOptions{OpenCodeAgent: "Build"},
			wantArgs: []string{"run", "run a quick task", "--agent", "Build"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			agent := NewOpenCode()
			cmd, err := agent.BuildCommand(tt.prompt, tt.opts)
			if err != nil {
				t.Fatalf("BuildCommand() error = %v", err)
			}
			if len(cmd.Args) < 1 {
				t.Fatal("BuildCommand() returned cmd with no args")
			}
			gotArgs := cmd.Args[1:] // Skip the command name ("opencode")
			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("args len = %d, want %d\ngot: %v\nwant: %v",
					len(gotArgs), len(tt.wantArgs), gotArgs, tt.wantArgs)
				return
			}
			for i, arg := range gotArgs {
				if arg != tt.wantArgs[i] {
					t.Errorf("args[%d] = %q, want %q", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestOpenCode_BuildCommand_CommandName(t *testing.T) {
	t.Parallel()
	agent := NewOpenCode()

	// Verify the command executable name
	cmd, err := agent.BuildCommand("test prompt", ExecOptions{})
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}
	if cmd.Args[0] != "opencode" {
		t.Errorf("command = %q, want %q", cmd.Args[0], "opencode")
	}
}

func TestOpenCode_BuildCommand_Pattern(t *testing.T) {
	t.Parallel()
	agent := NewOpenCode()

	// Test the expected pattern: opencode run <prompt> --command <command-name>
	opts := ExecOptions{ExtraArgs: []string{"--command", "autospec.specify"}}
	cmd, err := agent.BuildCommand("specify my feature", opts)
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}

	args := cmd.Args
	// Expected: ["opencode", "run", "specify my feature", "--command", "autospec.specify"]
	if len(args) != 5 {
		t.Fatalf("expected 5 args, got %d: %v", len(args), args)
	}
	if args[0] != "opencode" {
		t.Errorf("args[0] = %q, want %q", args[0], "opencode")
	}
	if args[1] != "run" {
		t.Errorf("args[1] = %q, want %q", args[1], "run")
	}
	if args[2] != "specify my feature" {
		t.Errorf("args[2] = %q, want %q", args[2], "specify my feature")
	}
	if args[3] != "--command" {
		t.Errorf("args[3] = %q, want %q", args[3], "--command")
	}
	if args[4] != "autospec.specify" {
		t.Errorf("args[4] = %q, want %q", args[4], "autospec.specify")
	}
}

// TestOpenCode_BuildCommand_SlashCommand verifies the slash command parsing
// works correctly for OpenCode's PromptMethodSubcommandWithFlag.
func TestOpenCode_BuildCommand_SlashCommand(t *testing.T) {
	t.Parallel()
	agent := NewOpenCode()

	tests := map[string]struct {
		prompt   string
		wantArgs []string
	}{
		"slash command with quoted args": {
			prompt:   `/autospec.specify "feature description"`,
			wantArgs: []string{"opencode", "run", "feature description", "--command", "autospec.specify"},
		},
		"slash command without args": {
			prompt:   `/autospec.plan`,
			wantArgs: []string{"opencode", "run", "", "--command", "autospec.plan"},
		},
		"slash command with unquoted args": {
			prompt:   `/autospec.tasks generate all tasks`,
			wantArgs: []string{"opencode", "run", "generate all tasks", "--command", "autospec.tasks"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmd, err := agent.BuildCommand(tt.prompt, ExecOptions{})
			if err != nil {
				t.Fatalf("BuildCommand() error = %v", err)
			}
			if len(cmd.Args) != len(tt.wantArgs) {
				t.Fatalf("args len = %d, want %d\ngot: %v\nwant: %v",
					len(cmd.Args), len(tt.wantArgs), cmd.Args, tt.wantArgs)
			}
			for i, arg := range cmd.Args {
				if arg != tt.wantArgs[i] {
					t.Errorf("args[%d] = %q, want %q", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

// TestOpenCode_BuildCommand_Interactive verifies that OpenCode uses --prompt flag
// for interactive mode (e.g., clarify, analyze stages).
func TestOpenCode_BuildCommand_Interactive(t *testing.T) {
	t.Parallel()
	agent := NewOpenCode()

	tests := map[string]struct {
		prompt   string
		opts     ExecOptions
		wantArgs []string
	}{
		"interactive mode uses --prompt flag": {
			prompt:   "/autospec.clarify",
			opts:     ExecOptions{Interactive: true},
			wantArgs: []string{"opencode", "--prompt", "/autospec.clarify"},
		},
		"interactive mode with extra args": {
			prompt:   "/autospec.analyze",
			opts:     ExecOptions{Interactive: true, ExtraArgs: []string{"--model", "opus"}},
			wantArgs: []string{"opencode", "--prompt", "/autospec.analyze", "--model", "opus"},
		},
		"non-interactive mode uses run subcommand": {
			prompt:   `/autospec.specify "feature"`,
			opts:     ExecOptions{Interactive: false},
			wantArgs: []string{"opencode", "run", "feature", "--command", "autospec.specify"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmd, err := agent.BuildCommand(tt.prompt, tt.opts)
			if err != nil {
				t.Fatalf("BuildCommand() error = %v", err)
			}
			if len(cmd.Args) != len(tt.wantArgs) {
				t.Fatalf("args len = %d, want %d\ngot: %v\nwant: %v",
					len(cmd.Args), len(tt.wantArgs), cmd.Args, tt.wantArgs)
			}
			for i, arg := range cmd.Args {
				if arg != tt.wantArgs[i] {
					t.Errorf("args[%d] = %q, want %q", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestOpenCode_ConfigureProject(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		existingJSON         string
		specsDir             string
		wantAlreadyConfig    bool
		wantPermissionsLen   int // Now expects 2: 1 bash + 1 edit (string, not patterns)
		wantWarning          bool
		wantCommandsDirExist bool
	}{
		"fresh project without opencode.json": {
			existingJSON:         "",
			specsDir:             "specs",
			wantAlreadyConfig:    false,
			wantPermissionsLen:   2, // Bash(autospec *) + Edit(allow)
			wantWarning:          false,
			wantCommandsDirExist: true,
		},
		"project with empty opencode.json": {
			existingJSON:         "{}",
			specsDir:             "specs",
			wantAlreadyConfig:    false,
			wantPermissionsLen:   2,
			wantWarning:          false,
			wantCommandsDirExist: true,
		},
		"project with existing unrelated permissions": {
			existingJSON:         `{"permission": {"bash": {"npm *": "allow"}}}`,
			specsDir:             "specs",
			wantAlreadyConfig:    false,
			wantPermissionsLen:   2,
			wantWarning:          false,
			wantCommandsDirExist: true,
		},
		"project already configured": {
			// Now requires both bash AND edit: "allow" to be considered already configured
			existingJSON:         `{"permission": {"bash": {"autospec *": "allow"}, "edit": "allow"}}`,
			specsDir:             "specs",
			wantAlreadyConfig:    true,
			wantPermissionsLen:   0,
			wantWarning:          false,
			wantCommandsDirExist: true,
		},
		"project with only bash permission": {
			// Bash only is no longer sufficient - needs edit: "allow" too
			existingJSON:         `{"permission": {"bash": {"autospec *": "allow"}}}`,
			specsDir:             "specs",
			wantAlreadyConfig:    false,
			wantPermissionsLen:   2,
			wantWarning:          false,
			wantCommandsDirExist: true,
		},
		"project with denied permission": {
			existingJSON:         `{"permission": {"bash": {"autospec *": "deny"}}}`,
			specsDir:             "specs",
			wantAlreadyConfig:    false,
			wantPermissionsLen:   2,
			wantWarning:          true,
			wantCommandsDirExist: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create temp directory
			tempDir := t.TempDir()

			// Write existing opencode.json if provided
			if tt.existingJSON != "" {
				jsonPath := filepath.Join(tempDir, "opencode.json")
				if err := os.WriteFile(jsonPath, []byte(tt.existingJSON), 0o644); err != nil {
					t.Fatalf("failed to write test opencode.json: %v", err)
				}
			}

			// Create OpenCode agent and call ConfigureProject
			// Use projectLevel=true to test project-level config (the test creates tempDir with opencode.json)
			agent := NewOpenCode()
			result, err := agent.ConfigureProject(tempDir, tt.specsDir, true)
			if err != nil {
				t.Fatalf("ConfigureProject() error = %v", err)
			}

			// Check AlreadyConfigured
			if result.AlreadyConfigured != tt.wantAlreadyConfig {
				t.Errorf("AlreadyConfigured = %v, want %v", result.AlreadyConfigured, tt.wantAlreadyConfig)
			}

			// Check PermissionsAdded length
			if len(result.PermissionsAdded) != tt.wantPermissionsLen {
				t.Errorf("PermissionsAdded len = %d, want %d", len(result.PermissionsAdded), tt.wantPermissionsLen)
			}

			// Check Warning
			hasWarning := result.Warning != ""
			if hasWarning != tt.wantWarning {
				t.Errorf("Warning present = %v, want %v (warning: %q)", hasWarning, tt.wantWarning, result.Warning)
			}

			// Check commands directory exists
			cmdDir := filepath.Join(tempDir, ".opencode", "command")
			_, err = os.Stat(cmdDir)
			exists := err == nil
			if exists != tt.wantCommandsDirExist {
				t.Errorf("commands dir exists = %v, want %v", exists, tt.wantCommandsDirExist)
			}
		})
	}
}

func TestOpenCode_ConfigureProject_CommandsInstalled(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	agent := NewOpenCode()

	_, err := agent.ConfigureProject(tempDir, "specs", true)
	if err != nil {
		t.Fatalf("ConfigureProject() error = %v", err)
	}

	// Verify command templates are installed
	cmdDir := filepath.Join(tempDir, ".opencode", "command")
	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		t.Fatalf("failed to read commands dir: %v", err)
	}

	// Should have at least one autospec.*.md file
	foundAutospec := false
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			if len(entry.Name()) > 9 && entry.Name()[:9] == "autospec." {
				foundAutospec = true
				break
			}
		}
	}
	if !foundAutospec {
		t.Error("no autospec.*.md files found in .opencode/command/")
	}
}

func TestOpenCode_ConfigureProject_PermissionInJSON(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	agent := NewOpenCode()

	_, err := agent.ConfigureProject(tempDir, "specs", true)
	if err != nil {
		t.Fatalf("ConfigureProject() error = %v", err)
	}

	// Read and verify opencode.json
	jsonPath := filepath.Join(tempDir, "opencode.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read opencode.json: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse opencode.json: %v", err)
	}

	// Check permission.bash["autospec *"] = "allow"
	perm, ok := config["permission"].(map[string]interface{})
	if !ok {
		t.Fatal("opencode.json missing 'permission' object")
	}
	bash, ok := perm["bash"].(map[string]interface{})
	if !ok {
		t.Fatal("opencode.json missing 'permission.bash' object")
	}
	autospecPerm, ok := bash[opencode.RequiredPattern].(string)
	if !ok {
		t.Fatalf("opencode.json missing 'permission.bash[%q]'", opencode.RequiredPattern)
	}
	if autospecPerm != opencode.PermissionAllow {
		t.Errorf("permission.bash[%q] = %q, want %q",
			opencode.RequiredPattern, autospecPerm, opencode.PermissionAllow)
	}
}

func TestOpenCode_ConfigureProject_Idempotency(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	agent := NewOpenCode()

	// First call
	result1, err := agent.ConfigureProject(tempDir, "specs", true)
	if err != nil {
		t.Fatalf("first ConfigureProject() error = %v", err)
	}
	if result1.AlreadyConfigured {
		t.Error("first call should not be AlreadyConfigured")
	}
	if len(result1.PermissionsAdded) == 0 {
		t.Error("first call should add permissions")
	}

	// Second call
	result2, err := agent.ConfigureProject(tempDir, "specs", true)
	if err != nil {
		t.Fatalf("second ConfigureProject() error = %v", err)
	}
	if !result2.AlreadyConfigured {
		t.Error("second call should be AlreadyConfigured")
	}
	if len(result2.PermissionsAdded) > 0 {
		t.Error("second call should not add permissions")
	}

	// Third call
	result3, err := agent.ConfigureProject(tempDir, "specs", true)
	if err != nil {
		t.Fatalf("third ConfigureProject() error = %v", err)
	}
	if !result3.AlreadyConfigured {
		t.Error("third call should be AlreadyConfigured")
	}
}

func TestOpenCode_ConfigureProject_PreservesExistingConfig(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Write existing opencode.json with other settings
	existingJSON := `{
  "theme": "dark",
  "permission": {
    "bash": {
      "npm *": "allow",
      "git *": "allow"
    }
  },
  "model": "claude-3-5-sonnet"
}`
	jsonPath := filepath.Join(tempDir, "opencode.json")
	if err := os.WriteFile(jsonPath, []byte(existingJSON), 0o644); err != nil {
		t.Fatalf("failed to write test opencode.json: %v", err)
	}

	agent := NewOpenCode()
	_, err := agent.ConfigureProject(tempDir, "specs", true)
	if err != nil {
		t.Fatalf("ConfigureProject() error = %v", err)
	}

	// Read and verify all settings are preserved
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read opencode.json: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse opencode.json: %v", err)
	}

	// Check existing fields preserved
	if config["theme"] != "dark" {
		t.Errorf("theme not preserved, got %v", config["theme"])
	}
	if config["model"] != "claude-3-5-sonnet" {
		t.Errorf("model not preserved, got %v", config["model"])
	}

	// Check existing permissions preserved
	perm := config["permission"].(map[string]interface{})
	bash := perm["bash"].(map[string]interface{})
	if bash["npm *"] != "allow" {
		t.Errorf("existing npm permission not preserved, got %v", bash["npm *"])
	}
	if bash["git *"] != "allow" {
		t.Errorf("existing git permission not preserved, got %v", bash["git *"])
	}
	// Check new permission added
	if bash[opencode.RequiredPattern] != opencode.PermissionAllow {
		t.Errorf("autospec permission not added, got %v", bash[opencode.RequiredPattern])
	}
}

func TestOpenCodeImplementsConfigurator(t *testing.T) {
	t.Parallel()

	// Compile-time check that OpenCode implements Configurator
	var _ Configurator = (*OpenCode)(nil)

	// Runtime check via IsConfigurator
	agent := NewOpenCode()
	if !IsConfigurator(agent) {
		t.Error("OpenCode should implement Configurator interface")
	}
}
