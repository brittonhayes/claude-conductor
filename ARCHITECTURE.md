# Architecture

This document explains the internal design of Claude Conductor.

## Core Principles

1. **Filesystem as the source of truth** - No in-memory state that can't be recovered from disk
2. **PIDs for liveness detection** - Lock files contain process IDs for automatic cleanup
3. **Polling with backoff** - Simple, reliable, debuggable
4. **No complex state machines** - Linear workflows: assign → wait → collect
5. **Crash-safe by design** - Everything restartable, nothing irreversible

## Directory Structure

```
claude-conductor/
├── bin/
│   ├── orchestrator.js    # Coordinator process (Node.js)
│   ├── worker.sh          # Worker daemon (Bash)
│   └── launch.sh          # Tmux launcher (Bash)
├── lib/
│   └── queue.js           # Queue management library (Node.js)
├── examples/
│   ├── orchestrator-helper.js   # High-level patterns
│   └── simple-demo.js           # Basic usage demo
└── README.md

Runtime state:
~/.claude-code/orchestrator/workers/
├── 0/
├── 1/
└── 2/
```

## Component Interactions

```
┌─────────────────────────────────────────────────────────────┐
│                     Orchestrator                            │
│  - Decomposes goals into tasks                              │
│  - Calls writeTask() to assign work                         │
│  - Polls readStatus() for completion                        │
│  - Calls readResult() to collect output                     │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  │ writes task.json
                  ▼
         ~/.claude-code/orchestrator/workers/N/
                  │
                  │ inotify/fswatch triggers
                  ▼
┌─────────────────────────────────────────────────────────────┐
│                      Worker                                 │
│  - Watches for task.json                                    │
│  - Acquires lock (writes PID)                               │
│  - Pipes task.prompt to Claude Code stdin                   │
│  - Captures stdout → result.json                            │
│  - Updates status.json                                      │
│  - Releases lock                                            │
└─────────────────────────────────────────────────────────────┘
```

## File Lifecycle

### task.json

```
Created by:  writeTask() in orchestrator
Read by:     worker.sh
Deleted by:  worker.sh (after processing)
Lock held:   During read and delete
```

### status.json

```
Created by:  createWorker() → "idle"
Updated by:  worker.sh → "working", "done", "error"
Read by:     orchestrator (polling)
Lock held:   During writes
```

### result.json

```
Created by:  worker.sh (after task completion)
Read by:     readResult() in orchestrator
Deleted by:  clearResult() in orchestrator (before next task)
Lock held:   During write
```

### .lock/

```
Created by:  acquireLock() (mkdir .lock)
Contains:    .lock/pid with process ID
Deleted by:  releaseLock() (rmdir .lock)
Checked by:  Stale lock cleanup (kill -0 $PID)
```

## Lock Protocol

**Why directory-based locks?**
- Atomic on all POSIX filesystems (mkdir is atomic)
- Easy to inspect (just ls)
- Can store metadata (PID file inside)

**Lock acquisition:**
```bash
1. mkdir .lock (atomic)
2. echo $$ > .lock/pid
3. If mkdir fails:
   - Read .lock/pid
   - Check if process exists (kill -0 $PID)
   - If dead, remove lock and retry
   - If alive, wait and retry
```

**Lock release:**
```bash
1. rm -rf .lock
```

**Stale lock cleanup:**
```bash
if [ -f .lock/pid ]; then
  old_pid=$(cat .lock/pid)
  if ! kill -0 $old_pid 2>/dev/null; then
    # Process is dead, lock is stale
    rm -rf .lock
  fi
fi
```

## State Transitions

### Worker States

```
IDLE → WORKING → DONE → IDLE
  ↓       ↓
  ↓     ERROR → IDLE
  ↓
  └────────────────────→ (loop)
```

**IDLE:**
- No task.json exists
- Worker is polling/watching for new task
- Ready to accept work

**WORKING:**
- task.json received
- Lock acquired
- Claude Code process running
- Periodic heartbeat possible (future)

**DONE:**
- Claude Code finished successfully
- result.json written
- status.json updated
- Waiting for orchestrator to read result

**ERROR:**
- Claude Code exited with error
- result.json contains error details
- status.json updated with error state
- Waiting for orchestrator to read result

## Error Handling

### Worker crashes

**Problem:** Worker dies while processing task

**Solution:**
- Lock contains PID
- Next worker acquiring lock checks if PID exists
- If dead, removes lock and processes task
- Result: Task is retried automatically

### Orchestrator crashes

**Problem:** Orchestrator dies while waiting for results

**Solution:**
- Workers continue processing
- Results are written to disk
- Orchestrator restart can:
  - Check status.json for all workers
  - Read available results
  - Resume or restart work

### Filesystem issues

**Problem:** Disk full, permissions errors, etc.

**Solution:**
- All file operations wrapped in try/catch
- Workers log errors to stderr
- status.json reflects error state
- Manual intervention required (by design - fail visible)

## Scalability Considerations

**Current design supports:**
- 1-10 workers comfortably
- Sub-second task assignment latency
- Minimal CPU overhead (inotify-based)

**Not designed for:**
- 100+ workers
- Sub-millisecond latency
- Cross-machine distribution
- High-frequency task scheduling

**Why these limits?**
- We're coordinating AI sessions, not microservices
- Simplicity > raw performance
- If you need more, use a real message queue

## Performance

### Task Assignment Latency

```
Orchestrator:
  writeTask()           ~1ms   (JSON serialize + file write)

Worker:
  inotify trigger       ~10ms  (kernel → userspace)
  read + validate       ~1ms   (JSON parse)
  acquireLock()         ~1ms   (mkdir)
  ───────────────────────────
  Total:                ~13ms  (worst case)
```

### Polling vs. inotify

**With inotify/fswatch:**
- Near-instant task pickup (~10ms)
- Zero CPU when idle
- Scales to many workers

**Without (polling fallback):**
- 1-second delay
- Constant CPU usage
- Works everywhere

## File Watching Implementation

### Linux (inotifywait)

```bash
inotifywait -q -e create,modify "$QUEUE_DIR"
```

**Pros:**
- Instant notification
- Kernel-level efficiency
- No polling overhead

**Cons:**
- Linux-only
- Requires inotify-tools package

### macOS (fswatch)

```bash
fswatch -0 "$QUEUE_DIR"
```

**Pros:**
- Near-instant notification
- Low CPU usage

**Cons:**
- macOS-only
- Requires fswatch package

### Fallback (polling)

```bash
while true; do
  if [ -f "$TASK_FILE" ]; then
    process_task
  fi
  sleep 1
done
```

**Pros:**
- Works everywhere
- No dependencies
- Simple to debug

**Cons:**
- 1-second latency
- Constant wake-ups

## Future Enhancements

Potential improvements (without violating design principles):

### Heartbeats

Add periodic status updates while working:

```json
{
  "status": "working",
  "taskId": 123,
  "progress": "Running tests... 45% complete",
  "timestamp": "..."
}
```

### Task Priorities

Add priority field to tasks, workers process high-priority first:

```json
{
  "id": 123,
  "prompt": "...",
  "priority": 10,
  "timestamp": "..."
}
```

### Result Streaming

For long-running tasks, stream partial results:

```
result.json       # Final result
result.stream/    # Incremental updates
  ├── 1.json
  ├── 2.json
  └── 3.json
```

### Work Stealing

Allow idle workers to steal tasks from overloaded workers:

1. Worker A gets 10-minute task
2. Workers B, C are idle
3. Workers B, C check A's task age
4. If > threshold, split work or take queued subtasks

### Distributed Mode

For cross-machine orchestration:

1. Replace local filesystem with NFS/shared mount
2. No code changes needed (just mount paths)
3. Works transparently

## Testing Strategy

### Unit Tests

Test queue.js functions in isolation:
- File read/write operations
- Lock acquisition/release
- Status transitions
- Error handling

### Integration Tests

Test full workflows:
1. Start workers
2. Assign tasks
3. Verify results
4. Test crash recovery

### Chaos Tests

Test resilience:
- Kill workers mid-task
- Fill disk during task
- Remove lock files manually
- Corrupt JSON files

## Debugging

### Inspect Queue State

```bash
# See all workers
ls ~/.claude-code/orchestrator/workers/

# Check worker status
cat ~/.claude-code/orchestrator/workers/0/status.json

# See pending task
cat ~/.claude-code/orchestrator/workers/0/task.json

# Read result
cat ~/.claude-code/orchestrator/workers/0/result.json

# Check lock status
ls -la ~/.claude-code/orchestrator/workers/0/.lock/
cat ~/.claude-code/orchestrator/workers/0/.lock/pid
```

### Trace Worker Activity

```bash
# Watch worker logs
./bin/worker.sh 0 ~/project 2>&1 | tee worker-0.log

# Monitor file changes
watch -n 0.5 'ls -lh ~/.claude-code/orchestrator/workers/0/'

# Follow status changes
watch -n 0.5 'cat ~/.claude-code/orchestrator/workers/*/status.json'
```

### Validate JSON Files

```bash
# Check all JSON files are valid
for f in ~/.claude-code/orchestrator/workers/*/*.json; do
  echo "Checking $f"
  jq empty "$f" || echo "Invalid JSON: $f"
done
```

## Security Considerations

### Race Conditions

**Problem:** Multiple processes writing same file

**Mitigation:** Lock protocol ensures exclusive access

### Symlink Attacks

**Problem:** Malicious task.json symlink to sensitive file

**Mitigation:**
- Workers run as same user as orchestrator
- No privilege escalation
- Could add symlink detection if needed

### Disk Space Exhaustion

**Problem:** Malicious large task/result files

**Mitigation:**
- Workers run with same quotas as user
- Could add size limits to JSON parsing

### Process Injection

**Problem:** Malicious prompt containing shell escapes

**Mitigation:**
- Prompts piped to stdin, not passed via command line
- No shell expansion in worker script
- Claude Code itself handles prompt validation

## Maintenance

### Cleanup Old Results

```bash
# Remove results older than 24 hours
find ~/.claude-code/orchestrator/workers/*/result.json \
  -mtime +1 -delete
```

### Reset All Workers

```bash
# Stop all workers
pkill -f worker.sh

# Clear queue
rm -rf ~/.claude-code/orchestrator/workers/*

# Reinitialize
./bin/orchestrator.js 3
```

### Monitor Disk Usage

```bash
# Check queue size
du -sh ~/.claude-code/orchestrator/
```

## Comparison to Alternatives

### vs. Redis Queue

**Conductor:** Simpler, no dependencies, transparent
**Redis:** Faster, more features, requires service

### vs. RabbitMQ

**Conductor:** No setup, works anywhere
**RabbitMQ:** Enterprise-grade, complex

### vs. Celery

**Conductor:** 200 lines of code, obvious behavior
**Celery:** Mature ecosystem, steep learning curve

### vs. Simple SSH

**Conductor:** Coordinated, result aggregation, status tracking
**SSH:** Manual, no orchestration, fire-and-forget

## Philosophy

Claude Conductor is intentionally simple:

- **No database** - Filesystem is the database
- **No server** - Just shell scripts and Node.js
- **No config** - Sensible defaults, override with args
- **No abstraction** - Every component is obvious
- **No magic** - Everything is a text file you can cat

This is a tool for **developers** who want to **understand** what's happening, not hide it behind abstractions.
