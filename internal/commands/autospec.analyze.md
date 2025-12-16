---
description: Analyze cross-artifact consistency and quality in YAML format.
version: "1.0.0"
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Goal

Identify inconsistencies, duplications, ambiguities, and underspecified items across the core artifacts (spec, plan, tasks) before implementation. This command MUST run only after `/autospec.tasks` has successfully produced a complete tasks file.

## Operating Constraints

**STRICTLY READ-ONLY**: Do **not** modify any files. Output a structured analysis YAML file. Offer an optional remediation plan (user must explicitly approve before any follow-up editing commands would be invoked manually).

**Constitution Authority**: The project constitution (`.autospec/memory/constitution.yaml` or `CLAUDE.md`) is **non-negotiable** within this analysis scope. Constitution conflicts are automatically CRITICAL and require adjustment of the spec, plan, or tasks.

## Execution Steps

### 1. Initialize Analysis Context

Run the prerequisites command to get feature paths:

```bash
autospec prereqs --json --require-tasks --include-tasks
```

Parse the JSON output for:
- `FEATURE_DIR`: The feature directory path
- `FEATURE_SPEC`: Path to the spec file (spec.yaml)
- `IMPL_PLAN`: Path to the plan file (plan.yaml)
- `TASKS`: Path to the tasks file (tasks.yaml)
- `AVAILABLE_DOCS`: List of optional documents found
- `AUTOSPEC_VERSION`: The autospec version (for _meta section)
- `CREATED_DATE`: ISO 8601 timestamp (for _meta section)

If the script fails, it will output an error message instructing the user to run the missing prerequisite command.

### 2. Load Artifacts (Progressive Disclosure)

Load only the minimal necessary context from each artifact:

**From spec.yaml**:
- Overview/Context
- Functional Requirements
- Non-Functional Requirements
- User Stories
- Edge Cases (if present)

**From plan.yaml**:
- Architecture/stack choices
- Data Model references
- Phases
- Technical constraints

**From tasks.yaml**:
- Task IDs
- Descriptions
- Phase grouping
- Parallel markers
- Referenced file paths

**From constitution**:
- Load `.autospec/memory/constitution.yaml` or `CLAUDE.md` for principle validation

### 3. Build Semantic Models

Create internal representations (do not include raw artifacts in output):
- **Requirements inventory**: Each functional + non-functional requirement with a stable key
- **User story/action inventory**: Discrete user actions with acceptance criteria
- **Task coverage mapping**: Map each task to one or more requirements or stories
- **Constitution rule set**: Extract principle names and MUST/SHOULD normative statements

### 4. Detection Passes (Token-Efficient Analysis)

Focus on high-signal findings. Limit to 50 findings total; aggregate remainder in overflow summary.

#### A. Duplication Detection
- Identify near-duplicate requirements
- Mark lower-quality phrasing for consolidation

#### B. Ambiguity Detection
- Flag vague adjectives (fast, scalable, secure, intuitive, robust) lacking measurable criteria
- Flag unresolved placeholders (TODO, TKTK, ???, `<placeholder>`, etc.)

#### C. Underspecification
- Requirements with verbs but missing object or measurable outcome
- User stories missing acceptance criteria alignment
- Tasks referencing files or components not defined in spec/plan

#### D. Constitution Alignment
- Any requirement or plan element conflicting with a MUST principle
- Missing mandated sections or quality gates from constitution

#### E. Coverage Gaps
- Requirements with zero associated tasks
- Tasks with no mapped requirement/story
- Non-functional requirements not reflected in tasks (e.g., performance, security)

#### F. Inconsistency
- Terminology drift (same concept named differently across files)
- Data entities referenced in plan but absent in spec (or vice versa)
- Task ordering contradictions
- Conflicting requirements

### 5. Severity Assignment

Use this heuristic to prioritize findings:
- **CRITICAL**: Violates constitution MUST, missing core spec artifact, or requirement with zero coverage that blocks baseline functionality
- **HIGH**: Duplicate or conflicting requirement, ambiguous security/performance attribute, untestable acceptance criterion
- **MEDIUM**: Terminology drift, missing non-functional task coverage, underspecified edge case
- **LOW**: Style/wording improvements, minor redundancy not affecting execution order

### 6. Generate analysis.yaml

```yaml
analysis:
  branch: "<current git branch>"
  timestamp: "<ISO 8601 timestamp>"
  spec_path: "<relative path to spec file>"
  plan_path: "<relative path to plan file>"
  tasks_path: "<relative path to tasks file>"
  constitution_path: "<path to constitution or 'Not found'>"

findings:
  - id: "DUP-001"
    category: "duplication"
    severity: "HIGH"
    location: "spec.yaml:requirements.functional[2]"
    summary: "Two similar requirements for user login"
    details: "FR-002 and FR-005 both describe user authentication flow"
    recommendation: "Merge into single requirement; keep clearer phrasing"

  - id: "AMB-001"
    category: "ambiguity"
    severity: "MEDIUM"
    location: "spec.yaml:requirements.non_functional[0]"
    summary: "Vague performance requirement"
    details: "'Fast response time' lacks specific threshold"
    recommendation: "Quantify with specific metric (e.g., '<200ms p95')"

  - id: "COV-001"
    category: "coverage"
    severity: "HIGH"
    location: "spec.yaml:requirements.functional[3]"
    summary: "FR-004 has no corresponding task"
    details: "Password reset requirement not covered in tasks.yaml"
    recommendation: "Add task in User Story phase for FR-004 implementation"

  - id: "CON-001"
    category: "constitution"
    severity: "CRITICAL"
    location: "tasks.yaml:phases[2].tasks[0]"
    summary: "Missing test task before implementation"
    details: "Constitution requires test-first development"
    recommendation: "Add test task before implementation task T011"

  - id: "INC-001"
    category: "inconsistency"
    severity: "MEDIUM"
    location: "plan.yaml:data_model vs spec.yaml:key_entities"
    summary: "Entity naming mismatch"
    details: "'User' in spec but 'Account' in plan data model"
    recommendation: "Standardize naming across artifacts"

coverage:
  requirements:
    - id: "FR-001"
      has_task: true
      task_ids: ["T010", "T011"]
      notes: ""
    - id: "FR-002"
      has_task: false
      task_ids: []
      notes: "COVERAGE GAP"

  user_stories:
    - id: "US-001"
      tasks_count: 5
      coverage_status: "complete"
    - id: "US-002"
      tasks_count: 0
      coverage_status: "missing"

constitution_alignment:
  status: "PASS"  # or "FAIL" if any CRITICAL constitution issues
  violations:
    - principle: "Test-First Development"
      status: "VIOLATION"
      details: "3 implementation tasks lack preceding test tasks"

unmapped_tasks:
  - task_id: "T015"
    title: "Refactor logging"
    issue: "No corresponding requirement or story"

metrics:
  total_requirements: <number>
  total_tasks: <number>
  coverage_percentage: <number>  # requirements with >=1 task
  ambiguity_count: <number>
  duplication_count: <number>
  critical_issues: <number>
  high_issues: <number>
  medium_issues: <number>
  low_issues: <number>

summary:
  overall_status: "<PASS|WARN|FAIL>"
  blocking_issues: <number of CRITICAL issues>
  actionable_improvements: <number of HIGH/MEDIUM issues>
  ready_for_implementation: <true|false>

_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "<AUTOSPEC_VERSION from step 1>"
  created: "<CREATED_DATE from step 1>"
  artifact_type: "analysis"
```

### 7. Write the analysis to `FEATURE_DIR/analysis.yaml`

### 8. Validate the YAML

```bash
autospec yaml check FEATURE_DIR/analysis.yaml
```
- If validation fails: fix YAML syntax errors and retry
- If validation passes: proceed to report

### 9. Report Next Actions

At end of analysis, output a concise summary:
- If CRITICAL issues exist: Recommend resolving before implementation
- If only LOW/MEDIUM: User may proceed, but provide improvement suggestions
- Provide explicit command suggestions for remediation

### 10. Offer Remediation

Ask the user: "Would you like me to suggest concrete remediation edits for the top N issues?" (Do NOT apply them automatically.)

## Operating Principles

### Context Efficiency
- **Minimal high-signal tokens**: Focus on actionable findings, not exhaustive documentation
- **Progressive disclosure**: Load artifacts incrementally; don't dump all content into analysis
- **Token-efficient output**: Limit findings to 50; summarize overflow
- **Deterministic results**: Rerunning without changes should produce consistent IDs and counts

### Analysis Guidelines
- **NEVER modify files** (this is read-only analysis)
- **NEVER hallucinate missing sections** (if absent, report them accurately)
- **Prioritize constitution violations** (these are always CRITICAL)
- **Use examples over exhaustive rules** (cite specific instances, not generic patterns)
- **Report zero issues gracefully** (emit success report with coverage statistics)
