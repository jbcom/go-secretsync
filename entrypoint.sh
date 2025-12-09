#!/bin/sh
# Entrypoint script for SecretSync GitHub Action
# This script builds the command line from inputs and handles optional flags

set -e

# Start building the command with base
CMD="secretsync pipeline"

# Add config (always required)
if [ -n "$INPUT_CONFIG" ]; then
    CMD="$CMD --config \"$INPUT_CONFIG\""
fi

# Add optional targets
if [ -n "$INPUT_TARGETS" ]; then
    CMD="$CMD --targets \"$INPUT_TARGETS\""
fi

# Add boolean flags
if [ "$INPUT_DRY_RUN" = "true" ]; then
    CMD="$CMD --dry-run"
fi

if [ "$INPUT_MERGE_ONLY" = "true" ]; then
    CMD="$CMD --merge-only"
fi

if [ "$INPUT_SYNC_ONLY" = "true" ]; then
    CMD="$CMD --sync-only"
fi

if [ "$INPUT_DISCOVER" = "true" ]; then
    CMD="$CMD --discover"
fi

# Add output format (always present)
if [ -n "$INPUT_OUTPUT_FORMAT" ]; then
    CMD="$CMD --output \"$INPUT_OUTPUT_FORMAT\""
fi

# Add diff flag
if [ "$INPUT_COMPUTE_DIFF" = "true" ]; then
    CMD="$CMD --diff"
fi

# Add exit-code flag
if [ "$INPUT_EXIT_CODE" = "true" ]; then
    CMD="$CMD --exit-code"
fi

# Add log level (always present)
if [ -n "$INPUT_LOG_LEVEL" ]; then
    CMD="$CMD --log-level \"$INPUT_LOG_LEVEL\""
fi

# Add log format (always present)
if [ -n "$INPUT_LOG_FORMAT" ]; then
    CMD="$CMD --log-format \"$INPUT_LOG_FORMAT\""
fi

# Debug: Print the command if log level is debug
if [ "$INPUT_LOG_LEVEL" = "debug" ]; then
    echo "Executing: $CMD"
fi

# Execute the command using eval to properly handle quoted arguments
eval exec "$CMD"
