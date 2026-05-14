# Codex Manual Testing

Use this checklist when changing Codex support in autospec. It verifies the user-facing behavior that unit tests cannot fully cover: installed CLI discovery, prompt delivery, project initialization, and safe autonomous-mode handling.

Run these checks in a disposable repository or worktree. Do not use a production project for yolo-mode testing.

## Prerequisites

- `make build` succeeds in the autospec repository.
- `codex --version` works on the machine used for real smoke tests.
- Codex is authenticated through ChatGPT login or API credentials managed by Codex itself.
- The test repository has git initialized.

`OPENAI_API_KEY` is optional for Codex. autospec must not fail health checks only because that variable is unset.

## Build The Local Binary

From the autospec repository:

```bash
make build
```

Use the built binary for the commands below:

```bash
AUTOSPEC_BIN=$PWD/bin/autospec
```

## Doctor Check

Verify Codex is treated as a supported production agent:

```bash
unset OPENAI_API_KEY
$AUTOSPEC_BIN doctor
```

Expected result:

- Codex appears with the supported CLI agents when installed.
- Missing `OPENAI_API_KEY` is not reported as a Codex failure.
- The displayed tested version is the current smoke-tested Codex version recorded in `internal/build/version.go`.

## Project Init

Create a disposable repository:

```bash
tmpdir=$(mktemp -d)
cd "$tmpdir"
git init
```

Initialize autospec for Codex only:

```bash
$AUTOSPEC_BIN init --project --ai codex --skip-permissions --no-constitution
```

Expected result:

- `.autospec/config.yml` records Codex in `default_agents`.
- `agent_preset: codex` is set when Codex is the only selected agent.
- `skip_permissions: true` is set for this disposable test repo.
- `.codex/config.toml` is created or updated with safe project metadata and `skills.config`.
- `.agents/skills/autospec-*/SKILL.md` files are installed for interactive Codex skills.
- No `.claude/commands/` directory is created for Codex-only setup.
- No `.opencode/command/` directory is created for Codex-only setup.

Check the generated files:

```bash
cat .autospec/config.yml
cat .codex/config.toml
cat .agents/skills/autospec-specify/SKILL.md
cat .agents/skills/autospec-clarify/SKILL.md
find . -maxdepth 4 -type f | sort
```

`.codex/config.toml` must not force destructive defaults such as disabled approvals or unrestricted sandboxing. For this manual test, write access comes from autospec `skip_permissions: true`, which maps to Codex `--dangerously-bypass-approvals-and-sandbox` at runtime.

The Codex skills should contain the actual prompt instructions from `internal/commands/autospec.*.md`. For example, `$autospec-specify "desc"` should load the specify prompt and `$autospec-clarify` should load the clarify prompt.

## Real Codex Smoke Test

Run this only in a disposable repository where Codex is allowed to modify files. This test must use the real `codex` binary.

```bash
$AUTOSPEC_BIN doctor
$AUTOSPEC_BIN constitution "Use test-first development, small focused changes, and artifact validation."
```

Expected result:

- `autospec doctor` reports Codex as installed.
- The constitution command runs through `codex exec`.
- `.autospec/constitution.yaml` is generated and passes autospec validation.
- Codex auth is handled by the Codex CLI, not by autospec env validation.

Generate the first workflow artifact:

```bash
$AUTOSPEC_BIN specify "Add a small README note that says this repository supports autospec Codex smoke testing"
```

Expected result:

- A numbered feature directory is created under `specs/`.
- `spec.yaml` is generated and passes autospec validation.
- The generated spec input references the README-note smoke-test request.

Continue through planning if the specify result is acceptable:

```bash
$AUTOSPEC_BIN plan
$AUTOSPEC_BIN tasks
```

Expected result:

- `plan.yaml` is generated from rendered prompt text.
- `tasks.yaml` is generated from rendered prompt text.
- Existing Claude and OpenCode command-template directories are not required for Codex.

## Prompt Delivery Observation

Run one stage with debug output enabled:

```bash
$AUTOSPEC_BIN --debug specify "Add another tiny README note for Codex prompt delivery observation"
```

Expected result:

- autospec reports `codex exec` as the command being executed.
- The visible execution preview begins with rendered prompt content, such as `## User Input`, not a raw `/autospec.*` slash command.
- The generated `spec.yaml` contains the user request.

This is a manual observation check. The source of truth remains the generated and validated artifact, not parsing terminal text.

## Autonomous Mode Mapping

Run this only in a disposable repository. Enabling `skip_permissions` maps autospec autonomous mode to Codex yolo mode.

```bash
$AUTOSPEC_BIN config set skip_permissions true --project
$AUTOSPEC_BIN --debug specify "Add a tiny README note while testing Codex autonomous mode"
```

Expected result:

- autospec reports `codex exec --dangerously-bypass-approvals-and-sandbox`.
- Codex runs without prompting for approvals.
- A valid `spec.yaml` is generated.

Disable autonomous mode before continuing normal manual tests:

```bash
$AUTOSPEC_BIN config set skip_permissions false --project
```

Run another debug stage and confirm the yolo flag is gone:

```bash
$AUTOSPEC_BIN --debug specify "Add a tiny README note while testing normal Codex mode"
```

Expected result:

- autospec reports `codex exec` without `--dangerously-bypass-approvals-and-sandbox`.
- A valid `spec.yaml` is generated.

## Codex Output Observation

Codex `exec` prints formatted terminal output by default. For manual inspection of Codex's machine-readable stream, run Codex directly in the disposable repository:

```bash
codex exec --json "Summarize this disposable autospec test repository without editing files" | tee codex-output.jsonl
```

Expected result:

- `codex-output.jsonl` contains one JSON object per line.
- Events include turn lifecycle records and agent-message records.
- This output is observational only; autospec currently validates generated workflow artifacts rather than parsing Codex JSONL.

## Claude And OpenCode Regression Checks

Codex changes should not alter the other supported agents.

In separate disposable repositories:

```bash
$AUTOSPEC_BIN init --project --ai claude --no-constitution
find . -maxdepth 4 -type f | sort
```

Expected result:

- Claude skills are installed under `.claude/skills/autospec.*/`.
- Claude settings behavior remains unchanged.

```bash
$AUTOSPEC_BIN init --project --ai opencode --no-constitution
find . -maxdepth 4 -type f | sort
```

Expected result:

- OpenCode command files are installed under `.opencode/command/`.
- Shared autospec skills are installed under `.agents/skills/`.
- OpenCode automation still relies on its run mode and configured permissions.

## Report Summaries

Record each manual run here when validating a Codex change:

```text
Date:
Tester:
autospec commit:
Codex version:
OS:

Doctor:
Init:
Real Codex smoke:
Prompt delivery observation:
Autonomous mode:
Codex JSONL observation:
Claude regression:
OpenCode regression:

Notes:
Follow-ups:
```

## Failure Triage

- If doctor reports missing `OPENAI_API_KEY` for Codex, check `internal/cliagent/codex.go` and health dependency handling.
- If Codex receives `/autospec.*` text, check prompt rendering in `internal/workflow/stage_executor.go`.
- If yolo mode does not pass `--dangerously-bypass-approvals-and-sandbox`, check the Codex agent definition and `skip_permissions` command construction.
- If `init --ai codex` creates Claude or OpenCode command directories, check init agent-selection and configurator dispatch.
- If `.codex/config.toml` contains aggressive sandbox or approval defaults, remove them; autospec should only write safe, discoverable project metadata.
