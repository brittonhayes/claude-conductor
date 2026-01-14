# Claude Conductor

> Filesystem-based orchestration system for Claude Code - one parent session coordinating multiple worker sessions

## Design Philosophy

**Mechanism, not policy. Do one thing well. Make it obvious how it works.**

Claude Conductor enables parallel work distribution across multiple Claude Code sessions using nothing but text files and Unix tools. No databases, no message queues, no complex state machines - just filesystem operations and simple shell scripts.

## How It Works

```
┌─────────────────────────────────────────────────────────────┐
│                    Orchestrator Session                     │
│         (Decomposes goals → Assigns tasks)                  │
└─────────────────┬───────────────┬───────────────┬───────────┘
                  │               │               │
                  ▼               ▼               ▼
         ~/.claude-code/orchestrator/workers/
                  │               │               │
         ┌────────┴────────┬──────┴──────┬────────┴────────┐
         │                 │             │                 │
    ┌────▼─────┐      ┌────▼─────┐ ┌────▼─────┐      ┌────▼─────┐
    │ Worker 0 │      │ Worker 1 │ │ Worker 2 │ ...  │ Worker N │
    │  (idle)  │      │(working) │ │  (done)  │      │ (error)  │
    └──────────┘      └──────────┘ └──────────┘      └──────────┘
```

### Task Queue Structure

```
~/.claude-code/orchestrator/workers/
├── 0/
│   ├── task.json      # Pending work
│   ├── status.json    # Current state (idle/working/done/error)
│   ├── result.json    # Completed output
│   └── .lock/         # Simple mutex with PID
├── 1/
│   ├── task.json
│   ├── status.json
│   ├── result.json
│   └── .lock/
└── 2/
    ├── task.json
    ├── status.json
    ├── result.json
    └── .lock/
```

## Components

### 1. Task Queue (`lib/queue.js`)

Simple Node.js library for reading/writing task files:

```javascript
import { writeTask, readStatus, readResult, waitForStatus } from './lib/queue.js';

// Assign a task
writeTask(workerId, {
  prompt: "Review auth.js for security issues",
  context: { file: "src/auth.js" }
});

// Check status
const status = readStatus(workerId);  // { status: 'working', timestamp: '...' }

// Wait for completion
await waitForStatus(workerId, 'done');

// Get result
const result = readResult(workerId);  // { output: '...', success: true }
```

### 2. Worker Script (`bin/worker.sh`)

Daemon that watches for tasks and executes them:

```bash
./bin/worker.sh <worker_id> [worktree_path]
```

- Uses `inotify` (Linux) or `fswatch` (macOS) for efficient file watching
- Acquires lock before processing
- Pipes task prompt to Claude Code stdin
- Writes result and updates status
- Handles stale locks automatically (checks PID)

### 3. Orchestrator (`bin/orchestrator.js`)

Coordinates work distribution:

```javascript
import { Orchestrator } from './bin/orchestrator.js';

const orch = new Orchestrator(3);  // 3 workers

// Parallel execution
const results = await orch.executeTasks([
  { prompt: "Review file A" },
  { prompt: "Review file B" },
  { prompt: "Review file C" }
]);

// Sequential execution (with dependencies)
const results = await orch.executeSequential([
  { prompt: "Build the project", stopOnError: true },
  { prompt: "Run tests", stopOnError: true },
  { prompt: "Deploy to staging" }
]);
```

### 4. Launcher (`bin/launch.sh`)

Creates tmux session with orchestrator + workers:

```bash
./bin/launch.sh --workers 3 --worktrees
```

**Tmux Layout:**

```
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
```

## Quick Start

### Installation

```bash
npm install -g claude-conductor

# Or from source
git clone <repo>
cd claude-conductor
npm link
```

### Basic Usage

**1. Launch the system:**

```bash
# Simple - 3 workers in current directory
conductor-launch

# With worktrees (recommended for code changes)
conductor-launch --workers 4 --worktrees

# Custom configuration
conductor-launch --workers 5 --name my-session --dir ~/projects/myapp
```

**2. In the orchestrator pane, use Claude Code normally:**

```
I need to review all files in src/ for security issues. Can you distribute
this work across the available workers?
```

**3. Claude Code (in orchestrator mode) will:**
- Decompose the task into subtasks
- Assign to workers via task files
- Poll for completion
- Aggregate and present results

### Manual Usage (without tmux)

**Start workers manually:**

```bash
# Terminal 1
./bin/worker.sh 0 ~/project

# Terminal 2
./bin/worker.sh 1 ~/project

# Terminal 3
./bin/worker.sh 2 ~/project
```

**Use the orchestrator programmatically:**

```javascript
import { Orchestrator } from 'claude-conductor';

const orch = new Orchestrator(3);

// Parallel code review
const files = ['auth.js', 'db.js', 'api.js'];
const tasks = files.map(f => ({
  prompt: `Review ${f} for bugs and security issues`
}));

const results = await orch.executeTasks(tasks);

for (const r of results) {
  console.log(`Worker ${r.workerId}: ${r.result.output}`);
}

orch.shutdown();
```

## Use Cases

### Parallel Code Review

```javascript
const files = await glob('src/**/*.js');
const tasks = files.map(file => ({
  prompt: `Review ${file} for:
    - Security vulnerabilities
    - Bug risks
    - Code style issues
    Provide concise summary.`,
  context: { file }
}));

await orchestrator.executeTasks(tasks);
```

### Parallel Testing

```javascript
const suites = ['auth', 'api', 'db', 'utils'];
const tasks = suites.map(suite => ({
  prompt: `Run tests for ${suite} module and report results`,
  context: { suite }
}));

await orchestrator.executeTasks(tasks);
```

### Sequential Pipeline

```javascript
await orchestrator.executeSequential([
  { prompt: "Install dependencies", stopOnError: true },
  { prompt: "Build the project", stopOnError: true },
  { prompt: "Run all tests", stopOnError: true },
  { prompt: "Generate documentation", stopOnError: false }
]);
```

### Decomposed Feature Implementation

```javascript
// Claude Code in orchestrator mode would intelligently decompose this
const goal = "Add user authentication with OAuth support";

// Example decomposition (done by Claude Code)
await orchestrator.executeSequential([
  { prompt: "Create database schema for users and sessions" },
  { prompt: "Implement OAuth flow handlers" },
  { prompt: "Add authentication middleware" },
  { prompt: "Write tests for auth system" },
  { prompt: "Update API documentation" }
]);
```

## Advanced Usage

### Custom Worktrees

Use git worktrees to give each worker an isolated copy:

```bash
# Launcher handles this automatically
conductor-launch --worktrees

# Or manually
git worktree add ~/claude-work/worker-0 main
git worktree add ~/claude-work/worker-1 main
git worktree add ~/claude-work/worker-2 main

# Start workers in their worktrees
./bin/worker.sh 0 ~/claude-work/worker-0
./bin/worker.sh 1 ~/claude-work/worker-1
./bin/worker.sh 2 ~/claude-work/worker-2
```

### Task Context

Pass additional context to tasks:

```javascript
writeTask(workerId, {
  prompt: "Fix the bug in the authentication flow",
  context: {
    bugId: 123,
    priority: "high",
    relatedFiles: ["auth.js", "session.js"],
    errorLog: "..."
  }
});
```

Workers receive this context and can use it in their work.

### Monitoring

Check worker status anytime:

```bash
# Using Node.js
node -e "
  import('./lib/queue.js').then(q => {
    for (const id of q.listWorkers()) {
      console.log(\`Worker \${id}: \${q.readStatus(id).status}\`);
    }
  });
"

# Or use orchestrator
./bin/orchestrator.js 3
```

### Lock Management

Locks contain PIDs for automatic stale lock cleanup:

```javascript
if (acquireLock(workerId)) {
  // Do work
  releaseLock(workerId);
}
// If lock is held by dead process, it's automatically removed
```

## Design Decisions

### Why filesystem-based?

1. **Transparent**: `cat ~/.claude-code/orchestrator/workers/0/task.json` shows exactly what's happening
2. **Debuggable**: No hidden state, everything is a text file
3. **Simple**: No additional services, daemons, or databases
4. **Reliable**: Filesystem atomicity guarantees
5. **Scriptable**: Any language can read/write JSON files

### Why not use a proper message queue?

We're coordinating 3-5 Claude Code sessions, not a distributed system. Complexity is the enemy. A few JSON files and inotify are plenty.

### Why locks instead of atomic file operations?

Lock files with PIDs enable:
- Stale lock detection (check if PID exists)
- Debugging (see which worker has the lock)
- Clean recovery (remove locks from dead processes)

### Why tmux instead of a custom UI?

- Already installed on most dev systems
- Scriptable and automatable
- Users can attach/detach freely
- No dependencies, no frameworks
- Works over SSH

## Troubleshooting

### Workers not picking up tasks

**Check if worker is running:**
```bash
ps aux | grep worker.sh
```

**Check worker status:**
```bash
cat ~/.claude-code/orchestrator/workers/0/status.json
```

**Check for stuck locks:**
```bash
ls -la ~/.claude-code/orchestrator/workers/*/\.lock
# Remove stale locks manually if needed
```

### Task files not being watched

**Install inotify-tools (Linux) or fswatch (macOS):**
```bash
# Linux
apt-get install inotify-tools

# macOS
brew install fswatch
```

**Workers fall back to polling without these tools (slower but functional).**

### Orchestrator can't find workers

**Ensure workers are initialized:**
```bash
./bin/orchestrator.js 3  # Creates worker directories
```

**Check queue directory:**
```bash
ls -la ~/.claude-code/orchestrator/workers/
```

### Results not appearing

**Check result file:**
```bash
cat ~/.claude-code/orchestrator/workers/0/result.json
```

**Check worker logs:**
Workers log to stderr - check your terminal or redirect to a file:
```bash
./bin/worker.sh 0 ~/project 2>worker-0.log
```

## Architecture

### State Machine

Workers follow a simple state machine:

```
     ┌──────┐
     │ idle │←─────────────┐
     └──┬───┘              │
        │                  │
   task.json created       │
        │                  │
        ▼                  │
   ┌─────────┐             │
   │ working │             │
   └────┬────┘             │
        │                  │
   success/error           │
        │                  │
        ▼                  │
   ┌────────┐    result    │
   │  done  │─────read─────┤
   └────────┘              │
        │                  │
   ┌────────┐              │
   │ error  │──────────────┘
   └────────┘
```

### File Formats

**task.json:**
```json
{
  "id": 1705234567890,
  "prompt": "Review auth.js for security issues",
  "context": {
    "file": "src/auth.js",
    "priority": "high"
  },
  "timestamp": "2024-01-14T12:34:56.789Z"
}
```

**status.json:**
```json
{
  "status": "working",
  "taskId": 1705234567890,
  "timestamp": "2024-01-14T12:35:00.123Z"
}
```

**result.json:**
```json
{
  "taskId": 1705234567890,
  "output": "Review complete. Found 2 issues:\n1. SQL injection risk...\n2. Missing input validation...",
  "success": true,
  "error": null,
  "timestamp": "2024-01-14T12:36:45.678Z"
}
```

**.lock/pid:**
```
12345
```

## API Reference

See [lib/queue.js](lib/queue.js) for the complete API.

**Core functions:**
- `initQueue()` - Initialize queue directory
- `createWorker(id)` - Create worker slot
- `writeTask(id, task)` - Assign task
- `readStatus(id)` - Get worker status
- `readResult(id)` - Get task result
- `waitForStatus(id, status, timeout)` - Wait for state change
- `acquireLock(id)` - Acquire worker lock
- `releaseLock(id)` - Release worker lock

## Examples

See [examples/](examples/) for complete examples:
- `orchestrator-helper.js` - High-level helper functions for common patterns

## Contributing

This is a Unix-philosophy tool: simple, focused, composable.

When contributing:
- Keep it simple (no frameworks, no abstraction layers)
- Keep it transparent (everything should be inspectable)
- Keep it scriptable (text files and simple APIs)
- Make it obvious (prefer explicit over clever)

## License

MIT

## Philosophy

> "Make each program do one thing well. To do a new job, build afresh rather than complicate old programs by adding new features."
>
> — Doug McIlroy, Unix Philosophy

Claude Conductor does one thing: coordinate Claude Code sessions via filesystem-based task queues. Nothing more, nothing less.
