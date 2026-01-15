# Claudes

Manage multiple Claude Code sessions in worktrees from a single terminal interface.

## Installation

```bash
go install github.com/brittonhayes/claudes/cmd/claudes@latest
```

Requires Go 1.21+ and Claude Code CLI (`claude`).

## Usage

### Spawn Parallel Sessions

```bash
claudes "Review auth.py" "Run all tests" "Check for TODOs"
```

This spawns 3 background Claude sessions and opens the TUI.

### From File

```bash
cat > prompts.txt <<EOF
Review all Python files for security issues


Run the test suite and fix any failures


Update documentation for new API endpoints
EOF

claudes -f prompts.txt
```

Prompts separated by 3+ newlines.

### View Active Sessions

```bash
claudes
```

Opens the TUI showing all active sessions:

```
Claudes

> 0  Review auth.py      Running      2m       Reviewing authentication...
  1  Run all tests       Running      2m       Running pytest suite...
  2  Check for TODOs     Complete     1m       Found 12 TODO items

[↑↓] navigate  [enter] resume  [d] delete  [r] refresh  [q] quit
```

### Resume Sessions

1. Navigate with ↑↓ or j/k
2. Press Enter on a session
3. Opens Claude Code UI with the resumed session
4. When you exit Claude Code, automatically returns to Claudes TUI

Sessions stay alive for continuous conversation with full Claude Code UI access.

### Delete Sessions

Press `d` on any session to delete it.

## Reference

### Command Options

```
-f FILE    Read prompts from file (- for stdin)
-h         Show help
```

## License

MIT
