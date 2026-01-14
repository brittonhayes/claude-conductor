# Claude Conductor

Launch multiple Claude Code sessions in tmux. One command, N parallel tasks.

## Installation

```bash
git clone https://github.com/brittonhayes/claude-conductor
cd claude-conductor
./bin/launch -h
```

## Usage

### Basic - From Command Line

```bash
./bin/launch "Review auth.py for bugs" "Run all tests" "Check for TODOs"
```

Creates a tmux session with 3 panes, each running a Claude Code session with the given prompt.

### From File

```bash
cat > tasks.txt <<EOF
Review all Python files for security issues
Run the test suite and fix any failures
Update documentation for new API endpoints
EOF

./bin/launch -f tasks.txt
```

### From Stdin

```bash
echo "Explain how the authentication system works" | ./bin/launch -f -
```

### With Git Worktrees

Isolate changes from each task in separate worktrees:

```bash
./bin/launch -w \
  "Refactor auth module" \
  "Add new API endpoint" \
  "Fix bug #123"
```

Each task runs in `~/.conductor-work/task-N/` with an isolated git worktree.

## Options

```
-f FILE    Read tasks from file (one per line, or - for stdin)
-n NAME    Session name (default: conductor)
-w         Use git worktrees (isolate changes per task)
-d DIR     Work directory for worktrees (default: ~/.conductor-work)
-h         Show help
```

## Examples

**Parallel code review:**
```bash
./bin/launch -w \
  "Review pkg/auth/*.go for security issues" \
  "Review pkg/api/*.go for error handling" \
  "Review pkg/db/*.go for SQL injection risks"
```

**Test different modules:**
```bash
./bin/launch \
  "Run unit tests for auth package" \
  "Run integration tests for API" \
  "Run e2e tests"
```

**Research tasks:**
```bash
./bin/launch \
  "Find all TODO comments and summarize them" \
  "List all external dependencies and their versions" \
  "Check for outdated npm packages"
```

## How It Works

1. Creates a tmux session with N panes
2. Optionally sets up git worktrees (if `-w` specified)
3. Pipes each task prompt to `claude-code` in its pane
4. Attaches to session so you can watch all tasks run

That's it. No daemons, no state files, no polling.

## Tmux Commands

```bash
# Detach from session (keep tasks running)
Ctrl-b d

# Reattach later
tmux attach -t conductor

# Kill session
tmux kill-session -t conductor

# List sessions
tmux ls
```

## Requirements

- `tmux`
- `claude-code` (Anthropic's Claude CLI)
- `git` (only if using `-w` for worktrees)

## Philosophy

This tool does one thing: launch Claude Code in multiple tmux panes.

- **No orchestration** - tmux coordinates the layout
- **No state tracking** - your eyes are the status monitor
- **No result aggregation** - read the terminal
- **No daemons** - just shell and tmux

Simple tools, loosely joined.

## License

MIT
