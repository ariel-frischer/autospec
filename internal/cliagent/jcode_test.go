package cliagent

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ariel-frischer/jcode-go"
)

type mockJcodeFactory struct {
	client jcodeClient
	err    error
}

func (m mockJcodeFactory) Open(context.Context, JcodeOptions, ExecOptions) (jcodeClient, func() error, error) {
	return m.client, func() error { return nil }, m.err
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
