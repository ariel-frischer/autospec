package cliagent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type lifecycleSelectionFake struct {
	connectHandle JcodeRuntimeHandle
	launchHandle  JcodeRuntimeHandle
	connectErr    error
	launchErr     error
	connectCalls  int
	launchCalls   int
}

func (fake *lifecycleSelectionFake) Reconnect(context.Context, JcodeRuntimeHandle) (JcodeRuntimeHandle, error) {
	return fake.connectHandle, fake.connectErr
}

func (fake *lifecycleSelectionFake) Restart(context.Context, JcodeRuntimeHandle) (JcodeRuntimeHandle, error) {
	return fake.launchHandle, fake.launchErr
}

func (fake *lifecycleSelectionFake) Connect(context.Context, JcodeRuntimeRequest) (JcodeRuntimeHandle, error) {
	fake.connectCalls++
	return fake.connectHandle, fake.connectErr
}

func (fake *lifecycleSelectionFake) Launch(context.Context, JcodeRuntimeRequest) (JcodeRuntimeHandle, error) {
	fake.launchCalls++
	return fake.launchHandle, fake.launchErr
}

func (*lifecycleSelectionFake) Cleanup(context.Context, JcodeRuntimeHandle) error { return nil }

func TestJcodeLifecycleControllerSelectsPolicyOwnership(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		policy        JcodeLifecyclePolicy
		connectHandle JcodeRuntimeHandle
		connectErr    error
		launchHandle  JcodeRuntimeHandle
		wantOwner     JcodeRuntimeOwnership
		wantState     JcodeRuntimeState
		wantLaunch    int
		wantErr       string
	}{
		"connect uses healthy shared runtime": {
			policy:        JcodeLifecyclePolicy{Mode: JcodeLifecycleModeConnect},
			connectHandle: JcodeRuntimeHandle{Ownership: JcodeRuntimeOwnershipShared, State: JcodeRuntimeStateReady},
			wantOwner:     JcodeRuntimeOwnershipShared,
			wantState:     JcodeRuntimeStateReady,
		},
		"connect rejects missing runtime": {
			policy:     JcodeLifecyclePolicy{Mode: JcodeLifecycleModeConnect},
			connectErr: errors.New("dial failed"),
			wantState:  JcodeRuntimeStateAbsent,
			wantErr:    "policy=connect",
		},
		"private uses owned runtime": {
			policy:       JcodeLifecyclePolicy{Mode: JcodeLifecycleModePrivate},
			launchHandle: JcodeRuntimeHandle{Ownership: JcodeRuntimeOwnershipPrivate, State: JcodeRuntimeStateReady},
			wantOwner:    JcodeRuntimeOwnershipPrivate,
			wantState:    JcodeRuntimeStateReady,
			wantLaunch:   1,
		},
		"auto falls back to private": {
			policy:       JcodeLifecyclePolicy{Mode: JcodeLifecycleModeAuto},
			connectErr:   errors.New("bridge disconnected"),
			launchHandle: JcodeRuntimeHandle{Ownership: JcodeRuntimeOwnershipPrivate, State: JcodeRuntimeStateReady},
			wantOwner:    JcodeRuntimeOwnershipPrivate,
			wantState:    JcodeRuntimeStateReady,
			wantLaunch:   1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fake := &lifecycleSelectionFake{
				connectHandle: tt.connectHandle,
				connectErr:    tt.connectErr,
				launchHandle:  tt.launchHandle,
			}
			controller := NewJcodeLifecycleController(tt.policy, fake)
			handle, diagnostic, err := controller.Select(context.Background(), JcodeRuntimeRequest{})
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Select() error = %v, want %q", err, tt.wantErr)
				}
				if diagnostic.State != tt.wantState {
					t.Fatalf("diagnostic state = %q, want %q", diagnostic.State, tt.wantState)
				}
				return
			}
			if err != nil {
				t.Fatalf("Select() error = %v", err)
			}
			if handle.Ownership != tt.wantOwner || handle.State != tt.wantState {
				t.Fatalf("handle = %#v, want owner=%q state=%q", handle, tt.wantOwner, tt.wantState)
			}
			if fake.launchCalls != tt.wantLaunch {
				t.Fatalf("launch calls = %d, want %d", fake.launchCalls, tt.wantLaunch)
			}
			if diagnostic.Policy != tt.policy.ResolvedMode() {
				t.Fatalf("diagnostic policy = %q, want %q", diagnostic.Policy, tt.policy.ResolvedMode())
			}
		})
	}
}

func TestJcodeLifecycleControllerNeverAcceptsUnrelatedRuntime(t *testing.T) {
	t.Parallel()

	fake := &lifecycleSelectionFake{
		connectHandle: JcodeRuntimeHandle{Ownership: JcodeRuntimeOwnershipUnrelated, State: JcodeRuntimeStateReady},
	}
	controller := NewJcodeLifecycleController(JcodeLifecyclePolicy{Mode: JcodeLifecycleModeConnect}, fake)
	_, diagnostic, err := controller.Select(context.Background(), JcodeRuntimeRequest{})
	if err == nil || !strings.Contains(err.Error(), "ownership") {
		t.Fatalf("Select() error = %v, want ownership error", err)
	}
	if diagnostic.State != JcodeRuntimeStateFailed {
		t.Fatalf("diagnostic state = %q, want failed", diagnostic.State)
	}
}

func TestJcodeLifecyclePolicyValidate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		policy  JcodeLifecyclePolicy
		wantErr string
	}{
		"connect policy": {
			policy: JcodeLifecyclePolicy{Mode: JcodeLifecycleModeConnect},
		},
		"private policy with startup command": {
			policy: JcodeLifecyclePolicy{
				Mode:            JcodeLifecycleModePrivate,
				StartupCommand:  "jcode serve",
				RestartAttempts: 2,
				RetryDelay:      100 * time.Millisecond,
			},
		},
		"auto policy with bounded reconnect": {
			policy: JcodeLifecyclePolicy{
				Mode:              JcodeLifecycleModeAuto,
				ReconnectAttempts: 3,
				RetryDelay:        time.Second,
			},
		},
		"empty mode uses connect default": {
			policy: JcodeLifecyclePolicy{},
		},
		"invalid mode": {
			policy:  JcodeLifecyclePolicy{Mode: "launch"},
			wantErr: "mode",
		},
		"negative reconnect attempts": {
			policy:  JcodeLifecyclePolicy{ReconnectAttempts: -1},
			wantErr: "reconnect",
		},
		"negative restart attempts": {
			policy:  JcodeLifecyclePolicy{RestartAttempts: -1},
			wantErr: "restart",
		},
		"negative retry delay": {
			policy:  JcodeLifecyclePolicy{RetryDelay: -time.Second},
			wantErr: "retry",
		},
		"too many reconnect attempts": {
			policy:  JcodeLifecyclePolicy{ReconnectAttempts: 11},
			wantErr: "at most",
		},
		"too long retry delay": {
			policy:  JcodeLifecyclePolicy{RetryDelay: 6 * time.Minute},
			wantErr: "at most",
		},
		"unsafe startup command": {
			policy:  JcodeLifecyclePolicy{Mode: JcodeLifecycleModePrivate, StartupCommand: "jcode;rm -rf /"},
			wantErr: "shell",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := tt.policy.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %v, want context %q", err, tt.wantErr)
			}
		})
	}
}

func TestJcodeLifecyclePolicyResolvedMode(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		policy JcodeLifecyclePolicy
		want   JcodeLifecycleMode
	}{
		"empty defaults to connect": {want: JcodeLifecycleModeConnect},
		"connect remains connect": {
			policy: JcodeLifecyclePolicy{Mode: JcodeLifecycleModeConnect},
			want:   JcodeLifecycleModeConnect,
		},
		"private remains private": {
			policy: JcodeLifecyclePolicy{Mode: JcodeLifecycleModePrivate},
			want:   JcodeLifecycleModePrivate,
		},
		"auto remains auto": {
			policy: JcodeLifecyclePolicy{Mode: JcodeLifecycleModeAuto},
			want:   JcodeLifecycleModeAuto,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := tt.policy.ResolvedMode(); got != tt.want {
				t.Fatalf("ResolvedMode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestJcodeRuntimeHandleOwnershipCapabilities(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		ownership   JcodeRuntimeOwnership
		wantRestart bool
		wantCleanup bool
	}{
		"shared runtime": {
			ownership: JcodeRuntimeOwnershipShared,
		},
		"private runtime": {
			ownership:   JcodeRuntimeOwnershipPrivate,
			wantRestart: true,
			wantCleanup: true,
		},
		"unrelated runtime": {
			ownership: JcodeRuntimeOwnershipUnrelated,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			handle := newJcodeRuntimeHandle(tt.ownership, JcodeRuntimeStateReady)
			if got := handle.CanRestart(); got != tt.wantRestart {
				t.Fatalf("CanRestart() = %t, want %t", got, tt.wantRestart)
			}
			if got := handle.CanCleanup(); got != tt.wantCleanup {
				t.Fatalf("CanCleanup() = %t, want %t", got, tt.wantCleanup)
			}
		})
	}
}

func TestJcodeRuntimeHandleValidateCounters(t *testing.T) {
	t.Parallel()

	policy := JcodeLifecyclePolicy{
		Mode:              JcodeLifecycleModeAuto,
		ReconnectAttempts: 2,
		RestartAttempts:   1,
	}
	tests := map[string]struct {
		handle  JcodeRuntimeHandle
		wantErr string
	}{
		"within configured limits": {
			handle: JcodeRuntimeHandle{
				Ownership:      JcodeRuntimeOwnershipPrivate,
				State:          JcodeRuntimeStateDisconnected,
				ReconnectCount: 2,
				RestartCount:   1,
			},
		},
		"negative reconnect count": {
			handle:  JcodeRuntimeHandle{State: JcodeRuntimeStateReady, ReconnectCount: -1},
			wantErr: "reconnect",
		},
		"reconnect count exceeds limit": {
			handle:  JcodeRuntimeHandle{State: JcodeRuntimeStateReady, ReconnectCount: 3},
			wantErr: "reconnect",
		},
		"negative restart count": {
			handle:  JcodeRuntimeHandle{State: JcodeRuntimeStateReady, RestartCount: -1},
			wantErr: "restart",
		},
		"restart count exceeds limit": {
			handle:  JcodeRuntimeHandle{State: JcodeRuntimeStateReady, RestartCount: 2},
			wantErr: "restart",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := tt.handle.Validate(policy)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %v, want context %q", err, tt.wantErr)
			}
		})
	}
}

func TestJcodeRuntimeStateValues(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		state JcodeRuntimeState
	}{
		"absent":       {state: JcodeRuntimeStateAbsent},
		"connecting":   {state: JcodeRuntimeStateConnecting},
		"ready":        {state: JcodeRuntimeStateReady},
		"disconnected": {state: JcodeRuntimeStateDisconnected},
		"failed":       {state: JcodeRuntimeStateFailed},
		"cleaned":      {state: JcodeRuntimeStateCleaned},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := validateJcodeRuntimeState(tt.state); err != nil {
				t.Fatalf("validateJcodeRuntimeState() error = %v", err)
			}
		})
	}
}

func TestJcodeLifecycleControllerRecoversWithinOwnershipLimits(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		policy      JcodeLifecyclePolicy
		handle      JcodeRuntimeHandle
		connectErr  error
		launchErr   error
		wantOwner   JcodeRuntimeOwnership
		wantState   JcodeRuntimeState
		wantAction  string
		wantFailure bool
	}{
		"shared reconnect succeeds": {
			policy:     JcodeLifecyclePolicy{Mode: JcodeLifecycleModeConnect, ReconnectAttempts: 1},
			handle:     JcodeRuntimeHandle{Ownership: JcodeRuntimeOwnershipShared, State: JcodeRuntimeStateDisconnected},
			wantOwner:  JcodeRuntimeOwnershipShared,
			wantState:  JcodeRuntimeStateReady,
			wantAction: "recovered shared",
		},
		"private restart succeeds": {
			policy:     JcodeLifecyclePolicy{Mode: JcodeLifecycleModePrivate, RestartAttempts: 1},
			handle:     NewOwnedJcodeRuntimeHandle(JcodeRuntimeStateDisconnected),
			wantOwner:  JcodeRuntimeOwnershipPrivate,
			wantState:  JcodeRuntimeStateReady,
			wantAction: "recovered private",
		},
		"shared reconnect exhaustion is bounded": {
			policy:      JcodeLifecyclePolicy{Mode: JcodeLifecycleModeConnect, ReconnectAttempts: 1},
			handle:      JcodeRuntimeHandle{Ownership: JcodeRuntimeOwnershipShared, State: JcodeRuntimeStateDisconnected},
			connectErr:  errors.New("bridge disconnected"),
			wantAction:  "retry the shared bridge",
			wantFailure: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fake := &lifecycleSelectionFake{
				connectHandle: JcodeRuntimeHandle{Ownership: JcodeRuntimeOwnershipShared, State: JcodeRuntimeStateReady},
				connectErr:    tt.connectErr,
				launchHandle:  JcodeRuntimeHandle{Ownership: JcodeRuntimeOwnershipPrivate, State: JcodeRuntimeStateReady, ownerToken: "run-owned"},
			}
			controller := NewJcodeLifecycleController(tt.policy, fake)
			handle, diagnostic, err := controller.Recover(context.Background(), tt.handle)
			if tt.wantFailure {
				if err == nil || !strings.Contains(diagnostic.NextAction, tt.wantAction) {
					t.Fatalf("Recover() error = %v, diagnostic = %#v", err, diagnostic)
				}
				return
			}
			if err != nil || handle.Ownership != tt.wantOwner || handle.State != tt.wantState {
				t.Fatalf("Recover() = (%#v, %v), want owner=%q state=%q", handle, err, tt.wantOwner, tt.wantState)
			}
			if !strings.Contains(diagnostic.NextAction, tt.wantAction) {
				t.Fatalf("diagnostic next action = %q, want %q", diagnostic.NextAction, tt.wantAction)
			}
		})
	}
}

func TestJcodeLifecycleControllerCleanupPreservesSharedOwnership(t *testing.T) {
	t.Parallel()
	fake := &lifecycleSelectionFake{}
	controller := NewJcodeLifecycleController(JcodeLifecyclePolicy{Mode: JcodeLifecycleModeConnect}, fake)
	shared := JcodeRuntimeHandle{Ownership: JcodeRuntimeOwnershipShared, State: JcodeRuntimeStateReady}
	if err := controller.Cleanup(context.Background(), shared); err != nil {
		t.Fatalf("Cleanup(shared) error = %v", err)
	}
}
