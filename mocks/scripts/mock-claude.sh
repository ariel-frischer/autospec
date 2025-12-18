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
#   MOCK_RESPONSE_FILE - Path to file containing response to return (optional)
#   MOCK_CALL_LOG      - Path to log file for recording calls (optional)
#   MOCK_EXIT_CODE     - Exit code to return (default: 0)
#   MOCK_DELAY         - Seconds to delay before responding (default: 0)
#   MOCK_ARTIFACT_DIR  - Directory to create artifact files in (optional)
#   MOCK_SPEC_NAME     - Spec name for artifact generation (default: 001-test-feature)
#
# Usage:
#   export MOCK_RESPONSE_FILE=/tmp/response.yaml
#   export MOCK_CALL_LOG=/tmp/calls.log
#   ./mock-claude.sh --print "Generate a spec"
#
# Artifact Generation:
#   When MOCK_ARTIFACT_DIR is set, the script parses the command to detect
#   which stage is being executed and creates the appropriate artifact file:
#   - /autospec.specify -> creates spec.yaml
#   - /autospec.plan    -> creates plan.yaml
#   - /autospec.tasks   -> creates tasks.yaml

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
      testable: true
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
  constitution_path: ".autospec/memory/constitution.yaml"
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
generate_artifact() {
    if [[ -z "${MOCK_ARTIFACT_DIR:-}" ]]; then
        return
    fi

    local spec_dir="${MOCK_ARTIFACT_DIR}/${SPEC_NAME}"
    local command="$*"

    if [[ "$command" == *"/autospec.specify"* ]]; then
        generate_spec "$spec_dir"
    elif [[ "$command" == *"/autospec.plan"* ]]; then
        generate_plan "$spec_dir"
    elif [[ "$command" == *"/autospec.tasks"* ]]; then
        generate_tasks "$spec_dir"
    elif [[ "$command" == *"/autospec.implement"* ]]; then
        mark_tasks_completed "$spec_dir"
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
