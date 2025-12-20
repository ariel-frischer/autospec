#!/bin/bash
# parallel-features.sh - Run 3 features in parallel

REPO_DIR=~/projects/myapp
FEATURES=(
    "auth:Add user authentication with OAuth"
    "profile:Add user profile with avatar upload"
    "search:Add full-text search functionality"
)

# Create worktrees
for feature in "${FEATURES[@]}"; do
    name="${feature%%:*}"
    git worktree add "${REPO_DIR}-${name}" -b "feature/${name}" 2>/dev/null || true
done

# Run autospec in parallel
pids=()
for feature in "${FEATURES[@]}"; do
    name="${feature%%:*}"
    desc="${feature#*:}"
    (
        cd "${REPO_DIR}-${name}" || exit 1
        autospec run -a "${desc}"
    ) &
    pids+=($!)
done

# Wait for all to complete
for pid in "${pids[@]}"; do
    wait $pid
done

echo "All features complete!"
