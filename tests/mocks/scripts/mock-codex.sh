#!/bin/bash
# mock-codex.sh - Records Codex argv, then delegates deterministic artifact output.

set -euo pipefail

if [[ -n "${MOCK_CALL_LOG:-}" ]]; then
    {
        echo "---"
        echo "agent: codex"
        echo "args:"
        for arg in "$@"; do
            escaped_arg=$(echo "$arg" | sed 's/"/\\"/g')
            echo "  - \"${escaped_arg}\""
        done
    } >> "${MOCK_CALL_LOG}"
fi

MOCK_CALL_LOG="" exec "${0%/*}/claude" "$@"
