package cliagent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	jcode "github.com/ariel-frischer/jcode-go"
	"github.com/ariel-frischer/jcode-go/protocol"
	"github.com/ariel-frischer/jcode-go/transport"
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
	SessionProfile string
	MaxTurns       int
	TokenBudget    int
	Deadline       string
	Lifecycle      JcodeLifecyclePolicy
}

type jcodeTurn interface {
	Next(context.Context) (jcode.TypedEvent, error)
	Cancel(context.Context) error
	Wait(context.Context) (jcode.TurnResult, error)
}
type jcodeSession interface {
	Configure(context.Context, JcodeSessionSettings) error
	StartTurn(context.Context, string, jcode.SendOptions) (jcodeTurn, error)
}

// JcodeSessionSettings contains non-secret per-stage settings for jcode.
// Provider credentials remain owned by the jcode runtime.
type JcodeSessionSettings struct {
	Model           string
	ReasoningEffort string
}
type jcodeClient interface {
	CreateSession(context.Context, jcode.CreateSessionOptions) (jcodeSession, error)
	Reconnect(context.Context) error
}
type jcodeFactory interface {
	Open(context.Context, JcodeOptions, ExecOptions) (jcodeClient, func() error, error)
}

type sdkJcodeFactory struct{}

func (sdkJcodeFactory) Open(ctx context.Context, options JcodeOptions, execOptions ExecOptions) (jcodeClient, func() error, error) {
	policy := options.Lifecycle
	if policy.Mode == "" {
		policy.Mode = JcodeLifecycleMode(options.Mode)
	}
	runtime := sdkJcodeRuntime{options: options, execOptions: execOptions}
	handle, _, err := NewJcodeLifecycleController(policy, runtime).Select(ctx, JcodeRuntimeRequest{Options: options, WorkDir: execOptions.WorkDir})
	if err != nil {
		return nil, nil, fmt.Errorf("opening jcode runtime: %w", err)
	}
	if handle.client == nil || handle.cleanup == nil {
		return nil, nil, fmt.Errorf("opening jcode runtime: lifecycle selection returned no client")
	}
	return handle.client, handle.cleanup, nil
}

type sdkJcodeRuntime struct {
	options     JcodeOptions
	execOptions ExecOptions
}

func (r sdkJcodeRuntime) Connect(ctx context.Context, _ JcodeRuntimeRequest) (JcodeRuntimeHandle, error) {
	socketPath := r.options.SocketPath
	clientOptions := jcode.Options{ClientName: "autospec/jcode"}
	if r.options.Lifecycle.ReconnectAttempts > 0 {
		clientOptions.Reconnect = jcode.ReconnectPolicy{
			Factory:     transport.UnixSocket(resolveJcodeSocket(socketPath)),
			MaxAttempts: r.options.Lifecycle.ReconnectAttempts,
			Backoff:     r.options.Lifecycle.RetryDelay,
			MaxBackoff:  r.options.Lifecycle.RetryDelay,
			Resume:      true,
		}
	}
	client, err := jcode.Connect(ctx, jcode.ConnectOptions{SocketPath: socketPath, ClientOptions: clientOptions})
	if err != nil {
		return JcodeRuntimeHandle{}, err
	}
	return JcodeRuntimeHandle{ID: "shared-runtime", Ownership: JcodeRuntimeOwnershipShared, State: JcodeRuntimeStateReady, client: sdkJcodeClient{client: client}, cleanup: client.Close}, nil
}

func (r sdkJcodeRuntime) Launch(ctx context.Context, _ JcodeRuntimeRequest) (JcodeRuntimeHandle, error) {
	inherit := r.options.InheritLogins
	client, err := jcode.Launch(ctx, jcode.LaunchOptions{
		Binary: r.options.Binary, JcodeHome: r.options.Home, WorkingDir: r.execOptions.WorkDir,
		InheritLogins: &inherit, StartupTimeout: r.options.StartupTimeout,
		CleanupTimeout: r.options.CleanupTimeout,
		ClientOptions:  jcode.Options{ClientName: "autospec/jcode"},
	})
	if err != nil {
		return JcodeRuntimeHandle{}, err
	}
	owner, ok := client.DetachInstance()
	if !ok {
		_ = client.Close()
		return JcodeRuntimeHandle{}, fmt.Errorf("detaching owned jcode runtime: launch returned no instance")
	}
	cleanup := func() error {
		if err := client.Close(); err != nil && !errors.Is(err, jcode.ErrClosed) {
			return fmt.Errorf("closing jcode client: %w", err)
		}
		if err := owner.Shutdown(); err != nil {
			return fmt.Errorf("shutting down owned jcode runtime: %w", err)
		}
		return nil
	}
	return JcodeRuntimeHandle{ID: "private-runtime", Ownership: JcodeRuntimeOwnershipPrivate, State: JcodeRuntimeStateReady, client: sdkJcodeClient{client: client}, cleanup: cleanup}, nil
}

func (sdkJcodeRuntime) Reconnect(ctx context.Context, handle JcodeRuntimeHandle) (JcodeRuntimeHandle, error) {
	client, ok := handle.client.(interface{ Reconnect(context.Context) error })
	if !ok {
		return handle, fmt.Errorf("jcode client does not support reconnect")
	}
	if err := client.Reconnect(ctx); err != nil {
		return handle, fmt.Errorf("reconnecting jcode runtime: %w", err)
	}
	handle.State = JcodeRuntimeStateReady
	return handle, nil
}

func (r sdkJcodeRuntime) Restart(ctx context.Context, handle JcodeRuntimeHandle) (JcodeRuntimeHandle, error) {
	if !handle.CanRestart() {
		return handle, fmt.Errorf("jcode runtime is not owned by this run")
	}
	if handle.cleanup != nil {
		if err := handle.cleanup(); err != nil {
			return handle, fmt.Errorf("stopping owned jcode runtime before restart: %w", err)
		}
	}
	return r.Launch(ctx, JcodeRuntimeRequest{})
}

func (sdkJcodeRuntime) Cleanup(_ context.Context, handle JcodeRuntimeHandle) error {
	if !handle.CanCleanup() || handle.cleanup == nil {
		return fmt.Errorf("jcode runtime is not owned by this run")
	}
	return handle.cleanup()
}

type sdkJcodeClient struct{ client *jcode.Client }

func (c sdkJcodeClient) Reconnect(ctx context.Context) error {
	return c.client.Reconnect(ctx)
}

func resolveJcodeSocket(socketPath string) string {
	if socketPath != "" {
		return socketPath
	}
	if value := os.Getenv("JCODE_API_SOCKET"); value != "" {
		return value
	}
	if value := os.Getenv("JCODE_RUNTIME_DIR"); value != "" {
		return filepath.Join(value, "jcode-api.sock")
	}
	if value := os.Getenv("XDG_RUNTIME_DIR"); value != "" {
		return filepath.Join(value, "jcode-api.sock")
	}
	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, ".jcode", "run", "jcode-api.sock")
	}
	return filepath.Join(os.TempDir(), "jcode-api.sock")
}

func (c sdkJcodeClient) CreateSession(ctx context.Context, options jcode.CreateSessionOptions) (jcodeSession, error) {
	session, err := c.client.CreateSession(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("creating jcode session: %w", err)
	}
	return sdkJcodeSession{client: c.client, session: session}, nil
}

type sdkJcodeSession struct {
	client  *jcode.Client
	session jcode.Session
}

func (s sdkJcodeSession) Configure(ctx context.Context, settings JcodeSessionSettings) error {
	if settings.Model != "" {
		if err := s.set(ctx, "set_model", struct {
			SessionID string `json:"session_id"`
			Model     string `json:"model"`
		}{s.session.ID, settings.Model}); err != nil {
			return fmt.Errorf("setting jcode model: %w", err)
		}
	}
	if settings.ReasoningEffort != "" {
		if err := s.set(ctx, "set_reasoning_effort", struct {
			SessionID string `json:"session_id"`
			Effort    string `json:"effort"`
		}{s.session.ID, settings.ReasoningEffort}); err != nil {
			return fmt.Errorf("setting jcode reasoning effort: %w", err)
		}
	}
	return nil
}

func (s sdkJcodeSession) set(ctx context.Context, request string, fields any) error {
	raw, err := protocol.NewRawRequest(request, fields)
	if err != nil {
		return fmt.Errorf("building %s request: %w", request, err)
	}
	frame, err := s.client.Request(ctx, raw)
	if err != nil {
		return fmt.Errorf("sending %s request: %w", request, err)
	}
	if failure, ok := frame.Event.(protocol.Error); ok {
		return fmt.Errorf("%s: %s", failure.Code, failure.Message)
	}
	return nil
}

func (s sdkJcodeSession) StartTurn(ctx context.Context, prompt string, options jcode.SendOptions) (jcodeTurn, error) {
	turn, err := s.session.StartTurn(ctx, prompt, options)
	if err != nil {
		return nil, fmt.Errorf("starting jcode turn: %w", err)
	}
	return sdkJcodeTurn{turn: turn}, nil
}

type sdkJcodeTurn struct{ turn *jcode.Turn }

func (t sdkJcodeTurn) Next(ctx context.Context) (jcode.TypedEvent, error) {
	return t.turn.Next(ctx)
}
func (t sdkJcodeTurn) Cancel(ctx context.Context) error { return t.turn.Cancel(ctx) }
func (t sdkJcodeTurn) Wait(ctx context.Context) (jcode.TurnResult, error) {
	return t.turn.Wait(ctx)
}

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
	if options.Lifecycle.Mode == "" {
		options.Lifecycle.Mode = JcodeLifecycleMode(options.Mode)
	}
	return &Jcode{options: options, factory: sdkJcodeFactory{}}
}
func (j *Jcode) Name() string             { return "jcode" }
func (j *Jcode) Capabilities() Caps       { return Caps{Automatable: true, Commandless: true} }
func (j *Jcode) Version() (string, error) { return "native-sdk", nil }
func (j *Jcode) Validate() error {
	mode := j.options.Lifecycle.ResolvedMode()
	if (mode == JcodeLifecycleModePrivate || mode == JcodeLifecycleModeAuto) && j.options.Binary != "" {
		if _, err := exec.LookPath(j.options.Binary); err != nil {
			return fmt.Errorf("jcode private runtime binary is unavailable")
		}
	}
	return nil
}
func (*Jcode) BuildCommand(string, ExecOptions) (*exec.Cmd, error) {
	return nil, fmt.Errorf("jcode uses the native SDK; command execution is unsupported")
}

func (j *Jcode) Execute(parent context.Context, prompt string, options ExecOptions) (result *Result, execErr error) {
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
	defer func() {
		if err := cleanup(); err != nil {
			execErr = errors.Join(execErr, fmt.Errorf("cleaning up jcode runtime: %w", err))
		}
	}()
	session, err := client.CreateSession(ctx, j.createSessionOptions(options.WorkDir))
	if err != nil {
		return nil, fmt.Errorf("creating jcode session: %w", err)
	}
	if err := session.Configure(ctx, JcodeSessionSettings{
		Model: options.Model, ReasoningEffort: options.ReasoningEffort,
	}); err != nil {
		return nil, fmt.Errorf("configuring jcode session: %w", err)
	}
	turnCtx, stopTurn := context.WithCancel(context.Background())
	defer stopTurn()
	turn, err := session.StartTurn(turnCtx, prompt, j.sendOptions())
	if err != nil {
		return nil, fmt.Errorf("starting jcode turn: %w", err)
	}
	output, err := streamTurn(ctx, turn, outputWriter(options.Stdout), j.options.CleanupTimeout, stopTurn)
	if err != nil {
		return nil, fmt.Errorf("streaming jcode output: %w", err)
	}
	result = &Result{ExitCode: 0, Duration: time.Since(started)}
	if options.Stdout == nil {
		result.Stdout = output
	}
	return result, nil
}

func (j *Jcode) createSessionOptions(workDir string) jcode.CreateSessionOptions {
	return jcode.CreateSessionOptions{
		WorkingDir: workDir,
		Profile:    j.options.SessionProfile,
	}
}

func (j *Jcode) sendOptions() jcode.SendOptions {
	return jcode.SendOptions{
		MaxTurns:    j.options.MaxTurns,
		TokenBudget: j.options.TokenBudget,
		Deadline:    j.options.Deadline,
	}
}

func streamTurn(ctx context.Context, turn jcodeTurn, writer io.Writer, cleanupTimeout time.Duration, stopTurn context.CancelFunc) (string, error) {
	output, streamErr := streamOutput(ctx, turn, writer)
	if streamErr != nil {
		streamErr = errors.Join(streamErr, cancelJcodeTurn(turn, cleanupTimeout))
		stopTurn()
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), boundedCleanupTimeout(cleanupTimeout))
	defer cancel()
	terminal, waitErr := turn.Wait(waitCtx)
	if waitErr != nil {
		return output, errors.Join(streamErr, fmt.Errorf("waiting for jcode turn: %w", waitErr))
	}
	if terminal.Err != nil {
		return output, errors.Join(streamErr, fmt.Errorf("jcode turn %s: %w", terminal.Kind, terminal.Err))
	}
	return output, streamErr
}

func cancelJcodeTurn(turn jcodeTurn, cleanupTimeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), boundedCleanupTimeout(cleanupTimeout))
	defer cancel()
	if err := turn.Cancel(ctx); err != nil {
		return fmt.Errorf("canceling jcode turn: %w", err)
	}
	return nil
}

func boundedCleanupTimeout(value time.Duration) time.Duration {
	if value > 0 {
		return value
	}
	return 5 * time.Second
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
func streamOutput(ctx context.Context, stream interface {
	Next(context.Context) (jcode.TypedEvent, error)
}, writer io.Writer) (string, error) {
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
