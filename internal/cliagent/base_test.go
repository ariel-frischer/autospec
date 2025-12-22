package cliagent

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestBaseAgent_Name(t *testing.T) {
	t.Parallel()
	agent := &BaseAgent{AgentName: "test-agent"}
	if got := agent.Name(); got != "test-agent" {
		t.Errorf("Name() = %q, want %q", got, "test-agent")
	}
}

func TestBaseAgent_Capabilities(t *testing.T) {
	t.Parallel()
	caps := Caps{Automatable: true, RequiredEnv: []string{"API_KEY"}}
	agent := &BaseAgent{AgentCaps: caps}
	got := agent.Capabilities()
	if !got.Automatable {
		t.Error("Capabilities().Automatable = false, want true")
	}
	if len(got.RequiredEnv) != 1 || got.RequiredEnv[0] != "API_KEY" {
		t.Errorf("Capabilities().RequiredEnv = %v, want [API_KEY]", got.RequiredEnv)
	}
}

func TestBaseAgent_Validate(t *testing.T) {
	// Note: Cannot use t.Parallel() for parent test when subtests use t.Setenv

	tests := map[string]struct {
		agent   *BaseAgent
		setEnv  map[string]string
		wantErr bool
		errMsg  string
	}{
		"valid agent - echo exists": {
			agent: &BaseAgent{
				AgentName: "test",
				Cmd:       "echo",
				AgentCaps: Caps{},
			},
			wantErr: false,
		},
		"invalid agent - command not found": {
			agent: &BaseAgent{
				AgentName: "test",
				Cmd:       "nonexistent-command-12345",
				AgentCaps: Caps{},
			},
			wantErr: true,
			errMsg:  "not found in PATH",
		},
		"missing required env var": {
			agent: &BaseAgent{
				AgentName: "test",
				Cmd:       "echo",
				AgentCaps: Caps{RequiredEnv: []string{"CLIAGENT_TEST_MISSING_VAR"}},
			},
			wantErr: true,
			errMsg:  "CLIAGENT_TEST_MISSING_VAR is not set",
		},
		"required env var present": {
			agent: &BaseAgent{
				AgentName: "test",
				Cmd:       "echo",
				AgentCaps: Caps{RequiredEnv: []string{"CLIAGENT_TEST_VAR"}},
			},
			setEnv:  map[string]string{"CLIAGENT_TEST_VAR": "value"},
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			for k, v := range tt.setEnv {
				t.Setenv(k, v)
			}
			err := tt.agent.Validate()
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

func TestBaseAgent_BuildCommand(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agent    *BaseAgent
		prompt   string
		opts     ExecOptions
		wantArgs []string
	}{
		"arg method": {
			agent: &BaseAgent{
				Cmd: "agent",
				AgentCaps: Caps{
					PromptDelivery: PromptDelivery{
						Method: PromptMethodArg,
						Flag:   "-p",
					},
				},
			},
			prompt:   "fix bug",
			opts:     ExecOptions{},
			wantArgs: []string{"-p", "fix bug"},
		},
		"positional method": {
			agent: &BaseAgent{
				Cmd: "agent",
				AgentCaps: Caps{
					PromptDelivery: PromptDelivery{
						Method: PromptMethodPositional,
					},
				},
			},
			prompt:   "fix bug",
			opts:     ExecOptions{},
			wantArgs: []string{"fix bug"},
		},
		"subcommand method": {
			agent: &BaseAgent{
				Cmd: "agent",
				AgentCaps: Caps{
					PromptDelivery: PromptDelivery{
						Method: PromptMethodSubcommand,
						Flag:   "exec",
					},
				},
			},
			prompt:   "fix bug",
			opts:     ExecOptions{},
			wantArgs: []string{"exec", "fix bug"},
		},
		"subcommand-arg method": {
			agent: &BaseAgent{
				Cmd: "agent",
				AgentCaps: Caps{
					PromptDelivery: PromptDelivery{
						Method:     PromptMethodSubcommandArg,
						Flag:       "run",
						PromptFlag: "-t",
					},
				},
			},
			prompt:   "fix bug",
			opts:     ExecOptions{},
			wantArgs: []string{"run", "-t", "fix bug"},
		},
		"with autonomous flag": {
			agent: &BaseAgent{
				Cmd: "agent",
				AgentCaps: Caps{
					PromptDelivery: PromptDelivery{
						Method: PromptMethodArg,
						Flag:   "-p",
					},
					AutonomousFlag: "--yolo",
				},
			},
			prompt:   "fix bug",
			opts:     ExecOptions{Autonomous: true},
			wantArgs: []string{"-p", "fix bug", "--yolo"},
		},
		"with extra args": {
			agent: &BaseAgent{
				Cmd: "agent",
				AgentCaps: Caps{
					PromptDelivery: PromptDelivery{
						Method: PromptMethodArg,
						Flag:   "-p",
					},
				},
			},
			prompt:   "fix bug",
			opts:     ExecOptions{ExtraArgs: []string{"--verbose", "--debug"}},
			wantArgs: []string{"-p", "fix bug", "--verbose", "--debug"},
		},
		"autonomous with extra args": {
			agent: &BaseAgent{
				Cmd: "agent",
				AgentCaps: Caps{
					PromptDelivery: PromptDelivery{
						Method: PromptMethodArg,
						Flag:   "-p",
					},
					AutonomousFlag: "--auto",
				},
			},
			prompt:   "task",
			opts:     ExecOptions{Autonomous: true, ExtraArgs: []string{"--extra"}},
			wantArgs: []string{"-p", "task", "--auto", "--extra"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmd, err := tt.agent.BuildCommand(tt.prompt, tt.opts)
			if err != nil {
				t.Fatalf("BuildCommand() error = %v", err)
			}
			if len(cmd.Args) < 1 {
				t.Fatal("BuildCommand() returned cmd with no args")
			}
			gotArgs := cmd.Args[1:] // Skip the command name
			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("args len = %d, want %d\ngot: %v\nwant: %v", len(gotArgs), len(tt.wantArgs), gotArgs, tt.wantArgs)
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

func TestBaseAgent_BuildCommand_WorkDir(t *testing.T) {
	t.Parallel()
	agent := &BaseAgent{
		Cmd: "echo",
		AgentCaps: Caps{
			PromptDelivery: PromptDelivery{Method: PromptMethodPositional},
		},
	}
	cmd, _ := agent.BuildCommand("test", ExecOptions{WorkDir: "/tmp"})
	if cmd.Dir != "/tmp" {
		t.Errorf("cmd.Dir = %q, want %q", cmd.Dir, "/tmp")
	}
}

func TestBaseAgent_BuildCommand_Env(t *testing.T) {
	t.Parallel()
	agent := &BaseAgent{
		Cmd: "echo",
		AgentCaps: Caps{
			PromptDelivery: PromptDelivery{Method: PromptMethodPositional},
			AutonomousEnv:  map[string]string{"AUTO_MODE": "true"},
		},
	}

	tests := map[string]struct {
		opts       ExecOptions
		wantEnvKey string
	}{
		"autonomous env added": {
			opts:       ExecOptions{Autonomous: true},
			wantEnvKey: "AUTO_MODE=true",
		},
		"user env added": {
			opts:       ExecOptions{Env: map[string]string{"USER_VAR": "value"}},
			wantEnvKey: "USER_VAR=value",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmd, _ := agent.BuildCommand("test", tt.opts)
			found := false
			for _, e := range cmd.Env {
				if e == tt.wantEnvKey {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("env should contain %q", tt.wantEnvKey)
			}
		})
	}
}

func TestBaseAgent_BuildCommand_UseSubscription(t *testing.T) {
	t.Parallel()
	agent := &BaseAgent{
		Cmd: "echo",
		AgentCaps: Caps{
			PromptDelivery: PromptDelivery{Method: PromptMethodPositional},
		},
	}

	t.Run("subscription mode sets empty ANTHROPIC_API_KEY", func(t *testing.T) {
		t.Parallel()
		cmd, _ := agent.BuildCommand("test", ExecOptions{UseSubscription: true})

		// When UseSubscription is true, ANTHROPIC_API_KEY= (empty) should be appended
		// This should be the LAST occurrence (takes precedence over any shell value)
		found := false
		for _, e := range cmd.Env {
			if e == "ANTHROPIC_API_KEY=" {
				found = true
			}
		}
		if !found {
			t.Error("env should contain ANTHROPIC_API_KEY= (empty) when UseSubscription is true")
		}
	})

	t.Run("subscription mode overrides shell API key", func(t *testing.T) {
		t.Parallel()
		cmd, _ := agent.BuildCommand("test", ExecOptions{UseSubscription: true})

		// Count occurrences of ANTHROPIC_API_KEY - the last one should be empty
		var lastValue string
		for _, e := range cmd.Env {
			if strings.HasPrefix(e, "ANTHROPIC_API_KEY=") {
				lastValue = e
			}
		}
		// The last occurrence should be the empty override
		if lastValue != "ANTHROPIC_API_KEY=" {
			t.Errorf("last ANTHROPIC_API_KEY value = %q, want ANTHROPIC_API_KEY= (empty override)", lastValue)
		}
	})
}

func TestBaseAgent_Execute(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agent        *BaseAgent
		prompt       string
		opts         ExecOptions
		wantExitCode int
		wantStdout   string
	}{
		"successful execution": {
			agent: &BaseAgent{
				AgentName: "test",
				Cmd:       "echo",
				AgentCaps: Caps{
					PromptDelivery: PromptDelivery{Method: PromptMethodPositional},
				},
			},
			prompt:       "hello world",
			opts:         ExecOptions{},
			wantExitCode: 0,
			wantStdout:   "hello world\n",
		},
		"command with non-zero exit": {
			agent: &BaseAgent{
				AgentName: "test",
				Cmd:       "sh",
				AgentCaps: Caps{
					PromptDelivery: PromptDelivery{
						Method: PromptMethodArg,
						Flag:   "-c",
					},
				},
			},
			prompt:       "exit 42",
			opts:         ExecOptions{},
			wantExitCode: 42,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := tt.agent.Execute(context.Background(), tt.prompt, tt.opts)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if result.ExitCode != tt.wantExitCode {
				t.Errorf("ExitCode = %d, want %d", result.ExitCode, tt.wantExitCode)
			}
			if tt.wantStdout != "" && result.Stdout != tt.wantStdout {
				t.Errorf("Stdout = %q, want %q", result.Stdout, tt.wantStdout)
			}
			if result.Duration <= 0 {
				t.Error("Duration should be positive")
			}
		})
	}
}

func TestBaseAgent_Execute_Timeout(t *testing.T) {
	t.Parallel()
	agent := &BaseAgent{
		AgentName: "test",
		Cmd:       "sleep",
		AgentCaps: Caps{
			PromptDelivery: PromptDelivery{Method: PromptMethodPositional},
		},
	}

	ctx := context.Background()
	opts := ExecOptions{Timeout: 50 * time.Millisecond}

	_, err := agent.Execute(ctx, "10", opts)
	if err == nil {
		t.Error("Execute() should return error on timeout")
	}
}

func TestBaseAgent_Version(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agent       *BaseAgent
		wantVersion string
		wantErr     bool
	}{
		"no version flag": {
			agent:       &BaseAgent{AgentName: "test", Cmd: "echo", VersionFlag: ""},
			wantVersion: "unknown",
			wantErr:     false,
		},
		"with version flag": {
			agent:       &BaseAgent{AgentName: "test", Cmd: "echo", VersionFlag: "1.0.0"},
			wantVersion: "1.0.0",
			wantErr:     false,
		},
		"command not found": {
			agent:       &BaseAgent{AgentName: "test", Cmd: "nonexistent12345", VersionFlag: "--version"},
			wantVersion: "",
			wantErr:     true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.agent.Version()
			if (err != nil) != tt.wantErr {
				t.Errorf("Version() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantVersion {
				t.Errorf("Version() = %q, want %q", got, tt.wantVersion)
			}
		})
	}
}

func TestBaseAgent_Execute_CustomStdout(t *testing.T) {
	t.Parallel()
	agent := &BaseAgent{
		AgentName: "test",
		Cmd:       "echo",
		AgentCaps: Caps{
			PromptDelivery: PromptDelivery{Method: PromptMethodPositional},
		},
	}

	// Use custom writer - stdout should be empty in result
	var buf strings.Builder
	opts := ExecOptions{Stdout: &buf}
	result, err := agent.Execute(context.Background(), "hello", opts)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Stdout != "" {
		t.Errorf("result.Stdout should be empty when custom writer used, got %q", result.Stdout)
	}
	if !strings.Contains(buf.String(), "hello") {
		t.Errorf("custom writer should contain output, got %q", buf.String())
	}
}

func TestBaseAgent_BuildCommand_Interactive(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agent       *BaseAgent
		prompt      string
		opts        ExecOptions
		wantArgs    []string
		excludeArgs []string
	}{
		"automated mode includes DefaultArgs": {
			agent: &BaseAgent{
				Cmd: "claude",
				AgentCaps: Caps{
					PromptDelivery: PromptDelivery{
						Method: PromptMethodArg,
						Flag:   "-p",
					},
					DefaultArgs: []string{"--verbose", "--output-format", "stream-json"},
				},
			},
			prompt:   "analyze code",
			opts:     ExecOptions{Interactive: false},
			wantArgs: []string{"-p", "analyze code", "--verbose", "--output-format", "stream-json"},
		},
		"interactive mode uses positional prompt, excludes DefaultArgs and -p": {
			agent: &BaseAgent{
				Cmd: "claude",
				AgentCaps: Caps{
					PromptDelivery: PromptDelivery{
						Method: PromptMethodArg,
						Flag:   "-p",
					},
					DefaultArgs: []string{"--verbose", "--output-format", "stream-json"},
				},
			},
			prompt:      "analyze code",
			opts:        ExecOptions{Interactive: true},
			wantArgs:    []string{"analyze code"},
			excludeArgs: []string{"-p", "--verbose", "--output-format", "stream-json"},
		},
		"interactive mode with extra args": {
			agent: &BaseAgent{
				Cmd: "claude",
				AgentCaps: Caps{
					PromptDelivery: PromptDelivery{
						Method: PromptMethodArg,
						Flag:   "-p",
					},
					DefaultArgs: []string{"--verbose", "--output-format", "stream-json"},
				},
			},
			prompt:      "clarify spec",
			opts:        ExecOptions{Interactive: true, ExtraArgs: []string{"--model", "opus"}},
			wantArgs:    []string{"clarify spec", "--model", "opus"},
			excludeArgs: []string{"-p", "--verbose", "--output-format", "stream-json"},
		},
		"interactive with autonomous still adds autonomous flag": {
			agent: &BaseAgent{
				Cmd: "claude",
				AgentCaps: Caps{
					PromptDelivery: PromptDelivery{
						Method: PromptMethodArg,
						Flag:   "-p",
					},
					DefaultArgs:    []string{"--verbose"},
					AutonomousFlag: "--dangerously-skip-permissions",
				},
			},
			prompt:      "analyze code",
			opts:        ExecOptions{Interactive: true, Autonomous: true},
			wantArgs:    []string{"analyze code", "--dangerously-skip-permissions"},
			excludeArgs: []string{"-p", "--verbose"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmd, err := tt.agent.BuildCommand(tt.prompt, tt.opts)
			if err != nil {
				t.Fatalf("BuildCommand() error = %v", err)
			}
			gotArgs := cmd.Args[1:] // Skip the command name

			// Check expected args
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

			// Check excluded args are not present
			for _, excluded := range tt.excludeArgs {
				for _, got := range gotArgs {
					if got == excluded {
						t.Errorf("args should not contain %q in interactive mode", excluded)
					}
				}
			}
		})
	}
}
