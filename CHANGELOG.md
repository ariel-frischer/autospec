# Changelog

All notable changes to autospec will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.5.0] - 2025-12-18

### Added
- `config set/get/toggle/keys` subcommands for CLI-based configuration management with `--user` and `--project` scope flags
- `--max-retries, -r` flag for `plan`, `tasks`, `constitution`, and `checklist` commands to override config retry limit

### Changed
- Improved internal codebase structure for faster future development and better reliability

### Fixed
- Description propagation in `run -a` now matches `autospec all` behavior (only specify stage receives description)

## [0.4.0] - 2025-12-18

### Added
- GitHub Pages documentation website with architecture overview, internals guide, FAQ, and troubleshooting pages
- `ContextMeta` struct to reduce redundant artifact file reads during phase execution
- `task block` and `task unblock` commands to mark tasks as blocked with documented reasons
- `BlockedReason` field in tasks.yaml to track why tasks are blocked (with validation warnings when missing)
- `risks` section in plan.yaml for documenting implementation risks and mitigation strategies
- Schema validation for YAML artifacts (validates structure, not just existence)
- `notes` field in tasks.yaml for additional task context (max 1000 chars)

### Changed
- CLI commands reorganized into subpackages (`stages/`, `config/`, `util/`, `admin/`, `shared/`) for improved maintainability
- Documentation restructured into feature cards for better presentation
- Custom sidebar styles for improved layout and usability
- Pre-flight validation now distinguishes between missing and invalid artifacts with specific error messages
- Retry state resets automatically when starting the specify stage

### Fixed
- Retry context instructions now dynamically injected only during retries (reduces token waste on first-run executions)
- Improved artifact validation shows both missing and invalid files in error output

## [0.3.2] - 2025-12-17

### Added
- `sauce` command to display the project source URL

### Changed
- Installer shows download progress bar for better visibility
- Default installation directory changed to `~/.local/bin`
- Installer now backs up existing binary before upgrading

### Fixed
- Improved installer reliability with better error handling and temp file cleanup
- Fixed POSIX compatibility issues in installer color output

## [0.3.1] - 2025-12-16

### Added
- ASCII art logo in installer

### Changed
- Installer uses `sh` instead of `bash` for better compatibility

## [0.3.0] - 2025-12-16

### Added

- `history` command with two-phase logging, status tracking, and `--status` filter
- Cross-platform notifications for command/stage completion (macOS, Linux)
- Claude settings validation and automatic permission configuration
- Profile management system for configuration presets
- Lifecycle wrapper for CLI commands (timing, notifications, history)
- Context injection for phase execution (performance optimization)
- Task-level execution mode with `--tasks` and `--from-task` flags
- `--single-session` flag for legacy single-session execution
- `--from-phase` and `--phase` flags for phase-level control
- `implement_method` config option for default execution mode
- Prerequisite validation for CLI commands (pre-flight artifact checks)
- Artifact validation for analysis, checklist, and constitution YAML files
- Optional stage commands: `constitution`, `clarify`, `checklist`, `analyze`
- `run` command with stage selection flags (`-s`, `-p`, `-t`, `-i`, `-a`, `-n`, `-r`, `-l`, `-z`)
- `--dry-run` flag for previewing actions
- `--debug` flag for verbose logging
- `update-task` command for task status management
- Spec status tracking with automatic completion marking
- `skip_confirmations` config and `AUTOSPEC_YES` environment variable
- `config migrate` command for config file migration
- Custom Claude command support with `{{PROMPT}}` placeholder
- claude-clean integration for readable streaming output
- Auto-updates to spec.yaml and tasks.yaml during execution
- Phase-isolated sessions (80%+ cost savings on large specs)
- Quickstart guide with interactive demo script
- Internals documentation guide
- Checklists documentation for requirement validation
- Shell completion support (bash, zsh, fish)

### Changed

- Renamed "phase" to "stage" throughout codebase for clarity
- Dropped Windows support; WSL recommended
- Long-running notification threshold: 30s → 2 minutes
- Renamed `full` command to `all`
- Refactored tests to map-based table-driven pattern
- Improved error handling with context wrapping

### Fixed

- Constitution requirement checks across all commands
- Task status tracking during implementation
- Artifact dependency validation
- Claude settings configuration in `init` command

## [0.2.0] - 2025-01-15

### Added

- Workflow progress indicators with spinners
- Command execution timeout support
- Timeout configuration via `AUTOSPEC_TIMEOUT` environment variable
- Exit code 5 for timeout errors
- Configurable timeout in config files (0 for infinite, 1-604800 seconds)

### Changed

- Enhanced workflow orchestration with better error handling
- Improved phase execution with clearer status messages

## [0.1.0] - 2025-01-01

### Added

- Initial Go binary implementation
- CLI commands: `workflow`, `specify`, `plan`, `tasks`, `implement`, `status`, `init`, `config`, `doctor`, `version`
- Cobra-based command structure with global flags
- Workflow orchestration (specify -> plan -> tasks -> implement)
- Hierarchical configuration system using koanf
- Configuration sources: environment variables, local config, global config, defaults
- Retry management with persistent state tracking
- Atomic file writes for retry state consistency
- Validation system with <10ms performance contract
- Spec detection from git branch or most recently modified directory
- Git integration helpers
- Pre-flight dependency checks (claude, specify CLIs)
- Claude execution modes: CLI, API, and custom command
- Custom command support with `{{PROMPT}}` placeholder
- Exit code conventions for programmatic use
- Cross-platform builds (Linux, macOS, Windows)

### Changed

- Migrated from bash scripts to Go binary
- Replaced manual validation with automated checks

### Deprecated

- Legacy bash scripts in `scripts/` (scheduled for removal)
- Bats tests in `tests/` (being replaced by Go tests)

[Unreleased]: https://github.com/ariel-frischer/autospec/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/ariel-frischer/autospec/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/ariel-frischer/autospec/compare/v0.3.2...v0.4.0
[0.3.2]: https://github.com/ariel-frischer/autospec/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/ariel-frischer/autospec/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/ariel-frischer/autospec/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/ariel-frischer/autospec/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/ariel-frischer/autospec/releases/tag/v0.1.0
