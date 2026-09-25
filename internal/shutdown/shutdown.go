// Package shutdown converts process termination signals into context
// cancellation so running agent process groups are reaped before autospec
// exits.
package shutdown

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ForceExitDelay bounds how long autospec keeps running after the first
// SIGINT/SIGTERM. It exceeds the agent SIGTERM grace plus SIGKILL reap window
// so cancelled agent process groups are cleaned up first.
const ForceExitDelay = 10 * time.Second

var (
	mu       sync.RWMutex
	current  = context.Background()
	received os.Signal
)

// Context returns the process-lifetime context installed by Install, or
// context.Background() when no handler is installed (tests, library use).
func Context() context.Context {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// Install cancels the returned context (also served by Context) on the first
// SIGINT or SIGTERM. A second signal, or ForceExitDelay elapsing, exits the
// process with the conventional 128+signal status. Call stop to restore the
// default signal behavior.
func Install() (ctx context.Context, stop func()) {
	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	stopped := make(chan struct{})
	mu.Lock()
	current, received = ctx, nil
	mu.Unlock()

	go watch(signals, stopped, cancel)
	var once sync.Once
	return ctx, func() {
		once.Do(func() {
			signal.Stop(signals)
			close(stopped)
			cancel()
		})
	}
}

// ExitCode returns 128+signal after a handled signal and ok=false otherwise.
func ExitCode() (code int, ok bool) {
	mu.RLock()
	defer mu.RUnlock()
	if received == nil {
		return 0, false
	}
	return signalExitCode(received), true
}

func watch(signals <-chan os.Signal, stopped <-chan struct{}, cancel context.CancelFunc) {
	var sig os.Signal
	select {
	case sig = <-signals:
	case <-stopped:
		return
	}
	mu.Lock()
	received = sig
	mu.Unlock()
	cancel()

	select {
	case sig = <-signals:
	case <-time.After(ForceExitDelay):
	case <-stopped:
		return
	}
	os.Exit(signalExitCode(sig))
}

func signalExitCode(sig os.Signal) int {
	if s, ok := sig.(syscall.Signal); ok {
		return 128 + int(s)
	}
	return 1
}
