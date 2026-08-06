package cliagent

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

	jcode "github.com/1jehuang/jcode-go"
)

// JcodeOptions selects the native jcode runtime mode.
type JcodeOptions struct {
	Mode           string
	SocketPath     string
	Binary         string
	Home           string
	InheritLogins  bool
	StartupTimeout time.Duration
	CleanupTimeout time.Duration
}

type jcodeEventStream interface {
	Next(context.Context) (jcode.TypedEvent, error)
	Close()
}
type jcodeSession interface {
	Send(context.Context, string) error
	Events(context.Context) jcodeEventStream
}
type jcodeClient interface {
	CreateSession(context.Context, string) (jcodeSession, error)
}
type jcodeFactory interface {
	Open(context.Context, JcodeOptions, ExecOptions) (jcodeClient, func() error, error)
}

type sdkJcodeFactory struct{}

func (sdkJcodeFactory) Open(ctx context.Context, options JcodeOptions, execOptions ExecOptions) (jcodeClient, func() error, error) {
	if options.Mode == "private" {
		inherit := options.InheritLogins
		client, err := jcode.Launch(ctx, jcode.LaunchOptions{
			Binary: options.Binary, JcodeHome: options.Home, WorkingDir: execOptions.WorkDir,
			InheritLogins: &inherit, StartupTimeout: options.StartupTimeout,
			CleanupTimeout: options.CleanupTimeout,
			ClientOptions:  jcode.Options{ClientName: "autospec/jcode"},
		})
		if err != nil {
			return nil, nil, fmt.Errorf("launching private jcode runtime: %w", err)
		}
		return sdkJcodeClient{client: client}, client.Close, nil
	}
	client, err := jcode.Connect(ctx, jcode.ConnectOptions{SocketPath: options.SocketPath, ClientOptions: jcode.Options{ClientName: "autospec/jcode"}})
	if err != nil {
		return nil, nil, fmt.Errorf("connecting to jcode runtime: %w", err)
	}
	return sdkJcodeClient{client: client}, client.Close, nil
}

type sdkJcodeClient struct{ client *jcode.Client }

func (c sdkJcodeClient) CreateSession(ctx context.Context, workDir string) (jcodeSession, error) {
	session, err := c.client.CreateSession(ctx, jcode.CreateSessionOptions{WorkingDir: workDir})
	if err != nil {
		return nil, fmt.Errorf("creating jcode session: %w", err)
	}
	return sdkJcodeSession{session: session}, nil
}

type sdkJcodeSession struct{ session jcode.Session }

func (s sdkJcodeSession) Send(ctx context.Context, prompt string) error {
	return s.session.Send(ctx, prompt, jcode.SendOptions{})
}
func (s sdkJcodeSession) Events(ctx context.Context) jcodeEventStream {
	return sdkJcodeStream{stream: s.session.Events(ctx)}
}

type sdkJcodeStream struct{ stream *jcode.TypedEventStream }

func (s sdkJcodeStream) Next(ctx context.Context) (jcode.TypedEvent, error) {
	return s.stream.Next(ctx)
}
func (s sdkJcodeStream) Close() { s.stream.Close() }

// Jcode implements the native jcode Go SDK agent.
type Jcode struct {
	options JcodeOptions
	factory jcodeFactory
}

// JcodeAgent is retained as an explicit name for callers that distinguish
// native agents from command-backed agents.
type JcodeAgent = Jcode

// NewJcode creates a jcode agent using connect mode and the SDK defaults.
func NewJcode() *Jcode { return NewJcodeWithOptions(JcodeOptions{Mode: "connect"}) }
func NewJcodeWithOptions(options JcodeOptions) *Jcode {
	if options.Mode == "" {
		options.Mode = "connect"
	}
	return &Jcode{options: options, factory: sdkJcodeFactory{}}
}
func (j *Jcode) Name() string             { return "jcode" }
func (j *Jcode) Capabilities() Caps       { return Caps{Automatable: true} }
func (j *Jcode) Version() (string, error) { return "native-sdk", nil }
func (j *Jcode) Validate() error {
	if j.options.Mode == "private" && j.options.Binary != "" {
		if _, err := exec.LookPath(j.options.Binary); err != nil {
			return fmt.Errorf("jcode binary %q not found: %w", j.options.Binary, err)
		}
	}
	return nil
}
func (*Jcode) BuildCommand(string, ExecOptions) (*exec.Cmd, error) {
	return nil, fmt.Errorf("jcode uses the native SDK; command execution is unsupported")
}

func (j *Jcode) Execute(parent context.Context, prompt string, options ExecOptions) (*Result, error) {
	started := time.Now()
	if j.factory == nil {
		j.factory = sdkJcodeFactory{}
	}
	ctx, cancel := executionContext(parent, options.Timeout)
	defer cancel()
	client, cleanup, err := j.factory.Open(ctx, j.options, options)
	if err != nil {
		return nil, fmt.Errorf("opening jcode runtime: %w", err)
	}
	defer func() { _ = cleanup() }()
	session, err := client.CreateSession(ctx, options.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("creating jcode session: %w", err)
	}
	stream := session.Events(ctx)
	if stream == nil {
		return nil, fmt.Errorf("creating jcode event stream: nil stream")
	}
	defer stream.Close()
	if err := session.Send(ctx, prompt); err != nil {
		return nil, fmt.Errorf("sending prompt to jcode: %w", err)
	}
	output, err := streamOutput(ctx, stream, outputWriter(options.Stdout))
	if err != nil {
		return nil, fmt.Errorf("streaming jcode output: %w", err)
	}
	result := &Result{ExitCode: 0, Duration: time.Since(started)}
	if options.Stdout == nil {
		result.Stdout = output
	}
	return result, nil
}
func executionContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(parent, timeout)
	}
	return context.WithCancel(parent)
}
func outputWriter(writer io.Writer) io.Writer {
	if writer != nil {
		return writer
	}
	return io.Discard
}
func streamOutput(ctx context.Context, stream jcodeEventStream, writer io.Writer) (string, error) {
	var output []byte
	for {
		event, err := stream.Next(ctx)
		if err != nil {
			return string(output), fmt.Errorf("reading event: %w", err)
		}
		switch value := event.(type) {
		case *jcode.TextDelta:
			if _, err := io.WriteString(writer, value.Text); err != nil {
				return string(output), fmt.Errorf("writing text delta: %w", err)
			}
			output = append(output, value.Text...)
		case *jcode.TurnDone:
			return string(output), nil
		case *jcode.PermissionRequest, jcode.UnknownEvent:
			continue
		}
	}
}
