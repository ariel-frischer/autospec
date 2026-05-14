# Codex Settings

autospec supports Codex through the non-interactive Codex CLI command:

```bash
codex exec --json "<rendered autospec prompt>"
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

Codex does not use slash-command files, so autospec does not install command templates for Codex. Workflow stages started through the autospec CLI send rendered prompt text directly to `codex exec`.

For interactive Codex sessions, project-level init installs one shared Agent Skill per autospec command template:

```text
.agents/skills/autospec-specify/SKILL.md
.agents/skills/autospec-plan/SKILL.md
.agents/skills/autospec-tasks/SKILL.md
.agents/skills/autospec-implement/SKILL.md
.agents/skills/autospec-constitution/SKILL.md
.agents/skills/autospec-clarify/SKILL.md
.agents/skills/autospec-checklist/SKILL.md
.agents/skills/autospec-analyze/SKILL.md
.agents/skills/autospec-worktree-setup/SKILL.md
```

Each skill is generated from the matching `internal/commands/autospec.*.md` prompt and registered in `.codex/config.toml` with `skills.config`. Use Codex-native skill syntax such as `$autospec-specify "Add user auth"` or `$autospec-clarify`; slash-style text such as `/autospec.specify "Add user auth"` is also described in the relevant skill for compatibility, but it is not a Codex-native slash command.

## Output

Codex `exec` is the non-interactive CLI mode. autospec uses compact output by default for automated Codex runs:

```yaml
codex_output:
  mode: compact
  max_lines_per_message: 40
  color: true
```

In compact mode autospec runs `codex exec --json`, parses the JSONL event stream, and displays color-coded concise agent messages, command summaries, file-change summaries, and useful reasoning/tool labels. Each displayed block is capped by `max_lines_per_message`; truncated blocks include a hint to switch to full mode. Set `codex_output.color: false` to disable ANSI color.

To restore Codex's native terminal transcript:

```yaml
codex_output:
  mode: full
```

Full mode runs `codex exec "<rendered autospec prompt>"` without `--json`. Interactive Codex sessions are unchanged.

For script parsing, Codex supports JSON Lines output:

```bash
codex exec --json "summarize the repo structure"
```

With `--json`, stdout is a JSONL event stream. Events include thread and turn lifecycle events, agent messages, reasoning summaries, command executions, file changes, MCP tool calls, web searches, and plan updates.

If only the final assistant message is needed, Codex can also write it to a file:

```bash
codex exec -o codex-final.txt "summarize the repo structure"
```

autospec still relies on Codex process exit status plus generated workflow artifacts (`spec.yaml`, `plan.yaml`, `tasks.yaml`) for workflow validation; compact output only changes what is shown while Codex runs.

## References

- Codex CLI reference: https://developers.openai.com/codex/cli/reference
- Codex non-interactive mode: https://developers.openai.com/codex/noninteractive
- Codex approvals and sandboxing: https://developers.openai.com/codex/agent-approvals-security
- Codex config reference: https://developers.openai.com/codex/config-reference
