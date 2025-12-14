# Claude Agent SDK Evaluation for auto-claude-speckit

**Date:** December 2025
**Purpose:** Evaluate whether to migrate from Go to Claude Agent SDK for parallel spec execution with isolation

---

## Executive Summary

**Recommendation: Stay with Go + integrate standalone sandbox runtime (`srt`)**

The Claude Agent SDK's sandboxing features come from a **standalone CLI tool** (`srt`) that any language can call, not from Python-specific magic. Git worktree support doesn't exist in any SDK and would require custom implementation regardless. The optimal path is keeping Go as the orchestrator while integrating the sandbox runtime for isolation.

---

## Research Findings

### 1. Official Claude Agent SDK

**Supported Languages:**
- Python (`claude-agent-sdk` on PyPI)
- TypeScript/JavaScript (`@anthropic-ai/claude-agent-sdk` on npm)
- **Go: No official SDK**

**Key Features:**
| Feature | Python/TS SDK | Description |
|---------|---------------|-------------|
| Subagents | Yes | Automatic parallelization for complex tasks |
| Session Forking | Yes | Branch conversations for parallel exploration |
| Custom Tools | Yes | `@tool` decorator for Python functions |
| Hooks | Yes | Pre/Post tool use callbacks |
| MCP Integration | Yes | Native Model Context Protocol support |
| Structured Output | Yes | JSON schema validation |
| Sandbox | Via `srt` | Wraps standalone sandbox-runtime tool |

### 2. Community Go SDKs

**Evaluated:**
- `github.com/yukifoo/claude-code-sdk-go`
- `github.com/severity1/claude-code-sdk-go`

**Verdict: NOT suitable for our needs**

| Feature | Go SDKs | Notes |
|---------|---------|-------|
| Basic Queries | Yes | Streaming and non-streaming |
| Session Management | Yes | Resume/continue via IDs |
| Tool Control | Yes | Allow/disallow lists |
| **Sandboxing** | **NO** | Not implemented |
| **Parallel Agents** | **NO** | Single agent only |
| **Git Worktrees** | **NO** | Not supported |

These are thin CLI wrappers that execute `claude` commands via subprocess. They don't implement any advanced features - they just pass through to the CLI.

### 3. Standalone Sandbox Runtime (`srt`)

**This is the key finding.** The sandbox is NOT a Python SDK feature - it's a standalone tool.

**Repository:** `github.com/anthropic-experimental/sandbox-runtime`
**Package:** `@anthropic-ai/sandbox-runtime` (npm)
**Installation:** `npm install -g @anthropic-ai/sandbox-runtime`

**How it works:**
```bash
# Sandbox any command
srt "<your-command>"

# With custom settings
srt --settings .srt-settings.json "claude /speckit.implement"
```

**Platform Support:**
- **Linux:** Uses `bubblewrap` for namespace isolation
- **macOS:** Uses `sandbox-exec` (Seatbelt) with generated profiles
- **Windows/WSL2:** Not supported

**Isolation Capabilities:**
- **Filesystem:** Allowlist-based read/write access
- **Network:** Proxy-based domain filtering
- **No containers required:** OS-level primitives only

**Can be called from Go:** Yes, via `exec.Command("srt", ...)`

### 4. Git Worktree Support

**NOT supported by ANY SDK.** This would require custom implementation regardless of language choice.

---

## Architecture Options

### Option A: Stay with Go + Integrate `srt` (Recommended)

```
┌─────────────────────────────────────────────────────┐
│                  Go Orchestrator                     │
│              (auto-claude-speckit)                   │
├─────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │
│  │  Spec 001   │  │  Spec 002   │  │  Spec 003   │  │
│  │  Goroutine  │  │  Goroutine  │  │  Goroutine  │  │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  │
│         │                │                │         │
│         ▼                ▼                ▼         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │
│  │ srt sandbox │  │ srt sandbox │  │ srt sandbox │  │
│  │ worktree/a  │  │ worktree/b  │  │ worktree/c  │  │
│  └─────────────┘  └─────────────┘  └─────────────┘  │
└─────────────────────────────────────────────────────┘
```

**Implementation:**
1. Keep current Go CLI architecture
2. Add git worktree management (custom Go code)
3. Execute specs in parallel goroutines
4. Each goroutine runs `srt "claude /speckit.implement"` in its worktree
5. Aggregate results

**Pros:**
- Minimal migration effort
- Cross-platform binary distribution preserved
- Sandbox isolation via `srt`
- Native Go concurrency for parallelism

**Cons:**
- Must implement worktree management
- `srt` requires npm installation (or bundle it)
- No subagent features

**Effort:** Medium (2-3 weeks implementation)

### Option B: Full Python Migration

```
┌─────────────────────────────────────────────────────┐
│              Python Orchestrator                     │
│            (new auto-claude-speckit)                 │
├─────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────┐│
│  │           Claude Agent SDK (Python)             ││
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐      ││
│  │  │ Subagent │  │ Subagent │  │ Subagent │      ││
│  │  │ Spec 001 │  │ Spec 002 │  │ Spec 003 │      ││
│  │  └──────────┘  └──────────┘  └──────────┘      ││
│  └─────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────┘
```

**Pros:**
- Native subagent support (automatic parallelization)
- Session forking for branching
- Custom tools via `@tool` decorator
- Hooks for lifecycle events
- Direct SDK integration (no CLI parsing)

**Cons:**
- **Full rewrite required** - 2000+ lines of Go code
- Python distribution complexity (venv, pip, etc.)
- Lose single-binary simplicity
- Still need custom git worktree implementation
- Sandbox still via `srt` (same as Go option)

**Effort:** High (4-6 weeks rewrite)

### Option C: Hybrid - Go Orchestrator + Python Workers

```
┌─────────────────────────────────────────────────────┐
│                  Go Orchestrator                     │
├─────────────────────────────────────────────────────┤
│         │                │                │         │
│         ▼                ▼                ▼         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │
│  │   Python    │  │   Python    │  │   Python    │  │
│  │   Worker    │  │   Worker    │  │   Worker    │  │
│  │ (SDK Agent) │  │ (SDK Agent) │  │ (SDK Agent) │  │
│  └─────────────┘  └─────────────┘  └─────────────┘  │
└─────────────────────────────────────────────────────┘
```

**Pros:**
- Best of both worlds
- Go handles orchestration, distribution
- Python workers get full SDK features
- Gradual migration path

**Cons:**
- Two runtime dependencies (Go binary + Python)
- IPC complexity between Go and Python
- Still need worktree implementation
- Deployment complexity

**Effort:** High (4-5 weeks)

---

## Feature Comparison Matrix

| Feature | Current Go | Go + srt | Python SDK | Hybrid |
|---------|------------|----------|------------|--------|
| Single binary | Yes | Yes* | No | No |
| Sandboxing | No | Yes | Yes | Yes |
| Parallel specs | No | Yes (goroutines) | Yes (subagents) | Yes |
| Git worktrees | No | Custom impl | Custom impl | Custom impl |
| Subagents | No | No | Yes | Yes |
| Session forking | No | No | Yes | Yes |
| Custom tools | No | No | Yes | Yes |
| Hooks | Shell scripts | Shell scripts | Python callbacks | Python callbacks |
| Migration effort | - | Medium | High | High |

*Requires bundled `srt` or npm dependency

---

## Recommended Implementation Plan

### Phase 1: Add Sandbox Support (Go + srt)

1. **Add `srt` integration to Go binary**
   ```go
   func ExecuteInSandbox(command string, workdir string) error {
       cmd := exec.Command("srt", "--settings", ".srt-settings.json", command)
       cmd.Dir = workdir
       return cmd.Run()
   }
   ```

2. **Create default `.srt-settings.json`**
   ```json
   {
     "sandbox": {
       "filesystem": {
         "allowWrite": ["./specs", "./tmp"],
         "denyRead": [".ssh", ".aws", ".env"]
       },
       "network": {
         "allowedDomains": ["api.anthropic.com", "claude.ai"]
       }
     }
   }
   ```

3. **Add `--sandbox` flag to CLI commands**
   ```bash
   autospec implement --sandbox 003-my-feature
   ```

### Phase 2: Add Parallel Spec Execution

1. **Implement git worktree management**
   ```go
   type Worktree struct {
       SpecName string
       Path     string
       Branch   string
   }

   func (w *Worktree) Create() error {
       return exec.Command("git", "worktree", "add", w.Path, "-b", w.Branch).Run()
   }

   func (w *Worktree) Remove() error {
       return exec.Command("git", "worktree", "remove", w.Path).Run()
   }
   ```

2. **Add parallel orchestration**
   ```go
   func RunParallelSpecs(specs []string) error {
       var wg sync.WaitGroup
       results := make(chan SpecResult, len(specs))

       for _, spec := range specs {
           wg.Add(1)
           go func(s string) {
               defer wg.Done()
               wt := CreateWorktree(s)
               defer wt.Remove()
               result := ExecuteInSandbox("claude /speckit.implement", wt.Path)
               results <- result
           }(spec)
       }

       wg.Wait()
       close(results)
       return aggregateResults(results)
   }
   ```

3. **Add CLI command**
   ```bash
   autospec parallel 001-feature 002-feature 003-feature
   ```

### Phase 3: Evaluate Python Migration (Future)

If subagents or advanced SDK features become critical:
1. Create Python worker script using `claude-agent-sdk`
2. Call from Go orchestrator via subprocess
3. Gradually migrate complex logic to Python
4. Keep Go for CLI/distribution

---

## Conclusions

1. **Don't migrate to Python just for sandboxing** - The sandbox is a standalone tool (`srt`) that Go can call directly.

2. **Git worktree support is custom work regardless** - No SDK provides this, so you'd implement it in any language.

3. **Go + srt is the pragmatic path** - Keep your existing architecture, add sandbox via `srt`, implement worktrees in Go.

4. **Python SDK is valuable for subagents** - If you need Claude to automatically parallelize complex tasks, the Python SDK's subagent feature is compelling. Consider hybrid approach.

5. **The Go community SDKs are not useful** - They're just CLI wrappers without advanced features. Don't bother.

---

## Resources

- **Sandbox Runtime:** https://github.com/anthropic-experimental/sandbox-runtime
- **Python SDK:** https://github.com/anthropics/claude-agent-sdk-python
- **TypeScript SDK:** https://github.com/anthropics/claude-agent-sdk-typescript
- **Claude Code Sandboxing:** https://www.anthropic.com/engineering/claude-code-sandboxing
- **Subagents Guide:** https://platform.claude.com/docs/en/agent-sdk/subagents
