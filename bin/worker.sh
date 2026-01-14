#!/usr/bin/env bash

# Worker wrapper script for Claude Code orchestration
# Watches for task.json, pipes prompts to Claude Code, writes results

set -euo pipefail

WORKER_ID="${1:-}"
WORKTREE_PATH="${2:-.}"

if [ -z "$WORKER_ID" ]; then
  echo "Usage: $0 <worker_id> [worktree_path]" >&2
  exit 1
fi

QUEUE_DIR="$HOME/.claude-code/orchestrator/workers/$WORKER_ID"
TASK_FILE="$QUEUE_DIR/task.json"
STATUS_FILE="$QUEUE_DIR/status.json"
RESULT_FILE="$QUEUE_DIR/result.json"
LOCK_FILE="$QUEUE_DIR/.lock"

# Ensure worker directory exists
mkdir -p "$QUEUE_DIR"

# Initialize status
echo "{\"status\":\"idle\",\"timestamp\":\"$(date -Iseconds)\"}" > "$STATUS_FILE"

log() {
  echo "[Worker $WORKER_ID] $*" >&2
}

# Acquire lock with PID
acquire_lock() {
  local retries=5
  local wait=1

  while [ $retries -gt 0 ]; do
    if mkdir "$LOCK_FILE" 2>/dev/null; then
      echo "$$" > "$LOCK_FILE/pid"
      return 0
    fi

    # Check if lock is stale
    if [ -f "$LOCK_FILE/pid" ]; then
      local old_pid=$(cat "$LOCK_FILE/pid")
      if ! kill -0 "$old_pid" 2>/dev/null; then
        log "Removing stale lock from PID $old_pid"
        rm -rf "$LOCK_FILE"
        continue
      fi
    fi

    sleep "$wait"
    retries=$((retries - 1))
    wait=$((wait * 2))
  done

  return 1
}

# Release lock
release_lock() {
  rm -rf "$LOCK_FILE"
}

# Cleanup on exit
cleanup() {
  release_lock 2>/dev/null || true
  log "Shutting down"
}

trap cleanup EXIT INT TERM

# Process a task
process_task() {
  local task_file="$1"

  if ! acquire_lock; then
    log "Failed to acquire lock"
    return 1
  fi

  # Read task
  if [ ! -f "$task_file" ]; then
    release_lock
    return 0
  fi

  local task_id=$(jq -r '.id // empty' "$task_file")
  local prompt=$(jq -r '.prompt // empty' "$task_file")

  if [ -z "$prompt" ]; then
    log "Invalid task file, skipping"
    rm -f "$task_file"
    release_lock
    return 0
  fi

  log "Processing task $task_id"

  # Update status to working
  echo "{\"status\":\"working\",\"taskId\":$task_id,\"timestamp\":\"$(date -Iseconds)\"}" > "$STATUS_FILE"

  # Execute task with Claude Code
  local output_file=$(mktemp)
  local error_file=$(mktemp)
  local success=true

  # Change to worktree directory
  cd "$WORKTREE_PATH"

  # Pipe prompt to Claude Code and capture output
  if echo "$prompt" | claude-code 2>"$error_file" 1>"$output_file"; then
    success=true
  else
    success=false
  fi

  # Read output
  local output=$(cat "$output_file")
  local errors=$(cat "$error_file")

  # Write result
  jq -n \
    --arg taskId "$task_id" \
    --arg output "$output" \
    --arg errors "$errors" \
    --argjson success "$success" \
    '{
      taskId: $taskId,
      output: $output,
      success: $success,
      error: (if $errors != "" then $errors else null end),
      timestamp: (now | todate)
    }' > "$RESULT_FILE"

  # Update status
  if [ "$success" = "true" ]; then
    echo "{\"status\":\"done\",\"taskId\":$task_id,\"timestamp\":\"$(date -Iseconds)\"}" > "$STATUS_FILE"
    log "Task $task_id completed successfully"
  else
    echo "{\"status\":\"error\",\"taskId\":$task_id,\"timestamp\":\"$(date -Iseconds)\"}" > "$STATUS_FILE"
    log "Task $task_id failed: $errors"
  fi

  # Cleanup
  rm -f "$output_file" "$error_file" "$task_file"
  release_lock
}

# Main loop - watch for tasks
log "Starting worker in $WORKTREE_PATH"

if command -v inotifywait >/dev/null 2>&1; then
  # Use inotify for efficient file watching (Linux)
  log "Using inotify for file watching"

  while true; do
    # Process any existing task
    if [ -f "$TASK_FILE" ]; then
      process_task "$TASK_FILE"
    fi

    # Wait for next task
    inotifywait -q -e create,modify "$QUEUE_DIR" 2>/dev/null || true

    # Small delay to let file writes complete
    sleep 0.1
  done
elif command -v fswatch >/dev/null 2>&1; then
  # Use fswatch for macOS
  log "Using fswatch for file watching"

  # Process any existing task first
  if [ -f "$TASK_FILE" ]; then
    process_task "$TASK_FILE"
  fi

  # Watch for changes
  fswatch -0 "$QUEUE_DIR" | while read -d "" event; do
    if [[ "$event" == *"task.json"* ]]; then
      sleep 0.1  # Let file write complete
      process_task "$TASK_FILE"
    fi
  done
else
  # Fallback to polling
  log "Using polling for file watching (install inotify-tools or fswatch for better performance)"

  while true; do
    if [ -f "$TASK_FILE" ]; then
      process_task "$TASK_FILE"
    fi
    sleep 1
  done
fi
