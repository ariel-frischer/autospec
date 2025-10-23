# Tasks: Command Execution Timeout

**Input**: Design documents from `/specs/003-command-timeout/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Tests are NOT requested in the specification, so test tasks are OMITTED from this plan. However, the constitution requires test-first development, so tests will be written during implementation as needed.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- Single Go binary project with `cmd/autospec/` and `internal/` packages
- Tests in `*_test.go` files alongside source files
- Integration tests in `integration/` directory

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Verify Go 1.25.1 is available and project builds successfully
- [X] T002 Review existing codebase structure (internal/config/, internal/workflow/) for integration points

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T003 Create TimeoutError type in internal/workflow/errors.go with Error() and Unwrap() methods
- [X] T004 Add unit tests for TimeoutError in internal/workflow/errors_test.go
- [X] T005 Define exit code constant (5) for timeout errors in internal/cli/exit_codes.go or appropriate location

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 2 - Configure Timeout Duration (Priority: P2)

**Goal**: Enable timeout configuration through the application's configuration system with validation

**Independent Test**: Set timeout values in config files and environment variables, verify they load correctly and invalid values are rejected

**Why P2 before P1**: Configuration must be in place before timeout enforcement can be implemented. This is a dependency reversal from the spec's priority order, but necessary for implementation.

### Implementation for User Story 2

- [X] T006 [US2] Update defaults.go to add default timeout value of 0 (no timeout) in internal/config/defaults.go
- [X] T007 [US2] Verify Configuration struct already has Timeout field with validation tag in internal/config/config.go:27
- [X] T008 [US2] Add unit tests for timeout config validation in internal/config/config_test.go
  - Test case: valid timeout (300)
  - Test case: missing timeout (defaults to 0)
  - Test case: timeout = 0 (valid, no timeout)
  - Test case: invalid timeout (negative value)
  - Test case: invalid timeout (too large, >3600)
  - Test case: environment variable override (AUTOSPEC_TIMEOUT)
  - Test case: non-numeric timeout value
- [ ] T009 [US2] Add integration test for config hierarchy in integration/config_test.go
  - Test config file < environment variable priority
  - Test local config < global config < env priority chain

**Checkpoint**: At this point, timeout configuration loads correctly with validation. Config value available but not yet used.

---

## Phase 4: User Story 1 - Prevent Indefinite Command Hangs (Priority: P1) 🎯 MVP

**Goal**: Implement core timeout mechanism that automatically aborts commands exceeding configured duration

**Independent Test**: Configure a timeout value, execute a command that exceeds the timeout, verify the command aborts with proper cleanup

### Implementation for User Story 1

- [X] T010 [US1] Add Timeout field to ClaudeExecutor struct in internal/workflow/claude.go
- [X] T011 [US1] Modify Execute() method to create context.WithTimeout when Timeout > 0 in internal/workflow/claude.go
- [X] T012 [US1] Modify Execute() to use exec.CommandContext instead of exec.Command in internal/workflow/claude.go
- [X] T013 [US1] Add timeout detection logic (check ctx.Err() == context.DeadlineExceeded) in internal/workflow/claude.go
- [X] T014 [US1] Return TimeoutError when timeout is detected in internal/workflow/claude.go
- [X] T015 [US1] Update ExecuteSpecKitCommand() to support timeout (delegates to Execute) in internal/workflow/claude.go
- [X] T016 [US1] Update StreamCommand() to support timeout in internal/workflow/claude.go
- [X] T017 [US1] Modify ClaudeExecutor creation in workflow orchestrator to pass cfg.Timeout in internal/workflow/executor.go
- [X] T018 [US1] Add unit tests for timeout enforcement in internal/workflow/claude_test.go
  - Test: Execute with Timeout=0 (no timeout, backward compatible)
  - Test: Execute with timeout, command completes before timeout
  - Test: Execute with timeout, command exceeds timeout
  - Test: TimeoutError includes correct metadata (timeout, command)
  - Test: Process cleanup verification (no orphaned processes)
- [X] T019 [US1] Add benchmark tests to verify <1% overhead in internal/workflow/claude_bench_test.go
  - Benchmark: Execute without timeout
  - Benchmark: Execute with timeout
  - Verify overhead < 1% (satisfies SC-004)
- [ ] T020 [US1] Add integration test for timeout with real command execution in integration/claude_test.go
  - Test: Long-running command with short timeout
  - Test: Config-to-executor timeout propagation

**Checkpoint**: At this point, timeout enforcement works. Commands abort after timeout, but error messages need improvement.

---

## Phase 5: User Story 3 - Receive Clear Timeout Feedback (Priority: P3)

**Goal**: Provide clear, actionable error messages when commands timeout

**Independent Test**: Trigger timeout conditions, verify error messages include timeout duration, command that timed out, and remediation suggestions

### Implementation for User Story 3

- [X] T021 [US3] Update TimeoutError.Error() to include clear message with hint in internal/workflow/errors.go
- [X] T022 [US3] Add CLI error handling for TimeoutError with exit code 5 in relevant CLI commands (internal/cli/workflow.go, full.go, implement.go, etc.)
- [X] T023 [US3] Ensure error output includes timeout duration and suggestion to increase timeout
- [X] T024 [US3] Add unit tests for error message format in internal/workflow/errors_test.go
  - Test: Error message includes timeout duration
  - Test: Error message includes command that timed out
  - Test: Error message includes hint about increasing timeout
  - Test: Unwrap returns context.DeadlineExceeded
- [ ] T025 [US3] Add integration test for timeout error propagation to CLI in integration/claude_test.go
  - Test: CLI exits with code 5 on timeout
  - Test: Error message format in stderr output

**Checkpoint**: All user stories complete. Timeout feature fully functional with good UX.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation and final quality improvements

- [X] T026 [P] Update CLAUDE.md with timeout configuration documentation
  - Add timeout field to configuration section
  - Add timeout behavior to architecture overview
  - Add timeout to common development patterns
  - Add troubleshooting guide for timeouts
- [X] T027 [P] Update quickstart.md validation scenarios (verify documented scenarios still work)
- [X] T028 [P] Add timeout examples to README or user-facing docs (if applicable)
- [ ] T029 Run quickstart.md validation to ensure all documented scenarios work correctly
- [X] T030 Final code review and cleanup (remove any debug code, ensure consistent error messages)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 2 (Phase 3)**: Depends on Foundational phase - Must complete BEFORE US1
- **User Story 1 (Phase 4)**: Depends on Foundational AND US2 (needs config to be loadable)
- **User Story 3 (Phase 5)**: Depends on US1 (needs timeout enforcement to exist for error handling)
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

**IMPORTANT**: The implementation order differs from the spec priority order due to technical dependencies:

- **User Story 2 (P2)**: MUST be implemented first - provides configuration infrastructure
- **User Story 1 (P1)**: Implemented second - depends on config (US2)
- **User Story 3 (P3)**: Implemented last - depends on timeout enforcement (US1)

**Spec Priority Order**: P1 (US1) → P2 (US2) → P3 (US3)
**Implementation Order**: P2 (US2) → P1 (US1) → P3 (US3)

This is acceptable because:
- US2 is foundational infrastructure (configuration)
- US1 cannot function without US2
- Each story is still independently testable once its dependencies are met

### Within Each User Story

- Foundational types (TimeoutError, config fields) before implementation
- Tests written alongside or before implementation (per constitution)
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- Phase 1: Tasks T001 and T002 can run in parallel (independent verification tasks)
- Phase 2: Tasks T003, T004, T005 can run in parallel (different files)
- Phase 3 (US2): Tasks T006, T007 can run in parallel (different files)
- Phase 6: Tasks T026, T027, T028 can run in parallel (different documentation files)

**User Stories CANNOT run in parallel** - they have strict dependencies:
- US2 must complete before US1 can start
- US1 must complete before US3 can start

---

## Parallel Example: Foundational Phase

```bash
# Launch all foundational tasks together (different files, no dependencies):
Task: "Create TimeoutError type in internal/workflow/errors.go"
Task: "Add unit tests for TimeoutError in internal/workflow/errors_test.go"
Task: "Define exit code constant (5) for timeout errors in internal/cli/exit_codes.go"
```

---

## Implementation Strategy

### MVP First (User Stories 2 + 1)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 2 (Configuration)
4. Complete Phase 4: User Story 1 (Timeout Enforcement)
5. **STOP and VALIDATE**: Test timeout enforcement with config
6. This gives us a working timeout feature (MVP!)

### Full Feature (All Stories)

1. Complete MVP (US2 + US1)
2. Add Phase 5: User Story 3 (Error Feedback)
3. **STOP and VALIDATE**: Test complete user experience
4. Complete Phase 6: Polish (Documentation)
5. Final validation with quickstart.md scenarios

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 2 → Test config loading → Can configure timeouts (partial value)
3. Add User Story 1 → Test timeout enforcement → Commands timeout (MVP!)
4. Add User Story 3 → Test error messages → Great UX (feature complete)
5. Add Polish → Documentation complete → Ship it!

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Tests are NOT explicitly requested in spec, but constitution requires test-first
- Tests will be written as part of implementation tasks
- Each user story builds on previous stories (strict dependency chain)
- Commit after each task or logical group
- Constitution principle: Test-First Development is NON-NEGOTIABLE
- All tests must be written before or during implementation, not after
