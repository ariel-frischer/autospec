---
description: Generate YAML task breakdown from implementation plan.
version: "1.0.0"
handoffs:
  - label: Analyze For Consistency
    agent: autospec.analyze
    prompt: Run a project analysis for consistency
  - label: Implement Project
    agent: autospec.implement
    prompt: Start the implementation in phases
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

1. **Setup**: Run the prerequisites script to get feature paths:

   ```bash
   scripts/autospec/check-prerequisites.sh --json --require-plan
   ```

   Parse the JSON output for:
   - `FEATURE_DIR`: The feature directory path
   - `FEATURE_SPEC`: Path to the spec file
   - `IMPL_PLAN`: Path to the plan file
   - `AVAILABLE_DOCS`: List of optional documents found

   If the script fails, it will output an error message instructing the user to run `/autospec.plan` first.

2. **Load design documents**: Read from FEATURE_DIR:
   - **Required**: `IMPL_PLAN` - tech stack, libraries, structure
   - **Required**: `FEATURE_SPEC` - user stories with priorities
   - **Optional**: data-model.md (entities), contracts/ (API endpoints), research.md (decisions)
   - Note: Not all projects have all documents. Generate tasks based on what's available.

3. **Execute task generation workflow**:
   - Extract tech stack, libraries, project structure from plan
   - Extract user stories with their priorities (P1, P2, P3, etc.) from spec
   - If data model exists: Extract entities and map to user stories
   - If contracts exist: Map endpoints to user stories
   - If research exists: Extract decisions for setup tasks
   - Generate tasks organized by user story (see Task Generation Rules below)
   - Generate dependency graph showing user story completion order
   - Create parallel execution opportunities per phase
   - Validate task completeness (each user story has all needed tasks)

4. **Generate tasks.yaml**: Create the YAML task file with this structure:

   ```yaml
   _meta:
     version: "1.0.0"
     generator: "autospec"
     generator_version: "<run autospec version to get this>"
     created: "<ISO 8601 timestamp>"
     artifact_type: "tasks"

   tasks:
     branch: "<current git branch>"
     created: "<today's date YYYY-MM-DD>"
     spec_path: "<relative path to spec file>"
     plan_path: "<relative path to plan file>"

   summary:
     total_tasks: <number>
     total_phases: <number>
     parallel_opportunities: <number of tasks marked parallelizable>
     estimated_complexity: "<low|medium|high>"

   phases:
     - number: 1
       title: "Setup"
       purpose: "Project initialization and new package structure"
       tasks:
         - id: "T001"
           title: "<task title with file path>"
           status: "Pending"  # Pending | InProgress | Completed
           type: "setup"  # setup | implementation | test | documentation | refactor
           parallel: false
           story_id: null  # null for setup/foundational tasks
           file_path: "<exact file path to create/modify>"
           dependencies: []
           acceptance_criteria:
             - "<criterion 1>"

     - number: 2
       title: "Foundational"
       purpose: "Core infrastructure that MUST be complete before user stories"
       tasks:
         - id: "T002"
           title: "<task title>"
           status: "Pending"
           type: "implementation"
           parallel: true  # Can run in parallel with T003
           story_id: null
           file_path: "<file path>"
           dependencies: ["T001"]
           acceptance_criteria:
             - "<criterion>"

     - number: 3
       title: "User Story 1 - <US-001 title from spec>"
       purpose: "<goal from user story>"
       story_reference: "US-001"
       independent_test: "<how to test this story independently>"
       tasks:
         - id: "T010"
           title: "<task with file path>"
           status: "Pending"
           type: "test"  # Tests first per constitution
           parallel: true
           story_id: "US-001"
           file_path: "<test file path>"
           dependencies: ["T002"]
           acceptance_criteria:
             - "<criterion>"

         - id: "T011"
           title: "<implementation task>"
           status: "Pending"
           type: "implementation"
           parallel: false
           story_id: "US-001"
           file_path: "<source file path>"
           dependencies: ["T010"]  # Depends on test being written
           acceptance_criteria:
             - "<criterion>"

     # Continue with more phases for each user story...

     - number: <final>
       title: "Polish & Cross-Cutting Concerns"
       purpose: "Improvements that affect multiple user stories"
       tasks:
         - id: "T099"
           title: "<polish task>"
           status: "Pending"
           type: "refactor"
           parallel: true
           story_id: null
           file_path: "<file path>"
           dependencies: ["<all prior phases>"]
           acceptance_criteria:
             - "<criterion>"

   dependencies:
     user_story_order:
       - story_id: "US-001"
         depends_on: []
         blocks: ["US-002"]
       - story_id: "US-002"
         depends_on: ["US-001"]
         blocks: []

     phase_order:
       - phase: 1
         blocks: [2]
       - phase: 2
         blocks: [3, 4, 5]

   parallel_execution:
     - phase: 2
       parallel_groups:
         - tasks: ["T002", "T003"]
           rationale: "Different packages, no dependencies"
     - phase: 3
       parallel_groups:
         - tasks: ["T010", "T011"]
           rationale: "Test and implementation can be developed together"

   implementation_strategy:
     mvp_scope:
       phases: [1, 2, 3]
       description: "Setup + Foundational + User Story 1"
       validation: "<how to validate MVP>"
     incremental_delivery:
       - milestone: "Foundation Ready"
         phases: [1, 2]
         deliverable: "<what's usable at this point>"
       - milestone: "MVP Complete"
         phases: [1, 2, 3]
         deliverable: "<what's usable>"
   ```

5. **Write the tasks** to `FEATURE_DIR/tasks.yaml`

6. **Validate the YAML**:
   ```bash
   autospec yaml check FEATURE_DIR/tasks.yaml
   ```
   - If validation fails: fix YAML syntax errors and retry
   - If validation passes: proceed to report

7. **Report**: Output:
   - Full path to tasks.yaml
   - Total task count
   - Task count per phase
   - Task count per user story
   - Parallel opportunities identified
   - Suggested MVP scope
   - Format validation confirmation

Context for task generation: $ARGUMENTS

The tasks.yaml should be immediately executable - each task must be specific enough that an LLM can complete it without additional context.

## Task Generation Rules

**CRITICAL**: Tasks MUST be organized by user story to enable independent implementation and testing.

**Tests are OPTIONAL**: Only generate test tasks if explicitly requested in the feature specification or if user requests TDD approach.

### Task ID Format

Every task MUST have:
1. **Task ID**: Sequential format T001, T002, T003... in execution order
2. **Parallel flag**: `parallel: true` if task can run alongside others (different files, no dependencies)
3. **Story ID**: Link to user story (US-001, US-002) for story-phase tasks, null for setup/foundational
4. **File path**: Exact path where work happens
5. **Dependencies**: List of task IDs that must complete first

### Task Organization

1. **From User Stories (spec)** - PRIMARY ORGANIZATION:
   - Each user story (P1, P2, P3...) gets its own phase
   - Map all related components to their story:
     - Models needed for that story
     - Services needed for that story
     - Endpoints/UI needed for that story
     - Tests specific to that story (if requested)
   - Mark story dependencies (most stories should be independent)

2. **From Plan Structure**:
   - Map each component from project_structure to appropriate phase
   - If tests requested: Each component → test task before implementation

3. **From Data Model**:
   - Map each entity to the user story(ies) that need it
   - If entity serves multiple stories: Put in earliest story or Foundational phase
   - Relationships → service layer tasks in appropriate story phase

4. **From Setup/Infrastructure**:
   - Shared infrastructure → Setup phase (Phase 1)
   - Foundational/blocking tasks → Foundational phase (Phase 2)
   - Story-specific setup → within that story's phase

### Phase Structure

- **Phase 1**: Setup (project initialization)
- **Phase 2**: Foundational (blocking prerequisites - MUST complete before user stories)
- **Phase 3+**: User Stories in priority order (P1, P2, P3...)
  - Within each story: Tests (if requested) → Models → Services → Endpoints → Integration
  - Each phase should be a complete, independently testable increment
- **Final Phase**: Polish & Cross-Cutting Concerns

### Task Types

- `setup`: Project/directory initialization
- `test`: Test file creation (should come before implementation if TDD)
- `implementation`: Core feature code
- `documentation`: README, docs, comments
- `refactor`: Code improvement without behavior change

### Task Status

- `Pending`: Not started
- `InProgress`: Currently being worked on
- `Completed`: Finished and verified
