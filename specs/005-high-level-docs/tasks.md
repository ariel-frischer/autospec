# Tasks: High-Level Documentation

**Input**: Design documents from `/specs/005-high-level-docs/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are included to validate documentation structure and completeness (constitution requirement).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each documentation file.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Documentation**: `docs/` at repository root
- **Tests**: Go tests in `internal/validation/docs_test.go` or similar
- All documentation files follow markdown format

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create documentation directory structure and validation framework

- [X] T001 Create docs/ directory at repository root
- [X] T002 [P] Review existing codebase structure (internal/cli/, internal/workflow/, internal/config/) for reference content
- [X] T003 [P] Review CLAUDE.md to identify complementary vs. duplicate content

**Checkpoint**: Directory structure ready for documentation files

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Create test framework that validates all documentation files

**⚠️ CRITICAL**: Tests must be written FIRST before any documentation files, per constitution requirement

- [X] T004 Create test file internal/validation/docs_test.go for documentation validation
- [X] T005 [P] Implement TestDocumentationFilesExist test (checks all 5 files exist in docs/)
- [X] T006 [P] Implement TestDocumentationLineCount test (validates each file ≤ 500 lines)
- [X] T007 [P] Implement TestDocumentationHeaders test (validates exactly one H1, logical nesting)
- [X] T008 [P] Implement TestInternalLinks test (validates cross-references to other docs)
- [X] T009 [P] Implement TestCodeReferences test (validates file:line format for code refs)
- [X] T010 [P] Implement TestMermaidDiagrams test (validates Mermaid syntax)
- [X] T011 [P] Implement TestCommandCompleteness test (validates all CLI commands documented)
- [X] T012 [P] Implement TestConfigCompleteness test (validates all config options documented)
- [X] T013 Run all tests and verify they FAIL (no documentation files exist yet)

**Checkpoint**: Test framework complete and failing - ready for documentation implementation

---

## Phase 3: User Story 1 - Quick Start Guide (Priority: P1) 🎯 MVP

**Goal**: Enable new users to understand the project and complete their first workflow within 10 minutes

**Independent Test**: Give documentation to someone unfamiliar with the project and measure if they can install and run first workflow within 10 minutes

### Implementation for User Story 1

- [X] T014 [P] [US1] Create docs/overview.md with project introduction (What is it, Key Features, Target Audience, Use Cases)
- [X] T015 [US1] Create docs/quickstart.md with Prerequisites section (Claude CLI, git, command line familiarity)
- [X] T016 [US1] Add Installation section to docs/quickstart.md (build from source, download binary, verification)
- [X] T017 [US1] Add Your First Workflow section to docs/quickstart.md (init, doctor, specify, plan, tasks steps with time estimates)
- [X] T018 [US1] Add Common Commands table to docs/quickstart.md (full, workflow, implement, status, help)
- [X] T019 [US1] Add Understanding the Workflow section with Mermaid diagram to docs/quickstart.md
- [X] T020 [US1] Add Configuration Basics section to docs/quickstart.md (JSON config with inline comments)
- [X] T021 [US1] Add Troubleshooting section to docs/quickstart.md (common first-time issues with solutions)
- [X] T022 [US1] Add Next Steps section to docs/quickstart.md (links to other docs)
- [X] T023 [US1] Validate overview.md is under 500 lines and meets content requirements (run tests T005-T007)
- [X] T024 [US1] Validate quickstart.md is under 500 lines and meets content requirements (run tests T005-T010)
- [X] T025 [US1] Manually test quickstart.md by following all commands in sequence
- [X] T026 [US1] Add cross-references from overview.md to quickstart.md and architecture.md

**Checkpoint**: At this point, User Story 1 (quick start documentation) should be complete, validated, and independently testable. New users can understand and use the tool.

---

## Phase 4: User Story 2 - Architecture Overview (Priority: P2)

**Goal**: Enable contributors and advanced users to understand system architecture and design decisions for effective contribution or troubleshooting

**Independent Test**: Ask a developer to locate and modify a specific component (e.g., "add a new validation function") using only the architecture docs

### Implementation for User Story 2

- [X] T027 [US2] Create docs/architecture.md with Component Overview section and high-level architecture Mermaid diagram
- [X] T028 [P] [US2] Add CLI Layer description to architecture.md (internal/cli/ package, command structure)
- [X] T029 [P] [US2] Add Workflow Orchestration description to architecture.md (internal/workflow/ package, phase execution)
- [X] T030 [P] [US2] Add Configuration description to architecture.md (internal/config/ package, layered config loading)
- [X] T031 [P] [US2] Add Validation description to architecture.md (internal/validation/ package, validation functions)
- [X] T032 [P] [US2] Add Retry Management description to architecture.md (internal/retry/ package, persistent state)
- [X] T033 [P] [US2] Add other component descriptions to architecture.md (spec detection, git integration, health checks, progress indicators)
- [X] T034 [US2] Add Execution Flow section with sequence diagram to architecture.md (User → CLI → Executor → Claude flow)
- [X] T035 [US2] Add phase execution with retry flowchart to architecture.md
- [X] T036 [US2] Add Key Patterns section to architecture.md (retry pattern, config layering, spec detection, exit codes)
- [X] T037 [US2] Add Package Structure section to architecture.md (purpose of each package, key files, code references)
- [X] T038 [US2] Add code references (file:line format) to all major components in architecture.md
- [X] T039 [US2] Validate architecture.md is under 500 lines (run test T005-T006)
- [X] T040 [US2] Validate all code references are correct format and valid (run test T009)
- [X] T041 [US2] Validate all Mermaid diagrams render correctly (run test T010)
- [X] T042 [US2] Add cross-references from quickstart.md and reference.md to architecture.md

**Checkpoint**: At this point, User Stories 1 AND 2 should both be complete. Contributors can understand the architecture and locate components.

---

## Phase 5: User Story 3 - Workflow Reference (Priority: P3)

**Goal**: Provide quick reference documentation for command options, configuration settings, and common patterns for efficient workflow execution

**Independent Test**: Ask a user to execute a specific advanced workflow (e.g., "run implement with custom timeout and retry settings") using only the reference docs

### Implementation for User Story 3

- [X] T043 [P] [US3] Create docs/reference.md with CLI Commands section
- [X] T044 [P] [US3] Document autospec full command in reference.md (syntax, description, flags, examples, exit codes)
- [X] T045 [P] [US3] Document autospec workflow command in reference.md (syntax, description, flags, examples, exit codes)
- [X] T046 [P] [US3] Document autospec specify command in reference.md (syntax, description, flags, examples, exit codes)
- [X] T047 [P] [US3] Document autospec plan command in reference.md (syntax, description, flags, examples, exit codes)
- [X] T048 [P] [US3] Document autospec tasks command in reference.md (syntax, description, flags, examples, exit codes)
- [X] T049 [P] [US3] Document autospec implement command in reference.md (syntax, description, flags, examples, exit codes)
- [X] T050 [P] [US3] Document autospec doctor command in reference.md (syntax, description, flags, examples, exit codes)
- [X] T051 [P] [US3] Document autospec status command in reference.md (syntax, description, flags, examples, exit codes)
- [X] T052 [P] [US3] Document autospec config command in reference.md (syntax, description, flags, examples, exit codes)
- [X] T053 [P] [US3] Document autospec init command in reference.md (syntax, description, flags, examples, exit codes)
- [X] T054 [P] [US3] Document autospec version command in reference.md (syntax, description, flags, examples, exit codes)
- [X] T055 [US3] Add Configuration Options section to reference.md with table format
- [X] T056 [P] [US3] Document claude_cmd option in reference.md (name, type, default, description, example)
- [X] T057 [P] [US3] Document specify_cmd option in reference.md (name, type, default, description, example)
- [X] T058 [P] [US3] Document max_retries option in reference.md (name, type, default, valid range, description, example)
- [X] T059 [P] [US3] Document specs_dir option in reference.md (name, type, default, description, example)
- [X] T060 [P] [US3] Document state_dir option in reference.md (name, type, default, description, example)
- [X] T061 [P] [US3] Document timeout option in reference.md (name, type, default, valid range 0 or 1-604800, description, example)
- [X] T062 [P] [US3] Document skip_preflight option in reference.md (name, type, default, description, example)
- [X] T063 [P] [US3] Document custom_claude_cmd option in reference.md (name, type, default, description, example)
- [X] T064 [US3] Add Exit Codes section to reference.md with table (code 0-5 with meanings and actions)
- [X] T065 [US3] Add File Locations section to reference.md (config files, state files, specs dirs with purposes)
- [X] T066 [US3] Add code references to command implementations (internal/cli/*.go) in reference.md
- [X] T067 [US3] Validate reference.md is under 500 lines (run test T005-T006)
- [X] T068 [US3] Validate all CLI commands are documented (run test T011)
- [X] T069 [US3] Validate all config options are documented (run test T012)
- [X] T070 [US3] Add cross-references from quickstart.md and troubleshooting.md to reference.md
- [X] T071 [P] [US3] Create docs/troubleshooting.md with Common Errors section
- [X] T072 [P] [US3] Document "claude: command not found" error in troubleshooting.md (error, cause, solution, related resources)
- [X] T073 [P] [US3] Document "autospec: command not found" error in troubleshooting.md (error, cause, solution, related resources)
- [X] T074 [P] [US3] Document "Validation failed: spec.md not found" error in troubleshooting.md (error, cause, solution, related resources)
- [X] T075 [P] [US3] Document "Retry limit exhausted" error in troubleshooting.md (error, cause, solution, related resources)
- [X] T076 [P] [US3] Document "Spec not detected" error in troubleshooting.md (error, cause, solution, related resources)
- [X] T077 [P] [US3] Document "Command timed out" error in troubleshooting.md (error, cause, solution, related resources)
- [X] T078 [US3] Add Exit Code Reference section to troubleshooting.md (link to reference.md, interpretation guidance)
- [X] T079 [US3] Add Debugging Tips section to troubleshooting.md (--debug flag, retry state, config show, doctor, auth)
- [X] T080 [P] [US3] Add FAQ entry "How do I reset retry state?" to troubleshooting.md
- [X] T081 [P] [US3] Add FAQ entry "Can I use a different Claude command?" to troubleshooting.md
- [X] T082 [P] [US3] Add FAQ entry "How do I skip preflight checks?" to troubleshooting.md
- [X] T083 [P] [US3] Add FAQ entry "What's the difference between 'workflow' and 'full' commands?" to troubleshooting.md
- [X] T084 [P] [US3] Add FAQ entry "How do I increase command timeout?" to troubleshooting.md
- [X] T085 [US3] Validate troubleshooting.md is under 500 lines (run test T005-T007)
- [X] T086 [US3] Add cross-references from quickstart.md and reference.md to troubleshooting.md

**Checkpoint**: All user stories (1, 2, 3) should now be independently functional and complete

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final validation, cross-reference verification, and quality assurance

- [X] T087 [P] Run all documentation tests and verify they PASS (TestDocumentationFilesExist through TestConfigCompleteness)
- [X] T088 [P] Validate all cross-references between documentation files are bidirectional and correct
- [X] T089 [P] Validate all code references point to valid files and line numbers
- [X] T090 [P] Validate all Mermaid diagrams render correctly in GitHub and common markdown viewers
- [X] T091 Validate reading level is appropriate (8th-grade level for accessibility)
- [X] T092 Check for consistent tone, formatting, and style across all documentation files
- [X] T093 Verify no duplicate content between docs/ and CLAUDE.md (complementary not duplicate)
- [X] T094 Manual review: Can a new user complete quickstart.md successfully? (User Story 1 test)
- [X] T095 Manual review: Can a contributor locate components using architecture.md? (User Story 2 test)
- [X] T096 Manual review: Can a user execute advanced workflows using reference.md? (User Story 3 test)
- [X] T097 Update README.md to link to docs/ directory (if appropriate)
- [X] T098 Create PR description summarizing documentation additions

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories (tests must exist first per constitution)
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - Creates overview.md and quickstart.md
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Creates architecture.md (may reference quickstart.md)
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - Creates reference.md and troubleshooting.md (may reference all previous docs)

### Within Each User Story

- Overview.md and quickstart.md (US1) can be written in parallel
- Within quickstart.md, sections can be written in any order, then assembled
- Architecture.md (US2) component descriptions can be written in parallel, then diagrams added
- Reference.md (US3) command documentation can be written in parallel (each command independent)
- Troubleshooting.md (US3) error entries and FAQ can be written in parallel
- Cross-references should be added after content is stable

### Parallel Opportunities

- All Setup tasks (T002-T003) can run in parallel
- All test implementation tasks (T005-T012) can run in parallel within Foundational phase
- US1 overview.md (T014) and quickstart.md sections can be worked on in parallel
- US2 component descriptions (T028-T033) can be written in parallel
- US3 command documentation (T044-T054) can be written in parallel
- US3 config options (T056-T063) can be written in parallel
- US3 error documentation (T072-T077) can be written in parallel
- US3 FAQ entries (T080-T084) can be written in parallel
- All polish validation tasks (T087-T090) can run in parallel

---

## Parallel Example: User Story 1

```bash
# Create both main files together:
Task: T014 "Create docs/overview.md with project introduction"
Task: T015 "Create docs/quickstart.md with Prerequisites section"

# Tests for validation can run together:
Task: T023 "Validate overview.md is under 500 lines and meets content requirements"
Task: T024 "Validate quickstart.md is under 500 lines and meets content requirements"
```

## Parallel Example: User Story 3

```bash
# Document all CLI commands in parallel:
Task: T044 "Document autospec full command"
Task: T045 "Document autospec workflow command"
Task: T046 "Document autospec specify command"
Task: T047 "Document autospec plan command"
Task: T048 "Document autospec tasks command"
Task: T049 "Document autospec implement command"
Task: T050 "Document autospec doctor command"
Task: T051 "Document autospec status command"
Task: T052 "Document autospec config command"
Task: T053 "Document autospec init command"
Task: T054 "Document autospec version command"

# Document all config options in parallel:
Task: T056 "Document claude_cmd option"
Task: T057 "Document specify_cmd option"
Task: T058 "Document max_retries option"
Task: T059 "Document specs_dir option"
Task: T060 "Document state_dir option"
Task: T061 "Document timeout option"
Task: T062 "Document skip_preflight option"
Task: T063 "Document custom_claude_cmd option"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T003)
2. Complete Phase 2: Foundational (T004-T013) - CRITICAL: Tests first!
3. Complete Phase 3: User Story 1 (T014-T026) - Quick start documentation
4. **STOP and VALIDATE**: Test with a new user following quickstart.md
5. Deploy/demo if ready (basic documentation live)

### Incremental Delivery

1. Complete Setup + Foundational → Test framework ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP! New users can get started)
3. Add User Story 2 → Test independently → Deploy/Demo (Contributors can understand architecture)
4. Add User Story 3 → Test independently → Deploy/Demo (Complete reference documentation)
5. Each story adds value without breaking previous documentation

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (overview + quickstart)
   - Developer B: User Story 2 (architecture)
   - Developer C: User Story 3 (reference + troubleshooting)
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files or independent sections, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story delivers a complete, independently valuable documentation set
- Tests are written FIRST per constitution (Phase 2 completes before any docs written)
- All documentation files must stay under 500 lines (validated by tests)
- Code references use file:line format (e.g., internal/cli/root.go:42)
- Mermaid diagrams for architecture and workflow visualization
- Cross-references should be bidirectional (if A links to B, B should link back to A)
- Commit after each logical task or group of related tasks
- Stop at any checkpoint to validate documentation independently
