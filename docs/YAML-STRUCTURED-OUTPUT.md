# YAML Structured Output

AutoSpec provides YAML-based workflow artifacts that enable programmatic parsing and better tooling integration. This replaces the traditional markdown-based workflow with structured, machine-readable output.

## Overview

The YAML structured output feature introduces:

- **YAML Artifacts**: Structured output files (`spec.yaml`, `plan.yaml`, `tasks.yaml`, etc.) with consistent schemas
- **Command Templates**: Claude Code slash commands (`/autospec.specify`, `/autospec.plan`, etc.) that generate YAML artifacts
- **Validation**: Built-in YAML syntax checking with `autospec yaml check`
- **Command Management**: Install, check, and manage command templates with `autospec commands`

## Quick Start

### 1. Install AutoSpec Commands

Install the YAML-based command templates into your project:

```bash
autospec commands install
```

This creates command templates in `.claude/commands/` and helper scripts in `.autospec/scripts/`.

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

Installs command templates and helper scripts:

```bash
autospec commands install [--target <dir>] [--scripts-target <dir>]
```

**Options**:
- `--target`: Directory for command templates (default: `.claude/commands`)
- `--scripts-target`: Directory for helper scripts (default: `.autospec/scripts`)

**Output**:
- Creates 7 command templates: `autospec.specify.md`, `autospec.plan.md`, `autospec.tasks.md`, `autospec.implement.md`, `autospec.checklist.md`, `autospec.analyze.md`, `autospec.constitution.md`
- Creates helper scripts: `common.sh`, `check-prerequisites.sh`, `create-new-feature.sh`

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

### autospec yaml check

Validates YAML syntax:

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
```

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
```

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

dependencies:
  phase_order:
    - phase: 1
      blocks: [2, 3]
```

## Workflow

The typical YAML-based workflow:

1. **Specify**: `/autospec.specify "feature description"` creates `spec.yaml`
2. **Plan**: `/autospec.plan` reads `spec.yaml` and creates `plan.yaml`
3. **Tasks**: `/autospec.tasks` reads both and creates `tasks.yaml`
4. **Analyze** (optional): `/autospec.analyze` checks consistency across artifacts
5. **Implement**: `/autospec.implement` executes tasks from `tasks.yaml`

Each command validates its output with `autospec yaml check` before completing.

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

### YAML syntax errors

Run `autospec yaml check <file>` to identify the line number with the error. Common issues:
- Tabs instead of spaces
- Missing quotes around strings with special characters
- Incorrect indentation

### Commands not found in Claude Code

Verify commands are installed:

```bash
ls .claude/commands/autospec.*.md
```

If missing, run `autospec commands install`.
