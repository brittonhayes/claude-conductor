# Claudes

Manage multiple Claude Code sessions from a single terminal interface.

## Installation

### Prerequisites

- Go 1.21+
- Claude Code CLI (`claude`)

### Using go install

```bash
go install github.com/brittonhayes/claudes/cmd/claudes@latest
```

### Build from Source

```bash
git clone https://github.com/brittonhayes/claudes
cd claudes
go build -o claudes ./cmd/claudes
sudo mv claudes /usr/local/bin/
```

Or install to ~/.local/bin:

```bash
go build -o claudes ./cmd/claudes
mkdir -p ~/.local/bin
mv claudes ~/.local/bin/
```

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

## Reference

### Command Options

```
-f FILE    Read prompts from file (- for stdin)
-h         Show help
```

## License

MIT
