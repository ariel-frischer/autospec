package testutil

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/cliagent"
)

func TestJcodeLifecycleFakeRecordsRuntimeOperations(t *testing.T) {
	t.Parallel()

	fake := NewJcodeLifecycleFake()
	request := cliagent.JcodeRuntimeRequest{WorkDir: "/tmp/work"}
	shared, err := fake.Connect(context.Background(), request)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	private, err := fake.Launch(context.Background(), request)
	if err != nil {
		t.Fatalf("Launch() error = %v", err)
	}
	if shared.Ownership != cliagent.JcodeRuntimeOwnershipShared {
		t.Fatalf("shared ownership = %q", shared.Ownership)
	}
	if !private.CanCleanup() {
		t.Fatal("private handle should allow cleanup")
	}
	if fake.ConnectCalls != 1 || fake.LaunchCalls != 1 {
		t.Fatalf("runtime calls = connect:%d launch:%d", fake.ConnectCalls, fake.LaunchCalls)
	}
	if len(fake.ConnectRequests) != 1 || len(fake.LaunchRequests) != 1 {
		t.Fatalf("runtime request records = connect:%d launch:%d", len(fake.ConnectRequests), len(fake.LaunchRequests))
	}
}

func TestJcodeLifecycleFakeRecordsRecoveryBoundaries(t *testing.T) {
	t.Parallel()

	fake := NewJcodeLifecycleFake()
	fake.ReconnectErrs = []error{errors.New("disconnect")}
	fake.RestartErrs = []error{errors.New("restart failed")}
	shared, _ := fake.Connect(context.Background(), cliagent.JcodeRuntimeRequest{})
	_, _ = fake.Reconnect(context.Background(), shared)
	private, _ := fake.Launch(context.Background(), cliagent.JcodeRuntimeRequest{})
	_, _ = fake.Restart(context.Background(), private)
	_ = fake.WaitReady(context.Background(), private)
	_ = fake.Sleep(context.Background(), 25*time.Millisecond)

	if fake.ReconnectAttempts[0] != 1 || fake.RestartAttempts[0] != 1 {
		t.Fatalf("attempt records = reconnect:%v restart:%v", fake.ReconnectAttempts, fake.RestartAttempts)
	}
	if fake.ReadinessCalls != 1 || fake.SleepDelays[0] != 25*time.Millisecond {
		t.Fatalf("readiness/delay records = %d/%v", fake.ReadinessCalls, fake.SleepDelays)
	}
}

func TestJcodeLifecycleFakeRejectsSharedCleanup(t *testing.T) {
	t.Parallel()

	fake := NewJcodeLifecycleFake()
	shared, err := fake.Connect(context.Background(), cliagent.JcodeRuntimeRequest{})
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if err := fake.Cleanup(context.Background(), shared); err == nil {
		t.Fatal("Cleanup(shared) error = nil")
	}
	if fake.CleanupCalls != 0 {
		t.Fatalf("CleanupCalls = %d, want 0", fake.CleanupCalls)
	}
}
