# Roadmap

Ideas under consideration. Nothing here is scheduled.

## Candidate agent presets

**Status: demand-driven.** None of these will be built unless users ask for them (issue, discussion, or direct request). Until then, `custom` covers any agent with a headless mode.

Currently supported: claude, codex, opencode, gemini, cline, goose, jcode, custom.

Candidates, in rough priority order:

| Agent | Why | Notes |
|---|---|---|
| omp (oh-my-pi) | Daily-driven by the maintainer; likely `.agents/skills/` compatible | Verify print/JSON mode |
| Cursor CLI (`cursor-agent`) | Large user base | `-p`, `--output-format stream-json` |
| GitHub Copilot CLI | Enterprise default | `-p`, per-tool allow flags |
| Amp (Sourcegraph) | Parallel-agent users; fits `--phases` and worktrees | `amp -x`, stream-json |
| Qwen Code | Gemini CLI fork; near-free reuse of `gemini.go` | Open-weight/local model users |

Lower priority: Crush (weak headless story), Aider (auto-commits clash with the retry/validation loop), Kiro CLI and Factory Droid (small user bases; Kiro has its own competing spec workflow).

Flag and feature details above are from memory and must be verified against current docs before implementation.

### Preset bar

A full preset needs all four; otherwise point users at `custom`:

1. Headless one-shot mode with meaningful exit codes.
2. Streamable output `cclean` can parse.
3. A skill/command install location for autospec prompts.
4. Permission pre-approval flags for file and shell access.
