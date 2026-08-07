package cliagent

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func jcodeCommandFixturePath(t *testing.T, name string) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	require.True(t, ok, "locate cliagent test source")
	return filepath.Join(filepath.Dir(source), "testdata", name)
}

func copyJcodeFixtureCommand(t *testing.T, name string) string {
	t.Helper()
	destination := filepath.Join(t.TempDir(), "jcode")
	data, err := os.ReadFile(jcodeCommandFixturePath(t, name))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(destination, data, 0o700))
	return destination
}

func inspectJcodeFixtureCommand(t *testing.T, command string, options ExecOptions) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(command, "exec", "--prompt", "fixture prompt")
	cmd.Dir = options.WorkDir
	cmd.Env = append(os.Environ(), "JCODE_FIXTURE_ENV=isolated")
	for key, value := range options.Env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	return cmd
}
