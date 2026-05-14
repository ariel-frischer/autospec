# Manual Testing Plan: Persist Active Feature Directory

## Scope

Validate active feature resolution across persisted state, explicit override, stale state handling, branch fallback, and artifact lookup. Do not execute these scenarios during the documentation task; record results later in Report Summaries.

## Scenarios

### Persisted Branch Mismatch

1. Create or select `specs/128-persist-feature-directory` so it becomes the active feature.
2. Check out a different branch name that does not match the feature directory suffix.
3. Run `autospec status`, `autospec plan`, `autospec tasks`, and `autospec implement --phase 1 --dry-run` where applicable.
4. Confirm each command resolves `specs/128-persist-feature-directory` and reports persisted active feature state as the source.

### Explicit Override

1. Persist `specs/128-persist-feature-directory` as the active feature.
2. Run `autospec status --spec <other-valid-feature>` or the equivalent positional spec form for commands that support it.
3. Run `autospec run -ti --spec <other-valid-feature> --dry-run`.
4. Confirm the explicit feature wins and persisted state is not used for that invocation.

### Stale Persisted State

1. Point the project-local active feature state at a deleted directory or a directory without `spec.yaml`.
2. Run `autospec status`.
3. Confirm the error names the selected directory, explains the missing or invalid artifact, and gives a recovery path.
4. Select an existing feature explicitly or remove the stale state, then confirm the command succeeds.

### Branch Fallback

1. Remove or clear active feature state.
2. Check out a branch named with a valid feature prefix, such as `128-persist-feature-directory` or another existing `NNN-*` branch.
3. Run `autospec status`, `autospec prereqs`, and one workflow stage command in dry-run mode if supported.
4. Confirm autospec resolves the feature through branch-prefix fallback.

### Artifact Lookup

1. Persist an active feature with valid `spec.yaml`, `plan.yaml`, and `tasks.yaml`.
2. Run path-based artifact validation for each artifact.
3. Run type-only artifact/schema lookups where supported.
4. Confirm path-based validation uses the supplied path and type-only lookup follows active feature resolution.

## Report Summaries

- Persisted branch mismatch:
- Explicit override:
- Stale persisted state:
- Branch fallback:
- Artifact lookup:
