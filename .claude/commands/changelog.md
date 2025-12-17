---
description: Draft changelog entries from commits, grouped and user-friendly.
---

## User Input

```text
$ARGUMENTS
```

## Instructions

1. Read the changelog guidelines: `.release/CLAUDE.md`
2. Get commits since last tag:
   ```bash
   git log $(git describe --tags --abbrev=0 2>/dev/null || echo "HEAD~50")..HEAD --oneline
   ```
3. Review current `[Unreleased]` section in `CHANGELOG.md`

## Task

Based on commits and guidelines:

1. **Group commits** into meaningful user-facing changes (many commits → single entry)
2. **Draft entries** that are user-friendly, benefit-focused, concise
3. **Categorize** into: Added, Changed, Fixed, Removed
4. **Show proposed entries** before making changes
5. **Ask version number** if preparing a release

Do NOT include internal refactors, test fixes, or CI changes unless they affect users.

## Argument Handling

Interpret `$ARGUMENTS` as:
- Version number (e.g., "0.3.0") → prepare changelog for that version
- "draft" → only show proposed entries, don't edit
- "apply" → update CHANGELOG.md directly
- Other text → additional context or instructions
- Empty → interactive mode, ask what to do
