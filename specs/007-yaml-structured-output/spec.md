# Feature Specification: YAML Structured Output

**Feature Branch**: `007-yaml-structured-output`
**Created**: 2025-12-13
**Status**: Draft
**Input**: User description: "Create YAML structured output format for SpecKit workflow artifacts to enable schema validation, programmatic parsing, and better tooling integration"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create YAML-Based Feature Specifications (Priority: P1)

As a developer using the SpecKit workflow, I want to generate feature specifications in YAML format so that I can programmatically validate required fields exist and parse specific sections without fragile regex patterns.

**Why this priority**: The specification is the foundation of the entire workflow. All other artifacts (plan, tasks, checklist) derive from it. Without a structured spec format, downstream artifacts cannot be reliably validated.

**Independent Test**: Can be fully tested by running `/autospec.specify "test feature"` and verifying the output `spec.yaml` file validates against the JSON schema, contains all required fields, and can be parsed by standard YAML libraries.

**Acceptance Scenarios**:

1. **Given** a user provides a feature description, **When** they run the `/autospec.specify` command, **Then** the system creates a valid `spec.yaml` file with all required fields populated
2. **Given** a generated `spec.yaml` file, **When** validated against the JSON schema, **Then** the validation passes with no errors
3. **Given** a `spec.yaml` file exists, **When** queried with standard YAML tools (yq), **Then** specific fields like user stories and requirements can be extracted reliably

---

### User Story 2 - Generate Structured Task Breakdowns (Priority: P2)

As a project manager, I want task breakdowns in YAML format so that I can extract tasks by phase, filter by status, and integrate with external project management tools.

**Why this priority**: Task extraction and status tracking are the most common programmatic operations. This directly enables CI/CD integration and automation workflows.

**Independent Test**: Can be fully tested by generating `tasks.yaml` from an existing spec and verifying tasks can be queried by phase number, filtered by status, and counted by category.

**Acceptance Scenarios**:

1. **Given** a valid `plan.yaml` exists, **When** the user runs `/autospec.tasks`, **Then** the system creates a `tasks.yaml` file with phases, tasks, and dependency information
2. **Given** a `tasks.yaml` file, **When** querying for Phase 1 tasks, **Then** all tasks belonging to Phase 1 are returned with correct structure
3. **Given** a `tasks.yaml` file with mixed task statuses, **When** filtering for "Pending" tasks, **Then** only pending tasks are returned

---

### User Story 3 - Install and Manage AutoSpec Commands (Priority: P2)

As a developer, I want to install autospec commands into my project's `.claude/commands/` directory so that I can use the new YAML-based workflow alongside or instead of the markdown workflow.

**Why this priority**: Without a mechanism to install commands, users cannot access the new functionality. This enables adoption while maintaining backward compatibility.

**Independent Test**: Can be fully tested by running `autospec commands install`, verifying files are created in `.claude/commands/`, and confirming the commands are executable via Claude Code.

**Acceptance Scenarios**:

1. **Given** a project without autospec commands installed, **When** the user runs `autospec commands install`, **Then** all autospec command files are created in `.claude/commands/`
2. **Given** outdated autospec commands exist, **When** the user runs `autospec commands check`, **Then** the system reports which commands need updating
3. **Given** installed autospec commands, **When** the user runs `autospec commands info`, **Then** version information including source SpecKit version is displayed

---

### User Story 4 - Validate YAML Artifacts Against Schemas (Priority: P3)

As a CI/CD engineer, I want to validate YAML artifacts against defined schemas so that I can fail builds when specifications are incomplete or malformed.

**Why this priority**: Schema validation ensures quality gates can be automated. This is essential for team workflows where multiple contributors modify specifications.

**Independent Test**: Can be fully tested by running `autospec yaml validate spec.yaml` on both valid and invalid files, verifying correct pass/fail results.

**Acceptance Scenarios**:

1. **Given** a valid `spec.yaml` file, **When** validated with `autospec yaml validate`, **Then** the command exits with code 0 and reports success
2. **Given** a `spec.yaml` missing required fields, **When** validated, **Then** the command exits with non-zero code and lists missing fields
3. **Given** a `tasks.yaml` with invalid enum values, **When** validated, **Then** the command reports the specific validation errors

---

### User Story 5 - Generate Implementation Plans in YAML (Priority: P3)

As a technical architect, I want implementation plans in YAML format so that I can extract technical context, review data models programmatically, and track research decisions.

**Why this priority**: Plans inform task generation. Structured plans enable better traceability between requirements, technical decisions, and implementation tasks.

**Independent Test**: Can be fully tested by running `/autospec.plan` on a spec and verifying the `plan.yaml` contains technical context, project structure, and data model sections.

**Acceptance Scenarios**:

1. **Given** a valid `spec.yaml`, **When** the user runs `/autospec.plan`, **Then** a `plan.yaml` file is created with technical context and project structure
2. **Given** a `plan.yaml` file, **When** querying for data model entities, **Then** entity definitions with fields and relationships are returned

---

### User Story 6 - Query YAML Artifacts Programmatically (Priority: P4)

As a developer building automation, I want CLI commands to query YAML artifacts so that I can extract specific information without writing custom parsing code.

**Why this priority**: While users can use external tools like yq, built-in query commands provide a consistent interface and handle edge cases.

**Independent Test**: Can be fully tested by running `autospec yaml query tasks.yaml '.phases[0].tasks'` and verifying correct JSON/YAML output.

**Acceptance Scenarios**:

1. **Given** a `tasks.yaml` file, **When** running `autospec yaml query tasks.yaml '.summary.total_tasks'`, **Then** the total task count is returned
2. **Given** a `spec.yaml` file, **When** running `autospec yaml tasks --phase=1`, **Then** only Phase 1 tasks are extracted and displayed

---

### User Story 7 - Migrate Existing Markdown to YAML (Priority: P5)

As an existing SpecKit user, I want to convert my existing markdown artifacts to YAML so that I can benefit from structured output without recreating specifications from scratch.

**Why this priority**: Migration tooling is a convenience feature. Users can manually recreate specs if needed, but automated migration accelerates adoption.

**Independent Test**: Can be fully tested by running `autospec migrate md-to-yaml specs/existing-feature/` and verifying the generated YAML files contain equivalent content.

**Acceptance Scenarios**:

1. **Given** a directory with `spec.md`, `plan.md`, and `tasks.md`, **When** running the migration command, **Then** corresponding `.yaml` files are created with equivalent content
2. **Given** a mixed format directory (some YAML, some markdown), **When** running migration, **Then** only markdown files are converted and existing YAML files are preserved

---

### Edge Cases

- What happens when a YAML file has syntax errors (invalid YAML)?
  - The system reports parsing errors with line numbers and does not proceed with validation
- How does the system handle partially filled YAML files during workflow interruption?
  - The validation reports which required fields are missing; the file is usable for analysis but fails strict validation
- What happens when schema versions mismatch between installed commands and YAML artifacts?
  - The system warns about version mismatch but attempts to validate using the artifact's declared schema version
- How does the system handle concurrent modifications to YAML artifacts?
  - YAML write operations are atomic (write to temp file, then rename); concurrent reads are safe

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST generate `spec.yaml` files conforming to the spec schema when `/autospec.specify` is executed
- **FR-002**: System MUST generate `plan.yaml` files conforming to the plan schema when `/autospec.plan` is executed
- **FR-003**: System MUST generate `tasks.yaml` files conforming to the tasks schema when `/autospec.tasks` is executed
- **FR-004**: System MUST include `_meta` section in all generated YAML files with schema version, generator, and timestamp
- **FR-005**: System MUST provide `autospec commands install` to copy command templates to `.claude/commands/`
- **FR-006**: System MUST provide `autospec commands check` to compare installed commands against embedded versions
- **FR-007**: System MUST provide `autospec yaml validate <file>` to validate YAML files against their schemas
- **FR-008**: System MUST embed command templates and JSON schemas in the Go binary using `go:embed`
- **FR-009**: System MUST maintain backward compatibility with existing `/speckit.*` commands producing markdown
- **FR-010**: System MUST prefer `.yaml` files over `.md` files when both exist for the same artifact
- **FR-011**: System MUST provide `autospec commands info` displaying version metadata including source SpecKit version
- **FR-012**: System MUST generate JSON Schema files for each YAML artifact type (spec, plan, tasks, checklist, analysis, constitution)

### Key Entities

- **YAML Artifact**: A structured output file (spec.yaml, plan.yaml, tasks.yaml, etc.) containing feature workflow data with schema metadata
- **JSON Schema**: Validation schema defining required/optional fields, types, and constraints for each artifact type
- **Command Template**: A markdown file defining a Claude Code slash command that generates or modifies YAML artifacts
- **Schema Version**: Semantic version string tracking artifact structure changes, stored in `_meta.schema_version`

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All generated YAML artifacts pass schema validation on first generation attempt
- **SC-002**: Users can extract any specific field from YAML artifacts using standard tools (yq, Python yaml) without custom parsing
- **SC-003**: Command installation completes in under 5 seconds for all 8 autospec commands
- **SC-004**: Schema validation reports specific field-level errors, not generic parse failures
- **SC-005**: Existing markdown-based workflows continue to function unchanged when YAML commands are installed
- **SC-006**: 100% of required schema fields are documented with descriptions in JSON Schema files

## Assumptions

- Users have access to Claude Code and can execute slash commands
- Standard YAML 1.2 format is sufficient (no need for YAML 1.1 compatibility)
- The Go binary (`autospec`) is the authoritative source for command templates and schemas
- Users who want programmatic access will use standard tools (yq, Python yaml library) or the CLI utilities provided

## Constraints

- Command templates must be markdown files compatible with Claude Code's command format
- JSON Schema draft-07 is used for validation (widely supported)
- YAML files must remain human-readable; complex nested structures should be documented
- Maximum YAML file size for validation is 10MB (practical limit for specification documents)

## Out of Scope

- GUI/web interface for YAML editing
- Real-time collaborative editing of YAML artifacts
- Automatic conversion of YAML back to markdown (export is one-way migration helper only)
- Custom user-defined schema extensions (schemas are fixed per version)
- Integration with specific project management tools (Jira, Linear, etc.) - this is a future feature
