package workflow

import (
	"context"
	"os/exec"
	"testing"

	"github.com/ariel-frischer/autospec/internal/cliagent"
	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/stretchr/testify/require"
)

type jcodeWorkflowCapture struct {
	opts cliagent.ExecOptions
}

func (c *jcodeWorkflowCapture) Name() string                { return "jcode" }
func (c *jcodeWorkflowCapture) Version() (string, error)    { return "test", nil }
func (c *jcodeWorkflowCapture) Validate() error             { return nil }
func (c *jcodeWorkflowCapture) Capabilities() cliagent.Caps { return cliagent.Caps{Automatable: true} }
func (c *jcodeWorkflowCapture) BuildCommand(string, cliagent.ExecOptions) (*exec.Cmd, error) {
	return exec.Command("true"), nil
}
func (c *jcodeWorkflowCapture) Execute(_ context.Context, _ string, opts cliagent.ExecOptions) (*cliagent.Result, error) {
	c.opts = opts
	return &cliagent.Result{ExitCode: 0}, nil
}

func TestJcodeWorkflow_PropagatesModelAndReasoningOptions(t *testing.T) {
	t.Parallel()

	agent := &jcodeWorkflowCapture{}
	executor := &Executor{
		Claude: &ClaudeExecutor{Agent: agent},
		Config: config.Configuration{
			Model:           "openai/gpt-6-luna",
			ReasoningEffort: "high",
		},
	}

	require.NoError(t, executor.execute("prompt", StagePlan))
	require.Equal(t, "openai/gpt-6-luna", agent.opts.Model)
	require.Equal(t, "high", agent.opts.ReasoningEffort)
	require.Equal(t, []string{"--model", "openai/gpt-6-luna", "-c", "model_reasoning_effort=high"}, agent.opts.ExtraArgs)
}
