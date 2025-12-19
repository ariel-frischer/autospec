---
description: Generate or update project constitution in YAML format.
version: "1.0.0"
handoffs:
  - label: Build Specification
    agent: autospec.specify
    prompt: Create a feature specification based on the constitution. I want to build...
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

You are creating or updating the project constitution. This file defines the non-negotiable principles and governance rules for the project that all specifications, plans, and implementations must adhere to.

Follow this execution flow:

1. **Get version info**: Get autospec version and current timestamp:

   ```bash
   echo "AUTOSPEC_VERSION=$(autospec version --plain | head -1)" && echo "CREATED_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
   ```

   Parse the output for `AUTOSPEC_VERSION` and `CREATED_DATE` (for _meta section).

2. **Load existing context**:
   - Check if `.autospec/memory/constitution.yaml` exists
   - Check if `.autospec/memory/constitution.md` exists (for migration)
   - Check if `CLAUDE.md` exists at project root
   - Extract any existing principles, governance rules, or project guidelines

3. **Collect/derive values**:
   - If user input supplies principles, use them
   - Otherwise infer from existing repo context (README, docs, prior constitution)
   - For governance dates:
     - `ratified`: Original adoption date (if unknown, use today)
     - `last_amended`: Today if changes are made
   - For versioning: Use semantic versioning
     - MAJOR: Backward incompatible governance/principle changes
     - MINOR: New principle/section added
     - PATCH: Clarifications, wording fixes

4. **Generate constitution.yaml**:

   ```yaml
   constitution:
     project_name: "<project name>"
     version: "1.0.0"
     ratified: "<YYYY-MM-DD>"
     last_amended: "<YYYY-MM-DD>"

   preamble: |
     <Brief statement of the project's purpose and why these principles matter.
     This should be 2-3 sentences that set the context for the principles below.>

   principles:
     - name: "Test-First Development"
       id: "PRIN-001"
       category: "quality"  # quality | process | architecture | security | governance
       priority: "NON-NEGOTIABLE"  # NON-NEGOTIABLE | MUST | SHOULD | MAY
       description: |
         All new code must have tests written before implementation.
         Tests define the expected behavior and serve as living documentation.
       rationale: "Ensures code quality and prevents regressions"
       enforcement:
         - mechanism: "Pre-commit hooks"
           description: "Automated checks prevent commits without tests"
         - mechanism: "CI pipeline"
           description: "Build fails if test coverage decreases"
       exceptions:
         - "Prototype/spike code explicitly marked as such"
         - "Configuration files and documentation"

     - name: "Performance Standards"
       id: "PRIN-002"
       category: "quality"
       priority: "MUST"
       description: |
         Validation functions must complete in <10ms.
         User-facing operations must complete in <1s.
       rationale: "Maintains responsive user experience"
       enforcement:
         - mechanism: "Benchmark tests"
           description: "Automated performance regression tests"
       exceptions: []

     - name: "Idempotency & Retry Logic"
       id: "PRIN-003"
       category: "architecture"
       priority: "MUST"
       description: |
         All operations must be idempotent where possible.
         Configurable retry limits for recoverable failures.
       rationale: "Enables reliable distributed operations"
       enforcement:
         - mechanism: "Code review"
           description: "Reviewers check for idempotent patterns"
       exceptions:
         - "One-time initialization operations"

   sections:
     - name: "Code Quality"
       content: |
         All code must pass linting and formatting checks.
         No warnings allowed in production builds.
         Dependencies must be explicitly versioned.

     - name: "Documentation"
       content: |
         Public APIs must have documentation.
         Architecture decisions must be recorded.
         Breaking changes must be documented in CHANGELOG.

     - name: "Security"
       content: |
         No secrets in code or version control.
         Dependencies must be regularly audited.
         User input must be validated and sanitized.

   governance:
     amendment_process:
       - step: 1
         action: "Propose change via pull request"
         requirements: "Include rationale and impact assessment"
       - step: 2
         action: "Review period"
         requirements: "Minimum 48 hours for team review"
       - step: 3
         action: "Approval"
         requirements: "Requires maintainer approval"
       - step: 4
         action: "Merge and version bump"
         requirements: "Update version and last_amended date"

     versioning_policy: |
       Constitution versions follow semantic versioning.
       MAJOR: Changes that invalidate existing compliant code.
       MINOR: New principles or expanded guidance.
       PATCH: Clarifications without behavioral change.

     compliance_review:
       frequency: "quarterly"
       process: "Review all principles for relevance and enforcement effectiveness"

     rules:
       - "Changes require review by at least one maintainer"
       - "Breaking changes require explicit team discussion"
       - "Emergency changes may bypass review with post-hoc documentation"

   sync_impact:
     # This section is auto-generated when constitution is updated
     version_change: "1.0.0 -> 1.0.0"
     modified_principles: []
     added_sections: []
     removed_sections: []
     templates_requiring_updates: []
     follow_up_todos: []

   _meta:
     version: "1.0.0"
     generator: "autospec"
     generator_version: "<AUTOSPEC_VERSION from step 1>"
     created: "<CREATED_DATE from step 1>"
     artifact_type: "constitution"
   ```

5. **Write the constitution** to `.autospec/memory/constitution.yaml`
   - Create `.autospec/memory/` directory if it doesn't exist

6. **Validate the artifact**:
   ```bash
   autospec artifact .autospec/memory/constitution.yaml
   ```
   - If validation fails: fix schema errors (missing required fields, invalid types/enums) and retry
   - If validation passes: proceed to report

7. **Report**: Output:
   - Full path to constitution.yaml
   - Version (new or updated)
   - Number of principles defined
   - Governance rules summary
   - Suggested commit message (e.g., `docs: establish project constitution v1.0.0`)

## Principle Categories

- **quality**: Code quality, testing, performance standards
- **process**: Development workflow, review requirements
- **architecture**: Technical patterns, structure guidelines
- **security**: Security requirements, data handling
- **governance**: Decision-making, change management

## Priority Levels

- **NON-NEGOTIABLE**: Cannot be bypassed under any circumstances
- **MUST**: Required unless explicitly exempted
- **SHOULD**: Strongly recommended with documented exceptions
- **MAY**: Optional best practice

## Guidelines

### Formatting & Style

- Use clear, declarative language
- Each principle should be testable/verifiable
- Avoid vague terms ("should", "may" without context)
- Wrap long content to keep readability
- Keep single blank line between sections

### For AI Generation

When creating this constitution from user input:

1. **Extract implicit principles**: Look for patterns in existing codebase that suggest unwritten rules
2. **Infer from tools**: Package.json scripts, Makefile targets, CI config reveal expectations
3. **Balance strictness**: Not everything needs to be NON-NEGOTIABLE
4. **Consider enforcement**: Each principle should have at least one enforcement mechanism
5. **Allow exceptions**: Most principles have edge cases; document them

### Validation Checklist

Before finalizing:
- [ ] All principles have unique IDs (PRIN-001, PRIN-002, etc.)
- [ ] All principles have enforcement mechanisms
- [ ] Priority levels are appropriate
- [ ] Governance section includes amendment process
- [ ] Version follows semantic versioning
- [ ] Dates are in ISO 8601 format (YYYY-MM-DD)
