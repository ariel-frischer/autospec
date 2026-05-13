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
$AUTOSPEC_BIN init --project --ai codex --no-constitution
```

Expected result:

- `.autospec/config.yml` records Codex in `default_agents`.
- `agent_preset: codex` is set when Codex is the only selected agent.
- `.codex/config.toml` is created or updated with safe project metadata only.
- No `.claude/commands/` directory is created for Codex-only setup.
- No `.opencode/command/` directory is created for Codex-only setup.

Check the generated files:

```bash
cat .autospec/config.yml
cat .codex/config.toml
find . -maxdepth 3 -type f | sort
```

`.codex/config.toml` must not force destructive defaults such as disabled approvals or unrestricted sandboxing. Those choices belong to the user, Codex config, or explicit autospec autonomous mode.

## Prompt Delivery With A Fake Codex CLI

Use a fake Codex binary to verify invocation shape without spending tokens or depending on live model behavior.

Create the shim in a temporary directory:

```bash
shimdir=$(mktemp -d)
cat > "$shimdir/codex" <<'SH'
#!/bin/sh
printf '%s\n' "$@" > "$PWD/codex-args.log"
exit 0
SH
chmod +x "$shimdir/codex"
```

Configure retries to avoid repeated fake-agent runs:

```bash
$AUTOSPEC_BIN config set max_retries 0 --project
```

Create a minimal constitution so `specify` reaches the agent invocation:

```bash
mkdir -p .autospec
cat > .autospec/constitution.yaml <<'YAML'
constitution:
  project_name: "codex-manual-test"
  version: "1.0.0"
  ratified: "2026-01-01"
  last_amended: "2026-01-01"
preamble: "Manual Codex testing constitution."
principles:
  - name: "Test-First Development"
    id: "PRIN-001"
    category: "quality"
    priority: "NON-NEGOTIABLE"
    description: "Tests come before implementation."
    rationale: "Keeps workflow output verifiable."
    enforcement: []
    exceptions: []
sections: []
governance:
  amendment_process: []
  versioning_policy: "Semantic versioning"
  compliance_review:
    frequency: "as needed"
    process: "Manual review"
  rules: []
sync_impact:
  version_change: "1.0.0 -> 1.0.0"
  modified_principles: []
  added_sections: []
  removed_sections: []
  templates_requiring_updates: []
  follow_up_todos: []
_meta:
  version: "1.0.0"
  generator: "manual"
  generator_version: "manual"
  created: "2026-01-01T00:00:00Z"
  artifact_type: "constitution"
YAML
```

Run a stage with the shim first on `PATH`:

```bash
PATH="$shimdir:$PATH" $AUTOSPEC_BIN specify --agent codex "Add a sample feature" || true
cat codex-args.log
```

Expected result:

- The first logged argument is `exec`.
- The next argument is rendered prompt text.
- The prompt text does not start with `/autospec.specify`.
- The prompt includes the user feature description.

The command may fail validation because the fake Codex CLI does not create `spec.yaml`; that is acceptable for this check.

Repeat the same shape check for constitution:

```bash
PATH="$shimdir:$PATH" $AUTOSPEC_BIN constitution --agent codex "Use test-first development" || true
cat codex-args.log
```

Expected result:

- The first logged argument is `exec`.
- The prompt is rendered text, not `/autospec.constitution`.
- The prompt includes the constitution input.

## Autonomous Mode Mapping

In the disposable repository, enable autonomous mode:

```bash
$AUTOSPEC_BIN config set skip_permissions true --project
PATH="$shimdir:$PATH" $AUTOSPEC_BIN specify --agent codex "Add another sample feature" || true
cat codex-args.log
```

Expected result:

- The logged command includes `exec`.
- The logged command includes `--dangerously-bypass-approvals-and-sandbox`.
- The rendered prompt is still passed as the final prompt argument.

Disable autonomous mode and confirm the flag is removed:

```bash
$AUTOSPEC_BIN config set skip_permissions false --project
PATH="$shimdir:$PATH" $AUTOSPEC_BIN specify --agent codex "Add a safe sample feature" || true
cat codex-args.log
```

Expected result:

- `--dangerously-bypass-approvals-and-sandbox` is absent.
- Codex still receives `exec` plus rendered prompt text.

## Real Codex Smoke Test

Run this only in a disposable repository where Codex is allowed to modify files.

```bash
$AUTOSPEC_BIN init --project --ai codex
$AUTOSPEC_BIN doctor
$AUTOSPEC_BIN specify --agent codex "Add a small README note"
```

Expected result:

- A numbered feature directory is created under `specs/`.
- `spec.yaml` is generated and passes autospec validation.
- Codex auth is handled by the Codex CLI, not by autospec env validation.

Continue through planning if the specify result is acceptable:

```bash
$AUTOSPEC_BIN plan --agent codex
$AUTOSPEC_BIN tasks --agent codex
```

Expected result:

- `plan.yaml` is generated from rendered prompt text.
- `tasks.yaml` is generated from rendered prompt text.
- Existing Claude and OpenCode command-template directories are not required for Codex.

## Claude And OpenCode Regression Checks

Codex changes should not alter the other supported agents.

In separate disposable repositories:

```bash
$AUTOSPEC_BIN init --project --ai claude --no-constitution
find . -maxdepth 4 -type f | sort
```

Expected result:

- Claude command templates are installed under `.claude/commands/`.
- Claude settings behavior remains unchanged.

```bash
$AUTOSPEC_BIN init --project --ai opencode --no-constitution
find . -maxdepth 4 -type f | sort
```

Expected result:

- OpenCode command files are installed under `.opencode/command/`.
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
Fake CLI prompt delivery:
Autonomous mode:
Real Codex smoke:
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
