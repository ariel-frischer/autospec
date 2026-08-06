package testutil

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ariel-frischer/autospec/internal/cliagent"
)

// JcodeLifecycleFake records deterministic runtime recovery operations.
type JcodeLifecycleFake struct {
	mu sync.Mutex

	SharedHandle  cliagent.JcodeRuntimeHandle
	PrivateHandle cliagent.JcodeRuntimeHandle
	ProcessHandle cliagent.JcodeProcessHandle
	ClockTime     time.Time

	ConnectErr    error
	LaunchErr     error
	StartErr      error
	StopErr       error
	CleanupErr    error
	SleepErr      error
	ReconnectErrs []error
	RestartErrs   []error
	ReadinessErrs []error

	ConnectCalls   int
	LaunchCalls    int
	StartCalls     int
	StopCalls      int
	CleanupCalls   int
	ReadinessCalls int
	ReconnectCalls int
	RestartCalls   int
	SleepCalls     int

	ConnectRequests   []cliagent.JcodeRuntimeRequest
	LaunchRequests    []cliagent.JcodeRuntimeRequest
	ProcessSpecs      []cliagent.JcodeProcessSpec
	ReconnectAttempts []int
	RestartAttempts   []int
	SleepDelays       []time.Duration
}

// NewJcodeLifecycleFake creates a fake with shared and owned-private fixtures.
func NewJcodeLifecycleFake() *JcodeLifecycleFake {
	return &JcodeLifecycleFake{
		SharedHandle: cliagent.JcodeRuntimeHandle{
			ID: "shared-runtime", Ownership: cliagent.JcodeRuntimeOwnershipShared,
			State: cliagent.JcodeRuntimeStateReady,
		},
		PrivateHandle: cliagent.NewOwnedJcodeRuntimeHandle(cliagent.JcodeRuntimeStateReady),
		ProcessHandle: cliagent.JcodeProcessHandle{ID: "private-process"},
		ClockTime:     time.Unix(0, 0),
	}
}

// Connect records a shared-runtime connection attempt.
func (fake *JcodeLifecycleFake) Connect(_ context.Context, request cliagent.JcodeRuntimeRequest) (cliagent.JcodeRuntimeHandle, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.ConnectCalls++
	fake.ConnectRequests = append(fake.ConnectRequests, request)
	if fake.ConnectErr != nil {
		return fake.SharedHandle, fake.ConnectErr
	}
	return fake.SharedHandle, nil
}

// Launch records an owned-private runtime launch.
func (fake *JcodeLifecycleFake) Launch(_ context.Context, request cliagent.JcodeRuntimeRequest) (cliagent.JcodeRuntimeHandle, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.LaunchCalls++
	fake.LaunchRequests = append(fake.LaunchRequests, request)
	if fake.LaunchErr != nil {
		return fake.PrivateHandle, fake.LaunchErr
	}
	return fake.PrivateHandle, nil
}

// Reconnect records a bounded reconnect attempt.
func (fake *JcodeLifecycleFake) Reconnect(_ context.Context, handle cliagent.JcodeRuntimeHandle) (cliagent.JcodeRuntimeHandle, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.ReconnectCalls++
	handle.ReconnectCount++
	fake.ReconnectAttempts = append(fake.ReconnectAttempts, handle.ReconnectCount)
	if err := popLifecycleError(&fake.ReconnectErrs); err != nil {
		return handle, err
	}
	handle.State = cliagent.JcodeRuntimeStateReady
	return handle, nil
}

// Restart records an owned-private restart attempt.
func (fake *JcodeLifecycleFake) Restart(_ context.Context, handle cliagent.JcodeRuntimeHandle) (cliagent.JcodeRuntimeHandle, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if !handle.CanRestart() {
		return handle, fmt.Errorf("cannot restart non-private jcode runtime")
	}
	fake.RestartCalls++
	handle.RestartCount++
	fake.RestartAttempts = append(fake.RestartAttempts, handle.RestartCount)
	if err := popLifecycleError(&fake.RestartErrs); err != nil {
		return handle, err
	}
	handle.State = cliagent.JcodeRuntimeStateReady
	return handle, nil
}

// Cleanup records cleanup for an owned-private runtime.
func (fake *JcodeLifecycleFake) Cleanup(_ context.Context, handle cliagent.JcodeRuntimeHandle) error {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if !handle.CanCleanup() {
		return fmt.Errorf("cannot clean up non-private jcode runtime")
	}
	fake.CleanupCalls++
	return fake.CleanupErr
}

// Start records an optional private startup process.
func (fake *JcodeLifecycleFake) Start(_ context.Context, spec cliagent.JcodeProcessSpec) (cliagent.JcodeProcessHandle, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.StartCalls++
	fake.ProcessSpecs = append(fake.ProcessSpecs, spec)
	return fake.ProcessHandle, fake.StartErr
}

// Stop records stopping a process created by this run.
func (fake *JcodeLifecycleFake) Stop(_ context.Context, _ cliagent.JcodeProcessHandle) error {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.StopCalls++
	return fake.StopErr
}

// WaitReady records readiness checks.
func (fake *JcodeLifecycleFake) WaitReady(_ context.Context, _ cliagent.JcodeRuntimeHandle) error {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.ReadinessCalls++
	return popLifecycleError(&fake.ReadinessErrs)
}

// Sleep records bounded retry delays.
func (fake *JcodeLifecycleFake) Sleep(_ context.Context, delay time.Duration) error {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.SleepCalls++
	fake.SleepDelays = append(fake.SleepDelays, delay)
	return fake.SleepErr
}

// Now returns the configured deterministic clock value.
func (fake *JcodeLifecycleFake) Now() time.Time {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return fake.ClockTime
}

func popLifecycleError(queue *[]error) error {
	if len(*queue) == 0 {
		return nil
	}
	err := (*queue)[0]
	*queue = (*queue)[1:]
	return err
}
