package cliagent

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// JcodeLifecycleMode selects the runtime recovery policy.
type JcodeLifecycleMode string

const (
	JcodeLifecycleModeConnect JcodeLifecycleMode = "connect"
	JcodeLifecycleModePrivate JcodeLifecycleMode = "private"
	JcodeLifecycleModeAuto    JcodeLifecycleMode = "auto"
)

const (
	maxJcodeRecoveryAttempts = 10
	maxJcodeRetryDelay       = 5 * time.Minute
)

// JcodeRuntimeState describes the observed lifecycle state of a runtime.
type JcodeRuntimeState string

const (
	JcodeRuntimeStateAbsent       JcodeRuntimeState = "absent"
	JcodeRuntimeStateConnecting   JcodeRuntimeState = "connecting"
	JcodeRuntimeStateReady        JcodeRuntimeState = "ready"
	JcodeRuntimeStateDisconnected JcodeRuntimeState = "disconnected"
	JcodeRuntimeStateFailed       JcodeRuntimeState = "failed"
	JcodeRuntimeStateCleaned      JcodeRuntimeState = "cleaned"
)

// JcodeRuntimeOwnership identifies who may manage a runtime.
type JcodeRuntimeOwnership string

const (
	JcodeRuntimeOwnershipShared    JcodeRuntimeOwnership = "shared"
	JcodeRuntimeOwnershipPrivate   JcodeRuntimeOwnership = "private"
	JcodeRuntimeOwnershipUnrelated JcodeRuntimeOwnership = "unrelated"
)

// JcodeLifecyclePolicy contains bounded runtime recovery settings.
type JcodeLifecyclePolicy struct {
	Mode              JcodeLifecycleMode
	StartupCommand    string
	ReconnectAttempts int
	RestartAttempts   int
	RetryDelay        time.Duration
}

// Validate checks policy values before a runtime is opened.
func (p JcodeLifecyclePolicy) Validate() error {
	mode := p.ResolvedMode()
	if mode != JcodeLifecycleModeConnect && mode != JcodeLifecycleModePrivate && mode != JcodeLifecycleModeAuto {
		return fmt.Errorf("invalid jcode lifecycle mode %q: must be connect, private, or auto", p.Mode)
	}
	if p.StartupCommand != "" && mode == JcodeLifecycleModeConnect {
		return fmt.Errorf("jcode startup command is only valid for private or auto mode")
	}
	if strings.TrimSpace(p.StartupCommand) == "" && p.StartupCommand != "" {
		return fmt.Errorf("jcode startup command must not be blank")
	}
	if p.ReconnectAttempts < 0 {
		return fmt.Errorf("jcode reconnect attempts must be non-negative")
	}
	if p.RestartAttempts < 0 {
		return fmt.Errorf("jcode restart attempts must be non-negative")
	}
	if p.RetryDelay < 0 {
		return fmt.Errorf("jcode retry delay must be non-negative")
	}
	if p.ReconnectAttempts > maxJcodeRecoveryAttempts {
		return fmt.Errorf("jcode reconnect attempts must be at most %d", maxJcodeRecoveryAttempts)
	}
	if p.RestartAttempts > maxJcodeRecoveryAttempts {
		return fmt.Errorf("jcode restart attempts must be at most %d", maxJcodeRecoveryAttempts)
	}
	if p.RetryDelay > maxJcodeRetryDelay {
		return fmt.Errorf("jcode retry delay must be at most %s", maxJcodeRetryDelay)
	}
	if strings.ContainsAny(p.StartupCommand, ";&|$`\n\r") {
		return fmt.Errorf("jcode startup command must not contain shell metacharacters")
	}
	return nil
}

// ResolvedMode returns the safe shared-runtime default when mode is unset.
func (p JcodeLifecyclePolicy) ResolvedMode() JcodeLifecycleMode {
	if p.Mode == "" {
		return JcodeLifecycleModeConnect
	}
	return p.Mode
}

// JcodeRuntimeHandle tracks runtime state and run ownership.
type JcodeRuntimeHandle struct {
	ID             string
	Ownership      JcodeRuntimeOwnership
	State          JcodeRuntimeState
	ReconnectCount int
	RestartCount   int
	ownerToken     string
	client         jcodeClient
	cleanup        func() error
}

func newJcodeRuntimeHandle(ownership JcodeRuntimeOwnership, state JcodeRuntimeState) JcodeRuntimeHandle {
	token := ""
	if ownership == JcodeRuntimeOwnershipPrivate {
		token = "run-owned"
	}
	return JcodeRuntimeHandle{Ownership: ownership, State: state, ownerToken: token}
}

// NewOwnedJcodeRuntimeHandle creates a handle for a runtime started by this run.
func NewOwnedJcodeRuntimeHandle(state JcodeRuntimeState) JcodeRuntimeHandle {
	return newJcodeRuntimeHandle(JcodeRuntimeOwnershipPrivate, state)
}

// CanRestart reports whether this run may restart the runtime.
func (h JcodeRuntimeHandle) CanRestart() bool {
	return h.Ownership == JcodeRuntimeOwnershipPrivate && h.ownerToken != ""
}

// CanCleanup reports whether this run may clean up the runtime.
func (h JcodeRuntimeHandle) CanCleanup() bool {
	return h.Ownership == JcodeRuntimeOwnershipPrivate && h.ownerToken != ""
}

// Validate checks observed counters against the configured bounds.
func (h JcodeRuntimeHandle) Validate(policy JcodeLifecyclePolicy) error {
	if err := policy.Validate(); err != nil {
		return fmt.Errorf("validating jcode lifecycle policy: %w", err)
	}
	if err := validateJcodeRuntimeState(h.State); err != nil {
		return fmt.Errorf("validating jcode runtime state: %w", err)
	}
	if h.ReconnectCount < 0 || h.ReconnectCount > policy.ReconnectAttempts {
		return fmt.Errorf("jcode reconnect count %d exceeds configured limit %d", h.ReconnectCount, policy.ReconnectAttempts)
	}
	if h.RestartCount < 0 || h.RestartCount > policy.RestartAttempts {
		return fmt.Errorf("jcode restart count %d exceeds configured limit %d", h.RestartCount, policy.RestartAttempts)
	}
	return nil
}

func validateJcodeRuntimeState(state JcodeRuntimeState) error {
	switch state {
	case JcodeRuntimeStateAbsent, JcodeRuntimeStateConnecting, JcodeRuntimeStateReady,
		JcodeRuntimeStateDisconnected, JcodeRuntimeStateFailed, JcodeRuntimeStateCleaned:
		return nil
	default:
		return fmt.Errorf("unknown jcode runtime state %q", state)
	}
}

// JcodeLifecycleDiagnostic describes a bounded lifecycle outcome.
type JcodeLifecycleDiagnostic struct {
	Policy     JcodeLifecycleMode
	State      JcodeRuntimeState
	Attempts   string
	NextAction string
	Message    string
}

// JcodeRuntimeRequest contains the SDK inputs for one runtime operation.
type JcodeRuntimeRequest struct {
	Options JcodeOptions
	WorkDir string
}

// JcodeProcessSpec describes an optional private startup process.
type JcodeProcessSpec struct {
	Command string
	WorkDir string
}

// JcodeProcessHandle identifies a process created by the current run.
type JcodeProcessHandle struct {
	ID string
}

type jcodeRuntime interface {
	Connect(context.Context, JcodeRuntimeRequest) (JcodeRuntimeHandle, error)
	Launch(context.Context, JcodeRuntimeRequest) (JcodeRuntimeHandle, error)
	Reconnect(context.Context, JcodeRuntimeHandle) (JcodeRuntimeHandle, error)
	Restart(context.Context, JcodeRuntimeHandle) (JcodeRuntimeHandle, error)
	Cleanup(context.Context, JcodeRuntimeHandle) error
}

type jcodeProcess interface {
	Start(context.Context, JcodeProcessSpec) (JcodeProcessHandle, error)
	Stop(context.Context, JcodeProcessHandle) error
}

type jcodeReadiness interface {
	WaitReady(context.Context, JcodeRuntimeHandle) error
}

type jcodeSleeper interface {
	Sleep(context.Context, time.Duration) error
}

type jcodeClock interface {
	Now() time.Time
}

// JcodeLifecycleController resolves a runtime without crossing ownership boundaries.
type JcodeLifecycleController struct {
	policy  JcodeLifecyclePolicy
	runtime jcodeRuntime
}

// NewJcodeLifecycleController creates a controller backed by an injected runtime boundary.
func NewJcodeLifecycleController(policy JcodeLifecyclePolicy, runtime jcodeRuntime) *JcodeLifecycleController {
	return &JcodeLifecycleController{policy: policy, runtime: runtime}
}

// Select chooses a healthy shared or owned-private runtime for one execution.
func (c *JcodeLifecycleController) Select(ctx context.Context, request JcodeRuntimeRequest) (JcodeRuntimeHandle, JcodeLifecycleDiagnostic, error) {
	if c == nil || c.runtime == nil {
		diagnostic := JcodeLifecycleDiagnostic{Policy: JcodeLifecycleModeConnect, State: JcodeRuntimeStateFailed, Attempts: "reconnect=0/0 restart=0/0", NextAction: "configure a jcode runtime boundary", Message: "jcode runtime lifecycle selection"}
		return JcodeRuntimeHandle{}, diagnostic, fmt.Errorf("selecting jcode runtime: runtime boundary is nil")
	}
	if err := c.policy.Validate(); err != nil {
		return JcodeRuntimeHandle{}, c.diagnostic(JcodeRuntimeStateFailed, "correct the jcode lifecycle configuration"), fmt.Errorf("selecting jcode runtime: %w", err)
	}
	switch c.policy.ResolvedMode() {
	case JcodeLifecycleModeConnect:
		return c.selectConnect(ctx, request)
	case JcodeLifecycleModePrivate:
		return c.selectPrivate(ctx, request)
	case JcodeLifecycleModeAuto:
		return c.selectAuto(ctx, request)
	default:
		return JcodeRuntimeHandle{}, c.diagnostic(JcodeRuntimeStateFailed, "choose connect, private, or auto"), fmt.Errorf("selecting jcode runtime: unsupported mode")
	}
}

// Recover applies the configured bounded recovery policy to a disconnected runtime.
func (c *JcodeLifecycleController) Recover(ctx context.Context, handle JcodeRuntimeHandle) (JcodeRuntimeHandle, JcodeLifecycleDiagnostic, error) {
	if err := handle.Validate(c.policy); err != nil {
		return handle, c.diagnostic(JcodeRuntimeStateFailed, "correct the jcode lifecycle configuration"), err
	}
	if handle.Ownership == JcodeRuntimeOwnershipShared {
		return c.reconnect(ctx, handle)
	}
	if handle.CanRestart() {
		return c.restart(ctx, handle)
	}
	return handle, c.diagnostic(JcodeRuntimeStateFailed, "use connect mode for shared runtimes or select a private runtime"), fmt.Errorf("jcode runtime recovery is not permitted for %s ownership", handle.Ownership)
}

func (c *JcodeLifecycleController) reconnect(ctx context.Context, handle JcodeRuntimeHandle) (JcodeRuntimeHandle, JcodeLifecycleDiagnostic, error) {
	for attempt := 1; attempt <= c.policy.ReconnectAttempts; attempt++ {
		if err := c.wait(ctx, attempt); err != nil {
			return handle, c.diagnostic(JcodeRuntimeStateDisconnected, "retry the shared bridge when it is available"), err
		}
		next, err := c.runtime.Reconnect(ctx, handle)
		if err == nil && next.State == JcodeRuntimeStateReady {
			next.ReconnectCount = attempt
			return next, c.diagnosticWithAttempts(next.State, attempt, next.RestartCount, "continue using the recovered shared bridge"), nil
		}
		handle.ReconnectCount = attempt
	}
	return handle, c.diagnosticWithAttempts(JcodeRuntimeStateDisconnected, handle.ReconnectCount, handle.RestartCount, "retry the shared bridge or switch to private/auto mode"), fmt.Errorf("jcode shared runtime reconnect exhausted")
}

func (c *JcodeLifecycleController) restart(ctx context.Context, handle JcodeRuntimeHandle) (JcodeRuntimeHandle, JcodeLifecycleDiagnostic, error) {
	for attempt := 1; attempt <= c.policy.RestartAttempts; attempt++ {
		if err := c.wait(ctx, attempt); err != nil {
			return handle, c.diagnostic(JcodeRuntimeStateDisconnected, "retry the owned private runtime when it is available"), err
		}
		next, err := c.runtime.Restart(ctx, handle)
		if err == nil && next.State == JcodeRuntimeStateReady {
			next.RestartCount = attempt
			return next, c.diagnosticWithAttempts(next.State, next.ReconnectCount, attempt, "continue using the recovered private bridge"), nil
		}
		handle.RestartCount = attempt
	}
	return handle, c.diagnosticWithAttempts(JcodeRuntimeStateDisconnected, handle.ReconnectCount, handle.RestartCount, "retry the shared bridge or switch to private/auto mode"), fmt.Errorf("jcode private runtime restart exhausted")
}

func (c *JcodeLifecycleController) wait(ctx context.Context, attempt int) error {
	if attempt == 1 || c.policy.RetryDelay == 0 {
		return nil
	}
	timer := time.NewTimer(c.policy.RetryDelay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("waiting for jcode recovery: %w", ctx.Err())
	}
}

// Cleanup closes only a runtime proven to be owned by this run.
func (c *JcodeLifecycleController) Cleanup(ctx context.Context, handle JcodeRuntimeHandle) error {
	if !handle.CanCleanup() {
		return nil
	}
	if err := c.runtime.Cleanup(ctx, handle); err != nil {
		return fmt.Errorf("cleaning up owned jcode runtime: %w", err)
	}
	return nil
}

func (c *JcodeLifecycleController) selectConnect(ctx context.Context, request JcodeRuntimeRequest) (JcodeRuntimeHandle, JcodeLifecycleDiagnostic, error) {
	handle, err := c.runtime.Connect(ctx, request)
	if err != nil {
		return c.runtimeError(classifyJcodeRuntimeError(err), "start or inspect the shared jcode bridge", err)
	}
	if handle.Ownership != JcodeRuntimeOwnershipShared || handle.State != JcodeRuntimeStateReady {
		return c.runtimeError(JcodeRuntimeStateFailed, "verify shared runtime ownership and select private or auto if needed", fmt.Errorf("shared runtime is not healthy"))
	}
	return handle, c.diagnostic(handle.State, "continue using the shared bridge"), nil
}

func (c *JcodeLifecycleController) selectPrivate(ctx context.Context, request JcodeRuntimeRequest) (JcodeRuntimeHandle, JcodeLifecycleDiagnostic, error) {
	handle, err := c.runtime.Launch(ctx, request)
	if err != nil {
		return c.runtimeError(JcodeRuntimeStateFailed, "check the private runtime startup configuration", err)
	}
	if handle.Ownership != JcodeRuntimeOwnershipPrivate || handle.State != JcodeRuntimeStateReady {
		return c.runtimeError(JcodeRuntimeStateFailed, "verify that the private runtime is owned by this run", fmt.Errorf("private runtime is not healthy"))
	}
	return handle, c.diagnostic(handle.State, "continue using the owned private runtime"), nil
}

func (c *JcodeLifecycleController) selectAuto(ctx context.Context, request JcodeRuntimeRequest) (JcodeRuntimeHandle, JcodeLifecycleDiagnostic, error) {
	handle, err := c.runtime.Connect(ctx, request)
	if err == nil && handle.Ownership == JcodeRuntimeOwnershipShared && handle.State == JcodeRuntimeStateReady {
		return handle, c.diagnostic(handle.State, "continue using the shared bridge"), nil
	}
	return c.selectPrivate(ctx, request)
}

func (c *JcodeLifecycleController) runtimeError(state JcodeRuntimeState, nextAction string, cause error) (JcodeRuntimeHandle, JcodeLifecycleDiagnostic, error) {
	diagnostic := c.diagnostic(state, nextAction)
	diagnostic.NextAction = nextAction
	return JcodeRuntimeHandle{}, diagnostic, &JcodeLifecycleError{Diagnostic: diagnostic, Cause: cause}
}

func (c *JcodeLifecycleController) diagnostic(state JcodeRuntimeState, nextAction string) JcodeLifecycleDiagnostic {
	return c.diagnosticWithAttempts(state, 0, 0, nextAction)
}

func (c *JcodeLifecycleController) diagnosticWithAttempts(state JcodeRuntimeState, reconnect, restart int, nextAction string) JcodeLifecycleDiagnostic {
	return JcodeLifecycleDiagnostic{
		Policy: c.policy.ResolvedMode(), State: state,
		Attempts:   fmt.Sprintf("reconnect=%d/%d restart=%d/%d", reconnect, c.policy.ReconnectAttempts, restart, c.policy.RestartAttempts),
		NextAction: nextAction, Message: "jcode runtime lifecycle operation",
	}
}

func classifyJcodeRuntimeError(err error) JcodeRuntimeState {
	if strings.Contains(strings.ToLower(err.Error()), "disconnect") {
		return JcodeRuntimeStateDisconnected
	}
	return JcodeRuntimeStateAbsent
}

// JcodeLifecycleError reports a redacted, actionable lifecycle failure.
type JcodeLifecycleError struct {
	Diagnostic JcodeLifecycleDiagnostic
	Cause      error
}

func (e *JcodeLifecycleError) Error() string {
	return fmt.Sprintf("jcode lifecycle failure: policy=%s state=%s attempts=%s next_action=%s", e.Diagnostic.Policy, e.Diagnostic.State, e.Diagnostic.Attempts, e.Diagnostic.NextAction)
}

func (e *JcodeLifecycleError) Unwrap() error { return e.Cause }
