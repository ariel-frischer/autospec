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
	session       jcodeSession
	err           error
	createOptions *[]jcode.CreateSessionOptions
}

func (m mockJcodeClient) CreateSession(_ context.Context, options jcode.CreateSessionOptions) (jcodeSession, error) {
	if m.createOptions != nil {
		*m.createOptions = append(*m.createOptions, options)
	}
	return m.session, m.err
}

func (mockJcodeClient) Reconnect(context.Context) error { return nil }

type mockJcodeSession struct {
	events      jcodeTurn
	err         error
	order       *[]string
	settings    *[]JcodeSessionSettings
	sendOptions *[]jcode.SendOptions
}

func (m mockJcodeSession) Configure(_ context.Context, settings JcodeSessionSettings) error {
	*m.order = append(*m.order, "configure")
	if m.settings != nil {
		*m.settings = append(*m.settings, settings)
	}
	return nil
}

func (m mockJcodeSession) StartTurn(_ context.Context, _ string, options jcode.SendOptions) (jcodeTurn, error) {
	*m.order = append(*m.order, "start")
	if m.sendOptions != nil {
		*m.sendOptions = append(*m.sendOptions, options)
	}
	return m.events, m.err
}

type mockJcodeEventStream struct {
	events      []jcode.TypedEvent
	index       int
	terminal    jcode.TurnResult
	cancelCount *int
	cancelErr   error
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
func (m *mockJcodeEventStream) Cancel(context.Context) error {
	if m.cancelCount != nil {
		*m.cancelCount++
	}
	return m.cancelErr
}
func (m *mockJcodeEventStream) Wait(context.Context) (jcode.TurnResult, error) {
	if m.terminal.Kind != "" || m.terminal.Err != nil {
		return m.terminal, nil
	}
	return jcode.TurnResult{Kind: jcode.TurnResultCompleted}, nil
}

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
	if got := strings.Join(order, ","); got != "configure,start" {
		t.Fatalf("operation order = %q, want configure,start", got)
	}
}

func TestJcodeAgent_MapsTypedSessionControls(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		options    JcodeOptions
		wantCreate jcode.CreateSessionOptions
		wantSend   jcode.SendOptions
	}{
		"omitted": {wantCreate: jcode.CreateSessionOptions{WorkingDir: "/repo"}},
		"session profile": {
			options:    JcodeOptions{SessionProfile: "bounded"},
			wantCreate: jcode.CreateSessionOptions{WorkingDir: "/repo", Profile: "bounded"},
		},
		"maximum turns": {options: JcodeOptions{MaxTurns: 7}, wantCreate: jcode.CreateSessionOptions{WorkingDir: "/repo"}, wantSend: jcode.SendOptions{MaxTurns: 7}},
		"token budget":  {options: JcodeOptions{TokenBudget: 4096}, wantCreate: jcode.CreateSessionOptions{WorkingDir: "/repo"}, wantSend: jcode.SendOptions{TokenBudget: 4096}},
		"deadline":      {options: JcodeOptions{Deadline: "2026-08-23T08:00:00Z"}, wantCreate: jcode.CreateSessionOptions{WorkingDir: "/repo"}, wantSend: jcode.SendOptions{Deadline: "2026-08-23T08:00:00Z"}},
		"all controls": {
			options:    JcodeOptions{SessionProfile: "bounded", MaxTurns: 7, TokenBudget: 4096, Deadline: "2026-08-23T08:00:00Z"},
			wantCreate: jcode.CreateSessionOptions{WorkingDir: "/repo", Profile: "bounded"},
			wantSend:   jcode.SendOptions{MaxTurns: 7, TokenBudget: 4096, Deadline: "2026-08-23T08:00:00Z"},
		},
	}

	for name, tt := range tests {
		name, tt := name, tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assertJcodeTypedSessionControls(t, tt.options, tt.wantCreate, tt.wantSend)
		})
	}
}

func assertJcodeTypedSessionControls(t *testing.T, options JcodeOptions, wantCreate jcode.CreateSessionOptions, wantSend jcode.SendOptions) {
	t.Helper()
	createOptions := []jcode.CreateSessionOptions{}
	sendOptions := []jcode.SendOptions{}
	order := []string{}
	stream := &mockJcodeEventStream{events: []jcode.TypedEvent{&jcode.TurnDone{}}}
	session := mockJcodeSession{events: stream, order: &order, sendOptions: &sendOptions}
	agent := &Jcode{options: options, factory: mockJcodeFactory{client: mockJcodeClient{session: session, createOptions: &createOptions}}}

	_, err := agent.Execute(context.Background(), "prompt", ExecOptions{WorkDir: "/repo"})
	require.NoError(t, err)
	require.Equal(t, []jcode.CreateSessionOptions{wantCreate}, createOptions)
	require.Equal(t, []jcode.SendOptions{wantSend}, sendOptions)
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
	if got := strings.Join(order, ","); got != "configure,start" {
		t.Fatalf("operation order = %q, want configure,start", got)
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
	if !strings.Contains(err.Error(), "starting jcode turn") {
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
	agent := NewJcodeExecWithOptions(binary, JcodeExecOptions{
		Provider: "openai", ProviderProfile: "team", SocketPath: "/tmp/jcode.sock",
		Trace: true, ToolProfile: "minimal", Tools: "bash,read",
		DisabledTools: "write", DisableBaseTools: true,
		MCPTools: "deferred", MCPToolsTokenThreshold: 4096,
	})
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
		"--quiet", "--no-update", "--no-selfdev",
		"--provider", "openai", "--provider-profile", "team",
		"--socket", "/tmp/jcode.sock", "--trace",
		"--tool-profile", "minimal", "--tools", "bash,read",
		"--disabled-tools", "write", "--disable-base-tools",
		"--mcp-tools", "deferred", "--mcp-tools-token-threshold", "4096",
		"--model", "openai/gpt-5.6-luna",
		"run", "fixture prompt",
	}, cmd.Args)
	require.Contains(t, cmd.Env, "JCODE_FIXTURE_ENV=isolated")
}

func TestJcodeExec_BuildCommandOmitsForkOnlyOptions(t *testing.T) {
	t.Parallel()

	binary, _ := installJcodeFixture(t)
	tests := map[string]struct {
		opts ExecOptions
		want []string
	}{
		"reasoning effort is SDK only": {
			opts: ExecOptions{ReasoningEffort: "medium"},
			want: []string{binary, "--quiet", "--no-update", "--no-selfdev", "run", "prompt"},
		},
		"arbitrary extra arguments are omitted": {
			opts: ExecOptions{
				Model:           "openai:gpt-5.6-sol",
				ReasoningEffort: "medium",
				ExtraArgs: []string{
					"--model", "openai:gpt-5.6-sol",
					"-c", "model_reasoning_effort=medium",
					"--trace",
				},
			},
			want: []string{binary, "--quiet", "--no-update", "--no-selfdev", "--model", "openai:gpt-5.6-sol", "run", "prompt"},
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
	result, err := NewJcodeExecWithOptions(binary, JcodeExecOptions{Trace: true}).Execute(context.Background(), "fixture prompt", opts)
	require.NoError(t, err)
	require.Equal(t, 0, result.ExitCode)

	invocation, err := os.ReadFile(logPath)
	require.NoError(t, err)
	got := string(invocation)
	require.Contains(t, got, "cwd: "+workDir)
	require.Contains(t, got, "env_fixture: isolated")
	for _, want := range []string{
		"  - --quiet\n", "  - --no-update\n", "  - --no-selfdev\n", "  - --trace\n",
		"  - --model\n", "  - openai/gpt-5.6-luna\n", "  - run\n", "  - fixture prompt\n",
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

func TestJcodeSDK_ExecuteSurfacesCleanupFailure(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("cleanup failed")
	agent := NewJcodeWithOptions(JcodeOptions{Mode: "connect"})
	agent.factory = mockJcodeFactory{
		client: mockJcodeClient{session: mockJcodeSession{
			events: &mockJcodeEventStream{events: []jcode.TypedEvent{&jcode.TurnDone{}}},
			order:  new([]string),
		}},
		cleanup: func() error { return wantErr },
	}

	_, err := agent.Execute(context.Background(), "prompt", ExecOptions{})

	require.ErrorIs(t, err, wantErr)
	require.Contains(t, err.Error(), "cleaning up jcode runtime")
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

func TestJcodeSDK_StreamTurnPreservesTerminalFailure(t *testing.T) {
	t.Parallel()

	wantErr := jcode.EventError{Code: "provider_error", ProviderCode: "rate_limit"}
	turn := &mockJcodeEventStream{terminal: jcode.TurnResult{
		Kind: jcode.TurnResultProviderError, Err: wantErr,
	}}

	_, err := streamTurn(context.Background(), turn, io.Discard, time.Second, func() {})

	require.Error(t, err)
	var eventErr jcode.EventError
	require.ErrorAs(t, err, &eventErr)
	require.Equal(t, "rate_limit", eventErr.ProviderCode)
}

func TestJcodeSDK_StreamTurnCancelsOnceOnContextDeadline(t *testing.T) {
	t.Parallel()

	cancelCount := 0
	turn := &blockingOwnedJcodeTurn{cancelCount: &cancelCount}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := streamTurn(ctx, turn, io.Discard, time.Second, func() {})

	require.Error(t, err)
	require.Equal(t, 1, cancelCount)
}

func TestJcodeSDK_StreamTurnCancelsOnLocalOutputFailure(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		streamErr error
		cancelErr error
	}{
		"preserves stream and cancellation failures": {
			streamErr: errors.New("write failed"),
			cancelErr: errors.New("cancel failed"),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cancelCount := 0
			turn := &mockJcodeEventStream{
				events:      []jcode.TypedEvent{&jcode.TextDelta{Text: "answer"}},
				cancelCount: &cancelCount,
				cancelErr:   tt.cancelErr,
			}

			_, err := streamTurn(context.Background(), turn, errorJcodeWriter{err: tt.streamErr}, time.Second, func() {})

			require.ErrorIs(t, err, tt.streamErr)
			require.ErrorIs(t, err, tt.cancelErr)
			require.Equal(t, 1, cancelCount)
		})
	}
}

type errorJcodeWriter struct{ err error }

func (w errorJcodeWriter) Write([]byte) (int, error) { return 0, w.err }

type blockingJcodeEventStream struct{}

func (*blockingJcodeEventStream) Next(ctx context.Context) (jcode.TypedEvent, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

type blockingOwnedJcodeTurn struct{ cancelCount *int }

func (*blockingOwnedJcodeTurn) Next(ctx context.Context) (jcode.TypedEvent, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
func (t *blockingOwnedJcodeTurn) Cancel(context.Context) error {
	*t.cancelCount++
	return nil
}
func (*blockingOwnedJcodeTurn) Wait(context.Context) (jcode.TurnResult, error) {
	return jcode.TurnResult{Kind: jcode.TurnResultLifecycleDeadlineExceeded, Err: context.DeadlineExceeded}, nil
}

func (*blockingJcodeEventStream) Cancel(context.Context) error { return nil }
func (*blockingJcodeEventStream) Wait(context.Context) (jcode.TurnResult, error) {
	return jcode.TurnResult{Kind: jcode.TurnResultLifecycleDeadlineExceeded, Err: context.DeadlineExceeded}, nil
}
