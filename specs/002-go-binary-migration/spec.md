# Feature Specification: Go Binary Migration

**Feature Branch**: `002-go-binary-migration`
**Created**: 2025-10-22
**Status**: Draft
**Input**: User description: "Transform the current bash-based validation tool into a single, cross-platform Go binary"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Simple Installation (Priority: P1)

As a developer new to Auto Claude SpecKit, I want to install the tool with a single command and start using it immediately, without needing to configure shell environments, install multiple dependencies, or understand bash scripting.

**Why this priority**: This is the first interaction users have with the tool. A complex installation process creates friction and prevents adoption. This story delivers immediate value - a working tool in under 30 seconds.

**Independent Test**: Can be fully tested by running a single installation command (`go install` or downloading a binary) and immediately executing `autospec --version`. Delivers a working CLI tool that can display help and version information.

**Acceptance Scenarios**:

1. **Given** a developer with Go installed, **When** they run `go install github.com/username/autospec@latest`, **Then** the autospec binary is available in their PATH and `autospec --version` displays version information
2. **Given** a developer on Linux/macOS/Windows without Go, **When** they download the binary for their platform from releases, **Then** the binary runs without requiring additional dependencies
3. **Given** a developer wants to verify installation, **When** they run `autospec --help`, **Then** they see comprehensive usage documentation with available commands

---

### User Story 2 - Cross-Platform Compatibility (Priority: P1)

As a developer working on Windows, macOS, or Linux, I want the same tool and workflow to work identically across all platforms, so teams can collaborate regardless of operating system.

**Why this priority**: Currently the tool only works on Linux/macOS due to bash dependencies. Supporting Windows is critical for enterprise adoption and team collaboration. This is equally important as installation since platform lock-in prevents tool adoption.

**Independent Test**: Can be tested by running the same workflow (`autospec workflow "feature"`) on Windows, macOS, and Linux systems and verifying identical behavior and output. Delivers immediate value to Windows users who currently cannot use the tool.

**Acceptance Scenarios**:

1. **Given** a developer on Windows, **When** they run `autospec workflow "Add user auth"`, **Then** the workflow completes successfully without bash/jq/git dependencies
2. **Given** a developer on macOS, **When** they run the same command, **Then** they get identical output and behavior as Windows
3. **Given** a developer on Linux, **When** they run the same command, **Then** they get identical output and behavior as other platforms
4. **Given** a team using mixed operating systems, **When** they share configuration files (.autospec/config.json), **Then** the configuration works across all platforms without modification

---

### User Story 3 - Pre-Flight Validation (Priority: P2)

As a developer running autospec for the first time in a project, I want clear feedback if the project isn't properly initialized with SpecKit, with actionable instructions on how to fix the issue, so I don't waste time with cryptic errors.

**Why this priority**: Poor error messages create frustration and support burden. This enhances user experience but isn't blocking for basic functionality - users can still run commands if they know their setup is correct.

**Independent Test**: Can be tested by running `autospec workflow` in an uninitialized directory and verifying helpful warning messages appear with specific fix instructions. Delivers immediate value through better error handling.

**Acceptance Scenarios**:

1. **Given** a project directory without .claude/commands/, **When** a developer runs `autospec workflow "feature"`, **Then** they see a warning listing missing directories and recommended setup steps
2. **Given** pre-flight check detects missing directories, **When** the warning is displayed, **Then** the user is prompted "Do you want to continue anyway? [y/N]" with default to safe option (N)
3. **Given** a user responds "n" to the prompt, **When** the operation is cancelled, **Then** they see a clear message and can run the suggested setup command
4. **Given** a user responds "y" to the prompt, **When** they continue despite warnings, **Then** the tool proceeds with a warning message
5. **Given** a CI/CD environment, **When** running `autospec workflow --skip-preflight`, **Then** validation checks are bypassed and workflow executes immediately

---

### User Story 4 - Custom Claude Command Configuration (Priority: P2)

As a power user with a customized Claude Code setup (using pipes, environment variables, or output processors), I want to configure autospec to use my exact Claude command template, so I don't have to change my workflow.

**Why this priority**: Advanced users have complex setups that work for them. Supporting customization prevents forcing users to adapt to the tool's defaults. However, this is P2 because most users will use the simple default configuration.

**Independent Test**: Can be tested by configuring a custom command template in .autospec/config.json and verifying it executes correctly. Delivers value to users with non-standard setups.

**Acceptance Scenarios**:

1. **Given** a user with a custom Claude command like `ANTHROPIC_API_KEY="" claude -p "{{PROMPT}}" | claude-clean`, **When** they configure this in .autospec/config.json, **Then** autospec uses this exact command with {{PROMPT}} replaced
2. **Given** a custom command with environment variable prefixes, **When** autospec executes it, **Then** environment variables are correctly parsed and applied
3. **Given** a custom command with pipe to external processor, **When** autospec executes it, **Then** the pipeline is correctly constructed and output is processed
4. **Given** a user switches from custom to simple configuration, **When** they remove custom_claude_cmd from config, **Then** autospec falls back to standard claude_cmd + claude_args

---

### User Story 5 - Automated Validation and Retry (Priority: P3)

As a developer running a complete SpecKit workflow, I want automatic validation of each phase's output with intelligent retry logic, so I don't have to manually check if spec.md, plan.md, or tasks.md were created correctly.

**Why this priority**: This enhances reliability but the core workflow works without it. Users can manually verify outputs. This is valuable for reducing errors but not blocking for basic usage.

**Independent Test**: Can be tested by running `autospec workflow` and intentionally causing a failure (missing output file), then verifying automatic retry occurs up to max attempts. Delivers value through improved reliability.

**Acceptance Scenarios**:

1. **Given** a workflow execution where spec.md creation fails, **When** autospec validates the output, **Then** it automatically retries up to max_retries times
2. **Given** retry count is tracked for a specific phase, **When** max retries is reached, **Then** autospec reports "retry limit exhausted" and fails gracefully
3. **Given** a phase completes successfully, **When** validation passes, **Then** retry count is reset for that phase
4. **Given** retry state stored in ~/.autospec/state/, **When** a user runs multiple commands, **Then** retry counts persist across command invocations

---

### User Story 6 - Fast Performance (Priority: P3)

As a developer in an active coding session, I want validation and workflow commands to complete in under 5 seconds for typical operations, so the tool doesn't interrupt my flow.

**Why this priority**: Performance is important for user experience but not blocking. A slower tool is still functional. This is P3 because it's an optimization concern rather than core functionality.

**Independent Test**: Can be tested by measuring execution time of `autospec status` and verifying it completes in under 1 second. Delivers value through improved responsiveness.

**Acceptance Scenarios**:

1. **Given** a developer runs `autospec status`, **When** checking implementation progress, **Then** the command completes in under 1 second
2. **Given** pre-flight validation is enabled, **When** checking .claude/commands/ and .specify/ directories, **Then** validation completes in under 100ms
3. **Given** a complete workflow execution, **When** running all three phases (specify, plan, tasks), **Then** total execution time is under 5 seconds (excluding Claude execution time)

---

### Edge Cases

- What happens when the binary runs on an unsupported architecture (e.g., ARM32)?
- How does the tool handle corrupted retry state files in ~/.autospec/state/?
- What happens if claude CLI is not in PATH when autospec tries to execute it?
- How does pre-flight check behave when .claude/commands/ exists but .specify/ doesn't (partial initialization)?
- What happens when a custom Claude command template has invalid syntax or references non-existent commands?
- How does the tool handle permissions issues when trying to write to ~/.autospec/config.json?
- What happens if a user runs autospec in a directory that's not a git repository?
- How does retry logic behave if the retry state file is locked by another process?

## Requirements *(mandatory)*

### Functional Requirements

#### Installation & Distribution
- **FR-001**: System MUST provide a single self-contained binary that runs without requiring bash, jq, git, grep, sed, or other shell utilities
- **FR-002**: System MUST support installation via `go install github.com/username/autospec@latest`
- **FR-003**: System MUST provide pre-built binaries for Linux (amd64, arm64), macOS (amd64, arm64), and Windows (amd64)
- **FR-004**: Binary MUST be under 15MB in size
- **FR-005**: System MUST start up in under 50ms

#### Core CLI Commands
- **FR-006**: System MUST implement `autospec init` to create .autospec/config.json with default configuration
- **FR-007**: System MUST implement `autospec workflow <feature-description>` to run complete specify→plan→tasks workflow by executing `/speckit.specify`, `/speckit.plan`, and `/speckit.tasks` commands via `claude -p`, validating that spec.md, plan.md, and tasks.md are created after each phase
- **FR-008**: System MUST implement `autospec specify <feature-description>` to create specifications by executing `/speckit.specify` command via `claude -p` and validating that spec.md exists in specs/ directory
- **FR-009**: System MUST implement `autospec plan` to create implementation plans by executing `/speckit.plan` command via `claude -p` and validating that plan.md exists in specs/ directory
- **FR-010**: System MUST implement `autospec tasks` to generate task breakdowns by executing `/speckit.tasks` command via `claude -p` and validating that tasks.md exists in specs/ directory
- **FR-011**: System MUST implement `autospec implement` to execute implementation by executing `/speckit.implement` command via `claude -p` and validating that all tasks in tasks.md are checked off upon completion
- **FR-012**: System MUST implement `autospec status` to show implementation progress
- **FR-013**: System MUST implement `autospec config` to display current configuration
- **FR-014**: System MUST implement `autospec version` and `autospec --version` to display version information

#### Pre-Flight Validation
- **FR-015**: System MUST perform pre-flight checks before executing workflow commands (unless --skip-preflight is used)
- **FR-016**: Pre-flight check MUST verify `specify` CLI is available in PATH
- **FR-017**: Pre-flight check MUST verify .claude/commands/ directory exists in current project
- **FR-018**: Pre-flight check MUST verify .specify/ directory exists in current project
- **FR-019**: When directories are missing, system MUST detect git root directory and display it in warning messages
- **FR-020**: When directories are missing, system MUST list all missing directories clearly
- **FR-021**: When directories are missing, system MUST show recommended setup command (`specify init . --ai claude --force`)
- **FR-022**: When directories are missing, system MUST prompt user "Do you want to continue anyway? [y/N]" with default to N
- **FR-023**: System MUST support --skip-preflight flag on all workflow commands to bypass pre-flight checks
- **FR-024**: Pre-flight check MUST complete in under 100ms

#### Validation Logic
- **FR-025**: System MUST validate existence of spec.md after /speckit.specify execution
- **FR-026**: System MUST validate existence of plan.md after /speckit.plan execution
- **FR-027**: System MUST validate existence of tasks.md after /speckit.tasks execution
- **FR-028**: System MUST count unchecked tasks in tasks.md by matching `- [ ]` and `* [ ]` patterns
- **FR-029**: System MUST validate all tasks are checked before allowing /speckit.implement completion
- **FR-030**: System MUST parse markdown structure to identify phases (## headings)
- **FR-031**: System MUST list incomplete phases with their unchecked task counts

#### Retry Logic
- **FR-032**: System MUST retry failed validations up to max_retries times (default: 3)
- **FR-033**: System MUST persist retry state to ~/.autospec/state/retry.json
- **FR-034**: Retry state MUST track spec name, phase, count, and last attempt timestamp
- **FR-035**: System MUST reset retry count when validation succeeds
- **FR-036**: System MUST report "retry limit exhausted" when max_retries is reached
- **FR-037**: Retry state MUST persist across command invocations

#### Configuration Management
- **FR-038**: System MUST support global configuration at ~/.autospec/config.json
- **FR-039**: System MUST support local repository configuration at .autospec/config.json
- **FR-040**: Local configuration MUST override global configuration
- **FR-041**: System MUST support environment variable overrides for all config values
- **FR-042**: System MUST validate configuration schema on load
- **FR-043**: Configuration MUST support simple mode with claude_cmd, claude_args, use_api_key
- **FR-044**: Configuration MUST support advanced mode with custom_claude_cmd template
- **FR-045**: Custom command template MUST support {{PROMPT}} placeholder
- **FR-046**: Custom command template MUST support environment variable prefixes (e.g., `ANTHROPIC_API_KEY=""`)
- **FR-047**: Custom command template MUST support pipe operators (e.g., `| claude-clean`)
- **FR-048**: When custom_claude_cmd is set, it MUST take precedence over simple mode configuration

#### Claude Integration
- **FR-049**: System MUST execute SpecKit slash commands (e.g., `/speckit.specify`) using `claude -p` flag to pass prompts/commands to Claude Code CLI
- **FR-050**: Default Claude command pattern MUST be: `ANTHROPIC_API_KEY="" claude -p "<command>" --dangerously-skip-permissions --verbose --output-format stream-json`
- **FR-051**: System MUST set `ANTHROPIC_API_KEY=""` (empty) to enforce using logged-in Claude Max/Pro account and avoid API key usage/charges
- **FR-052**: System MUST support optional output post-processing via pipe operators (e.g., `| claude-clean`) configured through custom_claude_cmd
- **FR-053**: System MUST generate temporary settings.json files for Claude execution when needed
- **FR-054**: System MUST stream Claude output to stdout in real-time
- **FR-055**: System MUST handle environment variables (ANTHROPIC_API_KEY) based on use_api_key setting or custom_claude_cmd configuration
- **FR-056**: System MUST support custom output processors via pipe operators in custom_claude_cmd template

#### Git Integration
- **FR-057**: System MUST detect current spec from git branch name
- **FR-058**: System MUST detect git repository root directory
- **FR-059**: System MUST handle cases where directory is not a git repository
- **FR-060**: Git operations MUST use pure Go implementation (go-git library) without requiring git binary

#### Error Handling
- **FR-061**: System MUST use standardized exit codes (0=success, 1=failed, 2=exhausted, 3=invalid, 4=missing deps)
- **FR-062**: System MUST provide actionable error messages with setup instructions
- **FR-063**: System MUST gracefully handle missing dependencies (claude, specify)
- **FR-064**: System MUST handle corrupted configuration files without crashing
- **FR-065**: System MUST handle file permission errors with clear messages

### Key Entities

- **Configuration**: Represents user settings including Claude command configuration, retry limits, specs directory, enabled hooks, and output processing preferences. Stored in JSON format at global (~/.autospec/) and local (.autospec/) levels with local overriding global.

- **Retry State**: Represents the retry tracking for a specific spec and phase combination. Contains spec name, phase name, current retry count, and last attempt timestamp. Persisted to ~/.autospec/state/retry.json to maintain state across command invocations.

- **Validation Result**: Represents the outcome of a validation check. Contains success/failure status, error details if any, and continuation prompt if validation failed. Used to decide whether to retry or proceed to next phase.

- **Spec Metadata**: Represents information about a feature specification including spec name, directory path, and detected spec number. Used to locate and validate spec artifacts (spec.md, plan.md, tasks.md).

- **Task**: Represents an individual task in tasks.md. Contains task description, checked/unchecked status, and parent phase. Used for counting unchecked tasks and determining implementation completion.

- **Phase**: Represents a section in tasks.md (identified by ## heading). Contains phase name and list of tasks within that phase. Used to identify which phases have incomplete work.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Developers can install the tool in under 30 seconds using a single command
- **SC-002**: Binary runs identically on Linux, macOS, and Windows without platform-specific configuration
- **SC-003**: Binary size is under 15MB for all platform builds
- **SC-004**: Tool startup time (e.g., `autospec --version`) is under 50ms
- **SC-005**: Pre-flight validation completes in under 100ms
- **SC-006**: Status command (`autospec status`) completes in under 1 second
- **SC-007**: Complete workflow execution (specify→plan→tasks) completes in under 5 seconds excluding Claude execution time
- **SC-008**: All 60+ existing bash tests have equivalent Go tests that pass
- **SC-009**: Zero runtime dependencies required beyond claude and specify CLIs
- **SC-010**: 90% of users successfully run their first workflow without consulting documentation beyond installation
- **SC-011**: Error messages include actionable next steps in 100% of failure cases
- **SC-012**: Custom Claude command configurations work correctly for all tested pipe and environment variable combinations

## Assumptions *(optional)*

### Claude Account Requirements
- Users have an active Claude Max or Claude Pro account logged in via `claude login`
- Setting `ANTHROPIC_API_KEY=""` (empty string) enforces using the logged-in account instead of API keys, preventing API usage charges
- Users who want to use API keys can override this by configuring `use_api_key: true` in their .autospec/config.json

### Claude CLI Availability
- The `claude` CLI tool is installed and available in the user's PATH
- The `claude -p` flag is available for passing prompts/commands directly to Claude Code

### Output Post-Processing
- The `| claude-clean` pipe is optional and user-configurable
- Users with custom output processing needs can configure custom_claude_cmd in their local .autospec/config.json
- Default configuration does not include `| claude-clean` to avoid requiring additional tools
