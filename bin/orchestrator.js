#!/usr/bin/env node

/**
 * Orchestrator for Claude Code - coordinates multiple worker sessions
 * Usage: orchestrator.js <num_workers> <goal>
 *
 * The orchestrator reads natural language goals, decomposes into tasks,
 * assigns to workers, and aggregates results.
 */

import {
  initQueue,
  createWorker,
  listWorkers,
  writeTask,
  readStatus,
  readResult,
  clearResult,
  writeStatus,
  waitForStatus
} from '../lib/queue.js';

class Orchestrator {
  constructor(numWorkers) {
    this.numWorkers = numWorkers;
    this.workers = [];

    // Initialize queue
    initQueue();

    // Create worker slots
    for (let i = 0; i < numWorkers; i++) {
      createWorker(i);
      this.workers.push(i);
    }

    console.log(`Orchestrator initialized with ${numWorkers} workers`);
  }

  /**
   * Find an idle worker
   */
  async findIdleWorker() {
    for (const workerId of this.workers) {
      const status = readStatus(workerId);
      if (status.status === 'idle' || status.status === 'done') {
        return workerId;
      }
    }
    return null;
  }

  /**
   * Wait for any worker to become available
   */
  async waitForAvailableWorker(timeoutMs = 300000) {
    const startTime = Date.now();

    while (Date.now() - startTime < timeoutMs) {
      const workerId = await this.findIdleWorker();
      if (workerId !== null) {
        return workerId;
      }

      // Wait a bit before checking again
      await new Promise(resolve => setTimeout(resolve, 1000));
    }

    throw new Error('Timeout waiting for available worker');
  }

  /**
   * Assign a task to a worker
   */
  async assignTask(workerId, prompt, context = {}) {
    console.log(`\n[Orchestrator] Assigning task to worker ${workerId}`);
    console.log(`[Orchestrator] Task: ${prompt.substring(0, 100)}...`);

    // Clear any previous result
    clearResult(workerId);

    // Write task
    const taskId = writeTask(workerId, { prompt, context });

    return taskId;
  }

  /**
   * Wait for a worker to complete its task
   */
  async waitForCompletion(workerId, timeoutMs = 300000) {
    console.log(`[Orchestrator] Waiting for worker ${workerId} to complete...`);

    const status = await waitForStatus(workerId, ['done', 'error'], timeoutMs);

    if (status.status === 'error') {
      console.error(`[Orchestrator] Worker ${workerId} encountered an error`);
    } else {
      console.log(`[Orchestrator] Worker ${workerId} completed task`);
    }

    return status;
  }

  /**
   * Get result from a worker
   */
  getResult(workerId) {
    return readResult(workerId);
  }

  /**
   * Execute tasks in parallel across workers
   */
  async executeTasks(tasks) {
    const results = [];
    const taskQueue = [...tasks];
    const activeWorkers = new Map();

    console.log(`\n[Orchestrator] Executing ${tasks.length} tasks across ${this.numWorkers} workers`);

    // Process tasks
    while (taskQueue.length > 0 || activeWorkers.size > 0) {
      // Assign tasks to idle workers
      while (taskQueue.length > 0) {
        const workerId = await this.findIdleWorker();
        if (workerId === null) break;

        const task = taskQueue.shift();
        const taskId = await this.assignTask(workerId, task.prompt, task.context);

        activeWorkers.set(workerId, { taskId, originalTask: task });
      }

      // Wait for any worker to complete
      if (activeWorkers.size > 0) {
        for (const workerId of activeWorkers.keys()) {
          const status = readStatus(workerId);

          if (status.status === 'done' || status.status === 'error') {
            const result = this.getResult(workerId);
            const workerInfo = activeWorkers.get(workerId);

            results.push({
              workerId,
              taskId: workerInfo.taskId,
              task: workerInfo.originalTask,
              result,
              status: status.status
            });

            // Reset worker to idle
            writeStatus(workerId, 'idle');
            activeWorkers.delete(workerId);

            console.log(`[Orchestrator] Worker ${workerId} finished (${results.length}/${tasks.length} complete)`);
          }
        }

        // Small delay before checking again
        await new Promise(resolve => setTimeout(resolve, 500));
      }
    }

    return results;
  }

  /**
   * Execute tasks sequentially with dependency handling
   */
  async executeSequential(tasks) {
    const results = [];

    console.log(`\n[Orchestrator] Executing ${tasks.length} tasks sequentially`);

    for (let i = 0; i < tasks.length; i++) {
      const task = tasks[i];
      const workerId = await this.waitForAvailableWorker();

      console.log(`\n[Orchestrator] Task ${i + 1}/${tasks.length}`);

      const taskId = await this.assignTask(workerId, task.prompt, task.context);
      await this.waitForCompletion(workerId);

      const result = this.getResult(workerId);
      results.push({
        workerId,
        taskId,
        task,
        result,
        status: readStatus(workerId).status
      });

      // Reset worker
      writeStatus(workerId, 'idle');

      // Check if we should continue based on result
      if (task.stopOnError && !result.success) {
        console.error(`[Orchestrator] Task failed, stopping execution`);
        break;
      }
    }

    return results;
  }

  /**
   * Get status of all workers
   */
  getWorkersStatus() {
    return this.workers.map(workerId => ({
      id: workerId,
      status: readStatus(workerId)
    }));
  }

  /**
   * Shut down orchestrator
   */
  shutdown() {
    console.log('\n[Orchestrator] Shutting down...');

    // Set all workers to idle
    for (const workerId of this.workers) {
      writeStatus(workerId, 'idle');
    }
  }
}

// CLI interface
async function main() {
  const args = process.argv.slice(2);

  if (args.length < 1) {
    console.error('Usage: orchestrator.js <num_workers> [--interactive]');
    console.error('');
    console.error('Options:');
    console.error('  --interactive    Start in interactive mode (for use with Claude Code)');
    console.error('');
    console.error('Examples:');
    console.error('  orchestrator.js 3 --interactive');
    console.error('  orchestrator.js 4');
    process.exit(1);
  }

  const numWorkers = parseInt(args[0]);
  const interactive = args.includes('--interactive');

  if (isNaN(numWorkers) || numWorkers < 1) {
    console.error('Error: num_workers must be a positive integer');
    process.exit(1);
  }

  const orchestrator = new Orchestrator(numWorkers);

  if (interactive) {
    console.log('\n='.repeat(60));
    console.log('CLAUDE CODE ORCHESTRATOR - Interactive Mode');
    console.log('='.repeat(60));
    console.log(`Workers: ${numWorkers}`);
    console.log(`Queue dir: ~/.claude-code/orchestrator/workers/`);
    console.log('');
    console.log('This orchestrator is ready to coordinate parallel work.');
    console.log('You can now describe your goal, and I will decompose it');
    console.log('into tasks and assign them to workers.');
    console.log('='.repeat(60));
    console.log('');

    // Keep process alive for interactive use
    process.stdin.resume();
  } else {
    // Non-interactive mode - print status and exit
    console.log('\nWorker Status:');
    console.log('─'.repeat(40));

    const statuses = orchestrator.getWorkersStatus();
    for (const worker of statuses) {
      console.log(`Worker ${worker.id}: ${worker.status.status}`);
    }

    console.log('');
    console.log('Orchestrator ready. Workers are waiting for tasks.');
  }

  // Cleanup on exit
  process.on('SIGINT', () => {
    orchestrator.shutdown();
    process.exit(0);
  });

  process.on('SIGTERM', () => {
    orchestrator.shutdown();
    process.exit(0);
  });
}

// Export for use as a module
export { Orchestrator };

// Run CLI if executed directly
if (import.meta.url === `file://${process.argv[1]}`) {
  main().catch(err => {
    console.error('Error:', err.message);
    process.exit(1);
  });
}
