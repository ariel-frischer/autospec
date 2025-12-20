# Constitution Improvement Proposals

> Generated: 2025-12-19
> Updated: 2025-12-19 (added findings from 20 additional conversations)
> Source: Analysis of Claude Code conversations for constitution gaps
> Tracking file: `.dev/feedback/constitution-review-sessions.txt`

## Executive Summary

Analysis of 35+ autospec-triggered conversations revealed patterns and behaviors that suggest improvements to `.autospec/memory/constitution.yaml`. These proposals focus on codifying observed best practices and closing gaps where implicit patterns should become explicit principles.

---

## Category 1: Artifact Quality Principles

### ~~PRIN-013: Cross-Artifact Traceability~~ (REJECTED)

**Status**: NOT NEEDED - Artifacts are organized by folder (`specs/<name>/`) so paths are inferred. Adding explicit path references wastes tokens and may cause agents to read additional files unnecessarily.

---

### PRIN-014: Assumption Documentation

**Observed Pattern**: Specs include explicit `assumptions:` sections listing what's taken as given (e.g., "The 'st' alias for status already exists").

**Gap**: Constitution doesn't require documenting assumptions, leading to implicit knowledge.

**Proposed Principle**:
```yaml
- name: "Explicit Assumptions"
  id: "PRIN-014"
  category: "quality"
  priority: "SHOULD"
  description: |
    Specifications must document assumptions that inform design decisions.
    Assumptions should be verifiable where possible.
    Invalid assumptions discovered during implementation require spec amendment.
  rationale: "Surfaces implicit knowledge, enables validation, prevents downstream surprises"
  enforcement:
    - mechanism: "spec.yaml schema"
      description: "assumptions: field is required (can be empty list)"
    - mechanism: "Code review"
      description: "Reviewers verify assumptions are documented and valid"
```

### PRIN-015: Scope Boundary Definition

**Observed Pattern**: Every spec includes `out_of_scope:` and `constraints:` sections that prevent scope creep and clarify boundaries.

**Gap**: Constitution doesn't explicitly require scope boundaries, though templates include them.

**Proposed Principle**:
```yaml
- name: "Explicit Scope Boundaries"
  id: "PRIN-015"
  category: "process"
  priority: "MUST"
  description: |
    All specifications must define explicit boundaries:
    - out_of_scope: What this feature explicitly does NOT include
    - constraints: Technical/process limitations that bound the solution
    Implementation must not exceed defined scope without spec amendment.
  rationale: "Prevents scope creep, enables focused implementation, supports estimation"
  enforcement:
    - mechanism: "spec.yaml schema"
      description: "out_of_scope and constraints fields are required"
    - mechanism: "Implementation review"
      description: "PRs checked against scope boundaries"
```

---

## Category 2: Task Execution Principles

### PRIN-016: Quality Gate as Final Phase

**Observed Pattern**: Every tasks.yaml ends with a "Validation and Quality Gates" phase containing: make test, make fmt, make lint, make build (in that order, with fmt/lint/build parallel after test).

**Gap**: This pattern is implicit in templates but not codified as a principle.

**Proposed Principle**:
```yaml
- name: "Quality Gate Final Phase"
  id: "PRIN-016"
  category: "quality"
  priority: "NON-NEGOTIABLE"
  description: |
    Every tasks.yaml must end with a Quality Gates phase containing:
    1. make test (runs first, blocks subsequent tasks)
    2. make fmt (parallel after test)
    3. make lint (parallel after test)
    4. make build (parallel after test)
    Implementation cannot be marked complete without all gates passing.
  rationale: "Ensures every feature meets baseline quality before merge"
  enforcement:
    - mechanism: "tasks.yaml schema"
      description: "Validator checks for quality gate phase with required tasks"
    - mechanism: "CI integration"
      description: "autospec st shows completion only when all gates pass"
```

### PRIN-017: Task Dependency and Parallelization

**Observed Pattern**: Tasks use `dependencies:` arrays and `parallel: true` flags. Test task runs first, then fmt/lint/build run in parallel.

**Gap**: Constitution doesn't specify how task dependencies should be structured.

**Proposed Principle**:
```yaml
- name: "Task Dependency Structure"
  id: "PRIN-017"
  category: "architecture"
  priority: "MUST"
  description: |
    Task dependencies must be explicitly declared:
    - dependencies: Array of task IDs that must complete first
    - parallel: Boolean indicating if task can run with siblings
    Tasks with no dependencies run first. Parallel tasks execute concurrently
    when all dependencies are satisfied. Circular dependencies are invalid.
  rationale: "Enables optimal execution order and parallelization"
  enforcement:
    - mechanism: "tasks.yaml schema"
      description: "Validator checks for valid dependency graph (no cycles)"
    - mechanism: "Execution engine"
      description: "autospec implement respects dependency ordering"
```

---

## Category 3: Design Decision Principles

### PRIN-018: Open Questions Resolution

**Observed Pattern**: plan.yaml includes `open_questions:` with `question`, `context`, and `proposed_resolution` fields. Questions are resolved before implementation.

**Gap**: Constitution doesn't require documenting design decisions and their rationale.

**Proposed Principle**:
```yaml
- name: "Design Decision Documentation"
  id: "PRIN-018"
  category: "process"
  priority: "SHOULD"
  description: |
    Plans must document design decisions via open_questions:
    - question: The decision point
    - context: Why this decision matters
    - proposed_resolution: The chosen approach and rationale
    Unresolved questions block task generation.
  rationale: "Preserves decision rationale, enables future review, prevents rediscovery"
  enforcement:
    - mechanism: "plan.yaml schema"
      description: "open_questions field required (can be empty)"
    - mechanism: "Workflow gate"
      description: "Tasks command warns if open_questions lack resolutions"
```

### PRIN-019: Existing Infrastructure Reuse

**Observed Pattern**: Plans consistently state "Must use existing infrastructure" (e.g., "CheckArtifactDependencies", "artifactDependencies map"). New code should extend, not duplicate.

**Gap**: Constitution mentions idempotency but not infrastructure reuse.

**Proposed Principle**:
```yaml
- name: "Infrastructure Reuse"
  id: "PRIN-019"
  category: "architecture"
  priority: "SHOULD"
  description: |
    New features should extend existing infrastructure rather than creating parallel systems.
    Before implementing new functionality, search for existing utilities, helpers, and patterns.
    Document why new infrastructure is needed when existing patterns don't fit.
  rationale: "Reduces code duplication, maintains consistency, simplifies maintenance"
  enforcement:
    - mechanism: "Code review"
      description: "PRs justify new infrastructure vs extending existing"
    - mechanism: "plan.yaml guidance"
      description: "Plans identify existing infrastructure to leverage"
```

---

## Category 4: Compatibility Principles

### PRIN-020: Backward Compatibility

**Observed Pattern**: Multiple specs include constraints like "Must maintain backward compatibility", "additive changes only", "Must not break existing command functionality".

**Gap**: Constitution doesn't have explicit backward compatibility principle.

**Proposed Principle**:
```yaml
- name: "Backward Compatibility"
  id: "PRIN-020"
  category: "architecture"
  priority: "MUST"
  description: |
    Changes must maintain backward compatibility unless explicitly breaking:
    - Additive changes preferred (new fields, new commands)
    - Existing behavior preserved for valid inputs
    - Deprecation period required for removals
    Breaking changes require CHANGELOG entry and version bump.
  rationale: "Protects existing users and integrations from unexpected breakage"
  enforcement:
    - mechanism: "spec.yaml constraints"
      description: "Templates inject backward compatibility as default constraint"
    - mechanism: "Semantic versioning"
      description: "Breaking changes require major version bump"
  exceptions:
    - "Security fixes may break compatibility with notice"
    - "Explicit deprecation after notice period"
```

---

## Category 5: Efficiency Principles

### PRIN-021: Context Efficiency

**Observed Pattern**: The phase-context-metadata feature was born from observing 15K tokens/session wasted on redundant reads. Metadata fields (`_context_meta`, `skip_reads`) optimize context usage.

**Gap**: Constitution focuses on code performance but not token/context efficiency.

**Proposed Principle**:
```yaml
- name: "Context Efficiency"
  id: "PRIN-021"
  category: "quality"
  priority: "SHOULD"
  description: |
    Workflow artifacts should minimize context consumption:
    - Bundle related information to avoid redundant reads
    - Include skip hints when content is duplicated
    - Prefer machine-readable metadata over prose instructions
    Target: Eliminate redundant artifact reads within sessions.
  rationale: "Reduces API costs, speeds execution, prevents context window exhaustion"
  enforcement:
    - mechanism: "Phase context generation"
      description: "_context_meta indicates which files are bundled"
    - mechanism: "Template design"
      description: "Templates reference bundled content, not separate files"
```

### PRIN-022: Fail-Fast Validation

**Observed Pattern**: The prereq-validation feature implements "fail fast on missing artifacts" to avoid wasting API costs on predictable failures.

**Gap**: Constitution mentions validation but not the fail-fast philosophy.

**Proposed Principle**:
```yaml
- name: "Fail-Fast Validation"
  id: "PRIN-022"
  category: "architecture"
  priority: "MUST"
  description: |
    Operations must validate prerequisites before expensive operations:
    - CLI commands check artifact dependencies before invoking Claude
    - Schema validation happens before workflow proceeds
    - Missing prerequisites produce immediate, actionable errors
  rationale: "Saves API costs, provides better user experience, enables quick iteration"
  enforcement:
    - mechanism: "CLI preflight"
      description: "All stage commands run prerequisite checks first"
    - mechanism: "Exit codes"
      description: "Exit code 3 for missing prerequisites (before Claude invocation)"
```

---

## Category 6: Test Quality Principles

### PRIN-023: Test Coverage Targets

**Observed Pattern**: Test coverage spec defined explicit targets (85% standard, 90% critical). Current tests vary from 37% to 96%.

**Gap**: Constitution says "Tests define expected behavior" but doesn't specify coverage targets.

**Proposed Addition to PRIN-001**:
```yaml
# Add to existing Test-First Development principle:
description: |
  All new code must have tests written before implementation.
  Tests define the expected behavior and serve as living documentation.
  Unit tests use table-driven patterns for comprehensive coverage.
  Coverage targets:
  - Critical packages (workflow, validation, config, errors): 90% minimum
  - Standard packages: 85% minimum
  - Coverage must not decrease without justification
```

---

## Summary Table

| ID | Name | Category | Priority | Key Benefit |
|----|------|----------|----------|-------------|
| PRIN-013 | Cross-Artifact Traceability | architecture | MUST | Audit trails |
| PRIN-014 | Explicit Assumptions | quality | SHOULD | Surface implicit knowledge |
| PRIN-015 | Explicit Scope Boundaries | process | MUST | Prevent scope creep |
| PRIN-016 | Quality Gate Final Phase | quality | NON-NEGOTIABLE | Baseline quality |
| PRIN-017 | Task Dependency Structure | architecture | MUST | Optimal execution |
| PRIN-018 | Design Decision Documentation | process | SHOULD | Preserve rationale |
| PRIN-019 | Infrastructure Reuse | architecture | SHOULD | Reduce duplication |
| PRIN-020 | Backward Compatibility | architecture | MUST | Protect users |
| PRIN-021 | Context Efficiency | quality | SHOULD | Reduce costs |
| PRIN-022 | Fail-Fast Validation | architecture | MUST | Better UX |
| PRIN-023 | Test Coverage Targets | quality | MUST | Measurable quality |

---

## Implementation Priority

**Phase 1 - High Impact** (recommend implementing first):
1. PRIN-020 (Backward Compatibility) - Protects users
2. PRIN-022 (Fail-Fast Validation) - Already partially implemented
3. PRIN-016 (Quality Gate Final Phase) - Already in practice

**Phase 2 - Process Improvement**:
4. PRIN-015 (Scope Boundaries) - Already in templates
5. PRIN-018 (Design Decisions) - Already in plan schema
6. PRIN-013 (Traceability) - Already partially implemented

**Phase 3 - Quality Enhancement**:
7. PRIN-023 (Coverage Targets) - Extends PRIN-001
8. PRIN-021 (Context Efficiency) - New concern
9. PRIN-014 (Assumptions) - Documentation improvement

**Phase 4 - Architecture**:
10. PRIN-017 (Task Dependencies) - Already in schema
11. PRIN-019 (Infrastructure Reuse) - Guidance principle

---

## Category 7: Specification Process Principles (NEW from additional analysis)

### PRIN-024: Understand Before Specifying

**Observed Pattern**: In multiple sessions, Claude gathered real data before writing specs:
- Test coverage spec ran `go test -cover` to get actual metrics
- Uninstall spec read `install.sh` to understand current install locations
- Status enhancement spec read existing `status.go` to identify bugs with line numbers

**Gap**: Constitution focuses on what specs should contain, not how to gather requirements.

**Proposed Principle**:
```yaml
- name: "Data-Driven Specification"
  id: "PRIN-024"
  category: "process"
  priority: "SHOULD"
  description: |
    Before writing specifications, gather concrete data about the current state:
    - For improvement features: Measure current metrics (coverage, performance, etc.)
    - For bug fixes: Identify specific locations (file:line) of issues
    - For new features: Review related existing code patterns
    Specifications should reference actual measured values, not assumptions.
  rationale: "Grounded specs lead to realistic plans and measurable success criteria"
  enforcement:
    - mechanism: "Template guidance"
      description: "Specify template prompts for data gathering before writing"
    - mechanism: "Review"
      description: "Specs with quantitative claims should cite measurement method"
```

### PRIN-025: Task Documents as Detailed Input

**Observed Pattern**: Many specs are generated from `.dev/tasks/*.md` files that contain:
- Problem statement with context
- Proposed solution approach
- Implementation notes
- Specific file paths to modify

**Gap**: Constitution doesn't recognize the role of pre-spec research documents.

**Proposed Principle**:
```yaml
- name: "Pre-Spec Research Documents"
  id: "PRIN-025"
  category: "process"
  priority: "SHOULD"
  description: |
    Complex features benefit from research documents before spec generation:
    - Store in .dev/tasks/ with descriptive names
    - Include problem statement, proposed approach, affected files
    - Reference from spec input field
    These documents capture discovery work that informs the specification.
  rationale: "Separates research from specification, preserves discovery context"
  enforcement:
    - mechanism: "Convention"
      description: ".dev/tasks/ directory for pre-spec research"
    - mechanism: "Spec input field"
      description: "Reference task doc path when spec is derived from research"
```

### PRIN-026: Proportional Specification Depth

**Observed Pattern**: Simple features (command aliases) get minimal specs with 2 user stories. Complex features (event-driven notifications) get comprehensive specs with 7+ user stories, extensive edge cases.

**Gap**: Constitution doesn't guide how detailed specs should be based on feature complexity.

**Proposed Principle**:
```yaml
- name: "Proportional Specification"
  id: "PRIN-026"
  category: "process"
  priority: "SHOULD"
  description: |
    Specification depth should match feature complexity:
    - Simple (1-2 files, no new patterns): 1-2 user stories, minimal edge cases
    - Medium (3-5 files, extends patterns): 3-4 user stories, key edge cases
    - Complex (6+ files, new patterns): 5+ user stories, comprehensive edge cases
    Over-specification wastes tokens; under-specification causes implementation gaps.
  rationale: "Balances thoroughness with efficiency based on actual complexity"
  enforcement:
    - mechanism: "Template heuristics"
      description: "Specify template suggests depth based on feature description analysis"
```

### PRIN-027: Codebase Exploration Before Planning

**Observed Pattern**: Plan sessions frequently use Task tool with Explore agent to understand:
- Existing config loading patterns
- CLI command structures
- State persistence mechanisms
- Package organization

**Gap**: Constitution doesn't formalize exploration as a planning prerequisite.

**Proposed Principle**:
```yaml
- name: "Exploration Before Planning"
  id: "PRIN-027"
  category: "process"
  priority: "SHOULD"
  description: |
    Before generating implementation plans, explore relevant codebase areas:
    - Identify existing patterns that should be followed
    - Locate infrastructure to reuse (extends PRIN-019)
    - Understand package boundaries and dependencies
    Plans should cite specific existing code patterns as references.
  rationale: "Plans grounded in existing code are more accurate and maintainable"
  enforcement:
    - mechanism: "Plan template"
      description: "Template prompts for codebase exploration before plan writing"
    - mechanism: "technical_context section"
      description: "Plans list existing patterns discovered during exploration"
```

---

## Category 8: Workflow Resilience Principles (NEW)

### PRIN-028: Graceful Tool Degradation

**Observed Pattern**: When Serena MCP fails ("language server not initialized"), Claude falls back to standard tools (Grep, Read, Glob) without interrupting the workflow.

**Gap**: Constitution doesn't address tool failure handling.

**Proposed Principle**:
```yaml
- name: "Graceful Tool Degradation"
  id: "PRIN-028"
  category: "architecture"
  priority: "SHOULD"
  description: |
    When advanced tools fail, workflows should degrade gracefully:
    - MCP tool failures trigger fallback to standard tools
    - Tool errors logged but don't block workflow progress
    - Workflows should not depend on any single tool exclusively
  rationale: "Ensures reliability despite tool availability variations"
  enforcement:
    - mechanism: "Template design"
      description: "Templates don't assume specific MCP tools are available"
    - mechanism: "Agent behavior"
      description: "Agents detect tool errors and use alternatives"
```

### PRIN-029: Immediate Artifact Validation

**Observed Pattern**: Every spec, plan, and tasks.yaml is validated immediately after creation using `autospec artifact` or `autospec yaml check`.

**Gap**: Constitution doesn't mandate post-creation validation.

**Proposed Principle**:
```yaml
- name: "Immediate Artifact Validation"
  id: "PRIN-029"
  category: "quality"
  priority: "MUST"
  description: |
    Every generated artifact must be validated immediately after creation:
    - Use `autospec yaml check` for syntax validation
    - Use `autospec artifact <type>` for schema validation
    - Invalid artifacts must be fixed before workflow proceeds
  rationale: "Catches errors early, prevents invalid artifacts from propagating"
  enforcement:
    - mechanism: "Template instructions"
      description: "All stage templates include validation step after write"
    - mechanism: "Workflow gate"
      description: "Subsequent stages check artifact validity before proceeding"
```

---

## Category 9: Clarification Workflow Principles (NEW)

### PRIN-030: Taxonomy-Based Ambiguity Analysis

**Observed Pattern**: The clarify command uses a structured taxonomy for analyzing specs:
- Functional Scope & Behavior
- Domain & Data Model
- Interaction & UX Flow
- Non-Functional Quality Attributes
- Integration & External Dependencies
- Edge Cases & Failure Handling
- Constraints & Tradeoffs
- Terminology & Consistency
- Completion Signals

**Gap**: Constitution doesn't codify the ambiguity detection approach.

**Proposed Principle**:
```yaml
- name: "Structured Ambiguity Detection"
  id: "PRIN-030"
  category: "quality"
  priority: "SHOULD"
  description: |
    Specification review should use a structured taxonomy to identify gaps:
    - Functional scope, domain model, UX flow
    - Non-functional requirements, integrations
    - Edge cases, constraints, terminology
    Each category rated Clear/Partial/Missing to prioritize clarifications.
  rationale: "Systematic analysis catches gaps that ad-hoc review misses"
  enforcement:
    - mechanism: "Clarify template"
      description: "Template guides structured category-by-category analysis"
```

### PRIN-031: Recommended Options with Rationale

**Observed Pattern**: When clarify asks questions, it always includes:
- A recommended option (usually Option A)
- Rationale for the recommendation
- 3-4 alternative options with descriptions

**Gap**: Constitution doesn't specify how decisions should be presented.

**Proposed Principle**:
```yaml
- name: "Decision Presentation Format"
  id: "PRIN-031"
  category: "process"
  priority: "SHOULD"
  description: |
    When presenting design decisions for user input:
    - Lead with a recommended option and rationale
    - Provide 3-4 mutually exclusive alternatives
    - Include description for each option's tradeoffs
    - Allow custom "Short" answer for unanticipated choices
  rationale: "Reduces decision fatigue while preserving user agency"
  enforcement:
    - mechanism: "Clarify template"
      description: "Template format includes recommendation with rationale"
```

---

## Updated Summary Table

| ID | Name | Category | Priority | Status |
|----|------|----------|----------|--------|
| ~~PRIN-013~~ | ~~Cross-Artifact Traceability~~ | - | - | REJECTED |
| PRIN-014 | Explicit Assumptions | quality | SHOULD | Proposed |
| PRIN-015 | Explicit Scope Boundaries | process | MUST | Proposed |
| PRIN-016 | Quality Gate Final Phase | quality | NON-NEGOTIABLE | Proposed |
| PRIN-017 | Task Dependency Structure | architecture | MUST | Proposed |
| PRIN-018 | Design Decision Documentation | process | SHOULD | Proposed |
| PRIN-019 | Infrastructure Reuse | architecture | SHOULD | Proposed |
| PRIN-020 | Backward Compatibility | architecture | MUST | Proposed |
| PRIN-021 | Context Efficiency | quality | SHOULD | Proposed |
| PRIN-022 | Fail-Fast Validation | architecture | MUST | Proposed |
| PRIN-023 | Test Coverage Targets | quality | MUST | Proposed |
| PRIN-024 | Data-Driven Specification | process | SHOULD | NEW |
| PRIN-025 | Pre-Spec Research Documents | process | SHOULD | NEW |
| PRIN-026 | Proportional Specification | process | SHOULD | NEW |
| PRIN-027 | Exploration Before Planning | process | SHOULD | NEW |
| PRIN-028 | Graceful Tool Degradation | architecture | SHOULD | NEW |
| PRIN-029 | Immediate Artifact Validation | quality | MUST | NEW |
| PRIN-030 | Structured Ambiguity Detection | quality | SHOULD | NEW |
| PRIN-031 | Recommended Options with Rationale | process | SHOULD | NEW |

---

## Next Steps

1. Review this document with maintainers
2. Prioritize which principles to add
3. For each approved principle:
   - Add to constitution.yaml
   - Update templates if needed
   - Add validation if enforcement requires it
4. Bump constitution version (1.2.0 -> 1.3.0 for this batch)
