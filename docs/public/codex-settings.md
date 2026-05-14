# Codex Settings

autospec supports Codex through the non-interactive Codex CLI command:

```bash
codex exec "<rendered autospec prompt>"
```

## Authentication

autospec does not require `OPENAI_API_KEY` for Codex. Authentication is owned by the Codex CLI and may use ChatGPT login or API credentials.

Optional environment variables:

| Variable | Purpose |
|----------|---------|
| `OPENAI_API_KEY` | API authentication when using API billing |
| `OPENAI_BASE_URL` | API-compatible base URL override |
| `CODEX_HOME` | Alternate Codex config and auth directory |

## Configuration Files

Codex user config lives at:

```text
~/.codex/config.toml
```

Project-level autospec setup creates:

```text
.codex/config.toml
```

The project file is intentionally minimal. autospec records project metadata only and does not write destructive defaults such as full-access sandboxing.

## Sandboxing And Approvals

Codex supports these controls for `codex exec`:

```bash
codex exec --sandbox workspace-write "task"
codex exec --ask-for-approval never "task"
codex exec --dangerously-bypass-approvals-and-sandbox "task"
```

autospec maps:

```yaml
skip_permissions: true
```

to:

```bash
--dangerously-bypass-approvals-and-sandbox
```

This is autospec's default because workflow runs are intended to complete unattended. Set `skip_permissions: false` if you want Codex sandbox and approval behavior controlled by `~/.codex/config.toml` or explicit Codex flags.

## Init

```bash
autospec init --ai codex
autospec init --project --ai codex
```

Codex does not use Claude/OpenCode slash-command files, so autospec does not install command templates for Codex. Workflow stages started through the autospec CLI send rendered prompt text directly to `codex exec`.

For interactive Codex sessions, project-level init installs one Codex skill per autospec command template:

```text
.codex/skills/autospec-specify/SKILL.md
.codex/skills/autospec-plan/SKILL.md
.codex/skills/autospec-tasks/SKILL.md
.codex/skills/autospec-implement/SKILL.md
.codex/skills/autospec-constitution/SKILL.md
.codex/skills/autospec-clarify/SKILL.md
.codex/skills/autospec-checklist/SKILL.md
.codex/skills/autospec-analyze/SKILL.md
.codex/skills/autospec-worktree-setup/SKILL.md
```

Each skill is generated from the matching `internal/commands/autospec.*.md` prompt and registered in `.codex/config.toml` with `skills.config`. Use Codex-native skill syntax such as `$autospec-specify "Add user auth"` or `$autospec-clarify`; slash-style text such as `/autospec.specify "Add user auth"` is also described in the relevant skill for compatibility, but it is not a Codex-native slash command.

## Output

Codex `exec` is the non-interactive CLI mode. It writes formatted terminal output by default, which is useful for humans watching a run.

For script parsing, Codex supports JSON Lines output:

```bash
codex exec --json "summarize the repo structure"
```

With `--json`, stdout is a JSONL event stream. Events include thread and turn lifecycle events, agent messages, reasoning summaries, command executions, file changes, MCP tool calls, web searches, and plan updates.

If only the final assistant message is needed, Codex can also write it to a file:

```bash
codex exec -o codex-final.txt "summarize the repo structure"
```

autospec currently relies on Codex process exit status plus generated workflow artifacts (`spec.yaml`, `plan.yaml`, `tasks.yaml`) rather than parsing Codex JSONL events.

## References

- Codex CLI reference: https://developers.openai.com/codex/cli/reference
- Codex non-interactive mode: https://developers.openai.com/codex/noninteractive
- Codex approvals and sandboxing: https://developers.openai.com/codex/agent-approvals-security
- Codex config reference: https://developers.openai.com/codex/config-reference
