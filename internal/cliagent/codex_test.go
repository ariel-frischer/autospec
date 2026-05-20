package cliagent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexCapabilities(t *testing.T) {
	t.Parallel()

	caps := NewCodex().Capabilities()

	if caps.PromptDelivery.Method != PromptMethodSubcommand {
		t.Errorf("PromptDelivery.Method = %q, want %q", caps.PromptDelivery.Method, PromptMethodSubcommand)
	}
	if caps.PromptDelivery.Flag != "exec" {
		t.Errorf("PromptDelivery.Flag = %q, want exec", caps.PromptDelivery.Flag)
	}
	if caps.AutonomousFlag != "--dangerously-bypass-approvals-and-sandbox" {
		t.Errorf("AutonomousFlag = %q, want --dangerously-bypass-approvals-and-sandbox", caps.AutonomousFlag)
	}
	if len(caps.RequiredEnv) != 0 {
		t.Errorf("RequiredEnv = %v, want empty", caps.RequiredEnv)
	}
	wantOptional := []string{"OPENAI_API_KEY", "CODEX_HOME", "OPENAI_BASE_URL"}
	for _, env := range wantOptional {
		if !containsString(caps.OptionalEnv, env) {
			t.Errorf("OptionalEnv = %v, want to contain %s", caps.OptionalEnv, env)
		}
	}
}

func TestCodexBuildCommand(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		opts     ExecOptions
		wantArgs []string
	}{
		"base command": {
			opts:     ExecOptions{},
			wantArgs: []string{"exec", "fix tests"},
		},
		"autonomous command": {
			opts:     ExecOptions{Autonomous: true},
			wantArgs: []string{"exec", "fix tests", "--dangerously-bypass-approvals-and-sandbox"},
		},
		"json output command": {
			opts:     ExecOptions{JSONOutput: true},
			wantArgs: []string{"exec", "--json", "fix tests"},
		},
		"json output autonomous command": {
			opts:     ExecOptions{JSONOutput: true, Autonomous: true},
			wantArgs: []string{"exec", "--json", "fix tests", "--dangerously-bypass-approvals-and-sandbox"},
		},
		"interactive command": {
			opts:     ExecOptions{Interactive: true},
			wantArgs: []string{"fix tests"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd, err := NewCodex().BuildCommand("fix tests", tt.opts)
			if err != nil {
				t.Fatalf("BuildCommand() error = %v", err)
			}
			gotArgs := cmd.Args[1:]
			if len(gotArgs) != len(tt.wantArgs) {
				t.Fatalf("args = %v, want %v", gotArgs, tt.wantArgs)
			}
			for i, want := range tt.wantArgs {
				if gotArgs[i] != want {
					t.Errorf("args[%d] = %q, want %q", i, gotArgs[i], want)
				}
			}
		})
	}
}

func TestCodexValidateDoesNotRequireOpenAIAPIKey(t *testing.T) {
	tempDir := t.TempDir()
	codexPath := filepath.Join(tempDir, "codex")
	if err := os.WriteFile(codexPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("writing codex shim: %v", err)
	}

	t.Setenv("PATH", tempDir)
	t.Setenv("OPENAI_API_KEY", "")

	if err := NewCodex().Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestCodexExecuteInvokesExecWithPrompt(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "codex.args")
	codexPath := filepath.Join(tempDir, "codex")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + shellQuote(logPath) + "\n"
	if err := os.WriteFile(codexPath, []byte(script), 0o755); err != nil {
		t.Fatalf("writing codex shim: %v", err)
	}

	t.Setenv("PATH", tempDir)

	_, err := NewCodex().Execute(context.Background(), "rendered prompt", ExecOptions{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading args log: %v", err)
	}
	want := "exec\nrendered prompt\n"
	if string(got) != want {
		t.Fatalf("args log = %q, want %q", string(got), want)
	}
}

func TestCodexExecuteAutonomousInvokesBypassFlag(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "codex.args")
	codexPath := filepath.Join(tempDir, "codex")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + shellQuote(logPath) + "\n"
	if err := os.WriteFile(codexPath, []byte(script), 0o755); err != nil {
		t.Fatalf("writing codex shim: %v", err)
	}

	t.Setenv("PATH", tempDir)

	_, err := NewCodex().Execute(context.Background(), "rendered prompt", ExecOptions{Autonomous: true})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading args log: %v", err)
	}
	want := "exec\nrendered prompt\n--dangerously-bypass-approvals-and-sandbox\n"
	if string(got) != want {
		t.Fatalf("args log = %q, want %q", string(got), want)
	}
}

func TestCodexConfigureProject(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	result, err := NewCodex().ConfigureProject(projectDir, "specs", true)
	if err != nil {
		t.Fatalf("ConfigureProject() error = %v", err)
	}

	if len(result.PermissionsAdded) != 0 {
		t.Errorf("PermissionsAdded = %v, want empty", result.PermissionsAdded)
	}
	if result.SettingsFilePath == "" {
		t.Fatal("SettingsFilePath is empty")
	}
	if !strings.HasSuffix(result.SettingsFilePath, filepath.Join(".codex", "config.toml")) {
		t.Errorf("SettingsFilePath = %q, want .codex/config.toml", result.SettingsFilePath)
	}
	if _, err := os.Stat(result.SettingsFilePath); err != nil {
		t.Fatalf("expected codex config to exist: %v", err)
	}
	configContent, err := os.ReadFile(result.SettingsFilePath)
	if err != nil {
		t.Fatalf("reading codex config: %v", err)
	}
	for _, skillName := range []string{
		"autospec-specify",
		"autospec-plan",
		"autospec-tasks",
		"autospec-implement",
		"autospec-constitution",
		"autospec-clarify",
		"autospec-analyze",
		"autospec-checklist",
		"autospec-worktree-setup",
	} {
		if !strings.Contains(string(configContent), `path = ".agents/skills/`+skillName+`"`) {
			t.Errorf("codex config missing %s registration:\n%s", skillName, string(configContent))
		}
		skillPath := filepath.Join(projectDir, ".agents", "skills", skillName, "SKILL.md")
		skillContent, err := os.ReadFile(skillPath)
		if err != nil {
			t.Fatalf("expected codex skill %s to exist: %v", skillName, err)
		}
		if !strings.Contains(string(skillContent), "name: "+skillName) {
			t.Errorf("codex skill %s missing frontmatter name:\n%s", skillName, string(skillContent))
		}
	}

	specifyPath := filepath.Join(projectDir, ".agents", "skills", "autospec-specify", "SKILL.md")
	specifyContent, err := os.ReadFile(specifyPath)
	if err != nil {
		t.Fatalf("reading specify skill: %v", err)
	}
	for _, want := range []string{
		`"$autospec-specify"`,
		`"/autospec.specify"`,
		"Generate a concise short name",
		"autospec new-feature",
	} {
		if !strings.Contains(string(specifyContent), want) {
			t.Errorf("codex specify skill missing %q:\n%s", want, string(specifyContent))
		}
	}

	clarifyPath := filepath.Join(projectDir, ".agents", "skills", "autospec-clarify", "SKILL.md")
	clarifyContent, err := os.ReadFile(clarifyPath)
	if err != nil {
		t.Fatalf("reading clarify skill: %v", err)
	}
	for _, want := range []string{
		`"$autospec-clarify"`,
		`"/autospec.clarify"`,
		"$autospec-plan",
		"Detect and reduce ambiguity",
	} {
		if !strings.Contains(string(clarifyContent), want) {
			t.Errorf("codex clarify skill missing %q:\n%s", want, string(clarifyContent))
		}
	}

	second, err := NewCodex().ConfigureProject(projectDir, "specs", true)
	if err != nil {
		t.Fatalf("second ConfigureProject() error = %v", err)
	}
	if !second.AlreadyConfigured {
		t.Error("second ConfigureProject() should report AlreadyConfigured")
	}
}

func TestCodexConfigureProjectRefreshesExistingSkill(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	skillPath := filepath.Join(projectDir, ".agents", "skills", "autospec-specify", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatalf("creating stale skill dir: %v", err)
	}
	if err := os.WriteFile(skillPath, []byte("stale autospec skill\n"), 0o644); err != nil {
		t.Fatalf("writing stale skill: %v", err)
	}

	_, err := NewCodex().ConfigureProject(projectDir, "specs", true)
	if err != nil {
		t.Fatalf("ConfigureProject() error = %v", err)
	}

	skillContent, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("reading refreshed skill: %v", err)
	}
	if !strings.Contains(string(skillContent), "name: autospec-specify") {
		t.Fatalf("skill was not refreshed:\n%s", string(skillContent))
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func shellQuote(path string) string {
	return "'" + strings.ReplaceAll(path, "'", "'\\''") + "'"
}
