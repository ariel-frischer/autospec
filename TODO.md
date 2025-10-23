# TODO

## Cleanup Tasks

- [ ] Remove legacy bash scripts from `scripts/` directory (deprecated in favor of Go binary)
  - [ ] Remove `scripts/lib/speckit-validation-lib.sh`
  - [ ] Remove `scripts/speckit-workflow-validate.sh`
  - [ ] Remove `scripts/speckit-implement-validate.sh`
  - [ ] Remove `scripts/hooks/` directory (or migrate to Go if hooks are still needed)
  - [ ] Remove corresponding bats tests in `tests/`
  - [ ] Update documentation references (if any remain)

## Feature Improvements

- [ ] Add `autospec doctor` command for dependency checking
  - [ ] Check for Claude CLI installation
  - [ ] Check for Specify CLI installation
  - [ ] Check for Git repository
  - [ ] Verify config file locations
  - [ ] Check specs directory exists

- [ ] Improve `autospec setup` or enhance `init` command
  - [ ] Interactive setup wizard
  - [ ] Dependency installation guidance
  - [ ] Run `specify init` automatically if needed
  - [ ] Verify installation after setup

- [ ] Add progress indicators during workflow execution
  - [ ] Show [1/3], [2/3], [3/3] progress
  - [ ] Add spinners for long-running operations
  - [ ] Show checkmarks when phases complete

- [ ] Enhance `autospec status` output
  - [ ] Add visual progress indicators (✓, ⏳, ✗)
  - [ ] Show percentage completion
  - [ ] Better formatting for task lists
  - [ ] Add `--json` flag support if missing

- [ ] Add shell completion support
  - [ ] Generate bash completion
  - [ ] Generate zsh completion
  - [ ] Generate fish completion
  - [ ] Add installation instructions to README

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
