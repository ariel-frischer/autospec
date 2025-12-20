#!/usr/bin/env bash
# test-summary.sh - Display test statistics summary
# Usage: ./scripts/test-summary.sh

set -uo pipefail

echo "📊 Test Summary"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "Running tests and collecting stats..."
echo ""

# Run tests once and capture output
OUTPUT=$(go test ./... -v -cover 2>&1) || true

# Count test results - use awk to get clean numbers
TOTAL=$(echo "$OUTPUT" | grep -c "^=== RUN" || true)
PASSED=$(echo "$OUTPUT" | grep -c "^--- PASS" || true)
FAILED=$(echo "$OUTPUT" | grep -c "^--- FAIL" || true)
SKIPPED=$(echo "$OUTPUT" | grep -c "^--- SKIP" || true)
PKGS=$(go list ./... 2>/dev/null | wc -l | awk '{print $1}')

# Count top-level vs subtests (subtests contain "/")
TOP_LEVEL=$(echo "$OUTPUT" | grep "^=== RUN" | grep -vc "/" || true)
SUBTESTS=$((TOTAL - TOP_LEVEL))

# Ensure we have valid numbers
TOTAL=${TOTAL:-0}
PASSED=${PASSED:-0}
FAILED=${FAILED:-0}
SKIPPED=${SKIPPED:-0}
TOP_LEVEL=${TOP_LEVEL:-0}

echo "  Total test runs:     $TOTAL"
echo "  ├─ Top-level tests:  $TOP_LEVEL"
echo "  └─ Subtests:         $SUBTESTS"
echo ""
echo "  ✅ Passed:           $PASSED"
echo "  ❌ Failed:           $FAILED"
if [ "$SKIPPED" -gt 0 ] 2>/dev/null; then
    echo "  ⏭️  Skipped:          $SKIPPED"
fi
echo ""
echo "  📦 Packages:         $PKGS"
echo ""
echo "Coverage by package:"
echo "$OUTPUT" | grep -E "^ok.*coverage" | head -15
echo ""
echo "(Run 'make test-cover' for detailed coverage report)"

# Exit with failure if any tests failed
if [ "$FAILED" -gt 0 ] 2>/dev/null; then
    exit 1
fi
