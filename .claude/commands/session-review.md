---
description: Review NEW Claude conversations to identify autospec improvement opportunities.
---

## User Input

```text
$ARGUMENTS
```

## Purpose

Analyze **NEW, UNREVIEWED** Claude Code conversations to identify patterns, inefficiencies, and improvement opportunities for the autospec workflow.

## CRITICAL: Only Analyze NEW Conversations

**DO NOT re-analyze conversations already documented in observations.md.**

1. Check `.dev/feedback/reviewed.txt` - these are ALREADY analyzed
2. Only process conversations NOT in that file
3. The helper script's `unreviewed` command filters automatically

## Context Files

- **Observations**: `.dev/tasks/observations.md` - Central observations document
- **Reviewed Registry**: `.dev/feedback/reviewed.txt` - Tracks which conversations have been analyzed
- **Helper Script**: `scripts/parse-claude-conversation.sh` - CLI tool for parsing conversations

## Token-Efficient Parsing Strategy

**CRITICAL**: Do NOT read entire conversation files. Follow this strategy:

### 1. Use Helper Script for Initial Triage

```bash
# List unreviewed autospec conversations
./scripts/parse-claude-conversation.sh unreviewed

# Get metadata about a specific conversation
./scripts/parse-claude-conversation.sh info ~/.claude/projects/<project>/<id>.jsonl

# Check tool usage patterns (quick overview)
./scripts/parse-claude-conversation.sh summary ~/.claude/projects/<project>/<id>.jsonl

# Detect common inefficiency patterns automatically
./scripts/parse-claude-conversation.sh issues ~/.claude/projects/<project>/<id>.jsonl
```

### 2. Targeted Parsing (Only When Needed)

```bash
# Parse first 300 lines (usually captures setup + first tasks)
./scripts/parse-claude-conversation.sh parse <file> 300

# Search for specific patterns without full parse
cclean -s plain <file> | grep -E "(redundant|retry|error|failed)" | head -20
```

### 3. Pattern Recognition Keywords

Look for these in parsed output:
- `phase-*.yaml` followed by `spec.yaml`/`tasks.yaml` → Redundant reads
- `checklists` → Unnecessary directory checks
- `exceeds maximum` → Large file handling issues
- `language server not initialized` → Serena MCP failures
- `dangerouslyDisableSandbox` → Sandbox workarounds
- Multiple `file_path:` for same file → Duplicate reads

## Workflow

### Step 1: Find Unreviewed Conversations

```bash
./scripts/parse-claude-conversation.sh unreviewed
```

This shows conversations that haven't been analyzed yet, filtered against `.dev/feedback/reviewed.txt`.

### Step 2: Quick Triage

For each unreviewed conversation:

```bash
# Quick metadata
./scripts/parse-claude-conversation.sh info <file>

# Automated issue detection
./scripts/parse-claude-conversation.sh issues <file>
```

### Step 3: Document Findings

Add observations to `.dev/tasks/observations.md` under the appropriate section:
- Per-conversation analysis (if notable issues found)
- Cross-session patterns (if pattern affects multiple sessions)
- Proposed improvements (if solution is clear)

### Step 4: Mark as Reviewed

```bash
./scripts/parse-claude-conversation.sh mark <short_id> <command_type>
# Example: ./scripts/parse-claude-conversation.sh mark 548be630 implement
```

## Argument Handling

Interpret `$ARGUMENTS` as:

- **Empty**: Run unreviewed check, show next conversation to analyze
- **`status`**: Show count of reviewed vs unreviewed conversations
- **`<id>`**: Analyze specific conversation by short ID (first 8 chars)
- **`<path>`**: Analyze specific conversation file
- **`all`**: Generate summary of all reviewed conversations
- **`patterns`**: Focus on cross-session pattern analysis
- **`<project>`**: Filter to specific project (e.g., `-home-ari-repos-autospec`)

## Issue Categories to Track

1. **Redundant Context Reading**
   - Phase context file read → then individual artifacts read
   - Same file read multiple times in session

2. **Large File Handling**
   - Token limit exceeded errors
   - Files requiring offset/limit workarounds

3. **Infrastructure Rediscovery**
   - Test helpers discovered repeatedly across phases
   - Mock script paths re-discovered

4. **Tool Failures & Fallbacks**
   - Serena MCP errors → standard tool fallback
   - Sandbox restrictions → retry with override

5. **Unnecessary Checks**
   - Checklists directory check when none exist
   - Validation for non-existent conditions

## Output Format

When documenting in observations.md, use this format:

```markdown
### File: <short_id> (<command_type> - <feature_name>)
**Command:** `/autospec.<type>`
**Issues:**
- Issue 1 description
- Issue 2 description

**Recommendations:**
- Recommendation 1
- Recommendation 2
```

## Important Notes

- **DO NOT** read raw JSONL files directly - always use `cclean` or the helper script
- **SKIP** conversations already in `.dev/feedback/reviewed.txt`
- **PRIORITIZE** implement sessions (highest token usage)
- **FOCUS** on actionable improvements (template changes, schema updates, code fixes)
- **QUANTIFY** issues when possible (e.g., "15K tokens wasted on redundant reads")
