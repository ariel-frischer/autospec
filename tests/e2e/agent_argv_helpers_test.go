//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

const argLinePrefix = `  - "`

// agentInvocation is one mock agent call recorded in MOCK_CALL_LOG.
type agentInvocation struct {
	agent string
	args  []string
}

// readAgentInvocations parses the mock call log written by the mock agent
// scripts. Multi-line arguments (rendered prompts) are rejoined with newlines.
func readAgentInvocations(t *testing.T, logPath string) []agentInvocation {
	t.Helper()
	data, err := os.ReadFile(logPath)
	require.NoError(t, err, "reading mock call log")

	var calls []agentInvocation
	inArgs := false
	for _, line := range strings.Split(string(data), "\n") {
		switch {
		case line == "---":
			calls = append(calls, agentInvocation{})
			inArgs = false
		case len(calls) == 0:
		case strings.HasPrefix(line, "agent: "):
			calls[len(calls)-1].agent = strings.TrimPrefix(line, "agent: ")
		case line == "args:":
			inArgs = true
		case inArgs && strings.HasPrefix(line, argLinePrefix):
			calls[len(calls)-1].args = append(calls[len(calls)-1].args, strings.TrimPrefix(line, argLinePrefix))
		case inArgs && len(calls[len(calls)-1].args) > 0:
			args := calls[len(calls)-1].args
			args[len(args)-1] += "\n" + line
		}
	}
	for i := range calls {
		for j, arg := range calls[i].args {
			calls[i].args[j] = strings.TrimSuffix(strings.TrimRight(arg, "\n"), `"`)
		}
	}
	return calls
}

// singleAgentInvocation returns the only call recorded for agent.
func singleAgentInvocation(t *testing.T, logPath, agent string) agentInvocation {
	t.Helper()
	var matches []agentInvocation
	for _, call := range readAgentInvocations(t, logPath) {
		if call.agent == agent {
			matches = append(matches, call)
		}
	}
	require.Len(t, matches, 1, "expected exactly one %s invocation", agent)
	return matches[0]
}

// flagValue returns the argument following flag and its index, or -1.
func flagValue(args []string, flag string) (string, int) {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag {
			return args[i+1], i
		}
	}
	return "", -1
}

// writeProjectConfig replaces the project config with the given YAML body.
func writeProjectConfig(t *testing.T, env *testutil.E2EEnv, body string) {
	t.Helper()
	path := filepath.Join(env.TempDir(), ".autospec", "config.yml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
}
