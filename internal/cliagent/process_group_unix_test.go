//go:build unix

package cliagent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Fake agent scripts. Each records its own PID (the process group ID when the
// run is isolated) and starts a background grandchild that writes a marker
// file after a delay, simulating an agent tool that outlives its parent.
const (
	grandchildDelay = 400 * time.Millisecond
	// holdsOutput keeps the grandchild attached to the agent's stdout/stderr.
	scriptHoldsOutput = `echo $$ > %[1]s/pid; (sleep 0.4; echo late > %[1]s/marker) & %[2]s`
	// detached redirects the grandchild's output so pipes close with the agent.
	scriptDetached = `(sleep 0.4; echo late > %[1]s/marker) </dev/null >/dev/null 2>&1 & echo $$ > %[1]s/pid; %[2]s`
)

type reapAgentFactory func() Agent

func reapAgents() map[string]reapAgentFactory {
	return map[string]reapAgentFactory{
		"base": func() Agent {
			return &BaseAgent{AgentName: "fake", Cmd: "sh", AgentCaps: Caps{
				PromptDelivery: PromptDelivery{Method: PromptMethodArg, Flag: "-c", InteractiveFlag: "-c"},
			}}
		},
		"custom": func() Agent {
			agent, err := NewCustomAgentFromConfig(CustomAgentConfig{Command: "sh", Args: []string{"-c", "{{PROMPT}}"}})
			if err != nil {
				panic(err)
			}
			return agent
		},
	}
}

func TestExecute_ReapsAgentProcessGroup(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		script  string
		tail    string
		timeout time.Duration
		cancel  bool
		wantErr error
	}{
		"context cancel with grandchild holding output": {script: scriptHoldsOutput, tail: "sleep 30", cancel: true, wantErr: context.Canceled},
		"timeout with grandchild holding output":        {script: scriptHoldsOutput, tail: "sleep 30", timeout: 300 * time.Millisecond, wantErr: context.DeadlineExceeded},
		"context cancel with detached grandchild":       {script: scriptDetached, tail: "sleep 30", cancel: true, wantErr: context.Canceled},
		"normal exit with grandchild holding output":    {script: scriptHoldsOutput, tail: "exit 0"},
		"normal exit with detached grandchild":          {script: scriptDetached, tail: "exit 0"},
	}

	for agentName, newAgent := range reapAgents() {
		for name, tt := range tests {
			t.Run(agentName+"/"+name, func(t *testing.T) {
				t.Parallel()
				dir := t.TempDir()
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if tt.cancel {
					go cancelWhenStarted(ctx, cancel, dir)
				}

				start := time.Now()
				_, err := newAgent().Execute(ctx, fmt.Sprintf(tt.script, dir, tt.tail), ExecOptions{Timeout: tt.timeout})
				elapsed := time.Since(start)

				assertExecuteErr(t, err, tt.wantErr)
				if elapsed >= grandchildDelay && tt.wantErr == nil {
					t.Errorf("Execute() waited %v for a background descendant; want prompt return", elapsed)
				}
				assertGroupGone(t, readPGID(t, dir))
				assertNoMarker(t, dir, "at return")
				time.Sleep(grandchildDelay + 300*time.Millisecond)
				assertNoMarker(t, dir, "after return")
			})
		}
	}
}

func TestExecute_ProcessGroupPlacement(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		interactive bool
		wantOwn     bool
	}{
		"non-interactive runs in its own process group": {wantOwn: true},
		"interactive keeps the caller's process group":  {interactive: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			agent := reapAgents()["base"]()
			opts := ExecOptions{Interactive: tt.interactive}
			result, err := agent.Execute(context.Background(), "echo $$; ps -o pgid= -p $$", opts)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			fields := strings.Fields(result.Stdout)
			if len(fields) != 2 {
				t.Fatalf("unexpected output %q", result.Stdout)
			}
			ownGroup := fields[0] == fields[1]
			if ownGroup != tt.wantOwn {
				t.Errorf("own process group = %v, want %v (pid %s, pgid %s)", ownGroup, tt.wantOwn, fields[0], fields[1])
			}
			if !tt.wantOwn && fields[1] != strconv.Itoa(syscall.Getpgrp()) {
				t.Errorf("pgid = %s, want caller pgid %d", fields[1], syscall.Getpgrp())
			}
		})
	}
}

func cancelWhenStarted(ctx context.Context, cancel context.CancelFunc, dir string) {
	for ctx.Err() == nil {
		if _, err := os.Stat(filepath.Join(dir, "pid")); err == nil {
			time.Sleep(50 * time.Millisecond) // let the grandchild start
			cancel()
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func assertExecuteErr(t *testing.T, err, want error) {
	t.Helper()
	if want == nil {
		if err != nil {
			t.Fatalf("Execute() error = %v, want nil", err)
		}
		return
	}
	if !errors.Is(err, want) {
		t.Fatalf("Execute() error = %v, want %v", err, want)
	}
}

func readPGID(t *testing.T, dir string) int {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "pid"))
	if err != nil {
		t.Fatalf("reading fake agent pid: %v", err)
	}
	pgid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("parsing fake agent pid %q: %v", data, err)
	}
	return pgid
}

func assertGroupGone(t *testing.T, pgid int) {
	t.Helper()
	if err := syscall.Kill(-pgid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Errorf("process group %d still has members after Execute returned (kill err = %v)", pgid, err)
	}
}

func assertNoMarker(t *testing.T, dir, when string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, "marker")); err == nil {
		t.Errorf("background descendant wrote marker %s", when)
	}
}
