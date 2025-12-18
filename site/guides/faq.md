---
title: FAQ
parent: Guides
nav_order: 2
---

# Frequently Asked Questions

Common questions about autospec and how it works.

---

## Why are `data_model` and `api_contracts` empty in plan.yaml?

**Short answer**: These sections are optional and only populated when applicable to your feature.

**Details**: Unlike SpecKit (which creates separate `data-model.md` and `/contracts/` files as mandatory deliverables), autospec embeds these as optional sections within `plan.yaml`. The schema marks them as `Required: false`.

```yaml
# This is valid - CLI tools often have no data model or API
data_model:
  entities: []
api_contracts:
  endpoints: []
```

**When they get populated**:
- `data_model`: Features with persistent entities, database models, or domain objects
- `api_contracts`: Features exposing REST/GraphQL endpoints or external interfaces

**When they stay empty**:
- CLI tools and utilities
- Internal refactors
- Configuration changes
- Documentation updates

This is intentional behavior, not an error.

---

## Why use `--phases` or `--tasks` instead of running everything in one session?

**Short answer**: Context isolation prevents LLM performance degradation and context pollution.

### The problem with single-session execution

When running all tasks in one Claude session (GitHub SpecKit's approach), Claude:

1. Reads the full `tasks.yaml`, `plan.yaml`, and `spec.yaml` on startup
2. Accumulates conversation context as it works through tasks
3. Must maintain mental state across dozens of tasks
4. Experiences performance degradation as context grows

This leads to:
- **Context pollution**: Earlier task discussions "contaminate" later task reasoning
- **LLM degradation**: Performance measurably decreases as context accumulates
- **Increased errors**: Claude may confuse similar tasks or forget earlier decisions
- **Harder debugging**: When something fails, the entire session's context is involved

### How phase/task isolation helps

```bash
# Phase-level: each phase gets a fresh context
autospec implement --phases

# Task-level: maximum isolation, each task is independent
autospec implement --tasks
```

| Benefit | Impact |
|---------|--------|
| **Fresh context** | Each session starts clean, no accumulated state |
| **Focused scope** | Claude sees only relevant tasks, not entire backlog |
| **Faster startup** | ~15-30 seconds saved per phase from fewer file reads |
| **Better accuracy** | No confusion from earlier task discussions |
| **Easier debugging** | Failures isolated to specific phase/task |
| **Resumable** | Use `--from-phase 3` or `--from-task T005` to continue after interruption |

### When to use each mode

| Mode | Best For |
|------|----------|
| Default (phases) | Balanced cost/context, natural recovery points |
| `--tasks` | Complex tasks, maximum accuracy, long-running implementations |
| Single-session (config) | Small specs (<5 tasks), quick iterations |

### Performance and cost analysis

Session splitting dramatically reduces costs due to how API billing works--each turn pays for the **entire conversation context**:

| Strategy | Input Tokens | Cost (Opus 4.5) | Savings |
|----------|-------------|-----------------|---------|
| Single session (38 tasks) | ~49.6M | ~$257 | baseline |
| Per-phase (10 sessions) | ~8.5M | ~$51 | **80%** |
| Per-task (38 sessions) | ~6.6M | ~$42 | **83%** |

{: .note }
> Based on analysis of a real 38-task, 10-phase spec.

**Why single sessions cost so much:**
- Starting context: ~35K tokens
- After 38 tasks: ~835K tokens (well beyond 200K limit!)
- You pay for the entire context on **every turn**

**Additional benefits:**
- **Time savings**: ~15-30 seconds per phase from eliminated redundant file reads
- **Quality**: LLM performance degrades gradually as context grows (latency increases, reduced output clarity)
- **Resumability**: Use `--from-phase` or `--from-task` to continue after interruption

{: .tip }
> For any non-trivial implementation, use `--phases` at minimum. For complex or critical work, use `--tasks`.

---

## What's the difference between spec.yaml and plan.yaml?

**spec.yaml** defines *what* to build:
- Feature description and goals
- User stories and acceptance criteria
- Requirements (functional and non-functional)
- Success criteria and metrics

**plan.yaml** defines *how* to build it:
- Technical approach and architecture decisions
- Data models and API contracts
- Implementation phases and deliverables
- Risks and mitigations
- Research findings and alternatives considered

---

## How does spec detection work?

autospec automatically detects which feature spec you're working on. Detection priority:

1. **Explicit**: User provides `--spec` flag
2. **Environment**: `SPECIFY_FEATURE` environment variable
3. **Git Branch**: Branch name matches pattern `NNN-feature-name`
4. **Fallback**: Most recently modified directory in `specs/`

For example, if you're on branch `002-user-authentication`, autospec looks for `specs/002-user-authentication/`.

See [Internals - Spec Detection](../architecture/internals#spec-detection) for more details.

---

## How do I reset retry state?

When you hit exit code 2 (retry limit exhausted):

```bash
# Reset all retry state
rm ~/.autospec/state/retry.json

# Then retry your command
autospec implement
```

Or manually edit the file to reset specific entries:

```bash
cat ~/.autospec/state/retry.json | jq 'del(.retries["002-user-auth:implement"])' > /tmp/retry.json
mv /tmp/retry.json ~/.autospec/state/retry.json
```

---

## Can I use autospec without Claude Code?

autospec is designed to work with Claude Code (the `claude` CLI). You can customize the command via:

```yaml
# .autospec/config.yml
claude_cmd: claude  # default
# or
custom_claude_cmd: "claude -p {{PROMPT}} | process-output"
```

However, autospec requires an AI assistant that can execute the embedded slash commands. Other Claude interfaces may work with modifications.

---

## How do I add a new documentation page?

autospec documentation uses Jekyll with Just the Docs theme. To add a page:

1. Create a markdown file with frontmatter:
   ```yaml
   ---
   title: My New Page
   parent: Guides  # or Reference, Architecture
   nav_order: 3    # controls position in navigation
   ---

   # My New Page

   Content here...
   ```

2. Place it in the appropriate directory:
   - `site/guides/` for how-to guides
   - `site/reference/` for reference documentation
   - `site/architecture/` for technical docs

3. Build locally to test: `cd site && bundle exec jekyll serve`

---

## Why does autospec require a constitution?

The constitution (`.autospec/memory/constitution.yaml`) defines your project's:
- Coding standards
- Architectural principles
- Testing requirements
- Documentation standards

It ensures Claude follows your project's conventions during implementation. Without it, Claude uses generic best practices which may not match your codebase.

Create one with:
```bash
autospec constitution
```

---

## How do I handle blocked tasks?

When Claude can't complete a task, it marks it as `Blocked` with a reason. Recommended workflow:

1. Check what's blocked: `autospec st`
2. Start an interactive Claude session: `claude`
3. Work through the blocked task interactively
4. Mark as complete: `autospec task complete T015`

See [Troubleshooting - Blocked Tasks](troubleshooting#blocked-tasks-workflow) for detailed guidance.

---

## What's the difference between `autospec run` and `autospec prep`?

**`autospec prep "feature"`**: Runs specify + plan + tasks stages
- Creates spec.yaml, plan.yaml, and tasks.yaml
- Stops before implementation
- Good for planning and review

**`autospec run -a`**: Runs all stages including implementation
- Full workflow from description to code
- Equivalent to: specify + plan + tasks + implement

**`autospec run` with flags**: Selective stage execution
- `-s`: specify
- `-p`: plan
- `-t`: tasks
- `-i`: implement
- `-a`: all (shortcut for `-spti`)

---

## How do I configure timeouts?

Set timeout in seconds:

```bash
# Environment variable (highest priority)
export AUTOSPEC_TIMEOUT=1800  # 30 minutes

# Project config
echo 'timeout: 1800' > .autospec/config.yml

# Global config
echo 'timeout: 1800' > ~/.config/autospec/config.yml

# Disable timeout
AUTOSPEC_TIMEOUT=0 autospec implement
```

Configuration priority: Environment > Local config > Global config > Default (300s)

---

## Where are state files stored?

| File | Location | Purpose |
|------|----------|---------|
| Retry state | `~/.autospec/state/retry.json` | Tracks retry counts per spec:stage |
| Global config | `~/.config/autospec/config.yml` | User-wide settings |
| Project config | `.autospec/config.yml` | Project-specific settings |
| Feature specs | `specs/NNN-feature/` | Specification artifacts |
| Phase context | `.autospec/context/` | Temporary files during --phases execution |
