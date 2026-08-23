# Jcode SDK Session Controls Smoke-Test Plan

## Purpose and safety boundary

This plan verifies the experimental `jcode.runner: sdk` session controls in a disposable repository:

- `jcode.session_profile`
- `jcode.max_turns`
- `jcode.token_budget`
- `jcode.deadline`

Do not run the live SDK steps by default. They may start or connect to a Jcode runtime and invoke an agent. Obtain explicit approval before any live, costly, credentialed, or production-adjacent execution. Never run this plan from the active Autospec development repository.

## Preconditions

- Use an Autospec binary built from the feature branch under review.
- Use a Jcode runtime compatible with `github.com/ariel-frischer/jcode-go` v0.1.6.
- Keep credentials outside configuration files and command output.
- Prefer an isolated private runtime. If connect mode is required, use only a disposable test runtime and socket.

## Disposable environment setup

Run setup from a neutral directory, not from the Autospec checkout:

```bash
export SMOKE_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/autospec-jcode-sdk-smoke.XXXXXX")"
export SMOKE_HOME="$SMOKE_ROOT/home"
export SMOKE_REPO="$SMOKE_ROOT/repo"
export SMOKE_CONFIG_DIR="$SMOKE_ROOT/configs"
export SMOKE_REPORT="$SMOKE_ROOT/report.txt"
mkdir -p "$SMOKE_HOME" "$SMOKE_REPO" "$SMOKE_CONFIG_DIR"
git -C "$SMOKE_REPO" init
printf '# Disposable Autospec Jcode SDK smoke repository\n' > "$SMOKE_REPO/README.md"
git -C "$SMOKE_REPO" add README.md
git -C "$SMOKE_REPO" -c user.name='Autospec Smoke' -c user.email='smoke@example.invalid' commit -m 'test: initialize disposable smoke repository'
printf 'SMOKE_ROOT=%s\nSMOKE_REPO=%s\n' "$SMOKE_ROOT" "$SMOKE_REPO" | tee "$SMOKE_REPORT"
```

For every command below, set `HOME="$SMOKE_HOME"` and pass an explicit `--config` path. Do not reuse the developer's home, project configuration, repository, socket, or runtime state.

## Base SDK configuration

Create a base configuration for an isolated private runtime. Set `binary` only when the test Jcode executable is not already on `PATH`.

```bash
cat > "$SMOKE_CONFIG_DIR/sdk-base.yml" <<EOF
agent_preset: jcode
jcode:
  runner: sdk
  mode: private
  home: "$SMOKE_ROOT/jcode-home"
  inherit_logins: false
  startup_timeout: 30s
  cleanup_timeout: 30s
EOF
HOME="$SMOKE_HOME" autospec --config "$SMOKE_CONFIG_DIR/sdk-base.yml" config show >> "$SMOKE_REPORT" 2>&1
```

The non-live `config show` check must load successfully and must report `runner: sdk` without adding an implicit session profile, turn limit, token limit, or deadline.

## Valid single-control cases

Create and inspect four independent configurations. Each file must explicitly select `runner: sdk` and configure only the named control in addition to the isolated runtime settings.

```bash
cp "$SMOKE_CONFIG_DIR/sdk-base.yml" "$SMOKE_CONFIG_DIR/profile.yml"
printf '  session_profile: bounded\n' >> "$SMOKE_CONFIG_DIR/profile.yml"

cp "$SMOKE_CONFIG_DIR/sdk-base.yml" "$SMOKE_CONFIG_DIR/max-turns.yml"
printf '  max_turns: 8\n' >> "$SMOKE_CONFIG_DIR/max-turns.yml"

cp "$SMOKE_CONFIG_DIR/sdk-base.yml" "$SMOKE_CONFIG_DIR/token-budget.yml"
printf '  token_budget: 16000\n' >> "$SMOKE_CONFIG_DIR/token-budget.yml"

cp "$SMOKE_CONFIG_DIR/sdk-base.yml" "$SMOKE_CONFIG_DIR/deadline.yml"
printf '  deadline: "2035-08-23T10:00:00+02:00"\n' >> "$SMOKE_CONFIG_DIR/deadline.yml"

for config in profile max-turns token-budget deadline; do
  HOME="$SMOKE_HOME" autospec --config "$SMOKE_CONFIG_DIR/$config.yml" config show >> "$SMOKE_REPORT" 2>&1
 done
```

Expected non-live result: every configuration loads, the configured value is present, and the other three controls remain unset.

With explicit approval for live validation, execute one bounded workflow per configuration from `SMOKE_REPO`. Use a trivial disposable feature prompt and record the exit code and sanitized output. Verify runtime instrumentation or SDK debug output shows the configured value in the corresponding typed option, while omitted controls retain zero values.

```bash
cd "$SMOKE_REPO"
HOME="$SMOKE_HOME" autospec --config "$SMOKE_CONFIG_DIR/profile.yml" run -s "Create a disposable text file named profile-smoke.txt" -y
# Repeat separately for max-turns.yml, token-budget.yml, and deadline.yml only after approval.
```

## Valid all-controls case

```bash
cat > "$SMOKE_CONFIG_DIR/all-controls.yml" <<EOF
agent_preset: jcode
jcode:
  runner: sdk
  mode: private
  home: "$SMOKE_ROOT/jcode-home-all"
  inherit_logins: false
  session_profile: bounded
  max_turns: 8
  token_budget: 16000
  deadline: "2035-08-23T10:00:00+02:00"
EOF
HOME="$SMOKE_HOME" autospec --config "$SMOKE_CONFIG_DIR/all-controls.yml" config show >> "$SMOKE_REPORT" 2>&1
```

Expected non-live result: all four values load unchanged. With explicit live-test approval, run one disposable workflow and verify `CreateSessionOptions.Profile` receives `bounded`, while `SendOptions` receives 8 turns, 16000 tokens, and the exact deadline string.

## Invalid-control cases

Each command must fail during configuration loading, before session creation or agent execution. Record the nonzero exit code and verify the message names the key and corrective requirement.

```bash
cat > "$SMOKE_CONFIG_DIR/blank-profile.yml" <<'EOF'
agent_preset: jcode
jcode:
  runner: sdk
  session_profile: "   "
EOF

cat > "$SMOKE_CONFIG_DIR/zero-turns.yml" <<'EOF'
agent_preset: jcode
jcode:
  runner: sdk
  max_turns: -1
EOF

cat > "$SMOKE_CONFIG_DIR/zero-token-budget.yml" <<'EOF'
agent_preset: jcode
jcode:
  runner: sdk
  token_budget: -1
EOF

cat > "$SMOKE_CONFIG_DIR/offset-free-deadline.yml" <<'EOF'
agent_preset: jcode
jcode:
  runner: sdk
  deadline: "2035-08-23T08:00:00"
EOF

for config in blank-profile zero-turns zero-token-budget offset-free-deadline; do
  if HOME="$SMOKE_HOME" autospec --config "$SMOKE_CONFIG_DIR/$config.yml" config show >> "$SMOKE_REPORT" 2>&1; then
    printf 'UNEXPECTED PASS: %s\n' "$config" | tee -a "$SMOKE_REPORT"
  else
    printf 'EXPECTED REJECTION: %s\n' "$config" | tee -a "$SMOKE_REPORT"
  fi
done
```

Expected diagnostics:

- `jcode.session_profile` requires a non-blank value when set.
- `jcode.max_turns` and `jcode.token_budget` must be positive when set.
- `jcode.deadline` requires RFC3339 with `Z` or a signed numeric UTC offset.

Repeat integer cases with `0` only if the configuration source can distinguish an explicitly configured zero from omission. The runtime contract treats zero as unset.

## Past-deadline and SDK-error diagnostics

A syntactically valid past deadline is configuration-valid and must reach the SDK unchanged:

```bash
cat > "$SMOKE_CONFIG_DIR/past-deadline.yml" <<EOF
agent_preset: jcode
jcode:
  runner: sdk
  mode: private
  home: "$SMOKE_ROOT/jcode-home-past"
  inherit_logins: false
  deadline: "2020-01-01T00:00:00Z"
EOF
HOME="$SMOKE_HOME" autospec --config "$SMOKE_CONFIG_DIR/past-deadline.yml" config show >> "$SMOKE_REPORT" 2>&1
```

Expected non-live result: configuration loading succeeds. With explicit live-test approval, run a disposable workflow and verify any SDK rejection retains the original diagnostic and adds concise Autospec operation context, such as creating the Jcode session, starting the Jcode turn, or running the Jcode turn.

To exercise a representative session-creation or turn-start failure without credentials, point only the disposable configuration at a deliberately unavailable disposable test runtime or use a test-only Jcode fixture. Do not stop, alter, or impersonate a shared developer runtime. Verify the final error preserves the SDK cause and causal chain.

## Exec isolation

Create two official exec configurations that differ only by the four SDK-only keys:

```bash
cat > "$SMOKE_CONFIG_DIR/exec-base.yml" <<'EOF'
agent_preset: jcode
jcode:
  runner: exec
EOF

cat > "$SMOKE_CONFIG_DIR/exec-with-sdk-keys.yml" <<'EOF'
agent_preset: jcode
jcode:
  runner: exec
  session_profile: bounded
  max_turns: 8
  token_budget: 16000
  deadline: "2035-08-23T10:00:00+02:00"
EOF

HOME="$SMOKE_HOME" autospec --config "$SMOKE_CONFIG_DIR/exec-base.yml" config show >> "$SMOKE_REPORT" 2>&1
HOME="$SMOKE_HOME" autospec --config "$SMOKE_CONFIG_DIR/exec-with-sdk-keys.yml" config show >> "$SMOKE_REPORT" 2>&1
```

With explicit approval for live validation, run the same disposable prompt with a recording Jcode CLI fixture for both files. Compare the complete argv and sanitized output. They must be identical, and none of `bounded`, `8`, `16000`, the deadline, or derived SDK-only flags may appear. Also verify an omitted runner still resolves to `exec`.

## Cleanup

Review the report before removal and copy only sanitized findings needed for the Report Summaries below. Then delete all disposable state:

```bash
cd "${TMPDIR:-/tmp}"
rm -rf -- "$SMOKE_ROOT"
unset SMOKE_ROOT SMOKE_HOME SMOKE_REPO SMOKE_CONFIG_DIR SMOKE_REPORT
```

Confirm no file in the active Autospec repository changed during the smoke workflow. Do not delete or modify any shared Jcode runtime, socket, home, login, or credential state.

## Report Summaries

### Automated validation performed during implementation

- `go test ./internal/config ./internal/cliagent` passed on 2026-08-23: 1,052 tests passed across both packages.
- The passing package suites include typed create/send option mapping, omitted and all-control cases, validation averaging under 10ms, wrapped causal SDK errors, and complete official exec argv equality with all four SDK-only keys configured.
- `autospec artifact` passed for `spec.yaml`, `plan.yaml`, and `tasks.yaml` with no schema or dependency errors.
- `make fmt`, `make lint`, `make test`, and `make build` all passed in a disposable regular clone of final commit `855504f`, with `TMPDIR=/var/tmp/autospec-tests` keeping temporary repositories outside parent Git metadata.
- The built binary loaded all four controls unchanged from an isolated configuration and rejected negative `jcode.max_turns` plus an offset-free `jcode.deadline` with key-specific actionable diagnostics.
- `make changelog-check`, `make docs-sync`, `git diff --check`, and final worktree cleanliness checks passed; the public docs and separately maintained site configuration reference both expose the new SDK-only controls.
- The first worktree-local `make test` attempt exposed pre-existing environment assumptions: `TMPDIR` was nested inside `/home/ari/.jcode`, which is a Git repository, and `internal/git` real-repository tests do not support linked-worktree metadata. The minimal isolated retries confirmed both causes; no feature code was changed to mask them.
- No real agent, external API, credential, shared runtime, or live SDK smoke test was invoked.

### Manual smoke execution

- Status: Not run by default.
- Approval: Not requested.
- Disposable root: Not created for live testing.
- Valid single-control cases: Not run live.
- Valid all-controls case: Not run live.
- Invalid-control cases: Not run live.
- Past-deadline and SDK-error cases: Not run live.
- Exec-isolation comparison: Not run live.
- Cleanup verification: Not applicable until approved execution.
