# Manual Testing Plan: 129-unify-model-selection

## Scope

Validate generic workflow model selection for Claude, Codex, and OpenCode.

## Smoke Steps

1. Build the updated binary with `make build`.
2. Run a dry workflow command with Claude and a generic model:
   `autospec plan --agent claude --model claude-opus-4-5-20251101 --skip-preflight`.
3. Run a dry workflow command with Codex and a generic model:
   `autospec plan --agent codex --model gpt-5.4 --skip-preflight`.
4. Run a dry workflow command with OpenCode and a generic model:
   `autospec plan --agent opencode --model anthropic/claude-sonnet-4-20250514 --skip-preflight`.
5. Confirm each command renders or executes the selected agent invocation with `--model <value>`.

## OpenCode Steps

1. Configure top-level `model`, run an OpenCode workflow without model flags, and confirm it is selected.
2. Confirm OpenCode workflow help exposes `--model` and does not expose any agent-specific model flag.
3. Confirm `autospec config keys` exposes top-level `model` and no OpenCode-specific model keys.

## Report Summaries

- Claude generic model:
- Codex generic model:
- OpenCode generic model:
- OpenCode generic model:
- Removed OpenCode-specific model surface:
