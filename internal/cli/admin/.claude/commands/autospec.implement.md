---
description: Execute the implementation plan by processing tasks defined in tasks.yaml.
version: "1.0.0"
handoffs:
  - label: Analyze For Consistency
    agent: autospec.analyze
    prompt: Run a project analysis for consistency
  - label: Generate Checklist
    agent: autospec.checklist
    prompt: Generate a quality checklist for this feature
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

1. **Setup**: Run the prerequisites command to get feature paths:

   ```bash
   autospec prereqs --json --require-tasks --include-tasks
   ```

   Parse the JSON output for:
   - `FEATURE_DIR`: The feature directory path
   - `FEATURE_SPEC`: Path to the spec file (spec.yaml)
   - `IMPL_PLAN`: Path to the plan file (plan.yaml)
   - `TASKS_FILE`: Path to the tasks file (tasks.yaml)
   - `AVAILABLE_DOCS`: List of optional documents found

   If the script fails, it will output an error message instructing the user to run `/autospec.tasks` first.

2. **Phase Context Metadata** (CRITICAL - Token Optimization):

   **IMMEDIATELY** after running prereqs, check if `--context-file` was used. If so, parse the `_context_meta` section FIRST before any other file reads.

   **`_context_meta` Fields**:
   - `phase_artifacts_bundled: true` - Indicates that spec.yaml, plan.yaml, and tasks.yaml (phase-filtered) are already bundled in this context file
   - `bundled_artifacts` - Lists the artifacts included: `["spec.yaml", "plan.yaml", "tasks.yaml (phase-filtered)"]`
   - `has_checklists` - Boolean indicating whether a `checklists/` directory exists for this feature
     - If `false`: **DO NOT** check for, scan, or read from the checklists directory - it doesn't exist, skip step 3 entirely
     - If `true`: Checklists directory exists, proceed to step 3
   - `skip_reads` - Explicit list of file paths that are already bundled and **MUST NOT** be read separately

   **CRITICAL INSTRUCTION**:
   ```
   DO NOT read files listed in skip_reads when _context_meta.phase_artifacts_bundled is true.
   DO NOT check for checklists directory when _context_meta.has_checklists is false.
   ```

   **Example `_context_meta` section**:
   ```yaml
   _context_meta:
     phase_artifacts_bundled: true
     bundled_artifacts:
       - spec.yaml
       - plan.yaml
       - tasks.yaml (phase-filtered)
     has_checklists: false
     skip_reads:
       - specs/my-feature/spec.yaml
       - specs/my-feature/plan.yaml
       - specs/my-feature/tasks.yaml
   ```

3. **Check checklists status** (SKIP if `_context_meta.has_checklists: false`):
   - Scan all `*.yaml` checklist files in the checklists/ directory
   - For each checklist YAML file, parse and count:
     - Total items: All items across all categories (`categories[].items[]`)
     - Passed items: Items where `status: "pass"`
     - Failed/Pending items: Items where `status: "fail"` or `status: "pending"`
   - Create a status table:

     ```text
     | Checklist     | Total | Passed | Not Passed | Status |
     |---------------|-------|--------|------------|--------|
     | ux.yaml       | 12    | 12     | 0          | PASS   |
     | api.yaml      | 8     | 5      | 3          | FAIL   |
     | security.yaml | 6     | 6      | 0          | PASS   |
     ```

   - Calculate overall status:
     - **PASS**: All checklists have 0 items with `status: "fail"` or `status: "pending"`
     - **FAIL**: One or more checklists have items not in `status: "pass"`

   - **If any checklist is incomplete**:
     - Display the table with incomplete item counts
     - **STOP** and ask: "Some checklists are incomplete. Do you want to proceed with implementation anyway? (yes/no)"
     - Wait for user response before continuing
     - If user says "no" or "wait" or "stop", halt execution
     - If user says "yes" or "proceed" or "continue", proceed to step 4

   - **If all checklists are complete**:
     - Display the table showing all checklists passed
     - Automatically proceed to step 4

4. **Load and analyze the implementation context** (if NOT using `--context-file`):

   **Note**: If you are using `--context-file`, the spec, plan, and tasks are already loaded from the context file. Skip reading these files individually and use the bundled data from the `spec:`, `plan:`, and `tasks:` sections of the context file instead.

   - **REQUIRED**: Read tasks.yaml for the complete task list and execution plan
   - **REQUIRED**: Read plan.yaml for:
     - `technical_context`: tech stack, dependencies, constraints
     - `data_model`: entities and relationships
     - `api_contracts`: API specifications
     - `research_findings`: technical decisions and rationale
     - `project_structure`: file organization
   - **REQUIRED**: Read spec.yaml for:
     - `user_stories`: acceptance scenarios
     - `requirements`: functional and non-functional
     - `success_criteria`: measurable outcomes

5. **Project Setup Verification**:
   - **REQUIRED**: Create/verify ignore files based on actual project setup:

   **Detection & Creation Logic**:
   - Check if the following command succeeds to determine if the repository is a git repo (create/verify .gitignore if so):

     ```sh
     git rev-parse --git-dir 2>/dev/null
     ```

   - Check if Dockerfile* exists or Docker in plan.yaml technical_context → create/verify .dockerignore
   - Check if .eslintrc* exists → create/verify .eslintignore
   - Check if eslint.config.* exists → ensure the config's `ignores` entries cover required patterns
   - Check if .prettierrc* exists → create/verify .prettierignore
   - Check if .npmrc or package.json exists → create/verify .npmignore (if publishing)
   - Check if terraform files (*.tf) exist → create/verify .terraformignore
   - Check if .helmignore needed (helm charts present) → create/verify .helmignore

   **If ignore file already exists**: Verify it contains essential patterns, append missing critical patterns only
   **If ignore file missing**: Create with full pattern set for detected technology

   **Common Patterns by Technology** (from plan.yaml `technical_context`):
   - **Node.js/JavaScript/TypeScript**: `node_modules/`, `dist/`, `build/`, `*.log`, `.env*`
   - **Python**: `__pycache__/`, `*.pyc`, `.venv/`, `venv/`, `dist/`, `*.egg-info/`
   - **Java**: `target/`, `*.class`, `*.jar`, `.gradle/`, `build/`
   - **C#/.NET**: `bin/`, `obj/`, `*.user`, `*.suo`, `packages/`
   - **Go**: `*.exe`, `*.test`, `vendor/`, `*.out`
   - **Ruby**: `.bundle/`, `log/`, `tmp/`, `*.gem`, `vendor/bundle/`
   - **PHP**: `vendor/`, `*.log`, `*.cache`, `*.env`
   - **Rust**: `target/`, `debug/`, `release/`, `*.rs.bk`, `*.rlib`, `*.prof*`, `.idea/`, `*.log`, `.env*`
   - **Kotlin**: `build/`, `out/`, `.gradle/`, `.idea/`, `*.class`, `*.jar`, `*.iml`, `*.log`, `.env*`
   - **C++**: `build/`, `bin/`, `obj/`, `out/`, `*.o`, `*.so`, `*.a`, `*.exe`, `*.dll`, `.idea/`, `*.log`, `.env*`
   - **C**: `build/`, `bin/`, `obj/`, `out/`, `*.o`, `*.a`, `*.so`, `*.exe`, `Makefile`, `config.log`, `.idea/`, `*.log`, `.env*`
   - **Swift**: `.build/`, `DerivedData/`, `*.swiftpm/`, `Packages/`
   - **R**: `.Rproj.user/`, `.Rhistory`, `.RData`, `.Ruserdata`, `*.Rproj`, `packrat/`, `renv/`
   - **Universal**: `.DS_Store`, `Thumbs.db`, `*.tmp`, `*.swp`, `.vscode/`, `.idea/`

   **Tool-Specific Patterns**:
   - **Docker**: `node_modules/`, `.git/`, `Dockerfile*`, `.dockerignore`, `*.log*`, `.env*`, `coverage/`
   - **ESLint**: `node_modules/`, `dist/`, `build/`, `coverage/`, `*.min.js`
   - **Prettier**: `node_modules/`, `dist/`, `build/`, `coverage/`, `package-lock.json`, `yarn.lock`, `pnpm-lock.yaml`
   - **Terraform**: `.terraform/`, `*.tfstate*`, `*.tfvars`, `.terraform.lock.hcl`
   - **Kubernetes/k8s**: `*.secret.yaml`, `secrets/`, `.kube/`, `kubeconfig*`, `*.key`, `*.crt`

6. **Parse tasks.yaml structure** and extract:
   - **Phases**: Setup, Foundational, User Story phases, Polish
   - **Task dependencies**: Sequential vs parallel execution from `parallel` field
   - **Task details**: id, title, status, type, file_path, dependencies, acceptance_criteria
   - **Execution flow**: Phase order and task dependency requirements
   - **User story mapping**: Which tasks belong to which user stories

7. **Execute implementation following the task plan**:
   - **Phase-by-phase execution**: Complete each phase before moving to the next
   - **Respect dependencies**: Run sequential tasks in order, parallel tasks can run together
   - **Follow TDD approach**: Execute test tasks before their corresponding implementation tasks (if tests exist)
   - **File-based coordination**: Tasks affecting the same files must run sequentially
   - **Validation checkpoints**: Verify each phase completion before proceeding

8. **Implementation execution rules**:
   - **Setup first**: Initialize project structure, dependencies, configuration
   - **Foundational next**: Complete blocking prerequisites before user stories
   - **User stories in order**: Complete each story phase before the next
   - **Tests before code**: If test tasks exist, write tests before implementation
   - **Polish last**: Cross-cutting concerns and refactoring at the end

9. **Progress tracking and task status updates**:

   **CRITICAL**: You MUST update task status in tasks.yaml as you work. This is non-negotiable.

   Use the `autospec update-task` command to update task status:
   ```bash
   autospec update-task <task_id> <status>
   ```

   **When starting a task**:
   ```bash
   autospec update-task T001 InProgress
   ```

   **When completing a task**:
   ```bash
   autospec update-task T001 Completed
   ```

   **If a task is blocked**:
   ```bash
   autospec update-task T001 Blocked
   ```

   Valid status values: `Pending`, `InProgress`, `Completed`, `Blocked`

   **Blocking tasks with reasons** (preferred method for documenting blockers):
   ```bash
   # Block a task and document why it's blocked
   autospec task block T001 --reason "Waiting for API access from third-party team"

   # Update the reason for an already blocked task
   autospec task block T001 --reason "Updated: API approved, waiting for credentials"
   ```

   **Unblocking tasks**:
   ```bash
   # Unblock a task (defaults to Pending status)
   autospec task unblock T001

   # Unblock and immediately set to InProgress
   autospec task unblock T001 --status InProgress
   ```

   **Listing tasks by status**:
   ```bash
   # List all tasks
   autospec task list

   # List only blocked tasks (shows reasons)
   autospec task list --blocked

   # List pending tasks
   autospec task list --pending

   # List in-progress tasks
   autospec task list --in-progress

   # List completed tasks
   autospec task list --completed

   # Combine filters
   autospec task list --blocked --pending
   ```

   **Implementation workflow for each task**:
   1. Mark task as InProgress: `autospec update-task T00X InProgress`
   2. Implement the task
   3. Verify implementation meets acceptance criteria
   4. Mark task as Completed: `autospec update-task T00X Completed`
   5. Move to next task

   **Handling blocked tasks**:
   1. If blocked by external dependency: `autospec task block T00X --reason "Reason"`
   2. Document the blocker clearly so others understand what needs resolution
   3. When blocker is resolved: `autospec task unblock T00X [--status InProgress]`

   - Report progress after each completed task
   - Halt execution if any non-parallel task fails
   - For parallel tasks, continue with successful tasks, report failed ones
   - Provide clear error messages with context for debugging
   - Suggest next steps if implementation cannot proceed

10. **Validate tasks.yaml after updates**:
   ```bash
   autospec artifact FEATURE_DIR/tasks.yaml
   ```
   - Ensure artifact schema remains valid after status updates
   - Fix any schema errors (missing fields, invalid types, invalid dependencies) before proceeding

11. **Completion validation**:
    - Verify all required tasks have `status: "Completed"`
    - Check that implemented features match the original specification
    - Validate that tests pass (if tests were generated)
    - Confirm the implementation follows the technical plan
    - Report final status with summary of completed work

12. **Report**: Output:
    - Feature directory path
    - Total tasks completed vs total tasks
    - Tasks completed per phase
    - Tasks completed per user story
    - Any failed or skipped tasks with reasons
    - Final validation status
    - Suggested next steps (if any tasks remain)

Context for implementation: $ARGUMENTS

Note: This command assumes tasks.yaml exists with a complete task breakdown. If tasks are incomplete or missing, suggest running `/autospec.tasks` first to generate the task list.
