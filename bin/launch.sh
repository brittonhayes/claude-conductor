#!/usr/bin/env bash

# Launcher script for Claude Code orchestration
# Creates tmux session with orchestrator + worker panes

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Default configuration
NUM_WORKERS=3
SESSION_NAME="claude-conductor"
USE_WORKTREES=false
BASE_DIR="$HOME/claude-work"

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    -w|--workers)
      NUM_WORKERS="$2"
      shift 2
      ;;
    -n|--name)
      SESSION_NAME="$2"
      shift 2
      ;;
    --worktrees)
      USE_WORKTREES=true
      shift
      ;;
    -d|--dir)
      BASE_DIR="$2"
      shift 2
      ;;
    -h|--help)
      cat <<EOF
Usage: $0 [options]

Launch Claude Code orchestration system with tmux.

Options:
  -w, --workers NUM     Number of worker sessions (default: 3)
  -n, --name NAME       Tmux session name (default: claude-conductor)
  --worktrees           Use git worktrees for workers
  -d, --dir DIR         Base directory for workers (default: ~/claude-work)
  -h, --help            Show this help message

Examples:
  # Launch with 3 workers in current directory
  $0

  # Launch with 4 workers using worktrees
  $0 --workers 4 --worktrees

  # Custom session name and directory
  $0 --name my-session --dir ~/projects/myapp

Layout:
  ┌─────────────────────────────────┬──────────────────────────────────┐
  │                                 │                                  │
  │        Orchestrator             │          Worker 0                │
  │      (Interactive)              │                                  │
  │                                 │                                  │
  ├─────────────────────────────────┼──────────────────────────────────┤
  │                                 │                                  │
  │         Worker 1                │          Worker 2                │
  │                                 │                                  │
  │                                 │                                  │
  └─────────────────────────────────┴──────────────────────────────────┘

EOF
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      echo "Run with --help for usage information" >&2
      exit 1
      ;;
  esac
done

# Check if tmux is installed
if ! command -v tmux >/dev/null 2>&1; then
  echo "Error: tmux is not installed" >&2
  echo "Install with: apt-get install tmux (Linux) or brew install tmux (macOS)" >&2
  exit 1
fi

# Check if session already exists
if tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
  echo "Session '$SESSION_NAME' already exists" >&2
  echo "Attach with: tmux attach -t $SESSION_NAME" >&2
  echo "Or kill existing: tmux kill-session -t $SESSION_NAME" >&2
  exit 1
fi

# Setup worktrees if requested
setup_worktrees() {
  if ! git rev-parse --git-dir >/dev/null 2>&1; then
    echo "Error: --worktrees requires a git repository" >&2
    exit 1
  fi

  local current_branch=$(git rev-parse --abbrev-ref HEAD)
  mkdir -p "$BASE_DIR"

  for i in $(seq 0 $((NUM_WORKERS - 1))); do
    local worktree_dir="$BASE_DIR/worker-$i"

    if [ ! -d "$worktree_dir" ]; then
      echo "Creating worktree for worker $i at $worktree_dir"
      git worktree add "$worktree_dir" "$current_branch" 2>/dev/null || {
        # Worktree might already exist, try to use it
        if [ -d "$worktree_dir/.git" ]; then
          echo "Using existing worktree at $worktree_dir"
        else
          echo "Error: Failed to create worktree at $worktree_dir" >&2
          exit 1
        fi
      }
    else
      echo "Using existing worktree at $worktree_dir"
    fi
  done
}

# Get worker directory
get_worker_dir() {
  local worker_id=$1

  if [ "$USE_WORKTREES" = true ]; then
    echo "$BASE_DIR/worker-$worker_id"
  else
    echo "$PWD"
  fi
}

# Setup worktrees if needed
if [ "$USE_WORKTREES" = true ]; then
  setup_worktrees
fi

echo "Creating tmux session: $SESSION_NAME"
echo "Workers: $NUM_WORKERS"
echo ""

# Create new tmux session (detached)
tmux new-session -d -s "$SESSION_NAME" -n "orchestrator"

# Set up orchestrator pane (pane 0)
tmux send-keys -t "$SESSION_NAME:0.0" "cd '$PWD'" C-m
tmux send-keys -t "$SESSION_NAME:0.0" "echo 'Orchestrator - Interactive Mode'" C-m
tmux send-keys -t "$SESSION_NAME:0.0" "echo 'Waiting for workers to start...'" C-m
tmux send-keys -t "$SESSION_NAME:0.0" "sleep 2" C-m

# Create worker panes
case $NUM_WORKERS in
  1)
    # Simple vertical split
    tmux split-window -h -t "$SESSION_NAME:0"
    ;;
  2)
    # Vertical split
    tmux split-window -h -t "$SESSION_NAME:0"
    tmux split-window -h -t "$SESSION_NAME:0"
    tmux select-layout -t "$SESSION_NAME:0" even-horizontal
    ;;
  3)
    # Orchestrator on left, 2 workers stacked on right
    tmux split-window -h -t "$SESSION_NAME:0"
    tmux split-window -v -t "$SESSION_NAME:0.1"
    tmux split-window -v -t "$SESSION_NAME:0.2"
    ;;
  4)
    # 2x2 grid
    tmux split-window -h -t "$SESSION_NAME:0"
    tmux split-window -v -t "$SESSION_NAME:0.0"
    tmux split-window -v -t "$SESSION_NAME:0.2"
    ;;
  *)
    # For more workers, create a grid
    tmux split-window -h -t "$SESSION_NAME:0"

    # Create remaining workers in the right pane
    for i in $(seq 1 $((NUM_WORKERS - 1))); do
      tmux split-window -v -t "$SESSION_NAME:0.$i"
    done

    tmux select-layout -t "$SESSION_NAME:0" tiled
    ;;
esac

# Start workers in their panes
for i in $(seq 0 $((NUM_WORKERS - 1))); do
  local pane_idx=$((i + 1))
  local worker_dir=$(get_worker_dir $i)

  tmux send-keys -t "$SESSION_NAME:0.$pane_idx" "cd '$worker_dir'" C-m
  tmux send-keys -t "$SESSION_NAME:0.$pane_idx" "echo 'Worker $i - Starting...'" C-m
  tmux send-keys -t "$SESSION_NAME:0.$pane_idx" "'$SCRIPT_DIR/worker.sh' $i '$worker_dir'" C-m
done

# Start orchestrator last (after workers are ready)
tmux send-keys -t "$SESSION_NAME:0.0" "'$SCRIPT_DIR/orchestrator.js' $NUM_WORKERS --interactive" C-m

# Select orchestrator pane
tmux select-pane -t "$SESSION_NAME:0.0"

echo ""
echo "✓ Tmux session '$SESSION_NAME' created"
echo ""
echo "Attach with:"
echo "  tmux attach -t $SESSION_NAME"
echo ""
echo "Detach with:"
echo "  Ctrl-b d"
echo ""
echo "Kill session:"
echo "  tmux kill-session -t $SESSION_NAME"
echo ""

# Auto-attach if running interactively
if [ -t 0 ]; then
  echo "Attaching to session in 2 seconds..."
  sleep 2
  tmux attach -t "$SESSION_NAME"
fi
