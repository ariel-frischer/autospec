---
description: Prepare a release (changelog, version bump, tag commands).
---

## User Input

```text
$ARGUMENTS
```

## Instructions

1. Read release guidelines: `.release/CLAUDE.md`
2. Check current state:
   ```bash
   git describe --tags --abbrev=0 2>/dev/null || echo "no tags yet"
   git branch --show-current
   grep "## \[" CHANGELOG.md | head -5
   ```

## Task

1. **Verify** changelog is ready (no pending updates needed)
2. **Confirm** version number with user (unless provided in arguments)
3. **Update CHANGELOG.md**:
   - Rename `[Unreleased]` → `[X.Y.Z] - YYYY-MM-DD`
   - Add fresh `[Unreleased]` section above
   - Update version links at bottom
4. **Test** extraction: `.release/extract-changelog.sh X.Y.Z`
5. **Show** release commands (don't execute unless asked):
   ```bash
   git add CHANGELOG.md
   git commit -m "chore: release vX.Y.Z"
   git checkout main
   git merge dev
   git tag -a vX.Y.Z -m "Release vX.Y.Z"
   git push origin main --tags
   ```

Remember: Tag on `main` after merging, not on `dev`.

## Argument Handling

Interpret `$ARGUMENTS` as:
- Version number (e.g., "0.3.0") → use that version, skip confirmation
- "check" → only verify readiness, don't make changes
- "dry-run" → show what would happen without executing
- Other text → additional context or instructions
- Empty → interactive mode, ask what version
