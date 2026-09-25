//go:build unix

package shutdown

import (
	"errors"
	"os"
	"syscall"
	"testing"
	"time"
)

// Not parallel: Install mutates process-wide signal handling.
func TestInstall_SignalCancelsContext(t *testing.T) {
	tests := map[string]struct {
		signal   syscall.Signal
		wantCode int
	}{
		"SIGINT":  {signal: syscall.SIGINT, wantCode: 130},
		"SIGTERM": {signal: syscall.SIGTERM, wantCode: 143},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, stop := Install()
			defer stop()
			if _, ok := ExitCode(); ok {
				t.Fatal("ExitCode() reported a signal before one was sent")
			}

			if err := syscall.Kill(os.Getpid(), tt.signal); err != nil {
				t.Fatalf("sending %v: %v", tt.signal, err)
			}
			select {
			case <-ctx.Done():
			case <-time.After(2 * time.Second):
				t.Fatalf("context not cancelled after %v", tt.signal)
			}
			if !errors.Is(Context().Err(), ctx.Err()) {
				t.Errorf("Context() err = %v, want installed context err %v", Context().Err(), ctx.Err())
			}
			if code, ok := ExitCode(); !ok || code != tt.wantCode {
				t.Errorf("ExitCode() = %d, %v; want %d, true", code, ok, tt.wantCode)
			}
		})
	}
}

func TestInstall_StopWithoutSignal(t *testing.T) {
	ctx, stop := Install()
	stop()
	stop() // idempotent
	if ctx.Err() == nil {
		t.Error("stop() should release the installed context")
	}
	if _, ok := ExitCode(); ok {
		t.Error("ExitCode() reported a signal that never arrived")
	}
}
