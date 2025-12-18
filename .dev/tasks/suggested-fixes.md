# Suggested Fixes from Session Observations

Based on analysis of 29+ autospec-triggered Claude sessions documented in `observations.md`, these are concrete fixes to reduce token waste and improve efficiency.

---

## Priority Matrix

| Fix | Severity | Token Savings | Effort |
|-----|----------|---------------|--------|
| 1. File Reading Discipline in implement.md | CRITICAL | 30-50K/session | Low |
| 2. Phase Context Metadata Flags | HIGH | 5-15K/session | Medium |
| 3. Context Efficiency in specify.md/plan.md | HIGH | 10-20K/session | Low |
| 4. Sandbox Pre-Approval Documentation | MEDIUM | 2-5K/session | Low |
| 5. Large File Handling Strategy | MEDIUM | 5-10K/session | Medium |

---

## Fix 1: File Reading Discipline in implement.md (CRITICAL)

**Problem:** Claude reads the same file 5-23 times per session. Example: `preflight_test.go` read 18 times, `parse-claude-conversation.sh` read 14 times.

**Root Cause:** No explicit guidance telling Claude that file contents remain in context after reading.

```bash
autospec specify "Add file reading discipline section to internal/commands/autospec.implement.md template. This addresses a CRITICAL issue where Claude reads the same file 5-23 times per session, wasting 30-50K tokens.

## Problem Statement
Analysis of 29 autospec sessions revealed extreme file re-reading:
- preflight_test.go read 18 times in one session
- workflow.go read 16 times in one session
- parse-claude-conversation.sh read 14 times in one session
- schema_validation.go read 6-7 times in sessions

Claude does not recognize that file contents remain in its context window after the initial Read tool call.

## Required Changes

Add a new section to internal/commands/autospec.implement.md after the context loading section:

### Section Title: 'CRITICAL: File Reading Discipline'

### Content to Add:

1. 'Read Once, Remember Forever' rule block explaining:
   - When you read a file with the Read tool, the content IS NOW in your context window
   - You can reference file contents without re-reading
   - DO NOT read the same file again unless you made changes and need to verify (max 1 re-read)

2. Maximum File Read Counts table:
   | Scenario | Max Reads |
   |----------|-----------|
   | Understanding a file | 1 |
   | Editing a file | 2 (before + after) |
   | Referencing while editing another | 0 (already have it) |
   | Debugging test failures | 2 |

3. 'Pre-Task File Discovery' protocol:
   - Before starting implementation, identify ALL files you will need
   - Read each file ONCE at the start
   - Note line numbers of relevant sections
   - Proceed with implementation WITHOUT re-reading

4. Explicit prohibitions:
   - DO NOT re-read files to 'make sure' you have the content
   - DO NOT re-read files when switching between tasks
   - DO NOT grep a file then read it fully then grep again

## Acceptance Criteria
- [ ] New 'CRITICAL: File Reading Discipline' section exists in implement.md
- [ ] Section appears prominently (near top, after context loading)
- [ ] Includes the Maximum File Read Counts table
- [ ] Includes Pre-Task File Discovery protocol
- [ ] Uses clear formatting with checkmarks and X marks for dos/donts

## Non-Functional Requirements
- Template change only, no Go code changes
- Section must be scannable (headers, bullets, tables)
- Use warning/critical styling to emphasize importance"
```

---

## Fix 2: Phase Context Metadata Flags (HIGH)

**Problem:** Claude reads phase context (which bundles spec/plan/tasks) then immediately reads spec.yaml, plan.yaml, tasks.yaml separately. Also checks for non-existent checklists directory 128 times across 6 sessions.

**Root Cause:** Phase context file doesn't explicitly tell Claude what's bundled and what to skip.

```bash
autospec specify "Add metadata flags to phase context generation in internal/workflow/phase_context.go. This addresses redundant artifact reads (15K tokens/session) and unnecessary checklists checks (128 checks across 6 sessions for non-existent directory).

## Problem Statement
Analysis shows Claude consistently:
1. Reads phase-X.yaml context file (contains bundled spec/plan/tasks)
2. Immediately reads spec.yaml separately (REDUNDANT)
3. Reads plan.yaml separately (REDUNDANT)
4. Reads tasks.yaml separately (REDUNDANT)
5. Checks for checklists/ directory (NEVER EXISTS in most projects)

The phase context header says 'This file bundles spec, plan, and phase-specific tasks' but Claude ignores this because there is no machine-readable metadata.

## Required Changes

### 1. Update PhaseContext struct in internal/workflow/types.go or phase_context.go

Add a ContextMeta field to the phase context YAML structure:

```yaml
_context_meta:
  phase_artifacts_bundled: true    # Signals DO NOT read individual artifacts
  bundled_artifacts:
    - spec.yaml
    - plan.yaml
    - tasks.yaml (phase-filtered)
  has_checklists: false            # Skip checklists directory check
  skip_reads:
    - 'specs/<feature>/spec.yaml'
    - 'specs/<feature>/plan.yaml'
    - 'specs/<feature>/tasks.yaml'
```

### 2. Update phase context generation

Modify the function that generates phase-X.yaml files (likely in internal/workflow/phase_context.go or similar) to:
- Always include _context_meta section at the top of generated files
- Set phase_artifacts_bundled: true
- List the bundled artifacts explicitly
- Check if checklists/ directory exists and set has_checklists accordingly
- Generate skip_reads list with actual paths

### 3. Update implement.md template

Add section explaining _context_meta:

```markdown
## Phase Context Metadata

The phase context file includes a '_context_meta' section:

- 'phase_artifacts_bundled: true' means DO NOT read spec.yaml, plan.yaml, or tasks.yaml separately
- 'has_checklists: false' means DO NOT check for checklists/ directory
- 'skip_reads' lists files that are already bundled - DO NOT read them

If _context_meta.phase_artifacts_bundled is true, you MUST NOT read individual artifact files.
```

## Acceptance Criteria
- [ ] PhaseContext struct includes ContextMeta field
- [ ] Generated phase-X.yaml files include _context_meta section
- [ ] _context_meta.phase_artifacts_bundled is always true for phase contexts
- [ ] _context_meta.has_checklists reflects actual directory existence
- [ ] _context_meta.skip_reads lists bundled artifact paths
- [ ] implement.md template documents _context_meta usage
- [ ] Existing tests pass after changes

## Non-Functional Requirements
- Backward compatible (old phase contexts without _context_meta still work)
- _context_meta section appears at TOP of generated YAML
- Performance: directory existence check <1ms"
```

---

## Fix 3: Context Efficiency Guidance in specify.md and plan.md (HIGH)

**Problem:** specify and plan commands also exhibit file re-reading, though less severe than implement. workflow.go read 16 times in one plan session.

```bash
autospec specify "Add context efficiency guidance to internal/commands/autospec.specify.md and internal/commands/autospec.plan.md templates. This reduces file re-reading during specification and planning phases.

## Problem Statement
While less severe than implement sessions, specify and plan sessions also show inefficiency:
- workflow.go read 16 times in one plan session
- executor_test.go read 9 times
- Multiple files read 3-4 times during codebase exploration

## Required Changes

### 1. Add to autospec.specify.md

Add 'Context Efficiency' section after the codebase exploration guidance:

```markdown
## Context Efficiency

When analyzing the codebase for specification:

1. **Grep Before Read**: Use Grep to locate patterns BEFORE reading entire files
2. **Read Sections**: For large files (>500 lines), use offset/limit to read only needed sections
3. **No Re-Reads**: Once you have read a file, DO NOT read it again in this session
4. **Track Mentally**: Keep mental note of files read - they are in your context

### Large File Strategy
For files exceeding 1000 lines:
- Use Grep to find function/class locations
- Read specific sections with offset and limit parameters
- Never attempt to read the entire file
```

### 2. Add to autospec.plan.md

Add similar 'Context Efficiency' section:

```markdown
## Context Efficiency

When exploring code for implementation planning:

1. **Targeted Discovery**: Search for specific patterns rather than reading entire files
2. **Symbol-First**: Use Serena MCP symbol tools when available for structured exploration
3. **Read Once**: Each file should be read at most once during planning
4. **Cache Coverage Data**: If spec.yaml already contains coverage analysis, do not re-run coverage commands

### Coverage Data Reuse
If the spec.yaml non_functional section contains coverage baseline data:
- DO NOT run 'go test -cover' again
- DO NOT run 'go tool cover -func' again
- Use the cached values from spec.yaml
```

## Acceptance Criteria
- [ ] autospec.specify.md contains 'Context Efficiency' section
- [ ] autospec.plan.md contains 'Context Efficiency' section
- [ ] Both include 'Large File Strategy' guidance
- [ ] plan.md includes 'Coverage Data Reuse' guidance
- [ ] Guidance is actionable with clear dos/donts

## Non-Functional Requirements
- Template changes only
- Sections should be concise (<20 lines each)
- Use consistent formatting with implement.md"
```

---

## Fix 4: Sandbox Pre-Approval Documentation (MEDIUM)

**Problem:** 90 sandbox workarounds across 6 sessions. Go build/test commands consistently require `dangerouslyDisableSandbox: true`.

```bash
autospec specify "Document sandbox pre-approval configuration for Go commands in CLAUDE.md and docs/claude-settings.md. This eliminates 90+ sandbox workaround prompts per 6 sessions.

## Problem Statement
Go build and test commands consistently fail in Claude Code sandbox:
- 'go build' requires GOCACHE workaround or sandbox disable
- 'go test' requires sandbox disable
- 'make build' and 'make test' require sandbox disable
- Average 15 sandbox prompts per implement session

Users must repeatedly approve sandbox overrides, adding friction and wasting tokens on error/retry cycles.

## Required Changes

### 1. Update CLAUDE.md

Add a 'Sandbox Configuration' section recommending these allowlist entries:

```markdown
## Sandbox Configuration

To avoid repeated sandbox prompts during Go development, add these to your Claude Code settings:

```json
{
  'permissions': {
    'allow': [
      'Bash(go build:*)',
      'Bash(go test:*)',
      'Bash(make build:*)',
      'Bash(make test:*)',
      'Bash(make fmt:*)',
      'Bash(make lint:*)',
      'Bash(GOCACHE=/tmp/claude/go-cache go build:*)',
      'Bash(GOCACHE=/tmp/claude/go-cache go test:*)',
      'Bash(GOCACHE=/tmp/claude/go-cache make:*)'
    ]
  }
}
```

These commands are safe for auto-approval as they only read source files and write to designated output directories.
```

### 2. Update docs/claude-settings.md

Add detailed explanation of each permission and why it is safe:

- go build: Compiles Go code, writes to ./bin or current directory
- go test: Runs tests, writes coverage files to designated locations
- make targets: Wrapper commands that invoke go build/test
- GOCACHE variant: Explicit cache directory for sandbox compatibility

### 3. Add to internal/commands/autospec.implement.md

Add note about expected sandbox behavior:

```markdown
## Sandbox Notes

Go build and test commands may trigger sandbox prompts. Recommended pre-approvals:
- 'Bash(go build:*)'
- 'Bash(go test:*)'
- 'Bash(make build:*)'
- 'Bash(make test:*)'

If sandbox errors occur, the GOCACHE workaround usually resolves them:
'GOCACHE=/tmp/claude/go-cache go build ./...'
```

## Acceptance Criteria
- [ ] CLAUDE.md contains Sandbox Configuration section
- [ ] docs/claude-settings.md documents each permission
- [ ] implement.md includes Sandbox Notes
- [ ] Permission list covers all common Go development commands
- [ ] Explanation of why each permission is safe

## Non-Functional Requirements
- Documentation only, no code changes
- JSON examples must be valid and copy-pasteable
- Include both standard and GOCACHE variants"
```

---

## Fix 5: Large File Handling Strategy (MEDIUM)

**Problem:** workflow_test.go (45K+ tokens) exceeds read limits, causing multiple grep→partial-read cycles.

```bash
autospec specify "Add large file handling hints to tasks.yaml schema and implement.md template. This provides reading strategies for files that exceed Claude's token limits.

## Problem Statement
Several files consistently exceed Claude's read limits:
- workflow_test.go: 45K+ tokens, read limit exceeded in 8/20 sessions
- troubleshooting.md: 960 lines exceeds 950 line limit
- Large test files require multiple offset/limit reads

Without guidance, Claude attempts to read these files fully, fails, then tries grep, then partial reads - wasting tokens on the discovery process.

## Required Changes

### 1. Update tasks.yaml schema (internal/validation/tasks_yaml.go)

Add optional _implementation_hints field to task schema:

```yaml
tasks:
  - id: T001
    # ... existing fields ...

_implementation_hints:
  large_files:
    - path: 'internal/workflow/workflow_test.go'
      size_estimate: '45K tokens'
      strategy: 'grep for function names, read sections with offset/limit'
      key_functions:
        - name: 'newTestOrchestratorWithSpecName'
          line: 3296
        - name: 'writeTestTasks'
          line: 3448
    - path: 'docs/troubleshooting.md'
      size_estimate: '960 lines'
      strategy: 'read sections by topic, use offset/limit'
```

### 2. Update tasks.yaml validation

Modify ValidateTasksYAML to accept but not require _implementation_hints field.

### 3. Update internal/commands/autospec.tasks.md

Add guidance to generate _implementation_hints when large files are identified during task generation:

```markdown
## Large File Detection

When generating tasks that involve files over 500 lines:

1. Note the file in _implementation_hints.large_files
2. Include size_estimate (lines or token estimate)
3. Suggest reading strategy
4. If known, include key function names and line numbers

This helps the implement phase read efficiently without discovery overhead.
```

### 4. Update internal/commands/autospec.implement.md

Add section on using _implementation_hints:

```markdown
## Large File Handling

Check _implementation_hints.large_files in tasks.yaml before reading:

1. If a file is listed with strategy, follow that strategy
2. For files marked 'grep for function names':
   - Use Grep to find function locations first
   - Read only the sections you need with offset/limit
3. For files with key_functions listed:
   - Go directly to those line numbers

### Default Strategy for Unlisted Large Files

If you encounter a file that exceeds read limits:
1. Use Grep to find relevant patterns
2. Read sections with offset/limit (500 lines at a time)
3. Note the file for future _implementation_hints
```

## Acceptance Criteria
- [ ] tasks.yaml schema accepts _implementation_hints field
- [ ] Validation passes with or without _implementation_hints
- [ ] autospec.tasks.md documents large file detection
- [ ] autospec.implement.md documents large file handling
- [ ] Example _implementation_hints structure is documented

## Non-Functional Requirements
- Schema change is backward compatible
- _implementation_hints is optional, not required
- No validation errors for existing tasks.yaml files
- Documentation includes concrete examples"
```

---

## Fix 6: Test Infrastructure Caching (MEDIUM)

**Problem:** Mock infrastructure (mock-claude.sh, MockClaudeExecutor, test helpers) rediscovered in every implement phase.

```bash
autospec specify "Add test infrastructure caching via spec-level notes.yaml artifact. This eliminates repeated discovery of mock scripts, test helpers, and test patterns across phases.

## Problem Statement
Each implement phase rediscovers the same test infrastructure:
- Location of mock-claude.sh (mocks/scripts/ vs tests/mocks/)
- MockClaudeExecutor in mocks_test.go
- Test helper functions (newTestOrchestratorWithSpecName, writeTestSpec)
- Coverage baseline and target functions

This discovery is repeated in EVERY phase of EVERY implement session.

## Required Changes

### 1. Create new artifact type: notes.yaml

Add validation for optional notes.yaml in specs/<feature>/:

```yaml
# specs/043-workflow-mock-coverage/notes.yaml
_discovered:
  test_infrastructure:
    mock_claude_script: 'mocks/scripts/mock-claude.sh'
    mock_executor: 'internal/workflow/mocks_test.go:MockClaudeExecutor'
    test_helpers:
      - name: 'newTestOrchestratorWithSpecName'
        file: 'internal/workflow/workflow_test.go'
        line: 3296
      - name: 'writeTestTasks'
        file: 'internal/workflow/workflow_test.go'
        line: 3448
  large_files:
    - path: 'internal/workflow/workflow_test.go'
      strategy: 'grep for function names, use offset/limit reads'
  coverage:
    baseline: '79.4%'
    target: '85%'
    zero_coverage_functions:
      - 'PromptUserToContinue:preflight.go:117'
      - 'runPreflightChecks:workflow.go:217'
```

### 2. Add notes.yaml validation (internal/validation/notes_yaml.go)

Create minimal validation that accepts the structure above. Notes.yaml is informational, not prescriptive, so validation should be permissive.

### 3. Update implement.md template

Add section on using notes.yaml:

```markdown
## Spec Notes (notes.yaml)

If specs/<feature>/notes.yaml exists, read it BEFORE exploring the codebase.

It may contain:
- _discovered.test_infrastructure: Paths to mock scripts and test helpers
- _discovered.large_files: Reading strategies for oversized files
- _discovered.coverage: Baseline metrics and target functions

This cache saves rediscovery time across phases.
```

### 4. Update Phase 1 guidance

In implement.md, add Phase 1 responsibility:

```markdown
## Phase 1 Responsibilities

After completing Phase 1 setup tasks:
1. Create or update specs/<feature>/notes.yaml
2. Document discovered test infrastructure paths
3. Note any large files and reading strategies
4. Record coverage baseline if measured

This benefits subsequent phases.
```

### 5. Bundle notes.yaml in phase context

Update phase context generation to include notes.yaml content if it exists.

## Acceptance Criteria
- [ ] notes.yaml validation exists and is permissive
- [ ] implement.md documents notes.yaml usage
- [ ] Phase 1 guidance includes notes.yaml creation
- [ ] Phase context generation bundles notes.yaml
- [ ] Example notes.yaml structure documented

## Non-Functional Requirements
- notes.yaml is OPTIONAL - not required for workflow
- Validation does not enforce specific fields
- Backward compatible with specs lacking notes.yaml
- notes.yaml can be created manually or by Claude during Phase 1"
```

---

## Implementation Order

1. **Fix 1** (implement.md file discipline) - Immediate, high impact, low effort
2. **Fix 3** (specify/plan context efficiency) - Quick template updates
3. **Fix 4** (sandbox documentation) - Documentation only
4. **Fix 2** (phase context metadata) - Requires Go code changes
5. **Fix 5** (large file hints) - Schema and template changes
6. **Fix 6** (notes.yaml caching) - New artifact type

---

## Success Metrics

After implementing these fixes, target metrics:

| Metric | Current | Target |
|--------|---------|--------|
| Duplicate file reads/session | 5-23 | ≤2 |
| Checklists checks (non-existent) | 10-49/session | 0-1 |
| Sandbox workarounds/session | 15 | 0 |
| Redundant artifact reads after phase context | 18-57 | 0 |
| Token waste/session | 30-50K | <10K |
| Test infrastructure rediscovery | Every phase | Phase 1 only |
