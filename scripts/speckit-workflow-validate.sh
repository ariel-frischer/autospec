#!/bin/bash
# SpecKit Workflow with Validation and Retry Logic
# Extends speckit-workflow.sh with automatic validation and retry

set -euo pipefail

# Source validation library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/speckit-validation-lib.sh"

# Configuration
RETRY_LIMIT="${SPECKIT_RETRY_LIMIT:-2}"
DRY_RUN="${SPECKIT_DRY_RUN:-false}"
VERBOSE="${SPECKIT_DEBUG:-false}"
SKIP_CONSTITUTION=false

# Find git root directory
GIT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null)
if [ -z "$GIT_ROOT" ]; then
    log_error "Not in a git repository"
    exit "$EXIT_MISSING_DEPS"
fi

# Change to git root
cd "$GIT_ROOT" || exit "$EXIT_VALIDATION_FAILED"

# ------------------------------------------------------------------------------
# Argument Parsing
# ------------------------------------------------------------------------------

show_help() {
    cat <<EOF
Usage: $0 <feature-description> [spec-title] [options]

Arguments:
  feature-description    What you want to specify/build (required)
  spec-title            Title for the spec (optional, defaults to feature-description)

Options:
  --retry-limit N       Maximum retry attempts per command (default: 2)
  --dry-run            Show what would be validated without executing
  --verbose            Enable detailed logging
  --skip-constitution  Skip constitution creation if missing
  --help               Show this help message

Environment Variables:
  SPECKIT_RETRY_LIMIT   Override default retry limit
  SPECKIT_DRY_RUN       Set to "true" for dry-run mode
  SPECKIT_DEBUG         Set to "true" for verbose logging
  ANTHROPIC_API_KEY     API key for Claude (can be empty for local auth)

Examples:
  # Standard workflow with validation
  $0 "Add Mastodon support"

  # Custom retry limit
  $0 "Add Mastodon support" --retry-limit 3

  # Dry run to see validation plan
  $0 "Add Mastodon support" --dry-run

Exit Codes:
  0 - All commands completed successfully
  1 - Validation failed, retries exhausted
  3 - Invalid arguments
  4 - Missing dependencies
EOF
}

# Parse arguments
FEATURE_DESC=""
SPEC_TITLE=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --help|-h)
            show_help
            exit 0
            ;;
        --retry-limit)
            RETRY_LIMIT="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --verbose)
            SPECKIT_DEBUG=true
            export SPECKIT_DEBUG
            shift
            ;;
        --skip-constitution)
            SKIP_CONSTITUTION=true
            shift
            ;;
        --*)
            log_error "Unknown option: $1"
            show_help
            exit "$EXIT_INVALID_ARGS"
            ;;
        *)
            if [ -z "$FEATURE_DESC" ]; then
                FEATURE_DESC="$1"
            elif [ -z "$SPEC_TITLE" ]; then
                SPEC_TITLE="$1"
            else
                log_error "Too many arguments"
                show_help
                exit "$EXIT_INVALID_ARGS"
            fi
            shift
            ;;
    esac
done

# Validate required arguments
if [ -z "$FEATURE_DESC" ]; then
    log_error "Feature description required"
    show_help
    exit "$EXIT_INVALID_ARGS"
fi

# Default spec title to feature description
SPEC_TITLE="${SPEC_TITLE:-$FEATURE_DESC}"

# Check dependencies
check_dependencies git claude jq grep sed || exit "$EXIT_MISSING_DEPS"

# ------------------------------------------------------------------------------
# Helper Functions
# ------------------------------------------------------------------------------

# Run a speckit command with validation and retry logic
# Usage: run_with_validation <command_type> <claude_command> <expected_file> <spec_name>
run_with_validation() {
    local cmd_type="$1"
    local claude_cmd="$2"
    local expected_file="$3"
    local spec_name="$4"

    log_debug "Running command: $cmd_type"
    log_debug "Expected file: $expected_file"
    log_debug "Spec name: $spec_name"

    # Find spec directory (may have number prefix like 002-)
    local spec_dir=""
    if [ -n "$spec_name" ]; then
        spec_dir=$(find "$SPECKIT_SPECS_DIR" -maxdepth 1 -type d -name "*${spec_name}*" | head -1)
    fi

    # IDEMPOTENCY CHECK: Skip if artifact already exists
    if [ -n "$spec_dir" ] && validate_file_exists "$expected_file" "$spec_dir"; then
        log_info "✓ Skipping $cmd_type: $expected_file already exists in $spec_dir"
        reset_retry_count "$spec_name" "$cmd_type"
        return "$EXIT_SUCCESS"
    fi

    # Get current retry count
    local retry_count
    retry_count=$(get_retry_count "$spec_name" "$cmd_type")

    # Check if retry limit exceeded
    if [ "$retry_count" -ge "$RETRY_LIMIT" ]; then
        log_error "Retry limit ($RETRY_LIMIT) exceeded for $cmd_type"
        log_error "Cannot proceed with workflow"
        return "$EXIT_RETRY_EXHAUSTED"
    fi

    # Dry run mode
    if [ "$DRY_RUN" = "true" ]; then
        echo "[DRY RUN] Would execute: $claude_cmd"
        echo "[DRY RUN] Would validate: $expected_file exists in specs/$spec_name"
        return "$EXIT_SUCCESS"
    fi

    # Execute Claude command
    log_info "Executing: $cmd_type..."
    ANTHROPIC_API_KEY="" claude -p "$claude_cmd" \
        --dangerously-skip-permissions \
        --verbose \
        --output-format stream-json | claude-clean

    if [ $? -ne 0 ]; then
        log_error "Claude command failed: $cmd_type"
        return "$EXIT_VALIDATION_FAILED"
    fi

    # Re-find spec directory after command execution (may have been created)
    if [ -z "$spec_dir" ] && [ -n "$spec_name" ]; then
        spec_dir=$(find "$SPECKIT_SPECS_DIR" -maxdepth 1 -type d -name "*${spec_name}*" | head -1)
    fi

    if [ -z "$spec_dir" ]; then
        log_error "Spec directory not found for: $spec_name"
        log_error "Looked in: $SPECKIT_SPECS_DIR"
        return "$EXIT_VALIDATION_FAILED"
    fi

    log_debug "Found spec directory: $spec_dir"

    # Validate expected file exists
    if validate_file_exists "$expected_file" "$spec_dir"; then
        log_info "✓ Validation: $expected_file created successfully"
        reset_retry_count "$spec_name" "$cmd_type"
        return "$EXIT_SUCCESS"
    else
        log_error "✗ Validation: $expected_file missing (attempt $((retry_count + 1))/$RETRY_LIMIT)"

        # Increment retry counter
        local new_count
        new_count=$(increment_retry_count "$spec_name" "$cmd_type")

        if [ "$new_count" -ge "$RETRY_LIMIT" ]; then
            log_error "Retry limit reached. Aborting workflow."
            return "$EXIT_RETRY_EXHAUSTED"
        fi

        log_info "Retrying $cmd_type..."
        # Recursive retry
        run_with_validation "$cmd_type" "$claude_cmd" "$expected_file" "$spec_name"
        return $?
    fi
}

# ------------------------------------------------------------------------------
# Main Workflow Execution
# ------------------------------------------------------------------------------

log_info "Running SpecKit workflow with validation..."
log_info "Feature: $FEATURE_DESC"
log_info "Spec Title: $SPEC_TITLE"
log_info "Retry limit: $RETRY_LIMIT"
echo ""

# Detect current spec from git branch or recent activity
DETECTED_SPEC=$(detect_current_spec)
if [ -n "$DETECTED_SPEC" ]; then
    log_info "Detected active spec: $DETECTED_SPEC"
    SPEC_NAME="$DETECTED_SPEC"
else
    log_debug "No existing spec detected, will use newly created spec"
    SPEC_NAME=""
fi

# Step 0: Constitution (optional)
if [ "$SKIP_CONSTITUTION" = "false" ] && [ ! -f ".specify/memory/constitution.md" ]; then
    log_info "Step 0/3: Constitution not found. Creating project principles..."

    if [ "$DRY_RUN" = "false" ]; then
        ANTHROPIC_API_KEY="" claude -p "/speckit.constitution - Establish project principles" \
            --dangerously-skip-permissions \
            --verbose \
            --output-format stream-json | claude-clean

        if [ $? -ne 0 ]; then
            log_error "speckit.constitution failed"
            exit "$EXIT_VALIDATION_FAILED"
        fi
    else
        echo "[DRY RUN] Would create constitution"
    fi
    echo ""
fi

# Step 1: Specify
log_info "Step 1/3: Creating specification..."
if ! run_with_validation \
    "specify" \
    "/speckit.specify $FEATURE_DESC" \
    "spec.md" \
    "$SPEC_NAME"; then
    log_error "Failed to create specification after $RETRY_LIMIT attempts"
    exit "$EXIT_RETRY_EXHAUSTED"
fi
echo ""

# Re-detect spec after creation (in case it was just created)
SPEC_NAME=$(detect_current_spec)
if [ -z "$SPEC_NAME" ]; then
    log_error "Could not detect spec after creation"
    exit "$EXIT_VALIDATION_FAILED"
fi
log_debug "Using spec: $SPEC_NAME"

# Step 2: Plan
log_info "Step 2/3: Creating implementation plan for '$SPEC_TITLE'..."
if ! run_with_validation \
    "plan" \
    "/speckit.plan (for spec: $SPEC_TITLE)" \
    "plan.md" \
    "$SPEC_NAME"; then
    log_error "Failed to create plan after $RETRY_LIMIT attempts"
    exit "$EXIT_RETRY_EXHAUSTED"
fi
echo ""

# Step 3: Tasks
log_info "Step 3/3: Generating actionable tasks for '$SPEC_TITLE'..."
if ! run_with_validation \
    "tasks" \
    "/speckit.tasks (for spec: $SPEC_TITLE)" \
    "tasks.md" \
    "$SPEC_NAME"; then
    log_error "Failed to generate tasks after $RETRY_LIMIT attempts"
    exit "$EXIT_RETRY_EXHAUSTED"
fi
echo ""

# Success
log_info "✓ SpecKit workflow completed successfully!"
log_info "You can now run /speckit.implement to execute the tasks"

exit "$EXIT_SUCCESS"
