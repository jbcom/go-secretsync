#!/bin/sh
# Entrypoint for SecretSync
# All configuration via environment variables - no action inputs needed
# This makes it work identically in GitHub Actions, GitLab CI, or local Docker

set -e

# Build argument list using positional parameters for safe execution
# This prevents command injection even with malicious environment variables
set -- pipeline

# Config file (required)
CONFIG="${SECRETSYNC_CONFIG:-config.yaml}"
set -- "$@" --config "$CONFIG"

# Optional: specific targets
if [ -n "$SECRETSYNC_TARGETS" ]; then
    set -- "$@" --targets "$SECRETSYNC_TARGETS"
fi

# Boolean flags
if [ "$SECRETSYNC_DRY_RUN" = "true" ]; then
    set -- "$@" --dry-run
fi

if [ "$SECRETSYNC_MERGE_ONLY" = "true" ]; then
    set -- "$@" --merge-only
fi

if [ "$SECRETSYNC_SYNC_ONLY" = "true" ]; then
    set -- "$@" --sync-only
fi

if [ "$SECRETSYNC_DISCOVER" = "true" ]; then
    set -- "$@" --discover
fi

if [ "$SECRETSYNC_DIFF" = "true" ]; then
    set -- "$@" --diff
fi

if [ "$SECRETSYNC_EXIT_CODE" = "true" ]; then
    set -- "$@" --exit-code
fi

# Output format (default: github for Actions, human otherwise)
OUTPUT="${SECRETSYNC_OUTPUT:-github}"
set -- "$@" --output "$OUTPUT"

# Logging
LOG_LEVEL="${SECRETSYNC_LOG_LEVEL:-info}"
set -- "$@" --log-level "$LOG_LEVEL"

LOG_FORMAT="${SECRETSYNC_LOG_FORMAT:-text}"
set -- "$@" --log-format "$LOG_FORMAT"

# Debug mode - print command
if [ "$LOG_LEVEL" = "debug" ] || [ "$SECRETSYNC_DEBUG" = "true" ]; then
    echo "Executing: secretsync $*"
fi

# Execute with properly quoted arguments to prevent command injection
exec secretsync "$@"
