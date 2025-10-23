#!/usr/bin/env bash
# Master validation script for GitHub issue templates
# Runs all validations and reports results

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Load validation library
# shellcheck source=../lib/validation_lib.sh
source "$SCRIPT_DIR/../lib/validation_lib.sh"

# Template directory
TEMPLATE_DIR="$PROJECT_ROOT/.github/ISSUE_TEMPLATE"

echo "GitHub Issue Templates Validation"
echo "=================================="
echo ""

# Run all validations
if validate_all_templates "$TEMPLATE_DIR"; then
  echo ""
  echo "=================================="
  echo "✓ All validations passed"
  exit 0
else
  echo ""
  echo "=================================="
  echo "✗ Validation failed"
  exit 1
fi
