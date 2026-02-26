# YAML Structured Output

AutoSpec provides YAML-based workflow artifacts that enable programmatic parsing and better tooling integration. This replaces the traditional markdown-based workflow with structured, machine-readable output.

## Overview

The YAML structured output feature introduces:

- **YAML Artifacts**: Structured output files (`spec.yaml`, `plan.yaml`, `tasks.yaml`, etc.) with consistent schemas
- **Command Templates**: Claude Code slash commands (`/autospec.specify`, `/autospec.plan`, etc.) that generate YAML artifacts
- **Schema Validation**: Full schema validation with `autospec artifact` (required fields, types, enums, cross-references)
- **Syntax Checking**: Simple YAML syntax checking with `autospec yaml check`
- **Command Management**: Install, check, and manage command templates with `autospec commands`

## Quick Start

### 1. Install AutoSpec Commands

Install the YAML-based command templates into your project:

```bash
autospec commands install
```

This creates command templates in `.claude/commands/`.

### 2. Check Installation

Verify installed commands match the embedded versions:

```bash
autospec commands check
```

### 3. Generate YAML Artifacts

Use the slash commands in Claude Code:

```
/autospec.specify "Add user authentication feature"
/autospec.plan
/autospec.tasks
```

Each command generates a corresponding YAML file in `specs/<feature-name>/`.

## Command Reference

### autospec commands install

Installs command templates:

```bash
autospec commands install [--target <dir>]
```

**Options**:
- `--target`: Directory for command templates (default: `.claude/commands`)

**Output**:
- Creates 7 command templates: `autospec.specify.md`, `autospec.plan.md`, `autospec.tasks.md`, `autospec.implement.md`, `autospec.checklist.md`, `autospec.analyze.md`, `autospec.constitution.md`

### autospec commands check

Compares installed commands against embedded versions:

```bash
autospec commands check [--target <dir>]
```

**Output**:
- `current`: Command matches embedded version
- `outdated`: Installed version differs from embedded version
- `missing`: Command not installed

### autospec commands info

Displays version metadata for installed commands:

```bash
autospec commands info [--target <dir>]
```

### autospec artifact (Schema Validation)

Validates artifacts against their schemas:

```bash
# Path-only format (preferred) - type inferred from filename
autospec artifact specs/001-feature/spec.yaml
autospec artifact specs/001-feature/plan.yaml
autospec artifact specs/001-feature/tasks.yaml
autospec artifact specs/001-feature/analysis.yaml
autospec artifact .autospec/constitution.yaml

# Checklist requires explicit type (filename varies)
autospec artifact checklist specs/001-feature/checklists/ux.yaml
```

**Validates**:
- Valid YAML syntax
- Required fields present
- Field types correct (strings, lists, enums)
- Cross-references valid (e.g., task dependencies)

**Exit codes**:
- `0`: Valid artifact
- `1`: Validation failed (with detailed errors)
- `3`: Invalid arguments

### autospec yaml check (Syntax Only)

Validates YAML syntax without schema checking:

```bash
autospec yaml check <file>
```

**Exit codes**:
- `0`: Valid YAML syntax
- `1`: Invalid YAML syntax (error message includes line number)

## YAML Artifact Structure

All generated YAML artifacts include a `_meta` section:

```yaml
_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "0.5.0"
  created: "2025-12-13T10:30:00Z"
  artifact_type: "spec"  # or plan, tasks, checklist, analysis, constitution
```

### spec.yaml

Feature specification with requirements and user stories:

```yaml
_meta:
  version: "1.0.0"
  generator: "autospec"
  artifact_type: "spec"

spec:
  name: "User Authentication"
  branch: "008-user-auth"
  status: "Draft"
  created: "2025-12-13"

user_stories:
  - id: "US-001"
    title: "User Login"
    priority: "P1"
    description: "As a user, I want to log in..."
    acceptance_scenarios:
      - given: "valid credentials"
        when: "user submits login form"
        then: "user is authenticated and redirected"

requirements:
  functional:
    - id: "FR-001"
      description: "System MUST validate user credentials"
  non_functional:
    - id: "NFR-001"
      description: "Login response time MUST be under 500ms"

ears_requirements:  # optional EARS-formatted requirements
  - id: "EARS-001"
    pattern: "ubiquitous"
    text: "The system shall validate all user inputs"
    test_type: "invariant"
  - id: "EARS-002"
    pattern: "event-driven"
    text: "When a user submits the login form, the system shall authenticate"
    test_type: "property"
    trigger: "user submits login form"
    expected: "user is authenticated"
```

### EARS Requirements (Optional)

EARS (Easy Approach to Requirements Syntax) provides machine-parseable requirements that map to test types. EARS is **opt-in** and must be explicitly enabled via `verification.ears_requirements: true` in your config. When enabled, EARS templates are injected into the specify command.

**Pattern Reference:**

| Pattern | Template | Required Fields | Test Type |
|---------|----------|-----------------|-----------|
| `ubiquitous` | "The [system] shall [action]" | - | `invariant` |
| `event-driven` | "When [trigger], the [system] shall [action]" | `trigger`, `expected` | `property` |
| `state-driven` | "While [state], the [system] shall [action]" | `state` | `state-machine` |
| `unwanted` | "If [condition], then the [system] shall [action]" | `condition` | `exception` |
| `optional` | "Where [feature], the [system] shall [action]" | `feature` | `feature-flag` |

**Validation Rules:**
- EARS IDs must match pattern `EARS-NNN` (e.g., `EARS-001`)
- IDs must be unique across both `requirements.functional` and `ears_requirements`
- Pattern-specific fields are required based on pattern type
- Empty `ears_requirements: []` is valid (no EARS requirements)

**Status field values:**
- `Draft` - Initial state when spec is created
- `Completed` - Automatically set when all implementation tasks finish

**Automatic completion:** When implementation completes (all tasks done), autospec automatically updates the spec.yaml:
- Sets `status` to `Completed`
- Adds `completed_at` with ISO 8601 timestamp (e.g., `2025-12-16T14:30:00Z`)

### plan.yaml

Implementation plan with technical context:

```yaml
_meta:
  version: "1.0.0"
  generator: "autospec"
  artifact_type: "plan"

plan:
  branch: "008-user-auth"
  spec_path: "specs/008-user-auth/spec.yaml"

technical_context:
  stack:
    - "Go 1.21"
    - "PostgreSQL 15"
  constraints:
    - "Must integrate with existing auth middleware"

project_structure:
  new_files:
    - path: "internal/auth/handler.go"
      purpose: "HTTP handlers for auth endpoints"

data_model:
  entities:
    - name: "User"
      fields:
        - name: "id"
          type: "uuid"
        - name: "email"
          type: "string"

risks:  # optional
  - risk: "Auth token expiry edge cases"
    likelihood: "medium"
    impact: "high"
    mitigation: "Add comprehensive token refresh tests"
```

See [risks.md](risks.md) for full schema and validation details.

### tasks.yaml

Task breakdown with dependencies:

```yaml
_meta:
  version: "1.0.0"
  generator: "autospec"
  artifact_type: "tasks"

tasks:
  branch: "008-user-auth"
  spec_path: "specs/008-user-auth/spec.yaml"
  plan_path: "specs/008-user-auth/plan.yaml"

summary:
  total_tasks: 15
  total_phases: 4
  parallel_opportunities: 5

phases:
  - number: 1
    title: "Setup"
    tasks:
      - id: "T001"
        title: "Initialize auth package"
        status: "Pending"
        type: "setup"
        parallel: false
        file_path: "internal/auth/"
        dependencies: []
        notes: ""  # optional free-form notes (max 1000 chars)

dependencies:
  phase_order:
    - phase: 1
      blocks: [2, 3]
```

### checklist.yaml

Quality checklists for validation (stored in `checklists/` subdirectory):

```yaml
_meta:
  version: "1.0.0"
  generator: "autospec"
  artifact_type: "checklist"

checklist:
  title: "Specification Quality Checklist"
  purpose: "Validate specification completeness"
  feature_ref: "../spec.yaml"

categories:
  - name: "Requirement Completeness"
    items:
      - id: "CHK001"
        description: "All functional requirements specified"
        quality_dimension: "completeness"
        spec_reference: "FR-001"
        status: "pass"  # pending | pass | fail
        notes: ""

summary:
  total_items: 10
  passed: 10
  failed: 0
  pending: 0
  pass_rate: "100%"
```

**Status values:**
- `pending`: Not yet evaluated
- `pass`: Item verified/completed
- `fail`: Item failed verification

## Workflow

The typical YAML-based workflow:

1. **Specify**: `/autospec.specify "feature description"` creates `spec.yaml`
2. **Plan**: `/autospec.plan` reads `spec.yaml` and creates `plan.yaml`
3. **Tasks**: `/autospec.tasks` reads both and creates `tasks.yaml`
4. **Checklist** (optional): `/autospec.checklist` creates `checklists/<domain>.yaml`
5. **Analyze** (optional): `/autospec.analyze` checks consistency across artifacts
6. **Implement**: `/autospec.implement` executes tasks, validates checklists first

Each command validates its output with `autospec artifact` for schema compliance before completing.

## SpecKit vs AutoSpec: Artifact Consolidation

The YAML workflow (`autospec.*`) consolidates multiple markdown files from the legacy workflow (`speckit.*`) into fewer, structured YAML artifacts.

### File Mapping

| SpecKit (Markdown)         | AutoSpec (YAML)                          |
|----------------------------|------------------------------------------|
| `spec.md`                  | `spec.yaml`                              |
| `plan.md`                  | `plan.yaml`                              |
| `research.md`              | `plan.yaml` → `research_findings`        |
| `data-model.md`            | `plan.yaml` → `data_model`               |
| `contracts/*.yaml`         | `plan.yaml` → `api_contracts`            |
| `quickstart.md`            | `plan.yaml` → `implementation_strategy`  |
| `tasks.md`                 | `tasks.yaml`                             |

### What's Consolidated

**plan.yaml now contains:**

```yaml
# Previously separate: research.md
research_findings:
  decisions:
    - topic: "Authentication approach"
      decision: "JWT tokens"
      rationale: "Stateless, scalable"
      alternatives_considered: ["sessions", "OAuth2"]

# Previously separate: data-model.md
data_model:
  entities:
    - name: "User"
      fields:
        - name: "id"
          type: "uuid"
      relationships:
        - target: "Session"
          type: "one-to-many"

# Previously separate: contracts/*.yaml
api_contracts:
  endpoints:
    - method: "POST"
      path: "/auth/login"
      request:
        content_type: "application/json"
      response:
        success_code: 200

# Previously separate: quickstart.md (test scenarios)
implementation_strategy:
  mvp_scope:
    phases: [1, 2, 3]
    validation: "User can login and access protected routes"
```

### Benefits of Consolidation

1. **Single source of truth**: All planning artifacts in one file
2. **Cross-referencing**: Easy to link entities to endpoints to tasks
3. **Programmatic access**: Query with `yq` instead of parsing markdown
4. **Validation**: Schema-based checking catches errors early
5. **Fewer files**: 3 YAML files vs 6+ markdown files

### Command Comparison

| Phase     | SpecKit Command      | AutoSpec Command      |
|-----------|----------------------|-----------------------|
| Specify   | `/speckit.specify`   | `/autospec.specify`   |
| Plan      | `/speckit.plan`      | `/autospec.plan`      |
| Tasks     | `/speckit.tasks`     | `/autospec.tasks`     |
| Implement | `/speckit.implement` | `/autospec.implement` |
| Analyze   | `/speckit.analyze`   | `/autospec.analyze`   |
| Checklist | `/speckit.checklist` | `/autospec.checklist` |

The `autospec.*` commands read and write YAML; `speckit.*` commands use markdown.

### CLI Commands for Workflow Support

The following CLI commands support the YAML workflow (replacing legacy shell scripts):

**autospec prereqs**

Returns YAML artifact paths in JSON format:

```bash
autospec prereqs --json --require-plan
# Output: {"FEATURE_DIR":"specs/<feature>","FEATURE_SPEC":"specs/<feature>/spec.yaml","IMPL_PLAN":"specs/<feature>/plan.yaml",...}
```

**autospec new-feature**

Creates feature branches and directories:

```bash
autospec new-feature --json "Add user authentication"
# Output: {"BRANCH_NAME":"008-user-auth","FEATURE_DIR":"specs/008-user-auth",...}
```

Key variables returned:

| Variable       | Value                          |
|----------------|--------------------------------|
| `FEATURE_SPEC` | `specs/<feature>/spec.yaml`    |
| `IMPL_PLAN`    | `specs/<feature>/plan.yaml`    |
| `TASKS`        | `specs/<feature>/tasks.yaml`   |
| `IS_GIT_REPO`  | `true` or `false`              |

## Updating Task Status

Use the `autospec update-task` command to update individual task status during implementation:

```bash
# Mark task T001 as in progress
autospec update-task T001 InProgress

# Mark task T001 as completed
autospec update-task T001 Completed

# Mark task as blocked
autospec update-task T015 Blocked
```

**Valid status values:**
- `Pending` - Task not yet started
- `InProgress` - Task currently being worked on
- `Completed` - Task finished successfully
- `Blocked` - Task blocked by dependency or issue

This command auto-detects the current feature from the git branch and updates the corresponding `tasks.yaml` file.

## Migration from Markdown

To convert existing markdown artifacts to YAML:

```bash
autospec migrate md-to-yaml specs/my-feature/
```

This converts `spec.md`, `plan.md`, and `tasks.md` to their YAML equivalents while preserving the original files.

## Querying YAML Artifacts

Use standard YAML tools to extract data:

```bash
# Get all user story IDs
yq '.user_stories[].id' spec.yaml

# Get pending tasks
yq '.phases[].tasks[] | select(.status == "Pending")' tasks.yaml

# Count tasks by phase
yq '.phases[] | .number as $n | .tasks | length | "\($n): \(.)"' tasks.yaml
```

## Version Compatibility

The `_meta.version` field tracks artifact schema versions:

- **1.0.0**: Initial YAML format (current)

When the autospec binary version is newer than the artifact version, commands proceed with best-effort parsing and emit a warning.

## Troubleshooting

### "No spec.yaml found"

Ensure you're on a feature branch (e.g., `007-feature-name`) and have run `/autospec.specify` first.

### YAML validation errors

For schema errors (missing fields, wrong types):
```bash
autospec artifact specs/001-feature/plan.yaml
```

For syntax errors only:
```bash
autospec yaml check <file>
```

Common issues:
- Tabs instead of spaces
- Missing quotes around strings with special characters
- Incorrect indentation
- Missing required fields (use `autospec artifact` for details)

### Commands not found in Claude Code

Verify commands are installed:

```bash
ls .claude/commands/autospec.*.md
```

If missing, run `autospec commands install`.
