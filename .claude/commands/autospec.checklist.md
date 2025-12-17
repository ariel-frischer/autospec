---
description: Generate YAML checklist for feature quality validation.
version: "1.0.0"
---

## Checklist Purpose: "Unit Tests for English"

**CRITICAL CONCEPT**: Checklists are **UNIT TESTS FOR REQUIREMENTS WRITING** - they validate the quality, clarity, and completeness of requirements in a given domain.

**NOT for verification/testing**:

- NOT "Verify the button clicks correctly"
- NOT "Test error handling works"
- NOT "Confirm the API returns 200"
- NOT checking if code/implementation matches the spec

**FOR requirements quality validation**:

- "Are visual hierarchy requirements defined for all card types?" (completeness)
- "Is 'prominent display' quantified with specific sizing/positioning?" (clarity)
- "Are hover state requirements consistent across all interactive elements?" (consistency)
- "Are accessibility requirements defined for keyboard navigation?" (coverage)
- "Does the spec define what happens when logo image fails to load?" (edge cases)

**Metaphor**: If your spec is code written in English, the checklist is its unit test suite. You're testing whether the requirements are well-written, complete, unambiguous, and ready for implementation - NOT whether the implementation works.

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Execution Steps

1. **Setup**: Run the prerequisites command to get feature paths:

   ```bash
   autospec prereqs --json --require-spec
   ```

   Parse the JSON output for:
   - `FEATURE_DIR`: The feature directory path
   - `FEATURE_SPEC`: Path to the spec file
   - `AVAILABLE_DOCS`: List of optional documents found
   - `AUTOSPEC_VERSION`: The autospec version (for _meta section)
   - `CREATED_DATE`: ISO 8601 timestamp (for _meta section)

   If the script fails, it will output an error message instructing the user to run `/autospec.specify` first.

2. **Clarify intent (dynamic)**: Derive up to THREE initial contextual clarifying questions. They MUST:
   - Be generated from the user's phrasing + extracted signals from spec/plan/tasks
   - Only ask about information that materially changes checklist content
   - Be skipped individually if already unambiguous in `$ARGUMENTS`
   - Prefer precision over breadth

   Generation algorithm:
   1. Extract signals: feature domain keywords (e.g., auth, latency, UX, API), risk indicators ("critical", "must", "compliance"), stakeholder hints ("QA", "review", "security team")
   2. Cluster signals into candidate focus areas (max 4) ranked by relevance
   3. Identify probable audience & timing (author, reviewer, QA, release) if not explicit
   4. Detect missing dimensions: scope breadth, depth/rigor, risk emphasis, exclusion boundaries
   5. Formulate questions from these archetypes:
      - Scope refinement (e.g., "Should this include integration touchpoints?")
      - Risk prioritization (e.g., "Which risk areas need mandatory gating checks?")
      - Depth calibration (e.g., "Lightweight sanity list or formal release gate?")
      - Audience framing (e.g., "Author-only or peer PR review?")

   Defaults when interaction impossible:
   - Depth: Standard
   - Audience: Reviewer (PR) if code-related; Author otherwise
   - Focus: Top 2 relevance clusters

3. **Understand user request**: Combine `$ARGUMENTS` + clarifying answers:
   - Derive checklist theme (e.g., security, review, deploy, ux)
   - Consolidate explicit must-have items mentioned by user
   - Map focus selections to category scaffolding

4. **Load feature context**: Read from FEATURE_DIR:
   - spec.yaml: Feature requirements and scope
   - plan.yaml if exists: Technical details, dependencies, data model, API contracts
   - tasks.yaml if exists: Implementation tasks

5. **Generate checklist.yaml** - Create "Unit Tests for Requirements":

   ```yaml
   checklist:
     feature: "<feature name from spec>"
     branch: "<current git branch>"
     spec_path: "<relative path to spec file>"
     domain: "<checklist domain: ux, api, security, performance, etc.>"
     audience: "<author|reviewer|qa|release>"
     depth: "<lightweight|standard|comprehensive>"

   categories:
     - name: "Requirement Completeness"
       description: "Are all necessary requirements documented?"
       items:
         - id: "CHK001"
           description: "Are all functional requirements specified for the primary user flow?"
           quality_dimension: "completeness"
           spec_reference: "FR-001"  # or null if checking for gap
           status: "pending"  # pending | pass | fail
           notes: ""

         - id: "CHK002"
           description: "Are error handling requirements defined for all API failure modes?"
           quality_dimension: "completeness"
           spec_reference: null
           status: "pending"
           notes: ""

     - name: "Requirement Clarity"
       description: "Are requirements specific and unambiguous?"
       items:
         - id: "CHK003"
           description: "Is 'fast loading' quantified with specific timing thresholds?"
           quality_dimension: "clarity"
           spec_reference: "NFR-001"
           status: "pending"
           notes: ""

     - name: "Requirement Consistency"
       description: "Do requirements align without conflicts?"
       items:
         - id: "CHK004"
           description: "Are navigation requirements consistent across all pages?"
           quality_dimension: "consistency"
           spec_reference: "FR-010"
           status: "pending"
           notes: ""

     - name: "Acceptance Criteria Quality"
       description: "Are success criteria measurable?"
       items:
         - id: "CHK005"
           description: "Can all success criteria be objectively verified?"
           quality_dimension: "measurability"
           spec_reference: "SC-001"
           status: "pending"
           notes: ""

     - name: "Scenario Coverage"
       description: "Are all flows and cases addressed?"
       items:
         - id: "CHK006"
           description: "Are requirements defined for zero-state scenarios?"
           quality_dimension: "coverage"
           spec_reference: null
           status: "pending"
           notes: ""

     - name: "Edge Case Coverage"
       description: "Are boundary conditions defined?"
       items:
         - id: "CHK007"
           description: "Is fallback behavior specified when external services fail?"
           quality_dimension: "edge_cases"
           spec_reference: null
           status: "pending"
           notes: ""

   summary:
     total_items: <number>
     passed: <number>
     failed: <number>
     pending: <number>
     pass_rate: "<percentage>"

   _meta:
     version: "1.0.0"
     generator: "autospec"
     generator_version: "<AUTOSPEC_VERSION from step 1>"
     created: "<CREATED_DATE from step 1>"
     artifact_type: "checklist"
   ```

6. **Write the checklist** to `FEATURE_DIR/checklists/<domain>.yaml`
   - Create `FEATURE_DIR/checklists/` directory if it doesn't exist
   - Use domain-based filename: `ux.yaml`, `api.yaml`, `security.yaml`, etc.

7. **Validate the artifact**:
   ```bash
   autospec artifact checklist FEATURE_DIR/checklists/<domain>.yaml
   ```
   - If validation fails: fix schema errors (missing required fields, invalid types/enums) and retry
   - If validation passes: proceed to report

8. **Report**: Output:
   - Full path to checklist.yaml
   - Item count by category
   - Gap markers count (requirements needing attention)
   - Checklist domain and audience

## HOW TO WRITE CHECKLIST ITEMS - "Unit Tests for English"

**WRONG** (Testing implementation):
- "Verify landing page displays 3 episode cards"
- "Test hover states work on desktop"
- "Confirm logo click navigates home"

**CORRECT** (Testing requirements quality):
- "Are the exact number and layout of featured episodes specified?" [Completeness]
- "Is 'prominent display' quantified with specific sizing/positioning?" [Clarity]
- "Are hover state requirements consistent across all interactive elements?" [Consistency]
- "Are keyboard navigation requirements defined for all interactive UI?" [Coverage]
- "Is the fallback behavior specified when logo image fails to load?" [Edge Cases]

### Quality Dimensions

- **completeness**: Are all necessary requirements present?
- **clarity**: Are requirements unambiguous and specific?
- **consistency**: Do requirements align with each other?
- **measurability**: Can requirements be objectively verified?
- **coverage**: Are all scenarios/edge cases addressed?
- **edge_cases**: Are boundary conditions defined?

### ABSOLUTELY PROHIBITED

- Any item starting with "Verify", "Test", "Confirm", "Check" + implementation behavior
- References to code execution, user actions, system behavior
- "Displays correctly", "works properly", "functions as expected"
- "Click", "navigate", "render", "load", "execute"
- Test cases, test plans, QA procedures
- Implementation details (frameworks, APIs, algorithms)

### REQUIRED PATTERNS

- "Are [requirement type] defined/specified/documented for [scenario]?"
- "Is [vague term] quantified/clarified with specific criteria?"
- "Are requirements consistent between [section A] and [section B]?"
- "Can [requirement] be objectively measured/verified?"
- "Are [edge cases/scenarios] addressed in requirements?"
- "Does the spec define [missing aspect]?"
