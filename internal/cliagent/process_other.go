//go:build !unix

package cliagent

import (
	"context"
	"os/exec"
)

// runIsolatedProcess falls back to direct-child handling on platforms without
// unix process groups.
func runIsolatedProcess(ctx context.Context, cmd *exec.Cmd) error {
	return runDirectProcess(ctx, cmd)
}
