#!/bin/bash
# mock-jcode.sh - Records jcode argv, then delegates deterministic artifact output.
#
# jcode exec mode is invoked as `jcode [global flags] run <prompt>`; the prompt
# (last argument) is forwarded to the Claude mock for artifact generation.

set -euo pipefail

if [[ -n "${MOCK_CALL_LOG:-}" ]]; then
    {
        echo "---"
        echo "agent: jcode"
        echo "args:"
        for arg in "$@"; do
            escaped_arg=$(echo "$arg" | sed 's/"/\\"/g')
            echo "  - \"${escaped_arg}\""
        done
    } >> "${MOCK_CALL_LOG}"
fi

if [[ $# -eq 0 ]]; then
    exit "${MOCK_EXIT_CODE:-0}"
fi

MOCK_CALL_LOG="" exec "${0%/*}/claude" "${@: -1}"
