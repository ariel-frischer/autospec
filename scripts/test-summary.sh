#!/usr/bin/env bash
# test-summary.sh - Display test statistics summary
# Usage: ./scripts/test-summary.sh

set -u

echo "📊 Test Summary"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "Running tests and collecting stats..."
echo ""

# Run tests once and save to temp file
TMPFILE=$(mktemp)
trap 'rm -f "$TMPFILE"' EXIT
go test ./... -v -cover >"$TMPFILE" 2>&1 || true

# Count test results (grep -c exits 1 if no matches, so ignore exit code)
# Note: subtests have indented "--- PASS" lines, so we match with optional whitespace
# Use -- to prevent pattern being interpreted as option
TOTAL=$(grep -c "^=== RUN" "$TMPFILE") || TOTAL=0
PASSED=$(grep -c -- "--- PASS" "$TMPFILE") || PASSED=0
FAILED=$(grep -c -- "--- FAIL" "$TMPFILE") || FAILED=0
SKIPPED=$(grep -c -- "--- SKIP" "$TMPFILE") || SKIPPED=0
PKGS=$(go list ./... 2>/dev/null | wc -l | tr -d ' ')

# Count top-level vs subtests (subtests contain "/")
TOP_LEVEL=$(grep "^=== RUN" "$TMPFILE" | grep -cv "/") || TOP_LEVEL=0
SUBTESTS=$((TOTAL - TOP_LEVEL))

echo "  Total test runs:     $TOTAL"
echo "  ├─ Top-level tests:  $TOP_LEVEL"
echo "  └─ Subtests:         $SUBTESTS"
echo ""
echo "  ✅ Passed:           $PASSED"
echo "  ❌ Failed:           $FAILED"
echo "  ⏭️  Skipped:          $SKIPPED"
echo ""
echo "  📦 Packages:         $PKGS"
echo ""
echo "Coverage by package:"
grep -E "^ok.*coverage" "$TMPFILE" | head -15
echo ""
echo "(Run 'make test-cover' for detailed coverage report)"

# Exit with failure if any tests failed
[ "$FAILED" = "0" ] || exit 1
