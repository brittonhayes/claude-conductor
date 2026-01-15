# Claude Conductor

Centralized management for multiple Claude Code sessions. Spawn parallel agents, monitor progress, and attach for follow-ups.

## Installation

### Prerequisites

- Go 1.21+
- Claude Code CLI (`claude`)

### Build from Source

```bash
git clone https://github.com/brittonhayes/claude-conductor
cd claude-conductor
go build -o conductor .
sudo mv conductor /usr/local/bin/
```

Or install to ~/.local/bin:

```bash
go build -o conductor .
mkdir -p ~/.local/bin
mv conductor ~/.local/bin/
```

## Usage

### Spawn Parallel Tasks

```bash
conductor "Review auth.py" "Run all tests" "Check for TODOs"
```

This spawns 3 background Claude sessions and opens the TUI.

### From File

```bash
cat > tasks.txt <<EOF
Review all Python files for security issues


Run the test suite and fix any failures


Update documentation for new API endpoints
EOF

conductor -f tasks.txt
```

Tasks separated by 3+ newlines.

### View Active Sessions

```bash
conductor
```

Opens the TUI showing all active sessions:

```
Claude Conductor

> 0  Review auth.py      Running      2m       Reviewing authentication...
  1  Run all tests       Running      2m       Running pytest suite...
  2  Check for TODOs     Complete     1m       Found 12 TODO items

[↑↓] navigate  [enter] attach  [d] delete  [r] refresh  [q] quit
```

### Attach and Follow Up

1. Navigate with ↑↓ or j/k
2. Press Enter on a session
3. Type your follow-up prompt
4. Watch the response stream
5. Automatically returns to TUI

Sessions stay alive for continuous conversation.

### Delete Sessions

Press `d` on any session to delete it.

## How It Works

1. Spawns Claude agents using the Agent SDK
2. Each session runs in background with named session ID
3. Output streams to `~/.conductor/outputs/`
4. TUI monitors sessions and lets you attach
5. Sessions persist until explicitly deleted

No tmux. No daemons. Pure Go and Claude SDK.

## Options

```
-f FILE    Read tasks from file (- for stdin)
-h         Show help
```

## Architecture

```
main.go      CLI entry, spawning, attach loop
session.go   Session state management
agent.go     Claude SDK integration
tui.go       Bubbletea interface
```

## License

MIT
