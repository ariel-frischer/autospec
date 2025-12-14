---
description: Generate YAML feature specification from natural language description.
version: "1.0.0"
handoffs:
  - label: Create Plan
    agent: autospec.plan
    prompt: Generate implementation plan from the specification
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty).

## Outline

The text the user typed after `/autospec.specify` in the triggering message **is** the feature description. Assume you always have it available in this conversation even if `$ARGUMENTS` appears literally below. Do not ask the user to repeat it unless they provided an empty command.

Given that feature description, do this:

1. **Generate a concise short name** (2-4 words) for the branch:
   - Analyze the feature description and extract the most meaningful keywords
   - Create a 2-4 word short name that captures the essence of the feature
   - Use action-noun format when possible (e.g., "add-user-auth", "fix-payment-bug")
   - Preserve technical terms and acronyms (OAuth2, API, JWT, etc.)
   - Keep it concise but descriptive enough to understand the feature at a glance
   - Examples:
     - "I want to add user authentication" → "user-auth"
     - "Implement OAuth2 integration for the API" → "oauth2-api-integration"
     - "Create a dashboard for analytics" → "analytics-dashboard"
     - "Fix payment processing timeout bug" → "fix-payment-timeout"

2. **Check for existing branches before creating new one**:

   a. First, fetch all remote branches to ensure we have the latest information:

      ```bash
      git fetch --all --prune
      ```

   b. Find the highest feature number across all sources for the short-name:
      - Remote branches: `git ls-remote --heads origin | grep -E 'refs/heads/[0-9]+-<short-name>$'`
      - Local branches: `git branch | grep -E '^[* ]*[0-9]+-<short-name>$'`
      - Specs directories: Check for directories matching `specs/[0-9]+-<short-name>`

   c. Determine the next available number:
      - Extract all numbers from all three sources
      - Find the highest number N
      - Use N+1 for the new branch number

   d. Create the feature directory and branch:
      - Create `specs/<number>-<short-name>/` directory
      - Create and checkout git branch `<number>-<short-name>`
      - Set FEATURE_DIR to the absolute path of the created directory

3. **Generate spec.yaml**: Create the YAML specification file with this structure:

   ```yaml
   _meta:
     version: "1.0.0"
     generator: "autospec"
     generator_version: "<run autospec version to get this>"
     created: "<ISO 8601 timestamp>"
     artifact_type: "spec"

   feature:
     branch: "<branch name from step 2>"
     created: "<today's date YYYY-MM-DD>"
     status: "Draft"
     input: "<original user description verbatim>"

   user_stories:
     - id: "US-001"
       title: "<story title>"
       priority: "P1"  # P1=must-have, P2=should-have, P3=nice-to-have
       as_a: "<role/actor>"
       i_want: "<action/capability>"
       so_that: "<benefit/value>"
       why_this_priority: "<justification for priority level>"
       independent_test: "<how this story can be tested in isolation>"
       acceptance_scenarios:
         - given: "<precondition/context>"
           when: "<action taken>"
           then: "<expected outcome>"

   requirements:
     functional:
       - id: "FR-001"
         description: "<MUST/SHOULD/MAY + requirement>"
         testable: true
         acceptance_criteria: "<how to verify this>"
     non_functional:
       - id: "NFR-001"
         category: "<performance|security|usability|reliability>"
         description: "<requirement>"
         measurable_target: "<specific metric>"

   success_criteria:
     measurable_outcomes:
       - id: "SC-001"
         description: "<user-focused, measurable outcome>"
         metric: "<how to measure>"
         target: "<specific value or threshold>"

   key_entities:
     - name: "<entity name>"
       description: "<what this entity represents>"
       attributes:
         - "<key attribute 1>"
         - "<key attribute 2>"

   edge_cases:
     - scenario: "<edge case description>"
       expected_behavior: "<what should happen>"

   assumptions:
     - "<assumption 1>"
     - "<assumption 2>"

   constraints:
     - "<constraint 1>"
     - "<constraint 2>"

   out_of_scope:
     - "<explicitly excluded item 1>"
     - "<explicitly excluded item 2>"
   ```

4. **Follow this execution flow**:

   1. Parse user description from Input
      If empty: ERROR "No feature description provided"
   2. Extract key concepts from description
      Identify: actors, actions, data, constraints
   3. For unclear aspects:
      - Make informed guesses based on context and industry standards
      - Only mark with `clarification_needed: "<specific question>"` if:
        - The choice significantly impacts feature scope or user experience
        - Multiple reasonable interpretations exist with different implications
        - No reasonable default exists
      - **LIMIT: Maximum 3 clarification_needed markers total**
      - Prioritize clarifications by impact: scope > security/privacy > user experience > technical details
   4. Fill user_stories section
      If no clear user flow: ERROR "Cannot determine user scenarios"
   5. Generate functional requirements
      Each requirement must be testable
      Use reasonable defaults for unspecified details (document in assumptions)
   6. Define success_criteria
      Create measurable, technology-agnostic outcomes
      Include both quantitative metrics (time, performance, volume) and qualitative measures
      Each criterion must be verifiable without implementation details
   7. Identify key_entities (if data involved)
   8. Document edge_cases, assumptions, constraints, out_of_scope

5. **Write the specification** to `FEATURE_DIR/spec.yaml`

6. **Validate the YAML**:
   ```bash
   autospec yaml check FEATURE_DIR/spec.yaml
   ```
   - If validation fails: fix YAML syntax errors and retry
   - If validation passes: proceed to report

7. **Report**: Output:
   - Branch name created
   - Full path to spec.yaml
   - Summary of user stories and requirements count
   - Any clarification_needed items found
   - Readiness for `/autospec.plan`

## Quick Guidelines

- Focus on **WHAT** users need and **WHY**
- Avoid HOW to implement (no tech stack, APIs, code structure)
- Written for business stakeholders, not developers
- All YAML output must be syntactically valid

### For AI Generation

When creating this spec from a user prompt:

1. **Make informed guesses**: Use context, industry standards, and common patterns to fill gaps
2. **Document assumptions**: Record reasonable defaults in the assumptions section
3. **Limit clarifications**: Maximum 3 clarification_needed fields - use only for critical decisions that:
   - Significantly impact feature scope or user experience
   - Have multiple reasonable interpretations with different implications
   - Lack any reasonable default
4. **Prioritize clarifications**: scope > security/privacy > user experience > technical details
5. **Think like a tester**: Every vague requirement should be made specific and measurable

**Examples of reasonable defaults** (don't ask about these):
- Data retention: Industry-standard practices for the domain
- Performance targets: Standard web/mobile app expectations unless specified
- Error handling: User-friendly messages with appropriate fallbacks
- Authentication method: Standard session-based or OAuth2 for web apps
- Integration patterns: RESTful APIs unless specified otherwise

### Success Criteria Guidelines

Success criteria must be:
1. **Measurable**: Include specific metrics (time, percentage, count, rate)
2. **Technology-agnostic**: No mention of frameworks, languages, databases, or tools
3. **User-focused**: Describe outcomes from user/business perspective, not system internals
4. **Verifiable**: Can be tested/validated without knowing implementation details

**Good examples**:
- "Users can complete checkout in under 3 minutes"
- "System supports 10,000 concurrent users"
- "95% of searches return results in under 1 second"

**Bad examples** (implementation-focused):
- "API response time is under 200ms" (too technical)
- "Database can handle 1000 TPS" (implementation detail)
- "React components render efficiently" (framework-specific)
