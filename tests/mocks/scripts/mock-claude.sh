#!/bin/bash
# mock-claude.sh - Simulates Claude CLI behavior for testing
#
# This script provides a test double for the Claude CLI that:
# - NEVER makes network calls or runs actual claude
# - Returns configurable responses
# - Logs all calls for verification
# - Simulates delays and failures
# - Generates artifact files when MOCK_ARTIFACT_DIR is set
#
# Environment Variables:
#   MOCK_RESPONSE_FILE  - Path to file containing response to return (optional)
#   MOCK_CALL_LOG       - Path to log file for recording calls (optional)
#   MOCK_EXIT_CODE      - Exit code to return (default: 0)
#   MOCK_DELAY          - Seconds to delay before responding (default: 0)
#   MOCK_ARTIFACT_DIR   - Directory to create artifact files in (optional)
#   MOCK_SPEC_NAME      - Spec name for artifact generation (default: 001-test-feature)
#   MOCK_VERSION_INFO   - JSON version data for update command testing (optional)
#   MOCK_NETWORK_STATUS - Connectivity status: connected/disconnected (optional)
#
# Usage:
#   export MOCK_RESPONSE_FILE=/tmp/response.yaml
#   export MOCK_CALL_LOG=/tmp/calls.log
#   ./mock-claude.sh --print "Generate a spec"
#
# Artifact Generation:
#   When MOCK_ARTIFACT_DIR is set, the script parses the command to detect
#   which stage is being executed and creates the appropriate artifact file:
#   - /autospec.specify    -> creates spec.yaml
#   - /autospec.plan       -> creates plan.yaml
#   - /autospec.tasks      -> creates tasks.yaml
#   - /autospec.clarify    -> updates spec.yaml with clarifications
#   - /autospec.checklist  -> creates checklist.yaml
#   - /autospec.analyze    -> creates analysis.yaml

set -euo pipefail

# Defaults
EXIT_CODE="${MOCK_EXIT_CODE:-0}"
DELAY="${MOCK_DELAY:-0}"
SPEC_NAME="${MOCK_SPEC_NAME:-001-test-feature}"

# Log the call if MOCK_CALL_LOG is set
log_call() {
    if [[ -n "${MOCK_CALL_LOG:-}" ]]; then
        local timestamp
        timestamp=$(date -Iseconds 2>/dev/null || date +%Y-%m-%dT%H:%M:%S)
        {
            echo "---"
            echo "timestamp: \"${timestamp}\""
            echo "agent: claude"
            echo "args:"
            for arg in "$@"; do
                # Escape special characters for YAML
                local escaped_arg
                escaped_arg=$(echo "$arg" | sed 's/"/\\"/g')
                echo "  - \"${escaped_arg}\""
            done
            echo "pid: $$"
            echo "response_file: \"${MOCK_RESPONSE_FILE:-}\""
            echo "exit_code: ${EXIT_CODE}"
            echo "delay: ${DELAY}"
            if [[ -n "${MOCK_VERSION_INFO:-}" ]]; then
                echo "version_info: \"${MOCK_VERSION_INFO}\""
            fi
            if [[ -n "${MOCK_NETWORK_STATUS:-}" ]]; then
                echo "network_status: \"${MOCK_NETWORK_STATUS}\""
            fi
        } >> "${MOCK_CALL_LOG}"
    fi
}

# Output response
output_response() {
    if [[ -n "${MOCK_RESPONSE_FILE:-}" && -f "${MOCK_RESPONSE_FILE}" ]]; then
        cat "${MOCK_RESPONSE_FILE}"
    fi
}

# Generate spec.yaml artifact
generate_spec() {
    local spec_dir="$1"
    mkdir -p "$spec_dir"
    cat > "$spec_dir/spec.yaml" << 'SPEC_EOF'
feature:
  branch: "001-test-feature"
  created: "2025-01-01"
  status: "Draft"
  input: "test feature"
user_stories:
  - id: "US-001"
    title: "Test"
    priority: "P1"
    as_a: "developer"
    i_want: "to test"
    so_that: "it works"
    why_this_priority: "required"
    independent_test: "run test"
    acceptance_scenarios:
      - given: "setup"
        when: "action"
        then: "result"
requirements:
  functional:
    - id: "FR-001"
      description: "test feature"
      acceptance_criteria: "test passes"
  non_functional:
    - id: "NFR-001"
      category: "code_quality"
      description: "quality"
      measurable_target: "target"
success_criteria:
  measurable_outcomes:
    - id: "SC-001"
      description: "success"
      metric: "metric"
      target: "target"
key_entities: []
edge_cases: []
assumptions: []
constraints: []
out_of_scope: []
_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "spec"
SPEC_EOF
}

# Generate plan.yaml artifact
generate_plan() {
    local spec_dir="$1"
    mkdir -p "$spec_dir"
    cat > "$spec_dir/plan.yaml" << 'PLAN_EOF'
plan:
  branch: "001-test-feature"
  created: "2025-01-01"
  spec_path: "specs/001-test-feature/spec.yaml"
summary: "Test plan"
technical_context:
  language: "Go"
  framework: "None"
  primary_dependencies: []
  storage: "None"
  testing:
    framework: "Go testing"
    approach: "Unit tests"
  target_platform: "Linux"
  project_type: "cli"
  performance_goals: "Fast"
  constraints: []
  scale_scope: "Small"
constitution_check:
  constitution_path: ".autospec/constitution.yaml"
  gates: []
research_findings:
  decisions: []
data_model:
  entities: []
api_contracts:
  endpoints: []
project_structure:
  documentation: []
  source_code: []
  tests: []
implementation_phases:
  - phase: 1
    name: "Test"
    goal: "Test"
    deliverables: []
risks: []
open_questions: []
_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "plan"
PLAN_EOF
}

# Generate tasks.yaml artifact
generate_tasks() {
    local spec_dir="$1"
    mkdir -p "$spec_dir"
    cat > "$spec_dir/tasks.yaml" << 'TASKS_EOF'
tasks:
  branch: "001-test-feature"
  created: "2025-01-01"
  spec_path: "specs/001-test-feature/spec.yaml"
  plan_path: "specs/001-test-feature/plan.yaml"
summary:
  total_tasks: 1
  total_phases: 1
  parallel_opportunities: 0
  estimated_complexity: "low"
phases:
  - number: 1
    title: "Test"
    purpose: "Test"
    tasks:
      - id: "T001"
        title: "Test task"
        status: "Pending"
        type: "implementation"
        parallel: false
        story_id: "US-001"
        file_path: "test.go"
        dependencies: []
        acceptance_criteria:
          - "Test passes"
dependencies:
  user_story_order: []
  phase_order: []
parallel_execution: []
implementation_strategy:
  mvp_scope:
    phases: [1]
    description: "MVP"
    validation: "Tests pass"
  incremental_delivery: []
_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "tasks"
TASKS_EOF
}

# Generate constitution.yaml artifact
generate_constitution() {
    # Constitution goes in .autospec/memory/ relative to project root (MOCK_ARTIFACT_DIR parent)
    local project_dir
    project_dir=$(dirname "${MOCK_ARTIFACT_DIR}")
    local constitution_dir="${project_dir}/.autospec/memory"
    mkdir -p "$constitution_dir"
    cat > "$constitution_dir/constitution.yaml" << 'CONSTITUTION_EOF'
constitution:
  project_name: "test-project"
  version: "1.0.0"
  ratified: "2025-01-01"
  last_amended: "2025-01-01"

preamble: "Test project constitution for automated testing."

principles:
  - name: "Test-First Development"
    id: "PRIN-001"
    category: "quality"
    priority: "NON-NEGOTIABLE"
    description: "All new code must have tests."
    rationale: "Ensures code quality"
    enforcement:
      - mechanism: "CI"
        description: "Tests run on commit"
    exceptions: []

sections:
  - name: "Code Quality"
    content: "All code must pass linting."

governance:
  amendment_process:
    - step: 1
      action: "Propose"
      requirements: "Include rationale"
  versioning_policy: "Semantic versioning"
  compliance_review:
    frequency: "quarterly"
    process: "Review"
  rules:
    - "Changes require review"

sync_impact:
  version_change: "1.0.0 -> 1.0.0"
  modified_principles: []
  added_sections: []
  removed_sections: []
  templates_requiring_updates: []
  follow_up_todos: []

_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "constitution"
CONSTITUTION_EOF
}

# Generate clarify artifact (updates spec with clarifications)
generate_clarify() {
    local spec_dir="$1"
    local spec_file="$spec_dir/spec.yaml"

    # If spec doesn't exist, just return - clarify requires existing spec
    if [[ ! -f "$spec_file" ]]; then
        return
    fi

    # Append clarifications section if not present
    if ! grep -q "^clarifications:" "$spec_file"; then
        cat >> "$spec_file" << 'CLARIFY_EOF'
clarifications:
  - question: "What is the expected behavior?"
    answer: "The system should work correctly."
    date: "2025-01-01"
    applied_to: "requirements.functional.FR-001"
CLARIFY_EOF
    fi
}

# Generate checklist.yaml artifact
generate_checklist() {
    local spec_dir="$1"
    mkdir -p "$spec_dir/checklists"
    cat > "$spec_dir/checklists/checklist.yaml" << 'CHECKLIST_EOF'
checklist:
  spec_path: "specs/001-test-feature/spec.yaml"
  created: "2025-01-01"
  version: "1.0.0"
categories:
  - name: "Quality"
    items:
      - id: "CK-001"
        description: "Code is tested"
        status: "pending"
        notes: ""
      - id: "CK-002"
        description: "Code is documented"
        status: "pending"
        notes: ""
summary:
  total: 2
  passed: 0
  failed: 0
  pending: 2
_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "checklist"
CHECKLIST_EOF
}

# Generate analysis.yaml artifact
generate_analysis() {
    local spec_dir="$1"
    mkdir -p "$spec_dir"
    cat > "$spec_dir/analysis.yaml" << 'ANALYSIS_EOF'
analysis:
  branch: "001-test-feature"
  timestamp: "2025-01-01T00:00:00Z"
  spec_path: "specs/001-test-feature/spec.yaml"
  plan_path: "specs/001-test-feature/plan.yaml"
  tasks_path: "specs/001-test-feature/tasks.yaml"
findings: []
coverage:
  requirements:
    - id: "FR-001"
      status: "covered"
      tasks: ["T001"]
  user_stories:
    - id: "US-001"
      status: "covered"
      tasks: ["T001"]
constitution_alignment:
  status: "PASS"
  violations: []
unmapped_tasks: []
metrics:
  total_requirements: 1
  total_tasks: 1
  coverage_percentage: 100
  critical_issues: 0
  high_issues: 0
  medium_issues: 0
  low_issues: 0
summary:
  overall_status: "PASS"
  blocking_issues: 0
  actionable_improvements: 0
  ready_for_implementation: true
_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "analysis"
ANALYSIS_EOF
}

# Update tasks.yaml to mark all tasks as Completed (simulates implementation)
mark_tasks_completed() {
    local spec_dir="$1"
    local tasks_file="$spec_dir/tasks.yaml"

    # Only update if tasks.yaml exists
    if [[ ! -f "$tasks_file" ]]; then
        return
    fi

    # Use sed to replace status: "Pending" with status: "Completed"
    # Also handle InProgress status
    if [[ "$(uname)" == "Darwin" ]]; then
        # macOS sed requires different syntax
        sed -i '' 's/status: "Pending"/status: "Completed"/g' "$tasks_file"
        sed -i '' 's/status: "InProgress"/status: "Completed"/g' "$tasks_file"
    else
        # GNU sed
        sed -i 's/status: "Pending"/status: "Completed"/g' "$tasks_file"
        sed -i 's/status: "InProgress"/status: "Completed"/g' "$tasks_file"
    fi
}

# Detect command type and generate appropriate artifact
# Supports both:
# 1. Slash command format: /autospec.plan
# 2. Rendered template format: detects by template body patterns
generate_artifact() {
    if [[ -z "${MOCK_ARTIFACT_DIR:-}" ]]; then
        return
    fi

    local spec_dir="${MOCK_ARTIFACT_DIR}/${SPEC_NAME}"
    local command="$*"

    # Rendered template detection (primary) - matches patterns in template body
    # These patterns appear after YAML frontmatter is stripped:
    # Plan: "**Write the plan** to" or "Generate plan.yaml"
    # Tasks: "**Write the tasks** to" or contains "IMPL_PLAN"
    # Implement: contains "TasksFile" and "Execute tasks"
    # Specify: "**Write the specification** to"
    # Constitution: "ratified:" pattern (constitution schema)
    # Clarify: "clarifications" section
    # Checklist: "checklist" and "categories"
    # Analyze: "cross-artifact" or "analysis.yaml"

    # IMPORTANT: Check template body patterns to identify the command type
    # Templates reference each other in docs, so use unique patterns
    if [[ "$command" == *"**Write the plan** to"* ]] || [[ "$command" == *"Generate plan.yaml"* ]]; then
        generate_plan "$spec_dir"
    elif [[ "$command" == *"**Write the tasks** to"* ]] || [[ "$command" == *"IMPL_PLAN"* ]]; then
        generate_tasks "$spec_dir"
    elif [[ "$command" == *"TASKS_FILE"* ]] || [[ "$command" == *"TasksFile"* ]]; then
        mark_tasks_completed "$spec_dir"
    elif [[ "$command" == *"**Write the specification** to"* ]]; then
        generate_spec "$spec_dir"
    elif [[ "$command" == *"ratified:"* ]] || [[ "$command" == *"constitution.yaml"* && "$command" == *"governance:"* ]]; then
        generate_constitution
    elif [[ "$command" == *"underspecified areas"* ]] || [[ "$command" == *"clarifications:"* ]]; then
        generate_clarify "$spec_dir"
    elif [[ "$command" == *"checklist"* ]] && [[ "$command" == *"categories:"* ]]; then
        generate_checklist "$spec_dir"
    elif [[ "$command" == *"cross-artifact"* ]] || [[ "$command" == *"analysis.yaml"* ]]; then
        generate_analysis "$spec_dir"
    # Fallback to slash command patterns for backward compatibility
    elif [[ "$command" == *"/autospec.implement"* ]]; then
        mark_tasks_completed "$spec_dir"
    elif [[ "$command" == *"/autospec.tasks"* ]]; then
        generate_tasks "$spec_dir"
    elif [[ "$command" == *"/autospec.plan"* ]]; then
        generate_plan "$spec_dir"
    elif [[ "$command" == *"/autospec.specify"* ]]; then
        generate_spec "$spec_dir"
    elif [[ "$command" == *"/autospec.constitution"* ]]; then
        generate_constitution
    elif [[ "$command" == *"/autospec.clarify"* ]]; then
        generate_clarify "$spec_dir"
    elif [[ "$command" == *"/autospec.checklist"* ]]; then
        generate_checklist "$spec_dir"
    elif [[ "$command" == *"/autospec.analyze"* ]]; then
        generate_analysis "$spec_dir"
    fi
}

# Main execution
main() {
    # Log the call with all arguments
    log_call "$@"

    # Apply delay if configured (for timeout testing)
    if [[ "${DELAY}" -gt 0 ]]; then
        sleep "${DELAY}"
    fi

    # Generate artifacts if configured
    generate_artifact "$@"

    # Output configured response
    output_response

    # Exit with configured code
    exit "${EXIT_CODE}"
}

# Run main with all arguments
main "$@"
