# Release & Changelog Guidelines

Instructions for updating CHANGELOG.md before releases.

## Changelog Philosophy

**Target audience**: End users, not developers. Write for someone who wants to know "what's new" without reading code.

## Update Process

1. Review commits since last release: `git log $(git describe --tags --abbrev=0)..HEAD --oneline`
2. Group related commits into single entries (10 commits about "validation" → one "Improved validation system" entry)
3. Update `## [Unreleased]` section
4. When releasing, rename `[Unreleased]` to `[X.Y.Z] - YYYY-MM-DD` and add fresh `[Unreleased]` above

## Writing Style

**Do:**
- Lead with user benefit: "Faster startup time" not "Optimized init sequence"
- Use active voice: "Added dark mode" not "Dark mode was added"
- Be specific but brief: "Export to CSV and JSON" not "Export functionality"
- Group aggressively: 5 error-handling commits → "Better error messages"

**Don't:**
- Mention internal refactors unless they affect users
- Include technical jargon (no "refactored X to use Y pattern")
- List every small fix separately
- Reference PR/issue numbers in the entry text

## Entry Format

```markdown
### Added
- New feature description (user benefit)

### Changed
- What changed and why it matters to users

### Fixed
- What was broken, now works

### Removed
- What's gone (mention migration path if needed)
```

## Grouping Examples

**Bad** (too granular):
```markdown
- Fixed validation error message formatting
- Fixed validation for empty strings
- Added validation for special characters
- Fixed edge case in email validation
```

**Good** (grouped):
```markdown
- Improved input validation with clearer error messages
```

**Bad** (too technical):
```markdown
- Refactored retry logic to use exponential backoff with jitter
- Migrated from sync.Mutex to sync.RWMutex for better concurrency
```

**Good** (user-focused):
```markdown
- More reliable retries on network failures
- Faster performance under heavy load
```

## Version Bumping

- **Patch** (0.0.X): Bug fixes only
- **Minor** (0.X.0): New features, backward compatible
- **Major** (X.0.0): Breaking changes

## Release Workflow

**Branch strategy:** Development happens on `dev`, releases from `main`.

```bash
# 1. Update CHANGELOG.md on dev
#    - Rename [Unreleased] → [X.Y.Z] - YYYY-MM-DD
#    - Add fresh [Unreleased] section above
#    - Update version links at bottom

# 2. Commit changelog on dev
git add CHANGELOG.md
git commit -m "chore: release vX.Y.Z"

# 3. Merge to main
git checkout main
git merge dev

# 4. Tag on main (not dev!)
git tag -a vX.Y.Z -m "Release vX.Y.Z"

# 5. Push main + tag
git push origin main --tags
```

**Important:** Always tag on `main` after merging. Tags point to commits, not branches—tagging on `dev` before merge means the release builds from a commit not on `main`.

**Local testing (no publish):**
```bash
goreleaser release --snapshot --clean    # Test build
.release/extract-changelog.sh X.Y.Z      # Test changelog extraction
```

## Pre-Release Checklist

1. [ ] All commits reviewed and grouped
2. [ ] Entries are user-friendly, not technical
3. [ ] Date is correct (YYYY-MM-DD)
4. [ ] Version links updated at bottom of CHANGELOG.md
5. [ ] `[Unreleased]` section ready for next cycle
6. [ ] Merged to `main` before tagging
7. [ ] Tag created on `main`, not `dev`
