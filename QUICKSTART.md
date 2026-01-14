# Quick Start Guide

Get up and running with Claude Conductor in under 5 minutes.

## Installation

```bash
# Clone the repository
git clone <repo-url>
cd claude-conductor

# Make scripts executable (already done if installed via npm)
chmod +x bin/*.sh bin/*.js

# Optional: Install globally
npm link
```

## Usage Patterns

### Pattern 1: Launch with tmux (Recommended)

**Start everything at once:**

```bash
./bin/launch.sh --workers 3
```

This creates a tmux session with:
- 1 orchestrator pane (interactive Claude Code)
- 3 worker panes (daemon mode)

**What you'll see:**

```
┌─────────────────────────────────┬──────────────────────────────────┐
│ Orchestrator - Interactive Mode │ Worker 0 - Starting...           │
│ Workers: 3                      │                                  │
│ Queue: ~/.claude-code/...       │                                  │
│                                 │                                  │
│ Ready to coordinate!            │                                  │
├─────────────────────────────────┼──────────────────────────────────┤
│ Worker 1 - Starting...          │ Worker 2 - Starting...           │
│                                 │                                  │
│                                 │                                  │
│                                 │                                  │
└─────────────────────────────────┴──────────────────────────────────┘
```

**Use it:**

In the orchestrator pane, just talk to Claude Code normally:

```
You: I need to review all .js files in src/ for security issues.
     Can you distribute this across the workers?

Claude: I'll analyze the src/ directory and distribute the review
        work across 3 workers. Let me break this down...
```

**Exit:**
- `Ctrl-b d` to detach (workers keep running)
- `tmux attach -t claude-conductor` to reattach
- `tmux kill-session -t claude-conductor` to stop everything

### Pattern 2: Manual Setup

**Terminal 1: Start orchestrator**
```bash
./bin/orchestrator.js 3 --interactive
```

**Terminals 2-4: Start workers**
```bash
./bin/worker.sh 0 .
./bin/worker.sh 1 .
./bin/worker.sh 2 .
```

**Terminal 5: Use programmatically**
```javascript
import { Orchestrator } from './bin/orchestrator.js';

const orch = new Orchestrator(3);

const results = await orch.executeTasks([
  { prompt: 'Task 1' },
  { prompt: 'Task 2' },
  { prompt: 'Task 3' }
]);

console.log(results);
orch.shutdown();
```

### Pattern 3: Worktrees (Isolated Changes)

**Use git worktrees to give each worker its own copy:**

```bash
./bin/launch.sh --workers 3 --worktrees --dir ~/claude-work
```

This creates:
```
~/claude-work/
├── worker-0/  (git worktree)
├── worker-1/  (git worktree)
└── worker-2/  (git worktree)
```

**Why?** Workers can make changes without conflicts. Perfect for:
- Parallel feature development
- Independent bug fixes
- Isolated experiments

## Common Tasks

### Task 1: Parallel Code Review

```javascript
import { Orchestrator } from 'claude-conductor';
import { glob } from 'glob';

const orch = new Orchestrator(3);

const files = await glob('src/**/*.js');
const tasks = files.map(f => ({
  prompt: `Review ${f} for bugs and security issues. Be concise.`
}));

const results = await orch.executeTasks(tasks);

for (const r of results) {
  console.log(`${r.task.prompt}:\n${r.result.output}\n`);
}

orch.shutdown();
```

### Task 2: Sequential Pipeline

```javascript
const orch = new Orchestrator(3);

await orch.executeSequential([
  { prompt: 'Install dependencies with npm install', stopOnError: true },
  { prompt: 'Build the project with npm run build', stopOnError: true },
  { prompt: 'Run all tests with npm test', stopOnError: true },
  { prompt: 'Generate API docs' }
]);

orch.shutdown();
```

### Task 3: Distributed Testing

```javascript
const orch = new Orchestrator(4);

const testSuites = ['unit', 'integration', 'e2e', 'performance'];
const tasks = testSuites.map(suite => ({
  prompt: `Run ${suite} tests and report pass/fail counts`
}));

await orch.executeTasks(tasks);
orch.shutdown();
```

### Task 4: Check Status

```javascript
import { listWorkers, readStatus } from 'claude-conductor/lib/queue.js';

for (const id of listWorkers()) {
  const status = readStatus(id);
  console.log(`Worker ${id}: ${status.status}`);
}
```

## Command Reference

### Launch Script

```bash
./bin/launch.sh [options]

Options:
  -w, --workers NUM     Number of workers (default: 3)
  -n, --name NAME       Session name (default: claude-conductor)
  --worktrees           Use git worktrees
  -d, --dir DIR         Base directory (default: ~/claude-work)
  -h, --help            Show help
```

### Orchestrator Script

```bash
./bin/orchestrator.js <num_workers> [--interactive]

Arguments:
  num_workers           Number of worker slots to create

Options:
  --interactive         Keep running for interactive use
```

### Worker Script

```bash
./bin/worker.sh <worker_id> [worktree_path]

Arguments:
  worker_id             Unique ID (0, 1, 2, ...)
  worktree_path         Working directory (default: .)
```

## File Locations

**Queue directory:**
```
~/.claude-code/orchestrator/workers/
├── 0/
│   ├── task.json      # Current task
│   ├── status.json    # Worker status
│   ├── result.json    # Task result
│   └── .lock/         # Lock with PID
├── 1/
└── 2/
```

**Inspect manually:**
```bash
# See all workers
ls ~/.claude-code/orchestrator/workers/

# Check worker 0 status
cat ~/.claude-code/orchestrator/workers/0/status.json

# Read worker 0 result
cat ~/.claude-code/orchestrator/workers/0/result.json
```

## Troubleshooting

### Workers not starting

**Check dependencies:**
```bash
# Linux
which inotifywait || sudo apt-get install inotify-tools

# macOS
which fswatch || brew install fswatch
```

Without these, workers fall back to polling (slower but functional).

### Tasks not being picked up

**Check worker status:**
```bash
cat ~/.claude-code/orchestrator/workers/0/status.json
```

**Check worker is running:**
```bash
ps aux | grep worker.sh
```

### Stuck locks

**Check lock:**
```bash
cat ~/.claude-code/orchestrator/workers/0/.lock/pid
kill -0 $(cat ~/.claude-code/orchestrator/workers/0/.lock/pid)
```

**Remove stale lock:**
```bash
rm -rf ~/.claude-code/orchestrator/workers/0/.lock
```

### Reset everything

**Kill all workers:**
```bash
pkill -f worker.sh
tmux kill-session -t claude-conductor
```

**Clear queue:**
```bash
rm -rf ~/.claude-code/orchestrator/workers/*
```

**Reinitialize:**
```bash
./bin/orchestrator.js 3
```

## Tips & Tricks

### 1. Monitor Workers in Real-Time

```bash
watch -n 0.5 'cat ~/.claude-code/orchestrator/workers/*/status.json | jq -r ".status"'
```

### 2. View Worker Logs

```bash
# Redirect stderr to log file
./bin/worker.sh 0 . 2>worker-0.log

# Tail in another terminal
tail -f worker-0.log
```

### 3. Custom Worker Count

More workers = more parallelism, but diminishing returns:

```bash
# Good for many independent tasks
./bin/launch.sh --workers 8

# Good for few heavy tasks
./bin/launch.sh --workers 2
```

### 4. Selective Worktrees

Use worktrees only for workers that need isolation:

```bash
# Worker 0 in main directory (read-only tasks)
./bin/worker.sh 0 .

# Workers 1-3 in worktrees (write tasks)
./bin/worker.sh 1 ~/claude-work/worker-1
./bin/worker.sh 2 ~/claude-work/worker-2
./bin/worker.sh 3 ~/claude-work/worker-3
```

### 5. Debug Mode

See what workers are doing:

```bash
# Verbose worker
bash -x ./bin/worker.sh 0 . 2>&1 | tee debug.log
```

## Next Steps

- Read [README.md](README.md) for complete documentation
- Check [ARCHITECTURE.md](ARCHITECTURE.md) for design details
- See [examples/](examples/) for code samples

## Getting Help

**Check queue state:**
```bash
./bin/orchestrator.js 3  # Shows worker status
```

**Validate setup:**
```bash
# Workers exist?
ls ~/.claude-code/orchestrator/workers/

# Can write?
echo '{"test":true}' > ~/.claude-code/orchestrator/workers/0/test.json
rm ~/.claude-code/orchestrator/workers/0/test.json
```

**Still stuck?** Check the [README.md](README.md) troubleshooting section.
