#!/bin/bash
# Sync docs/ to site/ with Jekyll frontmatter
# This script generates site/ pages from docs/ to avoid duplication
#
# Usage: ./scripts/sync-docs-to-site.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DOCS_DIR="$REPO_ROOT/docs"
SITE_DIR="$REPO_ROOT/site"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Generate a site doc from a docs/ source file
# Arguments: source_file dest_file title parent nav_order [mermaid] [description]
generate_doc() {
    local src="$1"
    local dest="$2"
    local title="$3"
    local parent="$4"
    local nav_order="$5"
    local mermaid="${6:-false}"
    local description="${7:-}"

    if [[ ! -f "$src" ]]; then
        log_error "Source file not found: $src"
        return 1
    fi

    # Create destination directory if needed
    mkdir -p "$(dirname "$dest")"

    # Write frontmatter
    if [[ "$mermaid" == "true" ]]; then
        cat > "$dest" << EOF
---
title: $title
parent: $parent
nav_order: $nav_order
mermaid: true
---

# $title
EOF
    else
        cat > "$dest" << EOF
---
title: $title
parent: $parent
nav_order: $nav_order
---

# $title
EOF
    fi

    # Add description if provided
    if [[ -n "$description" ]]; then
        echo "" >> "$dest"
        echo "$description" >> "$dest"
        echo "{: .fs-6 .fw-300 }" >> "$dest"
    fi

    # Append source content, skipping first line if it's a heading
    local first_line
    first_line=$(head -1 "$src")
    if [[ "$first_line" =~ ^#[[:space:]] ]]; then
        # Skip the first heading line since Jekyll will use the title
        tail -n +2 "$src" >> "$dest"
    else
        cat "$src" >> "$dest"
    fi

    log_info "Generated: $dest"
}

# Main sync function
main() {
    log_info "Syncing docs/ to site/..."
    echo ""

    # Reference docs (parent: Reference)
    log_info "Syncing Reference docs..."

    generate_doc \
        "$DOCS_DIR/public/agents.md" \
        "$SITE_DIR/reference/agents.md" \
        "Agent Configuration" \
        "Reference" \
        4

    generate_doc \
        "$DOCS_DIR/public/claude-settings.md" \
        "$SITE_DIR/reference/claude-settings.md" \
        "Claude Settings" \
        "Reference" \
        5

    generate_doc \
        "$DOCS_DIR/public/codex-settings.md" \
        "$SITE_DIR/reference/codex-settings.md" \
        "Codex Settings" \
        "Reference" \
        6

    generate_doc \
        "$DOCS_DIR/public/SHELL-COMPLETION.md" \
        "$SITE_DIR/reference/shell-completion.md" \
        "Shell Completion" \
        "Reference" \
        6

    generate_doc \
        "$DOCS_DIR/public/TIMEOUT.md" \
        "$SITE_DIR/reference/timeout.md" \
        "Timeout Configuration" \
        "Reference" \
        7

    generate_doc \
        "$DOCS_DIR/public/opencode-settings.md" \
        "$SITE_DIR/reference/opencode-settings.md" \
        "OpenCode Settings" \
        "Reference" \
        8

    generate_doc \
        "$DOCS_DIR/public/render-command.md" \
        "$SITE_DIR/reference/render-command.md" \
        "Render Command" \
        "Reference" \
        9

    # Guide docs (parent: Guides)
    echo ""
    log_info "Syncing Guide docs..."

    generate_doc \
        "$DOCS_DIR/public/task-sizing.md" \
        "$SITE_DIR/guides/task-sizing.md" \
        "Task Sizing" \
        "Guides" \
        5

    generate_doc \
        "$DOCS_DIR/public/checklists.md" \
        "$SITE_DIR/guides/checklists.md" \
        "Checklists" \
        "Guides" \
        6

    generate_doc \
        "$DOCS_DIR/public/self-update.md" \
        "$SITE_DIR/guides/self-update.md" \
        "Self-Update" \
        "Guides" \
        7

    generate_doc \
        "$DOCS_DIR/public/PREREQUISITES.md" \
        "$SITE_DIR/guides/prerequisites.md" \
        "Prerequisites" \
        "Guides" \
        1

    # Contributing docs (parent: Contributing)
    echo ""
    log_info "Syncing Contributing docs..."

    generate_doc \
        "$DOCS_DIR/internal/architecture.md" \
        "$SITE_DIR/contributing/architecture.md" \
        "Architecture" \
        "Contributing" \
        1 \
        "true"

    generate_doc \
        "$DOCS_DIR/internal/go-best-practices.md" \
        "$SITE_DIR/contributing/go-best-practices.md" \
        "Go Best Practices" \
        "Contributing" \
        2

    generate_doc \
        "$DOCS_DIR/internal/internals.md" \
        "$SITE_DIR/contributing/internals.md" \
        "Internals" \
        "Contributing" \
        3

    generate_doc \
        "$DOCS_DIR/internal/testing-mocks.md" \
        "$SITE_DIR/contributing/testing-mocks.md" \
        "Testing & Mocks" \
        "Contributing" \
        4

    generate_doc \
        "$DOCS_DIR/internal/events.md" \
        "$SITE_DIR/contributing/events.md" \
        "Events System" \
        "Contributing" \
        5

    generate_doc \
        "$DOCS_DIR/internal/YAML-STRUCTURED-OUTPUT.md" \
        "$SITE_DIR/contributing/yaml-schemas.md" \
        "YAML Schemas" \
        "Contributing" \
        6

    generate_doc \
        "$DOCS_DIR/internal/risks.md" \
        "$SITE_DIR/contributing/risks.md" \
        "Risks" \
        "Contributing" \
        7

    echo ""
    log_info "Sync complete!"
    echo ""
    log_info "Generated docs from docs/ to site/"
    log_info "Source of truth: docs/"
    log_info "Site pages: site/{reference,guides,contributing}/"
}

main "$@"
