---
layout: default
title: YAML Schemas
parent: Reference
nav_order: 3
---

# YAML Schemas
{: .no_toc }

Structure and validation rules for autospec YAML artifacts.
{: .fs-6 .fw-300 }

<details open markdown="block">
  <summary>
    Table of contents
  </summary>
  {: .text-delta }
1. TOC
{:toc}
</details>

---

## Overview

autospec uses YAML-based artifacts for machine-readable output and programmatic access. Each artifact includes a `_meta` section for versioning and validation.

### Artifact Types

| Artifact | File | Description |
|:---------|:-----|:------------|
| spec | `spec.yaml` | Feature specification with requirements |
| plan | `plan.yaml` | Implementation design and architecture |
| tasks | `tasks.yaml` | Task breakdown with dependencies |
| checklist | `checklists/*.yaml` | Quality validation checklists |
| analysis | `analysis.yaml` | Cross-artifact consistency analysis |
| constitution | `constitution.yaml` | Project principles and guidelines |

---

## Common Meta Section

All artifacts include this structure:

```yaml
_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "0.5.0"
  created: "2025-12-13T10:30:00Z"
  artifact_type: "spec"  # spec, plan, tasks, checklist, analysis, constitution
```

---

## spec.yaml

Feature specification with requirements and user stories.

### Structure

```yaml
_meta:
  version: "1.0.0"
  generator: "autospec"
  artifact_type: "spec"
  created: "2025-12-13T10:30:00Z"

feature:
  name: "User Authentication"
  branch: "008-user-auth"
  status: "Draft"  # Draft | Completed
  created: "2025-12-13"
  input: "Original feature description"
  completed_at: "2025-12-16T14:30:00Z"  # Set when status is Completed

user_stories:
  - id: "US-001"
    title: "User Login"
    as_a: "registered user"
    i_want: "to log in with my credentials"
    so_that: "I can access my account"
    priority: "P1"  # P1, P2, P3
    why_this_priority: "Core functionality required for all features"
    independent_test: "User can log in with valid credentials"
    acceptance_scenarios:
      - given: "valid credentials"
        when: "user submits login form"
        then: "user is authenticated and redirected to dashboard"

requirements:
  functional:
    - id: "FR-001"
      description: "System MUST validate user credentials against database"
      acceptance_criteria: "Login succeeds with valid email/password"
  non_functional:
    - id: "NFR-001"
      category: "performance"
      description: "Login response time MUST be under 500ms"
      measurable_target: "p95 latency < 500ms"

constraints:
  - "Must integrate with existing session management"
  - "OAuth providers: Google, GitHub only"

assumptions:
  - "Users have verified email addresses"
  - "Password requirements follow OWASP guidelines"

key_entities:
  - name: "User"
    description: "Registered application user"
    attributes:
      - "email"
      - "password_hash"
      - "created_at"

out_of_scope:
  - "Two-factor authentication"
  - "Social login besides Google/GitHub"

edge_cases:
  - scenario: "User enters wrong password 5 times"
    expected_behavior: "Account locked for 15 minutes"

success_criteria:
  measurable_outcomes:
    - id: "SC-001"
      description: "Users can authenticate successfully"
      metric: "Login success rate"
      target: ">99% for valid credentials"
```

### Status Values

| Status | Description |
|:-------|:------------|
| `Draft` | Initial state when spec is created |
| `Completed` | Set automatically when all tasks finish |

When implementation completes, autospec updates `status` to `Completed` and adds `completed_at` timestamp.

---

## plan.yaml

Implementation plan with technical context and architecture.

### Structure

```yaml
_meta:
  version: "1.0.0"
  generator: "autospec"
  artifact_type: "plan"
  created: "2025-12-13T11:00:00Z"

plan:
  branch: "008-user-auth"
  spec_path: "specs/008-user-auth/spec.yaml"
  created: "2025-12-13"

summary: |
  Implement user authentication using JWT tokens with
  bcrypt password hashing. Integration with existing
  middleware for session management.

technical_context:
  language: "Go"
  framework: "Gin"
  project_type: "web-api"
  target_platform: "Linux/Docker"
  primary_dependencies:
    - name: "golang-jwt/jwt"
      version: "5.0"
      purpose: "JWT token generation and validation"
  constraints:
    - "Must integrate with existing middleware"
  performance_goals: "p95 latency < 500ms"
  testing:
    framework: "testing + testify"
    approach: "Unit tests + integration tests"
  storage: "PostgreSQL"
  scale_scope: "1000 concurrent users"

research_findings:
  decisions:
    - topic: "Token type"
      decision: "JWT with short expiry"
      rationale: "Stateless, scalable, industry standard"
      alternatives_considered:
        - "Session cookies"
        - "Opaque tokens"

data_model:
  entities:
    - name: "User"
      description: "Application user account"
      fields:
        - name: "id"
          type: "uuid"
          constraints: "Primary key"
        - name: "email"
          type: "string"
          constraints: "Unique, not null"
        - name: "password_hash"
          type: "string"
          constraints: "Not null"
      relationships:
        - target: "Session"
          type: "one-to-many"
          description: "User has many sessions"

api_contracts:
  endpoints:
    - method: "POST"
      path: "/auth/login"
      description: "Authenticate user"
      request:
        content_type: "application/json"
        body_schema:
          email: "string (required)"
          password: "string (required)"
      response:
        success_code: 200
        success_schema:
          token: "string"
          expires_at: "timestamp"
        error_codes:
          - code: 401
            description: "Invalid credentials"

project_structure:
  source_code:
    - path: "internal/auth/handler.go"
      description: "HTTP handlers for auth endpoints"
    - path: "internal/auth/service.go"
      description: "Business logic for authentication"
  tests:
    - path: "internal/auth/handler_test.go"
      description: "Handler unit tests"
  documentation:
    - path: "docs/auth.md"
      description: "Authentication API documentation"

implementation_phases:
  - phase: 1
    name: "Setup"
    goal: "Initialize auth package structure"
    deliverables:
      - "Package directory structure"
      - "Dependency installation"
    dependencies: []
  - phase: 2
    name: "Core Implementation"
    goal: "Implement login/logout handlers"
    deliverables:
      - "Login endpoint"
      - "JWT generation"
    dependencies:
      - "Phase 1"

constitution_check:
  constitution_path: ".autospec/constitution.yaml"
  gates:
    - name: "Test-First Development"
      status: "PASS"
      notes: "Tests planned before implementation"

risks:
  - risk: "Token expiry edge cases"
    likelihood: "medium"
    impact: "high"
    mitigation: "Comprehensive refresh token tests"

open_questions:
  - question: "Should refresh tokens be stored in database?"
    context: "Trade-off between security and complexity"
    proposed_resolution: "Store in database with rotation"
```

### Consolidated Content

plan.yaml consolidates content that was previously in separate files:

| Previous File | Now In |
|:--------------|:-------|
| `research.md` | `research_findings` |
| `data-model.md` | `data_model` |
| `contracts/*.yaml` | `api_contracts` |
| `quickstart.md` | `implementation_phases` |

---

## tasks.yaml

Task breakdown with phases and dependencies.

### Structure

```yaml
_meta:
  version: "1.0.0"
  generator: "autospec"
  artifact_type: "tasks"
  created: "2025-12-13T12:00:00Z"

tasks:
  branch: "008-user-auth"
  spec_path: "specs/008-user-auth/spec.yaml"
  plan_path: "specs/008-user-auth/plan.yaml"
  created: "2025-12-13"

summary:
  total_tasks: 15
  total_phases: 4
  parallel_opportunities: 5
  estimated_complexity: "medium"

phases:
  - number: 1
    title: "Setup"
    purpose: "Initialize project structure and dependencies"
    tasks:
      - id: "T001"
        title: "Create auth package directory"
        status: "Pending"  # Pending, InProgress, Completed, Blocked
        type: "setup"  # setup, implementation, test
        parallel: false
        story_id: null
        file_path: "internal/auth/"
        dependencies: []
        acceptance_criteria:
          - "Directory internal/auth/ exists"
          - "Package declaration in place"
        notes: ""  # Optional free-form notes (max 1000 chars)

  - number: 2
    title: "User Story 1 - User Login (US-001)"
    purpose: "Implement user login functionality"
    story_reference: "US-001"
    independent_test: "User can log in with valid credentials"
    tasks:
      - id: "T002"
        title: "Write login handler tests"
        status: "Pending"
        type: "test"
        parallel: false
        story_id: "US-001"
        file_path: "internal/auth/handler_test.go"
        dependencies: ["T001"]
        acceptance_criteria:
          - "Tests cover valid login"
          - "Tests cover invalid credentials"
      - id: "T003"
        title: "Implement login handler"
        status: "Pending"
        type: "implementation"
        parallel: false
        story_id: "US-001"
        file_path: "internal/auth/handler.go"
        dependencies: ["T002"]
        acceptance_criteria:
          - "All tests pass"
          - "Returns JWT on success"

dependencies:
  user_story_order:
    - story_id: "US-001"
      depends_on: []
      blocks: ["US-002"]
  phase_order:
    - phase: 1
      blocks: [2, 3, 4]
    - phase: 2
      blocks: [3]

parallel_execution:
  - phase: 2
    parallel_groups:
      - tasks: ["T004", "T005"]
        rationale: "Independent test files"

implementation_strategy:
  mvp_scope:
    phases: [1, 2]
    description: "Basic login functionality"
    validation: "User can authenticate with valid credentials"
  incremental_delivery:
    - milestone: "Setup Complete"
      phases: [1]
      deliverable: "Project structure ready"
    - milestone: "MVP Ready"
      phases: [1, 2]
      deliverable: "Login working"
```

### Task Status Values

| Status | Description |
|:-------|:------------|
| `Pending` | Not started |
| `InProgress` | Currently being worked on |
| `Completed` | Finished successfully |
| `Blocked` | Blocked by dependency or issue |

Update task status with:

```bash
autospec update-task T001 InProgress
autospec update-task T001 Completed
```

---

## checklist.yaml

Quality checklists stored in `checklists/` subdirectory.

### Structure

```yaml
_meta:
  version: "1.0.0"
  generator: "autospec"
  artifact_type: "checklist"
  created: "2025-12-13T13:00:00Z"

checklist:
  title: "Security Checklist"
  purpose: "Validate security requirements"
  feature_ref: "../spec.yaml"

categories:
  - name: "Authentication"
    items:
      - id: "CHK001"
        description: "Passwords hashed with bcrypt"
        quality_dimension: "security"
        spec_reference: "NFR-002"
        status: "pass"  # pending, pass, fail
        notes: "Using bcrypt with cost factor 12"
      - id: "CHK002"
        description: "JWT tokens have expiry"
        quality_dimension: "security"
        spec_reference: "FR-003"
        status: "pending"
        notes: ""

summary:
  total_items: 10
  passed: 5
  failed: 1
  pending: 4
  pass_rate: "50%"
```

### Checklist Status Values

| Status | Description |
|:-------|:------------|
| `pending` | Not yet evaluated |
| `pass` | Item verified |
| `fail` | Item failed verification |

---

## Validation

### Schema Validation

Validate artifacts against their schemas:

```bash
# Type inferred from filename
autospec artifact specs/001/spec.yaml
autospec artifact specs/001/plan.yaml
autospec artifact specs/001/tasks.yaml

# Explicit type (required for checklists)
autospec artifact checklist specs/001/checklists/security.yaml
```

**Validates:**
- Valid YAML syntax
- Required fields present
- Field types correct
- Enum values valid
- Cross-references valid (e.g., task dependencies)

### Syntax-Only Validation

Quick syntax check without schema validation:

```bash
autospec yaml check <file>
```

---

## Querying Artifacts

Use standard YAML tools:

```bash
# Get all user story IDs
yq '.user_stories[].id' spec.yaml

# Get pending tasks
yq '.phases[].tasks[] | select(.status == "Pending")' tasks.yaml

# Count tasks by phase
yq '.phases[] | .number as $n | .tasks | length | "\($n): \(.)"' tasks.yaml

# Get high-impact risks
yq '.risks[] | select(.impact == "high")' plan.yaml
```

---

## Troubleshooting

### Common Validation Errors

| Error | Cause | Fix |
|:------|:------|:----|
| Missing required field | Field not present | Add the field |
| Invalid enum value | Value not in allowed list | Use valid value |
| Invalid type | Wrong data type | Check expected type |
| Invalid reference | Task dependency doesn't exist | Fix task ID |

### YAML Syntax Issues

| Issue | Example | Fix |
|:------|:--------|:----|
| Tabs | `\t` characters | Use spaces |
| Unquoted special chars | `description: foo: bar` | Quote the string |
| Wrong indentation | Inconsistent spaces | Use 2-space indent |

Run `autospec yaml check <file>` for syntax errors with line numbers.

---

## See Also

- [CLI Commands](cli) - Commands for artifact validation
- [Configuration](configuration) - File locations and state directories
- [Quickstart Guide](/autospec/quickstart) - Generate your first artifacts
- [FAQ](/autospec/guides/faq) - Common questions about YAML artifacts
