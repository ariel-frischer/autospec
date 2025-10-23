# Tasks: GitHub Issue Templates

**Input**: Design documents from `/specs/006-github-issue-templates/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Validation tests are included as this is a documentation/configuration feature that requires verification.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

This project uses repository root paths:
- Templates: `.github/ISSUE_TEMPLATE/`
- Tests: `tests/github_templates/`
- Validation library: `tests/lib/validation_lib.sh` (new)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create directory structure and validation infrastructure

- [X] T001 Create .github/ISSUE_TEMPLATE/ directory
- [X] T002 [P] Create tests/github_templates/ directory for validation tests
- [X] T003 [P] Create tests/lib/validation_lib.sh file for validation functions

**Checkpoint**: Directory structure ready for template files

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Implement validation library that all user stories will use

**⚠️ CRITICAL**: No template files should be created without validation functions ready

- [X] T004 [P] Implement validate_yaml_syntax() function in tests/lib/validation_lib.sh
- [X] T005 [P] Implement validate_required_fields() function in tests/lib/validation_lib.sh
- [X] T006 [P] Implement validate_template_sections() function in tests/lib/validation_lib.sh
- [X] T007 [P] Implement validate_config_file() function in tests/lib/validation_lib.sh
- [X] T008 Implement validate_all_templates() function in tests/lib/validation_lib.sh (depends on T004-T007)

**Checkpoint**: Foundation ready - template creation can now begin

---

## Phase 3: User Story 1 - Bug Reporter Submits Structured Bug Report (Priority: P1) 🎯 MVP

**Goal**: Create bug report template that guides contributors to provide complete information for reproducing bugs

**Independent Test**: Create a test issue on GitHub using the bug report template and verify all required sections are present (Bug Description, Steps to Reproduce, Expected Behavior, Actual Behavior, Environment)

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T009 [P] [US1] Create test for bug_report.md file existence in tests/github_templates/bug_report_test.sh
- [X] T010 [P] [US1] Create test for YAML frontmatter validation in tests/github_templates/bug_report_test.sh
- [X] T011 [P] [US1] Create test for required YAML fields (name, about) in tests/github_templates/bug_report_test.sh
- [X] T012 [P] [US1] Create test for required sections presence in tests/github_templates/bug_report_test.sh

### Implementation for User Story 1

- [X] T013 [US1] Create bug_report.md with YAML frontmatter in .github/ISSUE_TEMPLATE/bug_report.md
- [X] T014 [US1] Add Bug Description section to .github/ISSUE_TEMPLATE/bug_report.md
- [X] T015 [US1] Add Steps to Reproduce section to .github/ISSUE_TEMPLATE/bug_report.md
- [X] T016 [US1] Add Expected Behavior section to .github/ISSUE_TEMPLATE/bug_report.md
- [X] T017 [US1] Add Actual Behavior section to .github/ISSUE_TEMPLATE/bug_report.md
- [X] T018 [US1] Add Environment section with project-specific fields to .github/ISSUE_TEMPLATE/bug_report.md
- [X] T019 [US1] Add Additional Context section to .github/ISSUE_TEMPLATE/bug_report.md
- [X] T020 [US1] Run validation tests to verify bug_report.md structure

**Checkpoint**: At this point, User Story 1 should be fully functional - contributors can create bug reports with structured information

---

## Phase 4: User Story 2 - User Requests Feature with Clear Justification (Priority: P2)

**Goal**: Create feature request template that guides users to describe problems and use cases rather than jumping to implementation details

**Independent Test**: Create a test issue on GitHub using the feature request template and verify it guides users to describe the problem statement, use case, proposed solution, and alternatives considered

### Tests for User Story 2

- [X] T021 [P] [US2] Create test for feature_request.md file existence in tests/github_templates/feature_request_test.sh
- [X] T022 [P] [US2] Create test for YAML frontmatter validation in tests/github_templates/feature_request_test.sh
- [X] T023 [P] [US2] Create test for required YAML fields (name, about) in tests/github_templates/feature_request_test.sh
- [X] T024 [P] [US2] Create test for required sections presence in tests/github_templates/feature_request_test.sh

### Implementation for User Story 2

- [X] T025 [US2] Create feature_request.md with YAML frontmatter in .github/ISSUE_TEMPLATE/feature_request.md
- [X] T026 [US2] Add Problem Statement section to .github/ISSUE_TEMPLATE/feature_request.md
- [X] T027 [US2] Add Use Case section to .github/ISSUE_TEMPLATE/feature_request.md
- [X] T028 [US2] Add Proposed Solution section to .github/ISSUE_TEMPLATE/feature_request.md
- [X] T029 [US2] Add Alternatives Considered section to .github/ISSUE_TEMPLATE/feature_request.md
- [X] T030 [US2] Add Additional Context section to .github/ISSUE_TEMPLATE/feature_request.md
- [X] T031 [US2] Run validation tests to verify feature_request.md structure

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently - contributors can submit both bug reports and feature requests

---

## Phase 5: User Story 3 - Maintainer Manages Issue Template Configuration (Priority: P3)

**Goal**: Create configuration file that allows maintainers to control template behavior (blank issues enabled/disabled, contact links for external resources)

**Independent Test**: Modify config.yml settings (toggle blank_issues_enabled, add/remove contact links) and verify GitHub respects the configuration changes

### Tests for User Story 3

- [X] T032 [P] [US3] Create test for config.yml file existence in tests/github_templates/config_test.sh
- [X] T033 [P] [US3] Create test for YAML syntax validation in tests/github_templates/config_test.sh
- [X] T034 [P] [US3] Create test for blank_issues_enabled boolean validation in tests/github_templates/config_test.sh
- [X] T035 [P] [US3] Create test for contact_links structure validation in tests/github_templates/config_test.sh

### Implementation for User Story 3

- [X] T036 [US3] Create config.yml with blank_issues_enabled setting in .github/ISSUE_TEMPLATE/config.yml
- [X] T037 [US3] Add contact_links section with GitHub Discussions link in .github/ISSUE_TEMPLATE/config.yml
- [X] T038 [US3] Add contact_links entry for Documentation in .github/ISSUE_TEMPLATE/config.yml
- [X] T039 [US3] Run validation tests to verify config.yml structure

**Checkpoint**: All user stories should now be independently functional - complete template system with bug reports, feature requests, and configuration

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Integration testing, documentation, and GitHub UI verification

- [X] T040 [P] Create master validation script tests/github_templates/validate_all.sh that runs all validations
- [X] T041 [P] Update README.md with section on issue templates and contributor guidance
- [ ] T042 [P] Update CONTRIBUTING.md (if exists) with template usage instructions
- [ ] T043 Run quickstart.md validation by following all steps manually
- [ ] T044 Test bug_report.md template on GitHub UI (create test issue and verify rendering)
- [ ] T045 Test feature_request.md template on GitHub UI (create test issue and verify rendering)
- [ ] T046 Test config.yml settings on GitHub UI (verify blank issues disabled and contact links appear)
- [ ] T047 Verify all template YAML fields apply correctly (title prefixes, labels, assignees)
- [X] T048 Add shell script validation with shellcheck for tests/lib/validation_lib.sh
- [ ] T049 Create .github/workflows/validate-templates.yml CI workflow (optional - automate validation)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-5)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - No dependencies on other stories

### Within Each User Story

- Tests MUST be written and FAIL before implementation begins
- Template file creation before section additions
- All sections added before validation testing
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel (T002, T003)
- All Foundational validation functions marked [P] can run in parallel (T004-T007)
- All tests for User Story 1 marked [P] can run in parallel (T009-T012)
- All tests for User Story 2 marked [P] can run in parallel (T021-T024)
- All tests for User Story 3 marked [P] can run in parallel (T032-T035)
- Once Foundational phase completes, all three user stories can start in parallel (if team capacity allows)
- All Polish tasks marked [P] can run in parallel (T040-T042, T048)

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together:
Task: "Create test for bug_report.md file existence in tests/github_templates/bug_report_test.sh"
Task: "Create test for YAML frontmatter validation in tests/github_templates/bug_report_test.sh"
Task: "Create test for required YAML fields (name, about) in tests/github_templates/bug_report_test.sh"
Task: "Create test for required sections presence in tests/github_templates/bug_report_test.sh"

# After tests fail, implement template:
# T013-T019 must run sequentially (all modify same file)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T003)
2. Complete Phase 2: Foundational (T004-T008) - CRITICAL - blocks all stories
3. Complete Phase 3: User Story 1 (T009-T020)
4. **STOP and VALIDATE**: Test User Story 1 independently on GitHub UI
5. Deploy/demo if ready (contributors can now file structured bug reports)

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP! - bug reports work)
3. Add User Story 2 → Test independently → Deploy/Demo (bug reports + feature requests work)
4. Add User Story 3 → Test independently → Deploy/Demo (full template system with configuration)
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (bug report template)
   - Developer B: User Story 2 (feature request template)
   - Developer C: User Story 3 (configuration file)
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each template file is complete
- Stop at any checkpoint to validate story independently on GitHub UI
- Manual GitHub UI testing is REQUIRED (automated tests only validate syntax/structure, not rendering)
- Validation functions follow performance contract: <100ms per function
- All validation functions return standard exit codes (0=success, 1=fail, 2=optional missing)
