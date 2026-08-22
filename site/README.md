# autospec Documentation Site

This directory contains the Jekyll-based documentation site for autospec, deployed to [ariel-frischer.github.io/autospec](https://ariel-frischer.github.io/autospec/).

## How It Works

**Source of truth:** `docs/public/` and `docs/internal/`

The site is generated from two sources:
1. **Static pages** - Manually maintained in `site/` (index.md, quickstart.md, etc.)
2. **Synced docs** - Generated from `docs/` by `scripts/sync-docs-to-site.sh`

### Directory Structure

```
docs/
├── public/           # User-facing docs → site/{reference,guides}/
│   ├── agents.md
│   ├── quickstart.md
│   └── ...
└── internal/         # Contributor docs → site/contributing/
    ├── architecture.md
    ├── go-best-practices.md
    └── ...

site/
├── index.md          # Static - manually maintained
├── quickstart.md     # Static - manually maintained
├── reference/        # Generated from docs/public/
├── guides/           # Generated from docs/public/
└── contributing/     # Generated from docs/internal/
```

The generated public pages include agent, Claude, Codex, OpenCode, shell
completion, timeout, render-command, and prerequisite documentation. Keep
`docs/public/` as the source of truth for those pages and run the sync script
after changing them. The landing page, quickstart, CLI reference, and
configuration reference remain web-specific pages, so changes to their
duplicated setup or command examples should be checked against the matching
public source docs in the same change.

## Deployment

Deployment is fully automated via GitHub Actions (`.github/workflows/docs.yml`):

1. Push to `main` with changes to `docs/**` or `site/**`
2. GitHub Actions runs `scripts/sync-docs-to-site.sh`
3. Jekyll builds the site
4. Site deploys to GitHub Pages

**Generated files are not committed** - they're in `.gitignore` and created during CI.

## Local Development

```bash
# Generate docs from docs/ to site/
./scripts/sync-docs-to-site.sh

# Serve locally
cd site
bundle install
bundle exec jekyll serve --livereload
```

Open http://localhost:4000/autospec/

## Adding New Docs

### User-facing docs (public)
1. Add to `docs/public/`
2. Update `scripts/sync-docs-to-site.sh` with the new file mapping
3. Add the generated destination to `site/.gitignore` when the page is created during CI
4. Update `site/{reference,guides}/index.md` with a link

### Contributor docs (internal)
1. Add to `docs/internal/`
2. Update `scripts/sync-docs-to-site.sh` with the new file mapping
3. Update `site/contributing/index.md` with a link

## Theme

Uses [Just the Docs](https://just-the-docs.com/) theme with dark mode.

Configuration in `_config.yml`.
