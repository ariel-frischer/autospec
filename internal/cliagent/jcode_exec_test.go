package cliagent_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/cliagent"
	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/stretchr/testify/require"
)

type jcodeExecSnapshot struct {
	path     string
	args     []string
	dir      string
	env      map[string]string
	log      string
	exitCode int
}

func TestJcodeExec_SDKSessionControlsDoNotChangeInvocation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is not supported on Windows")
	}

	binDir := installExecFixture(t)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	workDir := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "jcode.log")
	baseline := captureExecSnapshot(t, config.JcodeConfig{Runner: config.JcodeRunnerExec}, workDir, logPath)

	tests := sdkControlCases()
	for name, controls := range tests {
		t.Run(name, func(t *testing.T) {
			controls.Runner = config.JcodeRunnerExec
			got := captureExecSnapshot(t, controls, workDir, logPath)
			require.Equal(t, baseline, got)
		})
	}
}

func sdkControlCases() map[string]config.JcodeConfig {
	return map[string]config.JcodeConfig{
		"session profile": {SessionProfile: "bounded"},
		"maximum turns":   {MaxTurns: 7},
		"token budget":    {TokenBudget: 4096},
		"deadline":        {Deadline: "2026-08-23T08:00:00Z"},
		"all settings": {
			SessionProfile: "bounded", MaxTurns: 7, TokenBudget: 4096,
			Deadline: "2026-08-23T08:00:00Z",
		},
	}
}

func captureExecSnapshot(t *testing.T, jcodeConfig config.JcodeConfig, workDir, logPath string) jcodeExecSnapshot {
	t.Helper()
	require.NoError(t, os.RemoveAll(logPath))
	agent, err := (&config.Configuration{AgentPreset: "jcode", Jcode: jcodeConfig}).GetAgent()
	require.NoError(t, err)
	execAgent, ok := agent.(*cliagent.JcodeExec)
	require.True(t, ok)
	opts := cliagent.ExecOptions{WorkDir: workDir, Env: map[string]string{
		"JCODE_FIXTURE_ENV": "isolated", "JCODE_FIXTURE_LOG": logPath,
	}}
	cmd, err := execAgent.BuildCommand("fixture prompt", opts)
	require.NoError(t, err)
	result, err := execAgent.Execute(context.Background(), "fixture prompt", opts)
	require.NoError(t, err)
	log, err := os.ReadFile(logPath)
	require.NoError(t, err)
	return jcodeExecSnapshot{
		path: cmd.Path, args: cmd.Args, dir: cmd.Dir,
		env: selectedExecEnv(cmd.Env), log: string(log), exitCode: result.ExitCode,
	}
}

func selectedExecEnv(env []string) map[string]string {
	selected := make(map[string]string)
	for _, entry := range env {
		for _, key := range []string{"JCODE_FIXTURE_ENV", "JCODE_FIXTURE_LOG"} {
			if strings.HasPrefix(entry, key+"=") {
				selected[key] = strings.TrimPrefix(entry, key+"=")
			}
		}
	}
	return selected
}

func installExecFixture(t *testing.T) string {
	t.Helper()
	binDir := t.TempDir()
	fixture := filepath.Join("testdata", "fake-jcode.sh")
	data, err := os.ReadFile(fixture)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "jcode"), data, 0o755))
	return binDir
}
