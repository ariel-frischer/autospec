# FAQ

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

## Why use `--phases` or `--tasks` instead of running everything in one session?

**Short answer**: Context isolation prevents LLM performance degradation and context pollution.

**The Problem with Single-Session Execution:**

When running all tasks in one Claude session (GitHub SpecKit's approach), Claude:

1. Reads the full `tasks.yaml`, `plan.yaml`, and `spec.yaml` on startup
2. Accumulates conversation context as it works through tasks
3. Must maintain mental state across dozens of tasks
4. Experiences performance degradation as context grows

This leads to:
- **Context pollution**: Earlier task discussions "contaminate" later task reasoning
- **LLM degradation**: Performance measurably decreases as context accumulates (see [research](research/claude-opus-4.5-context-performance.md))
- **Increased errors**: Claude may confuse similar tasks or forget earlier decisions
- **Harder debugging**: When something fails, the entire session's context is involved

**How Phase/Task Isolation Helps:**

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

**When to Use Each Mode:**

| Mode | Best For |
|------|----------|
| Default (phases) | Balanced cost/context, natural recovery points |
| `--tasks` | Complex tasks, maximum accuracy, long-running implementations |
| Single-session (config) | Small specs (<5 tasks), quick iterations |

**Performance & Cost Analysis:**

Session splitting dramatically reduces costs due to how API billing works—each turn pays for the **entire conversation context**:

| Strategy | Input Tokens | Cost (Opus 4.5) | Savings |
|----------|-------------|-----------------|---------|
| Single session (38 tasks) | ~49.6M | ~$257 | baseline |
| Per-phase (10 sessions) | ~8.5M | ~$51 | **80%** |
| Per-task (38 sessions) | ~6.6M | ~$42 | **83%** |

*Based on analysis of a real 38-task, 10-phase spec. See [research/claude-opus-4.5-context-performance.md](research/claude-opus-4.5-context-performance.md) for methodology.*

**Why single sessions cost so much:**
- Starting context: ~35K tokens
- After 38 tasks: ~835K tokens (well beyond 200K limit!)
- You pay for the entire context on **every turn**

**Additional benefits:**
- **Time savings**: ~15-30 seconds per phase from eliminated redundant file reads
- **Quality**: LLM performance degrades gradually as context grows (latency increases, reduced output clarity)
- **Resumability**: Use `--from-phase` or `--from-task` to continue after interruption

**Recommendation**: For any non-trivial implementation, use `--phases` at minimum. For complex or critical work, use `--tasks`.
