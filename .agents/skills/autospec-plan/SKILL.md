---
name: autospec-plan
description: "Generate YAML implementation plan from feature specification."
---

# autospec-plan

This Agent Skill is generated from autospec.plan. When the user invokes "$autospec-plan" or "/autospec.plan",
load and follow these instructions directly. Treat the text after the skill or
command name as "$ARGUMENTS". Do not route back through "autospec plan"; this skill
is the prompt for the stage.

Project specs directory: ./specs

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Pre-computed Context

The following paths have been pre-computed and are available for use:

- **FEATURE_DIR**: `{{.FeatureDir}}`
- **FEATURE_SPEC**: `{{.FeatureSpec}}`
- **AUTOSPEC_VERSION**: `{{.AutospecVersion}}`
- **CREATED_DATE**: `{{.CreatedDate}}`

## Outline

1. **Load context**:
   - Read the spec file at `{{.FeatureSpec}}`
   - Read project constitution if exists (`.autospec/constitution.yaml` or `AGENTS.md`, falling back to agent-specific file like `CLAUDE.md`)
   - Extract: feature description, user stories, requirements, constraints

3. **Execute plan workflow**:

   **Phase 0: Outline & Research**

   a. Identify technical unknowns from the spec:
      - For each unclear technology choice → research task
      - For each dependency → best practices research
      - For each integration → patterns research

   b. Resolve unknowns through exploration:
      - Examine existing codebase patterns
      - Consider project constraints
      - Make informed technology decisions

   c. Document research findings for inclusion in plan

   **Phase 1: Design & Architecture**

   a. Define technical context based on spec and research:
      - Language/framework (detect from existing code or choose)
      - Primary dependencies
      - Storage requirements
      - Testing approach
      - Target platform

   b. Design project structure:
      - Documentation files to create
      - Source code organization
      - Test file locations

   c. Identify data model entities from spec requirements

   d. Design API contracts if applicable

4. **Generate plan.yaml**: Create the YAML plan file with this structure:

   ```yaml
   plan:
     branch: "<current git branch>"
     created: "<today's date YYYY-MM-DD>"
     spec_path: "<relative path to spec file>"

   summary: |
     <1-2 paragraph summary of the implementation approach.
     Explain key technical decisions and how they address the spec requirements.>

   technical_context:
     language: "<primary language>"
     framework: "<framework if applicable, or 'None'>"
     primary_dependencies:
       - name: "<dependency name>"
         version: "<version constraint>"
         purpose: "<why needed>"
     storage: "<storage technology or 'None'>"
     testing:
       framework: "<test framework>"
       approach: "<unit/integration/e2e strategy>"
     target_platform: "<platform(s)>"
     project_type: "<cli|web|mobile|library|service>"
     performance_goals: "<specific targets from spec>"
     constraints:
       - "<constraint from spec or technical>"
     scale_scope: "<expected scale/scope>"

   constitution_check:
     constitution_path: "<path to constitution file or 'Not found'>"
     gates:
       - name: "<principle name from constitution>"
         status: "PASS"  # or "FAIL" or "N/A"
         notes: "<how this plan addresses the principle>"

   research_findings:
     decisions:
       - topic: "<what was researched>"
         decision: "<what was chosen>"
         rationale: "<why chosen>"
         alternatives_considered:
           - "<alternative 1>"
           - "<alternative 2>"

   data_model:
     entities:
       - name: "<entity name>"
         description: "<what it represents>"
         fields:
           - name: "<field name>"
             type: "<data type>"
             description: "<purpose>"
             constraints: "<validation rules>"
         relationships:
           - target: "<related entity>"
             type: "<one-to-many|many-to-many|etc>"
             description: "<relationship meaning>"

   api_contracts:
     endpoints:
       - method: "<HTTP method>"
         path: "<endpoint path>"
         description: "<what it does>"
         request:
           content_type: "<content type>"
           body_schema: "<inline schema or reference>"
         response:
           success_code: 200
           body_schema: "<inline schema or reference>"
         errors:
           - code: 400
             description: "<when this occurs>"

   project_structure:
     documentation:
       - path: "<relative path>"
         description: "<purpose of this file>"
     source_code:
       - path: "<relative path or pattern>"
         description: "<what this contains>"
     tests:
       - path: "<relative path or pattern>"
         description: "<what tests live here>"

   implementation_phases:
     - phase: 1
       name: "<phase name>"
       goal: "<what this phase accomplishes>"
       deliverables:
         - "<deliverable 1>"
         - "<deliverable 2>"
     - phase: 2
       name: "<phase name>"
       goal: "<what this phase accomplishes>"
       dependencies:
         - "Phase 1"
       deliverables:
         - "<deliverable>"

   open_questions:
     - question: "<unresolved question>"
       context: "<why it matters>"
       proposed_resolution: "<suggested approach>"

   _meta:
     version: "1.0.0"
     generator: "autospec"
     generator_version: "{{.AutospecVersion}}"
     created: "{{.CreatedDate}}"
     artifact_type: "plan"
   ```

5. **Write the plan** to `{{.FeatureDir}}/plan.yaml`

6. **Validate the artifact**:
   ```bash
   autospec artifact {{.FeatureDir}}/plan.yaml
   ```
   - If validation fails: fix schema errors (missing required fields, invalid types) and retry
   - If validation passes: proceed to report

7. **Report**: Output:
   - Branch name
   - Full path to plan.yaml
   - Summary of technical context
   - Number of implementation phases
   - Any constitution gate failures (CRITICAL if any FAIL)
   - Readiness for `$autospec-tasks`

## Key Rules

- Output MUST be valid YAML (use `autospec artifact {{.FeatureDir}}/plan.yaml` to verify schema compliance)
- Technical context should reflect actual project setup (detect from existing code)
- Constitution gates are mandatory if constitution exists
- Research findings should document all significant technical decisions
- Data model should be derived from spec requirements
- Project structure should follow existing codebase conventions
- All YAML arrays use list syntax (not JSON inline)
- Multi-line strings use `|` or `>` block scalar style
