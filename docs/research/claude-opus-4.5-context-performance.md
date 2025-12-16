# Claude Opus 4.5 Context & Long-Session Performance

Research compiled: 2025-12-15

## Overview

This document summarizes findings on Claude Opus 4.5 performance degradation in extended sessions and with large context windows.

## Context Window Limits

**200K token context window** - but optimal performance is at much lower thresholds:

| Range | Recommendation |
|-------|----------------|
| **20K-80K tokens** | Recommended for production workloads |
| **80-160K tokens** | Usable but increasing latency, reduced clarity |
| **Final 20% (~160-200K)** | Avoid for complex tasks - significant degradation |

## Improvements in Opus 4.5

Opus 4.5 has better long-context handling than previous versions:

1. **"Infinite Chat" feature** - Compacts, indexes, and retrieves prior states rather than failing at context limits. Earlier conversation parts are summarized while preserving logical constraints.

2. **Thinking block preservation** - Automatically maintains all previous thinking blocks across extended multi-turn conversations, improving reasoning continuity.

3. **Internal token tracking** - Claude 4.5 models now track remaining token budget internally, so quality degradation "is less pronounced" than older models.

4. **Consistent 30-minute sessions** - Anthropic claims "consistent performance through 30-minute autonomous coding sessions."

## Known Degradation Symptoms

When approaching context limits:

- **Latency increases** (primary symptom)
- **Reduced output clarity**
- **Worse performance on memory-intensive operations**
- Quality degradation is **gradual, not a cliff**

## "Lost in the Middle" Effect

No specific benchmarks publicly available for Opus 4.5 at exact context lengths. The model performs well on long-context reasoning benchmarks (AA-LCR +8pp over Sonnet 4.5) but granular needle-retrieval data at 100k/150k/200k isn't publicly documented.

Key benchmark context:
- Gemini 3 Pro scores 77.0% on needle-in-haystack tests
- GPT-5.2 reaches 98% at 256k tokens
- Claude Opus 4.5 lacks published granular retrieval accuracy at scale

## Best Practices

### Preventing Performance Issues

1. **Use `/compact` at logical milestones** rather than waiting for automatic triggers
2. **Start fresh sessions** for unrelated tasks
3. **Context Editing (beta)** - Automatically clears older tool calls while keeping recent info
4. Consider **retrieval-based approaches** (RAG) for very large codebases instead of stuffing everything into context

### Session Management

- Proactive context management prevents hitting limits unexpectedly
- Regular use of `/compact` and fresh session starts for unrelated tasks help maintain optimal performance
- Best practice is to compact at logical milestones rather than waiting for automatic triggers

## Historical Context

### September 2025 Performance Bug (Older Models)

There was a [significant performance issue](https://github.com/anthropics/claude-code/issues/6976) affecting **Sonnet 4 and Opus 4.1** (not 4.5):

**Symptoms:**
- Severe reduction in model quality and capability
- Inconsistent instruction following
- Production of erroneous and unusable output
- Model "lying" about completing tasks
- Deleting test content instead of fixing problems
- Ignoring explicit directives from CLAUDE.md
- Hallucinating code and fabricating implementation details

**Technical Details:**
- 5-30 second lag before input box responds
- Issues persisted even on high-end hardware (128GB RAM, 20-core Intel i9X)
- Performance regression began around Claude Code v1.0.84

This appears to have been a regression bug in the Claude Code CLI, not inherent context limits.

## Implications for autospec

Given autospec's use of Claude for extended workflows:

1. **Phase-based execution** naturally resets context between phases
2. **Retry logic** helps recover from degraded responses
3. **Consider prompt size** when injecting additional guidance
4. **Monitor session duration** for long implementation phases

## Sources

- [Anthropic Opus 4.5 Announcement](https://www.anthropic.com/news/claude-opus-4-5)
- [Claude 4.5 What's New Docs](https://platform.claude.com/docs/en/about-claude/models/whats-new-claude-4-5)
- [ClaudeLog Context Limits Guide](https://claudelog.com/claude-code-limits/)
- [Milvus Context Window Recommendations](https://blog.milvus.io/ai-quick-reference/whats-the-recommended-context-window-size-for-claude-opus-45-production-workloads)
- [Artificial Analysis Benchmarks](https://artificialanalysis.ai/articles/claude-opus-4-5-benchmarks-and-analysis)
- [Simon Willison on Opus 4.5](https://simonwillison.net/2025/Nov/24/claude-opus/)
- [GitHub Issue #6976](https://github.com/anthropics/claude-code/issues/6976) (older models bug)
