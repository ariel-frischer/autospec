---
name: autospec-clarify
description: "Identify underspecified areas in YAML spec and encode clarifications back into the spec."
---

# autospec-clarify

This Agent Skill is generated from autospec.clarify. When the user invokes "$autospec-clarify" or "/autospec.clarify",
load and follow these instructions directly. Treat the text after the skill or
command name as "$ARGUMENTS". Do not route back through "autospec clarify"; this skill
is the prompt for the stage.

Project specs directory: ./specs

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

Goal: Detect and reduce ambiguity or missing decision points in the active feature specification and record the clarifications directly in the spec.yaml file.

Note: This clarification workflow should run BEFORE `$autospec-plan`. If the user explicitly states they are skipping clarification (e.g., exploratory spike), you may proceed, but must warn that downstream rework risk increases.

## Pre-computed Context

The following paths have been pre-computed and are available for use:

- **FEATURE_DIR**: `{{.FeatureDir}}`
- **FEATURE_SPEC**: `{{.FeatureSpec}}`

1. **Load and analyze** the spec file at `{{.FeatureSpec}}`. Perform a structured ambiguity & coverage scan using this taxonomy. For each category, mark status: Clear / Partial / Missing.

   **Functional Scope & Behavior:**
   - Core user goals & success criteria
   - Explicit out-of-scope declarations
   - User roles / personas differentiation

   **Domain & Data Model:**
   - Entities, attributes, relationships
   - Identity & uniqueness rules
   - Lifecycle/state transitions
   - Data volume / scale assumptions

   **Interaction & UX Flow:**
   - Critical user journeys / sequences
   - Error/empty/loading states
   - Accessibility or localization notes

   **Non-Functional Quality Attributes:**
   - Performance (latency, throughput targets)
   - Scalability (horizontal/vertical, limits)
   - Reliability & availability (uptime, recovery expectations)
   - Observability (logging, metrics, tracing signals)
   - Security & privacy (authN/Z, data protection, threat assumptions)
   - Compliance / regulatory constraints (if any)

   **Integration & External Dependencies:**
   - External services/APIs and failure modes
   - Data import/export formats
   - Protocol/versioning assumptions

   **Edge Cases & Failure Handling:**
   - Negative scenarios
   - Rate limiting / throttling
   - Conflict resolution (e.g., concurrent edits)

   **Constraints & Tradeoffs:**
   - Technical constraints (language, storage, hosting)
   - Explicit tradeoffs or rejected alternatives

   **Terminology & Consistency:**
   - Canonical glossary terms
   - Avoided synonyms / deprecated terms

   **Completion Signals:**
   - Acceptance criteria testability
   - Measurable Definition of Done style indicators

   **Misc / Placeholders:**
   - TODO markers / unresolved decisions
   - Ambiguous adjectives ("robust", "intuitive") lacking quantification

3. **Generate candidate questions** (maximum 5). Apply these constraints:
   - Maximum of 10 total questions across the whole session
   - Each question must be answerable with EITHER:
     - A short multiple-choice selection (2-5 distinct, mutually exclusive options), OR
     - A one-word / short-phrase answer (explicitly constrain: "Answer in <=5 words")
   - Only include questions whose answers materially impact architecture, data modeling, task decomposition, test design, UX behavior, operational readiness, or compliance validation
   - Ensure category coverage balance: attempt to cover the highest impact unresolved categories first
   - Exclude questions already answered, trivial stylistic preferences, or plan-level execution details
   - Favor clarifications that reduce downstream rework risk or prevent misaligned acceptance tests

4. **Sequential questioning loop** (interactive):
   - Present EXACTLY ONE question at a time
   - For multiple-choice questions:
     - **Analyze all options** and determine the **most suitable option** based on best practices, common patterns, risk reduction, and alignment with project goals
     - Present your **recommended option prominently** at the top with clear reasoning (1-2 sentences)
     - Format as: `**Recommended:** Option [X] - <reasoning>`
     - Then render all options as a Markdown table:

     | Option | Description |
     |--------|-------------|
     | A | <Option A description> |
     | B | <Option B description> |
     | C | <Option C description> |
     | Short | Provide a different short answer (<=5 words) |

     - After the table: `You can reply with the option letter (e.g., "A"), accept the recommendation by saying "yes" or "recommended", or provide your own short answer.`
   - For short-answer style (no meaningful discrete options):
     - Provide your **suggested answer** based on best practices and context
     - Format as: `**Suggested:** <your proposed answer> - <brief reasoning>`
     - Then output: `Format: Short answer (<=5 words). You can accept the suggestion by saying "yes" or "suggested", or provide your own answer.`
   - After the user answers:
     - If the user replies with "yes", "recommended", or "suggested", use your previously stated recommendation/suggestion as the answer
     - Otherwise, validate the answer maps to one option or fits the <=5 word constraint
     - If ambiguous, ask for a quick disambiguation
   - Stop asking when:
     - All critical ambiguities resolved early, OR
     - User signals completion ("done", "good", "no more"), OR
     - You reach 5 asked questions
   - Never reveal future queued questions in advance
   - If no valid questions exist at start, immediately report no critical ambiguities

5. **Integration after EACH accepted answer** (incremental update approach):
   - Maintain in-memory representation of the spec.yaml plus the raw file contents
   - For the first integrated answer in this session, ensure a `clarifications:` section exists in the YAML
   - Add clarification entry in this format:

     ```yaml
     clarifications:
       - date: "<YYYY-MM-DD>"
         question: "<the question asked>"
         answer: "<the answer provided>"
         applied_to: "<section(s) updated>"
     ```

   - Then immediately apply the clarification to the most appropriate section(s):
     - Functional ambiguity -> Update or add items in `requirements.functional`
     - User interaction / actor distinction -> Update `user_stories` section
     - Data shape / entities -> Update `key_entities` section
     - Non-functional constraint -> Add/modify in `requirements.non_functional`
     - Edge case / negative flow -> Add to `edge_cases` section
     - Terminology conflict -> Normalize term across spec
   - If the clarification invalidates an earlier ambiguous statement, replace that statement
   - Save the spec file AFTER each integration (atomic overwrite)
   - Preserve YAML formatting: do not reorder unrelated sections; keep structure intact
   - Keep each inserted clarification minimal and testable

6. **Validate the artifact** after each write:
   ```bash
   autospec artifact {{.FeatureSpec}}
   ```
   - If validation fails: fix schema errors (missing required fields, invalid types) and retry
   - If validation passes: proceed

7. **Report completion** (after questioning loop ends):
   - Number of questions asked & answered
   - Path to updated spec.yaml
   - Sections touched (list names)
   - Coverage summary table listing each taxonomy category with Status:
     - Resolved (was Partial/Missing and addressed)
     - Deferred (exceeds question quota or better suited for planning)
     - Clear (already sufficient)
     - Outstanding (still Partial/Missing but low impact)
   - If any Outstanding or Deferred remain, recommend whether to proceed to `$autospec-plan` or run `$autospec-clarify` again
   - Suggested next command

## Key Rules

- Output MUST be valid YAML (use `autospec artifact {{.FeatureSpec}}` to verify schema compliance)
- If no meaningful ambiguities found, respond: "No critical ambiguities detected worth formal clarification." and suggest proceeding
- If spec file missing, instruct user to run `$autospec-specify` first
- Never exceed 5 total asked questions (clarification retries for a single question do not count as new questions)
- Avoid speculative tech stack questions unless the absence blocks functional clarity
- Respect user early termination signals ("stop", "done", "proceed")
- If quota reached with unresolved high-impact categories remaining, explicitly flag them under Deferred with rationale
