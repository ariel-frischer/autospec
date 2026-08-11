package cliagent

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ariel-frischer/jcode-go"
	"github.com/stretchr/testify/require"
)

func jcodeFixturePath(t *testing.T, name string) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	require.True(t, ok, "locate cliagent test source")
	return filepath.Join(filepath.Dir(source), "testdata", name)
}

func installJcodeFixture(t *testing.T) (string, string) {
	t.Helper()
	binDir := t.TempDir()
	fixture := jcodeFixturePath(t, "fake-jcode.sh")
	binary := filepath.Join(binDir, "jcode")
	data, err := os.ReadFile(fixture)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(binary, data, 0o755))
	return binary, filepath.Join(t.TempDir(), "jcode-invocation.log")
}

func jcodeFixtureOptions(logPath string) ExecOptions {
	return ExecOptions{
		Env: map[string]string{
			"JCODE_FIXTURE_LOG": logPath,
		},
	}
}

type mockJcodeFactory struct {
	client  jcodeClient
	err     error
	cleanup func() error
}

func (m mockJcodeFactory) Open(context.Context, JcodeOptions, ExecOptions) (jcodeClient, func() error, error) {
	cleanup := m.cleanup
	if cleanup == nil {
		cleanup = func() error { return nil }
	}
	return m.client, cleanup, m.err
}

type mockJcodeClient struct {
	session jcodeSession
	err     error
}

func (m mockJcodeClient) CreateSession(context.Context, string) (jcodeSession, error) {
	return m.session, m.err
}

func (mockJcodeClient) Reconnect(context.Context) error { return nil }

type mockJcodeSession struct {
	events   jcodeEventStream
	err      error
	order    *[]string
	settings *[]JcodeSessionSettings
}

func (m mockJcodeSession) Configure(_ context.Context, settings JcodeSessionSettings) error {
	*m.order = append(*m.order, "configure")
	if m.settings != nil {
		*m.settings = append(*m.settings, settings)
	}
	return nil
}

func (m mockJcodeSession) Send(context.Context, string) error {
	*m.order = append(*m.order, "send")
	return m.err
}

func (m mockJcodeSession) Events(context.Context) jcodeEventStream {
	*m.order = append(*m.order, "subscribe")
	return m.events
}

type mockJcodeEventStream struct {
	events []jcode.TypedEvent
	index  int
}

func (m *mockJcodeEventStream) Next(context.Context) (jcode.TypedEvent, error) {
	if m.index >= len(m.events) {
		return nil, io.EOF
	}
	event := m.events[m.index]
	m.index++
	return event, nil
}

func (*mockJcodeEventStream) Close() {}

func TestJcodeAgent_ExecuteStreamsTypedEvents(t *testing.T) {
	t.Parallel()

	order := []string{}
	stream := &mockJcodeEventStream{events: []jcode.TypedEvent{
		&jcode.TextDelta{Text: "answer"},
		&jcode.ReasoningDelta{Text: "reason"},
		&jcode.TurnDone{},
	}}
	agent := &JcodeAgent{
		factory: mockJcodeFactory{client: mockJcodeClient{session: mockJcodeSession{events: stream, order: &order}}},
	}
	var stdout strings.Builder

	result, err := agent.Execute(context.Background(), "prompt", ExecOptions{WorkDir: "/repo", Stdout: &stdout})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}
	if got := stdout.String(); got != "answer" {
		t.Fatalf("stdout = %q, want %q", got, "answer")
	}
	if got := strings.Join(order, ","); got != "configure,send,subscribe" {
		t.Fatalf("operation order = %q, want configure,send,subscribe", got)
	}
}

func TestNewJcodeWithOptionsPreservesLifecyclePolicy(t *testing.T) {
	t.Parallel()

	agent := NewJcodeWithOptions(JcodeOptions{
		Mode: "auto",
		Lifecycle: JcodeLifecyclePolicy{
			Mode:              JcodeLifecycleModeAuto,
			StartupCommand:    "jcode serve",
			ReconnectAttempts: 2,
			RestartAttempts:   1,
			RetryDelay:        250 * time.Millisecond,
		},
	})
	if agent.options.Lifecycle.Mode != JcodeLifecycleModeAuto {
		t.Fatalf("lifecycle mode = %q, want auto", agent.options.Lifecycle.Mode)
	}
	if agent.options.Lifecycle.StartupCommand != "jcode serve" {
		t.Fatalf("startup command = %q, want configured command", agent.options.Lifecycle.StartupCommand)
	}
}

func TestJcodeValidateRedactsPrivateBinaryPath(t *testing.T) {
	t.Parallel()

	agent := NewJcodeWithOptions(JcodeOptions{
		Mode:   "private",
		Binary: "/tmp/secret-jcode-binary",
	})
	err := agent.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want missing binary error")
	}
	if strings.Contains(err.Error(), "secret-jcode-binary") {
		t.Fatalf("Validate() exposed binary path: %v", err)
	}
}

func TestJcodeAgent_ExecuteConfiguresSessionBeforePrompt(t *testing.T) {
	t.Parallel()

	order := []string{}
	settings := []JcodeSessionSettings{}
	stream := &mockJcodeEventStream{events: []jcode.TypedEvent{&jcode.TurnDone{}}}
	agent := &JcodeAgent{
		factory: mockJcodeFactory{client: mockJcodeClient{session: mockJcodeSession{
			events: stream, order: &order, settings: &settings,
		}}},
	}

	_, err := agent.Execute(context.Background(), "prompt", ExecOptions{
		Model:           "openai/gpt-5.6-luna",
		ReasoningEffort: "max",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	want := []JcodeSessionSettings{{Model: "openai/gpt-5.6-luna", ReasoningEffort: "max"}}
	if !reflect.DeepEqual(settings, want) {
		t.Fatalf("settings = %#v, want %#v", settings, want)
	}
	if got := strings.Join(order, ","); got != "configure,send,subscribe" {
		t.Fatalf("operation order = %q, want configure,send,subscribe", got)
	}
}

func TestJcodeAgent_ExecuteWrapsSessionFailures(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("permission denied")
	agent := &JcodeAgent{
		factory: mockJcodeFactory{client: mockJcodeClient{session: mockJcodeSession{
			err: wantErr, order: new([]string), events: &mockJcodeEventStream{},
		}}},
	}

	_, err := agent.Execute(context.Background(), "prompt", ExecOptions{})

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want wrapped session error", err)
	}
	if !strings.Contains(err.Error(), "sending prompt") {
		t.Fatalf("Execute() error = %q, want operation context", err)
	}
}

func TestJcodeAgent_BuildCommandIsUnsupported(t *testing.T) {
	t.Parallel()

	var agent Agent = NewJcode()
	_, err := agent.BuildCommand("prompt", ExecOptions{})

	if err == nil {
		t.Fatal("BuildCommand() error = nil, want unsupported-native error")
	}
	if !strings.Contains(err.Error(), "native") || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("BuildCommand() error = %q, want explicit unsupported-native error", err)
	}
}

func TestJcodeAgent_ImplementsAgentWithoutExecCommand(t *testing.T) {
	t.Parallel()

	var agent Agent = NewJcode()
	if agent.Name() != "jcode" {
		t.Fatalf("Name() = %q, want jcode", agent.Name())
	}
	if _, ok := agent.(*BaseAgent); ok {
		t.Fatal("jcode must not use BaseAgent exec implementation")
	}
	if _, ok := interface{}(agent).(interface {
		BuildCommand(string, ExecOptions) (*exec.Cmd, error)
	}); !ok {
		t.Fatal("jcode must satisfy the Agent command boundary")
	}
}

func TestJcodeExec_BuildCommandContract(t *testing.T) {
	t.Parallel()

	binary, _ := installJcodeFixture(t)
	agent := NewJcodeExec(binary)
	workDir := t.TempDir()

	cmd, err := agent.BuildCommand("fixture prompt", ExecOptions{
		WorkDir:         workDir,
		Model:           "openai/gpt-5.6-luna",
		ReasoningEffort: "max",
		Env:             map[string]string{"JCODE_FIXTURE_ENV": "isolated"},
	})
	require.NoError(t, err)
	require.Equal(t, binary, cmd.Path)
	require.Equal(t, workDir, cmd.Dir)
	require.Equal(t, []string{
		binary,
		"run", "--quiet",
		"--model", "openai/gpt-5.6-luna",
		"--reasoning-effort", "max",
		"fixture prompt",
	}, cmd.Args)
	require.Contains(t, cmd.Env, "JCODE_FIXTURE_ENV=isolated")
}

func TestJcodeExec_BuildCommandReasoningEffort(t *testing.T) {
	t.Parallel()

	binary, _ := installJcodeFixture(t)
	tests := map[string]struct {
		opts ExecOptions
		want []string
	}{
		"direct option uses current jcode flag": {
			opts: ExecOptions{ReasoningEffort: "medium"},
			want: []string{binary, "run", "--quiet", "--reasoning-effort", "medium", "prompt"},
		},
		"workflow codex arguments are translated": {
			opts: ExecOptions{ExtraArgs: []string{"-c", "model_reasoning_effort=medium", "--trace"}},
			want: []string{binary, "run", "--quiet", "--reasoning-effort", "medium", "prompt", "--trace"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd, err := NewJcodeExec(binary).BuildCommand("prompt", tt.opts)
			require.NoError(t, err)
			require.Equal(t, tt.want, cmd.Args)
		})
	}
}

func TestJcodeExec_ExecuteTransportsPromptAndOptions(t *testing.T) {
	t.Parallel()

	binary, logPath := installJcodeFixture(t)
	workDir := t.TempDir()
	opts := jcodeFixtureOptions(logPath)
	opts.WorkDir = workDir
	opts.Model = "openai/gpt-5.6-luna"
	opts.ReasoningEffort = "high"
	opts.Env["JCODE_FIXTURE_ENV"] = "isolated"
	opts.ExtraArgs = []string{"--trace"}
	result, err := NewJcodeExec(binary).Execute(context.Background(), "fixture prompt", opts)
	require.NoError(t, err)
	require.Equal(t, 0, result.ExitCode)

	invocation, err := os.ReadFile(logPath)
	require.NoError(t, err)
	got := string(invocation)
	require.Contains(t, got, "cwd: "+workDir)
	require.Contains(t, got, "env_fixture: isolated")
	for _, want := range []string{
		"  - run\n", "  - --quiet\n", "  - --model\n", "  - openai/gpt-5.6-luna\n",
		"  - --reasoning-effort\n", "  - high\n", "  - fixture prompt\n", "  - --trace\n",
	} {
		require.Contains(t, got, want)
	}
}

func TestJcodeExec_ExecuteReportsCommandFailures(t *testing.T) {
	t.Parallel()

	binary, _ := installJcodeFixture(t)
	_, err := NewJcodeExec(binary).Execute(context.Background(), "prompt", ExecOptions{
		Env: map[string]string{"JCODE_FIXTURE_EXIT": "23"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "jcode exec")
	require.Contains(t, err.Error(), "exit status 23")
}

func TestJcodeExec_BuildCommandReportsMissingExecutable(t *testing.T) {
	t.Parallel()

	_, err := NewJcodeExec("definitely-missing-jcode").BuildCommand("prompt", ExecOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "jcode executable")
	require.Contains(t, err.Error(), "definitely-missing-jcode")
}

func TestJcodeSDK_ExplicitLifecycleSelectionAndSafety(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		options  JcodeOptions
		wantMode string
	}{
		"connect mode": {options: JcodeOptions{Mode: "connect"}, wantMode: "connect"},
		"private mode": {options: JcodeOptions{Mode: "private", Binary: "/custom/jcode"}, wantMode: "private"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			agent := NewJcodeWithOptions(tt.options)
			require.Equal(t, tt.wantMode, agent.options.Mode)
			require.IsType(t, sdkJcodeFactory{}, agent.factory)
		})
	}
}

func TestJcodeSDK_DefaultConstructorDoesNotOverrideExplicitExecSelection(t *testing.T) {
	t.Parallel()

	defaultAgent := NewJcode()
	execAgent := NewJcodeExec("")

	require.Equal(t, "connect", defaultAgent.options.Mode)
	require.Equal(t, defaultJcodeExecBinary, execAgent.binary)
	require.NotSame(t, defaultAgent, execAgent)
}

func TestJcodeSDK_ExecuteCleansUpAndWrapsLifecycleErrors(t *testing.T) {
	t.Parallel()

	cleanupCalled := false
	wantErr := errors.New("session unavailable")
	agent := NewJcodeWithOptions(JcodeOptions{Mode: "connect"})
	agent.factory = mockJcodeFactory{
		client: mockJcodeClient{err: wantErr},
		cleanup: func() error {
			cleanupCalled = true
			return nil
		},
	}

	_, err := agent.Execute(context.Background(), "prompt", ExecOptions{})

	require.ErrorIs(t, err, wantErr)
	require.Contains(t, err.Error(), "creating jcode session")
	require.True(t, cleanupCalled)
}

func TestJcodeSDK_ExecuteHonorsConfiguredTimeout(t *testing.T) {
	t.Parallel()

	agent := NewJcodeWithOptions(JcodeOptions{Mode: "connect"})
	agent.factory = mockJcodeFactory{client: mockJcodeClient{
		session: mockJcodeSession{events: &blockingJcodeEventStream{}, order: new([]string)},
	}}

	started := time.Now()
	_, err := agent.Execute(context.Background(), "prompt", ExecOptions{Timeout: 20 * time.Millisecond})

	require.Error(t, err)
	require.Contains(t, err.Error(), "streaming jcode output")
	require.Less(t, time.Since(started), time.Second)
}

type blockingJcodeEventStream struct{}

func (*blockingJcodeEventStream) Next(ctx context.Context) (jcode.TypedEvent, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func (*blockingJcodeEventStream) Close() {}
