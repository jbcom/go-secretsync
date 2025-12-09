#!/bin/sh
# Entrypoint for SecretSync
# All configuration via environment variables - no action inputs needed
# This makes it work identically in GitHub Actions, GitLab CI, or local Docker

set -e

# Build command from environment variables
CMD="secretsync pipeline"

# Config file (required)
CONFIG="${SECRETSYNC_CONFIG:-config.yaml}"
CMD="$CMD --config \"$CONFIG\""

# Optional: specific targets
if [ -n "$SECRETSYNC_TARGETS" ]; then
    CMD="$CMD --targets \"$SECRETSYNC_TARGETS\""
fi

# Boolean flags
if [ "$SECRETSYNC_DRY_RUN" = "true" ]; then
    CMD="$CMD --dry-run"
fi

if [ "$SECRETSYNC_MERGE_ONLY" = "true" ]; then
    CMD="$CMD --merge-only"
fi

if [ "$SECRETSYNC_SYNC_ONLY" = "true" ]; then
    CMD="$CMD --sync-only"
fi

if [ "$SECRETSYNC_DISCOVER" = "true" ]; then
    CMD="$CMD --discover"
fi

if [ "$SECRETSYNC_DIFF" = "true" ]; then
    CMD="$CMD --diff"
fi

if [ "$SECRETSYNC_EXIT_CODE" = "true" ]; then
    CMD="$CMD --exit-code"
fi

# Output format (default: github for Actions, human otherwise)
OUTPUT="${SECRETSYNC_OUTPUT:-github}"
CMD="$CMD --output \"$OUTPUT\""

# Logging
LOG_LEVEL="${SECRETSYNC_LOG_LEVEL:-info}"
CMD="$CMD --log-level \"$LOG_LEVEL\""

LOG_FORMAT="${SECRETSYNC_LOG_FORMAT:-text}"
CMD="$CMD --log-format \"$LOG_FORMAT\""

# Debug mode - print command
if [ "$LOG_LEVEL" = "debug" ] || [ "$SECRETSYNC_DEBUG" = "true" ]; then
    echo "Executing: $CMD"
fi

# Execute
eval exec $CMD
