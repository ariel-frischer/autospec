#!/usr/bin/env bash

# DEPRECATION WARNING: This script is deprecated and will be removed in a future version.
# Please use the Go command instead: autospec prereqs
#
# Example:
#   autospec prereqs --json --require-spec
#   autospec prereqs --json --require-tasks --include-tasks
#   autospec prereqs --paths-only
#
>&2 echo "[DEPRECATED] check-prerequisites.sh is deprecated. Use 'autospec prereqs' instead."

# Consolidated prerequisite checking script
#
# This script provides unified prerequisite checking for Spec-Driven Development workflow.
# It replaces the functionality previously spread across multiple scripts.
#
# Usage: ./check-prerequisites.sh [OPTIONS]
#
# OPTIONS:
#   --json              Output in JSON format
#   --require-tasks     Require tasks file to exist (for implementation phase)
#   --require-spec      Require spec file to exist
#   --require-plan      Require plan file to exist (default behavior)
#   --include-tasks     Include tasks file in AVAILABLE_DOCS list
#   --paths-only        Only output path variables (no validation)
#   --help, -h          Show help message
#
# OUTPUTS:
#   JSON mode: {"FEATURE_DIR":"...", "AVAILABLE_DOCS":["..."]}
#   Text mode: FEATURE_DIR:... \n AVAILABLE_DOCS: \n ✓/✗ file
#   Paths only: REPO_ROOT: ... \n BRANCH: ... \n FEATURE_DIR: ... etc.
#
# NOTE: Supports both YAML (.yaml) and Markdown (.md) file formats.
#       YAML files are preferred when both exist.

set -e

# Parse command line arguments
JSON_MODE=false
REQUIRE_TASKS=false
REQUIRE_SPEC=false
REQUIRE_PLAN=true  # Default: require plan
INCLUDE_TASKS=false
PATHS_ONLY=false

for arg in "$@"; do
    case "$arg" in
        --json)
            JSON_MODE=true
            ;;
        --require-tasks)
            REQUIRE_TASKS=true
            ;;
        --require-spec)
            REQUIRE_SPEC=true
            REQUIRE_PLAN=false  # If only spec required, don't require plan
            ;;
        --require-plan)
            REQUIRE_PLAN=true
            ;;
        --include-tasks)
            INCLUDE_TASKS=true
            ;;
        --paths-only)
            PATHS_ONLY=true
            ;;
        --help|-h)
            cat << 'EOF'
Usage: check-prerequisites.sh [OPTIONS]

Prerequisite checking for autospec YAML workflow.
Works with YAML files only (.yaml).

OPTIONS:
  --json              Output in JSON format
  --require-spec      Require spec.yaml to exist
  --require-plan      Require plan.yaml to exist (default)
  --require-tasks     Require tasks.yaml to exist (for implementation phase)
  --include-tasks     Include tasks.yaml in AVAILABLE_DOCS list
  --paths-only        Only output path variables (no prerequisite validation)
  --help, -h          Show this help message

EXAMPLES:
  # Check spec prerequisites (spec.yaml required)
  ./check-prerequisites.sh --json --require-spec

  # Check plan prerequisites (plan.yaml required)
  ./check-prerequisites.sh --json

  # Check implementation prerequisites (plan.yaml + tasks.yaml required)
  ./check-prerequisites.sh --json --require-tasks --include-tasks

  # Get feature paths only (no validation)
  ./check-prerequisites.sh --paths-only

EOF
            exit 0
            ;;
        *)
            echo "ERROR: Unknown option '$arg'. Use --help for usage information." >&2
            exit 1
            ;;
    esac
done

# Source common functions
SCRIPT_DIR="$(CDPATH="" cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# Get feature paths and validate branch
eval $(get_feature_paths)
check_feature_branch "$CURRENT_BRANCH" "$HAS_GIT" || exit 1

# If paths-only mode, output paths and exit (support JSON + paths-only combined)
if $PATHS_ONLY; then
    if $JSON_MODE; then
        # Minimal JSON paths payload (no validation performed)
        printf '{"REPO_ROOT":"%s","BRANCH":"%s","FEATURE_DIR":"%s","FEATURE_SPEC":"%s","IMPL_PLAN":"%s","TASKS":"%s","AUTOSPEC_VERSION":"%s","CREATED_DATE":"%s"}\n' \
            "$REPO_ROOT" "$CURRENT_BRANCH" "$FEATURE_DIR" "$FEATURE_SPEC" "$IMPL_PLAN" "$TASKS" "$AUTOSPEC_VERSION" "$CREATED_DATE"
    else
        echo "REPO_ROOT: $REPO_ROOT"
        echo "BRANCH: $CURRENT_BRANCH"
        echo "FEATURE_DIR: $FEATURE_DIR"
        echo "FEATURE_SPEC: $FEATURE_SPEC"
        echo "IMPL_PLAN: $IMPL_PLAN"
        echo "TASKS: $TASKS"
    fi
    exit 0
fi

# Validate required directories and files
if [[ ! -d "$FEATURE_DIR" ]]; then
    echo "ERROR: Feature directory not found: $FEATURE_DIR" >&2
    echo "Run /autospec.specify first to create the feature structure." >&2
    exit 1
fi

# Check for spec file if required (YAML-only)
if $REQUIRE_SPEC && [[ ! -f "$FEATURE_SPEC" ]]; then
    echo "ERROR: No spec.yaml found in $FEATURE_DIR" >&2
    echo "Run /autospec.specify first to create the spec." >&2
    exit 1
fi

# Check for plan file if required (YAML-only)
if $REQUIRE_PLAN && [[ ! -f "$IMPL_PLAN" ]]; then
    echo "ERROR: No plan.yaml found in $FEATURE_DIR" >&2
    echo "Run /autospec.plan first to create the plan." >&2
    exit 1
fi

# Check for tasks file if required (YAML-only)
if $REQUIRE_TASKS && [[ ! -f "$TASKS" ]]; then
    echo "ERROR: No tasks.yaml found in $FEATURE_DIR" >&2
    echo "Run /autospec.tasks first to create tasks." >&2
    exit 1
fi

# Build list of available documents (YAML artifacts only)
docs=()

# Include spec.yaml if it exists
[[ -f "$FEATURE_SPEC" ]] && docs+=("$(basename "$FEATURE_SPEC")")

# Include plan.yaml if it exists
[[ -f "$IMPL_PLAN" ]] && docs+=("$(basename "$IMPL_PLAN")")

# Include tasks.yaml if requested and it exists
if $INCLUDE_TASKS && [[ -f "$TASKS" ]]; then
    docs+=("$(basename "$TASKS")")
fi

# Check for checklists directory
if [[ -d "$FEATURE_DIR/checklists" ]] && [[ -n "$(ls -A "$FEATURE_DIR/checklists" 2>/dev/null)" ]]; then
    docs+=("checklists/")
fi

# Output results
if $JSON_MODE; then
    # Build JSON array of documents
    if [[ ${#docs[@]} -eq 0 ]]; then
        json_docs="[]"
    else
        json_docs=$(printf '"%s",' "${docs[@]}")
        json_docs="[${json_docs%,}]"
    fi

    printf '{"FEATURE_DIR":"%s","FEATURE_SPEC":"%s","IMPL_PLAN":"%s","TASKS":"%s","AVAILABLE_DOCS":%s,"AUTOSPEC_VERSION":"%s","CREATED_DATE":"%s"}\n' \
        "$FEATURE_DIR" "$FEATURE_SPEC" "$IMPL_PLAN" "$TASKS" "$json_docs" "$AUTOSPEC_VERSION" "$CREATED_DATE"
else
    # Text output
    echo "FEATURE_DIR:$FEATURE_DIR"
    echo "FEATURE_SPEC:$FEATURE_SPEC"
    echo "IMPL_PLAN:$IMPL_PLAN"
    echo "TASKS:$TASKS"
    echo "AVAILABLE_DOCS:"

    # Show status of YAML artifacts
    check_file "$FEATURE_SPEC" "$(basename "$FEATURE_SPEC")"
    check_file "$IMPL_PLAN" "$(basename "$IMPL_PLAN")"

    if $INCLUDE_TASKS; then
        check_file "$TASKS" "$(basename "$TASKS")"
    fi

    check_dir "$FEATURE_DIR/checklists" "checklists/"
fi
