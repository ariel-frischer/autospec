package cliagent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

const (
	// agentGroupGrace is how long an agent process group may take to exit
	// after SIGTERM before it is killed.
	agentGroupGrace = 3 * time.Second
	// agentGroupReapTimeout bounds the wait for a SIGKILLed group to disappear.
	agentGroupReapTimeout = 2 * time.Second
	// agentOutputDrainTimeout bounds output draining after the group is gone,
	// covering descendants that escaped the group while holding the pipes.
	agentOutputDrainTimeout = time.Second
)

// runAgentProcess starts cmd and blocks until it exits or ctx is done.
//
// Isolated runs (every non-interactive agent spawn) get their own process
// group on unix. On cancellation or timeout the whole group receives SIGTERM,
// then SIGKILL after a grace period; after a normal exit, leftover group
// members are terminated the same way. The call returns only once the group
// is gone, so no descendant can mutate the workspace afterwards.
// Interactive runs must stay in the terminal's foreground process group and
// are not isolated.
//
// The returned error is ctx.Err() on cancellation, a wrapped start error, or
// the raw cmd.Wait error (e.g. *exec.ExitError) otherwise.
func runAgentProcess(ctx context.Context, cmd *exec.Cmd, isolate bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if isolate {
		return runIsolatedProcess(ctx, cmd)
	}
	return runDirectProcess(ctx, cmd)
}

// runAgentResult runs cmd via runAgentProcess and reports its captured output.
// A non-zero exit is returned through Result.ExitCode, not as an error.
func runAgentResult(ctx context.Context, cmd *exec.Cmd, isolate bool, stdout, stderr *bytes.Buffer) (*Result, error) {
	start := time.Now()
	err := runAgentProcess(ctx, cmd, isolate)
	result := &Result{Duration: time.Since(start), Stdout: stdout.String(), Stderr: stderr.String()}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

// runDirectProcess runs cmd in the caller's process group, killing only the
// direct child on cancellation.
func runDirectProcess(ctx context.Context, cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting process: %w", err)
	}
	done := waitAsync(cmd)
	select {
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		<-done
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func waitAsync(cmd *exec.Cmd) <-chan error {
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	return done
}
