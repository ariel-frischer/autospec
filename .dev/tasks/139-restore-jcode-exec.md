# Manual testing plan: official Jcode exec and opt-in SDK

## Scope

Validate the built Autospec binary against an isolated temporary repository. Do not use a production repository, shared Jcode daemon, or paid provider session unless separately approved.

## Official CLI exec mode

1. Install or select the official `1jehuang/jcode` binary on an isolated `PATH`.
2. Run `autospec init --project --ai jcode` and confirm `.autospec/config.yml` records `agent_preset: jcode` and `default_agents: ["jcode"]`.
3. Leave `jcode.runner` unset, invoke a short workflow through an argv-capturing wrapper, and verify the command shape is `jcode --quiet --no-update --no-selfdev [globals] run <one prompt>`.
4. Configure provider, provider profile, socket, trace, tool profile, tools, disabled tools, base-tool disabling, MCP mode, MCP threshold, and a model. Confirm every configured global precedes `run` and zero values are omitted.
5. Set generic reasoning effort and agent extra arguments. Confirm neither becomes an exec argument.
6. Set `jcode.binary` while leaving `runner: exec`. Confirm Autospec still resolves `jcode` from `PATH`.
7. Set `runner: custom` and a temporary executable. Confirm the custom executable is used.
8. Remove the official binary from the isolated `PATH`. Confirm the failure names the missing `jcode` executable and does not fall back to SDK or another agent.

## SDK opt-in mode

1. Configure `jcode.runner: sdk` only in an isolated configuration.
2. Exercise connect mode against a disposable runtime and confirm Autospec never stops the shared runtime.
3. Exercise private mode with temporary home and socket directories. Confirm the run-owned runtime is stopped and temporary state is removed.
4. Cancel a turn and confirm cancellation is bounded and sent once.
5. Simulate a provider error and transport disconnect. Confirm provider codes remain visible, terminal failure is returned, and no mid-turn reconnect occurs.

## Dependency checks

1. Build the site with the locked `json` 2.19.9 gem.
2. Exercise repository clone, worktree, and reference-validation paths with go-git 5.19.2.
3. Confirm no release, tag, remote push, or live production validation occurs.

## Report Summaries

- Official exec mode:
- Init selection:
- SDK opt-in and ownership:
- Dependency behavior:
- Cleanup and safety:
- Remaining issues:
