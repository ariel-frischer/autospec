# TODO

## Bugs/Issues

- [ ] Fix spinner scrollback pollution when Claude outputs interactively
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

- [ ] Improve `autospec setup` or enhance `init` command
  - [ ] Interactive setup wizard
  - [ ] Dependency installation guidance
  - [ ] Run `specify init` automatically if needed
  - [ ] Verify installation after setup

- [X] Add progress indicators during workflow execution
  - [X] Show [1/3], [2/3], [3/3] progress
  - [X] Add spinners for long-running operations
  - [X] Show checkmarks when phases complete

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
