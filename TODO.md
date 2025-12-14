# TODO

## Backlog

| ID | Task | Priority | Status |
|----|------|----------|--------|
| 009 | [Implement Retry Loop](tasks/009-implement-retry-loop.md) | High | Todo |
| 010 | [Parallel Sandbox Execution](tasks/010-parallel-sandbox-execution.md) | Medium | Research Complete |

## Bugs/Issues

- [ ] Setting ANTHROPIC_API_KEY="" is no longer relevant with the new claude
      setup which CAN switch between API/vs max plan usage with or without api keys through web login auth 
      so rm that and any config options for it.

- Our spinner implementation doesnt fucking work because its a cli not a tui
- also we will not build a tui
- [X] Fix spinner scrollback pollution when Claude outputs interactively
  - Spinner frames appear throughout Claude's output in scrollback
  - Root cause: Spinner writes to stdout concurrently with Claude's streaming output
  - Proposed fix: Configure spinner to write to stderr instead of stdout
  - This keeps progress visible but prevents interference between streams

## Cleanup Tasks

- [ ] Remove legacy bash scripts from `scripts/` directory (deprecated in favor of Go binary)
  - [ ] Remove `scripts/lib/speckit-validation-lib.sh`
  - [ ] Remove `scripts/speckit-workflow-validate.sh`
  - [ ] Remove `scripts/speckit-implement-validate.sh`
  - [ ] Remove `scripts/hooks/` directory (or migrate to Go if hooks are still needed)
  - [ ] Remove corresponding bats tests in `tests/`
  - [ ] Update documentation references (if any remain)

## Feature Improvements

- [X] Implement timeout functionality for Claude CLI command execution
  - [X] Use `timeout` config setting to abort long-running commands
  - [X] Add context with deadline to command execution
  - [X] Update documentation when implemented

- [X] Add `autospec doctor` command for dependency checking
  - [X] Check for Claude CLI installation
  - [X] Check for Specify CLI installation
  - [X] Check for Git repository
  - [X] Verify config file locations
  - [X] Check specs directory exists

- [X] Add progress indicators during workflow execution
  - [X] Show [1/3], [2/3], [3/3] progress
  - [X] Add spinners for long-running operations
  - [X] Show checkmarks when phases complete

- [X] Unify setup into `autospec init` (single command for all initialization)
  - [X] Flow (in order):
    1. Auto-install missing `.autospec/scripts/` and claude commands (no prompt)
    2. If config exists: ask "Update config?" → prompt for changes if yes
    3. If no config: create with sensible defaults
  - [X] Make `autospec commands install` redundant (deprecated)
  - [X] `autospec init --force` to overwrite config without prompting

- [ ] Enhance `autospec status` output
  - [ ] Add visual progress indicators (✓, ⏳, ✗)
  - [ ] Show percentage completion
  - [ ] Better formatting for task lists
  - [ ] Add `--json` flag support if missing

- [ ] Improve CLI help and examples
  - [ ] Add usage examples to each command's help
  - [ ] Better error messages with actionable next steps
  - [ ] Add `--dry-run` flag to more commands

- [ ] Add example/demo commands
  - [ ] `autospec example` - show example feature descriptions
  - [ ] `autospec demo` - run demo workflow with sample data
  - [ ] `autospec templates` - list available templates

## Documentation

- [ ] Add troubleshooting section for common errors
- [ ] Create video tutorial or GIF demos
- [ ] Add more use case examples
- [ ] Document all CLI flags and options comprehensively
- [ ] Add FAQ.md with common questions and troubleshooting
- [ ] Create DEVELOPMENT.md separate from CONTRIBUTORS.md with local setup
- [ ] Add INTEGRATION.md with CI/CD examples and pre-commit hooks

## Repository Infrastructure

### GitHub Integration
- [X] Create `.github/` directory structure
- [X] Add issue templates (`.github/ISSUE_TEMPLATE/`)
  - [X] bug_report.md
  - [X] feature_request.md
  - [X] config.yml
- [X] Add pull request template (`.github/PULL_REQUEST_TEMPLATE.md`)
- [X] Add GitHub Actions workflows (`.github/workflows/`)
  - [X] ci.yml - Run tests and linting on every PR
  - [X] release.yml - Automated releases with goreleaser
  - [X] docs.yml - Deploy docs to GitHub Pages

### Community & Governance
- [ ] Add CHANGELOG.md for version tracking
- [ ] Create CONTRIBUTING.md (distinct from CONTRIBUTORS.md)
  - [ ] How to submit issues/PRs
  - [ ] Development setup
  - [ ] Code style guidelines
  - [ ] Commit message conventions
  - [ ] Testing requirements
- [ ] Add SECURITY.md with vulnerability reporting policy
- [ ] Add CODE_OF_CONDUCT.md (Contributor Covenant or similar)

### Examples & Demos
- [ ] Create `examples/` directory
  - [ ] Add `simple-feature/` with complete example spec
  - [ ] Add `config-examples/` with various config.json examples
  - [ ] Add examples README.md guide
- [ ] Create `assets/` directory for media
  - [ ] Terminal recordings (asciinema)
  - [ ] GIFs of workflow execution
  - [ ] Architecture diagrams

### Installation & Distribution
- [ ] Add installation scripts
  - [ ] install.sh for Unix-like systems (curl | sh installer)
  - [ ] install.ps1 for Windows (PowerShell installer)
- [X] Add `.goreleaser.yml` for automated releases
  - [X] Multi-platform builds
  - [X] GitHub releases with binaries
  - [ ] Homebrew tap integration
  - [X] Checksums and signatures
- [ ] Add badges to README.md
  - [ ] CI status badge
  - [ ] Go Report Card
  - [ ] License badge
  - [ ] Release version badge

### Development Tools
- [ ] Add Docker support
  - [ ] Dockerfile for running autospec in container
  - [ ] docker-compose.yml for integration testing
- [ ] Add `.pre-commit-config.yaml` for pre-commit hooks
- [ ] Enhance Makefile
  - [ ] Add `make release` target
  - [ ] Add `make snapshot` target
  - [ ] Add `make coverage` target with HTML output
- [ ] Create `benchmarks/` directory
  - [ ] Baseline benchmarks
  - [ ] Regression testing
  - [ ] Performance tracking over releases
