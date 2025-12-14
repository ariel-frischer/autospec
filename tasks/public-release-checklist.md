# Public Release Checklist

Checklist for preparing Auto Claude SpecKit for public release.

## Documentation

- [x] **README.md** - Comprehensive documentation with badges
  - [x] Installation instructions (install.sh, go install, manual)
  - [x] Usage examples for all commands
  - [ ] Screenshots/GIFs demonstrating workflow (optional)
  - [x] Badges (CI, Go Report Card, License, Release)
- [x] **CONTRIBUTING.md** - Contribution guidelines
- [x] **CHANGELOG.md** - Notable changes documented
- [x] **SECURITY.md** - Security policy and reporting
- [x] **LICENSE** - MIT license
- [ ] **CLAUDE.md** - Decide: keep public or move to .dev/
- [x] **PREREQUISITES.md** - Dependencies documented

## Code Quality

- [x] All tests passing (`make test`)
- [ ] Code formatted (`go fmt ./...`)
- [ ] No linting issues (`go vet ./...`)
- [ ] No TODO/FIXME comments that shouldn't be public
- [ ] Remove any debug/test code
- [ ] Review error messages for clarity

## Security & Secrets

- [x] No hardcoded credentials, API keys, or tokens (verified)
- [ ] No personal paths or usernames in code
- [x] .gitignore covers sensitive files
- [x] No secrets in git history (verified with git log -S)

## Repository Hygiene

### Git History Audit

- [x] **Remove binary from history**
  - ~~`autospec-test` (9.7MB) removed with git filter-repo~~
- [x] No embarrassing/unprofessional commit messages (reviewed)
- [x] No WIP/fixup commits that need cleanup

### Branch Cleanup

Current branches to consider removing before public release:
- [ ] `001-go-binary-migration` (superseded)
- [ ] `002-go-binary-migration` (superseded)
- [ ] `003-command-timeout` (merged?)
- [ ] `004-workflow-progress-indicators` (merged?)
- [ ] `005-high-level-docs` (merged?)
- [ ] `006-github-issue-templates` (merged?)
- [ ] `007-yaml-structured-output` (merged?)
- [ ] `008-flexible-phase-workflow` (merged?)
- [ ] `009-optional-phase-commands` (merged?)
- [ ] `010-cli-help-examples` (merged?)
- [ ] `011-yaml-config-migration` (current - merge to main first)

### Other Cleanup

- [ ] Ensure main branch is default
- [ ] Remove unused files at root
- [ ] Tags are semantic versioned
- [ ] .gitattributes configured (if needed)

## GitHub Setup

- [x] **.github/ directory**
  - [x] `ISSUE_TEMPLATE/` (bug report, feature request)
  - [x] `PULL_REQUEST_TEMPLATE.md`
  - [ ] `CODEOWNERS` (optional)
  - [ ] `FUNDING.yml` (optional)
  - [ ] `dependabot.yml` for dependency updates

- [x] **GitHub Actions CI/CD**
  - [x] `ci.yml` - Build/test on push/PR
  - [x] `docs.yml` - Documentation deployment
  - [x] `release.yml` - Release workflow with GoReleaser

- [ ] **Repository settings** (after pushing)
  - [ ] Description set
  - [ ] Topics/tags (go, cli, claude, ai, workflow, automation)
  - [ ] Website URL
  - [ ] Social preview image

## Community Standards

- [x] **CODE_OF_CONDUCT.md** - Added (Contributor Covenant v2.0)
- [ ] Issue labels configured (bug, enhancement, good first issue, etc.)
- [ ] Branch protection rules for main

## Release Preparation

- [ ] Version number set correctly
- [ ] Create initial release v1.0.0 with binaries
- [ ] Verify binary checksums
- [ ] Test installation from scratch:
  - [ ] `curl -fsSL ... | bash` works
  - [ ] `go install` works
  - [ ] Manual build works

## Pre-Release Final Checks

- [ ] Clone repo fresh and verify build works
- [ ] Run full test suite
- [ ] Check all links in documentation
- [ ] Review README as a new user would
- [ ] Verify `autospec doctor` works on clean system

## Post-Release

- [ ] Announce release (if applicable)
- [ ] Monitor issues for early feedback
- [ ] Set up notifications for new issues/PRs

---

## Current Status

### Already Complete
- [x] README.md with comprehensive documentation
- [x] LICENSE (MIT)
- [x] CONTRIBUTING.md
- [x] SECURITY.md
- [x] CHANGELOG.md
- [x] CODE_OF_CONDUCT.md (Contributor Covenant v2.0)
- [x] .gitignore (updated with autospec-* patterns)
- [x] Makefile with build/test/lint targets
- [x] Makefile autobump commands (patch/minor/major/snapshot)
- [x] Tests exist and pass
- [x] .github/ with issue templates, PR template, workflows
- [x] CI/CD GitHub Actions (ci.yml, release.yml, docs.yml)
- [x] GoReleaser configuration (.goreleaser.yml)
- [x] README badges
- [x] No secrets in git history
- [x] **autospec-test binary removed from history** (9.7MB saved)
- [x] .dev/docs/ with release documentation (releases.md, autobump.md, install.md)

### Needs Attention (Priority Order)

1. ~~**Force push rewritten history to remote**~~ DONE

2. **Clean up feature branches**
   ```bash
   # Delete local branches
   git branch -D 001-go-binary-migration 002-go-binary-migration ...

   # Delete remote branches (after force push)
   git push origin --delete 001-go-binary-migration ...
   ```

3. ~~**Run tests and verify build**~~ DONE - All tests passing

4. ~~**Add CODE_OF_CONDUCT.md**~~ DONE

5. **Add dependabot.yml**

6. **Final review of CLAUDE.md** - decide if public-appropriate

7. ~~**Add .dev/docs/ with release documentation**~~ DONE
   - releases.md - Release process for GitHub/GitLab
   - autobump.md - Makefile version management
   - install.md - Unified install script docs

8. ~~**Add autobump Makefile commands**~~ DONE
   - `make patch` / `make p` - Bump patch version
   - `make minor` - Bump minor version
   - `make major` - Bump major version
   - `make snapshot` / `make s` - Local build (no publish)

### Notes

- History was rewritten - all commit SHAs changed
- Origin remote restored: git@gitlab.com:ariel-frischer/auto-claude-speckit.git
- Force push required to sync remote
