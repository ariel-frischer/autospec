package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Note: These tests cannot run in parallel because they use the global rootCmd
// which has shared state. Each test changes directory and executes commands.

func TestConfigSetCommand(t *testing.T) {
	tests := map[string]struct {
		args           []string
		setup          func(t *testing.T, dir string)
		wantOutput     []string
		wantErr        bool
		wantErrContain string
	}{
		"set value with project flag": {
			args: []string{"config", "set", "max_retries", "5", "--project"},
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, ".autospec"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantOutput: []string{"Set max_retries = 5 in project config"},
		},
		"invalid key with project": {
			args: []string{"config", "set", "invalid.key", "value", "--project"},
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, ".autospec"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr:        true,
			wantErrContain: "unknown configuration key",
		},
		"unsupported stage model": {
			args: []string{"config", "set", "models.unsupported", "some-model", "--project"},
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, ".autospec"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr:        true,
			wantErrContain: "unknown configuration key",
		},
		"invalid value type with project": {
			args: []string{"config", "set", "max_retries", "not-a-number", "--project"},
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, ".autospec"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr:        true,
			wantErrContain: "invalid integer",
		},
		"set enum value": {
			args: []string{"config", "set", "notifications.type", "sound", "--project"},
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, ".autospec"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantOutput: []string{"Set notifications.type = sound"},
		},
		"invalid enum value with project": {
			args: []string{"config", "set", "notifications.type", "invalid", "--project"},
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, ".autospec"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr:        true,
			wantErrContain: "valid options: sound, visual, both",
		},
		"set nested boolean": {
			args: []string{"config", "set", "notifications.enabled", "true", "--project"},
			setup: func(t *testing.T, dir string) {
				if err := os.MkdirAll(filepath.Join(dir, ".autospec"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantOutput: []string{"Set notifications.enabled = true"},
		},
		"project flag without project dir": {
			args:           []string{"config", "set", "max_retries", "5", "--project"},
			wantErr:        true,
			wantErrContain: "not in a project directory",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			origDir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.Chdir(origDir) }()

			if err := os.Chdir(tmpDir); err != nil {
				t.Fatal(err)
			}

			if tt.setup != nil {
				tt.setup(t, tmpDir)
			}

			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs(tt.args)

			err = rootCmd.Execute()

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.wantErrContain != "" && !strings.Contains(err.Error(), tt.wantErrContain) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErrContain)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := buf.String()
			for _, want := range tt.wantOutput {
				if !strings.Contains(output, want) {
					t.Errorf("output = %q, want to contain %q", output, want)
				}
			}
		})
	}
}

func TestConfigSetAndGetStageModels(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	projectDir := filepath.Join(tmpDir, ".autospec")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(projectDir, "config.yml")
	if err := os.WriteFile(configPath, []byte("max_retries: 4\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := map[string]string{"constitution": "provider/constitution-model", "implement": "provider/implement-model"}
	for stage, model := range tests {
		runConfigCommand(t, []string{"config", "set", "models." + stage, model, "--project"}, "Set models."+stage+" = "+model)
		runConfigCommand(t, []string{"config", "get", "models." + stage, "--project"}, "models."+stage+": "+model)
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "max_retries: 4") {
		t.Errorf("unrelated config was modified: %s", content)
	}
}

func runConfigCommand(t *testing.T, args []string, want string) {
	t.Helper()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("%v: %v", args, err)
	}
	if !strings.Contains(buf.String(), want) {
		t.Errorf("output = %q, want to contain %q", buf.String(), want)
	}
}

func TestConfigGetCommand(t *testing.T) {
	tests := map[string]struct {
		args           []string
		setup          func(t *testing.T, dir string)
		wantOutput     []string
		wantErr        bool
		wantErrContain string
	}{
		"get default value with project": {
			args: []string{"config", "get", "max_retries", "--project"},
			setup: func(t *testing.T, dir string) {
				// Create empty .autospec dir
				if err := os.MkdirAll(filepath.Join(dir, ".autospec"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantOutput: []string{"max_retries:", "not set"},
		},
		"get value from project config": {
			args: []string{"config", "get", "max_retries"},
			setup: func(t *testing.T, dir string) {
				projectDir := filepath.Join(dir, ".autospec")
				if err := os.MkdirAll(projectDir, 0o755); err != nil {
					t.Fatal(err)
				}
				configPath := filepath.Join(projectDir, "config.yml")
				if err := os.WriteFile(configPath, []byte("max_retries: 7\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantOutput: []string{"max_retries: 7", "project config"},
		},
		"unknown key": {
			args:           []string{"config", "get", "unknown.key"},
			wantErr:        true,
			wantErrContain: "unknown configuration key",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			origDir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.Chdir(origDir) }()

			if err := os.Chdir(tmpDir); err != nil {
				t.Fatal(err)
			}

			if tt.setup != nil {
				tt.setup(t, tmpDir)
			}

			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs(tt.args)

			err = rootCmd.Execute()

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.wantErrContain != "" && !strings.Contains(err.Error(), tt.wantErrContain) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErrContain)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := buf.String()
			for _, want := range tt.wantOutput {
				if !strings.Contains(output, want) {
					t.Errorf("output = %q, want to contain %q", output, want)
				}
			}
		})
	}
}

func TestConfigToggleCommand(t *testing.T) {
	tests := map[string]struct {
		args           []string
		setup          func(t *testing.T, dir string)
		wantOutput     []string
		wantErr        bool
		wantErrContain string
	}{
		"toggle from false to true": {
			args: []string{"config", "toggle", "notifications.enabled", "--project"},
			setup: func(t *testing.T, dir string) {
				projectDir := filepath.Join(dir, ".autospec")
				if err := os.MkdirAll(projectDir, 0o755); err != nil {
					t.Fatal(err)
				}
				configPath := filepath.Join(projectDir, "config.yml")
				if err := os.WriteFile(configPath, []byte("notifications:\n  enabled: false\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantOutput: []string{"Toggled notifications.enabled: false -> true"},
		},
		"toggle from true to false": {
			args: []string{"config", "toggle", "notifications.enabled", "--project"},
			setup: func(t *testing.T, dir string) {
				projectDir := filepath.Join(dir, ".autospec")
				if err := os.MkdirAll(projectDir, 0o755); err != nil {
					t.Fatal(err)
				}
				configPath := filepath.Join(projectDir, "config.yml")
				if err := os.WriteFile(configPath, []byte("notifications:\n  enabled: true\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantOutput: []string{"Toggled notifications.enabled: true -> false"},
		},
		"toggle missing key creates as true": {
			args: []string{"config", "toggle", "skip_preflight", "--project"},
			setup: func(t *testing.T, dir string) {
				projectDir := filepath.Join(dir, ".autospec")
				if err := os.MkdirAll(projectDir, 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantOutput: []string{"Toggled skip_preflight: false -> true"},
		},
		"toggle non-boolean key fails": {
			args:           []string{"config", "toggle", "max_retries"},
			wantErr:        true,
			wantErrContain: "not a boolean",
		},
		"toggle unknown key fails": {
			args:           []string{"config", "toggle", "unknown.key"},
			wantErr:        true,
			wantErrContain: "unknown configuration key",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			origDir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.Chdir(origDir) }()

			if err := os.Chdir(tmpDir); err != nil {
				t.Fatal(err)
			}

			if tt.setup != nil {
				tt.setup(t, tmpDir)
			}

			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs(tt.args)

			err = rootCmd.Execute()

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.wantErrContain != "" && !strings.Contains(err.Error(), tt.wantErrContain) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErrContain)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := buf.String()
			for _, want := range tt.wantOutput {
				if !strings.Contains(output, want) {
					t.Errorf("output = %q, want to contain %q", output, want)
				}
			}
		})
	}
}

func TestConfigKeysCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"config", "keys"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	expectedKeys := []string{
		"max_retries",
		"notifications.enabled",
		"notifications.type",
		"timeout",
		"skip_preflight",
		"specs_dir",
		"models.constitution",
		"models.specify",
		"models.clarify",
		"models.plan",
		"models.tasks",
		"models.checklist",
		"models.analyze",
		"models.implement",
	}

	for _, key := range expectedKeys {
		if !strings.Contains(output, key) {
			t.Errorf("output should contain key %q, got %q", key, output)
		}
	}
}
