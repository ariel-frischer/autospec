# Implementation Plan: Go Binary Migration

**Branch**: `002-go-binary-migration` | **Date**: 2025-10-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/002-go-binary-migration/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Transform the current bash-based Auto Claude SpecKit validation tool into a single, cross-platform Go binary that provides the same validation and workflow orchestration capabilities without requiring bash, jq, git, or other shell utilities. The Go binary will support all major platforms (Linux, macOS, Windows) and provide a simple installation experience via `go install` or pre-built binaries.

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**: NEEDS CLARIFICATION (cobra for CLI, go-git for git operations, spf13/viper for config - need to research best practices)
**Storage**: File-based (JSON for config at ~/.autospec/config.json and .autospec/config.json, retry state at ~/.autospec/state/retry.json)
**Testing**: Go testing package (testing), table-driven tests, need to determine test coverage tool
**Target Platform**: Cross-platform (Linux amd64/arm64, macOS amd64/arm64, Windows amd64)
**Project Type**: Single CLI binary
**Performance Goals**: Startup <50ms, validation <100ms, status command <1s, workflow orchestration <5s (excluding Claude execution)
**Constraints**: Binary size <15MB, zero runtime dependencies beyond claude and specify CLIs, must support custom command templates with pipes and env vars
**Scale/Scope**: Single-user CLI tool, orchestrates 3-5 SpecKit workflow phases, manages ~10 configuration parameters, validates markdown files up to several MB

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Validation-First
**Status**: ✅ PASS
- Go binary will maintain all existing validation logic from bash scripts
- Automatic retry mechanisms will be preserved (max 3 attempts)
- All workflow transitions validated before proceeding

### II. Hook-Based Enforcement
**Status**: ⚠️ NOT APPLICABLE (with note)
- This migration focuses on the CLI tool itself, not the Claude Code hooks
- Existing hook scripts in `scripts/hooks/` will continue to work as-is
- Hooks call the validation logic which will be migrated to Go
- Future work may migrate hooks themselves, but not in this feature scope

### III. Test-First Development
**Status**: ✅ PASS
- All 60+ existing bash tests will be ported to Go tests before implementation
- Go testing framework supports table-driven tests for comprehensive coverage
- Will maintain test-first approach: write tests, then implementation
- Test coverage must not decrease below 60+ baseline

### IV. Performance Standards
**Status**: ✅ PASS
- Performance requirements explicitly defined in spec (startup <50ms, validation <100ms, status <1s)
- Go compiled binaries are typically faster than bash scripts
- Performance targets align with constitution (<1s for validation operations)

### V. Idempotency & Retry Logic
**Status**: ✅ PASS
- All retry mechanisms will be ported from bash to Go
- Persistent retry state at ~/.autospec/state/retry.json
- Standardized exit codes preserved (0=success, 1=failed, 2=exhausted, 3=invalid, 4=missing deps)
- Idempotent operations maintained

**Overall Gate Status**: ✅ PASS - Proceed to Phase 0

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->

```text
# [REMOVE IF UNUSED] Option 1: Single project (DEFAULT)
src/
├── models/
├── services/
├── cli/
└── lib/

tests/
├── contract/
├── integration/
└── unit/

# [REMOVE IF UNUSED] Option 2: Web application (when "frontend" + "backend" detected)
backend/
├── src/
│   ├── models/
│   ├── services/
│   └── api/
└── tests/

frontend/
├── src/
│   ├── components/
│   ├── pages/
│   └── services/
└── tests/

# [REMOVE IF UNUSED] Option 3: Mobile + API (when "iOS/Android" detected)
api/
└── [same as backend above]

ios/ or android/
└── [platform-specific structure: feature modules, UI flows, platform tests]
```

**Structure Decision**: [Document the selected structure and reference the real
directories captured above]

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
