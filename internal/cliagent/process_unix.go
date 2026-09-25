//go:build unix

package cliagent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

const groupPollInterval = 10 * time.Millisecond

// runIsolatedProcess runs cmd as the leader of a new process group and reaps
// the whole group before returning. See runAgentProcess.
func runIsolatedProcess(ctx context.Context, cmd *exec.Cmd) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	pipes, err := attachOutputPipes(cmd)
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		pipes.abort()
		return fmt.Errorf("starting process: %w", err)
	}
	pipes.started()
	pgid := cmd.Process.Pid
	done := waitAsync(cmd)

	var runErr error
	select {
	case <-ctx.Done():
		stopProcessGroup(pgid)
		<-done
		runErr = ctx.Err()
	case runErr = <-done:
		stopProcessGroup(pgid)
	}
	pipes.drain(agentOutputDrainTimeout)
	return runErr
}

// stopProcessGroup sends SIGTERM to every member of the group, escalates to
// SIGKILL after agentGroupGrace, and waits for the group to disappear.
func stopProcessGroup(pgid int) {
	if !processGroupAlive(pgid) {
		return
	}
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	if waitProcessGroupGone(pgid, agentGroupGrace) {
		return
	}
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
	waitProcessGroupGone(pgid, agentGroupReapTimeout)
}

func processGroupAlive(pgid int) bool {
	return !errors.Is(syscall.Kill(-pgid, 0), syscall.ESRCH)
}

func waitProcessGroupGone(pgid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for processGroupAlive(pgid) {
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(groupPollInterval)
	}
	return true
}

// outputPipes replaces non-file stdout/stderr writers with OS pipes owned by
// this package. exec.Cmd.Wait would otherwise block until every descendant
// holding the pipes exits, which prevents detecting the leader's exit.
type outputPipes struct {
	readers []*os.File
	writers []*os.File
	dests   []io.Writer
	copies  sync.WaitGroup
}

func attachOutputPipes(cmd *exec.Cmd) (*outputPipes, error) {
	pipes := &outputPipes{}
	stdout, err := pipes.attach(cmd.Stdout)
	if err != nil {
		return nil, err
	}
	stderr := stdout
	if !sameWriter(cmd.Stderr, cmd.Stdout) {
		if stderr, err = pipes.attach(cmd.Stderr); err != nil {
			pipes.abort()
			return nil, err
		}
	}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	return pipes, nil
}

// attach returns the writer the child should use for dest, creating a pipe
// when dest is not already an *os.File.
func (p *outputPipes) attach(dest io.Writer) (io.Writer, error) {
	if dest == nil {
		return nil, nil
	}
	if _, ok := dest.(*os.File); ok {
		return dest, nil
	}
	r, w, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("creating output pipe: %w", err)
	}
	p.readers = append(p.readers, r)
	p.writers = append(p.writers, w)
	p.dests = append(p.dests, dest)
	return w, nil
}

// started closes the parent's write ends and begins copying child output.
func (p *outputPipes) started() {
	closeFiles(p.writers)
	for i, r := range p.readers {
		p.copies.Add(1)
		go func(r *os.File, dest io.Writer) {
			defer p.copies.Done()
			_, _ = io.Copy(dest, r)
		}(r, p.dests[i])
	}
}

// drain waits for output copying to finish, force-closing the read ends if an
// escaped descendant keeps the pipes open past timeout.
func (p *outputPipes) drain(timeout time.Duration) {
	finished := make(chan struct{})
	go func() {
		p.copies.Wait()
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(timeout):
		closeFiles(p.readers)
		<-finished
	}
	closeFiles(p.readers)
}

func (p *outputPipes) abort() {
	closeFiles(p.writers)
	closeFiles(p.readers)
}

func closeFiles(files []*os.File) {
	for _, f := range files {
		_ = f.Close()
	}
}

// sameWriter reports whether a and b are the same writer, tolerating
// uncomparable dynamic types.
func sameWriter(a, b io.Writer) (same bool) {
	defer func() {
		if recover() != nil {
			same = false
		}
	}()
	return a != nil && a == b
}
