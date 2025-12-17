# Changelog

All notable changes to autospec will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/ariel-frischer/autospec/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/ariel-frischer/autospec/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/ariel-frischer/autospec/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/ariel-frischer/autospec/releases/tag/v0.1.0
