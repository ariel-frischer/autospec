#!/usr/bin/env bash
# Extract changelog section for a specific version from CHANGELOG.md
# Usage: ./extract-changelog.sh [version]
# If no version provided, extracts from git tag or defaults to Unreleased

set -euo pipefail

CHANGELOG_FILE="${CHANGELOG_FILE:-CHANGELOG.md}"
OUTPUT_FILE="${OUTPUT_FILE:-.release/notes.md}"

# Get version from argument, git tag, or default to Unreleased
if [[ -n "${1:-}" ]]; then
    VERSION="$1"
elif [[ -n "${GITHUB_REF:-}" ]] && [[ "$GITHUB_REF" == refs/tags/v* ]]; then
    VERSION="${GITHUB_REF#refs/tags/v}"
else
    VERSION="Unreleased"
fi

echo "Extracting changelog for version: $VERSION" >&2

# Extract the section between this version header and the next version header
# Handles both [X.Y.Z] and [Unreleased] formats
if [[ "$VERSION" == "Unreleased" ]]; then
    PATTERN="## [Unreleased]"
else
    PATTERN="## [${VERSION}]"
fi

# Extract content: from version header to next ## header (exclusive)
# Using string comparison instead of regex to avoid escaping issues
awk -v pattern="$PATTERN" '
    index($0, pattern) == 1 { found=1; next }
    found && /^## \[/ { exit }
    found { print }
' "$CHANGELOG_FILE" | sed -e '/^$/N;/^\n$/d' -e '1{/^$/d}' > "$OUTPUT_FILE"

# Check if we got content
if [[ ! -s "$OUTPUT_FILE" ]]; then
    echo "Warning: No changelog content found for version $VERSION" >&2
    echo "No changelog entry for this version." > "$OUTPUT_FILE"
fi

echo "Release notes written to: $OUTPUT_FILE" >&2
cat "$OUTPUT_FILE"
