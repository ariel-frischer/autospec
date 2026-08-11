package cliagent

import (
	"context"
	"fmt"
	"os/exec"
)

const defaultJcodeExecBinary = "jcode"

// JcodeExec implements the production jcode CLI integration.
type JcodeExec struct {
	base   BaseAgent
	binary string
}

// NewJcodeExec creates a command-backed jcode agent. An empty binary uses jcode
// from PATH, while a non-empty value is used exactly as configured.
func NewJcodeExec(binary string) *JcodeExec {
	if binary == "" {
		binary = defaultJcodeExecBinary
	}
	return &JcodeExec{
		binary: binary,
		base: BaseAgent{
			AgentName:   "jcode",
			Cmd:         binary,
			VersionFlag: "--version",
			AgentCaps: Caps{
				Automatable: true,
				PromptDelivery: PromptDelivery{
					Method: PromptMethodPositional,
				},
			},
		},
	}
}

func (j *JcodeExec) Name() string { return j.base.Name() }

func (j *JcodeExec) Version() (string, error) { return j.base.Version() }

func (j *JcodeExec) Validate() error { return j.base.Validate() }

func (j *JcodeExec) Capabilities() Caps { return j.base.Capabilities() }

// BuildCommand constructs the jcode exec CLI contract.
func (j *JcodeExec) BuildCommand(prompt string, opts ExecOptions) (*exec.Cmd, error) {
	if _, err := exec.LookPath(j.binary); err != nil {
		return nil, fmt.Errorf("jcode executable %q not found: %w", j.binary, err)
	}

	args := []string{"run", "--quiet"}
	model, reasoningEffort := sessionSettings(opts)
	if model != "" {
		args = append(args, "--model", model)
	}
	if reasoningEffort != "" {
		args = append(args, "--reasoning-effort", reasoningEffort)
	}
	args = append(args, sanitizePromptForCLI(prompt))
	args = append(args, withoutSessionSettings(opts.ExtraArgs)...)

	cmd := exec.Command(j.binary, args...)
	j.base.configureCmd(cmd, opts)
	return cmd, nil
}

func sessionSettings(opts ExecOptions) (string, string) {
	model, effort := ParseSessionSettings(opts.ExtraArgs)
	if opts.Model != "" {
		model = opts.Model
	}
	if opts.ReasoningEffort != "" {
		effort = opts.ReasoningEffort
	}
	return model, effort
}

func withoutSessionSettings(args []string) []string {
	filtered := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--model" && i+1 < len(args) {
			i++
			continue
		}
		if args[i] == "-c" && i+1 < len(args) {
			_, reasoningEffort := ParseSessionSettings(args[i : i+2])
			if reasoningEffort != "" {
				i++
				continue
			}
		}
		filtered = append(filtered, args[i])
	}
	return filtered
}

// Execute runs jcode and treats a non-zero CLI exit as a contextual error.
func (j *JcodeExec) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Result, error) {
	cmd, err := j.BuildCommand(prompt, opts)
	if err != nil {
		return nil, fmt.Errorf("building jcode exec command: %w", err)
	}
	result, err := j.base.runCommand(ctx, cmd, opts)
	if err != nil {
		return nil, fmt.Errorf("executing jcode exec: %w", err)
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("executing jcode exec: exit status %d: %s", result.ExitCode, result.Stderr)
	}
	return result, nil
}

var _ Agent = (*JcodeExec)(nil)
