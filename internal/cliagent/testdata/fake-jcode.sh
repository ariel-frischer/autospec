#!/bin/sh
set -eu

log_file="${JCODE_FIXTURE_LOG:-}"
exit_code="${JCODE_FIXTURE_EXIT:-0}"

if [ -n "$log_file" ]; then
  {
    printf '%s\n' '---'
    printf 'cwd: %s\n' "$(pwd)"
    printf 'env_fixture: %s\n' "${JCODE_FIXTURE_ENV:-}"
    printf 'args:\n'
    for arg in "$@"; do
      printf '  - %s\n' "$arg"
    done
  } >> "$log_file"
fi

if [ "$exit_code" -ne 0 ]; then
  printf 'fixture jcode failure (exit %s)\n' "$exit_code" >&2
fi
exit "$exit_code"
