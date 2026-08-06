package cliagent

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"

	"github.com/1jehuang/jcode-go"
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

type mockJcodeSession struct {
	events jcodeEventStream
	err    error
	order  *[]string
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
	if got := strings.Join(order, ","); got != "subscribe,send" {
		t.Fatalf("operation order = %q, want subscribe,send", got)
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
