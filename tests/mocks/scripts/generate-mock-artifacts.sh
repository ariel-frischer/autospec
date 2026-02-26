#!/bin/bash
# generate-mock-artifacts.sh - Creates valid autospec artifacts for testing
#
# This script generates valid spec.yaml, plan.yaml, and tasks.yaml files
# that pass autospec validation, for use in testing without real Claude calls.
#
# Usage:
#   ./generate-mock-artifacts.sh [OPTIONS]
#
# Options:
#   -o, --output DIR     Output directory (default: creates temp dir)
#   -f, --feature NAME   Feature name (default: "test-feature")
#   -t, --tasks COUNT    Number of tasks to generate (default: 3)
#   -h, --help           Show help

set -euo pipefail

# Defaults
FEATURE_NAME="test-feature"
TASK_COUNT=3
OUTPUT_DIR=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -f|--feature)
            FEATURE_NAME="$2"
            shift 2
            ;;
        -t|--tasks)
            TASK_COUNT="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -o, --output DIR     Output directory (default: creates temp dir)"
            echo "  -f, --feature NAME   Feature name (default: test-feature)"
            echo "  -t, --tasks COUNT    Number of tasks to generate (default: 3)"
            echo "  -h, --help           Show help"
            exit 0
            ;;
        *)
            echo "Unknown option: $1" >&2
            exit 1
            ;;
    esac
done

# Create output directory if not specified
if [[ -z "${OUTPUT_DIR}" ]]; then
    OUTPUT_DIR=$(mktemp -d)
fi

mkdir -p "${OUTPUT_DIR}"

TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
DATE_ONLY=$(date -u +"%Y-%m-%d")

# Generate spec.yaml
cat > "${OUTPUT_DIR}/spec.yaml" << EOF
feature:
  branch: "${FEATURE_NAME}"
  created: "${DATE_ONLY}"
  status: "Draft"
  input: "Mock feature for testing"

user_stories:
  - id: "US-001"
    title: "Test user story"
    priority: "P1"
    as_a: "developer"
    i_want: "to test the feature"
    so_that: "tests pass successfully"
    why_this_priority: "Testing is critical"
    independent_test: "Run unit tests"
    acceptance_scenarios:
      - given: "a test environment"
        when: "tests are executed"
        then: "all tests pass"

requirements:
  functional:
    - id: "FR-001"
      description: "Test requirement"
      testable: true
      acceptance_criteria: "Tests pass"
  non_functional:
    - id: "NFR-001"
      category: "code_quality"
      description: "Code quality requirement"
      measurable_target: "100% pass rate"

success_criteria:
  measurable_outcomes:
    - id: "SC-001"
      description: "All tests pass"
      metric: "Pass rate"
      target: "100%"

key_entities: []
edge_cases: []
assumptions:
  - "Testing environment is available"
constraints:
  - "Must not make real API calls"
out_of_scope:
  - "Production deployment"

_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "mock-generator"
  created: "${TIMESTAMP}"
  artifact_type: "spec"
EOF

# Generate plan.yaml
cat > "${OUTPUT_DIR}/plan.yaml" << EOF
plan:
  branch: "${FEATURE_NAME}"
  created: "${DATE_ONLY}"
  spec_path: "specs/${FEATURE_NAME}/spec.yaml"

summary: |
  Mock implementation plan for testing purposes.
  This plan is auto-generated for test scenarios.

technical_context:
  language: "Go"
  framework: "None"
  primary_dependencies: []
  storage: "None"
  testing:
    framework: "Go testing"
    approach: "Unit tests with mocks"
  target_platform: "Linux, macOS, Windows"
  project_type: "cli"
  performance_goals: "Fast test execution"
  constraints:
    - "No real API calls"
  scale_scope: "Unit tests"

constitution_check:
  constitution_path: ".autospec/constitution.yaml"
  gates: []

research_findings:
  decisions:
    - topic: "Testing approach"
      decision: "Use mocks for all external dependencies"
      rationale: "Ensures deterministic and fast tests"
      alternatives_considered:
        - "Real API calls (rejected: slow and costly)"

data_model:
  entities: []

api_contracts:
  endpoints: []

project_structure:
  documentation: []
  source_code: []
  tests:
    - path: "internal/*_test.go"
      description: "Unit tests"

implementation_phases:
  - phase: 1
    name: "Setup"
    goal: "Prepare test infrastructure"
    deliverables:
      - "Mock setup"
EOF

# Add phases for each task
for i in $(seq 2 $((TASK_COUNT > 1 ? 2 : 1))); do
    cat >> "${OUTPUT_DIR}/plan.yaml" << EOF
  - phase: ${i}
    name: "Implementation Phase ${i}"
    goal: "Implement features"
    deliverables:
      - "Feature implementation"
EOF
done

cat >> "${OUTPUT_DIR}/plan.yaml" << EOF

risks:
  - risk: "Test flakiness"
    likelihood: "low"
    impact: "medium"
    mitigation: "Use deterministic mocks"

open_questions: []

_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "mock-generator"
  created: "${TIMESTAMP}"
  artifact_type: "plan"
EOF

# Generate tasks.yaml
cat > "${OUTPUT_DIR}/tasks.yaml" << EOF
tasks:
  branch: "${FEATURE_NAME}"
  created: "${DATE_ONLY}"
  spec_path: "specs/${FEATURE_NAME}/spec.yaml"
  plan_path: "specs/${FEATURE_NAME}/plan.yaml"

summary:
  total_tasks: ${TASK_COUNT}
  total_phases: 2
  parallel_opportunities: $((TASK_COUNT > 2 ? TASK_COUNT - 2 : 0))
  estimated_complexity: "low"

phases:
  - number: 1
    title: "Setup"
    purpose: "Prepare test infrastructure"
    tasks:
      - id: "T001"
        title: "Setup mock infrastructure"
        status: "Pending"
        type: "setup"
        parallel: false
        story_id: "US-001"
        file_path: "mocks/"
        dependencies: []
        acceptance_criteria:
          - "Mock infrastructure created"
EOF

# Generate additional tasks
if [[ ${TASK_COUNT} -gt 1 ]]; then
    cat >> "${OUTPUT_DIR}/tasks.yaml" << EOF
  - number: 2
    title: "Implementation"
    purpose: "Implement feature"
    tasks:
EOF
    for i in $(seq 2 "${TASK_COUNT}"); do
        TASK_ID=$(printf "T%03d" "${i}")
        PARALLEL="true"
        [[ ${i} -eq ${TASK_COUNT} ]] && PARALLEL="false"
        cat >> "${OUTPUT_DIR}/tasks.yaml" << EOF
      - id: "${TASK_ID}"
        title: "Implement task ${i}"
        status: "Pending"
        type: "implementation"
        parallel: ${PARALLEL}
        story_id: "US-001"
        file_path: "internal/feature.go"
        dependencies: ["T001"]
        acceptance_criteria:
          - "Task ${i} implemented"
EOF
    done
fi

cat >> "${OUTPUT_DIR}/tasks.yaml" << EOF

dependencies:
  user_story_order:
    - story_id: "US-001"
      depends_on: []
      blocks: []
  phase_order:
    - phase: 1
      blocks: [2]

parallel_execution:
  - phase: 2
    parallel_groups:
      - tasks: [$(for i in $(seq 2 "${TASK_COUNT}"); do printf "\"T%03d\"" "${i}"; [[ ${i} -lt ${TASK_COUNT} ]] && printf ", "; done)]
        rationale: "Independent implementation tasks"

implementation_strategy:
  mvp_scope:
    phases: [1, 2]
    description: "Complete feature"
    validation: "All tests pass"
  incremental_delivery:
    - milestone: "Setup Complete"
      phases: [1]
      deliverable: "Infrastructure ready"

_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "mock-generator"
  created: "${TIMESTAMP}"
  artifact_type: "tasks"
EOF

# Output the directory path
echo "${OUTPUT_DIR}"
