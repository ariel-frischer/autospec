# YAML-First Changelog Management

autospec uses a YAML-first approach for changelog management. `internal/changelog/changelog.yaml` is the single source of truth, and `CHANGELOG.md` is automatically generated from it.

## Workflow Overview

1. **Add**: Run `chlog add <category> "Description"` to update `internal/changelog/changelog.yaml`
2. **Sync**: Run `chlog sync` or `make changelog-sync` to regenerate `CHANGELOG.md`
3. **Check**: Run `chlog check` or `make changelog-check`
4. **Commit**: Commit the YAML source and generated markdown together

Never edit `CHANGELOG.md` directly—it is auto-generated.

## YAML Schema

The changelog follows the [Keep a Changelog](https://keepachangelog.com/) format:

```yaml
project: autospec
versions:
  unreleased:
    added:
      - "New feature description"
    fixed:
      - "Bug fix description"

  0.9.0:
    date: "2026-01-16"
    added:
      - "Feature added in this release"
    changed:
      - "Changed behavior description"
    deprecated:
      - "Feature being deprecated"
    removed:
      - "Removed feature"
    fixed:
      - "Bug fix"
    security:
      - "Security fix"
```

### Categories

| Category | Description |
|----------|-------------|
| `added` | New features |
| `changed` | Changes to existing functionality |
| `deprecated` | Features marked for removal |
| `removed` | Features that were removed |
| `fixed` | Bug fixes |
| `security` | Security-related changes |

### Version Format

- Use bare semver format in YAML: `0.9.0` (not `v0.9.0`)
- The CLI accepts either format: `autospec changelog v0.9.0` or `autospec changelog 0.9.0`
- Use `unreleased` for changes not yet released
- Released versions require a `date` field in `YYYY-MM-DD` format

## CLI Commands

### View Changelog

```bash
# Show 5 most recent entries (default)
autospec changelog

# Show all entries for a specific version
autospec changelog v0.9.0
autospec changelog 0.9.0       # v prefix optional

# Show unreleased changes
autospec changelog unreleased

# Show last N entries
autospec changelog --last 10

# Plain text output (no colors/icons)
autospec changelog --plain
```

### Extract Release Notes

Extract changelog entries for a specific version in markdown format, useful for GitHub release notes:

```bash
autospec changelog extract v0.9.0
```

This outputs markdown suitable for use in CI/CD release workflows.

### Sync and Validate

```bash
# Add a new unreleased entry
chlog add changed "Describe the user-facing change"

# Regenerate CHANGELOG.md from YAML
chlog sync
# or:
autospec changelog sync
# or: make changelog-sync

# Validate CHANGELOG.md matches YAML
chlog check
# or:
autospec changelog check
# or: make changelog-check
```

The sync operation is idempotent—running it multiple times produces identical output.

## Make Targets

| Target | Description |
|--------|-------------|
| `make changelog-sync` | Regenerate CHANGELOG.md from YAML |
| `make changelog-check` | Validate CHANGELOG.md matches YAML (used in CI) |

## chlog Configuration

This repository includes `.chlog.yaml` so `chlog` commands run from the repository root use:

| Setting | Value |
|---------|-------|
| `changelog_file` | `internal/changelog/changelog.yaml` |
| `public_file` | `CHANGELOG.md` |
| `strict_categories` | `true` |

## Embedded Changelog

The changelog is embedded in the autospec binary at build time using Go's `embed` directive. This allows:

- Viewing changelog without network access
- Showing changes up to when the binary was built
- Integration with update commands to show what's new

## Adding a Changelog Entry

1. Run `chlog add` with the appropriate category:

```bash
chlog add added "Your new feature description"
chlog add fixed "Your bug fix description"
```

2. Confirm the entry appears under the appropriate category in `unreleased`:

```yaml
versions:
  unreleased:
    added:
      - "Your new feature description"
```

3. Run `chlog sync` or `make changelog-sync` to update `CHANGELOG.md`
4. Commit both files

## Release Workflow

When preparing a release:

1. The `/release` command moves `unreleased` entries to a new version
2. A fresh `unreleased` section is created for future changes
3. `make changelog-sync` regenerates `CHANGELOG.md`
4. `autospec changelog extract <version>` provides release notes for GitHub

## CI Integration

Add `make changelog-check` to your CI pipeline to ensure `CHANGELOG.md` stays in sync:

```yaml
- name: Check changelog sync
  run: make changelog-check
```

This fails the build if someone manually edited `CHANGELOG.md` without updating the YAML source.

## Troubleshooting

### "CHANGELOG.md is out of sync with changelog.yaml"

Run `make changelog-sync` to regenerate `CHANGELOG.md` from the YAML source.

### "cannot find changelog.yaml source file"

Ensure `internal/changelog/changelog.yaml` exists in your repository.

### "Version not found"

The specified version doesn't exist in the changelog. Run `autospec changelog` to see available versions.
