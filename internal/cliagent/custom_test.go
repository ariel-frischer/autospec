package cliagent

import (
	"context"
	"strings"
	"testing"
)

func TestNewCustomAgent(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		template string
		wantErr  bool
		errMsg   string
	}{
		"valid template": {
			template: "echo {{PROMPT}}",
			wantErr:  false,
		},
		"missing placeholder": {
			template: "echo hello",
			wantErr:  true,
			errMsg:   "must contain {{PROMPT}}",
		},
		"complex template": {
			template: "aider --model sonnet --yes-always --message {{PROMPT}}",
			wantErr:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := NewCustomAgent(tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCustomAgent() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestCustomAgent_Name(t *testing.T) {
	t.Parallel()
	agent, _ := NewCustomAgent("echo {{PROMPT}}")
	if got := agent.Name(); got != "custom" {
		t.Errorf("Name() = %q, want %q", got, "custom")
	}
}

func TestCustomAgent_Version(t *testing.T) {
	t.Parallel()
	agent, _ := NewCustomAgent("echo {{PROMPT}}")
	ver, err := agent.Version()
	if err != nil {
		t.Fatalf("Version() error = %v", err)
	}
	if ver != "custom" {
		t.Errorf("Version() = %q, want %q", ver, "custom")
	}
}

func TestCustomAgent_Capabilities(t *testing.T) {
	t.Parallel()
	agent, _ := NewCustomAgent("echo {{PROMPT}}")
	caps := agent.Capabilities()
	if !caps.Automatable {
		t.Error("Automatable should be true")
	}
	if caps.PromptDelivery.Method != PromptMethodTemplate {
		t.Errorf("PromptDelivery.Method = %q, want %q", caps.PromptDelivery.Method, PromptMethodTemplate)
	}
}

func TestCustomAgent_Validate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		template string
		wantErr  bool
		errMsg   string
	}{
		"valid command": {
			template: "echo {{PROMPT}}",
			wantErr:  false,
		},
		"command not found": {
			template: "nonexistent-cmd-12345 {{PROMPT}}",
			wantErr:  true,
			errMsg:   "not found in PATH",
		},
		"invalid shell syntax": {
			template: "echo '{{PROMPT}}", // unmatched quote
			wantErr:  true,
			errMsg:   "invalid template",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			agent, err := NewCustomAgent(tt.template)
			if err != nil {
				t.Fatalf("NewCustomAgent() error = %v", err)
			}
			err = agent.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestCustomAgent_BuildCommand(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		template string
		prompt   string
		wantCmd  string
		wantArgs []string
	}{
		"basic substitution": {
			template: "echo {{PROMPT}}",
			prompt:   "hello world",
			wantCmd:  "echo",
			wantArgs: []string{"hello world"},
		},
		"multiple args": {
			template: "myapp --message {{PROMPT}} --verbose",
			prompt:   "do something",
			wantCmd:  "myapp",
			wantArgs: []string{"--message", "do something", "--verbose"},
		},
		"prompt with special chars": {
			template: "echo {{PROMPT}}",
			prompt:   "hello \"world\" with 'quotes'",
			wantCmd:  "echo",
			wantArgs: []string{"hello \"world\" with 'quotes'"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			agent, _ := NewCustomAgent(tt.template)
			cmd, err := agent.BuildCommand(tt.prompt, ExecOptions{})
			if err != nil {
				t.Fatalf("BuildCommand() error = %v", err)
			}
			if cmd.Path == "" {
				t.Error("cmd.Path should not be empty")
			}
			// Args[0] is the program path, actual args start at [1]
			gotArgs := cmd.Args[1:]
			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("args = %v, want %v", gotArgs, tt.wantArgs)
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

func TestCustomAgent_Execute(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		template   string
		prompt     string
		wantStdout string
		wantExit   int
	}{
		"echo prompt": {
			template:   "echo {{PROMPT}}",
			prompt:     "hello",
			wantStdout: "hello\n",
			wantExit:   0,
		},
		"exit code": {
			template: "sh -c {{PROMPT}}",
			prompt:   "exit 42",
			wantExit: 42,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			agent, _ := NewCustomAgent(tt.template)
			result, err := agent.Execute(context.Background(), tt.prompt, ExecOptions{})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if result.ExitCode != tt.wantExit {
				t.Errorf("ExitCode = %d, want %d", result.ExitCode, tt.wantExit)
			}
			if tt.wantStdout != "" && result.Stdout != tt.wantStdout {
				t.Errorf("Stdout = %q, want %q", result.Stdout, tt.wantStdout)
			}
		})
	}
}

func TestCustomAgent_PromptWithNewlines(t *testing.T) {
	t.Parallel()
	agent, _ := NewCustomAgent("echo {{PROMPT}}")
	prompt := "line1\nline2\nline3"
	cmd, err := agent.BuildCommand(prompt, ExecOptions{})
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}
	// The prompt should be preserved with newlines
	if len(cmd.Args) < 2 {
		t.Fatal("expected at least 2 args")
	}
	if !strings.Contains(cmd.Args[1], "\n") {
		t.Errorf("prompt should contain newlines, got %q", cmd.Args[1])
	}
}

func TestCustomAgent_WorkDir(t *testing.T) {
	t.Parallel()
	agent, _ := NewCustomAgent("pwd {{PROMPT}}")
	cmd, _ := agent.BuildCommand("ignored", ExecOptions{WorkDir: "/tmp"})
	if cmd.Dir != "/tmp" {
		t.Errorf("cmd.Dir = %q, want %q", cmd.Dir, "/tmp")
	}
}

func TestCustomAgent_Env(t *testing.T) {
	t.Parallel()
	agent, _ := NewCustomAgent("echo {{PROMPT}}")
	cmd, _ := agent.BuildCommand("test", ExecOptions{
		Env: map[string]string{"MY_VAR": "my_value"},
	})
	found := false
	for _, e := range cmd.Env {
		if e == "MY_VAR=my_value" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected MY_VAR=my_value in env")
	}
}
