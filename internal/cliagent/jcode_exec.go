package cliagent

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
)

const defaultJcodeExecBinary = "jcode"

// JcodeExec implements the production jcode CLI integration.
type JcodeExec struct {
	base    BaseAgent
	binary  string
	options JcodeExecOptions
}

// JcodeExecOptions contains the stable upstream jcode wrapper flags supported
// by autospec. Values are passed as argv and never through a shell.
type JcodeExecOptions struct {
	Provider               string
	ProviderProfile        string
	SocketPath             string
	Trace                  bool
	ToolProfile            string
	Tools                  string
	DisabledTools          string
	DisableBaseTools       bool
	MCPTools               string
	MCPToolsTokenThreshold int
}

// NewJcodeExec creates a command-backed jcode agent. An empty binary uses jcode
// from PATH, while a non-empty value is used exactly as configured.
func NewJcodeExec(binary string) *JcodeExec {
	return NewJcodeExecWithOptions(binary, JcodeExecOptions{})
}

// NewJcodeExecWithOptions creates a command-backed jcode agent with stable
// upstream wrapper options.
func NewJcodeExecWithOptions(binary string, options JcodeExecOptions) *JcodeExec {
	if binary == "" {
		binary = defaultJcodeExecBinary
	}
	return &JcodeExec{
		binary:  binary,
		options: options,
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

	args := []string{"--quiet", "--no-update", "--no-selfdev"}
	args = append(args, j.globalArgs()...)
	model, _ := sessionSettings(opts)
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, "run", sanitizePromptForCLI(prompt))

	cmd := exec.Command(j.binary, args...)
	j.base.configureCmd(cmd, opts)
	return cmd, nil
}

func (j *JcodeExec) globalArgs() []string {
	args := make([]string, 0, 20)
	args = appendStringFlag(args, "--provider", j.options.Provider)
	args = appendStringFlag(args, "--provider-profile", j.options.ProviderProfile)
	args = appendStringFlag(args, "--socket", j.options.SocketPath)
	if j.options.Trace {
		args = append(args, "--trace")
	}
	args = appendStringFlag(args, "--tool-profile", j.options.ToolProfile)
	args = appendStringFlag(args, "--tools", j.options.Tools)
	args = appendStringFlag(args, "--disabled-tools", j.options.DisabledTools)
	if j.options.DisableBaseTools {
		args = append(args, "--disable-base-tools")
	}
	args = appendStringFlag(args, "--mcp-tools", j.options.MCPTools)
	if j.options.MCPToolsTokenThreshold > 0 {
		args = append(args, "--mcp-tools-token-threshold", strconv.Itoa(j.options.MCPToolsTokenThreshold))
	}
	return args
}

func appendStringFlag(args []string, flag, value string) []string {
	if value == "" {
		return args
	}
	return append(args, flag, value)
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
