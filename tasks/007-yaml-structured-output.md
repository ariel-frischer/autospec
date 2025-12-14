# Feature Planning: YAML Structured Output for SpecKit

**Created**: 2025-12-13
**Status**: Planning
**Branch Candidate**: `007-yaml-structured-output`

## Problem Statement

The current SpecKit workflow produces markdown (`.md`) files for all artifacts:
- `spec.md` - Feature specification
- `plan.md` - Implementation plan
- `tasks.md` - Task breakdown
- `checklist.md` - Quality checklists
- `research.md`, `data-model.md`, `quickstart.md` - Supporting documents

### Why Markdown is Problematic

1. **Cannot Validate Fields**: No way to enforce required vs optional fields
2. **Hard to Parse Programmatically**: Extracting specific sections (e.g., "all Phase 1 tasks") requires fragile regex or markdown parsing
3. **No Schema Validation**: Invalid structure goes undetected until runtime
4. **Limited Tooling Integration**: Cannot easily feed data into other tools, dashboards, or automation
5. **Inconsistent Structure**: AI may produce slightly different heading levels or formatting

### Why YAML is Better

1. **Structured & Parseable**: Standard libraries in every language
2. **Schema Validation**: JSON Schema or YAML schema can validate structure
3. **Field Validation**: Clearly define required vs optional fields
4. **Human Readable**: More readable than JSON, maintains hierarchy
5. **Query-Friendly**: Easy to extract `tasks.phases[0].tasks` or `spec.user_stories[*].priority`
6. **Type Safety**: Can enforce types (string, number, array, etc.)

---

## Solution Overview

Create parallel `autospec.*` commands that produce YAML output files instead of markdown, while keeping the original `speckit.*` commands intact. This provides:

- **Backward Compatibility**: Existing speckit commands continue to work
- **Opt-In Migration**: Users choose when to switch to YAML
- **Structured Output**: New commands produce `.yaml` files with defined schemas

### Command Mapping

| Original SpecKit Command | New AutoSpec Command | Output File |
|--------------------------|---------------------|-------------|
| `/speckit.specify` | `/autospec.specify` | `spec.yaml` |
| `/speckit.plan` | `/autospec.plan` | `plan.yaml` |
| `/speckit.tasks` | `/autospec.tasks` | `tasks.yaml` |
| `/speckit.checklist` | `/autospec.checklist` | `{domain}.yaml` |
| `/speckit.clarify` | `/autospec.clarify` | Updates `spec.yaml` |
| `/speckit.implement` | `/autospec.implement` | Updates `tasks.yaml` |
| `/speckit.analyze` | `/autospec.analyze` | `analysis.yaml` |
| `/speckit.constitution` | `/autospec.constitution` | `constitution.yaml` |
| `/speckit.taskstoissues` | `/autospec.taskstoissues` | (no file output) |

---

## YAML Schema Definitions

### 1. spec.yaml - Feature Specification

```yaml
# Schema: spec.yaml
# Version: 1.0.0

_meta:
  schema_version: "1.0.0"
  generated_by: "autospec.specify"
  generated_at: "2025-12-13T10:30:00Z"

feature:
  name: string                    # REQUIRED - Feature name
  branch: string                  # REQUIRED - Git branch name (e.g., "007-yaml-structured-output")
  created: date                   # REQUIRED - ISO 8601 date
  status: enum                    # REQUIRED - Draft | InProgress | Complete | Archived
  input: string                   # REQUIRED - Original user description

user_stories:                     # REQUIRED - At least one story
  - id: string                    # REQUIRED - e.g., "US-001"
    title: string                 # REQUIRED - Brief title
    priority: enum                # REQUIRED - P1 | P2 | P3 | P4 | P5
    description: string           # REQUIRED - Plain language description
    rationale: string             # REQUIRED - Why this priority
    independent_test: string      # REQUIRED - How to test independently
    acceptance_scenarios:         # REQUIRED - At least one scenario
      - given: string             # REQUIRED
        when: string              # REQUIRED
        then: string              # REQUIRED

edge_cases:                       # OPTIONAL - List of edge case questions
  - question: string              # REQUIRED if section present
    answer: string                # OPTIONAL - If resolved

requirements:
  functional:                     # REQUIRED - At least one FR
    - id: string                  # REQUIRED - e.g., "FR-001"
      description: string         # REQUIRED - MUST/SHOULD statement
      testable: boolean           # REQUIRED - Is this testable?
      clarification_needed: string # OPTIONAL - If unclear

  non_functional:                 # OPTIONAL
    - id: string
      category: enum              # Performance | Security | Scalability | Reliability | Accessibility
      description: string
      metric: string              # OPTIONAL - Measurable target

  entities:                       # OPTIONAL - Key data entities
    - name: string
      description: string
      attributes: [string]        # OPTIONAL - Key attributes
      relationships: [string]     # OPTIONAL - Related entities

success_criteria:                 # REQUIRED - At least one criterion
  - id: string                    # REQUIRED - e.g., "SC-001"
    description: string           # REQUIRED - Measurable outcome
    metric_type: enum             # OPTIONAL - Time | Percentage | Count | Rate | Qualitative
    target: string                # OPTIONAL - Specific target value

assumptions:                      # OPTIONAL
  - description: string

constraints:                      # OPTIONAL
  - description: string

out_of_scope:                     # OPTIONAL
  - description: string

clarifications:                   # OPTIONAL - Added by autospec.clarify
  - session: date
    question: string
    answer: string
    sections_updated: [string]
```

### 2. plan.yaml - Implementation Plan

```yaml
# Schema: plan.yaml
# Version: 1.0.0

_meta:
  schema_version: "1.0.0"
  generated_by: "autospec.plan"
  generated_at: "2025-12-13T10:30:00Z"
  spec_ref: "spec.yaml"           # Reference to source spec

feature:
  name: string                    # REQUIRED
  branch: string                  # REQUIRED
  date: date                      # REQUIRED

summary: string                   # REQUIRED - 1-2 sentence overview

technical_context:
  language: string                # REQUIRED - e.g., "Go 1.21"
  version: string                 # OPTIONAL - Language version
  primary_dependencies:           # REQUIRED - At least one
    - name: string
      version: string             # OPTIONAL
      purpose: string             # OPTIONAL
  storage: string                 # OPTIONAL - Database/storage tech
  testing_framework: string       # REQUIRED
  target_platform: string         # REQUIRED - e.g., "Linux server"
  project_type: enum              # REQUIRED - Single | Web | Mobile | Library
  performance_goals:              # OPTIONAL
    - metric: string
      target: string
  constraints:                    # OPTIONAL
    - description: string
  scale_scope: string             # OPTIONAL - Expected scale

constitution_check:
  passed: boolean                 # REQUIRED
  violations:                     # OPTIONAL - If any violations
    - principle: string
      violation: string
      justification: string       # REQUIRED if violation exists
      alternative_rejected: string

project_structure:
  type: enum                      # REQUIRED - Single | Web | Mobile
  directories:                    # REQUIRED
    - path: string                # e.g., "src/models/"
      purpose: string             # e.g., "Data models"
  decision_rationale: string      # REQUIRED - Why this structure

research:                         # OPTIONAL - Phase 0 output
  unknowns_resolved:
    - topic: string
      decision: string
      rationale: string
      alternatives_considered: [string]

data_model:                       # OPTIONAL - Phase 1 output
  entities:
    - name: string
      fields:
        - name: string
          type: string
          required: boolean
          validation: string      # OPTIONAL
      relationships:
        - target: string
          type: enum              # OneToOne | OneToMany | ManyToMany
          description: string

contracts:                        # OPTIONAL - Phase 1 API contracts
  - endpoint: string              # e.g., "POST /api/users"
    method: enum                  # GET | POST | PUT | PATCH | DELETE
    description: string
    request_schema: object        # OPTIONAL - JSON Schema reference
    response_schema: object       # OPTIONAL
    errors:
      - code: integer
        description: string
```

### 3. tasks.yaml - Task Breakdown

```yaml
# Schema: tasks.yaml
# Version: 1.0.0

_meta:
  schema_version: "1.0.0"
  generated_by: "autospec.tasks"
  generated_at: "2025-12-13T10:30:00Z"
  plan_ref: "plan.yaml"
  spec_ref: "spec.yaml"

feature:
  name: string                    # REQUIRED
  branch: string                  # REQUIRED

summary:
  total_tasks: integer            # REQUIRED - Auto-calculated
  tasks_by_phase: object          # REQUIRED - { "setup": 3, "foundational": 5, ... }
  tasks_by_story: object          # REQUIRED - { "US1": 8, "US2": 6, ... }
  parallel_opportunities: integer # REQUIRED - Count of [P] tasks
  mvp_scope: string               # REQUIRED - Description of MVP

phases:                           # REQUIRED - At least Phase 1 (Setup)
  - id: string                    # REQUIRED - e.g., "phase-1-setup"
    number: integer               # REQUIRED - 1, 2, 3...
    name: string                  # REQUIRED - e.g., "Setup"
    type: enum                    # REQUIRED - Setup | Foundational | UserStory | Polish
    purpose: string               # REQUIRED - Brief description

    # For UserStory phases only
    user_story_ref: string        # CONDITIONAL - e.g., "US-001" (required if type=UserStory)
    goal: string                  # CONDITIONAL - What this story delivers
    independent_test: string      # CONDITIONAL - How to verify independently

    tasks:                        # REQUIRED - At least one task per phase
      - id: string                # REQUIRED - e.g., "T001"
        description: string       # REQUIRED - Clear action description
        file_path: string         # REQUIRED - Exact file path
        parallel: boolean         # REQUIRED - Can run in parallel? (default: false)
        story_ref: string         # OPTIONAL - e.g., "US1" (for user story phases)
        status: enum              # REQUIRED - Pending | InProgress | Complete | Blocked
        depends_on: [string]      # OPTIONAL - Task IDs this depends on
        category: enum            # OPTIONAL - Model | Service | Endpoint | Test | Config | Docs

    checkpoint: string            # OPTIONAL - Validation point description

dependencies:
  phase_order:                    # REQUIRED - Phase dependency graph
    - phase: string               # Phase ID
      depends_on: [string]        # Phase IDs this depends on

  user_story_order:               # OPTIONAL - Story dependency graph
    - story: string               # Story ID
      depends_on: [string]        # Story IDs (usually empty for independence)
      can_parallel: boolean       # Can run in parallel with others?

  task_rules:                     # REQUIRED - Execution rules
    - rule: string                # e.g., "Tests before implementation"
      description: string

parallel_execution:               # OPTIONAL - Examples of parallel execution
  - group_name: string
    task_ids: [string]
    description: string

implementation_strategy:
  approach: enum                  # REQUIRED - MVP | Incremental | Parallel
  mvp_phases: [string]            # REQUIRED - Phase IDs for MVP
  incremental_delivery:           # OPTIONAL
    - milestone: string
      phases: [string]
      deliverable: string
```

### 4. checklist.yaml - Quality Checklist

```yaml
# Schema: checklist.yaml
# Version: 1.0.0

_meta:
  schema_version: "1.0.0"
  generated_by: "autospec.checklist"
  generated_at: "2025-12-13T10:30:00Z"
  spec_ref: "spec.yaml"

checklist:
  type: string                    # REQUIRED - e.g., "ux", "security", "api"
  feature: string                 # REQUIRED - Feature name
  purpose: string                 # REQUIRED - What this checklist validates

  focus:
    areas: [string]               # REQUIRED - Focus areas selected
    depth: enum                   # REQUIRED - Lightweight | Standard | Comprehensive
    audience: enum                # REQUIRED - Author | Reviewer | QA | Release

categories:                       # REQUIRED - At least one category
  - name: string                  # REQUIRED - Category name
    dimension: enum               # REQUIRED - Completeness | Clarity | Consistency | Measurability | Coverage
    items:                        # REQUIRED - At least one item
      - id: string                # REQUIRED - e.g., "CHK001"
        question: string          # REQUIRED - The checklist question
        spec_reference: string    # OPTIONAL - e.g., "Spec FR-001"
        marker: enum              # OPTIONAL - Gap | Ambiguity | Conflict | Assumption
        checked: boolean          # REQUIRED - Default false
        notes: string             # OPTIONAL - Findings or comments

summary:
  total_items: integer            # REQUIRED - Auto-calculated
  checked_count: integer          # REQUIRED - Auto-calculated
  unchecked_count: integer        # REQUIRED - Auto-calculated
  status: enum                    # REQUIRED - Pass | Fail | Partial
```

### 5. analysis.yaml - Cross-Artifact Analysis

```yaml
# Schema: analysis.yaml
# Version: 1.0.0

_meta:
  schema_version: "1.0.0"
  generated_by: "autospec.analyze"
  generated_at: "2025-12-13T10:30:00Z"
  artifacts_analyzed:
    - "spec.yaml"
    - "plan.yaml"
    - "tasks.yaml"

findings:                         # REQUIRED
  - id: string                    # REQUIRED - e.g., "A1", "D3"
    category: enum                # REQUIRED - Duplication | Ambiguity | Underspecification | Constitution | Coverage | Inconsistency
    severity: enum                # REQUIRED - Critical | High | Medium | Low
    locations: [string]           # REQUIRED - File:line references
    summary: string               # REQUIRED - Brief description
    recommendation: string        # REQUIRED - Suggested fix

coverage:
  requirements:                   # REQUIRED
    - key: string                 # Requirement key/slug
      has_task: boolean
      task_ids: [string]
      notes: string

  unmapped_tasks: [string]        # OPTIONAL - Task IDs without requirements

constitution_alignment:
  passed: boolean
  issues: [string]                # OPTIONAL - If any issues

metrics:
  total_requirements: integer
  total_tasks: integer
  coverage_percentage: number
  ambiguity_count: integer
  duplication_count: integer
  critical_issues_count: integer

next_actions:                     # REQUIRED
  - action: string
    priority: enum                # High | Medium | Low
    command_suggestion: string    # OPTIONAL - e.g., "/autospec.specify"
```

### 6. constitution.yaml - Project Constitution

```yaml
# Schema: constitution.yaml
# Version: 1.0.0

_meta:
  schema_version: "1.0.0"
  generated_by: "autospec.constitution"
  generated_at: "2025-12-13T10:30:00Z"

constitution:
  project_name: string            # REQUIRED
  version: string                 # REQUIRED - Semantic version
  ratification_date: date         # REQUIRED - Original adoption
  last_amended_date: date         # REQUIRED - Last change

principles:                       # REQUIRED - At least one principle
  - id: string                    # REQUIRED - e.g., "P1"
    name: string                  # REQUIRED - Short name
    description: string           # REQUIRED - Full description
    rules:                        # REQUIRED - At least one rule
      - type: enum                # MUST | SHOULD | MAY
        statement: string
    rationale: string             # OPTIONAL - Why this principle
    exceptions: [string]          # OPTIONAL - When it doesn't apply

governance:
  amendment_procedure: string     # REQUIRED
  versioning_policy: string       # REQUIRED
  compliance_review: string       # REQUIRED

sync_impact:                      # OPTIONAL - Auto-generated on updates
  version_change:
    from: string
    to: string
  modified_principles: [object]
  added_sections: [string]
  removed_sections: [string]
  templates_updated: [string]
  pending_updates: [string]
```

---

## Field Validation Rules Summary

### Required Fields by Artifact

| Artifact | Required Fields |
|----------|-----------------|
| `spec.yaml` | feature.*, user_stories[*].*, requirements.functional[*].*, success_criteria[*].* |
| `plan.yaml` | feature.*, summary, technical_context.language/dependencies/testing/platform/type, project_structure.* |
| `tasks.yaml` | feature.*, summary.*, phases[*].*, phases[*].tasks[*].id/description/file_path/status |
| `checklist.yaml` | checklist.*, categories[*].*, categories[*].items[*].id/question/checked |
| `analysis.yaml` | findings[*].*, coverage.requirements[*].*, metrics.*, next_actions[*].action |
| `constitution.yaml` | constitution.*, principles[*].*, governance.* |

### Optional Fields (Include When Relevant)

- `spec.yaml`: edge_cases, non_functional, entities, assumptions, constraints, out_of_scope, clarifications
- `plan.yaml`: research, data_model, contracts, performance_goals, scale_scope
- `tasks.yaml`: depends_on, checkpoint, parallel_execution, incremental_delivery
- `checklist.yaml`: spec_reference, marker, notes
- `analysis.yaml`: unmapped_tasks, constitution_alignment.issues

---

## Command Generation Architecture

### Key Design Decision: AutoSpec Binary Generates Commands

The `autospec` Go binary will be responsible for generating Claude command files. This provides:

- **Single Source of Truth**: Commands are embedded in the binary or stored in `internal/`
- **Version Control**: Track which speckit version was used to generate commands
- **Reproducibility**: Same binary version always generates identical commands
- **No External Dependency**: Don't rely on external `speckit` CLI being installed

### Directory Structure

```
internal/
├── commands/                      # Command template sources (NOT .claude/)
│   ├── metadata.yaml              # Version tracking metadata
│   ├── autospec.specify.md        # YAML spec generation command
│   ├── autospec.plan.md           # YAML plan generation command
│   ├── autospec.tasks.md          # YAML tasks generation command
│   ├── autospec.checklist.md      # YAML checklist generation command
│   ├── autospec.clarify.md        # YAML spec clarification command
│   ├── autospec.implement.md      # YAML tasks execution command
│   ├── autospec.analyze.md        # YAML analysis generation command
│   └── autospec.constitution.md   # YAML constitution management command
├── schemas/                       # JSON Schema files for validation
│   ├── spec.schema.json
│   ├── plan.schema.json
│   ├── tasks.schema.json
│   ├── checklist.schema.json
│   ├── analysis.schema.json
│   └── constitution.schema.json
└── templates/                     # YAML output templates
    ├── spec-template.yaml
    ├── plan-template.yaml
    ├── tasks-template.yaml
    └── checklist-template.yaml
```

### Version Metadata File

`internal/commands/metadata.yaml`:

```yaml
# AutoSpec Command Generation Metadata
# This file tracks the source version used to generate these commands

generation:
  generated_at: "2025-12-13T00:00:00Z"
  generated_by: "autospec"
  autospec_version: "0.1.0"           # This binary's version

source:
  # Original SpecKit version these commands are derived from
  speckit_cli_version: "0.0.22"
  speckit_template_version: "0.0.90"
  speckit_release_date: "2025-12-04"

  # When we last synced with upstream speckit
  last_sync_date: "2025-12-13"
  sync_commit: ""                     # If tracking speckit git repo

compatibility:
  # Minimum Claude Code version required
  min_claude_code_version: "1.0.0"

  # Schema versions for YAML output
  schema_versions:
    spec: "1.0.0"
    plan: "1.0.0"
    tasks: "1.0.0"
    checklist: "1.0.0"
    analysis: "1.0.0"
    constitution: "1.0.0"

commands:
  - name: "autospec.specify"
    file: "autospec.specify.md"
    description: "Create feature specification (YAML output)"
    outputs: ["spec.yaml"]

  - name: "autospec.plan"
    file: "autospec.plan.md"
    description: "Create implementation plan (YAML output)"
    outputs: ["plan.yaml", "research.yaml", "data-model.yaml"]

  - name: "autospec.tasks"
    file: "autospec.tasks.md"
    description: "Generate task breakdown (YAML output)"
    outputs: ["tasks.yaml"]

  - name: "autospec.checklist"
    file: "autospec.checklist.md"
    description: "Generate quality checklist (YAML output)"
    outputs: ["{domain}.yaml"]

  - name: "autospec.clarify"
    file: "autospec.clarify.md"
    description: "Clarify specification requirements"
    outputs: []  # Updates spec.yaml in place

  - name: "autospec.implement"
    file: "autospec.implement.md"
    description: "Execute implementation tasks"
    outputs: []  # Updates tasks.yaml in place

  - name: "autospec.analyze"
    file: "autospec.analyze.md"
    description: "Cross-artifact analysis"
    outputs: ["analysis.yaml"]

  - name: "autospec.constitution"
    file: "autospec.constitution.md"
    description: "Manage project constitution"
    outputs: ["constitution.yaml"]
```

### CLI Commands for Command Management

```bash
# Generate/install commands to .claude/commands/
autospec commands install
autospec commands install --force     # Overwrite existing

# Show command metadata and version info
autospec commands info
# Output:
#   AutoSpec Version: 0.1.0
#   Based on SpecKit: CLI 0.0.22, Templates 0.0.90
#   Commands: 8 available
#   Schema Version: 1.0.0

# List available commands
autospec commands list

# Check if installed commands are up-to-date
autospec commands check
# Output:
#   .claude/commands/autospec.specify.md: OK (v1.0.0)
#   .claude/commands/autospec.plan.md: OUTDATED (v0.9.0 -> v1.0.0)

# Update specific command
autospec commands update autospec.plan

# Show diff between installed and latest
autospec commands diff autospec.plan

# Remove installed commands
autospec commands uninstall
```

### Go Implementation

```go
// internal/commands/embed.go
package commands

import "embed"

//go:embed *.md metadata.yaml
var CommandFiles embed.FS

//go:embed ../schemas/*.json
var SchemaFiles embed.FS

// GetMetadata returns parsed metadata
func GetMetadata() (*Metadata, error) { ... }

// GetCommand returns a specific command template
func GetCommand(name string) ([]byte, error) { ... }

// InstallCommands writes commands to .claude/commands/
func InstallCommands(targetDir string, force bool) error { ... }
```

---

## Implementation Plan

### Phase 1: Schema & Validation Infrastructure

1. Create JSON Schema files for each YAML schema
   - `internal/schemas/spec.schema.json`
   - `internal/schemas/plan.schema.json`
   - `internal/schemas/tasks.schema.json`
   - `internal/schemas/checklist.schema.json`
   - `internal/schemas/analysis.schema.json`
   - `internal/schemas/constitution.schema.json`

2. Add schema validation to Go codebase
   - `internal/schema/validator.go` - Generic YAML schema validator
   - Use `xeipuuv/gojsonschema` or similar library

3. Create YAML template files
   - `internal/templates/spec-template.yaml`
   - `internal/templates/plan-template.yaml`
   - `internal/templates/tasks-template.yaml`
   - `internal/templates/checklist-template.yaml`

### Phase 2: Create AutoSpec Commands

Create command template files in `internal/commands/` (NOT `.claude/commands/`):

1. `autospec.specify.md` - YAML spec generation
2. `autospec.plan.md` - YAML plan generation
3. `autospec.tasks.md` - YAML tasks generation
4. `autospec.checklist.md` - YAML checklist generation
5. `autospec.clarify.md` - YAML spec clarification
6. `autospec.implement.md` - YAML tasks execution
7. `autospec.analyze.md` - YAML analysis generation
8. `autospec.constitution.md` - YAML constitution management

Create `internal/commands/metadata.yaml` with version tracking.

### Phase 2.5: Command Management CLI

Add CLI commands for managing command installation:

1. `autospec commands install` - Copy commands to `.claude/commands/`
2. `autospec commands info` - Show version metadata
3. `autospec commands list` - List available commands
4. `autospec commands check` - Verify installed commands are current
5. `autospec commands update` - Update specific commands
6. `autospec commands uninstall` - Remove installed commands

### Phase 3: Go CLI Integration

1. Add YAML parsing utilities
   - `internal/yaml/parser.go` - Parse YAML artifacts
   - `internal/yaml/query.go` - Query specific fields/phases

2. Update validation logic
   - `internal/validation/yaml_validation.go` - Validate YAML artifacts
   - Support for required/optional field checking

3. Add CLI commands for YAML operations
   - `autospec yaml validate <file>` - Validate against schema
   - `autospec yaml query <file> <path>` - Query specific fields
   - `autospec yaml tasks --phase=1` - Extract phase-specific tasks

### Phase 4: Migration & Tooling

1. Create migration utility
   - `autospec migrate md-to-yaml <spec-dir>` - Convert existing MD to YAML

2. Add export functionality
   - `autospec export yaml-to-md <file>` - Generate MD from YAML (for human reading)
   - `autospec export yaml-to-json <file>` - Generate JSON (for API integration)

3. Update workflow orchestrator
   - Support both `.md` and `.yaml` artifact detection
   - Add `--format=yaml` flag to workflow commands

---

## Migration Strategy

### Backward Compatibility

- Original `/speckit.*` commands remain unchanged
- New `/autospec.*` commands operate in parallel
- Workflow orchestrator auto-detects file format

### Gradual Adoption

1. **Phase A**: Create YAML commands, run alongside MD
2. **Phase B**: Add `--format` flag to existing commands
3. **Phase C**: Make YAML default for new specs
4. **Phase D**: Deprecate MD-only workflow (optional)

### Coexistence Rules

- If `spec.yaml` exists, prefer it over `spec.md`
- If only `spec.md` exists, use it (backward compatible)
- Never auto-convert without explicit user action

---

## Benefits & Use Cases

### Programmatic Task Extraction

```bash
# Extract all Phase 1 tasks
yq '.phases[] | select(.number == 1) | .tasks[].description' tasks.yaml

# Get all pending tasks
yq '.phases[].tasks[] | select(.status == "Pending")' tasks.yaml

# Count tasks by user story
yq '.summary.tasks_by_story' tasks.yaml
```

### Validation Automation

```bash
# Validate spec has all required fields
autospec yaml validate spec.yaml

# Check if all user stories have acceptance scenarios
yq '.user_stories[] | select(.acceptance_scenarios | length == 0)' spec.yaml
```

### Integration with External Tools

```python
import yaml

with open('tasks.yaml') as f:
    tasks = yaml.safe_load(f)

# Extract phase 2 tasks for Jira import
phase2_tasks = [t for p in tasks['phases'] if p['number'] == 2 for t in p['tasks']]
```

### CI/CD Integration

```yaml
# GitHub Action example
- name: Validate Spec
  run: autospec yaml validate specs/${{ github.ref_name }}/spec.yaml

- name: Check Task Coverage
  run: |
    coverage=$(yq '.metrics.coverage_percentage' analysis.yaml)
    if [ "$coverage" -lt 80 ]; then
      echo "Coverage too low: $coverage%"
      exit 1
    fi
```

---

## Open Questions

1. **Schema Versioning**: How to handle schema evolution? Suggest `_meta.schema_version` field.

2. **Tooling Requirements**: Should `yq` be a required dependency, or bundle YAML query in Go CLI?

3. **Mixed Format Support**: Allow `spec.yaml` + `tasks.md` or require all-or-nothing?

4. **Template Customization**: Should users be able to extend schemas with custom fields?

5. **Frontmatter**: Keep YAML frontmatter in commands (`---\ndescription: ...\n---`) or move to separate config?

---

## Next Steps

1. [ ] Review and finalize YAML schemas
2. [ ] Create `internal/commands/` directory structure
3. [ ] Create `internal/commands/metadata.yaml` with version tracking (SpecKit 0.0.22/0.0.90)
4. [ ] Create JSON Schema files in `internal/schemas/`
5. [ ] Implement `autospec.specify.md` command (first command) in `internal/commands/`
6. [ ] Add Go embed for command files (`internal/commands/embed.go`)
7. [ ] Implement `autospec commands install` CLI command
8. [ ] Implement `autospec commands info` CLI command
9. [ ] Add schema validation to Go codebase
10. [ ] Create remaining autospec commands in `internal/commands/`
11. [ ] Add CLI utilities for YAML operations
12. [ ] Write migration tooling
13. [ ] Update documentation

---

## Version Tracking Summary

**Source SpecKit Version** (commands derived from):
- CLI Version: `0.0.22`
- Template Version: `0.0.90`
- Released: `2025-12-04`
- Compiled/Synced: `2025-12-13`

**AutoSpec Schema Version**: `1.0.0` (initial release)

This metadata will be embedded in `internal/commands/metadata.yaml` and accessible via `autospec commands info`.

---

## References

- Current SpecKit commands: `.claude/commands/speckit.*.md`
- Current templates: `.specify/templates/*.md`
- JSON Schema spec: https://json-schema.org/
- YAML 1.2 spec: https://yaml.org/spec/1.2.2/
- yq documentation: https://mikefarah.gitbook.io/yq/
- Go embed documentation: https://pkg.go.dev/embed
