# Manual Validation Plan: jcode exec default

## Scope

Validate the production-compatible `jcode` command runner, explicit custom binary
selection, and explicit native SDK compatibility without invoking costly provider
operations by default.

## Manual Checks

1. Build Autospec with `make build` and confirm the binary is produced.
2. In a disposable repository, configure `agent_preset: jcode` with no
   `jcode.runner` and run a harmless workflow using a fixture `jcode` executable.
   Confirm Autospec invokes `jcode run --quiet` with the rendered prompt as the
   positional message and forwards the model when configured.
3. Configure `jcode.runner: custom` and `jcode.binary` to a fixture executable.
   Confirm the selected executable is used exactly and a missing executable
   produces an actionable error.
4. Configure `jcode.runner: sdk` only with an authorized compatible native
   runtime. Confirm SDK lifecycle behavior remains available and is not selected
   when runner is omitted.
5. Run `autospec config keys` and `autospec config show` to confirm the runner,
   binary, mode, and default values are documented consistently.
6. Confirm a non-zero fixture runner exit is surfaced as an execution error and
   does not silently fall back to another runner.

## Report Summaries

- Automated focused tests: passed for `internal/config`, `internal/cliagent`,
  and `internal/workflow`.
- `make fmt`: passed.
- `make lint`: passed.
- `make build`: passed.
- `make changelog-sync`: passed.
- Full `make test`: blocked by pre-existing
  `internal/cli/TestInitGitForNewFeature_SkipFetch` failures caused by unrelated
  process-wide working-directory test interference.
- Live provider or native SDK smoke test: not run. No external runtime or
  provider credentials were used.
